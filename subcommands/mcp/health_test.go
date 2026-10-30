package mcp

import (
	"bytes"
	"testing"

	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func TestRepositoryHealth(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap1 := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap1.Close()
	snap2 := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy CHANGED"))
	snap2.Close()

	out, err := repositoryHealth(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 2, out.Snapshots)
	require.NotNil(t, out.Newest)
	require.NotNil(t, out.Oldest)
	require.GreaterOrEqual(t, out.Newest.Timestamp, out.Oldest.Timestamp)
	require.NotEmpty(t, out.Newest.Age)
	require.Greater(t, out.LogicalSize, int64(0))
	require.Greater(t, out.StorageSize, int64(0))

	// both snapshots come from the same mock source
	require.Len(t, out.Sources, 1)
	require.Equal(t, 2, out.Sources[0].Snapshots)
	require.Equal(t, out.Newest.ID, out.Sources[0].Latest.ID)
}

func TestRepositoryHealthEmptyRepository(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	out, err := repositoryHealth(ctx, repo)
	require.NoError(t, err)
	require.Zero(t, out.Snapshots)
	require.Nil(t, out.Newest)
	require.Nil(t, out.Oldest)
	require.Empty(t, out.Sources)
}

func TestListTags(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap.Close()

	// the harness creates untagged snapshots; the point is an empty result
	// rather than an error
	out, err := listTags(repo)
	require.NoError(t, err)
	require.Empty(t, out.Tags)
}

func TestListSnapshotsTagFilter(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap.Close()

	out, err := listSnapshots(repo, listSnapshotsInput{})
	require.NoError(t, err)
	require.Len(t, out.Snapshots, 1)

	// filtering on a tag no snapshot carries returns nothing rather than
	// everything
	out, err = listSnapshots(repo, listSnapshotsInput{Tag: "nope"})
	require.NoError(t, err)
	require.Empty(t, out.Snapshots)
}

func TestListSnapshotsPaging(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap1 := ptesting.GenerateSnapshot(t, repo, mockFiles("one"))
	snap1.Close()
	snap2 := ptesting.GenerateSnapshot(t, repo, mockFiles("two"))
	snap2.Close()

	first, err := listSnapshots(repo, listSnapshotsInput{Limit: 1})
	require.NoError(t, err)
	require.Len(t, first.Snapshots, 1)
	require.True(t, first.Truncated)
	require.Equal(t, 1, first.NextOffset)

	second, err := listSnapshots(repo, listSnapshotsInput{Offset: first.NextOffset, Limit: 1})
	require.NoError(t, err)
	require.Len(t, second.Snapshots, 1)
	require.False(t, second.Truncated)
	require.NotEqual(t, first.Snapshots[0].ID, second.Snapshots[0].ID)

	// paging past the end is empty, not an error
	past, err := listSnapshots(repo, listSnapshotsInput{Offset: 10})
	require.NoError(t, err)
	require.Empty(t, past.Snapshots)
}
