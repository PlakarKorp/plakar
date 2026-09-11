package sync

import (
	"bytes"
	"testing"

	"github.com/PlakarKorp/kloset/logging"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/config"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// cov3ParseCtx builds a minimal context sufficient for the early Parse branches
// (argument validation) that run before any peer-store resolution.
func cov3ParseCtx(t *testing.T) *appcontext.AppContext {
	t.Helper()
	ctx := appcontext.NewAppContext()
	ctx.Stdout = bytes.NewBuffer(nil)
	ctx.Stderr = bytes.NewBuffer(nil)
	ctx.SetLogger(logging.NewLogger(ctx.Stdout, ctx.Stderr))
	ctx.Config = config.NewConfig()
	return ctx
}

// --- Parse: argument-shape validation (pre store-resolution) ---------------

func TestCov3SyncParseTooManyArgs(t *testing.T) {
	err := (&Sync{}).Parse(cov3ParseCtx(t), []string{"a", "to", "b", "c"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "too many arguments")
}

func TestCov3SyncParseNoArgs(t *testing.T) {
	err := (&Sync{}).Parse(cov3ParseCtx(t), []string{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "usage: sync")
}

func TestCov3SyncParseOneArg(t *testing.T) {
	err := (&Sync{}).Parse(cov3ParseCtx(t), []string{"to"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "usage: sync")
}

func TestCov3SyncParseInvalidDirection(t *testing.T) {
	err := (&Sync{}).Parse(cov3ParseCtx(t), []string{"sideways", "somerepo"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid direction")
}

func TestCov3SyncParseUnresolvablePeer(t *testing.T) {
	// "to @nope": direction is valid, so Parse proceeds to GetRepository which
	// fails because @nope is not configured.
	err := (&Sync{}).Parse(cov3ParseCtx(t), []string{"to", "@nope"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "peer store")
}

func TestCov3SyncParseUnopenablePeerLocation(t *testing.T) {
	// A bare path with no scheme/configured store is treated as a location but
	// storage.Open must fail to open it.
	err := (&Sync{}).Parse(cov3ParseCtx(t), []string{"to", "/nonexistent/plakar/repo/path"})
	require.Error(t, err)
}

func TestCov3SyncParseThreeArgsWithFilterWarns(t *testing.T) {
	// Three positional args + a locate filter set => the "snapshot specified,
	// filters will be ignored" warning branch is exercised. The peer is still
	// unopenable, so Parse ultimately errors, but the warning path is covered.
	ctx := cov3ParseCtx(t)
	err := (&Sync{}).Parse(ctx, []string{"-name", "foo", "deadbeef", "to", "/nonexistent/peer"})
	require.Error(t, err)
}

// --- Execute: same-store rejection ----------------------------------------

func TestCov3SyncExecuteSameStore(t *testing.T) {
	// Syncing a repository to itself must be rejected. We point the peer at the
	// very same on-disk location as the local repo so both resolve to the same
	// RepositoryID, origin and root.
	fixture := setupSync(t, nil, nil)
	fixture.localCtx.Config.Repositories["self"] = map[string]string{
		"location": fixture.localRepo.Root(),
	}

	cmd := &Sync{}
	require.NoError(t, cmd.Parse(fixture.localCtx, []string{"to", "@self"}))
	status, err := cmd.Execute(fixture.localCtx, fixture.localRepo)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "same store")
}

// --- Execute: no matching snapshot ----------------------------------------

func TestCov3SyncExecuteNoMatchingSnapshot(t *testing.T) {
	// Empty source repo + direction "to": LocateSnapshotIDs returns nothing and
	// Execute short-circuits with status 0 and an informational log.
	fixture := setupSync(t, nil, nil)

	cmd := &Sync{}
	require.NoError(t, cmd.Parse(fixture.localCtx, []string{"to", fixture.peerArg}))
	status, err := cmd.Execute(fixture.localCtx, fixture.localRepo)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Contains(t, fixture.output.String(), "No matching snapshot found")
}

// --- Execute: memory packfile storage option ------------------------------

func TestCov3SyncExecuteMemoryPackfiles(t *testing.T) {
	// Exercise the PackfileTempStorage == "memory" branch (no temp dir created)
	// with a real one-snapshot sync.
	fixture := setupSync(t, nil, nil)
	snap := ptesting.GenerateSnapshot(t, fixture.localRepo, mockFiles)
	defer snap.Close()

	cmd := &Sync{}
	require.NoError(t, cmd.Parse(fixture.localCtx, []string{"-packfiles", "memory", "to", fixture.peerArg}))
	status, err := cmd.Execute(fixture.localCtx, fixture.localRepo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	peerIDs := snapshotIDs(t, fixture.peerRepo)
	require.Contains(t, peerIDs, snap.Header.Identifier)
}
