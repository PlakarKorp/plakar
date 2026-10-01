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

package dup

import (
	"fmt"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/spf13/cobra"
)

type Dup struct {
	subcommands.SubcommandBase

	SnapshotIDS []string
}

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Dup{} }, 0, "dup")
}

func (cmd *Dup) CobraCommand() *cobra.Command {
	return &cobra.Command{
		Use: "dup [OPTIONS] [SNAPSHOT[:PATH]]...",
	}
}

func (cmd *Dup) Parse(ctx *appcontext.AppContext, args []string) error {
	rest, err := subcommands.ParseCobra(cmd, args)
	if err != nil {
		return err
	}

	if len(rest) == 0 {
		return fmt.Errorf("at least one parameter is required")
	}

	cmd.SnapshotIDS = rest

	return nil
}

func (cmd *Dup) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	errors := 0
	for _, snapshotPath := range cmd.SnapshotIDS {
		snap, pathname, err := locate.OpenSnapshotByPath(repo, snapshotPath)
		if err != nil {
			ctx.GetLogger().Error("digest: %s: %s", pathname, err)
			errors++
			continue
		}

		newSnap, err := snap.Dup(&snapshot.BuilderOptions{})
		if err != nil {
			ctx.GetLogger().Error("dup: %s: %s", pathname, err)
			errors++
			continue
		}
		newSnap.Close()

		snap.Close()
	}

	return 0, nil
}
