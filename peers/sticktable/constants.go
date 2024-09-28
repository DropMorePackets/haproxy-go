package sticktable

import "strconv"

//go:generate stringer -type KeyType -output constants_string.go

type KeyType int

// This is the different key types of the stick tables.
// Same definitions as in HAProxy sources.
const (
	KeyTypeAny KeyType = iota
	KeyTypeBoolean
	KeyTypeSignedInteger
	KeyTypeAddress
	KeyTypeIPv4Address
	KeyTypeIPv6Address
	KeyTypeString
	KeyTypeBinary
	KeyTypeMethod
)

func (t KeyType) New() MapKey {
	switch t {
	case KeyTypeSignedInteger:
		return new(SignedIntegerKey)
	case KeyTypeIPv4Address:
		return new(IPv4AddressKey)
	case KeyTypeIPv6Address:
		return new(IPv6AddressKey)
	case KeyTypeString:
		return new(StringKey)
	case KeyTypeBinary:
		return new(BinaryKey)
	default:
		panic("unknown key type: " + t.String())
	}
}

type DataType int

func (d DataType) String() string {
	switch d {
	case DataTypeServerId:
		return "server_id"
	case DataTypeGPT0:
		return "gpt0"
	case DataTypeGPC0:
		return "gpc0"
	case DataTypeGPC0Rate:
		return "gpc0_rate"
	case DataTypeConnectionsCounter:
		return "conn_cnt"
	case DataTypeConnectionRate:
		return "conn_rate"
	case DataTypeNumberOfCurrentConnections:
		return "conn_cur"
	case DataTypeSessionsCounter:
		return "sess_cnt"
	case DataTypeSessionRate:
		return "sess_rate"
	case DataTypeHttpRequestsCounter:
		return "http_req_cnt"
	case DataTypeHttpRequestsRate:
		return "http_req_rate"
	case DataTypeErrorsCounter:
		return "http_err_cnt"
	case DataTypeErrorsRate:
		return "http_err_rate"
	case DataTypeBytesInCounter:
		return "bytes_in_cnt"
	case DataTypeBytesInRate:
		return "bytes_in_rate"
	case DataTypeBytesOutCounter:
		return "bytes_out_cnt"
	case DataTypeBytesOutRate:
		return "bytes_out_rate"
	case DataTypeGPC1:
		return "gpc1"
	case DataTypeGPC1Rate:
		return "gpc1_rate"
	case DataTypeServerKey:
		return "server_key"
	case DataTypeHttpFailCounter:
		return "http_fail_cnt"
	case DataTypeHttpFailRate:
		return "http_fail_rate"
	case DataTypeGPTArray:
		return "gpt"
	case DataTypeGPCArray:
		return "gpc"
	case DataTypeGPCRateArray:
		return "gpc_rate"
	case DataTypeGlitchCounter:
		return "glitch_cnt"
	case DataTypeGlitchRate:
		return "glitch_rate"
	default:
		return "StickTableUpdateMessageType(" + strconv.FormatInt(int64(d), 10) + ")"
	}
}

func (d DataType) IsDelay() bool {
	switch d {
	case DataTypeGPC0Rate,
		DataTypeConnectionRate,
		DataTypeSessionRate,
		DataTypeHttpRequestsRate,
		DataTypeErrorsRate,
		DataTypeBytesInRate,
		DataTypeBytesOutRate,
		DataTypeGPC1Rate,
		DataTypeHttpFailRate:
		return true
	default:
		return false
	}
}

// The types of extra data we can store in a stick table
const (
	// DataTypeServerId represents the server ID to use with this
	// represents a stream if > 0
	DataTypeServerId DataType = iota
	// DataTypeGPT0 represents a General Purpose Flag 0.
	DataTypeGPT0
	// DataTypeGPC0 represents a General Purpose Counter 0 (unsigned 32-bit integer)
	DataTypeGPC0
	// DataTypeGPC0Rate represents a General Purpose Counter 0's event rate
	DataTypeGPC0Rate
	// DataTypeConnectionsCounter represents a cumulated number of connections
	DataTypeConnectionsCounter
	// DataTypeConnectionRate represents an incoming connection rate
	DataTypeConnectionRate
	// DataTypeNumberOfCurrentConnections represents a concurrent number of connections
	DataTypeNumberOfCurrentConnections
	// DataTypeSessionsCounter represents a cumulated number of sessions (accepted connections)
	DataTypeSessionsCounter
	// DataTypeSessionRate represents an accepted sessions rate
	DataTypeSessionRate
	// DataTypeHttpRequestsCounter represents a cumulated number of incoming HTTP requests
	DataTypeHttpRequestsCounter
	// DataTypeHttpRequestsRate represents an incoming HTTP request rate
	DataTypeHttpRequestsRate
	// DataTypeErrorsCounter represents a cumulated number of HTTP requests errors (4xx)
	DataTypeErrorsCounter
	// DataTypeErrorsRate represents an HTTP request error rate
	DataTypeErrorsRate
	// DataTypeBytesInCounter represents a cumulated bytes count from client to servers
	DataTypeBytesInCounter
	// DataTypeBytesInRate represents a bytes rate from client to servers
	DataTypeBytesInRate
	// DataTypeBytesOutCounter represents a cumulated bytes count from servers to client
	DataTypeBytesOutCounter
	// DataTypeBytesOutRate represents a bytes rate from servers to client
	DataTypeBytesOutRate
	// DataTypeGPC1 represents a General Purpose Counter 1 (unsigned 32-bit integer)
	DataTypeGPC1
	// DataTypeGPC1Rate represents a General Purpose Counter 1's event rate
	DataTypeGPC1Rate
	// DataTypeServerKey represents the server key
	DataTypeServerKey
	// DataTypeHttpFailCounter represents a cumulated number of HTTP server failures
	DataTypeHttpFailCounter
	// DataTypeHttpFailRate represents an HTTP server failures rate
	DataTypeHttpFailRate
	// DataTypeGPTArray represents an array of gpt
	DataTypeGPTArray
	// DataTypeGPCArray represents an array of gpc
	DataTypeGPCArray
	// DataTypeGPCRateArray represents an array of gpc_rate
	DataTypeGPCRateArray
	// DataTypeGlitchCounter represents a cumulated number of front glitches
	DataTypeGlitchCounter
	// DataTypeGlitchRate represents a rate of front glitches
	DataTypeGlitchRate
)

func (d DataType) New() MapData {
	switch d {
	case DataTypeServerId:
		return new(SignedIntegerData)
	case DataTypeGPT0:
		return new(UnsignedIntegerData)
	case DataTypeGPC0:
		return new(UnsignedIntegerData)
	case DataTypeGPC0Rate:
		return new(FreqData)
	case DataTypeConnectionsCounter:
		return new(UnsignedIntegerData)
	case DataTypeConnectionRate:
		return new(FreqData)
	case DataTypeNumberOfCurrentConnections:
		return new(UnsignedIntegerData)
	case DataTypeSessionsCounter:
		return new(UnsignedIntegerData)
	case DataTypeSessionRate:
		return new(FreqData)
	case DataTypeHttpRequestsCounter:
		return new(UnsignedIntegerData)
	case DataTypeHttpRequestsRate:
		return new(FreqData)
	case DataTypeErrorsCounter:
		return new(UnsignedIntegerData)
	case DataTypeErrorsRate:
		return new(FreqData)
	case DataTypeBytesInCounter:
		return new(UnsignedLongLongData)
	case DataTypeBytesInRate:
		return new(FreqData)
	case DataTypeBytesOutCounter:
		return new(UnsignedLongLongData)
	case DataTypeBytesOutRate:
		return new(FreqData)
	case DataTypeGPC1:
		return new(UnsignedIntegerData)
	case DataTypeGPC1Rate:
		return new(FreqData)
	case DataTypeServerKey:
		return new(DictData)
	case DataTypeHttpFailCounter:
		return new(UnsignedIntegerData)
	case DataTypeHttpFailRate:
		return new(FreqData)
	case DataTypeGPTArray:
		return new(DictData)
	case DataTypeGPCArray:
		return new(DictData)
	case DataTypeGPCRateArray:
		return new(DictData)
	case DataTypeGlitchCounter:
		return new(UnsignedIntegerData)
	case DataTypeGlitchRate:
		return new(FreqData)
	default:
		return nil
	}
}
