package mcp

import (
	"bytes"
	"context"
	"encoding/hex"
	"strings"
	"testing"

	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// mockFiles is the tree the inspection tests run against.
func mockFiles(content string) []ptesting.MockFile {
	return []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, content),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
	}
}

func TestSnapshotDetails(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap.Close()

	snapID := hex.EncodeToString(snap.Header.GetIndexShortID())

	out, err := snapshotDetails(repo, snapshotDetailsInput{Snapshot: snapID})
	require.NoError(t, err)
	require.Equal(t, snapID, out.ShortID)
	require.NotEmpty(t, out.ID)
	require.NotEmpty(t, out.Timestamp)
	require.Equal(t, uint64(2), out.Summary.Files)

	_, err = snapshotDetails(repo, snapshotDetailsInput{})
	require.Error(t, err)
}

func TestSearchFiles(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap.Close()

	out, err := searchFiles(context.Background(), repo, searchFilesInput{Name: "dummy.txt"})
	require.NoError(t, err)
	require.NotEmpty(t, out.Matches)
	for _, match := range out.Matches {
		require.True(t, strings.HasSuffix(match.Path, "dummy.txt"))
	}
}

func TestSearchFilesLimitTruncates(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap.Close()

	out, err := searchFiles(context.Background(), repo, searchFilesInput{Limit: 1})
	require.NoError(t, err)
	require.Len(t, out.Matches, 1)
	require.True(t, out.Truncated)
}

func TestSearchFilesUnknownSnapshot(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	_, err := searchFiles(context.Background(), repo, searchFilesInput{Snapshot: "ffffffff"})
	require.Error(t, err)
}

func TestStatEntry(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap.Close()

	snapID := hex.EncodeToString(snap.Header.GetIndexShortID())

	out, err := statEntry(repo, statEntryInput{Snapshot: snapID, Path: "/subdir/dummy.txt"})
	require.NoError(t, err)
	require.Equal(t, "dummy.txt", out.Name)
	require.False(t, out.IsDir)
	require.Equal(t, int64(len("hello dummy")), out.Size)

	out, err = statEntry(repo, statEntryInput{Snapshot: snapID, Path: "/subdir"})
	require.NoError(t, err)
	require.True(t, out.IsDir)

	_, err = statEntry(repo, statEntryInput{Snapshot: snapID, Path: "/subdir/missing.txt"})
	require.Error(t, err)
}

func TestDiffSnapshots(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap1 := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap1.Close()
	snap2 := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy CHANGED"))
	snap2.Close()

	id1 := hex.EncodeToString(snap1.Header.GetIndexShortID())
	id2 := hex.EncodeToString(snap2.Header.GetIndexShortID())

	// a file that changed between the two snapshots
	out, err := diffSnapshots(repo, diffSnapshotsInput{
		Snapshot1: id1,
		Snapshot2: id2,
		Path:      "/subdir/dummy.txt",
	}, 1<<20)
	require.NoError(t, err)
	require.False(t, out.Identical)
	require.Contains(t, out.Diff, "+hello dummy CHANGED")
	require.Contains(t, out.Diff, "-hello dummy")

	// a file that did not change
	out, err = diffSnapshots(repo, diffSnapshotsInput{
		Snapshot1: id1,
		Snapshot2: id2,
		Path:      "/subdir/foo.txt",
	}, 1<<20)
	require.NoError(t, err)
	require.True(t, out.Identical)
	require.Empty(t, out.Diff)
}

func TestDiffSnapshotsRequiresBothSnapshots(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap.Close()
	snapID := hex.EncodeToString(snap.Header.GetIndexShortID())

	// the CLI diffs against the local filesystem when the second argument is
	// missing; the tool must refuse instead of reaching outside the repository.
	_, err := diffSnapshots(repo, diffSnapshotsInput{Snapshot1: snapID, Path: "/subdir/dummy.txt"}, 1<<20)
	require.Error(t, err)

	_, err = diffSnapshots(repo, diffSnapshotsInput{Snapshot1: snapID, Snapshot2: snapID}, 1<<20)
	require.Error(t, err)
}

func TestDigestFile(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap.Close()
	snapID := hex.EncodeToString(snap.Header.GetIndexShortID())

	out, err := digestFile(repo, digestFileInput{Snapshot: snapID, Path: "/subdir/dummy.txt"})
	require.NoError(t, err)
	require.Equal(t, "SHA256", out.Algorithm)
	require.Len(t, out.Digest, 64)

	// the algorithm is honoured, and an unknown one is rejected rather than
	// silently falling back
	out, err = digestFile(repo, digestFileInput{Snapshot: snapID, Path: "/subdir/dummy.txt", Algorithm: "blake3"})
	require.NoError(t, err)
	require.Equal(t, "BLAKE3", out.Algorithm)

	_, err = digestFile(repo, digestFileInput{Snapshot: snapID, Path: "/subdir/dummy.txt", Algorithm: "NOPE"})
	require.Error(t, err)

	_, err = digestFile(repo, digestFileInput{Snapshot: snapID, Path: "/subdir"})
	require.Error(t, err)
}

func TestDigestFileMatchesAcrossSnapshots(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap1 := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap1.Close()
	snap2 := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap2.Close()

	id1 := hex.EncodeToString(snap1.Header.GetIndexShortID())
	id2 := hex.EncodeToString(snap2.Header.GetIndexShortID())

	first, err := digestFile(repo, digestFileInput{Snapshot: id1, Path: "/subdir/dummy.txt"})
	require.NoError(t, err)
	second, err := digestFile(repo, digestFileInput{Snapshot: id2, Path: "/subdir/dummy.txt"})
	require.NoError(t, err)

	require.Equal(t, first.Digest, second.Digest, "identical contents must hash identically")
}

func TestCheckSnapshot(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap.Close()
	snapID := hex.EncodeToString(snap.Header.GetIndexShortID())

	// metadata-only by default, so a healthy snapshot passes
	out, err := checkSnapshot(ctx, repo, checkSnapshotInput{Snapshot: snapID})
	require.NoError(t, err)
	require.True(t, out.OK)
	require.False(t, out.Deep)
	require.Empty(t, out.Error)

	// the deep flag is reported back so a caller knows which was run
	out, err = checkSnapshot(ctx, repo, checkSnapshotInput{Snapshot: snapID, Deep: true})
	require.NoError(t, err)
	require.True(t, out.OK)
	require.True(t, out.Deep)

	_, err = checkSnapshot(ctx, repo, checkSnapshotInput{})
	require.Error(t, err)
}

func TestSnapshotErrors(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap.Close()
	snapID := hex.EncodeToString(snap.Header.GetIndexShortID())

	// a snapshot of a healthy tree records no errors
	out, err := snapshotErrors(repo, snapshotErrorsInput{Snapshot: snapID})
	require.NoError(t, err)
	require.Empty(t, out.Errors)
	require.False(t, out.Truncated)

	_, err = snapshotErrors(repo, snapshotErrorsInput{})
	require.Error(t, err)
}

func TestRepositoryLocks(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	out, err := repositoryLocks(repo)
	require.NoError(t, err)
	require.NotNil(t, out.Locks)
}

func TestRepositoryInfoReportsSizes(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	snap := ptesting.GenerateSnapshot(t, repo, mockFiles("hello dummy"))
	snap.Close()

	out, err := repositoryInfo(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 1, out.Snapshots)
	require.Greater(t, out.LogicalSize, int64(0))
	require.Greater(t, out.StorageSize, int64(0))
}

// TestParseRedirectsStdout guards the stdio contract: stdout carries the
// JSON-RPC stream, so nothing else may write to it.
func TestParseRedirectsStdout(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	defer ctx.Close()

	cmd := &Mcp{}
	require.NoError(t, cmd.Parse(ctx, []string{}))

	ctx.GetLogger().Stdout("progress line")
	require.Empty(t, bufOut.String(), "stdout must stay clean for the JSON-RPC stream")
	require.Contains(t, bufErr.String(), "progress line")
}
