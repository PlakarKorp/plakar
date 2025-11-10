package pools

import (
	"bytes"
	"sync"
)

// BufferPool offers reusable bytes.Buffer instances with a configurable capacity.
type BufferPool struct {
	pool sync.Pool
	cap  int
}

// NewBufferPool builds a pool whose buffers start with the provided capacity.
func NewBufferPool(capacity int) *BufferPool {
	if capacity < 0 {
		capacity = 0
	}

	bp := &BufferPool{cap: capacity}
	bp.pool.New = func() any {
		buf := bytes.NewBuffer(make([]byte, 0, capacity))
		buf.Reset()
		return buf
	}
	return bp
}

// Get retrieves a clean buffer from the pool.
func (b *BufferPool) Get() *bytes.Buffer {
	buf := b.pool.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

// Put returns the buffer to the pool. Nil buffers are ignored.
func (b *BufferPool) Put(buf *bytes.Buffer) {
	if buf == nil {
		return
	}
	if b.cap > 0 && buf.Cap() > b.cap*4 {
		// Drop excessively large buffers to avoid unbounded growth.
		return
	}
	buf.Reset()
	b.pool.Put(buf)
}

// ReportPool is a generic pool wrapper built on sync.Pool.
type ReportPool[T any] struct {
	pool  sync.Pool
	reset func(T)
}

// NewReportPool constructs a typed pool. newFn must not be nil.
func NewReportPool[T any](newFn func() T, resetFn func(T)) *ReportPool[T] {
	if newFn == nil {
		panic("pools: newFn must not be nil")
	}
	return &ReportPool[T]{
		pool:  sync.Pool{New: func() any { return newFn() }},
		reset: resetFn,
	}
}

// Get returns an instance from the pool.
func (p *ReportPool[T]) Get() T {
	return p.pool.Get().(T)
}

// Put releases the instance back to the pool, applying the reset function when present.
func (p *ReportPool[T]) Put(value T) {
	if p.reset != nil {
		p.reset(value)
	}
	p.pool.Put(value)
}
