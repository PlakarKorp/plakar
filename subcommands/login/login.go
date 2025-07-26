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
	subcommands.Register(func() subcommands.Subcommand { return &Login{} }, subcommands.AgentSupport, "login")
}

func (cmd *Login) Parse(ctx *appcontext.AppContext, args []string) error {
	var opt_nospawn bool
	var opt_github bool
	var opt_email string

	flags := flag.NewFlagSet("login", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.BoolVar(&opt_nospawn, "no-spawn", false, "don't spawn browser")
	flags.BoolVar(&opt_github, "github", false, "login with GitHub")
	flags.StringVar(&opt_email, "email", "", "login with email")
	flags.Parse(args)

	if opt_github && opt_email != "" {
		return fmt.Errorf("specify either -github or -email, not both")
	}

	if !opt_github && opt_email == "" {
		fmt.Println("no provided login method, defaulting to GitHub")
		opt_github = true
	}

	if opt_nospawn && !opt_github {
		return fmt.Errorf("the -no-spawn option is only valid with -github")
	}

	cmd.Github = opt_github
	cmd.Email = opt_email
	cmd.NoSpawn = opt_nospawn
	cmd.RepositorySecret = ctx.GetSecret()

	return nil
}

type Login struct {
	subcommands.SubcommandBase

	Github  bool
	Email   string
	NoSpawn bool
}

func (cmd *Login) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	var err error

	if cmd.Email != "" {
		if addr, err := utils.ValidateEmail(cmd.Email); err != nil {
			return 1, fmt.Errorf("invalid email address: %w", err)
		} else {
			cmd.Email = addr
		}
	}

	flow, err := plogin.NewLoginFlow(ctx, repo.Configuration().RepositoryID, cmd.NoSpawn)
	if err != nil {
		return 1, err
	}
	defer flow.Close()

	var token string
	if cmd.Github {
		token, err = flow.Run("github", map[string]string{"repository_id": repo.Configuration().RepositoryID.String()})
	} else if cmd.Email != "" {
		token, err = flow.Run("email", map[string]string{"email": cmd.Email, "repository_id": repo.Configuration().RepositoryID.String()})
	} else {
		return 1, fmt.Errorf("invalid login method")
	}
	if err != nil {
		return 1, err
	}

	if err := ctx.GetCookies().PutAuthToken(token); err != nil {
		return 1, fmt.Errorf("failed to store token in cache: %w", err)
	}

	return 0, nil
}
