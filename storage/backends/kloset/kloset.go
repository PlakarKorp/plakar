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
	"fmt"
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

	configOffset int64
	configLength int64

	packfileOffset int64
	packfileLength int64

	stateOffset int64
	stateLength int64
}

var stateMAC = objects.MAC{0x0f, 0x0e, 0x0d, 0x0c, 0x0b, 0x0a, 0x09, 0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, 0x00}
var packfileMAC = objects.MAC{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f}

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

	fp.Write(config)

	s.configOffset = 12
	s.configLength = int64(len(config))
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

	_, err = fp.Seek(-48, io.SeekEnd)
	if err != nil {
		return nil, err
	}

	binary.Read(s.fp, binary.LittleEndian, &s.configOffset)
	binary.Read(s.fp, binary.LittleEndian, &s.configLength)
	binary.Read(s.fp, binary.LittleEndian, &s.packfileOffset)
	binary.Read(s.fp, binary.LittleEndian, &s.packfileLength)
	binary.Read(s.fp, binary.LittleEndian, &s.stateOffset)
	binary.Read(s.fp, binary.LittleEndian, &s.stateLength)

	_, err = fp.Seek(s.configOffset, io.SeekStart)
	if err != nil {
		return nil, err
	}

	config := make([]byte, s.configLength)
	_, err = io.ReadFull(fp, config)
	if err != nil {
		return nil, err
	}
	s.config = config

	return s.config, nil
}

func (s *Store) Close() error {
	if s.mode&storage.ModeWrite != 0 {
		binary.Write(s.fp, binary.LittleEndian, s.configOffset)
		binary.Write(s.fp, binary.LittleEndian, s.configLength)
		binary.Write(s.fp, binary.LittleEndian, s.packfileOffset)
		binary.Write(s.fp, binary.LittleEndian, s.packfileLength)
		binary.Write(s.fp, binary.LittleEndian, s.stateOffset)
		binary.Write(s.fp, binary.LittleEndian, s.stateLength)
	}
	return s.fp.Close()
}

func (s *Store) Mode() storage.Mode {
	return s.mode
}

// states
func (s *Store) GetStates() ([]objects.MAC, error) {
	if s.mode&storage.ModeWrite != 0 {
		return []objects.MAC{}, nil
	}

	return []objects.MAC{
		stateMAC,
	}, nil
}

func (s *Store) PutState(mac objects.MAC, rd io.Reader) error {
	if s.mode&storage.ModeWrite == 0 {
		return storage.ErrNotWritable
	}

	s.stateOffset = s.packfileOffset + s.packfileLength
	nbytes, err := io.Copy(s.fp, rd)
	if err != nil {
		return err
	}
	s.stateLength = nbytes

	return nil
}

func (s *Store) GetState(mac objects.MAC) (io.Reader, error) {
	if mac != stateMAC {
		return nil, fmt.Errorf("invalid MAC: %s", mac)
	}
	return io.NewSectionReader(s.fp, s.stateOffset, s.stateLength), nil
}

func (s *Store) DeleteState(mac objects.MAC) error {
	if s.mode&storage.ModeWrite == 0 {
		return storage.ErrNotWritable
	}
	return nil
}

// packfiles
func (s *Store) GetPackfiles() ([]objects.MAC, error) {
	return []objects.MAC{
		packfileMAC,
	}, nil
}

func (s *Store) PutPackfile(mac objects.MAC, rd io.Reader) error {
	if s.mode&storage.ModeWrite == 0 {
		return storage.ErrNotWritable
	}

	s.packfileOffset = s.configOffset + s.configLength
	nbytes, err := io.Copy(s.fp, rd)
	if err != nil {
		return err
	}
	s.packfileLength = nbytes

	return nil
}

func (s *Store) GetPackfile(mac objects.MAC) (io.Reader, error) {
	return io.NewSectionReader(s.fp, s.packfileOffset, s.packfileLength), nil
}

func (s *Store) GetPackfileBlob(mac objects.MAC, offset uint64, length uint32) (io.Reader, error) {
	return io.NewSectionReader(s.fp, s.packfileOffset+int64(offset), int64(length)), nil
}

func (s *Store) DeletePackfile(mac objects.MAC) error {
	if s.mode&storage.ModeWrite == 0 {
		return storage.ErrNotWritable
	}
	return nil
}

/* Locks */
func (s *Store) GetLocks() ([]objects.MAC, error) {
	return []objects.MAC{}, nil
}

func (s *Store) PutLock(lockID objects.MAC, rd io.Reader) error {
	if s.mode&storage.ModeWrite == 0 {
		return storage.ErrNotWritable
	}
	return nil
}

func (s *Store) GetLock(lockID objects.MAC) (io.Reader, error) {
	return bytes.NewBuffer([]byte{}), nil
}

func (s *Store) DeleteLock(lockID objects.MAC) error {
	if s.mode&storage.ModeWrite == 0 {
		return storage.ErrNotWritable
	}
	return nil
}
