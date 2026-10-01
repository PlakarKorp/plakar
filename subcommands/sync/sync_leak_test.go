package sync

import (
	"context"
	"maps"
	"testing"

	bfs "github.com/PlakarKorp/integrations/fs/storage"
	"github.com/PlakarKorp/kloset/connectors/storage"
	"github.com/stretchr/testify/require"
)

// capturedPeerConfigs records, in call order, a snapshot of the config map
// exactly as seen by the peer store's backend constructor -- i.e. what
// sync.go actually handed to storage.Open() -- for every store opened
// through the "synctestfs" scheme registered below.
var capturedPeerConfigs []map[string]string

func init() {
	storage.Register("synctestfs", 0, func(ctx context.Context, proto string, storeConfig map[string]string) (storage.Store, error) {
		capturedPeerConfigs = append(capturedPeerConfigs, maps.Clone(storeConfig))

		fsConfig := maps.Clone(storeConfig)
		fsConfig["location"] = "fs://" + storeConfig["location"][len("synctestfs://"):]
		return bfs.NewStore(ctx, "fs", fsConfig)
	})
}

// TestSyncPeerPassphraseNeverReachesTheBackend guards against the class of
// bug fixed (only partially) by 60dacfc3 ("sync: remove passphrase from the
// peer repository options"): storeConfig["passphrase"] /
// storeConfig["passphrase_cmd"] must never reach the peer store's backend
// constructor, since every storage backend (including third-party connector
// plugins) receives the whole config map verbatim.
//
// Execute() strips them before its storage.Open() call, but Parse() calls
// storage.Open() on the very same storeConfig *before* it ever reads (let
// alone deletes) "passphrase" -- so this currently fails on the first
// (Parse) capture.
func TestSyncPeerPassphraseNeverReachesTheBackend(t *testing.T) {
	capturedPeerConfigs = nil

	peerPassphrase := []byte("QsDfG654321&^%*%!")
	fixture := setupSync(t, nil, peerPassphrase)

	// Repoint the peer config at our spy scheme instead of plain "fs://" so
	// every storage.Open() on the peer goes through the capture above.
	peerCfg := fixture.localCtx.Config.Repositories["peer"]
	peerCfg["location"] = "synctestfs://" + fixture.peerRepo.Root()

	runSync(t, fixture, []string{"to", fixture.peerArg})

	// Parse(), Execute() and the "cached" state rebuild each open the peer
	// store; exactly how many that is is an implementation detail, but none
	// of them may ever see the passphrase.
	require.NotEmpty(t, capturedPeerConfigs, "the peer store was never opened")
	for i, cfg := range capturedPeerConfigs {
		_, hasPass := cfg["passphrase"]
		_, hasPassCmd := cfg["passphrase_cmd"]
		require.False(t, hasPass, "call #%d: backend received the peer passphrase in cleartext", i)
		require.False(t, hasPassCmd, "call #%d: backend received passphrase_cmd", i)
	}
}
