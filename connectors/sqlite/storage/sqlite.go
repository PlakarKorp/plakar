/*
 * Copyright (c) 2021 Gilles Chehade <gilles@poolp.org>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies. THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package database

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/PlakarKorp/kloset/location"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/storage"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

type Store struct {
	backend string

	conn    *sql.DB
	wrMutex sync.Mutex

	Repository string
	location   string
}

func init() {
	storage.Register("sqlite", location.FLAG_LOCALFS|location.FLAG_FILE, NewStore)
}

func NewStore(ctx context.Context, proto string, storeConfig map[string]string) (storage.Store, error) {
	if proto != "sqlite" {
		return nil, fmt.Errorf("unsupported database backend: %s", proto)
	}

	location := storeConfig["location"]
	return &Store{
		backend:  proto,
		location: location,
	}, nil
}

func (s *Store) Location() string {
	return s.location
}

func (s *Store) connect(addr string) error {
	conn, err := sql.Open(s.backend, addr)
	if err != nil {
		return err
	}
	s.conn = conn

	if s.backend == "sqlite" {
		_, err = s.conn.Exec("PRAGMA journal_mode=WAL;")
		if err != nil {
			return nil
		}
		_, err = s.conn.Exec("PRAGMA busy_timeout=2000;")
		if err != nil {
			return nil
		}

	}

	return nil
}

func (s *Store) Create(ctx context.Context, config []byte) error {
	location := strings.TrimPrefix(s.location, "sqlite://")
	err := s.connect(location)
	if err != nil {
		return err
	}

	statement, err := s.conn.Prepare(`CREATE TABLE IF NOT EXISTS configuration (
		value	BLOB
	);`)
	if err != nil {
		return err
	}
	defer statement.Close()
	statement.Exec()

	statement, err = s.conn.Prepare(`CREATE TABLE IF NOT EXISTS states (
		mac	VARCHAR(64) NOT NULL PRIMARY KEY,
		data		BLOB
	);`)
	if err != nil {
		return err
	}
	defer statement.Close()
	statement.Exec()

	statement, err = s.conn.Prepare(`CREATE TABLE IF NOT EXISTS packfiles (
		mac	VARCHAR(64) NOT NULL PRIMARY KEY,
		data		BLOB
	);`)
	if err != nil {
		return err
	}
	defer statement.Close()
	statement.Exec()

	statement, err = s.conn.Prepare(`CREATE TABLE IF NOT EXISTS locks (
		mac	VARCHAR(64) NOT NULL PRIMARY KEY,
		data		BLOB
	);`)
	if err != nil {
		return err
	}
	defer statement.Close()
	statement.Exec()

	statement, err = s.conn.Prepare(`INSERT INTO configuration(value) VALUES(?)`)
	if err != nil {
		return err
	}
	defer statement.Close()

	_, err = statement.Exec(config)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) Open(ctx context.Context) ([]byte, error) {
	location := strings.TrimPrefix(s.location, "sqlite://")
	err := s.connect(location)
	if err != nil {
		return nil, err
	}

	var buffer []byte

	err = s.conn.QueryRow(`SELECT value FROM configuration`).Scan(&buffer)
	if err != nil {
		return nil, err
	}
	return buffer, nil
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
	rows, err := s.conn.Query("SELECT mac FROM states")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	macs := make([]objects.MAC, 0)
	for rows.Next() {
		var mac []byte
		err = rows.Scan(&mac)
		if err != nil {
			return nil, err
		}
		var mac32 objects.MAC
		copy(mac32[:], mac)
		macs = append(macs, mac32)
	}
	return macs, nil
}

func (s *Store) PutState(mac objects.MAC, rd io.Reader) (int64, error) {
	data, err := io.ReadAll(rd)
	if err != nil {
		return 0, err
	}

	statement, err := s.conn.Prepare(`INSERT INTO states (mac, data) VALUES(?, ?)`)
	if err != nil {
		return 0, err
	}
	defer statement.Close()

	s.wrMutex.Lock()
	_, err = statement.Exec(mac[:], data)
	s.wrMutex.Unlock()
	if err != nil {
		var sqliteErr *sqlite.Error
		if !errors.As(err, &sqliteErr) {
			return 0, err
		}
		if sqliteErr.Code() != sqlite3.SQLITE_CONSTRAINT {
			return 0, err
		}
	}

	return int64(len(data)), nil
}

func (s *Store) GetState(mac objects.MAC) (io.Reader, error) {
	var data []byte
	err := s.conn.QueryRow(`SELECT data FROM states WHERE mac=?`, mac[:]).Scan(&data)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(data), nil
}

func (s *Store) DeleteState(mac objects.MAC) error {
	statement, err := s.conn.Prepare(`DELETE FROM states WHERE mac=?`)
	if err != nil {
		return err
	}
	defer statement.Close()

	s.wrMutex.Lock()
	_, err = statement.Exec(mac[:])
	s.wrMutex.Unlock()
	if err != nil {
		// if err is that it's already present, we should discard err and assume a concurrent write
		return err
	}
	return nil
}

// packfiles
func (s *Store) GetPackfiles() ([]objects.MAC, error) {
	rows, err := s.conn.Query("SELECT mac FROM packfiles")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	macs := make([]objects.MAC, 0)
	for rows.Next() {
		var mac []byte
		err = rows.Scan(&mac)
		if err != nil {
			return nil, err
		}
		var mac32 objects.MAC
		copy(mac32[:], mac)
		macs = append(macs, mac32)
	}
	return macs, nil
}

func (s *Store) PutPackfile(mac objects.MAC, rd io.Reader) (int64, error) {
	data, err := io.ReadAll(rd)
	if err != nil {
		return 0, err
	}

	statement, err := s.conn.Prepare(`INSERT INTO packfiles (mac, data) VALUES(?, ?)`)
	if err != nil {
		return 0, err
	}
	defer statement.Close()

	s.wrMutex.Lock()
	_, err = statement.Exec(mac[:], data)
	s.wrMutex.Unlock()
	if err != nil {
		var sqliteErr *sqlite.Error
		if !errors.As(err, &sqliteErr) {
			return 0, err
		}
		if sqliteErr.Code() != sqlite3.SQLITE_CONSTRAINT {
			return 0, err
		}
	}

	return int64(len(data)), nil
}

func (s *Store) GetPackfile(mac objects.MAC) (io.Reader, error) {
	var data []byte
	err := s.conn.QueryRow(`SELECT data FROM packfiles WHERE mac=?`, mac[:]).Scan(&data)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func (s *Store) GetPackfileBlob(mac objects.MAC, offset uint64, length uint32) (io.Reader, error) {
	var data []byte
	err := s.conn.QueryRow(`SELECT substr(data, ?, ?) FROM packfiles WHERE mac=?`, offset+1, length, mac[:]).Scan(&data)
	if err != nil {
		if err == sql.ErrNoRows {
			err = repository.ErrBlobNotFound
		}
		return nil, err
	}
	return bytes.NewBuffer(data), nil
}

func (s *Store) DeletePackfile(mac objects.MAC) error {
	statement, err := s.conn.Prepare(`DELETE FROM packfiles WHERE mac=?`)
	if err != nil {
		return err
	}
	defer statement.Close()

	s.wrMutex.Lock()
	_, err = statement.Exec(mac[:])
	s.wrMutex.Unlock()
	if err != nil {
		// if err is that it's already present, we should discard err and assume a concurrent write
		return err
	}
	return nil
}

func (s *Store) GetLocks() ([]objects.MAC, error) {
	rows, err := s.conn.Query("SELECT mac FROM locks")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ret := make([]objects.MAC, 0)
	for rows.Next() {
		var mac string
		err = rows.Scan(&mac)
		if err != nil {
			return nil, err
		}

		lockID, err := hex.DecodeString(mac)
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

func (s *Store) PutLock(lockID objects.MAC, rd io.Reader) (int64, error) {
	data, err := io.ReadAll(rd)
	if err != nil {
		return 0, err
	}

	statement, err := s.conn.Prepare(`INSERT INTO locks (mac, data) VALUES(?, ?)`)
	if err != nil {
		return 0, err
	}
	defer statement.Close()

	s.wrMutex.Lock()
	_, err = statement.Exec(hex.EncodeToString(lockID[:]), data)
	s.wrMutex.Unlock()
	if err != nil {
		var sqliteErr *sqlite.Error
		if !errors.As(err, &sqliteErr) {
			return 0, err
		}
		if sqliteErr.Code() != sqlite3.SQLITE_CONSTRAINT {
			return 0, err
		}
	}

	return int64(len(data)), nil
}

func (s *Store) GetLock(lockID objects.MAC) (io.Reader, error) {
	var data []byte
	err := s.conn.QueryRow(`SELECT data FROM locks WHERE mac=?`, hex.EncodeToString(lockID[:])).Scan(&data)
	if err != nil {
		return nil, err
	}
	return bytes.NewBuffer(data), nil
}

func (s *Store) DeleteLock(lockID objects.MAC) error {
	statement, err := s.conn.Prepare(`DELETE FROM locks WHERE mac=?`)
	if err != nil {
		return err
	}
	defer statement.Close()

	s.wrMutex.Lock()
	_, err = statement.Exec(hex.EncodeToString(lockID[:]))
	s.wrMutex.Unlock()
	if err != nil {
		// if err is that it's already present, we should discard err and assume a concurrent write
		return err
	}
	return nil
}
