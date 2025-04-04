package repository

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/PlakarKorp/plakar/caching"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/repository/packer"
	"github.com/PlakarKorp/plakar/repository/state"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/storage"
	"github.com/PlakarKorp/plakar/versioning"
)

type RepositoryWriter struct {
	*Repository

	transactionMtx sync.RWMutex
	deltaState     *state.LocalState

	PackerManager  *packer.PackerManager
	currentStateID objects.MAC
}

func (r *Repository) newRepositoryWriter(cache *caching.ScanCache, id objects.MAC) *RepositoryWriter {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "NewRepositoryWriter(): %s", time.Since(t0))
	}()

	rw := RepositoryWriter{
		Repository:     r,
		deltaState:     r.state.Derive(cache),
		currentStateID: id,
	}

	rw.PackerManager = packer.NewPackerManager(rw.AppContext(), &rw.configuration, rw.GetMACHasher, rw.PutPackfile)

	// XXX: Better placement for this
	go rw.PackerManager.Run()

	return &rw
}

func (r *RepositoryWriter) FlushTransaction(newCache *caching.ScanCache, id objects.MAC) error {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repositorywriter", "FlushTransaction(): %s", time.Since(t0))
	}()

	r.transactionMtx.Lock()
	oldState := r.deltaState
	r.deltaState = r.state.Derive(newCache)
	r.transactionMtx.Unlock()

	return r.internalCommit(oldState, id)
}

func (r *RepositoryWriter) CommitTransaction(id objects.MAC) error {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "CommitTransaction(): %s", time.Since(t0))
	}()

	err := r.internalCommit(r.deltaState, id)
	r.transactionMtx.Lock()
	r.deltaState = nil
	r.transactionMtx.Unlock()

	return err
}

func (r *RepositoryWriter) internalCommit(state *state.LocalState, id objects.MAC) error {
	pr, pw := io.Pipe()

	/* By using a pipe and a goroutine we bound the max size in memory. */
	go func() {
		defer pw.Close()

		if err := state.SerializeToStream(pw); err != nil {
			pw.CloseWithError(err)
		}
	}()

	err := r.PutState(id, pr)
	if err != nil {
		return err
	}

	/* We are commiting the transaction, publish the new state to our local aggregated state. */
	return r.state.PutState(id)
}

func (r *RepositoryWriter) BlobExists(Type resources.Type, mac objects.MAC) bool {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repositorywriter", "BlobExists(%s, %x): %s", Type, mac, time.Since(t0))
	}()

	if _, exists := r.PackerManager.InflightMACs[Type].Load(mac); exists {
		return true
	}

	return r.state.BlobExists(Type, mac)
}

func (r *RepositoryWriter) PutBlobIfNotExists(Type resources.Type, mac objects.MAC, data []byte) error {
	if r.BlobExists(Type, mac) {
		return nil
	}
	return r.PutBlob(Type, mac, data)
}

func (r *RepositoryWriter) PutBlob(Type resources.Type, mac objects.MAC, data []byte) error {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repositorywriter", "PutBlob(%s, %x): %s", Type, mac, time.Since(t0))
	}()

	if _, exists := r.PackerManager.InflightMACs[Type].LoadOrStore(mac, struct{}{}); exists {
		// tell prom exporter that we collided a blob
		return nil
	}

	encodedReader, err := r.Encode(bytes.NewReader(data))
	if err != nil {
		return err
	}

	encoded, err := io.ReadAll(encodedReader)
	if err != nil {
		return err
	}

	r.PackerManager.PutBlob(Type, mac, encoded)

	return nil
}

func (r *RepositoryWriter) DeleteStateResource(Type resources.Type, mac objects.MAC) error {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "DeleteStateResource(%s, %x): %s", Type.String(), mac, time.Since(t0))
	}()

	r.transactionMtx.RLock()
	defer r.transactionMtx.RUnlock()
	return r.deltaState.DeleteResource(Type, mac)
}

func (r *RepositoryWriter) PutPackfile(packer *packer.Packer) error {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "PutPackfile(%x): %s", r.currentStateID, time.Since(t0))
	}()

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

	encryptedIndex, err := r.EncodeBuffer(serializedIndex)
	if err != nil {
		return err
	}

	encryptedFooter, err := r.EncodeBuffer(serializedFooter)
	if err != nil {
		return err
	}

	serializedPackfile := append(serializedData, encryptedIndex...)
	serializedPackfile = append(serializedPackfile, encryptedFooter...)

	/* it is necessary to track the footer _encrypted_ length */
	encryptedFooterLength := make([]byte, 4)
	binary.LittleEndian.PutUint32(encryptedFooterLength, uint32(len(encryptedFooter)))
	serializedPackfile = append(serializedPackfile, encryptedFooterLength...)

	mac := r.ComputeMAC(serializedPackfile)

	rd, err := storage.Serialize(r.GetMACHasher(), resources.RT_PACKFILE, versioning.GetCurrentVersion(resources.RT_PACKFILE), bytes.NewBuffer(serializedPackfile))
	if err != nil {
		return err
	}

	if err = r.store.PutPackfile(mac, rd); err != nil {
		return err
	}

	if r.deltaState == nil {
		panic("Put outside of transaction")
	}

	r.transactionMtx.RLock()
	defer r.transactionMtx.RUnlock()
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

					if err := r.deltaState.PutDelta(delta); err != nil {
						return err
					}

					if err := r.state.PutDelta(delta); err != nil {
						return err
					}
				}
			}
		}
	}

	if err := r.deltaState.PutPackfile(r.currentStateID, mac); err != nil {
		return err
	}

	return r.state.PutPackfile(r.currentStateID, mac)
}
