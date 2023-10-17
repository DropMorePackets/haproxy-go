package spop

import (
	"log"
	"runtime"
	"sync"
)

type queue struct {
	elems        []*frame
	lock         sync.RWMutex
	notEmptyCond *sync.Cond
	notFullCond  *sync.Cond
	tail         int
	head         int
	size         int
}

func newQueue(cap int) *queue {
	q := &queue{
		elems: make([]*frame, cap),
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

func (bq *queue) Put(f *frame) {
	bq.lock.Lock()
	defer bq.lock.Unlock()

	for bq.isFull() {
		bq.notFullCond.Wait()
	}

	bq.elems[bq.tail] = f
	bq.tail = (bq.tail + 1) % len(bq.elems)
	bq.size++

	bq.notEmptyCond.Signal()
}

func (bq *queue) Get() *frame {
	bq.lock.Lock()
	defer bq.lock.Unlock()

	defer bq.notFullCond.Signal()

	for bq.isEmpty() {
		bq.notEmptyCond.Wait()
	}

	item := bq.elems[bq.head]
	bq.head = (bq.head + 1) % len(bq.elems)
	bq.size--

	if item == nil {
		panic("invalid item")
	}

	return item
}

type asyncScheduler struct {
	q  *queue
	pc *protocolClient
}

func newAsyncScheduler(pc *protocolClient) *asyncScheduler {
	a := asyncScheduler{
		q:  newQueue(runtime.NumCPU() * 2),
		pc: pc,
	}

	for i := 0; i < runtime.NumCPU(); i++ {
		go a.queueWorker()
	}

	return &a
}

func (a *asyncScheduler) queueWorker() {
	for {
		f := a.q.Get()
		if err := a.pc.frameHandler(f); err != nil {
			log.Println(err)
			continue
		}
	}
}

func (a *asyncScheduler) schedule(f *frame) {
	a.q.Put(f)
}
