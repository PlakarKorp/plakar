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
	"flag"
	"fmt"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/services"
	"github.com/PlakarKorp/plakar/subcommands"
)

type ServicesDisable struct {
	subcommands.SubcommandBase

	Service string
}

func (cmd *ServicesDisable) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("services-disable", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: service disable SERVICE_NAME\n")
	}
	flags.Parse(args)

	if flags.NArg() != 1 {
		return fmt.Errorf("invalid number of arguments, expected 1 but got %d", flags.NArg())
	}

	cmd.Service = flags.Arg(0)

	return nil
}

func (cmd *ServicesDisable) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	authToken, err := ctx.GetCookies().GetAuthToken()
	if err != nil {
		return 1, err
	} else if authToken == "" {
		return 1, fmt.Errorf("access to services requires login, please run `plakar login`")
	}

	sc := services.NewServiceConnector(ctx, authToken)
	err = sc.SetServiceStatus(cmd.Service, false)
	if err != nil {
		return 1, err
	}
	fmt.Fprintf(ctx.Stdout, "disabled\n")

	return 0, nil
}
