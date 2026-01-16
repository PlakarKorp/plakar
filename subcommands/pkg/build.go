/*
 * Copyright (c) 2025 Omar Polo <omar.polo@plakar.io>
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

package pkg

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/pkg"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"golang.org/x/mod/semver"
)

var namere = regexp.MustCompile("^[_a-zA-Z0-9]+$")

type PkgBuild struct {
	subcommands.SubcommandBase

	Recipe pkg.Recipe
}

func (cmd *PkgBuild) Parse(ctx *appcontext.AppContext, args []string) error {
	flags := flag.NewFlagSet("pkg build", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s recipe.yaml\n",
			flags.Name())
	}

	flags.Parse(args)

	if flags.NArg() != 1 {
		return fmt.Errorf("wrong usage")
	}

	recipe := flags.Arg(0)
	if err := getRecipe(ctx, recipe, &cmd.Recipe); err != nil {
		return fmt.Errorf("failed to parse the %q recipe: %w", flags.Arg(0), err)
	}

	if !namere.Match([]byte(cmd.Recipe.Name)) {
		return fmt.Errorf("not a valid plugin name: %s", cmd.Recipe.Name)
	}
	if !semver.IsValid(cmd.Recipe.Version) {
		return fmt.Errorf("not a valid version string: %s", cmd.Recipe.Version)
	}

	return nil
}

func (cmd *PkgBuild) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	recipe := &cmd.Recipe

	pattern := fmt.Sprintf("build-%s-%s-*", recipe.Name, recipe.Version)
	datadir, err := os.MkdirTemp("", pattern)
	if err != nil {
		return 1, fmt.Errorf("failed to create a temp dir: %w", err)
	}
	defer os.RemoveAll(datadir)

	if err := clone(datadir, recipe); err != nil {
		return 1, fmt.Errorf("failed to clone %s: %s: %w", recipe.Repository, recipe.Version, err)
	}

	args := []string{"-C", datadir}

	if os.Getenv("GOOS") == "windows" || runtime.GOOS == "windows" {
		args = append(args, "EXT=.exe")
	}

	make := exec.Command("make", args...)
	fmt.Fprintln(ctx.Stderr, make.String())
	if err := make.Run(); err != nil {
		return 1, fmt.Errorf("make failed: %w", err)
	}

	manifest := filepath.Join(datadir, "manifest.yaml")

	// a bit hacky, needed until we move the plugin routines from
	// commands to a lib:
	create := PkgCreate{}
	if err := create.Parse(ctx, []string{manifest}); err != nil {
		return 1, err
	}
	create.Out = recipe.PkgName()
	return create.Execute(ctx, repo)
}

func clone(destdir string, recipe *pkg.Recipe) error {
	git := exec.Command("git", "clone", "--depth=1", "--branch", recipe.Version,
		recipe.Repository, destdir)
	if err := git.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	// TODO: how to check the checksum?  should it be the commit
	// id pointed by the tag?  (or the tag itself?)

	return nil
}
