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

// Package archive tags archives from their extension alone, without reading
// their content.
package archive

import (
	"context"
	"path"
	"strings"

	"github.com/PlakarKorp/kloset/connectors"
	"github.com/PlakarKorp/kloset/connectors/analyzer"
)

var kinds = map[string]string{
	".zip": "archive.zip",
	".tar": "archive.tar",
	".tgz": "archive.tar.gzip",
	".gz":  "archive.gzip",
	".bz2": "archive.bzip2",
	".xz":  "archive.xz",
	".zst": "archive.zstd",
	".7z":  "archive.7z",
	".rar": "archive.rar",
}

func init() {
	if err := analyzer.Register("archive", New); err != nil {
		panic(err)
	}
}

type Archive struct{}

func New(ctx context.Context, opts *connectors.Options, name string, config map[string]string) (analyzer.Analyzer, error) {
	return &Archive{}, nil
}

func (a *Archive) Name() string          { return "archive" }
func (a *Archive) Version() string       { return "0.1.0" }
func (a *Archive) Scope() analyzer.Scope { return analyzer.ScopeFile }

func (a *Archive) Filter() analyzer.Filter {
	exts := make([]string, 0, len(kinds))
	for ext := range kinds {
		exts = append(exts, ext)
	}
	return analyzer.Filter{Extensions: exts, Content: analyzer.ContentNone}
}

func (a *Archive) NewProcessor(ctx context.Context, file analyzer.File) (analyzer.Processor, error) {
	return &processor{class: kinds[strings.ToLower(path.Ext(file.Path))]}, nil
}

func (a *Archive) Observe(ctx context.Context, pathname string, classes []analyzer.Class) error {
	return nil
}

func (a *Archive) Finalize(ctx context.Context) ([]analyzer.Class, error) {
	return nil, nil
}

func (a *Archive) Close(ctx context.Context) error {
	return nil
}

type processor struct {
	class string
}

func (p *processor) Write(b []byte) (bool, error) {
	return false, nil
}

func (p *processor) Close() ([]analyzer.Class, error) {
	if p.class == "" {
		return nil, nil
	}
	return []analyzer.Class{{Name: p.class, Probability: 1}}, nil
}
