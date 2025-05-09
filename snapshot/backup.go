package snapshot

import (
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
	"github.com/PlakarKorp/plakar/events"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/snapshot/header"
	"github.com/PlakarKorp/plakar/snapshot/importer"
	"github.com/PlakarKorp/plakar/snapshot/vfs"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gobwas/glob"
)

type BackupContext struct {
	imp            importer.Importer
	maxConcurrency uint64

	scanCache *caching.ScanCache
	vfsCache  *caching.VFSCache

	stateId objects.MAC

	flushTick  *time.Ticker
	flushEnd   chan bool
	flushEnded chan bool

	erridx   *btree.BTree[string, int, []byte]
	xattridx *btree.BTree[string, int, []byte]
	ctidx    *btree.BTree[string, int, objects.MAC]
}

type BackupOptions struct {
	MaxConcurrency uint64
	Name           string
	Tags           []string
	Excludes       []glob.Glob
	NoCheckpoint   bool
	NoCommit       bool
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

func (snapshot *Builder) skipExcludedPathname(options *BackupOptions, record *importer.ScanResult) bool {
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

func (snap *Builder) importerJob(backupCtx *BackupContext, options *BackupOptions) (chan *importer.ScanRecord, error) {
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

		ctx := snap.AppContext()
	loop:
		for {
			var (
				_record *importer.ScanResult
				ok      bool
			)

			select {
			case _record, ok = <-scanner:
				if !ok {
					break loop
				}
			case <-ctx.Done():
				break loop
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
						// do we really need to do this?  if so, maybe we
						// should propagate the original error as well.
						ctx.Cancel()
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

		for range scanner {
			// drain the importer channel since we might
			// have been cancelled while the importer is
			// trying to still produce some records.
		}

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

func (snap *Builder) flushDeltaState(bc *BackupContext) {
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

func (snap *Builder) Backup(imp importer.Importer, options *BackupOptions) error {
	beginTime := time.Now()
	snap.Event(events.StartEvent())
	defer snap.Event(events.DoneEvent())

	done, err := snap.Lock()
	if err != nil {
		snap.repository.PackerManager.Wait()
		return err
	}
	defer snap.Unlock(done)

	snap.Header.GetSource(0).Importer.Origin = imp.Origin()
	snap.Header.GetSource(0).Importer.Type = imp.Type()
	snap.Header.Tags = append(snap.Header.Tags, options.Tags...)

	if options.Name == "" {
		snap.Header.Name = imp.Root() + " @ " + snap.Header.GetSource(0).Importer.Origin
	} else {
		snap.Header.Name = options.Name
	}
	snap.Header.GetSource(0).Importer.Directory = imp.Root()

	backupCtx, err := snap.prepareBackup(imp, options)
	if err != nil {
		snap.repository.PackerManager.Wait()
		return err
	}

	/* checkpoint handling */
	if !options.NoCheckpoint {
		backupCtx.flushTick = time.NewTicker(1 * time.Hour)
		go snap.flushDeltaState(backupCtx)
	}

	/* importer */
	filesChannel, err := snap.importerJob(backupCtx, options)
	if err != nil {
		snap.repository.PackerManager.Wait()
		return err
	}

	/* scanner */
	if err := snap.processFiles(backupCtx, filesChannel); err != nil {
		snap.repository.PackerManager.Wait()
		return err
	}

	/* tree builders */
	vfsHeader, rootSummary, indexes, err := snap.persistTrees(backupCtx)
	if err != nil {
		snap.repository.PackerManager.Wait()
		return nil
	}

	snap.Header.Duration = time.Since(beginTime)
	snap.Header.GetSource(0).VFS = *vfsHeader
	snap.Header.GetSource(0).Summary = *rootSummary
	snap.Header.GetSource(0).Indexes = indexes

	return snap.Commit(backupCtx, !options.NoCommit)
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

func (snap *Builder) chunkify(imp importer.Importer, record *importer.ScanRecord) (*objects.Object, int64, error) {
	var rd io.ReadCloser
	var err error

	if record.IsXattr {
		rd, err = imp.NewExtendedAttributeReader(record.Pathname, record.XattrName)
	} else {
		rd, err = imp.NewReader(record.Pathname)
	}

	if err != nil {
		return nil, -1, err
	}
	defer rd.Close()

	object := objects.NewObject()

	objectHasher, releaseGlobalHasher := snap.repository.GetPooledMACHasher()
	defer releaseGlobalHasher()

	var cdcOffset uint64
	var object_t32 objects.MAC

	var totalEntropy float64
	var totalFreq [256]float64
	var totalDataSize int64

	// Helper function to process a chunk
	processChunk := func(idx int, data []byte) error {
		var chunk_t32 objects.MAC

		chunkHasher, releaseChunkHasher := snap.repository.GetPooledMACHasher()
		if idx == 0 {
			if object.ContentType == "" {
				object.ContentType = mimetype.Detect(data).String()
			}
		}

		chunkHasher.Write(data)
		copy(chunk_t32[:], chunkHasher.Sum(nil))
		releaseChunkHasher()

		entropyScore, freq := entropy(data)
		if len(data) > 0 {
			for i := range 256 {
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
		totalDataSize += int64(len(data))

		return snap.repository.PutBlobIfNotExists(resources.RT_CHUNK, chunk.ContentMAC, data)
	}

	chk, err := snap.repository.Chunker(rd)
	if err != nil {
		return nil, -1, err
	}

	ctx := snap.AppContext()
	for i := 0; ; i++ {
		if i%1024 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, -1, err
			}
		}

		cdcChunk, err := chk.Next()
		if err != nil && err != io.EOF {
			return nil, -1, err
		}
		if cdcChunk == nil {
			// on an empty file, we need to compute an empty chunk for the first block
			// should we make go-cdc-chunkers return an empty chunk instead of nil?
			if i != 0 {
				break
			}
			cdcChunk = []byte{}
		}

		chunkCopy := make([]byte, len(cdcChunk))
		copy(chunkCopy, cdcChunk)

		objectHasher.Write(chunkCopy)

		if err := processChunk(i, chunkCopy); err != nil {
			return nil, -1, err
		}
		if err == io.EOF {
			break
		}
	}

	if totalDataSize > 0 {
		object.Entropy = totalEntropy / float64(totalDataSize)
	} else {
		object.Entropy = 0.0
	}

	if object.ContentType == "" {
		object.ContentType = mime.TypeByExtension(path.Ext(record.Pathname))
	}

	copy(object_t32[:], objectHasher.Sum(nil))
	object.ContentMAC = object_t32
	return object, totalDataSize, nil
}

func (snap *Builder) Commit(bc *BackupContext, commit bool) error {
	// First thing is to stop the ticker, as we don't want any concurrent flushes to run.
	// Maybe this could be stopped earlier.

	// If we end up in here without a BackupContext we come from Sync and we
	// can't rely on the flusher
	if bc != nil && bc.flushTick != nil {
		bc.flushTick.Stop()
	}

	serializedHdr, err := snap.Header.Serialize()
	if err != nil {
		return err
	}

	if kp := snap.AppContext().Keypair; kp != nil {
		serializedHdrMAC := snap.repository.ComputeMAC(serializedHdr)
		signature := kp.Sign(serializedHdrMAC[:])
		if err := snap.repository.PutBlob(resources.RT_SIGNATURE, snap.Header.Identifier, signature); err != nil {
			return err
		}
	}

	if err := snap.repository.PutBlob(resources.RT_SNAPSHOT, snap.Header.Identifier, serializedHdr); err != nil {
		return err
	}

	if !commit {
		return nil
	}

	snap.repository.PackerManager.Wait()

	// We are done with packfiles we can flush the last state, either through
	// the flusher, or manually here.
	if bc != nil && bc.flushTick != nil {
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

func (snap *Builder) prepareBackup(imp importer.Importer, backupOpts *BackupOptions) (*BackupContext, error) {

	maxConcurrency := backupOpts.MaxConcurrency
	if maxConcurrency == 0 {
		maxConcurrency = uint64(snap.AppContext().MaxConcurrency)
	}

	vfsCache, err := snap.AppContext().GetCache().VFS(snap.repository.Configuration().RepositoryID, imp.Type(), imp.Origin())
	if err != nil {
		return nil, err
	}

	backupCtx := &BackupContext{
		imp:            imp,
		maxConcurrency: maxConcurrency,
		scanCache:      snap.scanCache,
		vfsCache:       vfsCache,
		flushEnd:       make(chan bool),
		flushEnded:     make(chan bool),
		stateId:        snap.Header.Identifier,
	}

	errstore := caching.DBStore[string, []byte]{
		Prefix: "__error__",
		Cache:  snap.scanCache,
	}

	xattrstore := caching.DBStore[string, []byte]{
		Prefix: "__xattr__",
		Cache:  snap.scanCache,
	}

	ctstore := caching.DBStore[string, objects.MAC]{
		Prefix: "__contenttype__",
		Cache:  snap.scanCache,
	}

	if erridx, err := btree.New(&errstore, strings.Compare, 50); err != nil {
		return nil, err
	} else {
		backupCtx.erridx = erridx
	}

	if xattridx, err := btree.New(&xattrstore, vfs.PathCmp, 50); err != nil {
		return nil, err
	} else {
		backupCtx.xattridx = xattridx
	}

	if ctidx, err := btree.New(&ctstore, strings.Compare, 50); err != nil {
		return nil, err
	} else {
		backupCtx.ctidx = ctidx
	}

	return backupCtx, nil
}

func (snap *Builder) processFiles(backupCtx *BackupContext, filesChannel <-chan *importer.ScanRecord) error {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, backupCtx.maxConcurrency)

	ctx := snap.AppContext()

loop:
	for {
		select {
		case <-ctx.Done():
			break loop

		case record, ok := <-filesChannel:
			if !ok {
				break loop
			}

			semaphore <- struct{}{}
			wg.Add(1)

			go func(record *importer.ScanRecord) {
				defer wg.Done()
				defer func() { <-semaphore }()

				snap.Event(events.FileEvent(snap.Header.Identifier, record.Pathname))
				if err := snap.processFileRecord(backupCtx, record); err != nil {
					snap.Event(events.FileErrorEvent(snap.Header.Identifier, record.Pathname, err.Error()))
					backupCtx.recordError(record.Pathname, err)
				} else {
					snap.Event(events.FileOKEvent(snap.Header.Identifier, record.Pathname, record.FileInfo.Size()))
				}
			}(record)
		}
	}
	wg.Wait()
	for range filesChannel {
		// drain the filesChannel to consume the items that
		// the importerJob might still have inflight.
	}
	return ctx.Err()
}

func (snap *Builder) processFileRecord(backupCtx *BackupContext, record *importer.ScanRecord) error {
	vfsCache := backupCtx.vfsCache

	var err error
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
			if (record.FileInfo.Size() == -1 && cachedFileEntry.Stat().EqualIgnoreSize(&record.FileInfo)) || cachedFileEntry.Stat().Equal(&record.FileInfo) {
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
							objectMAC = snap.repository.ComputeMAC(data)
							objectSerialized = data
						}
					}
				}
			}
		}
	}

	// Chunkify the file if it is a regular file and we don't have a cached object
	if record.FileInfo.Mode().IsRegular() {
		if object == nil || !snap.repository.BlobExists(resources.RT_OBJECT, objectMAC) {
			var dataSize int64
			object, dataSize, err = snap.chunkify(backupCtx.imp, record)
			if err != nil {
				return err
			}

			// file size may have changed between the scan and chunkify
			record.FileInfo.Lsize = dataSize

			objectSerialized, err = object.Serialize()
			if err != nil {
				return err
			}
			objectMAC = snap.repository.ComputeMAC(objectSerialized)
			if err := vfsCache.PutObject(objectMAC, objectSerialized); err != nil {
				return err
			}

			if err := snap.repository.PutBlob(resources.RT_OBJECT, objectMAC, objectSerialized); err != nil {
				return err
			}
		}
	}

	// xattrs are a special case
	if record.IsXattr {
		backupCtx.recordXattr(record, objectMAC, object.Size())
		return nil
	}

	if fileEntry == nil || !snap.repository.BlobExists(resources.RT_VFS_ENTRY, cachedFileEntryMAC) {
		fileEntry = vfs.NewEntry(path.Dir(record.Pathname), record)
		if object != nil {
			fileEntry.Object = objectMAC
		}

		serialized, err := fileEntry.ToBytes()
		if err != nil {
			return err
		}

		fileEntryMAC := snap.repository.ComputeMAC(serialized)
		if err := snap.repository.PutBlob(resources.RT_VFS_ENTRY, fileEntryMAC, serialized); err != nil {
			return err
		}

		// Store the newly generated FileEntry in the cache for future runs
		if err := vfsCache.PutFilename(record.Pathname, serialized); err != nil {
			return err
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
			return err
		}

		if err := vfsCache.PutFileSummary(record.Pathname, seralizedFileSummary); err != nil {
			return err
		}
	}

	if object != nil {
		parts := strings.SplitN(object.ContentType, ";", 2)
		mime := parts[0]

		k := fmt.Sprintf("/%s%s", mime, fileEntry.Path())
		bytes, err := fileEntry.ToBytes()
		if err != nil {
			return err
		}
		if err := backupCtx.ctidx.Insert(k, snap.repository.ComputeMAC(bytes)); err != nil {
			return err
		}
	}

	return backupCtx.recordEntry(fileEntry)
}

func (snap *Builder) persistTrees(backupCtx *BackupContext) (*header.VFS, *vfs.Summary, []header.Index, error) {
	vfsHeader, summary, err := snap.persistVFS(backupCtx)
	if err != nil {
		return nil, nil, nil, err
	}

	indexes, err := snap.persistIndexes(backupCtx)
	if err != nil {
		return nil, nil, nil, err
	}

	return vfsHeader, summary, indexes, nil
}

func (snap *Builder) persistVFS(backupCtx *BackupContext) (*header.VFS, *vfs.Summary, error) {
	errcsum, err := persistMACIndex(snap, backupCtx.erridx,
		resources.RT_ERROR_BTREE, resources.RT_ERROR_NODE, resources.RT_ERROR_ENTRY)
	if err != nil {
		return nil, nil, err
	}

	filestore := caching.DBStore[string, []byte]{
		Prefix: "__path__",
		Cache:  snap.scanCache,
	}
	fileidx, err := btree.New(&filestore, vfs.PathCmp, 50)
	if err != nil {
		return nil, nil, err
	}

	var rootSummary *vfs.Summary

	diriter := backupCtx.scanCache.EnumerateKeysWithPrefix("__directory__:", true)
	for dirPath, bytes := range diriter {
		select {
		case <-snap.AppContext().Done():
			return nil, nil, snap.AppContext().Err()
		default:
		}

		dirEntry, err := vfs.EntryFromBytes(bytes)
		if err != nil {
			return nil, nil, err
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
				return nil, nil, err
			}

			data, err := backupCtx.vfsCache.GetFileSummary(childPath)
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
			return nil, nil, err
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
			return nil, nil, err
		}

		dirEntry.Summary.UpdateAverages()

		serializedSummary, err := dirEntry.Summary.ToBytes()
		if err != nil {
			backupCtx.recordError(dirPath, err)
			return nil, nil, err
		}

		err = snap.scanCache.PutSummary(dirPath, serializedSummary)
		if err != nil {
			backupCtx.recordError(dirPath, err)
			return nil, nil, err
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
			return nil, nil, err
		}

		mac := snap.repository.ComputeMAC(serialized)
		if err := snap.repository.PutBlobIfNotExists(resources.RT_VFS_ENTRY, mac, serialized); err != nil {
			return nil, nil, err
		}

		if err := fileidx.Insert(dirPath, serialized); err != nil && err != btree.ErrExists {
			return nil, nil, err
		}

		if err := backupCtx.recordEntry(dirEntry); err != nil {
			return nil, nil, err
		}
	}

	rootcsum, err := persistIndex(snap, fileidx, resources.RT_VFS_BTREE,
		resources.RT_VFS_NODE, func(data []byte) (objects.MAC, error) {
			return snap.repository.ComputeMAC(data), nil
		})
	if err != nil {
		return nil, nil, err
	}

	xattrcsum, err := persistMACIndex(snap, backupCtx.xattridx,
		resources.RT_XATTR_BTREE, resources.RT_XATTR_NODE, resources.RT_XATTR_ENTRY)
	if err != nil {
		return nil, nil, err
	}

	vfsHeader := &header.VFS{
		Root:   rootcsum,
		Xattrs: xattrcsum,
		Errors: errcsum,
	}

	return vfsHeader, rootSummary, nil

}

func (snap *Builder) persistIndexes(backupCtx *BackupContext) ([]header.Index, error) {
	ctmac, err := persistIndex(snap, backupCtx.ctidx,
		resources.RT_BTREE_ROOT, resources.RT_BTREE_NODE, func(mac objects.MAC) (objects.MAC, error) {
			return mac, nil
		})
	if err != nil {
		return nil, err
	}

	return []header.Index{
		{
			Name:  "content-type",
			Type:  "btree",
			Value: ctmac,
		},
	}, nil
}
