package encoding

import (
	"fmt"
	"net/netip"
	"reflect"
	"strings"
)

const tagName = "spoe"

// Unmarshal unmarshals KV entries from the scanner into the provided struct.
// The struct should have fields tagged with `spoe:"keyname"` to map KV entry
// names to struct fields.
//
// Supported field types:
//   - string, []byte (for DataTypeString and DataTypeBinary)
//   - int32, int64, uint32, uint64 (for integer types)
//   - bool (for DataTypeBool)
//   - netip.Addr (for DataTypeIPV4 and DataTypeIPV6)
//   - pointer types for optional fields (nil if key not found)
//
// Example:
//
//	type RequestData struct {
//	    Headers []byte    `spoe:"headers"`
//	    Status  int32     `spoe:"status-code"`
//	    IP      netip.Addr `spoe:"client-ip"`
//	    Optional *string  `spoe:"optional-field"`
//	}
func (k *KVScanner) Unmarshal(v any) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("unmarshal target must be a non-nil pointer to struct")
	}

	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("unmarshal target must be a pointer to struct")
	}

	rt := rv.Type()

	// Build a slice of field info to avoid string allocations during lookup
	type fieldInfo struct {
		keyStr    string // cached for NameEquals and error messages
		fieldIdx  int
		field     reflect.Value // cached to avoid repeated rv.Field() calls
		fieldKind reflect.Kind  // cached to avoid repeated Kind() calls
		isPointer bool          // cached to avoid repeated checks
	}
	fields := make([]fieldInfo, 0, rt.NumField())
	pointerFieldIndices := make([]int, 0, rt.NumField()) // track pointer field indices for final cleanup
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		tag := field.Tag.Get(tagName)
		if tag == "" || tag == "-" {
			continue
		}

		// Handle comma-separated options (e.g., "keyname,omitempty")
		// Use IndexByte to avoid allocation from strings.Split
		commaIdx := strings.IndexByte(tag, ',')
		var key string
		if commaIdx >= 0 {
			key = tag[:commaIdx]
		} else {
			key = tag
		}
		if key != "" {
			fv := rv.Field(i)
			fk := fv.Kind()
			isPtr := fk == reflect.Pointer
			fields = append(fields, fieldInfo{
				keyStr:    key,
				fieldIdx:  i,
				field:     fv,
				fieldKind: fk,
				isPointer: isPtr,
			})
			if isPtr {
				pointerFieldIndices = append(pointerFieldIndices, i)
			}
		}
	}

	entry := AcquireKVEntry()
	defer ReleaseKVEntry(entry)

	// Track which pointer fields have been set (to clear unset ones later)
	setPointerFields := make(map[int]bool, len(pointerFieldIndices))

	for k.Next(entry) {
		var fi *fieldInfo
		// Use NameEquals to avoid string allocation during lookup
		for i := range fields {
			if entry.NameEquals(fields[i].keyStr) {
				fi = &fields[i]
				break
			}
		}
		if fi == nil {
			// Unknown key, skip it
			continue
		}

		if !fi.field.CanSet() {
			return fmt.Errorf("field %s is not settable", rt.Field(fi.fieldIdx).Name)
		}

		if err := setFieldValue(fi.field, fi.fieldKind, entry); err != nil {
			return fmt.Errorf("field %s (key %q): %w", rt.Field(fi.fieldIdx).Name, fi.keyStr, err)
		}

		// Track if this is a pointer field that was set
		if fi.isPointer {
			setPointerFields[fi.fieldIdx] = true
		}
	}

	if err := k.Error(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	// Set pointer fields to nil if they weren't set (important for pooled structs)
	// Only iterate through known pointer fields instead of all fields
	for _, idx := range pointerFieldIndices {
		if !setPointerFields[idx] {
			rv.Field(idx).Set(reflect.Zero(rt.Field(idx).Type))
		}
	}

	return nil
}

func setFieldValue(field reflect.Value, fieldKind reflect.Kind, entry *KVEntry) error {
	fieldType := field.Type()

	// Handle pointer types
	if fieldKind == reflect.Pointer {
		if entry.dataType == DataTypeNull {
			field.Set(reflect.Zero(fieldType))
			return nil
		}

		// Create new value of the pointed-to type
		elemType := fieldType.Elem()
		elemValue := reflect.New(elemType).Elem()
		if err := setValue(elemValue, elemType.Kind(), entry); err != nil {
			return err
		}
		field.Set(elemValue.Addr())
		return nil
	}

	return setValue(field, fieldKind, entry)
}

var netipAddrType = reflect.TypeOf((*netip.Addr)(nil)).Elem()

func setValue(field reflect.Value, fieldKind reflect.Kind, entry *KVEntry) error {
	fieldType := field.Type()

	switch fieldKind {
	case reflect.String:
		if entry.dataType != DataTypeString {
			return fmt.Errorf("expected string, got %d", entry.dataType)
		}
		// Value() returns string for DataTypeString
		field.SetString(entry.Value().(string))

	case reflect.Slice:
		if fieldType.Elem().Kind() != reflect.Uint8 {
			return fmt.Errorf("unsupported slice type: %s", fieldType)
		}
		// []byte
		if entry.dataType != DataTypeString && entry.dataType != DataTypeBinary {
			return fmt.Errorf("expected string or binary, got %d", entry.dataType)
		}
		// Copy the bytes to avoid referencing the underlying buffer
		val := entry.ValueBytes()
		cp := make([]byte, len(val))
		copy(cp, val)
		field.SetBytes(cp)

	case reflect.Int32:
		if entry.dataType != DataTypeInt32 {
			return fmt.Errorf("expected int32, got %d", entry.dataType)
		}
		field.SetInt(entry.ValueInt())

	case reflect.Int64:
		if entry.dataType != DataTypeInt64 {
			return fmt.Errorf("expected int64, got %d", entry.dataType)
		}
		field.SetInt(entry.ValueInt())

	case reflect.Uint32:
		if entry.dataType != DataTypeUInt32 {
			return fmt.Errorf("expected uint32, got %d", entry.dataType)
		}
		field.SetUint(uint64(entry.ValueInt()))

	case reflect.Uint64:
		if entry.dataType != DataTypeUInt64 {
			return fmt.Errorf("expected uint64, got %d", entry.dataType)
		}
		field.SetUint(uint64(entry.ValueInt()))

	case reflect.Bool:
		if entry.dataType != DataTypeBool {
			return fmt.Errorf("expected bool, got %d", entry.dataType)
		}
		field.SetBool(entry.ValueBool())

	default:
		// Check for netip.Addr (using cached type)
		if fieldType == netipAddrType {
			if entry.dataType != DataTypeIPV4 && entry.dataType != DataTypeIPV6 {
				return fmt.Errorf("expected IP address, got %d", entry.dataType)
			}
			addr := entry.ValueAddr()
			field.Set(reflect.ValueOf(addr))
			return nil
		}

		return fmt.Errorf("unsupported field type: %s", fieldType)
	}

	return nil
}
