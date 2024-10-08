// Code generated by "stringer -type KeyType -output constants_string.go"; DO NOT EDIT.

package sticktable

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[KeyTypeAny-0]
	_ = x[KeyTypeBoolean-1]
	_ = x[KeyTypeSignedInteger-2]
	_ = x[KeyTypeAddress-3]
	_ = x[KeyTypeIPv4Address-4]
	_ = x[KeyTypeIPv6Address-5]
	_ = x[KeyTypeString-6]
	_ = x[KeyTypeBinary-7]
	_ = x[KeyTypeMethod-8]
}

const _KeyType_name = "KeyTypeAnyKeyTypeBooleanKeyTypeSignedIntegerKeyTypeAddressKeyTypeIPv4AddressKeyTypeIPv6AddressKeyTypeStringKeyTypeBinaryKeyTypeMethod"

var _KeyType_index = [...]uint8{0, 10, 24, 44, 58, 76, 94, 107, 120, 133}

func (i KeyType) String() string {
	if i < 0 || i >= KeyType(len(_KeyType_index)-1) {
		return "KeyType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _KeyType_name[_KeyType_index[i]:_KeyType_index[i+1]]
}
