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
	"context"
	"flag"
	"fmt"
	"maps"
	"path"
	"strings"
	"time"

	"github.com/PlakarKorp/kloset/connectors"
	"github.com/PlakarKorp/kloset/connectors/exporter"
	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
)

const (
	restoreClearedDeviceID uint64 = 0
	restoreClearedInodeID  uint64 = 0
	restoreSingleLinkCount uint16 = 1
)

type Restore struct {
	subcommands.SubcommandBase

	OptName            string
	OptCategory        string
	OptEnvironment     string
	OptPerimeter       string
	OptJob             string
	OptTag             string
	OptSkipPermissions bool
	OptIgnoreCtime     bool
	OptIgnoreInode     bool
	Opts               map[string]string

	Target    string
	Strip     string
	Snapshots []string
}

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Restore{} }, 0, "restore")
}

func (cmd *Restore) Parse(ctx *appcontext.AppContext, args []string) error {
	var pullPath string

	cmd.Opts = make(map[string]string)

	flags := flag.NewFlagSet("restore", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS] [SNAPSHOT[:PATH]]...\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.StringVar(&cmd.OptName, "name", "", "filter by name")
	flags.StringVar(&cmd.OptCategory, "category", "", "filter by category")
	flags.StringVar(&cmd.OptEnvironment, "environment", "", "filter by environment")
	flags.StringVar(&cmd.OptPerimeter, "perimeter", "", "filter by perimeter")
	flags.StringVar(&cmd.OptJob, "job", "", "filter by job")
	flags.StringVar(&cmd.OptTag, "tag", "", "filter by tag")
	flags.Var(utils.NewOptsFlag(cmd.Opts), "o", "specify extra exporter options")

	flags.StringVar(&pullPath, "to", "", "base directory where pull will restore")
	flags.BoolVar(&cmd.OptSkipPermissions, "skip-permissions", false, "do not restore file permissions")
	flags.BoolVar(&cmd.OptIgnoreCtime, "ignore-ctime", false, "do not restore file timestamps")
	flags.BoolVar(&cmd.OptIgnoreInode, "ignore-inode", false, "do not restore hardlinks from inode metadata")
	flags.Parse(args)

	if flags.NArg() != 0 {
		if cmd.OptName != "" || cmd.OptCategory != "" || cmd.OptEnvironment != "" || cmd.OptPerimeter != "" || cmd.OptJob != "" || cmd.OptTag != "" {
			ctx.GetLogger().Warn("snapshot specified, filters will be ignored")
		}
	} else if flags.NArg() > 1 {
		return fmt.Errorf("multiple restore paths specified, please specify only one")
	}

	if pullPath == "" {
		pullPath = fmt.Sprintf("%s/plakar-%s", ctx.CWD, time.Now().Format("20060102150405"))
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.Target = pullPath
	cmd.Snapshots = flags.Args()

	return nil
}

func (cmd *Restore) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	var snapshots []string
	if len(cmd.Snapshots) == 0 {
		locateOptions := locate.NewDefaultLocateOptions()
		locateOptions.Filters.Latest = true

		locateOptions.Filters.Name = cmd.OptName
		locateOptions.Filters.Category = cmd.OptCategory
		locateOptions.Filters.Environment = cmd.OptEnvironment
		locateOptions.Filters.Perimeter = cmd.OptPerimeter
		locateOptions.Filters.Job = cmd.OptJob
		locateOptions.Filters.Tags = []string{cmd.OptTag}

		snapshotIDs, err := locate.LocateSnapshotIDs(repo, locateOptions)
		if err != nil {
			return 1, fmt.Errorf("ls: could not fetch snapshots list: %w", err)
		}
		for _, snapshotID := range snapshotIDs {
			snapshots = append(snapshots, fmt.Sprintf("%x:", snapshotID))
		}
	} else {
		for _, snapshotPath := range cmd.Snapshots {
			prefix, path := locate.ParseSnapshotPath(snapshotPath)

			locateOptions := locate.NewDefaultLocateOptions()
			locateOptions.Filters.Latest = true
			locateOptions.Filters.Name = cmd.OptName
			locateOptions.Filters.Category = cmd.OptCategory
			locateOptions.Filters.Environment = cmd.OptEnvironment
			locateOptions.Filters.Perimeter = cmd.OptPerimeter
			locateOptions.Filters.Job = cmd.OptJob
			locateOptions.Filters.Tags = []string{cmd.OptTag}
			locateOptions.Filters.IDs = []string{prefix}

			snapshotIDs, err := locate.LocateSnapshotIDs(repo, locateOptions)
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
		remote, ok := ctx.Config.GetDestination(cmd.Target[1:])
		if !ok {
			return 1, fmt.Errorf("could not resolve exporter: %s", cmd.Target)
		}
		if _, ok := remote["location"]; !ok {
			return 1, fmt.Errorf("could not resolve exporter location: %s", cmd.Target)
		} else {
			exporterConfig = remote
		}
	}

	maps.Copy(exporterConfig, cmd.Opts)

	var exporterInstance exporter.Exporter
	var err error
	options := ctx.ExporterOpts()

	exporterInstance, err = exporter.NewExporter(ctx.GetInner(), options, exporterConfig)
	if err != nil {
		return 1, err
	}
	defer exporterInstance.Close(ctx)
	exporterInstance = newRestoreMetadataExporter(exporterInstance, cmd.OptIgnoreCtime, cmd.OptIgnoreInode, time.Now)

	opts := &snapshot.ExportOptions{}
	if cmd.OptSkipPermissions {
		opts.SkipPermissions = true
	}

	for _, snapPath := range snapshots {
		snap, pathname, relative, err := locate.OpenSnapshotByPathRelative(repo, snapPath)
		if err != nil {
			return 1, err
		}

		if relative != "" {
			if !strings.HasSuffix(relative, "/") {
				opts.Strip = path.Dir(pathname)
			} else {
				opts.Strip = pathname
			}
		}

		err = snap.Export(exporterInstance, pathname, opts)
		if err != nil {
			return 1, err
		}

		snap.Close()
	}
	return 0, nil
}

type restoreMetadataExporter struct {
	exporter.Exporter
	skipTimes     bool
	skipHardlinks bool
	now           func() time.Time
}

func newRestoreMetadataExporter(exp exporter.Exporter, skipTimes bool, skipHardlinks bool, now func() time.Time) exporter.Exporter {
	if !skipTimes && !skipHardlinks {
		return exp
	}
	return &restoreMetadataExporter{
		Exporter:      exp,
		skipTimes:     skipTimes,
		skipHardlinks: skipHardlinks,
		now:           now,
	}
}

func (exp *restoreMetadataExporter) Export(ctx context.Context, records <-chan *connectors.Record, results chan<- *connectors.Result) error {
	filtered := make(chan *connectors.Record)
	go func() {
		defer close(filtered)
		for {
			select {
			case <-ctx.Done():
				return
			case record, ok := <-records:
				if !ok {
					return
				}

				select {
				case <-ctx.Done():
					return
				case filtered <- exp.filterRecord(record):
				}
			}
		}
	}()

	return exp.Exporter.Export(ctx, filtered, results)
}

func (exp *restoreMetadataExporter) filterRecord(record *connectors.Record) *connectors.Record {
	if record == nil {
		return nil
	}

	filtered := *record
	fileinfo := filtered.FileInfo

	if exp.skipTimes {
		fileinfo.LmodTime = exp.now()
	}
	if exp.skipHardlinks {
		fileinfo.Ldev = restoreClearedDeviceID
		fileinfo.Lino = restoreClearedInodeID
		fileinfo.Lnlink = restoreSingleLinkCount
	}

	filtered.FileInfo = fileinfo
	return &filtered
}

var _ exporter.Exporter = (*restoreMetadataExporter)(nil)
