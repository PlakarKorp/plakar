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
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cached"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
)

type Sync struct {
	subcommands.SubcommandBase

	PeerRepositoryLocation string
	PeerRepositorySecret   []byte

	Direction           string
	PackfileTempStorage string

	SrcLocateOptions *locate.LocateOptions
}

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Sync{} }, 0, "sync")
}

func (cmd *Sync) Parse(ctx *appcontext.AppContext, args []string) error {
	cmd.SrcLocateOptions = locate.NewDefaultLocateOptions()

	flags := flag.NewFlagSet("sync", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [SNAPSHOT] to REPOSITORY\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s [SNAPSHOT] from REPOSITORY\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s [SNAPSHOT] with REPOSITORY\n", flags.Name())
		flags.PrintDefaults()
	}

	cmd.SrcLocateOptions.InstallLocateFlags(flags)
	flags.StringVar(&cmd.PackfileTempStorage, "packfiles", "", "memory or a path to a directory to store temporary packfiles")

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
		if !cmd.SrcLocateOptions.Empty() {
			ctx.GetLogger().Warn("snapshot specified, filters will be ignored")
		}
		cmd.SrcLocateOptions.Filters.IDs = []string{args[0]}
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
	_, err = repository.NewNoRebuild(peerCtx.GetInner(), peerCtx.GetSecret(), peerStore, peerStoreSerializedConfig, true)
	if err != nil {
		return err
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.PeerRepositoryLocation = peerRepositoryPath
	cmd.PeerRepositorySecret = peerSecret
	cmd.Direction = direction

	return nil
}

func (cmd *Sync) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	storeConfig, err := ctx.Config.GetRepository(cmd.PeerRepositoryLocation)
	if err != nil {
		return 1, fmt.Errorf("peer store: %w", err)
	}

	peerStore, peerStoreSerializedConfig, err := storage.Open(ctx.GetInner(), storeConfig)
	if err != nil {
		return 1, fmt.Errorf("could not open peer store %s: %s", cmd.PeerRepositoryLocation, err)
	}

	peerCtx := appcontext.NewAppContextFrom(ctx)
	peerCtx.SetSecret(cmd.PeerRepositorySecret)
	peerCtx.StoreConfig = storeConfig
	peerRepository, err := repository.New(peerCtx.GetInner(), peerCtx.GetSecret(), peerStore, peerStoreSerializedConfig)
	if err != nil {
		return 1, fmt.Errorf("could not open peer store %s: %s", cmd.PeerRepositoryLocation, err)
	}

	if repo.Configuration().RepositoryID == peerRepository.Configuration().RepositoryID {
		if repo.Origin() == peerRepository.Origin() && repo.Root() == peerRepository.Root() {
			return 1, fmt.Errorf("cannot synchronize snapshots to the same store")
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
		return 1, fmt.Errorf("cannot synchronize snapshots from cloned stores")
	}

	if cmd.PackfileTempStorage != "memory" {
		tmpDir, err := os.MkdirTemp(cmd.PackfileTempStorage, "plakar-sync-"+repo.Configuration().RepositoryID.String()+"-*")
		if err != nil {
			return 1, err
		}
		cmd.PackfileTempStorage = tmpDir
		defer os.RemoveAll(cmd.PackfileTempStorage)
	} else {
		cmd.PackfileTempStorage = ""
	}

	var srcRepository *repository.Repository
	var dstRepository *repository.Repository

	srcStoreConfig := ctx.StoreConfig
	if cmd.Direction == "to" {
		srcRepository = repo
		dstRepository = peerRepository
	} else if cmd.Direction == "from" {
		srcRepository = peerRepository
		dstRepository = repo
		srcStoreConfig = storeConfig
	} else if cmd.Direction == "with" {
		srcRepository = repo
		dstRepository = peerRepository
	} else {
		return 1, fmt.Errorf("could not synchronize %s: invalid direction, must be to, from or with", cmd.PeerRepositoryLocation)
	}

	srcLocation := srcRepository.Origin()
	dstLocation := dstRepository.Origin()

	srcSnapshotsMap := make(map[objects.MAC]struct{})
	dstSnapshotsMap := make(map[objects.MAC]struct{})

	for objMAC, err := range srcRepository.ListSnapshots() {
		if err != nil {
			return 1, err
		}
		srcSnapshotsMap[objMAC] = struct{}{}
	}

	for objMAC, err := range dstRepository.ListSnapshots() {
		if err != nil {
			return 1, err
		}
		dstSnapshotsMap[objMAC] = struct{}{}
	}

	srcSyncList := make([]objects.MAC, 0)

	srcSnapshotIDs, err := locate.LocateSnapshotIDs(srcRepository, cmd.SrcLocateOptions)
	if err != nil {
		return 1, fmt.Errorf("could not locate snapshots in store %s: %s", dstLocation, err)
	}
	if cmd.Direction != "with" {
		if len(srcSnapshotIDs) == 0 {
			ctx.GetLogger().Info("No matching snapshot found in store %s", srcLocation)
			return 0, nil
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
			return 1, err
		}

		err := synchronize(ctx, peerCtx, srcRepository, dstRepository, srcStoreConfig, snapshotID, cmd.PackfileTempStorage)
		if err != nil {
			ctx.GetLogger().Error("failed to synchronize snapshot %x from store %s: %s",
				snapshotID[:4], srcLocation, err)
		} else {
			srcSynced++
		}
	}

	if cmd.Direction == "with" {
		dstSnapshotIDs, err := locate.LocateSnapshotIDs(dstRepository, cmd.SrcLocateOptions)
		if err != nil {
			return 1, fmt.Errorf("could not locate snapshots in store %s: %s", dstLocation, err)
		}

		dstSyncList := make([]objects.MAC, 0)
		for _, snapshotID := range dstSnapshotIDs {
			if _, exists := srcSnapshotsMap[snapshotID]; !exists {
				dstSyncList = append(dstSyncList, snapshotID)
			}
		}

		dstSynced := 0
		for _, snapshotID := range dstSyncList {
			if err := ctx.Err(); err != nil {
				return 1, err
			}
			err := synchronize(ctx, peerCtx, dstRepository, srcRepository, storeConfig, snapshotID, cmd.PackfileTempStorage)
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
	} else if cmd.Direction == "to" {
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

	return 0, nil
}

func synchronize(ctx, peerCtx *appcontext.AppContext, srcRepository, dstRepository *repository.Repository, srcStoreConfig map[string]string, snapshotID objects.MAC, packfileDir string) error {
	srcLocation := srcRepository.Origin()
	dstLocation := dstRepository.Origin()

	ctx.GetLogger().Info("Synchronizing snapshot %x from %s to %s", snapshotID, srcLocation, dstLocation)
	srcSnapshot, err := snapshot.Load(srcRepository, snapshotID)
	if err != nil {
		return err
	}
	defer srcSnapshot.Close()

	dstSnapshot, err := snapshot.Create(dstRepository, repository.DefaultType, packfileDir, srcSnapshot.Header.Identifier, &snapshot.BuilderOptions{
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
