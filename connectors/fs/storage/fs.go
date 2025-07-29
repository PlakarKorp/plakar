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

package fs

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/PlakarKorp/kloset/location"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/storage"
)

type Store struct {
	location  string
	packfiles Buckets
	states    Buckets
}

func init() {
	storage.Register("fs", location.FLAG_LOCALFS, NewStore)
}

func NewStore(ctx context.Context, proto string, storeConfig map[string]string) (storage.Store, error) {
	return &Store{
		location: storeConfig["location"],
	}, nil
}

func (s *Store) Location(ctx context.Context) (string, error) {
	return s.location, nil
}

func (s *Store) Path(args ...string) string {
	root := strings.TrimPrefix(s.location, "fs://")

	args = append(args, "")
	copy(args[1:], args)
	args[0] = root

	return filepath.Join(args...)
}

func (s *Store) Create(ctx context.Context, config []byte) error {
	dirfp, err := os.Open(s.Path())
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		err = os.MkdirAll(s.Path(), 0700)
		if err != nil {
			return err
		}
	} else {
		defer dirfp.Close()
		entries, err := dirfp.Readdir(1)
		if err != nil && !errors.Is(err, io.EOF) {
			return err
		}
		if len(entries) > 0 {
			return fmt.Errorf("directory %s is not empty", s.Path())
		}
	}

	s.packfiles = NewBuckets(s.Path("packfiles"))
	if err := s.packfiles.Create(); err != nil {
		return err
	}

	s.states = NewBuckets(s.Path("states"))
	if err := s.states.Create(); err != nil {
		return err
	}

	err = os.Mkdir(s.Path("locks"), 0700)
	if err != nil {
		return err
	}

	_, err = WriteToFileAtomic(s.Path("CONFIG"), bytes.NewReader(config))
	return err
}

func (s *Store) Open(ctx context.Context) ([]byte, error) {
	s.packfiles = NewBuckets(s.Path("packfiles"))
	s.states = NewBuckets(s.Path("states"))

	rd, err := os.Open(s.Path("CONFIG"))
	if err != nil {
		return nil, err
	}
	defer rd.Close() // do we care about err?

	data, err := io.ReadAll(rd)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (s *Store) Mode(ctx context.Context) (storage.Mode, error) {
	return storage.ModeRead | storage.ModeWrite, nil
}

func (s *Store) Size(ctx context.Context) (int64, error) {
	var size int64
	location := strings.TrimPrefix(s.location, "fs://")
	err := filepath.WalkDir(location, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}

		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			size += info.Size()
		}
		return nil
	})
	return size, err
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

func (s *Store) GetLocks(ctx context.Context) ([]objects.MAC, error) {
	ret := make([]objects.MAC, 0)

	locksdir, err := os.ReadDir(s.Path("locks"))
	if err != nil {
		return nil, err
	}

	for _, lock := range locksdir {
		if !lock.Type().IsRegular() {
			continue
		}

		lockID, err := hex.DecodeString(lock.Name())
		if err != nil {
			return nil, err
		}

		if len(lockID) != 32 {
			continue
		}

		ret = append(ret, objects.MAC(lockID))
	}

	return ret, nil
}

func (s *Store) PutLock(ctx context.Context, lockID objects.MAC, rd io.Reader) (int64, error) {
	return WriteToFileAtomicTempDir(filepath.Join(s.Path("locks"), hex.EncodeToString(lockID[:])), rd, s.Path(""))
}

func (s *Store) GetLock(ctx context.Context, lockID objects.MAC) (io.ReadCloser, error) {
	fp, err := os.Open(filepath.Join(s.Path("locks"), hex.EncodeToString(lockID[:])))
	if err != nil {
		return nil, err
	}

	return fp, nil
}

func (s *Store) DeleteLock(ctx context.Context, lockID objects.MAC) error {
	if err := os.Remove(filepath.Join(s.Path("locks"), hex.EncodeToString(lockID[:]))); err != nil {
		return err
	}

	return nil
}
