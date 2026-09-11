package diff

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	_ "github.com/PlakarKorp/integrations/fs/exporter"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// ---------- isbinary / binaryeq unit tests ----------

func TestCov2IsBinaryNonReaderAt(t *testing.T) {
	// strings.Reader implements ReaderAt; use an io.Reader that does not.
	require.False(t, isbinary(struct{ plainReader }{}))
}

type plainReader struct{}

func (plainReader) Read(p []byte) (int, error) { return 0, nil }

func TestCov2IsBinaryTextAndBinary(t *testing.T) {
	require.False(t, isbinary(strings.NewReader("plain text\nwith tabs\t")))
	require.True(t, isbinary(strings.NewReader(string([]byte{0x00, 0x01}))))
	require.True(t, isbinary(strings.NewReader(string([]byte{0x7f}))))
}

func TestCov2BinaryEq(t *testing.T) {
	same, err := binaryeq(strings.NewReader("hello"), strings.NewReader("hello"))
	require.NoError(t, err)
	require.True(t, same)

	same, err = binaryeq(strings.NewReader("hello"), strings.NewReader("world"))
	require.NoError(t, err)
	require.False(t, same)

	// different lengths
	same, err = binaryeq(strings.NewReader("short"), strings.NewReader("longerinput"))
	require.NoError(t, err)
	require.False(t, same)
}

// ---------- Execute: diff a single snapshot path against local fs ----------

func TestCov2DiffAgainstLocalFS(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bytes.NewBuffer(nil), nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile("etc/hostname-xyz.txt", 0644, "snapshot-content\n"),
	})
	defer snap.Close()

	id := hex.EncodeToString(snap.Header.GetIndexShortID())
	cmd := &Diff{}
	// only one path arg -> Path2 == "" -> compares against os.DirFS("/")
	require.NoError(t, cmd.Parse(ctx, []string{id + ":/etc/hostname-xyz.txt"}))
	status, err := cmd.Execute(ctx, repo)
	// the local file almost certainly doesn't exist -> open error from vfs2
	require.Error(t, err)
	require.Equal(t, 1, status)
}

// ---------- Execute: identical text files produce no diff body ----------

func TestCov2DiffIdenticalTextNoOutput(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bytes.NewBuffer(nil), nil)
	snap1 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile("a.txt", 0644, "same\n"),
	})
	defer snap1.Close()
	snap2 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile("a.txt", 0644, "same\n"),
	})
	defer snap2.Close()

	id1 := hex.EncodeToString(snap1.Header.GetIndexShortID())
	id2 := hex.EncodeToString(snap2.Header.GetIndexShortID())
	cmd := &Diff{}
	require.NoError(t, cmd.Parse(ctx, []string{id1 + ":/a.txt", id2 + ":/a.txt"}))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.NotContains(t, bufOut.String(), "@@")
}

// ---------- Execute: identical binary files produce no "differ" line ----------

func TestCov2DiffIdenticalBinaryNoOutput(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bytes.NewBuffer(nil), nil)
	bin := string([]byte{0x00, 0x01, 0x02, 'z'})
	snap1 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile("blob", 0644, bin),
	})
	defer snap1.Close()
	snap2 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile("blob", 0644, bin),
	})
	defer snap2.Close()

	id1 := hex.EncodeToString(snap1.Header.GetIndexShortID())
	id2 := hex.EncodeToString(snap2.Header.GetIndexShortID())
	cmd := &Diff{}
	require.NoError(t, cmd.Parse(ctx, []string{id1 + ":/blob", id2 + ":/blob"}))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.NotContains(t, bufOut.String(), "differ")
}

// ---------- Execute: open-error when path missing in snapshot ----------

func TestCov2DiffMissingPathInSnapshot(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockFile("a.txt", 0644, "a"),
	})
	defer snap.Close()

	id := hex.EncodeToString(snap.Header.GetIndexShortID())
	cmd := &Diff{}
	require.NoError(t, cmd.Parse(ctx, []string{id + ":/nope.txt", id + ":/a.txt"}))
	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "could not diff pathnames")
}

// ---------- Execute: recursive diff with adds, removals, type mismatch ----------

func TestCov2DiffRecursiveOnlyInAndMismatch(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bytes.NewBuffer(nil), nil)

	snap1 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/only1.txt", 0644, "1"),
		ptesting.NewMockFile("subdir/shared.txt", 0644, "left"),
		// "x" is a file here
		ptesting.NewMockFile("subdir/x", 0644, "file"),
	})
	defer snap1.Close()
	snap2 := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/only2.txt", 0644, "2"),
		ptesting.NewMockFile("subdir/shared.txt", 0644, "right"),
		// "x" is a directory here -> type mismatch
		ptesting.NewMockDir("subdir/x"),
		ptesting.NewMockFile("subdir/x/inner.txt", 0644, "deep"),
	})
	defer snap2.Close()

	id1 := hex.EncodeToString(snap1.Header.GetIndexShortID())
	id2 := hex.EncodeToString(snap2.Header.GetIndexShortID())
	cmd := &Diff{}
	require.NoError(t, cmd.Parse(ctx, []string{
		"-recursive",
		id1 + ":/subdir",
		id2 + ":/subdir",
	}))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	out := bufOut.String()
	require.Contains(t, out, "only1.txt")
	require.Contains(t, out, "only2.txt")
	require.Contains(t, out, "File type mismatch")
}

// ---------- Parse: highlight + recursive flags both set ----------

func TestCov2ParseFlagsCombined(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	_ = repo
	cmd := &Diff{}
	require.NoError(t, cmd.Parse(ctx, []string{"-highlight", "-recursive", "x", "y"}))
	require.True(t, cmd.Highlight)
	require.True(t, cmd.Recursive)
	require.Equal(t, "x", cmd.Path1)
	require.Equal(t, "y", cmd.Path2)
}
