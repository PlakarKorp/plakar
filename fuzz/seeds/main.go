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

// Command seeds writes one <fuzzer>_seed_corpus.zip per seeded target into
// the directory given as its only argument, which is $OUT when OSS-Fuzz
// runs it from oss-fuzz-build.sh.
package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/PlakarKorp/plakar/fuzz"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: seeds <output directory>")
		os.Exit(2)
	}
	seeds, err := fuzz.Seeds()
	if err != nil {
		fmt.Fprintf(os.Stderr, "seeds: %v\n", err)
		os.Exit(1)
	}
	for name, inputs := range seeds {
		if err := writeCorpus(filepath.Join(os.Args[1], name+"_seed_corpus.zip"), inputs); err != nil {
			fmt.Fprintf(os.Stderr, "seeds: %v\n", err)
			os.Exit(1)
		}
	}
}

func writeCorpus(path string, inputs [][]byte) error {
	fp, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fp.Close()

	zw := zip.NewWriter(fp)
	for _, input := range inputs {
		sum := sha256.Sum256(input)
		name := hex.EncodeToString(sum[:])
		w, err := zw.Create(name)
		if err != nil {
			return fmt.Errorf("%s: %s: %w", path, name, err)
		}
		if _, err := w.Write(input); err != nil {
			return fmt.Errorf("%s: %s: %w", path, name, err)
		}
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if err := fp.Close(); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}
