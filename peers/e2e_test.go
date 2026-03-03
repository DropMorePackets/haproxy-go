//go:build e2e

package peers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/dropmorepackets/haproxy-go/peers/sticktable"
	"github.com/dropmorepackets/haproxy-go/pkg/testutil"
)

func TestE2E(t *testing.T) {
	success := make(chan bool)
	a := Peer{Handler: HandlerFunc(func(_ context.Context, u *sticktable.EntryUpdate) {
		log.Println(u)
		success <- true
	})}

	// create the listener synchronously to prevent a race
	l := testutil.TCPListener(t)
	// ignore errors as the listener will be closed by t.Cleanup
	go a.Serve(l)

	cfg := testutil.HAProxyConfig{
		FrontendPort: fmt.Sprintf("%d", testutil.TCPPort(t)),
		CustomFrontendConfig: `
	http-request track-sc0 src table st_src_global
    http-request track-sc2 req.hdr(Host) table st_be_name
`,
		CustomConfig: `
backend st_be_name
    stick-table type string size 1m expire 10m  store http_req_rate(10s) peers mypeers

backend st_src_global
	stick-table type ip size 1m expire 10m store http_req_rate(10s),conn_rate(10s),bytes_in_rate(10s) peers mypeers
`,
		PeerAddr: l.Addr().String(),
	}

	t.Run("receive update", func(t *testing.T) {
		cfg.Run(t)

		for i := 0; i < 10; i++ {
			_, err := http.Get("http://127.0.0.1:" + cfg.FrontendPort)
			if err != nil {
				t.Fatal(err)
			}
		}

		tm := time.NewTimer(1 * time.Second)
		defer tm.Stop()
		select {
		case v := <-success:
			if !v {
				t.Fail()
			}
		case <-tm.C:
			t.Error("timeout")
		}
	})
}

func TestE2EWriter(t *testing.T) {
	writerCh := make(chan *Writer, 1)
	a := Peer{HandlerSource: func() Handler {
		return &writerE2EHandler{writerCh: writerCh}
	}}

	l := testutil.TCPListener(t)
	go a.Serve(l)

	cfg := testutil.HAProxyConfig{
		FrontendPort: fmt.Sprintf("%d", testutil.TCPPort(t)),
		CustomFrontendConfig: `
	http-request track-sc0 src table st_blocklist
	http-request deny deny_status 403 if { sc0_get_gpc0 gt 0 }
`,
		CustomConfig: `
backend st_blocklist
	stick-table type ip size 200k expire 5m store gpc0 peers mypeers
`,
		PeerAddr: l.Addr().String(),
	}

	t.Run("push entry blocks request", func(t *testing.T) {
		cfg.Run(t)

		var w *Writer
		select {
		case w = <-writerCh:
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for HAProxy peer connection")
		}

		time.Sleep(500 * time.Millisecond)

		resp, err := http.Get("http://127.0.0.1:" + cfg.FrontendPort)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 before push, got %d", resp.StatusCode)
		}

		tableDef := &sticktable.Definition{
			StickTableID: 0,
			Name:         "st_blocklist",
			KeyType:      sticktable.KeyTypeIPv4Address,
			KeyLength:    4,
			DataTypes: []sticktable.DataTypeDefinition{
				{DataType: sticktable.DataTypeGPC0},
			},
			Expiry: 300000,
		}

		if err := w.SendTableDefinition(tableDef); err != nil {
			t.Fatal(err)
		}

		key := sticktable.IPv4AddressKey(netip.MustParseAddr("127.0.0.1"))
		gpc0 := sticktable.UnsignedIntegerData(1)
		entry := &sticktable.EntryUpdate{
			StickTable: tableDef,
			Key:        &key,
			Data:       []sticktable.MapData{&gpc0},
		}
		if err := w.SendEntry(entry); err != nil {
			t.Fatal(err)
		}

		time.Sleep(1 * time.Second)

		resp, err = http.Get("http://127.0.0.1:" + cfg.FrontendPort)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("expected 403 after push, got %d", resp.StatusCode)
		}
	})
}

type writerE2EHandler struct {
	writerCh chan *Writer
	once     sync.Once
}

func (h *writerE2EHandler) HandleUpdate(_ context.Context, u *sticktable.EntryUpdate) {
	log.Println(u)
}

func (h *writerE2EHandler) HandleHandshake(ctx context.Context, _ *Handshake) {
	h.once.Do(func() {
		h.writerCh <- WriterFromContext(ctx)
	})
}

func (h *writerE2EHandler) Close() error { return nil }

func TestE2EWriterTimedEntry(t *testing.T) {
	writerCh := make(chan *Writer, 1)
	a := Peer{HandlerSource: func() Handler {
		return &writerE2EHandler{writerCh: writerCh}
	}}

	l := testutil.TCPListener(t)
	go a.Serve(l)

	cfg := testutil.HAProxyConfig{
		FrontendPort: fmt.Sprintf("%d", testutil.TCPPort(t)),
		CustomConfig: `
backend st_timed
	stick-table type ip size 200k expire 5m peers mypeers
`,
		BackendConfig: `
	http-request set-var(txn.lookup_ip) str(127.0.0.2)
	http-request return status 200 content-type "text/plain" hdr X-Expire %[var(txn.lookup_ip),table_expire(st_timed)] string "OK\n"
`,
		PeerAddr: l.Addr().String(),
	}

	t.Run("push timed entry with 60s expiry", func(t *testing.T) {
		cfg.Run(t)

		var w *Writer
		select {
		case w = <-writerCh:
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for HAProxy peer connection")
		}

		time.Sleep(500 * time.Millisecond)

		tableDef := &sticktable.Definition{
			StickTableID: 0,
			Name:         "st_timed",
			KeyType:      sticktable.KeyTypeIPv4Address,
			KeyLength:    4,
			Expiry:       300000,
		}

		if err := w.SendTableDefinition(tableDef); err != nil {
			t.Fatal(err)
		}

		key := sticktable.IPv4AddressKey(netip.MustParseAddr("127.0.0.2"))
		entry := &sticktable.EntryUpdate{
			StickTable: tableDef,
			Key:        &key,
			WithExpiry: true,
			Expiry:     60000,
		}
		if err := w.SendEntry(entry); err != nil {
			t.Fatal(err)
		}

		time.Sleep(1 * time.Second)

		resp, err := http.Get("http://127.0.0.1:" + cfg.FrontendPort)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()

		xexpire := resp.Header.Get("X-Expire")
		t.Logf("X-Expire: %s", xexpire)

		if xexpire == "" || xexpire == "0" {
			t.Errorf("expected non-zero X-Expire header, got %q", xexpire)
		}
	})
}

func TestE2EWriterBulkEntries(t *testing.T) {
	writerCh := make(chan *Writer, 1)
	a := Peer{HandlerSource: func() Handler {
		return &writerE2EHandler{writerCh: writerCh}
	}}

	l := testutil.TCPListener(t)
	go a.Serve(l)

	cfg := testutil.HAProxyConfig{
		FrontendPort: fmt.Sprintf("%d", testutil.TCPPort(t)),
		CustomConfig: `
backend st_bulk
	stick-table type ip size 200k expire 5m peers mypeers
`,
		BackendConfig: `
	http-request return status 200 content-type "text/plain" hdr X-Count %[table_cnt(st_bulk)] string "OK\n"
`,
		PeerAddr: l.Addr().String(),
	}

	t.Run("push 20 entries", func(t *testing.T) {
		cfg.Run(t)

		var w *Writer
		select {
		case w = <-writerCh:
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for HAProxy peer connection")
		}

		time.Sleep(500 * time.Millisecond)

		tableDef := &sticktable.Definition{
			StickTableID: 0,
			Name:         "st_bulk",
			KeyType:      sticktable.KeyTypeIPv4Address,
			KeyLength:    4,
			Expiry:       300000,
		}

		if err := w.SendTableDefinition(tableDef); err != nil {
			t.Fatal(err)
		}

		for i := 0; i < 20; i++ {
			ip := netip.AddrFrom4([4]byte{10, 0, 0, byte(i + 1)})
			key := sticktable.IPv4AddressKey(ip)
			entry := &sticktable.EntryUpdate{
				StickTable: tableDef,
				Key:        &key,
				WithExpiry: true,
				Expiry:     60000,
			}
			if err := w.SendEntry(entry); err != nil {
				t.Fatalf("sending entry %d (%s): %v", i, ip, err)
			}
		}

		time.Sleep(1 * time.Second)

		resp, err := http.Get("http://127.0.0.1:" + cfg.FrontendPort)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()

		xcount := resp.Header.Get("X-Count")
		t.Logf("X-Count: %s", xcount)

		if xcount != "20" {
			t.Errorf("expected X-Count=20, got %q", xcount)
		}
	})
}
