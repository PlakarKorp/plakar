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
	"sort"
	"strings"
	"time"

	"github.com/PlakarKorp/plakar/chunking"
	"github.com/PlakarKorp/plakar/compression"
	"github.com/PlakarKorp/plakar/encryption"
	"github.com/PlakarKorp/plakar/hashing"
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

type Configuration struct {
	Version      versioning.Version `msgpack:"-"`
	Timestamp    time.Time
	RepositoryID uuid.UUID

	Packfile    packfile.Configuration
	Chunking    chunking.Configuration
	Hashing     hashing.Configuration
	Compression *compression.Configuration
	Encryption  *encryption.Configuration
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
	Create(config []byte) error
	Open() ([]byte, error)
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

var backends = make(map[string]func(map[string]string) (Store, error))

func NewStore(name string, storeConfig map[string]string) (Store, error) {
	if backend, exists := backends[name]; !exists {
		return nil, fmt.Errorf("backend '%s' does not exist", name)
	} else {
		return backend(storeConfig)
	}
}

func Register(backend func(map[string]string) (Store, error), names ...string) {
	for _, name := range names {
		if _, ok := backends[name]; ok {
			log.Fatalf("backend '%s' registered twice", name)
		}
		backends[name] = backend
	}
}

func Backends() []string {
	ret := make([]string, 0)
	for backendName := range backends {
		ret = append(ret, backendName)
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i] < ret[j]
	})
	return ret
}

func allowedInUri(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
		c == '+' || c == '-' || c == '.'
}

func New(storeConfig map[string]string) (Store, error) {
	location, ok := storeConfig["location"]
	if !ok {
		return nil, fmt.Errorf("missing location")
	}

	// extract the protocol
	proto := "fs"
	for i, c := range location {
		if !allowedInUri(c) {
			if i != 0 && strings.HasPrefix(location[i:], "://") {
				proto = location[:i]
			}
			break
		}
	}

	return NewStore(proto, storeConfig)
}

func Open(storeConfig map[string]string) (Store, []byte, error) {
	store, err := New(storeConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", flag.CommandLine.Name(), err)
		return nil, nil, err
	}

	serializedConfig, err := store.Open()
	if err != nil {
		return nil, nil, err
	}

	return store, serializedConfig, nil
}

func Create(storeConfig map[string]string, configuration []byte) (Store, error) {
	store, err := New(storeConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", flag.CommandLine.Name(), err)
		return nil, err
	}

	if err = store.Create(configuration); err != nil {
		return nil, err
	} else {
		return store, nil
	}
}
