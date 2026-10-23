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

// Package fuzz holds the fuzz targets OSS-Fuzz runs against plakar: every
// parser that reads input plakar did not produce itself, whether from the
// command line, a configuration file, a plugin registry or a repository.
//
// Each target is an ordinary Go fuzz test, so `go test ./fuzz` runs the
// seeds as regression tests and `go test -fuzz=<Target> ./fuzz` fuzzes
// locally. OSS-Fuzz builds the same functions as libFuzzer binaries with
// oss-fuzz-build.sh in this directory; see README.md.
package fuzz
