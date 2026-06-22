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
	"flag"
	"fmt"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/server/httpd"
	"github.com/PlakarKorp/plakar/subcommands"
)

func init() {
	subcommands.Register(Server, subcommands.BeforeRepositoryWithStorage, "server")
}

func Server(ctx *appcontext.AppContext, repo *repository.Repository, args []string) error {
	var (
		listenAddr  string
		allowDelete bool
		cert        string
		key         string
	)

	flags := flag.NewFlagSet("server", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Fprintf(flags.Output(), "Usage: %s [OPTIONS]\n", flags.Name())
		fmt.Fprintf(flags.Output(), "\nOPTIONS:\n")
		flags.PrintDefaults()
	}

	flags.StringVar(&listenAddr, "listen", "localhost:9876", "address to listen on")
	flags.BoolVar(&allowDelete, "allow-delete", false, "enable delete operations")
	flags.StringVar(&cert, "cert", "", "Full certificate chain")
	flags.StringVar(&key, "key", "", "Certificate private key")

	flags.Parse(args)

	var protocol string
	if cert != "" && key != "" {
		protocol = "https"
	} else {
		protocol = "http"
	}
	ctx.GetLogger().Info("listening on %s://%s", protocol, listenAddr)
	return httpd.Server(ctx, repo, listenAddr, !allowDelete, cert, key)
}
