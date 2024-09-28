package sticktable

import (
	"net/netip"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMarshalUnmarshalValues(t *testing.T) {
	t.Run("MapKey", func(t *testing.T) {
		t.Run("SignedIntegerKey", func(t *testing.T) {
			in := SignedIntegerKey(1337)
			var out SignedIntegerKey

			b := make([]byte, 256)
			n, err := in.Marshal(b, 1)
			if err != nil {
				t.Fatal(err)
			}

			_, err = out.Unmarshal(b[:n], 1)
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(in, out); diff != "" {
				t.Errorf("Unmarshal() mismatch (-want +got):\n%s", diff)
			}
		})
		t.Run("IPv4AddressKey", func(t *testing.T) {
			in := IPv4AddressKey(netip.MustParseAddr("127.0.0.1"))
			var out IPv4AddressKey

			b := make([]byte, 256)
			n, err := in.Marshal(b, 4)
			if err != nil {
				t.Fatal(err)
			}

			_, err = out.Unmarshal(b[:n], 4)
			if err != nil {
				t.Fatal(err)
			}

			if in != out {
				t.Errorf("Unmarshal() mismatch:\n%v != %v", in, out)
			}
		})
		t.Run("IPv6AddressKey", func(t *testing.T) {
			in := IPv6AddressKey(netip.MustParseAddr("fe80::1"))
			var out IPv6AddressKey

			b := make([]byte, 256)
			n, err := in.Marshal(b, 16)
			if err != nil {
				t.Fatal(err)
			}

			_, err = out.Unmarshal(b[:n], 16)
			if err != nil {
				t.Fatal(err)
			}

			if in != out {
				t.Errorf("Unmarshal() mismatch:\n%v != %v", in, out)
			}
		})
		t.Run("StringKey", func(t *testing.T) {
			in := StringKey("foobar")
			var out StringKey

			b := make([]byte, 256)
			n, err := in.Marshal(b, uint64(len(in)))
			if err != nil {
				t.Fatal(err)
			}

			_, err = out.Unmarshal(b[:n], uint64(len(in)))
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(in, out); diff != "" {
				t.Errorf("Unmarshal() mismatch (-want +got):\n%s", diff)
			}
		})
		t.Run("BinaryKey", func(t *testing.T) {
			in := BinaryKey("foobar")
			var out BinaryKey

			b := make([]byte, 256)
			n, err := in.Marshal(b, uint64(len(in)))
			if err != nil {
				t.Fatal(err)
			}

			_, err = out.Unmarshal(b[:n], uint64(len(in)))
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(in, out); diff != "" {
				t.Errorf("Unmarshal() mismatch (-want +got):\n%s", diff)
			}
		})

	})

	t.Run("MapData", func(t *testing.T) {
		t.Run("FreqData", func(t *testing.T) {
			in := FreqData{
				CurrentTick:   1,
				CurrentPeriod: 2,
				LastPeriod:    3,
			}
			var out FreqData

			b := make([]byte, 256)
			n, err := in.Marshal(b)
			if err != nil {
				t.Fatal(err)
			}

			_, err = out.Unmarshal(b[:n])
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(in, out); diff != "" {
				t.Errorf("Unmarshal() mismatch (-want +got):\n%s", diff)
			}
		})
		t.Run("SignedIntegerData", func(t *testing.T) {
			in := SignedIntegerData(1337)
			var out SignedIntegerData

			b := make([]byte, 256)
			n, err := in.Marshal(b)
			if err != nil {
				t.Fatal(err)
			}

			_, err = out.Unmarshal(b[:n])
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(in, out); diff != "" {
				t.Errorf("Unmarshal() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("UnsignedIntegerData", func(t *testing.T) {
			in := UnsignedIntegerData(1337)
			var out UnsignedIntegerData

			b := make([]byte, 256)
			n, err := in.Marshal(b)
			if err != nil {
				t.Fatal(err)
			}

			_, err = out.Unmarshal(b[:n])
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(in, out); diff != "" {
				t.Errorf("Unmarshal() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("UnsignedLongLongData", func(t *testing.T) {
			in := UnsignedLongLongData(1337)
			var out UnsignedLongLongData

			b := make([]byte, 256)
			n, err := in.Marshal(b)
			if err != nil {
				t.Fatal(err)
			}

			_, err = out.Unmarshal(b[:n])
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(in, out); diff != "" {
				t.Errorf("Unmarshal() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("DictData", func(t *testing.T) {
			in := DictData{
				ID:    1,
				Value: []byte("foobar"),
			}
			var out DictData

			b := make([]byte, 256)
			n, err := in.Marshal(b)
			if err != nil {
				t.Fatal(err)
			}

			_, err = out.Unmarshal(b[:n])
			if err != nil {
				t.Fatal(err)
			}

			if diff := cmp.Diff(in, out); diff != "" {
				t.Errorf("Unmarshal() mismatch (-want +got):\n%s", diff)
			}
		})
	})
}
