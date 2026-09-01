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
	"runtime"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/pkg"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
)

type PkgList struct {
	subcommands.SubcommandBase
	devel    bool
	LongName bool
	ListAll  bool
	Signers  bool
}

func (cmd *PkgList) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("pkg list", flag.ExitOnError)
	flags.BoolVar(&cmd.LongName, "long", false, "show full package name")
	flags.BoolVar(&cmd.ListAll, "available", false, "list available prebuilt packages")
	flags.BoolVar(&cmd.Signers, "signers", false, "show which key signed each installed package")
	flags.BoolVar(&cmd.devel, "devel", false, "use the devel tree")
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
	pkgmgr := ctx.GetPkgManager()

	var edition string
	if cmd.devel {
		edition = "devel"
	}

	// signer reports who signed an installed package. Reported from the
	// signature retained at install time, so it needs no network and
	// stays accurate for a package whose key has since rotated out.
	signer := func(name, version string) string {
		verifier := ctx.GetPkgVerifier()
		if verifier == nil {
			return ""
		}

		sig, err := pkgmgr.Signature(&pkg.Package{
			Name:            name,
			Version:         version,
			OperatingSystem: runtime.GOOS,
			Architecture:    runtime.GOARCH,
		})
		if err != nil || sig == nil {
			return "unsigned"
		}

		return verifier.Signer(sig)
	}

	print := func(name, version, os, arch string) {
		if cmd.LongName {
			fmt.Fprintf(ctx.Stdout, "%s_%s_%s_%s.ptar", name, version, os, arch)
		} else {
			fmt.Fprintf(ctx.Stdout, "%s@%s", name, version)
		}

		if cmd.Signers && !cmd.ListAll {
			fmt.Fprintf(ctx.Stdout, "\t%s", signer(name, version))
		}

		fmt.Fprintln(ctx.Stdout)
	}

	integrations, err := pkgmgr.Query(&pkg.QueryOptions{
		OnlyLocal: !cmd.ListAll,
		Edition:   edition,
	})
	if err != nil {
		return 1, err
	}
	for _, integration := range integrations {
		version := integration.Installation.Version
		if integration.LatestVersion != "" {
			version = integration.LatestVersion
		}

		print(integration.Name, version, runtime.GOOS, runtime.GOARCH)
	}

	return 0, nil
}
