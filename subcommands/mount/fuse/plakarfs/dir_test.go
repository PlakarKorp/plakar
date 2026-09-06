//go:build linux || darwin

package plakarfs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	_ "github.com/PlakarKorp/integrations/fs/importer"
	"github.com/PlakarKorp/kloset/connectors/importer"
	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func TestSnapshotDirectoryStartsAtSourceRoot(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, nil, nil, nil)
	src := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(src, "myfile.txt"), []byte("data"), 0o644))

	snap := backupDirectory(t, repo, ctx, src)
	defer snap.Close()

	pfs := NewFS(ctx, repo, locate.NewDefaultLocateOptions(), nil)
	root, err := NewDirectory(pfs, nil, nil, "")
	require.NoError(t, err)

	shortID := fmt.Sprintf("%x", snap.Header.GetIndexShortID())
	root.readDirSnapshotMapping = map[string]objects.MAC{shortID: snap.Header.Identifier}

	node, err := root.Lookup(context.Background(), shortID)
	require.NoError(t, err)
	dir := node.(*Dir)

	entries, err := dir.ReadDirAll(context.Background())
	require.NoError(t, err)

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name)
	}
	slices.Sort(names)
	require.Equal(t, []string{"myfile.txt"}, names)
}

func backupDirectory(t *testing.T, repo *repository.Repository, ctx *appcontext.AppContext, src string) *snapshot.Snapshot {
	t.Helper()

	imp, err := importer.NewImporter(ctx.GetInner(), ctx.ImporterOpts(), map[string]string{"location": src})
	require.NoError(t, err)
	defer imp.Close(ctx.GetInner())

	source, err := snapshot.NewSource(ctx.GetInner(), imp)
	require.NoError(t, err)

	builder, err := snapshot.Create(repo, repository.DefaultType, "", objects.NilMac, &snapshot.BuilderOptions{Name: "test"})
	require.NoError(t, err)

	require.NoError(t, builder.Backup(source))
	require.NoError(t, builder.Commit())
	id := builder.Header.Identifier
	require.NoError(t, builder.Close())
	require.NoError(t, repo.RebuildState())

	snap, err := snapshot.Load(repo, id)
	require.NoError(t, err)
	return snap
}
