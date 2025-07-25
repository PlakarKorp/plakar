/*
 * Copyright (c) 2025 Eric Faurot <eric@faurot.net>
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

package sftp

import (
	"fmt"
	"io"
	"path"

	"github.com/pkg/sftp"
)

func WriteToFileAtomic(sftpClient *sftp.Client, filename string, rd io.Reader) (int64, error) {
	return WriteToFileAtomicTempDir(sftpClient, filename, rd, path.Dir(filename))
}

func WriteToFileAtomicTempDir(sftpClient *sftp.Client, filename string, rd io.Reader, tmpdir string) (int64, error) {
	tmp := fmt.Sprintf("%s.tmp", filename)
	f, err := sftpClient.Create(tmp)
	if err != nil {
		return 0, err
	}

	var nbytes int64
	if nbytes, err = f.ReadFromWithConcurrency(rd, 0); err != nil {
		f.Close()
		sftpClient.Remove(f.Name())
		return 0, err
	}

	if err = f.Close(); err != nil {
		sftpClient.Remove(f.Name())
		return 0, err
	}

	err = sftpClient.Rename(f.Name(), filename)
	if err != nil {
		sftpClient.Remove(f.Name())
		return 0, err
	}

	return nbytes, nil
}
