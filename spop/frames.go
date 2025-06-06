package spop

import (
	"fmt"
	"io"
	"strings"

	"github.com/dropmorepackets/haproxy-go/pkg/encoding"
)

type frameFlag uint32

const (
	frameFlagFin  frameFlag = 1
	frameFlagAbrt frameFlag = 2
)

type frameType byte

const (
	// Frames sent by HAProxy
	frameTypeIDHaproxyHello      frameType = 1
	frameTypeIDHaproxyDisconnect frameType = 2
	frameTypeIDNotify            frameType = 3

	// Frames sent by the agents
	frameTypeIDAgentHello      frameType = 101
	frameTypeIDAgentDisconnect frameType = 102
	frameTypeIDAck             frameType = 103
)

type frameWriter interface {
	io.WriterTo
}

var (
	_ frameWriter = (*AgentDisconnectFrame)(nil)
	_ frameWriter = (*AgentHelloFrame)(nil)
)

type AgentDisconnectFrame struct {
	ErrCode errorCode
}

func (a *AgentDisconnectFrame) WriteTo(w io.Writer) (int64, error) {
	f := acquireFrame()
	defer releaseFrame(f)

	f.frameType = frameTypeIDAgentDisconnect
	f.meta.FrameID = 0
	f.meta.StreamID = 0
	f.meta.Flags = frameFlagFin

	if err := f.encodeHeader(); err != nil {
		return 0, err
	}

	kvw := encoding.NewKVWriter(f.buf.WriteBytes(), 0)
	if err := kvw.SetUInt32("status-code", uint32(a.ErrCode)); err != nil {
		return 0, err
	}

	if err := kvw.SetString("message", a.ErrCode.String()); err != nil {
		return 0, err
	}

	f.buf.AdvanceW(kvw.Off())

	return f.WriteTo(w)
}

const (
	helloKeyMaxFrameSize      = "max-frame-size"
	helloKeySupportedVersions = "supported-versions"
	helloKeyVersion           = "version"
	helloKeyCapabilities      = "capabilities"
	helloKeyHealthcheck       = "healthcheck"
	helloKeyEngineID          = "engine-id"

	//lint:ignore U1000 These will probably be implemented again
	capabilityNameAsync      = "async"
	capabilityNamePipelining = "pipelining"
)

type AgentHelloFrame struct {
	Version      string
	Capabilities []string
	MaxFrameSize uint32
}

func (a *AgentHelloFrame) WriteTo(w io.Writer) (int64, error) {
	f := acquireFrame()
	defer releaseFrame(f)

	f.frameType = frameTypeIDAgentHello
	f.meta.FrameID = 0
	f.meta.StreamID = 0
	f.meta.Flags = frameFlagFin

	if err := f.encodeHeader(); err != nil {
		return 0, err
	}

	kvw := encoding.NewKVWriter(f.buf.WriteBytes(), 0)
	if err := kvw.SetString(helloKeyVersion, a.Version); err != nil {
		return 0, err
	}

	if err := kvw.SetUInt32(helloKeyMaxFrameSize, a.MaxFrameSize); err != nil {
		return 0, err
	}

	err := kvw.SetString(helloKeyCapabilities, strings.Join(a.Capabilities, ","))
	if err != nil {
		return 0, err
	}
	f.buf.AdvanceW(kvw.Off())

	return f.WriteTo(w)
}

type AckFrame struct {
	ActionWriterCallback func(*encoding.ActionWriter) error
	FrameID              uint64
	StreamID             uint64
}

func (a *AckFrame) WriteTo(w io.Writer) (int64, error) {
	f := acquireFrame()
	defer releaseFrame(f)

	f.frameType = frameTypeIDAck
	f.meta.FrameID = a.FrameID
	f.meta.StreamID = a.StreamID
	f.meta.Flags = frameFlagFin

	if err := f.encodeHeader(); err != nil {
		return 0, fmt.Errorf("encoding header: %w", err)
	}

	aw := encoding.AcquireActionWriter(f.buf.WriteBytes(), 0)
	defer encoding.ReleaseActionWriter(aw)

	// TODO: errors are not correctly handled and will result in an invalid state.
	if err := a.ActionWriterCallback(aw); err != nil {
		return 0, err
	}

	f.buf.AdvanceW(aw.Off())

	return f.WriteTo(w)
}
