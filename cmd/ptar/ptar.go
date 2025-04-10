package main

import (
	"bytes"
	"flag"
	"io"
	"math"
	"os"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/caching"
	"github.com/PlakarKorp/plakar/compression"
	"github.com/PlakarKorp/plakar/hashing"
	"github.com/PlakarKorp/plakar/logging"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/snapshot"
	"github.com/PlakarKorp/plakar/snapshot/importer"
	"github.com/PlakarKorp/plakar/storage"
	"github.com/PlakarKorp/plakar/versioning"

	_ "github.com/PlakarKorp/plakar/snapshot/importer/fs"
	_ "github.com/PlakarKorp/plakar/storage/backends/platar"
)

func main() {
	flag.Parse()

	if flag.NArg() != 2 {
		panic("bleh")
	}

	location := flag.Arg(0)
	data := flag.Arg(1)

	storageConfig := storage.NewConfiguration()
	storageConfig.Compression = compression.NewDefaultConfiguration()
	storageConfig.Hashing = *hashing.NewDefaultConfiguration()
	storageConfig.Encryption = nil // temporarily disable encryption
	storageConfig.Packfile.MaxSize = math.MaxUint64
	hasher := hashing.GetHasher(storage.DEFAULT_HASHING_ALGORITHM)

	serializedConfiguration, err := storageConfig.ToBytes()
	if err != nil {
		panic(err)
	}

	rd, err := storage.Serialize(hasher, resources.RT_CONFIG, versioning.GetCurrentVersion(resources.RT_CONFIG), bytes.NewReader(serializedConfiguration))
	if err != nil {
		panic(err)
	}
	wrappedConfig, err := io.ReadAll(rd)
	if err != nil {
		panic(err)
	}

	st, err := storage.Create(map[string]string{"location": location}, wrappedConfig)
	if err != nil {
		panic(err)
	}

	logger := logging.NewLogger(os.Stdout, os.Stderr)
	cache := caching.NewManager("/tmp/kloset-cache")

	appCtx := appcontext.NewAppContext()
	appCtx.SetLogger(logger)
	appCtx.SetCache(cache)

	repo, err := repository.New(appCtx, st, wrappedConfig)
	if err != nil {
		panic(err)
	}
	_ = repo

	imp, err := importer.NewImporter(map[string]string{"location": data})
	if err != nil {
		panic(err)
	}

	snap, err := snapshot.Create(repo)
	if err != nil {
		panic(err)
	}

	backupOptions := &snapshot.BackupOptions{
		MaxConcurrency: 4,
		NoCheckpoint:   true,
	}

	err = snap.Backup(imp, backupOptions)
	if err != nil {
		panic(err)
	}

	st.Close()

}
