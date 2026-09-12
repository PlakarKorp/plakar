package mount

import (
	"bytes"
	"io/fs"
	"testing"
	"testing/fstest"

	_ "github.com/PlakarKorp/integrations/fs/exporter"
	"github.com/PlakarKorp/plakar/subcommands"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

const (
	snapshotSubFSTestFile    = "a.txt"
	snapshotSubFSTestContent = "x"
)

func TestMountRegisteredFactory(t *testing.T) {
	cmd, _, _ := subcommands.Lookup([]string{"mount"})
	require.NotNil(t, cmd)
	require.IsType(t, &Mount{}, cmd)
}

func TestMountParseRepositoryLevel(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	_ = repo

	cmd := &Mount{}
	require.NoError(t, cmd.Parse(ctx, []string{"-to", "/mnt/x"}))
	require.Equal(t, "/mnt/x", cmd.Mountpoint)
	require.Equal(t, "", cmd.SnapshotPath)
	require.NotNil(t, cmd.LocateOptions)
}

func TestMountParseSnapshotLevel(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	_ = repo

	cmd := &Mount{}
	require.NoError(t, cmd.Parse(ctx, []string{"-to", "/mnt/x", "abc123"}))
	require.Equal(t, "abc123", cmd.SnapshotPath)
	require.True(t, cmd.LocateOptions.Empty(), "snapshot-level parse resets LocateOptions")
}

func TestMountParseAllowOthers(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	_ = repo

	cmd := &Mount{}
	require.NoError(t, cmd.Parse(ctx, []string{"-allow-others", "-to", "/mnt/x"}))
	require.True(t, cmd.AllowOthers)
}

func TestSnapshotSubFSRootPathUsesSnapshotRoot(t *testing.T) {
	snapshotFS := fstest.MapFS{
		snapshotSubFSTestFile: &fstest.MapFile{Data: []byte(snapshotSubFSTestContent)},
	}

	subFS, err := snapshotSubFS(snapshotFS, snapshotRootPath)
	require.NoError(t, err)

	data, err := fs.ReadFile(subFS, snapshotSubFSTestFile)
	require.NoError(t, err)
	require.Equal(t, []byte(snapshotSubFSTestContent), data)
}

func TestMountExecuteBadSnapshot(t *testing.T) {
	// A snapshot path that resolves to nothing fails before any mount is
	// attempted (so this is safe on all platforms, FUSE never engages).
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile("a.txt", 0644, "x"),
	})
	defer snap.Close()

	cmd := &Mount{}
	require.NoError(t, cmd.Parse(ctx, []string{"-to", "/mnt/x", "deadbeefdeadbeef:/"}))
	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
}
