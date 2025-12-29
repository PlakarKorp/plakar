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
	"os"
	"time"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Archive{} }, 0, "archive")
}

func (cmd *Archive) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("archive", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS] [SNAPSHOT[:PATH]]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.StringVar(&cmd.Output, "output", "", "archive pathname")
	flags.BoolVar(&cmd.Rebase, "rebase", false, "strip pathname when pulling")
	flags.StringVar(&cmd.Format, "format", "tarball", "archive format: tar, tarball, zip")
	flags.Parse(args)

	if flags.NArg() == 0 {
		return fmt.Errorf("need at least one snapshot ID to pull")
	}
	cmd.SnapshotPrefix = flags.Arg(0)

	supportedFormats := map[string]string{
		"tar":     "tar",
		"tarball": "tar.gz",
		"zip":     "zip",
	}
	if _, ok := supportedFormats[cmd.Format]; !ok {
		return fmt.Errorf("unsupported format %s", cmd.Format)
	}

	if cmd.Output == "" {
		cmd.Output = fmt.Sprintf("plakar-%s.%s", time.Now().UTC().Format(time.RFC3339), supportedFormats[cmd.Format])
	}

	return nil
}

type Archive struct {
	subcommands.SubcommandBase

	Rebase         bool
	Output         string
	Format         string
	SnapshotPrefix string
}

func (cmd *Archive) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	snap, pathname, err := locate.OpenSnapshotByPath(repo, cmd.SnapshotPrefix)
	if err != nil {
		return 1, fmt.Errorf("archive: could not open snapshot: %s", cmd.SnapshotPrefix)
	}
	defer snap.Close()

	var out io.Writer
	if cmd.Output == "-" {
		out = os.Stdout
	} else {
		tmp, err := os.CreateTemp("", "plakar-archive-")
		if err != nil {
			return 1, fmt.Errorf("archive: %s: %w", pathname, err)
		}
		defer os.Remove(tmp.Name())
		out = tmp
	}

	if err = snap.Archive(out, cmd.Format, []string{pathname}, cmd.Rebase); err != nil {
		return 1, err
	}

	if outCloser, isCloser := out.(io.Closer); isCloser {
		if err := outCloser.Close(); err != nil {
			return 1, err
		}
	}

	if out, isFile := out.(*os.File); isFile {
		if err := os.Rename(out.Name(), cmd.Output); err != nil {
			return 1, err
		}
	}
	return 0, nil
}
