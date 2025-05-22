/*
 * Copyright (c) 2023 Gilles Chehade <gilles@poolp.org>
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

package importer

import (
	"fmt"
	"io"
	"log"
	"path/filepath"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/location"
	"github.com/PlakarKorp/plakar/objects"
)

type ScanResult struct {
	Record *ScanRecord
	Error  *ScanError
}

type ExtendedAttributes struct {
	Name  string
	Value []byte
}

type ScanRecord struct {
	Pathname           string
	Target             string
	FileInfo           objects.FileInfo
	ExtendedAttributes []string
	FileAttributes     uint32
	IsXattr            bool
	XattrName          string
	XattrType          objects.Attribute
}

type ScanError struct {
	Pathname string
	Err      error
}

type Importer interface {
	Origin() string
	Type() string
	Root() string
	Scan() (<-chan *ScanResult, error)
	NewReader(string) (io.ReadCloser, error)
	NewExtendedAttributeReader(string, string) (io.ReadCloser, error)
	Close() error
}

type ImporterFn func(*appcontext.AppContext, string, map[string]string) (Importer, error)

var backends = location.New[ImporterFn]("fs")

func Register(name string, backend ImporterFn) {
	if !backends.Register(name, backend) {
		log.Fatalf("backend '%s' registered twice", name)
	}
}

func Backends() []string {
	return backends.Names()
}

func NewImporter(ctx *appcontext.AppContext, config map[string]string) (Importer, error) {
	location, ok := config["location"]
	if !ok {
		return nil, fmt.Errorf("missing location")
	}
	print("location: ", location)
	err := backends.LoadPlugin(ctx)
	if err == nil {
		return nil, fmt.Errorf("failed to load plugin: %w", err)
	}

	proto, location, backend, ok := backends.Lookup(location)
	if !ok {
		return nil, fmt.Errorf("unsupported importer protocol")
	}

	if proto == "fs" && !filepath.IsAbs(location) {
		location = filepath.Join(ctx.CWD, location)
		config["location"] = "fs://" + location
	} else {
		config["location"] = proto + "://" + location
	}
	return backend(ctx, proto, config)
}

func NewScanRecord(pathname, target string, fileinfo objects.FileInfo, xattr []string) *ScanResult {
	return &ScanResult{
		Record: &ScanRecord{
			Pathname:           pathname,
			Target:             target,
			FileInfo:           fileinfo,
			ExtendedAttributes: xattr,
		},
	}
}

func NewScanXattr(pathname, xattr string, kind objects.Attribute) *ScanResult {
	return &ScanResult{
		Record: &ScanRecord{
			Pathname:  pathname,
			IsXattr:   true,
			XattrName: xattr,
			XattrType: kind,
		},
	}
}

func NewScanError(pathname string, err error) *ScanResult {
	return &ScanResult{
		Error: &ScanError{
			Pathname: pathname,
			Err:      err,
		},
	}
}
