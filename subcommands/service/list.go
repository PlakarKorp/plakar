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

type ServiceList struct {
	subcommands.SubcommandBase
}

func (cmd *ServiceList) CobraCommand() *cobra.Command {
	return &cobra.Command{
		Use: "service list",
	}
}

func (cmd *ServiceList) Parse(ctx *appcontext.AppContext, args []string) error {
	rest, err := subcommands.ParseCobra(cmd, args)
	if err != nil {
		return err
	}

	if len(rest) > 0 {
		return fmt.Errorf("invalid argument: %s", rest[0])
	}

	return nil
}

func (cmd *ServiceList) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	sc, err := getClient(ctx)
	if err != nil {
		return 1, err
	}

	list, err := sc.GetServiceList()
	if err != nil {
		return 1, err
	}
	for _, svc := range list {
		fmt.Fprintf(ctx.Stdout, "%s\n", svc.Name)
	}

	return 0, nil
}
