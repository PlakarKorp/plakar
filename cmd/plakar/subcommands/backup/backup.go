/*
 * Copyright (c) 2021 Gilles Chehade <gilles@poolp.org>
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

package backup

import (
	"bufio"
	"flag"
	"os"
	"strings"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/rpc"
	"github.com/PlakarKorp/plakar/rpc/backup"
	"github.com/gobwas/glob"
)

func init() {
	subcommands.Register("backup", parse_cmd_backup)
}

type excludeFlags []string

func (e *excludeFlags) String() string {
	return strings.Join(*e, ",")
}

func (e *excludeFlags) Set(value string) error {
	*e = append(*e, value)
	return nil
}

func parse_cmd_backup(ctx *appcontext.AppContext, repo *repository.Repository, args []string) (rpc.RPC, error) {
	var opt_tags string
	var opt_excludes string
	var opt_exclude excludeFlags
	var opt_concurrency uint64
	var opt_quiet bool
	var opt_identity string

	excludes := []glob.Glob{}
	flags := flag.NewFlagSet("backup", flag.ExitOnError)
	flags.Uint64Var(&opt_concurrency, "concurrency", uint64(ctx.MaxConcurrency), "maximum number of parallel tasks")
	flags.StringVar(&opt_identity, "identity", "", "use identity from keyring")
	flags.StringVar(&opt_tags, "tag", "", "tag to assign to this snapshot")
	flags.StringVar(&opt_excludes, "excludes", "", "file containing a list of exclusions")
	flags.Var(&opt_exclude, "exclude", "file containing a list of exclusions")
	flags.BoolVar(&opt_quiet, "quiet", false, "suppress output")
	//flags.BoolVar(&opt_stdio, "stdio", false, "output one line per file to stdout instead of the default interactive output")
	flags.Parse(args)

	for _, item := range opt_exclude {
		excludes = append(excludes, glob.MustCompile(item))
	}

	if opt_excludes != "" {
		fp, err := os.Open(opt_excludes)
		if err != nil {
			ctx.GetLogger().Error("%s", err)
			return nil, err
		}
		defer fp.Close()

		scanner := bufio.NewScanner(fp)
		for scanner.Scan() {
			pattern, err := glob.Compile(scanner.Text())
			if err != nil {
				ctx.GetLogger().Error("%s", err)
				return nil, err
			}
			excludes = append(excludes, pattern)
		}
		if err := scanner.Err(); err != nil {
			ctx.GetLogger().Error("%s", err)
			return nil, err
		}
	}
	_ = excludes

	return &backup.Backup{
		RepositoryLocation: repo.Location(),
		RepositorySecret:   ctx.GetSecret(),
		Concurrency:        opt_concurrency,
		Identity:           opt_identity,
		Tags:               opt_tags,
		Excludes:           excludes,
		Exclude:            opt_exclude,
		Quiet:              opt_quiet,
		Path:               flags.Arg(0),
	}, nil
}
