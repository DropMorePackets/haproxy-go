package encoding

import (
	"fmt"
	"net"
	"net/netip"
	"sync"
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
	k.Reset()
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

	dataType DataType

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

func (k *KVEntry) ValueAddr() netip.Addr {
	addr, ok := netip.AddrFromSlice(k.byteVal)
	if !ok {
		panic("invalid addr decode")
	}
	return addr
}

func (k *KVEntry) NameEquals(s string) bool {
	// bytes.Equal describes this operation as alloc free
	return string(k.name) == s
}

func (k *KVEntry) Type() DataType {
	return k.dataType
}

// Value returns the typed value for the KVEntry. It can allocate memory
// which is why assertions and direct type access is recommended.
func (k *KVEntry) Value() any {
	switch k.dataType {
	case DataTypeNull:
		return nil
	case DataTypeBool:
		return k.boolVar
	case DataTypeInt32:
		return int32(k.intVal)
	case DataTypeInt64:
		return k.intVal
	case DataTypeUInt32:
		return uint32(k.intVal)
	case DataTypeUInt64:
		return uint64(k.intVal)
	case DataTypeIPV4, DataTypeIPV6:
		addr, ok := netip.AddrFromSlice(k.byteVal)
		if !ok {
			panic("invalid addr decode")
		}
		return addr
	case DataTypeString:
		return string(k.byteVal)
	case DataTypeBinary:
		return k.byteVal
	default:
		panic("unknown datatype")
	}
}

func (k *KVEntry) Reset() {
	k.name = nil
	k.dataType = 0
	k.byteVal = nil
	k.boolVar = false
	k.intVal = 0
}

func (k *KVScanner) Next(e *KVEntry) bool {
	if len(k.buf) == 0 {
		return false
	}

	if e == nil {
		panic("KVEntry cant be nil")
	}
	e.Reset()
	k.left--

	nameLen, n, err := Varint(k.buf)
	if err != nil {
		k.lastErr = err
		return false
	}
	k.buf = k.buf[n:]

	e.name = k.buf[:nameLen]
	k.buf = k.buf[nameLen:]

	e.dataType = DataType(k.buf[0] & dataTypeMask)
	// just always decode the boolVar even tho its wrong.
	e.boolVar = k.buf[0]&dataFlagTrue > 0
	k.buf = k.buf[1:]

	switch e.dataType {
	case DataTypeNull, DataTypeBool:
		// noop

	case DataTypeInt32, DataTypeInt64,
		DataTypeUInt32, DataTypeUInt64:
		e.intVal, n, k.lastErr = Varint(k.buf)
		if k.lastErr != nil {
			return false
		}

		k.buf = k.buf[n:]

	case DataTypeIPV4:
		e.byteVal = k.buf[:net.IPv4len]
		k.buf = k.buf[net.IPv4len:]

	case DataTypeIPV6:
		e.byteVal = k.buf[:net.IPv6len]
		k.buf = k.buf[net.IPv6len:]

	case DataTypeString:
		nameLen, n, err := Varint(k.buf)
		if err != nil {
			k.lastErr = err
			return false
		}
		k.buf = k.buf[n:]

		e.byteVal = k.buf[:nameLen]
		k.buf = k.buf[nameLen:]

	case DataTypeBinary:
		valLen, n, err := Varint(k.buf)
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
