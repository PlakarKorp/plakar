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

package login

import (
	_ "embed"
	"flag"
	"fmt"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	plogin "github.com/PlakarKorp/plakar/login"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Login{} }, subcommands.BeforeRepositoryOpen, "login")
}

func (cmd *Login) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("login", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.BoolVar(&cmd.Status, "status", false, "do not login, just display the status")
	flags.BoolVar(&cmd.NoSpawn, "no-spawn", false, "don't spawn browser")
	flags.BoolVar(&cmd.Github, "github", false, "login with GitHub")
	flags.StringVar(&cmd.Email, "email", "", "login with email")
	flags.BoolVar(&cmd.Env, "env", false, "use token from environment variable PLAKAR_TOKEN")
	flags.Parse(args)

	if flags.NArg() > 0 {
		return fmt.Errorf("too many arguments")
	}

	if cmd.Status {
		if cmd.Github || cmd.Email != "" || cmd.NoSpawn || cmd.Env {
			return fmt.Errorf("the -status option must be used alone")
		}
	} else {
		if cmd.Github {
			if cmd.Email != "" || cmd.Env {
				return fmt.Errorf("the -github option cannot be used with -email or -env")
			}
		} else if cmd.Email != "" {
			if cmd.Env {
				return fmt.Errorf("the -email option cannot be used with -env")
			}
			if addr, err := utils.ValidateEmail(cmd.Email); err != nil {
				return fmt.Errorf("invalid email address: %w", err)
			} else {
				cmd.Email = addr
			}
		} else if !cmd.Env {
			fmt.Println("no provided login method, defaulting to GitHub")
			cmd.Github = true
		}
		if cmd.NoSpawn && !cmd.Github {
			return fmt.Errorf("the -no-spawn option is only valid with -github")
		}
	}

	return nil
}

type Login struct {
	subcommands.SubcommandBase

	Status  bool
	Github  bool
	Email   string
	Env     bool
	NoSpawn bool
}

func (cmd *Login) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	if cmd.Status {
		token, _ := ctx.GetCookies().GetAuthToken()
		status := "not logged in"
		if token != "" {
			status = "logged in"
		}
		fmt.Fprintf(ctx.Stdout, "%s\n", status)
		return 0, nil
	}

	var token string

	if cmd.Env {
		if token = ctx.GetCookies().GetAuthEnvToken(); token == "" {
			return 1, fmt.Errorf("no auth token found in environment variable PLAKAR_TOKEN")
		}
	} else {
		flow, err := plogin.NewLoginFlow(ctx, cmd.NoSpawn)
		if err != nil {
			return 1, err
		}
		defer flow.Close()

		if cmd.Github {
			token, err = flow.Run("github", map[string]string{})
		} else if cmd.Email != "" {
			token, err = flow.Run("email", map[string]string{"email": cmd.Email})
		} else {
			return 1, fmt.Errorf("invalid login method")
		}
		if err != nil {
			return 1, err
		}
	}

	if err := ctx.GetCookies().PutAuthToken(token); err != nil {
		return 1, fmt.Errorf("failed to store token in cache: %w", err)
	}

	return 0, nil
}
