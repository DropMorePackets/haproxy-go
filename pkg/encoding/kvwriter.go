package encoding

import (
	"fmt"
	"net/netip"
	"sync"
)

var kvWriterPool = sync.Pool{
	New: func() any {
		return NewKVWriter(nil, 0)
	},
}

func AcquireKVWriter(buf []byte, off int) *KVWriter {
	w := kvWriterPool.Get().(*KVWriter)
	w.data = buf
	w.off = off
	return w
}

func ReleaseKVWriter(w *KVWriter) {
	w.data = nil
	w.off = 0
	kvWriterPool.Put(w)
}

type KVWriter struct {
	data []byte
	off  int
}

func NewKVWriter(buf []byte, off int) *KVWriter {
	return &KVWriter{
		data: buf,
		off:  off,
	}
}

func (aw *KVWriter) Off() int {
	return aw.off
}

func (aw *KVWriter) Bytes() []byte {
	return aw.data[:aw.off]
}

func (aw *KVWriter) writeKey(name []byte) error {
	n, err := PutBytes(aw.data[aw.off:], name)
	if err != nil {
		return err
	}

	aw.off += n
	return nil
}

func (aw *KVWriter) SetString(name string, v string) error {
	if err := aw.writeKey([]byte(name)); err != nil {
		return err
	}

	aw.data[aw.off] = byte(DataTypeString)
	aw.off++

	n, err := PutBytes(aw.data[aw.off:], []byte(v))
	if err != nil {
		return err
	}
	aw.off += n

	return nil
}

func (aw *KVWriter) SetBinary(name string, v []byte) error {
	if err := aw.writeKey([]byte(name)); err != nil {
		return err
	}

	aw.data[aw.off] = byte(DataTypeBinary)
	aw.off++

	n, err := PutBytes(aw.data[aw.off:], v)
	if err != nil {
		return err
	}
	aw.off += n

	return nil
}

func (aw *KVWriter) SetNull(name string) error {
	if err := aw.writeKey([]byte(name)); err != nil {
		return err
	}

	aw.data[aw.off] = byte(DataTypeNull)
	aw.off++

	return nil
}
func (aw *KVWriter) SetBool(name string, v bool) error {
	if err := aw.writeKey([]byte(name)); err != nil {
		return err
	}

	aw.data[aw.off] = byte(DataTypeBool)
	if v {
		aw.data[aw.off] |= dataFlagTrue
	}
	aw.off++

	return nil
}

func (aw *KVWriter) SetUInt32(name string, v uint32) error {
	return aw.setInt(name, DataTypeUInt32, int64(v))
}

func (aw *KVWriter) SetInt32(name string, v int32) error {
	return aw.setInt(name, DataTypeInt32, int64(v))
}

func (aw *KVWriter) setInt(name string, d DataType, v int64) error {
	if err := aw.writeKey([]byte(name)); err != nil {
		return err
	}

	aw.data[aw.off] = byte(d)
	aw.off++

	n, err := PutVarint(aw.data[aw.off:], v)
	if err != nil {
		return err
	}
	aw.off += n

	return nil
}

func (aw *KVWriter) SetInt64(name string, v int64) error {
	return aw.setInt(name, DataTypeInt64, v)
}
func (aw *KVWriter) SetUInt64(name string, v uint64) error {
	return aw.setInt(name, DataTypeUInt64, int64(v))
}

func (aw *KVWriter) SetAddr(name string, v netip.Addr) error {
	if err := aw.writeKey([]byte(name)); err != nil {
		return err
	}

	switch {
	case v.Is6():
		aw.data[aw.off] = byte(DataTypeIPV6)
	case v.Is4():
		aw.data[aw.off] = byte(DataTypeIPV4)
	default:
		return fmt.Errorf("invalid address")
	}
	aw.off++

	n, err := PutAddr(aw.data[aw.off:], v)
	if err != nil {
		return err
	}
	aw.off += n

	return nil
}
