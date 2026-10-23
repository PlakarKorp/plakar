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
	"crypto/sha256"
	"testing"

	"github.com/PlakarKorp/kloset/packfile"
	"github.com/PlakarKorp/kloset/versioning"
)

// A whole packfile and its two trailing structures, the index and the
// footer, as read from storage. The footer's offsets and counts drive every
// later read, so this is where a hostile repository would lie first.
func FuzzPackfileCodec(f *testing.F) {
	seeds, err := Seeds()
	if err != nil {
		f.Fatal(err)
	}
	for _, seed := range seeds["packfile_codec"] {
		f.Add(seed)
	}
	ver := versioning.FromString(packfile.VERSION)

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = packfile.NewInMemoryFooterFromBytes(ver, data)
		_, _ = packfile.NewInMemoryIndexFromBytes(ver, data)

		p, err := packfile.NewPackfileInMemoryFromBytes(sha256.New(), ver, data)
		if err != nil {
			return
		}
		_ = p.Entries()
		_ = p.Size()
	})
}
