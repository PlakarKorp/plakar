/*
 * Copyright (c) 2025 Julien Castets <julien.castets@plakar.io>
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

package services

import (
	"fmt"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/spf13/cobra"
)

type ServiceDisable struct {
	subcommands.SubcommandBase

	Service string
}

func (cmd *ServiceDisable) CobraCommand() *cobra.Command {
	return &cobra.Command{
		Use: "service disable",
	}
}

func (cmd *ServiceDisable) Parse(ctx *appcontext.AppContext, args []string) error {
	rest, err := subcommands.ParseCobra(cmd, args)
	if err != nil {
		return err
	}

	if len(rest) != 1 {
		return fmt.Errorf("invalid number of arguments, expected 1 but got %d", len(rest))
	}

	cmd.Service = rest[0]
	cmd.RepositorySecret = ctx.GetSecret()

	return nil
}

func (cmd *ServiceDisable) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	sc, err := getClient(ctx)
	if err != nil {
		return 1, err
	}

	if err := sc.SetServiceStatus(cmd.Service, false); err != nil {
		return 1, err
	}
	fmt.Fprintf(ctx.Stdout, "disabled\n")

	return 0, nil
}
