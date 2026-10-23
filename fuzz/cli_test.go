/*
 * Copyright (c) 2026 Gilles Chehade <gilles@poolp.org>
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

package fuzz

import (
	"io"
	"testing"

	"github.com/spf13/pflag"

	"github.com/PlakarKorp/plakar/subcommands"
)

// The command line rewrite that lets plakar accept -flag as well as --flag,
// followed by the pflag parse it feeds. The flag set mirrors the shapes
// plakar subcommands declare: one-letter long options, shorthands, bools
// that take no value and options that take the next argument.
func FuzzSingleDash(f *testing.F) {
	f.Add("-o", "out", "-quiet")
	f.Add("--name=x", "-v", "--")
	f.Add("-concurrency", "4", "-nx")
	f.Add("-o=", "-", "---")

	f.Fuzz(func(t *testing.T, a, b, c string) {
		flags := pflag.NewFlagSet("fuzz", pflag.ContinueOnError)
		flags.SetOutput(io.Discard)
		flags.String("o", "", "")
		flags.StringP("name", "n", "", "")
		flags.Bool("quiet", false, "")
		flags.BoolP("verbose", "v", false, "")
		flags.Int("concurrency", 0, "")

		args := []string{a, b, c}
		out, err := subcommands.SingleDash(flags, args)
		if err != nil {
			return
		}
		if len(out) < len(args) {
			t.Fatalf("SingleDash dropped arguments: %q -> %q", args, out)
		}
		_ = flags.Parse(out)
	})
}
