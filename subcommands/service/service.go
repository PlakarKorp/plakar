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
	subcommands.MustRegister(func() subcommands.Subcommand { return &ServiceList{} }, subcommands.AgentSupport, "service", "list")
	subcommands.MustRegister(func() subcommands.Subcommand { return &ServiceStatus{} }, subcommands.AgentSupport, "service", "status")
	subcommands.MustRegister(func() subcommands.Subcommand { return &ServiceEnable{} }, subcommands.AgentSupport, "service", "enable")
	subcommands.MustRegister(func() subcommands.Subcommand { return &ServiceDisable{} }, subcommands.AgentSupport, "service", "disable")
	subcommands.MustRegister(func() subcommands.Subcommand { return &ServiceSet{} }, subcommands.AgentSupport, "service", "set")
	subcommands.MustRegister(func() subcommands.Subcommand { return &ServiceUnset{} }, subcommands.AgentSupport, "service", "unset")
	subcommands.MustRegister(func() subcommands.Subcommand { return &ServiceAdd{} }, subcommands.AgentSupport, "service", "add")
	subcommands.MustRegister(func() subcommands.Subcommand { return &ServiceRm{} }, subcommands.AgentSupport, "service", "rm")
	subcommands.MustRegister(func() subcommands.Subcommand { return &ServiceShow{} }, subcommands.AgentSupport, "service", "show")
	subcommands.MustRegister(func() subcommands.Subcommand { return &Service{} }, subcommands.BeforeRepositoryOpen, "service")
}

type Service struct {
	subcommands.SubcommandBase
}

func (_ *Service) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("service", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s list\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s add <name> <key>=<value>...\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s rm <name>\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s set <name> <key>=<value>...\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s unset <name> <key>...\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s status <name>\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s enable <name>\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s disable <name>\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s show <name>\n", flags.Name())
	}
	flags.Parse(args)

	if flags.NArg() > 0 {
		return fmt.Errorf("invalid argument: %s", flags.Arg(0))
	}
	return fmt.Errorf("no action specified")
}

func (cmd *Service) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	return 1, fmt.Errorf("no action specified")
}

func getClient(ctx *appcontext.AppContext) (*services.ServiceConnector, error) {
	authToken, err := ctx.GetCookies().GetAuthToken()
	if err != nil {
		return nil, err
	} else if authToken == "" {
		return nil, fmt.Errorf("access to services requires login, please run `plakar login`")
	}

	return services.NewServiceConnector(ctx, authToken), nil
}
