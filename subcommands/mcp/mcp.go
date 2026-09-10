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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Mcp{} }, 0, "mcp")
}

type Mcp struct {
	subcommands.SubcommandBase

	MaxFileSize  int64
	AllowBackup  bool
	AllowDelete  bool
	AllowRestore bool
	AllowSync    bool
	RestoreRoot  string
}

func (cmd *Mcp) CobraCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "mcp [-max-file-size SIZE] [-allow-backup] [-allow-delete] [-allow-sync] [-allow-restore -restore-root DIR]",
		Short: "serve the repository over the Model Context Protocol",
	}
	c.Flags().Int64Var(&cmd.MaxFileSize, "max-file-size", 1<<20, "maximum size of a file returned by read_file")
	c.Flags().BoolVar(&cmd.AllowBackup, "allow-backup", false, "enable the backup tool, allowing clients to create snapshots")
	c.Flags().BoolVar(&cmd.AllowDelete, "allow-delete", false, "enable the remove and prune tools, allowing clients to delete snapshots permanently")
	c.Flags().BoolVar(&cmd.AllowRestore, "allow-restore", false, "enable the restore tool, allowing clients to write snapshot contents under -restore-root")
	c.Flags().StringVar(&cmd.RestoreRoot, "restore-root", "", "directory restores are confined to, required with -allow-restore")
	c.Flags().BoolVar(&cmd.AllowSync, "allow-sync", false, "enable the sync tool, allowing clients to push snapshots to a peer repository from the configuration")
	return c
}

func (cmd *Mcp) Parse(ctx *appcontext.AppContext, args []string) error {
	rest, err := subcommands.ParseCobra(cmd, args)
	if err != nil {
		return err
	}

	if len(rest) != 0 {
		return fmt.Errorf("too many arguments")
	}

	if cmd.MaxFileSize <= 0 {
		return fmt.Errorf("max-file-size must be greater than zero")
	}

	// A restore writes to the local filesystem on behalf of a model, so it is
	// never allowed to pick where: the operator names the one directory it may
	// touch, and it has to exist up front.
	if cmd.AllowRestore {
		if cmd.RestoreRoot == "" {
			return fmt.Errorf("allow-restore requires -restore-root")
		}
		root, err := filepath.Abs(cmd.RestoreRoot)
		if err != nil {
			return err
		}
		st, err := os.Stat(root)
		if err != nil {
			return fmt.Errorf("restore-root: %w", err)
		}
		if !st.IsDir() {
			return fmt.Errorf("restore-root: %s: not a directory", cmd.RestoreRoot)
		}
		cmd.RestoreRoot = root
	} else if cmd.RestoreRoot != "" {
		return fmt.Errorf("restore-root requires -allow-restore")
	}

	// stdio is the transport MCP clients use when they spawn the server as a
	// child process, and stdout then carries nothing but the JSON-RPC stream.
	// Plenty of things write to stdout on the way to serving though -- the
	// state rebuild progress lines, the check progress lines -- and a single
	// stray byte makes the stream unparseable, so everything meant for a human
	// goes to stderr from here on. This has to happen in Parse rather than
	// Execute: the repository is opened, and possibly rebuilt, in between.
	ctx.Stdout = ctx.Stderr
	ctx.GetLogger().SetOutput(ctx.Stderr)

	cmd.RepositorySecret = ctx.GetSecret()

	return nil
}

func (cmd *Mcp) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	description := "Read-only access to a Plakar repository: list snapshots, browse their contents and read files."
	enabled := make([]string, 0, 4)
	if cmd.AllowBackup {
		enabled = append(enabled, "creating snapshots")
	}
	if cmd.AllowDelete {
		enabled = append(enabled, "removing snapshots")
	}
	if cmd.AllowRestore {
		enabled = append(enabled, "restoring files")
	}
	if cmd.AllowSync {
		enabled = append(enabled, "syncing to a peer repository")
	}
	if len(enabled) > 0 {
		description += " Also enabled: " + strings.Join(enabled, ", ") + "."
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:        "plakar",
		Title:       "Plakar",
		Description: description,
		Version:     utils.VERSION,
	}, nil)

	cmd.registerTools(ctx, repo, server)
	cmd.registerResources(repo, server)

	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		return 1, err
	}

	return 0, nil
}
