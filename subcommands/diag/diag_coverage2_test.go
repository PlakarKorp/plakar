package diag

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	_ "github.com/PlakarKorp/integrations/fs/exporter"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/stretchr/testify/require"
)

// --- diag locks: non-empty lock list (loop body) ---------------------------

func TestCov2DiagLocksWithLocks(t *testing.T) {
	repo, snap, ctx, bufOut := covGenSnapshot(t)
	defer snap.Close()

	// Put an exclusive and a shared lock so the Execute loop body runs for both
	// the exclusive and shared formatting branches.
	exclusive := repository.NewExclusiveLock("host-excl")
	var exbuf bytes.Buffer
	require.NoError(t, exclusive.SerializeToStream(&exbuf))
	_, err := repo.PutLock(objects.MAC{0xAA, 0x01}, &exbuf)
	require.NoError(t, err)

	shared := repository.NewSharedLock("host-shared")
	var shbuf bytes.Buffer
	require.NoError(t, shared.SerializeToStream(&shbuf))
	_, err = repo.PutLock(objects.MAC{0xBB, 0x02}, &shbuf)
	require.NoError(t, err)

	bufOut.Reset()
	status, err := runDiag(t, ctx, repo, []string{"diag", "locks"})
	require.NoError(t, err)
	require.Equal(t, 0, status)

	out := bufOut.String()
	require.Contains(t, out, "exclusive")
	require.Contains(t, out, "shared")
	require.Contains(t, out, "host-excl")
	require.Contains(t, out, "host-shared")
}

// --- diag state: error branches --------------------------------------------

func TestCov2DiagStateInvalidHashLength(t *testing.T) {
	repo, snap, ctx, _ := covGenSnapshot(t)
	defer snap.Close()

	// Hash that is not 64 hex chars -> "invalid packfile hash" error.
	status, err := runDiag(t, ctx, repo, []string{"diag", "state", "deadbeef"})
	require.Error(t, err)
	require.Equal(t, 1, status)
}

func TestCov2DiagStateInvalidHex(t *testing.T) {
	repo, snap, ctx, _ := covGenSnapshot(t)
	defer snap.Close()

	// 64 chars but not valid hex -> hex.DecodeString error.
	bad := strings.Repeat("z", 64)
	status, err := runDiag(t, ctx, repo, []string{"diag", "state", bad})
	require.Error(t, err)
	require.Equal(t, 1, status)
}

func TestCov2DiagStateNotFound(t *testing.T) {
	repo, snap, ctx, _ := covGenSnapshot(t)
	defer snap.Close()

	// Well-formed hash that does not name an existing state -> GetState error.
	missing := strings.Repeat("00", 32)
	status, err := runDiag(t, ctx, repo, []string{"diag", "state", missing})
	require.Error(t, err)
	require.Equal(t, 1, status)
}

// --- diag packfile: list-all and error branches ----------------------------

func TestCov2DiagPackfileListAll(t *testing.T) {
	repo, snap, ctx, bufOut := covGenSnapshot(t)
	defer snap.Close()

	// No args -> list every packfile MAC (the len(Args)==0 branch).
	bufOut.Reset()
	status, err := runDiag(t, ctx, repo, []string{"diag", "packfile"})
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.NotEmpty(t, strings.TrimSpace(bufOut.String()))
}

func TestCov2DiagPackfileInvalidHashLength(t *testing.T) {
	repo, snap, ctx, _ := covGenSnapshot(t)
	defer snap.Close()

	status, err := runDiag(t, ctx, repo, []string{"diag", "packfile", "abcd"})
	require.Error(t, err)
	require.Equal(t, 1, status)
}

func TestCov2DiagPackfileInvalidHex(t *testing.T) {
	repo, snap, ctx, _ := covGenSnapshot(t)
	defer snap.Close()

	bad := strings.Repeat("z", 64)
	status, err := runDiag(t, ctx, repo, []string{"diag", "packfile", bad})
	require.Error(t, err)
	require.Equal(t, 1, status)
}

func TestCov2DiagPackfileNotFound(t *testing.T) {
	repo, snap, ctx, _ := covGenSnapshot(t)
	defer snap.Close()

	missing := strings.Repeat("00", 32)
	status, err := runDiag(t, ctx, repo, []string{"diag", "packfile", missing})
	require.Error(t, err)
	require.Equal(t, 1, status)
}

// --- diag contenttype: open error and listing ------------------------------

func TestCov2DiagContentTypeBadSnapshot(t *testing.T) {
	repo, snap, ctx, _ := covGenSnapshot(t)
	defer snap.Close()

	// Unknown snapshot prefix -> OpenSnapshotByPath error.
	status, err := runDiag(t, ctx, repo, []string{"diag", "contenttype", "deadbeef:/"})
	require.Error(t, err)
	require.Equal(t, 1, status)
}

func TestCov2DiagContentTypeRoot(t *testing.T) {
	repo, snap, ctx, bufOut := covGenSnapshot(t)
	defer snap.Close()

	// Root path (empty -> "/") exercises the pathname-normalization branch and
	// iterates the content-type index over the whole tree.
	indexID := snap.Header.GetIndexID()
	bufOut.Reset()
	status, err := runDiag(t, ctx, repo, []string{"diag", "contenttype", hex.EncodeToString(indexID[:])})
	require.NoError(t, err)
	require.Equal(t, 0, status)
}

// --- diag search: open error and parse usage error -------------------------

func TestCov2DiagSearchBadSnapshot(t *testing.T) {
	repo, snap, ctx, _ := covGenSnapshot(t)
	defer snap.Close()

	status, err := runDiag(t, ctx, repo, []string{"diag", "search", "deadbeef:/"})
	require.Error(t, err)
	require.Equal(t, 1, status)
}

func TestCov2DiagSearchUsageError(t *testing.T) {
	_, snap, ctx, _ := covGenSnapshot(t)
	defer snap.Close()

	// Three positional args hits the default branch of the NArg switch.
	subcommand, _, rest := subcommands.Lookup([]string{"diag", "search", "a", "b", "c"})
	require.NotNil(t, subcommand)
	err := subcommand.Parse(ctx, rest)
	require.Error(t, err)
}

// --- diag xattr: open error ------------------------------------------------

func TestCov2DiagXattrBadSnapshot(t *testing.T) {
	repo, snap, ctx, _ := covGenSnapshot(t)
	defer snap.Close()

	status, err := runDiag(t, ctx, repo, []string{"diag", "xattr", "deadbeef:/"})
	require.Error(t, err)
	require.Equal(t, 1, status)
}

func TestCov2DiagXattrRoot(t *testing.T) {
	repo, snap, ctx, bufOut := covGenSnapshot(t)
	defer snap.Close()

	// Root path normalization + scan over the (empty) xattr tree.
	indexID := snap.Header.GetIndexID()
	bufOut.Reset()
	status, err := runDiag(t, ctx, repo, []string{"diag", "xattr", hex.EncodeToString(indexID[:])})
	require.NoError(t, err)
	require.Equal(t, 0, status)
}
