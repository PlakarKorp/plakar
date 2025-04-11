package packer

import (
	"bytes"
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
	encoder func(io.Reader) (io.Reader, error)
	hasher  hash.Hash

	Index         []Blob
	Reader        io.Reader
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

func NewPackWriter(putter func(*PackWriter) error, encoder func(io.Reader) (io.Reader, error), hasher func() hash.Hash) *PackWriter {
	pipesync := make(chan struct{}, 1)
	pr, pw := io.Pipe()

	p := &PackWriter{
		encoder:  encoder,
		hasher:   hasher(),
		Index:    make([]Blob, 0), // temporary
		writer:   pw,
		Reader:   pr,
		pipesync: pipesync,
	}
	// packfilewriter -> pw -> pipe -> pr <- putter (io.ReadAll())
	go func() {
		defer pr.Close()
		defer func() { close(pipesync) }()
		if err := putter(p); err != nil {
			pr.CloseWithError(err)
		}
	}()

	return p
}

func (pwr *PackWriter) WriteBlob(resourceType resources.Type, version versioning.Version, mac objects.MAC, data []byte, flags uint32) error {
	encodedReader, err := pwr.encoder(bytes.NewReader(data))
	if err != nil {
		return err
	}

	nbytes, err := io.Copy(pwr.writer, encodedReader)
	if err != nil {
		return err
	}

	pwr.Index = append(pwr.Index, Blob{
		Type:    resourceType,
		Version: version,
		MAC:     mac,
		Offset:  uint64(pwr.currentOffset),
		Length:  uint32(nbytes),
		Flags:   flags,
	})

	pwr.currentOffset += uint64(nbytes)
	pwr.Footer.Count++
	pwr.Footer.IndexOffset = pwr.currentOffset

	return nil
}

func (pwr *PackWriter) writeAndSum(writer io.Writer, data any) error {
	if err := binary.Write(writer, binary.LittleEndian, data); err != nil {
		return err
	}
	if err := binary.Write(pwr.hasher, binary.LittleEndian, data); err != nil {
		return err
	}
	return nil
}

func (pwr *PackWriter) serializeIndex() error {
	pr, pw := io.Pipe()

	encoder, err := pwr.encoder(pr)
	if err != nil {
		return err
	}

	done := make(chan struct{})
	go func() {
		defer func() { close(done) }()
		_, err := io.Copy(pwr.writer, encoder)
		if err != nil {
			pw.CloseWithError(err)
		}
	}()

	for _, record := range pwr.Index {
		if err := pwr.writeAndSum(pw, record.Type); err != nil {
			pw.CloseWithError(err)
			return err
		}
		if err := pwr.writeAndSum(pw, record.Version); err != nil {
			pw.CloseWithError(err)
			return err
		}
		if err := pwr.writeAndSum(pw, record.MAC); err != nil {
			pw.CloseWithError(err)
			return err
		}
		if err := pwr.writeAndSum(pw, record.Offset); err != nil {
			pw.CloseWithError(err)
			return err
		}
		if err := pwr.writeAndSum(pw, record.Length); err != nil {
			pw.CloseWithError(err)
			return err
		}
		if err := pwr.writeAndSum(pw, record.Flags); err != nil {
			pw.CloseWithError(err)
			return err
		}
	}

	if err := pw.Close(); err != nil {
		return pw.Close()
	}
	<-done

	return nil
}

func (pwr *PackWriter) serializeFooter() error {
	pwr.Footer.IndexMAC = objects.MAC(pwr.hasher.Sum(nil))

	pr, pw := io.Pipe()

	encoder, err := pwr.encoder(pr)
	if err != nil {
		return err
	}

	done := make(chan int64)
	go func() {
		defer func() { close(done) }()
		n, err := io.Copy(pwr.writer, encoder)
		if err != nil {
			pw.CloseWithError(err)
		} else {
			done <- n
		}
	}()

	if err := binary.Write(pw, binary.LittleEndian, pwr.Footer.Timestamp); err != nil {
		pw.CloseWithError(err)
		return err
	}
	if err := binary.Write(pw, binary.LittleEndian, pwr.Footer.Count); err != nil {
		pw.CloseWithError(err)
		return err
	}
	if err := binary.Write(pw, binary.LittleEndian, pwr.Footer.IndexOffset); err != nil {
		pw.CloseWithError(err)
		return err
	}
	if err := binary.Write(pw, binary.LittleEndian, pwr.Footer.IndexMAC); err != nil {
		pw.CloseWithError(err)
		return err
	}
	if err := binary.Write(pw, binary.LittleEndian, pwr.Footer.Flags); err != nil {
		pw.CloseWithError(err)
		return err
	}

	if err := pw.Close(); err != nil {
		return pw.Close()
	}
	nbytes := <-done

	encryptedFooterLength := make([]byte, 4)
	binary.LittleEndian.PutUint32(encryptedFooterLength, uint32(nbytes))
	if err := binary.Write(pwr.writer, binary.LittleEndian, encryptedFooterLength); err != nil {
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
