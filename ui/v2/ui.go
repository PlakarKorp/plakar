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

package v2

import (
	"embed"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"net/http"
	"os"
	"path"

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
}

//go:embed frontend/*
var content embed.FS

func Ui(repo *repository.Repository, ctx *appcontext.AppContext, addr string, opts *UiOptions) error {
	server := http.NewServeMux()
	if err := api.SetupRoutes(server, repo, ctx, opts.Token); err != nil {
		return err
	}

	// Serve files from the ./frontend directory
	server.HandleFunc("/{path...}", func(w http.ResponseWriter, r *http.Request) {
		p := path.Join("frontend", r.PathValue("path"))

		_, err := content.Open(p)
		if os.IsNotExist(err) {
			// File does not exist, serve index.html
			index, err := content.ReadFile(path.Join("frontend", "index.html"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write(index)
			return
		} else if err != nil {
			// If we got an error (that wasn't that the file doesn't exist) stating the
			// file, return a 500 internal server error and stop
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		statics, err := fs.Sub(content, "frontend")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.FileServer(http.FS(statics)).ServeHTTP(w, r)
	})

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
	if opts.Token == "" {
		url = fmt.Sprintf("http://%s", addr)
	} else {
		url = fmt.Sprintf("http://%s?plakar_token=%s", addr, opts.Token)
	}

	if !opts.NoSpawn {
		if err := utils.BrowserTrySpawn(url); err != nil {
			repo.Logger().Printf("failed to launch browser: %s", err)
			repo.Logger().Printf("you can access the webUI at %s", url)
			return err
		}
	} else {
		fmt.Fprintf(repo.AppContext().Stdout, "launching webUI at %s\n", url)
	}

	var handler http.Handler = server
	if opts.Cors {
		handler = corsMiddleware(server)
	}

	s := &http.Server{Addr: addr, Handler: handler}
	go func() {
		<-repo.AppContext().Done()
		s.Shutdown(repo.AppContext().Context)
	}()

	return s.ListenAndServe()
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
