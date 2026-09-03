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
	"fmt"
	"maps"
	"path"
	"strings"
	"time"

	"github.com/PlakarKorp/kloset/connectors/exporter"
	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/spf13/cobra"
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
	Opts               map[string]string

	Target    string
	Strip     string
	Snapshots []string

	pullPath string
}

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Restore{} }, 0, "restore")
}

func (cmd *Restore) CobraCommand() *cobra.Command {
	cmd.Opts = make(map[string]string)

	c := &cobra.Command{
		Use: "restore [OPTIONS] [SNAPSHOT[:PATH]]...",
	}
	c.Flags().StringVar(&cmd.OptName, "name", "", "filter by name")
	c.Flags().StringVar(&cmd.OptCategory, "category", "", "filter by category")
	c.Flags().StringVar(&cmd.OptEnvironment, "environment", "", "filter by environment")
	c.Flags().StringVar(&cmd.OptPerimeter, "perimeter", "", "filter by perimeter")
	c.Flags().StringVar(&cmd.OptJob, "job", "", "filter by job")
	c.Flags().StringVar(&cmd.OptTag, "tag", "", "filter by tag")
	c.Flags().Var(subcommands.GoValue(utils.NewOptsFlag(cmd.Opts)), "o", "specify extra exporter options")
	c.Flags().StringVar(&cmd.pullPath, "to", "", "base directory where pull will restore")
	c.Flags().BoolVar(&cmd.OptSkipPermissions, "skip-permissions", false, "do not restore file permissions")
	return c
}

func (cmd *Restore) Parse(ctx *appcontext.AppContext, args []string) error {
	rest, err := subcommands.ParseCobra(cmd, args)
	if err != nil {
		return err
	}

	if len(rest) != 0 {
		if cmd.OptName != "" || cmd.OptCategory != "" || cmd.OptEnvironment != "" || cmd.OptPerimeter != "" || cmd.OptJob != "" || cmd.OptTag != "" {
			ctx.GetLogger().Warn("snapshot specified, filters will be ignored")
		}
	} else if len(rest) > 1 {
		return fmt.Errorf("multiple restore paths specified, please specify only one")
	}

	if cmd.pullPath == "" {
		cmd.pullPath = fmt.Sprintf("%s/plakar-%s", ctx.CWD, time.Now().Format("20060102150405"))
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.Target = cmd.pullPath
	cmd.Snapshots = rest

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

		backupErrorCount, err := snapshotBackupErrorCount(snap)
		if err != nil {
			ctx.GetLogger().Warn("could not inspect snapshot backup errors: %s", err)
		} else if backupErrorCount != 0 {
			errorWord := "errors"
			if backupErrorCount == 1 {
				errorWord = "error"
			}
			ctx.GetLogger().Warn("snapshot contains %d backup %s; restored files are limited to data that was backed up", backupErrorCount, errorWord)
		}

		snap.Close()
	}
	return 0, nil
}

func snapshotBackupErrorCount(snap *snapshot.Snapshot) (int, error) {
	fsc, err := snap.Filesystem()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, err := range fsc.Errors("/") {
		if err != nil {
			return 0, err
		}
		count++
	}
	return count, nil
}
