package peers

import (
	"fmt"
	"log"

	"github.com/dropmorepackets/haproxy-go/peers/sticktable"
)

func (t ErrorMessageType) OnMessage(m *rawMessage, c *Conn) error {
	switch t {
	case ErrorMessageProtocol:
		return fmt.Errorf("protocol error")
	case ErrorMessageSizeLimit:
		return fmt.Errorf("message size limit")
	default:
		return fmt.Errorf("unknown error message type: %s", t)
	}
}

func (t ControlMessageType) OnMessage(m *rawMessage, c *Conn) error {
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
	default:
		return fmt.Errorf("unknown control message type: %s", t)
	}
}

func (t StickTableUpdateMessageType) OnMessage(m *rawMessage, c *Conn) error {
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
	log.Printf("%+v", e)

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

	c.handler.Update(&e)

	return nil
}
