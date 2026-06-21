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
	subcommands.Register(Login, subcommands.BeforeRepositoryOpen, "login")
}

func Login(ctx *appcontext.AppContext, repo *repository.Repository, args []string) error {
	var (
		status  bool
		github  bool
		email   string
		env     bool
		nospawn bool
	)

	flags := flag.NewFlagSet("login", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.BoolVar(&status, "status", false, "do not login, just display the status")
	flags.BoolVar(&nospawn, "no-spawn", false, "don't spawn browser")
	flags.BoolVar(&github, "github", false, "login with GitHub")
	flags.StringVar(&email, "email", "", "login with email")
	flags.BoolVar(&env, "env", false, "use token from environment variable PLAKAR_TOKEN")
	flags.Parse(args)

	if flags.NArg() > 0 {
		return fmt.Errorf("too many arguments")
	}

	if status {
		if github || email != "" || nospawn || env {
			return fmt.Errorf("the -status option must be used alone")
		}
	} else {
		if github {
			if email != "" || env {
				return fmt.Errorf("the -github option cannot be used with -email or -env")
			}
		} else if email != "" {
			if env {
				return fmt.Errorf("the -email option cannot be used with -env")
			}
			addr, err := utils.ValidateEmail(email)
			if err != nil {
				return fmt.Errorf("invalid email address: %w", err)
			}
			email = addr
		} else if !env {
			fmt.Println("no provided login method, defaulting to GitHub")
			github = true
		}
		if nospawn && !github {
			return fmt.Errorf("the -no-spawn option is only valid with -github")
		}
	}

	if status {
		token, _ := ctx.GetCookies().GetAuthToken()
		status := "not logged in"
		if token != "" {
			status = "logged in"
		}
		fmt.Fprintf(ctx.Stdout, "%s\n", status)
		return nil
	}

	var token string

	if env {
		if token = ctx.GetCookies().GetAuthEnvToken(); token == "" {
			return fmt.Errorf("no auth token found in environment variable PLAKAR_TOKEN")
		}
	} else {
		flow, err := plogin.NewLoginFlow(ctx, nospawn)
		if err != nil {
			return err
		}
		defer flow.Close()

		if github {
			token, err = flow.Run("github", map[string]string{})
		} else if email != "" {
			token, err = flow.Run("email", map[string]string{"email": email})
		} else {
			return fmt.Errorf("invalid login method")
		}
		if err != nil {
			return err
		}
	}

	if err := ctx.GetCookies().PutAuthToken(token); err != nil {
		return fmt.Errorf("failed to store token in cache: %w", err)
	}

	return nil
}
