package btree

func persist[K any, PA, PB comparable, VA, VB any](b *BTree[K, PA, VA], store Storer[K, PB, VB], conv func(VA) (VB, error), node *Node[K, PA, VA], lastptr **PB) (PB, error) {
	var ptrs []PB
	var zero PB
	var vals []VB

	for i := len(node.Pointers) - 1; i >= 0; i-- {
		child, err := b.cache.Get(node.Pointers[i])
		if err != nil {
			return zero, err
		}

		ptr, err := persist(b, store, conv, child, lastptr)
		if err != nil {
			return zero, err
		}
		if child.isleaf() {
			*lastptr = new(PB)
			**lastptr = ptr
		}
		ptrs = append(ptrs, ptr)
	}

	for i := range node.Values {
		val, err := conv(node.Values[i])
		if err != nil {
			return zero, err
		}
		vals = append(vals, val)
	}

	// reverse pointers
	for i := len(ptrs)/2 - 1; i >= 0; i-- {
		opp := len(ptrs) - 1 - i
		ptrs[i], ptrs[opp] = ptrs[opp], ptrs[i]
	}

	newnode := &Node[K, PB, VB]{
		Keys:     node.Keys,
		Values:   vals,
		Pointers: ptrs,
	}
	if node.isleaf() && *lastptr != nil {
		newnode.Next = *lastptr
	}
	return store.Put(newnode)
}

// Persist converts a BTree from one storage backend to another.  The
// given store only needs to provide a working Put method, since by
// design Persist inserts the nodes in post-order from the right-most
// leaf, in a way that's suitable for a content-addressed store, and
// never updates existing nodes nor retrieves inserted ones.
func Persist[K any, PA, PB comparable, VA, VB any](b *BTree[K, PA, VA], store Storer[K, PB, VB], conv func(VA) (VB, error)) (ptr PB, err error) {
	root, err := b.cache.Get(b.Root)
	if err != nil {
		return
	}

	var lastptr *PB
	return persist(b, store, conv, root, &lastptr)
}
