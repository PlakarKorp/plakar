package main

import (
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PlakarKorp/kloset/caching"
	"github.com/PlakarKorp/kloset/caching/pebble"
	"github.com/PlakarKorp/kloset/connectors/storage"
	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/logging"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/spf13/cobra"
)

// completionBudget is what we give ourselves to answer.  A shell waits on us
// while the user holds TAB, and an unreachable store takes seconds to report
// itself, so we would rather offer nothing than hang the terminal.
const completionBudget = 700 * time.Millisecond

// completeSnapshots offers the snapshots of the repository, and the entries of
// a snapshot once the argument names one.
//
// It is deliberately quiet: any failure returns no candidates at all, because
// the shell shows whatever we print and a diagnostic would land in the middle
// of the user's command line.
func completeSnapshots(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	none := cobra.ShellCompDirectiveNoFileComp

	// Opening a store can block for as long as the network takes to give
	// up, which is far longer than a TAB may hang the shell.  Nothing here
	// is worth waiting for, so we abandon the attempt rather than cancel
	// it: the process is about to exit anyway.
	done := make(chan []string, 1)
	go func() {
		done <- completeSnapshotsSlow(toComplete)
	}()

	var out []string
	select {
	case out = <-done:
	case <-time.After(completionBudget):
		return nil, none
	}

	// A snapshot is still waiting for its ":path", and a directory for the
	// rest of the path, so the shell must not close the word behind us.
	if len(out) > 0 && !isComplete(out) {
		return out, none | cobra.ShellCompDirectiveNoSpace
	}
	return out, none
}

// isComplete reports whether every candidate names something the user cannot
// type further into: a file inside a snapshot.
func isComplete(candidates []string) bool {
	for _, c := range candidates {
		if !strings.Contains(c, ":") || strings.HasSuffix(c, "/") {
			return false
		}
	}
	return true
}

func completeSnapshotsSlow(toComplete string) []string {
	snapID, path, hasPath := strings.Cut(toComplete, ":")

	ctx, repo, closer, err := completionRepository()
	if err != nil {
		return nil
	}
	defer closer()

	if hasPath {
		return completeEntries(ctx, repo, snapID, path)
	}
	return completeSnapshotIDs(ctx, repo, snapID)
}

func completeSnapshotIDs(ctx *appcontext.AppContext, repo *repository.Repository, prefix string) []string {
	ids, err := locate.LocateSnapshotIDs(repo, locate.NewDefaultLocateOptions())
	if err != nil {
		return nil
	}

	var out []string
	for _, id := range ids {
		if ctx.Err() != nil {
			return out
		}

		// The short ID is the head of the identifier, so there is no
		// need to load the snapshot to spell it.
		short := hex.EncodeToString(id[:4])
		if strings.HasPrefix(short, prefix) {
			out = append(out, short)
		}
	}
	return out
}

// completeEntries offers what lives directly under the directory the user has
// typed so far, so that walking a snapshot costs one directory at a time.
func completeEntries(ctx *appcontext.AppContext, repo *repository.Repository, snapID, path string) []string {
	snap, _, err := locate.OpenSnapshotByPath(repo, snapID)
	if err != nil {
		return nil
	}
	defer snap.Close()

	fs, err := snap.Filesystem()
	if err != nil {
		return nil
	}

	dir := path
	if !strings.HasSuffix(dir, "/") {
		dir = filepath.Dir(dir)
	}
	if dir == "." {
		dir = "/"
	}

	entry, err := fs.GetEntry(dir)
	if err != nil || !entry.Stat().Mode().IsDir() {
		return nil
	}

	dents, err := entry.Getdents(fs)
	if err != nil {
		return nil
	}

	// The vfs can hand back the same name twice, once as a directory and
	// once not; keep the directory, which is the one worth descending into.
	isDir := make(map[string]bool)
	var names []string
	for child, err := range dents {
		if err != nil || ctx.Err() != nil {
			break
		}
		name := child.Name()
		if _, seen := isDir[name]; !seen {
			names = append(names, name)
		}
		isDir[name] = isDir[name] || child.Stat().Mode().IsDir()
	}

	var out []string
	for _, name := range names {
		full := filepath.Join(dir, name)
		if isDir[name] {
			full += "/"
		}
		if strings.HasPrefix(full, path) {
			out = append(out, snapID+":"+full)
		}
	}
	return out
}

// completionRepository opens the repository the command line points at, under
// the budget above.  The returned closer must always be called.
func completionRepository() (*appcontext.AppContext, *repository.Repository, func(), error) {
	ctx := appcontext.NewAppContext()
	// Anything we would print lands in the middle of the command line the
	// user is typing, so the logger goes nowhere.
	ctx.SetLogger(logging.NewLogger(io.Discard, io.Discard))
	nop := func() { ctx.Close() }

	configDir, err := utils.GetConfigDir("plakar")
	if err != nil {
		return nil, nil, nop, err
	}
	ctx.ConfigDir = configDir
	if err := ctx.ReloadConfig(); err != nil {
		return nil, nil, nop, err
	}

	cacheDir, err := utils.GetCacheDir("plakar")
	if err != nil {
		return nil, nil, nop, err
	}
	ctx.CacheDir = cacheDir
	ctx.SetCache(caching.NewManager(pebble.Constructor(cacheDir)))
	nop = func() { ctx.GetCache().Close(); ctx.Close() }

	repositoryPath := completionRepositoryPath(ctx)
	storeConfig, err := ctx.Config.GetRepository(repositoryPath)
	if err != nil {
		return nil, nil, nop, err
	}
	passphrase, err := getPassphraseFromEnv(ctx, storeConfig, repositoryPath)
	if err != nil {
		return nil, nil, nop, err
	}
	if passphrase != "" {
		ctx.KeyFromFile = passphrase
	}

	store, serialized, err := storage.Open(ctx.GetInner(), storeConfig)
	if err != nil {
		return nil, nil, nop, err
	}

	closer := func() {
		store.Close(ctx)
		ctx.GetCache().Close()
		ctx.Close()
	}

	if err := completionUnlock(ctx, serialized); err != nil {
		closer()
		return nil, nil, func() {}, err
	}

	repo, err := repository.New(ctx.GetInner(), ctx.GetSecret(), store, serialized)
	if err != nil {
		closer()
		return nil, nil, func() {}, err
	}

	return ctx, repo, func() { repo.Close(); closer() }, nil
}

// completionUnlock only handles the repositories we can open without asking
// anything: prompting for a passphrase while the user is holding TAB is not an
// option, so an encrypted repository simply offers nothing.
func completionUnlock(ctx *appcontext.AppContext, serialized []byte) error {
	config, err := storage.NewConfigurationFromWrappedBytes(serialized)
	if err != nil {
		return err
	}
	if config.Encryption == nil {
		return nil
	}

	passphrase, err := getPassphraseFromEnv(ctx, ctx.StoreConfig)
	if err != nil || passphrase == "" {
		return ErrCantUnlock
	}
	ctx.KeyFromFile = passphrase
	return setupEncryption(ctx, config)
}

// completionRepositoryPath resolves the repository the way entryPoint does,
// including the "at REPOSITORY" prefix the shell passes back to us.
func completionRepositoryPath(ctx *appcontext.AppContext) string {
	args := os.Args[1:]
	for i, arg := range args {
		if arg == "at" && i+1 < len(args) {
			return args[i+1]
		}
	}

	if path := os.Getenv("PLAKAR_REPOSITORY"); path != "" {
		return path
	}
	if def := ctx.Config.DefaultRepository; def != "" {
		return "@" + def
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return "fs:" + filepath.Join(home, ".plakar")
}

// installSnapshotCompletion teaches the commands that take a snapshot how to
// complete one.
func installSnapshotCompletion(root *cobra.Command) {
	for _, path := range snapshotArgCommands {
		c, _, err := root.Find(strings.Split(path, " "))
		if err != nil || c == root {
			continue
		}
		c.ValidArgsFunction = completeSnapshots
	}
}

// snapshotArgCommands are the commands whose arguments name a snapshot.
var snapshotArgCommands = []string{
	"archive",
	"cat",
	"check",
	"diff",
	"digest",
	"dup",
	"info",
	"ls",
	"mount",
	"restore",
	"rm",
	"diag snapshot",
	"diag vfs",
	"diag xattr",
	"diag contenttype",
	"diag dirpack",
	"diag chunks",
}
