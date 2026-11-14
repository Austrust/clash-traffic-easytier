package mesh

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"clashtraffic/internal/store"
	"clashtraffic/internal/tailnet"
)

func easyNetwork(key string, devs ...*device) {
	tailnetOf(devs...)
	for _, d := range devs {
		original := d.node.TailnetStatus
		d.node.Provider = func() string { return "EasyTier" }
		d.node.TailnetStatus = func(ctx context.Context) (*tailnet.Status, error) {
			s, err := original(ctx)
			s.Provider, s.Secret = "EasyTier", key
			peers := s.Peers[:0]
			for _, p := range s.Peers {
				if !p.Desktop() {
					continue
				}
				p.Trusted = true
				p.OS = "" // EasyTier route discovery does not report OS.
				peers = append(peers, p)
			}
			s.Peers = peers
			return s, err
		}
	}
}

func TestEasyTierReusesIncrementalSync(t *testing.T) {
	a := newDevice(t, macID, "10.144.0.1")
	b := newDevice(t, winID, "10.144.0.2")
	easyNetwork("shared-test-key-123456", a, b)
	h := time.Now().Unix() / 3600 * 3600
	b.store.Add(store.Key{Hour: h, App: "Browser", Host: "example.org"}, "", store.Counter{Down: 123})
	if err := b.store.Flush(); err != nil {
		t.Fatal(err)
	}
	b.node.scan(context.Background())
	a.node.scan(context.Background())
	s := a.node.Status()
	if s.Provider != "EasyTier" || len(s.Peers) != 1 || s.Peers[0].State != PeerApp {
		t.Fatalf("discovery failed: %+v", s)
	}
	q := store.Query{From: h * 1000, To: (h + 3600) * 1000}
	if r := a.store.Query(q); r.Down != 123 {
		t.Fatalf("got %d", r.Down)
	}
	a.node.scan(context.Background())
	if a.node.Status().Peers[0].Pulled != 0 {
		t.Fatal("unchanged data fetched again")
	}
	// Rotation rejects the old key without losing previously collected data.
	old := b.node.TailnetStatus
	b.node.TailnetStatus = func(ctx context.Context) (*tailnet.Status, error) {
		s, e := old(ctx)
		s.Secret = "different-test-key-123"
		return s, e
	}
	b.node.scan(context.Background())
	a.node.scan(context.Background())
	if a.node.Status().Peers[0].State == PeerApp {
		t.Fatal("mismatched key accepted")
	}
	if r := a.store.Query(q); r.Down != 123 {
		t.Fatal("history changed on authentication failure")
	}
}

func TestEasyTierRequiresKeyAndSource(t *testing.T) {
	d := newDevice(t, macID, "10.144.0.1")
	easyNetwork("shared-test-key-123456", d)
	d.node.scan(context.Background())
	for _, token := range []string{"", "wrong", "shared-test-key-123456"} {
		req, _ := http.NewRequest(http.MethodGet, d.srv.URL+PathHello, nil)
		if token != "" {
			if err := signRequest(token, req); err != nil {
				t.Fatal(err)
			}
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		want := http.StatusForbidden
		if token == "shared-test-key-123456" {
			want = http.StatusOK
		}
		if resp.StatusCode != want {
			t.Fatalf("got %d want %d", resp.StatusCode, want)
		}
	}
	d.node.mu.Lock()
	d.node.allowed = nil
	d.node.mu.Unlock()
	if _, err := d.node.get(context.Background(), d.srv.URL+PathHello, 4096); err == nil {
		t.Fatal("key bypassed source allowlist")
	}
	d.node.TailnetStatus = func(context.Context) (*tailnet.Status, error) { return nil, errors.New("disconnected") }
	d.node.scan(context.Background())
	if s := d.node.Status(); s.Available || s.Serving || s.Provider != "EasyTier" {
		t.Fatalf("bad disconnect state %+v", s)
	}
}

func TestMeshDoesNotFollowRedirects(t *testing.T) {
	n := New(t.TempDir(), macID, 47399, "")
	n.secret = "shared-test-key-123456"
	called := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, http.StatusFound) }))
	defer source.Close()
	if _, err := n.get(context.Background(), source.URL, 1024); err == nil || called {
		t.Fatal("followed redirect carrying sync key")
	}
}

func TestEasyTierRejectsUnsignedResponses(t *testing.T) {
	n := New(t.TempDir(), macID, 47399, "")
	n.secret = "shared-test-key-123456"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("sent reusable credential")
		}
		w.Write([]byte(`{"app":"clash-traffic"}`))
	}))
	defer srv.Close()
	if _, err := n.get(context.Background(), srv.URL, 4096); err == nil {
		t.Fatal("accepted unsigned peer data")
	}
}

func TestEasyTierProofCannotBeForwardedToAnotherPeer(t *testing.T) {
	d := newDevice(t, macID, "10.144.0.1")
	easyNetwork("shared-test-key-123456", d)
	d.node.scan(context.Background())
	req, _ := http.NewRequest(http.MethodGet, d.srv.URL+PathHello, nil)
	// A hostile discovered device knows the proof addressed to itself, but
	// cannot change its signed destination to the real server's address.
	req.Host = "10.144.0.99:47321"
	if err := signRequest("shared-test-key-123456", req); err != nil {
		t.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatal("accepted proof for a different destination")
	}
	if len(d.node.trigger) != 0 {
		t.Fatal("invalid authentication triggered a scan loop")
	}
}
