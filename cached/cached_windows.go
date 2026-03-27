package cached

import (
	"fmt"
	"time"

	"github.com/PlakarKorp/kloset/connectors/storage"
	"github.com/PlakarKorp/kloset/encryption"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/google/uuid"
)

func CacheSnapshotHeader(ctx *appcontext.AppContext, snapshotID objects.MAC, repoID uuid.UUID, storeConfig map[string]string) (int, error) {
	t0 := time.Now()
	defer func() {
		ctx.GetLogger().Trace("cached", "cache snapshot (snap=%x, store=%s): %s", snapshotID, repoID, time.Since(t0))
	}()

	var serializedConfig []byte
	store, serializedConfig, err := storage.Open(ctx.GetInner(), storeConfig)
	if err != nil {
		return -1, fmt.Errorf("failed to open storage: %w", err)
	}

	key, err := getSecret(ctx, ctx.GetSecret(), serializedConfig)
	if err != nil {
		return -1, fmt.Errorf("failed to setup secret: %w", err)
	}

	repo, err := repository.NewNoRebuild(ctx.GetInner(), key, store, serializedConfig, false)
	if err != nil {
		return -1, fmt.Errorf("failed to open repository: %w", err)
	}

	if repoID != repo.Configuration().RepositoryID {
		return -1, fmt.Errorf("invalid uuid given %q repository id is %q", repoID.String(), repo.Configuration().RepositoryID.String())
	}

	if err := repo.CacheSnapshot(snapshotID); err != nil {
		return -1, err
	}

	return 0, nil
}

func RebuildStateFromStateFile(ctx *appcontext.AppContext, stateID objects.MAC, repoID uuid.UUID, storeConfig map[string]string, fireAndForget bool) (int, error) {
	t0 := time.Now()
	defer func() {
		ctx.GetLogger().Trace("cached", "rebuild from local statefile (file=%x, store=%s): %s", stateID, repoID, time.Since(t0))
	}()

	var serializedConfig []byte
	store, serializedConfig, err := storage.Open(ctx.GetInner(), storeConfig)
	if err != nil {
		return -1, fmt.Errorf("failed to open storage: %w", err)
	}

	key, err := getSecret(ctx, ctx.GetSecret(), serializedConfig)
	if err != nil {
		return -1, fmt.Errorf("failed to setup secret: %w", err)
	}

	repo, err := repository.NewNoRebuild(ctx.GetInner(), key, store, serializedConfig, false)
	if err != nil {
		return -1, fmt.Errorf("failed to open repository: %w", err)
	}

	if repoID != repo.Configuration().RepositoryID {
		return -1, fmt.Errorf("invalid uuid given %q repository id is %q", repoID.String(), repo.Configuration().RepositoryID.String())
	}

	if err := repo.IngestStateFile(stateID); err != nil {
		return -1, err
	}

	return 0, nil
}

func RebuildStateFromStore(ctx *appcontext.AppContext, repoID uuid.UUID, storeConfig map[string]string, fireAndForget bool) (int, error) {
	t0 := time.Now()
	defer func() {
		ctx.GetLogger().Trace("cached", "rebuild from store (store=%s): %s", repoID, time.Since(t0))
	}()

	var serializedConfig []byte
	store, serializedConfig, err := storage.Open(ctx.GetInner(), storeConfig)
	if err != nil {
		return -1, fmt.Errorf("failed to open storage: %w", err)
	}

	key, err := getSecret(ctx, ctx.GetSecret(), serializedConfig)
	if err != nil {
		return -1, fmt.Errorf("failed to setup secret: %w", err)
	}

	repo, err := repository.NewNoRebuild(ctx.GetInner(), key, store, serializedConfig, false)
	if err != nil {
		return -1, fmt.Errorf("failed to open repository: %w", err)
	}

	if repoID != repo.Configuration().RepositoryID {
		return -1, fmt.Errorf("invalid uuid given %q repository id is %q", repoID.String(), repo.Configuration().RepositoryID.String())
	}

	if err := repo.RebuildState(); err != nil {
		return -1, err
	}

	return 0, nil
}

// A bit of copy pasta, we'll clean this up globally.
func getSecret(ctx *appcontext.AppContext, secret []byte, storageConfig []byte) ([]byte, error) {
	config, err := storage.NewConfigurationFromWrappedBytes(storageConfig)
	if err != nil {
		return nil, err
	}

	if config.Encryption == nil {
		return nil, nil
	}

	key := secret
	if !encryption.VerifyCanary(config.Encryption, key) {
		return nil, fmt.Errorf("failed to verify key")
	}

	return key, nil
}
