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

package diff

import (
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/kloset/snapshot/vfs"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/alecthomas/chroma/quick"
	"github.com/pmezard/go-difflib/difflib"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Diff{} }, 0, "diff")
}

func (cmd *Diff) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("diff", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS] SNAPSHOT:PATH SNAPSHOT[:PATH]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}
	flags.BoolVar(&cmd.Highlight, "highlight", false, "highlight output")
	flags.BoolVar(&cmd.Recursive, "recursive", false, "recursive diff of directories")
	flags.Parse(args)

	if flags.NArg() == 1 {
		cmd.Path1 = flags.Arg(0)
		cmd.Path2 = ""
	} else if flags.NArg() == 2 {
		cmd.Path1 = flags.Arg(0)
		cmd.Path2 = flags.Arg(1)
	} else {
		return fmt.Errorf("needs at least a snapshot ID and/or snapshot file to diff")
	}
	cmd.RepositorySecret = ctx.GetSecret()

	return nil
}

type Diff struct {
	subcommands.SubcommandBase

	Highlight bool
	Recursive bool
	Path1     string
	Path2     string
}

func (cmd *Diff) Name() string {
	return "diff"
}

func (cmd *Diff) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	snap1, pathname1, err := locate.OpenSnapshotByPath(repo, cmd.Path1)
	if err != nil {
		return 1, fmt.Errorf("diff: could not open snapshot: %s", cmd.Path1)
	}
	defer snap1.Close()
	vfs1, err := snap1.Filesystem()
	if err != nil {
		return 1, fmt.Errorf("diff: could not get filesystem for snapshot: %s", cmd.Path1)
	}
	id1 := fmt.Sprintf("%x", snap1.Header.GetIndexShortID())

	var pathname2 string
	var id2 string
	var vfs2 fs.FS

	if cmd.Path2 == "" {
		vfs2 = os.DirFS("/")
		id2 = "local"
	} else {
		var snap2 *snapshot.Snapshot
		snap2, pathname2, err = locate.OpenSnapshotByPath(repo, cmd.Path2)
		if err != nil {
			return 1, fmt.Errorf("diff: could not open snapshot: %s", cmd.Path2)
		}
		defer snap2.Close()
		vfs2, err = snap2.Filesystem()
		if err != nil {
			return 1, fmt.Errorf("diff: could not get filesystem for snapshot: %s", cmd.Path2)
		}
		id2 = fmt.Sprintf("%x", snap2.Header.GetIndexShortID())
	}

	var diff string

	if pathname1 == "" && pathname2 == "" {
		pathname1 = "/"
		pathname2 = "/"
	} else if pathname1 == "" && pathname2 != "" {
		pathname1 = pathname2
	} else if pathname1 != "" && pathname2 == "" {
		pathname2 = pathname1
	}

	diff, err = cmd.diff_pathnames(ctx, id1, vfs1, pathname1, id2, vfs2, pathname2)
	if err != nil {
		return 1, fmt.Errorf("diff: could not diff pathnames: %w", err)
	}

	if cmd.Highlight {
		err = quick.Highlight(os.Stdout, diff, "diff", "terminal", "dracula")
		if err != nil {
			return 1, fmt.Errorf("diff: could not highlight diff: %w", err)
		}
	} else {
		fmt.Printf("%s", diff)
	}
	return 0, nil
}

func (cmd *Diff) diff_pathnames(ctx *appcontext.AppContext, id1 string, vfs1 fs.FS, pathname1 string, id2 string, vfs2 fs.FS, pathname2 string) (string, error) {
	fsobj1, err := vfs1.Open(pathname1)
	if err != nil {
		return "", fmt.Errorf("could not open path %s in snapshot %s: %w", pathname1, id1, err)
	}
	defer fsobj1.Close()

	if _, ok := vfs2.(*vfs.Filesystem); !ok {
		// on non vfs.Filesystem, strip root !
		pathname2 = strings.TrimPrefix(pathname2, "/")
	}

	fsobj2, err := vfs2.Open(pathname2)
	if err != nil {
		return "", fmt.Errorf("could not open path %s in snapshot %s: %w", pathname2, id2, err)
	}
	defer fsobj2.Close()

	st1, err := fsobj1.Stat()
	if err != nil {
		return "", fmt.Errorf("could not stat path %s: %w", id1, err)
	}
	st2, err := fsobj2.Stat()
	if err != nil {
		return "", fmt.Errorf("could not stat path %ss: %w", id2, err)
	}

	if st1.IsDir() && st2.IsDir() {
		if cmd.Recursive {
			return cmd.diff_directories_recursive(ctx, vfs1, pathname1, vfs2, pathname1)
		}
		return cmd.diff_directories_flat(ctx, pathname1, fsobj1, pathname2, fsobj2)
	} else if st1.IsDir() || st2.IsDir() {
		return "", fmt.Errorf("can't diff different file types")
	} else {
		return cmd.diff_readers(id1, pathname1, fsobj1, id2, pathname2, fsobj2)
	}
}

func (cmd *Diff) diff_directories_flat(_ *appcontext.AppContext, pathname1 string, fsobj1 fs.File, pathname2 string, fsobj2 fs.File) (string, error) {
	// non VFS have their / stripped, reintroduce it
	if !strings.HasPrefix(pathname2, "/") {
		pathname2 = "/" + pathname2 // Ensure pathname starts with a slash
	}

	dir1, ok1 := fsobj1.(fs.ReadDirFile)
	dir2, ok2 := fsobj2.(fs.ReadDirFile)
	if !ok1 || !ok2 {
		return "", fmt.Errorf("both fs.File must implement fs.ReadDirFile")
	}

	entries1, err1 := dir1.ReadDir(-1)
	entries2, err2 := dir2.ReadDir(-1)
	if err1 != nil {
		return "", fmt.Errorf("error reading directory 1: %w", err1)
	}
	if err2 != nil {
		return "", fmt.Errorf("error reading directory 2: %w", err2)
	}

	map1 := map[string]fs.DirEntry{}
	map2 := map[string]fs.DirEntry{}
	for _, e := range entries1 {
		map1[e.Name()] = e
	}
	for _, e := range entries2 {
		map2[e.Name()] = e
	}

	var output strings.Builder
	visited := map[string]bool{}

	for name, e1 := range map1 {
		visited[name] = true
		if e2, ok := map2[name]; ok {
			if e1.IsDir() && e2.IsDir() {
				output.WriteString(fmt.Sprintf("Common subdirectories: %s and %s\n", name, name))
			} else if e1.IsDir() != e2.IsDir() {
				output.WriteString(fmt.Sprintf("File type mismatch: %s (dir=%v) vs %s (dir=%v)\n", name, e1.IsDir(), name, e2.IsDir()))
			}
		} else {
			output.WriteString(fmt.Sprintf("Only in %s: %s\n", pathname1, name))
		}
	}
	for name := range map2 {
		if !visited[name] {
			output.WriteString(fmt.Sprintf("Only in %s: %s\n", pathname2, name))
		}
	}

	return output.String(), nil
}

func (cmd *Diff) diff_directories_recursive(ctx *appcontext.AppContext, fs1 fs.FS, path1 string, fs2 fs.FS, path2 string) (string, error) {
	var output strings.Builder

	entries1, err1 := fs.ReadDir(fs1, path1)
	entries2, err2 := fs.ReadDir(fs2, path2)

	if err1 != nil && err2 != nil {
		return "", fmt.Errorf("cannot read both directories: %w / %w", err1, err2)
	}

	map1 := make(map[string]fs.DirEntry)
	map2 := make(map[string]fs.DirEntry)

	for _, e := range entries1 {
		map1[e.Name()] = e
	}
	for _, e := range entries2 {
		map2[e.Name()] = e
	}

	allNames := make(map[string]struct{})
	for name := range map1 {
		allNames[name] = struct{}{}
	}
	for name := range map2 {
		allNames[name] = struct{}{}
	}

	var sortedNames []string
	for name := range allNames {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	for _, name := range sortedNames {
		e1, ok1 := map1[name]
		e2, ok2 := map2[name]

		full1 := path.Join(path1, name)
		full2 := path.Join(path2, name)

		// non VFS have their / stripped, reintroduce it
		if !strings.HasPrefix(path2, "/") {
			path2 = "/" + path2 // Ensure pathname starts with a slash
		}

		switch {
		case ok1 && !ok2:
			output.WriteString(fmt.Sprintf("Only in %s: %s\n", path1, name))

		case !ok1 && ok2:
			output.WriteString(fmt.Sprintf("Only in %s: %s\n", path2, name))

		case ok1 && ok2:
			if e1.IsDir() && e2.IsDir() {
				output.WriteString(fmt.Sprintf("Common subdirectories: %s and %s\n", full1, full2))
				sub, err := cmd.diff_directories_recursive(ctx, fs1, full1, fs2, full2)
				if err != nil {
					return "", err
				}
				output.WriteString(sub)

			} else if e1.IsDir() != e2.IsDir() {
				output.WriteString(fmt.Sprintf("File type mismatch: %s vs %s\n", full1, full2))
			}
		}
	}

	return output.String(), nil

}

func (cmd *Diff) diff_readers(id1 string, pathname1 string, rd1 io.Reader, id2 string, pathname2 string, rd2 io.Reader) (string, error) {
	buf1, err := io.ReadAll(rd1)
	if err != nil {
		return "", err
	}
	buf2, err := io.ReadAll(rd2)
	if err != nil {
		return "", err
	}

	// non VFS have their / stripped, reintroduce it
	if !strings.HasPrefix(pathname2, "/") {
		pathname2 = "/" + pathname2 // Ensure pathname starts with a slash
	}

	FromFile := fmt.Sprintf("%s:%s", id1, utils.SanitizeText(pathname1))
	ToFile := fmt.Sprintf("%s:%s", id2, utils.SanitizeText(pathname2))
	ToFile = strings.TrimPrefix(ToFile, "local:") // Remove leading slash for better readability

	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(buf1)),
		B:        difflib.SplitLines(string(buf2)),
		FromFile: FromFile,
		ToFile:   ToFile,
		Context:  3,
	}
	text, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		return "", err
	}
	return text, nil
}
