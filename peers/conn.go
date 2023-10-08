package peers

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"syscall"
	"time"
)

type Conn struct {
	conn                net.Conn
	r                   *bufio.Reader
	nextHeartbeat       *time.Ticker
	lastMessageTimer    *time.Timer
	lastTableDefinition *StickTableDefinition
	lastEntryUpdate     *EntryUpdate

	handler Handler
}

func (c *Conn) Close() error {
	return c.conn.Close()
}

func (c *Conn) peerHandshake() error {
	scanner := bufio.NewScanner(c.r)
	//    protocol identifier   : HAProxyS
	//    version               : 2.1
	//    remote peer identifier: the peer name this "hello" message is sent to.
	//    local peer identifier : the name of the peer which sends this "hello" message.
	//    process ID            : the ID of the process handling this peer session.
	//    relative process ID   : the haproxy's relative process ID (0 if nbproc == 1).

	type handshake struct {
		protocolIdentifier  string
		version             string
		remotePeer          string
		localPeerIdentifier string
		processID           int
		relativeProcessID   int
	}

	var h handshake
	scanner.Scan()
	_, _ = fmt.Sscanf(scanner.Text(), "%s %s", &h.protocolIdentifier, &h.version)

	scanner.Scan()
	h.remotePeer = scanner.Text()

	scanner.Scan()
	_, _ = fmt.Sscanf(scanner.Text(), "%s %d %d", &h.localPeerIdentifier, &h.processID, &h.relativeProcessID)

	log.Printf("%+v", h)

	_, err := c.conn.Write([]byte(fmt.Sprintf("%d\n", StatusHandshakeSucceeded)))
	if err != nil {
		_ = c.conn.Close()
		return fmt.Errorf("handshake failed: %v", err)
	}

	return nil
}

func (c *Conn) Handshake() error {
	if err := c.peerHandshake(); err != nil {
		return err
	}

	c.resetHeartbeat()
	c.resetLastMessage()
	go c.heartbeat()
	go c.lastMessage()

	return nil
}

var unknownBuf []byte

// Read should be called in a loop. It handles all Messages and returns errors,
// which can be safely ignored. They are mostly for Informational purposes.
func (c *Conn) Read() error {
	defer func() {
		if len(unknownBuf) != 0 {
			log.Println(unknownBuf)
		}
	}()

	// All the messages are made at least of a two bytes length header.
	header := make([]byte, 2)
	_, err := c.r.Read(header)
	if err != nil {
		return err
	}

	c.resetLastMessage()

	switch m := MessageClass(header[0]); m {
	case MessageClassControl:
		unknownBuf = unknownBuf[:0]
		return c.controlMessage(ControlMessageType(header[1]))
	case MessageClassError:
		unknownBuf = unknownBuf[:0]
		return c.errorMessage(ErrorMessageType(header[1]))
	case MessageClassStickTableUpdates:
		unknownBuf = unknownBuf[:0]
		return c.stickTableUpdate(StickTableMessageType(header[1]))
	default:
		unknownBuf = append(unknownBuf, header...)
		return fmt.Errorf("unknown message class: %s", m)
	}
}

func (c *Conn) controlMessage(t ControlMessageType) error {
	switch t {
	case ControlMessageSyncRequest:
		_, _ = c.conn.Write([]byte{byte(MessageClassControl), byte(ControlMessageSyncPartial)})
		return nil
	case ControlMessageSyncFinished:
		return nil
	case ControlMessageSyncPartial:
		return nil
	case ControlMessageSyncConfirmed:
		return nil
	case ControlMessageHeartbeat:
		return nil
	}

	return fmt.Errorf("unknown control message type: %s", t)
}

func (c *Conn) stickTableUpdate(t StickTableMessageType) error {
	switch t {
	case StickTableMessageStickTableDefinition:
		var std StickTableDefinition
		if err := std.Unmarshal(c.r); err != nil {
			return err
		}

		c.lastTableDefinition = &std

		//log.Printf("%+v", std)

		return nil
	case StickTableMessageStickTableSwitch:
		panic(t)
		return nil
	case StickTableMessageUpdateAcknowledge:
		panic(t)
		return nil
	case StickTableMessageEntryUpdate,
		StickTableMessageUpdateTimed,
		StickTableMessageIncrementalEntryUpdate,
		StickTableMessageIncrementalEntryUpdateTimed:
		return c.stickTableEntryUpdate(t)
		// Just continue to the next switch statement
	default:
		return fmt.Errorf("unknown stick-table message type: %s", t)
	}

	return nil
}

func (c *Conn) stickTableEntryUpdate(t StickTableMessageType) error {
	e := EntryUpdate{
		StickTable: c.lastTableDefinition,
	}

	if c.lastEntryUpdate != nil {
		e.LocalUpdateID = c.lastEntryUpdate.LocalUpdateID + 1
	}

	switch t {
	case StickTableMessageEntryUpdate:
		e.withLocalUpdateID = true
	case StickTableMessageUpdateTimed:
		e.withLocalUpdateID = true
		e.withExpiry = true
	case StickTableMessageIncrementalEntryUpdate:
	case StickTableMessageIncrementalEntryUpdateTimed:
		e.withExpiry = true
	}

	if err := e.Unmarshal(c.r); err != nil {
		return err
	}

	c.lastEntryUpdate = &e

	c.handler.Update(&e)

	return nil
}

func (c *Conn) errorMessage(t ErrorMessageType) error {
	switch t {
	case ErrorMessageProtocol:
		return fmt.Errorf("protocol error")
	case ErrorMessageSizeLimit:
		return fmt.Errorf("message size limit")
	}

	return fmt.Errorf("unknown error message type: %s", t)
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
			_ = c.conn.Close()
			return
		}
	}
}

func (c *Conn) lastMessage() {
	<-c.lastMessageTimer.C
	log.Println("last message timer expired: closing connection")
	_ = c.conn.Close()
}

func (c *Conn) serve() {
	defer c.Close()

	if err := c.Handshake(); err != nil {
		panic(err)
	}

	for {
		err := c.Read()
		if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF || errors.Is(err, syscall.ECONNRESET)) {
			return
		}

		if err != nil {
			panic(err)
		}
	}
}
