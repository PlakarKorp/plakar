package snapshot

import (
	"errors"

	"github.com/PlakarKorp/plakar/btree"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/vmihailenco/msgpack/v5"
)

var (
	ErrReadOnly = errors.New("read-only store")
)

// RepositoryStore implements btree.Storer
type SnapshotStore[K any, V any] struct {
	readonly bool
	blobtype resources.Type
	snap     *Snapshot
}

func (s *SnapshotStore[K, V]) Get(sum objects.MAC) (*btree.Node[K, objects.MAC, V], error) {
	bytes, err := s.snap.repository.GetBlobBytes(s.blobtype, sum)
	if err != nil {
		return nil, err
	}
	node := &btree.Node[K, objects.MAC, V]{}
	err = msgpack.Unmarshal(bytes, node)
	return node, err
}

func (s *SnapshotStore[K, V]) Update(sum objects.MAC, node *btree.Node[K, objects.MAC, V]) error {
	return ErrReadOnly
}

func (s *SnapshotStore[K, V]) Put(node *btree.Node[K, objects.MAC, V]) (objects.MAC, error) {
	if s.readonly {
		return objects.MAC{}, ErrReadOnly
	}

	bytes, err := msgpack.Marshal(node)
	if err != nil {
		return objects.MAC{}, err
	}

	mac := s.snap.repository.ComputeMAC(bytes)
	return mac, s.snap.repository.PutBlobIfNotExists(s.blobtype, mac, bytes)
}

// persistIndex saves a btree[K, P, V] index to the snapshot.  The
// pointer type P is converted to a MAC.
func persistIndex[K any, P comparable, VA, VB any](snap *Snapshot, tree *btree.BTree[K, P, VA], rootres, noderes resources.Type, conv func(VA) (VB, error)) (mac objects.MAC, err error) {
	root, err := btree.Persist(tree, &SnapshotStore[K, VB]{
		readonly: false,
		blobtype: noderes,
		snap:     snap,
	}, conv)
	if err != nil {
		return
	}

	bytes, err := msgpack.Marshal(&btree.BTree[K, objects.MAC, VB]{
		Order: tree.Order,
		Root:  root,
	})
	if err != nil {
		return
	}

	mac = snap.repository.ComputeMAC(bytes)
	return mac, snap.repository.PutBlobIfNotExists(rootres, mac, bytes)
}

func persistMACIndex[K any, P comparable](snap *Snapshot, tree *btree.BTree[K, P, []byte], rootres, noderes, entryres resources.Type) (objects.MAC, error) {
	return persistIndex(snap, tree, rootres, noderes,
		func(data []byte) (objects.MAC, error) {
			mac := snap.repository.ComputeMAC(data)
			return mac, snap.repository.PutBlobIfNotExists(entryres, mac, data)
		})
}
