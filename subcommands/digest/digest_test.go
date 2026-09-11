package digest

import (
	"bytes"
	"encoding/hex"
	"os"
	"strings"
	"testing"

	_ "github.com/PlakarKorp/integrations/fs/exporter"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func generateSnapshot(t *testing.T, bufOut *bytes.Buffer, bufErr *bytes.Buffer) (*repository.Repository, *snapshot.Snapshot, *appcontext.AppContext) {
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockDir("another_subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
		ptesting.NewMockFile("another_subdir/bar.txt", 0644, "hello bar"),
	})
	return repo, snap, ctx
}

func TestExecuteCmdDigestDefault(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	indexId := snap.Header.GetIndexID()
	args := []string{hex.EncodeToString(indexId[:])}

	subcommand := &Digest{}
	err := subcommand.Parse(ctx, args)
	require.NoError(t, err)
	require.NotNil(t, subcommand)

	status, err := subcommand.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// output should look like this
	// SHA256 (/tmp/tmp_to_backup3363028982/another_subdir/bar) = b585d5afa0d0a97a7c217eeb9d9adf08fc63188d4204fc7d537a178224b477e6
	// SHA256 (/tmp/tmp_to_backup3363028982/subdir/dummy.txt) = f4da3ebff9dbd21cfb270054dee6948f96de93f68f525e0bf4067ce2f9e2d639
	// SHA256 (/tmp/tmp_to_backup3363028982/subdir/foo.txt) = 6c8aa524fae27a3607f9c4204567b65d48341b3bcc0e36e9e50856aaaf073d21
	// SHA256 (/tmp/tmp_to_backup3363028982/subdir/to_exclude) = dd7117865f65a87aba1e171b82e073914a2cdffb1b34407dea682f62c3dc72e0

	output := bufOut.String()
	require.Contains(t, output, "dummy.txt")
	lines := strings.Split(strings.Trim(output, "\n"), "\n")
	for _, line := range lines {
		require.Contains(t, line, "SHA256 (")
	}
}

func TestExecuteCmdDigestBadSnapshotPath(t *testing.T) {
	// An unresolvable snapshot path is logged as an error and skipped; Execute
	// still returns 0.
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	cmd := &Digest{}
	require.NoError(t, cmd.Parse(ctx, []string{"deadbeefdeadbeef:/nope"}))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Contains(t, bufErr.String(), "digest:")
}

func TestExecuteCmdDigestSingleFile(t *testing.T) {
	// Targeting one regular file exercises the GetEntry + non-dir path that
	// produces a single digest line.
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	indexId := snap.Header.GetIndexID()
	id := hex.EncodeToString(indexId[:])
	cmd := &Digest{}
	require.NoError(t, cmd.Parse(ctx, []string{id + ":/subdir/dummy.txt"}))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Contains(t, bufOut.String(), "dummy.txt")
	require.Equal(t, 1, len(strings.Split(strings.Trim(bufOut.String(), "\n"), "\n")))
}

func TestExecuteCmdDigestCancelledContext(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	indexId := snap.Header.GetIndexID()
	id := hex.EncodeToString(indexId[:])
	cmd := &Digest{}
	require.NoError(t, cmd.Parse(ctx, []string{id}))

	ctx.GetInner().Cancel(nil)
	// displayDigests bails on ctx.Err(); Execute swallows it and still returns 0,
	// but produces no digest output.
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Empty(t, strings.TrimSpace(bufOut.String()))
}

func TestExecuteCmdDigestNoParam(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	_, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	args := []string{}

	subcommand := &Digest{}
	err := subcommand.Parse(ctx, args)
	require.Error(t, err, "at least one parameter is required")
}

func TestExecuteCmdDigestWrongHashing(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	_, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	indexId := snap.Header.GetIndexID()
	args := []string{"-hashing", "md5", hex.EncodeToString(indexId[:])}

	subcommand := &Digest{}
	err := subcommand.Parse(ctx, args)
	require.Error(t, err, "at least one parameter is required")
}
