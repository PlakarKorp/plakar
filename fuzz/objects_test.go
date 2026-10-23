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

	"github.com/PlakarKorp/kloset/objects"
)

// The msgpack records a repository stores for objects, chunks and cached
// paths, and the two textual forms of a MAC: hex on the command line and
// a JSON string in the API.
func FuzzObjectCodec(f *testing.F) {
	seeds, err := Seeds()
	if err != nil {
		f.Fatal(err)
	}
	for _, seed := range seeds["object_codec"] {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		if o, err := objects.NewObjectFromBytes(data); err == nil {
			_, _ = o.Serialize()
		}
		if c, err := objects.NewChunkFromBytes(data); err == nil {
			_, _ = c.Serialize()
		}
		if p, err := objects.NewCachedPathFromBytes(data); err == nil {
			_, _ = p.Serialize()
		}

		var mac objects.MAC
		_ = mac.ParseHex(string(data))
		_ = mac.UnmarshalJSON(data)
	})
}
