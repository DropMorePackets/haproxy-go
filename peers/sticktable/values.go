package sticktable

import (
	"encoding/binary"
	"fmt"
	"net/netip"

	"github.com/dropmorepackets/haproxy-go/pkg/encoding"
)

type MapKey interface {
	fmt.Stringer
	Unmarshal(b []byte, keySize uint64) (int, error)
}

type SignedIntegerKey int32

func (v *SignedIntegerKey) Unmarshal(b []byte, keySize uint64) (int, error) {
	*v = SignedIntegerKey(binary.BigEndian.Uint32(b))
	return 4, nil
}

func (v *SignedIntegerKey) String() string {
	return fmt.Sprintf("%d", *v)
}

type IPv4AddressKey netip.Addr

func (v *IPv4AddressKey) Unmarshal(b []byte, keySize uint64) (int, error) {
	if keySize != 4 {
		return 0, fmt.Errorf("invalid ipv4 key size: %d", keySize)
	}

	*v = IPv4AddressKey(netip.AddrFrom4([4]byte(b)))
	return 4, nil
}

func (v *IPv4AddressKey) String() string {
	return (*netip.Addr)(v).String()
}

type IPv6AddressKey netip.Addr

func (v *IPv6AddressKey) Unmarshal(b []byte, keySize uint64) (int, error) {
	if keySize != 16 {
		return 0, fmt.Errorf("invalid ipv6 key size: %d", keySize)
	}

	*v = IPv6AddressKey(netip.AddrFrom16([16]byte(b)))

	return 16, nil
}

func (v *IPv6AddressKey) String() string {
	return (*netip.Addr)(v).String()
}

type StringKey string

func (v *StringKey) Unmarshal(b []byte, keySize uint64) (int, error) {
	valueLength, n, err := encoding.Varint(b)
	if err != nil {
		return n, err
	}
	if valueLength == 0 {
		return n, nil
	}
	*v = StringKey(b[n:valueLength])
	return n + int(valueLength), nil
}

func (v *StringKey) String() string {
	return string(*v)
}

type BinaryKey []byte

func (v *BinaryKey) Unmarshal(b []byte, keySize uint64) (int, error) {
	*v = b[:keySize]
	return int(keySize), nil
}

func (v *BinaryKey) String() string {
	return fmt.Sprintf("%v", *v)
}

type MapData interface {
	fmt.Stringer
	Unmarshal(b []byte) (int, error)
}

type FreqData struct {
	CurrentTick   uint64
	CurrentPeriod uint64
	LastPeriod    uint64
}

func (f *FreqData) String() string {
	return fmt.Sprintf("tick/cur/last: %d/%d/%d", f.CurrentTick, f.CurrentPeriod, f.LastPeriod)
}

func (f *FreqData) Unmarshal(b []byte) (int, error) {
	var offset int
	// start date of current period (wrapping ticks)
	currentTick, n, err := encoding.Varint(b[offset:])
	if err != nil {
		return n, err
	}
	f.CurrentTick = currentTick
	offset += n

	// cumulated value for current period
	currentPeriod, n, err := encoding.Varint(b[offset:])
	if err != nil {
		return n, err
	}
	f.CurrentPeriod = currentPeriod
	offset += n

	// value for last period
	lastPeriod, n, err := encoding.Varint(b[offset:])
	if err != nil {
		return n, err
	}
	f.LastPeriod = lastPeriod
	offset += n

	return offset, nil
}

type SignedIntegerData int32

func (v *SignedIntegerData) Unmarshal(b []byte) (int, error) {
	value, n, err := encoding.Varint(b)
	if err != nil {
		return n, err
	}

	*v = SignedIntegerData(value)
	return n, nil
}

func (v *SignedIntegerData) String() string {
	return fmt.Sprintf("%d", *v)
}

type UnsignedIntegerData uint32

func (v *UnsignedIntegerData) Unmarshal(b []byte) (int, error) {
	value, n, err := encoding.Varint(b)
	if err != nil {
		return n, err
	}

	*v = UnsignedIntegerData(value)
	return n, nil
}

func (v *UnsignedIntegerData) String() string {
	return fmt.Sprintf("%d", *v)
}

type UnsignedLongLongData uint64

func (v *UnsignedLongLongData) Unmarshal(b []byte) (int, error) {
	value, n, err := encoding.Varint(b)
	if err != nil {
		return n, err
	}

	*v = UnsignedLongLongData(value)
	return n, nil
}

func (v *UnsignedLongLongData) String() string {
	return fmt.Sprintf("%d", *v)
}

type DictData struct {
	ID    uint64
	Value []byte
}

func (f *DictData) String() string {
	if f.ID == 0 {
		return "No Entry"
	}

	return fmt.Sprintf("%d: %v", f.ID, f.Value)
}

func (f *DictData) Unmarshal(b []byte) (int, error) {
	var offset int
	length, n, err := encoding.Varint(b[offset:])
	offset += n
	if err != nil {
		return offset, err
	}

	// No entries
	if length == 0 {
		return offset, nil
	}

	id, n, err := encoding.Varint(b[offset:])
	offset += n
	if err != nil {
		return offset, err
	}
	f.ID = id

	if length == 1 {
		return offset, nil
	}

	valueLength, n, err := encoding.Varint(b[offset:])
	offset += n
	if err != nil {
		return offset, err
	}

	if valueLength == 0 {
		return offset, nil
	}

	value := make([]byte, valueLength)
	offset += copy(value, b[offset:])
	f.Value = value

	return offset, nil
}
