package diff

import (
	"bytes"
	"encoding/hex"
	"testing"

	_ "github.com/PlakarKorp/integrations/fs/exporter"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// Flat (non-recursive) directory diff that exercises the "Common
// subdirectories", "Only in", and "File type mismatch" branches of
// diff_directories_flat in one pass.
func TestDiffCov80FlatDirAllBranches(t *testing.T) {
	t.Parallel()
	bufOut := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bytes.NewBuffer(nil), nil)

	snap1 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("top"),
		ptesting.NewMockDir("top/common"),
		ptesting.NewMockFile("top/common/c.txt", 0644, "c"),
		ptesting.NewMockFile("top/onlyleft.txt", 0644, "l"),
		// "x" is a file on the left
		ptesting.NewMockFile("top/x", 0644, "file"),
	})
	defer snap1.Close()
	snap2 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("top"),
		ptesting.NewMockDir("top/common"),
		ptesting.NewMockFile("top/common/c.txt", 0644, "c"),
		ptesting.NewMockFile("top/onlyright.txt", 0644, "r"),
		// "x" is a directory on the right -> type mismatch
		ptesting.NewMockDir("top/x"),
		ptesting.NewMockFile("top/x/inner.txt", 0644, "deep"),
	})
	defer snap2.Close()

	id1 := hex.EncodeToString(snap1.Header.GetIndexShortID())
	id2 := hex.EncodeToString(snap2.Header.GetIndexShortID())
	cmd := &Diff{}
	require.NoError(t, cmd.Parse(ctx, []string{id1 + ":/top", id2 + ":/top"}))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	out := bufOut.String()
	require.Contains(t, out, "Common subdirectories: common")
	require.Contains(t, out, "onlyleft.txt")
	require.Contains(t, out, "onlyright.txt")
	require.Contains(t, out, "File type mismatch: x")
}

// Two differing binary files produce the "Binary files ... differ" line via
// diff_readers' isbinary branch.
func TestDiffCov80BinaryFilesDiffer(t *testing.T) {
	t.Parallel()
	bufOut := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bytes.NewBuffer(nil), nil)

	snap1 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile("blob", 0644, string([]byte{0x00, 0x01, 0x02, 'a'})),
	})
	defer snap1.Close()
	snap2 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile("blob", 0644, string([]byte{0x00, 0x01, 0x02, 'b'})),
	})
	defer snap2.Close()

	id1 := hex.EncodeToString(snap1.Header.GetIndexShortID())
	id2 := hex.EncodeToString(snap2.Header.GetIndexShortID())
	cmd := &Diff{}
	require.NoError(t, cmd.Parse(ctx, []string{id1 + ":/blob", id2 + ":/blob"}))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Contains(t, bufOut.String(), "Binary files")
	require.Contains(t, bufOut.String(), "differ")
}

// Differing text files with -highlight: drives the Execute highlight branch
// (quick.Highlight on a non-empty unified diff).
func TestDiffCov80HighlightTextDiff(t *testing.T) {
	t.Parallel()
	bufOut := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bytes.NewBuffer(nil), nil)

	snap1 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile("f.txt", 0644, "line one\nline two\n"),
	})
	defer snap1.Close()
	snap2 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile("f.txt", 0644, "line one\nline CHANGED\n"),
	})
	defer snap2.Close()

	id1 := hex.EncodeToString(snap1.Header.GetIndexShortID())
	id2 := hex.EncodeToString(snap2.Header.GetIndexShortID())
	cmd := &Diff{}
	require.NoError(t, cmd.Parse(ctx, []string{"-highlight", id1 + ":/f.txt", id2 + ":/f.txt"}))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	// Highlighted output is non-empty and contains the changed content marker.
	require.NotEmpty(t, bufOut.String())
	require.Contains(t, bufOut.String(), "CHANGED")
}

// Diffing a directory against a regular file is a "different file types"
// error from diff_pathnames.
func TestDiffCov80DirVsFileMismatch(t *testing.T) {
	t.Parallel()
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("dir"),
		ptesting.NewMockFile("dir/f.txt", 0644, "x"),
		ptesting.NewMockFile("plain.txt", 0644, "y"),
	})
	defer snap.Close()

	id := hex.EncodeToString(snap.Header.GetIndexShortID())
	cmd := &Diff{}
	require.NoError(t, cmd.Parse(ctx, []string{id + ":/dir", id + ":/plain.txt"}))
	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "different file types")
}

// Recursive diff where a nested regular file differs in content: drives the
// e1.IsRegular && e2.IsRegular branch of diff_directories_recursive, which
// opens both readers and runs diff_readers, emitting a unified diff body.
func TestDiffCov80RecursiveRegularFileContentDiff(t *testing.T) {
	t.Parallel()
	bufOut := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bytes.NewBuffer(nil), nil)

	snap1 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("r"),
		ptesting.NewMockDir("r/nested"),
		ptesting.NewMockFile("r/nested/data.txt", 0644, "alpha\nbeta\ngamma\n"),
	})
	defer snap1.Close()
	snap2 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("r"),
		ptesting.NewMockDir("r/nested"),
		ptesting.NewMockFile("r/nested/data.txt", 0644, "alpha\nBETA\ngamma\n"),
	})
	defer snap2.Close()

	id1 := hex.EncodeToString(snap1.Header.GetIndexShortID())
	id2 := hex.EncodeToString(snap2.Header.GetIndexShortID())
	cmd := &Diff{}
	require.NoError(t, cmd.Parse(ctx, []string{"-recursive", id1 + ":/r", id2 + ":/r"}))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	out := bufOut.String()
	require.Contains(t, out, "Common subdirectories")
	require.Contains(t, out, "BETA")
}

// Second snapshot fails to open: exercises the Path2 open-error branch.
func TestDiffCov80SecondSnapshotOpenError(t *testing.T) {
	t.Parallel()
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile("a.txt", 0644, "a"),
	})
	defer snap.Close()

	id := hex.EncodeToString(snap.Header.GetIndexShortID())
	cmd := &Diff{}
	require.NoError(t, cmd.Parse(ctx, []string{id + ":/a.txt", "deadbeefdeadbeef:/a.txt"}))
	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "could not open snapshot")
}
