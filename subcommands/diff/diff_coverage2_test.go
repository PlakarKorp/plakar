package diff

import (
	"bytes"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	_ "github.com/PlakarKorp/integrations/fs/exporter"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

var (
	errCov2DiffWalk   = errors.New("diff test walk failure")
	errCov2DiffWriter = errors.New("diff test writer failure")
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
	require.Contains(t, out, "- /subdir/only1.txt")
	require.Contains(t, out, "+ /subdir/only2.txt")
	require.Contains(t, out, "~ /subdir/shared.txt")
	require.Contains(t, out, "- /subdir/x")
	require.Contains(t, out, "+ /subdir/x/inner.txt")
	require.NotContains(t, out, "Only in")
	require.NotContains(t, out, "Common subdirectories")
	require.NotContains(t, out, "File type mismatch")
}

func TestCov2DiffRecursiveReadDirErrors(t *testing.T) {
	cmd := &Diff{}
	emptyFS := fstest.MapFS{}
	rootFS := fstest.MapFS{
		"present": {Mode: fs.ModeDir},
	}

	err := cmd.diff_directories_recursive(io.Discard, "left", emptyFS, "missing", "right", rootFS, ".")
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot read directory missing")

	err = cmd.diff_directories_recursive(io.Discard, "left", rootFS, ".", "right", emptyFS, "missing")
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot read directory missing")
}

func TestCov2DiffRecursiveOnlyEntryErrors(t *testing.T) {
	cmd := &Diff{}
	ghostFS := cov2ReadDirOnlyFS{
		entries: []fs.DirEntry{cov2DirEntry{name: "ghost"}},
	}
	emptyFS := fstest.MapFS{}

	err := cmd.diff_directories_recursive(io.Discard, "left", ghostFS, ".", "right", emptyFS, ".")
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot stat recursive diff entry")

	err = cmd.diff_directories_recursive(io.Discard, "left", emptyFS, ".", "right", ghostFS, ".")
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot stat recursive diff entry")
}

func TestCov2DiffRecursiveWriteStringError(t *testing.T) {
	cmd := &Diff{}
	leftFS := fstest.MapFS{
		"f.txt": {Data: []byte("left\n")},
	}
	rightFS := fstest.MapFS{
		"f.txt": {Data: []byte("right\n")},
	}

	err := cmd.diff_directories_recursive(cov2FailingWriter{}, "left", leftFS, ".", "right", rightFS, ".")
	require.ErrorIs(t, err, errCov2DiffWriter)
}

func TestCov2RecursiveOnlyEmptyDirectoryAndDisplayRoot(t *testing.T) {
	out := bytes.NewBuffer(nil)
	emptyDirFS := fstest.MapFS{
		"empty": {Mode: fs.ModeDir},
	}

	require.NoError(t, writeRecursiveOnly(out, diffAddedMarker, emptyDirFS, "empty"))
	require.Equal(t, "+ /empty\n", out.String())
	require.Equal(t, "/", displayRecursivePath(""))
}

func TestCov2RecursiveOnlyWalkError(t *testing.T) {
	out := bytes.NewBuffer(nil)

	err := writeRecursiveOnly(out, diffAddedMarker, cov2WalkErrorFS{}, "root")
	require.ErrorIs(t, err, errCov2DiffWalk)
	require.Contains(t, err.Error(), "cannot walk recursive diff entry root")
}

type cov2ReadDirOnlyFS struct {
	entries []fs.DirEntry
}

func (fsys cov2ReadDirOnlyFS) Open(string) (fs.File, error) {
	return nil, fs.ErrNotExist
}

func (fsys cov2ReadDirOnlyFS) ReadDir(string) ([]fs.DirEntry, error) {
	return fsys.entries, nil
}

type cov2DirEntry struct {
	name  string
	isDir bool
}

func (entry cov2DirEntry) Name() string {
	return entry.name
}

func (entry cov2DirEntry) IsDir() bool {
	return entry.isDir
}

func (entry cov2DirEntry) Type() fs.FileMode {
	if entry.isDir {
		return fs.ModeDir
	}
	return 0
}

func (entry cov2DirEntry) Info() (fs.FileInfo, error) {
	return nil, fs.ErrNotExist
}

type cov2FailingWriter struct{}

func (cov2FailingWriter) Write([]byte) (int, error) {
	return 0, errCov2DiffWriter
}

type cov2WalkErrorFS struct{}

func (cov2WalkErrorFS) Open(name string) (fs.File, error) {
	if name == "root" {
		return &cov2DirFile{
			entries: []fs.DirEntry{cov2DirEntry{name: "bad", isDir: true}},
		}, nil
	}
	return nil, errCov2DiffWalk
}

type cov2DirFile struct {
	entries []fs.DirEntry
	read    bool
}

func (file *cov2DirFile) Stat() (fs.FileInfo, error) {
	return cov2FileInfo{mode: fs.ModeDir}, nil
}

func (file *cov2DirFile) Read([]byte) (int, error) {
	return 0, io.EOF
}

func (file *cov2DirFile) Close() error {
	return nil
}

func (file *cov2DirFile) ReadDir(int) ([]fs.DirEntry, error) {
	if file.read {
		return nil, io.EOF
	}
	file.read = true
	return file.entries, nil
}

type cov2FileInfo struct {
	mode fs.FileMode
}

func (info cov2FileInfo) Name() string {
	return "root"
}

func (info cov2FileInfo) Size() int64 {
	return 0
}

func (info cov2FileInfo) Mode() fs.FileMode {
	return info.mode
}

func (info cov2FileInfo) ModTime() time.Time {
	return time.Time{}
}

func (info cov2FileInfo) IsDir() bool {
	return info.mode.IsDir()
}

func (info cov2FileInfo) Sys() any {
	return nil
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
