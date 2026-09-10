package mcp

import (
	"bytes"
	"context"
	"encoding/hex"
	"testing"

	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func TestListFilesOffset(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap.Close()
	snapID := hex.EncodeToString(snap.Header.GetIndexShortID())

	all, err := listFiles(repo, listFilesInput{Snapshot: snapID, Path: "/subdir"})
	require.NoError(t, err)
	require.Len(t, all.Entries, 2)

	rest, err := listFiles(repo, listFilesInput{Snapshot: snapID, Path: "/subdir", Offset: 1})
	require.NoError(t, err)
	require.Len(t, rest.Entries, 1)
	require.Equal(t, all.Entries[1], rest.Entries[0])

	past, err := listFiles(repo, listFilesInput{Snapshot: snapID, Path: "/subdir", Offset: 10})
	require.NoError(t, err)
	require.Empty(t, past.Entries)
	require.False(t, past.Truncated)
}

func TestReadFileOffset(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap.Close()
	snapID := hex.EncodeToString(snap.Header.GetIndexShortID())

	out, err := readFile(repo, readFileInput{Snapshot: snapID, Path: "/subdir/dummy.txt", Offset: 6}, 1<<20)
	require.NoError(t, err)
	require.Equal(t, "dummy", out.Content)
	require.Equal(t, int64(6), out.Offset)
	require.Equal(t, int64(len("hello dummy")), out.Size)
	require.False(t, out.Truncated)

	_, err = readFile(repo, readFileInput{Snapshot: snapID, Path: "/subdir/dummy.txt", Offset: -1}, 1<<20)
	require.Error(t, err)
}

func TestReadFilePagesThroughLargeFile(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap.Close()
	snapID := hex.EncodeToString(snap.Header.GetIndexShortID())

	// with max-file-size 6, "hello dummy" takes two reads
	first, err := readFile(repo, readFileInput{Snapshot: snapID, Path: "/subdir/dummy.txt"}, 6)
	require.NoError(t, err)
	require.Equal(t, "hello ", first.Content)
	require.True(t, first.Truncated)

	second, err := readFile(repo, readFileInput{Snapshot: snapID, Path: "/subdir/dummy.txt", Offset: 6}, 6)
	require.NoError(t, err)
	require.Equal(t, "dummy", second.Content)
	require.False(t, second.Truncated)

	require.Equal(t, "hello dummy", first.Content+second.Content)
}

func TestSearchFilesOffset(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap.Close()

	first, err := searchFiles(context.Background(), repo, searchFilesInput{Limit: 1})
	require.NoError(t, err)
	require.Len(t, first.Matches, 1)
	require.True(t, first.Truncated)
	require.Equal(t, 1, first.NextOffset)

	second, err := searchFiles(context.Background(), repo, searchFilesInput{Limit: 1, Offset: first.NextOffset})
	require.NoError(t, err)
	require.Len(t, second.Matches, 1)
	require.NotEqual(t, first.Matches[0].Path, second.Matches[0].Path)
}
