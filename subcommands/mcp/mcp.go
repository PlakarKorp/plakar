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
	"fmt"

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

	MaxOutput int64
}

func (cmd *Mcp) CobraCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "mcp [-max-output SIZE]",
		Short: "serve the repository over the Model Context Protocol",
	}
	c.Flags().Int64Var(&cmd.MaxOutput, "max-output", 1<<20, "maximum size of the output of a command, and of a file served as a resource")
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

	if cmd.MaxOutput <= 0 {
		return fmt.Errorf("max-output must be greater than zero")
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
	if err := cmd.newServer(ctx, repo).Run(ctx, &mcp.StdioTransport{}); err != nil {
		return 1, err
	}

	return 0, nil
}

// newServer builds the MCP server and registers its tools and resources.
func (cmd *Mcp) newServer(ctx *appcontext.AppContext, repo *repository.Repository) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:        "plakar",
		Title:       "Plakar",
		Description: "Read-only access to a Plakar repository through its command line: list snapshots, browse their contents, read and compare files.",
		Version:     utils.VERSION,
	}, nil)

	cmd.registerTools(ctx, repo, server)
	cmd.registerResources(repo, server)

	return server
}
