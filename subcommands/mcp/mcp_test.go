package mcp

import (
	"bytes"
	"encoding/hex"
	"os"
	"testing"

	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func TestMcpParseDefaults(t *testing.T) {
	_, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	cmd := &Mcp{}
	require.NoError(t, cmd.Parse(ctx, []string{}))
	require.Equal(t, int64(1<<20), cmd.MaxFileSize)
}

func TestMcpParseMaxFileSize(t *testing.T) {
	_, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	cmd := &Mcp{}
	require.NoError(t, cmd.Parse(ctx, []string{"-max-file-size", "4096"}))
	require.Equal(t, int64(4096), cmd.MaxFileSize)
}

func TestMcpParseRejectsBadInput(t *testing.T) {
	_, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	cmd := &Mcp{}
	require.Error(t, cmd.Parse(ctx, []string{"extra"}))

	cmd = &Mcp{}
	require.Error(t, cmd.Parse(ctx, []string{"-max-file-size", "0"}))
}

func TestListSnapshots(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	})
	snap.Close()

	out, err := listSnapshots(repo)
	require.NoError(t, err)
	require.Len(t, out.Snapshots, 1)
	require.Equal(t, hex.EncodeToString(snap.Header.GetIndexShortID()), out.Snapshots[0].ID)
	require.NotEmpty(t, out.Snapshots[0].Timestamp)
}

func TestListFiles(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
	})
	snap.Close()

	snapID := hex.EncodeToString(snap.Header.GetIndexShortID())

	out, err := listFiles(repo, listFilesInput{Snapshot: snapID, Path: "/subdir"})
	require.NoError(t, err)
	require.False(t, out.Truncated)

	names := make([]string, 0, len(out.Entries))
	for _, entry := range out.Entries {
		names = append(names, entry.Name)
	}
	require.ElementsMatch(t, []string{"dummy.txt", "foo.txt"}, names)
}

func TestListFilesRequiresSnapshot(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	_, err := listFiles(repo, listFilesInput{})
	require.Error(t, err)
}

func TestReadFile(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	})
	snap.Close()

	snapID := hex.EncodeToString(snap.Header.GetIndexShortID())

	out, err := readFile(repo, readFileInput{Snapshot: snapID, Path: "/subdir/dummy.txt"}, 1<<20)
	require.NoError(t, err)
	require.Equal(t, "hello dummy", out.Content)
	require.False(t, out.Binary)
	require.False(t, out.Truncated)
}

func TestReadFileTruncates(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	})
	snap.Close()

	snapID := hex.EncodeToString(snap.Header.GetIndexShortID())

	out, err := readFile(repo, readFileInput{Snapshot: snapID, Path: "/subdir/dummy.txt"}, 5)
	require.NoError(t, err)
	require.Equal(t, "hello", out.Content)
	require.True(t, out.Truncated)

	// A file sitting exactly on the limit is not reported as truncated.
	out, err = readFile(repo, readFileInput{Snapshot: snapID, Path: "/subdir/dummy.txt"}, int64(len("hello dummy")))
	require.NoError(t, err)
	require.Equal(t, "hello dummy", out.Content)
	require.False(t, out.Truncated)
}

func TestReadFileErrors(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	})
	snap.Close()

	snapID := hex.EncodeToString(snap.Header.GetIndexShortID())

	_, err := readFile(repo, readFileInput{Snapshot: snapID}, 1<<20)
	require.Error(t, err)

	_, err = readFile(repo, readFileInput{Snapshot: snapID, Path: "/subdir/missing.txt"}, 1<<20)
	require.Error(t, err)

	// a directory is not a regular file
	_, err = readFile(repo, readFileInput{Snapshot: snapID, Path: "/subdir"}, 1<<20)
	require.Error(t, err)
}

func TestRepositoryInfo(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	})
	snap.Close()

	out, err := repositoryInfo(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 1, out.Snapshots)
	require.NotEmpty(t, out.Hashing)
	require.NotEmpty(t, out.Chunking)
}

func TestSnapshotPath(t *testing.T) {
	require.Equal(t, "abc:/", snapshotPath("abc", ""))
	require.Equal(t, "abc:/etc", snapshotPath("abc", "/etc"))
	require.Equal(t, "abc:/etc", snapshotPath("abc", "etc"))
}
