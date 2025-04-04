package snapshot

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"hash"
	"io"
	"math/big"
	"runtime"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/packfile"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/versioning"
)

func init() {
	// used for paddinng random bytes
	versioning.Register(resources.RT_RANDOM, versioning.FromString("1.0.0"))
}

type PackerManager struct {
	snapshot       *Snapshot
	inflightMACs   map[resources.Type]*sync.Map
	packerChan     chan interface{}
	packerChanDone chan struct{}
}

func NewPackerManager(snapshot *Snapshot) *PackerManager {
	inflightsMACs := make(map[resources.Type]*sync.Map)
	for _, Type := range resources.Types() {
		inflightsMACs[Type] = &sync.Map{}
	}
	return &PackerManager{
		snapshot:       snapshot,
		inflightMACs:   inflightsMACs,
		packerChan:     make(chan interface{}, runtime.NumCPU()*2+1),
		packerChanDone: make(chan struct{}),
	}
}

func (mgr *PackerManager) Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	packerResultChan := make(chan *Packer, runtime.NumCPU())

	flusherGroup, _ := errgroup.WithContext(ctx)
	flusherGroup.Go(func() error {
		for packer := range packerResultChan {
			if packer == nil || packer.Size() == 0 {
				continue
			}

			packer.AddPadding(int(mgr.snapshot.repository.Configuration().Chunking.MinSize))

			if err := mgr.snapshot.PutPackfile(packer); err != nil {
				return fmt.Errorf("failed to flush packer: %w", err)
			}

			for _, record := range packer.Packfile.Index {
				mgr.inflightMACs[record.Type].Delete(record.MAC)
			}
		}
		return nil
	})

	workerGroup, workerCtx := errgroup.WithContext(ctx)
	for i := 0; i < runtime.NumCPU(); i++ {
		workerGroup.Go(func() error {
			var packer *Packer

			for {
				select {
				case <-workerCtx.Done():
					return workerCtx.Err()
				case msg, ok := <-mgr.packerChan:
					if !ok {
						if packer != nil && packer.Size() > 0 {
							packerResultChan <- packer
						}
						return nil
					}

					pm, ok := msg.(*PackerMsg)
					if !ok {
						return fmt.Errorf("unexpected message type")
					}

					if packer == nil {
						packer = NewPacker(mgr.snapshot.Repository().GetMACHasher())
						packer.AddPadding(int(mgr.snapshot.repository.Configuration().Chunking.MinSize))
					}

					if !packer.AddBlobIfNotExists(pm.Type, pm.Version, pm.MAC, pm.Data, pm.Flags) {
						continue
					}

					if packer.Size() > uint32(mgr.snapshot.repository.Configuration().Packfile.MaxSize) {
						packerResultChan <- packer
						packer = nil
					}
				}
			}
		})
	}

	// Wait for workers to finish.
	if err := workerGroup.Wait(); err != nil {
		mgr.snapshot.Logger().Error("Worker group error: %s", err)
		cancel() // Propagate cancellation.
	}

	// Close the result channel and wait for the flusher to finish.
	close(packerResultChan)
	if err := flusherGroup.Wait(); err != nil {
		mgr.snapshot.Logger().Error("Flusher group error: %s", err)
	}

	// Signal completion.
	mgr.packerChanDone <- struct{}{}
	close(mgr.packerChanDone)
}

func (mgr *PackerManager) Wait() {
	close(mgr.packerChan)
	<-mgr.packerChanDone
}

type PackerMsg struct {
	Timestamp time.Time
	Type      resources.Type
	Version   versioning.Version
	MAC       objects.MAC
	Data      []byte
	Flags     uint32
}

type Packer struct {
	Blobs    map[resources.Type]map[[32]byte][]byte
	Packfile *packfile.PackFile
}

func NewPacker(hasher hash.Hash) *Packer {
	blobs := make(map[resources.Type]map[[32]byte][]byte)
	for _, Type := range resources.Types() {
		blobs[Type] = make(map[[32]byte][]byte)
	}
	return &Packer{
		Packfile: packfile.New(hasher),
		Blobs:    blobs,
	}
}

func (packer *Packer) AddPadding(maxSize int) error {
	if maxSize < 0 {
		return fmt.Errorf("invalid padding size")
	}
	if maxSize == 0 {
		return nil
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(maxSize)-1))
	if err != nil {
		return err
	}
	paddingSize := uint32(n.Uint64()) + 1

	buffer := make([]byte, paddingSize)
	_, err = rand.Read(buffer)
	if err != nil {
		return fmt.Errorf("failed to generate random padding: %w", err)
	}

	mac := objects.MAC{}
	_, err = rand.Read(mac[:])
	if err != nil {
		return fmt.Errorf("failed to generate random padding MAC: %w", err)
	}

	packer.AddBlobIfNotExists(resources.RT_RANDOM, versioning.GetCurrentVersion(resources.RT_RANDOM), mac, buffer, 0)

	return nil
}

func (packer *Packer) AddBlobIfNotExists(Type resources.Type, version versioning.Version, mac [32]byte, data []byte, flags uint32) bool {
	if _, ok := packer.Blobs[Type]; !ok {
		packer.Blobs[Type] = make(map[[32]byte][]byte)
	}
	if _, ok := packer.Blobs[Type][mac]; ok {
		return false
	}
	packer.Blobs[Type][mac] = data
	packer.Packfile.AddBlob(Type, version, mac, data, flags)
	return true
}

func (packer *Packer) Size() uint32 {
	return packer.Packfile.Size()
}

func (packer *Packer) Types() []resources.Type {
	ret := make([]resources.Type, 0, len(packer.Blobs))
	for k := range packer.Blobs {
		ret = append(ret, k)
	}
	return ret
}

func (snap *Snapshot) PutBlob(Type resources.Type, mac [32]byte, data []byte) error {
	snap.Logger().Trace("snapshot", "%x: PutBlob(%s, %064x) len=%d", snap.Header.GetIndexShortID(), Type, mac, len(data))

	if snap.repository.InTransaction() {
		if _, exists := snap.packerManager.inflightMACs[Type].LoadOrStore(mac, struct{}{}); exists {
			// tell prom exporter that we collided a blob
			return nil
		}
	}

	encodedReader, err := snap.repository.Encode(bytes.NewReader(data))
	if err != nil {
		return err
	}

	encoded, err := io.ReadAll(encodedReader)
	if err != nil {
		return err
	}

	snap.packerManager.packerChan <- &PackerMsg{Type: Type, Version: versioning.GetCurrentVersion(Type), Timestamp: time.Now(), MAC: mac, Data: encoded}
	return nil
}

func (snap *Snapshot) GetBlob(Type resources.Type, mac [32]byte) ([]byte, error) {
	snap.Logger().Trace("snapshot", "%x: GetBlob(%s, %x)", snap.Header.GetIndexShortID(), Type, mac)
	rd, err := snap.repository.GetBlob(Type, mac)
	if err != nil {
		return nil, err
	}

	return io.ReadAll(rd)
}

func (snap *Snapshot) BlobExists(Type resources.Type, mac [32]byte) bool {
	snap.Logger().Trace("snapshot", "%x: CheckBlob(%s, %064x)", snap.Header.GetIndexShortID(), Type, mac)

	if snap.repository.InTransaction() {
		if _, exists := snap.packerManager.inflightMACs[Type].Load(mac); exists {
			return true
		}
	}

	return snap.repository.BlobExists(Type, mac)
}

func (snap *Snapshot) PutBlobIfNotExists(Type resources.Type, mac [32]byte, data []byte) error {
	snap.Logger().Trace("snapshot", "%x: PutBlobIfNotExists(%s, %064x) len=%d", snap.Header.GetIndexShortID(), Type, mac, len(data))
	if snap.BlobExists(Type, mac) {
		return nil
	}
	return snap.PutBlob(Type, mac, data)
}
