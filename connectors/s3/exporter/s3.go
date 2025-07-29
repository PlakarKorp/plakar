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
	"strconv"
	"strings"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/snapshot/exporter"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Exporter struct {
	minioClient *minio.Client
	ctx         context.Context

	rootDir string
}

func init() {
	exporter.Register("s3", 0, NewS3Exporter)
}

func connect(location *url.URL, useSsl bool, accessKeyID, secretAccessKey string) (*minio.Client, error) {
	endpoint := location.Host

	// Initialize minio client object.
	return minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSsl,
	})
}

func NewS3Exporter(ctx context.Context, opts *exporter.Options, name string, config map[string]string) (exporter.Exporter, error) {
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

	err = conn.MakeBucket(ctx, strings.TrimPrefix(parsed.Path, "/"), minio.MakeBucketOptions{})
	if err != nil {
		if minio.ToErrorResponse(err).Code != "BucketAlreadyOwnedByYou" {
			return nil, err
		}
	}

	return &S3Exporter{
		rootDir:     parsed.Path,
		minioClient: conn,
		ctx:         ctx,
	}, nil
}

func (p *S3Exporter) Root(ctx context.Context) (string, error) {
	return p.rootDir, nil
}

func (p *S3Exporter) CreateDirectory(ctx context.Context, pathname string) error {
	return nil
}

func (p *S3Exporter) StoreFile(ctx context.Context, pathname string, fp io.Reader, size int64) error {
	_, err := p.minioClient.PutObject(p.ctx,
		strings.TrimPrefix(p.rootDir, "/"),
		strings.TrimPrefix(pathname, p.rootDir+"/"),
		fp, size, minio.PutObjectOptions{})
	return err
}

func (p *S3Exporter) SetPermissions(ctx context.Context, pathname string, fileinfo *objects.FileInfo) error {
	return nil
}

func (p *S3Exporter) Close(ctx context.Context) error {
	return nil
}
