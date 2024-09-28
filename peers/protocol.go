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

type protocolClient struct {
	ctx       context.Context
	ctxCancel context.CancelFunc
	rw        io.ReadWriter
	br        *bufio.Reader

	nextHeartbeat       *time.Ticker
	lastMessageTimer    *time.Timer
	lastTableDefinition *sticktable.Definition
	lastEntryUpdate     *sticktable.EntryUpdate

	handler Handler
}

func newProtocolClient(ctx context.Context, rw io.ReadWriter, handler Handler) *protocolClient {
	var c protocolClient
	c.rw = rw
	c.br = bufio.NewReader(rw)
	c.handler = handler
	c.ctx, c.ctxCancel = context.WithCancel(ctx)
	return &c
}

func (c *protocolClient) Close() error {
	defer c.ctxCancel()
	if c.ctx.Err() != nil {
		return c.ctx.Err()
	}

	return c.handler.Close()
}

func (c *protocolClient) peerHandshake() error {
	var h Handshake
	if _, err := h.ReadFrom(c.br); err != nil {
		return err
	}

	c.handler.HandleHandshake(c.ctx, &h)

	if _, err := c.rw.Write([]byte(fmt.Sprintf("%d\n", HandshakeStatusHandshakeSucceeded))); err != nil {
		return fmt.Errorf("handshake failed: %v", err)
	}

	return nil
}

func (c *protocolClient) resetHeartbeat() {
	// a peer sends heartbeat messages to peers it is
	// connected to after periods of 3s of inactivity (i.e. when there is no
	// stick-table to synchronize for 3s).
	if c.nextHeartbeat == nil {
		c.nextHeartbeat = time.NewTicker(time.Second * 3)
		return
	}

	c.nextHeartbeat.Reset(time.Second * 3)
}

func (c *protocolClient) resetLastMessage() {
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

func (c *protocolClient) heartbeat() {
	for range c.nextHeartbeat.C {
		_, err := c.rw.Write([]byte{byte(MessageClassControl), byte(ControlMessageHeartbeat)})
		if err != nil {
			_ = c.Close()
			return
		}
	}
}

func (c *protocolClient) lastMessage() {
	<-c.lastMessageTimer.C
	log.Println("last message timer expired: closing connection")
	_ = c.Close()
}

func (c *protocolClient) Serve() error {
	if err := c.peerHandshake(); err != nil {
		return fmt.Errorf("handshake: %v", err)
	}

	c.resetHeartbeat()
	c.resetLastMessage()
	go c.heartbeat()
	go c.lastMessage()

	for {
		var m rawMessage

		if _, err := m.ReadFrom(c.br); err != nil {
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

func (c *protocolClient) messageHandler(m *rawMessage) error {
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

func (t ErrorMessageType) OnMessage(m *rawMessage, c *protocolClient) error {
	switch t {
	case ErrorMessageProtocol:
		return fmt.Errorf("protocol error")
	case ErrorMessageSizeLimit:
		return fmt.Errorf("message size limit")
	default:
		return fmt.Errorf("unknown error message type: %s", t)
	}
}

func (t ControlMessageType) OnMessage(m *rawMessage, c *protocolClient) error {
	switch t {
	case ControlMessageSyncRequest:
		_, _ = c.rw.Write([]byte{byte(MessageClassControl), byte(ControlMessageSyncPartial)})
		return nil
	case ControlMessageSyncFinished:
		return nil
	case ControlMessageSyncPartial:
		return nil
	case ControlMessageSyncConfirmed:
		return nil
	case ControlMessageHeartbeat:
		return nil
	default:
		return fmt.Errorf("unknown control message type: %s", t)
	}
}

func (t StickTableUpdateMessageType) OnMessage(m *rawMessage, c *protocolClient) error {
	switch t {
	case StickTableUpdateMessageTypeStickTableDefinition:
		var std sticktable.Definition
		if _, err := std.Unmarshal(m.Data); err != nil {
			return err
		}
		c.lastTableDefinition = &std

		return nil
	case StickTableUpdateMessageTypeStickTableSwitch:
		log.Printf("not implemented: %s", t)
		return nil
	case StickTableUpdateMessageTypeUpdateAcknowledge:
		log.Printf("not implemented: %s", t)
		return nil
	case StickTableUpdateMessageTypeEntryUpdate,
		StickTableUpdateMessageTypeUpdateTimed,
		StickTableUpdateMessageTypeIncrementalEntryUpdate,
		StickTableUpdateMessageTypeIncrementalEntryUpdateTimed:
		// All entry update messages are handled in a separate switch case
		// following this one.
		break
	default:
		return fmt.Errorf("unknown stick-table update message type: %s", t)
	}

	if c.lastTableDefinition == nil {
		return fmt.Errorf("cannot process entry update without table definition")
	}

	e := sticktable.EntryUpdate{
		StickTable: c.lastTableDefinition,
	}

	if c.lastEntryUpdate != nil {
		e.LocalUpdateID = c.lastEntryUpdate.LocalUpdateID + 1
	}

	switch t {
	case StickTableUpdateMessageTypeEntryUpdate:
		e.WithLocalUpdateID = true
	case StickTableUpdateMessageTypeUpdateTimed:
		e.WithLocalUpdateID = true
		e.WithExpiry = true
	case StickTableUpdateMessageTypeIncrementalEntryUpdate:
	case StickTableUpdateMessageTypeIncrementalEntryUpdateTimed:
		e.WithExpiry = true
	}

	if _, err := e.Unmarshal(m.Data); err != nil {
		return err
	}

	c.lastEntryUpdate = &e

	c.handler.HandleUpdate(c.ctx, &e)

	return nil
}
