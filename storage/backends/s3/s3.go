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

package s3

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/storage"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Store struct {
	location    string
	Repository  string
	minioClient *minio.Client
	bucketName  string

	useSsl          bool
	accessKey       string
	secretAccessKey string

	storageClass string

	putObjectOptions minio.PutObjectOptions
}

func init() {
	storage.Register(NewStore, "s3")
}

func NewStore(storeConfig map[string]string) (storage.Store, error) {
	var accessKey string
	if value, ok := storeConfig["access_key"]; !ok {
		return nil, fmt.Errorf("missing access_key")
	} else {
		accessKey = value
	}

	var secretAccessKey string
	if value, ok := storeConfig["secret_access_key"]; !ok {
		return nil, fmt.Errorf("missing secret_access_key")
	} else {
		secretAccessKey = value
	}

	useSsl := true
	if value, ok := storeConfig["use_tls"]; ok {
		tmp, err := strconv.ParseBool(value)
		if err != nil {
			return nil, fmt.Errorf("invalid use_tls value")
		}
		useSsl = tmp
	}

	storageClass := "standard"
	if value, ok := storeConfig["storage_class"]; ok {
		storageClass = strings.ToUpper(value)
		if storageClass != "STANDARD" && storageClass != "REDUCED_REDUNDANCY" && storageClass != "STANDARD_IA" && storageClass != "ONEZONE_IA" && storageClass != "INTELLIGENT_TIERING" && storageClass != "GLACIER" && storageClass != "GLACIER_IR" && storageClass != "DEEP_ARCHIVE" {
			return nil, fmt.Errorf("invalid storage_class value")
		}
	}

	return &Store{
		location:        storeConfig["location"],
		accessKey:       accessKey,
		secretAccessKey: secretAccessKey,
		useSsl:          useSsl,
		storageClass:    storageClass,

		putObjectOptions: minio.PutObjectOptions{
			// Some providers (eg. BlackBlaze) return the error
			// "Unsupported header 'x-amz-checksum-algorithm'" if SendContentMd5
			// is not set.
			StorageClass:   storageClass,
			SendContentMd5: true,
		},
	}, nil
}

func (s *Store) Location() string {
	return s.location
}

func (s *Store) connect(location *url.URL) error {
	endpoint := location.Host
	useSSL := s.useSsl

	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s.accessKey, s.secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return fmt.Errorf("create minio client: %w", err)
	}

	s.minioClient = minioClient
	return nil
}

func (s *Store) Create(config []byte) error {
	parsed, err := url.Parse(s.location)
	if err != nil {
		return fmt.Errorf("parse location: %w", err)
	}

	err = s.connect(parsed)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	s.bucketName = parsed.RequestURI()[1:]

	exists, err := s.minioClient.BucketExists(context.Background(), s.bucketName)
	if err != nil {
		return fmt.Errorf("check if bucket exists: %w", err)
	}
	if !exists {
		err = s.minioClient.MakeBucket(context.Background(), s.bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("make bucket: %w", err)
		}
	}

	_, err = s.minioClient.StatObject(context.Background(), s.bucketName, "CONFIG", minio.StatObjectOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code != "NoSuchKey" {
			return fmt.Errorf("stat object CONFIG: %w", err)
		}
	} else {
		return fmt.Errorf("bucket already initialized")
	}

	if s.Mode()&storage.ModeRead == 0 {
		_, err = s.minioClient.PutObject(context.Background(), s.bucketName, "CONFIG.frozen", bytes.NewReader(config), int64(len(config)), s.putObjectOptions)
		if err != nil {
			return fmt.Errorf("put object CONFIG.frozen: %w", err)
		}
	}

	putObjectOptions := s.putObjectOptions
	putObjectOptions.StorageClass = "STANDARD"

	_, err = s.minioClient.PutObject(context.Background(), s.bucketName, "CONFIG", bytes.NewReader(config), int64(len(config)), putObjectOptions)
	if err != nil {
		return fmt.Errorf("put object CONFIG: %w", err)
	}

	return nil
}

func (s *Store) Open() ([]byte, error) {
	parsed, err := url.Parse(s.location)
	if err != nil {
		return nil, fmt.Errorf("parse location: %w", err)
	}

	err = s.connect(parsed)
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}

	s.bucketName = parsed.RequestURI()[1:]

	exists, err := s.minioClient.BucketExists(context.Background(), s.bucketName)
	if err != nil {
		return nil, fmt.Errorf("error checking if bucket exists: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("bucket does not exist")
	}

	object, err := s.minioClient.GetObject(context.Background(), s.bucketName, "CONFIG", minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting object: %w", err)
	}
	stat, err := object.Stat()
	if err != nil {
		return nil, fmt.Errorf("error getting object stat: %w", err)
	}

	data := make([]byte, stat.Size)
	_, err = object.Read(data)
	if err != nil {
		if err != io.EOF {
			return nil, fmt.Errorf("error reading object: %w", err)
		}
	}
	object.Close()

	return data, nil
}

func (s *Store) Close() error {
	return nil
}

func (s *Store) Mode() storage.Mode {
	if s.storageClass == "GLACIER" || s.storageClass == "DEEP_ARCHIVE" {
		return storage.ModeWrite
	}
	return storage.ModeRead | storage.ModeWrite
}

// states
func (s *Store) GetStates() ([]objects.MAC, error) {
	ret := make([]objects.MAC, 0)
	for object := range s.minioClient.ListObjects(context.Background(), s.bucketName, minio.ListObjectsOptions{
		Prefix:    "states/",
		Recursive: true,
	}) {
		if strings.HasPrefix(object.Key, "states/") && len(object.Key) >= 10 {
			t, err := hex.DecodeString(object.Key[10:])
			if err != nil {
				return nil, fmt.Errorf("decode state key: %w", err)
			}
			if len(t) != 32 {
				continue
			}
			var t32 objects.MAC
			copy(t32[:], t)
			ret = append(ret, t32)
		}
	}
	return ret, nil
}

func (s *Store) PutState(mac objects.MAC, rd io.Reader) error {
	_, err := s.minioClient.PutObject(context.Background(), s.bucketName, fmt.Sprintf("states/%02x/%016x", mac[0], mac), rd, -1, s.putObjectOptions)
	if err != nil {
		return fmt.Errorf("put object: %w", err)
	}

	return nil
}

func (s *Store) GetState(mac objects.MAC) (io.Reader, error) {
	object, err := s.minioClient.GetObject(context.Background(), s.bucketName, fmt.Sprintf("states/%02x/%016x", mac[0], mac), minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object: %w", err)
	}

	return object, nil
}

func (s *Store) DeleteState(mac objects.MAC) error {
	err := s.minioClient.RemoveObject(context.Background(), s.bucketName, fmt.Sprintf("states/%02x/%016x", mac[0], mac), minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("remove object: %w", err)
	}
	return nil
}

// packfiles
func (s *Store) GetPackfiles() ([]objects.MAC, error) {
	ret := make([]objects.MAC, 0)
	for object := range s.minioClient.ListObjects(context.Background(), s.bucketName, minio.ListObjectsOptions{
		Prefix:    "packfiles/",
		Recursive: true,
	}) {
		if strings.HasPrefix(object.Key, "packfiles/") && len(object.Key) >= 13 {
			t, err := hex.DecodeString(object.Key[13:])
			if err != nil {
				return nil, fmt.Errorf("decode packfile key: %w", err)
			}
			if len(t) != 32 {
				continue
			}
			var t32 objects.MAC
			copy(t32[:], t)
			ret = append(ret, t32)
		}
	}
	return ret, nil
}

func (s *Store) PutPackfile(mac objects.MAC, rd io.Reader) error {
	_, err := s.minioClient.PutObject(context.Background(), s.bucketName, fmt.Sprintf("packfiles/%02x/%016x", mac[0], mac), rd, -1, s.putObjectOptions)
	if err != nil {
		return fmt.Errorf("put object: %w", err)
	}
	return nil
}

func (s *Store) GetPackfile(mac objects.MAC) (io.Reader, error) {
	object, err := s.minioClient.GetObject(context.Background(), s.bucketName, fmt.Sprintf("packfiles/%02x/%016x", mac[0], mac), minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object: %w", err)
	}
	return object, nil
}

func (s *Store) GetPackfileBlob(mac objects.MAC, offset uint64, length uint32) (io.Reader, error) {
	opts := minio.GetObjectOptions{}
	object, err := s.minioClient.GetObject(context.Background(), s.bucketName, fmt.Sprintf("packfiles/%02x/%016x", mac[0], mac), opts)
	if err != nil {
		return nil, fmt.Errorf("get object: %w", err)
	}

	buffer := make([]byte, length)
	if nbytes, err := object.ReadAt(buffer, int64(offset)); err != nil {
		return nil, fmt.Errorf("read object: %w", err)
	} else if nbytes != int(length) {
		return nil, fmt.Errorf("short read")
	}

	return bytes.NewBuffer(buffer), nil
}

func (s *Store) DeletePackfile(mac objects.MAC) error {
	err := s.minioClient.RemoveObject(context.Background(), s.bucketName, fmt.Sprintf("packfiles/%02x/%016x", mac[0], mac), minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("remove object: %w", err)
	}
	return nil
}

func (s *Store) GetLocks() ([]objects.MAC, error) {
	ret := make([]objects.MAC, 0)
	for object := range s.minioClient.ListObjects(context.Background(), s.bucketName, minio.ListObjectsOptions{
		Prefix:    "locks/",
		Recursive: true,
	}) {
		if strings.HasPrefix(object.Key, "locks/") && len(object.Key) >= 6 {
			t, err := hex.DecodeString(object.Key[6:])
			if err != nil {
				return nil, fmt.Errorf("decode lock key: %w", err)
			}
			if len(t) != 32 {
				continue
			}
			ret = append(ret, objects.MAC(t))
		}
	}

	return ret, nil
}

func (s *Store) PutLock(lockID objects.MAC, rd io.Reader) error {
	putObjectOptions := s.putObjectOptions
	putObjectOptions.StorageClass = "STANDARD"

	_, err := s.minioClient.PutObject(context.Background(), s.bucketName, fmt.Sprintf("locks/%016x", lockID), rd, -1, putObjectOptions)
	if err != nil {
		return fmt.Errorf("put object: %w", err)
	}
	return nil
}

func (s *Store) GetLock(lockID objects.MAC) (io.Reader, error) {
	object, err := s.minioClient.GetObject(context.Background(), s.bucketName, fmt.Sprintf("locks/%016x", lockID), minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object: %w", err)
	}
	return object, nil
}

func (s *Store) DeleteLock(lockID objects.MAC) error {
	err := s.minioClient.RemoveObject(context.Background(), s.bucketName, fmt.Sprintf("locks/%016x", lockID), minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("remove object: %w", err)
	}
	return nil
}
