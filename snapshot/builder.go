package snapshot

import (
	"bytes"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/caching"
	"github.com/PlakarKorp/plakar/events"
	"github.com/PlakarKorp/plakar/logging"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/snapshot/header"
	"github.com/google/uuid"
)

type Builder struct {
	repository *repository.RepositoryWriter

	//This is protecting the above two pointers, not their underlying objects
	deltaMtx   sync.RWMutex
	scanCache  *caching.ScanCache
	deltaCache *caching.ScanCache

	Header *header.Header
}

func Create(repo *repository.Repository) (*Builder, error) {
	identifier := objects.RandomMAC()
	scanCache, err := repo.AppContext().GetCache().Scan(identifier)
	if err != nil {
		return nil, err
	}

	snap := &Builder{
		scanCache:  scanCache,
		deltaCache: scanCache,

		Header: header.NewHeader("default", identifier),
	}
	snap.repository = repo.NewRepositoryWriter(scanCache, snap.Header.Identifier)

	if snap.AppContext().Identity != uuid.Nil {
		snap.Header.Identity.Identifier = snap.AppContext().Identity
		snap.Header.Identity.PublicKey = snap.AppContext().Keypair.PublicKey
	}
	snap.Header.SetContext("Hostname", snap.AppContext().Hostname)
	snap.Header.SetContext("Username", snap.AppContext().Username)
	snap.Header.SetContext("OperatingSystem", snap.AppContext().OperatingSystem)
	snap.Header.SetContext("MachineID", snap.AppContext().MachineID)
	snap.Header.SetContext("CommandLine", snap.AppContext().CommandLine)
	snap.Header.SetContext("ProcessID", fmt.Sprintf("%d", snap.AppContext().ProcessID))
	snap.Header.SetContext("Architecture", snap.AppContext().Architecture)
	snap.Header.SetContext("NumCPU", fmt.Sprintf("%d", runtime.NumCPU()))
	snap.Header.SetContext("MaxProcs", fmt.Sprintf("%d", runtime.GOMAXPROCS(0)))
	snap.Header.SetContext("Client", snap.AppContext().Client)

	repo.Logger().Trace("snapshot", "%x: Create()", snap.Header.GetIndexShortID())
	return snap, nil
}

func (snap *Builder) Repository() *repository.Repository {
	return snap.repository.Repository
}

func (snap *Builder) Close() error {
	snap.Logger().Trace("snapshotBuilder", "%x: Close(): %x", snap.Header.Identifier, snap.Header.GetIndexShortID())

	if snap.scanCache != nil {
		return snap.scanCache.Close()
	}

	return nil
}

func (snap *Builder) Logger() *logging.Logger {
	return snap.AppContext().GetLogger()
}

func (snap *Builder) AppContext() *appcontext.AppContext {
	return snap.repository.AppContext()
}

func (snap *Builder) Event(evt events.Event) {
	snap.AppContext().Events().Send(evt)
}

func (snap *Builder) Lock() (chan bool, error) {
	lockless, _ := strconv.ParseBool(os.Getenv("PLAKAR_LOCKLESS"))
	lockDone := make(chan bool)
	if lockless {
		return lockDone, nil
	}

	lock := repository.NewSharedLock(snap.AppContext().Hostname)

	buffer := &bytes.Buffer{}
	err := lock.SerializeToStream(buffer)
	if err != nil {
		return nil, err
	}

	_, err = snap.repository.PutLock(snap.Header.Identifier, buffer)
	if err != nil {
		return nil, err
	}

	// We installed the lock, now let's see if there is a conflicting exclusive lock or not.
	locksID, err := snap.repository.GetLocks()
	if err != nil {
		// We still need to delete it, and we need to do so manually.
		snap.repository.DeleteLock(snap.Header.Identifier)
		return nil, err
	}

	for _, lockID := range locksID {
		version, rd, err := snap.repository.GetLock(lockID)
		if err != nil {
			snap.repository.DeleteLock(snap.Header.Identifier)
			return nil, err
		}

		lock, err := repository.NewLockFromStream(version, rd)
		if err != nil {
			snap.repository.DeleteLock(snap.Header.Identifier)
			return nil, err
		}

		/* Kick out stale locks */
		if lock.IsStale() {
			err := snap.repository.DeleteLock(lockID)
			if err != nil {
				snap.repository.DeleteLock(snap.Header.Identifier)
				return nil, err
			}
		}

		// There is an exclusive lock in place, we need to abort.
		if lock.Exclusive {
			err := snap.repository.DeleteLock(snap.Header.Identifier)
			if err != nil {
				return nil, err
			}

			return nil, fmt.Errorf("Can't take repository lock, it's already locked by maintenance.")
		}
	}

	// The following bit is a "ping" mechanism, Lock() is a bit badly named at this point,
	// we are just refreshing the existing lock so that the watchdog doesn't removes us.
	go func() {
		for {
			select {
			case <-lockDone:
				snap.repository.DeleteLock(snap.Header.Identifier)
				return
			case <-time.After(repository.LOCK_REFRESH_RATE):
				lock := repository.NewSharedLock(snap.AppContext().Hostname)

				buffer := &bytes.Buffer{}

				// We ignore errors here on purpose, it's tough to handle them
				// correctly, and if they happen we will be ripped by the
				// watchdog anyway.
				lock.SerializeToStream(buffer)
				snap.repository.PutLock(snap.Header.Identifier, buffer)
			}
		}
	}()

	return lockDone, nil
}

func (snap *Builder) Unlock(ping chan bool) {
	close(ping)
}
