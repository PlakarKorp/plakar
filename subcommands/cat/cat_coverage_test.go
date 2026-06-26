package cat

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"testing"

	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func gzipString(t *testing.T, s string) string {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	_, err := w.Write([]byte(s))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.String()
}

// An entirely unknown snapshot prefix fails OpenSnapshotByPath and is reported
// via the logger, returning exit status 1.
func TestExecuteCmdCatUnknownSnapshot(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	})
	snap.Close()

	args := []string{"deadbeef:subdir/dummy.txt"}

	cmd := &Cat{}
	require.NoError(t, cmd.Parse(ctx, args))

	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, bufErr.String(), "cat:")
}

// Multiple paths in one invocation are concatenated to stdout in order.
func TestExecuteCmdCatMultiplePaths(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
	})
	snap.Close()

	args := []string{":subdir/dummy.txt", ":subdir/foo.txt"}

	cmd := &Cat{}
	require.NoError(t, cmd.Parse(ctx, args))

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Equal(t, "hello dummyhello foo", bufOut.String())
}

// -decompress on a file that is not gzip is a no-op (the content type guard is
// not satisfied) and the raw bytes are emitted unchanged.
func TestExecuteCmdCatDecompressNonGzip(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	})
	snap.Close()

	args := []string{"-decompress", ":subdir/dummy.txt"}

	cmd := &Cat{}
	require.NoError(t, cmd.Parse(ctx, args))
	require.True(t, cmd.Decompress)

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Equal(t, "hello dummy", bufOut.String())
}

// -decompress on a genuine gzip file walks the gzip.NewReader branch and emits
// the decompressed content.
func TestExecuteCmdCatDecompressGzip(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/data.gz", 0644, gzipString(t, "decompressed payload")),
	})
	snap.Close()

	args := []string{"-decompress", ":subdir/data.gz"}

	cmd := &Cat{}
	require.NoError(t, cmd.Parse(ctx, args))

	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Equal(t, "decompressed payload", bufOut.String())
}

// A path that resolves to a directory (not a regular file) is reported and
// counts as an error, while other valid paths still succeed.
func TestExecuteCmdCatMixedErrors(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	})
	snap.Close()

	args := []string{
		":subdir/dummy.txt",
		fmt.Sprintf("%s:subdir/missing.txt", ""),
	}

	cmd := &Cat{}
	require.NoError(t, cmd.Parse(ctx, args))

	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
	// the valid file was still emitted
	require.Contains(t, bufOut.String(), "hello dummy")
	require.Contains(t, bufErr.String(), "no such file")
}

// Parse with no arguments is rejected.
func TestExecuteCmdCatParseNoArgs(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	_, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	cmd := &Cat{}
	err := cmd.Parse(ctx, []string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "at least one parameter is required")
}
