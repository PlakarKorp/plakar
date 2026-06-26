package diag

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// run is a small helper that looks up, parses and executes a diag subcommand,
// returning the status and execution error. Parse errors are surfaced via the
// returned parseErr.
func run(t *testing.T, ctx *appcontext.AppContext, repo *repository.Repository, args ...string) (status int, execErr error, parseErr error) {
	t.Helper()
	subcommand, _, rest := subcommands.Lookup(args)
	require.NotNil(t, subcommand)
	if perr := subcommand.Parse(ctx, rest); perr != nil {
		return 0, nil, perr
	}
	status, execErr = subcommand.Execute(ctx, repo)
	return status, execErr, nil
}

func TestDiagStateListAndErrors(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	// No args: lists state ids (one per line).
	status, execErr, parseErr := run(t, ctx, repo, "diag", "state")
	require.NoError(t, parseErr)
	require.NoError(t, execErr)
	require.Equal(t, 0, status)
	require.NotEmpty(t, strings.TrimSpace(bufOut.String()))

	// Wrong-length hash -> Execute error.
	bufOut.Reset()
	status, execErr, parseErr = run(t, ctx, repo, "diag", "state", "deadbeef")
	require.NoError(t, parseErr)
	require.Error(t, execErr)
	require.Equal(t, 1, status)
	require.Contains(t, execErr.Error(), "invalid packfile hash")

	// Right length but invalid hex -> Execute error.
	bufOut.Reset()
	status, execErr, parseErr = run(t, ctx, repo, "diag", "state", strings.Repeat("zz", 32))
	require.NoError(t, parseErr)
	require.Error(t, execErr)
	require.Equal(t, 1, status)
}

func TestDiagPackfileListAndErrors(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	// No args: lists packfile ids.
	status, execErr, parseErr := run(t, ctx, repo, "diag", "packfile")
	require.NoError(t, parseErr)
	require.NoError(t, execErr)
	require.Equal(t, 0, status)
	require.NotEmpty(t, strings.TrimSpace(bufOut.String()))

	// Wrong-length hash -> Execute error.
	bufOut.Reset()
	status, execErr, parseErr = run(t, ctx, repo, "diag", "packfile", "deadbeef")
	require.NoError(t, parseErr)
	require.Error(t, execErr)
	require.Equal(t, 1, status)
	require.Contains(t, execErr.Error(), "invalid packfile hash")

	// Right length but invalid hex -> Execute error.
	bufOut.Reset()
	status, execErr, parseErr = run(t, ctx, repo, "diag", "packfile", strings.Repeat("zz", 32))
	require.NoError(t, parseErr)
	require.Error(t, execErr)
	require.Equal(t, 1, status)

	// Valid length & hex but unknown packfile -> GetPackfile error.
	bufOut.Reset()
	status, execErr, parseErr = run(t, ctx, repo, "diag", "packfile", strings.Repeat("00", 32))
	require.NoError(t, parseErr)
	require.Error(t, execErr)
	require.Equal(t, 1, status)
}

func TestDiagObjectErrors(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	// Missing argument -> Parse error.
	_, _, parseErr := run(t, ctx, repo, "diag", "object")
	require.Error(t, parseErr)

	// Wrong-length hash -> Execute error.
	status, execErr, parseErr := run(t, ctx, repo, "diag", "object", "deadbeef")
	require.NoError(t, parseErr)
	require.Error(t, execErr)
	require.Equal(t, 1, status)
	require.Contains(t, execErr.Error(), "invalid object hash")

	// Right length, invalid hex -> Execute error.
	status, execErr, parseErr = run(t, ctx, repo, "diag", "object", strings.Repeat("zz", 32))
	require.NoError(t, parseErr)
	require.Error(t, execErr)
	require.Equal(t, 1, status)

	// Valid hash that does not exist -> GetBlob error.
	status, execErr, parseErr = run(t, ctx, repo, "diag", "object", strings.Repeat("00", 32))
	require.NoError(t, parseErr)
	require.Error(t, execErr)
	require.Equal(t, 1, status)
}

func TestDiagVFSErrors(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	// Missing argument -> Parse error.
	_, _, parseErr := run(t, ctx, repo, "diag", "vfs")
	require.Error(t, parseErr)

	// Unknown snapshot prefix -> Execute error from OpenSnapshotByPath.
	status, execErr, parseErr := run(t, ctx, repo, "diag", "vfs", strings.Repeat("00", 32))
	require.NoError(t, parseErr)
	require.Error(t, execErr)
	require.Equal(t, 1, status)

	// Valid snapshot but missing path -> GetEntry error.
	indexID := snap.Header.GetIndexID()
	status, execErr, parseErr = run(t, ctx, repo,
		"diag", "vfs", hex.EncodeToString(indexID[:])+":/does/not/exist")
	require.NoError(t, parseErr)
	require.Error(t, execErr)
	require.Equal(t, 1, status)
}

func TestDiagXattrErrorsAndDirVariant(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	// Missing argument -> Parse error.
	_, _, parseErr := run(t, ctx, repo, "diag", "xattr")
	require.Error(t, parseErr)

	// Unknown snapshot prefix -> Execute error.
	status, execErr, parseErr := run(t, ctx, repo, "diag", "xattr", strings.Repeat("00", 32))
	require.NoError(t, parseErr)
	require.Error(t, execErr)
	require.Equal(t, 1, status)

	// Snapshot with no path: pathname defaults to "/" and the directory branch
	// runs over an empty xattr tree (no output, status 0).
	bufOut.Reset()
	indexID := snap.Header.GetIndexID()
	status, execErr, parseErr = run(t, ctx, repo,
		"diag", "xattr", hex.EncodeToString(indexID[:]))
	require.NoError(t, parseErr)
	require.NoError(t, execErr)
	require.Equal(t, 0, status)
}

func TestDiagContentTypeErrorsAndVariant(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	// Missing argument -> Parse error.
	_, _, parseErr := run(t, ctx, repo, "diag", "contenttype")
	require.Error(t, parseErr)

	// Unknown snapshot prefix -> Execute error.
	status, execErr, parseErr := run(t, ctx, repo, "diag", "contenttype", strings.Repeat("00", 32))
	require.NoError(t, parseErr)
	require.Error(t, execErr)
	require.Equal(t, 1, status)

	// No path: defaults pathname to "/" and walks the whole content-type index.
	bufOut.Reset()
	indexID := snap.Header.GetIndexID()
	status, execErr, parseErr = run(t, ctx, repo,
		"diag", "contenttype", hex.EncodeToString(indexID[:]))
	require.NoError(t, parseErr)
	require.NoError(t, execErr)
	require.Equal(t, 0, status)
}

func TestDiagSearchVariantsAndErrors(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	// No args -> Parse error (default branch of the NArg switch).
	_, _, parseErr := run(t, ctx, repo, "diag", "search")
	require.Error(t, parseErr)

	// Three args -> Parse error.
	_, _, parseErr = run(t, ctx, repo, "diag", "search", "a", "b", "c")
	require.Error(t, parseErr)

	indexID := snap.Header.GetIndexID()
	// Two-arg form with a mime filter exercises the case-2 parse branch.
	bufOut.Reset()
	status, execErr, parseErr := run(t, ctx, repo,
		"diag", "search", hex.EncodeToString(indexID[:])+":subdir/", "text/plain")
	require.NoError(t, parseErr)
	require.NoError(t, execErr)
	require.Equal(t, 0, status)

	// Unknown snapshot prefix -> Execute error.
	bufOut.Reset()
	status, execErr, parseErr = run(t, ctx, repo, "diag", "search", strings.Repeat("00", 32))
	require.NoError(t, parseErr)
	require.Error(t, execErr)
	require.Equal(t, 1, status)
}

func TestDiagSnapshotBadIDError(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, snap, ctx := generateSnapshot(t, bufOut, bufErr)
	defer snap.Close()

	// Unknown snapshot id -> Execute error.
	status, execErr, parseErr := run(t, ctx, repo, "diag", "snapshot", strings.Repeat("00", 32))
	require.NoError(t, parseErr)
	require.Error(t, execErr)
	require.Equal(t, 1, status)
}

func TestDiagRepositoryEncrypted(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	passphrase := []byte("a-very-strong-test-passphrase")
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, &passphrase)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	})
	defer snap.Close()

	// Default `diag` over an encrypted repo prints the Encryption block.
	status, execErr, parseErr := run(t, ctx, repo, "diag")
	require.NoError(t, parseErr)
	require.NoError(t, execErr)
	require.Equal(t, 0, status)

	output := bufOut.String()
	require.Contains(t, output, "Encryption:")
	require.Contains(t, output, "KDF:")
	require.Contains(t, output, "Canary:")
	require.Contains(t, output, "Snapshots: 1")
}
