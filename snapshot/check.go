package snapshot

import (
	"bytes"
	"errors"
	"fmt"
	"hash"

	"github.com/PlakarKorp/plakar/events"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/snapshot/vfs"
	"golang.org/x/sync/errgroup"
)

type CheckOptions struct {
	MaxConcurrency uint64
	FastCheck      bool
}

var (
	ErrRootCorrupted   = errors.New("root corrupted")
	ErrObjectMissing   = errors.New("object is missing")
	ErrObjectCorrupted = errors.New("object corrupted")
	ErrChunkMissing    = errors.New("chunk is missing")
	ErrChunkCorrupted  = errors.New("chunk corrupted")
)

func checkChunk(snap *Snapshot, chunk *objects.Chunk, hasher hash.Hash, fast bool) error {
	chunkStatus, err := snap.checkCache.GetChunkStatus(chunk.ContentMAC)
	if err != nil {
		return err
	}

	// if chunkStatus is nil, we've never seen this chunk and we
	// have to process it.  It is zero if it's fine, or an error
	// otherwise.
	var seen bool
	if chunkStatus != nil {
		if len(chunkStatus) != 0 {
			return fmt.Errorf("%s", string(chunkStatus))
		}
		if fast {
			return nil
		}
		seen = true
	}

	snap.Event(events.ChunkEvent(snap.Header.Identifier, chunk.ContentMAC))

	if fast {
		if !snap.repository.BlobExists(resources.RT_CHUNK, chunk.ContentMAC) {
			snap.Event(events.ChunkMissingEvent(snap.Header.Identifier, chunk.ContentMAC))
			snap.checkCache.PutChunkStatus(chunk.ContentMAC, []byte(ErrChunkMissing.Error()))
			return ErrChunkMissing
		}

		snap.Event(events.ChunkOKEvent(snap.Header.Identifier, chunk.ContentMAC))
		snap.checkCache.PutChunkStatus(chunk.ContentMAC, []byte(""))
		return nil
	}

	data, err := snap.repository.GetBlobBytes(resources.RT_CHUNK, chunk.ContentMAC)
	if err != nil {
		snap.Event(events.ChunkMissingEvent(snap.Header.Identifier, chunk.ContentMAC))
		snap.checkCache.PutChunkStatus(chunk.ContentMAC, []byte(ErrChunkMissing.Error()))
		return ErrChunkMissing
	}

	hasher.Write(data)
	if seen {
		return nil
	}

	mac := snap.repository.ComputeMAC(data)
	if !bytes.Equal(mac[:], chunk.ContentMAC[:]) {
		snap.Event(events.ChunkCorruptedEvent(snap.Header.Identifier, chunk.ContentMAC))
		snap.checkCache.PutChunkStatus(chunk.ContentMAC, []byte(ErrChunkCorrupted.Error()))
		return ErrChunkCorrupted
	}

	snap.Event(events.ChunkOKEvent(snap.Header.Identifier, chunk.ContentMAC))
	snap.checkCache.PutChunkStatus(chunk.ContentMAC, []byte(""))
	return nil
}

func checkObject(snap *Snapshot, fileEntry *vfs.Entry, fast bool) error {
	objectStatus, err := snap.checkCache.GetObjectStatus(fileEntry.Object)
	if err != nil {
		return err
	}

	// if objectStatus is nil, we've never seen this object and we
	// have to process it.  It is zero if it's fine, or an error
	// otherwise.
	if objectStatus != nil {
		if len(objectStatus) != 0 {
			return fmt.Errorf("%s", string(objectStatus))
		}
		return nil
	}

	object, err := snap.LookupObject(fileEntry.Object)
	if err != nil {
		snap.Event(events.ObjectMissingEvent(snap.Header.Identifier, fileEntry.Object))
		snap.checkCache.PutObjectStatus(fileEntry.Object, []byte(ErrObjectMissing.Error()))
		return ErrObjectMissing
	}

	hasher := snap.repository.GetMACHasher()
	snap.Event(events.ObjectEvent(snap.Header.Identifier, object.ContentMAC))

	var failed bool
	for i := range object.Chunks {
		if err := checkChunk(snap, &object.Chunks[i], hasher, fast); err != nil {
			failed = true
		}
	}

	if failed {
		snap.Event(events.ObjectCorruptedEvent(snap.Header.Identifier, object.ContentMAC))
		snap.checkCache.PutObjectStatus(fileEntry.Object, []byte(ErrObjectCorrupted.Error()))
		return ErrObjectCorrupted
	}

	if !fast {
		if !bytes.Equal(hasher.Sum(nil), object.ContentMAC[:]) {
			snap.Event(events.ObjectCorruptedEvent(snap.Header.Identifier, object.ContentMAC))
			snap.checkCache.PutObjectStatus(fileEntry.Object, []byte(ErrObjectCorrupted.Error()))
			return ErrObjectCorrupted
		}
	}

	snap.Event(events.ObjectOKEvent(snap.Header.Identifier, object.ContentMAC))
	snap.checkCache.PutObjectStatus(fileEntry.Object, []byte(""))
	return nil
}

func checkEntry(snap *Snapshot, opts *CheckOptions, entrypath string, e *vfs.Entry, wg *errgroup.Group) error {
	entryMAC := e.MAC
	entryStatus, err := snap.checkCache.GetVFSEntryStatus(entryMAC)
	if err != nil {
		return err
	}
	if entryStatus != nil {
		if len(entryStatus) == 0 {
			return nil
		} else {
			return fmt.Errorf("%s", string(entryStatus))
		}
	}

	snap.Event(events.PathEvent(snap.Header.Identifier, entrypath))

	if e.Stat().Mode().IsDir() {
		snap.Event(events.DirectoryEvent(snap.Header.Identifier, entrypath))
		snap.Event(events.DirectoryOKEvent(snap.Header.Identifier, entrypath))
		snap.checkCache.PutVFSEntryStatus(entryMAC, []byte(""))
		return nil
	}

	if !e.Stat().Mode().IsRegular() {
		snap.checkCache.PutVFSEntryStatus(entryMAC, []byte(""))
		return nil
	}

	snap.Event(events.FileEvent(snap.Header.Identifier, entrypath))

	wg.Go(func() error {
		err := checkObject(snap, e, opts.FastCheck)
		if err != nil {
			snap.Event(events.FileCorruptedEvent(snap.Header.Identifier, entrypath))
			snap.checkCache.PutVFSEntryStatus(entryMAC, []byte(err.Error()))
			return err
		}

		snap.Event(events.FileOKEvent(snap.Header.Identifier, entrypath, e.Size()))
		snap.checkCache.PutVFSEntryStatus(entryMAC, []byte(""))
		return nil
	})
	return nil
}

func (snap *Snapshot) Check(pathname string, opts *CheckOptions) error {
	snap.Event(events.StartEvent())
	defer snap.Event(events.DoneEvent())

	vfsStatus, err := snap.checkCache.GetVFSStatus(snap.Header.GetSource(0).VFS.Root)
	if err != nil {
		return err
	}

	// if vfsStatus is nil, we've never seen this vfs and we have
	// to process it.  It is zero if it's fine, or an error
	// otherwise.
	if vfsStatus != nil {
		if len(vfsStatus) != 0 {
			return fmt.Errorf("%s", string(vfsStatus))
		}
		return nil
	}

	fs, err := snap.Filesystem()
	if err != nil {
		return err
	}

	maxConcurrency := opts.MaxConcurrency
	if maxConcurrency == 0 {
		maxConcurrency = uint64(snap.AppContext().MaxConcurrency)
	}

	wg := new(errgroup.Group)
	wg.SetLimit(int(maxConcurrency))

	var failed bool
	err = fs.WalkDir(pathname, func(entrypath string, e *vfs.Entry, err error) error {
		if err != nil {
			snap.Event(events.PathErrorEvent(snap.Header.Identifier, entrypath, err.Error()))
			return err
		}

		if err := snap.AppContext().Err(); err != nil {
			return err
		}

		if err := checkEntry(snap, opts, entrypath, e, wg); err != nil {
			// don't stop at the first error; we need to
			// process all the entries to report all the
			// findings.
			failed = true
		}

		return nil
	})
	if err != nil {
		snap.checkCache.PutVFSStatus(snap.Header.GetSource(0).VFS.Root, []byte(err.Error()))
		return err
	}
	if err := wg.Wait(); err != nil {
		snap.checkCache.PutVFSStatus(snap.Header.GetSource(0).VFS.Root, []byte(err.Error()))
		return err
	}
	if failed {
		snap.checkCache.PutVFSStatus(snap.Header.GetSource(0).VFS.Root,
			[]byte(ErrRootCorrupted.Error()))
		return ErrRootCorrupted
	}

	snap.checkCache.PutVFSStatus(snap.Header.GetSource(0).VFS.Root, []byte(""))
	return nil
}

/**/
/*
func (snap *Snapshot) CheckPackfile(pathname string, opts *CheckOptions) (bool, error) {
	snap.Event(events.StartEvent())
	defer snap.Event(events.DoneEvent())

	vfsStatus, err := snap.checkCache.GetVFSStatus(snap.Header.GetSource(0).VFS.Root)
	if err != nil {
		return false, err
	}
	if vfsStatus != nil {
		if len(vfsStatus) == 0 {
			return true, nil
		}
		return false, fmt.Errorf("%s", string(vfsStatus))
	}

	iter, err := snap.ListPackfiles()
	if err != nil {
		return false, err
	}

	maxConcurrency := opts.MaxConcurrency
	if maxConcurrency == 0 {
		maxConcurrency = uint64(snap.AppContext().MaxConcurrency)
	}
	maxConcurrencyChan := make(chan bool, maxConcurrency)
	var wg sync.WaitGroup
	defer wg.Wait()
	defer close(maxConcurrencyChan)

	var processed sync.Map
	complete := true

	for packfileID, err := range iter {
		if err != nil {
			snap.checkCache.PutPackfileStatus(packfileID, []byte(err.Error()))
			complete = false
			continue
		}

		if _, loaded := processed.LoadOrStore(packfileID, struct{}{}); loaded {
			// packfileID already being processed
			continue
		}

		packfileStatus, err := snap.checkCache.GetPackfileStatus(packfileID)
		if err != nil {
			complete = false
			continue
		}
		if packfileStatus != nil {
			if len(packfileStatus) == 0 {
				continue
			}
			complete = false
			continue
		}

		wg.Add(1)
		maxConcurrencyChan <- true
		go func(packfileID objects.MAC) {
			defer wg.Done()
			defer func() { <-maxConcurrencyChan }()

			rd, err := snap.repository.Store().GetPackfile(packfileID)
			if err != nil {
				snap.checkCache.PutPackfileStatus(packfileID, []byte(err.Error()))
				return
			}

			_, _, err = storage.Deserialize(snap.repository.GetMACHasher(), resources.RT_PACKFILE, rd)
			if err != nil {
				snap.checkCache.PutPackfileStatus(packfileID, []byte(err.Error()))
				return
			}

			_, err = io.Copy(io.Discard, rd)
			if err != nil {
				snap.checkCache.PutPackfileStatus(packfileID, []byte(err.Error()))
				return
			}
			snap.checkCache.PutPackfileStatus(packfileID, []byte(""))
		}(packfileID)
	}

	wg.Wait()

	if !complete {
		snap.checkCache.PutVFSStatus(snap.Header.GetSource(0).VFS.Root, []byte("check failed: packfile error"))
		return false, fmt.Errorf("check failed: packfile error")
	}

	snap.checkCache.PutVFSStatus(snap.Header.GetSource(0).VFS.Root, []byte(""))
	return true, nil
}
*/
