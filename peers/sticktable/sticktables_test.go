package sticktable

import (
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
