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

package services

import (
	"flag"
	"fmt"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/services"
	"github.com/PlakarKorp/plakar/subcommands"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Services{} }, subcommands.AgentSupport, "services")
}

func (cmd *Services) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("services", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}
	flags.Parse(args)

	if flags.NArg() != 2 {
		flags.Usage()
		return fmt.Errorf("invalid number of arguments")
	}

	action := flags.Arg(0)
	parameter := flags.Arg(1)

	if action != "enable" && action != "disable" && action != "status" {
		flags.Usage()
		return fmt.Errorf("invalid action: %s, should be enable, disable, or status", action)
	}

	cmd.Action = action
	cmd.Parameter = parameter
	return nil
}

type Services struct {
	subcommands.SubcommandBase

	Action    string
	Parameter string
}

func (cmd *Services) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	if authToken, err := ctx.GetCookies().GetAuthToken(); err != nil {
		return 1, err
	} else if authToken == "" {
		return 1, fmt.Errorf("access to services requires login, please run `plakar login`")
	} else {
		switch cmd.Action {
		case "status":
			sc := services.NewServiceConnector(ctx, authToken)
			status, err := sc.GetServiceStatus(cmd.Parameter)
			if err != nil {
				return 1, err
			}
			if status {
				fmt.Fprintf(ctx.Stdout, "status: enabled\n")
			} else {
				fmt.Fprintf(ctx.Stdout, "status: disabled\n")
			}

			config, err := sc.GetServiceConfiguration(cmd.Parameter)
			if err != nil {
				return 1, err
			}
			if len(config) == 0 {
				fmt.Fprintf(ctx.Stdout, "no configuration\n")
				return 0, nil
			}
			fmt.Fprintf(ctx.Stdout, "\n")
			fmt.Fprintf(ctx.Stdout, "configuration:\n")
			for k, v := range config {
				fmt.Fprintf(ctx.Stdout, "- %s: %s\n", k, v)
			}

		case "enable":
			sc := services.NewServiceConnector(ctx, authToken)
			err := sc.SetServiceStatus(cmd.Parameter, true)
			if err != nil {
				return 1, err
			}
			fmt.Fprintf(ctx.Stdout, "enabled\n")

		case "disable":
			sc := services.NewServiceConnector(ctx, authToken)
			err := sc.SetServiceStatus(cmd.Parameter, false)
			if err != nil {
				return 1, err
			}
			fmt.Fprintf(ctx.Stdout, "disabled\n")
		}
		return 0, nil
	}
}
