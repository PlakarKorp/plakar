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
	"io"
	"log"
	"os"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/repository"
)

func init() {
	subcommands.Register("archive", cmd_archive)
}

func cmd_archive(ctx *appcontext.AppContext, repo *repository.Repository, args []string) int {
	var opt_rebase bool
	var opt_output string
	var opt_format string

	flags := flag.NewFlagSet("archive", flag.ExitOnError)
	flags.StringVar(&opt_output, "output", "", "archive pathname")
	flags.BoolVar(&opt_rebase, "rebase", false, "strip pathname when pulling")
	flags.StringVar(&opt_format, "format", "tarball", "archive format")
	flags.Parse(args)

	if flags.NArg() == 0 {
		log.Fatalf("%s: need at least one snapshot ID to pull", flag.CommandLine.Name())
	}

	supportedFormats := map[string]string{
		"tar":     ".tar",
		"tarball": ".tar.gz",
		"zip":     ".zip",
	}
	if _, ok := supportedFormats[opt_format]; !ok {
		log.Fatalf("%s: unsupported format %s", flag.CommandLine.Name(), opt_format)
	}

	snapshotPrefix, pathname := utils.ParseSnapshotID(flags.Arg(0))
	snap, err := utils.OpenSnapshotByPrefix(repo, snapshotPrefix)
	if err != nil {
		log.Fatalf("%s: could not open snapshot: %s", flag.CommandLine.Name(), snapshotPrefix)
	}
	defer snap.Close()

	if opt_output == "" {
		opt_output = fmt.Sprintf("plakar-%s.%s", time.Now().UTC().Format(time.RFC3339), supportedFormats[opt_format])
	}

	var out io.WriteCloser
	if opt_output == "-" {
		out = os.Stdout
	} else {
		tmp, err := os.CreateTemp("", "plakar-archive-")
		if err != nil {
			log.Fatalf("%s: %s: %s", flag.CommandLine.Name(), pathname, err)
		}
		defer os.Remove(tmp.Name())
		out = tmp
	}

	if err = snap.Archive(out, opt_format, []string{pathname}, opt_rebase); err != nil {
		log.Fatal(err)
	}

	if err := out.Close(); err != nil {
		return 1
	}
	if out, isFile := out.(*os.File); isFile {
		if err := os.Rename(out.Name(), opt_output); err != nil {
			return 1
		}
	}

	return 0
}
