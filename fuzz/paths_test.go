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
	"testing"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/snapshot/header"

	"github.com/PlakarKorp/plakar/utils"
)

// The "snapshotID:path" argument most subcommands take, as plakar and
// kloset each split it.
func FuzzSnapshotPath(f *testing.F) {
	f.Add("abc123:/etc/passwd")
	f.Add("/only/a/path")
	f.Add("abc123:")
	f.Add("a:b:c")

	f.Fuzz(func(t *testing.T, s string) {
		_, _ = utils.ParseSnapshotID(s)
		_, _ = locate.ParseSnapshotPath(s)
	})
}

// The -sort option: a comma-separated list of field names, optionally
// prefixed with "-", for snapshot headers and for file entries.
func FuzzSortKeys(f *testing.F) {
	f.Add("Timestamp,-Name")
	f.Add("-ModTime,Size,Size")
	f.Add(",,")

	f.Fuzz(func(t *testing.T, s string) {
		_, _ = header.ParseSortKeys(s)
		_, _ = objects.ParseFileInfoSortKeys(s)
	})
}
