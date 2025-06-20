package ptar

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type ReadWriteSeekStatReadAtCloser interface {
	io.Reader   // Read(p []byte) (n int, err error)
	io.Seeker   // Seek(offset int64, whence int) (int64, error)
	io.Closer   // Close() error
	io.ReaderAt // ReadAt(p []byte, off int64) (n int, err error)
	io.Writer   // Write(p []byte) (n int, err error)
	Stat() (os.FileInfo, error)
}

type HTTPReader struct {
	client *http.Client
	url    string
	offset int64
	size   int64
}

func NewHTTPReader(url string) (*HTTPReader, error) {
	var resp *http.Response
	var err error

	resp, err = http.Head(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("could not open ptar: %s", resp.Status)
	}

	contentLength, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	if err != nil {
		return nil, err
	}

	hr := HTTPReader{
		client: &http.Client{},
		url:    url,
		offset: 0,
		size:   int64(contentLength),
	}
	return &hr, nil
}

func (hr *HTTPReader) Read(buf []byte) (int, error) {
	req, err := http.NewRequest("GET", hr.url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", hr.offset, hr.offset+int64(len(buf))))
	resp, err := hr.client.Do(req)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return 0, fmt.Errorf("NOT OK")
	}

	n, err := resp.Body.Read(buf)
	hr.offset += int64(n)
	return n, err
}

func (hr *HTTPReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		if offset >= hr.size {
			return 0, io.EOF
		}
		hr.offset = offset
	case io.SeekCurrent:
		if hr.offset+offset >= hr.size {
			return 0, io.EOF
		}
		hr.offset += offset
	case io.SeekEnd:
		if offset > hr.size {
			return 0, io.EOF
		}
		hr.offset = hr.size + offset
	}
	return hr.offset, nil
}

func (hr *HTTPReader) ReadAt(buf []byte, off int64) (int, error) {
	if off >= hr.size {
		return 0, io.EOF
	}

	end := off + int64(len(buf)) - 1
	if end >= hr.size {
		end = hr.size - 1
	}

	req, err := http.NewRequest("GET", hr.url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", off, end))

	resp, err := hr.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return 0, fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	n, err := io.ReadFull(resp.Body, buf[:end-off+1])
	if err != nil && err != io.ErrUnexpectedEOF {
		return n, err
	}
	return n, nil
}

type dummyFileInfo struct {
	name string
	size int64
}

func (d *dummyFileInfo) Name() string       { return "http_reader" }
func (d *dummyFileInfo) Size() int64        { return d.size }
func (d *dummyFileInfo) Mode() os.FileMode  { return 0644 }
func (d *dummyFileInfo) ModTime() time.Time { return time.Time{} }
func (d *dummyFileInfo) IsDir() bool        { return false }
func (d *dummyFileInfo) Sys() interface{}   { return nil }

func (hr *HTTPReader) Stat() (os.FileInfo, error) {
	// Since HTTP does not provide file info, we can return a dummy FileInfo
	return &dummyFileInfo{
		size: hr.size,
	}, nil
}

func (hr *HTTPReader) Write(p []byte) (int, error) {
	return 0, fmt.Errorf("HTTPReader does not support Write")
}

func (hr *HTTPReader) Close() error {
	return nil
}
