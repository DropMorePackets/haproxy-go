package spop

import (
	"log"
	"runtime"
	"sync"
)

type queue struct {
	notEmptyCond *sync.Cond
	notFullCond  *sync.Cond
	elems        []queueElem
	tail         int
	head         int
	size         int
	lock         sync.RWMutex
}

type queueElem struct {
	f  *frame
	pc *protocolClient
}

func newQueue(cap int) *queue {
	q := &queue{
		elems: make([]queueElem, cap),
	}

	q.notEmptyCond = sync.NewCond(&q.lock)
	q.notFullCond = sync.NewCond(&q.lock)

	return q
}

func (bq *queue) isFull() bool {
	return bq.size == len(bq.elems)
}

func (bq *queue) isEmpty() bool {
	return bq.size <= 0
}

func (bq *queue) Put(f *frame, pc *protocolClient) {
	bq.lock.Lock()
	defer bq.lock.Unlock()

	for bq.isFull() {
		bq.notFullCond.Wait()
	}

	bq.elems[bq.tail] = queueElem{f, pc}
	bq.tail = (bq.tail + 1) % len(bq.elems)
	bq.size++

	bq.notEmptyCond.Signal()
}

func (bq *queue) Get() queueElem {
	bq.lock.Lock()
	defer bq.lock.Unlock()

	defer bq.notFullCond.Signal()

	for bq.isEmpty() {
		bq.notEmptyCond.Wait()
	}

	item := bq.elems[bq.head]
	bq.head = (bq.head + 1) % len(bq.elems)
	bq.size--

	return item
}

type asyncScheduler struct {
	q *queue
}

func newAsyncScheduler() *asyncScheduler {
	a := asyncScheduler{
		q: newQueue(runtime.NumCPU() * 2),
	}

	for i := 0; i < runtime.NumCPU(); i++ {
		go a.queueWorker()
	}

	return &a
}

func (a *asyncScheduler) queueWorker() {
	for {
		qe := a.q.Get()
		// Use wrap panic to prevent loosing worker goroutines to panics
		err := wrapPanic(func() error {
			return qe.pc.frameHandler(qe.f)
		})
		if err != nil {
			log.Println(err)
			continue
		}
	}
}

func (a *asyncScheduler) schedule(f *frame, pc *protocolClient) {
	a.q.Put(f, pc)
}
