package mcp

import (
	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/PlakarKorp/integrations/fs/exporter"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func TestResolveRestoreDestination(t *testing.T) {
	root := t.TempDir()

	target, err := resolveRestoreDestination(root, "")
	require.NoError(t, err)
	require.Equal(t, root, target)

	target, err = resolveRestoreDestination(root, "sub/dir")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, "sub", "dir"), target)

	// a path that dips below the root and comes back up stays confined
	target, err = resolveRestoreDestination(root, "a/../b")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, "b"), target)

	_, err = resolveRestoreDestination(root, "/etc")
	require.Error(t, err, "absolute destinations are refused")

	_, err = resolveRestoreDestination(root, "..")
	require.Error(t, err)

	_, err = resolveRestoreDestination(root, "../escape")
	require.Error(t, err)

	_, err = resolveRestoreDestination(root, "a/../../escape")
	require.Error(t, err)
}

func TestRestoreFiles(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap.Close()
	snapID := hex.EncodeToString(snap.Header.GetIndexShortID())

	root := t.TempDir()

	out, err := restoreFiles(ctx, repo, restoreFilesInput{
		Snapshot: snapID,
		Path:     "/subdir/dummy.txt",
	}, root)
	require.NoError(t, err)
	require.Equal(t, root, out.Destination)

	content, err := os.ReadFile(filepath.Join(root, "dummy.txt"))
	require.NoError(t, err)
	require.Equal(t, "hello dummy", string(content))
}

func TestRestoreFilesDirectoryIntoSubdirectory(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap.Close()
	snapID := hex.EncodeToString(snap.Header.GetIndexShortID())

	root := t.TempDir()

	out, err := restoreFiles(ctx, repo, restoreFilesInput{
		Snapshot:    snapID,
		Path:        "/subdir",
		Destination: "recovered",
	}, root)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, "recovered"), out.Destination)

	// a restored directory spills its contents into the destination
	content, err := os.ReadFile(filepath.Join(root, "recovered", "dummy.txt"))
	require.NoError(t, err)
	require.Equal(t, "hello dummy", string(content))

	content, err = os.ReadFile(filepath.Join(root, "recovered", "foo.txt"))
	require.NoError(t, err)
	require.Equal(t, "hello foo", string(content))
}

func TestRestoreFilesRejectsBadInput(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap.Close()
	snapID := hex.EncodeToString(snap.Header.GetIndexShortID())

	root := t.TempDir()

	_, err := restoreFiles(ctx, repo, restoreFilesInput{}, root)
	require.Error(t, err)

	_, err = restoreFiles(ctx, repo, restoreFilesInput{
		Snapshot:    snapID,
		Path:        "/subdir/dummy.txt",
		Destination: "../escape",
	}, root)
	require.Error(t, err)

	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	require.Empty(t, entries, "a refused restore must write nothing")
}

func TestSyncSnapshotsRejectsBadPeer(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	// only peers from the configuration are acceptable
	_, err := syncSnapshots(ctx, repo, syncSnapshotsInput{Peer: "/tmp/anywhere"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "configuration")

	_, err = syncSnapshots(ctx, repo, syncSnapshotsInput{Peer: "@no-such-peer"})
	require.Error(t, err)
}
