package packfile

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash"
	"io"
	"time"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/versioning"
)

const VERSION = "1.0.0"

type Blob struct {
	Type    resources.Type
	Version versioning.Version
	MAC     objects.MAC
	Offset  uint64
	Length  uint32
	Flags   uint32
}

const BLOB_RECORD_SIZE = 56

type PackFile struct {
	hasher hash.Hash
	Blobs  []byte
	Index  []Blob
	Footer PackFileFooter
}

type PackFileFooter struct {
	Version     versioning.Version `msgpack:"-"`
	Timestamp   int64
	Count       uint32
	IndexOffset uint64
	IndexMAC    objects.MAC
	Flags       uint32
}

const FOOTER_SIZE = 56

type Configuration struct {
	MinSize uint64
	AvgSize uint64
	MaxSize uint64
}

func NewDefaultConfiguration() *Configuration {
	return &Configuration{
		MaxSize: (20 << 10) << 10,
	}
}

func NewFooterFromBytes(version versioning.Version, serialized []byte) (PackFileFooter, error) {
	var footer PackFileFooter

	reader := bytes.NewReader(serialized)
	footer.Version = version
	if err := binary.Read(reader, binary.LittleEndian, &footer.Timestamp); err != nil {
		return footer, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &footer.Count); err != nil {
		return footer, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &footer.IndexOffset); err != nil {
		return footer, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &footer.IndexMAC); err != nil {
		return footer, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &footer.Flags); err != nil {
		return footer, err
	}
	return footer, nil
}

func NewIndexFromBytes(version versioning.Version, serialized []byte) ([]Blob, error) {
	reader := bytes.NewReader(serialized)
	index := make([]Blob, 0)
	for reader.Len() > 0 {
		var resourceType resources.Type
		var resourceVersion versioning.Version
		var mac objects.MAC
		var blobOffset uint64
		var blobLength uint32
		var blobFlags uint32

		if err := binary.Read(reader, binary.LittleEndian, &resourceType); err != nil {
			return nil, err
		}
		if err := binary.Read(reader, binary.LittleEndian, &resourceVersion); err != nil {
			return nil, err
		}
		if err := binary.Read(reader, binary.LittleEndian, &mac); err != nil {
			return nil, err
		}
		if err := binary.Read(reader, binary.LittleEndian, &blobOffset); err != nil {
			return nil, err
		}
		if err := binary.Read(reader, binary.LittleEndian, &blobLength); err != nil {
			return nil, err
		}
		if err := binary.Read(reader, binary.LittleEndian, &blobFlags); err != nil {
			return nil, err
		}
		index = append(index, Blob{
			Type:    resourceType,
			Version: resourceVersion,
			MAC:     mac,
			Offset:  blobOffset,
			Length:  blobLength,
			Flags:   blobFlags,
		})
	}
	return index, nil
}

func New(hasher hash.Hash) *PackFile {
	return &PackFile{
		hasher: hasher,
		Blobs:  make([]byte, 0),
		Index:  make([]Blob, 0),
		Footer: PackFileFooter{
			Version:   versioning.FromString(VERSION),
			Timestamp: time.Now().UnixNano(),
			Count:     0,
		},
	}
}

func NewFromBytes(hasher hash.Hash, version versioning.Version, serialized []byte) (*PackFile, error) {
	reader := bytes.NewReader(serialized)
	var footer PackFileFooter
	_, err := reader.Seek(-FOOTER_SIZE, io.SeekEnd)
	if err != nil {
		return nil, err
	}

	footer.Version = version

	if err := binary.Read(reader, binary.LittleEndian, &footer.Timestamp); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &footer.Count); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &footer.IndexOffset); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &footer.IndexMAC); err != nil {
		return nil, err
	}
	if err := binary.Read(reader, binary.LittleEndian, &footer.Flags); err != nil {
		return nil, err
	}

	_, err = reader.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}
	data := make([]byte, footer.IndexOffset)
	if err := binary.Read(reader, binary.LittleEndian, data); err != nil {
		return nil, err
	}

	// we won't read the totalLength again
	remaining := reader.Len() - FOOTER_SIZE

	p := New(hasher)
	p.Footer = footer
	p.Blobs = data
	p.hasher.Reset()
	for remaining > 0 {
		var resourceType resources.Type
		var resourceVersion versioning.Version
		var mac objects.MAC
		var blobOffset uint64
		var blobLength uint32
		var blobFlags uint32

		if err := binary.Read(reader, binary.LittleEndian, &resourceType); err != nil {
			return nil, err
		}
		if err := binary.Read(reader, binary.LittleEndian, &resourceVersion); err != nil {
			return nil, err
		}
		if err := binary.Read(reader, binary.LittleEndian, &mac); err != nil {
			return nil, err
		}
		if err := binary.Read(reader, binary.LittleEndian, &blobOffset); err != nil {
			return nil, err
		}
		if err := binary.Read(reader, binary.LittleEndian, &blobLength); err != nil {
			return nil, err
		}
		if err := binary.Read(reader, binary.LittleEndian, &blobFlags); err != nil {
			return nil, err
		}

		if blobOffset+uint64(blobLength) > p.Footer.IndexOffset {
			return nil, fmt.Errorf("blob offset + blob length exceeds total length of packfile")
		}

		if err := binary.Write(p.hasher, binary.LittleEndian, resourceType); err != nil {
			return nil, err
		}
		if err := binary.Write(p.hasher, binary.LittleEndian, resourceVersion); err != nil {
			return nil, err
		}
		if err := binary.Write(p.hasher, binary.LittleEndian, mac); err != nil {
			return nil, err
		}
		if err := binary.Write(p.hasher, binary.LittleEndian, blobOffset); err != nil {
			return nil, err
		}
		if err := binary.Write(p.hasher, binary.LittleEndian, blobLength); err != nil {
			return nil, err
		}
		if err := binary.Write(p.hasher, binary.LittleEndian, blobFlags); err != nil {
			return nil, err
		}
		p.Index = append(p.Index, Blob{
			Type:    resourceType,
			Version: resourceVersion,
			MAC:     mac,
			Offset:  blobOffset,
			Length:  blobLength,
			Flags:   blobFlags,
		})
		remaining -= BLOB_RECORD_SIZE
	}
	mac := objects.MAC(p.hasher.Sum(nil))
	if mac != p.Footer.IndexMAC {
		return nil, fmt.Errorf("index mac mismatch")
	}

	return p, nil
}

func (p *PackFile) Serialize() ([]byte, error) {
	var buffer bytes.Buffer
	if err := binary.Write(&buffer, binary.LittleEndian, p.Blobs); err != nil {
		return nil, err
	}

	p.hasher.Reset()
	for _, blob := range p.Index {
		if err := binary.Write(&buffer, binary.LittleEndian, blob.Type); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, blob.Version); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, blob.MAC); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, blob.Offset); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, blob.Length); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, blob.Flags); err != nil {
			return nil, err
		}

		if err := binary.Write(p.hasher, binary.LittleEndian, blob.Type); err != nil {
			return nil, err
		}
		if err := binary.Write(p.hasher, binary.LittleEndian, blob.Version); err != nil {
			return nil, err
		}
		if err := binary.Write(p.hasher, binary.LittleEndian, blob.MAC); err != nil {
			return nil, err
		}
		if err := binary.Write(p.hasher, binary.LittleEndian, blob.Offset); err != nil {
			return nil, err
		}
		if err := binary.Write(p.hasher, binary.LittleEndian, blob.Length); err != nil {
			return nil, err
		}
		if err := binary.Write(p.hasher, binary.LittleEndian, blob.Flags); err != nil {
			return nil, err
		}
	}
	p.Footer.IndexMAC = objects.MAC(p.hasher.Sum(nil))

	if err := binary.Write(&buffer, binary.LittleEndian, p.Footer.Timestamp); err != nil {
		return nil, err
	}
	if err := binary.Write(&buffer, binary.LittleEndian, p.Footer.Count); err != nil {
		return nil, err
	}
	if err := binary.Write(&buffer, binary.LittleEndian, p.Footer.IndexOffset); err != nil {
		return nil, err
	}
	if err := binary.Write(&buffer, binary.LittleEndian, p.Footer.IndexMAC); err != nil {
		return nil, err
	}
	if err := binary.Write(&buffer, binary.LittleEndian, p.Footer.Flags); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (p *PackFile) SerializeData() ([]byte, error) {
	var buffer bytes.Buffer
	if err := binary.Write(&buffer, binary.LittleEndian, p.Blobs); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func (p *PackFile) SerializeIndex() ([]byte, error) {
	var buffer bytes.Buffer
	p.hasher.Reset()
	for _, blob := range p.Index {
		if err := binary.Write(&buffer, binary.LittleEndian, blob.Type); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, blob.Version); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, blob.MAC); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, blob.Offset); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, blob.Length); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, blob.Flags); err != nil {
			return nil, err
		}

		if err := binary.Write(p.hasher, binary.LittleEndian, blob.Type); err != nil {
			return nil, err
		}
		if err := binary.Write(p.hasher, binary.LittleEndian, blob.Version); err != nil {
			return nil, err
		}
		if err := binary.Write(p.hasher, binary.LittleEndian, blob.MAC); err != nil {
			return nil, err
		}
		if err := binary.Write(p.hasher, binary.LittleEndian, blob.Offset); err != nil {
			return nil, err
		}
		if err := binary.Write(p.hasher, binary.LittleEndian, blob.Length); err != nil {
			return nil, err
		}
		if err := binary.Write(p.hasher, binary.LittleEndian, blob.Flags); err != nil {
			return nil, err
		}
	}
	return buffer.Bytes(), nil
}

func (p *PackFile) SerializeFooter() ([]byte, error) {
	var buffer bytes.Buffer
	p.hasher.Reset()
	for _, blob := range p.Index {
		if err := binary.Write(&buffer, binary.LittleEndian, blob.Type); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, blob.Version); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, blob.MAC); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, blob.Offset); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, blob.Length); err != nil {
			return nil, err
		}
		if err := binary.Write(&buffer, binary.LittleEndian, blob.Flags); err != nil {
			return nil, err
		}

		if err := binary.Write(p.hasher, binary.LittleEndian, blob.Type); err != nil {
			return nil, err
		}
		if err := binary.Write(p.hasher, binary.LittleEndian, blob.Version); err != nil {
			return nil, err
		}
		if err := binary.Write(p.hasher, binary.LittleEndian, blob.MAC); err != nil {
			return nil, err
		}
		if err := binary.Write(p.hasher, binary.LittleEndian, blob.Offset); err != nil {
			return nil, err
		}
		if err := binary.Write(p.hasher, binary.LittleEndian, blob.Length); err != nil {
			return nil, err
		}
		if err := binary.Write(p.hasher, binary.LittleEndian, blob.Flags); err != nil {
			return nil, err
		}
	}
	p.Footer.IndexMAC = objects.MAC(p.hasher.Sum(nil))

	buffer.Reset()
	if err := binary.Write(&buffer, binary.LittleEndian, p.Footer.Timestamp); err != nil {
		return nil, err
	}
	if err := binary.Write(&buffer, binary.LittleEndian, p.Footer.Count); err != nil {
		return nil, err
	}
	if err := binary.Write(&buffer, binary.LittleEndian, p.Footer.IndexOffset); err != nil {
		return nil, err
	}
	if err := binary.Write(&buffer, binary.LittleEndian, p.Footer.IndexMAC); err != nil {
		return nil, err
	}
	if err := binary.Write(&buffer, binary.LittleEndian, p.Footer.Flags); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func (p *PackFile) AddBlob(resourceType resources.Type, version versioning.Version, mac objects.MAC, data []byte, flags uint32) {
	p.Index = append(p.Index, Blob{
		Type:    resourceType,
		Version: version,
		MAC:     mac,
		Offset:  uint64(len(p.Blobs)),
		Length:  uint32(len(data)),
		Flags:   flags,
	})
	p.Blobs = append(p.Blobs, data...)
	p.Footer.Count++
	p.Footer.IndexOffset = uint64(len(p.Blobs))
}

func (p *PackFile) GetBlob(mac objects.MAC) ([]byte, bool) {
	for _, blob := range p.Index {
		if blob.MAC == mac {
			return p.Blobs[blob.Offset : blob.Offset+uint64(blob.Length)], true
		}
	}
	return nil, false
}

func (p *PackFile) Size() uint32 {
	return uint32(len(p.Blobs))
}
