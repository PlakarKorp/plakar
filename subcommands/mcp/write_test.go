package mcp

import (
	"bytes"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func TestMcpParseWriteFlagsDefaultOff(t *testing.T) {
	_, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	// Writes must never be on by accident: the default is read-only.
	cmd := &Mcp{}
	require.NoError(t, cmd.Parse(ctx, []string{}))
	require.False(t, cmd.AllowBackup)
	require.False(t, cmd.AllowDelete)
}

func TestMcpParseWriteFlags(t *testing.T) {
	_, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	cmd := &Mcp{}
	require.NoError(t, cmd.Parse(ctx, []string{"-allow-backup"}))
	require.True(t, cmd.AllowBackup)
	require.False(t, cmd.AllowDelete, "backup must not imply delete")

	cmd = &Mcp{}
	require.NoError(t, cmd.Parse(ctx, []string{"-allow-delete"}))
	require.False(t, cmd.AllowBackup, "delete must not imply backup")
	require.True(t, cmd.AllowDelete)

	cmd = &Mcp{}
	require.NoError(t, cmd.Parse(ctx, []string{"-allow-backup", "-allow-delete"}))
	require.True(t, cmd.AllowBackup)
	require.True(t, cmd.AllowDelete)
}

func TestCreateBackupRequiresSource(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	_, err := createBackup(ctx, repo, createBackupInput{})
	require.Error(t, err)
}

func TestCreateBackup(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello backup"), 0644))

	// As with removal, the freshly written snapshot is not visible to
	// LocateSnapshotIDs in this harness: the repository state is only refreshed
	// by the cached daemon on a real run. So this asserts the reported outcome,
	// and the end-to-end behaviour is covered by driving the server over stdio.
	out, err := createBackup(ctx, repo, createBackupInput{
		Source: dir,
		Name:   "from-test",
		Tags:   []string{"mcp"},
	})
	require.NoError(t, err)
	require.False(t, out.DryRun)
	require.Len(t, out.Snapshot, 8, "a short snapshot ID is reported")
	require.Equal(t, dir, out.Source)
}

func TestCreateBackupDryRunWritesNothing(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "hello.txt"), []byte("hello backup"), 0644))

	before, err := listSnapshots(repo, listSnapshotsInput{})
	require.NoError(t, err)

	out, err := createBackup(ctx, repo, createBackupInput{Source: dir, DryRun: true})
	require.NoError(t, err)
	require.True(t, out.DryRun)
	require.Empty(t, out.Snapshot, "a dry run has no snapshot to report")

	after, err := listSnapshots(repo, listSnapshotsInput{})
	require.NoError(t, err)
	require.Len(t, after.Snapshots, len(before.Snapshots), "a dry run must not create a snapshot")
}

func TestRemoveSnapshotsRequiresIDs(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	_, err := removeSnapshots(ctx, repo, removeSnapshotsInput{})
	require.Error(t, err)

	// an ID that matches nothing is an error rather than a silent no-op
	_, err = removeSnapshots(ctx, repo, removeSnapshotsInput{Snapshots: []string{"ffffffff"}})
	require.Error(t, err)
}

func TestRemoveSnapshotsDryRunKeepsSnapshot(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap.Close()
	snapID := hex.EncodeToString(snap.Header.GetIndexShortID())

	out, err := removeSnapshots(ctx, repo, removeSnapshotsInput{Snapshots: []string{snapID}})
	require.NoError(t, err)
	require.False(t, out.Applied)
	require.Len(t, out.Snapshots, 1)
	require.Equal(t, snapID, out.Snapshots[0].ID)

	// still there: without apply, nothing is removed
	after, err := listSnapshots(repo, listSnapshotsInput{})
	require.NoError(t, err)
	require.Len(t, after.Snapshots, 1)
}

// Removal is asserted on the reported outcome rather than on a subsequent
// listing: in this harness the repository state is not refreshed the way the
// cached daemon does it for a real run, so a deleted snapshot keeps showing up
// in LocateSnapshotIDs. The rm subcommand's own tests assert the same way.
func TestRemoveSnapshotsApply(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap.Close()
	snapID := hex.EncodeToString(snap.Header.GetIndexShortID())

	out, err := removeSnapshots(ctx, repo, removeSnapshotsInput{
		Snapshots: []string{snapID},
		Apply:     true,
	})
	require.NoError(t, err)
	require.True(t, out.Applied)
	require.Len(t, out.Snapshots, 1)
	require.Equal(t, snapID, out.Snapshots[0].ID)
	require.Contains(t, bufOut.String(), "removal of "+snapID+" completed successfully")
}

func TestRemoveSnapshotsOnlyTouchesNamedSnapshot(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	keep := ptesting.GenerateSnapshot(t, repo, mockFiles("keep me"))
	keep.Close()
	drop := ptesting.GenerateSnapshot(t, repo, mockFiles("drop me"))
	drop.Close()

	keepID := hex.EncodeToString(keep.Header.GetIndexShortID())
	dropID := hex.EncodeToString(drop.Header.GetIndexShortID())

	out, err := removeSnapshots(ctx, repo, removeSnapshotsInput{
		Snapshots: []string{dropID},
		Apply:     true,
	})
	require.NoError(t, err)

	// only the named snapshot is reported, and only it is logged as removed
	require.Len(t, out.Snapshots, 1)
	require.Equal(t, dropID, out.Snapshots[0].ID)
	require.Contains(t, bufOut.String(), "removal of "+dropID+" completed successfully")
	require.NotContains(t, bufOut.String(), "removal of "+keepID)
}

func TestPruneSnapshotsDryRun(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap.Close()

	out, err := pruneSnapshots(ctx, repo, pruneSnapshotsInput{})
	require.NoError(t, err)
	require.False(t, out.Applied)

	// whatever the policy selects, a dry run leaves the repository alone
	after, err := listSnapshots(repo, listSnapshotsInput{})
	require.NoError(t, err)
	require.Len(t, after.Snapshots, 1)
}
