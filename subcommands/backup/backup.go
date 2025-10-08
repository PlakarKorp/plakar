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
	"os/exec"
	"strings"
	"time"

	"github.com/PlakarKorp/kloset/exclude"
	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/kloset/snapshot/importer"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/dustin/go-humanize"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Backup{} }, subcommands.AgentSupport, "backup")
}

type ignoreFlags []string

func (e *ignoreFlags) String() string {
	return strings.Join(*e, ",")
}

func (e *ignoreFlags) Set(value string) error {
	*e = append(*e, value)
	return nil
}

type tagFlags string

// Called by the flag package to print the default / help.
func (e *tagFlags) String() string {
	return string(*e)
}

// Called once per flag occurrence to set the value.
func (e *tagFlags) Set(value string) error {
	if *e != "" {
		return fmt.Errorf("tags should be specified only once, as a comma-separated list")
	}
	*e = tagFlags(value)
	return nil
}

func (e *tagFlags) asList() []string {
	tags := string(*e)
	if tags == "" {
		return []string{}
	}
	return strings.Split(tags, ",")
}

func (cmd *Backup) Parse(ctx *appcontext.AppContext, args []string) error {
	var opt_ignore_file string
	var opt_ignore ignoreFlags
	var opt_tags tagFlags

	excludes := []string{}

	cmd.Opts = make(map[string]string)

	flags := flag.NewFlagSet("backup", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS] path\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s [OPTIONS] @LOCATION\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.Uint64Var(&cmd.Concurrency, "concurrency", uint64(ctx.MaxConcurrency), "maximum number of parallel tasks")
	flags.Var(&opt_tags, "tag", "comma-separated list of tags to apply to the snapshot")
	flags.StringVar(&opt_ignore_file, "ignore-file", "", "path to a file containing newline-separated gitignore patterns, treated as -ignore")
	flags.Var(&opt_ignore, "ignore", "gitignore pattern to exclude files, can be specified multiple times to add several exclusion patterns")
	flags.StringVar(&cmd.PackfileTempStorage, "packfiles", "memory", "memory or a path to a directory to store temporary packfiles")
	flags.BoolVar(&cmd.Quiet, "quiet", false, "suppress output")
	flags.BoolVar(&cmd.Silent, "silent", false, "suppress ALL output")
	flags.BoolVar(&cmd.OptCheck, "check", false, "check the snapshot after creating it")
	flags.Var(utils.NewOptsFlag(cmd.Opts), "o", "specify extra importer options")
	flags.BoolVar(&cmd.DryRun, "scan", false, "do not actually perform a backup, just list the files")
	flags.Var(locate.NewTimeFlag(&cmd.ForcedTimestamp), "force-timestamp", "force a timestamp")
	flags.StringVar(&cmd.PreHook, "pre-hook", "", "command to execute before backup")
	flags.StringVar(&cmd.PostHook, "post-hook", "", "command to execute after successful backup")
	//flags.BoolVar(&opt_stdio, "stdio", false, "output one line per file to stdout instead of the default interactive output")
	flags.Parse(args)

	if flags.NArg() > 1 {
		return fmt.Errorf("Too many arguments")
	}

	if !cmd.ForcedTimestamp.IsZero() {
		if cmd.ForcedTimestamp.After(time.Now()) {
			return fmt.Errorf("forced timestamp cannot be in the future")
		}
	}

	if opt_ignore_file != "" {
		lines, err := LoadIgnoreFile(opt_ignore_file)
		if err != nil {
			return err
		}
		for _, line := range lines {
			excludes = append(excludes, line)
		}
	}

	for _, item := range opt_ignore {
		excludes = append(excludes, item)
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.Excludes = excludes
	cmd.Path = flags.Arg(0)
	cmd.Tags = opt_tags.asList()

	if cmd.Path == "" {
		cmd.Path = "fs:" + ctx.CWD
	}

	return nil
}

type Backup struct {
	subcommands.SubcommandBase

	Job                 string
	Concurrency         uint64
	Tags                []string
	Excludes            []string
	Silent              bool
	Quiet               bool
	Path                string
	OptCheck            bool
	Opts                map[string]string
	DryRun              bool
	PackfileTempStorage string
	ForcedTimestamp     time.Time
	PreHook             string
	PostHook            string
}

func (cmd *Backup) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	ret, err, _, _ := cmd.DoBackup(ctx, repo)
	return ret, err
}

func (cmd *Backup) DoBackup(ctx *appcontext.AppContext, repo *repository.Repository) (int, error, objects.MAC, error) {
	opts := &snapshot.BackupOptions{
		MaxConcurrency: cmd.Concurrency,
		Name:           "default",
		Tags:           cmd.Tags,
		Excludes:       cmd.Excludes,
	}

	if !cmd.ForcedTimestamp.IsZero() {
		opts.ForcedTimestamp = cmd.ForcedTimestamp
	}

	scanDir := "fs:" + ctx.CWD
	if cmd.Path != "" {
		scanDir = cmd.Path
	}

	if strings.HasPrefix(scanDir, "@") {
		remote, ok := ctx.Config.GetSource(scanDir[1:])
		if !ok {
			return 1, fmt.Errorf("could not resolve importer: %s", scanDir), objects.MAC{}, nil
		}
		if _, ok := remote["location"]; !ok {
			return 1, fmt.Errorf("could not resolve importer location: %s", scanDir), objects.MAC{}, nil
		} else {
			// inherit all the options -- but the ones
			// specified in the command line takes the
			// precedence.
			for k, v := range remote {
				if _, found := cmd.Opts[k]; !found {
					cmd.Opts[k] = v
				}
			}
		}
	}

	// Now that we have resolved the possible @ syntax let's apply the scandir.
	if _, found := cmd.Opts["location"]; !found {
		cmd.Opts["location"] = scanDir
	}

	imp, err := importer.NewImporter(ctx.GetInner(), ctx.ImporterOpts(), cmd.Opts)
	if err != nil {
		return 1, fmt.Errorf("failed to create an importer for %s: %s", scanDir, err), objects.MAC{}, nil
	}
	defer imp.Close(ctx)

	if cmd.DryRun {
		if err := dryrun(ctx, imp, cmd.Excludes); err != nil {
			return 1, err, objects.MAC{}, nil
		}
		return 0, nil, objects.MAC{}, nil
	}

	// Execute pre-backup hook
	if err := executeHook(ctx, cmd.PreHook); err != nil {
		return 1, fmt.Errorf("pre-backup hook failed: %w", err), objects.MAC{}, nil
	}

	if cmd.PackfileTempStorage != "memory" {
		tmpDir, err := os.MkdirTemp(cmd.PackfileTempStorage, "plakar-backup-"+repo.Configuration().RepositoryID.String()+"-*")
		if err != nil {
			return 1, err, objects.NilMac, nil
		}
		cmd.PackfileTempStorage = tmpDir
		defer os.RemoveAll(cmd.PackfileTempStorage)
	} else {
		cmd.PackfileTempStorage = ""
	}

	snap, err := snapshot.Create(repo, repository.DefaultType, cmd.PackfileTempStorage)
	if err != nil {
		ctx.GetLogger().Error("%s", err)
		return 1, err, objects.MAC{}, nil
	}
	defer snap.Close()

	if cmd.Job != "" {
		snap.Header.Job = cmd.Job
	}

	if cmd.Silent {
		if err := snap.Backup(imp, opts); err != nil {
			return 1, fmt.Errorf("failed to create snapshot: %w", err), objects.MAC{}, nil
		}
	} else {
		root, err := imp.Root(ctx)
		if err != nil {
			return 1, fmt.Errorf("failed to get importer root: %w", err), objects.MAC{}, nil
		}

		ep := startEventsProcessor(ctx, root, true, cmd.Quiet)
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

	// Execute post-backup hook
	if err := executeHook(ctx, cmd.PostHook); err != nil {
		ctx.GetLogger().Warn("post-backup hook failed: %s", err)
	}

	totalSize := snap.Header.GetSource(0).Summary.Directory.Size + snap.Header.GetSource(0).Summary.Below.Size

	ctx.GetLogger().Info("backup: created %s snapshot %x of size %s in %s (wrote %s)",
		"unsigned",
		snap.Header.GetIndexShortID(),
		humanize.IBytes(totalSize),
		snap.Header.Duration,
		humanize.IBytes(uint64(snap.Repository().WBytes())),
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

func LoadIgnoreFile(filename string) ([]string, error) {
	fp, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("unable to open excludes file: %w", err)
	}
	defer fp.Close()

	var lines []string
	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Trim(line, " \t\r") == "" {
			continue
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

func executeHook(ctx *appcontext.AppContext, hook string) error {
	if hook == "" {
		return nil
	}
	ctx.GetLogger().Info("executing hook: %s", hook)
	cmd := exec.Command("sh", "-c", hook)
	cmd.Stdout = ctx.Stdout
	cmd.Stderr = ctx.Stderr
	return cmd.Run()
}

func dryrun(ctx *appcontext.AppContext, imp importer.Importer, excludePatterns []string) error {
	scanner, err := imp.Scan(ctx)
	if err != nil {
		return fmt.Errorf("failed to scan: %w", err)
	}

	excludes := exclude.NewRuleSet()
	if err := excludes.AddRulesFromArray(excludePatterns); err != nil {
		return fmt.Errorf("failed to setup exclude rules: %w", err)
	}

	errors := false
	for record := range scanner {
		var pathname string
		var isDir bool
		switch {
		case record.Record != nil:
			pathname = record.Record.Pathname
			isDir = record.Record.FileInfo.IsDir()
		case record.Error != nil:
			pathname = record.Error.Pathname
			isDir = false
		}

		if excludes.IsExcluded(pathname, isDir) {
			if record.Record != nil {
				record.Record.Close()
			}
			continue
		}

		switch {
		case record.Error != nil:
			errors = true
			fmt.Fprintf(ctx.Stderr, "%s: %s\n",
				record.Error.Pathname, record.Error.Err)
		case record.Record != nil:
			fmt.Fprintln(ctx.Stdout, record.Record.Pathname)
			record.Record.Close()
		}
	}

	if errors {
		return fmt.Errorf("failed to scan some files")
	}
	return nil
}
