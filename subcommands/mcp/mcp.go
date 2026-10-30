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
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/server/httpd"
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
	ListenAddr   string
	Token        string
	Cert         string
	Key          string
}

func (cmd *Mcp) CobraCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "mcp [-listen ADDR [-token TOKEN] [-cert FILE -key FILE]] [-max-file-size SIZE] [-allow-backup] [-allow-delete] [-allow-sync] [-allow-restore -restore-root DIR]",
		Short: "serve the repository over the Model Context Protocol",
	}
	c.Flags().Int64Var(&cmd.MaxFileSize, "max-file-size", 1<<20, "maximum size of a file returned by read_file")
	c.Flags().BoolVar(&cmd.AllowBackup, "allow-backup", false, "enable the backup tool, allowing clients to create snapshots")
	c.Flags().BoolVar(&cmd.AllowDelete, "allow-delete", false, "enable the remove and prune tools, allowing clients to delete snapshots permanently")
	c.Flags().BoolVar(&cmd.AllowRestore, "allow-restore", false, "enable the restore tool, allowing clients to write snapshot contents under -restore-root")
	c.Flags().StringVar(&cmd.RestoreRoot, "restore-root", "", "directory restores are confined to, required with -allow-restore")
	c.Flags().BoolVar(&cmd.AllowSync, "allow-sync", false, "enable the sync tool, allowing clients to push snapshots to a peer repository from the configuration")
	c.Flags().StringVar(&cmd.ListenAddr, "listen", "", "address to serve MCP over HTTP on, e.g. localhost:9877; stdio is used when unset")
	c.Flags().StringVar(&cmd.Token, "token", "", "bearer token required from HTTP clients, required with -listen unless the address is loopback")
	c.Flags().StringVar(&cmd.Cert, "cert", "", "full certificate chain, serves HTTPS with -key")
	c.Flags().StringVar(&cmd.Key, "key", "", "certificate private key, serves HTTPS with -cert")
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
			return fmt.Errorf("restore-root: %w", err)
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

	if (cmd.Cert == "") != (cmd.Key == "") {
		return fmt.Errorf("cert and key must be given together")
	}

	if cmd.ListenAddr == "" {
		if cmd.Token != "" {
			return fmt.Errorf("token requires -listen")
		}
		if cmd.Cert != "" {
			return fmt.Errorf("cert and key require -listen")
		}
	} else {
		if err := cmd.parseListenAddr(); err != nil {
			return err
		}
	}

	// stdio is the transport MCP clients use when they spawn the server as a
	// child process, and stdout then carries nothing but the JSON-RPC stream.
	// Plenty of things write to stdout on the way to serving though -- the
	// state rebuild progress lines, the check progress lines -- and a single
	// stray byte makes the stream unparseable, so everything meant for a human
	// goes to stderr from here on. Harmless over HTTP, where stdout carries no
	// protocol, so it is not worth making conditional. This has to happen in
	// Parse rather than Execute: the repository is opened, and possibly
	// rebuilt, in between.
	ctx.Stdout = ctx.Stderr
	ctx.GetLogger().SetOutput(ctx.Stderr)

	cmd.RepositorySecret = ctx.GetSecret()

	return nil
}

// shutdownGrace bounds how long a drain waits for in-flight tool calls before
// open sessions are dropped.
const shutdownGrace = 5 * time.Second

// defaultListenAddr is the address suggested when -listen is given something
// that cannot be bound. Loopback, because the tools reach the repository.
const defaultListenAddr = "localhost:9877"

func (cmd *Mcp) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	server := cmd.newServer(ctx, repo)

	if cmd.ListenAddr != "" {
		return cmd.serveHTTP(ctx, server)
	}

	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		return 1, err
	}

	return 0, nil
}

// newServer builds the MCP server and registers everything the flags enable.
// Both transports serve the same one: the tools do not know which is in use.
func (cmd *Mcp) newServer(ctx *appcontext.AppContext, repo *repository.Repository) *mcp.Server {
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

	return server
}

// serveHTTP serves the MCP streamable HTTP transport until the context is
// cancelled.
func (cmd *Mcp) serveHTTP(ctx *appcontext.AppContext, server *mcp.Server) (int, error) {
	handler := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return server },
		nil,
	)

	httpServer := &http.Server{
		Handler: httpd.Auth(cmd.Token, handler),
	}

	// Bind before announcing, so a port already in use reports the failure
	// rather than a listening line the server never honours.
	listener, err := net.Listen("tcp", cmd.ListenAddr)
	if err != nil {
		return 1, fmt.Errorf("listen: %w", err)
	}

	scheme := "http"
	if cmd.Cert != "" {
		scheme = "https"
	}
	ctx.GetLogger().Info("listening on %s://%s", scheme, listener.Addr())

	// Shutdown lets in-flight tool calls drain; a restore or a check can be
	// well into its work when the operator interrupts. It will not return on
	// its own here: a connected client keeps an SSE stream open for as long as
	// the session lasts, and Shutdown waits on it. So give the drain a deadline
	// and then Close what is left, or an operator interrupting a server with a
	// client attached would wait forever.
	done := make(chan struct{})
	go func() {
		defer close(done)
		<-ctx.Done()

		stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownGrace)
		defer cancel()
		if err := httpServer.Shutdown(stopCtx); err != nil {
			httpServer.Close()
		}
	}()

	if cmd.Cert != "" {
		err = httpServer.ServeTLS(listener, cmd.Cert, cmd.Key)
	} else {
		err = httpServer.Serve(listener)
	}
	<-done

	// Serve always reports why it stopped, and a requested shutdown is not a
	// failure.
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return 1, err
	}

	return 0, nil
}

// parseListenAddr validates -listen and decides whether a token is mandatory.
//
// The flag takes a bind address, not a URL, but "-listen https://host/path" is
// the natural thing to try: say so plainly instead of failing later in
// net.Listen with a message about a missing port.
func (cmd *Mcp) parseListenAddr() error {
	if strings.Contains(cmd.ListenAddr, "://") {
		return fmt.Errorf("listen takes an address to bind, not a URL: try -listen %s",
			defaultListenAddr)
	}

	host, _, err := net.SplitHostPort(cmd.ListenAddr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	// An MCP session can read the repository, and with the write gates enabled
	// it can delete snapshots or restore over files. On loopback the reachable
	// set is whoever is already on the machine, which is the same set that
	// could run plakar directly, so a token stays optional there. Any wider
	// bind puts those tools on the network, and going unauthenticated has to be
	// spelled out rather than fallen into by leaving a flag off.
	if cmd.Token == "" && !isLoopbackHost(host) {
		return fmt.Errorf("listen on %s exposes the repository beyond this host: "+
			"set -token, or bind a loopback address such as %s",
			cmd.ListenAddr, defaultListenAddr)
	}

	return nil
}

// isLoopbackHost reports whether binding host only accepts connections from
// this machine. An unspecified address (an empty host, 0.0.0.0, ::) accepts
// from anywhere, and a name is not resolved here: neither counts as loopback.
func isLoopbackHost(host string) bool {
	if host == "" {
		return false
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return strings.EqualFold(host, "localhost")
	}
	return ip.IsLoopback()
}
