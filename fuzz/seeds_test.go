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
	"errors"
	"testing"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/packfile"
	"github.com/PlakarKorp/kloset/repository/state"
	"github.com/PlakarKorp/kloset/snapshot/header"
	"github.com/PlakarKorp/kloset/snapshot/vfs"
	"github.com/PlakarKorp/kloset/versioning"
)

var errNoDecoder = errors.New("no state record decoder accepts this seed")

// A seed that no longer decodes is worse than none: the corpus OSS-Fuzz
// starts from would then exit every target at its first return. Each
// corpus is decoded here with the decoder its target feeds it to, so a
// change in kloset's serialization fails this test rather than silently
// weakening the fuzzers.
func TestSeedsDecode(t *testing.T) {
	seeds, err := Seeds()
	if err != nil {
		t.Fatal(err)
	}

	decoders := map[string]func([]byte) error{
		"snapshot_header": func(b []byte) error {
			_, err := header.NewFromBytes(b)
			return err
		},
		"object_codec": func(b []byte) error {
			var mac objects.MAC
			if _, err := objects.NewObjectFromBytes(b); err == nil {
				return nil
			}
			if _, err := objects.NewChunkFromBytes(b); err == nil {
				return nil
			}
			if err := mac.ParseHex(string(b)); err == nil {
				return nil
			}
			return mac.UnmarshalJSON(b)
		},
		"vfs_entry": func(b []byte) error {
			_, err := vfs.EntryFromBytes(b)
			return err
		},
		"packfile_codec": func(b []byte) error {
			_, err := packfile.NewPackfileInMemoryFromBytes(sha256.New(), versioning.FromString(packfile.VERSION), b)
			return err
		},
		"state_records": func(b []byte) error {
			for _, decode := range []func([]byte) error{
				func(b []byte) error { _, err := state.DeltaEntryFromBytes(b); return err },
				func(b []byte) error { _, err := state.DeleteEntryFromBytes(b); return err },
				func(b []byte) error { _, err := state.PackfileEntryFromBytes(b); return err },
				func(b []byte) error { _, err := state.ColouredEntryFromBytes(b); return err },
			} {
				if err := decode(b); err == nil {
					return nil
				}
			}
			return errNoDecoder
		},
		"state_config": func(b []byte) error {
			_, err := state.ConfigurationEntryFromBytes(b)
			return err
		},
		"state_metadata": func(b []byte) error {
			_, err := state.MetadataFromBytes(b)
			return err
		},
	}

	for name, inputs := range seeds {
		decode, ok := decoders[name]
		if !ok {
			t.Errorf("%s: seeded but no decoder listed here", name)
			continue
		}
		if len(inputs) == 0 {
			t.Errorf("%s: no seeds", name)
		}
		for i, input := range inputs {
			if err := decode(input); err != nil {
				t.Errorf("%s seed %d does not decode: %v", name, i, err)
			}
		}
	}
	for name := range decoders {
		if _, ok := seeds[name]; !ok {
			t.Errorf("%s: decoder listed but nothing seeded", name)
		}
	}
}
