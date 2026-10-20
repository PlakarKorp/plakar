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

package web

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/plakar/api"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/utils"
)

type UiOptions struct {
	MaxConcurrency uint64
	NoSpawn        bool
	Cors           bool
	Token          string
	Cert           string
	Key            string
	NoRefresh      bool
}

// Short: browsers hold unused connections, counted as in flight for 5s.
const shutdownTimeout = time.Second

//go:embed all:build/*
var content embed.FS

// Only release tags carry the built UI: on main, build/ holds nothing but a
// .gitkeep, and this page explains how to get a binary with the UI.
//
//go:embed placeholder.html
var placeholder []byte

func Ui(repo *repository.Repository, ctx *appcontext.AppContext, addr string, opts *UiOptions) error {
	server := http.NewServeMux()
	api.SetupRoutes(server, repo, ctx, opts.Token, opts.NoRefresh)

	statics, err := staticHandler(content)
	if err != nil {
		return err
	}
	server.Handle("/{path...}", statics)

	if addr == "" {
		var port uint16
		for {
			port = uint16(rand.Uint32() % 0xffff)
			if port >= 1024 {
				break
			}
		}
		addr = fmt.Sprintf("localhost:%d", port)
	}

	var url string
	var protocol string
	if opts.Cert != "" && opts.Key != "" {
		protocol = "https"
	} else {
		protocol = "http"
	}
	if opts.Token == "" {
		url = fmt.Sprintf("%s://%s", protocol, addr)
	} else {
		url = fmt.Sprintf("%s://%s?plakar_token=%s", protocol, addr, opts.Token)
	}

	if !opts.NoSpawn {
		if err := utils.BrowserTrySpawn(url); err != nil {
			repo.Logger().Printf("failed to launch browser: %s", err)
			repo.Logger().Printf("you can access the webUI at %s", url)
		}
	}
	fmt.Fprintf(repo.AppContext().Stdout, "launching webUI at %s\n", url)

	var handler http.Handler = server
	if opts.Cors {
		handler = corsMiddleware(server)
	}

	s := &http.Server{Addr: addr, Handler: handler}
	errc := make(chan error, 1)
	go func() {
		if protocol == "https" {
			errc <- s.ListenAndServeTLS(opts.Cert, opts.Key)
		} else {
			errc <- s.ListenAndServe()
		}
	}()

	select {
	case err := <-errc:
		return err
	case <-repo.AppContext().Done():
	}

	sctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := s.Shutdown(sctx); err != nil {
		_ = s.Close()
	}
	<-errc
	return nil
}

// staticHandler serves the UI built in content's build/ directory, or the
// placeholder page when it was not built.
func staticHandler(content fs.FS) (http.Handler, error) {
	if _, err := fs.Stat(content, "build/index.html"); errors.Is(err, fs.ErrNotExist) {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write(placeholder)
		}), nil
	}

	statics, err := fs.Sub(content, "build")
	if err != nil {
		return nil, err
	}
	staticsFS := http.FS(statics)

	// Serve files from the ./build directory
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := path.Join("build", r.PathValue("path"))

		// If the asset does not exist, serve index.html.
		// All other errors are handled by the file server below.
		_, err := fs.Stat(content, p)
		if err != nil && os.IsNotExist(err) {
			http.ServeFileFS(w, r, content, "build/index.html")
			return
		}

		http.FileServer(staticsFS).ServeHTTP(w, r)
	}), nil
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, PUT, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Pass the request to the next handler
		next.ServeHTTP(w, r)
	})
}
