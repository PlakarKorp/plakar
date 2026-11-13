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

package mcp

import (
	"bytes"
	"os"
	"testing"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/repository"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// countSnapshots says how many snapshots the repository lists.
func countSnapshots(t *testing.T, repo *repository.Repository) int {
	t.Helper()
	ids, err := locate.LocateSnapshotIDs(repo, locate.NewDefaultLocateOptions())
	require.NoError(t, err)
	return len(ids)
}

func init() {
	os.Setenv("TZ", "UTC")
}

// mockFiles is the small tree the write and resource tests snapshot.
func mockFiles(content string) []ptesting.MockFile {
	return []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, content),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
	}
}

func TestMcpParseDefaults(t *testing.T) {
	_, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	cmd := &Mcp{}
	require.NoError(t, cmd.Parse(ctx, []string{}))
	require.Equal(t, int64(1<<20), cmd.MaxOutput)
}

func TestMcpParseMaxOutput(t *testing.T) {
	_, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	cmd := &Mcp{}
	require.NoError(t, cmd.Parse(ctx, []string{"-max-output", "4096"}))
	require.Equal(t, int64(4096), cmd.MaxOutput)
}

func TestMcpParseRejectsBadInput(t *testing.T) {
	_, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	cmd := &Mcp{}
	require.Error(t, cmd.Parse(ctx, []string{"extra"}))

	cmd = &Mcp{}
	require.Error(t, cmd.Parse(ctx, []string{"-max-output", "0"}))
}

// TestParseRedirectsStdout guards the stdio contract: stdout carries the
// JSON-RPC stream, so nothing else may write to it.
func TestParseRedirectsStdout(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	_, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	defer ctx.Close()

	cmd := &Mcp{}
	require.NoError(t, cmd.Parse(ctx, []string{}))

	ctx.GetLogger().Stdout("progress line")
	require.Empty(t, bufOut.String(), "stdout must stay clean for the JSON-RPC stream")
	require.Contains(t, bufErr.String(), "progress line")
}

func TestSnapshotPath(t *testing.T) {
	require.Equal(t, "abc:/", snapshotPath("abc", ""))
	require.Equal(t, "abc:/etc", snapshotPath("abc", "/etc"))
	require.Equal(t, "abc:/etc", snapshotPath("abc", "etc"))
}
