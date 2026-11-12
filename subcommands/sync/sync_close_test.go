package sync

import (
	"context"
	"maps"
	"strings"
	"sync/atomic"
	"testing"

	bfs "github.com/PlakarKorp/integrations/fs/storage"
	"github.com/PlakarKorp/kloset/connectors/storage"
	"github.com/stretchr/testify/require"
)

// openPeerStores counts the stores opened through the "synctrackfs" scheme
// and not closed yet.
var openPeerStores atomic.Int64

type trackedStore struct {
	storage.Store
}

func (s *trackedStore) Close(ctx context.Context) error {
	openPeerStores.Add(-1)
	return s.Store.Close(ctx)
}

func init() {
	storage.Register("synctrackfs", 0, func(ctx context.Context, proto string, storeConfig map[string]string) (storage.Store, error) {
		fsConfig := maps.Clone(storeConfig)
		fsConfig["location"] = "fs://" + strings.TrimPrefix(storeConfig["location"], "synctrackfs://")
		st, err := bfs.NewStore(ctx, "fs", fsConfig)
		if err != nil {
			return nil, err
		}
		openPeerStores.Add(1)
		return &trackedStore{Store: st}, nil
	})
}

func TestSyncClosesPeerStore(t *testing.T) {
	// cached also opens the peer store, and keeps it until it stops: check
	// once the fixture, registered after this, is torn down.
	t.Cleanup(func() {
		require.Zero(t, openPeerStores.Load(), "peer stores left open")
	})
	fixture := setupSync(t, nil, nil)
	fixture.peerArg = "synctrackfs://" + fixture.peerRepo.Root()

	runSync(t, fixture, []string{"to", fixture.peerArg})
}
