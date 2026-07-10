package spop

import "testing"

func newTestAsyncScheduler() *asyncScheduler {
	scheduler := &asyncScheduler{q: newQueue(2)}
	go scheduler.queueWorker()
	return scheduler
}

func TestQueueGetClearsConsumedSlot(t *testing.T) {
	q := newQueue(1)
	f := acquireFrame()
	defer releaseFrame(f)
	client := &protocolClient{}

	q.Put(f, client)
	got := q.Get()
	if got.f != f || got.pc != client {
		t.Fatal("queue returned an unexpected element")
	}
	if q.elems[0].f != nil || q.elems[0].pc != nil {
		t.Fatal("queue retained the consumed element")
	}
}
