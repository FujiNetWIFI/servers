package main

import (
	"sync"
)

// ringbuffer is a very niave ringbuffer that holds pointers to strings and returns
// them in the order they were added. It uses locking to prevent threading issues.
type ringbuffer struct {
	buffer []*string
	pos    int
	mtx    sync.RWMutex
}

func newRingBuffer(size int) *ringbuffer {
	return &ringbuffer{
		buffer: make([]*string, size),
		mtx:    sync.RWMutex{},
	}
}

func (rb *ringbuffer) add(s *string) {
	rb.mtx.Lock()
	defer rb.mtx.Unlock()

	rb.buffer[rb.pos] = s
	rb.pos++
	if rb.pos > len(rb.buffer)-1 {
		rb.pos = 0
	}
}

func (rb *ringbuffer) readAll() []*string {
	res := make([]*string, len(rb.buffer))

	rb.mtx.RLock()
	defer rb.mtx.RUnlock()
	copy(res[:len(rb.buffer)-rb.pos], rb.buffer[rb.pos:])
	copy(res[len(rb.buffer)-rb.pos:], rb.buffer[:rb.pos])

	return res
}
