package peers

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/dropmorepackets/haproxy-go/peers/sticktable"
	"github.com/dropmorepackets/haproxy-go/pkg/encoding"
)

type Conn struct {
	ctx  context.Context
	conn net.Conn
	r    *bufio.Reader

	nextHeartbeat       *time.Ticker
	lastMessageTimer    *time.Timer
	lastTableDefinition *sticktable.Definition
	lastEntryUpdate     *sticktable.EntryUpdate

	handler Handler
}

func (c *Conn) Close() error {
	return c.conn.Close()
}

func (c *Conn) peerHandshake() error {
	var h Handshake
	if _, err := h.ReadFrom(c.r); err != nil {
		return err
	}

	if _, err := c.conn.Write([]byte(fmt.Sprintf("%d\n", HandshakeStatusHandshakeSucceeded))); err != nil {
		return fmt.Errorf("handshake failed: %v", err)
	}

	return nil
}

func (c *Conn) resetHeartbeat() {
	// a peer sends heartbeat messages to peers it is
	// connected to after periods of 3s of inactivity (i.e. when there is no
	// stick-table to synchronize for 3s).
	if c.nextHeartbeat == nil {
		c.nextHeartbeat = time.NewTicker(time.Second * 3)
		return
	}

	c.nextHeartbeat.Reset(time.Second * 3)
}

func (c *Conn) resetLastMessage() {
	// After a successful peer protocol handshake between two peers,
	// if one of them does not send any other peer
	// protocol messages (i.e. no heartbeat and no stick-table update messages)
	// during a 5s period, it is considered as no more alive by its remote peer
	// which closes the session and then tries to reconnect to the peer which
	// has just disappeared.
	if c.lastMessageTimer == nil {
		c.lastMessageTimer = time.NewTimer(time.Second * 5)
		return
	}

	c.lastMessageTimer.Reset(time.Second * 5)
}

func (c *Conn) heartbeat() {
	for range c.nextHeartbeat.C {
		_, err := c.conn.Write([]byte{byte(MessageClassControl), byte(ControlMessageHeartbeat)})
		if err != nil {
			_ = c.Close()
			return
		}
	}
}

func (c *Conn) lastMessage() {
	<-c.lastMessageTimer.C
	log.Println("last message timer expired: closing connection")
	_ = c.Close()
}

func (c *Conn) Serve() error {
	if err := c.peerHandshake(); err != nil {
		return fmt.Errorf("handshake: %v", err)
	}

	c.resetHeartbeat()
	c.resetLastMessage()
	go c.heartbeat()
	go c.lastMessage()

	for {
		var m rawMessage

		if _, err := m.ReadFrom(c.r); err != nil {
			if c.ctx.Err() != nil {
				return c.ctx.Err()
			}

			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return nil
			}

			return fmt.Errorf("reading message: %v", err)
		}

		c.resetLastMessage()
		if err := c.messageHandler(&m); err != nil {
			return fmt.Errorf("message handler: %v", err)
		}
	}
}

func (c *Conn) messageHandler(m *rawMessage) error {
	switch m.MessageClass {
	case MessageClassControl:
		return ControlMessageType(m.MessageType).OnMessage(m, c)
	case MessageClassError:
		return ErrorMessageType(m.MessageType).OnMessage(m, c)
	case MessageClassStickTableUpdates:
		return StickTableUpdateMessageType(m.MessageType).OnMessage(m, c)
	default:
		return fmt.Errorf("unknown message class: %s", m.MessageClass)
	}
}

type byteReader interface {
	io.ByteReader
	io.Reader
}

type rawMessage struct {
	MessageClass MessageClass
	MessageType  byte

	Data []byte
}

func (m *rawMessage) ReadFrom(r byteReader) (int64, error) {
	// All the messages are made at least of a two bytes length header.
	header := make([]byte, 2)
	n, err := io.ReadFull(r, header)
	if err != nil {
		return int64(n), err
	}

	m.MessageClass = MessageClass(header[0])
	m.MessageType = header[1]

	var readData int
	// All messages with type >= 128 have a payload
	if m.MessageType >= 128 {
		dataLength, err := encoding.ReadVarint(r)
		if err != nil {
			return int64(n), fmt.Errorf("failed decoding data length: %v", err)
		}

		m.Data = make([]byte, dataLength)
		readData, err = io.ReadFull(r, m.Data)
		if err != nil {
			return int64(n + readData), fmt.Errorf("failed reading message data: %v", err)
		}
		if uint64(readData) != dataLength {
			return int64(n + readData), fmt.Errorf("invalid amount read: %d != %d", dataLength, readData)
		}
	}

	return int64(n + readData), nil
}

// Handshake is composed by these fields:
//
//	protocol identifier   : HAProxyS
//	version               : 2.1
//	remote peer identifier: the peer name this "hello" message is sent to.
//	local peer identifier : the name of the peer which sends this "hello" message.
//	process ID            : the ID of the process handling this peer session.
//	relative process ID   : the haproxy's relative process ID (0 if nbproc == 1).
type Handshake struct {
	ProtocolIdentifier  string
	Version             string
	RemotePeer          string
	LocalPeerIdentifier string
	ProcessID           int
	RelativeProcessID   int
}

func (h *Handshake) ReadFrom(r io.Reader) (n int64, err error) {
	scanner := bufio.NewScanner(r)

	scanner.Scan()
	_, err = fmt.Sscanf(scanner.Text(), "%s %s", &h.ProtocolIdentifier, &h.Version)
	if err != nil {
		return -1, err
	}

	scanner.Scan()
	h.RemotePeer = scanner.Text()

	scanner.Scan()
	_, err = fmt.Sscanf(scanner.Text(), "%s %d %d", &h.LocalPeerIdentifier, &h.ProcessID, &h.RelativeProcessID)
	if err != nil {
		return -1, err
	}

	//TODO: find out how many bytes where read.
	return -1, scanner.Err()
}
