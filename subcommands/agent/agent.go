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

package agent

import (
	"flag"
	"fmt"
	"runtime"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
)

func init() {
	if runtime.GOOS != "windows" {
		subcommands.Register(func() subcommands.Subcommand { return &AgentStop{} },
			subcommands.BeforeRepositoryOpen|subcommands.AgentSupport|subcommands.IgnoreVersion, "agent", "stop")
		subcommands.Register(func() subcommands.Subcommand { return &AgentStart{} },
			subcommands.BeforeRepositoryOpen, "agent", "start")
		subcommands.Register(func() subcommands.Subcommand { return &Agent{} },
			subcommands.BeforeRepositoryOpen, "agent")
		subcommands.Register(func() subcommands.Subcommand { return &Cached{} },
			subcommands.BeforeRepositoryOpen, "cached")
	}
}

func (cmd *Agent) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("agent", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s start | stop\n", flags.Name())
	}
	flags.Parse(args)

	return fmt.Errorf("no action specified")
}

type Agent struct {
	subcommands.SubcommandBase
}

func (cmd *Agent) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	return 0, fmt.Errorf("no action specified")
}
