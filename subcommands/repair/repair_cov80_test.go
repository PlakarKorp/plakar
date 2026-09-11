package repair

import (
	"bytes"
	"testing"
	"time"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// installExclusiveLock writes an exclusive lock with the given timestamp under a
// random MAC on behalf of another host, so repair's Lock() acquisition observes
// it. Returns the lock id.
func installExclusiveLock(t *testing.T, repo *repository.Repository, hostname string, when time.Time) objects.MAC {
	t.Helper()
	lockID := objects.RandomMAC()
	lock := repository.NewExclusiveLock(hostname)
	lock.Timestamp = when

	buf := &bytes.Buffer{}
	require.NoError(t, lock.SerializeToStream(buf))
	_, err := repo.PutLock(lockID, buf)
	require.NoError(t, err)
	return lockID
}

func lockExists(t *testing.T, repo *repository.Repository, id objects.MAC) bool {
	t.Helper()
	locks, err := repo.GetLocks()
	require.NoError(t, err)
	for _, l := range locks {
		if l == id {
			return true
		}
	}
	return false
}

// TestCov80RepairApplyConflictingLockAborts drives the conflicting-lock branch
// of Lock(): a non-stale exclusive lock from another host is present, so
// repair -apply must fail to acquire the lock and return status 1. This is the
// largest uncovered branch of Lock().
func TestCov80RepairApplyConflictingLockAborts(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("d"),
		ptesting.NewMockFile("d/f.txt", 0644, "payload"),
	})
	defer snap.Close()

	stranger := installExclusiveLock(t, repo, "another-host", time.Now())

	cmd := &Repair{}
	require.NoError(t, cmd.Parse(ctx, []string{"-apply"}))
	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
	require.Contains(t, err.Error(), "can't take exclusive lock")

	// The stranger's lock must remain untouched.
	require.True(t, lockExists(t, repo, stranger), "conflicting lock must be preserved")
}

// TestCov80RepairApplyStaleLockKickedOut drives the stale-lock kick-out branch
// of Lock(): a stale exclusive lock (older than the lock TTL) is removed and
// repair -apply proceeds to completion with status 0.
func TestCov80RepairApplyStaleLockKickedOut(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("d"),
		ptesting.NewMockFile("d/f.txt", 0644, "payload"),
	})
	defer snap.Close()

	// TTL is 2 * LOCK_REFRESH_RATE (well under 30m), so this is stale.
	stale := installExclusiveLock(t, repo, "ghost-host", time.Now().Add(-30*time.Minute))

	cmd := &Repair{}
	require.NoError(t, cmd.Parse(ctx, []string{"-apply"}))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// The stale lock must have been kicked out.
	require.False(t, lockExists(t, repo, stale), "stale lock must be removed by the kick-out path")
}

// TestCov80RepairApplyReleasesOwnLock confirms repair -apply removes its own
// lock after a successful run (the Unlock path).
func TestCov80RepairApplyReleasesOwnLock(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("d"),
		ptesting.NewMockFile("d/f.txt", 0644, "payload"),
	})
	defer snap.Close()

	before, err := repo.GetLocks()
	require.NoError(t, err)
	require.Empty(t, before)

	cmd := &Repair{}
	require.NoError(t, cmd.Parse(ctx, []string{"-apply"}))
	status, err := cmd.Execute(ctx, repo)
	require.NoError(t, err)
	require.Equal(t, 0, status)

	// Unlock is asynchronous (the ping goroutine performs DeleteLock), so poll.
	require.Eventually(t, func() bool {
		locks, lerr := repo.GetLocks()
		require.NoError(t, lerr)
		return len(locks) == 0
	}, 2*time.Second, 10*time.Millisecond, "repair must release its own lock on exit")
}
