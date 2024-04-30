package encoding

import (
	"net/netip"
)

type DataType byte

const (
	DataTypeNull   DataType = 0
	DataTypeBool   DataType = 1
	DataTypeInt32  DataType = 2
	DataTypeUInt32 DataType = 3
	DataTypeInt64  DataType = 4
	DataTypeUInt64 DataType = 5
	DataTypeIPV4   DataType = 6
	DataTypeIPV6   DataType = 7
	DataTypeString DataType = 8
	DataTypeBinary DataType = 9

	dataTypeMask byte = 0x0F
	dataFlagTrue byte = 0x10
)

func PutBytes(b []byte, v []byte) (int, error) {
	l := len(v)
	n, err := PutVarint(b, uint64(l))
	if err != nil {
		return 0, err
	}

	if l+n > len(b) {
		return 0, ErrInsufficientSpace
	}

	copy(b[n:], v)
	return n + l, nil
}

func PutAddr(b []byte, ip netip.Addr) (int, error) {
	s := ip.AsSlice()
	if len(b) < len(s) {
		return 0, ErrInsufficientSpace
	}

	copy(b, s)
	return len(s), nil
}
