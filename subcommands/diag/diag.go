/*
 * Copyright (c) 2021 Gilles Chehade <gilles@poolp.org>
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

package diag

import (
	"github.com/PlakarKorp/plakar/subcommands"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &DiagSnapshot{} }, 0, "diag", "snapshot")
	subcommands.Register(func() subcommands.Subcommand { return &DiagBlobSearch{} }, 0, "diag", "blobsearch")
	subcommands.Register(func() subcommands.Subcommand { return &DiagState{} }, 0, "diag", "state")
	subcommands.Register(func() subcommands.Subcommand { return &DiagPackfile{} }, 0, "diag", "packfile")
	subcommands.Register(func() subcommands.Subcommand { return &DiagObject{} }, 0, "diag", "object")
	subcommands.Register(func() subcommands.Subcommand { return &DiagVFS{} }, 0, "diag", "vfs")
	subcommands.Register(func() subcommands.Subcommand { return &DiagXattr{} }, 0, "diag", "xattr")
	subcommands.Register(func() subcommands.Subcommand { return &DiagContentType{} }, 0, "diag", "contenttype")
	subcommands.Register(func() subcommands.Subcommand { return &DiagLocks{} }, 0, "diag", "locks")
	subcommands.Register(func() subcommands.Subcommand { return &DiagSearch{} }, 0, "diag", "search")
	subcommands.Register(func() subcommands.Subcommand { return &DiagDirPack{} }, 0, "diag", "dirpack")
	subcommands.Register(func() subcommands.Subcommand { return &DiagBlob{} }, 0, "diag", "blob")
	subcommands.Register(func() subcommands.Subcommand { return &DiagChunks{} }, 0, "diag", "chunks")
	subcommands.Register(func() subcommands.Subcommand { return &DiagChunkmap{} }, 0, "diag", "chunkmap")
	subcommands.Register(func() subcommands.Subcommand { return &DiagRepository{} }, 0, "diag")
}
