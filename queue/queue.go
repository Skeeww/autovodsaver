package queue

import (
	"sync"

	"enssat.tv/autovodsaver/constants"
)

type FifoQueue struct {
	mu     *sync.Mutex
	buffer []*constants.VideoWatched
}

func NewFifo() *FifoQueue {
	return &FifoQueue{
		mu:     &sync.Mutex{},
		buffer: make([]*constants.VideoWatched, 0),
	}
}

func (q *FifoQueue) Enqueue(element *constants.VideoWatched) {
	q.mu.Lock()
	q.buffer = append(q.buffer, element)
	q.mu.Unlock()
}

func (q *FifoQueue) Dequeue() *constants.VideoWatched {
	for len(q.buffer) == 0 {
		// Wait for an element to be pushed
	}
	q.mu.Lock()
	element := q.buffer[0]
	q.mu.Unlock()
	return element
}

func (q *FifoQueue) Size() int {
	q.mu.Lock()
	size := len(q.buffer)
	q.mu.Unlock()
	return size
}
