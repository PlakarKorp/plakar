package packer

import (
	"encoding/binary"
	"hash"
	"io"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/resources"
	"github.com/PlakarKorp/plakar/versioning"
)

const VERSION = "1.0.0"

func init() {
	versioning.Register(resources.RT_PACKFILE, versioning.FromString(VERSION))
}

type Blob struct {
	Type    resources.Type
	Version versioning.Version
	MAC     objects.MAC
	Offset  uint64
	Length  uint32
	Flags   uint32
}

const BLOB_RECORD_SIZE = 56

type PackWriter struct {
	hasher hash.Hash

	Index         []Blob
	writer        io.WriteCloser
	currentOffset uint64

	Footer PackFooter

	pipesync chan struct{}
}

type PackFooter struct {
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

func NewPackWriter(putter func(io.Reader) error, hasher hash.Hash) *PackWriter {
	pipesync := make(chan struct{}, 1)
	pr, pw := io.Pipe()

	// packfilewriter -> pw -> pipe -> pr <- putter (io.ReadAll())
	go func() {
		defer pr.Close()
		defer func() { close(pipesync) }()
		if err := putter(pr); err != nil {
			pr.CloseWithError(err)
		}
	}()

	return &PackWriter{
		hasher:   hasher,
		Index:    make([]Blob, 0), // temporary
		writer:   pw,
		pipesync: pipesync,
	}
}

func (pwr *PackWriter) WriteBlob(resourceType resources.Type, version versioning.Version, mac objects.MAC, data []byte, flags uint32) error {
	pwr.Index = append(pwr.Index, Blob{
		Type:    resourceType,
		Version: version,
		MAC:     mac,
		Offset:  uint64(pwr.currentOffset),
		Length:  uint32(len(data)),
		Flags:   flags,
	})

	nbytes, err := pwr.writer.Write(data)
	if err != nil {
		return err
	}
	if nbytes != len(data) {
		return err
	}

	pwr.currentOffset += uint64(nbytes)
	pwr.Footer.Count++
	pwr.Footer.IndexOffset = pwr.currentOffset

	return nil
}

func (pwr *PackWriter) writeAndSum(data any) error {
	if err := binary.Write(pwr.writer, binary.LittleEndian, data); err != nil {
		return err
	}
	if err := binary.Write(pwr.hasher, binary.LittleEndian, data); err != nil {
		return err
	}
	return nil
}

func (pwr *PackWriter) serializeIndex() error {
	for _, record := range pwr.Index {
		if err := pwr.writeAndSum(record.Type); err != nil {
			return err
		}
		if err := pwr.writeAndSum(record.Version); err != nil {
			return err
		}
		if err := pwr.writeAndSum(record.MAC); err != nil {
			return err
		}
		if err := pwr.writeAndSum(record.Offset); err != nil {
			return err
		}
		if err := pwr.writeAndSum(record.Length); err != nil {
			return err
		}
		if err := pwr.writeAndSum(record.Flags); err != nil {
			return err
		}
	}
	return nil
}

func (pwr *PackWriter) serializeFooter() error {
	pwr.Footer.IndexMAC = objects.MAC(pwr.hasher.Sum(nil))
	if err := binary.Write(pwr.writer, binary.LittleEndian, pwr.Footer.Timestamp); err != nil {
		return err
	}
	if err := binary.Write(pwr.writer, binary.LittleEndian, pwr.Footer.Count); err != nil {
		return err
	}
	if err := binary.Write(pwr.writer, binary.LittleEndian, pwr.Footer.IndexOffset); err != nil {
		return err
	}
	if err := binary.Write(pwr.writer, binary.LittleEndian, pwr.Footer.IndexMAC); err != nil {
		return err
	}
	if err := binary.Write(pwr.writer, binary.LittleEndian, pwr.Footer.Flags); err != nil {
		return err
	}
	return nil
}

func (pwr *PackWriter) Size() uint64 {
	return pwr.currentOffset
}

func (pwr *PackWriter) Finalize() error {
	pwr.hasher.Reset()
	if err := pwr.serializeIndex(); err != nil {
		return err
	}
	if err := pwr.serializeFooter(); err != nil {
		return err
	}
	pwr.writer.Close()
	pwr.writer = nil
	<-pwr.pipesync
	return nil
}

func (pwr *PackWriter) Abort() {
	if pwr.writer != nil {
		pwr.writer.Close()
		pwr.writer = nil
	}
}
