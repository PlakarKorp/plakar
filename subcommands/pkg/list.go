/*
 * Copyright (c) 2025 Eric Faurot <eric.faurot@plakar.io>
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
	"flag"
	"fmt"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/plugins"
	"github.com/PlakarKorp/plakar/subcommands"
)

type PkgList struct {
	subcommands.SubcommandBase
	LongName bool
	ListAll  bool
}

func (cmd *PkgList) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("pkg list", flag.ExitOnError)
	flags.BoolVar(&cmd.LongName, "long", false, "show full package name")
	flags.BoolVar(&cmd.ListAll, "available", false, "list available prebuilt packages")
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.Parse(args)

	if flags.NArg() != 0 {
		return fmt.Errorf("too many arguments")
	}

	return nil
}

func (cmd *PkgList) Execute(ctx *appcontext.AppContext, _ *repository.Repository) (int, error) {
	var packages []plugins.Package
	var err error

	if cmd.ListAll {
		var filter plugins.IntegrationFilter
		integrations, err := ctx.GetPlugins().ListIntegrations(filter)
		if err != nil {
			return 1, err
		}
		for _, int := range integrations {
			if int.Installation.Available {
				pkg := ctx.GetPlugins().IntegrationAsPackage(&int)
				packages = append(packages, pkg)
			}
		}
	} else {
		packages, err = ctx.GetPlugins().ListInstalledPackages()
		if err != nil {
			return 1, err
		}
	}

	for _, pkg := range packages {
		if cmd.LongName {
			fmt.Println(pkg.PkgName())
		} else {
			fmt.Println(pkg.PkgNameAndVersion())
		}
	}

	return 0, nil
}
