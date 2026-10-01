/*
 * Copyright (c) 2025 Omar Polo <omar.polo@plakar.io>
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
	"os"
	"path/filepath"
	"strings"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/pkg"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/spf13/cobra"
)

type PkgAdd struct {
	subcommands.SubcommandBase

	upgrade       bool
	devel         bool
	allowUnsigned bool
	container     bool
	Args          []string
}

func (cmd *PkgAdd) CobraCommand() *cobra.Command {
	c := &cobra.Command{
		Use: "pkg add",
	}
	c.Flags().BoolVar(&cmd.upgrade, "u", false, "Update packages")
	c.Flags().BoolVar(&cmd.devel, "devel", false, "Opt-in to the devel integration tree")
	c.Flags().BoolVar(&cmd.allowUnsigned, "allow-unsigned", false,
		"Install packages that carry no signature")
	c.Flags().BoolVar(&cmd.container, "container", false, "Install the container flavor of remote packages")
	return c
}

func (cmd *PkgAdd) Parse(ctx *appcontext.AppContext, args []string) error {
	rest, err := subcommands.ParseCobra(cmd, args)
	if err != nil {
		return err
	}

	if len(rest) < 1 && !cmd.upgrade {
		return fmt.Errorf("not enough arguments")
	}

	cmd.Args = rest
	for i, name := range cmd.Args {
		absolute := name
		if !filepath.IsAbs(absolute) {
			absolute = filepath.Join(ctx.CWD, absolute)
		}

		if info, err := os.Stat(absolute); err == nil && info.Mode().IsRegular() {
			cmd.Args[i] = absolute
			continue
		}

		if filepath.IsAbs(name) {
			return fmt.Errorf("file not found: %s", name)
		}
	}

	return nil
}

func (cmd *PkgAdd) Execute(ctx *appcontext.AppContext, _ *repository.Repository) (int, error) {
	pkgmgr := ctx.GetPkgManager()

	var edition string
	if cmd.devel {
		edition = "devel"
	}

	if cmd.allowUnsigned {
		verifier := ctx.GetPkgVerifier()
		if verifier == nil {
			return 1, fmt.Errorf("-allow-unsigned: no package verifier configured")
		}

		verifier.SetAllowUnsigned(true)

		fmt.Fprintln(ctx.Stderr, "WARNING: -allow-unsigned: installing unsigned packages.")
		fmt.Fprintln(ctx.Stderr, "WARNING: their contents and origin cannot be verified.")
	}

	for _, plugin := range cmd.Args {
		plugin, version, _ := strings.Cut(plugin, "@")
		if version == "latest" {
			// "latest" spells out the default: leave the version empty so
			// the manager picks it from the recipe instead of looking for
			// a release named "latest".
			version = ""
		}
		addopts := pkg.AddOptions{
			Edition:       edition,
			ImplicitFetch: true,
			Version:       version,
			Upgrade:       cmd.upgrade,
		}
		if cmd.container {
			addopts.Container = true
		}
		if err := pkgmgr.Add(plugin, &addopts); err != nil {
			if cmd.upgrade && errors.Is(err, pkg.ErrAlreadyInstalled) {
				continue
			}
			if errors.Is(err, pkg.ErrAuthorizationRequired) {
				return 1, fmt.Errorf("failed to install %s: authentication required. Run `plakar login` and try again", filepath.Base(plugin))
			}
			return 1, fmt.Errorf("failed to install %s: %w",
				filepath.Base(plugin), err)
		}
	}

	if len(cmd.Args) == 0 && cmd.upgrade {
		for plugin, err := range pkgmgr.List() {
			if err != nil {
				return 1, err
			}

			addopts := pkg.AddOptions{
				Edition:       edition,
				ImplicitFetch: true,
				Upgrade:       cmd.upgrade,
				// Keep whatever flavor is installed rather than
				// silently switching a container install to native.
				Container: plugin.OperatingSystem == pkg.OSContainer,
			}

			if err := pkgmgr.Add(plugin.Name, &addopts); err != nil {
				if errors.Is(err, pkg.ErrAlreadyInstalled) {
					continue
				}
				return 1, fmt.Errorf("failed to update %s: %w",
					plugin.Name, err)
			}
		}
	}

	return 0, nil
}
