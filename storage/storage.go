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

package storage

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/chunking"
	"github.com/PlakarKorp/plakar/compression"
	"github.com/PlakarKorp/plakar/encryption"
	"github.com/PlakarKorp/plakar/hashing"
	"github.com/PlakarKorp/plakar/location"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/packfile"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/versioning"
	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
)

const VERSION string = "1.0.0"

func init() {
	versioning.Register(resources.RT_CONFIG, versioning.FromString(VERSION))
}

var ErrNotWritable = fmt.Errorf("storage is not writable")
var ErrNotReadable = fmt.Errorf("storage is not readable")
var ErrInvalidLocation = fmt.Errorf("invalid location")
var ErrInvalidMagic = fmt.Errorf("invalid magic")
var ErrInvalidVersion = fmt.Errorf("invalid version")

type Configuration struct {
	Version      versioning.Version `msgpack:"-" json:"version"`
	Timestamp    time.Time          `json:"timestamp"`
	RepositoryID uuid.UUID          `json:"repository_id"`

	Packfile    packfile.Configuration     `json:"packfile"`
	Chunking    chunking.Configuration     `json:"chunking"`
	Hashing     hashing.Configuration      `json:"hashing"`
	Compression *compression.Configuration `json:"compression"`
	Encryption  *encryption.Configuration  `json:"encryption"`
}

func NewConfiguration() *Configuration {
	return &Configuration{
		Version:      versioning.FromString(VERSION),
		Timestamp:    time.Now(),
		RepositoryID: uuid.Must(uuid.NewRandom()),

		Packfile: *packfile.NewDefaultConfiguration(),
		Chunking: *chunking.NewDefaultConfiguration(),
		Hashing:  *hashing.NewDefaultConfiguration(),

		Compression: compression.NewDefaultConfiguration(),
		Encryption:  encryption.NewDefaultConfiguration(),
	}
}

func NewConfigurationFromBytes(version versioning.Version, data []byte) (*Configuration, error) {
	var configuration Configuration
	err := msgpack.Unmarshal(data, &configuration)
	if err != nil {
		return nil, err
	}
	configuration.Version = version
	return &configuration, nil
}

func NewConfigurationFromWrappedBytes(data []byte) (*Configuration, error) {
	var configuration Configuration

	version := versioning.Version(binary.LittleEndian.Uint32(data[12:16]))

	data = data[:len(data)-int(STORAGE_FOOTER_SIZE)]
	data = data[STORAGE_HEADER_SIZE:]

	err := msgpack.Unmarshal(data, &configuration)
	if err != nil {
		return nil, err
	}
	configuration.Version = version
	return &configuration, nil
}

func (c *Configuration) ToBytes() ([]byte, error) {
	return msgpack.Marshal(c)
}

type Mode uint32

const (
	ModeWrite Mode = 1 << 1
	ModeRead  Mode = 1 << 2
)

type Store interface {
	Create(ctx *appcontext.AppContext, config []byte) error
	Open(ctx *appcontext.AppContext) ([]byte, error)
	Location() string
	Mode() Mode
	Size() int64 // this can be costly, call with caution

	GetStates() ([]objects.MAC, error)
	PutState(mac objects.MAC, rd io.Reader) (int64, error)
	GetState(mac objects.MAC) (io.Reader, error)
	DeleteState(mac objects.MAC) error

	GetPackfiles() ([]objects.MAC, error)
	PutPackfile(mac objects.MAC, rd io.Reader) (int64, error)
	GetPackfile(mac objects.MAC) (io.Reader, error)
	GetPackfileBlob(mac objects.MAC, offset uint64, length uint32) (io.Reader, error)
	DeletePackfile(mac objects.MAC) error

	GetLocks() ([]objects.MAC, error)
	PutLock(lockID objects.MAC, rd io.Reader) (int64, error)
	GetLock(lockID objects.MAC) (io.Reader, error)
	DeleteLock(lockID objects.MAC) error

	Close() error
}

type StoreFn func(*appcontext.AppContext, string, map[string]string) (Store, error)

var backends = location.New[StoreFn]("fs")

func Register(backend StoreFn, names ...string) {
	for _, name := range names {
		if !backends.Register(name, backend) {
			log.Fatalf("backend '%s' registered twice", name)
		}
	}
}

func Backends() []string {
	return backends.Names()
}

func New(ctx *appcontext.AppContext, storeConfig map[string]string) (Store, error) {
	location, ok := storeConfig["location"]
	if !ok {
		return nil, fmt.Errorf("missing location")
	}

	proto, location, backend, ok := backends.Lookup(location)
	if !ok {
		return nil, fmt.Errorf("backend '%s' does not exist", proto)
	}

	if proto == "fs" && !filepath.IsAbs(location) {
		location = filepath.Join(ctx.CWD, location)
		storeConfig["location"] = "fs://" + location
	} else if proto == "ptar" && !filepath.IsAbs(location) {
		location = filepath.Join(ctx.CWD, location)
		storeConfig["location"] = "ptar://" + location
	} else {
		storeConfig["location"] = proto + "://" + location
	}
	return backend(ctx, proto, storeConfig)
}

func Open(ctx *appcontext.AppContext, storeConfig map[string]string) (Store, []byte, error) {
	store, err := New(ctx, storeConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", flag.CommandLine.Name(), err)
		return nil, nil, err
	}

	serializedConfig, err := store.Open(ctx)
	if err != nil {
		return nil, nil, err
	}

	return store, serializedConfig, nil
}

func Create(ctx *appcontext.AppContext, storeConfig map[string]string, configuration []byte) (Store, error) {
	store, err := New(ctx, storeConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", flag.CommandLine.Name(), err)
		return nil, err
	}

	if err = store.Create(ctx, configuration); err != nil {
		return nil, err
	} else {
		return store, nil
	}
}
