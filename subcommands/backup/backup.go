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
	"fmt"
	"maps"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/PlakarKorp/kloset/connectors"
	"github.com/PlakarKorp/kloset/connectors/importer"
	"github.com/PlakarKorp/kloset/events"
	"github.com/PlakarKorp/kloset/exclude"
	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/location"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/kloset/snapshot/vfs"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cached"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/spf13/cobra"
)

type Backup struct {
	subcommands.SubcommandBase

	Job                 string
	Tags                []string
	Excludes            []string
	Sources             []string
	OptCheck            bool
	Opts                map[string]string
	DryRun              bool
	PackfileTempStorage string
	ForcedTimestamp     time.Time
	PreHook             string
	PostHook            string
	FailHook            string
	NoXattr             bool
	Cache               string
	NoProgress          bool
	Name                string
	Category            string
	Environment         string
	Perimeter           string

	optIgnoreFiles ignoreFlags
	optIgnore      ignoreFlags
	optTags        tagFlags
}

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Backup{} }, 0, "backup")
}

type ignoreFlags []string

const excludeIncludePrefix = "!"

func (e *ignoreFlags) String() string {
	return strings.Join(*e, ",")
}

func (e *ignoreFlags) Set(value string) error {
	*e = append(*e, value)
	return nil
}

func importerExcludes(excludes []string) []string {
	for _, rule := range excludes {
		if strings.HasPrefix(rule, excludeIncludePrefix) {
			return nil
		}
	}
	return excludes
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

func (cmd *Backup) CobraCommand() *cobra.Command {
	cmd.Opts = make(map[string]string)

	c := &cobra.Command{
		Use: "backup [OPTIONS] path",
	}
	c.Flags().Var(subcommands.GoValue(&cmd.optTags), "tag", "comma-separated list of tags to apply to the snapshot")
	c.Flags().StringVar(&cmd.Name, "name", "default", "backup name")
	c.Flags().StringVar(&cmd.Category, "category", "", "backup category")
	c.Flags().StringVar(&cmd.Environment, "environment", "", "backup environment")
	c.Flags().StringVar(&cmd.Perimeter, "perimeter", "", "backup perimeter")
	c.Flags().StringVar(&cmd.Job, "job", "", "backup job")
	c.Flags().Var(subcommands.GoValue(&cmd.optIgnoreFiles), "ignore-file", "path to a file containing newline-separated gitignore patterns, treated as -ignore; can be specified multiple times")
	c.Flags().Var(subcommands.GoValue(&cmd.optIgnore), "ignore", "gitignore pattern to exclude files, can be specified multiple times to add several exclusion patterns")
	c.Flags().StringVar(&cmd.PackfileTempStorage, "packfiles", "", "memory or a path to a directory to store temporary packfiles")
	c.Flags().BoolVar(&cmd.OptCheck, "check", false, "check the snapshot after creating it")
	c.Flags().Var(subcommands.GoValue(utils.NewOptsFlag(cmd.Opts)), "o", "specify extra importer options")
	c.Flags().BoolVar(&cmd.DryRun, "dry-run", false, "do not actually perform a backup")
	c.Flags().BoolVar(&cmd.NoXattr, "no-xattr", false, "do not back up extended attributes")
	c.Flags().StringVar(&cmd.Cache, "cache", "vfs", "path to store vfs cache, 'no' for uncached and 'vfs' for the default in memory cache")
	c.Flags().BoolVar(&cmd.NoProgress, "no-progress", false, "do not display progress")
	c.Flags().Var(subcommands.GoValue(locate.NewTimeFlag(&cmd.ForcedTimestamp)), "force-timestamp", "force a timestamp")
	return c
}

func (cmd *Backup) Parse(ctx *appcontext.AppContext, args []string) error {
	rest, err := subcommands.ParseCobra(cmd, args)
	if err != nil {
		return err
	}

	excludes := []string{}

	if !cmd.ForcedTimestamp.IsZero() {
		if cmd.ForcedTimestamp.After(time.Now()) {
			return fmt.Errorf("forced timestamp cannot be in the future")
		}
	}

	for _, ignoreFile := range cmd.optIgnoreFiles {
		lines, err := utils.LoadIgnoreFile(ignoreFile)
		if err != nil {
			return err
		}
		excludes = append(excludes, lines...)
	}

	for _, item := range cmd.optIgnore {
		excludes = append(excludes, item)
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.Excludes = excludes
	cmd.Tags = cmd.optTags.asList()

	// If no tags were provided via CLI flag, check PLAKAR_TAGS env var
	if len(cmd.Tags) == 0 {
		if envTags, ok := os.LookupEnv("PLAKAR_TAGS"); ok && envTags != "" {
			parts := strings.Split(envTags, ",")
			var tags []string
			for _, t := range parts {
				t = strings.TrimSpace(t)
				if t != "" {
					tags = append(tags, t)
				}
			}
			cmd.Tags = tags
		}
	}

	cmd.Sources = rest

	if len(cmd.Sources) == 0 {
		cmd.Sources = append(cmd.Sources, "fs:"+ctx.CWD)
	}

	return nil
}

func (cmd *Backup) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	ret, err, _, _ := cmd.DoBackup(ctx, repo)
	return ret, err
}

func (cmd *Backup) DoBackup(ctx *appcontext.AppContext, repo *repository.Repository) (int, error, objects.MAC, error) {
	emitter := repo.Emitter("import")
	defer emitter.Close()

	opts := &snapshot.BuilderOptions{
		Name:           cmd.Name,
		Tags:           cmd.Tags,
		Job:            cmd.Job,
		Category:       cmd.Category,
		Environment:    cmd.Environment,
		Perimeter:      cmd.Perimeter,
		NoXattr:        cmd.NoXattr,
		StateRefresher: stateRefresher(ctx, repo),
	}

	if !cmd.ForcedTimestamp.IsZero() {
		opts.ForcedTimestamp = cmd.ForcedTimestamp
	}

	sourcesPerOrig := make(map[string][]importer.Importer)
	// If we are doing a fake run for statistics instantiate separate importers,
	// otherwise it makes plugin development harder than needed.
	sourcesPerOrigForStats := make(map[string][]importer.Importer)

	for _, source := range cmd.Sources {
		scanDir := "fs:" + ctx.CWD
		if source != "" {
			scanDir = source
		}

		// We are going to mutate this, so do a copy
		cmdOptsCopy := make(map[string]string)
		maps.Copy(cmdOptsCopy, cmd.Opts)

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
					if _, found := cmdOptsCopy[k]; !found {
						cmdOptsCopy[k] = v
					}
				}
			}
		}

		// Now that we have resolved the possible @ syntax let's apply the scandir.
		if _, found := cmdOptsCopy["location"]; !found {
			cmdOptsCopy["location"] = scanDir
		}

		excludes := exclude.NewRuleSet()
		if err := excludes.AddRulesFromArray(cmd.Excludes); err != nil {
			return 1, fmt.Errorf("failed to setup exclude rules: %w", err), objects.MAC{}, nil
		}

		importerOpts := ctx.ImporterOpts()
		importerOpts.Excludes = importerExcludes(cmd.Excludes)

		imp, err := importer.NewImporter(ctx.GetInner(), importerOpts, cmdOptsCopy)
		if err != nil {
			return 1, fmt.Errorf("failed to create an importer for %s: %s", scanDir, err), objects.MAC{}, nil
		}
		defer imp.Close(ctx)

		var (
			typ  = imp.Type()
			orig = imp.Origin()
		)

		importerKey := typ + ":" + orig
		sourcesPerOrig[importerKey] = append(sourcesPerOrig[importerKey], imp)

		if !cmd.NoProgress && (imp.Flags()&location.FLAG_STREAM) == 0 {
			imp, err := importer.NewImporter(ctx.GetInner(), importerOpts, cmdOptsCopy)
			if err != nil {
				return 1, fmt.Errorf("failed to create an importer for %s: %s", scanDir, err), objects.MAC{}, nil
			}
			defer imp.Close(ctx)
			sourcesPerOrigForStats[importerKey] = append(sourcesPerOrigForStats[importerKey], imp)
		}
	}

	// XXX - until we unlock multi-source
	if len(sourcesPerOrig) != 1 {
		return 1, fmt.Errorf("multi-source backup not supported yet"), objects.MAC{}, nil
	}

	if cmd.PackfileTempStorage == "memory" {
		cmd.PackfileTempStorage = ""
	} else {
		tmpDir, err := os.MkdirTemp(cmd.PackfileTempStorage, "plakar-backup-"+repo.Configuration().RepositoryID.String()+"-*")
		if err != nil {
			return 1, err, objects.NilMac, nil
		}
		cmd.PackfileTempStorage = tmpDir
		defer os.RemoveAll(cmd.PackfileTempStorage)
	}

	// Execute pre-backup hook
	if err := executeHook(ctx, cmd.PreHook); err != nil {
		return 1, fmt.Errorf("pre-backup hook failed: %w", err), objects.MAC{}, nil
	}

	snap, err := snapshot.Create(repo, repository.DefaultType, cmd.PackfileTempStorage, objects.NilMac, opts)
	if err != nil {
		ctx.GetLogger().Error("%s", err)
		return 1, err, objects.MAC{}, nil
	}
	defer snap.Close()

	if cmd.Job != "" {
		snap.Header.Job = cmd.Job
	}

	// Actual import of sources.
	for key, sourceImporters := range sourcesPerOrig {
		source, err := snapshot.NewSource(repo.AppContext(), sourceImporters...)
		if err != nil {
			return 1, err, objects.NilMac, nil
		}

		if err := source.SetExcludes(cmd.Excludes); err != nil {
			return 1, err, objects.MAC{}, nil
		}

		if cmd.DryRun {
			if err := dryrun(ctx, source, emitter); err != nil {
				return 1, err, objects.MAC{}, nil
			}
			return 0, nil, objects.MAC{}, nil
		}

		var parentVFS *vfs.Filesystem

		if cmd.Cache == "vfs" {
			parentID, _, err := locate.Match(repo, &locate.LocateOptions{
				Filters: locate.LocateFilters{
					Latest: true,
					Roots: []string{
						source.Root(),
					},
					Types: []string{
						source.Type(),
					},
					Origins: []string{
						source.Origin(),
					},
				},
			})
			if err != nil {
				return 1, nil, objects.MAC{}, err
			}

			if len(parentID) != 0 {
				parent, err := snapshot.Load(repo, parentID[0])
				if err != nil {
					fmt.Printf("Failed to load parent snapshot %x: %s\n", parentID[0], err)
				} else {
					defer parent.Close()

					parentVFS, err = parent.FilesystemWithCache()
					if err != nil {
						fmt.Printf("Failed to get parent VFS for snapshot %x: %s\n", parentID[0], err)
					}
				}
			}
		}
		snap.WithVFSCache(parentVFS)

		if !cmd.NoProgress && (source.Flags()&location.FLAG_STREAM) == 0 {
			source, err := snapshot.NewSource(repo.AppContext(), sourcesPerOrigForStats[key]...)
			if err != nil {
				return 1, err, objects.NilMac, nil
			}

			if err := source.SetExcludes(cmd.Excludes); err != nil {
				return 1, err, objects.MAC{}, nil
			}

			go func() {
				fsSummary := statistics(ctx, source)
				emitter.FilesystemSummary(
					fsSummary.FileCount,
					fsSummary.DirCount,
					fsSummary.SymlinkCount,
					fsSummary.XattrCount,
					fsSummary.TotalSize,
				)
			}()
		}

		if err := snap.Backup(source); err != nil {
			err = ctx.ErrorCause(err)
			if err := executeHook(ctx, cmd.FailHook); err != nil {
				ctx.GetLogger().Warn("post-backup fail hook failed: %s", err)
			}
			return 1, fmt.Errorf("failed to backup source: %w", err), objects.MAC{}, nil
		}
	}

	if err := snap.Commit(); err != nil {
		err = ctx.ErrorCause(err)
		if err := executeHook(ctx, cmd.FailHook); err != nil {
			ctx.GetLogger().Warn("post-backup fail hook failed: %s", err)
		}
		return 1, fmt.Errorf("failed to commit snapshot: %w", err), objects.MAC{}, nil
	}

	if cmd.OptCheck {
		_, err := cached.RebuildStateFromStore(ctx, repo.Configuration().RepositoryID, ctx.StoreConfig, false)
		if err != nil {
			return 1, fmt.Errorf("failed to rebuild state %w", err), objects.MAC{}, nil
		}

		checkOptions := &snapshot.CheckOptions{
			FastCheck: false,
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
			if err := executeHook(ctx, cmd.FailHook); err != nil {
				ctx.GetLogger().Warn("post-backup fail hook failed: %s", err)
			}
			return 1, fmt.Errorf("failed to check snapshot: %w", err), objects.MAC{}, nil
		}
	}

	// Execute post-backup hook
	if err := executeHook(ctx, cmd.PostHook); err != nil {
		ctx.GetLogger().Warn("post-backup hook failed: %s", err)
	}

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

func executeHook(ctx *appcontext.AppContext, hook string) error {
	if hook == "" {
		return nil
	}
	ctx.GetLogger().Info("executing hook: %s", hook)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/C", hook)
	default: // assume unix-esque
		cmd = exec.Command("/bin/sh", "-c", hook)
	}

	cmd.Stdout = ctx.Stdout
	cmd.Stderr = ctx.Stderr
	return cmd.Run()
}

func ack(record *connectors.Record, results chan<- *connectors.Result) {
	if results == nil {
		record.Close()
	} else {
		results <- record.Ok()
	}
}

func progress(ctx *appcontext.AppContext, imp importer.Importer, fn func(<-chan *connectors.Record, chan<- *connectors.Result)) error {
	var (
		size    = ctx.MaxConcurrency
		records = make(chan *connectors.Record, size)
		retch   = make(chan struct{}, 1)
	)

	var results chan *connectors.Result
	if (imp.Flags() & location.FLAG_NEEDACK) != 0 {
		results = make(chan *connectors.Result, size)
	}

	go func() {
		fn(records, results)
		if results != nil {
			close(results)
		}
		close(retch)
	}()

	err := imp.Import(ctx, records, results)
	<-retch
	return err
}

func dryrun(ctx *appcontext.AppContext, source *snapshot.Source, emitter *events.Emitter) error {
	var errors bool
	for _, imp := range source.Importers() {
		err := progress(ctx, imp, func(records <-chan *connectors.Record, results chan<- *connectors.Result) {
			for record := range records {
				ack(record, results)

				var (
					pathname = record.Pathname
					isDir    = false
				)

				if record.Err == nil && record.FileInfo.Lmode.IsDir() {
					isDir = true
				}

				if source.GetExcludes().IsExcluded(pathname, isDir) {
					continue
				}

				emitter.Path(pathname)
				switch {
				case record.Err != nil:
					errors = true
					if record.IsXattr {
						emitter.Xattr(pathname)
						emitter.XattrError(pathname, record.Err)
					} else if record.Target != "" {
						emitter.Symlink(pathname)
						emitter.SymlinkError(pathname, record.Err)
					} else if record.FileInfo.IsDir() {
						emitter.Directory(pathname)
						emitter.DirectoryError(pathname, record.Err)
					} else {
						emitter.File(pathname)
						emitter.FileError(pathname, record.Err)
					}
					emitter.PathError(pathname, record.Err)
				default:
					if record.IsXattr {
						emitter.Xattr(pathname)
						emitter.XattrOk(pathname, -1)
					} else if record.Target != "" {
						emitter.Symlink(pathname)
						emitter.SymlinkOk(pathname)
					} else if record.FileInfo.IsDir() {
						emitter.Directory(pathname)
						emitter.DirectoryOk(pathname, record.FileInfo)
					} else {
						emitter.File(pathname)
						emitter.FileOk(pathname, record.FileInfo)
					}
					emitter.PathOk(pathname)
				}
			}
		})
		if err != nil {
			return err
		}
	}
	if errors {
		return fmt.Errorf("failed to scan some files")
	}
	return nil
}

// We don't want to go through cached, if we need to refresh the state call
// Repository.RebuildState
var stateRefresher = func(ctx *appcontext.AppContext, repo *repository.Repository) func(mac objects.MAC, finalRefresh bool) error {
	return func(mac objects.MAC, finalRefresh bool) error {
		// If we are in the final refresh, turn this request into a fire and
		// forget one, to improve the UX.
		_, err := cached.RebuildStateFromStateFile(ctx, mac, repo.Configuration().RepositoryID, ctx.StoreConfig, finalRefresh)
		return err
	}
}

type FilesystemSummary struct {
	FileCount    uint64
	DirCount     uint64
	SymlinkCount uint64
	XattrCount   uint64
	TotalSize    uint64
}

func statistics(ctx *appcontext.AppContext, source *snapshot.Source) FilesystemSummary {
	errorCount := uint64(0)
	directoryCount := uint64(0)
	fileCount := uint64(0)
	symlinkCount := uint64(0)
	xattrCount := uint64(0)
	totalSize := uint64(0)

	for _, imp := range source.Importers() {
		progress(ctx, imp, func(records <-chan *connectors.Record, results chan<- *connectors.Result) {
			for record := range records {
				ack(record, results)

				var (
					pathname = record.Pathname
					isDir    = false
				)

				if record.Err == nil && record.FileInfo.Lmode.IsDir() {
					isDir = true
				}

				if source.GetExcludes().IsExcluded(pathname, isDir) {
					continue
				}

				switch {
				case record.Err != nil:
					errorCount++
				case record.IsXattr:
					xattrCount++
				case record.FileInfo.Lmode.IsDir():
					directoryCount++
				case record.FileInfo.Mode()&os.ModeSymlink != 0:
					symlinkCount++
				default:
					fileCount++
					totalSize += uint64(record.FileInfo.Size())
				}
			}
		})
	}

	return FilesystemSummary{
		FileCount:    fileCount,
		DirCount:     directoryCount,
		SymlinkCount: symlinkCount,
		XattrCount:   xattrCount,
		TotalSize:    totalSize,
	}
}
