package peers

//go:generate stringer -type HandshakeStatus
//go:generate stringer -type MessageClass
//go:generate stringer -type ControlMessageType
//go:generate stringer -type ErrorMessageType
//go:generate stringer -type StickTableMessageType
//go:generate stringer -type StickTableKeyType
//go:generate stringer -type StickTableDataType

// HandshakeStatus represents the Handshake States
// +-------------+---------------------------------+
// | status code |         signification           |
// +-------------+---------------------------------+
// |     200     |      Handshake succeeded        |
// +-------------+---------------------------------+
// |     300     |        Try again later          |
// +-------------+---------------------------------+
// |     501     |         Protocol error          |
// +-------------+---------------------------------+
// |     502     |          Bad version            |
// +-------------+---------------------------------+
// |     503     | Local peer identifier mismatch  |
// +-------------+---------------------------------+
// |     504     | Remote peer identifier mismatch |
// +-------------+---------------------------------+
type HandshakeStatus int

const (
	StatusHandshakeSucceeded           HandshakeStatus = 200
	StatusTryAgainLater                HandshakeStatus = 300
	StatusProtocolError                HandshakeStatus = 501
	StatusBadVersion                   HandshakeStatus = 502
	StatusLocalPeerIdentifierMismatch  HandshakeStatus = 503
	StatusRemotePeerIdentifierMismatch HandshakeStatus = 504
)

// MessageClass represents the message classes.
// There exist four classes of messages:
// +------------+---------------------+--------------+
// | class byte |    signification    | message size |
// +------------+---------------------+--------------+
// |      0     |      control        |   fixed (2)  |
// +------------+---------------------+--------------|
// |      1     |       error         |   fixed (2)  |
// +------------+---------------------+--------------|
// |     10     | stick-table updates |   variable   |
// +------------+---------------------+--------------|
// |    255     |      reserved       |              |
// +------------+---------------------+--------------+
type MessageClass byte

// HAPPP message classes
const (
	//	PEER_MSG_CLASS_CONTROL    = 0,
	MessageClassControl MessageClass = iota
	//	PEER_MSG_CLASS_ERROR,
	MessageClassError
	//	PEER_MSG_CLASS_STICKTABLE = 0x0a,
	MessageClassStickTableUpdates MessageClass = 10
	//	PEER_MSG_CLASS_RESERVED   = 0xff,
	MessageClassReserved MessageClass = 0x0ff
)

// ControlMessageType represents the control message types.
// There exists five types of such control messages:
// +------------+--------------------------------------------------------+
// | type byte  |                   signification                        |
// +------------+--------------------------------------------------------+
// |      0     | synchronisation request: ask a remote peer for a full  |
// |            | synchronization                                        |
// +------------+--------------------------------------------------------+
// |      1     | synchronization finished: signal a remote peer that    |
// |            | local updates have been pushed and local is considered |
// |            | up to date.                                            |
// +------------+--------------------------------------------------------+
// |      2     | synchronization partial: signal a remote peer that     |
// |            | local updates have been pushed and local is not        |
// |            | considered up to date.                                 |
// +------------+--------------------------------------------------------+
// |      3     | synchronization confirmed: acknowledge a finished or   |
// |            | partial synchronization message.                       |
// +------------+--------------------------------------------------------+
// |      4     | Heartbeat message.                                     |
// +------------+--------------------------------------------------------+
type ControlMessageType byte

// Control messages
const (
	//	PEER_MSG_CTRL_RESYNCREQ = 0,
	ControlMessageSyncRequest ControlMessageType = iota
	//	PEER_MSG_CTRL_RESYNCFINISHED,
	ControlMessageSyncFinished
	//	PEER_MSG_CTRL_RESYNCPARTIAL,
	ControlMessageSyncPartial
	//	PEER_MSG_CTRL_RESYNCCONFIRM,
	ControlMessageSyncConfirmed
	//	PEER_MSG_CTRL_HEARTBEAT,
	ControlMessageHeartbeat
)

// ErrorMessageType represents the error message types.
// There exits two types of such error messages:
// +-----------+------------------+
// | type byte |   signification  |
// +-----------+------------------+
// |      0    |  protocol error  |
// +-----------+------------------+
// |      1    | size limit error |
// +-----------+------------------+
type ErrorMessageType byte

// Error messages
const (
	//	PEER_MSG_ERR_PROTOCOL = 0,
	ErrorMessageProtocol ErrorMessageType = iota
	//	PEER_MSG_ERR_SIZELIMIT,
	ErrorMessageSizeLimit
)

// StickTableMessageType represents the stick-table update message types.
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
type StickTableMessageType byte

// Stick table messages
const (
	//	PEER_MSG_STKT_UPDATE = 0x80,
	StickTableMessageEntryUpdate StickTableMessageType = iota + 0x80
	//	PEER_MSG_STKT_INCUPDATE,
	StickTableMessageIncrementalEntryUpdate
	//	PEER_MSG_STKT_DEFINE,
	StickTableMessageStickTableDefinition
	//	PEER_MSG_STKT_SWITCH,
	StickTableMessageStickTableSwitch
	//	PEER_MSG_STKT_ACK,
	StickTableMessageUpdateAcknowledge
	//	PEER_MSG_STKT_UPDATE_TIMED,
	StickTableMessageUpdateTimed
	//	PEER_MSG_STKT_INCUPDATE_TIMED,
	StickTableMessageIncrementalEntryUpdateTimed
)

type StickTableKeyType int

// This is the different key types of the stick tables.
// Same definitions as in HAProxy sources.
const (
	//	SMP_T_ANY,       /* any type */
	StickTableKeyTypeAny StickTableKeyType = iota
	//	SMP_T_BOOL,      /* boolean */
	StickTableKeyTypeBoolean
	//	SMP_T_SINT,      /* signed 64bits integer type */
	StickTableKeyTypeSignedInteger
	//	SMP_T_ADDR,      /* ipv4 or ipv6, only used for input type compatibility */
	StickTableKeyTypeAddress
	//	SMP_T_IPV4,      /* ipv4 type */
	StickTableKeyTypeIPv4Address
	//	SMP_T_IPV6,      /* ipv6 type */
	StickTableKeyTypeIPv6Address
	//	SMP_T_STR,       /* char string type */
	StickTableKeyTypeString
	//	SMP_T_BIN,       /* buffer type */
	StickTableKeyTypeBinary
	//	SMP_T_METH,      /* contain method */
	StickTableKeyTypeMethod
)

var StickTableKeyTypes = map[StickTableKeyType]func() MapKey{
	StickTableKeyTypeSignedInteger: func() MapKey { return new(SignedIntegerKey) },
	StickTableKeyTypeIPv4Address:   func() MapKey { return new(IPv4AddressKey) },
	StickTableKeyTypeIPv6Address:   func() MapKey { return new(IPv6AddressKey) },
	StickTableKeyTypeString:        func() MapKey { return new(StringKey) },
	StickTableKeyTypeBinary:        func() MapKey { return new(BinaryKey) },
}

type StickTableDataType int

// The types of extra data we can store in a stick table
const (
	//	STKTABLE_DT_SERVER_ID,    /* the server ID to use with this stream if > 0 */
	StickTableDataTypeServerId StickTableDataType = iota
	//	STKTABLE_DT_GPT0,         /* General Purpose Flag 0. */
	StickTableDataTypeGPT0
	//	STKTABLE_DT_GPC0,         /* General Purpose Counter 0 (unsigned 32-bit integer) */
	StickTableDataTypeGPC0
	//	STKTABLE_DT_GPC0_RATE,    /* General Purpose Counter 0's event rate */
	StickTableDataTypeGPC0Rate
	//	STKTABLE_DT_CONN_CNT,     /* cumulated number of connections */
	StickTableDataTypeConnectionsCounter
	//	STKTABLE_DT_CONN_RATE,    /* incoming connection rate */
	StickTableDataTypeConnectionRate
	//	STKTABLE_DT_CONN_CUR,     /* concurrent number of connections */
	StickTableDataTypeNumberOfCurrentConnections
	//	STKTABLE_DT_SESS_CNT,     /* cumulated number of sessions (accepted connections) */
	StickTableDataTypeSessionsCounter
	//	STKTABLE_DT_SESS_RATE,    /* accepted sessions rate */
	StickTableDataTypeSessionRate
	//	STKTABLE_DT_HTTP_REQ_CNT, /* cumulated number of incoming HTTP requests */
	StickTableDataTypeHttpRequestsCounter
	//	STKTABLE_DT_HTTP_REQ_RATE,/* incoming HTTP request rate */
	StickTableDataTypeHttpRequestsRate
	//	STKTABLE_DT_HTTP_ERR_CNT, /* cumulated number of HTTP requests errors (4xx) */
	StickTableDataTypeErrorsCounter
	//	STKTABLE_DT_HTTP_ERR_RATE,/* HTTP request error rate */
	StickTableDataTypeErrorsRate
	//	STKTABLE_DT_BYTES_IN_CNT, /* cumulated bytes count from client to servers */
	StickTableDataTypeBytesInCounter
	//	STKTABLE_DT_BYTES_IN_RATE,/* bytes rate from client to servers */
	StickTableDataTypeBytesInRate
	//	STKTABLE_DT_BYTES_OUT_CNT,/* cumulated bytes count from servers to client */
	StickTableDataTypeBytesOutCounter
	//	STKTABLE_DT_BYTES_OUT_RATE,/* bytes rate from servers to client */
	StickTableDataTypeBytesOutRate
	//	STKTABLE_DT_GPC1,         /* General Purpose Counter 1 (unsigned 32-bit integer) */
	StickTableDataTypeGPC1
	//	STKTABLE_DT_GPC1_RATE,    /* General Purpose Counter 1's event rate */
	StickTableDataTypeGPC1Rate
	//	STKTABLE_DT_SERVER_KEY,   /* The server key */
	StickTableDataTypeServerKey
	//	STKTABLE_DT_HTTP_FAIL_CNT, /* cumulated number of HTTP server failures */
	StickTableDataTypeHttpFailCounter
	//	STKTABLE_DT_HTTP_FAIL_RATE,/* HTTP server failures rate */
	StickTableDataTypeHttpFailRate
	//	STKTABLE_DT_GPT,           /* array of gpt */
	StickTableDataTypeGPTArray
	//	STKTABLE_DT_GPC,           /* array of gpc */
	StickTableDataTypeGPCArray
	//	STKTABLE_DT_GPC_RATE,      /* array of gpc_rate */
	StickTableDataTypeGPCRateArray
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

type stickTableDataTypeInfo struct {
	Name    string
	New     func() MapData
	IsDelay bool
}

var StickTableDataTypes = map[StickTableDataType]stickTableDataTypeInfo{
	StickTableDataTypeServerId:                   {Name: "server_id", New: NewSignedIntegerData},
	StickTableDataTypeGPT0:                       {Name: "gpt0", New: NewUnsignedIntegerData},
	StickTableDataTypeGPC0:                       {Name: "gpc0", New: NewUnsignedIntegerData},
	StickTableDataTypeGPC0Rate:                   {Name: "gpc0_rate", New: NewFreqData, IsDelay: true},
	StickTableDataTypeConnectionsCounter:         {Name: "conn_cnt", New: NewUnsignedIntegerData},
	StickTableDataTypeConnectionRate:             {Name: "conn_rate", New: NewFreqData, IsDelay: true},
	StickTableDataTypeNumberOfCurrentConnections: {Name: "conn_cur", New: NewUnsignedIntegerData},
	StickTableDataTypeSessionsCounter:            {Name: "sess_cnt", New: NewUnsignedIntegerData},
	StickTableDataTypeSessionRate:                {Name: "sess_rate", New: NewFreqData, IsDelay: true},
	StickTableDataTypeHttpRequestsCounter:        {Name: "http_req_cnt", New: NewUnsignedIntegerData},
	StickTableDataTypeHttpRequestsRate:           {Name: "http_req_rate", New: NewFreqData, IsDelay: true},
	StickTableDataTypeErrorsCounter:              {Name: "http_err_cnt", New: NewUnsignedIntegerData},
	StickTableDataTypeErrorsRate:                 {Name: "http_err_rate", New: NewFreqData, IsDelay: true},
	StickTableDataTypeBytesInCounter:             {Name: "bytes_in_cnt", New: NewUnsignedLongLongData},
	StickTableDataTypeBytesInRate:                {Name: "bytes_in_rate", New: NewFreqData, IsDelay: true},
	StickTableDataTypeBytesOutCounter:            {Name: "bytes_out_cnt", New: NewUnsignedLongLongData},
	StickTableDataTypeBytesOutRate:               {Name: "bytes_out_rate", New: NewFreqData, IsDelay: true},
	StickTableDataTypeGPC1:                       {Name: "gpc1", New: NewUnsignedIntegerData},
	StickTableDataTypeGPC1Rate:                   {Name: "gpc1_rate", New: NewFreqData, IsDelay: true},
	StickTableDataTypeServerKey:                  {Name: "server_key", New: NewDictData},
	StickTableDataTypeHttpFailCounter:            {Name: "http_fail_cnt", New: NewUnsignedIntegerData},
	StickTableDataTypeHttpFailRate:               {Name: "http_fail_rate", New: NewFreqData, IsDelay: true},
	StickTableDataTypeGPTArray:                   {Name: "gpt", New: NewDictData},
	StickTableDataTypeGPCArray:                   {Name: "gpc", New: NewDictData},
	StickTableDataTypeGPCRateArray:               {Name: "gpc_rate", New: NewDictData},
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
