/*
 * Copyright (c) 2023 Gilles Chehade <gilles@poolp.org>
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

package null

import (
	"bytes"
	"context"
	"io"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/storage"
)

type Store struct {
	config     []byte
	Repository string
	location   string
}

func init() {
	// storage.Register(NewStore, "null")
}

func NewStore(ctx context.Context, storeConfig map[string]string) (storage.Store, error) {
	return &Store{
		location: storeConfig["location"],
	}, nil
}

func (s *Store) Location() string {
	return s.location
}

func (s *Store) Create(ctx context.Context, config []byte) error {
	s.config = config
	return nil
}

func (s *Store) Open(ctx context.Context) ([]byte, error) {
	return s.config, nil
}

func (s *Store) Close() error {
	return nil
}

func (s *Store) Mode() storage.Mode {
	return storage.ModeRead | storage.ModeWrite
}

func (s *Store) Size() int64 {
	return -1
}

// states
func (s *Store) GetStates() ([]objects.MAC, error) {
	return []objects.MAC{}, nil
}

func (s *Store) PutState(mac objects.MAC, rd io.Reader) (int64, error) {
	return 0, nil
}

func (s *Store) GetState(mac objects.MAC) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewBuffer([]byte{})), nil
}

func (s *Store) DeleteState(mac objects.MAC) error {
	return nil
}

// packfiles
func (s *Store) GetPackfiles() ([]objects.MAC, error) {
	return []objects.MAC{}, nil
}

func (s *Store) PutPackfile(mac objects.MAC, rd io.Reader) (int64, error) {
	return 0, nil
}

func (s *Store) GetPackfile(mac objects.MAC) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewBuffer([]byte{})), nil
}

func (s *Store) GetPackfileBlob(mac objects.MAC, offset uint64, length uint32) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewBuffer([]byte{})), nil
}

func (s *Store) DeletePackfile(mac objects.MAC) error {
	return nil
}

/* Locks */
func (s *Store) GetLocks() ([]objects.MAC, error) {
	return []objects.MAC{}, nil
}

func (s *Store) PutLock(lockID objects.MAC, rd io.Reader) (int64, error) {
	return 0, nil
}

func (s *Store) GetLock(lockID objects.MAC) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewBuffer([]byte{})), nil
}

func (s *Store) DeleteLock(lockID objects.MAC) error {
	return nil
}
