package newenc

type dataType byte

const (
	dataTypeNull   dataType = 0
	dataTypeBool   dataType = 1
	dataTypeInt32  dataType = 2
	dataTypeUInt32 dataType = 3
	dataTypeInt64  dataType = 4
	dataTypeUInt64 dataType = 5
	dataTypeIPV4   dataType = 6
	dataTypeIPV6   dataType = 7
	dataTypeString dataType = 8
	dataTypeBinary dataType = 9

	dataTypeMask byte = 0x0F
	dataFlagTrue byte = 0x10
)
