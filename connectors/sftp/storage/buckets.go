/*
 * Copyright (c) 2025 Eric Faurot <eric@faurot.net>
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

package sftp

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"sync"

	"github.com/PlakarKorp/kloset/caching/lru"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/pkg/sftp"
	"golang.org/x/sync/errgroup"
)

type fileHandle struct {
	file *sftp.File
	stat os.FileInfo

	mtx sync.Mutex
}

type Buckets struct {
	client *sftp.Client
	path   string

	fileCache *lru.Cache[objects.MAC, *fileHandle]
}

func NewBuckets(sftpClient *sftp.Client, path string) Buckets {
	return Buckets{
		client: sftpClient,
		path:   path,
		fileCache: lru.New(10000, func(_ objects.MAC, h *fileHandle) error {
			h.file.Close()
			return nil
		}),
	}
}

func (buckets *Buckets) Create() error {
	var g errgroup.Group

	for i := 0; i < 256; i++ {
		i := i // capture the current value of i
		g.Go(func() error {
			dir := path.Join(buckets.path, fmt.Sprintf("%02x", i))
			if err := buckets.client.MkdirAll(dir); err != nil {
				return err
			}
			if err := buckets.client.Chmod(dir, 0755); err != nil {
				return err
			}
			return nil
		})
	}

	return g.Wait()
}

func (buckets *Buckets) List() ([]objects.MAC, error) {
	ret := make([]objects.MAC, 0)
	var mu sync.Mutex

	wg := sync.WaitGroup{}
	for i := 0; i < 256; i++ {
		path := path.Join(buckets.path, fmt.Sprintf("%02x", i))
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			entries, err := buckets.client.ReadDir(path)
			if err != nil {
				return
			}
			for _, entry := range entries {
				if entry.Name() == "." || entry.Name() == ".." {
					continue
				}
				if entry.IsDir() {
					continue
				}
				t, err := hex.DecodeString(entry.Name())
				if err != nil {
					continue
				}
				if len(t) != 32 {
					continue
				}
				var t32 objects.MAC
				copy(t32[:], t)

				mu.Lock()
				ret = append(ret, t32)
				mu.Unlock()
			}
		}(path)
	}
	wg.Wait()
	return ret, nil
}

func (buckets *Buckets) Path(mac objects.MAC) string {
	return path.Join(buckets.path,
		fmt.Sprintf("%02x", mac[0]),
		fmt.Sprintf("%064x", mac))
}

// Returns the file out of the cache
func (buckets *Buckets) openFile(mac objects.MAC) (*fileHandle, error) {
	var fh *fileHandle
	fh, ok := buckets.fileCache.Get(mac)

	if !ok {
		fp, err := buckets.client.Open(buckets.Path(mac))
		if err != nil {
			return nil, err
		}

		st, err := fp.Stat()
		if err != nil {
			return nil, err
		}

		fh = &fileHandle{fp, st, sync.Mutex{}}
		buckets.fileCache.Put(mac, fh)
	}

	return fh, nil
}

func (buckets *Buckets) Get(mac objects.MAC) (io.Reader, error) {
	fp, err := buckets.client.Open(buckets.Path(mac))
	if err != nil {
		return nil, err
	}
	return ClosingReader(fp)
}

func (buckets *Buckets) GetBlob(mac objects.MAC, offset uint64, length uint32) (io.Reader, error) {
	fh, err := buckets.openFile(mac)
	fh.mtx.Lock()
	defer fh.mtx.Unlock()

	if err != nil {
		return nil, err
	}

	if _, err := fh.file.Seek(int64(offset), io.SeekStart); err != nil {
		return nil, err
	}

	if fh.stat.Size() == 0 {
		return bytes.NewBuffer([]byte{}), nil
	}

	if int64(length) > (fh.stat.Size() - int64(offset)) {
		return nil, fmt.Errorf("invalid length")
	}

	ret := make([]byte, length)
	if _, err := io.ReadFull(fh.file, ret); err != nil {
		return nil, err
	}

	return bytes.NewBuffer(ret), nil
}

func (buckets *Buckets) Remove(mac objects.MAC) error {
	return buckets.client.Remove(buckets.Path(mac))
}

func (buckets *Buckets) Put(mac objects.MAC, rd io.Reader) (int64, error) {
	return WriteToFileAtomicTempDir(buckets.client, buckets.Path(mac), rd, buckets.path)
}
