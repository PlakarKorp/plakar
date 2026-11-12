package ptar

import (
	"context"
	"maps"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	fsimporter "github.com/PlakarKorp/integrations/fs/importer"
	bfs "github.com/PlakarKorp/integrations/fs/storage"
	"github.com/PlakarKorp/kloset/connectors"
	"github.com/PlakarKorp/kloset/connectors/importer"
	"github.com/PlakarKorp/kloset/connectors/storage"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// openStores and openImporters count the connectors opened through the
// "ptartrackfs" scheme and not closed yet; peakStores and peakImporters record
// the most that were ever open at once.
var openStores, openImporters, peakStores, peakImporters atomic.Int64

func trackOpen(open, peak *atomic.Int64) {
	n := open.Add(1)
	for {
		p := peak.Load()
		if n <= p || peak.CompareAndSwap(p, n) {
			return
		}
	}
}

type trackedStore struct {
	storage.Store
}

func (s *trackedStore) Close(ctx context.Context) error {
	openStores.Add(-1)
	return s.Store.Close(ctx)
}

type trackedImporter struct {
	importer.Importer
}

func (i *trackedImporter) Close(ctx context.Context) error {
	openImporters.Add(-1)
	return i.Importer.Close(ctx)
}

func init() {
	storage.Register("ptartrackfs", 0, func(ctx context.Context, proto string, storeConfig map[string]string) (storage.Store, error) {
		fsConfig := maps.Clone(storeConfig)
		fsConfig["location"] = "fs://" + strings.TrimPrefix(storeConfig["location"], "ptartrackfs://")
		st, err := bfs.NewStore(ctx, "fs", fsConfig)
		if err != nil {
			return nil, err
		}
		trackOpen(&openStores, &peakStores)
		return &trackedStore{Store: st}, nil
	})
	importer.Register("ptartrackfs", 0, func(ctx context.Context, opts *connectors.Options, name string, config map[string]string) (importer.Importer, error) {
		imp, err := fsimporter.NewFSImporter(ctx, opts, name, config)
		if err != nil {
			return nil, err
		}
		trackOpen(&openImporters, &peakImporters)
		return &trackedImporter{Importer: imp}, nil
	})
}

func TestPtarClosesPeerStoresAndImporters(t *testing.T) {
	args := []string{"-plaintext", "-o", filepath.Join(t.TempDir(), "test.ptar")}
	for range 2 {
		srcRepo, _ := ptesting.GenerateRepository(t, nil, nil, nil)
		srcSnap := ptesting.GenerateSnapshot(t, srcRepo, []ptesting.MockFile{
			ptesting.NewMockFile("file.txt", 0644, "hello"),
		})
		defer srcSnap.Close()
		args = append(args, "-k", "ptartrackfs://"+srcRepo.Root())
	}
	for range 2 {
		sourceDir := ptesting.GenerateFiles(t, []ptesting.MockFile{
			ptesting.NewMockFile("file.txt", 0644, "hello"),
		})
		args = append(args, "ptartrackfs://"+sourceDir)
	}
	repo, ctx := ptesting.GenerateRepositoryWithoutConfig(t, nil, nil, nil)

	cmd := &Ptar{}
	require.NoError(t, cmd.Parse(ctx, args))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	require.Zero(t, openStores.Load(), "peer stores left open")
	require.Zero(t, openImporters.Load(), "importers left open")
	require.EqualValues(t, 1, peakStores.Load(), "peer stores held open across targets")
	require.EqualValues(t, 1, peakImporters.Load(), "importers held open across targets")
}
