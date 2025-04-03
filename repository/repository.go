package repository

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"io"
	"iter"
	"math/big"
	"math/bits"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	chunkers "github.com/PlakarKorp/go-cdc-chunkers"
	_ "github.com/PlakarKorp/go-cdc-chunkers/chunkers/fastcdc"
	_ "github.com/PlakarKorp/go-cdc-chunkers/chunkers/ultracdc"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/caching"
	"github.com/PlakarKorp/plakar/compression"
	"github.com/PlakarKorp/plakar/encryption"
	"github.com/PlakarKorp/plakar/hashing"
	"github.com/PlakarKorp/plakar/logging"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/packfile"
	"github.com/PlakarKorp/plakar/repository/packer"
	"github.com/PlakarKorp/plakar/repository/state"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/storage"
	"github.com/PlakarKorp/plakar/versioning"
	"github.com/google/uuid"
)

var (
	ErrPackfileNotFound = errors.New("packfile not found")
	ErrBlobNotFound     = errors.New("blob not found")
	ErrNotReadable      = errors.New("repository is not readable")
)

type Repository struct {
	store         storage.Store
	state         *state.LocalState
	configuration storage.Configuration
	appContext    *appcontext.AppContext

	wBytes atomic.Int64
	rBytes atomic.Int64

	storageSize      int64
	storageSizeDirty bool
	transactionMtx   sync.RWMutex
	deltaState       *state.LocalState

	PackerManager  *packer.PackerManager
	currentStateID objects.MAC
}

func Inexistent(ctx *appcontext.AppContext, storeConfig map[string]string) (*Repository, error) {
	st, err := storage.New(storeConfig)
	if err != nil {
		return nil, err
	}
	defer st.Close()

	return &Repository{
		store:            st,
		configuration:    *storage.NewConfiguration(),
		appContext:       ctx,
		storageSize:      -1,
		storageSizeDirty: true,
	}, nil
}

func New(ctx *appcontext.AppContext, store storage.Store, config []byte) (*Repository, error) {
	t0 := time.Now()
	defer func() {
		ctx.GetLogger().Trace("repository", "New(store=%p): %s", store, time.Since(t0))
	}()

	var hasher hash.Hash
	if ctx.GetSecret() != nil {
		hasher = hashing.GetMACHasher(storage.DEFAULT_HASHING_ALGORITHM, ctx.GetSecret())
	} else {
		hasher = hashing.GetHasher(storage.DEFAULT_HASHING_ALGORITHM)
	}

	version, unwrappedConfigRd, err := storage.Deserialize(hasher, resources.RT_CONFIG, bytes.NewReader(config))
	if err != nil {
		return nil, err
	}

	unwrappedConfig, err := io.ReadAll(unwrappedConfigRd)
	if err != nil {
		return nil, err
	}

	configInstance, err := storage.NewConfigurationFromBytes(version, unwrappedConfig)
	if err != nil {
		return nil, err
	}

	r := &Repository{
		store:            store,
		configuration:    *configInstance,
		appContext:       ctx,
		storageSize:      -1,
		storageSizeDirty: true,
	}

	if err := r.RebuildState(); err != nil {
		return nil, err
	}

	cacheInstance, err := r.AppContext().GetCache().Repository(r.Configuration().RepositoryID)
	if err != nil {
		return nil, err
	}

	clientVersion := r.appContext.Client
	if !cacheInstance.HasCookie(clientVersion) {
		if err := cacheInstance.PutCookie(clientVersion); err != nil {
			return nil, err
		}
	}

	return r, nil
}

func NewNoRebuild(ctx *appcontext.AppContext, store storage.Store, config []byte) (*Repository, error) {
	t0 := time.Now()
	defer func() {
		ctx.GetLogger().Trace("repository", "NewNoRebuild(store=%p): %s", store, time.Since(t0))
	}()

	var hasher hash.Hash
	if ctx.GetSecret() != nil {
		hasher = hashing.GetMACHasher(storage.DEFAULT_HASHING_ALGORITHM, ctx.GetSecret())
	} else {
		hasher = hashing.GetHasher(storage.DEFAULT_HASHING_ALGORITHM)
	}

	version, unwrappedConfigRd, err := storage.Deserialize(hasher, resources.RT_CONFIG, bytes.NewReader(config))
	if err != nil {
		return nil, err
	}

	unwrappedConfig, err := io.ReadAll(unwrappedConfigRd)
	if err != nil {
		return nil, err
	}

	configInstance, err := storage.NewConfigurationFromBytes(version, unwrappedConfig)
	if err != nil {
		return nil, err
	}

	r := &Repository{
		store:            store,
		configuration:    *configInstance,
		appContext:       ctx,
		storageSize:      -1,
		storageSizeDirty: true,
	}

	return r, nil
}

func (r *Repository) RebuildState() error {
	cacheInstance, err := r.AppContext().GetCache().Repository(r.Configuration().RepositoryID)
	if err != nil {
		return err
	}

	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "rebuildState(): %s", time.Since(t0))
	}()

	/* Use on-disk local state, and merge it with repository's own state */
	aggregatedState := state.NewLocalState(cacheInstance)

	// identify local states
	localStates, err := cacheInstance.GetStates()
	if err != nil {
		return err
	}

	// identify remote states
	remoteStates, err := r.GetStates()
	if err != nil {
		return err
	}

	remoteStatesMap := make(map[objects.MAC]struct{})
	for _, stateID := range remoteStates {
		remoteStatesMap[stateID] = struct{}{}
	}

	// build delta of local and remote states
	localStatesMap := make(map[objects.MAC]struct{})
	outdatedStates := make([]objects.MAC, 0)
	for stateID := range localStates {
		localStatesMap[stateID] = struct{}{}

		if _, exists := remoteStatesMap[stateID]; !exists {
			outdatedStates = append(outdatedStates, stateID)
		}
	}

	missingStates := make([]objects.MAC, 0)
	for _, stateID := range remoteStates {
		if _, exists := localStatesMap[stateID]; !exists {
			missingStates = append(missingStates, stateID)
		}
	}

	rebuilt := false
	for _, stateID := range missingStates {
		version, remoteStateRd, err := r.GetState(stateID)
		if err != nil {
			return err
		}

		if err := aggregatedState.MergeState(version, stateID, remoteStateRd); err != nil {
			return err
		}
		rebuilt = true
	}

	// delete local states that are not present in remote
	for _, stateID := range outdatedStates {
		if err := aggregatedState.DelState(stateID); err != nil {
			return err
		}
		rebuilt = true
	}

	r.state = aggregatedState

	// The first Serial id is our repository ID, this allows us to deal
	// naturally with concurrent first backups.
	r.state.UpdateSerialOr(r.configuration.RepositoryID)

	if rebuilt {
		r.storageSizeDirty = true
	}

	return nil
}

func (r *Repository) AppContext() *appcontext.AppContext {
	return r.appContext
}

func (r *Repository) Store() storage.Store {
	return r.store
}

func (r *Repository) StorageSize() int64 {
	if r.storageSizeDirty {
		r.storageSize = r.store.Size()
		r.storageSizeDirty = false
	}
	return r.storageSize
}

func (r *Repository) RBytes() int64 {
	return r.rBytes.Load()
}

func (r *Repository) WBytes() int64 {
	return r.wBytes.Load()
}

func (r *Repository) Close() error {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "Close(): %s", time.Since(t0))
	}()
	return nil
}

func (r *Repository) Decode(input io.Reader) (io.Reader, error) {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "Decode: %s", time.Since(t0))
	}()

	stream := input
	if r.AppContext().GetSecret() != nil {
		tmp, err := encryption.DecryptStream(r.configuration.Encryption, r.AppContext().GetSecret(), stream)
		if err != nil {
			return nil, err
		}
		stream = tmp
	}

	if r.configuration.Compression != nil {
		tmp, err := compression.InflateStream(r.configuration.Compression.Algorithm, stream)
		if err != nil {
			return nil, err
		}
		stream = tmp
	}

	return stream, nil
}

func (r *Repository) Encode(input io.Reader) (io.Reader, error) {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "Encode: %s", time.Since(t0))
	}()

	stream := input
	if r.configuration.Compression != nil {
		tmp, err := compression.DeflateStream(r.configuration.Compression.Algorithm, stream)
		if err != nil {
			return nil, err
		}
		stream = tmp
	}

	if r.AppContext().GetSecret() != nil {
		tmp, err := encryption.EncryptStream(r.configuration.Encryption, r.AppContext().GetSecret(), stream)
		if err != nil {
			return nil, err
		}
		stream = tmp
	}

	return stream, nil
}

func (r *Repository) DecodeBuffer(buffer []byte) ([]byte, error) {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "Decode(%d bytes): %s", len(buffer), time.Since(t0))
	}()

	rd, err := r.Decode(bytes.NewBuffer(buffer))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(rd)
}

func (r *Repository) EncodeBuffer(buffer []byte) ([]byte, error) {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "Encode(%d): %s", len(buffer), time.Since(t0))
	}()

	rd, err := r.Encode(bytes.NewBuffer(buffer))
	if err != nil {
		return nil, err
	}
	return io.ReadAll(rd)
}

func (r *Repository) GetMACHasher() hash.Hash {
	secret := r.AppContext().GetSecret()
	if secret == nil {
		// unencrypted repo, derive 32-bytes "secret" from RepositoryID
		// so ComputeMAC can be used similarly to encrypted repos
		hasher := hashing.GetHasher(r.Configuration().Hashing.Algorithm)
		hasher.Write(r.configuration.RepositoryID[:])
		secret = hasher.Sum(nil)
	}
	return hashing.GetMACHasher(r.Configuration().Hashing.Algorithm, secret)
}

func (r *Repository) ComputeMAC(data []byte) objects.MAC {
	hasher := r.GetMACHasher()
	hasher.Write(data)
	result := hasher.Sum(nil)

	if len(result) != 32 {
		panic("hasher returned invalid length")
	}

	var mac objects.MAC
	copy(mac[:], result)

	return mac
}

func (r *Repository) Chunker(rd io.ReadCloser) (*chunkers.Chunker, error) {
	chunkingAlgorithm := r.configuration.Chunking.Algorithm
	chunkingMinSize := r.configuration.Chunking.MinSize
	chunkingNormalSize := r.configuration.Chunking.NormalSize
	chunkingMaxSize := r.configuration.Chunking.MaxSize

	return chunkers.NewChunker(strings.ToLower(chunkingAlgorithm), rd, &chunkers.ChunkerOpts{
		MinSize:    int(chunkingMinSize),
		NormalSize: int(chunkingNormalSize),
		MaxSize:    int(chunkingMaxSize),
	})
}

func (r *Repository) StartPackerManager(stateID objects.MAC) {
	r.PackerManager = packer.NewPackerManager(r.AppContext(), &r.configuration, r.GetMACHasher, r.PutPackfile)
	r.currentStateID = stateID
	go r.PackerManager.Run()
}

func (r *Repository) StartTransaction(cache *caching.ScanCache) {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "StartTransaction(): %s", time.Since(t0))
	}()
	r.transactionMtx.Lock()
	defer r.transactionMtx.Unlock()
	r.deltaState = r.state.Derive(cache)
}

func (r *Repository) FlushTransaction(newCache *caching.ScanCache, id objects.MAC) error {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "FlushTransaction(): %s", time.Since(t0))
	}()

	r.transactionMtx.Lock()
	oldState := r.deltaState
	r.deltaState = r.state.Derive(newCache)
	r.transactionMtx.Unlock()

	return r.internalCommit(oldState, id)
}

func (r *Repository) CommitTransaction(id objects.MAC) error {
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

func (r *Repository) internalCommit(state *state.LocalState, id objects.MAC) error {
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

func (r *Repository) InTransaction() bool {
	return r.deltaState != nil
}

func (r *Repository) Location() string {
	return r.store.Location()
}

func (r *Repository) Configuration() storage.Configuration {
	return r.configuration
}

func (r *Repository) GetSnapshots() ([]objects.MAC, error) {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "GetSnapshots(): %s", time.Since(t0))
	}()

	ret := make([]objects.MAC, 0)
	for snapshotID := range r.state.ListSnapshots() {
		ret = append(ret, snapshotID)
	}
	return ret, nil
}

func (r *Repository) DeleteSnapshot(snapshotID objects.MAC) error {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "DeleteSnapshot(%x): %s", snapshotID, time.Since(t0))
	}()

	identifier := objects.RandomMAC()
	sc, err := r.AppContext().GetCache().Scan(identifier)
	if err != nil {
		return err
	}
	deltaState := r.state.Derive(sc)

	ret := deltaState.DeleteResource(resources.RT_SNAPSHOT, snapshotID)
	if ret != nil {
		return ret
	}

	buffer := &bytes.Buffer{}
	err = deltaState.SerializeToStream(buffer)
	if err != nil {
		return err
	}

	mac := r.ComputeMAC(buffer.Bytes())
	if err := r.PutState(mac, buffer); err != nil {
		return err
	}

	return nil
}

func (r *Repository) GetStates() ([]objects.MAC, error) {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "GetStates(): %s", time.Since(t0))
	}()

	return r.store.GetStates()
}

func (r *Repository) GetState(mac objects.MAC) (versioning.Version, io.Reader, error) {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "GetState(%x): %s", mac, time.Since(t0))
	}()

	rd, err := r.store.GetState(mac)
	if err != nil {
		return versioning.Version(0), nil, err
	}

	version, rd, err := storage.Deserialize(r.GetMACHasher(), resources.RT_STATE, rd)
	if err != nil {
		return versioning.Version(0), nil, err
	}

	rd, err = r.Decode(rd)
	if err != nil {
		return versioning.Version(0), nil, err
	}
	return version, rd, err
}

func (r *Repository) PutState(mac objects.MAC, rd io.Reader) error {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "PutState(%x, ...): %s", mac, time.Since(t0))
	}()

	rd, err := r.Encode(rd)
	if err != nil {
		return err
	}

	rd, err = storage.Serialize(r.GetMACHasher(), resources.RT_STATE, versioning.GetCurrentVersion(resources.RT_STATE), rd)
	if err != nil {
		return err
	}

	nbytes, err := r.store.PutState(mac, rd)
	r.wBytes.Add(nbytes)
	return err
}

func (r *Repository) DeleteState(mac objects.MAC) error {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "DeleteState(%x, ...): %s", mac, time.Since(t0))
	}()

	return r.store.DeleteState(mac)
}

func (r *Repository) GetPackfiles() ([]objects.MAC, error) {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "GetPackfiles(): %s", time.Since(t0))
	}()

	return r.store.GetPackfiles()
}

func (r *Repository) GetPackfile(mac objects.MAC) (*packfile.PackFile, error) {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "GetPackfile(%x, ...): %s", mac, time.Since(t0))
	}()

	hasher := r.GetMACHasher()

	rd, err := r.store.GetPackfile(mac)
	if err != nil {
		return nil, err
	}

	packfileVersion, rd, err := storage.Deserialize(hasher, resources.RT_PACKFILE, rd)
	if err != nil {
		return nil, err
	}

	rawPackfile, err := io.ReadAll(rd)
	if err != nil {
		return nil, err
	}

	footerBufLength := binary.LittleEndian.Uint32(rawPackfile[len(rawPackfile)-4:])
	rawPackfile = rawPackfile[:len(rawPackfile)-4]

	footerbuf := rawPackfile[len(rawPackfile)-int(footerBufLength):]
	rawPackfile = rawPackfile[:len(rawPackfile)-int(footerBufLength)]

	footerbuf, err = r.DecodeBuffer(footerbuf)
	if err != nil {
		return nil, err
	}

	footer, err := packfile.NewFooterFromBytes(packfileVersion, footerbuf)
	if err != nil {
		return nil, err
	}

	indexbuf := rawPackfile[int(footer.IndexOffset):]
	rawPackfile = rawPackfile[:int(footer.IndexOffset)]

	indexbuf, err = r.DecodeBuffer(indexbuf)
	if err != nil {
		return nil, err
	}

	hasher.Reset()
	hasher.Write(indexbuf)

	if !bytes.Equal(hasher.Sum(nil), footer.IndexMAC[:]) {
		return nil, fmt.Errorf("packfile: index MAC mismatch")
	}

	rawPackfile = append(rawPackfile, indexbuf...)
	rawPackfile = append(rawPackfile, footerbuf...)

	hasher.Reset()
	p, err := packfile.NewFromBytes(hasher, packfileVersion, rawPackfile)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func padmeLength(L uint32) (uint32, error) {
	// Determine the bit-length of L.
	bitLen := 32 - bits.LeadingZeros32(L)

	// Compute overhead as 2^(floor(bitLen/2)).
	overhead := uint32(1 << (bitLen / 2))

	// Generate a random number r in [0, overhead)
	rBig, err := rand.Int(rand.Reader, big.NewInt(int64(overhead)))
	if err != nil {
		return 0, err
	}
	r := uint32(rBig.Int64())

	return r, nil
}

func randomShift(n uint32) (uint64, error) {
	if n == 0 {
		return 0, nil
	}

	max := big.NewInt(int64(n) + 1)
	r, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0, err
	}

	return uint64(r.Int64()), nil
}

func (r *Repository) GetPackfileBlob(loc state.Location) (io.ReadSeeker, error) {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "GetPackfileBlob(%x, %d, %d): %s", loc.Packfile, loc.Offset, loc.Length, time.Since(t0))
	}()

	offset := loc.Offset
	length := loc.Length

	overhead, err := padmeLength(length)
	if err != nil {
		return nil, err
	}

	offsetDelta, err := randomShift(overhead)
	if err != nil {
		return nil, err
	}
	if offsetDelta > offset {
		offsetDelta = offset
	}
	lengthDelta := uint32(uint64(overhead) - offsetDelta)

	rd, err := r.store.GetPackfileBlob(loc.Packfile, offset+uint64(storage.STORAGE_HEADER_SIZE)-offsetDelta, length+uint32(offsetDelta)+lengthDelta)
	if err != nil {
		return nil, err
	}

	// discard the first offsetDelta bytes
	_, err = io.ReadFull(rd, make([]byte, offsetDelta))
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(rd)
	if err != nil {
		return nil, err
	}

	// discard the last lengthDelta bytes
	data = data[:length]

	decoded, err := r.DecodeBuffer(data)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(decoded), nil
}

func (r *Repository) PutPackfile(packer *packer.Packer) error {
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

	nbytes, err := r.store.PutPackfile(mac, rd)
	r.wBytes.Add(nbytes)
	if err != nil {
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

// Deletes a packfile from the store. Warning this is a true delete and is unrecoverable.
func (r *Repository) DeletePackfile(mac objects.MAC) error {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "DeletePackfile(%x): %s", mac, time.Since(t0))
	}()

	return r.store.DeletePackfile(mac)
}

// Removes the packfile from the state, making it unreachable.
func (r *Repository) RemovePackfile(packfileMAC objects.MAC) error {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "RemovePackfile(%x): %s", packfileMAC, time.Since(t0))
	}()
	return r.state.DelPackfile(packfileMAC)
}

func (r *Repository) HasDeletedPackfile(mac objects.MAC) (bool, error) {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "HasDeletedPackfile(%x): %s", mac, time.Since(t0))
	}()

	return r.state.HasDeletedResource(resources.RT_PACKFILE, mac)
}

func (r *Repository) ListDeletedPackfiles() iter.Seq2[objects.MAC, time.Time] {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "ListDeletedPackfiles(): %s", time.Since(t0))
	}()

	return func(yield func(objects.MAC, time.Time) bool) {
		for snap, err := range r.state.ListDeletedResources(resources.RT_PACKFILE) {

			if err != nil {
				r.Logger().Error("Failed to fetch deleted packfile %s", err)
			}

			if !yield(snap.Blob, snap.When) {
				return
			}
		}
	}
}

func (r *Repository) ListDeletedSnapShots() iter.Seq2[objects.MAC, time.Time] {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "ListDeletedSnapShots(): %s", time.Since(t0))
	}()

	return func(yield func(objects.MAC, time.Time) bool) {
		for snap, err := range r.state.ListDeletedResources(resources.RT_SNAPSHOT) {

			if err != nil {
				r.Logger().Error("Failed to fetch deleted snapshot %s", err)
			}

			if !yield(snap.Blob, snap.When) {
				return
			}
		}
	}
}

// Removes the deleted packfile entry from the state.
func (r *Repository) RemoveDeletedPackfile(packfileMAC objects.MAC) error {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "RemoveDeletedPackfile(%x): %s", packfileMAC, time.Since(t0))
	}()

	return r.state.DelDeletedResource(resources.RT_PACKFILE, packfileMAC)
}

func (r *Repository) GetPackfileForBlob(Type resources.Type, mac objects.MAC) (objects.MAC, bool, error) {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "GetPackfileForBlob(%x): %s", mac, time.Since(t0))
	}()

	packfile, exists, err := r.state.GetSubpartForBlob(Type, mac)

	return packfile.Packfile, exists, err
}

func (r *Repository) GetBlob(Type resources.Type, mac objects.MAC) (io.ReadSeeker, error) {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "GetBlob(%s, %x): %s", Type, mac, time.Since(t0))
	}()

	loc, exists, err := r.state.GetSubpartForBlob(Type, mac)
	if err != nil {
		return nil, err
	}

	if !exists {
		return nil, ErrPackfileNotFound
	}

	rd, err := r.GetPackfileBlob(loc)
	if err != nil {
		return nil, err
	}

	r.rBytes.Add(int64(loc.Length))

	return rd, nil
}

func (r *Repository) GetBlobBytes(Type resources.Type, mac objects.MAC) ([]byte, error) {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "GetBlobByte(%s, %x): %s", Type, mac, time.Since(t0))
	}()

	rd, err := r.GetBlob(Type, mac)
	if err != nil {
		return nil, err
	}

	return io.ReadAll(rd)
}

func (r *Repository) BlobExists(Type resources.Type, mac objects.MAC) bool {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "BlobExists(%s, %x): %s", Type, mac, time.Since(t0))
	}()

	if r.InTransaction() {
		if _, exists := r.PackerManager.InflightMACs[Type].Load(mac); exists {
			return true
		}
	}

	return r.state.BlobExists(Type, mac)
}

func (r *Repository) PutBlobIfNotExists(Type resources.Type, mac objects.MAC, data []byte) error {
	if r.BlobExists(Type, mac) {
		return nil
	}
	return r.PutBlob(Type, mac, data)
}

func (r *Repository) PutBlob(Type resources.Type, mac objects.MAC, data []byte) error {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "PutBlob(%s, %x): %s", Type, mac, time.Since(t0))
	}()

	if r.InTransaction() {
		if _, exists := r.PackerManager.InflightMACs[Type].LoadOrStore(mac, struct{}{}); exists {
			// tell prom exporter that we collided a blob
			return nil
		}
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

// Removes the provided blob from our state, making it unreachable
func (r *Repository) RemoveBlob(Type resources.Type, mac, packfileMAC objects.MAC) error {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "DeleteBlob(%s, %x, %x): %s", Type, mac, packfileMAC, time.Since(t0))
	}()
	return r.state.DelDelta(Type, mac, packfileMAC)
}

// State mutation
func (r *Repository) DeleteStateResource(Type resources.Type, mac objects.MAC) error {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "DeleteStateResource(%s, %x): %s", Type.String(), mac, time.Since(t0))
	}()

	if r.deltaState == nil {
		panic("Put outside of transaction")
	}

	r.transactionMtx.RLock()
	defer r.transactionMtx.RUnlock()
	return r.deltaState.DeleteResource(Type, mac)
}

func (r *Repository) ListOrphanBlobs() iter.Seq2[state.DeltaEntry, error] {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "ListSnapshots(): %s", time.Since(t0))
	}()
	return r.state.ListOrphanDeltas()
}

func (r *Repository) ListSnapshots() iter.Seq[objects.MAC] {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "ListSnapshots(): %s", time.Since(t0))
	}()
	return r.state.ListSnapshots()
}

func (r *Repository) ListPackfiles() iter.Seq[objects.MAC] {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "ListPackfiles(): %s", time.Since(t0))
	}()
	return r.state.ListPackfiles()
}

// Saves the full aggregated state to the repository, might be heavy handed use
// with care.
func (r *Repository) PutCurrentState() error {
	pr, pw := io.Pipe()

	/* By using a pipe and a goroutine we bound the max size in memory. */
	go func() {
		defer pw.Close()
		if err := r.state.SerializeToStream(pw); err != nil {
			pw.CloseWithError(err)
		}
	}()

	newSerial := uuid.New()
	r.state.Metadata.Serial = newSerial
	r.state.Metadata.Timestamp = time.Now()
	id := r.ComputeMAC(newSerial[:])

	return r.PutState(id, pr)
}

func (r *Repository) Logger() *logging.Logger {
	return r.AppContext().GetLogger()
}

func (r *Repository) GetLocks() ([]objects.MAC, error) {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "GetLocks(): %s", time.Since(t0))
	}()

	return r.store.GetLocks()
}

func (r *Repository) GetLock(lockID objects.MAC) (versioning.Version, io.Reader, error) {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "GetLock(%x): %s", lockID, time.Since(t0))
	}()

	rd, err := r.store.GetLock(lockID)
	if err != nil {
		return versioning.Version(0), nil, err
	}

	version, rd, err := storage.Deserialize(r.GetMACHasher(), resources.RT_LOCK, rd)
	if err != nil {
		return versioning.Version(0), nil, err
	}

	rd, err = r.Decode(rd)
	if err != nil {
		return versioning.Version(0), nil, err
	}
	return version, rd, err
}

func (r *Repository) PutLock(lockID objects.MAC, rd io.Reader) (int64, error) {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "PutLock(%x, ...): %s", lockID, time.Since(t0))
	}()

	rd, err := r.Encode(rd)
	if err != nil {
		return 0, err
	}

	rd, err = storage.Serialize(r.GetMACHasher(), resources.RT_LOCK, versioning.GetCurrentVersion(resources.RT_LOCK), rd)
	if err != nil {
		return 0, err
	}

	return r.store.PutLock(lockID, rd)
}

func (r *Repository) DeleteLock(lockID objects.MAC) error {
	t0 := time.Now()
	defer func() {
		r.Logger().Trace("repository", "DeleteLock(%x, ...): %s", lockID, time.Since(t0))
	}()

	return r.store.DeleteLock(lockID)
}
