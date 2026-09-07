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
	"fmt"
	"io"
	"time"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/repository/state"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/spf13/cobra"
)

type Repair struct {
	subcommands.SubcommandBase

	Apply bool

	repository *repository.Repository
	repairID   objects.MAC
}

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Repair{} }, 0, "repair")
}

func (cmd *Repair) CobraCommand() *cobra.Command {
	c := &cobra.Command{
		Use: "repair",
	}
	c.Flags().BoolVar(&cmd.Apply, "apply", false, "do the actual repair")
	return c
}

func (cmd *Repair) Parse(ctx *appcontext.AppContext, args []string) error {
	if _, err := subcommands.ParseCobra(cmd, args); err != nil {
		return err
	}

	cmd.RepositorySecret = ctx.GetSecret()

	return nil
}

func (cmd *Repair) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	cmd.repository = repo
	cmd.repairID = objects.RandomMAC()

	if cmd.Apply {
		done, err := cmd.Lock()
		if err != nil {
			return 1, err
		}

		defer cmd.Unlock(done)
	}

	oldCache, err := repo.AppContext().GetCache().Repository(repo.Configuration().RepositoryID)
	if err != nil {
		return 1, err
	}

	repo.RebuildStateWithCache(oldCache)
	remoteStates, err := repo.GetStates()
	if err != nil {
		return 1, err
	}

	remoteStatesMap := make(map[objects.MAC]struct{}, 0)
	for _, stateID := range remoteStates {
		remoteStatesMap[stateID] = struct{}{}
	}

	packfilesPerState := make(map[objects.MAC][]objects.MAC, 0)
	for pe, err := range repo.ListPackfileEntries() {
		if err != nil {
			return 1, err
		}
		if _, ok := remoteStatesMap[pe.StateID]; ok {
			continue
		}
		packfilesPerState[pe.StateID] = append(packfilesPerState[pe.StateID], pe.Packfile)
	}

	for stateID, packfiles := range packfilesPerState {
		if !cmd.Apply {
			ctx.GetLogger().Info("found missing state %x\n", stateID)
			continue
		} else {
			ctx.GetLogger().Info("repairing missing state %x\n", stateID)
		}

		scanCache, err := repo.AppContext().GetCache().Scan(stateID)
		if err != nil {
			return 1, err
		}

		deltaState, err := state.NewLocalState(scanCache)
		if err != nil {
			return 1, err
		}

		for _, pf := range packfiles {
			p, err := repo.GetPackfile(pf)
			if err != nil {
				return 1, err
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
					return 1, err
				}
			}

			if err := deltaState.PutPackfile(stateID, pf); err != nil {
				return 1, err
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
			return 1, err
		}

		scanCache.Close()
	}

	if !cmd.Apply {
		if len(packfilesPerState) == 0 {
			ctx.GetLogger().Info("no repairs needed\n")
		} else {
			ctx.GetLogger().Info("to apply these repairs, run `plakar repair -apply`\n")
		}
	}

	return 0, nil
}

func (cmd *Repair) Lock() (chan bool, error) {
	lockDone := make(chan bool)
	lock := repository.NewExclusiveLock(cmd.repository.AppContext().Hostname)

	buffer := &bytes.Buffer{}
	err := lock.SerializeToStream(buffer)
	if err != nil {
		return nil, err
	}

	_, err = cmd.repository.PutLock(cmd.repairID, buffer)
	if err != nil {
		return nil, err
	}

	// We installed the lock, now let's see if there is a conflicting exclusive lock or not.
	locksID, err := cmd.repository.GetLocks()
	if err != nil {
		// We still need to delete it, and we need to do so manually.
		cmd.repository.DeleteLock(cmd.repairID)
		return nil, err
	}

	for _, lockID := range locksID {
		if lockID == cmd.repairID {
			continue
		}

		rd, err := cmd.repository.GetLock(lockID)
		if err != nil {
			cmd.repository.DeleteLock(cmd.repairID)
			return nil, err
		}

		lock, err := repository.NewLockFromStream(rd)
		rd.Close()
		if err != nil {
			cmd.repository.DeleteLock(cmd.repairID)
			return nil, err
		}

		/* Kick out stale locks */
		if lock.IsStale() {
			err := cmd.repository.DeleteLock(lockID)
			if err != nil {
				cmd.repository.DeleteLock(cmd.repairID)
				return nil, err
			}

			continue
		}

		// There is a lock in place, we need to abort.
		err = cmd.repository.DeleteLock(cmd.repairID)
		if err != nil {
			return nil, err
		}

		return nil, fmt.Errorf("can't take exclusive lock, repository is already locked")
	}

	// The following bit is a "ping" mechanism, Lock() is a bit badly named at this point,
	// we are just refreshing the existing lock so that the watchdog doesn't removes us.
	go func() {
		for {
			select {
			case <-lockDone:
				cmd.repository.DeleteLock(cmd.repairID)
				return
			case <-time.After(repository.LOCK_REFRESH_RATE):
				lock := repository.NewExclusiveLock(cmd.repository.AppContext().Hostname)

				buffer := &bytes.Buffer{}

				// We ignore errors here on purpose, it's tough to handle them
				// correctly, and if they happen we will be ripped by the
				// watchdog anyway.
				lock.SerializeToStream(buffer)
				cmd.repository.PutLock(cmd.repairID, buffer)
			}
		}
	}()

	return lockDone, nil
}

func (cmd *Repair) Unlock(ping chan bool) {
	close(ping)
}
