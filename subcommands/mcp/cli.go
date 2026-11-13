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
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/PlakarKorp/kloset/logging"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/subcommands/help"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	// A command is only registered once its package is linked in, which main
	// does for all of them; the tool links the ones it runs so that they are
	// there wherever the server is built.
	_ "github.com/PlakarKorp/plakar/subcommands/cat"
	_ "github.com/PlakarKorp/plakar/subcommands/check"
	_ "github.com/PlakarKorp/plakar/subcommands/diag"
	_ "github.com/PlakarKorp/plakar/subcommands/diff"
	_ "github.com/PlakarKorp/plakar/subcommands/digest"
	_ "github.com/PlakarKorp/plakar/subcommands/info"
	_ "github.com/PlakarKorp/plakar/subcommands/locate"
	_ "github.com/PlakarKorp/plakar/subcommands/ls"
)

// The read side of the server is the command line itself: the plakar tool
// runs a plakar command in this process, against the served repository, and
// returns what it printed. There is no second implementation of ls or cat to
// keep in step with the first, so a change to a command reaches an MCP
// client the day it lands.

// readCommand is a plakar command the plakar tool may run: one that only
// reads the repository and writes nothing but its output.
type readCommand struct {
	// path is the command as registered, "diag locks" for plakar diag locks.
	path string
	// use says what the command does and shows a typical invocation, for the
	// model reading the tool description.
	use string
}

// readCommands is everything the plakar tool runs. A registered command absent
// from here is refused by name before its arguments are even parsed, so the
// write commands cannot be reached however they are spelled. Also left out:
// "diag chunkmap", which writes a file, and "diag dirpack", which prints to
// the process stdout, where stdio carries the protocol.
var readCommands = []readCommand{
	{"ls", "list the snapshots, or the entries of a directory in a snapshot: ls, ls -tags, ls -recursive SNAPSHOT:/path"},
	{"cat", "print the contents of files in a snapshot: cat SNAPSHOT:/path"},
	{"info", "describe the repository, or a snapshot, or with -errors the entries a snapshot could not read: info, info SNAPSHOT, info -errors SNAPSHOT"},
	{"locate", "find the snapshots holding files whose name matches a pattern: locate '*.pdf', locate -snapshot SNAPSHOT report.txt"},
	{"diff", "compare a file or directory between two snapshots: diff SNAPSHOT:/path SNAPSHOT, diff -recursive SNAPSHOT:/dir SNAPSHOT"},
	{"digest", "compute the digest of files in a snapshot: digest SNAPSHOT:/path"},
	{"check", "verify the integrity of snapshots, -fast skips re-hashing the data: check -fast SNAPSHOT"},
	{"diag", "describe the repository internals: diag"},
	{"diag snapshot", "dump the header of a snapshot: diag snapshot SNAPSHOT"},
	{"diag locks", "list the locks held on the repository: diag locks"},
	{"diag vfs", "dump the metadata of an entry, ownership, mode, content type and xattrs included: diag vfs SNAPSHOT:/path"},
	{"diag xattr", "dump the extended attributes of an entry: diag xattr SNAPSHOT:/path"},
	{"diag contenttype", "list the entries of a snapshot by content type: diag contenttype SNAPSHOT:/path"},
	{"diag search", "search the entries of a snapshot: diag search SNAPSHOT:/path PATTERN"},
	{"diag chunks", "list the chunks of a file: diag chunks SNAPSHOT:/path"},
	{"diag object", "dump an object by its MAC: diag object MAC"},
	{"diag blob", "dump a blob by type and MAC: diag blob TYPE MAC"},
	{"diag packfile", "dump a packfile by its MAC: diag packfile MAC"},
	{"diag state", "dump the state of the repository: diag state"},
	{"help", "show the manual of a command: help ls, help diag"},
}

// runInput is the command line of a plakar call.
type runInput struct {
	Args []string `json:"args" jsonschema:"the words after plakar, each its own string: the command, then its options and arguments, as in [\"ls\", \"-recursive\", \"1a2b3c4d:/etc\"]"`
}

// runOutput is what the command printed and how it ended.
type runOutput struct {
	Stdout    string `json:"stdout" jsonschema:"what the command printed on its standard output"`
	Stderr    string `json:"stderr,omitempty" jsonschema:"what the command printed on its standard error, errors included"`
	ExitCode  int    `json:"exit_code" jsonschema:"0 when the command succeeded, 2 on a usage error, 1 otherwise"`
	Truncated bool   `json:"truncated,omitempty" jsonschema:"true when the output passed the size limit and was cut; narrow the command rather than repeat it"`
	Binary    bool   `json:"binary,omitempty" jsonschema:"true when the output was not text and was left out"`
}

// errOutputLimit stops a command that writes past the limit, so a cat of a
// large file does not read it to the end for nothing.
var errOutputLimit = errors.New("output limit reached")

// maxStderr bounds what a command may say on its standard error.
const maxStderr = 64 << 10

// cappedBuffer keeps the first limit bytes written to it and refuses the rest.
type cappedBuffer struct {
	buf       bytes.Buffer
	limit     int64
	truncated bool
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if b.truncated {
		return 0, errOutputLimit
	}
	room := b.limit - int64(b.buf.Len())
	if int64(len(p)) > room {
		b.buf.Write(p[:room])
		b.truncated = true
		return int(room), errOutputLimit
	}
	return b.buf.Write(p)
}

// commandTree is the registered commands as cobra resolves them, built once
// per server: Resolve picks the longest match, so "diag locks" is found as
// itself and not as "diag" with an argument.
func commandTree() *cobra.Command {
	root := &cobra.Command{Use: "plakar"}
	subcommands.Tree(root)
	return root
}

// toolDescription tells the model what the plakar tool runs and how.
func (cmd *Mcp) toolDescription() string {
	var b strings.Builder
	b.WriteString("Run a read-only plakar command against the repository and return what it prints, exactly as the command line would. ")
	b.WriteString("Pass the words that follow \"plakar\" as separate args. ")
	b.WriteString("A snapshot is named by its ID or by a unique prefix of it, followed by :/path for an entry inside it. ")
	fmt.Fprintf(&b, "Output past %d bytes is cut and flagged as truncated: narrow the command instead of repeating it. ", cmd.MaxOutput)
	b.WriteString("Run help COMMAND for the full options of a command.\nCommands:")
	for _, c := range readCommands {
		fmt.Fprintf(&b, "\n- %s: %s", c.path, c.use)
	}
	return b.String()
}

// registerCLI adds the plakar tool.
func (cmd *Mcp) registerCLI(ctx *appcontext.AppContext, repo *repository.Repository, server *mcp.Server) {
	root := commandTree()
	mcp.AddTool(server, &mcp.Tool{
		Name:        "plakar",
		Description: cmd.toolDescription(),
		Annotations: &mcp.ToolAnnotations{
			Title:        "Run a plakar command",
			ReadOnlyHint: true,
		},
	}, func(callCtx context.Context, _ *mcp.CallToolRequest, in runInput) (*mcp.CallToolResult, runOutput, error) {
		out, err := cmd.runPlakar(callCtx, ctx, repo, root, in.Args)
		if err != nil {
			return nil, runOutput{}, err
		}
		// A failed command is a tool error that still carries its output:
		// the model reads why from stderr instead of a bare failure.
		return &mcp.CallToolResult{IsError: out.ExitCode != 0}, out, nil
	})
}

// runPlakar resolves args to a registered command, checks it is one the tool
// may run, and executes it in a context of its own whose output is captured.
// An error means the command never ran; a command that ran and failed is
// reported through the exit code and stderr, as on the command line.
func (cmd *Mcp) runPlakar(callCtx context.Context, ctx *appcontext.AppContext, repo *repository.Repository, root *cobra.Command, args []string) (runOutput, error) {
	if len(args) == 0 {
		return runOutput{}, fmt.Errorf("args is required: the plakar command to run, such as [\"ls\"]")
	}

	sub, path, rest := subcommands.Resolve(root, args)
	if sub == nil {
		return runOutput{}, fmt.Errorf("unknown command %q; the commands this tool runs are: %s", args[0], readCommandNames())
	}
	name := strings.Join(path, " ")
	if !isReadCommand(name) {
		return runOutput{}, fmt.Errorf("plakar %s is not available through this tool, which only runs read-only commands: %s", name, readCommandNames())
	}

	// help renders to the process stdout for a terminal; the model wants the
	// manual itself.
	if name == "help" {
		doc, err := help.Document(strings.Join(rest, "-"))
		if err != nil {
			return runOutput{}, fmt.Errorf("no manual for %q; try help on its own for the list of commands", strings.Join(rest, " "))
		}
		return runOutput{Stdout: string(doc)}, nil
	}

	// The command runs in a context of its own: its output goes to buffers,
	// its events to a bus nobody listens to, and it stops when the client
	// gives up on the call or the server shuts down.
	child := appcontext.NewAppContextFrom(ctx)
	defer child.Close()
	child.Context, child.Cancel = context.WithCancelCause(callCtx)
	stop := context.AfterFunc(ctx, func() { child.Cancel(ctx.Err()) })
	defer stop()
	child.Config = ctx.Config
	child.StoreConfig = ctx.StoreConfig
	child.Quiet = true
	child.SetSecret(ctx.GetSecret())

	// stderr carries diagnostics, which stay short; its own cap only guards
	// against a command that chatters, and is kept apart from the output
	// limit so that the notes added below always fit.
	stdout := &cappedBuffer{limit: cmd.MaxOutput}
	stderr := &cappedBuffer{limit: maxStderr}
	child.Stdout = stdout
	child.Stderr = stderr
	// Commands report their errors through the logger, so it has to write
	// into the captured stderr too, or the reason would land in the server
	// log while the model sees only an exit code.
	child.SetLogger(logging.NewLogger(stdout, stderr))

	// notes is what the command line itself would print after the command,
	// appended to stderr once the command is done.
	var notes strings.Builder

	out := runOutput{}
	if err := sub.Parse(child, rest); err != nil {
		var usage subcommands.ErrUsage
		if errors.As(err, &usage) {
			out.ExitCode = 2
		} else {
			out.ExitCode = 1
		}
		fmt.Fprintf(&notes, "plakar: %s\nusage: plakar %s\n", err, sub.CobraCommand().Use)
		out.Stderr = stderr.buf.String() + notes.String()
		return out, nil
	}

	status, err := sub.Execute(child, repo)
	if err != nil {
		fmt.Fprintf(&notes, "plakar: %s\n", child.ErrorCause(err))
		if status == 0 {
			status = 1
		}
	}
	out.ExitCode = status

	if stdout.truncated {
		out.Truncated = true
		fmt.Fprintf(&notes, "plakar: output cut at %d bytes\n", cmd.MaxOutput)
	}
	if utf8.Valid(stdout.buf.Bytes()) {
		out.Stdout = stdout.buf.String()
	} else {
		out.Binary = true
	}
	out.Stderr = strings.ToValidUTF8(stderr.buf.String()+notes.String(), "\uFFFD")

	return out, nil
}

func isReadCommand(name string) bool {
	for _, c := range readCommands {
		if c.path == name {
			return true
		}
	}
	return false
}

func readCommandNames() string {
	names := make([]string, 0, len(readCommands))
	for _, c := range readCommands {
		names = append(names, c.path)
	}
	return strings.Join(names, ", ")
}
