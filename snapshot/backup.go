package snapshot

import (
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
	"github.com/PlakarKorp/plakar/packfile"
	"github.com/PlakarKorp/plakar/snapshot/importer"
	"github.com/PlakarKorp/plakar/snapshot/vfs"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gobwas/glob"
)

type BackupContext struct {
	aborted        atomic.Bool
	abortedReason  error
	imp            importer.Importer
	sc             *caching.ScanCache
	maxConcurrency chan bool
	tree           *btree.BTree[string, int, ErrorItem]
	mutree         sync.Mutex
}

type BackupOptions struct {
	MaxConcurrency uint64
	Name           string
	Tags           []string
	Excludes       []glob.Glob
}

func (bc *BackupContext) recordError(path string, err error) error {
	bc.mutree.Lock()
	e := bc.tree.Insert(path, ErrorItem{
		Name:  path,
		Error: err.Error(),
	})
	bc.mutree.Unlock()
	return e
}

func (snapshot *Snapshot) skipExcludedPathname(options *BackupOptions, record importer.ScanResult) bool {
	var pathname string
	switch record := record.(type) {
	case importer.ScanError:
		pathname = record.Pathname
	case importer.ScanRecord:
		pathname = record.Pathname
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

func (snap *Snapshot) importerJob(backupCtx *BackupContext, options *BackupOptions) (chan importer.ScanRecord, error) {
	scanner, err := backupCtx.imp.Scan()
	if err != nil {
		return nil, err
	}

	wg := sync.WaitGroup{}
	filesChannel := make(chan importer.ScanRecord, 1000)

	go func() {
		for _record := range scanner {

			if backupCtx.aborted.Load() {
				break
			}
			if snap.skipExcludedPathname(options, _record) {
				continue
			}

			backupCtx.maxConcurrency <- true
			wg.Add(1)
			go func(record importer.ScanResult) {
				defer func() {
					<-backupCtx.maxConcurrency
					wg.Done()
				}()

				switch record := record.(type) {
				case importer.ScanError:
					if record.Pathname == backupCtx.imp.Root() || len(record.Pathname) < len(backupCtx.imp.Root()) {
						backupCtx.aborted.Store(true)
						backupCtx.abortedReason = record.Err
						return
					}
					backupCtx.recordError(record.Pathname, record.Err)
					snap.Event(events.PathErrorEvent(snap.Header.Identifier, record.Pathname, record.Err.Error()))

				case importer.ScanRecord:
					snap.Event(events.PathEvent(snap.Header.Identifier, record.Pathname))

					serializedRecord, err := record.ToBytes()
					if err != nil {
						backupCtx.recordError(record.Pathname, err)
						return
					}

					pathname := record.Pathname
					if record.FileInfo.Mode().IsDir() && pathname != "/" {
						pathname += "/"
					}

					if err := backupCtx.sc.PutPathname(pathname, serializedRecord); err != nil {
						backupCtx.recordError(record.Pathname, err)
						return
					}
					if !record.FileInfo.Mode().IsDir() {
						filesChannel <- record
					}
				}
			}(_record)
		}
		wg.Wait()
		close(filesChannel)
	}()

	return filesChannel, nil
}

func (snap *Snapshot) Backup(scanDir string, options *BackupOptions) error {
	snap.Event(events.StartEvent())
	defer snap.Event(events.DoneEvent())

	sc2, err := snap.repository.Context().GetCache().Scan(snap.Header.Identifier)
	if err != nil {
		return err
	}
	defer sc2.Close()

	imp, err := importer.NewImporter(scanDir)
	if err != nil {
		return err
	}
	defer imp.Close()

	vfsCache, err := snap.Repository().Context().GetCache().VFS(imp.Type(), imp.Origin())
	if err != nil {
		return err
	}

	cf, err := classifier.NewClassifier(snap.Context())
	if err != nil {
		return err
	}
	defer cf.Close()

	snap.Header.Importer.Origin = imp.Origin()
	snap.Header.Importer.Type = imp.Type()
	snap.Header.Tags = append(snap.Header.Tags, options.Tags...)

	if options.Name == "" {
		snap.Header.Name = scanDir + " @ " + snap.Header.Importer.Origin
	} else {
		snap.Header.Name = options.Name
	}

	if !strings.Contains(scanDir, "://") {
		scanDir, err = filepath.Abs(scanDir)
		if err != nil {
			snap.Logger().Warn("%s", err)
			return err
		}
	} else {
		scanDir = imp.Root()
	}
	snap.Header.Importer.Directory = scanDir

	maxConcurrency := options.MaxConcurrency
	if maxConcurrency == 0 {
		maxConcurrency = uint64(snap.Context().GetMaxConcurrency())
	}

	backupCtx := &BackupContext{
		imp:            imp,
		sc:             sc2,
		maxConcurrency: make(chan bool, maxConcurrency),
	}

	ds := caching.DBStore[string, ErrorItem]{
		Prefix: "__error__",
		Cache:  sc2,
	}
	backupCtx.tree, err = btree.New(&ds, strings.Compare, 50)
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

	/* scanner */
	scannerWg := sync.WaitGroup{}
	for _record := range filesChannel {
		backupCtx.maxConcurrency <- true
		scannerWg.Add(1)
		go func(record importer.ScanRecord) {
			defer func() {
				<-backupCtx.maxConcurrency
				scannerWg.Done()
			}()

			snap.Event(events.FileEvent(snap.Header.Identifier, record.Pathname))

			var fileEntry *vfs.FileEntry
			var object *objects.Object

			var cachedFileEntry *vfs.FileEntry
			var cachedFileEntryChecksum objects.Checksum

			// Check if the file entry and underlying objects are already in the cache
			if data, err := vfsCache.GetFilename(record.Pathname); err != nil {
				snap.Logger().Warn("VFS CACHE: Error getting filename: %v", err)
			} else if data != nil {
				cachedFileEntry, err = vfs.FileEntryFromBytes(data)
				if err != nil {
					snap.Logger().Warn("VFS CACHE: Error unmarshaling filename: %v", err)
				} else {
					cachedFileEntryChecksum = snap.repository.Checksum(data)
					if cachedFileEntry.Stat().ModTime().Equal(record.FileInfo.ModTime()) && cachedFileEntry.Stat().Size() == record.FileInfo.Size() {
						fileEntry = cachedFileEntry
						if fileEntry.Type == importer.RecordTypeFile {
							data, err := vfsCache.GetObject(cachedFileEntry.Object.Checksum)
							if err != nil {
								snap.Logger().Warn("VFS CACHE: Error getting object: %v", err)
							} else if data != nil {
								cachedObject, err := objects.NewObjectFromBytes(data)
								if err != nil {
									snap.Logger().Warn("VFS CACHE: Error unmarshaling object: %v", err)
								} else {
									object = cachedObject
								}
							}
						}
					}
				}
			}

			// Chunkify the file if it is a regular file and we don't have a cached object
			if record.FileInfo.Mode().IsRegular() {
				if object == nil || !snap.BlobExists(packfile.TYPE_OBJECT, object.Checksum) {
					object, err = snap.chunkify(imp, cf, record)
					if err != nil {
						backupCtx.recordError(record.Pathname, err)
						return
					}

					serializedObject, err := object.Serialize()
					if err != nil {
						backupCtx.recordError(record.Pathname, err)
						return
					}

					if err := vfsCache.PutObject(object.Checksum, serializedObject); err != nil {
						backupCtx.recordError(record.Pathname, err)
						return
					}
				}
			}

			if object != nil {
				if !snap.BlobExists(packfile.TYPE_OBJECT, object.Checksum) {
					data, err := object.Serialize()
					if err != nil {
						backupCtx.recordError(record.Pathname, err)
						return
					}
					err = snap.PutBlob(packfile.TYPE_OBJECT, object.Checksum, data)
					if err != nil {
						backupCtx.recordError(record.Pathname, err)
						return
					}
				}
			}

			var fileEntryChecksum objects.Checksum
			if fileEntry != nil && snap.BlobExists(packfile.TYPE_FILE, cachedFileEntryChecksum) {
				fileEntryChecksum = cachedFileEntryChecksum
			} else {
				fileEntry = vfs.NewFileEntry(path.Dir(record.Pathname), &record)
				if object != nil {
					fileEntry.Object = object
				}

				classifications := cf.Processor(record.Pathname).File(fileEntry)
				for _, result := range classifications {
					fileEntry.AddClassification(result.Analyzer, result.Classes)
				}

				serialized, err := fileEntry.Serialize()
				if err != nil {
					backupCtx.recordError(record.Pathname, err)
					return
				}

				fileEntryChecksum = snap.repository.Checksum(serialized)
				err = snap.PutBlob(packfile.TYPE_FILE, fileEntryChecksum, serialized)
				if err != nil {
					backupCtx.recordError(record.Pathname, err)
					return
				}

				// Store the newly generated FileEntry in the cache for future runs
				err = vfsCache.PutFilename(record.Pathname, serialized)
				if err != nil {
					backupCtx.recordError(record.Pathname, err)
					return
				}

				fileSummary := &vfs.FileSummary{
					Type:    record.Type,
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
					backupCtx.recordError(record.Pathname, err)
					return
				}

				err = vfsCache.PutFileSummary(record.Pathname, seralizedFileSummary)
				if err != nil {
					backupCtx.recordError(record.Pathname, err)
					return
				}
			}

			// Record the checksum of the FileEntry in the cache
			err = sc2.PutChecksum(record.Pathname, fileEntryChecksum)
			if err != nil {
				backupCtx.recordError(record.Pathname, err)
				return
			}
			snap.Event(events.FileOKEvent(snap.Header.Identifier, record.Pathname))
		}(_record)
	}
	scannerWg.Wait()

	errcsum, err := persistIndex(snap, backupCtx.tree, packfile.TYPE_ERROR)
	if err != nil {
		return err
	}

	var rootSummary *vfs.Summary

	directories, err := sc2.EnumerateKeysWithPrefixReverse("__pathname__", true)
	if err != nil {
		return err
	}
	for record := range directories {
		dirEntry := vfs.NewDirectoryEntry(path.Dir(record.Pathname), &record)

		childrenChan, err := sc2.EnumerateImmediateChildPathnames(record.Pathname, true)
		if err != nil {
			return err
		}

		/* children */
		var lastChecksum *objects.Checksum
		for child := range childrenChan {
			childChecksum, err := sc2.GetChecksum(child.Pathname)
			if err != nil {
				continue
			}
			childEntry := &vfs.ChildEntry{
				Lchecksum: childChecksum,
				LfileInfo: child.FileInfo,
			}
			if child.FileInfo.Mode().IsDir() {
				data, err := sc2.GetSummary(child.Pathname)
				if err != nil {
					continue
				}

				childSummary, err := vfs.SummaryFromBytes(data)
				if err != nil {
					continue
				}

				dirEntry.Summary.UpdateBelow(childSummary)
				childEntry.Lsummary = childSummary
			} else {
				data, err := vfsCache.GetFileSummary(child.Pathname)
				if err != nil {
					continue
				}

				fileSummary, err := vfs.FileSummaryFromBytes(data)
				if err != nil {
					continue
				}

				dirEntry.Summary.UpdateWithFileSummary(fileSummary)
			}

			if lastChecksum != nil {
				childEntry.Successor = lastChecksum
			}
			childEntrySerialized, err := childEntry.ToBytes()
			if err != nil {
				continue
			}
			childEntryChecksum := snap.repository.Checksum(childEntrySerialized)
			lastChecksum = &childEntryChecksum

			if !snap.BlobExists(packfile.TYPE_CHILD, childEntryChecksum) {
				if err := snap.PutBlob(packfile.TYPE_CHILD, childEntryChecksum, childEntrySerialized); err != nil {
					continue
				}
			}
			dirEntry.Summary.Directory.Children++
		}
		dirEntry.Children = lastChecksum

		iter, err := backupCtx.tree.ScanFrom(record.Pathname)
		if err != nil {
			return err
		}
		for iter.Next() {
			_, errentry := iter.Current()
			if !strings.HasPrefix(errentry.Name, record.Pathname) {
				break
			}
			dirEntry.Summary.Below.Errors++
		}

		dirEntry.Summary.UpdateAverages()

		classifications := cf.Processor(record.Pathname).Directory(dirEntry)
		for _, result := range classifications {
			dirEntry.AddClassification(result.Analyzer, result.Classes)
		}

		serialized, err := dirEntry.Serialize()
		if err != nil {
			return err
		}
		dirEntryChecksum := snap.repository.Checksum(serialized)

		if !snap.BlobExists(packfile.TYPE_DIRECTORY, dirEntryChecksum) {
			err = snap.PutBlob(packfile.TYPE_DIRECTORY, dirEntryChecksum, serialized)
			if err != nil {
				backupCtx.recordError(record.Pathname, err)
				return err
			}
		}
		err = sc2.PutChecksum(record.Pathname, dirEntryChecksum)
		if err != nil {
			backupCtx.recordError(record.Pathname, err)
			return err
		}

		serializedSummary, err := dirEntry.Summary.ToBytes()
		if err != nil {
			backupCtx.recordError(record.Pathname, err)
			return err
		}

		err = sc2.PutSummary(record.Pathname, serializedSummary)
		if err != nil {
			backupCtx.recordError(record.Pathname, err)
			return err
		}

		snap.Event(events.DirectoryOKEvent(snap.Header.Identifier, record.Pathname))
		if record.Pathname == "/" {
			rootSummary = &dirEntry.Summary
		}
	}

	if backupCtx.aborted.Load() {
		return backupCtx.abortedReason
	}

	value, err := sc2.GetChecksum("/")
	if err != nil {
		return err
	}

	snap.Header.Root = value
	//snap.Header.Metadata = metadataChecksum
	snap.Header.Duration = time.Since(beginTime)
	snap.Header.Summary = *rootSummary
	snap.Header.Errors = errcsum

	/*
		for _, key := range snap.Metadata.ListKeys() {
			objectType := strings.Split(key, ";")[0]
			objectKind := strings.Split(key, "/")[0]
			if objectType == "" {
				objectType = "unknown"
				objectKind = "unknown"
			}
			if _, exists := snap.Header.FileKind[objectKind]; !exists {
				snap.Header.FileKind[objectKind] = 0
			}
			snap.Header.FileKind[objectKind] += uint64(len(snap.Metadata.ListValues(key)))

			if _, exists := snap.Header.FileType[objectType]; !exists {
				snap.Header.FileType[objectType] = 0
			}
			snap.Header.FileType[objectType] += uint64(len(snap.Metadata.ListValues(key)))
		}

		for key, value := range snap.Header.FileType {
			snap.Header.FilePercentType[key] = math.Round((float64(value)/float64(snap.Header.FilesCount)*100)*100) / 100
		}
		for key, value := range snap.Header.FileKind {
			snap.Header.FilePercentKind[key] = math.Round((float64(value)/float64(snap.Header.FilesCount)*100)*100) / 100
		}
		for key, value := range snap.Header.FileExtension {
			snap.Header.FilePercentExtension[key] = math.Round((float64(value)/float64(snap.Header.FilesCount)*100)*100) / 100
		}
	*/
	return snap.Commit()
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

func (snap *Snapshot) chunkify(imp importer.Importer, cf *classifier.Classifier, record importer.ScanRecord) (*objects.Object, error) {
	rd, err := imp.NewReader(record.Pathname)
	if err != nil {
		return nil, err
	}
	defer rd.Close()

	cprocessor := cf.Processor(record.Pathname)

	object := objects.NewObject()
	object.ContentType = mime.TypeByExtension(path.Ext(record.Pathname))

	objectHasher := snap.repository.Hasher()

	var firstChunk = true
	var cdcOffset uint64
	var object_t32 objects.Checksum

	var totalEntropy float64
	var totalFreq [256]float64
	var totalDataSize uint64

	// Helper function to process a chunk
	processChunk := func(data []byte) error {
		var chunk_t32 objects.Checksum
		chunkHasher := snap.repository.Hasher()

		if firstChunk {
			if object.ContentType == "" {
				object.ContentType = mimetype.Detect(data).String()
			}
			firstChunk = false
		}
		objectHasher.Write(data)
		cprocessor.Write(data)

		chunkHasher.Reset()
		chunkHasher.Write(data)
		copy(chunk_t32[:], chunkHasher.Sum(nil))

		entropyScore, freq := entropy(data)
		if len(data) > 0 {
			for i := 0; i < 256; i++ {
				totalFreq[i] += freq[i]
				freq[i] /= float64(len(data))
			}
		}
		chunk := objects.Chunk{Checksum: chunk_t32, Length: uint32(len(data)), Entropy: entropyScore, Distribution: freq}
		object.Chunks = append(object.Chunks, chunk)
		cdcOffset += uint64(len(data))

		totalEntropy += chunk.Entropy * float64(len(data))
		totalDataSize += uint64(len(data))

		if !snap.BlobExists(packfile.TYPE_CHUNK, chunk.Checksum) {
			return snap.PutBlob(packfile.TYPE_CHUNK, chunk.Checksum, data)
		}
		return nil
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
		for i := 0; i < 256; i++ {
			totalFreq[i] /= float64(totalDataSize)
		}
	} else {
		object.Entropy = 0.0
		object.Distribution = [256]float64{}
	}

	copy(object_t32[:], objectHasher.Sum(nil))
	object.Checksum = object_t32

	classifications := cprocessor.Finalize()
	for _, result := range classifications {
		object.AddClassification(result.Analyzer, result.Classes)
	}

	return object, nil
}
