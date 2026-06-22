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
)

func init() {
	subcommands.Register(Restore, 0, "restore")
}

func Restore(ctx *appcontext.AppContext, repo *repository.Repository, args []string) error {
	var (
		target       string
		name         string
		category     string
		environment  string
		perimeter    string
		job          string
		tag          string
		skipPerms    bool
		exporterOpts = make(map[string]string)
	)

	flags := flag.NewFlagSet("restore", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS] [SNAPSHOT[:PATH]]...\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.StringVar(&name, "name", "", "filter by name")
	flags.StringVar(&category, "category", "", "filter by category")
	flags.StringVar(&environment, "environment", "", "filter by environment")
	flags.StringVar(&perimeter, "perimeter", "", "filter by perimeter")
	flags.StringVar(&job, "job", "", "filter by job")
	flags.StringVar(&tag, "tag", "", "filter by tag")
	flags.Var(utils.NewOptsFlag(exporterOpts), "o", "specify extra exporter options")

	flags.StringVar(&target, "to", "", "base directory where pull will restore")
	flags.BoolVar(&skipPerms, "skip-permissions", false, "do not restore file permissions")
	flags.Parse(args)

	if flags.NArg() != 0 {
		if name != "" || category != "" || environment != "" || perimeter != "" || job != "" || tag != "" {
			ctx.GetLogger().Warn("snapshot specified, filters will be ignored")
		}
	} else if flags.NArg() > 1 {
		return fmt.Errorf("multiple restore paths specified, please specify only one")
	}

	if target == "" {
		target = fmt.Sprintf("%s/plakar-%s", ctx.CWD, time.Now().Format("20060102150405"))
	}

	var snapshots []string
	if len(flags.Args()) == 0 {
		locateOptions := locate.NewDefaultLocateOptions()
		locateOptions.Filters.Latest = true

		locateOptions.Filters.Name = name
		locateOptions.Filters.Category = category
		locateOptions.Filters.Environment = environment
		locateOptions.Filters.Perimeter = perimeter
		locateOptions.Filters.Job = job
		locateOptions.Filters.Tags = []string{tag}

		snapshotIDs, err := locate.LocateSnapshotIDs(repo, locateOptions)
		if err != nil {
			return fmt.Errorf("ls: could not fetch snapshots list: %w", err)
		}
		for _, snapshotID := range snapshotIDs {
			snapshots = append(snapshots, fmt.Sprintf("%x:", snapshotID))
		}
	} else {
		for _, snapshotPath := range flags.Args() {
			prefix, path := locate.ParseSnapshotPath(snapshotPath)

			locateOptions := locate.NewDefaultLocateOptions()
			locateOptions.Filters.Latest = true
			locateOptions.Filters.Name = name
			locateOptions.Filters.Category = category
			locateOptions.Filters.Environment = environment
			locateOptions.Filters.Perimeter = perimeter
			locateOptions.Filters.Job = job
			locateOptions.Filters.Tags = []string{tag}
			locateOptions.Filters.IDs = []string{prefix}

			snapshotIDs, err := locate.LocateSnapshotIDs(repo, locateOptions)
			if err != nil {
				return fmt.Errorf("ls: could not fetch snapshots list: %w", err)
			}
			for _, snapshotID := range snapshotIDs {
				snapshots = append(snapshots, fmt.Sprintf("%x:%s", snapshotID, path))
			}
		}
	}

	if len(snapshots) == 0 {
		return fmt.Errorf("no snapshots found")
	} else if len(snapshots) > 1 {
		return fmt.Errorf("multiple snapshots found, please specify one")
	}

	exporterConfig := map[string]string{
		"location": target,
	}
	if strings.HasPrefix(target, "@") {
		remote, ok := ctx.Config.GetDestination(target[1:])
		if !ok {
			return fmt.Errorf("could not resolve exporter: %s", target)
		}
		if _, ok := remote["location"]; !ok {
			return fmt.Errorf("could not resolve exporter location: %s", target)
		} else {
			exporterConfig = remote
		}
	}

	maps.Copy(exporterConfig, exporterOpts)

	var exporterInstance exporter.Exporter
	var err error
	options := ctx.ExporterOpts()

	exporterInstance, err = exporter.NewExporter(ctx.GetInner(), options, exporterConfig)
	if err != nil {
		return err
	}
	defer exporterInstance.Close(ctx)

	opts := &snapshot.ExportOptions{}
	if skipPerms {
		opts.SkipPermissions = true
	}

	for _, snapPath := range snapshots {
		snap, pathname, relative, err := locate.OpenSnapshotByPathRelative(repo, snapPath)
		if err != nil {
			return err
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
			return err
		}

		snap.Close()
	}
	return nil
}
