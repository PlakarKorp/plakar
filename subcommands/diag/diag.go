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
	subcommands.Register(Snapshot, 0, "diag", "snapshot")
	subcommands.Register(BlobSearch, 0, "diag", "blobsearch")
	subcommands.Register(State, 0, "diag", "state")
	subcommands.Register(Packfile, 0, "diag", "packfile")
	subcommands.Register(Object, 0, "diag", "object")
	subcommands.Register(VFS, 0, "diag", "vfs")
	subcommands.Register(Xattr, 0, "diag", "xattr")
	subcommands.Register(ContentType, 0, "diag", "contenttype")
	subcommands.Register(Locks, 0, "diag", "locks")
	subcommands.Register(Search, 0, "diag", "search")
	subcommands.Register(DirPack, 0, "diag", "dirpack")
	subcommands.Register(Blob, 0, "diag", "blob")
	subcommands.Register(Chunks, 0, "diag", "chunks")
	subcommands.Register(Repository, 0, "diag")
}
