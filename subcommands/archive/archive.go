/*
 * Copyright (c) 2023 Gilles Chehade <gilles@poolp.org>
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

package archive

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
)

func init() {
	subcommands.Register(Archive, 0, "archive")
}

func Archive(ctx *appcontext.AppContext, repo *repository.Repository, args []string) error {
	var (
		rebase bool
		output string
		format string
	)

	flags := flag.NewFlagSet("archive", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS] [SNAPSHOT[:PATH]]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.StringVar(&output, "output", "", "archive pathname")
	flags.BoolVar(&rebase, "rebase", false, "strip pathname when pulling")
	flags.StringVar(&format, "format", "tarball", "archive format: tar, tarball, zip")
	flags.Parse(args)

	if flags.NArg() == 0 {
		return fmt.Errorf("need at least one snapshot ID to pull")
	}

	prefix := flags.Arg(0)

	supportedFormats := map[string]string{
		"tar":     "tar",
		"tarball": "tar.gz",
		"zip":     "zip",
	}
	if _, ok := supportedFormats[format]; !ok {
		return fmt.Errorf("unsupported format %s", format)
	}

	if output == "" {
		output = fmt.Sprintf("plakar-%s.%s", time.Now().UTC().Format(time.RFC3339), supportedFormats[format])
	}

	snap, pathname, err := locate.OpenSnapshotByPath(repo, prefix)
	if err != nil {
		return fmt.Errorf("archive: could not open snapshot: %s", prefix)
	}
	defer snap.Close()

	out := os.Stdout
	if output != "-" {
		out, err = os.Create(output)
		if err != nil {
			return fmt.Errorf("failed to create %s: %w", output, err)
		}
		defer out.Close()
	}

	if err = snap.Archive(out, format, []string{pathname}, rebase); err != nil {
		return err
	}

	return nil
}
