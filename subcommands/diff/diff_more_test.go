package diff

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"

	_ "github.com/PlakarKorp/integration-fs/exporter"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func shortID(snap *snapshot.Snapshot) string {
	id := snap.Header.GetIndexShortID()
	return hex.EncodeToString(id[:])
}

func twoSnapshots(t *testing.T, bufOut, bufErr *bytes.Buffer, files1, files2 []ptesting.MockFile) (*repository.Repository, *appcontext.AppContext, *snapshot.Snapshot, *snapshot.Snapshot) {
	t.Helper()
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap1 := ptesting.GenerateSnapshot(t, repo, files1)
	snap2 := ptesting.GenerateSnapshot(t, repo, files2)
	t.Cleanup(func() { snap1.Close(); snap2.Close() })
	return repo, ctx, snap1, snap2
}

func TestDiffParse(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)

	// One positional arg: Path1 set, Path2 empty.
	cmd := &Diff{}
	err := cmd.Parse(ctx, []string{"abc:/x"})
	require.NoError(t, err)
	require.Equal(t, "abc:/x", cmd.Path1)
	require.Equal(t, "", cmd.Path2)
	require.Equal(t, "diff", cmd.Name())

	// Two positional args.
	cmd = &Diff{}
	err = cmd.Parse(ctx, []string{"a:/x", "b:/y"})
	require.NoError(t, err)
	require.Equal(t, "a:/x", cmd.Path1)
	require.Equal(t, "b:/y", cmd.Path2)

	// Flags are honoured.
	cmd = &Diff{}
	err = cmd.Parse(ctx, []string{"-highlight", "-recursive", "a:/x", "b:/y"})
	require.NoError(t, err)
	require.True(t, cmd.Highlight)
	require.True(t, cmd.Recursive)

	// No positional args -> error.
	cmd = &Diff{}
	err = cmd.Parse(ctx, []string{})
	require.Error(t, err)
}

func TestDiffExecuteSameFile(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	files := []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	}
	repo, ctx, snap1, snap2 := twoSnapshots(t, bufOut, bufErr, files, files)

	cmd := &Diff{
		Path1: fmt.Sprintf("%s:/subdir/dummy.txt", shortID(snap1)),
		Path2: fmt.Sprintf("%s:/subdir/dummy.txt", shortID(snap2)),
	}
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	// Identical files produce no unified-diff body.
	require.NotContains(t, bufOut.String(), "@@")
}

func TestDiffExecuteChangedFile(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, ctx, snap1, snap2 := twoSnapshots(t, bufOut, bufErr,
		[]ptesting.MockFile{
			ptesting.NewMockDir("subdir"),
			ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy\n"),
		},
		[]ptesting.MockFile{
			ptesting.NewMockDir("subdir"),
			ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy changed\n"),
		},
	)

	cmd := &Diff{
		Path1: fmt.Sprintf("%s:/subdir/dummy.txt", shortID(snap1)),
		Path2: fmt.Sprintf("%s:/subdir/dummy.txt", shortID(snap2)),
	}
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	output := bufOut.String()
	require.Contains(t, output, "@@")
	require.Contains(t, output, "-hello dummy")
	require.Contains(t, output, "+hello dummy changed")
}

func TestDiffExecuteHighlight(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, ctx, snap1, snap2 := twoSnapshots(t, bufOut, bufErr,
		[]ptesting.MockFile{
			ptesting.NewMockDir("subdir"),
			ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy\n"),
		},
		[]ptesting.MockFile{
			ptesting.NewMockDir("subdir"),
			ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy changed\n"),
		},
	)

	cmd := &Diff{
		Highlight: true,
		Path1:     fmt.Sprintf("%s:/subdir/dummy.txt", shortID(snap1)),
		Path2:     fmt.Sprintf("%s:/subdir/dummy.txt", shortID(snap2)),
	}
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	// Highlighted output still carries the changed content.
	require.Contains(t, bufOut.String(), "hello dummy changed")
}

func TestDiffExecuteDirFlat(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, ctx, snap1, snap2 := twoSnapshots(t, bufOut, bufErr,
		[]ptesting.MockFile{
			ptesting.NewMockDir("subdir"),
			ptesting.NewMockDir("subdir/common"),
			ptesting.NewMockFile("subdir/only1.txt", 0644, "one"),
			ptesting.NewMockFile("subdir/shared.txt", 0644, "shared"),
		},
		[]ptesting.MockFile{
			ptesting.NewMockDir("subdir"),
			ptesting.NewMockDir("subdir/common"),
			ptesting.NewMockFile("subdir/only2.txt", 0644, "two"),
			ptesting.NewMockFile("subdir/shared.txt", 0644, "shared"),
		},
	)

	cmd := &Diff{
		Path1: fmt.Sprintf("%s:/subdir", shortID(snap1)),
		Path2: fmt.Sprintf("%s:/subdir", shortID(snap2)),
	}
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	output := bufOut.String()
	require.Contains(t, output, "only1.txt")
	require.Contains(t, output, "only2.txt")
	require.Contains(t, output, "Common subdirectories")
}

func TestDiffExecuteDirRecursive(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, ctx, snap1, snap2 := twoSnapshots(t, bufOut, bufErr,
		[]ptesting.MockFile{
			ptesting.NewMockDir("subdir"),
			ptesting.NewMockDir("subdir/nested"),
			ptesting.NewMockFile("subdir/nested/file.txt", 0644, "content a\n"),
			ptesting.NewMockFile("subdir/top.txt", 0644, "top\n"),
		},
		[]ptesting.MockFile{
			ptesting.NewMockDir("subdir"),
			ptesting.NewMockDir("subdir/nested"),
			ptesting.NewMockFile("subdir/nested/file.txt", 0644, "content b\n"),
			ptesting.NewMockFile("subdir/top.txt", 0644, "top\n"),
		},
	)

	cmd := &Diff{
		Recursive: true,
		Path1:     fmt.Sprintf("%s:/subdir", shortID(snap1)),
		Path2:     fmt.Sprintf("%s:/subdir", shortID(snap2)),
	}
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	output := bufOut.String()
	require.Contains(t, output, "Common subdirectories")
	require.Contains(t, output, "nested")
	// nested/file.txt differs -> unified diff emitted.
	require.Contains(t, output, "-content a")
	require.Contains(t, output, "+content b")
}

func TestDiffExecuteBinary(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	// Embed a NUL byte to make the content classify as binary.
	repo, ctx, snap1, snap2 := twoSnapshots(t, bufOut, bufErr,
		[]ptesting.MockFile{
			ptesting.NewMockDir("subdir"),
			ptesting.NewMockFile("subdir/bin.dat", 0644, "abc\x00def"),
		},
		[]ptesting.MockFile{
			ptesting.NewMockDir("subdir"),
			ptesting.NewMockFile("subdir/bin.dat", 0644, "abc\x00xyz"),
		},
	)

	cmd := &Diff{
		Path1: fmt.Sprintf("%s:/subdir/bin.dat", shortID(snap1)),
		Path2: fmt.Sprintf("%s:/subdir/bin.dat", shortID(snap2)),
	}
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Contains(t, bufOut.String(), "Binary files")
}

func TestDiffExecuteTypeMismatch(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	// snap1 has /subdir/item as a file, snap2 has /subdir/item as a dir.
	repo, ctx, snap1, snap2 := twoSnapshots(t, bufOut, bufErr,
		[]ptesting.MockFile{
			ptesting.NewMockDir("subdir"),
			ptesting.NewMockFile("subdir/item", 0644, "i am a file"),
		},
		[]ptesting.MockFile{
			ptesting.NewMockDir("subdir"),
			ptesting.NewMockDir("subdir/item"),
			ptesting.NewMockFile("subdir/item/inner.txt", 0644, "inner"),
		},
	)

	cmd := &Diff{
		Path1: fmt.Sprintf("%s:/subdir/item", shortID(snap1)),
		Path2: fmt.Sprintf("%s:/subdir/item", shortID(snap2)),
	}
	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "different file types")
}

func TestDiffExecuteBadSnapshot(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello"),
	})
	defer snap.Close()

	// First path bad.
	cmd := &Diff{Path1: "deadbeef:/subdir/dummy.txt"}
	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)

	// Second path bad.
	cmd = &Diff{
		Path1: fmt.Sprintf("%s:/subdir/dummy.txt", shortID(snap)),
		Path2: "deadbeef:/subdir/dummy.txt",
	}
	status, err = cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
}
