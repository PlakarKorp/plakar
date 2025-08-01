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

package sftp

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"path"
	"strings"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/storage"
	plakarsftp "github.com/PlakarKorp/plakar/sftp"
	"github.com/pkg/sftp"
)

type Store struct {
	packfiles Buckets
	states    Buckets
	client    *sftp.Client

	config   map[string]string
	endpoint *url.URL
}

func init() {
	storage.Register("sftp", 0, NewStore)
}

func NewStore(ctx context.Context, proto string, storeConfig map[string]string) (storage.Store, error) {
	location := storeConfig["location"]
	if location == "" {
		return nil, fmt.Errorf("missing location")
	}

	parsed, err := url.Parse(location)
	if err != nil {
		return nil, err
	}

	return &Store{
		config:   storeConfig,
		endpoint: parsed,
	}, nil
}

func (s *Store) Location(ctx context.Context) (string, error) {
	return s.config["location"], nil
}

func (s *Store) Path(args ...string) string {
	root := strings.TrimPrefix(s.config["location"], "sftp://")
	atoms := strings.Split(root, "/")
	if len(atoms) == 0 {
		return "/"
	} else {
		root = "/" + strings.Join(atoms[1:], "/")
	}

	args = append(args, "")
	copy(args[1:], args)
	args[0] = root

	return path.Join(args...)
}

func (s *Store) Create(ctx context.Context, config []byte) error {
	client, err := plakarsftp.Connect(s.endpoint, s.config)
	if err != nil {
		return err
	}
	s.client = client

	dirfp, err := client.ReadDir(s.Path())
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		err = client.MkdirAll(s.Path())
		if err != nil {
			return err
		}
		err = client.Chmod(s.Path(), 0700)
		if err != nil {
			return err
		}
	} else {
		if len(dirfp) > 0 {
			return fmt.Errorf("directory %s is not empty", s.config["location"])
		}
	}
	s.packfiles = NewBuckets(client, s.Path("packfiles"))
	if err := s.packfiles.Create(); err != nil {
		return err
	}

	s.states = NewBuckets(client, s.Path("states"))
	if err := s.states.Create(); err != nil {
		return err
	}

	err = client.Mkdir(s.Path("locks"))
	if err != nil {
		return err
	}

	_, err = WriteToFileAtomic(client, s.Path("CONFIG"), bytes.NewReader(config))
	return err
}

func (s *Store) Open(ctx context.Context) ([]byte, error) {
	client, err := plakarsftp.Connect(s.endpoint, s.config)
	if err != nil {
		return nil, err
	}
	s.client = client

	rd, err := client.Open(s.Path("CONFIG"))
	if err != nil {
		return nil, err
	}
	defer rd.Close() // do we care about err?

	data, err := io.ReadAll(rd)
	if err != nil {
		return nil, err
	}

	s.packfiles = NewBuckets(client, s.Path("packfiles"))

	s.states = NewBuckets(client, s.Path("states"))

	return data, nil
}

func (s *Store) GetPackfiles(ctx context.Context) ([]objects.MAC, error) {
	return s.packfiles.List()
}

func (s *Store) GetPackfile(ctx context.Context, mac objects.MAC) (io.ReadCloser, error) {
	fp, err := s.packfiles.Get(mac)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			err = repository.ErrPackfileNotFound
		}
		return nil, err
	}

	return fp, nil
}

func (s *Store) GetPackfileBlob(ctx context.Context, mac objects.MAC, offset uint64, length uint32) (io.ReadCloser, error) {
	res, err := s.packfiles.GetBlob(mac, offset, length)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			err = repository.ErrPackfileNotFound
		}
		return nil, err
	}
	return res, nil
}

func (s *Store) Mode(ctx context.Context) (storage.Mode, error) {
	return storage.ModeRead | storage.ModeWrite, nil
}

func (s *Store) Size(ctx context.Context) (int64, error) {
	return -1, nil
}

func (s *Store) DeletePackfile(ctx context.Context, mac objects.MAC) error {
	return s.packfiles.Remove(mac)
}

func (s *Store) PutPackfile(ctx context.Context, mac objects.MAC, rd io.Reader) (int64, error) {
	return s.packfiles.Put(mac, rd)
}

func (s *Store) Close(ctx context.Context) error {
	return nil
}

/* Indexes */
func (s *Store) GetStates(ctx context.Context) ([]objects.MAC, error) {
	return s.states.List()
}

func (s *Store) PutState(ctx context.Context, mac objects.MAC, rd io.Reader) (int64, error) {
	return s.states.Put(mac, rd)
}

func (s *Store) GetState(ctx context.Context, mac objects.MAC) (io.ReadCloser, error) {
	return s.states.Get(mac)
}

func (s *Store) DeleteState(ctx context.Context, mac objects.MAC) error {
	return s.states.Remove(mac)
}

/* Locks */
func (s *Store) GetLocks(ctx context.Context) (ret []objects.MAC, err error) {
	entries, err := s.client.ReadDir(s.Path("locks"))
	if err != nil {
		return
	}

	for i := range entries {
		var t []byte
		t, err = hex.DecodeString(entries[i].Name())
		if err != nil {
			return
		}
		if len(t) != 32 {
			continue
		}
		ret = append(ret, objects.MAC(t))
	}
	return
}

func (s *Store) PutLock(ctx context.Context, lockID objects.MAC, rd io.Reader) (int64, error) {
	return WriteToFileAtomicTempDir(s.client, path.Join(s.Path("locks"), hex.EncodeToString(lockID[:])), rd, s.Path(""))
}

func (s *Store) GetLock(ctx context.Context, lockID objects.MAC) (io.ReadCloser, error) {
	fp, err := s.client.Open(path.Join(s.Path("locks"), hex.EncodeToString(lockID[:])))
	if err != nil {
		return nil, err
	}

	return fp, nil
}

func (s *Store) DeleteLock(ctx context.Context, lockID objects.MAC) error {
	return s.client.Remove(path.Join(s.Path("locks"), hex.EncodeToString(lockID[:])))
}
