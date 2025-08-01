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
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/snapshot/importer"
)

type S3Importer struct {
	minioClient *minio.Client
	ctx         context.Context

	bucket  string
	host    string
	scanDir string

	ino uint64
}

func init() {
	importer.Register("s3", 0, NewS3Importer)
}

func connect(location *url.URL, useSsl bool, accessKeyID, secretAccessKey string) (*minio.Client, error) {
	endpoint := location.Host

	// Initialize minio client object.
	return minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSsl,
	})
}

func NewS3Importer(ctx context.Context, opts *importer.Options, name string, config map[string]string) (importer.Importer, error) {
	target := config["location"]

	var accessKey string
	if tmp, ok := config["access_key"]; !ok {
		return nil, fmt.Errorf("missing access_key")
	} else {
		accessKey = tmp
	}

	var secretAccessKey string
	if tmp, ok := config["secret_access_key"]; !ok {
		return nil, fmt.Errorf("missing secret_access_key")
	} else {
		secretAccessKey = tmp
	}

	useSsl := true
	if value, ok := config["use_tls"]; ok {
		tmp, err := strconv.ParseBool(value)
		if err != nil {
			return nil, fmt.Errorf("invalid use_tls value")
		}
		useSsl = tmp
	}

	parsed, err := url.Parse(target)
	if err != nil {
		return nil, err
	}

	conn, err := connect(parsed, useSsl, accessKey, secretAccessKey)
	if err != nil {
		return nil, err
	}

	atoms := strings.Split(parsed.RequestURI()[1:], "/")
	bucket := atoms[0]
	scanDir := path.Clean("/" + strings.Join(atoms[1:], "/"))

	return &S3Importer{
		bucket:      bucket,
		scanDir:     scanDir,
		minioClient: conn,
		host:        parsed.Host,
		ctx:         ctx,
	}, nil
}

func (p *S3Importer) Scan(ctx context.Context) (<-chan *importer.ScanResult, error) {
	result := make(chan *importer.ScanResult)
	go func() {
		defer close(result)

		// Create scandir entries.
		parent := p.scanDir
		for {
			fi := objects.NewFileInfo(
				path.Base(parent),
				0,
				0700|os.ModeDir,
				time.Unix(0, 0),
				0,
				0,
				0,
				0,
				0,
			)
			result <- importer.NewScanRecord(parent, "", fi, nil, nil)

			if parent == "/" {
				break
			}
			parent = path.Dir(parent)
		}

		prefix := strings.TrimPrefix(p.scanDir, "/")

		for object := range p.minioClient.ListObjects(p.ctx, p.bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: true}) {
			// Create a record for each of the parent directories of the object.
			// Two objects in a same directory will generate the same records for this directory, but the backup layer ignores duplicates.
			parent := path.Dir("/" + object.Key)
			for {
				// p.scanDir directories have already been created above.
				if parent == "/"+prefix {
					break
				}

				fi := objects.NewFileInfo(
					path.Base(parent),
					0,
					0700|os.ModeDir,
					time.Unix(0, 0),
					0,
					0,
					0,
					0,
					0,
				)
				result <- importer.NewScanRecord("/"+parent, "", fi, nil, nil)
				parent = path.Dir(parent)
			}

			fi := objects.NewFileInfo(
				path.Base("/"+object.Key),
				object.Size,
				0700,
				object.LastModified,
				1,
				0,
				0,
				0,
				0,
			)
			result <- importer.NewScanRecord("/"+object.Key, "", fi, nil, func() (io.ReadCloser, error) {
				return p.minioClient.GetObject(p.ctx, p.bucket, object.Key, minio.GetObjectOptions{})
			})
		}

	}()
	return result, nil
}

func (p *S3Importer) Close(ctx context.Context) error {
	return nil
}

func (p *S3Importer) Root(ctx context.Context) (string, error) {
	return p.scanDir, nil
}

func (p *S3Importer) Origin(ctx context.Context) (string, error) {
	return p.host + "/" + p.bucket, nil
}

func (p *S3Importer) Type(ctx context.Context) (string, error) {
	return "s3", nil
}
