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
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"time"

	"github.com/PlakarKorp/kloset/connectors"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/packfile"
	"github.com/PlakarKorp/kloset/repository/state"
	"github.com/PlakarKorp/kloset/resources"
	"github.com/PlakarKorp/kloset/snapshot/header"
	"github.com/PlakarKorp/kloset/snapshot/vfs"
	"github.com/PlakarKorp/kloset/versioning"
	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
)

// Seeds returns well-formed inputs for the targets that take a single byte
// slice, keyed by the fuzzer name oss-fuzz-build.sh gives them. The targets
// add them with F.Add for `go test`, and seeds/main.go packs them into the
// seed corpora OSS-Fuzz starts from, where F.Add is a no-op. They are built,
// not stored, so they always match the serialization the current kloset
// writes; seeds_test.go checks that each one still decodes.
func Seeds() (map[string][][]byte, error) {
	seeds := map[string][][]byte{}
	add := func(name string, b []byte) {
		seeds[name] = append(seeds[name], b)
	}

	h, err := header.NewHeader("fuzz", objects.MAC{1}).Serialize()
	if err != nil {
		return nil, fmt.Errorf("snapshot header seed: %w", err)
	}
	add("snapshot_header", h)

	o, err := objects.NewObject().Serialize()
	if err != nil {
		return nil, fmt.Errorf("object seed: %w", err)
	}
	add("object_codec", o)
	c, err := objects.NewChunk().Serialize()
	if err != nil {
		return nil, fmt.Errorf("chunk seed: %w", err)
	}
	add("object_codec", c)
	add("object_codec", []byte(hex.EncodeToString(make([]byte, 32))))
	add("object_codec", []byte(`"`+hex.EncodeToString(make([]byte, 32))+`"`))

	// NewEntry names the entry after its FileInfo, and the decoder
	// refuses an entry with no name.
	record := &connectors.Record{Pathname: "/fuzz", FileInfo: objects.FileInfo{Lname: "fuzz"}}
	e, err := vfs.NewEntry("/", record).ToBytes()
	if err != nil {
		return nil, fmt.Errorf("vfs entry seed: %w", err)
	}
	add("vfs_entry", e)

	p, err := seedPackfile()
	if err != nil {
		return nil, fmt.Errorf("packfile seed: %w", err)
	}
	add("packfile_codec", p)

	when := time.Unix(1700000000, 0).UTC()
	add("state_records", (&state.DeltaEntry{Type: resources.RT_OBJECT, Blob: objects.MAC{1}}).ToBytes())
	add("state_records", (&state.DeleteEntry{BlobType: resources.RT_OBJECT, Blob: objects.MAC{2}}).ToBytes())
	add("state_records", (&state.PackfileEntry{Packfile: objects.MAC{3}, Timestamp: when}).ToBytes())
	add("state_records", (&state.ColouredEntry{Type: resources.RT_OBJECT, Blob: objects.MAC{4}, When: when}).ToBytes())

	add("state_config", (&state.ConfigurationEntry{Key: "fuzz", Value: []byte("v"), CreatedAt: when}).ToBytes())
	m, err := msgpack.Marshal(&state.Metadata{
		Parent:    objects.MAC{5},
		Version:   versioning.FromString(state.VERSION),
		Timestamp: when,
		Serial:    uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	})
	if err != nil {
		return nil, fmt.Errorf("state metadata seed: %w", err)
	}
	add("state_metadata", m)

	return seeds, nil
}

func seedPackfile() ([]byte, error) {
	p, err := packfile.NewPackfileInMemory(func() hash.Hash { return sha256.New() })
	if err != nil {
		return nil, err
	}
	ver := versioning.FromString(packfile.VERSION)
	if err := p.AddBlob(resources.RT_OBJECT, ver, objects.MAC{1}, []byte("first blob"), 0); err != nil {
		return nil, err
	}
	if err := p.AddBlob(resources.RT_CHUNK, ver, objects.MAC{2}, []byte("second blob"), 0); err != nil {
		return nil, err
	}
	rd, _, err := p.Serialize(func(r io.Reader) (io.Reader, error) { return r, nil })
	if err != nil {
		return nil, err
	}
	serialized, err := io.ReadAll(rd)
	if err != nil {
		return nil, err
	}
	// Serialize appends the encoded footer's length, which the storage
	// layer strips before handing the bytes to the decoder. With the
	// identity encoder above that is the last four bytes.
	if len(serialized) < 4 {
		return nil, fmt.Errorf("serialized packfile too short: %d bytes", len(serialized))
	}
	return serialized[:len(serialized)-4], nil
}
