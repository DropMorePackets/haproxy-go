package spop

import (
	"bytes"
	"context"
	"encoding/binary"
	"testing"

	"github.com/dropmorepackets/haproxy-go/pkg/encoding"
)

func TestProtocolMaxFrameSizeOffer(t *testing.T) {
	tests := []struct {
		name    string
		offer   uint32
		wantErr bool
	}{
		{name: "at current limit", offer: uint32(maxFrameSize)},
		{name: "above initial limit", offer: 262140},
		{name: "above HAProxy limit", offer: ^uint32(0), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rw bytes.Buffer
			client := newProtocolClient(
				context.Background(),
				&rw,
				nil,
				HandlerFunc(func(context.Context, *encoding.ActionWriter, *encoding.Message) {}),
			)
			frame := testHelloFrame(t, tt.offer)
			defer releaseFrame(frame)

			err := client.onHAProxyHello(frame)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected max-frame-size offer to be rejected")
				}
				return
			}
			if err != nil {
				t.Fatalf("handle HAPROXY-HELLO: %v", err)
			}
			if client.maxFrameSize != tt.offer {
				t.Fatalf("expected negotiated size %d, got %d", tt.offer, client.maxFrameSize)
			}
			if got := agentHelloMaxFrameSize(t, &rw); got != tt.offer {
				t.Fatalf("expected AGENT-HELLO size %d, got %d", tt.offer, got)
			}
		})
	}
}

func agentHelloMaxFrameSize(t *testing.T, rw *bytes.Buffer) uint32 {
	t.Helper()
	f := acquireFrame()
	defer releaseFrame(f)
	if _, err := f.ReadFrom(rw); err != nil {
		t.Fatalf("read AGENT-HELLO: %v", err)
	}
	if f.frameType != frameTypeIDAgentHello {
		t.Fatalf("expected AGENT-HELLO, got frame type %d", f.frameType)
	}

	s := encoding.AcquireKVScanner(f.buf.ReadBytes(), -1)
	defer encoding.ReleaseKVScanner(s)
	k := encoding.AcquireKVEntry()
	defer encoding.ReleaseKVEntry(k)
	for s.Next(k) {
		if k.NameEquals(helloKeyMaxFrameSize) {
			return uint32(k.ValueInt())
		}
	}
	if err := s.Error(); err != nil {
		t.Fatalf("scan AGENT-HELLO: %v", err)
	}
	t.Fatal("AGENT-HELLO missing max-frame-size")
	return 0
}

func testHelloFrame(t *testing.T, offer uint32) *frame {
	t.Helper()
	f := acquireFrame()
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
	if err := writer.SetUInt32(helloKeyMaxFrameSize, offer); err != nil {
		releaseFrame(f)
		t.Fatal(err)
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
