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

package main

import (
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/PlakarKorp/plakar/cached"
	"github.com/PlakarKorp/plakar/exitcodes"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"
)

// startFakeCached listens on cachedir/cached.sock and answers the version
// handshake with version. With the running version it then reads the
// request and hangs up, which the client takes as a successful rebuild; any
// other version stands for a cached daemon from another plakar release.
func startFakeCached(t *testing.T, cacheDir, version string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(cacheDir, 0700))
	ln, err := net.Listen("unix", filepath.Join(cacheDir, "cached.sock"))
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			serveFakeCached(conn, version)
			conn.Close()
		}
	}()
	t.Cleanup(func() {
		ln.Close()
		wg.Wait()
	})
}

func serveFakeCached(conn net.Conn, version string) {
	dec := msgpack.NewDecoder(conn)
	var theirs []byte
	if err := dec.Decode(&theirs); err != nil {
		return
	}
	if err := msgpack.NewEncoder(conn).Encode([]byte(version)); err != nil {
		return
	}
	if version == utils.GetVersion() {
		var req cached.RequestPkt
		_ = dec.Decode(&req)
	}
}

// shortCacheDir returns a cache directory short enough for a unix socket
// path; t.TempDir() is too long on macOS.
func shortCacheDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "pc")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// A failed state rebuild used to exit 0 without running the command, so a
// cron'd backup facing a stale cached looked successful.
func TestEntryPointFailedStateRebuildExitsNonZero(t *testing.T) {
	cacheDir := shortCacheDir(t)

	t.Setenv("PLAKAR_INSECURE_PLAINTEXT", "1")
	repo := filepath.Join(t.TempDir(), "repo")

	status, stdout, stderr := runEntryPoint(t, "-cachedir", cacheDir, "at", repo, "create", "-plaintext")
	require.Equal(t, 0, status, "stdout=%s\nstderr=%s", stdout, stderr)

	startFakeCached(t, cacheDir, "v0.0.0-stale")

	for _, args := range [][]string{
		{"ls"},
		{"backup", t.TempDir()},
	} {
		argv := append([]string{"-cachedir", cacheDir, "at", repo}, args...)
		status, stdout, stderr := runEntryPoint(t, argv...)
		require.Equal(t, exitcodes.CachedVersionMismatch, status, "%v: stdout=%s\nstderr=%s", args, stdout, stderr)
		require.Contains(t, stderr, cached.ErrWrongVersion.Error())
		require.Contains(t, stderr, "plakar cached stop")
	}
}
