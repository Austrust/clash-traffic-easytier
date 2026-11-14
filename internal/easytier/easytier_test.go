package easytier

import (
	"testing"
)

const routes = `[
 {"ipv4":"10.144.0.1/24","hostname":"local","next_hop_hostname":"Local","path_len":0},
 {"ipv4":"10.144.0.2","hostname":"direct","next_hop_hostname":"DIRECT","path_len":1},
 {"ipv4":"10.144.0.3","hostname":"relayed","next_hop_hostname":"direct","path_len":2},
 {"ipv4":"10.144.0.3","hostname":"duplicate","path_len":2},
 {"ipv4":"","hostname":"public-relay","path_len":1},
 {"ipv4":"127.0.0.1","hostname":"invalid","path_len":1}
]`

func TestParseRoutedPeers(t *testing.T) {
	s, err := Parse([]byte(routes))
	if err != nil {
		t.Fatal(err)
	}
	if s.Self.IPs[0].String() != "10.144.0.1" || len(s.Peers) != 2 {
		t.Fatalf("unexpected discovery: %+v", s)
	}
	if s.Peers[1].HostName != "relayed" || !s.Own(s.Peers[1]) {
		t.Fatal("relayed peer missing")
	}
	if s.Own(s.Self) {
		t.Fatal("unmarked device trusted")
	}
}

func TestRejectAmbiguousAndInvalidRoutes(t *testing.T) {
	for _, input := range []string{
		`[]`, `{}`, `not JSON`,
		`[{"instance_name":"one","result":[]},{"instance_name":"two","result":[]}]`,
		`[{"ipv4":"0.0.0.0","next_hop_hostname":"Local","path_len":0}]`,
		`[{"ipv4":"10.0.0.1","next_hop_hostname":"Local","path_len":0},{"ipv4":"10.0.1.1","next_hop_hostname":"Local","path_len":0}]`,
	} {
		if _, err := Parse([]byte(input)); err == nil {
			t.Fatalf("accepted invalid input %s", input)
		}
	}
}

func TestValidate(t *testing.T) {
	for _, rpc := range []string{"", "127.0.0.1:15888", "[::1]:15888"} {
		if err := (Config{RPC: rpc, Secret: "a-long-pairing-key"}).Validate(); err != nil {
			t.Fatal(err)
		}
	}
	for _, cfg := range []Config{
		{RPC: "10.0.0.1:15888", Secret: "a-long-pairing-key"},
		{RPC: "127.0.0.1:0", Secret: "a-long-pairing-key"},
		{Secret: "short"}, {Secret: "long-but-invalid\r\nkey"},
	} {
		if cfg.Validate() == nil {
			t.Fatalf("accepted invalid config")
		}
	}
}
