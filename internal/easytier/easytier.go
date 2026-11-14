// Package easytier discovers virtual addresses from EasyTier's route table,
// including peers reached through a relay. It never reads the network password.
package easytier

import (
	"clashtraffic/internal/tailnet"
	"encoding/json"
	"errors"
	"net/netip"
	"strings"
)

type Config struct {
	CLI      string
	RPC      string
	Instance string
	Secret   string
}

// Validate keeps management requests on this computer and requires an explicit
// application-level pairing key. Do not reuse EasyTier's network password.
func (c Config) Validate() error {
	if len(c.Secret) < 16 || strings.ContainsAny(c.Secret, "\r\n") {
		return errors.New("EasyTier: use the same sync key of at least 16 characters on your devices")
	}
	if c.RPC != "" {
		if _, err := normalizeRPC(c.RPC, false); err != nil {
			return err
		}
	}
	return nil
}

type route struct {
	IP     string `json:"ipv4"`
	Host   string `json:"hostname"`
	Hop    string `json:"next_hop_hostname"`
	Length *int   `json:"path_len"`
}

func address(s string) (netip.Addr, error) {
	if p, err := netip.ParsePrefix(s); err == nil {
		s = p.Addr().String()
	}
	ip, err := netip.ParseAddr(s)
	if err != nil || !ip.Is4() || !ip.IsGlobalUnicast() || ip.IsLoopback() {
		return netip.Addr{}, errors.New("invalid virtual IPv4 address")
	}
	return ip, nil
}

// Parse accepts the documented `-o json route` array. Multiple local instances
// are deliberately not merged: the user selects one via Instance/RPC settings.
func Parse(data []byte) (*tailnet.Status, error) {
	var rows []route
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, errors.New("EasyTier: unsupported route JSON; use a CLI supporting -o json route")
	}
	st := &tailnet.Status{Provider: "EasyTier", State: "Running", Tailnet: "EasyTier"}
	seen := map[netip.Addr]bool{}
	for _, r := range rows {
		if r.Length == nil {
			return nil, errors.New("EasyTier: select one instance; route output has no path_len")
		}
		ip, err := address(r.IP)
		if *r.Length == 0 && r.Hop == "Local" {
			if err != nil {
				return nil, errors.New("EasyTier: no virtual IPv4; enable the EasyTier TUN interface")
			}
			if len(st.Self.IPs) != 0 {
				return nil, errors.New("EasyTier: multiple local routes; select one instance")
			}
			st.Self = tailnet.Node{HostName: r.Host, IPs: []netip.Addr{ip}, Online: true}
		} else if err == nil && *r.Length > 0 && !seen[ip] {
			st.Peers = append(st.Peers, tailnet.Node{HostName: r.Host, IPs: []netip.Addr{ip}, Online: true, Trusted: true})
			seen[ip] = true
		}
	}
	if len(st.Self.IPs) == 0 {
		return nil, errors.New("EasyTier: no local route; check the RPC endpoint and selected instance")
	}
	self := st.Self.IPs[0]
	peers := st.Peers[:0]
	for _, p := range st.Peers {
		if p.IPs[0] != self {
			peers = append(peers, p)
		}
	}
	st.Peers = peers
	return st, nil
}
