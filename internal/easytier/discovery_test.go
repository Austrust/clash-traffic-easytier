package easytier

import (
	"context"
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServiceDiscoveryAndPerInstanceProbe(t *testing.T) {
	dir := t.TempDir()
	id := "47a5adc0-46cc-4c74-b6a0-931c5ba8ca57"
	os.WriteFile(filepath.Join(dir, "network.toml"), []byte("instance_name = 'office'\ninstance_id = '"+id+"'\nnetwork_secret = 'do-not-read'"), 0600)
	var calls []string
	run := func(ctx context.Context, cli string, args ...string) ([]byte, error) {
		s := strings.Join(args, " ")
		calls = append(calls, s)
		if s == "--version" {
			return []byte("easytier-cli 2.6.4"), nil
		}
		if !strings.Contains(s, "127.0.0.1:15999") || !strings.Contains(s, "--instance-id "+id) {
			return nil, errors.New("wrong endpoint or instance")
		}
		return []byte(routes), nil
	}
	report := discover(context.Background(), Config{CLI: "test-cli"}, inventory{Endpoints: []endpoint{{RPC: "127.0.0.1:15999", Dir: dir, Source: "Windows service"}}}, run, map[netip.Addr]bool{netip.MustParseAddr("10.144.0.1"): true})
	if report.Error != "" || len(report.Candidates) != 1 || report.Candidates[0].Name != "office" {
		t.Fatalf("%+v", report)
	}
	if len(calls) != 2 {
		t.Fatalf("unexpected probing: %v", calls)
	}
}

func TestWrongManualPortFallsBackAndPreservesError(t *testing.T) {
	run := func(ctx context.Context, cli string, args ...string) ([]byte, error) {
		if len(args) == 1 {
			return []byte("v2.5"), nil
		}
		if args[1] == "127.0.0.1:15999" {
			return []byte(routes), nil
		}
		return nil, errors.New("exit status 1: connection refused")
	}
	r := discover(context.Background(), Config{CLI: "test", RPC: "127.0.0.1:15888"}, inventory{}, run, map[netip.Addr]bool{netip.MustParseAddr("10.144.0.1"): true})
	if r.Error != "" || len(r.Candidates) != 1 || r.Candidates[0].RPC != "127.0.0.1:15999" {
		t.Fatalf("%+v", r)
	}
	if len(r.Errors) != 1 || !strings.Contains(r.Errors[0], "connection refused") {
		t.Fatalf("lost diagnostic: %v", r.Errors)
	}
}

func TestDecodeMultiInstanceAndRequireSelection(t *testing.T) {
	wrapped := `[{"instance_id":"one","instance_name":"Office","result":` + routes + `},{"instance_id":"two","instance_name":"Home","result":` + routes + `}]`
	run := func(ctx context.Context, cli string, args ...string) ([]byte, error) {
		if len(args) == 1 {
			return []byte("2.7.0"), nil
		}
		return []byte(wrapped), nil
	}
	r := discover(context.Background(), Config{CLI: "test"}, inventory{}, run, map[netip.Addr]bool{netip.MustParseAddr("10.144.0.1"): true})
	if len(r.Candidates) != 2 || !strings.Contains(r.Error, "Multiple") {
		t.Fatalf("%+v", r)
	}
	if r.Candidates[0].Name != "Home" || r.Candidates[1].Name != "Office" {
		t.Fatal("instance names lost")
	}
}

func TestNormalizeAndRedact(t *testing.T) {
	for _, s := range []string{"tcp://127.0.0.1:15999", "localhost:15999", "15999", "0.0.0.0:15999"} {
		if got, err := normalizeRPC(s, true); err != nil || got != "127.0.0.1:15999" {
			t.Fatalf("%s: %s %v", s, got, err)
		}
	}
	if _, err := normalizeRPC("192.168.0.1:15999", true); err == nil {
		t.Fatal("remote RPC accepted")
	}
	if _, err := normalizeRPC("0.0.0.0:15999", false); err == nil {
		t.Fatal("manual wildcard accepted")
	}
	text := safeError("connection refused\nnetwork_secret=hidden\nhttps://example.test/api?token=hidden")
	if strings.Contains(text, "hidden") || !strings.Contains(text, "connection refused") {
		t.Fatal(text)
	}
}

// Opt-in integration probe is read-only: it queries the user's already running
// local service, never starts listeners or reads Clash Traffic credentials.
func TestLocalServiceDiscovery(t *testing.T) {
	cli := os.Getenv("EASYTIER_TEST_CLI")
	if cli == "" {
		t.Skip("set EASYTIER_TEST_CLI for a read-only local integration probe")
	}
	r := Probe(context.Background(), Config{CLI: cli})
	if r.Error != "" {
		t.Fatal(r.Error)
	}
	if len(r.Candidates) != 1 {
		t.Fatalf("found %d networks", len(r.Candidates))
	}
	t.Logf("CLI %s; source %s; RPC %s; %d routed peers", r.Version, r.Candidates[0].Source, r.Candidates[0].RPC, r.Candidates[0].Peers)
}
