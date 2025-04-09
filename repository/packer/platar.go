package packer

import (
	"context"
	"fmt"
	"hash"
	"runtime"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/caching"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/storage"
	"github.com/PlakarKorp/plakar/versioning"
)

type platarPackerManager struct {
	packingCache *caching.PackingCache

	packerChan     chan interface{}
	packerChanDone chan struct{}

	storageConf *storage.Configuration
	hashFactory func() hash.Hash
	appCtx      *appcontext.AppContext

	// XXX: Temporary hack callback-based to ease the transition diff.
	// To be revisited with either an interface or moving this file inside repository/
	flush func(*Packer) error
}

func NewPlatarPackerManager(ctx *appcontext.AppContext, storageConfiguration *storage.Configuration, hashFactory func() hash.Hash, flusher func(*Packer) error) (PackerManagerInt, error) {
	cache, err := ctx.GetCache().Packing()
	if err != nil {
		return nil, err
	}

	return &platarPackerManager{
		packingCache:   cache,
		packerChan:     make(chan interface{}, runtime.NumCPU()*2+1),
		packerChanDone: make(chan struct{}),
		storageConf:    storageConfiguration,
		hashFactory:    hashFactory,
		appCtx:         ctx,
		flush:          flusher,
	}, nil
}

func (mgr *platarPackerManager) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	packerResultChan := make(chan *Packer, runtime.NumCPU())

	packer := NewPacker(mgr.hashFactory())
	workerGroup, workerCtx := errgroup.WithContext(ctx)
	for i := 0; i < 1; i++ {
		workerGroup.Go(func() error {
			for {
				select {
				case <-workerCtx.Done():
					return workerCtx.Err()
				case msg, ok := <-mgr.packerChan:
					if !ok {
						if packer != nil && packer.size() > 0 {
							packerResultChan <- packer
						}
						return nil
					}

					pm, ok := msg.(*PackerMsg)
					if !ok {
						return fmt.Errorf("unexpected message type")
					}

					if !packer.addBlobIfNotExists(pm.Type, pm.Version, pm.MAC, pm.Data, pm.Flags) {
						continue
					}
				}
			}
		})
	}

	// Wait for workers to finish.
	if err := workerGroup.Wait(); err != nil {
		mgr.appCtx.GetLogger().Error("Worker group error: %s", err)
		cancel() // Propagate cancellation.
	}

	flusherGroup, _ := errgroup.WithContext(ctx)
	flusherGroup.Go(func() error {
		if packer.size() == 0 {
			return nil
		}

		if err := mgr.flush(packer); err != nil {
			return fmt.Errorf("failed to flush packer: %w", err)
		}

		return nil
	})

	// Close the result channel and wait for the flusher to finish.
	close(packerResultChan)
	if err := flusherGroup.Wait(); err != nil {
		mgr.appCtx.GetLogger().Error("Flusher group error: %s", err)
	}

	// Signal completion.
	mgr.packerChanDone <- struct{}{}
	close(mgr.packerChanDone)

	mgr.packingCache.Close()
}

func (mgr *platarPackerManager) Wait() {
	close(mgr.packerChan)
	<-mgr.packerChanDone
}

func (mgr *platarPackerManager) InsertIfNotPresent(Type resources.Type, mac objects.MAC) (bool, error) {
	// XXX: This is not atomic, as such it leaves a possibility of missed dedup (unlikely though). Needs to be fixed by using a batch.
	has, err := mgr.packingCache.HasBlob(Type, mac)
	if err != nil {
		return false, err
	}

	if has {
		return true, nil
	}

	if err := mgr.packingCache.PutBlob(Type, mac); err != nil {
		return false, err
	}

	return false, nil
}

func (mgr *platarPackerManager) Put(Type resources.Type, mac objects.MAC, data []byte) error {
	mgr.packerChan <- &PackerMsg{Type: Type, Version: versioning.GetCurrentVersion(Type), Timestamp: time.Now(), MAC: mac, Data: data}
	return nil
}

func (mgr *platarPackerManager) Exists(Type resources.Type, mac objects.MAC) (bool, error) {
	return mgr.packingCache.HasBlob(Type, mac)
}
