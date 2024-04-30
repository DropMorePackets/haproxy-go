package encoding

import (
	"fmt"
	"net/netip"
	"sync"
)

type actionType byte

const (
	ActionTypeSetVar   actionType = 1
	ActionTypeUnsetVar actionType = 2
)

type varScope byte

const (
	VarScopeProcess     varScope = 0
	VarScopeSession     varScope = 1
	VarScopeTransaction varScope = 2
	VarScopeRequest     varScope = 3
	VarScopeResponse    varScope = 4
)

var actionWriterPool = sync.Pool{
	New: func() any {
		return NewActionWriter(nil, 0)
	},
}

func AcquireActionWriter(buf []byte, off int) *ActionWriter {
	w := actionWriterPool.Get().(*ActionWriter)
	w.data = buf
	w.off = off
	return w
}

func ReleaseActionWriter(w *ActionWriter) {
	w.data = nil
	w.off = 0
	actionWriterPool.Put(w)
}

type ActionWriter struct {
	data []byte
	off  int
}

func NewActionWriter(buf []byte, off int) *ActionWriter {
	return &ActionWriter{
		data: buf,
		off:  off,
	}
}

func (aw *ActionWriter) Off() int {
	return aw.off
}

func (aw *ActionWriter) Bytes() []byte {
	return aw.data[:aw.off]
}

func (aw *ActionWriter) actionHeader(t actionType, s varScope, name []byte) error {
	aw.data[aw.off] = byte(t)
	aw.off++

	// NB-Args
	var nbArgs byte
	switch t {
	case ActionTypeSetVar:
		nbArgs = 3
	case ActionTypeUnsetVar:
		nbArgs = 2
	default:
		panic("unknown action type")
	}

	aw.data[aw.off] = nbArgs
	aw.off++

	aw.data[aw.off] = byte(s)
	aw.off++

	n, err := PutBytes(aw.data[aw.off:], name)
	if err != nil {
		return err
	}

	aw.off += n
	return nil
}

func (aw *ActionWriter) Unset(s varScope, name string) error {
	return aw.actionHeader(ActionTypeUnsetVar, s, []byte(name))
}

func (aw *ActionWriter) SetStringBytes(s varScope, name string, v []byte) error {
	if err := aw.actionHeader(ActionTypeSetVar, s, []byte(name)); err != nil {
		return err
	}

	aw.data[aw.off] = byte(DataTypeString)
	aw.off++

	n, err := PutBytes(aw.data[aw.off:], v)
	if err != nil {
		return err
	}
	aw.off += n

	return nil
}
func (aw *ActionWriter) SetString(s varScope, name string, v string) error {
	return aw.SetStringBytes(s, name, []byte(v))
}

func (aw *ActionWriter) SetBinary(s varScope, name string, v []byte) error {
	if err := aw.actionHeader(ActionTypeSetVar, s, []byte(name)); err != nil {
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

func (aw *ActionWriter) SetNull(s varScope, name string) error {
	if err := aw.actionHeader(ActionTypeSetVar, s, []byte(name)); err != nil {
		return err
	}

	aw.data[aw.off] = byte(DataTypeNull)
	aw.off++

	return nil
}
func (aw *ActionWriter) SetBool(s varScope, name string, v bool) error {
	if err := aw.actionHeader(ActionTypeSetVar, s, []byte(name)); err != nil {
		return err
	}

	aw.data[aw.off] = byte(DataTypeBool)
	if v {
		aw.data[aw.off] |= dataFlagTrue
	}
	aw.off++

	return nil
}

func (aw *ActionWriter) SetUInt32(s varScope, name string, v uint32) error {
	return aw.SetInt64(s, name, int64(v))
}

func (aw *ActionWriter) SetInt32(s varScope, name string, v int32) error {
	return aw.SetInt64(s, name, int64(v))
}

func (aw *ActionWriter) SetInt64(s varScope, name string, v int64) error {
	if err := aw.actionHeader(ActionTypeSetVar, s, []byte(name)); err != nil {
		return err
	}

	aw.data[aw.off] = byte(DataTypeInt64)
	aw.off++

	n, err := PutVarint(aw.data[aw.off:], uint64(v))
	if err != nil {
		return err
	}
	aw.off += n

	return nil
}
func (aw *ActionWriter) SetUInt64(s varScope, name string, v uint64) error {
	return aw.SetInt64(s, name, int64(v))
}

func (aw *ActionWriter) SetAddr(s varScope, name string, v netip.Addr) error {
	if err := aw.actionHeader(ActionTypeSetVar, s, []byte(name)); err != nil {
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
