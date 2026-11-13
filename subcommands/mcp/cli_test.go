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
	"context"
	"encoding/hex"
	"sync"
	"testing"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// cliFixture is a repository with one snapshot, and the pieces runPlakar
// takes.
type cliFixture struct {
	repo   *repository.Repository
	ctx    *appcontext.AppContext
	root   *cobra.Command
	snapID string
}

func newCLIFixture(t *testing.T) *cliFixture {
	t.Helper()
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	t.Cleanup(func() { ctx.Close() })

	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/image.bin", 0644, "\xff\xfe\x00binary\x80"),
	})
	snapID := hex.EncodeToString(snap.Header.GetIndexShortID())
	snap.Close()

	return &cliFixture{repo: repo, ctx: ctx, root: commandTree(), snapID: snapID}
}

func (f *cliFixture) run(t *testing.T, cmd *Mcp, args ...string) runOutput {
	t.Helper()
	out, err := cmd.runPlakar(context.Background(), f.ctx, f.repo, f.root, args)
	require.NoError(t, err)
	return out
}

func TestRunPlakarLs(t *testing.T) {
	f := newCLIFixture(t)
	cmd := &Mcp{MaxOutput: 1 << 20}

	out := f.run(t, cmd, "ls")
	require.Equal(t, 0, out.ExitCode, out.Stderr)
	require.Contains(t, out.Stdout, f.snapID)
	require.False(t, out.Truncated)

	out = f.run(t, cmd, "ls", f.snapID+":/subdir")
	require.Equal(t, 0, out.ExitCode, out.Stderr)
	require.Contains(t, out.Stdout, "dummy.txt")
	require.Contains(t, out.Stdout, "image.bin")
}

func TestRunPlakarCat(t *testing.T) {
	f := newCLIFixture(t)
	cmd := &Mcp{MaxOutput: 1 << 20}

	out := f.run(t, cmd, "cat", f.snapID+":/subdir/dummy.txt")
	require.Equal(t, 0, out.ExitCode, out.Stderr)
	require.Equal(t, "hello dummy", out.Stdout)
	require.False(t, out.Binary)
}

// Output that is not text is left out and said to be, since the result has
// to survive JSON.
func TestRunPlakarBinaryOutput(t *testing.T) {
	f := newCLIFixture(t)
	cmd := &Mcp{MaxOutput: 1 << 20}

	out := f.run(t, cmd, "cat", f.snapID+":/subdir/image.bin")
	require.Equal(t, 0, out.ExitCode, out.Stderr)
	require.True(t, out.Binary)
	require.Empty(t, out.Stdout)
}

// Past the limit the output is cut, the command is stopped, and the result
// says so.
func TestRunPlakarTruncates(t *testing.T) {
	f := newCLIFixture(t)
	cmd := &Mcp{MaxOutput: 5}

	out := f.run(t, cmd, "cat", f.snapID+":/subdir/dummy.txt")
	require.True(t, out.Truncated)
	require.Equal(t, "hello", out.Stdout)
	require.Contains(t, out.Stderr, "output cut at 5 bytes")
}

// A command that fails reports it the way the command line does: a non-zero
// exit code and the reason on stderr.
func TestRunPlakarFailure(t *testing.T) {
	f := newCLIFixture(t)
	cmd := &Mcp{MaxOutput: 1 << 20}

	out := f.run(t, cmd, "cat", f.snapID+":/subdir/missing.txt")
	require.NotEqual(t, 0, out.ExitCode)
	require.Contains(t, out.Stderr, "missing.txt: no such file", "the reason reaches the caller, not the server log")
	require.Empty(t, out.Stdout)
}

func TestRunPlakarUsageError(t *testing.T) {
	f := newCLIFixture(t)
	cmd := &Mcp{MaxOutput: 1 << 20}

	out := f.run(t, cmd, "ls", "-bogus")
	require.Equal(t, 2, out.ExitCode)
	require.Contains(t, out.Stderr, "bogus")
	require.Contains(t, out.Stderr, "usage: plakar ls")
	require.Empty(t, out.Stdout)
}

// Anything but a listed read-only command is refused before it is parsed,
// let alone run.
func TestRunPlakarRefusesOtherCommands(t *testing.T) {
	f := newCLIFixture(t)
	cmd := &Mcp{MaxOutput: 1 << 20}

	// Registered in this test binary, so refused by name.
	for _, args := range [][]string{
		{"mcp"},
		{"diag", "chunkmap", f.snapID},
		{"diag", "dirpack", f.snapID},
	} {
		_, err := cmd.runPlakar(context.Background(), f.ctx, f.repo, f.root, args)
		require.Error(t, err, "%v", args)
		require.Contains(t, err.Error(), "not available", "%v", args)
	}

	// Not necessarily linked into this test binary: unknown here, refused by
	// name in the real one, and either way they do not run.
	for _, args := range [][]string{
		{"backup", t.TempDir()},
		{"rm", f.snapID},
		{"restore", f.snapID},
		{"nosuchcommand"},
	} {
		_, err := cmd.runPlakar(context.Background(), f.ctx, f.repo, f.root, args)
		require.Error(t, err, "%v", args)
	}

	_, err := cmd.runPlakar(context.Background(), f.ctx, f.repo, f.root, nil)
	require.Error(t, err)

	// Still one snapshot: nothing above touched the repository.
	out := f.run(t, cmd, "ls")
	require.Equal(t, 0, out.ExitCode, out.Stderr)
	require.Equal(t, 1, bytes.Count([]byte(out.Stdout), []byte(f.snapID)))
}

func TestRunPlakarDiag(t *testing.T) {
	f := newCLIFixture(t)
	cmd := &Mcp{MaxOutput: 1 << 20}

	out := f.run(t, cmd, "diag", "locks")
	require.Equal(t, 0, out.ExitCode, out.Stderr)

	out = f.run(t, cmd, "diag", "vfs", f.snapID+":/subdir/dummy.txt")
	require.Equal(t, 0, out.ExitCode, out.Stderr)
	require.Contains(t, out.Stdout, "dummy.txt")
}

func TestRunPlakarInfo(t *testing.T) {
	f := newCLIFixture(t)
	cmd := &Mcp{MaxOutput: 1 << 20}

	out := f.run(t, cmd, "info")
	require.Equal(t, 0, out.ExitCode, out.Stderr)
	require.NotEmpty(t, out.Stdout)

	out = f.run(t, cmd, "info", f.snapID)
	require.Equal(t, 0, out.ExitCode, out.Stderr)
	require.Contains(t, out.Stdout, f.snapID)
}

// help hands out the manual rather than rendering it for a terminal.
func TestRunPlakarHelp(t *testing.T) {
	f := newCLIFixture(t)
	cmd := &Mcp{MaxOutput: 1 << 20}

	out := f.run(t, cmd, "help", "ls")
	require.Equal(t, 0, out.ExitCode)
	require.Contains(t, out.Stdout, "plakar-ls")

	out = f.run(t, cmd, "help")
	require.Equal(t, 0, out.ExitCode)
	require.Contains(t, out.Stdout, "plakar")

	_, err := cmd.runPlakar(context.Background(), f.ctx, f.repo, f.root, []string{"help", "nosuchcommand"})
	require.Error(t, err)
}

// Calls do not share output: each runs in a context of its own.
func TestRunPlakarConcurrent(t *testing.T) {
	f := newCLIFixture(t)
	cmd := &Mcp{MaxOutput: 1 << 20}

	var wg sync.WaitGroup
	for i := range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if i%2 == 0 {
				out, err := cmd.runPlakar(context.Background(), f.ctx, f.repo, f.root, []string{"cat", f.snapID + ":/subdir/dummy.txt"})
				require.NoError(t, err)
				require.Equal(t, "hello dummy", out.Stdout)
			} else {
				out, err := cmd.runPlakar(context.Background(), f.ctx, f.repo, f.root, []string{"ls"})
				require.NoError(t, err)
				require.Contains(t, out.Stdout, f.snapID)
				require.NotContains(t, out.Stdout, "hello dummy")
			}
		}()
	}
	wg.Wait()
}

// Through the protocol: the tool is listed, runs, and a failed command comes
// back as a tool error that still carries its output.
func TestPlakarToolOverMCP(t *testing.T) {
	f := newCLIFixture(t)
	cmd := &Mcp{MaxOutput: 1 << 20}
	server := cmd.newServer(f.ctx, f.repo)

	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	require.NoError(t, err)
	defer serverSession.Close()

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v1"}, nil)
	session, err := client.Connect(context.Background(), clientTransport, nil)
	require.NoError(t, err)
	defer session.Close()

	tools, err := session.ListTools(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, tools.Tools, 1)
	require.Equal(t, "plakar", tools.Tools[0].Name)
	require.Contains(t, tools.Tools[0].Description, "- ls:")
	require.True(t, tools.Tools[0].Annotations.ReadOnlyHint)

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "plakar",
		Arguments: map[string]any{"args": []string{"cat", f.snapID + ":/subdir/dummy.txt"}},
	})
	require.NoError(t, err)
	require.False(t, res.IsError)
	structured, ok := res.StructuredContent.(map[string]any)
	require.True(t, ok)
	require.Equal(t, "hello dummy", structured["stdout"])

	res, err = session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "plakar",
		Arguments: map[string]any{"args": []string{"cat", f.snapID + ":/subdir/missing.txt"}},
	})
	require.NoError(t, err)
	require.True(t, res.IsError)
	structured, ok = res.StructuredContent.(map[string]any)
	require.True(t, ok)
	require.NotEqual(t, float64(0), structured["exit_code"])

	res, err = session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "plakar",
		Arguments: map[string]any{"args": []string{"rm", f.snapID}},
	})
	require.NoError(t, err)
	require.True(t, res.IsError)
}
