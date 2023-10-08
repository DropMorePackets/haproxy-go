package spop

import (
	"encoding/hex"
	"testing"

	criteo "github.com/criteo/haproxy-spoe-go"
	"github.com/dropmorepackets/haproxy-go/pkg/encoding"
	"github.com/negasus/haproxy-spoe-go/message"
)

var (
	// a raw message from the test environment at @babiel
	msgInput, _ = hex.DecodeString("06766572696679050c636f6f6b69655f76616c7565001064657374696e6174696f6e5f686f7374080d69702e62616269656c2e636f6d0b71756572795f76616c75650009736f757263655f6970060a0900010866726f6e74656e64082067656e66726f6e74656e645f36303031302d64646f735f62616269656c5f6970")
	dis         Dispatcher
)

type Dispatcher struct {
}

func (d *Dispatcher) ServeCriteo(messages *criteo.MessageIterator) ([]criteo.Action, error) {
	for messages.Next() {
		for messages.Message.Args.Next() {
		}
	}

	if err := messages.Error(); err != nil {
		return nil, err
	}

	return nil, nil
}

func (d *Dispatcher) Servedropmorepackets(w *encoding.ActionWriter, m *encoding.Message) {
	k := encoding.AcquireKVEntry()
	defer encoding.ReleaseKVEntry(k)

	for m.KV.Next(k) {
	}

	if err := m.KV.Error(); err != nil {
		panic(err)
	}
}

func (d *Dispatcher) ServeNegasus(m *message.Messages) {
	for i := 0; i < m.Len(); i++ {
		m, err := m.GetByIndex(i)
		if err != nil {
			panic(err)
		}

		for range m.KV.Data() {
		}
	}
}

func BenchmarkCriteo(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = dis.ServeCriteo(criteo.NewMessageIterator(msgInput))
		}
	})
}

func BenchmarkNegasus(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		m := message.NewMessages()
		for pb.Next() {
			if err := m.Decode(msgInput); err != nil {
				b.Fatal(err)
			}

			dis.ServeNegasus(m)
			*m = (*m)[:0] // let's be fair and reuse the slice.
		}
	})
}

func Benchmarkdropmorepackets(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		// I am unfair against myself as these structures aren't always
		// reacquired, but let's do it anyway.
		for pb.Next() {
			m := encoding.AcquireMessage()
			// we don't write any actions right now
			w := encoding.AcquireActionWriter(nil, 0)
			s := encoding.AcquireMessageScanner(msgInput)

			for s.Next(m) {
				dis.Servedropmorepackets(w, m)
			}

			if err := s.Error(); err != nil {
				b.Fatal(err)
			}

			encoding.ReleaseMessageScanner(s)
			encoding.ReleaseActionWriter(w)
			encoding.ReleaseMessage(m)
		}
	})
}
