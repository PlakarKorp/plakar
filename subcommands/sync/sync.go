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

	"github.com/PlakarKorp/kloset/appcontext"
	"github.com/PlakarKorp/kloset/encryption"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/kloset/storage"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Sync{} }, subcommands.AgentSupport, "sync")
}

func (cmd *Sync) Parse(ctx *appcontext.AppContext, args []string) error {
	cmd.SrcLocateOptions = utils.NewDefaultLocateOptions()

	flags := flag.NewFlagSet("sync", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [SNAPSHOT] to REPOSITORY\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s [SNAPSHOT] from REPOSITORY\n", flags.Name())
		flags.PrintDefaults()
	}
	cmd.SrcLocateOptions.InstallFlags(flags)

	flags.Parse(args)

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
		cmd.SrcLocateOptions.Prefix = args[0]
		direction = args[1]
		peerRepositoryPath = args[2]

	default:
		return fmt.Errorf("usage: sync [SNAPSHOT] to|from REPOSITORY")
	}

	if direction != "to" && direction != "from" && direction != "with" {
		return fmt.Errorf("invalid direction, must be to, from or with")
	}

	storeConfig, err := ctx.GetRepositoryConfig(peerRepositoryPath)
	if err != nil {
		return fmt.Errorf("peer repository: %w", err)
	}

	peerStore, peerStoreSerializedConfig, err := storage.Open(ctx, storeConfig)
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
		} else {
			for {
				passphrase, err := utils.GetPassphrase("destination repository")
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
	_, err = repository.NewNoRebuild(peerCtx, peerStore, peerStoreSerializedConfig)
	if err != nil {
		return err
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.PeerRepositoryLocation = peerRepositoryPath
	cmd.PeerRepositorySecret = peerSecret
	cmd.Direction = direction

	return nil
}

type Sync struct {
	subcommands.SubcommandBase

	PeerRepositoryLocation string
	PeerRepositorySecret   []byte

	Direction string

	SrcLocateOptions *utils.LocateOptions
}

func (cmd *Sync) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	storeConfig, err := ctx.GetRepositoryConfig(cmd.PeerRepositoryLocation)
	if err != nil {
		return 1, fmt.Errorf("peer repository: %w", err)
	}

	peerStore, peerStoreSerializedConfig, err := storage.Open(ctx, storeConfig)
	if err != nil {
		return 1, fmt.Errorf("could not open peer store %s: %s", cmd.PeerRepositoryLocation, err)
	}

	peerCtx := appcontext.NewAppContextFrom(ctx)
	peerCtx.SetSecret(cmd.PeerRepositorySecret)
	peerRepository, err := repository.New(peerCtx, peerStore, peerStoreSerializedConfig)
	if err != nil {
		return 1, fmt.Errorf("could not open peer repository %s: %s", cmd.PeerRepositoryLocation, err)
	}

	var srcRepository *repository.Repository
	var dstRepository *repository.Repository

	if cmd.Direction == "to" {
		srcRepository = repo
		dstRepository = peerRepository
	} else if cmd.Direction == "from" {
		srcRepository = peerRepository
		dstRepository = repo
	} else if cmd.Direction == "with" {
		srcRepository = repo
		dstRepository = peerRepository
	} else {
		return 1, fmt.Errorf("could not synchronize %s: invalid direction, must be to, from or with", peerStore.Location())
	}

	srcSnapshots, err := srcRepository.GetSnapshots()
	if err != nil {
		return 1, fmt.Errorf("could not get list of snapshots from source repository %s: %s", srcRepository.Location(), err)
	}

	dstSnapshots, err := dstRepository.GetSnapshots()
	if err != nil {
		return 1, fmt.Errorf("could not get list of snapshots from peer repository %s: %s", dstRepository.Location(), err)
	}

	srcSnapshotsMap := make(map[objects.MAC]struct{})
	dstSnapshotsMap := make(map[objects.MAC]struct{})

	for _, snapshotID := range srcSnapshots {
		srcSnapshotsMap[snapshotID] = struct{}{}
	}

	for _, snapshotID := range dstSnapshots {
		dstSnapshotsMap[snapshotID] = struct{}{}
	}

	srcSyncList := make([]objects.MAC, 0)

	srcSnapshotIDs, err := utils.LocateSnapshotIDs(srcRepository, cmd.SrcLocateOptions)
	if err != nil {
		return 1, fmt.Errorf("could not locate snapshots in source repository %s: %s", dstRepository.Location(), err)
	}

	for _, snapshotID := range srcSnapshotIDs {
		if _, exists := dstSnapshotsMap[snapshotID]; !exists {
			srcSyncList = append(srcSyncList, snapshotID)
		}
	}

	for _, snapshotID := range srcSyncList {
		if err := ctx.Err(); err != nil {
			return 1, err
		}

		err := synchronize(ctx, srcRepository, dstRepository, snapshotID)
		if err != nil {
			ctx.GetLogger().Error("failed to synchronize snapshot %x from source repository %s: %s",
				snapshotID[:4], srcRepository.Location(), err)
		}
	}

	if cmd.Direction == "with" {
		dstSnapshotIDs, err := utils.LocateSnapshotIDs(dstRepository, cmd.SrcLocateOptions)
		if err != nil {
			return 1, fmt.Errorf("could not locate snapshots in peer repository %s: %s", dstRepository.Location(), err)
		}

		dstSyncList := make([]objects.MAC, 0)
		for _, snapshotID := range dstSnapshotIDs {
			if _, exists := srcSnapshotsMap[snapshotID]; !exists {
				dstSyncList = append(dstSyncList, snapshotID)
			}
		}

		for _, snapshotID := range dstSyncList {
			if err := ctx.Err(); err != nil {
				return 1, err
			}
			err := synchronize(ctx, dstRepository, srcRepository, snapshotID)
			if err != nil {
				ctx.GetLogger().Error("failed to synchronize snapshot %x from peer repository %s: %s",
					snapshotID[:4], dstRepository.Location(), err)
			}
		}
		ctx.GetLogger().Info("sync: synchronization between %s and %s completed: %d snapshots synchronized",
			srcRepository.Location(),
			dstRepository.Location(),
			len(srcSyncList)+len(dstSyncList))
	} else if cmd.Direction == "to" {
		ctx.GetLogger().Info("sync: synchronization from %s to %s completed: %d snapshots synchronized",
			srcRepository.Location(),
			dstRepository.Location(),
			len(srcSyncList))
	} else {
		ctx.GetLogger().Info("sync: synchronization from %s to %s completed: %d snapshots synchronized",
			dstRepository.Location(),
			srcRepository.Location(),
			len(srcSyncList))
	}

	return 0, nil
}

func synchronize(ctx *appcontext.AppContext, srcRepository, dstRepository *repository.Repository, snapshotID objects.MAC) error {
	ctx.GetLogger().Info("Synchronizing snapshot %x from %s to %s", snapshotID, srcRepository.Location(), dstRepository.Location())
	srcSnapshot, err := snapshot.Load(srcRepository, snapshotID)
	if err != nil {
		return err
	}
	defer srcSnapshot.Close()

	dstSnapshot, err := snapshot.Create(dstRepository, repository.DefaultType)
	if err != nil {
		return err
	}
	defer dstSnapshot.Close()

	// overwrite the header, we want to keep the original snapshot info
	dstSnapshot.Header = srcSnapshot.Header

	if err := srcSnapshot.Synchronize(dstSnapshot); err != nil {
		return err
	}

	err = dstSnapshot.Commit(nil, true)

	ctx.GetLogger().Info("Synchronization of %x finished", snapshotID)
	return err
}
