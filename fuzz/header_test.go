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

	"github.com/PlakarKorp/kloset/snapshot/header"
)

// A snapshot header as read back from a repository. A header the decoder
// accepts must serialize again: plakar rewrites headers it did not author
// when it syncs or tags a snapshot.
func FuzzSnapshotHeader(f *testing.F) {
	seeds, err := Seeds()
	if err != nil {
		f.Fatal(err)
	}
	for _, seed := range seeds["snapshot_header"] {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		h, err := header.NewFromBytes(data)
		if err != nil {
			return
		}
		if _, err := h.Serialize(); err != nil {
			t.Fatalf("accepted header does not serialize: %v", err)
		}
	})
}
