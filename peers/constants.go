package peers

//go:generate stringer -type HandshakeStatus,MessageClass,ControlMessageType,ErrorMessageType,StickTableUpdateMessageType -output=constants_string.go

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
	HandshakeStatusHandshakeSucceeded           HandshakeStatus = 200
	HandshakeStatusTryAgainLater                HandshakeStatus = 300
	HandshakeStatusProtocolError                HandshakeStatus = 501
	HandshakeStatusBadVersion                   HandshakeStatus = 502
	HandshakeStatusLocalPeerIdentifierMismatch  HandshakeStatus = 503
	HandshakeStatusRemotePeerIdentifierMismatch HandshakeStatus = 504
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
	MessageClassReserved MessageClass = 255
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

// StickTableUpdateMessageType represents the stick-table update message types.
// There exist five types of such stick-table update messages:
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
// |    132    | Update message acknowledgement |
// +-----------+--------------------------------+
type StickTableUpdateMessageType byte

// Stick table messages
const (
	StickTableUpdateMessageTypeEntryUpdate StickTableUpdateMessageType = iota + 0x80
	StickTableUpdateMessageTypeIncrementalEntryUpdate
	StickTableUpdateMessageTypeStickTableDefinition
	StickTableUpdateMessageTypeStickTableSwitch
	StickTableUpdateMessageTypeUpdateAcknowledge
	StickTableUpdateMessageTypeUpdateTimed
	StickTableUpdateMessageTypeIncrementalEntryUpdateTimed
)
