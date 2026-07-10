package spop

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/dropmorepackets/haproxy-go/pkg/encoding"
)

func TestProtocolNegotiatesMaxFrameSize(t *testing.T) {
	tests := []struct {
		name     string
		agentMax uint32
		peerMax  uint32
		want     uint32
	}{
		{name: "peer offer is larger", agentMax: DefaultMaxFrameSize, peerMax: 262140, want: DefaultMaxFrameSize},
		{name: "agent maximum is larger", agentMax: 262140, peerMax: DefaultMaxFrameSize, want: DefaultMaxFrameSize},
		{name: "limits are equal", agentMax: 262140, peerMax: 262140, want: 262140},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := newFramePool(tt.agentMax)
			var rw bytes.Buffer
			client := newProtocolClient(context.Background(), &rw, nil, pool, HandlerFunc(func(
				context.Context,
				*encoding.ActionWriter,
				*encoding.Message,
			) {
			}))

			frame := testHelloFrame(t, pool, &tt.peerMax)
			if err := client.onHAProxyHello(frame); err != nil {
				releaseFrame(frame)
				t.Fatalf("handle HAPROXY-HELLO: %v", err)
			}
			releaseFrame(frame)

			if client.maxFrameSize != tt.want {
				t.Fatalf("expected negotiated maximum %d, got %d", tt.want, client.maxFrameSize)
			}
			response := readTestWireFrame(t, &rw)
			if response.frameType != frameTypeIDAgentHello {
				t.Fatalf("expected AGENT-HELLO, got frame type %d", response.frameType)
			}
			if got := testHelloMaxFrameSize(t, response.payload); got != tt.want {
				t.Fatalf("expected advertised maximum %d, got %d", tt.want, got)
			}
		})
	}
}

func TestProtocolRejectsInvalidMaxFrameSize(t *testing.T) {
	tests := []struct {
		name    string
		peerMax *uint32
		want    errorCode
	}{
		{name: "missing", want: ErrorNoFrameSize},
		{name: "below protocol minimum", peerMax: uint32Ptr(minFrameSize - 1), want: ErrorBadFrameSize},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := newFramePool(DefaultMaxFrameSize)
			client := newProtocolClient(context.Background(), &bytes.Buffer{}, nil, pool, HandlerFunc(func(
				context.Context,
				*encoding.ActionWriter,
				*encoding.Message,
			) {
			}))
			frame := testHelloFrame(t, pool, tt.peerMax)
			defer releaseFrame(frame)

			err := client.onHAProxyHello(frame)
			if err == nil {
				t.Fatal("expected HAPROXY-HELLO to be rejected")
			}
			if got := protocolErrorCode(err); got != tt.want {
				t.Fatalf("expected protocol error code %d, got %d", tt.want, got)
			}
		})
	}
}

func TestProtocolRejectsHelloAboveAdvertisedMaxFrameSize(t *testing.T) {
	const peerMax uint32 = minFrameSize
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()
	if err := clientConn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}

	client := newProtocolClient(
		context.Background(),
		serverConn,
		nil,
		newFramePool(DefaultMaxFrameSize),
		HandlerFunc(func(context.Context, *encoding.ActionWriter, *encoding.Message) {}),
	)
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- client.Serve()
	}()

	payload := append(testHelloPayload(t, peerMax), bytes.Repeat([]byte{0}, int(peerMax))...)
	writeTestWireFrame(t, clientConn, frameTypeIDHaproxyHello, 0, 0, payload)
	disconnect := readTestWireFrame(t, clientConn)
	if disconnect.frameType != frameTypeIDAgentDisconnect {
		t.Fatalf("expected AGENT-DISCONNECT, got frame type %d", disconnect.frameType)
	}
	if code := testDisconnectCode(t, disconnect.payload); code != ErrorBadFrameSize {
		t.Fatalf("expected disconnect code %d, got %d", ErrorBadFrameSize, code)
	}

	select {
	case err := <-serveDone:
		if err == nil || !strings.Contains(err.Error(), "exceeds peer maximum 256") {
			t.Fatalf("expected peer maximum error, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("protocol did not reject the oversized HAPROXY-HELLO")
	}
}

func TestAgentLargeFrameRoundTrip(t *testing.T) {
	const maxFrameSize uint32 = 262140
	requestBody := bytes.Repeat([]byte{0xa5}, 90<<10)
	responseBody := bytes.Repeat([]byte{0x5a}, 90<<10)
	handlerErr := make(chan error, 1)

	handler := HandlerFunc(func(_ context.Context, writer *encoding.ActionWriter, message *encoding.Message) {
		if string(message.NameBytes()) != "large" {
			handlerErr <- fmt.Errorf("expected message %q, got %q", "large", message.NameBytes())
			return
		}

		entry := encoding.AcquireKVEntry()
		defer encoding.ReleaseKVEntry(entry)
		if !message.KV.Next(entry) {
			handlerErr <- fmt.Errorf("missing payload entry: %v", message.KV.Error())
			return
		}
		if !entry.NameEquals("payload") || !bytes.Equal(entry.ValueBytes(), requestBody) {
			handlerErr <- fmt.Errorf("received an unexpected payload")
			return
		}
		if err := writer.SetBinary(encoding.VarScopeTransaction, "result", responseBody); err != nil {
			handlerErr <- fmt.Errorf("write response action: %w", err)
			return
		}
		handlerErr <- nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	agentDone := make(chan error, 1)
	agent := Agent{
		BaseContext:  ctx,
		Handler:      handler,
		MaxFrameSize: maxFrameSize,
	}
	go func() {
		agentDone <- agent.Serve(listener)
	}()

	clientConn, err := net.DialTimeout("tcp", listener.Addr().String(), 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer clientConn.Close()
	if err := clientConn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}

	writeTestWireFrame(t, clientConn, frameTypeIDHaproxyHello, 0, 0, testHelloPayload(t, maxFrameSize))
	hello := readTestWireFrame(t, clientConn)
	if hello.frameType != frameTypeIDAgentHello {
		t.Fatalf("expected AGENT-HELLO, got frame type %d", hello.frameType)
	}
	if got := testHelloMaxFrameSize(t, hello.payload); got != maxFrameSize {
		t.Fatalf("expected negotiated maximum %d, got %d", maxFrameSize, got)
	}

	const streamID, frameID = 17, 23
	writeTestWireFrame(
		t,
		clientConn,
		frameTypeIDNotify,
		streamID,
		frameID,
		testNotifyPayload(t, "large", "payload", requestBody),
	)
	ack := readTestWireFrame(t, clientConn)
	if ack.frameType != frameTypeIDAck {
		t.Fatalf("expected ACK, got frame type %d", ack.frameType)
	}
	if ack.streamID != streamID || ack.frameID != frameID {
		t.Fatalf("expected stream/frame IDs %d/%d, got %d/%d", streamID, frameID, ack.streamID, ack.frameID)
	}
	if ack.length <= DefaultMaxFrameSize {
		t.Fatalf("expected ACK larger than the default maximum, got %d bytes", ack.length)
	}
	expectedAction := make([]byte, len(responseBody)+256)
	actionWriter := encoding.NewActionWriter(expectedAction, 0)
	if err := actionWriter.SetBinary(encoding.VarScopeTransaction, "result", responseBody); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(ack.payload, actionWriter.Bytes()) {
		t.Fatal("ACK contained an unexpected action payload")
	}
	if err := <-handlerErr; err != nil {
		t.Fatal(err)
	}

	if err := clientConn.Close(); err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case <-agentDone:
	case <-time.After(5 * time.Second):
		t.Fatal("agent did not stop after cancellation")
	}
}

func TestProtocolEnforcesNegotiatedMaxFrameSize(t *testing.T) {
	const (
		agentMax uint32 = 262140
		peerMax  uint32 = 4096
	)
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()
	if err := clientConn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}

	client := newProtocolClient(
		context.Background(),
		serverConn,
		nil,
		newFramePool(agentMax),
		HandlerFunc(func(context.Context, *encoding.ActionWriter, *encoding.Message) {}),
	)
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- client.Serve()
	}()

	writeTestWireFrame(t, clientConn, frameTypeIDHaproxyHello, 0, 0, testHelloPayload(t, peerMax))
	hello := readTestWireFrame(t, clientConn)
	if got := testHelloMaxFrameSize(t, hello.payload); got != peerMax {
		t.Fatalf("expected negotiated maximum %d, got %d", peerMax, got)
	}

	length := make([]byte, uint32Len)
	binary.BigEndian.PutUint32(length, peerMax+1)
	if _, err := clientConn.Write(length); err != nil {
		t.Fatalf("write oversized frame length: %v", err)
	}
	disconnect := readTestWireFrame(t, clientConn)
	if disconnect.frameType != frameTypeIDAgentDisconnect {
		t.Fatalf("expected AGENT-DISCONNECT, got frame type %d", disconnect.frameType)
	}
	if code := testDisconnectCode(t, disconnect.payload); code != ErrorTooBig {
		t.Fatalf("expected disconnect code %d, got %d", ErrorTooBig, code)
	}

	select {
	case err := <-serveDone:
		if err == nil || !strings.Contains(err.Error(), "exceeds maximum 4096") {
			t.Fatalf("expected negotiated maximum error, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("protocol did not reject the oversized frame")
	}
}

func TestProtocolClosesOnAckAboveNegotiatedMaxFrameSize(t *testing.T) {
	const (
		agentMax uint32 = 262140
		peerMax  uint32 = DefaultMaxFrameSize
	)
	responseBody := bytes.Repeat([]byte{0x5a}, 90<<10)
	handlerErr := make(chan error, 1)
	handler := HandlerFunc(func(_ context.Context, writer *encoding.ActionWriter, _ *encoding.Message) {
		handlerErr <- writer.SetBinary(encoding.VarScopeTransaction, "result", responseBody)
	})

	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()
	if err := clientConn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}

	client := newProtocolClient(
		context.Background(),
		serverConn,
		newTestAsyncScheduler(),
		newFramePool(agentMax),
		handler,
	)
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- client.Serve()
	}()

	writeTestWireFrame(t, clientConn, frameTypeIDHaproxyHello, 0, 0, testHelloPayload(t, peerMax))
	hello := readTestWireFrame(t, clientConn)
	if got := testHelloMaxFrameSize(t, hello.payload); got != peerMax {
		t.Fatalf("expected negotiated maximum %d, got %d", peerMax, got)
	}
	writeTestWireFrame(
		t,
		clientConn,
		frameTypeIDNotify,
		1,
		1,
		testNotifyPayload(t, "large", "payload", []byte("request")),
	)

	disconnect := readTestWireFrame(t, clientConn)
	if disconnect.frameType != frameTypeIDAgentDisconnect {
		t.Fatalf("expected AGENT-DISCONNECT, got frame type %d", disconnect.frameType)
	}
	if code := testDisconnectCode(t, disconnect.payload); code != ErrorTooBig {
		t.Fatalf("expected disconnect code %d, got %d", ErrorTooBig, code)
	}
	if err := <-handlerErr; err != nil {
		t.Fatalf("write action: %v", err)
	}

	select {
	case err := <-serveDone:
		if err == nil || !strings.Contains(err.Error(), "exceeds maximum 65535") {
			t.Fatalf("expected negotiated maximum error, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("protocol did not close after rejecting the ACK")
	}
}

func TestProtocolRepliesToHAProxyDisconnect(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()
	if err := clientConn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}

	client := newProtocolClient(
		context.Background(),
		serverConn,
		newTestAsyncScheduler(),
		newFramePool(DefaultMaxFrameSize),
		HandlerFunc(func(context.Context, *encoding.ActionWriter, *encoding.Message) {}),
	)
	serveDone := make(chan error, 1)
	go func() {
		serveDone <- client.Serve()
	}()

	writeTestWireFrame(
		t,
		clientConn,
		frameTypeIDHaproxyHello,
		0,
		0,
		testHelloPayload(t, DefaultMaxFrameSize),
	)
	_ = readTestWireFrame(t, clientConn)
	writeTestWireFrame(
		t,
		clientConn,
		frameTypeIDHaproxyDisconnect,
		0,
		0,
		testDisconnectPayload(t, ErrorNone),
	)

	disconnect := readTestWireFrame(t, clientConn)
	if disconnect.frameType != frameTypeIDAgentDisconnect {
		t.Fatalf("expected AGENT-DISCONNECT, got frame type %d", disconnect.frameType)
	}
	if code := testDisconnectCode(t, disconnect.payload); code != ErrorNone {
		t.Fatalf("expected disconnect code %d, got %d", ErrorNone, code)
	}

	select {
	case err := <-serveDone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected canceled protocol result, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("protocol did not close after HAPROXY-DISCONNECT")
	}
}

func TestProtocolTerminationIsBounded(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()
	client := newProtocolClient(
		context.Background(),
		serverConn,
		nil,
		newFramePool(DefaultMaxFrameSize),
		HandlerFunc(func(context.Context, *encoding.ActionWriter, *encoding.Message) {}),
	)

	done := make(chan struct{})
	go func() {
		client.terminate(newProtocolError(ErrorTooBig, "test termination"))
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("protocol termination blocked on the disconnect write")
	}
}

func TestProtocolTerminationSkipsWriteWithoutDeadline(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	defer serverConn.Close()
	defer clientConn.Close()
	client := newProtocolClient(
		context.Background(),
		&writeDeadlineErrorConn{Conn: serverConn},
		nil,
		newFramePool(DefaultMaxFrameSize),
		HandlerFunc(func(context.Context, *encoding.ActionWriter, *encoding.Message) {}),
	)

	done := make(chan struct{})
	go func() {
		client.terminate(newProtocolError(ErrorTooBig, "test termination"))
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("protocol termination attempted an unbounded disconnect write")
	}
}

func TestFramePoolsKeepLimitsIsolated(t *testing.T) {
	const smallMax uint32 = 4096
	const largeMax uint32 = 262140
	smallPool := newFramePool(smallMax)
	largePool := newFramePool(largeMax)

	for i := 0; i < 2; i++ {
		smallFrame := smallPool.acquire()
		if _, err := smallFrame.ReadFrom(testFrameBytes(t, smallMax+1)); err == nil {
			releaseFrame(smallFrame)
			t.Fatal("expected the small pool to reject an oversized frame")
		}
		releaseFrame(smallFrame)

		largeFrame := largePool.acquire()
		if _, err := largeFrame.ReadFrom(testFrameBytes(t, DefaultMaxFrameSize+1)); err != nil {
			releaseFrame(largeFrame)
			t.Fatalf("large pool rejected a valid frame: %v", err)
		}
		releaseFrame(largeFrame)
	}
}

type testWireFrame struct {
	payload   []byte
	length    uint32
	streamID  uint64
	frameID   uint64
	frameType frameType
}

type writeDeadlineErrorConn struct {
	net.Conn
}

func (c *writeDeadlineErrorConn) SetWriteDeadline(time.Time) error {
	return errors.ErrUnsupported
}

func testHelloFrame(t *testing.T, pool *framePool, maxFrameSize *uint32) *frame {
	t.Helper()
	f := pool.acquire()
	f.frameType = frameTypeIDHaproxyHello
	f.meta.Flags = frameFlagFin
	if err := f.encodeHeader(); err != nil {
		releaseFrame(f)
		t.Fatal(err)
	}
	headerLen := f.buf.Len()

	writer := encoding.NewKVWriter(f.buf.WriteBytes(), 0)
	if err := writer.SetString(helloKeySupportedVersions, version); err != nil {
		releaseFrame(f)
		t.Fatal(err)
	}
	if maxFrameSize != nil {
		if err := writer.SetUInt32(helloKeyMaxFrameSize, *maxFrameSize); err != nil {
			releaseFrame(f)
			t.Fatal(err)
		}
	}
	if err := writer.SetString(helloKeyCapabilities, ""); err != nil {
		releaseFrame(f)
		t.Fatal(err)
	}
	f.buf.AdvanceW(writer.Off())
	binary.BigEndian.PutUint32(f.length, uint32(f.buf.Len()))
	f.buf.AdvanceR(headerLen)
	return f
}

func testHelloPayload(t *testing.T, maxFrameSize uint32) []byte {
	t.Helper()
	buf := make([]byte, 128)
	writer := encoding.NewKVWriter(buf, 0)
	if err := writer.SetString(helloKeySupportedVersions, version); err != nil {
		t.Fatal(err)
	}
	if err := writer.SetUInt32(helloKeyMaxFrameSize, maxFrameSize); err != nil {
		t.Fatal(err)
	}
	if err := writer.SetString(helloKeyCapabilities, ""); err != nil {
		t.Fatal(err)
	}
	return writer.Bytes()
}

func testNotifyPayload(t *testing.T, messageName, entryName string, value []byte) []byte {
	t.Helper()
	buf := make([]byte, len(value)+256)
	off, err := encoding.PutBytes(buf, []byte(messageName))
	if err != nil {
		t.Fatal(err)
	}
	buf[off] = 1
	off++

	writer := encoding.NewKVWriter(buf[off:], 0)
	if err := writer.SetBinary(entryName, value); err != nil {
		t.Fatal(err)
	}
	off += writer.Off()
	return buf[:off]
}

func testDisconnectPayload(t *testing.T, code errorCode) []byte {
	t.Helper()
	buf := make([]byte, 128)
	writer := encoding.NewKVWriter(buf, 0)
	if err := writer.SetUInt32("status-code", uint32(code)); err != nil {
		t.Fatal(err)
	}
	if err := writer.SetString("message", code.String()); err != nil {
		t.Fatal(err)
	}
	return writer.Bytes()
}

func writeTestWireFrame(
	t *testing.T,
	w io.Writer,
	frameType frameType,
	streamID uint64,
	frameID uint64,
	payload []byte,
) {
	t.Helper()
	body := make([]byte, 1+uint32Len+20+len(payload))
	body[0] = byte(frameType)
	binary.BigEndian.PutUint32(body[1:1+uint32Len], uint32(frameFlagFin))
	off := 1 + uint32Len
	n, err := encoding.PutVarint(body[off:], streamID)
	if err != nil {
		t.Fatal(err)
	}
	off += n
	n, err = encoding.PutVarint(body[off:], frameID)
	if err != nil {
		t.Fatal(err)
	}
	off += n
	copy(body[off:], payload)
	body = body[:off+len(payload)]

	wire := make([]byte, uint32Len+len(body))
	binary.BigEndian.PutUint32(wire, uint32(len(body)))
	copy(wire[uint32Len:], body)
	if _, err := w.Write(wire); err != nil {
		t.Fatalf("write frame: %v", err)
	}
}

func readTestWireFrame(t *testing.T, r io.Reader) testWireFrame {
	t.Helper()
	lengthBytes := make([]byte, uint32Len)
	if _, err := io.ReadFull(r, lengthBytes); err != nil {
		t.Fatalf("read frame length: %v", err)
	}
	length := binary.BigEndian.Uint32(lengthBytes)
	body := make([]byte, length)
	if _, err := io.ReadFull(r, body); err != nil {
		t.Fatalf("read frame body: %v", err)
	}
	if len(body) < 1+uint32Len {
		t.Fatalf("frame body is too short: %d", len(body))
	}

	off := 1 + uint32Len
	streamID, n, err := encoding.Varint(body[off:])
	if err != nil {
		t.Fatalf("read stream ID: %v", err)
	}
	off += n
	frameID, n, err := encoding.Varint(body[off:])
	if err != nil {
		t.Fatalf("read frame ID: %v", err)
	}
	off += n

	return testWireFrame{
		payload:   body[off:],
		length:    length,
		streamID:  streamID,
		frameID:   frameID,
		frameType: frameType(body[0]),
	}
}

func testHelloMaxFrameSize(t *testing.T, payload []byte) uint32 {
	t.Helper()
	scanner := encoding.NewKVScanner(payload, -1)
	entry := encoding.AcquireKVEntry()
	defer encoding.ReleaseKVEntry(entry)
	for scanner.Next(entry) {
		if entry.NameEquals(helloKeyMaxFrameSize) {
			if entry.Type() != encoding.DataTypeUInt32 {
				t.Fatalf("%q has type %d, expected uint32", helloKeyMaxFrameSize, entry.Type())
			}
			return uint32(entry.ValueInt())
		}
	}
	if err := scanner.Error(); err != nil {
		t.Fatalf("scan HELLO payload: %v", err)
	}
	t.Fatalf("HELLO payload is missing %q", helloKeyMaxFrameSize)
	return 0
}

func testDisconnectCode(t *testing.T, payload []byte) errorCode {
	t.Helper()
	scanner := encoding.NewKVScanner(payload, -1)
	entry := encoding.AcquireKVEntry()
	defer encoding.ReleaseKVEntry(entry)
	for scanner.Next(entry) {
		if entry.NameEquals("status-code") {
			if entry.Type() != encoding.DataTypeUInt32 {
				t.Fatalf("status-code has type %d, expected uint32", entry.Type())
			}
			return errorCode(entry.ValueInt())
		}
	}
	if err := scanner.Error(); err != nil {
		t.Fatalf("scan disconnect payload: %v", err)
	}
	t.Fatal("disconnect payload is missing status-code")
	return ErrorUnknown
}

func testFrameBytes(t *testing.T, length uint32) *bytes.Reader {
	t.Helper()
	wire := make([]byte, uint32Len+length)
	binary.BigEndian.PutUint32(wire, length)
	if length >= 7 {
		wire[uint32Len] = byte(frameTypeIDNotify)
		binary.BigEndian.PutUint32(wire[uint32Len+1:uint32Len+5], uint32(frameFlagFin))
	}
	return bytes.NewReader(wire)
}

func uint32Ptr(value uint32) *uint32 {
	return &value
}
