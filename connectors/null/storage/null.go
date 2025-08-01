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

func (s *Store) Location(ctx context.Context) (string, error) {
	return s.location, nil
}

func (s *Store) Create(ctx context.Context, config []byte) error {
	s.config = config
	return nil
}

func (s *Store) Open(ctx context.Context) ([]byte, error) {
	return s.config, nil
}

func (s *Store) Close(ctx context.Context) error {
	return nil
}

func (s *Store) Mode(ctx context.Context) (storage.Mode, error) {
	return storage.ModeRead | storage.ModeWrite, nil
}

func (s *Store) Size(ctx context.Context) (int64, error) {
	return -1, nil
}

// states
func (s *Store) GetStates(ctx context.Context) ([]objects.MAC, error) {
	return []objects.MAC{}, nil
}

func (s *Store) PutState(ctx context.Context, mac objects.MAC, rd io.Reader) (int64, error) {
	return 0, nil
}

func (s *Store) GetState(ctx context.Context, mac objects.MAC) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewBuffer([]byte{})), nil
}

func (s *Store) DeleteState(ctx context.Context, mac objects.MAC) error {
	return nil
}

// packfiles
func (s *Store) GetPackfiles(ctx context.Context) ([]objects.MAC, error) {
	return []objects.MAC{}, nil
}

func (s *Store) PutPackfile(ctx context.Context, mac objects.MAC, rd io.Reader) (int64, error) {
	return 0, nil
}

func (s *Store) GetPackfile(ctx context.Context, mac objects.MAC) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewBuffer([]byte{})), nil
}

func (s *Store) GetPackfileBlob(ctx context.Context, mac objects.MAC, offset uint64, length uint32) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewBuffer([]byte{})), nil
}

func (s *Store) DeletePackfile(ctx context.Context, mac objects.MAC) error {
	return nil
}

/* Locks */
func (s *Store) GetLocks(ctx context.Context) ([]objects.MAC, error) {
	return []objects.MAC{}, nil
}

func (s *Store) PutLock(ctx context.Context, lockID objects.MAC, rd io.Reader) (int64, error) {
	return 0, nil
}

func (s *Store) GetLock(ctx context.Context, lockID objects.MAC) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewBuffer([]byte{})), nil
}

func (s *Store) DeleteLock(ctx context.Context, lockID objects.MAC) error {
	return nil
}
