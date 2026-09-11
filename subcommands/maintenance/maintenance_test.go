package maintenance

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/PlakarKorp/integrations/fs/exporter"
	"github.com/PlakarKorp/kloset/connectors/storage"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

// --- helpers ---------------------------------------------------------------

// resetEnv clears all maintenance-relevant env vars for the duration of the
// test. Using t.Setenv ensures they're restored at test end.
func resetEnv(t *testing.T) {
	t.Helper()
	t.Setenv("PLAKAR_GRACEPERIOD", "")
	t.Setenv("PLAKAR_NODELETION", "")
	t.Setenv("PLAKAR_LOCKLESS", "")
}

// freshRepo builds a fresh fs-backed repository with cleared env. Returns
// repo, ctx, and the captured stdout/stderr buffers.
func freshRepo(t *testing.T) (*repository.Repository, *appcontext.AppContext, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	resetEnv(t)
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	return repo, ctx, bufOut, bufErr
}

// runMaintenance constructs and runs a Maintenance subcommand. Output buffers
// are cleared first so each call sees only its own log lines. Because Unlock
// only closes the ping channel and the background goroutine performs the
// actual DeleteLock asynchronously, we wait for the lock count to settle
// back to its pre-Execute value before returning. Otherwise back-to-back
// runs can race: run N+1's Lock() calls GetLocks() and sees run N's
// not-yet-deleted lock, then races GetLock(N's MAC) against the background
// DeleteLock and fails with "no such file or directory".
//
//nolint:staticcheck // ST1008: test helper, error kept in fixed position alongside other outputs
func runMaintenance(t *testing.T, ctx *appcontext.AppContext, repo *repository.Repository,
	bufOut, bufErr *bytes.Buffer) (int, error, string, string) {
	t.Helper()
	bufOut.Reset()
	bufErr.Reset()

	locksBefore, err := repo.GetLocks()
	require.NoError(t, err)
	preCount := len(locksBefore)

	cmd := &Maintenance{}
	require.NoError(t, cmd.Parse(ctx, nil))
	status, err := cmd.Execute(ctx, repo)

	// Wait for any lock the maintenance goroutine installed to drain. If
	// Execute errored without ever taking a lock the count may already be
	// at preCount; Eventually returns immediately in that case.
	require.Eventually(t, func() bool {
		locks, lerr := repo.GetLocks()
		if lerr != nil {
			return false
		}
		return len(locks) <= preCount
	}, 2*time.Second, 10*time.Millisecond,
		"maintenance lock must be released before next run")

	return status, err, bufOut.String(), bufErr.String()
}

func simpleFiles() []ptesting.MockFile {
	return []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/a.txt", 0644, "alpha"),
		ptesting.NewMockFile("subdir/b.txt", 0644, "bravo"),
	}
}

func extraFiles(prefix string) []ptesting.MockFile {
	return []ptesting.MockFile{
		ptesting.NewMockDir(prefix),
		ptesting.NewMockFile(prefix+"/x.txt", 0644, prefix+"-x"),
		ptesting.NewMockFile(prefix+"/y.txt", 0644, strings.Repeat(prefix, 1024)),
	}
}

// storePackfiles returns the packfile MACs currently visible in the underlying
// storage (not the state index). Used to verify the actual delete on disk.
func storePackfiles(t *testing.T, repo *repository.Repository) map[objects.MAC]struct{} {
	t.Helper()
	list, err := repo.Store().List(repo.AppContext(), storage.StorageResourcePackfile)
	require.NoError(t, err)
	out := make(map[objects.MAC]struct{}, len(list))
	for _, m := range list {
		out[m] = struct{}{}
	}
	return out
}

// statePackfiles returns the packfiles known to the in-memory state of the
// repository (i.e. reachable from snapshots).
func statePackfiles(t *testing.T, repo *repository.Repository) map[objects.MAC]struct{} {
	t.Helper()
	out := make(map[objects.MAC]struct{})
	for m := range repo.ListPackfiles() {
		out[m] = struct{}{}
	}
	return out
}

// colouredPackfiles returns the packfiles currently flagged for deletion.
func colouredPackfiles(t *testing.T, repo *repository.Repository) map[objects.MAC]time.Time {
	t.Helper()
	out := make(map[objects.MAC]time.Time)
	for m, when := range repo.ListColouredPackfiles() {
		out[m] = when
	}
	return out
}

// rawPackfileContents downloads the raw bytes of a packfile from the store,
// used in NODELETION verification.
func rawPackfileContents(t *testing.T, repo *repository.Repository, mac objects.MAC) ([]byte, error) {
	t.Helper()
	rd, err := repo.Store().Get(repo.AppContext(), storage.StorageResourcePackfile, mac, nil)
	if err != nil {
		return nil, err
	}
	defer rd.Close()
	return io.ReadAll(rd)
}

// preInstallExclusiveLock writes an exclusive lock with the given timestamp on
// behalf of someone other than us, so the maintenance lock acquisition has to
// observe it.
func preInstallExclusiveLock(t *testing.T, repo *repository.Repository, hostname string, when time.Time) objects.MAC {
	t.Helper()
	lockID := objects.RandomMAC()
	lock := repository.NewExclusiveLock(hostname)
	// override timestamp via reflection-free path: re-serialize a forged Lock
	// using the same wire format.  We can do this because the Lock struct is
	// exported.
	lock.Timestamp = when

	buf := &bytes.Buffer{}
	require.NoError(t, lock.SerializeToStream(buf))
	_, err := repo.PutLock(lockID, buf)
	require.NoError(t, err)
	return lockID
}

// requireLockExists / requireLockGone assert presence/absence in the store.
func requireLockExists(t *testing.T, repo *repository.Repository, id objects.MAC) {
	t.Helper()
	locks, err := repo.GetLocks()
	require.NoError(t, err)
	for _, l := range locks {
		if l == id {
			return
		}
	}
	t.Fatalf("expected lock %x to exist", id)
}

func requireLockGone(t *testing.T, repo *repository.Repository, id objects.MAC) {
	t.Helper()
	locks, err := repo.GetLocks()
	require.NoError(t, err)
	for _, l := range locks {
		if l == id {
			t.Fatalf("expected lock %x to be gone, but still present", id)
		}
	}
}

// --- Parse ----------------------------------------------------------------

func TestParseEmptyArgs(t *testing.T) {
	resetEnv(t)
	_, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	cmd := &Maintenance{}
	err := cmd.Parse(ctx, nil)
	require.NoError(t, err)
}

func TestParseExtraArgsIgnored(t *testing.T) {
	resetEnv(t)
	_, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	cmd := &Maintenance{}
	// Maintenance takes no flags or positional args, but extra args should
	// not break Parse (subcommand simply doesn't look at them).
	err := cmd.Parse(ctx, []string{"ignored", "args"})
	require.NoError(t, err)
}

func TestParseCapturesSecret(t *testing.T) {
	resetEnv(t)
	pass := []byte("hunter2")
	_, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), &pass)
	cmd := &Maintenance{}
	require.NoError(t, cmd.Parse(ctx, nil))
	require.NotNil(t, cmd.RepositorySecret, "Parse must capture ctx.GetSecret() so encrypted repos can be opened during Execute")
	require.Equal(t, ctx.GetSecret(), cmd.RepositorySecret)
}

// --- Happy paths / no-op ---------------------------------------------------

func TestExecuteEmptyRepository(t *testing.T) {
	repo, ctx, bufOut, bufErr := freshRepo(t)
	status, err, out, errOut := runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Contains(t, out, "maintenance: Coloured 0 packfiles (0 orphaned) for deletion")
	require.Contains(t, out, "maintenance: 0 blobs and 0 packfiles were removed")
	require.NotContains(t, out, "scheduled to be removed in") // nothing coloured, no grace line
	require.Empty(t, errOut)
}

func TestExecuteOneSnapshotNoOp(t *testing.T) {
	repo, ctx, bufOut, bufErr := freshRepo(t)
	ptesting.GenerateSnapshot(t, repo, simpleFiles())

	before := storePackfiles(t, repo)
	require.NotEmpty(t, before, "snapshot should have produced at least one packfile")

	status, err, out, errOut := runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Contains(t, out, "maintenance: Coloured 0 packfiles (0 orphaned) for deletion")
	require.Contains(t, out, "maintenance: 0 blobs and 0 packfiles were removed")
	require.Empty(t, errOut)

	after := storePackfiles(t, repo)
	require.Equal(t, before, after, "maintenance must not touch packfiles when nothing is unreferenced")
}

func TestExecuteMultipleSnapshotsNoOp(t *testing.T) {
	repo, ctx, bufOut, bufErr := freshRepo(t)
	ptesting.GenerateSnapshot(t, repo, simpleFiles(), ptesting.WithName("snap1"))
	ptesting.GenerateSnapshot(t, repo, extraFiles("set-a"), ptesting.WithName("snap2"))
	ptesting.GenerateSnapshot(t, repo, extraFiles("set-b"), ptesting.WithName("snap3"))

	before := storePackfiles(t, repo)
	stateBefore := statePackfiles(t, repo)
	require.Equal(t, len(before), len(stateBefore), "store and state should be in sync before maintenance")

	status, err, _, errOut := runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Empty(t, errOut)

	require.Equal(t, before, storePackfiles(t, repo))
	require.Equal(t, stateBefore, statePackfiles(t, repo))
}

func TestExecuteIdempotent(t *testing.T) {
	repo, ctx, bufOut, bufErr := freshRepo(t)
	ptesting.GenerateSnapshot(t, repo, simpleFiles())

	for i := 0; i < 3; i++ {
		status, err, out, errOut := runMaintenance(t, ctx, repo, bufOut, bufErr)
		require.NoError(t, err, "run %d", i)
		require.Equal(t, 0, status, "run %d", i)
		require.Contains(t, out, "Coloured 0 packfiles", "run %d", i)
		require.Contains(t, out, "0 blobs and 0 packfiles were removed", "run %d", i)
		require.Empty(t, errOut, "run %d", i)
	}
}

// --- Coloring pass --------------------------------------------------------

// primeAndDelete is the canonical fixture for "snapshot's packfiles must be
// coloured": (1) run maintenance once so its cache learns about every live
// snapshot's (packfile, snapshot) mapping, (2) DeleteSnapshot the target,
// then (3) RebuildState so that ListDeletedSnapShots actually reports it on
// the next run. After this helper, a second maintenance run should colour
// the target snapshot's exclusive packfiles.
func primeAndDelete(t *testing.T, ctx *appcontext.AppContext, repo *repository.Repository,
	bufOut, bufErr *bytes.Buffer, snapID objects.MAC) {
	t.Helper()
	status, err, _, _ := runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err, "priming maintenance run must succeed")
	require.Equal(t, 0, status)
	require.NoError(t, repo.DeleteSnapshot(snapID))
	require.NoError(t, repo.RebuildState())
}

// colourRunAndRebuild runs maintenance to perform a colour pass, then
// rebuilds the repository state so the newly-committed colour entries become
// visible to the next maintenance Execute and to test assertions about
// repo.ListColouredPackfiles().
func colourRunAndRebuild(t *testing.T, ctx *appcontext.AppContext, repo *repository.Repository,
	bufOut, bufErr *bytes.Buffer) string {
	t.Helper()
	status, err, out, _ := runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.NoError(t, repo.RebuildState())
	return out
}

func TestColourPassMarksUnreferencedPackfiles(t *testing.T) {
	repo, ctx, bufOut, bufErr := freshRepo(t)
	snap1 := ptesting.GenerateSnapshot(t, repo, simpleFiles(), ptesting.WithName("snap1"))
	ptesting.GenerateSnapshot(t, repo, extraFiles("only-in-two"), ptesting.WithName("snap2"))

	storeBefore := storePackfiles(t, repo)

	primeAndDelete(t, ctx, repo, bufOut, bufErr, snap1.Header.GetIndexID())

	// Run the colour pass and explicitly rebuild so we can observe the
	// state-level effect. (Maintenance commits its colourings to remote
	// state but does NOT merge them into the in-memory state, so a
	// RebuildState is required before assertions.)
	out := colourRunAndRebuild(t, ctx, repo, bufOut, bufErr)

	coloured := colouredPackfiles(t, repo)
	require.NotEmpty(t, coloured, "snap1's packfiles must be coloured for deletion")
	// Output must claim a positive coloured count and zero orphans (the
	// packfiles are still tracked via the deleted-snapshot path, not the
	// storage-vs-state diff that defines "orphan").
	require.Regexp(t, `maintenance: Coloured [1-9]\d* packfiles \(0 orphaned\) for deletion`, out)
	require.Contains(t, out, "scheduled to be removed in")
	// Within the default 7d grace period nothing must actually be deleted.
	require.Contains(t, out, "maintenance: 0 blobs and 0 packfiles were removed")
	require.Equal(t, storeBefore, storePackfiles(t, repo), "store contents must be unchanged within grace period")
}

func TestColourPassDetectsOrphanInsideGrace(t *testing.T) {
	// Default 7d grace. Synthetic orphan with a recent footer timestamp is
	// inside the grace window and must NOT be reported as orphaned (and not
	// coloured).
	repo, ctx, bufOut, bufErr := freshRepo(t)
	snap1 := ptesting.GenerateSnapshot(t, repo, simpleFiles(), ptesting.WithName("snap1"))
	ptesting.GenerateSnapshot(t, repo, extraFiles("safe"), ptesting.WithName("snap2"))

	// Snapshot is deleted but state is NOT yet aware that we removed it.
	// Calling DeleteSnapshot alone (without RebuildState) leaves the
	// packfiles still referenced through the live state on disk, but adds
	// snap1 to ListDeletedSnapShots.  Maintenance.updateCache then triggers
	// the deleted-snapshot cleanup path.
	require.NoError(t, repo.DeleteSnapshot(snap1.Header.GetIndexID()))

	status, err, out, _ := runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	// Without RebuildState, snap1's packfiles are still tracked, but they
	// are not coloured either because the live state still indexes them.
	require.Contains(t, out, "Coloured 0 packfiles (0 orphaned)")
}

func TestColourThenSweepDeletesAfterRebuild(t *testing.T) {
	// Full cycle: prime → delete → maintenance(colour) → rebuild →
	// maintenance(sweep with shrunken grace). The colour and sweep passes
	// CANNOT share state within a single Execute (CommitTransaction writes
	// to remote but does not merge into the in-memory state), so we must
	// rebuild between them.
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	resetEnv(t)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap1 := ptesting.GenerateSnapshot(t, repo, simpleFiles(), ptesting.WithName("snap1"))
	ptesting.GenerateSnapshot(t, repo, extraFiles("survivor"), ptesting.WithName("snap2"))

	storeBefore := storePackfiles(t, repo)
	primeAndDelete(t, ctx, repo, bufOut, bufErr, snap1.Header.GetIndexID())
	out1 := colourRunAndRebuild(t, ctx, repo, bufOut, bufErr)
	require.Regexp(t, `Coloured [1-9]\d* packfiles`, out1)
	coloured := colouredPackfiles(t, repo)
	require.NotEmpty(t, coloured)

	t.Setenv("PLAKAR_GRACEPERIOD", "1ns")
	status, err, out2, _ := runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Regexp(t, `[1-9]\d* packfiles were removed`, out2)

	storeAfter := storePackfiles(t, repo)
	require.Less(t, len(storeAfter), len(storeBefore),
		"sweep must have deleted at least one packfile from the store")
}

func TestColourPassSkipsAlreadyDeletedPackfiles(t *testing.T) {
	// After a colour pass marks packfiles, a subsequent run within the grace
	// period should not re-colour them.
	repo, ctx, bufOut, bufErr := freshRepo(t)
	snap1 := ptesting.GenerateSnapshot(t, repo, simpleFiles(), ptesting.WithName("snap1"))
	ptesting.GenerateSnapshot(t, repo, extraFiles("keep"), ptesting.WithName("snap2"))
	primeAndDelete(t, ctx, repo, bufOut, bufErr, snap1.Header.GetIndexID())

	// First real coloring run, followed by RebuildState so the new
	// colourings are visible.
	out1 := colourRunAndRebuild(t, ctx, repo, bufOut, bufErr)
	require.Regexp(t, `Coloured [1-9]\d* packfiles`, out1)
	colouredFirst := colouredPackfiles(t, repo)
	require.NotEmpty(t, colouredFirst)

	// Second run, still inside the 7d grace window, must NOT re-colour
	// anything and must NOT delete anything (deletionTime is in the future
	// relative to cutoff).
	status, err, out2, errOut2 := runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Contains(t, out2, "Coloured 0 packfiles (0 orphaned)")
	require.Contains(t, out2, "0 blobs and 0 packfiles were removed")
	require.Empty(t, errOut2)

	// And the still-coloured set must be the same as before.
	require.NoError(t, repo.RebuildState())
	require.Equal(t, len(colouredFirst), len(colouredPackfiles(t, repo)))
}

// --- Sweep pass safety ---------------------------------------------------

func TestSweepRespectsGracePeriod(t *testing.T) {
	// Coloured packfiles within grace must remain on disk.
	repo, ctx, bufOut, bufErr := freshRepo(t)
	snap1 := ptesting.GenerateSnapshot(t, repo, simpleFiles(), ptesting.WithName("snap1"))
	ptesting.GenerateSnapshot(t, repo, extraFiles("p"), ptesting.WithName("snap2"))
	storeBefore := storePackfiles(t, repo)
	primeAndDelete(t, ctx, repo, bufOut, bufErr, snap1.Header.GetIndexID())

	// Colour the packfiles, then rebuild so coloured set is visible.
	colourRunAndRebuild(t, ctx, repo, bufOut, bufErr)
	require.NotEmpty(t, colouredPackfiles(t, repo))

	// Now run another sweep WITHIN the default 7d grace window.
	status, err, out, _ := runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Contains(t, out, "0 blobs and 0 packfiles were removed")
	require.Equal(t, storeBefore, storePackfiles(t, repo),
		"SAFETY: maintenance must NOT delete any packfile while it's still inside the grace window")
}

func TestSweepDeletesAfterGracePeriod(t *testing.T) {
	// Full safety lifecycle: a packfile is coloured, the grace period is
	// then shrunk, the next maintenance run sweeps it from disk.
	repo, ctx, bufOut, bufErr := freshRepo(t)
	snap1 := ptesting.GenerateSnapshot(t, repo, simpleFiles(), ptesting.WithName("snap1"))
	ptesting.GenerateSnapshot(t, repo, extraFiles("survivor"), ptesting.WithName("snap2"))
	storeBefore := storePackfiles(t, repo)
	primeAndDelete(t, ctx, repo, bufOut, bufErr, snap1.Header.GetIndexID())

	colourRunAndRebuild(t, ctx, repo, bufOut, bufErr)
	coloured := colouredPackfiles(t, repo)
	require.NotEmpty(t, coloured)

	// Now shrink the grace and re-run.
	t.Setenv("PLAKAR_GRACEPERIOD", "1ns")
	status, err, out, _ := runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Regexp(t, `[1-9]\d* packfiles were removed`, out)

	storeAfter := storePackfiles(t, repo)
	require.Less(t, len(storeAfter), len(storeBefore))
	// Verify the coloured packfiles are actually gone from disk.
	for mac := range coloured {
		_, ok := storeAfter[mac]
		require.False(t, ok, "coloured packfile %x should have been deleted", mac)
	}
}

func TestSweepNoDeletionEnv(t *testing.T) {
	// PLAKAR_NODELETION=true must keep packfiles on disk even though the
	// state is updated.
	repo, ctx, bufOut, bufErr := freshRepo(t)
	snap1 := ptesting.GenerateSnapshot(t, repo, simpleFiles(), ptesting.WithName("snap1"))
	ptesting.GenerateSnapshot(t, repo, extraFiles("keepme"), ptesting.WithName("snap2"))
	storeBefore := storePackfiles(t, repo)
	primeAndDelete(t, ctx, repo, bufOut, bufErr, snap1.Header.GetIndexID())

	// First, colour.
	colourRunAndRebuild(t, ctx, repo, bufOut, bufErr)
	coloured := colouredPackfiles(t, repo)
	require.NotEmpty(t, coloured)

	// Now sweep with NODELETION and a tiny grace.
	t.Setenv("PLAKAR_GRACEPERIOD", "1ns")
	t.Setenv("PLAKAR_NODELETION", "true")
	status, err, out, _ := runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Regexp(t, `[1-9]\d* packfiles were removed`, out)

	// SAFETY: the physical files must still be on disk despite the state
	// claiming they were removed.
	storeAfter := storePackfiles(t, repo)
	require.Equal(t, len(storeBefore), len(storeAfter), "NODELETION must preserve all packfiles on disk")
	for mac := range coloured {
		_, ok := storeAfter[mac]
		require.True(t, ok, "NODELETION: coloured packfile %x should still be on disk", mac)
		// And it must be readable.
		body, err := rawPackfileContents(t, repo, mac)
		require.NoError(t, err)
		require.NotEmpty(t, body)
	}
}

func TestSweepUncoloursReusedPackfile(t *testing.T) {
	// Simulate a concurrent backup that reuses a coloured packfile by
	// pre-populating the maintenance cache. The sweep must detect this and
	// uncolour rather than delete.
	repo, ctx, bufOut, bufErr := freshRepo(t)
	snap1 := ptesting.GenerateSnapshot(t, repo, simpleFiles(), ptesting.WithName("snap1"))
	ptesting.GenerateSnapshot(t, repo, extraFiles("k"), ptesting.WithName("snap2"))
	storeBefore := storePackfiles(t, repo)
	primeAndDelete(t, ctx, repo, bufOut, bufErr, snap1.Header.GetIndexID())

	// Colour first.
	colourRunAndRebuild(t, ctx, repo, bufOut, bufErr)
	coloured := colouredPackfiles(t, repo)
	require.NotEmpty(t, coloured)

	// Pick one coloured packfile and re-link it via the maintenance cache,
	// pretending a concurrent backup just used it.
	var revived objects.MAC
	for m := range coloured {
		revived = m
		break
	}
	mcache, err := repo.AppContext().GetCache().Maintenance(repo.Configuration().RepositoryID)
	require.NoError(t, err)
	require.NoError(t, mcache.PutPackfile(objects.RandomMAC(), revived))

	// Run sweep with an expired grace so the coloured packfile would
	// otherwise be deleted.
	t.Setenv("PLAKAR_GRACEPERIOD", "1ns")
	status, err, out, errOut := runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	require.Contains(t, errOut, "Concurrent backup used", "should warn about the uncoloured packfile")
	// The revived packfile must remain on disk.
	storeAfter := storePackfiles(t, repo)
	_, stillThere := storeAfter[revived]
	require.True(t, stillThere, "SAFETY: a re-used packfile must NEVER be deleted by maintenance")
	require.LessOrEqual(t, len(storeBefore)-len(storeAfter), len(coloured)-1)

	// The "uncolour" state change is written to the delta state but only
	// committed when at least one packfile is actually deleted (see
	// sweepPass: `if len(toDelete) > 0 { CommitTransaction }`). When ALL
	// coloured packfiles are revived (as in this test) the uncolouring is
	// lost — but that does NOT compromise safety: the cache.HasPackfile
	// check inside sweepPass still prevents the next run from deleting it.
	// We pin the safety property (packfile still on disk) here; we don't
	// pin the uncolour observability since it depends on the toDelete>0
	// shortcut in sweepPass.
	_ = out
}

func TestSweepUncolourPersistsWhenSomeDeletionsOccur(t *testing.T) {
	// When at least one packfile IS deleted in the same sweep, the
	// CommitTransaction call DOES go through, and any other UncolourPackfile
	// calls in the same pass are persisted. We arrange exactly that: two
	// candidate coloured packfiles, only one of which is revived via the
	// cache.
	repo, ctx, bufOut, bufErr := freshRepo(t)
	// Use two snapshots with distinctive contents so the colouring picks
	// up >=2 packfiles when both are deleted.
	snap1 := ptesting.GenerateSnapshot(t, repo, extraFiles("doomed1"), ptesting.WithName("snap1"))
	snap2 := ptesting.GenerateSnapshot(t, repo, extraFiles("doomed2"), ptesting.WithName("snap2"))
	survivor := ptesting.GenerateSnapshot(t, repo, extraFiles("survivor"), ptesting.WithName("snap3"))

	// Prime the cache with all three.
	status, err, _, _ := runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// Delete two snapshots, keep survivor.
	require.NoError(t, repo.DeleteSnapshot(snap1.Header.GetIndexID()))
	require.NoError(t, repo.DeleteSnapshot(snap2.Header.GetIndexID()))
	require.NoError(t, repo.RebuildState())

	colourRunAndRebuild(t, ctx, repo, bufOut, bufErr)
	coloured := colouredPackfiles(t, repo)
	if len(coloured) < 2 {
		t.Skipf("need at least 2 coloured packfiles for this test; got %d", len(coloured))
	}

	// Revive ONE coloured packfile by faking a concurrent backup.
	mcache, err := repo.AppContext().GetCache().Maintenance(repo.Configuration().RepositoryID)
	require.NoError(t, err)
	var revived objects.MAC
	for m := range coloured {
		revived = m
		break
	}
	require.NoError(t, mcache.PutPackfile(objects.RandomMAC(), revived))

	t.Setenv("PLAKAR_GRACEPERIOD", "1ns")
	status, err, _, errOut := runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Contains(t, errOut, "Concurrent backup used")

	// The revived packfile must still be on disk.
	storeAfter := storePackfiles(t, repo)
	_, stillThere := storeAfter[revived]
	require.True(t, stillThere, "SAFETY: revived packfile must not be deleted")

	// Because toDelete>0 (the other coloured packfile was deleted), the
	// uncolour transaction WAS committed; after RebuildState the revived
	// packfile must no longer appear in the coloured set.
	require.NoError(t, repo.RebuildState())
	stillColoured := colouredPackfiles(t, repo)
	_, stillFlagged := stillColoured[revived]
	require.False(t, stillFlagged, "revived packfile should be uncoloured when at least one deletion forced a commit")

	_ = survivor
}

// --- Grace period env parsing --------------------------------------------

// graceHumanFormatCase prepares a repo with snap1+snap2, primes the cache,
// deletes snap1, then runs maintenance with the given GRACEPERIOD env value
// and asserts the human-readable duration line.
func graceHumanFormatCase(t *testing.T, gracePeriod, expected string) {
	t.Helper()
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	resetEnv(t)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap1 := ptesting.GenerateSnapshot(t, repo, simpleFiles(), ptesting.WithName("snap1"))
	ptesting.GenerateSnapshot(t, repo, extraFiles("p"), ptesting.WithName("snap2"))
	primeAndDelete(t, ctx, repo, bufOut, bufErr, snap1.Header.GetIndexID())

	t.Setenv("PLAKAR_GRACEPERIOD", gracePeriod)
	_, err, out, _ := runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)
	require.Contains(t, out, "scheduled to be removed in "+expected,
		"GRACEPERIOD=%q should format as %q", gracePeriod, expected)
}

func TestGracePeriodInvalidStringFallsBackToDefault(t *testing.T) {
	graceHumanFormatCase(t, "not a duration", "7d")
}

func TestGracePeriod48hHumanFormat(t *testing.T) {
	graceHumanFormatCase(t, "48h", "2d")
}

func TestGracePeriod50hHumanFormat(t *testing.T) {
	graceHumanFormatCase(t, "50h", "2d2h0m0s")
}

func TestGracePeriodSubDayHumanFormat(t *testing.T) {
	graceHumanFormatCase(t, "1h", "1h0m0s")
}

// --- Locking --------------------------------------------------------------

func TestExecuteLocklessMode(t *testing.T) {
	// PLAKAR_LOCKLESS=true skips the on-disk lock entirely. We can prove it
	// by pre-installing a hostile lock that *would* otherwise block us.
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	ptesting.GenerateSnapshot(t, repo, simpleFiles())

	preInstallExclusiveLock(t, repo, "someone-else", time.Now())
	t.Setenv("PLAKAR_LOCKLESS", "true")

	status, err, out, _ := runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Contains(t, out, "Coloured 0 packfiles")
}

func TestExecuteConflictingLockAborts(t *testing.T) {
	resetEnv(t)
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	ptesting.GenerateSnapshot(t, repo, simpleFiles())

	storeBefore := storePackfiles(t, repo)
	stranger := preInstallExclusiveLock(t, repo, "another-host", time.Now())

	status, err, _, _ := runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.Error(t, err)
	require.Contains(t, err.Error(), "can't take exclusive lock")
	require.Equal(t, 1, status)

	// The stranger's lock must remain.
	requireLockExists(t, repo, stranger)
	// And maintenance must NOT have touched any packfile.
	require.Equal(t, storeBefore, storePackfiles(t, repo),
		"SAFETY: maintenance must not modify the store if it could not acquire the lock")
}

func TestExecuteStaleLock(t *testing.T) {
	// If we have a stale lock it's expected we remove it and just proceed as
	// usual.
	resetEnv(t)
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	ptesting.GenerateSnapshot(t, repo, simpleFiles())

	// Place a stale exclusive lock (TTL is 2 * LOCK_REFRESH_RATE = 10m).
	stale := preInstallExclusiveLock(t, repo, "ghost-host", time.Now().Add(-30*time.Minute))

	status, err, _, _ := runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// Stale lock got cleared by the kick-out path.
	requireLockGone(t, repo, stale)
}

func TestExecuteLockReleasedAfterRun(t *testing.T) {
	resetEnv(t)
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	ptesting.GenerateSnapshot(t, repo, simpleFiles())

	beforeLocks, err := repo.GetLocks()
	require.NoError(t, err)
	require.Empty(t, beforeLocks)

	status, err, _, _ := runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// Unlock is asynchronous: Unlock() only closes the ping channel and the
	// background goroutine then performs DeleteLock. Poll briefly until the
	// lock disappears (or fail after ~1s).
	require.Eventually(t, func() bool {
		locks, err := repo.GetLocks()
		require.NoError(t, err)
		return len(locks) == 0
	}, time.Second, 10*time.Millisecond, "maintenance must remove its own lock on exit")
}

// --- Deleted snapshot cache cleanup ---------------------------------------

func TestUpdateCacheDeletesRemovedSnapshots(t *testing.T) {
	// First run populates the maintenance cache, second run after deletion
	// must remove the dead snapshot's entry. We verify it indirectly: a
	// third run after RebuildState colours those packfiles successfully,
	// proving the cache no longer references them.
	repo, ctx, bufOut, bufErr := freshRepo(t)
	snap1 := ptesting.GenerateSnapshot(t, repo, simpleFiles(), ptesting.WithName("snap1"))
	ptesting.GenerateSnapshot(t, repo, extraFiles("alive"), ptesting.WithName("snap2"))

	// Pre-warm the cache.
	status, err, _, _ := runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// Delete snap1 (DeleteSnapshot alone, no RebuildState yet → snap1
	// surfaces through ListDeletedSnapShots).
	require.NoError(t, repo.DeleteSnapshot(snap1.Header.GetIndexID()))

	// Second run: updateCache should walk ListDeletedSnapShots and prune
	// snap1's entry from the maintenance cache.
	status, err, _, _ = runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// Now actually rebuild and run a colour pass.
	require.NoError(t, repo.RebuildState())
	status, err, out, _ := runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Regexp(t, `Coloured [1-9]\d* packfiles`, out,
		"packfiles from deleted snap1 must be coloured for deletion after rebuild")
}

// --- Output safety --------------------------------------------------------

// --- Orphan packfile (storage has it, state doesn't) -----------------------

// injectOrphanPackfile copies an existing packfile's raw bytes to a freshly
// chosen MAC in the store, so the result is a packfile present in the
// storage but absent from the state index — exactly the situation maintenance
// must detect via the GetPackfiles vs ListPackfiles diff.
func injectOrphanPackfile(t *testing.T, repo *repository.Repository) objects.MAC {
	t.Helper()
	// pick any existing packfile to copy
	var source objects.MAC
	for m := range repo.ListPackfiles() {
		source = m
		break
	}
	if source == (objects.MAC{}) {
		t.Fatalf("need at least one existing packfile to clone an orphan from")
	}
	rd, err := repo.Store().Get(repo.AppContext(), storage.StorageResourcePackfile, source, nil)
	require.NoError(t, err)
	defer rd.Close()
	raw, err := io.ReadAll(rd)
	require.NoError(t, err)

	orphan := objects.RandomMAC()
	_, err = repo.Store().Put(repo.AppContext(), storage.StorageResourcePackfile, orphan, bytes.NewReader(raw))
	require.NoError(t, err)
	return orphan
}

func TestColourPassDetectsStorageOnlyOrphan(t *testing.T) {
	// Default 7d grace: the orphan packfile we inject borrows the original
	// packfile's recent footer timestamp, so it sits comfortably inside the
	// grace window and must NOT be coloured.
	repo, ctx, bufOut, bufErr := freshRepo(t)
	ptesting.GenerateSnapshot(t, repo, simpleFiles())
	orphan := injectOrphanPackfile(t, repo)
	storeBefore := storePackfiles(t, repo)
	require.Contains(t, storeBefore, orphan)

	status, err, out, _ := runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Contains(t, out, "Coloured 0 packfiles (0 orphaned)",
		"recent orphan inside grace must not be reported")
	require.Equal(t, storeBefore, storePackfiles(t, repo),
		"SAFETY: orphan inside grace must not be deleted")
}

func TestColourPassDetectsStorageOnlyOrphanOutsideGrace(t *testing.T) {
	// Same injection, but with grace=1ns the orphan is reported and the
	// full colour+sweep cycle removes it.
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	resetEnv(t)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	ptesting.GenerateSnapshot(t, repo, simpleFiles())
	orphan := injectOrphanPackfile(t, repo)

	t.Setenv("PLAKAR_GRACEPERIOD", "1ns")
	status, err, out, _ := runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Regexp(t, `Coloured \d+ packfiles \([1-9]\d* orphaned\)`, out)

	// After the rebuild, the orphan should be gone from storage too. But
	// note: in the same Execute the sweep cannot delete what was just
	// coloured (CommitTransaction does not merge into the in-memory
	// state), so we run maintenance a second time after rebuild to
	// actually sweep.
	require.NoError(t, repo.RebuildState())
	_, err, _, _ = runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)

	storeAfter := storePackfiles(t, repo)
	_, stillThere := storeAfter[orphan]
	require.False(t, stillThere, "orphan past grace must be removed from storage")
}

// --- Output stays sane ----------------------------------------------------

func TestNoStrayMessagesWhenNothingToDo(t *testing.T) {
	repo, ctx, bufOut, bufErr := freshRepo(t)
	ptesting.GenerateSnapshot(t, repo, simpleFiles())

	status, err, out, errOut := runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.NotContains(t, out, "scheduled to be removed in",
		"the 'scheduled to be removed' line must only appear when coloring is non-zero")
	require.NotContains(t, errOut, "Concurrent backup",
		"no concurrent-backup warning when nothing was coloured")
}
