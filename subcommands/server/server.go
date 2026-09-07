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

package server

import (
	"net/http"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/server/httpd"
	"github.com/PlakarKorp/plakar/subcommands"
	"github.com/spf13/cobra"
)

func init() {
	subcommands.Register(func() subcommands.Subcommand { return &Server{} }, subcommands.BeforeRepositoryWithStorage, "server")
}

func (cmd *Server) CobraCommand() *cobra.Command {
	c := &cobra.Command{
		Use: "server [OPTIONS]",
	}
	c.Flags().StringVar(&cmd.ListenAddr, "listen", "localhost:9876", "address to listen on")
	c.Flags().StringVar(&cmd.Token, "token", "", "Bearer token")
	c.Flags().BoolVar(&cmd.allowDelete, "allow-delete", false, "enable delete operations")
	c.Flags().StringVar(&cmd.Cert, "cert", "", "Full certificate chain")
	c.Flags().StringVar(&cmd.Key, "key", "", "Certificate private key")
	return c
}

func (cmd *Server) Parse(ctx *appcontext.AppContext, args []string) error {
	if _, err := subcommands.ParseCobra(cmd, args); err != nil {
		return err
	}

	cmd.RepositorySecret = ctx.GetSecret()
	cmd.NoDelete = !cmd.allowDelete

	return nil
}

type Server struct {
	subcommands.SubcommandBase

	ListenAddr string
	NoDelete   bool
	Token      string
	Cert       string
	Key        string

	allowDelete bool
}

func (cmd *Server) Execute(ctx *appcontext.AppContext, repo *repository.Repository) (int, error) {
	var protocol string
	if cmd.Cert != "" && cmd.Key != "" {
		protocol = "https"
	} else {
		protocol = "http"
	}
	ctx.GetLogger().Info("listening on %s://%s", protocol, cmd.ListenAddr)
	mux := httpd.Mux(repo.Store(), cmd.NoDelete)

	server := &http.Server{
		Addr:    cmd.ListenAddr,
		Handler: httpd.Logging(httpd.Auth(cmd.Token, mux)),
	}
	go func() {
		<-ctx.Done()
		server.Shutdown(ctx)
	}()

	var err error
	if cmd.Cert != "" && cmd.Key != "" {
		err = server.ListenAndServeTLS(cmd.Cert, cmd.Key)
	} else {
		err = server.ListenAndServe()
	}

	if err != nil {
		return 1, err
	}
	return 0, nil
}
