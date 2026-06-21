/*
 * Copyright (c) 2025 Gilles Chehade <gilles@poolp.org>
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

package repair

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/repository/state"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
)

func init() {
	subcommands.Register(Repair, 0, "repair")
}

func Repair(ctx *appcontext.AppContext, repo *repository.Repository, args []string) error {
	var (
		Apply bool
	)

	flags := flag.NewFlagSet("repair", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}
	flags.BoolVar(&Apply, "apply", false, "do the actual repair")
	flags.Parse(args)

	repairID := objects.RandomMAC()

	if Apply {
		done, err := lock(ctx, repo, repairID)
		if err != nil {
			return err
		}

		defer close(done)
	}

	oldCache, err := repo.AppContext().GetCache().Repository(repo.Configuration().RepositoryID)
	if err != nil {
		return err
	}

	repo.RebuildStateWithCache(oldCache)
	remoteStates, err := repo.GetStates()
	if err != nil {
		return err
	}

	remoteStatesMap := make(map[objects.MAC]struct{}, 0)
	for _, stateID := range remoteStates {
		remoteStatesMap[stateID] = struct{}{}
	}

	packfilesPerState := make(map[objects.MAC][]objects.MAC, 0)
	for pe, err := range repo.ListPackfileEntries() {
		if err != nil {
			return err
		}
		if _, ok := remoteStatesMap[pe.StateID]; ok {
			continue
		}
		packfilesPerState[pe.StateID] = append(packfilesPerState[pe.StateID], pe.Packfile)
	}

	for stateID, packfiles := range packfilesPerState {
		if !Apply {
			ctx.GetLogger().Info("found missing state %x\n", stateID)
			continue
		} else {
			ctx.GetLogger().Info("repairing missing state %x\n", stateID)
		}

		scanCache, err := repo.AppContext().GetCache().Scan(stateID)
		if err != nil {
			return err
		}

		deltaState, err := state.NewLocalState(scanCache)
		if err != nil {
			return err
		}

		for _, pf := range packfiles {
			p, err := repo.GetPackfile(pf)
			if err != nil {
				return err
			}

			if deltaState.Metadata.Timestamp.UnixNano() > p.Footer.Timestamp {
				deltaState.Metadata.Timestamp = time.Unix(0, p.Footer.Timestamp)
			}

			for _, entry := range p.Index {
				delta := &state.DeltaEntry{
					Type:    entry.Type,
					Version: entry.Version,
					Blob:    entry.MAC,
					Location: state.Location{
						Packfile: pf,
						Offset:   entry.Offset,
						Length:   entry.Length,
					},
				}
				if err := deltaState.PutDelta(delta); err != nil {
					return err
				}
			}

			if err := deltaState.PutPackfile(stateID, pf); err != nil {
				return err
			}
		}

		pr, pw := io.Pipe()
		go func() {
			defer pw.Close()

			if err := deltaState.SerializeToStream(pw); err != nil {
				pw.CloseWithError(err)
			}
		}()
		err = repo.PutState(stateID, pr)
		if err != nil {
			return err
		}

		scanCache.Close()
	}

	if !Apply {
		if len(packfilesPerState) == 0 {
			ctx.GetLogger().Info("no repairs needed\n")
		} else {
			ctx.GetLogger().Info("to apply these repairs, run `plakar repair -apply`\n")
		}
	}

	return nil
}

func lock(ctx *appcontext.AppContext, repo *repository.Repository, repairID objects.MAC) (chan bool, error) {
	lockDone := make(chan bool)
	lock := repository.NewExclusiveLock(ctx.Hostname)

	buffer := &bytes.Buffer{}
	err := lock.SerializeToStream(buffer)
	if err != nil {
		return nil, err
	}

	_, err = repo.PutLock(repairID, buffer)
	if err != nil {
		return nil, err
	}

	// We installed the lock, now let's see if there is a conflicting exclusive lock or not.
	locksID, err := repo.GetLocks()
	if err != nil {
		// We still need to delete it, and we need to do so manually.
		repo.DeleteLock(repairID)
		return nil, err
	}

	for _, lockID := range locksID {
		if lockID == repairID {
			continue
		}

		rd, err := repo.GetLock(lockID)
		if err != nil {
			repo.DeleteLock(repairID)
			return nil, err
		}

		lock, err := repository.NewLockFromStream(rd)
		rd.Close()
		if err != nil {
			repo.DeleteLock(repairID)
			return nil, err
		}

		/* Kick out stale locks */
		if lock.IsStale() {
			err := repo.DeleteLock(lockID)
			if err != nil {
				repo.DeleteLock(repairID)
				return nil, err
			}

			continue
		}

		// There is a lock in place, we need to abort.
		err = repo.DeleteLock(repairID)
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
				repo.DeleteLock(repairID)
				return
			case <-time.After(repository.LOCK_REFRESH_RATE):
				lock := repository.NewExclusiveLock(ctx.Hostname)

				buffer := &bytes.Buffer{}

				// We ignore errors here on purpose, it's tough to handle them
				// correctly, and if they happen we will be ripped by the
				// watchdog anyway.
				lock.SerializeToStream(buffer)
				repo.PutLock(repairID, buffer)
			}
		}
	}()

	return lockDone, nil
}
