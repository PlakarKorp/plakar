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
)

func init() {
	subcommands.Register(Backup, 0, "backup")
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

func Backup(ctx *appcontext.AppContext, repo *repository.Repository, args []string) error {
	var (
		job            string
		check          bool
		opts           = make(map[string]string)
		dryRun         bool
		packfiles      string
		forceTimestamp time.Time
		prehook        string
		posthook       string
		failhook       string
		noXattr        bool
		cache          string
		noProgress     bool
		name           string
		category       string
		environment    string
		perimeter      string
	)

	var opt_ignore_files ignoreFlags
	var opt_ignore ignoreFlags
	var opt_tags tagFlags

	excludes := []string{}

	flags := flag.NewFlagSet("backup", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS] path\n", flags.Name())
		fmt.Fprintf(flags.Output(), "       %s [OPTIONS] @LOCATION\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.Var(&opt_tags, "tag", "comma-separated list of tags to apply to the snapshot")
	flags.StringVar(&name, "name", "default", "backup name")
	flags.StringVar(&category, "category", "", "backup category")
	flags.StringVar(&environment, "environment", "", "backup environment")
	flags.StringVar(&perimeter, "perimeter", "", "backup perimeter")
	flags.StringVar(&job, "job", "", "backup job")
	flags.Var(&opt_ignore_files, "ignore-file", "path to a file containing newline-separated gitignore patterns, treated as -ignore; can be specified multiple times")
	flags.Var(&opt_ignore, "ignore", "gitignore pattern to exclude files, can be specified multiple times to add several exclusion patterns")
	flags.StringVar(&packfiles, "packfiles", "", "memory or a path to a directory to store temporary packfiles")
	flags.BoolVar(&check, "check", false, "check the snapshot after creating it")
	flags.Var(utils.NewOptsFlag(opts), "o", "specify extra importer options")
	flags.BoolVar(&dryRun, "dry-run", false, "do not actually perform a backup")
	flags.BoolVar(&noXattr, "no-xattr", false, "do not back up extended attributes")
	flags.StringVar(&cache, "cache", "vfs", "path to store vfs cache, 'no' for uncached and 'vfs' for the default in memory cache")
	flags.BoolVar(&noProgress, "no-progress", false, "do not display progress")
	flags.StringVar(&prehook, "pre-hook", "", "pre hook command")
	flags.StringVar(&posthook, "post-hook", "", "post hook command")

	flags.Var(locate.NewTimeFlag(&forceTimestamp), "force-timestamp", "force a timestamp")
	flags.Parse(args)

	if !forceTimestamp.IsZero() {
		if forceTimestamp.After(time.Now()) {
			return fmt.Errorf("forced timestamp cannot be in the future")
		}
	}

	for _, ignoreFile := range opt_ignore_files {
		lines, err := LoadIgnoreFile(ignoreFile)
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

	tags := opt_tags.asList()
	if len(tags) == 0 {
		if envTags, ok := os.LookupEnv("PLAKAR_TAGS"); ok && envTags != "" {
			parts := strings.Split(envTags, ",")
			for _, t := range parts {
				t = strings.TrimSpace(t)
				if t != "" {
					tags = append(tags, t)
				}
			}
		}
	}

	Sources := flags.Args()
	if len(Sources) == 0 {
		Sources = append(Sources, "fs:"+ctx.CWD)
	}

	emitter := repo.Emitter("import")
	defer emitter.Close()

	builderOpts := &snapshot.BuilderOptions{
		Name:           name,
		Tags:           tags,
		Job:            job,
		Category:       category,
		Environment:    environment,
		Perimeter:      perimeter,
		NoXattr:        noXattr,
		StateRefresher: stateRefresher(ctx, repo),
	}

	if !forceTimestamp.IsZero() {
		builderOpts.ForcedTimestamp = forceTimestamp
	}

	sourcesPerOrig := make(map[string][]importer.Importer)
	// If we are doing a fake run for statistics instantiate separate importers,
	// otherwise it makes plugin development harder than needed.
	sourcesPerOrigForStats := make(map[string][]importer.Importer)

	for _, source := range Sources {
		scanDir := "fs:" + ctx.CWD
		if source != "" {
			scanDir = source
		}

		// We are going to mutate this, so do a copy
		cmdOptsCopy := make(map[string]string)
		maps.Copy(cmdOptsCopy, opts)

		if strings.HasPrefix(scanDir, "@") {
			remote, ok := ctx.Config.GetSource(scanDir[1:])
			if !ok {
				return fmt.Errorf("could not resolve importer: %s", scanDir)
			}
			if _, ok := remote["location"]; !ok {
				return fmt.Errorf("could not resolve importer location: %s", scanDir)
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

		e := exclude.NewRuleSet()
		if err := e.AddRulesFromArray(excludes); err != nil {
			return fmt.Errorf("failed to setup exclude rules: %w", err)
		}

		importerOpts := ctx.ImporterOpts()
		importerOpts.Excludes = excludes

		imp, err := importer.NewImporter(ctx.GetInner(), importerOpts, cmdOptsCopy)
		if err != nil {
			return fmt.Errorf("failed to create an importer for %s: %s", scanDir, err)
		}
		defer imp.Close(ctx)

		var (
			typ  = imp.Type()
			orig = imp.Origin()
		)

		importerKey := typ + ":" + orig
		sourcesPerOrig[importerKey] = append(sourcesPerOrig[importerKey], imp)

		if !noProgress && (imp.Flags()&location.FLAG_STREAM) == 0 {
			imp, err := importer.NewImporter(ctx.GetInner(), importerOpts, cmdOptsCopy)
			if err != nil {
				return fmt.Errorf("failed to create an importer for %s: %s", scanDir, err)
			}
			defer imp.Close(ctx)
			sourcesPerOrigForStats[importerKey] = append(sourcesPerOrigForStats[importerKey], imp)
		}
	}

	// XXX - until we unlock multi-source
	if len(sourcesPerOrig) != 1 {
		return fmt.Errorf("multi-source backup not supported yet")
	}

	if packfiles == "memory" {
		packfiles = ""
	} else {
		tmpDir, err := os.MkdirTemp(packfiles, "plakar-backup-"+repo.Configuration().RepositoryID.String()+"-*")
		if err != nil {
			return err
		}
		packfiles = tmpDir
		defer os.RemoveAll(packfiles)
	}

	// Execute pre-backup hook
	if err := executeHook(ctx, prehook); err != nil {
		return fmt.Errorf("pre-backup hook failed: %w", err)
	}

	snap, err := snapshot.Create(repo, repository.DefaultType, packfiles, objects.NilMac, builderOpts)
	if err != nil {
		ctx.GetLogger().Error("%s", err)
		return err
	}
	defer snap.Close()

	if job != "" {
		snap.Header.Job = job
	}

	// Actual import of sources.
	for key, sourceImporters := range sourcesPerOrig {
		source, err := snapshot.NewSource(repo.AppContext(), sourceImporters...)
		if err != nil {
			return err
		}

		if err := source.SetExcludes(excludes); err != nil {
			return err
		}

		if dryRun {
			if err := dryrun(ctx, source, emitter); err != nil {
				return err
			}
			return nil
		}

		var parentVFS *vfs.Filesystem

		if cache == "vfs" {
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
				return nil
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

		if !noProgress && (source.Flags()&location.FLAG_STREAM) == 0 {
			source, err := snapshot.NewSource(repo.AppContext(), sourcesPerOrigForStats[key]...)
			if err != nil {
				return err
			}

			if err := source.SetExcludes(excludes); err != nil {
				return err
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
			if err := executeHook(ctx, failhook); err != nil {
				ctx.GetLogger().Warn("post-backup fail hook failed: %s", err)
			}
			return fmt.Errorf("failed to backup source: %w", err)
		}
	}

	if err := snap.Commit(); err != nil {
		if err := executeHook(ctx, failhook); err != nil {
			ctx.GetLogger().Warn("post-backup fail hook failed: %s", err)
		}
		return fmt.Errorf("failed to commit snapshot: %w", err)
	}

	if check {
		_, err := cached.RebuildStateFromStore(ctx, repo.Configuration().RepositoryID, ctx.StoreConfig, false)
		if err != nil {
			return fmt.Errorf("failed to rebuild state %w", err)
		}

		checkOptions := &snapshot.CheckOptions{
			FastCheck: false,
		}

		checkSnap, err := snapshot.Load(repo, snap.Header.Identifier)
		if err != nil {
			return fmt.Errorf("failed to load snapshot: %w", err)
		}
		defer checkSnap.Close()

		checkCache, err := ctx.GetCache().Check()
		if err != nil {
			return err
		}
		defer checkCache.Close()

		checkSnap.SetCheckCache(checkCache)

		if err := checkSnap.Check("/", checkOptions); err != nil {
			if err := executeHook(ctx, failhook); err != nil {
				ctx.GetLogger().Warn("post-backup fail hook failed: %s", err)
			}
			return fmt.Errorf("failed to check snapshot: %w", err)
		}
	}

	// Execute post-backup hook
	if err := executeHook(ctx, posthook); err != nil {
		ctx.GetLogger().Warn("post-backup hook failed: %s", err)
	}

	return nil
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
