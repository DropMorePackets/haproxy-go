package spop

import (
	"encoding/binary"
	"fmt"
	"github.com/fionera/haproxy-go/pkg/encoding"
	"io"
	"strings"
)

func (c *conn) readFrame(f *frame) bool {
	if _, err := c.bufConn.Read(f.length); err != nil {
		f.err = err
		return false
	}
	frameLen := binary.BigEndian.Uint32(f.length)

	f.buf.Reset()
	dataBuf := f.buf.WriteNBytes(int(frameLen))

	// read full frame into buffer
	n, err := c.bufConn.Read(dataBuf)
	if err != nil {
		f.err = err
		return false
	}

	if n != int(frameLen) {
		f.err = io.ErrUnexpectedEOF
		return false
	}

	f.readHeader()

	return f.err == nil
}

func (c *conn) onHello(f *frame) error {
	if c.gotHello {
		panic("duplicate hello frame")
	}
	c.gotHello = true

	k := encoding.AcquireKVEntry()
	defer encoding.ReleaseKVEntry(k)

	s := encoding.AcquireKVScanner(f.buf.ReadBytes(), -1)
	for s.Next(k) {
		switch {
		case k.NameEquals(helloKeyMaxFrameSize):
			if c.maxFrameSize > uint32(k.ValueInt()) {
				c.maxFrameSize = uint32(k.ValueInt())
			} else {
				return fmt.Errorf("maxFrameSize bigger than maximum allowed size")
			}
		case k.NameEquals(helloKeyEngineID):
			//TODO: This does copy the engine id but yolo?
			c.engineID = string(k.ValueBytes())
			//case k.NameEquals(helloKeySupportedVersions):
			//case k.NameEquals(helloKeyCapabilities):
			//case k.NameEquals(helloKeyHealthcheck):
		}
	}

	if s.Error() != nil {
		return s.Error()
	}

	return c.writeHello()
}

func (c *conn) onNotify(f *frame) error {
	m := encoding.AcquireMessage()
	defer encoding.ReleaseMessage(m)

	rf := acquireFrame()
	defer releaseFrame(rf)

	rf.frameType = frameTypeIDAck
	rf.meta.FrameID = f.meta.FrameID
	rf.meta.StreamID = f.meta.StreamID
	rf.meta.Flags = frameFlagFin

	rf.writeHeader()
	if rf.err != nil {
		return f.err
	}

	w := encoding.AcquireActionWriter(rf.buf.WriteBytes(), 0)
	defer encoding.ReleaseActionWriter(w)

	s := encoding.AcquireMessageScanner(f.buf.ReadBytes())
	defer encoding.ReleaseMessageScanner(s)
	for s.Next(m) {
		c.handler.HandleSPOE(w, m)

		if err := m.KV().Discard(); err != nil {
			return err
		}
	}
	rf.buf.AdvanceW(w.Off())

	if s.Error() != nil {
		return s.Error()
	}

	return c.writeFrame(rf)
}

func (c *conn) onDisconnect(f *frame) error {
	//TODO: read disconnect reason and return error if required?
	return nil
}

func (c *conn) writeHello() error {
	f := acquireFrame()
	defer releaseFrame(f)

	f.frameType = frameTypeIDAgentHello
	f.meta.FrameID = 0
	f.meta.StreamID = 0
	f.meta.Flags = frameFlagFin

	f.writeHeader()
	if f.err != nil {
		return f.err
	}

	w := encoding.NewKVWriter(f.buf.WriteBytes(), 0)
	if err := w.SetString(helloKeyVersion, version); err != nil {
		return err
	}

	if err := w.SetUInt32(helloKeyMaxFrameSize, c.maxFrameSize); err != nil {
		return err
	}

	// TODO: Hardcode caps?
	err := w.SetString(helloKeyCapabilities, strings.Join([]string{capabilityNamePipelining, capabilityNameAsync}, ","))
	if err != nil {
		return err
	}
	f.buf.AdvanceW(w.Off())

	return c.writeFrame(f)
}

func (c *conn) writeDisconnect() error {
	if !c.gotHello {
		return nil
	}

	f := acquireFrame()
	defer releaseFrame(f)

	f.frameType = frameTypeIDAgentDisconnect
	f.meta.FrameID = 0
	f.meta.StreamID = 0
	f.meta.Flags = frameFlagFin

	f.writeHeader()
	if f.err != nil {
		return f.err
	}

	errCode := spoeErrorNone
	if c.lastErr != nil {
		//TODO: do proper error messages
		errCode = spoeErrorUnknown
	}

	w := encoding.NewKVWriter(f.buf.WriteBytes(), 0)
	if err := w.SetUInt32("status-code", uint32(errCode)); err != nil {
		return err
	}

	if err := w.SetString("message", errCode.String()); err != nil {
		return err
	}

	f.buf.AdvanceW(w.Off())

	return c.writeFrame(f)
}
