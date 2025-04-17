package snapshot

import (
	"bytes"
	"strings"

	"github.com/PlakarKorp/plakar/btree"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/resources"
)

func (snap *Snapshot) getidx(name, kind string) (objects.MAC, bool) {
	source := snap.Header.GetSource(0)
	for i := range source.Indexes {
		if source.Indexes[i].Name == name && source.Indexes[i].Type == kind {
			return source.Indexes[i].Value, true
		}
	}
	return objects.MAC{}, false
}

func (snap *Snapshot) ContentTypeIdx() (*btree.BTree[string, objects.MAC, objects.MAC], error) {
	mac, found := snap.getidx("content-type", "btree")
	if !found {
		return nil, nil
	}

	d, err := snap.repository.GetBlobBytes(resources.RT_BTREE_ROOT, mac)
	if err != nil {
		return nil, err
	}

	store := SnapshotStore[string, objects.MAC]{
		blobtype:   resources.RT_BTREE_NODE,
		snapReader: snap,
	}
	return btree.Deserialize(bytes.NewReader(d), &store, strings.Compare)
}

func (snap *Snapshot) MACIdx() (*btree.BTree[objects.MACTuple, objects.MAC, struct{}], objects.MAC, error) {
	mac, found := snap.getidx("mac", "btree")
	if !found {
		return nil, objects.MAC{}, nil
	}

	d, err := snap.repository.GetBlobBytes(resources.RT_BTREE_ROOT, mac)
	if err != nil {
		return nil, objects.MAC{}, err
	}

	store := SnapshotStore[objects.MACTuple, struct{}]{
		blobtype: resources.RT_BTREE_NODE,
		snapReader: snap,
	}
	it, err := btree.Deserialize(bytes.NewReader(d), &store, objects.MACTupleCompare)
	return it, mac, err
}
