package btree

type item[K any, P comparable, V any] struct {
	dirty bool
	node  *Node[K, P, V]
}

type lru[P comparable] struct {
	ptr  P
	next *lru[P]
}

type cache[K any, P comparable, V any] struct {
	size   int
	target int
	store  Storer[K, P, V]
	items  map[P]*item[K, P, V]
	head   *lru[P]
	tail   *lru[P]

	hits int
	miss int
}

func cachefor[K any, P comparable, V any](store Storer[K, P, V], order int) *cache[K, P, V] {
	target := order
	return &cache[K, P, V]{
		target: target,
		store:  store,
		items:  make(map[P]*item[K, P, V], target),
	}
}

func (c *cache[K, P, V]) cache(ptr P, node *Node[K, P, V]) {
	c.size++
	c.items[ptr] = &item[K, P, V]{node: node}
	lru := &lru[P]{ptr: ptr}

	if c.head == nil {
		c.head = lru
		c.tail = lru
	} else {
		c.tail.next = lru
		c.tail = lru
	}
}

func (c *cache[K, P, V]) flush(ptr P) error {
	item := c.items[ptr]
	if item.dirty {
		if err := c.store.Update(ptr, item.node); err != nil {
			return err
		}
	}

	delete(c.items, ptr)
	c.size--
	return nil
}

func (c *cache[K, P, V]) Get(ptr P) (*Node[K, P, V], error) {
	if item, ok := c.items[ptr]; ok {
		c.hits++
		return item.node, nil
	}

	c.miss++

	node, err := c.store.Get(ptr)
	if err != nil {
		return nil, err
	}

	if c.size == c.target {
		if err := c.flush(c.head.ptr); err != nil {
			return nil, err
		}
		c.head = c.head.next
	}

	c.cache(ptr, node)
	return node, nil
}

func (c *cache[K, P, V]) Update(ptr P, node *Node[K, P, V]) error {
	if item, ok := c.items[ptr]; ok {
		item.node = node
		item.dirty = true
		return nil
	}

	return c.store.Update(ptr, node)
}

func (c *cache[K, P, V]) Put(node *Node[K, P, V]) (P, error) {
	return c.store.Put(node)
}

func (c *cache[K, P, V]) flushall() error {
	for el := c.head; el != nil; el = el.next {
		if err := c.flush(el.ptr); err != nil {
			return err
		}
	}
	c.head = nil
	c.tail = nil
	return nil
}
