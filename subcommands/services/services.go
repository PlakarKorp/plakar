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
	"github.com/PlakarKorp/plakar/subcommands"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &ServicesStatus{} }, subcommands.AgentSupport, "services", "status")
	subcommands.Register(func() subcommands.Subcommand { return &ServicesEnable{} }, subcommands.AgentSupport, "services", "enable")
	subcommands.Register(func() subcommands.Subcommand { return &ServicesDisable{} }, subcommands.AgentSupport, "services", "disable")
	subcommands.Register(func() subcommands.Subcommand { return &Services{} }, subcommands.BeforeRepositoryOpen, "services")
}

type Services struct {
	subcommands.SubcommandBase
}

func (_ *Services) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("services", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s status | enable | disable\n", flags.Name())
	}
	flags.Parse(args)

	return fmt.Errorf("no action specified")
}

func (cmd *Services) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	return 1, fmt.Errorf("no action specified")
}
