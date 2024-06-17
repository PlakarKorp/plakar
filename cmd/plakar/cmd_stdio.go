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

package main

import (
	"flag"

	"github.com/PlakarLabs/plakar/server/plakard"
)

func init() {
	// registerCommand("stdio", cmd_stdio)
}

func cmd_stdio(ctx Plakar, args []string) int {
	var noDelete bool

	flags := flag.NewFlagSet("stdio", flag.ExitOnError)
	flags.BoolVar(&noDelete, "no-delete", false, "disable delete operations")
	flags.Parse(args)

	options := &plakard.ServerOptions{
		NoDelete: noDelete,
	}
	if err := plakard.Stdio(options); err != nil {
		return 1
	}
	return 0
}
