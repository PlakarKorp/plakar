/*
 * Copyright (c) 2026 Cedric Grard <cedric.grard@gmail.com>
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

package pkg

import (
	"errors"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/pkg"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/config"
	"github.com/PlakarKorp/plakar/signify"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/spf13/cobra"
)

const (
	// OfficialDistURL is the official package distribution tree.
	OfficialDistURL = "https://plakar.io/dist/plugins/kloset/"

	// DefaultEdition is the edition packages are installed from
	// unless -devel is given.
	DefaultEdition = "community"
)

const registryUsage = "usage: plakar pkg registry add <name> <url> | rm <name> | list"

type PkgRegistry struct {
	subcommands.SubcommandBase
}

func (cmd *PkgRegistry) CobraCommand() *cobra.Command {
	return &cobra.Command{
		Use: "pkg registry",
	}
}

func (cmd *PkgRegistry) Parse(ctx *appcontext.AppContext, args []string) error {
	if _, err := subcommands.ParseCobra(cmd, args); err != nil {
		return err
	}
	return errors.New(registryUsage)
}

func (cmd *PkgRegistry) Execute(ctx *appcontext.AppContext, _ *repository.Repository) (int, error) {
	return 1, errors.New(registryUsage)
}

type PkgRegistryAdd struct {
	subcommands.SubcommandBase

	Name string
	URL  string
}

func (cmd *PkgRegistryAdd) CobraCommand() *cobra.Command {
	return &cobra.Command{
		Use: "pkg registry add NAME URL",
	}
}

func (cmd *PkgRegistryAdd) Parse(ctx *appcontext.AppContext, args []string) error {
	rest, err := subcommands.ParseCobra(cmd, args)
	if err != nil {
		return err
	}
	if len(rest) != 2 {
		return fmt.Errorf("usage: plakar pkg registry add <name> <url>")
	}
	cmd.Name, cmd.URL = rest[0], rest[1]
	return nil
}

func (cmd *PkgRegistryAdd) Execute(ctx *appcontext.AppContext, _ *repository.Repository) (int, error) {
	u, err := url.Parse(cmd.URL)
	if err != nil {
		return 1, fmt.Errorf("registry %q: %w", cmd.Name, err)
	}
	if u.Scheme != "https" {
		return 1, fmt.Errorf("registry %q: the URL must be https", cmd.Name)
	}

	regs := append(ctx.Config.PkgRegistries(), pkg.Registry{Name: cmd.Name, URL: cmd.URL})
	if err := pkg.CheckRegistries(regs); err != nil {
		return 1, err
	}

	trust, err := signify.LoadTrustStore(ctx.ConfigDir)
	if err != nil {
		return 1, fmt.Errorf("failed to load the package trust store: %w", err)
	}

	if ctx.Config.Registries == nil {
		ctx.Config.Registries = make(map[string]config.RegistryConfig)
	}
	ctx.Config.Registries[cmd.Name] = config.RegistryConfig{URL: cmd.URL}
	if err := config.Save(ctx.ConfigDir, ctx.Config); err != nil {
		return 1, err
	}

	origin := u.JoinPath(DefaultEdition).String()
	if len(trust.For(origin)) == 0 {
		dir := filepath.Join(ctx.ConfigDir, "trust")
		fmt.Fprintf(ctx.Stderr, "No trusted key covers registry %s yet: its packages will be refused.\n", cmd.Name)
		fmt.Fprintf(ctx.Stderr, "To trust its signing key, copy it to %s and\n",
			filepath.Join(dir, "<key>.pub"))
		fmt.Fprintf(ctx.Stderr, "write the line %s to %s.\n", cmd.URL,
			filepath.Join(dir, "<key>.scope"))
	}

	return 0, nil
}

type PkgRegistryRm struct {
	subcommands.SubcommandBase

	Name string
}

func (cmd *PkgRegistryRm) CobraCommand() *cobra.Command {
	return &cobra.Command{
		Use: "pkg registry rm NAME",
	}
}

func (cmd *PkgRegistryRm) Parse(ctx *appcontext.AppContext, args []string) error {
	rest, err := subcommands.ParseCobra(cmd, args)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return fmt.Errorf("usage: plakar pkg registry rm <name>")
	}
	cmd.Name = rest[0]
	return nil
}

func (cmd *PkgRegistryRm) Execute(ctx *appcontext.AppContext, _ *repository.Repository) (int, error) {
	if _, ok := ctx.Config.Registries[cmd.Name]; !ok {
		return 1, fmt.Errorf("registry %q does not exist", cmd.Name)
	}
	delete(ctx.Config.Registries, cmd.Name)
	if err := config.Save(ctx.ConfigDir, ctx.Config); err != nil {
		return 1, err
	}
	return 0, nil
}

type PkgRegistryList struct {
	subcommands.SubcommandBase
}

func (cmd *PkgRegistryList) CobraCommand() *cobra.Command {
	return &cobra.Command{
		Use: "pkg registry list",
	}
}

func (cmd *PkgRegistryList) Parse(ctx *appcontext.AppContext, args []string) error {
	rest, err := subcommands.ParseCobra(cmd, args)
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return fmt.Errorf("too many arguments")
	}
	return nil
}

func (cmd *PkgRegistryList) Execute(ctx *appcontext.AppContext, _ *repository.Repository) (int, error) {
	fmt.Fprintf(ctx.Stdout, "official\t%s\n", OfficialDistURL)
	for _, reg := range ctx.Config.PkgRegistries() {
		fmt.Fprintf(ctx.Stdout, "%s\t%s\n", reg.Name, reg.URL)
	}
	return 0, nil
}
