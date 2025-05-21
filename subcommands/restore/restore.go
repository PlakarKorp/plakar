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

package restore

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/PlakarKorp/kloset/appcontext"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/kloset/snapshot/exporter"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Restore{} }, subcommands.AgentSupport, "restore")
}

func (cmd *Restore) Parse(ctx *appcontext.AppContext, args []string) error {
	var pullPath string

	flags := flag.NewFlagSet("restore", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS] [SNAPSHOT[:PATH]]...\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.Uint64Var(&cmd.Concurrency, "concurrency", uint64(ctx.MaxConcurrency), "maximum number of parallel tasks")
	flags.StringVar(&cmd.OptName, "name", "", "filter by name")
	flags.StringVar(&cmd.OptCategory, "category", "", "filter by category")
	flags.StringVar(&cmd.OptEnvironment, "environment", "", "filter by environment")
	flags.StringVar(&cmd.OptPerimeter, "perimeter", "", "filter by perimeter")
	flags.StringVar(&cmd.OptJob, "job", "", "filter by job")
	flags.StringVar(&cmd.OptTag, "tag", "", "filter by tag")

	flags.StringVar(&pullPath, "to", "", "base directory where pull will restore")
	flags.BoolVar(&cmd.Quiet, "quiet", false, "do not print progress")
	flags.BoolVar(&cmd.Silent, "silent", false, "do not print ANY progress")
	flags.Parse(args)

	if flags.NArg() != 0 {
		if cmd.OptName != "" || cmd.OptCategory != "" || cmd.OptEnvironment != "" || cmd.OptPerimeter != "" || cmd.OptJob != "" || cmd.OptTag != "" {
			ctx.GetLogger().Warn("snapshot specified, filters will be ignored")
		}
	} else if flags.NArg() > 1 {
		return fmt.Errorf("multiple restore paths specified, please specify only one")
	}

	if pullPath == "" {
		pullPath = fmt.Sprintf("%s/plakar-%s", ctx.CWD, time.Now().Format(time.RFC3339))
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.Target = pullPath
	cmd.Snapshots = flags.Args()

	return nil
}

type Restore struct {
	subcommands.SubcommandBase

	OptName        string
	OptCategory    string
	OptEnvironment string
	OptPerimeter   string
	OptJob         string
	OptTag         string

	Target      string
	Strip       string
	Concurrency uint64
	Quiet       bool
	Silent      bool
	Snapshots   []string
}

func (cmd *Restore) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	if !cmd.Silent {
		go eventsProcessorStdio(ctx, cmd.Quiet)
	}
	var snapshots []string
	if len(cmd.Snapshots) == 0 {
		locateOptions := utils.NewDefaultLocateOptions()
		locateOptions.MaxConcurrency = ctx.MaxConcurrency
		locateOptions.SortOrder = utils.LocateSortOrderAscending
		locateOptions.Latest = true

		locateOptions.Name = cmd.OptName
		locateOptions.Category = cmd.OptCategory
		locateOptions.Environment = cmd.OptEnvironment
		locateOptions.Perimeter = cmd.OptPerimeter
		locateOptions.Job = cmd.OptJob
		locateOptions.Tag = cmd.OptTag

		snapshotIDs, err := utils.LocateSnapshotIDs(repo, locateOptions)
		if err != nil {
			return 1, fmt.Errorf("ls: could not fetch snapshots list: %w", err)
		}
		for _, snapshotID := range snapshotIDs {
			snapshots = append(snapshots, fmt.Sprintf("%x:", snapshotID))
		}
	} else {
		for _, snapshotPath := range cmd.Snapshots {
			prefix, path := utils.ParseSnapshotPath(snapshotPath)

			locateOptions := utils.NewDefaultLocateOptions()
			locateOptions.MaxConcurrency = ctx.MaxConcurrency
			locateOptions.SortOrder = utils.LocateSortOrderAscending
			locateOptions.Latest = true

			locateOptions.Name = cmd.OptName
			locateOptions.Category = cmd.OptCategory
			locateOptions.Environment = cmd.OptEnvironment
			locateOptions.Perimeter = cmd.OptPerimeter
			locateOptions.Job = cmd.OptJob
			locateOptions.Tag = cmd.OptTag
			locateOptions.Prefix = prefix

			snapshotIDs, err := utils.LocateSnapshotIDs(repo, locateOptions)
			if err != nil {
				return 1, fmt.Errorf("ls: could not fetch snapshots list: %w", err)
			}
			for _, snapshotID := range snapshotIDs {
				snapshots = append(snapshots, fmt.Sprintf("%x:%s", snapshotID, path))
			}
		}
	}

	if len(snapshots) == 0 {
		return 1, fmt.Errorf("no snapshots found")
	} else if len(snapshots) > 1 {
		return 1, fmt.Errorf("multiple snapshots found, please specify one")
	}

	exporterConfig := map[string]string{
		"location": cmd.Target,
	}
	if strings.HasPrefix(cmd.Target, "@") {
		remote, ok := ctx.GetRemoteConfig(cmd.Target[1:])
		if !ok {
			return 1, fmt.Errorf("could not resolve exporter: %s", cmd.Target)
		}
		if _, ok := remote["location"]; !ok {
			return 1, fmt.Errorf("could not resolve exporter location: %s", cmd.Target)
		} else {
			exporterConfig = remote
		}
	}

	var exporterInstance exporter.Exporter
	var err error
	exporterInstance, err = exporter.NewExporter(ctx, exporterConfig)
	if err != nil {
		return 1, err
	}
	defer exporterInstance.Close()

	opts := &snapshot.RestoreOptions{
		MaxConcurrency: cmd.Concurrency,
	}

	for _, snapPath := range snapshots {
		snap, pathname, err := utils.OpenSnapshotByPath(repo, snapPath)
		if err != nil {
			return 1, err
		}
		opts.Strip = snap.Header.GetSource(0).Importer.Directory

		err = snap.Restore(exporterInstance, exporterInstance.Root(), pathname, opts)

		if err != nil {
			return 1, err
		}
		ctx.GetLogger().Info("restore: restoration of %x:%s at %s completed successfully",
			snap.Header.GetIndexShortID(),
			pathname,
			cmd.Target)
		snap.Close()
	}
	return 0, nil
}
