package http

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

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/PlakarKorp/kloset/locate"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/kloset/snapshot/header"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/cached"
)

type ListFn func(ctx context.Context, w http.ResponseWriter, r *http.Request)
type OpenFn func(ctx context.Context, snapshot string) (fs.FS, error)

func ExecuteHTTP(ctx *appcontext.AppContext, repo *repository.Repository, mountpoint string, locateOptions *locate.LocateOptions, chrootfs fs.FS) error {
	addr := strings.TrimPrefix(mountpoint, "http://")

	var handler http.Handler
	handler = NewDynamicSnapshotHandler(
		func(innertctx context.Context, w http.ResponseWriter, r *http.Request) {
			_, err := cached.RebuildStateFromStore(ctx, repo.Configuration().RepositoryID, ctx.StoreConfig, false)
			if err != nil {
				http.Error(w, "failed to rebuild state", http.StatusInternalServerError)
				return
			}

			snapshotIDs, _, err := locate.Match(repo, locateOptions)
			if err != nil {
				http.Error(w, "failed to list snapshots", http.StatusInternalServerError)
				return
			}

			headers := make([]header.Header, 0, len(snapshotIDs))
			for _, snapID := range snapshotIDs {
				snap, err := snapshot.Load(repo, snapID)
				if err != nil {
					continue
				}
				headers = append(headers, *snap.Header)
				snap.Close()
			}

			sort.Slice(headers, func(i, j int) bool {
				return bytes.Compare(headers[i].Identifier[:], headers[j].Identifier[:]) < 0
			})

			fmt.Fprintf(w, "<!doctype html>\n")
			fmt.Fprintf(w, "<meta name=\"viewport\" content=\"width=device-width\">\n")
			fmt.Fprintf(w, "<pre>\n")
			for _, hdr := range headers {
				snapURL := fmt.Sprintf("/%x/", hdr.Identifier)
				fmt.Fprintf(w, "<a href=\"%s\">%x</a>\n", snapURL, hdr.Identifier[0:4])
			}
			fmt.Fprintf(w, "</pre>\n")
		},
		func(innerctx context.Context, snapshotID_string string) (fs.FS, error) {
			snapshotID, err := hex.DecodeString(snapshotID_string)
			if err != nil {
				return nil, err
			}
			snap, err := snapshot.Load(repo, objects.MAC(snapshotID))
			if err != nil {
				return nil, err
			}
			defer snap.Close()

			return snap.Filesystem()
		},
	)

	if chrootfs != nil {
		handler = http.FileServer(http.FS(chrootfs))
	}

	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
		// Optional: bind request contexts to app ctx
		BaseContext: func(_ net.Listener) context.Context { return ctx },
	}

	errCh := make(chan error, 1)
	go func() {
		ctx.GetLogger().Info("HTTP serving at http://%s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	// Wait for either ctx cancellation (Ctrl-C in your app) or server error.
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		<-errCh // wait for ListenAndServe to return
		return nil
	case err := <-errCh:
		if err != nil {
			return err
		}
		return nil
	}
}

func NewDynamicSnapshotHandler(list ListFn, open OpenFn) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := path.Clean(r.URL.Path)
		if p == "." {
			p = "/"
		}

		if p == "/" {
			list(r.Context(), w, r)
			return
		}

		p = strings.TrimPrefix(p, "/")
		snap, _, _ := strings.Cut(p, "/")

		snapFS, err := open(r.Context(), snap)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, "failed to open snapshot", http.StatusInternalServerError)
			return
		}
		http.StripPrefix("/"+snap, http.FileServer(http.FS(snapFS))).ServeHTTP(w, r)
	})
}
