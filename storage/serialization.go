package storage

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash"
	"io"

	"github.com/PlakarKorp/plakar/hashing"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/versioning"
)

const (
	DEFAULT_HASHING_ALGORITHM        = hashing.DEFAULT_HASHING_ALGORITHM
	STORAGE_HEADER_SIZE       uint32 = 16
	STORAGE_FOOTER_SIZE       uint32 = 32
)

const (
	PHASE_HEADER = iota
	PHASE_DATA
	PHASE_FOOTER
)

type deserializeReader struct {
	inner    io.Reader
	leftOver []byte

	hasher hash.Hash
	hmac   [32]byte

	eof bool
}

func newDeserializeReader(hasher hash.Hash, resourceType resources.Type, inner io.Reader) (versioning.Version, *deserializeReader, error) {
	buf := make([]byte, STORAGE_HEADER_SIZE)
	_, err := io.ReadFull(inner, buf)
	if err != nil {
		return versioning.Version(0), nil, err
	}

	magic := buf[0:8]
	if !bytes.Equal(magic, []byte("_PLAKAR_")) {
		return versioning.Version(0), nil, fmt.Errorf("invalid plakar magic: %s", magic)
	}
	parsedResourceType := resources.Type(binary.LittleEndian.Uint32(buf[8:12]))
	parsedResourceVersion := versioning.Version(binary.LittleEndian.Uint32(buf[12:16]))

	if parsedResourceType != resourceType {
		return versioning.Version(0), nil, fmt.Errorf("invalid resource type")
	}

	hasher.Reset()
	hasher.Write(buf)

	return parsedResourceVersion, &deserializeReader{
		inner:  inner,
		hasher: hasher,
		hmac:   [32]byte{},
	}, nil
}

func (s *deserializeReader) Read(p []byte) (int, error) {
	total := 0

	for total < len(p) {
		// 1. If not at EOF, read a chunk from the inner reader.
		if !s.eof {
			const chunkSize = 4096
			buf := make([]byte, chunkSize)
			n, err := s.inner.Read(buf)
			if n > 0 {
				s.leftOver = append(s.leftOver, buf[:n]...)
			}
			if err != nil {
				if err == io.EOF {
					s.eof = true
				} else {
					return total, err
				}
			}
		}

		flushable := len(s.leftOver) - 32
		if flushable < 0 {
			flushable = 0
		}

		avail := len(p) - total
		if flushable > 0 {
			nFlush := flushable
			if nFlush > avail {
				nFlush = avail
			}
			copy(p[total:total+nFlush], s.leftOver[:nFlush])
			s.hasher.Write(s.leftOver[:nFlush])
			total += nFlush
			s.leftOver = s.leftOver[nFlush:]
		}

		if s.eof && len(s.leftOver) == 32 {
			copy(s.hmac[:], s.leftOver)
			if !bytes.Equal(s.hmac[:], s.hasher.Sum(nil)) {
				return 0, fmt.Errorf("hmac mismatch")
			}
			return total, io.EOF
		}

		if flushable == 0 && !s.eof {
			continue
		}

		if total == len(p) {
			break
		}
	}

	return total, nil
}

func Deserialize(hasher hash.Hash, resourceType resources.Type, input io.Reader) (versioning.Version, io.Reader, error) {
	return newDeserializeReader(hasher, resourceType, input)
}

type serializeReader struct {
	header       []byte
	headerOffset int

	inner io.Reader

	hasher     hash.Hash
	hmac       [32]byte
	hmacOffset int

	phase int
}

func newSerializeReader(hasher hash.Hash, resourceType resources.Type, version versioning.Version, inner io.Reader) *serializeReader {
	header := make([]byte, STORAGE_HEADER_SIZE)
	copy(header[0:8], []byte("_PLAKAR_"))
	binary.LittleEndian.PutUint32(header[8:12], uint32(resourceType))
	binary.LittleEndian.PutUint32(header[12:16], uint32(version))

	hasher.Reset()
	hasher.Write(header)

	return &serializeReader{
		header: header,
		inner:  inner,
		hasher: hasher,
		phase:  PHASE_HEADER,
		hmac:   [32]byte{},
	}
}

func (s *serializeReader) Read(p []byte) (n int, err error) {
	total := 0

	for total < len(p) {
		switch s.phase {
		case PHASE_HEADER:
			if s.headerOffset < len(s.header) {
				n = copy(p[total:], s.header[s.headerOffset:])
				s.headerOffset += n
				total += n

				// we filled p, let's do another round !
				if total == len(p) {
					return total, nil
				}
			}
			if s.headerOffset == len(s.header) {
				s.phase = PHASE_DATA
			}

		case PHASE_DATA:
			n, err := s.inner.Read(p[total:])

			// can return err WITH data because EOF
			if n > 0 {
				s.hasher.Write(p[total : total+n])
				total += n
			}

			// if error and not EOF, we return err
			// if EOF, we move to the footer phase
			// if no error we return partial buffer
			if err != nil && err != io.EOF {
				return total, err
			} else {
				if err == io.EOF {
					s.phase = PHASE_FOOTER
					copy(s.hmac[:32], s.hasher.Sum(nil))
					continue
				}
				return total, nil
			}

		case PHASE_FOOTER:
			if s.hmacOffset < len(s.hmac) {
				n = copy(p[total:], s.hmac[s.hmacOffset:])
				s.hmacOffset += n
				total += n
			}
			if s.hmacOffset == len(s.hmac) {
				return total, io.EOF
			} else {
				return total, nil
			}

		default:
			panic("invalid phase, logic error")
		}
	}

	return total, nil
}

func Serialize(hasher hash.Hash, resourceType resources.Type, version versioning.Version, input io.Reader) (io.Reader, error) {
	return newSerializeReader(hasher, resourceType, version, input), nil
}
