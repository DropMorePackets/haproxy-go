package encoding

import (
	"net/netip"
)

func PutBytes(b []byte, v []byte) (int, error) {
	l := len(v)
	n, err := PutVarint(b, int64(l))
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
