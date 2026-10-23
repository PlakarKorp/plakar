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

	"github.com/PlakarKorp/plakar/utils"
)

// A -since / -before style value: an absolute date in one of the accepted
// layouts, or a human duration such as "2 weeks ago".
func FuzzTimeFlag(f *testing.F) {
	f.Add("2026-01-02")
	f.Add("2026-01-02T15:04:05Z")
	f.Add("2026/01/02 15:04")
	f.Add("3 days")
	f.Add("1h30m")
	f.Add("2 weeks ago")
	f.Add("-5y")

	f.Fuzz(func(t *testing.T, s string) {
		_, _ = utils.ParseTimeFlag(s)
	})
}
