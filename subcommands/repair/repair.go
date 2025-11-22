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

type Repair struct {
	subcommands.SubcommandBase

	Apply bool
}

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Repair{} }, subcommands.AgentSupport, "repair")
}

func (cmd *Repair) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("repair", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}
	flags.BoolVar(&cmd.Apply, "apply", false, "do the actual repair")
	flags.Parse(args)

	cmd.RepositorySecret = ctx.GetSecret()

	return nil
}

func (cmd *Repair) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
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
		ctx.GetLogger().Info("repairing missing remote state %x\n", stateID)

		if !cmd.Apply {
			continue
		}

		scanCache, err := repo.AppContext().GetCache().Scan(stateID)
		if err != nil {
			return 1, err
		}

		deltaState := state.NewLocalState(scanCache)

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

	return 0, nil
}
