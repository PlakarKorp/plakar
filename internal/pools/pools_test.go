package pools

import (
	"bytes"
	"sync"
	"testing"
)

func TestBufferPoolReuse(t *testing.T) {
	pool := NewBufferPool(64)

	buf := pool.Get()
	if buf.Len() != 0 {
		t.Fatalf("expected empty buffer, got %d", buf.Len())
	}

	_, _ = buf.WriteString("hello")
	pool.Put(buf)

	buf2 := pool.Get()
	if buf2.Len() != 0 {
		t.Fatalf("expected buffer reset, got len %d", buf2.Len())
	}

	if buf != buf2 {
		t.Fatalf("expected same buffer instance to be reused")
	}
}

func TestBufferPoolDropsHugeBuffers(t *testing.T) {
	pool := NewBufferPool(32)

	buf := bytes.NewBuffer(make([]byte, 1000))
	pool.Put(buf)

	buf2 := pool.Get()
	if buf2.Cap() > 128 {
		t.Fatalf("expected large buffer to be dropped, got cap %d", buf2.Cap())
	}
}

type dummy struct {
	value int
}

func TestReportPool(t *testing.T) {
	pool := NewReportPool(func() *dummy { return &dummy{} }, func(d *dummy) {
		d.value = 0
	})

	item := pool.Get()
	item.value = 42
	pool.Put(item)

	again := pool.Get()
	if again.value != 0 {
		t.Fatalf("reset function should clear value, got %d", again.value)
	}
}

func TestReportPoolConcurrency(t *testing.T) {
	pool := NewReportPool(func() *dummy { return &dummy{} }, func(d *dummy) { d.value = 0 })

	wg := sync.WaitGroup{}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			item := pool.Get()
			item.value = idx
			pool.Put(item)
		}(i)
	}

	wg.Wait()
}
