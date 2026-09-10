package mcp

import (
	"bytes"
	"encoding/hex"
	"testing"

	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func TestCompareSnapshots(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap1 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/gone.txt", 0644, "soon gone"),
		ptesting.NewMockFile("subdir/same.txt", 0644, "unchanged"),
	})
	snap1.Close()
	snap2 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy CHANGED"),
		ptesting.NewMockFile("subdir/new.txt", 0644, "brand new"),
		ptesting.NewMockFile("subdir/same.txt", 0644, "unchanged"),
	})
	snap2.Close()

	id1 := hex.EncodeToString(snap1.Header.GetIndexShortID())
	id2 := hex.EncodeToString(snap2.Header.GetIndexShortID())

	out, err := compareSnapshots(repo, compareSnapshotsInput{Snapshot1: id1, Snapshot2: id2})
	require.NoError(t, err)
	require.Equal(t, 1, out.Added)
	require.Equal(t, 1, out.Removed)
	require.Equal(t, 1, out.Modified)
	require.False(t, out.Truncated)

	changes := make(map[string]string)
	for _, change := range out.Changes {
		changes[change.Path] = change.Change
	}
	require.Equal(t, "added", changes["/subdir/new.txt"])
	require.Equal(t, "removed", changes["/subdir/gone.txt"])
	require.Equal(t, "modified", changes["/subdir/dummy.txt"])
	require.NotContains(t, changes, "/subdir/same.txt", "an unchanged file is not a change")
}

func TestCompareSnapshotsIdentical(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap1 := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap1.Close()
	snap2 := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap2.Close()

	id1 := hex.EncodeToString(snap1.Header.GetIndexShortID())
	id2 := hex.EncodeToString(snap2.Header.GetIndexShortID())

	out, err := compareSnapshots(repo, compareSnapshotsInput{Snapshot1: id1, Snapshot2: id2})
	require.NoError(t, err)
	require.Empty(t, out.Changes)
	require.Zero(t, out.Added+out.Removed+out.Modified)
}

func TestCompareSnapshotsRequiresBothSnapshots(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap.Close()
	snapID := hex.EncodeToString(snap.Header.GetIndexShortID())

	_, err := compareSnapshots(repo, compareSnapshotsInput{Snapshot1: snapID})
	require.Error(t, err)
	_, err = compareSnapshots(repo, compareSnapshotsInput{Snapshot2: snapID})
	require.Error(t, err)
	_, err = compareSnapshots(repo, compareSnapshotsInput{Snapshot1: snapID, Snapshot2: "ffffffff"})
	require.Error(t, err)
}

func TestFileHistory(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	for _, content := range []string{"version one", "version one", "version two"} {
		snap := ptesting.GenerateSnapshot(t, repo, mockFiles(content))
		snap.Close()
	}

	out, err := fileHistory(repo, fileHistoryInput{Path: "/subdir/dummy.txt"})
	require.NoError(t, err)
	require.Len(t, out.Versions, 3)
	require.False(t, out.Truncated)

	// newest first: "version two" (changed), "version one" (unchanged),
	// "version one" (first appearance counts as changed)
	require.True(t, out.Versions[0].Changed)
	require.False(t, out.Versions[1].Changed)
	require.True(t, out.Versions[2].Changed)

	require.Equal(t, out.Versions[1].Object, out.Versions[2].Object,
		"identical contents must map to the same object")
	require.NotEqual(t, out.Versions[0].Object, out.Versions[1].Object)
	require.GreaterOrEqual(t, out.Versions[0].SnapshotTime, out.Versions[2].SnapshotTime)
}

func TestFileHistoryRejectsBadInput(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap.Close()

	_, err := fileHistory(repo, fileHistoryInput{})
	require.Error(t, err)

	_, err = fileHistory(repo, fileHistoryInput{Path: "/subdir"})
	require.Error(t, err, "a directory has no content history")

	out, err := fileHistory(repo, fileHistoryInput{Path: "/subdir/missing.txt"})
	require.NoError(t, err)
	require.Empty(t, out.Versions, "a path present in no snapshot has an empty history")
}
