package snapshot

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/btree"
	"github.com/PlakarKorp/plakar/caching"
	"github.com/PlakarKorp/plakar/events"
	"github.com/PlakarKorp/plakar/logging"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/snapshot/header"
	"github.com/PlakarKorp/plakar/snapshot/vfs"
	"github.com/google/uuid"
)

var (
	ErrNotFound = errors.New("snapshot not found")
)

type Snapshot struct {
	repository *repository.Repository
	scanCache  *caching.ScanCache
	checkCache *caching.CheckCache

	deltaCache *caching.ScanCache
	//This is protecting the above two pointers, not their underlying objects
	deltaMtx sync.RWMutex

	filesystem *vfs.Filesystem

	SkipDirs []string

	Header *header.Header
}

func LogicalSize(repo *repository.Repository) (int, int64, error) {
	nSnapshots := 0
	totalSize := int64(0)
	for snapshotID := range repo.ListSnapshots() {
		snap, err := Load(repo, snapshotID)
		if err != nil {
			continue
		}
		nSnapshots++
		totalSize += int64(snap.Header.GetSource(0).Summary.Directory.Size + snap.Header.GetSource(0).Summary.Below.Size)
		snap.Close()
	}

	return nSnapshots, totalSize, nil
}

func Create(repo *repository.Repository) (*Snapshot, error) {
	identifier := objects.RandomMAC()
	scanCache, err := repo.AppContext().GetCache().Scan(identifier)
	if err != nil {
		return nil, err
	}

	snap := &Snapshot{
		repository: repo,
		scanCache:  scanCache,
		deltaCache: scanCache,

		Header: header.NewHeader("default", identifier),
	}

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

	repo.StartTransaction(scanCache)
	repo.StartPackerManager(snap.Header.Identifier)

	repo.Logger().Trace("snapshot", "%x: New()", snap.Header.GetIndexShortID())
	return snap, nil
}

func Load(repo *repository.Repository, Identifier objects.MAC) (*Snapshot, error) {
	hdr, _, err := GetSnapshot(repo, Identifier)
	if err != nil {
		return nil, err
	}

	snapshot := &Snapshot{}
	snapshot.repository = repo
	snapshot.Header = hdr

	repo.Logger().Trace("snapshot", "%x: Load()", snapshot.Header.GetIndexShortID())
	return snapshot, nil
}

func Clone(repo *repository.Repository, Identifier objects.MAC) (*Snapshot, error) {
	snap, err := Load(repo, Identifier)
	if err != nil {
		return nil, err
	}
	snap.Header.Timestamp = time.Now()

	uuidBytes, err := uuid.Must(uuid.NewRandom()).MarshalBinary()
	if err != nil {
		return nil, err
	}

	snap.Header.Identifier = repo.ComputeMAC(uuidBytes[:])
	repo.StartPackerManager(snap.Header.Identifier)

	repo.Logger().Trace("snapshot", "%x: Clone(): %s", snap.Header.Identifier, snap.Header.GetIndexShortID())
	return snap, nil
}

func Fork(repo *repository.Repository, Identifier objects.MAC) (*Snapshot, error) {
	identifier := objects.RandomMAC()
	snap, err := Clone(repo, Identifier)
	if err != nil {
		return nil, err
	}

	snap.Header.Identifier = identifier

	snap.Logger().Trace("snapshot", "%x: Fork(): %x", snap.Header.Identifier, snap.Header.GetIndexShortID())
	return snap, nil
}

func (snap *Snapshot) Close() error {
	snap.Logger().Trace("snapshot", "%x: Close(): %x", snap.Header.Identifier, snap.Header.GetIndexShortID())

	if snap.scanCache != nil {
		return snap.scanCache.Close()
	}

	return nil
}

func (snap *Snapshot) AppContext() *appcontext.AppContext {
	return snap.Repository().AppContext()
}

func (snap *Snapshot) Event(evt events.Event) {
	snap.AppContext().Events().Send(evt)
}

func GetSnapshot(repo *repository.Repository, Identifier objects.MAC) (*header.Header, bool, error) {
	repo.Logger().Trace("snapshot", "repository.GetSnapshot(%x)", Identifier)

	// Try to get snapshot from cache first
	cache, err := repo.AppContext().GetCache().Repository(repo.Configuration().RepositoryID)
	if err == nil {
		if snapshotBytes, err := cache.GetSnapshot(Identifier); err == nil {
			if snapshotBytes != nil {
				hdr, err := header.NewFromBytes(snapshotBytes)
				if err == nil {
					return hdr, true, nil
				}
			}
		}
	}

	rd, err := repo.GetBlob(resources.RT_SNAPSHOT, Identifier)
	if err != nil {
		if errors.Is(err, repository.ErrBlobNotFound) {
			err = ErrNotFound
		}
		return nil, false, err
	}

	buffer, err := io.ReadAll(rd)
	if err != nil {
		return nil, false, err
	}

	hdr, err := header.NewFromBytes(buffer)
	if err != nil {
		return nil, false, err
	}

	if cache != nil {
		_ = cache.PutSnapshot(Identifier, buffer) // optionally handle/log error
	}

	return hdr, false, nil
}

func (snap *Snapshot) Repository() *repository.Repository {
	return snap.repository
}

func (snap *Snapshot) LookupObject(mac objects.MAC) (*objects.Object, error) {
	buffer, err := snap.repository.GetBlobBytes(resources.RT_OBJECT, mac)
	if err != nil {
		return nil, err
	}
	return objects.NewObjectFromBytes(buffer)
}

func getPackfileForBlobWithError(snap *Snapshot, res resources.Type, mac objects.MAC) (objects.MAC, error) {
	packfile, exists, err := snap.repository.GetPackfileForBlob(res, mac)
	if err != nil {
		return objects.MAC{}, fmt.Errorf("Error %s while trying to locate packfile for blob %x of type %s", err, mac, res)
	} else if !exists {
		return objects.MAC{}, fmt.Errorf("Could not find packfile for blob %x of type %s", mac, res)
	} else {
		return packfile, nil
	}
}

func (snap *Snapshot) ListPackfiles() (iter.Seq2[objects.MAC, error], error) {
	pvfs, err := snap.Filesystem()
	if err != nil {
		return nil, err
	}

	return func(yield func(objects.MAC, error) bool) {
		if !yield(getPackfileForBlobWithError(snap, resources.RT_SNAPSHOT, snap.Header.Identifier)) {
			return
		}

		if snap.Header.Identity.Identifier != uuid.Nil {
			if !yield(getPackfileForBlobWithError(snap, resources.RT_SIGNATURE, snap.Header.Identifier)) {
				return
			}
		}

		if !yield(getPackfileForBlobWithError(snap, resources.RT_VFS_BTREE, snap.Header.Sources[0].VFS.Root)) {
			return
		}

		/* Iterate over all the VFS, resolving both Nodes and actual VFS entries. */
		fsIter := pvfs.IterNodes()
		for fsIter.Next() {
			macNode, node := fsIter.Current()
			if !yield(getPackfileForBlobWithError(snap, resources.RT_VFS_NODE, macNode)) {
				return
			}

			for _, entry := range node.Values {
				if !yield(getPackfileForBlobWithError(snap, resources.RT_VFS_ENTRY, entry)) {
					return
				}

				vfsEntry, err := pvfs.ResolveEntry(entry)
				if err != nil {
					if !yield(objects.MAC{}, fmt.Errorf("Failed to resolve entry %x", entry)) {
						return
					}
				}

				if vfsEntry.HasObject() {
					if !yield(getPackfileForBlobWithError(snap, resources.RT_OBJECT, vfsEntry.Object)) {
						return
					}

					for _, chunk := range vfsEntry.ResolvedObject.Chunks {
						if !yield(getPackfileForBlobWithError(snap, resources.RT_CHUNK, chunk.ContentMAC)) {
							return
						}
					}

				}

			}

		}

		if !yield(getPackfileForBlobWithError(snap, resources.RT_ERROR_BTREE, snap.Header.Sources[0].VFS.Errors)) {
			return
		}
		errIter := pvfs.IterErrorNodes()
		for errIter.Next() {
			macNode, node := errIter.Current()
			if !yield(getPackfileForBlobWithError(snap, resources.RT_ERROR_NODE, macNode)) {
				return
			}

			for _, error := range node.Values {
				if !yield(getPackfileForBlobWithError(snap, resources.RT_ERROR_ENTRY, error)) {
					return
				}
			}
		}

		if !yield(getPackfileForBlobWithError(snap, resources.RT_XATTR_BTREE, snap.Header.Sources[0].VFS.Xattrs)) {
			return
		}
		xattrIter := pvfs.XattrNodes()
		for xattrIter.Next() {
			mac, node := xattrIter.Current()
			if !yield(getPackfileForBlobWithError(snap, resources.RT_XATTR_NODE, mac)) {
				return
			}

			for _, error := range node.Values {
				if !yield(getPackfileForBlobWithError(snap, resources.RT_XATTR_ENTRY, error)) {
					return
				}
			}
		}

		// Lastly going over the indexes.
		if !yield(getPackfileForBlobWithError(snap, resources.RT_BTREE_ROOT, snap.Header.GetSource(0).Indexes[0].Value)) {
			return
		}
		rd, err := snap.Repository().GetBlob(resources.RT_BTREE_ROOT, snap.Header.GetSource(0).Indexes[0].Value)
		if err != nil {
			if !yield(objects.MAC{}, fmt.Errorf("Failed to load Index root entry %s", err)) {
				return
			}
		}

		store := repository.NewRepositoryStore[string, objects.MAC](snap.Repository(), resources.RT_BTREE_NODE)
		tree, err := btree.Deserialize(rd, store, strings.Compare)
		if err != nil {
			if !yield(objects.MAC{}, fmt.Errorf("Failed to deserialize root entry %s", err)) {
				return
			}
		}

		indexIter := tree.IterDFS()
		for indexIter.Next() {
			mac, _ := indexIter.Current()
			if !yield(getPackfileForBlobWithError(snap, resources.RT_BTREE_NODE, mac)) {
				return
			}
		}

	}, nil
}

func (snap *Snapshot) Lock() (chan bool, error) {
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

func (snap *Snapshot) Unlock(ping chan bool) {
	close(ping)
}

func (snap *Snapshot) Logger() *logging.Logger {
	return snap.AppContext().GetLogger()
}

func (snap *Snapshot) SetCheckCache(cache *caching.CheckCache) {
	snap.checkCache = cache
}
