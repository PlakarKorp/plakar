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

	"github.com/PlakarKorp/kloset/repository/state"
	"github.com/PlakarKorp/kloset/versioning"
)

// The fixed-size records of a repository state: deltas, deletions,
// packfile references and colourings.
func FuzzStateRecords(f *testing.F) {
	seeds, err := Seeds()
	if err != nil {
		f.Fatal(err)
	}
	for _, seed := range seeds["state_records"] {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = state.DeltaEntryFromBytes(data)
		_, _ = state.DeleteEntryFromBytes(data)
		_, _ = state.PackfileEntryFromBytes(data)
		_, _ = state.ColouredEntryFromBytes(data)
	})
}

// The configuration entries of a state, the one record type with
// variable-length fields.
func FuzzStateConfig(f *testing.F) {
	seeds, err := Seeds()
	if err != nil {
		f.Fatal(err)
	}
	for _, seed := range seeds["state_config"] {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = state.ConfigurationEntryFromBytes(data)
	})
}

// The metadata header every open of a repository reads before trusting
// anything else, in its msgpack form and as the raw header prefix.
func FuzzStateMetadata(f *testing.F) {
	seeds, err := Seeds()
	if err != nil {
		f.Fatal(err)
	}
	for _, seed := range seeds["state_metadata"] {
		f.Add(seed)
	}
	ver := versioning.FromString(state.VERSION)

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = state.MetadataFromBytes(data)
		_, _ = state.ReadHeader(bytes.NewReader(data), ver)
	})
}
