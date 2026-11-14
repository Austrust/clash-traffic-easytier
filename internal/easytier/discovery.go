package easytier

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"clashtraffic/internal/tailnet"
)

// Candidate describes an instance which actually answered and has a local TUN IP.
type Candidate struct {
	RPC      string `json:"rpc"`
	Instance string `json:"instance"`
	Name     string `json:"name"`
	IP       string `json:"ip"`
	Host     string `json:"host"`
	Peers    int    `json:"peers"`
	Source   string `json:"source"`
	status   *tailnet.Status
}

// Discovery contains no network or synchronization secrets.
type Discovery struct {
	CLI        string      `json:"cli"`
	Version    string      `json:"version"`
	Candidates []Candidate `json:"candidates"`
	Errors     []string    `json:"errors"`
	Error      string      `json:"error"`
}

type endpoint struct{ RPC, Dir, Source string }
type inventory struct {
	Endpoints []endpoint
	Dirs      []string
	Warning   string
}
type instance struct{ ID, Name string }
type runner func(context.Context, string, ...string) ([]byte, error)

func normalizeRPC(s string, discovered bool) (string, error) {
	s = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), "tcp://"))
	if p, err := strconv.Atoi(s); err == nil && p > 0 && p < 65536 {
		s = "127.0.0.1:" + s
	}
	s = strings.Replace(s, "localhost:", "127.0.0.1:", 1)
	a, err := netip.ParseAddrPort(s)
	if err == nil && discovered && a.Addr().IsUnspecified() {
		a = netip.AddrPortFrom(netip.MustParseAddr("127.0.0.1"), a.Port())
	}
	if err != nil || !a.Addr().IsLoopback() || a.Port() == 0 {
		return "", errors.New("EasyTier: RPC must be a local loopback address and port")
	}
	return a.String(), nil
}

// cappedBuffer bounds output without breaking a child process's stderr writes.
type cappedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	left := b.limit - b.Len()
	if left > 0 {
		if left < len(p) {
			p = p[:left]
		}
		b.Buffer.Write(p)
	}
	return n, nil
}

var ansi = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
var sensitive = regexp.MustCompile(`(?i)(secret|password|token|private[_-]?key|authorization)`)
var urlQuery = regexp.MustCompile(`(https?://[^\s?]+)\?[^\s]+`)

func safeError(s string) string {
	s = ansi.ReplaceAllString(strings.ToValidUTF8(s, "?"), "")
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		if sensitive.MatchString(line) {
			lines = append(lines, "[credential detail redacted]")
		} else {
			lines = append(lines, urlQuery.ReplaceAllString(line, "$1?[redacted]"))
		}
	}
	s = strings.TrimSpace(strings.Join(lines, "\n"))
	if len(s) > 1200 {
		s = strings.ToValidUTF8(s[:1200], "") + "…"
	}
	return s
}

func runCLI(ctx context.Context, cli string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, cli, args...)
	hideWindow(cmd)
	out := &cappedBuffer{limit: 2 << 20}
	stderr := &cappedBuffer{limit: 8192}
	cmd.Stdout, cmd.Stderr = out, stderr
	err := cmd.Run()
	if err != nil {
		if ctx.Err() != nil {
			return nil, errors.New("CLI timed out; check RPC endpoint and service state")
		}
		detail := safeError(stderr.String())
		if detail == "" {
			detail = safeError(out.String())
		}
		if detail == "" {
			detail = "no diagnostic text from CLI"
		}
		return nil, fmt.Errorf("%w: %s", err, detail)
	}
	if out.Len() >= out.limit {
		return nil, errors.New("CLI output exceeded 2 MiB")
	}
	return out.Bytes(), nil
}

func findCLI(configured string, dirs []string) string {
	if configured = strings.Trim(strings.TrimSpace(configured), `"`); configured != "" {
		return filepath.Clean(configured)
	}
	if p, err := exec.LookPath("easytier-cli"); err == nil {
		return p
	}
	name := "easytier-cli"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if p, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(p))
	}
	for _, env := range []string{"ProgramFiles", "LOCALAPPDATA"} {
		if p := os.Getenv(env); p != "" {
			for _, app := range []string{"EasyTier", "EasyTier GUI", "easytier-gui", "Programs/EasyTier", "Programs/EasyTier GUI"} {
				dirs = append(dirs, filepath.Join(p, app))
			}
		}
	}
	dirs = append(dirs, "/usr/local/bin", "/usr/bin", "/opt/homebrew/bin")
	for _, dir := range dirs {
		for _, sub := range []string{"", "bin", "resources"} {
			p := filepath.Join(dir, sub, name)
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return p
			}
		}
	}
	return ""
}

var metadata = regexp.MustCompile(`^\s*(instance_id|instance_name|network_name)\s*=\s*("(?:[^"\\]|\\.)*"|'[^']*')`)
var uuid = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func parseMetadata(data []byte) instance {
	var m instance
	for _, line := range strings.Split(string(data), "\n") {
		v := metadata.FindStringSubmatch(line)
		if len(v) != 3 {
			continue
		}
		val := strings.Trim(v[2], "'")
		if strings.HasPrefix(v[2], `"`) {
			val, _ = strconv.Unquote(v[2])
		}
		switch v[1] {
		case "instance_id":
			if uuid.MatchString(val) {
				m.ID = val
			}
		case "instance_name":
			m.Name = val
		case "network_name":
			if m.Name == "" {
				m.Name = val
			}
		}
	}
	return m
}
func instances(dir string) []instance {
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []instance
	for _, e := range entries {
		if len(out) >= 32 {
			break
		}
		if e.IsDir() || filepath.Ext(e.Name()) != ".toml" {
			continue
		}
		info, err := e.Info()
		if err != nil || info.Size() > 1<<20 {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		m := parseMetadata(b)
		if m.ID != "" {
			out = append(out, m)
		}
	}
	return out
}

func routeArgs(rpc, selector string) []string {
	args := []string{"-p", rpc, "-o", "json"}
	if selector != "" {
		flag := "--instance-name"
		if uuid.MatchString(selector) {
			flag = "--instance-id"
		}
		args = append(args, flag, selector)
	}
	return append(args, "route")
}

// decodeResults handles both older flat route arrays and newer per-instance
// result envelopes. Never combine networks with overlapping virtual IPs.
func decodeResults(data []byte) ([]Candidate, error) {
	var items []json.RawMessage
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, errors.New("unsupported route JSON; use a matching EasyTier CLI")
	}
	if len(items) == 0 {
		return nil, errors.New("no running network instances")
	}
	var first struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(items[0], &first); err != nil {
		return nil, err
	}
	if first.Result == nil {
		s, err := Parse(data)
		if err != nil {
			return nil, err
		}
		return []Candidate{{status: s}}, nil
	}
	var out []Candidate
	for _, raw := range items {
		var item struct {
			ID     string          `json:"instance_id"`
			Name   string          `json:"instance_name"`
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, err
		}
		s, err := Parse(item.Result)
		if err != nil {
			continue
		}
		out = append(out, Candidate{Instance: item.ID, Name: item.Name, status: s})
	}
	if len(out) == 0 {
		return nil, errors.New("no instance has a usable virtual IPv4 route")
	}
	return out, nil
}

// Probe reads only local service metadata and route tables. Pairing keys are
// not required for diagnostics and are never passed to the CLI.
func Probe(ctx context.Context, c Config) Discovery {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	host := hostInventory(ctx)
	addrs, _ := net.InterfaceAddrs()
	local := map[netip.Addr]bool{}
	for _, a := range addrs {
		if ip, err := address(a.String()); err == nil {
			local[ip] = true
		}
	}
	return discover(ctx, c, host, runCLI, local)
}

func discover(ctx context.Context, c Config, host inventory, run runner, local map[netip.Addr]bool) Discovery {
	report := Discovery{Candidates: []Candidate{}, Errors: []string{}}
	report.CLI = findCLI(c.CLI, host.Dirs)
	if report.CLI == "" {
		report.Error = "EasyTier CLI not found; place easytier-cli next to the app or choose its path"
		return report
	}
	if v, err := run(ctx, report.CLI, "--version"); err == nil {
		report.Version = strings.TrimSpace(string(v))
	}
	var targets []endpoint
	if c.RPC != "" {
		rpc, err := normalizeRPC(c.RPC, false)
		if err != nil {
			report.Error = err.Error()
			return report
		}
		targets = append(targets, endpoint{RPC: rpc, Source: "configured"})
		for _, ep := range host.Endpoints {
			if ep.RPC == rpc {
				targets[0].Dir = ep.Dir
				break
			}
		}
	}
	targets = append(targets, host.Endpoints...)
	targets = append(targets, endpoint{RPC: "127.0.0.1:15999", Source: "GUI default"}, endpoint{RPC: "127.0.0.1:15888", Source: "core default"})
	seen := map[string]bool{}
	for _, ep := range targets {
		// Listener enumeration can include EasyTier's P2P transport ports.
		// Once a management endpoint works, do not probe those transports.
		if len(report.Candidates) > 0 && ep.Source == "EasyTier listener" {
			continue
		}
		if len(report.Candidates) > 0 && (ep.Source == "GUI default" || ep.Source == "core default") {
			break
		}
		if ctx.Err() != nil {
			report.Errors = append(report.Errors, "Discovery deadline reached")
			break
		}
		rpc, err := normalizeRPC(ep.RPC, true)
		if err != nil || seen[rpc] {
			continue
		}
		seen[rpc] = true
		if len(seen) > 8 {
			break
		}
		selections := []instance{{Name: strings.TrimSpace(c.Instance)}}
		if c.Instance == "" {
			if known := instances(ep.Dir); len(known) > 0 {
				selections = known
			}
		}
		before := len(report.Candidates)
		for _, sel := range selections {
			selector := sel.Name
			if sel.ID != "" {
				selector = sel.ID
			}
			data, err := run(ctx, report.CLI, routeArgs(rpc, selector)...)
			if err != nil {
				report.Errors = append(report.Errors, fmt.Sprintf("%s [%s]: %s", rpc, selector, safeError(err.Error())))
				continue
			}
			found, err := decodeResults(data)
			if err != nil {
				report.Errors = append(report.Errors, rpc+": "+err.Error())
				continue
			}
			for _, item := range found {
				ip := item.status.Self.IPs[0]
				if !local[ip] {
					report.Errors = append(report.Errors, rpc+": virtual IPv4 is not assigned locally; enable TUN")
					continue
				}
				item.RPC, item.Source = rpc, ep.Source
				if item.Instance == "" {
					item.Instance = selector
				}
				if item.Name == "" {
					item.Name = sel.Name
				}
				if item.Name == "" {
					item.Name = item.status.Self.HostName
				}
				item.IP, item.Host, item.Peers = ip.String(), item.status.Self.HostName, len(item.status.Peers)
				item.status.Tailnet = item.Name
				report.Candidates = append(report.Candidates, item)
			}
		}
		// A usable explicitly selected endpoint wins, preserving existing setups.
		if ep.Source == "configured" && len(report.Candidates) > before {
			break
		}
		// Defaults only run when detected running services did not answer.
		if len(report.Candidates) > 0 && (ep.Source == "GUI default" || ep.Source == "core default") {
			break
		}
	}
	sort.SliceStable(report.Candidates, func(i, j int) bool { return report.Candidates[i].Name < report.Candidates[j].Name })
	if len(report.Candidates) == 0 {
		report.Error = "EasyTier discovery failed"
		if len(report.Errors) > 0 {
			report.Error += "\n" + strings.Join(report.Errors, "\n")
		}
		if host.Warning != "" {
			report.Error += "\n" + host.Warning
		}
	}
	if len(report.Candidates) > 1 {
		report.Error = "Multiple EasyTier networks found; use Detect EasyTier in Settings to select one"
	}
	return report
}

func Get(ctx context.Context, c Config) (*tailnet.Status, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	report := Probe(ctx, c)
	if report.Error != "" {
		return nil, errors.New(report.Error)
	}
	s := report.Candidates[0].status
	s.Secret = c.Secret
	return s, nil
}
