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

package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/poolpOrg/plakar/cache"
	"github.com/poolpOrg/plakar/encryption"
	"github.com/poolpOrg/plakar/logger"
	"github.com/poolpOrg/plakar/storage"
)

func (store *DatabaseStore) connect(addr string) error {
	var connectionString string
	if strings.HasPrefix(addr, "sqlite://") {
		store.backend = "sqlite3"
		connectionString = addr[9:]
	} else {
		return fmt.Errorf("unsupported database backend: %s", addr)
	}

	conn, err := sql.Open(store.backend, connectionString)
	if err != nil {
		return err
	}
	store.conn = conn

	if store.backend == "sqlite3" {
		_, err = store.conn.Exec("PRAGMA foreign_keys = ON")
		if err != nil {
			return nil
		}
	}

	return nil
}

func (store *DatabaseStore) Create(repository string, config storage.StoreConfig) error {
	t0 := time.Now()
	defer func() {
		logger.Profile("Create(%s): %s", repository, time.Since(t0))
	}()

	err := store.connect(repository)
	if err != nil {
		return err
	}

	statement, err := store.conn.Prepare(`CREATE TABLE IF NOT EXISTS configuration (
		configKey	VARCHAR(32) NOT NULL PRIMARY KEY,
		configValue	VARCHAR(64)
	);`)
	if err != nil {
		return err
	}
	defer statement.Close()
	statement.Exec()

	statement, err = store.conn.Prepare(`CREATE TABLE IF NOT EXISTS indexes (
		indexUuid	VARCHAR(36) NOT NULL PRIMARY KEY,
		indexBlob	BLOB
	);`)
	if err != nil {
		return err
	}
	defer statement.Close()
	statement.Exec()

	statement, err = store.conn.Prepare(`CREATE TABLE IF NOT EXISTS objects (
		objectChecksum	VARCHAR(64) NOT NULL PRIMARY KEY,
		objectBlob		BLOB
	);`)
	if err != nil {
		return err
	}
	defer statement.Close()
	statement.Exec()

	statement, err = store.conn.Prepare(`CREATE TABLE IF NOT EXISTS chunks (
		chunkChecksum	VARCHAR(64) NOT NULL PRIMARY KEY,
		chunkBlob		BLOB
	);`)
	if err != nil {
		return err
	}
	defer statement.Close()
	statement.Exec()

	statement, err = store.conn.Prepare(`CREATE TABLE IF NOT EXISTS chunksReferences (
		indexUuid		VARCHAR(36) NOT NULL,
		chunkChecksum	VARCHAR(64) NOT NULL,

		FOREIGN KEY (chunkChecksum)
			REFERENCES chunks (chunkChecksum)
	);`)
	if err != nil {
		return err
	}
	defer statement.Close()
	statement.Exec()

	statement, err = store.conn.Prepare(`CREATE TABLE IF NOT EXISTS objectsReferences (
		indexUuid		VARCHAR(36) NOT NULL,
		objectChecksum	VARCHAR(64) NOT NULL,

		FOREIGN KEY (objectChecksum)
			REFERENCES objects (objectChecksum)

	);`)
	if err != nil {
		return err
	}
	defer statement.Close()
	statement.Exec()

	statement, err = store.conn.Prepare(`INSERT INTO configuration(configKey, configValue) VALUES(?, ?)`)
	defer statement.Close()
	if err != nil {
		return err
	}
	_, err = statement.Exec("Uuid", config.Uuid)
	if err != nil {
		return err
	}

	_, err = statement.Exec("Compressed", config.Compressed)
	if err != nil {
		return err
	}

	_, err = statement.Exec("Encrypted", config.Encrypted)
	if err != nil {
		return err
	}

	return nil
}

func (store *DatabaseStore) GetCache() *cache.Cache {
	return store.Cache
}

func (store *DatabaseStore) GetKeypair() *encryption.Keypair {
	return store.Keypair
}

func (store *DatabaseStore) SetCache(localCache *cache.Cache) error {
	store.Cache = localCache
	return nil
}

func (store *DatabaseStore) SetKeypair(localKeypair *encryption.Keypair) error {
	store.Keypair = localKeypair
	return nil
}

func (store *DatabaseStore) Open(repository string) error {
	t0 := time.Now()
	defer func() {
		logger.Profile("Open(%s): %s", repository, time.Since(t0))
	}()

	err := store.connect(repository)
	if err != nil {
		return err
	}

	storeConfig := storage.StoreConfig{}
	err = store.conn.QueryRow(`SELECT configValue FROM configuration WHERE configKey='Uuid'`).Scan(&storeConfig.Uuid)
	if err != nil {
		return err
	}
	err = store.conn.QueryRow(`SELECT configValue FROM configuration WHERE configKey='Compressed'`).Scan(&storeConfig.Compressed)
	if err != nil {
		return err
	}
	err = store.conn.QueryRow(`SELECT configValue FROM configuration WHERE configKey='Encrypted'`).Scan(&storeConfig.Encrypted)
	if err != nil {
		return err
	}
	store.config = storeConfig

	return nil

}

func (store *DatabaseStore) Configuration() storage.StoreConfig {
	return store.config
}

func (store *DatabaseStore) Transaction() (storage.Transaction, error) {
	t0 := time.Now()
	defer func() {
		logger.Profile("Transaction(): %s", time.Since(t0))
	}()

	Uuid, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	dbTx, err := store.conn.BeginTx(context.TODO(), nil)
	if err != nil {
		return nil, err
	}

	tx := &DatabaseTransaction{}
	tx.dbTx = dbTx
	tx.Uuid = Uuid.String()
	tx.store = store

	return tx, nil
}

func (store *DatabaseStore) GetIndexes() ([]string, error) {
	t0 := time.Now()
	defer func() {
		logger.Profile("GetIndexes(): %s", time.Since(t0))
	}()

	rows, err := store.conn.Query("SELECT indexUuid FROM indexes")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var indexes []string
	for rows.Next() {
		var indexUuid string
		err = rows.Scan(&indexUuid)
		if err != nil {
			return nil, err
		}
		indexes = append(indexes, indexUuid)
	}
	return indexes, nil
}

func (store *DatabaseStore) GetIndex(Uuid string) ([]byte, error) {
	t0 := time.Now()
	defer func() {
		logger.Profile("GetIndex(%s): %s", Uuid, time.Since(t0))
	}()

	var data []byte
	err := store.conn.QueryRow(`SELECT indexBlob FROM indexes WHERE indexUuid=?`, Uuid).Scan(&data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (store *DatabaseStore) GetObject(checksum string) ([]byte, error) {
	t0 := time.Now()
	defer func() {
		logger.Profile("GetObject(%s): %s", checksum, time.Since(t0))
	}()

	var data []byte
	err := store.conn.QueryRow(`SELECT objectBlob FROM objects WHERE objectChecksum=?`, checksum).Scan(&data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (store *DatabaseStore) GetChunk(checksum string) ([]byte, error) {
	t0 := time.Now()
	defer func() {
		logger.Profile("GetChunk(%s): %s", checksum, time.Since(t0))
	}()

	var data []byte
	err := store.conn.QueryRow(`SELECT chunkBlob FROM chunks WHERE chunkChecksum=?`, checksum).Scan(&data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (store *DatabaseStore) Purge(id string) error {
	t0 := time.Now()
	defer func() {
		logger.Profile("Purge(%s): %s", id, time.Since(t0))
	}()

	return nil
}

//////

func (transaction *DatabaseTransaction) GetUuid() string {
	return transaction.Uuid
}
func (transaction *DatabaseTransaction) ReferenceChunks(keys []string) ([]bool, error) {
	t0 := time.Now()
	defer func() {
		logger.Profile("tx[%s].ReferenceChunks([%d keys]): %s", transaction.GetUuid(), len(keys), time.Since(t0))
	}()

	ret := make([]bool, 0)
	for _, key := range keys {
		res, err := transaction.dbTx.Exec("INSERT OR REPLACE INTO chunksReferences (indexUuid, chunkChecksum) VALUES(?, ?)", transaction.GetUuid(), key)
		if err != nil {
			// there has to be a better way ...
			if err.Error() == "FOREIGN KEY constraint failed" {
				ret = append(ret, false)
				continue
			}
			return nil, err
		}
		count, err := res.RowsAffected()
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		ret = append(ret, count != int64(0))
	}

	return ret, nil
}

func (transaction *DatabaseTransaction) ReferenceObjects(keys []string) ([]bool, error) {
	t0 := time.Now()
	defer func() {
		logger.Profile("tx[%s].ReferenceObjects([%d keys]): %s", transaction.GetUuid(), len(keys), time.Since(t0))
	}()

	ret := make([]bool, 0)
	for _, key := range keys {
		res, err := transaction.dbTx.Exec("INSERT OR REPLACE INTO objectsReferences (indexUuid, objectChecksum) VALUES(?, ?)", transaction.GetUuid(), key)
		if err != nil {
			// there has to be a better way ...
			if err.Error() == "FOREIGN KEY constraint failed" {
				ret = append(ret, false)
				continue
			}
			return nil, err
		}
		count, err := res.RowsAffected()
		if err != nil {
			fmt.Println(err)
			return nil, err
		}
		ret = append(ret, count != int64(0))
	}

	return ret, nil
}

func (transaction *DatabaseTransaction) PutObject(checksum string, data []byte) error {
	t0 := time.Now()
	defer func() {
		logger.Profile("tx[%s].PutObject(%s) <- %d bytes: %s", transaction.GetUuid(), checksum, len(data), time.Since(t0))
	}()

	statement, err := transaction.dbTx.Prepare(`INSERT INTO objects (objectChecksum, objectBlob) VALUES(?, ?)`)
	if err != nil {
		return err
	}
	defer statement.Close()

	_, err = statement.Exec(checksum, data)
	if err != nil {
		// if err is that it's already present, we should discard err and assume a concurrent write
		return err
	}

	return nil
}

func (transaction *DatabaseTransaction) PutChunk(checksum string, data []byte) error {
	t0 := time.Now()
	defer func() {
		logger.Profile("tx[%s].PutChunk(%s) <- %d bytes: %s", transaction.GetUuid(), checksum, len(data), time.Since(t0))
	}()

	statement, err := transaction.dbTx.Prepare(`INSERT INTO chunks (chunkChecksum, chunkBlob) VALUES(?, ?)`)
	if err != nil {
		return err
	}
	defer statement.Close()

	_, err = statement.Exec(checksum, data)
	if err != nil {
		// if err is that it's already present, we should discard err and assume a concurrent write
		return err
	}

	return nil
}

func (transaction *DatabaseTransaction) PutIndex(data []byte) error {
	t0 := time.Now()
	defer func() {
		logger.Profile("tx[%s].PutIndex() <- %d bytes: %s", transaction.GetUuid(), len(data), time.Since(t0))
	}()

	statement, err := transaction.dbTx.Prepare(`INSERT INTO indexes (indexUuid, indexBlob) VALUES(?, ?)`)
	if err != nil {
		return err
	}
	defer statement.Close()

	_, err = statement.Exec(transaction.GetUuid(), data)
	if err != nil {
		return err
	}

	return nil
}

func (transaction *DatabaseTransaction) Commit() error {
	t0 := time.Now()
	defer func() {
		logger.Profile("tx[%s].Commit(): %s", transaction.GetUuid(), time.Since(t0))
	}()

	return transaction.dbTx.Commit()
}
