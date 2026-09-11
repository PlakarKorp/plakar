package maintenance

import (
	"bytes"
	"testing"

	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// --- Parse ----------------------------------------------------------------

// TestCov3ParseNilCtxSecret confirms Parse copies a nil secret without error
// (unencrypted repo path) and leaves RepositorySecret nil.
func TestCov3ParseNilSecret(t *testing.T) {
	resetEnv(t)
	_, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	cmd := &Maintenance{}
	require.NoError(t, cmd.Parse(ctx, []string{}))
	require.Nil(t, cmd.RepositorySecret)
}

// --- Grace-period human formatting boundary -------------------------------

// TestCov3GracePeriodExactly24h pins the boundary where duration.Hours() is not
// strictly greater than 24, so the sub-day formatting branch is taken and the
// duration is printed verbatim rather than as "1d".
func TestCov3GracePeriodExactly24h(t *testing.T) {
	graceHumanFormatCase(t, "24h", "24h0m0s")
}

// TestCov3GracePeriodMultiDayNoRemainder pins the whole-days branch with no
// leftover (72h => "3d", not "3d0s").
func TestCov3GracePeriodMultiDayNoRemainder(t *testing.T) {
	graceHumanFormatCase(t, "72h", "3d")
}

// --- Colour + sweep over multiple deleted snapshots -----------------------

// TestCov3SweepDeletesMultiplePackfiles drives the sweep deletion loop over
// more than one packfile in a single pass, exercising the toDelete iteration
// and the DeletePackfile call for several entries.
func TestCov3SweepDeletesMultiplePackfiles(t *testing.T) {
	repo, ctx, bufOut, bufErr := freshRepo(t)
	snap1 := ptesting.GenerateSnapshot(t, repo, extraFiles("gone-a"), ptesting.WithName("s1"))
	snap2 := ptesting.GenerateSnapshot(t, repo, extraFiles("gone-b"), ptesting.WithName("s2"))
	ptesting.GenerateSnapshot(t, repo, extraFiles("stays"), ptesting.WithName("s3"))

	storeBefore := storePackfiles(t, repo)

	// Prime the maintenance cache with all three snapshots.
	status, err, _, _ := runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// Delete two snapshots and make the deletion visible to the state.
	require.NoError(t, repo.DeleteSnapshot(snap1.Header.GetIndexID()))
	require.NoError(t, repo.DeleteSnapshot(snap2.Header.GetIndexID()))
	require.NoError(t, repo.RebuildState())

	// Colour the now-unreferenced packfiles, then make the colourings visible.
	out := colourRunAndRebuild(t, ctx, repo, bufOut, bufErr)
	require.Regexp(t, `Coloured [1-9]\d* packfiles`, out)
	coloured := colouredPackfiles(t, repo)
	require.NotEmpty(t, coloured)

	// Shrink the grace and sweep: every coloured packfile must be deleted.
	t.Setenv("PLAKAR_GRACEPERIOD", "1ns")
	status, err, out, _ = runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Regexp(t, `[1-9]\d* packfiles were removed`, out)

	storeAfter := storePackfiles(t, repo)
	require.Less(t, len(storeAfter), len(storeBefore))
	for mac := range coloured {
		_, ok := storeAfter[mac]
		require.False(t, ok, "coloured packfile %x must have been swept", mac)
	}
}

// --- Orphan past grace gets swept in the second Execute -------------------

// TestCov3OrphanColourThenSweep injects a storage-only orphan with grace=1ns so
// it is coloured on the first Execute, then (after a rebuild) deleted on the
// second Execute. This drives the orphan-detection branch of colourPass plus
// the sweep deletion of an orphan that was never referenced by any snapshot.
func TestCov3OrphanColourThenSweep(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	resetEnv(t)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	ptesting.GenerateSnapshot(t, repo, simpleFiles())
	orphan := injectOrphanPackfile(t, repo)

	t.Setenv("PLAKAR_GRACEPERIOD", "1ns")

	// First Execute colours the orphan (it sits outside the 1ns grace window).
	status, err, out, _ := runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Regexp(t, `Coloured \d+ packfiles \([1-9]\d* orphaned\)`, out)

	// Make the colouring visible, then sweep on the second Execute.
	require.NoError(t, repo.RebuildState())
	status, err, _, _ = runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	_, stillThere := storePackfiles(t, repo)[orphan]
	require.False(t, stillThere, "orphan past grace must be swept from storage")
}

// --- Coloured-but-not-yet-due packfile is skipped by sweep ----------------

// TestCov3SweepSkipsFutureDeletion colours a packfile, then runs a sweep still
// inside the grace window. The sweep's `deletionTime.After(cutoff)` guard must
// keep the packfile, leaving the coloured set intact.
func TestCov3SweepSkipsFutureDeletion(t *testing.T) {
	repo, ctx, bufOut, bufErr := freshRepo(t)
	snap1 := ptesting.GenerateSnapshot(t, repo, simpleFiles(), ptesting.WithName("s1"))
	ptesting.GenerateSnapshot(t, repo, extraFiles("keep"), ptesting.WithName("s2"))
	primeAndDelete(t, ctx, repo, bufOut, bufErr, snap1.Header.GetIndexID())

	colourRunAndRebuild(t, ctx, repo, bufOut, bufErr)
	colouredBefore := colouredPackfiles(t, repo)
	require.NotEmpty(t, colouredBefore)

	// A sweep within the default 7d grace must not remove anything.
	status, err, out, errOut := runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Contains(t, out, "0 blobs and 0 packfiles were removed")
	require.Empty(t, errOut)

	require.NoError(t, repo.RebuildState())
	require.Equal(t, len(colouredBefore), len(colouredPackfiles(t, repo)),
		"coloured-but-not-due packfiles must stay coloured after an in-grace sweep")
}
