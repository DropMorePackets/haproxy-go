package peers

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"net"

	"github.com/dropmorepackets/haproxy-go/pkg/encoding"
)

type MapKey interface {
	fmt.Stringer
	Unmarshal(r *bufio.Reader, keySize int64) error
}

type SignedIntegerKey int32

func (v *SignedIntegerKey) Unmarshal(r *bufio.Reader, keySize int64) error {
	value := make([]byte, 4)
	if _, err := r.Read(value); err != nil {
		return err
	}

	*v = SignedIntegerKey(binary.BigEndian.Uint32(value))
	return nil
}

func (v *SignedIntegerKey) String() string {
	return fmt.Sprintf("%d", *v)
}

type IPv4AddressKey net.IP

func (v *IPv4AddressKey) Unmarshal(r *bufio.Reader, keySize int64) error {
	value := make([]byte, keySize)
	if _, err := r.Read(value); err != nil {
		return err
	}

	*v = value
	return nil
}

func (v *IPv4AddressKey) String() string {
	return (*net.IP)(v).String()
}

type IPv6AddressKey net.IP

func (v *IPv6AddressKey) Unmarshal(r *bufio.Reader, keySize int64) error {
	value := make([]byte, keySize)
	if _, err := r.Read(value); err != nil {
		return err
	}

	*v = value
	return nil
}

func (v *IPv6AddressKey) String() string {
	return (*net.IP)(v).String()
}

type StringKey string

func (v *StringKey) Unmarshal(r *bufio.Reader, keySize int64) error {
	valueLength, err := encoding.ReadVarint(r)
	if err != nil {
		return err
	}

	value := make([]byte, valueLength)
	if _, err := r.Read(value); err != nil {
		return err
	}

	*v = StringKey(value)
	return nil
}

func (v *StringKey) String() string {
	return string(*v)
}

type BinaryKey []byte

func (v *BinaryKey) Unmarshal(r *bufio.Reader, keySize int64) error {
	value := make([]byte, keySize)
	if _, err := r.Read(value); err != nil {
		return err
	}

	*v = value
	return nil
}

func (v *BinaryKey) String() string {
	return fmt.Sprintf("%v", *v)
}

type MapData interface {
	fmt.Stringer
	Unmarshal(r *bufio.Reader) error
}

type FreqData struct {
	CurrentTick   int64
	CurrentPeriod int64
	LastPeriod    int64
}

func (f *FreqData) String() string {
	return fmt.Sprintf("tick/cur/last: %d/%d/%d", f.CurrentTick, f.CurrentPeriod, f.LastPeriod)
}

func (f *FreqData) Unmarshal(r *bufio.Reader) error {
	// start date of current period (wrapping ticks)
	currentTick, err := encoding.ReadVarint(r)
	if err != nil {
		return err
	}
	f.CurrentTick = currentTick

	// cumulated value for current period
	currentPeriod, err := encoding.ReadVarint(r)
	if err != nil {
		return err
	}
	f.CurrentPeriod = currentPeriod

	// value for last period
	lastPeriod, err := encoding.ReadVarint(r)
	if err != nil {
		return err
	}
	f.LastPeriod = lastPeriod

	return nil
}

type SignedIntegerData int32

func (v *SignedIntegerData) Unmarshal(r *bufio.Reader) error {
	value, err := encoding.ReadVarint(r)
	if err != nil {
		return err
	}

	*v = SignedIntegerData(value)
	return nil
}

func (v *SignedIntegerData) String() string {
	return fmt.Sprintf("%d", *v)
}

type UnsignedIntegerData uint32

func (v *UnsignedIntegerData) Unmarshal(r *bufio.Reader) error {
	value, err := encoding.ReadVarint(r)
	if err != nil {
		return err
	}

	*v = UnsignedIntegerData(value)
	return nil
}

func (v *UnsignedIntegerData) String() string {
	return fmt.Sprintf("%d", *v)
}

type UnsignedLongLongData int64

func (v *UnsignedLongLongData) Unmarshal(r *bufio.Reader) error {
	value, err := encoding.ReadVarint(r)
	if err != nil {
		return err
	}

	*v = UnsignedLongLongData(value)
	return nil
}

func (v *UnsignedLongLongData) String() string {
	return fmt.Sprintf("%d", *v)
}

type DictData struct {
	ID    int64
	Value []byte
}

func (f *DictData) String() string {
	if f.ID == 0 {
		return "No Entry"
	}

	return fmt.Sprintf("%d: %v", f.ID, f.Value)
}

func (f *DictData) Unmarshal(r *bufio.Reader) error {
	length, err := encoding.ReadVarint(r)
	if err != nil {
		return err
	}

	// No entries
	if length == 0 {
		return nil
	}

	id, err := encoding.ReadVarint(r)
	if err != nil {
		return err
	}
	f.ID = id

	if length == 1 {
		return nil
	}

	valueLength, err := encoding.ReadVarint(r)
	if err != nil {
		return err
	}

	if valueLength == 0 {
		return nil
	}

	value := make([]byte, valueLength)
	if _, err := r.Read(value); err != nil {
		return err
	}
	f.Value = value

	return nil
}
