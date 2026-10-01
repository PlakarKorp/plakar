/*
 * Copyright (c) 2025 Matthieu Masson <matthieu.masson@plakar.io>
 * Copyright (c) 2025 Omar Polo <omar.polo@plakar.io>
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

package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/PlakarKorp/kloset/connectors"
	"github.com/PlakarKorp/kloset/location"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/pkg"
)

type pkgerImporter struct {
	cwd          string
	manifest     *pkg.Manifest
	manifestData []byte
}

type itemtype int

const (
	itexe itemtype = iota
	itjson
	itextra
)

func (imp *pkgerImporter) Origin() string        { return "" }
func (imp *pkgerImporter) Type() string          { return "pkger" }
func (imp *pkgerImporter) Root() string          { return "/" }
func (imp *pkgerImporter) Flags() location.Flags { return 0 }

func absolutify(cwd, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(cwd, path)
}

func (imp *pkgerImporter) dofile(p string, ch chan<- *connectors.Record, it itemtype) error {
	absolute := absolutify(imp.cwd, p)

	relative := absolute
	relative, _ = strings.CutPrefix(relative, imp.cwd)
	relative, _ = strings.CutPrefix(relative, string(os.PathSeparator))
	relative = filepath.ToSlash(relative)
	name := path.Join("/", relative)

	if !strings.HasPrefix(absolute, imp.cwd) {
		return fmt.Errorf("not below the manifest: %s", name)
	}

	fp, err := os.Open(absolute)
	if err != nil {
		return fmt.Errorf("Failed to open file: %w", err)
	}

	fi, err := fp.Stat()
	if err != nil {
		return fmt.Errorf("Failed to stat file: %w", err)
	}

	switch it {
	case itexe:
		var isexe bool
		if os.Getenv("GOOS") == "windows" || runtime.GOOS == "windows" {
			isexe = strings.HasSuffix(fi.Name(), ".exe")
		} else {
			isexe = (fi.Mode() & 0111) != 0
		}

		if !isexe {
			return fmt.Errorf("Not executable: %s", absolute)
		}

	case itjson:
		content := make(map[string]any)
		if err := json.NewDecoder(fp).Decode(&content); err != nil {
			return fmt.Errorf("invalid json: %s: %w", absolute, err)
		}
		if _, err := fp.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("seek failed: %w", err)
		}
	}

	ch <- &connectors.Record{
		Pathname: name,
		FileInfo: objects.FileInfoFromStat(fi),
		Reader:   fp,
	}

	return nil
}

func (imp *pkgerImporter) Import(ctx context.Context, records chan<- *connectors.Record, results <-chan *connectors.Result) error {
	defer close(records)

	records <- &connectors.Record{
		Pathname: "/manifest.yaml",
		FileInfo: objects.NewFileInfo("manifest.yaml", int64(len(imp.manifestData)), 0644, time.Now(), 0, 0, 0, 0, 1),
		Reader:   io.NopCloser(bytes.NewReader(imp.manifestData)),
	}
	for _, conn := range imp.manifest.Connectors {
		// Image connectors carry no executable; the binary lives in the image.
		if conn.Executable != "" {
			if err := imp.dofile(conn.Executable, records, itexe); err != nil {
				return err
			}
		}
		if conn.Validator != "" {
			if err := imp.dofile(conn.Validator, records, itjson); err != nil {
				return err
			}
		}
		for _, file := range conn.ExtraFiles {
			if err := imp.dofile(file, records, itextra); err != nil {
				return err
			}
		}
	}
	return nil
}

func (imp *pkgerImporter) Ping(ctx context.Context) error {
	return nil
}

func (imp *pkgerImporter) Close(ctx context.Context) error {
	return nil
}
