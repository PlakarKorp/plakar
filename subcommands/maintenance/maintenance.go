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

package maintenance

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/PlakarKorp/kloset/caching"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/resources"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"golang.org/x/sync/errgroup"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Maintenance{} }, subcommands.AgentSupport, "maintenance")
}

func (cmd *Maintenance) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("maintenance", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s\n", flags.Name())
	}
	flags.Parse(args)

	cmd.RepositorySecret = ctx.GetSecret()

	return nil
}

type Maintenance struct {
	subcommands.SubcommandBase

	repository    *repository.Repository
	maintenanceID objects.MAC
	cutoff        time.Time
}

// Builds the local cache of snapshot -> packfiles
func (cmd *Maintenance) updateCache(ctx *appcontext.AppContext, cache *caching.MaintenanceCache) error {
	wg := new(errgroup.Group)
	wg.SetLimit(ctx.MaxConcurrency)

	for snapshotID := range cmd.repository.ListSnapshots() {
		wg.Go(func() error {
			snapshot, err := snapshot.Load(cmd.repository, snapshotID)
			if err != nil {
				return err
			}
			defer snapshot.Close()

			ok, err := cache.HasSnapshot(snapshotID)
			if err != nil {
				return err
			}

			if ok {
				return nil
			}

			iter, err := snapshot.ListPackfiles()
			if err != nil {
				return err
			}

			for packfile, err := range iter {
				if err != nil {
					return err
				}

				if err := ctx.Err(); err != nil {
					return err
				}

				if err := cache.PutPackfile(snapshotID, packfile); err != nil {
					return err
				}
			}

			cache.PutSnapshot(snapshotID, nil)
			return nil
		})
	}

	if err := wg.Wait(); err != nil {
		return err
	}

	// While ListSnapshots doesn't return deleted snapshots, we still need to
	// go over them to remove previously added one to our local cache.
	for snapshotID := range cmd.repository.ListDeletedSnapShots() {
		ok, err := cache.HasSnapshot(snapshotID)
		if err != nil {
			return err
		}

		if !ok {
			continue
		}

		cache.DeleletePackfiles(snapshotID)
		cache.DeleteSnapshot(snapshotID)
	}

	return nil
}

func (cmd *Maintenance) colourPass(ctx *appcontext.AppContext, cache *caching.MaintenanceCache) error {
	var packfiles map[objects.MAC]struct{} = make(map[objects.MAC]struct{})
	for packfileMAC := range cmd.repository.ListPackfiles() {
		packfiles[packfileMAC] = struct{}{}
	}

	// Now go over the list of packfile as given by the storage so that we can
	// identify orphaned packfiles (eg. from an aborted backup)
	repoPackfiles, err := cmd.repository.GetPackfiles()
	if err != nil {
		return err
	}

	orphanedPackfiles := 0
	for _, packfileMAC := range repoPackfiles {
		_, ok := packfiles[packfileMAC]
		if ok {
			continue
		}

		// Maybe it was already colored by a previous run, let's not open the
		// packfile once again
		has, err := cmd.repository.HasDeletedPackfile(packfileMAC)
		if err != nil {
			return err
		}

		if has {
			continue
		}

		// The packfile is not available through state, so it's orphaned, but
		// we must take some care as it might be from an in progress backup. In
		// order to avoid deleting those we rely on the grace period. Sadly
		// this means we have to load the packfile from the repository,
		// hopefully those are rare enough that it's not a problem in practice.
		packfile, err := cmd.repository.GetPackfile(packfileMAC)
		if err != nil {
			return err
		}

		packfileDate := time.Unix(0, packfile.Footer.Timestamp)
		if packfileDate.Before(cmd.cutoff) {
			orphanedPackfiles++
			packfiles[packfileMAC] = struct{}{}
		}
	}

	sc, err := cmd.repository.AppContext().GetCache().Scan(cmd.maintenanceID)
	if err != nil {
		return err
	}

	// First pass, coloring, we just flag those packfiles as being selected for deletion.
	// For now we keep the same serial so that those delete gets merged in.
	// Once we do the real deletion we will rebuild the aggregated view
	// excluding those resources alltogether.
	repoWriter := cmd.repository.NewRepositoryWriter(sc, cmd.maintenanceID, repository.DefaultType, "")

	coloredPackfiles := 0
	for packfile := range packfiles {
		if cache.HasPackfile(packfile) {
			continue
		}

		has, err := cmd.repository.HasDeletedPackfile(packfile)
		if err != nil {
			return err
		}

		if !has {
			coloredPackfiles++
			if err := repoWriter.DeleteStateResource(resources.RT_PACKFILE, packfile); err != nil {
				return err
			}
		}
	}

	fmt.Fprintf(ctx.Stdout, "maintenance: Coloured %d packfiles (%d orphaned) for deletion\n", coloredPackfiles, orphanedPackfiles)

	if coloredPackfiles > 0 {
		duration, err := time.ParseDuration(os.Getenv("PLAKAR_GRACEPERIOD"))
		if err != nil {
			duration = 30 * 24 * time.Hour
		}

		humanDuration := duration.String()
		if duration.Hours() > 24 {
			days := duration / (24 * time.Hour)
			left := duration - (days * 24 * time.Hour)

			humanDuration = fmt.Sprintf("%dd", days)

			if left != 0 {
				humanDuration += left.String()
			}
		}

		fmt.Fprintf(ctx.Stdout, "Coloured packfiles are scheduled to be removed in %s\n", humanDuration)

		if err := repoWriter.CommitTransaction(cmd.maintenanceID); err != nil {
			return err
		}
	}

	return nil
}

func (cmd *Maintenance) sweepPass(ctx *appcontext.AppContext, cache *caching.MaintenanceCache) error {
	doDeletion, _ := strconv.ParseBool(os.Getenv("PLAKAR_DODELETION"))

	// First go over all the packfiles coloured by first pass.
	blobRemoved := 0
	toDelete := map[objects.MAC]struct{}{}
	for packfileMAC, deletionTime := range cmd.repository.ListDeletedPackfiles() {
		if deletionTime.After(cmd.cutoff) {
			continue
		}

		// At this point we have to re-check if our packfile is really unused,
		// because we could have had a concurrent backup with the coloring
		// phase.
		if cache.HasPackfile(packfileMAC) {
			fmt.Fprintf(ctx.Stderr, "maintenance: Concurrent backup used %x, uncolouring the packfile.\n", packfileMAC)
			cmd.repository.RemoveDeletedPackfile(packfileMAC)
			continue
		}

		// First thing we remove the packfile entry from our state, this means
		// that now effectively all of its blob are unreachable
		if err := cmd.repository.RemovePackfile(packfileMAC); err != nil {
			fmt.Fprintf(ctx.Stderr, "maintenance: Failed to remove packfile %s from state\n", packfileMAC)
			continue
		}

		cmd.repository.RemoveDeletedPackfile(packfileMAC)
		toDelete[packfileMAC] = struct{}{}
	}

	// Second garbage collect dangling blobs in our state. This is the blobs we
	// just orphaned plus potential orphan blobs from aborted backups etc.
	for blob, err := range cmd.repository.ListOrphanBlobs() {
		if err != nil {
			fmt.Fprintf(ctx.Stderr, "maintenance: Failed to fetch orphaned blob\n")
			continue
		}

		blobRemoved++
		if err := cmd.repository.RemoveBlob(blob.Type, blob.Blob, blob.Location.Packfile); err != nil {
			// No hurt in this failing, we just have cruft left around, but they are unreachable anyway.
			fmt.Fprintf(ctx.Stderr, "maintenance: garbage orphaned blobs pass failed to remove blob %x, type %s\n", blob.Blob, blob.Type)
		}
	}

	fmt.Fprintf(ctx.Stdout, "maintenance: %d blobs and %d packfiles were removed\n", blobRemoved, len(toDelete))

	if len(toDelete) > 0 {
		if err := cmd.repository.PutCurrentState(); err != nil {
			return err
		}
	}

	if doDeletion {
		for packfileMAC := range toDelete {
			if err := cmd.repository.DeletePackfile(packfileMAC); err != nil {
				fmt.Fprintf(ctx.Stderr, "maintenance: Sweep pass failed to delete packfile %x, skipping it\n", packfileMAC)
			}
		}
	}

	return nil
}

func (cmd *Maintenance) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	// the maintenance algorithm is a bit tricky and needs to be done in the correct sequence,
	// here's what it has to do:
	//
	// 1. fetch all packfiles in the repository
	// 2. iterate over snapshots
	// 3. for each snapshot, resolve the packfiles it references
	// 4. remove the packfiles from the list of packfiles
	// 5. remaining packfiles should be marked as deleted in the state
	// 6. remove the packfile in repository once it's flagged as deleted AND all snapshots have been `snapshot.Check`-ed
	// 7. rebuild a new aggregate state with a new serial without the deleted packfiles

	cmd.repository = repo

	// This need to be configurable per repo, but we don't have a mechanism yet (comes in a PR soon!)
	duration, err := time.ParseDuration(os.Getenv("PLAKAR_GRACEPERIOD"))
	if err != nil {
		duration = 30 * 24 * time.Hour
	}

	cmd.cutoff = time.Now().Add(-duration)

	// This random id generation for non snapshot state should probably be encapsulated somewhere.
	cmd.maintenanceID = objects.RandomMAC()

	done, err := cmd.Lock()
	if err != nil {
		return 1, err
	}
	defer cmd.Unlock(done)

	cache, err := repo.AppContext().GetCache().Maintenance(repo.Configuration().RepositoryID)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "maintenance: Failed to open local cache %s\n", err)
		return 1, err
	}

	if err := cmd.updateCache(ctx, cache); err != nil {
		fmt.Fprintf(ctx.Stderr, "maintenance: Failed to update local cache %s\n", err)
		return 1, err
	}

	if err := cmd.colourPass(ctx, cache); err != nil {
		fmt.Fprintf(ctx.Stderr, "maintenance: Colouring pass failed %s\n", err)
		return 1, err
	}

	if err := cmd.sweepPass(ctx, cache); err != nil {
		fmt.Fprintf(ctx.Stderr, "maintenance: Sweep pass failed %s\n", err)
		return 1, err
	}

	return 0, nil
}

func (cmd *Maintenance) Lock() (chan bool, error) {
	lockless, _ := strconv.ParseBool(os.Getenv("PLAKAR_LOCKLESS"))
	lockDone := make(chan bool)
	if lockless {
		return lockDone, nil
	}

	lock := repository.NewExclusiveLock(cmd.repository.AppContext().Hostname)

	buffer := &bytes.Buffer{}
	err := lock.SerializeToStream(buffer)
	if err != nil {
		return nil, err
	}

	_, err = cmd.repository.PutLock(cmd.maintenanceID, buffer)
	if err != nil {
		return nil, err
	}

	// We installed the lock, now let's see if there is a conflicting exclusive lock or not.
	locksID, err := cmd.repository.GetLocks()
	if err != nil {
		// We still need to delete it, and we need to do so manually.
		cmd.repository.DeleteLock(cmd.maintenanceID)
		return nil, err
	}

	for _, lockID := range locksID {
		if lockID == cmd.maintenanceID {
			continue
		}

		rd, err := cmd.repository.GetLock(lockID)
		if err != nil {
			cmd.repository.DeleteLock(cmd.maintenanceID)
			return nil, err
		}

		lock, err := repository.NewLockFromStream(rd)
		rd.Close()
		if err != nil {
			cmd.repository.DeleteLock(cmd.maintenanceID)
			return nil, err
		}

		/* Kick out stale locks */
		if lock.IsStale() {
			err := cmd.repository.DeleteLock(lockID)
			if err != nil {
				cmd.repository.DeleteLock(cmd.maintenanceID)
				return nil, err
			}
		}

		// There is a lock in place, we need to abort.
		err = cmd.repository.DeleteLock(cmd.maintenanceID)
		if err != nil {
			return nil, err
		}

		return nil, fmt.Errorf("Can't take exclusive lock, repository is already locked")
	}

	// The following bit is a "ping" mechanism, Lock() is a bit badly named at this point,
	// we are just refreshing the existing lock so that the watchdog doesn't removes us.
	go func() {
		for {
			select {
			case <-lockDone:
				cmd.repository.DeleteLock(cmd.maintenanceID)
				return
			case <-time.After(repository.LOCK_REFRESH_RATE):
				lock := repository.NewExclusiveLock(cmd.repository.AppContext().Hostname)

				buffer := &bytes.Buffer{}

				// We ignore errors here on purpose, it's tough to handle them
				// correctly, and if they happen we will be ripped by the
				// watchdog anyway.
				lock.SerializeToStream(buffer)
				cmd.repository.PutLock(cmd.maintenanceID, buffer)
			}
		}
	}()

	return lockDone, nil
}

func (cmd *Maintenance) Unlock(ping chan bool) {
	close(ping)
}
