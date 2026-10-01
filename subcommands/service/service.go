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
	"fmt"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/services"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/spf13/cobra"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &ServiceList{} }, 0, "service", "list")
	subcommands.Register(func() subcommands.Subcommand { return &ServiceStatus{} }, 0, "service", "status")
	subcommands.Register(func() subcommands.Subcommand { return &ServiceEnable{} }, 0, "service", "enable")
	subcommands.Register(func() subcommands.Subcommand { return &ServiceDisable{} }, 0, "service", "disable")
	subcommands.Register(func() subcommands.Subcommand { return &ServiceSet{} }, 0, "service", "set")
	subcommands.Register(func() subcommands.Subcommand { return &ServiceUnset{} }, 0, "service", "unset")
	subcommands.Register(func() subcommands.Subcommand { return &ServiceAdd{} }, 0, "service", "add")
	subcommands.Register(func() subcommands.Subcommand { return &ServiceRm{} }, 0, "service", "rm")
	subcommands.Register(func() subcommands.Subcommand { return &ServiceShow{} }, 0, "service", "show")
	subcommands.Register(func() subcommands.Subcommand { return &Service{} }, subcommands.BeforeRepositoryOpen, "service")
}

type Service struct {
	subcommands.SubcommandBase
}

func (cmd *Service) CobraCommand() *cobra.Command {
	return &cobra.Command{
		Use: "service",
	}
}

func (cmd *Service) Parse(ctx *appcontext.AppContext, args []string) error {
	rest, err := subcommands.ParseCobra(cmd, args)
	if err != nil {
		return err
	}

	if len(rest) > 0 {
		return fmt.Errorf("invalid argument: %s", rest[0])
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
