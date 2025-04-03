package snapshot

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/PlakarKorp/plakar/events"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/snapshot/vfs"
)

type CheckOptions struct {
	MaxConcurrency uint64
	FastCheck      bool
}

func snapshotCheckPath(snap *Snapshot, opts *CheckOptions, concurrency chan bool, wg *sync.WaitGroup) func(entrypath string, e *vfs.Entry, err error) error {
	return func(entrypath string, e *vfs.Entry, err error) error {

		if err != nil {
			snap.Event(events.PathErrorEvent(snap.Header.Identifier, entrypath, err.Error()))
			return err
		}

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

		objectStatus, err := snap.checkCache.GetObjectStatus(e.Object)
		if err != nil {
			return err
		}
		if objectStatus != nil {
			if len(objectStatus) == 0 {
				return nil
			} else {
				return fmt.Errorf("%s", string(objectStatus))
			}
		}

		snap.Event(events.FileEvent(snap.Header.Identifier, entrypath))
		concurrency <- true
		wg.Add(1)
		go func(_fileEntry *vfs.Entry, path string, _entryMAC objects.MAC) {
			defer wg.Done()
			defer func() { <-concurrency }()

			object, err := snap.LookupObject(_fileEntry.Object)
			if err != nil {
				snap.Event(events.ObjectMissingEvent(snap.Header.Identifier, _fileEntry.Object))
				return
			}

			hasher := snap.repository.GetMACHasher()
			snap.Event(events.ObjectEvent(snap.Header.Identifier, object.ContentMAC))
			complete := true

			for _, chunk := range object.Chunks {
				chunkStatus, err := snap.checkCache.GetChunkStatus(chunk.ContentMAC)
				if err != nil {
					complete = false
					continue
				}
				if objectStatus != nil {
					if len(chunkStatus) == 0 {
						continue
					} else {
						complete = false
						continue
					}
				}

				snap.Event(events.ChunkEvent(snap.Header.Identifier, chunk.ContentMAC))
				if opts.FastCheck {
					if !snap.repository.BlobExists(resources.RT_CHUNK, chunk.ContentMAC) {
						snap.Event(events.ChunkMissingEvent(snap.Header.Identifier, chunk.ContentMAC))
						snap.checkCache.PutChunkStatus(chunk.ContentMAC, []byte("chunk missing"))
						complete = false
						break
					}
					snap.checkCache.PutChunkStatus(chunk.ContentMAC, []byte(""))
					snap.Event(events.ChunkOKEvent(snap.Header.Identifier, chunk.ContentMAC))
				} else {
					data, err := snap.repository.GetBlobBytes(resources.RT_CHUNK, chunk.ContentMAC)
					if err != nil {
						snap.Event(events.ChunkMissingEvent(snap.Header.Identifier, chunk.ContentMAC))
						snap.checkCache.PutChunkStatus(chunk.ContentMAC, []byte("chunk missing"))
						complete = false
						break
					}
					snap.checkCache.PutChunkStatus(chunk.ContentMAC, []byte(""))
					snap.Event(events.ChunkOKEvent(snap.Header.Identifier, chunk.ContentMAC))

					hasher.Write(data)

					mac := snap.repository.ComputeMAC(data)
					if !bytes.Equal(mac[:], chunk.ContentMAC[:]) {
						snap.Event(events.ChunkCorruptedEvent(snap.Header.Identifier, chunk.ContentMAC))
						complete = false
						break
					}
				}
			}

			if !complete {
				snap.Event(events.ObjectCorruptedEvent(snap.Header.Identifier, object.ContentMAC))
			} else {
				snap.Event(events.ObjectOKEvent(snap.Header.Identifier, object.ContentMAC))
			}

			if !opts.FastCheck {
				if !bytes.Equal(hasher.Sum(nil), object.ContentMAC[:]) {
					snap.Event(events.ObjectCorruptedEvent(snap.Header.Identifier, object.ContentMAC))
					snap.Event(events.FileCorruptedEvent(snap.Header.Identifier, path))
					return
				}
			}
			snap.Event(events.FileOKEvent(snap.Header.Identifier, entrypath, e.Size()))

			if err != nil {
				snap.checkCache.PutObjectStatus(_fileEntry.Object, []byte(err.Error()))
				snap.checkCache.PutVFSEntryStatus(_entryMAC, []byte(err.Error()))
			} else {
				snap.checkCache.PutObjectStatus(_fileEntry.Object, []byte(""))
				snap.checkCache.PutVFSEntryStatus(_entryMAC, []byte(""))
			}

		}(e, entrypath, entryMAC)
		return nil
	}
}

func (snap *Snapshot) Check(pathname string, opts *CheckOptions) (bool, error) {
	snap.Event(events.StartEvent())
	defer snap.Event(events.DoneEvent())

	vfsStatus, err := snap.checkCache.GetVFSStatus(snap.Header.GetSource(0).VFS.Root)
	if err != nil {
		return false, err
	}
	if vfsStatus != nil {
		if len(vfsStatus) == 0 {
			return true, nil
		} else {
			return false, fmt.Errorf("%s", string(vfsStatus))
		}
	}

	fs, err := snap.Filesystem()
	if err != nil {
		return false, err
	}

	maxConcurrency := opts.MaxConcurrency
	if maxConcurrency == 0 {
		maxConcurrency = uint64(snap.AppContext().MaxConcurrency)
	}

	maxConcurrencyChan := make(chan bool, maxConcurrency)
	wg := sync.WaitGroup{}
	defer wg.Wait()
	defer close(maxConcurrencyChan)

	err = fs.WalkDir(pathname, snapshotCheckPath(snap, opts, maxConcurrencyChan, &wg))
	if err != nil {
		return false, err
	}
	wg.Wait()

	if err != nil {
		snap.checkCache.PutVFSStatus(snap.Header.GetSource(0).VFS.Root, []byte(err.Error()))
	} else {
		snap.checkCache.PutVFSStatus(snap.Header.GetSource(0).VFS.Root, []byte(""))
	}

	return true, nil
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
