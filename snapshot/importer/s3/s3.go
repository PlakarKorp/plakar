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

package s3

import (
	"context"
	"io"
	"io/fs"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/snapshot/importer"
)

type S3Importer struct {
	importer.ImporterBackend

	minioClient *minio.Client
	rootDir     string
	host        string
}

func init() {
	importer.Register("s3", NewS3Importer)
}

func connect(location *url.URL) (*minio.Client, error) {
	endpoint := location.Host
	accessKeyID := location.User.Username()
	secretAccessKey, _ := location.User.Password()
	useSSL := false

	// Initialize minio client object.
	return minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
}

func NewS3Importer(location string) (importer.ImporterBackend, error) {
	parsed, err := url.Parse(location)
	if err != nil {
		return nil, err
	}

	conn, err := connect(parsed)
	if err != nil {
		return nil, err
	}

	return &S3Importer{
		rootDir:     parsed.RequestURI()[1:],
		minioClient: conn,
		host:        parsed.Host,
	}, nil
}

func (p *S3Importer) Scan() (<-chan importer.ScanResult, error) {
	c := make(chan importer.ScanResult)

	go func() {
		directories := make(map[string]objects.FileInfo)
		files := make(map[string]objects.FileInfo)
		ino := uint64(0)
		fi := objects.NewFileInfo(
			"/",
			0,
			0700|fs.ModeDir,
			time.Now(),
			0,
			ino,
			0,
			0,
			1,
		)
		directories["/"] = fi
		ino++

		for object := range p.minioClient.ListObjects(context.Background(), p.rootDir, minio.ListObjectsOptions{Prefix: "", Recursive: true}) {
			atoms := strings.Split(object.Key, "/")

			for i := 0; i < len(atoms)-1; i++ {
				dir := strings.Join(atoms[0:i+1], "/")
				if _, exists := directories[dir]; !exists {
					fi := objects.NewFileInfo(
						atoms[i],
						0,
						0700|fs.ModeDir,
						time.Now(),
						0,
						ino,
						0,
						0,
						1,
					)
					directories["/"+dir] = fi
					ino++
				}
			}

			stat := objects.NewFileInfo(
				atoms[len(atoms)-1],
				object.Size,
				0700,
				object.LastModified,
				0,
				ino,
				0,
				0,
				1,
			)
			ino++
			files["/"+object.Key] = stat
		}

		directoryNames := make([]string, 0)
		for name := range directories {
			directoryNames = append(directoryNames, name)
		}

		fileNames := make([]string, 0)
		for name := range files {
			fileNames = append(fileNames, name)
		}

		sort.Slice(directoryNames, func(i, j int) bool {
			return len(directoryNames[i]) < len(directoryNames[j])
		})
		sort.Slice(fileNames, func(i, j int) bool {
			return len(fileNames[i]) < len(fileNames[j])
		})

		for _, directory := range directoryNames {
			c <- importer.ScanRecord{Type: importer.RecordTypeDirectory, Pathname: directory, FileInfo: directories[directory]}
		}
		for _, filename := range fileNames {
			c <- importer.ScanRecord{Type: importer.RecordTypeFile, Pathname: filename, FileInfo: files[filename]}
		}
		close(c)
	}()
	return c, nil
}

func (p *S3Importer) NewReader(pathname string) (io.ReadCloser, error) {
	obj, err := p.minioClient.GetObject(context.Background(), p.rootDir, pathname,
		minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (p *S3Importer) Close() error {
	return nil
}

func (p *S3Importer) Root() string {
	return p.rootDir
}

func (p *S3Importer) Origin() string {
	return p.host
}

func (p *S3Importer) Type() string {
	return "s3"
}
