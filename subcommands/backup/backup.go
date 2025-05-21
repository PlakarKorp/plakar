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
	"fmt"
	"os"
	"strings"

	"github.com/PlakarKorp/kloset/appcontext"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/kloset/snapshot/importer"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/dustin/go-humanize"
	"github.com/gobwas/glob"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Backup{} }, subcommands.AgentSupport, "backup")
}

type excludeFlags []string

func (e *excludeFlags) String() string {
	return strings.Join(*e, ",")
}

func (e *excludeFlags) Set(value string) error {
	*e = append(*e, value)
	return nil
}

func (cmd *Backup) Parse(ctx *appcontext.AppContext, args []string) error {
	var opt_excludes string
	var opt_exclude excludeFlags
	excludes := []string{}

	flags := flag.NewFlagSet("backup", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS] path\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s [OPTIONS] @LOCATION\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.Uint64Var(&cmd.Concurrency, "concurrency", uint64(ctx.MaxConcurrency), "maximum number of parallel tasks")
	flags.StringVar(&cmd.Tags, "tag", "", "tag to assign to this snapshot")
	flags.StringVar(&opt_excludes, "excludes", "", "path to a file containing newline-separated regex patterns, treated as -exclude")
	flags.Var(&opt_exclude, "exclude", "glob pattern to exclude files, can be specified multiple times to add several exclusion patterns")
	flags.BoolVar(&cmd.Quiet, "quiet", false, "suppress output")
	flags.BoolVar(&cmd.Silent, "silent", false, "suppress ALL output")
	flags.BoolVar(&cmd.OptCheck, "check", false, "check the snapshot after creating it")
	//flags.BoolVar(&opt_stdio, "stdio", false, "output one line per file to stdout instead of the default interactive output")
	flags.Parse(args)

	for _, item := range opt_exclude {
		if _, err := glob.Compile(item); err != nil {
			return fmt.Errorf("failed to compile exclude pattern: %s", item)
		}
		excludes = append(excludes, item)
	}

	if opt_excludes != "" {
		fp, err := os.Open(opt_excludes)
		if err != nil {
			return fmt.Errorf("unable to open excludes file: %w", err)
		}
		defer fp.Close()

		scanner := bufio.NewScanner(fp)
		for scanner.Scan() {
			line := scanner.Text()
			_, err := glob.Compile(line)
			if err != nil {
				return fmt.Errorf("failed to compile exclude pattern: %s", line)
			}
			excludes = append(excludes, line)
		}
		if err := scanner.Err(); err != nil {
			ctx.GetLogger().Error("%s", err)
			return err
		}
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.Excludes = excludes
	cmd.Path = flags.Arg(0)

	return nil
}

type Backup struct {
	subcommands.SubcommandBase

	Job         string
	Concurrency uint64
	Tags        string
	Excludes    []string
	Silent      bool
	Quiet       bool
	Path        string
	OptCheck    bool
}

func (cmd *Backup) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	ret, err, _, _ := cmd.DoBackup(ctx, repo)
	return ret, err
}

func (cmd *Backup) DoBackup(ctx *appcontext.AppContext, repo *repository.Repository) (int, error, objects.MAC, error) {
	snap, err := snapshot.Create(repo, repository.DefaultType)
	if err != nil {
		ctx.GetLogger().Error("%s", err)
		return 1, err, objects.MAC{}, nil
	}
	defer snap.Close()

	if cmd.Job != "" {
		snap.Header.Job = cmd.Job
	}

	var tags []string
	if cmd.Tags == "" {
		tags = []string{}
	} else {
		tags = []string{cmd.Tags}
	}

	excludes := []glob.Glob{}
	for _, item := range cmd.Excludes {
		g, err := glob.Compile(item)
		if err != nil {
			return 1, fmt.Errorf("failed to compile exclude pattern: %s", item), objects.MAC{}, nil
		}
		excludes = append(excludes, g)
	}

	opts := &snapshot.BackupOptions{
		MaxConcurrency: cmd.Concurrency,
		Name:           "default",
		Tags:           tags,
		Excludes:       excludes,
	}

	scanDir := ctx.CWD
	if cmd.Path != "" {
		scanDir = cmd.Path
	}

	importerConfig := map[string]string{
		"location": scanDir,
	}
	if strings.HasPrefix(scanDir, "@") {
		remote, ok := ctx.GetRemoteConfig(scanDir[1:])
		if !ok {
			return 1, fmt.Errorf("could not resolve importer: %s", scanDir), objects.MAC{}, nil
		}
		if _, ok := remote["location"]; !ok {
			return 1, fmt.Errorf("could not resolve importer location: %s", scanDir), objects.MAC{}, nil
		} else {
			importerConfig = remote
		}
	}

	imp, err := importer.NewImporter(ctx, importerConfig)
	if err != nil {
		return 1, fmt.Errorf("failed to create an importer for %s: %s", scanDir, err), objects.MAC{}, nil
	}
	defer imp.Close()

	if cmd.Silent {
		if err := snap.Backup(imp, opts); err != nil {
			return 1, fmt.Errorf("failed to create snapshot: %w", err), objects.MAC{}, nil
		}
	} else {
		ep := startEventsProcessor(ctx, imp.Root(), true, cmd.Quiet)
		if err := snap.Backup(imp, opts); err != nil {
			ep.Close()
			return 1, fmt.Errorf("failed to create snapshot: %w", err), objects.MAC{}, nil
		}
		ep.Close()
	}

	if cmd.OptCheck {
		repo.RebuildState()

		checkOptions := &snapshot.CheckOptions{
			MaxConcurrency: cmd.Concurrency,
			FastCheck:      false,
		}

		checkSnap, err := snapshot.Load(repo, snap.Header.Identifier)
		if err != nil {
			return 1, fmt.Errorf("failed to load snapshot: %w", err), objects.MAC{}, nil
		}
		defer checkSnap.Close()

		checkCache, err := ctx.GetCache().Check()
		if err != nil {
			return 1, err, objects.MAC{}, nil
		}
		defer checkCache.Close()

		checkSnap.SetCheckCache(checkCache)

		if err := checkSnap.Check("/", checkOptions); err != nil {
			return 1, fmt.Errorf("failed to check snapshot: %w", err), objects.MAC{}, nil
		}
	}

	totalSize := snap.Header.GetSource(0).Summary.Directory.Size + snap.Header.GetSource(0).Summary.Below.Size

	ctx.GetLogger().Info("backup: created %s snapshot %x of size %s in %s (wrote %s)",
		"unsigned",
		snap.Header.GetIndexShortID(),
		humanize.Bytes(totalSize),
		snap.Header.Duration,
		humanize.Bytes(uint64(snap.Repository().WBytes())),
	)

	totalErrors := uint64(0)
	for i := 0; i < len(snap.Header.Sources); i++ {
		s := snap.Header.GetSource(i)
		totalErrors += s.Summary.Directory.Errors + s.Summary.Below.Errors
	}
	var warning error
	if totalErrors > 0 {
		warning = fmt.Errorf("%d errors during backup", totalErrors)
	}
	return 0, nil, snap.Header.Identifier, warning
}
