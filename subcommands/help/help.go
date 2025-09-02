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

package help

import (
	"embed"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/charmbracelet/glamour"
	"github.com/muesli/termenv"
	"golang.org/x/term"
)

//go:embed docs/*
var docs embed.FS

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Help{} }, subcommands.BeforeRepositoryOpen, "help")
}

func (cmd *Help) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("help", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
		fmt.Fprint(flags.Output(), "\nTo view the man page for a specific command, run 'plakar help SUBCOMMAND'.\n")
	}
	flags.StringVar(&cmd.Style, "style", "dracula", "style to use")
	flags.Parse(args)

	cmd.Command = strings.Join(flags.Args(), "-")
	return nil
}

type Help struct {
	subcommands.SubcommandBase

	Style   string
	Command string
}

func (cmd *Help) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	document := "docs/plakar.md"
	if cmd.Command != "" {
		document = fmt.Sprintf("docs/plakar-%s.md", cmd.Command)
	}

	content, err := docs.ReadFile(document)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd.Command)
		return 1, err
	}

	disableColors := false
	if _, nocolor := os.LookupEnv("NO_COLOR"); nocolor {
		disableColors = true
	} else if !term.IsTerminal(int(os.Stdout.Fd())) {
		disableColors = true
	}

	options := []glamour.TermRendererOption{}
	if disableColors {
		options = []glamour.TermRendererOption{
			glamour.WithColorProfile(termenv.Ascii),
		}
	} else {
		options = []glamour.TermRendererOption{
			glamour.WithStandardStyle(cmd.Style),
			glamour.WithColorProfile(termenv.TrueColor),
		}
	}
	r, err := glamour.NewTermRenderer(
		options...,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create renderer: %s\n", err)
		return 1, err
	}

	out, err := r.RenderBytes(content)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to render: %s\n", err)
		return 1, err
	}

	fmt.Print(string(out))

	return 0, err
}
