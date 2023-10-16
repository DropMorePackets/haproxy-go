package sticktable

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/dropmorepackets/haproxy-go/pkg/encoding"
)

type DataTypeDefinition struct {
	DataType DataType
	Counter  uint64
	Period   uint64
}

type Definition struct {
	StickTableID uint64
	Name         string
	KeyType      KeyType
	KeyLength    uint64
	DataTypes    []DataTypeDefinition
	Expiry       uint64
}

func (s *Definition) Unmarshal(b []byte) (int, error) {
	var offset int
	stickTableID, n, err := encoding.Varint(b[offset:])
	offset += n
	if err != nil {
		return offset, err
	}
	s.StickTableID = stickTableID

	nameLength, n, err := encoding.Varint(b[offset:])
	offset += n
	if err != nil {
		return offset, err
	}

	name := make([]byte, nameLength)
	offset += copy(name, b[offset:])
	s.Name = string(name)

	keyType, n, err := encoding.Varint(b[offset:])
	offset += n
	if err != nil {
		return offset, err
	}
	s.KeyType = KeyType(keyType)

	keyLength, n, err := encoding.Varint(b[offset:])
	offset += n
	if err != nil {
		return offset, err
	}
	s.KeyLength = keyLength

	dataTypes, n, err := encoding.Varint(b[offset:])
	offset += n
	if err != nil {
		return offset, err
	}

	expiry, n, err := encoding.Varint(b[offset:])
	offset += n
	if err != nil {
		return offset, err
	}
	s.Expiry = expiry

	// The data types are values from 0 to 64. Currently only 24 are implemented,
	// but we iterate over all possible values to capture potentially missing ones.
	for i := 0; i < 64; i++ {
		if (dataTypes>>i)&1 == 1 {

			d := DataTypeDefinition{
				DataType: DataType(i),
			}

			info := d.DataType.New()
			if info == nil {
				return offset, fmt.Errorf("unknown data type: %v", d.DataType)
			}

			if d.DataType.IsDelay() {
				counter, n, err := encoding.Varint(b[offset:])
				offset += n
				if err != nil {
					return offset, err
				}
				d.Counter = counter

				period, n, err := encoding.Varint(b[offset:])
				offset += n
				if err != nil {
					return offset, err
				}
				d.Period = period
			}

			s.DataTypes = append(s.DataTypes, d)
		}
	}
	return offset, nil
}

type EntryUpdate struct {
	StickTable        *Definition
	WithLocalUpdateID bool
	WithExpiry        bool

	LocalUpdateID uint32
	Key           MapKey
	Data          []MapData
	Expiry        uint32
}

func (e *EntryUpdate) String() string {
	var data []string
	for i, d := range e.Data {
		data = append(data, fmt.Sprintf("%s: %s", e.StickTable.DataTypes[i].DataType.String(), d.String()))
	}

	return fmt.Sprintf("EntryUpdate %d: %s - %s", e.LocalUpdateID, e.Key, strings.Join(data, " | "))
}

func (e *EntryUpdate) Unmarshal(b []byte) (int, error) {
	var offset int
	// We already have a correct update ID loaded from the caller,
	// so we just override it when the message has its own
	if e.WithLocalUpdateID {
		e.LocalUpdateID = binary.BigEndian.Uint32(b[offset:])
		offset += 4
	}

	if e.WithExpiry {
		e.Expiry = binary.BigEndian.Uint32(b[offset:])
		offset += 4
	}

	e.Key = e.StickTable.KeyType.New()
	if e.Key == nil {
		return offset, fmt.Errorf("unknown key type: %v", e.StickTable.KeyType)
	}

	n, err := e.Key.Unmarshal(b[offset:], e.StickTable.KeyLength)
	if err != nil {
		return offset, err
	}
	offset += n

	for _, dataType := range e.StickTable.DataTypes {
		data := dataType.DataType.New()
		if data == nil {
			return offset, fmt.Errorf("unknown data type: %v", dataType)
		}

		n, err := data.Unmarshal(b[offset:])
		if err != nil {
			return offset, err
		}
		offset += n

		e.Data = append(e.Data, data)
	}

	return offset, nil
}
