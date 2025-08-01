/*
 * Copyright (c) 2025 Gilles Chehade <gilles@plakar.io>
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

package stdio

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/snapshot/exporter"
)

type StdioExporter struct {
	filePath string
	appCtx   context.Context
	w        io.Writer
}

func init() {
	exporter.Register("stdout", 0, NewStdioExporter)
	exporter.Register("stderr", 0, NewStdioExporter)
}

func NewStdioExporter(appCtx context.Context, opts *exporter.Options, name string, config map[string]string) (exporter.Exporter, error) {
	var w io.Writer

	switch name {
	case "stdout":
		w = opts.Stdout
	case "stderr":
		w = opts.Stderr
	default:
		return nil, fmt.Errorf("unknown stdio backend %s", name)
	}

	return &StdioExporter{
		filePath: strings.TrimPrefix(config["location"], name+"://"),
		appCtx:   appCtx,
		w:        w,
	}, nil
}

func (p *StdioExporter) Root(ctx context.Context) (string, error) {
	return "/", nil
}

func (p *StdioExporter) CreateDirectory(ctx context.Context, pathname string) error {
	// can't mkdir on Stdio
	return nil
}

func (p *StdioExporter) StoreFile(ctx context.Context, pathname string, fp io.Reader, size int64) error {
	_, err := io.Copy(p.w, fp)
	return err
}

func (p *StdioExporter) SetPermissions(ctx context.Context, pathname string, fileinfo *objects.FileInfo) error {
	// can't chown/chmod on Stdio
	return nil
}

func (p *StdioExporter) Close(ctx context.Context) error {
	return nil
}
