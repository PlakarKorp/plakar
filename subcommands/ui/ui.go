//go:build go1.16
// +build go1.16

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

package ui

import (
	"fmt"
	"os"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	v2 "github.com/PlakarKorp/plakar/ui/v2"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

type Ui struct {
	subcommands.SubcommandBase

	Addr      string
	Cors      bool
	NoAuth    bool
	NoSpawn   bool
	NoRefresh bool
	Cert      string
	Key       string
}

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Ui{} }, 0, "ui")
}

func (cmd *Ui) CobraCommand() *cobra.Command {
	c := &cobra.Command{
		Use: "ui [OPTIONS]",
	}
	c.Flags().StringVar(&cmd.Addr, "addr", "", "address to listen on (default: random port on localhost)")
	c.Flags().BoolVar(&cmd.Cors, "cors", false, "enable CORS")
	c.Flags().BoolVar(&cmd.NoAuth, "no-auth", false, "don't use authentication")
	c.Flags().BoolVar(&cmd.NoSpawn, "no-spawn", false, "don't spawn browser")
	c.Flags().BoolVar(&cmd.NoRefresh, "no-refresh", false, "don't refresh the local state")
	c.Flags().StringVar(&cmd.Cert, "cert", "", "Full certificate chain")
	c.Flags().StringVar(&cmd.Key, "key", "", "Certificate private key")
	return c
}

func (cmd *Ui) Parse(ctx *appcontext.AppContext, args []string) error {
	rest, err := subcommands.ParseCobra(cmd, args)
	if err != nil {
		return err
	}

	if len(rest) > 0 {
		return fmt.Errorf("too many arguments")
	}

	cmd.RepositorySecret = ctx.GetSecret()

	return nil
}

func (cmd *Ui) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	ui_opts := v2.UiOptions{
		NoSpawn:   cmd.NoSpawn,
		NoRefresh: cmd.NoRefresh,
		Cors:      cmd.Cors,
		Token:     "",
		Cert:      cmd.Cert,
		Key:       cmd.Key,
	}

	if !cmd.NoAuth {
		if uiToken := os.Getenv("PLAKAR_UI_TOKEN"); uiToken != "" {
			ui_opts.Token = uiToken
		} else {
			ui_opts.Token = uuid.NewString()
		}
	}

	err := v2.Ui(repo, ctx, cmd.Addr, &ui_opts)
	if err != nil {
		fmt.Fprintf(ctx.Stderr, "ui: %s\n", err)
		return 1, err
	}
	return 0, err
}
