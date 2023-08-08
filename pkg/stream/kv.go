package stream

import (
	"fmt"
	"net"
	"sync"

	"github.com/fionera/haproxy-go/pkg/encoding"
)

var kvEntryPool = sync.Pool{
	New: func() any {
		return &KVEntry{}
	},
}

var kvScannerPool = sync.Pool{
	New: func() any {
		return NewKVScanner(nil, 0)
	},
}

func AcquireKVScanner(b []byte, count int) *KVScanner {
	s := kvScannerPool.Get().(*KVScanner)
	s.buf = b
	s.left = count
	return s
}

func ReleaseKVScanner(s *KVScanner) {
	s.lastErr = nil
	s.buf = nil
	s.left = 0
	kvScannerPool.Put(s)
}

func AcquireKVEntry() *KVEntry {
	return kvEntryPool.Get().(*KVEntry)
}

func ReleaseKVEntry(k *KVEntry) {
	k.name = nil
	k.dataType = 0
	k.byteVal = nil
	k.boolVar = false
	k.intVal = 0

	kvEntryPool.Put(k)
}

type KVScanner struct {
	buf     []byte
	left    int
	lastErr error
}

// RemainingBuf returns the remaining length of the buffer
func (k *KVScanner) RemainingBuf() int {
	return len(k.buf)
}

func NewKVScanner(b []byte, count int) *KVScanner {
	return &KVScanner{buf: b, left: count}
}

func (k *KVScanner) Error() error {
	return k.lastErr
}

type KVEntry struct {
	name []byte

	dataType byte

	// if the content is a varint, we directly decode it.
	// else its decoded on the fly
	byteVal []byte
	boolVar bool
	intVal  int64
}

func (k *KVEntry) NameBytes() []byte {
	return k.name
}

func (k *KVEntry) ValueBytes() []byte {
	return k.byteVal
}

func (k *KVEntry) ValueInt() int64 {
	return k.intVal
}

func (k *KVEntry) ValueBool() bool {
	return k.boolVar
}

func (k *KVEntry) NameEquals(s string) bool {
	// bytes.Equal describes this operation as alloc free
	return string(k.name) == s
}

const (
	dataTypeNull   byte = 0
	dataTypeBool   byte = 1
	dataTypeInt32  byte = 2
	dataTypeUInt32 byte = 3
	dataTypeInt64  byte = 4
	dataTypeUInt64 byte = 5
	dataTypeIPV4   byte = 6
	dataTypeIPV6   byte = 7
	dataTypeString byte = 8
	dataTypeBinary byte = 9
)

func (k *KVScanner) Next(e *KVEntry) bool {
	if len(k.buf) == 0 {
		return false
	}

	if e == nil {
		panic("KVEntry cant be nil")
	}
	k.left--

	nameLen, n, err := encoding.Varint(k.buf)
	if err != nil {
		k.lastErr = err
		return false
	}
	k.buf = k.buf[n:]

	e.name = k.buf[:nameLen]
	k.buf = k.buf[nameLen:]

	const dataTypeMask byte = 0x0F
	const dataFlagTrue byte = 0x10

	e.dataType = k.buf[0] & dataTypeMask
	// just always decode the boolVar even tho its wrong.
	e.boolVar = k.buf[0]&dataFlagTrue > 0
	k.buf = k.buf[1:]

	switch e.dataType {
	case dataTypeNull, dataTypeBool:
		// noop

	case dataTypeInt32, dataTypeInt64,
		dataTypeUInt32, dataTypeUInt64:
		e.intVal, n, k.lastErr = encoding.Varint(k.buf)
		if k.lastErr != nil {
			return false
		}

		k.buf = k.buf[n:]

	case dataTypeIPV4:
		e.byteVal = k.buf[:net.IPv4len]
		k.buf = k.buf[net.IPv4len:]

	case dataTypeIPV6:
		e.byteVal = k.buf[:net.IPv6len]
		k.buf = k.buf[net.IPv6len:]

	case dataTypeString:
		nameLen, n, err := encoding.Varint(k.buf)
		if err != nil {
			k.lastErr = err
			return false
		}
		k.buf = k.buf[n:]

		e.byteVal = k.buf[:nameLen]
		k.buf = k.buf[nameLen:]

	case dataTypeBinary:
		valLen, n, err := encoding.Varint(k.buf)
		if err != nil {
			k.lastErr = err
			return false
		}
		k.buf = k.buf[n:]

		e.byteVal = k.buf[:valLen]
		k.buf = k.buf[valLen:]

	default:
		k.lastErr = fmt.Errorf("unknown data type: %x", e.dataType)
		return false
	}

	return true
}

func (k *KVScanner) Discard() error {
	if k.RemainingBuf() == 0 {
		return nil
	}

	e := AcquireKVEntry()
	defer ReleaseKVEntry(e)
	for k.Next(e) {
	}

	return k.Error()
}
