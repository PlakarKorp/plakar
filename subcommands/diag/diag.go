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
	subcommands.MustRegister(func() subcommands.Subcommand { return &DiagSnapshot{} }, subcommands.AgentSupport, "diag", "snapshot")
	subcommands.MustRegister(func() subcommands.Subcommand { return &DiagBlobSearch{} }, subcommands.AgentSupport, "diag", "blobsearch")
	subcommands.MustRegister(func() subcommands.Subcommand { return &DiagState{} }, subcommands.AgentSupport, "diag", "state")
	subcommands.MustRegister(func() subcommands.Subcommand { return &DiagPackfile{} }, subcommands.AgentSupport, "diag", "packfile")
	subcommands.MustRegister(func() subcommands.Subcommand { return &DiagObject{} }, subcommands.AgentSupport, "diag", "object")
	subcommands.MustRegister(func() subcommands.Subcommand { return &DiagVFS{} }, subcommands.AgentSupport, "diag", "vfs")
	subcommands.MustRegister(func() subcommands.Subcommand { return &DiagXattr{} }, subcommands.AgentSupport, "diag", "xattr")
	subcommands.MustRegister(func() subcommands.Subcommand { return &DiagContentType{} }, subcommands.AgentSupport, "diag", "contenttype")
	subcommands.MustRegister(func() subcommands.Subcommand { return &DiagLocks{} }, subcommands.AgentSupport, "diag", "locks")
	subcommands.MustRegister(func() subcommands.Subcommand { return &DiagSearch{} }, subcommands.AgentSupport, "diag", "search")
	subcommands.MustRegister(func() subcommands.Subcommand { return &DiagDirPack{} }, subcommands.AgentSupport, "diag", "dirpack")
	subcommands.MustRegister(func() subcommands.Subcommand { return &DiagBlob{} }, subcommands.AgentSupport, "diag", "blob")
	subcommands.MustRegister(func() subcommands.Subcommand { return &DiagRepository{} }, subcommands.AgentSupport, "diag")
}
