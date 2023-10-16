package sticktable

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/dropmorepackets/haproxy-go/pkg/encoding"
)

type DataTypeDefinition struct {
	DataType DataType
	Counter  int64
	Period   int64
}

type Definition struct {
	StickTableID int64
	Name         string
	KeyType      KeyType
	KeyLength    int64
	DataTypes    []DataTypeDefinition
	Expiry       int64
}

func (s *Definition) Unmarshal(r *bufio.Reader) error {
	// We don't need the Message length
	_, err := encoding.ReadVarint(r)
	if err != nil {
		return err
	}

	stickTableID, err := encoding.ReadVarint(r)
	if err != nil {
		return err
	}
	s.StickTableID = stickTableID

	nameLength, err := encoding.ReadVarint(r)
	if err != nil {
		return err
	}

	name := make([]byte, nameLength)
	if _, err := r.Read(name); err != nil {
		return err
	}
	s.Name = string(name)

	keyType, err := encoding.ReadVarint(r)
	if err != nil {
		return err
	}
	s.KeyType = KeyType(keyType)

	keyLength, err := encoding.ReadVarint(r)
	if err != nil {
		return err
	}
	s.KeyLength = keyLength

	dataTypes, err := encoding.ReadVarint(r)
	if err != nil {
		return err
	}

	expiry, err := encoding.ReadVarint(r)
	if err != nil {
		return err
	}
	s.Expiry = expiry

	for dataType := range DataTypes {
		if (dataTypes>>dataType)&1 == 1 {

			d := DataTypeDefinition{
				DataType: dataType,
			}

			info, ok := DataTypes[dataType]
			if !ok {
				return fmt.Errorf("unknown data type: %v", dataType)
			}

			if info.IsDelay {
				counter, err := encoding.ReadVarint(r)
				if err != nil {
					return err
				}
				d.Counter = counter

				period, err := encoding.ReadVarint(r)
				if err != nil {
					return err
				}
				d.Period = period
			}

			s.DataTypes = append(s.DataTypes, d)
		}
	}
	return nil
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
		dt := e.StickTable.DataTypes[i].DataType
		name := DataTypes[dt].Name
		data = append(data, fmt.Sprintf("%s: %s", name, d.String()))
	}

	return fmt.Sprintf("EntryUpdate %d: %s - %s", e.LocalUpdateID, e.Key, strings.Join(data, " | "))
}

func (e *EntryUpdate) Unmarshal(r *bufio.Reader) error {
	// We don't need the length
	_, err := encoding.ReadVarint(r)
	if err != nil {
		return err
	}

	// We already have a correct update ID loaded from the caller,
	// so we just override it when the message has its own
	if e.WithLocalUpdateID {
		localUpdateID := make([]byte, 4)
		if _, err := r.Read(localUpdateID); err != nil {
			return err
		}
		e.LocalUpdateID = binary.BigEndian.Uint32(localUpdateID)
	}

	if e.WithExpiry {
		expiry := make([]byte, 4)
		if _, err := r.Read(expiry); err != nil {
			return err
		}
		e.Expiry = binary.BigEndian.Uint32(expiry)
	}

	New, ok := KeyTypes[e.StickTable.KeyType]
	if !ok {
		return fmt.Errorf("unknown key type: %v", e.StickTable.KeyType)
	}

	var key = New()
	if err := key.Unmarshal(r, e.StickTable.KeyLength); err != nil {
		return err
	}
	e.Key = key

	for _, dataType := range e.StickTable.DataTypes {
		info, ok := DataTypes[dataType.DataType]
		if !ok {
			return fmt.Errorf("unknown data type: %v", dataType)
		}

		var data = info.New()
		if err := data.Unmarshal(r); err != nil {
			return err
		}
		e.Data = append(e.Data, data)
	}

	return nil
}
