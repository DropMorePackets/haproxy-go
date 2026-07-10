package sticktable

import (
	"net/netip"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMarshalUnmarshalMessages(t *testing.T) {
	stickTableDefinition := &Definition{
		StickTableID: 1337,
		Name:         "foobar",
		KeyType:      KeyTypeString,
		KeyLength:    11,
		DataTypes: []DataTypeDefinition{
			{
				DataType: DataTypeBytesInRate,
				Counter:  12,
				Period:   1,
			},
		},
		Expiry: 13,
	}

	stringKey := StringKey("1234567890a")
	entryUpdate := &EntryUpdate{
		StickTable:        stickTableDefinition,
		WithLocalUpdateID: true,
		WithExpiry:        true,
		LocalUpdateID:     23,
		Key:               &stringKey,
		Data: []MapData{
			&FreqData{
				CurrentTick:   1,
				CurrentPeriod: 2,
				LastPeriod:    3,
			},
		},
		Expiry: 42,
	}

	t.Run("sticktable definition", func(t *testing.T) {
		b := make([]byte, 256)
		n, err := stickTableDefinition.Marshal(b)
		if err != nil {
			t.Fatal(err)
		}

		var d Definition
		_, err = d.Unmarshal(b[:n])
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(*stickTableDefinition, d); diff != "" {
			t.Errorf("Unmarshal() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("entry update", func(t *testing.T) {
		b := make([]byte, 256)
		n, err := entryUpdate.Marshal(b)
		if err != nil {
			t.Fatal(err)
		}

		var d EntryUpdate
		d.StickTable = entryUpdate.StickTable
		d.WithExpiry = entryUpdate.WithExpiry
		d.WithLocalUpdateID = entryUpdate.WithLocalUpdateID
		_, err = d.Unmarshal(b[:n])
		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(*entryUpdate, d); diff != "" {
			t.Errorf("Unmarshal() mismatch (-want +got):\n%s", diff)
		}
	})
}

// TestEntryUpdateMarshalEncodesData guards against a regression where
// EntryUpdate.Marshal encoded the data section by calling MapData.Unmarshal
// instead of Marshal. That produced a zeroed data section on the wire and, as a
// side effect, mutated the source EntryUpdate's data to match — which hid the
// bug from a round-trip test that compares against the same (now-mutated) value.
//
// This test uses an independent expected value and also asserts the source is
// left unchanged, so it fails if the data is not actually encoded.
func TestEntryUpdateMarshalEncodesData(t *testing.T) {
	def := &Definition{
		Name:      "reputation",
		KeyType:   KeyTypeIPv4Address,
		KeyLength: 4,
		DataTypes: []DataTypeDefinition{{DataType: DataTypeGPT0}},
	}
	key := IPv4AddressKey(netip.MustParseAddr("1.2.3.4"))
	data := UnsignedIntegerData(42)
	e := &EntryUpdate{
		StickTable:        def,
		WithLocalUpdateID: true,
		LocalUpdateID:     7,
		Key:               &key,
		Data:              []MapData{&data},
	}

	b := make([]byte, 64)
	n, err := e.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}

	// Marshalling must not mutate the source data.
	if data != 42 {
		t.Fatalf("Marshal mutated source data to %d, want 42", data)
	}

	// The encoded bytes must decode back to the original value.
	got := EntryUpdate{StickTable: def, WithLocalUpdateID: true}
	if _, err := got.Unmarshal(b[:n]); err != nil {
		t.Fatal(err)
	}
	if len(got.Data) != 1 || got.Data[0].String() != "42" {
		t.Errorf("round-trip data = %v, want [42]", got.Data)
	}
}
