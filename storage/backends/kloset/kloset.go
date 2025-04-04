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

package kloset

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"strings"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/storage"
	"github.com/PlakarKorp/plakar/versioning"
)

type Store struct {
	config     []byte
	Repository string
	location   string

	mode storage.Mode

	fp *os.File
}

func init() {
	storage.Register(NewStore, "kloset")
}

func NewStore(storeConfig map[string]string) (storage.Store, error) {
	return &Store{
		location: storeConfig["location"],
	}, nil
}

func (s *Store) Location() string {
	return s.location
}

func (s *Store) Create(config []byte) error {
	s.config = config
	s.mode = storage.ModeRead | storage.ModeWrite

	location := strings.TrimPrefix(s.location, "kloset://")
	if location == "" {
		return storage.ErrInvalidLocation
	}

	fp, err := os.OpenFile(location, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	s.fp = fp

	fp.Write([]byte{'_', 'K', 'L', 'O', 'S', 'E', 'T', '_'})

	version := versioning.FromString("1.0.0")

	versionBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(versionBytes, uint32(version))
	fp.Write(versionBytes)

	configLengthBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(configLengthBytes, uint32(len(config)))
	fp.Write(configLengthBytes)
	fp.Write(config)
	return nil
}

func (s *Store) Open() ([]byte, error) {
	s.mode = storage.ModeRead

	location := strings.TrimPrefix(s.location, "kloset://")
	if location == "" {
		return nil, storage.ErrInvalidLocation
	}

	fp, err := os.Open(location)
	if err != nil {
		return nil, err
	}
	s.fp = fp

	magic := make([]byte, 8)
	_, err = io.ReadFull(fp, magic)
	if err != nil {
		return nil, err
	}
	if string(magic) != "_KLOSET_" {
		return nil, storage.ErrInvalidMagic
	}

	versionBytes := make([]byte, 4)
	_, err = io.ReadFull(fp, versionBytes)
	if err != nil {
		return nil, err
	}

	//version := binary.LittleEndian.Uint32(versionBytes)
	//if version != 1 {
	//	return nil, storage.ErrInvalidVersion
	//}

	configLengthBytes := make([]byte, 4)
	_, err = io.ReadFull(fp, configLengthBytes)
	if err != nil {
		return nil, err
	}
	configLength := binary.LittleEndian.Uint32(configLengthBytes)

	config := make([]byte, configLength)
	_, err = io.ReadFull(fp, config)
	if err != nil {
		return nil, err
	}
	s.config = config

	return s.config, nil
}

func (s *Store) Close() error {
	return s.fp.Close()
}

func (s *Store) Mode() storage.Mode {
	return s.mode
}

// states
func (s *Store) GetStates() ([]objects.MAC, error) {
	if s.mode&storage.ModeRead == 0 {
		return []objects.MAC{}, storage.ErrNotReadable
	}
	return []objects.MAC{}, nil
}

func (s *Store) PutState(mac objects.MAC, rd io.Reader) error {
	if s.mode&storage.ModeWrite == 0 {
		return storage.ErrNotWritable
	}
	return nil
}

func (s *Store) GetState(mac objects.MAC) (io.Reader, error) {
	if s.mode&storage.ModeRead == 0 {
		return nil, storage.ErrNotReadable
	}
	return bytes.NewBuffer([]byte{}), nil
}

func (s *Store) DeleteState(mac objects.MAC) error {
	if s.mode&storage.ModeWrite == 0 {
		return storage.ErrNotWritable
	}
	return nil
}

// packfiles
func (s *Store) GetPackfiles() ([]objects.MAC, error) {
	if s.mode&storage.ModeRead == 0 {
		return []objects.MAC{}, storage.ErrNotReadable
	}
	return []objects.MAC{}, nil
}

func (s *Store) PutPackfile(mac objects.MAC, rd io.Reader) error {
	if s.mode&storage.ModeWrite == 0 {
		return storage.ErrNotWritable
	}

	_, err := io.Copy(s.fp, rd)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) GetPackfile(mac objects.MAC) (io.Reader, error) {
	if s.mode&storage.ModeRead == 0 {
		return nil, storage.ErrNotReadable
	}
	return bytes.NewBuffer([]byte{}), nil
}

func (s *Store) GetPackfileBlob(mac objects.MAC, offset uint64, length uint32) (io.Reader, error) {
	if s.mode&storage.ModeRead == 0 {
		return nil, storage.ErrNotReadable
	}
	return bytes.NewBuffer([]byte{}), nil
}

func (s *Store) DeletePackfile(mac objects.MAC) error {
	if s.mode&storage.ModeWrite == 0 {
		return storage.ErrNotWritable
	}
	return nil
}

/* Locks */
func (s *Store) GetLocks() ([]objects.MAC, error) {
	if s.mode&storage.ModeRead == 0 {
		return []objects.MAC{}, storage.ErrNotReadable
	}
	return []objects.MAC{}, nil
}

func (s *Store) PutLock(lockID objects.MAC, rd io.Reader) error {
	if s.mode&storage.ModeWrite == 0 {
		return storage.ErrNotWritable
	}
	return nil
}

func (s *Store) GetLock(lockID objects.MAC) (io.Reader, error) {
	if s.mode&storage.ModeRead == 0 {
		return nil, storage.ErrNotReadable
	}
	return bytes.NewBuffer([]byte{}), nil
}

func (s *Store) DeleteLock(lockID objects.MAC) error {
	if s.mode&storage.ModeWrite == 0 {
		return storage.ErrNotWritable
	}
	return nil
}
