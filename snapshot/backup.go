package snapshot

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"mime"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PlakarKorp/plakar/btree"
	"github.com/PlakarKorp/plakar/caching"
	"github.com/PlakarKorp/plakar/classifier"
	"github.com/PlakarKorp/plakar/events"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/repository/state"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/snapshot/header"
	"github.com/PlakarKorp/plakar/snapshot/importer"
	"github.com/PlakarKorp/plakar/snapshot/vfs"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gobwas/glob"
)

type BackupContext struct {
	aborted        atomic.Bool
	abortedReason  error
	imp            importer.Importer
	maxConcurrency uint64
	scanCache      *caching.ScanCache

	stateId objects.MAC

	flushTick  *time.Ticker
	flushEnd   chan bool
	flushEnded chan bool

	erridx   *btree.BTree[string, int, []byte]
	xattridx *btree.BTree[string, int, []byte]
}

type BackupOptions struct {
	MaxConcurrency uint64
	Name           string
	Tags           []string
	Excludes       []glob.Glob
}

func (bc *BackupContext) recordEntry(entry *vfs.Entry) error {
	path := entry.Path()

	bytes, err := entry.ToBytes()
	if err != nil {
		return err
	}

	if entry.FileInfo.IsDir() {
		return bc.scanCache.PutDirectory(path, bytes)
	}
	return bc.scanCache.PutFile(path, bytes)
}

func (bc *BackupContext) recordError(path string, err error) error {
	entry := vfs.NewErrorItem(path, err.Error())
	serialized, e := entry.ToBytes()
	if e != nil {
		return err
	}

	return bc.erridx.Insert(path, serialized)
}

func (bc *BackupContext) recordXattr(record *importer.ScanRecord, objectMAC objects.MAC, size int64) error {
	xattr := vfs.NewXattr(record, objectMAC, size)
	serialized, err := xattr.ToBytes()
	if err != nil {
		return err
	}

	return bc.xattridx.Insert(xattr.ToPath(), serialized)
}

func (snapshot *Snapshot) skipExcludedPathname(options *BackupOptions, record *importer.ScanResult) bool {
	var pathname string
	switch {
	case record.Record != nil:
		pathname = record.Record.Pathname
	case record.Error != nil:
		pathname = record.Error.Pathname
	}

	if pathname == "/" {
		return false
	}

	doExclude := false
	for _, exclude := range options.Excludes {
		if exclude.Match(pathname) {
			doExclude = true
			break
		}
	}
	return doExclude
}

func (snap *Snapshot) importerJob(backupCtx *BackupContext, options *BackupOptions) (chan *importer.ScanRecord, error) {
	scanner, err := backupCtx.imp.Scan()
	if err != nil {
		return nil, err
	}

	wg := sync.WaitGroup{}
	filesChannel := make(chan *importer.ScanRecord, 1000)
	repoLocation := snap.repository.Location()

	go func() {
		startEvent := events.StartImporterEvent()
		startEvent.SnapshotID = snap.Header.Identifier
		snap.Event(startEvent)

		nFiles := uint64(0)
		nDirectories := uint64(0)
		size := uint64(0)

		concurrencyChan := make(chan struct{}, backupCtx.maxConcurrency)

		for _record := range scanner {
			if backupCtx.aborted.Load() {
				break
			}
			if snap.skipExcludedPathname(options, _record) {
				continue
			}

			concurrencyChan <- struct{}{}
			wg.Add(1)
			go func(record *importer.ScanResult) {
				defer func() {
					<-concurrencyChan
					wg.Done()
				}()

				switch {
				case record.Error != nil:
					record := record.Error
					if record.Pathname == backupCtx.imp.Root() || len(record.Pathname) < len(backupCtx.imp.Root()) {
						backupCtx.aborted.Store(true)
						backupCtx.abortedReason = record.Err
						return
					}
					backupCtx.recordError(record.Pathname, record.Err)
					snap.Event(events.PathErrorEvent(snap.Header.Identifier, record.Pathname, record.Err.Error()))

				case record.Record != nil:
					record := record.Record
					snap.Event(events.PathEvent(snap.Header.Identifier, record.Pathname))

					if strings.HasPrefix(record.Pathname, repoLocation+"/") {
						snap.Logger().Warn("skipping entry from repository: %s", record.Pathname)
						// skip repository directory
						return
					}

					if record.FileInfo.Mode().IsDir() {
						atomic.AddUint64(&nDirectories, +1)
						entry := vfs.NewEntry(path.Dir(record.Pathname), record)
						if err := backupCtx.recordEntry(entry); err != nil {
							backupCtx.recordError(record.Pathname, err)
						}
						return
					}

					filesChannel <- record
					if record.IsXattr {
						return
					}

					atomic.AddUint64(&nFiles, +1)
					if record.FileInfo.Mode().IsRegular() {
						atomic.AddUint64(&size, uint64(record.FileInfo.Size()))
					}
					// if snapshot root is a file, then reset to the parent directory
					if snap.Header.GetSource(0).Importer.Directory == record.Pathname {
						snap.Header.GetSource(0).Importer.Directory = filepath.Dir(record.Pathname)
					}
				}
			}(_record)
		}
		wg.Wait()
		close(filesChannel)
		doneEvent := events.DoneImporterEvent()
		doneEvent.SnapshotID = snap.Header.Identifier
		doneEvent.NumFiles = nFiles
		doneEvent.NumDirectories = nDirectories
		doneEvent.Size = size
		snap.Event(doneEvent)
	}()

	return filesChannel, nil
}

func (snap *Snapshot) flushDeltaState(bc *BackupContext) {
	for {
		select {
		case <-bc.flushEnd:
			// End of backup we push the last and final State. No need to take any locks at this point.
			err := snap.repository.CommitTransaction(bc.stateId)
			if err != nil {
				// XXX: ERROR HANDLING
				snap.Logger().Warn("Failed to push the final state to the repository %s", err)
			}

			// See below
			if snap.deltaCache != snap.scanCache {
				snap.deltaCache.Close()
			}

			bc.flushEnded <- true
			close(bc.flushEnded)
			return
		case <-bc.flushTick.C:
			// Now make a new state backed by a new cache.
			snap.deltaMtx.Lock()
			oldCache := snap.deltaCache
			oldStateId := bc.stateId

			identifier := objects.RandomMAC()
			deltaCache, err := snap.repository.AppContext().GetCache().Scan(identifier)
			if err != nil {
				// XXX: ERROR HANDLING
				snap.deltaMtx.Unlock()
				snap.Logger().Warn("Failed to open deltaCache %s\n", err)
				break
			}

			bc.stateId = identifier
			snap.deltaMtx.Unlock()

			// Now that the backup is free to progress we can serialize and push
			// the resulting statefile to the repo.
			err = snap.repository.FlushTransaction(deltaCache, oldStateId)
			if err != nil {
				// XXX: ERROR HANDLING
				snap.Logger().Warn("Failed to push the state to the repository %s", err)
			}

			// The first cache is always the scanCache, only in this function we
			// allocate a new and different one, so when we first hit this function
			// do not close the deltaCache, as it'll be closed at the end of the
			// backup because it's used by other parts of the code.
			if oldCache != snap.scanCache {
				oldCache.Close()
			}
		}
	}
}

func (snap *Snapshot) Backup(imp importer.Importer, options *BackupOptions) error {
	snap.Event(events.StartEvent())
	defer snap.Event(events.DoneEvent())

	done, err := snap.Lock()
	if err != nil {
		return err
	}
	defer snap.Unlock(done)

	vfsCache, err := snap.AppContext().GetCache().VFS(snap.repository.Configuration().RepositoryID, imp.Type(), imp.Origin())
	if err != nil {
		return err
	}
	cf, err := classifier.NewClassifier(snap.AppContext())
	if err != nil {
		return err
	}
	defer cf.Close()

	snap.Header.GetSource(0).Importer.Origin = imp.Origin()
	snap.Header.GetSource(0).Importer.Type = imp.Type()
	snap.Header.Tags = append(snap.Header.Tags, options.Tags...)

	if options.Name == "" {
		snap.Header.Name = imp.Root() + " @ " + snap.Header.GetSource(0).Importer.Origin
	} else {
		snap.Header.Name = options.Name
	}

	snap.Header.GetSource(0).Importer.Directory = imp.Root()

	maxConcurrency := options.MaxConcurrency
	if maxConcurrency == 0 {
		maxConcurrency = uint64(snap.AppContext().MaxConcurrency)
	}

	backupCtx := &BackupContext{
		imp:            imp,
		maxConcurrency: maxConcurrency,
		scanCache:      snap.scanCache,
		flushTick:      time.NewTicker(1 * time.Hour),
		flushEnd:       make(chan bool),
		flushEnded:     make(chan bool),
		stateId:        snap.Header.Identifier,
	}

	go snap.flushDeltaState(backupCtx)

	errstore := caching.DBStore[string, []byte]{
		Prefix: "__error__",
		Cache:  snap.scanCache,
	}
	backupCtx.erridx, err = btree.New(&errstore, strings.Compare, 50)
	if err != nil {
		return err
	}

	xattrstore := caching.DBStore[string, []byte]{
		Prefix: "__xattr__",
		Cache:  snap.scanCache,
	}
	backupCtx.xattridx, err = btree.New(&xattrstore, vfs.PathCmp, 50)
	if err != nil {
		return err
	}

	ctstore := caching.DBStore[string, objects.MAC]{
		Prefix: "__contenttype__",
		Cache:  snap.scanCache,
	}
	ctidx, err := btree.New(&ctstore, strings.Compare, 50)
	if err != nil {
		return err
	}

	/* backup starts now */
	beginTime := time.Now()

	/* importer */
	filesChannel, err := snap.importerJob(backupCtx, options)
	if err != nil {
		return err
	}

	concurrencyChan := make(chan struct{}, maxConcurrency)

	/* scanner */
	scannerWg := sync.WaitGroup{}
	for _record := range filesChannel {
		select {
		case <-snap.AppContext().GetContext().Done():
			return snap.AppContext().GetContext().Err()
		default:
		}

		concurrencyChan <- struct{}{}
		scannerWg.Add(1)
		go func(record *importer.ScanRecord) {
			defer func() {
				<-concurrencyChan
				scannerWg.Done()
			}()

			var err error

			snap.Event(events.FileEvent(snap.Header.Identifier, record.Pathname))

			var fileEntry *vfs.Entry
			var object *objects.Object
			var objectMAC objects.MAC
			var objectSerialized []byte

			var cachedFileEntry *vfs.Entry
			var cachedFileEntryMAC objects.MAC

			// Check if the file entry and underlying objects are already in the cache
			if data, err := vfsCache.GetFilename(record.Pathname); err != nil {
				snap.Logger().Warn("VFS CACHE: Error getting filename: %v", err)
			} else if data != nil {
				cachedFileEntry, err = vfs.EntryFromBytes(data)
				if err != nil {
					snap.Logger().Warn("VFS CACHE: Error unmarshaling filename: %v", err)
				} else {
					cachedFileEntryMAC = snap.repository.ComputeMAC(data)
					if cachedFileEntry.Stat().Equal(&record.FileInfo) {
						fileEntry = cachedFileEntry
						if fileEntry.FileInfo.Mode().IsRegular() {
							data, err := vfsCache.GetObject(cachedFileEntry.Object)
							if err != nil {
								snap.Logger().Warn("VFS CACHE: Error getting object: %v", err)
							} else if data != nil {
								cachedObject, err := objects.NewObjectFromBytes(data)
								if err != nil {
									snap.Logger().Warn("VFS CACHE: Error unmarshaling object: %v", err)
								} else {
									object = cachedObject
									objectMAC = snap.Repository().ComputeMAC(data)
									objectSerialized = data
								}
							}
						}
					}
				}
			}

			if object != nil {
				if err := snap.PutBlobIfNotExists(resources.RT_OBJECT, objectMAC, objectSerialized); err != nil {
					snap.Event(events.FileErrorEvent(snap.Header.Identifier, record.Pathname, err.Error()))
					backupCtx.recordError(record.Pathname, err)
					return
				}
			}

			// Chunkify the file if it is a regular file and we don't have a cached object
			if record.FileInfo.Mode().IsRegular() {
				if object == nil || !snap.BlobExists(resources.RT_OBJECT, objectMAC) {
					object, err = snap.chunkify(imp, cf, record)
					if err != nil {
						snap.Event(events.FileErrorEvent(snap.Header.Identifier, record.Pathname, err.Error()))
						backupCtx.recordError(record.Pathname, err)
						return
					}
					objectSerialized, err = object.Serialize()
					if err != nil {
						snap.Event(events.FileErrorEvent(snap.Header.Identifier, record.Pathname, err.Error()))
						backupCtx.recordError(record.Pathname, err)
						return
					}
					objectMAC = snap.repository.ComputeMAC(objectSerialized)
					if err := vfsCache.PutObject(objectMAC, objectSerialized); err != nil {
						snap.Event(events.FileErrorEvent(snap.Header.Identifier, record.Pathname, err.Error()))
						backupCtx.recordError(record.Pathname, err)
						return
					}

					if err := snap.PutBlob(resources.RT_OBJECT, objectMAC, objectSerialized); err != nil {
						snap.Event(events.FileErrorEvent(snap.Header.Identifier, record.Pathname, err.Error()))
						backupCtx.recordError(record.Pathname, err)
						return
					}
				}
			}

			// xattrs are a special case
			if record.IsXattr {
				backupCtx.recordXattr(record, objectMAC, object.Size())
				return
			}

			if fileEntry == nil || !snap.BlobExists(resources.RT_VFS_ENTRY, cachedFileEntryMAC) {
				fileEntry = vfs.NewEntry(path.Dir(record.Pathname), record)
				if object != nil {
					fileEntry.Object = objectMAC
				}

				classifications := cf.Processor(record.Pathname).File(fileEntry)
				for _, result := range classifications {
					fileEntry.AddClassification(result.Analyzer, result.Classes)
				}

				serialized, err := fileEntry.ToBytes()
				if err != nil {
					snap.Event(events.FileErrorEvent(snap.Header.Identifier, record.Pathname, err.Error()))
					backupCtx.recordError(record.Pathname, err)
					return
				}

				fileEntryMAC := snap.repository.ComputeMAC(serialized)
				if err := snap.PutBlob(resources.RT_VFS_ENTRY, fileEntryMAC, serialized); err != nil {
					snap.Event(events.FileErrorEvent(snap.Header.Identifier, record.Pathname, err.Error()))
					backupCtx.recordError(record.Pathname, err)
					return
				}

				// Store the newly generated FileEntry in the cache for future runs
				if err := vfsCache.PutFilename(record.Pathname, serialized); err != nil {
					snap.Event(events.FileErrorEvent(snap.Header.Identifier, record.Pathname, err.Error()))
					backupCtx.recordError(record.Pathname, err)
					return
				}

				fileSummary := &vfs.FileSummary{
					Size:    uint64(record.FileInfo.Size()),
					Mode:    record.FileInfo.Mode(),
					ModTime: record.FileInfo.ModTime().Unix(),
				}
				if object != nil {
					fileSummary.Objects++
					fileSummary.Chunks += uint64(len(object.Chunks))
					fileSummary.ContentType = object.ContentType
					fileSummary.Entropy = object.Entropy
				}

				seralizedFileSummary, err := fileSummary.Serialize()
				if err != nil {
					snap.Event(events.FileErrorEvent(snap.Header.Identifier, record.Pathname, err.Error()))
					backupCtx.recordError(record.Pathname, err)
					return
				}

				if err := vfsCache.PutFileSummary(record.Pathname, seralizedFileSummary); err != nil {
					snap.Event(events.FileErrorEvent(snap.Header.Identifier, record.Pathname, err.Error()))
					backupCtx.recordError(record.Pathname, err)
					return
				}
			}

			if object != nil {
				parts := strings.SplitN(object.ContentType, ";", 2)
				mime := parts[0]

				k := fmt.Sprintf("/%s%s", mime, fileEntry.Path())
				bytes, err := fileEntry.ToBytes()
				if err != nil {
					snap.Event(events.FileErrorEvent(snap.Header.Identifier, record.Pathname, err.Error()))
					backupCtx.recordError(record.Pathname, err)
					return
				}
				if err := ctidx.Insert(k, snap.repository.ComputeMAC(bytes)); err != nil {
					snap.Event(events.FileErrorEvent(snap.Header.Identifier, record.Pathname, err.Error()))
					backupCtx.recordError(record.Pathname, err)
					return
				}
			}

			if err := backupCtx.recordEntry(fileEntry); err != nil {
				snap.Event(events.FileErrorEvent(snap.Header.Identifier, record.Pathname, err.Error()))
				backupCtx.recordError(record.Pathname, err)
				return
			}

			snap.Event(events.FileOKEvent(snap.Header.Identifier, record.Pathname, record.FileInfo.Size()))
		}(_record)
	}
	scannerWg.Wait()

	errcsum, err := persistMACIndex(snap, backupCtx.erridx,
		resources.RT_ERROR_BTREE, resources.RT_ERROR_NODE, resources.RT_ERROR_ENTRY)
	if err != nil {
		return err
	}

	filestore := caching.DBStore[string, []byte]{
		Prefix: "__path__",
		Cache:  snap.scanCache,
	}
	fileidx, err := btree.New(&filestore, vfs.PathCmp, 50)
	if err != nil {
		return err
	}

	var rootSummary *vfs.Summary

	diriter := backupCtx.scanCache.EnumerateKeysWithPrefix("__directory__:", true)
	for dirPath, bytes := range diriter {
		select {
		case <-snap.AppContext().GetContext().Done():
			return snap.AppContext().GetContext().Err()
		default:
		}

		dirEntry, err := vfs.EntryFromBytes(bytes)
		if err != nil {
			return err
		}

		prefix := dirPath
		if prefix != "/" {
			prefix += "/"
		}

		childiter := backupCtx.scanCache.EnumerateKeysWithPrefix("__file__:"+prefix, false)

		for relpath, bytes := range childiter {
			if strings.Contains(relpath, "/") {
				continue
			}

			// bytes is a slice that will be reused in the next iteration,
			// swapping below our feet, so make a copy out of it
			dupBytes := make([]byte, len(bytes))
			copy(dupBytes, bytes)

			childPath := prefix + relpath

			if err := fileidx.Insert(childPath, dupBytes); err != nil && err != btree.ErrExists {
				return err
			}

			data, err := vfsCache.GetFileSummary(childPath)
			if err != nil {
				continue
			}

			fileSummary, err := vfs.FileSummaryFromBytes(data)
			if err != nil {
				continue
			}

			dirEntry.Summary.Directory.Children++
			dirEntry.Summary.UpdateWithFileSummary(fileSummary)
		}

		subDirIter := backupCtx.scanCache.EnumerateKeysWithPrefix("__directory__:"+prefix, false)
		for relpath := range subDirIter {
			if relpath == "" || strings.Contains(relpath, "/") {
				continue
			}

			childPath := prefix + relpath
			data, err := snap.scanCache.GetSummary(childPath)
			if err != nil {
				continue
			}

			childSummary, err := vfs.SummaryFromBytes(data)
			if err != nil {
				continue
			}
			dirEntry.Summary.Directory.Children++
			dirEntry.Summary.Directory.Directories++
			dirEntry.Summary.UpdateBelow(childSummary)
		}

		erriter, err := backupCtx.erridx.ScanFrom(prefix)
		if err != nil {
			return err
		}
		for erriter.Next() {
			path, _ := erriter.Current()
			if !strings.HasPrefix(path, prefix) {
				break
			}
			if strings.Contains(path[len(prefix):], "/") {
				break
			}
			dirEntry.Summary.Below.Errors++
		}
		if err := erriter.Err(); err != nil {
			return err
		}

		dirEntry.Summary.UpdateAverages()

		classifications := cf.Processor(dirPath).Directory(dirEntry)
		for _, result := range classifications {
			dirEntry.AddClassification(result.Analyzer, result.Classes)
		}

		serializedSummary, err := dirEntry.Summary.ToBytes()
		if err != nil {
			backupCtx.recordError(dirPath, err)
			return err
		}

		err = snap.scanCache.PutSummary(dirPath, serializedSummary)
		if err != nil {
			backupCtx.recordError(dirPath, err)
			return err
		}

		snap.Event(events.DirectoryOKEvent(snap.Header.Identifier, dirPath))
		if dirPath == "/" {
			if rootSummary != nil {
				panic("double /!")
			}
			rootSummary = dirEntry.Summary
		}

		serialized, err := dirEntry.ToBytes()
		if err != nil {
			return err
		}

		mac := snap.repository.ComputeMAC(serialized)
		if err := snap.PutBlobIfNotExists(resources.RT_VFS_ENTRY, mac, serialized); err != nil {
			return err
		}

		if err := fileidx.Insert(dirPath, serialized); err != nil && err != btree.ErrExists {
			return err
		}

		if err := backupCtx.recordEntry(dirEntry); err != nil {
			return err
		}
	}

	// hits, miss, cachesize := fileidx.Stats()
	// log.Printf("before persist: fileidx: hits/miss/size: %d/%d/%d", hits, miss, cachesize)

	rootcsum, err := persistIndex(snap, fileidx, resources.RT_VFS_BTREE,
		resources.RT_VFS_NODE, func(data []byte) (objects.MAC, error) {
			return snap.repository.ComputeMAC(data), nil
		})
	if err != nil {
		return err
	}

	// hits, miss, cachesize = fileidx.Stats()
	// log.Printf("after persist: fileidx: hits/miss/size: %d/%d/%d", hits, miss, cachesize)

	xattrcsum, err := persistMACIndex(snap, backupCtx.xattridx,
		resources.RT_XATTR_BTREE, resources.RT_XATTR_NODE, resources.RT_XATTR_ENTRY)
	if err != nil {
		return err
	}

	// hits, miss, cachesize = ctidx.Stats()
	// log.Printf("before persist: ctidx: hits/miss/size: %d/%d/%d", hits, miss, cachesize)

	ctmac, err := persistIndex(snap, ctidx, resources.RT_BTREE_ROOT, resources.RT_BTREE_NODE, func(mac objects.MAC) (objects.MAC, error) {
		return mac, nil
	})
	if err != nil {
		return err
	}

	// hits, miss, cachesize = ctidx.Stats()
	// log.Printf("after persist: ctidx: hits/miss/size: %d/%d/%d", hits, miss, cachesize)

	if backupCtx.aborted.Load() {
		return backupCtx.abortedReason
	}

	snap.Header.GetSource(0).VFS = header.VFS{
		Root:   rootcsum,
		Xattrs: xattrcsum,
		Errors: errcsum,
	}
	snap.Header.Duration = time.Since(beginTime)
	snap.Header.GetSource(0).Summary = *rootSummary
	snap.Header.GetSource(0).Indexes = []header.Index{
		{
			Name:  "content-type",
			Type:  "btree",
			Value: ctmac,
		},
	}

	return snap.Commit(backupCtx)
}

func entropy(data []byte) (float64, [256]float64) {
	if len(data) == 0 {
		return 0.0, [256]float64{}
	}

	// Count the frequency of each byte value
	var freq [256]float64
	for _, b := range data {
		freq[b]++
	}

	// Calculate the entropy
	entropy := 0.0
	dataSize := float64(len(data))
	for _, f := range freq {
		if f > 0 {
			p := f / dataSize
			entropy -= p * math.Log2(p)
		}
	}
	return entropy, freq
}

func (snap *Snapshot) chunkify(imp importer.Importer, cf *classifier.Classifier, record *importer.ScanRecord) (*objects.Object, error) {
	var rd io.ReadCloser
	var err error

	if record.IsXattr {
		rd, err = imp.NewExtendedAttributeReader(record.Pathname, record.XattrName)
	} else {
		rd, err = imp.NewReader(record.Pathname)
	}

	if err != nil {
		return nil, err
	}
	defer rd.Close()

	object := objects.NewObject()
	object.ContentType = mime.TypeByExtension(path.Ext(record.Pathname))

	objectHasher := snap.repository.GetMACHasher()

	var firstChunk = true
	var cdcOffset uint64
	var object_t32 objects.MAC

	var totalEntropy float64
	var totalFreq [256]float64
	var totalDataSize uint64

	// Helper function to process a chunk
	processChunk := func(data []byte) error {
		var chunk_t32 objects.MAC
		chunkHasher := snap.repository.GetMACHasher()

		if firstChunk {
			if object.ContentType == "" {
				object.ContentType = mimetype.Detect(data).String()
			}
			firstChunk = false
		}
		objectHasher.Write(data)

		chunkHasher.Reset()
		chunkHasher.Write(data)
		copy(chunk_t32[:], chunkHasher.Sum(nil))

		entropyScore, freq := entropy(data)
		if len(data) > 0 {
			for i := 0; i < 256; i++ {
				totalFreq[i] += freq[i]
			}
		}
		chunk := objects.NewChunk()
		chunk.ContentMAC = chunk_t32
		chunk.Length = uint32(len(data))
		chunk.Entropy = entropyScore

		object.Chunks = append(object.Chunks, *chunk)
		cdcOffset += uint64(len(data))

		totalEntropy += chunk.Entropy * float64(len(data))
		totalDataSize += uint64(len(data))

		return snap.PutBlobIfNotExists(resources.RT_CHUNK, chunk.ContentMAC, data)
	}

	if record.FileInfo.Size() == 0 {
		// Produce an empty chunk for empty file
		if err := processChunk([]byte{}); err != nil {
			return nil, err
		}
	} else if record.FileInfo.Size() < int64(snap.repository.Configuration().Chunking.MinSize) {
		// Small file case: read entire file into memory
		buf, err := io.ReadAll(rd)
		if err != nil {
			return nil, err
		}
		if err := processChunk(buf); err != nil {
			return nil, err
		}
	} else {
		// Large file case: chunk file with chunker
		chk, err := snap.repository.Chunker(rd)
		if err != nil {
			return nil, err
		}
		for {
			cdcChunk, err := chk.Next()
			if err != nil && err != io.EOF {
				return nil, err
			}
			if cdcChunk == nil {
				break
			}
			if err := processChunk(cdcChunk); err != nil {
				return nil, err
			}
			if err == io.EOF {
				break
			}
		}
	}

	if totalDataSize > 0 {
		object.Entropy = totalEntropy / float64(totalDataSize)
	} else {
		object.Entropy = 0.0
	}

	copy(object_t32[:], objectHasher.Sum(nil))
	object.ContentMAC = object_t32
	return object, nil
}

func (snap *Snapshot) PutPackfile(packer *Packer) error {

	repo := snap.repository

	serializedData, err := packer.Packfile.SerializeData()
	if err != nil {
		return fmt.Errorf("could not serialize pack file data %s", err.Error())
	}
	serializedIndex, err := packer.Packfile.SerializeIndex()
	if err != nil {
		return fmt.Errorf("could not serialize pack file index %s", err.Error())
	}
	serializedFooter, err := packer.Packfile.SerializeFooter()
	if err != nil {
		return fmt.Errorf("could not serialize pack file footer %s", err.Error())
	}

	encryptedIndex, err := repo.EncodeBuffer(serializedIndex)
	if err != nil {
		return err
	}

	encryptedFooter, err := repo.EncodeBuffer(serializedFooter)
	if err != nil {
		return err
	}

	serializedPackfile := append(serializedData, encryptedIndex...)
	serializedPackfile = append(serializedPackfile, encryptedFooter...)

	/* it is necessary to track the footer _encrypted_ length */
	encryptedFooterLength := make([]byte, 4)
	binary.LittleEndian.PutUint32(encryptedFooterLength, uint32(len(encryptedFooter)))
	serializedPackfile = append(serializedPackfile, encryptedFooterLength...)

	mac := snap.repository.ComputeMAC(serializedPackfile)

	repo.Logger().Trace("snapshot", "%x: PutPackfile(%x, ...)", snap.Header.GetIndexShortID(), mac)
	err = snap.repository.PutPackfile(mac, bytes.NewBuffer(serializedPackfile))
	if err != nil {
		return fmt.Errorf("could not write pack file %s", err.Error())
	}

	snap.deltaMtx.RLock()
	defer snap.deltaMtx.RUnlock()
	for _, Type := range packer.Types() {
		for blobMAC := range packer.Blobs[Type] {
			for idx, blob := range packer.Packfile.Index {
				if blob.MAC == blobMAC && blob.Type == Type {
					delta := &state.DeltaEntry{
						Type:    blob.Type,
						Version: packer.Packfile.Index[idx].Version,
						Blob:    blobMAC,
						Location: state.Location{
							Packfile: mac,
							Offset:   packer.Packfile.Index[idx].Offset,
							Length:   packer.Packfile.Index[idx].Length,
						},
					}

					if err := snap.repository.PutStateDelta(delta); err != nil {
						return err
					}

				}
			}
		}
	}

	if err := snap.repository.PutStatePackfile(snap.Header.Identifier, mac); err != nil {
		return err
	}

	return nil
}

func (snap *Snapshot) Commit(bc *BackupContext) error {
	// First thing is to stop the ticker, as we don't want any concurrent flushes to run.
	// Maybe this could be stopped earlier.

	// If we end up in here without a BackupContext we come from Sync and we
	// can't rely on the flusher
	if bc != nil {
		bc.flushTick.Stop()
	}

	serializedHdr, err := snap.Header.Serialize()
	if err != nil {
		return err
	}

	if kp := snap.AppContext().Keypair; kp != nil {
		serializedHdrMAC := snap.repository.ComputeMAC(serializedHdr)
		signature := kp.Sign(serializedHdrMAC[:])
		if err := snap.PutBlob(resources.RT_SIGNATURE, snap.Header.Identifier, signature); err != nil {
			return err
		}
	}

	if err := snap.PutBlob(resources.RT_SNAPSHOT, snap.Header.Identifier, serializedHdr); err != nil {
		return err
	}
	snap.packerManager.Wait()

	// We are done with packfiles we can flush the last state, either through
	// the flusher, or manually here.
	if bc != nil {
		bc.flushEnd <- true
		close(bc.flushEnd)
		<-bc.flushEnded
	} else {
		err = snap.repository.CommitTransaction(snap.Header.Identifier)
		if err != nil {
			snap.Logger().Warn("Failed to push the state to the repository %s", err)
			return err
		}
	}

	cache, err := snap.AppContext().GetCache().Repository(snap.repository.Configuration().RepositoryID)
	if err == nil {
		_ = cache.PutSnapshot(snap.Header.Identifier, serializedHdr)
	}

	snap.Logger().Trace("snapshot", "%x: Commit()", snap.Header.GetIndexShortID())
	return nil
}

func buildSerializedDeltaState(deltaState *state.LocalState) io.Reader {
	pr, pw := io.Pipe()

	/* By using a pipe and a goroutine we bound the max size in memory. */
	go func() {
		defer pw.Close()
		if err := deltaState.SerializeToStream(pw); err != nil {
			pw.CloseWithError(err)
		}
	}()

	return pr
}
