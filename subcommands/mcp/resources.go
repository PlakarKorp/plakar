/*
 * Copyright (c) 2025 Gilles Chehade <gilles@poolp.org>
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

package mcp

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerResources exposes snapshot contents as MCP resources, so a client can
// attach a backed-up file directly instead of routing it through a tool call.
// Resources are reads, so they are always available.
func (cmd *Mcp) registerResources(repo *repository.Repository, server *mcp.Server) {
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "snapshot_file",
		Title:       "File in a snapshot",
		Description: "The contents of a file stored in a snapshot, addressed as plakar://SNAPSHOT/PATH with a snapshot ID from list_snapshots.",
		URITemplate: "plakar://{snapshot}/{+path}",
	}, func(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		uri := req.Params.URI

		rest, ok := strings.CutPrefix(uri, "plakar://")
		if !ok {
			return nil, mcp.ResourceNotFoundError(uri)
		}
		snapshotID, pathname, ok := strings.Cut(rest, "/")
		if !ok || snapshotID == "" || pathname == "" {
			return nil, mcp.ResourceNotFoundError(uri)
		}
		// A client that encodes per the advertised RFC 6570 template escapes
		// reserved characters, so undo that before treating it as a path.
		decoded, err := url.PathUnescape(pathname)
		if err != nil {
			return nil, mcp.ResourceNotFoundError(uri)
		}
		pathname = decoded

		data, contentType, err := readResource(repo, snapshotID, "/"+pathname, cmd.MaxFileSize)
		if err != nil {
			return nil, err
		}

		contents := &mcp.ResourceContents{URI: uri, MIMEType: contentType}
		// Blob survives JSON encoding for content that text cannot carry.
		if utf8.Valid(data) {
			contents.Text = string(data)
		} else {
			contents.Blob = data
		}

		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{contents}}, nil
	})
}

// readResource returns the full contents of a file in a snapshot. Unlike
// read_file there is no way to page, so a file past the size limit is refused
// rather than silently cut.
func readResource(repo *repository.Repository, snapshotID, pathname string, maxFileSize int64) ([]byte, string, error) {
	snap, resolved, err := locate.OpenSnapshotByPath(repo, snapshotPath(snapshotID, pathname))
	if err != nil {
		return nil, "", err
	}
	defer snap.Close()

	pvfs, err := snap.Filesystem()
	if err != nil {
		return nil, "", err
	}

	entry, err := pvfs.GetEntry(resolved)
	if err != nil {
		return nil, "", fmt.Errorf("%s: no such file", pathname)
	}
	if !entry.Stat().Mode().IsRegular() {
		return nil, "", fmt.Errorf("%s: not a regular file", pathname)
	}
	if entry.Stat().Size() > maxFileSize {
		return nil, "", fmt.Errorf("%s: larger than max-file-size, read it with the read_file tool instead", pathname)
	}

	file, err := entry.Open(pvfs)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxFileSize))
	if err != nil {
		return nil, "", err
	}

	contentType := ""
	if entry.ResolvedObject != nil {
		contentType = entry.ResolvedObject.ContentType
	}

	return data, contentType, nil
}
