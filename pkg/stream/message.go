package stream

import (
	"sync"

	"github.com/fionera/haproxy-go/pkg/encoding"
)

var messagePool = sync.Pool{
	New: func() any {
		return &Message{}
	},
}

var messageScannerPool = sync.Pool{
	New: func() any {
		return NewMessageScanner(nil)
	},
}

func AcquireMessageScanner(buf []byte) *MessageScanner {
	s := messageScannerPool.Get().(*MessageScanner)
	s.buf = buf
	s.lastErr = nil
	return s
}

func ReleaseMessageScanner(s *MessageScanner) {
	s.buf = nil
	s.lastErr = nil
	messageScannerPool.Put(s)
}

func AcquireMessage() *Message {
	return messagePool.Get().(*Message)
}

func ReleaseMessage(m *Message) {
	m.name = nil
	m.scanner = nil
	m.kvEntryCount = 0

	messagePool.Put(m)
}

type Message struct {
	name []byte

	kvEntryCount byte
	scanner      *KVScanner
}

func (m *Message) NameBytes() []byte {
	return m.name
}

func (m *Message) KV() *KVScanner {
	return m.scanner
}

type MessageScanner struct {
	buf     []byte
	lastErr error
}

func NewMessageScanner(b []byte) *MessageScanner {
	return &MessageScanner{buf: b}
}

func (s *MessageScanner) Error() error {
	return s.lastErr
}

func (s *MessageScanner) Next(m *Message) bool {
	if m.scanner != nil {
		// if the scanner is still existing from a previous read
		// forward the current slice to the correct position
		s.buf = s.buf[len(s.buf)-m.scanner.RemainingBuf():]
		ReleaseKVScanner(m.scanner)
	}

	if len(s.buf) == 0 {
		return false
	}

	nameLen, n, err := encoding.Varint(s.buf)
	if err != nil {
		s.lastErr = err
		return false
	}
	s.buf = s.buf[n:]

	m.name = s.buf[:nameLen]
	s.buf = s.buf[nameLen:]

	m.kvEntryCount = s.buf[0]
	s.buf = s.buf[1:]

	m.scanner = AcquireKVScanner(s.buf, int(m.kvEntryCount))

	return true
}
