/*
 * Copyright (c) 2021 Gilles Chehade <gilles@poolp.org>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package sync

import (
	"flag"
	"fmt"
	"os"

	"github.com/PlakarKorp/kloset/connectors/storage"
	"github.com/PlakarKorp/kloset/encryption"
	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/kloset/snapshot/vfs"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cached"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
)

func init() {
	subcommands.Register(Sync, 0, "sync")
}

func Sync(ctx *appcontext.AppContext, repo *repository.Repository, args []string) error {
	var (
		packfileTempStorage string
		cache               string
		srcLocOpts          = locate.NewDefaultLocateOptions()
	)

	flags := flag.NewFlagSet("sync", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [SNAPSHOT] to REPOSITORY\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s [SNAPSHOT] from REPOSITORY\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s [SNAPSHOT] with REPOSITORY\n", flags.Name())
		flags.PrintDefaults()
	}

	srcLocOpts.InstallLocateFlags(flags)
	flags.StringVar(&packfileTempStorage, "packfiles", "", "memory or a path to a directory to store temporary packfiles")
	flags.StringVar(&cache, "cache", "vfs", "path to store vfs cache, 'no' for uncached and 'vfs' for the default in memory cache")

	flags.Parse(args)

	if flags.NArg() > 3 {
		return fmt.Errorf("Too many arguments")
	}

	direction := ""
	peerRepositoryPath := ""

	args = flags.Args()
	switch len(args) {
	case 2:
		direction = args[0]
		peerRepositoryPath = args[1]
	case 3:
		if !srcLocOpts.Empty() {
			ctx.GetLogger().Warn("snapshot specified, filters will be ignored")
		}
		srcLocOpts.Filters.IDs = []string{args[0]}
		direction = args[1]
		peerRepositoryPath = args[2]

	default:
		return fmt.Errorf("usage: sync [SNAPSHOT] to|from|with REPOSITORY")
	}

	if direction != "to" && direction != "from" && direction != "with" {
		return fmt.Errorf("invalid direction, must be to, from or with")
	}

	storeConfig, err := ctx.Config.GetRepository(peerRepositoryPath)
	if err != nil {
		return fmt.Errorf("peer store: %w", err)
	}

	peerStore, peerStoreSerializedConfig, err := storage.Open(ctx.GetInner(), storeConfig)
	if err != nil {
		return err
	}

	peerStoreConfig, err := storage.NewConfigurationFromWrappedBytes(peerStoreSerializedConfig)
	if err != nil {
		return err
	}

	var peerSecret []byte
	if peerStoreConfig.Encryption != nil {
		if pass, ok := storeConfig["passphrase"]; ok {
			key, err := encryption.DeriveKey(peerStoreConfig.Encryption.KDFParams, []byte(pass))
			if err != nil {
				return err
			}
			if !encryption.VerifyCanary(peerStoreConfig.Encryption, key) {
				return fmt.Errorf("invalid passphrase")
			}
			peerSecret = key
		} else if cmd, ok := storeConfig["passphrase_cmd"]; ok {
			passphrase, err := utils.GetPassphraseFromCommand(cmd)
			if err != nil {
				return fmt.Errorf("failed to read passphrase from command: %w", err)
			}
			key, err := encryption.DeriveKey(peerStoreConfig.Encryption.KDFParams, []byte(passphrase))
			if err != nil {
				return err
			}
			if !encryption.VerifyCanary(peerStoreConfig.Encryption, key) {
				return fmt.Errorf("invalid passphrase")
			}
			peerSecret = key
		} else {
			for {
				passphrase, err := utils.GetPassphrase("destination store")
				if err != nil {
					fmt.Fprintf(os.Stderr, "%s\n", err)
					continue
				}

				key, err := encryption.DeriveKey(peerStoreConfig.Encryption.KDFParams, passphrase)
				if err != nil {
					return err
				}
				if !encryption.VerifyCanary(peerStoreConfig.Encryption, key) {
					return fmt.Errorf("invalid passphrase")
				}
				peerSecret = key
				break
			}
		}
	}

	peerCtx := appcontext.NewAppContextFrom(ctx)
	peerCtx.SetSecret(peerSecret)
	peerCtx.StoreConfig = storeConfig
	_, err = repository.NewNoRebuild(peerCtx.GetInner(), peerCtx.GetSecret(), peerStore, peerStoreSerializedConfig, true)
	if err != nil {
		return err
	}

	peerRepository, err := repository.NewNoRebuild(peerCtx.GetInner(), peerCtx.GetSecret(), peerStore, peerStoreSerializedConfig, true)
	if err != nil {
		return fmt.Errorf("could not open peer repository %s: %w", peerRepositoryPath, err)
	}

	if _, err = cached.RebuildStateFromStore(peerCtx, peerRepository.Configuration().RepositoryID, storeConfig, false); err != nil {
		return fmt.Errorf("failed to rebuild peer repository's state %s: %w", peerRepositoryPath, err)
	}

	if repo.Configuration().RepositoryID == peerRepository.Configuration().RepositoryID {
		if repo.Origin() == peerRepository.Origin() && repo.Root() == peerRepository.Root() {
			return fmt.Errorf("cannot synchronize snapshots to the same store")
		}

		ctx.GetLogger().Error("ATTENTION")
		ctx.GetLogger().Error("")
		ctx.GetLogger().Error("both stores have the same identifier but different origins or roots.")
		ctx.GetLogger().Error("")
		ctx.GetLogger().Error("this means one store was created using `plakar clone` instead of `plakar create`,")
		ctx.GetLogger().Error("but `plakar clone` is now deprecated as it was unsafe to use.")
		ctx.GetLogger().Error("")
		ctx.GetLogger().Error("DON'T WORRY, here's a plan!")
		ctx.GetLogger().Error("")
		ctx.GetLogger().Error("STEP 1: run `plakar check` on both ends")
		ctx.GetLogger().Error("STEP 2: if no error, recreate your target store using `plakar create` and sync again")
		ctx.GetLogger().Error("STEP 3: if errors were found, contact support@plakar.io and we will take care of you")
		ctx.GetLogger().Error("")
		return fmt.Errorf("cannot synchronize snapshots from cloned stores")
	}

	if packfileTempStorage != "memory" {
		tmpDir, err := os.MkdirTemp(packfileTempStorage, "plakar-sync-"+repo.Configuration().RepositoryID.String()+"-*")
		if err != nil {
			return err
		}
		packfileTempStorage = tmpDir
		defer os.RemoveAll(packfileTempStorage)
	} else {
		packfileTempStorage = ""
	}

	var srcRepository *repository.Repository
	var dstRepository *repository.Repository

	srcStoreConfig := ctx.StoreConfig
	if direction == "to" {
		srcRepository = repo
		dstRepository = peerRepository
	} else if direction == "from" {
		srcRepository = peerRepository
		dstRepository = repo
		srcStoreConfig = storeConfig
		tmp := ctx
		ctx = peerCtx
		peerCtx = tmp
	} else if direction == "with" {
		srcRepository = repo
		dstRepository = peerRepository
	} else {
		return fmt.Errorf("could not synchronize %s: invalid direction, must be to, from or with",
			peerRepositoryPath)
	}

	srcLocation := srcRepository.Origin()
	dstLocation := dstRepository.Origin()

	srcSnapshotsMap := make(map[objects.MAC]struct{})
	dstSnapshotsMap := make(map[objects.MAC]struct{})

	for objMAC, err := range srcRepository.ListSnapshots() {
		if err != nil {
			return err
		}
		srcSnapshotsMap[objMAC] = struct{}{}
	}

	for objMAC, err := range dstRepository.ListSnapshots() {
		if err != nil {
			return err
		}
		dstSnapshotsMap[objMAC] = struct{}{}
	}

	srcSyncList := make([]objects.MAC, 0)

	srcSnapshotIDs, err := locate.LocateSnapshotIDs(srcRepository, srcLocOpts)
	if err != nil {
		return fmt.Errorf("could not locate snapshots in store %s: %s", dstLocation, err)
	}
	if direction != "with" {
		if len(srcSnapshotIDs) == 0 {
			ctx.GetLogger().Info("No matching snapshot found in store %s", srcLocation)
			return nil
		}
	}

	for _, snapshotID := range srcSnapshotIDs {
		if _, exists := dstSnapshotsMap[snapshotID]; !exists {
			srcSyncList = append(srcSyncList, snapshotID)
		}
	}

	srcSynced := 0
	for _, snapshotID := range srcSyncList {
		if err := ctx.Err(); err != nil {
			return err
		}

		err := synchronize(ctx, peerCtx, srcRepository, dstRepository, srcStoreConfig, snapshotID, packfileTempStorage, cache)
		if err != nil {
			ctx.GetLogger().Error("failed to synchronize snapshot %x from store %s: %s",
				snapshotID[:4], srcLocation, err)
		} else {
			srcSynced++
		}
	}

	if direction == "with" {

		dstSnapshotIDs, err := locate.LocateSnapshotIDs(dstRepository, srcLocOpts)
		if err != nil {
			return fmt.Errorf("could not locate snapshots in store %s: %s", dstLocation, err)
		}

		srcRepository = peerRepository
		dstRepository = repo
		srcStoreConfig = storeConfig
		tmp := ctx
		ctx = peerCtx
		peerCtx = tmp

		dstSyncList := make([]objects.MAC, 0)
		for _, snapshotID := range dstSnapshotIDs {
			if _, exists := srcSnapshotsMap[snapshotID]; !exists {
				dstSyncList = append(dstSyncList, snapshotID)
			}
		}

		dstSynced := 0
		for _, snapshotID := range dstSyncList {
			if err := ctx.Err(); err != nil {
				return err
			}

			err := synchronize(ctx, peerCtx, srcRepository, dstRepository, srcStoreConfig, snapshotID,
				packfileTempStorage, cache)
			if err != nil {
				ctx.GetLogger().Error("failed to synchronize snapshot %x from peer store %s: %s",
					snapshotID[:4], dstLocation, err)
			} else {
				dstSynced++
			}
		}
		ctx.GetLogger().Info("sync: synchronization between %s and %s completed: %d snapshots synchronized",
			srcLocation,
			dstLocation,
			srcSynced+dstSynced)
	} else if direction == "to" {
		ctx.GetLogger().Info("sync: synchronization from %s to %s completed: %d snapshots synchronized",
			srcLocation,
			dstLocation,
			srcSynced)
	} else {
		ctx.GetLogger().Info("sync: synchronization from %s to %s completed: %d snapshots synchronized",
			dstLocation,
			srcLocation,
			srcSynced)
	}

	return nil
}

func synchronize(
	ctx, peerCtx *appcontext.AppContext,
	srcRepository, dstRepository *repository.Repository,
	srcStoreConfig map[string]string,
	snapshotID objects.MAC,
	PackfileTempStorage string,
	Cache string,
) error {
	srcLocation := srcRepository.Origin()
	dstLocation := dstRepository.Origin()

	ctx.GetLogger().Info("Synchronizing snapshot %x from %s to %s", snapshotID, srcLocation, dstLocation)
	srcSnapshot, err := snapshot.Load(srcRepository, snapshotID)
	if err != nil {
		return err
	}
	defer srcSnapshot.Close()

	dstSnapshot, err := snapshot.Create(dstRepository, repository.DefaultType, PackfileTempStorage, srcSnapshot.Header.Identifier, &snapshot.BuilderOptions{
		NoCommit:       false,
		NoCheckpoint:   false,
		StateRefresher: stateRefresher(peerCtx, dstRepository),
	})
	if err != nil {
		return err
	}
	defer dstSnapshot.Close()

	// overwrite the header, we want to keep the original snapshot info
	dstSnapshot.Header = srcSnapshot.Header

	var parentVFS *vfs.Filesystem
	if Cache == "vfs" {
		parentID, _, err := locate.Match(dstRepository, &locate.LocateOptions{
			Filters: locate.LocateFilters{
				Latest: true,
				Roots: []string{
					srcSnapshot.Header.GetSource(0).Importer.Directory,
				},
				Types: []string{
					srcSnapshot.Header.GetSource(0).Importer.Type,
				},
				Origins: []string{
					srcSnapshot.Header.GetSource(0).Importer.Origin,
				},
			},
		})
		if err != nil {
			return err
		}

		if len(parentID) != 0 {
			parent, err := snapshot.Load(dstRepository, parentID[0])
			if err != nil {
				return err
			}
			defer parent.Close()

			parentVFS, err = parent.FilesystemWithCache()
			if err != nil {
				return err
			}
		}
	}

	dstSnapshot.WithVFSCache(parentVFS)

	if err := srcSnapshot.Synchronize(dstSnapshot); err != nil {
		return err
	}

	ctx.GetLogger().Info("Synchronization of %x finished", snapshotID)
	return err
}

// We don't want to go through cached, if we need to refresh the state call
// Repository.RebuildState
var stateRefresher = func(ctx *appcontext.AppContext, repo *repository.Repository) func(mac objects.MAC, finalRefresh bool) error {
	return func(mac objects.MAC, finalRefresh bool) error {
		// If we are in the final refresh, turn this request into a fire and
		// forget one, to improve the UX.
		_, err := cached.RebuildStateFromStateFile(ctx, mac, repo.Configuration().RepositoryID, ctx.StoreConfig, false)
		return err
	}
}
