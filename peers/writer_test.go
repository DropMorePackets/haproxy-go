package peers

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/dropmorepackets/haproxy-go/peers/sticktable"
	"github.com/google/go-cmp/cmp"
)

// helperDialPeer performs the client-side handshake to connect to a Peer server.
// HAProxy would normally do this. In tests, we simulate HAProxy connecting to us.
func helperDialPeer(t *testing.T, addr, localPeer, remotePeer string) net.Conn {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dialing peer: %v", err)
	}

	h := NewHandshake(remotePeer)
	h.LocalPeerIdentifier = localPeer
	if _, err = h.WriteTo(conn); err != nil {
		conn.Close()
		t.Fatalf("writing handshake: %v", err)
	}

	br := bufio.NewReader(conn)
	line, err := br.ReadString('\n')
	if err != nil {
		conn.Close()
		t.Fatalf("reading handshake status: %v", err)
	}

	var status int
	if _, err = fmt.Sscanf(line, "%d\n", &status); err != nil {
		conn.Close()
		t.Fatalf("parsing status %q: %v", line, err)
	}

	if HandshakeStatus(status) != HandshakeStatusHandshakeSucceeded {
		conn.Close()
		t.Fatalf("handshake failed with status %d", status)
	}

	return conn
}

func TestWriterSendEntry(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	writerReady := make(chan *Writer, 1)
	peer := &Peer{
		BaseContext: ctx,
		HandlerSource: func() Handler {
			return &testHandler{
				onHandshake: func(ctx context.Context, h *Handshake) {
					writerReady <- WriterFromContext(ctx)
				},
			}
		},
	}
	go peer.Serve(l)

	conn := helperDialPeer(t, l.Addr().String(), "haproxy_peer", "go_peer")
	defer conn.Close()

	var w *Writer
	select {
	case w = <-writerReady:
	case <-ctx.Done():
		t.Fatal("timeout waiting for writer")
	}

	tableDef := &sticktable.Definition{
		StickTableID: 0,
		Name:         "test_table",
		KeyType:      sticktable.KeyTypeString,
		KeyLength:    50,
		DataTypes: []sticktable.DataTypeDefinition{
			{DataType: sticktable.DataTypeGPC0},
		},
		Expiry: 600000,
	}

	if err := w.SendTableDefinition(tableDef); err != nil {
		t.Fatal(err)
	}

	key := sticktable.StringKey("192.168.1.1")
	gpc0 := sticktable.UnsignedIntegerData(42)
	entry := &sticktable.EntryUpdate{
		StickTable: tableDef,
		Key:        &key,
		Data:       []sticktable.MapData{&gpc0},
	}

	if err := w.SendEntry(entry); err != nil {
		t.Fatal(err)
	}
}

func TestWriterSendMultipleEntries(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	writerReady := make(chan *Writer, 1)
	peer := &Peer{
		BaseContext: ctx,
		HandlerSource: func() Handler {
			return &testHandler{
				onHandshake: func(ctx context.Context, h *Handshake) {
					writerReady <- WriterFromContext(ctx)
				},
			}
		},
	}
	go peer.Serve(l)

	conn := helperDialPeer(t, l.Addr().String(), "haproxy_peer", "go_peer")
	defer conn.Close()

	var w *Writer
	select {
	case w = <-writerReady:
	case <-ctx.Done():
		t.Fatal("timeout waiting for writer")
	}

	tableDef := &sticktable.Definition{
		StickTableID: 0,
		Name:         "multi_table",
		KeyType:      sticktable.KeyTypeString,
		KeyLength:    50,
		DataTypes: []sticktable.DataTypeDefinition{
			{DataType: sticktable.DataTypeConnectionsCounter},
			{DataType: sticktable.DataTypeBytesInCounter},
		},
		Expiry: 600000,
	}

	if err := w.SendTableDefinition(tableDef); err != nil {
		t.Fatal(err)
	}

	const numEntries = 10
	for i := 0; i < numEntries; i++ {
		key := sticktable.StringKey(fmt.Sprintf("key_%d", i))
		connCnt := sticktable.UnsignedIntegerData(uint32(i * 10))
		bytesIn := sticktable.UnsignedLongLongData(uint64(i * 1000))
		entry := &sticktable.EntryUpdate{
			StickTable: tableDef,
			Key:        &key,
			Data:       []sticktable.MapData{&connCnt, &bytesIn},
		}

		if err := w.SendEntry(entry); err != nil {
			t.Fatalf("sending entry %d: %v", i, err)
		}
	}

	if w.nextUpdateID != numEntries {
		t.Errorf("expected nextUpdateID %d, got %d", numEntries, w.nextUpdateID)
	}
}

func TestWriterSendTimedEntry(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	writerReady := make(chan *Writer, 1)
	peer := &Peer{
		BaseContext: ctx,
		HandlerSource: func() Handler {
			return &testHandler{
				onHandshake: func(ctx context.Context, h *Handshake) {
					writerReady <- WriterFromContext(ctx)
				},
			}
		},
	}
	go peer.Serve(l)

	conn := helperDialPeer(t, l.Addr().String(), "haproxy_peer", "go_peer")
	defer conn.Close()

	var w *Writer
	select {
	case w = <-writerReady:
	case <-ctx.Done():
		t.Fatal("timeout waiting for writer")
	}

	tableDef := &sticktable.Definition{
		StickTableID: 0,
		Name:         "timed_table",
		KeyType:      sticktable.KeyTypeIPv4Address,
		KeyLength:    4,
		DataTypes: []sticktable.DataTypeDefinition{
			{DataType: sticktable.DataTypeSessionsCounter},
		},
		Expiry: 300000,
	}

	if err := w.SendTableDefinition(tableDef); err != nil {
		t.Fatal(err)
	}

	key := sticktable.IPv4AddressKey(netip.MustParseAddr("10.0.0.1"))
	sessCnt := sticktable.UnsignedIntegerData(99)
	entry := &sticktable.EntryUpdate{
		StickTable: tableDef,
		Key:        &key,
		Data:       []sticktable.MapData{&sessCnt},
		WithExpiry: true,
		Expiry:     60000,
	}

	if err := w.SendEntry(entry); err != nil {
		t.Fatal(err)
	}
}

// TestWriterRoundTrip verifies that data written by the Writer can be read
// and decoded correctly by the protocol client's message handler (full loop).
func TestWriterRoundTrip(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	updates := make(chan *sticktable.EntryUpdate, 10)

	peerB := &Peer{
		BaseContext: ctx,
		Handler: HandlerFunc(func(_ context.Context, u *sticktable.EntryUpdate) {
			updates <- u
		}),
	}
	go peerB.Serve(l)

	conn := helperDialPeer(t, l.Addr().String(), "peer_a", "peer_b")
	defer conn.Close()

	w := newWriter(conn, &sync.Mutex{})

	tableDef := &sticktable.Definition{
		StickTableID: 0,
		Name:         "roundtrip_table",
		KeyType:      sticktable.KeyTypeString,
		KeyLength:    50,
		DataTypes: []sticktable.DataTypeDefinition{
			{DataType: sticktable.DataTypeGPC0},
			{DataType: sticktable.DataTypeHttpRequestsRate, Counter: 1, Period: 10000},
		},
		Expiry: 600000,
	}

	if err := w.SendTableDefinition(tableDef); err != nil {
		t.Fatal(err)
	}

	key := sticktable.StringKey("test_key")
	gpc0 := sticktable.UnsignedIntegerData(42)
	rate := sticktable.FreqData{
		CurrentTick:   500,
		CurrentPeriod: 10,
		LastPeriod:    8,
	}
	entry := &sticktable.EntryUpdate{
		StickTable: tableDef,
		Key:        &key,
		Data:       []sticktable.MapData{&gpc0, &rate},
	}

	if err := w.SendEntry(entry); err != nil {
		t.Fatal(err)
	}

	select {
	case u := <-updates:
		if u.StickTable.Name != "roundtrip_table" {
			t.Errorf("expected table name %q, got %q", "roundtrip_table", u.StickTable.Name)
		}
		if u.Key.String() != "test_key" {
			t.Errorf("expected key %q, got %q", "test_key", u.Key.String())
		}
		if u.LocalUpdateID != 0 {
			t.Errorf("expected update ID 0, got %d", u.LocalUpdateID)
		}

		gotGPC0 := u.Data[0].(*sticktable.UnsignedIntegerData)
		if *gotGPC0 != 42 {
			t.Errorf("expected gpc0 value 42, got %d", *gotGPC0)
		}

		wantRate := &sticktable.FreqData{
			CurrentTick:   500,
			CurrentPeriod: 10,
			LastPeriod:    8,
		}
		gotRate := u.Data[1].(*sticktable.FreqData)
		if diff := cmp.Diff(wantRate, gotRate); diff != "" {
			t.Errorf("FreqData mismatch (-want +got):\n%s", diff)
		}
	case <-ctx.Done():
		t.Fatal("timeout waiting for roundtrip update")
	}
}

// testHandler is a Handler implementation for testing that allows
// overriding individual methods.
type testHandler struct {
	onUpdate    func(context.Context, *sticktable.EntryUpdate)
	onHandshake func(context.Context, *Handshake)
}

func (h *testHandler) HandleUpdate(ctx context.Context, u *sticktable.EntryUpdate) {
	if h.onUpdate != nil {
		h.onUpdate(ctx, u)
	}
}

func (h *testHandler) HandleHandshake(ctx context.Context, hs *Handshake) {
	if h.onHandshake != nil {
		h.onHandshake(ctx, hs)
	}
}

func (h *testHandler) Close() error { return nil }
