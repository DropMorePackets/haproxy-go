package sticktable

//go:generate stringer -type MessageType
//go:generate stringer -type KeyType
//go:generate stringer -type DataType

// MessageType represents the stick-table update message types.
// There exits five types of such stick-table update messages:
// +-----------+--------------------------------+
// | type byte |          signification         |
// +-----------+--------------------------------+
// |    128    |          Entry update          |
// +-----------+--------------------------------+
// |    129    |    Incremental entry update    |
// +-----------+--------------------------------+
// |    130    |     Stick-table definition     |
// +-----------+--------------------------------+
// |    131    |   Stick-table switch (unused)  |
// +-----------+--------------------------------+
// |    133    | Update message acknowledgement |
// +-----------+--------------------------------+
type MessageType byte

// Stick table messages
const (
	//	PEER_MSG_STKT_UPDATE = 0x80,
	MessageEntryUpdate MessageType = iota + 0x80
	//	PEER_MSG_STKT_INCUPDATE,
	MessageIncrementalEntryUpdate
	//	PEER_MSG_STKT_DEFINE,
	MessageStickTableDefinition
	//	PEER_MSG_STKT_SWITCH,
	MessageStickTableSwitch
	//	PEER_MSG_STKT_ACK,
	MessageUpdateAcknowledge
	//	PEER_MSG_STKT_UPDATE_TIMED,
	MessageUpdateTimed
	//	PEER_MSG_STKT_INCUPDATE_TIMED,
	MessageIncrementalEntryUpdateTimed
)

type KeyType int

// This is the different key types of the stick tables.
// Same definitions as in HAProxy sources.
const (
	//	SMP_T_ANY,       /* any type */
	KeyTypeAny KeyType = iota
	//	SMP_T_BOOL,      /* boolean */
	KeyTypeBoolean
	//	SMP_T_SINT,      /* signed 64bits integer type */
	KeyTypeSignedInteger
	//	SMP_T_ADDR,      /* ipv4 or ipv6, only used for input type compatibility */
	KeyTypeAddress
	//	SMP_T_IPV4,      /* ipv4 type */
	KeyTypeIPv4Address
	//	SMP_T_IPV6,      /* ipv6 type */
	KeyTypeIPv6Address
	//	SMP_T_STR,       /* char string type */
	KeyTypeString
	//	SMP_T_BIN,       /* buffer type */
	KeyTypeBinary
	//	SMP_T_METH,      /* contain method */
	KeyTypeMethod
)

var KeyTypes = map[KeyType]func() MapKey{
	KeyTypeSignedInteger: func() MapKey { return new(SignedIntegerKey) },
	KeyTypeIPv4Address:   func() MapKey { return new(IPv4AddressKey) },
	KeyTypeIPv6Address:   func() MapKey { return new(IPv6AddressKey) },
	KeyTypeString:        func() MapKey { return new(StringKey) },
	KeyTypeBinary:        func() MapKey { return new(BinaryKey) },
}

type DataType int

// The types of extra data we can store in a stick table
const (
	//	STKTABLE_DT_SERVER_ID,    /* the server ID to use with this stream if > 0 */
	DataTypeServerId DataType = iota
	//	STKTABLE_DT_GPT0,         /* General Purpose Flag 0. */
	DataTypeGPT0
	//	STKTABLE_DT_GPC0,         /* General Purpose Counter 0 (unsigned 32-bit integer) */
	DataTypeGPC0
	//	STKTABLE_DT_GPC0_RATE,    /* General Purpose Counter 0's event rate */
	DataTypeGPC0Rate
	//	STKTABLE_DT_CONN_CNT,     /* cumulated number of connections */
	DataTypeConnectionsCounter
	//	STKTABLE_DT_CONN_RATE,    /* incoming connection rate */
	DataTypeConnectionRate
	//	STKTABLE_DT_CONN_CUR,     /* concurrent number of connections */
	DataTypeNumberOfCurrentConnections
	//	STKTABLE_DT_SESS_CNT,     /* cumulated number of sessions (accepted connections) */
	DataTypeSessionsCounter
	//	STKTABLE_DT_SESS_RATE,    /* accepted sessions rate */
	DataTypeSessionRate
	//	STKTABLE_DT_HTTP_REQ_CNT, /* cumulated number of incoming HTTP requests */
	DataTypeHttpRequestsCounter
	//	STKTABLE_DT_HTTP_REQ_RATE,/* incoming HTTP request rate */
	DataTypeHttpRequestsRate
	//	STKTABLE_DT_HTTP_ERR_CNT, /* cumulated number of HTTP requests errors (4xx) */
	DataTypeErrorsCounter
	//	STKTABLE_DT_HTTP_ERR_RATE,/* HTTP request error rate */
	DataTypeErrorsRate
	//	STKTABLE_DT_BYTES_IN_CNT, /* cumulated bytes count from client to servers */
	DataTypeBytesInCounter
	//	STKTABLE_DT_BYTES_IN_RATE,/* bytes rate from client to servers */
	DataTypeBytesInRate
	//	STKTABLE_DT_BYTES_OUT_CNT,/* cumulated bytes count from servers to client */
	DataTypeBytesOutCounter
	//	STKTABLE_DT_BYTES_OUT_RATE,/* bytes rate from servers to client */
	DataTypeBytesOutRate
	//	STKTABLE_DT_GPC1,         /* General Purpose Counter 1 (unsigned 32-bit integer) */
	DataTypeGPC1
	//	STKTABLE_DT_GPC1_RATE,    /* General Purpose Counter 1's event rate */
	DataTypeGPC1Rate
	//	STKTABLE_DT_SERVER_KEY,   /* The server key */
	DataTypeServerKey
	//	STKTABLE_DT_HTTP_FAIL_CNT, /* cumulated number of HTTP server failures */
	DataTypeHttpFailCounter
	//	STKTABLE_DT_HTTP_FAIL_RATE,/* HTTP server failures rate */
	DataTypeHttpFailRate
	//	STKTABLE_DT_GPT,           /* array of gpt */
	DataTypeGPTArray
	//	STKTABLE_DT_GPC,           /* array of gpc */
	DataTypeGPCArray
	//	STKTABLE_DT_GPC_RATE,      /* array of gpc_rate */
	DataTypeGPCRateArray
)

// 	[STKTABLE_DT_SERVER_ID]     = { .name = "server_id",      .std_type = STD_T_SINT  },
//	[STKTABLE_DT_GPT0]          = { .name = "gpt0",           .std_type = STD_T_UINT  },
//	[STKTABLE_DT_GPC0]          = { .name = "gpc0",           .std_type = STD_T_UINT  },
//	[STKTABLE_DT_GPC0_RATE]     = { .name = "gpc0_rate",      .std_type = STD_T_FRQP, .arg_type = ARG_T_DELAY  },
//	[STKTABLE_DT_CONN_CNT]      = { .name = "conn_cnt",       .std_type = STD_T_UINT  },
//	[STKTABLE_DT_CONN_RATE]     = { .name = "conn_rate",      .std_type = STD_T_FRQP, .arg_type = ARG_T_DELAY  },
//	[STKTABLE_DT_CONN_CUR]      = { .name = "conn_cur",       .std_type = STD_T_UINT, .is_local = 1 },
//	[STKTABLE_DT_SESS_CNT]      = { .name = "sess_cnt",       .std_type = STD_T_UINT  },
//	[STKTABLE_DT_SESS_RATE]     = { .name = "sess_rate",      .std_type = STD_T_FRQP, .arg_type = ARG_T_DELAY  },
//	[STKTABLE_DT_HTTP_REQ_CNT]  = { .name = "http_req_cnt",   .std_type = STD_T_UINT  },
//	[STKTABLE_DT_HTTP_REQ_RATE] = { .name = "http_req_rate",  .std_type = STD_T_FRQP, .arg_type = ARG_T_DELAY  },
//	[STKTABLE_DT_HTTP_ERR_CNT]  = { .name = "http_err_cnt",   .std_type = STD_T_UINT  },
//	[STKTABLE_DT_HTTP_ERR_RATE] = { .name = "http_err_rate",  .std_type = STD_T_FRQP, .arg_type = ARG_T_DELAY  },
//	[STKTABLE_DT_BYTES_IN_CNT]  = { .name = "bytes_in_cnt",   .std_type = STD_T_ULL   },
//	[STKTABLE_DT_BYTES_IN_RATE] = { .name = "bytes_in_rate",  .std_type = STD_T_FRQP, .arg_type = ARG_T_DELAY },
//	[STKTABLE_DT_BYTES_OUT_CNT] = { .name = "bytes_out_cnt",  .std_type = STD_T_ULL   },
//	[STKTABLE_DT_BYTES_OUT_RATE]= { .name = "bytes_out_rate", .std_type = STD_T_FRQP, .arg_type = ARG_T_DELAY },
//	[STKTABLE_DT_GPC1]          = { .name = "gpc1",           .std_type = STD_T_UINT  },
//	[STKTABLE_DT_GPC1_RATE]     = { .name = "gpc1_rate",      .std_type = STD_T_FRQP, .arg_type = ARG_T_DELAY  },
//	[STKTABLE_DT_SERVER_KEY]    = { .name = "server_key",     .std_type = STD_T_DICT  },
//	[STKTABLE_DT_HTTP_FAIL_CNT] = { .name = "http_fail_cnt",  .std_type = STD_T_UINT  },
//	[STKTABLE_DT_HTTP_FAIL_RATE]= { .name = "http_fail_rate", .std_type = STD_T_FRQP, .arg_type = ARG_T_DELAY  },
//	[STKTABLE_DT_GPT]           = { .name = "gpt",            .std_type = STD_T_UINT, .is_array = 1 },
//	[STKTABLE_DT_GPC]           = { .name = "gpc",            .std_type = STD_T_UINT, .is_array = 1 },
//	[STKTABLE_DT_GPC_RATE]      = { .name = "gpc_rate",       .std_type = STD_T_FRQP, .is_array = 1, .arg_type = ARG_T_DELAY },

type dataTypeInfo struct {
	Name    string
	New     func() MapData
	IsDelay bool
}

var DataTypes = map[DataType]dataTypeInfo{
	DataTypeServerId:                   {Name: "server_id", New: NewSignedIntegerData},
	DataTypeGPT0:                       {Name: "gpt0", New: NewUnsignedIntegerData},
	DataTypeGPC0:                       {Name: "gpc0", New: NewUnsignedIntegerData},
	DataTypeGPC0Rate:                   {Name: "gpc0_rate", New: NewFreqData, IsDelay: true},
	DataTypeConnectionsCounter:         {Name: "conn_cnt", New: NewUnsignedIntegerData},
	DataTypeConnectionRate:             {Name: "conn_rate", New: NewFreqData, IsDelay: true},
	DataTypeNumberOfCurrentConnections: {Name: "conn_cur", New: NewUnsignedIntegerData},
	DataTypeSessionsCounter:            {Name: "sess_cnt", New: NewUnsignedIntegerData},
	DataTypeSessionRate:                {Name: "sess_rate", New: NewFreqData, IsDelay: true},
	DataTypeHttpRequestsCounter:        {Name: "http_req_cnt", New: NewUnsignedIntegerData},
	DataTypeHttpRequestsRate:           {Name: "http_req_rate", New: NewFreqData, IsDelay: true},
	DataTypeErrorsCounter:              {Name: "http_err_cnt", New: NewUnsignedIntegerData},
	DataTypeErrorsRate:                 {Name: "http_err_rate", New: NewFreqData, IsDelay: true},
	DataTypeBytesInCounter:             {Name: "bytes_in_cnt", New: NewUnsignedLongLongData},
	DataTypeBytesInRate:                {Name: "bytes_in_rate", New: NewFreqData, IsDelay: true},
	DataTypeBytesOutCounter:            {Name: "bytes_out_cnt", New: NewUnsignedLongLongData},
	DataTypeBytesOutRate:               {Name: "bytes_out_rate", New: NewFreqData, IsDelay: true},
	DataTypeGPC1:                       {Name: "gpc1", New: NewUnsignedIntegerData},
	DataTypeGPC1Rate:                   {Name: "gpc1_rate", New: NewFreqData, IsDelay: true},
	DataTypeServerKey:                  {Name: "server_key", New: NewDictData},
	DataTypeHttpFailCounter:            {Name: "http_fail_cnt", New: NewUnsignedIntegerData},
	DataTypeHttpFailRate:               {Name: "http_fail_rate", New: NewFreqData, IsDelay: true},
	DataTypeGPTArray:                   {Name: "gpt", New: NewDictData},
	DataTypeGPCArray:                   {Name: "gpc", New: NewDictData},
	DataTypeGPCRateArray:               {Name: "gpc_rate", New: NewDictData},
}

// The equivalent standard types of the stored data
var (
	//	STD_T_SINT = 0,           /* data is of type signed int */
	NewSignedIntegerData = func() MapData { return new(SignedIntegerData) }
	//	STD_T_UINT,               /* data is of type unsigned int */
	NewUnsignedIntegerData = func() MapData { return new(UnsignedIntegerData) }
	//	STD_T_ULL,                /* data is of type unsigned long long */
	NewUnsignedLongLongData = func() MapData { return new(UnsignedLongLongData) }
	//	STD_T_FRQP,               /* data is of type freq_ctr */
	NewFreqData = func() MapData { return new(FreqData) }
	//	STD_T_DICT,               /* data is of type key of dictionary entry */
	NewDictData = func() MapData { return new(DictData) }
)
