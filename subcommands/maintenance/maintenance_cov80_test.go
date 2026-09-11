package maintenance

import (
	"bytes"
	"testing"
	"time"

	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// TestCov80MaintenanceLocklessSweepDeletes runs a full colour+sweep deletion
// cycle in PLAKAR_LOCKLESS mode. The existing lockless test only covers a no-op
// run; this drives the lockless branch of Lock() together with a sweep that
// actually removes packfiles, so the lockless path is exercised alongside real
// deletion work.
func TestCov80MaintenanceLocklessSweepDeletes(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	resetEnv(t)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap1 := ptesting.GenerateSnapshot(t, repo, simpleFiles(), ptesting.WithName("snap1"))
	ptesting.GenerateSnapshot(t, repo, extraFiles("keep"), ptesting.WithName("snap2"))

	storeBefore := storePackfiles(t, repo)
	primeAndDelete(t, ctx, repo, bufOut, bufErr, snap1.Header.GetIndexID())

	// Colour the now-unreferenced packfiles in lockless mode, then rebuild.
	t.Setenv("PLAKAR_LOCKLESS", "true")
	colourRunAndRebuild(t, ctx, repo, bufOut, bufErr)
	coloured := colouredPackfiles(t, repo)
	require.NotEmpty(t, coloured)

	// Sweep with an expired grace; lockless deletion must still work.
	t.Setenv("PLAKAR_GRACEPERIOD", "1ns")
	status, err, out, _ := runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Regexp(t, `[1-9]\d* packfiles were removed`, out)

	storeAfter := storePackfiles(t, repo)
	require.Less(t, len(storeAfter), len(storeBefore))
	for mac := range coloured {
		_, ok := storeAfter[mac]
		require.False(t, ok, "lockless sweep must delete coloured packfile %x", mac)
	}
}

// TestCov80MaintenanceLocklessLeavesNoLock confirms that in lockless mode no
// lock is ever installed in the store (the early-return branch of Lock()).
func TestCov80MaintenanceLocklessLeavesNoLock(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	resetEnv(t)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	ptesting.GenerateSnapshot(t, repo, simpleFiles())

	t.Setenv("PLAKAR_LOCKLESS", "true")

	cmd := &Maintenance{}
	require.NoError(t, cmd.Parse(ctx, nil))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// Lockless mode must not have written any lock.
	locks, err := repo.GetLocks()
	require.NoError(t, err)
	require.Empty(t, locks, "lockless maintenance must not install a lock")
}

// TestCov80MaintenanceStaleLockThenWork removes a stale conflicting lock and
// then proceeds to do real colouring work in the same run. The existing stale
// lock test runs against a repo with nothing to colour; here the run also
// colours packfiles, exercising the stale-lock kick-out path followed by a
// productive colour pass.
func TestCov80MaintenanceStaleLockThenWork(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	resetEnv(t)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap1 := ptesting.GenerateSnapshot(t, repo, simpleFiles(), ptesting.WithName("snap1"))
	ptesting.GenerateSnapshot(t, repo, extraFiles("keep"), ptesting.WithName("snap2"))

	primeAndDelete(t, ctx, repo, bufOut, bufErr, snap1.Header.GetIndexID())

	// Install a stale lock so the kick-out path runs, then a colour pass.
	stale := preInstallExclusiveLock(t, repo, "ghost", time.Now().Add(-30*time.Minute))

	status, err, out, _ := runMaintenance(t, ctx, repo, bufOut, bufErr)
	require.NoError(t, err)
	require.Equal(t, 0, status)
	require.Regexp(t, `Coloured [1-9]\d* packfiles`, out)

	requireLockGone(t, repo, stale)
}
