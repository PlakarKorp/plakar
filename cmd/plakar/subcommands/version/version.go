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

package version

import (
	"flag"
	"fmt"

	"github.com/PlakarKorp/plakar/cmd/plakar/subcommands"
	"github.com/PlakarKorp/plakar/context"
	"github.com/PlakarKorp/plakar/repository"
	"golang.org/x/mod/semver"
)

const VERSION = "v0.4.22-alpha"

func init() {
	subcommands.Register("version", cmd_version)
}

func cmd_version(ctx *context.Context, _ *repository.Repository, args []string) int {
	flags := flag.NewFlagSet("version", flag.ExitOnError)
	flags.Parse(args)

	if !semver.IsValid(VERSION) {
		panic("invalid version string: " + VERSION)
	}
	fmt.Println(VERSION)

	return 0
}