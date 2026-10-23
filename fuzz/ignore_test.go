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
	"bytes"
	"testing"

	"github.com/PlakarKorp/kloset/exclude"
)

// An exclude file, in the gitignore syntax `plakar backup -exclude-file`
// reads, matched against a path.
func FuzzIgnoreFile(f *testing.F) {
	f.Add([]byte("*.o\n!keep.o\n# comment\nbuild/\n"), "src/main.o", false)
	f.Add([]byte("/abs/**/x\n\\#literal\ntrailing\\ \n"), "/abs/a/b/x", true)
	f.Add([]byte("[a-z]*.{go,rs}\n**\n"), "pkg/main.go", false)

	f.Fuzz(func(t *testing.T, rules []byte, path string, isDir bool) {
		ruleset := exclude.NewRuleSet()
		if err := ruleset.AddRulesFromReader(bytes.NewReader(rules)); err != nil {
			return
		}
		_, _, _ = ruleset.Match(path, isDir)
	})
}
