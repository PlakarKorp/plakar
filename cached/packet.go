package cached

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

type Op = uint32

const (
	OpenReq Op = iota
	OpenRes

	GetReq
	GetRes

	HasReq
	HasRes

	DeleteReq
	DeleteRes

	CloseReq
	CloseRes
)

const (
	hdrlen     = 16
	maxpayload = 8 * 1024 * 1024
)

var (
	ErrPayloadTooLong = errors.New("payload too long")
	ErrNameTooLong    = errors.New("name too long")
	ErrKeyTooLong     = errors.New("key too long")
	ErrValueTooLong   = errors.New("value too long")
)

// XXX add versioning when the cached is opened.
//
// packets layout:
//
// 32 bit type tag +
// 32 bit payload length +
// 32 bit instance number +
// 32 bit gap +
// payload bytes packet specific fields.
//
// XXX: how to transmit errors?
//
// strings: pascal-style 2 bytes for length + data
// []byte: 4 bytes length + data
//
// Messages:
//
// OpenReq: name, repoid, scheme, origin string, ondelete
// OpenRes: handle uint32
//
// GetReq: key []byte
// GetRes: val []byte
//
// HasReq: key []byte
// HasRes: 32byte "boolean"
//
// DeleteReq: key []byte
// DeleteRes: 32byte "boolean"
//
// CloseReq: (void)
// CloseRes: (void)

func getmessage(r io.Reader) (Op, uint32, []byte, error) {
	var hdr [hdrlen]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return 0, 0, nil, err
	}

	op := binary.LittleEndian.Uint32(hdr[0:4])
	instance := binary.LittleEndian.Uint32(hdr[4:8])
	payload := binary.LittleEndian.Uint32(hdr[8:12])
	if payload > maxpayload {
		return 0, 0, nil, ErrPayloadTooLong
	}

	body := make([]byte, 0, payload)
	if _, err := io.ReadFull(r, body[:]); err != nil {
		return op, instance, nil, err
	}

	return op, instance, body, nil
}

func openreq(w io.Writer, name, repoid, scheme, origin string, delete bool) error {
	if len(name) > math.MaxUint16 || len(repoid) > math.MaxUint16 ||
		len(scheme) > math.MaxUint16 || len(origin) > math.MaxUint16 {
		return ErrNameTooLong
	}

	payload := 4*2 + len(name) + len(repoid) + len(scheme) + len(origin) + 1
	if payload > maxpayload {
		return fmt.Errorf("message too long")
	}

	buf := make([]byte, 0, hdrlen+payload)

	binary.LittleEndian.PutUint32(buf, OpenReq)
	binary.LittleEndian.PutUint32(buf, uint32(payload))
	binary.LittleEndian.PutUint64(buf, 0)

	binary.LittleEndian.PutUint16(buf, uint16(len(name)))
	buf = append(buf, []byte(name)...)

	binary.LittleEndian.PutUint16(buf, uint16(len(repoid)))
	buf = append(buf, []byte(repoid)...)

	binary.LittleEndian.PutUint16(buf, uint16(len(scheme)))
	buf = append(buf, []byte(scheme)...)

	binary.LittleEndian.PutUint16(buf, uint16(len(origin)))
	buf = append(buf, []byte(origin)...)

	if delete {
		buf = append(buf, 1)
	} else {
		buf = append(buf, 0)
	}

	_, err := w.Write(buf)
	return err
}

func openres(w io.Writer, instance uint32) error {
	buf := make([]byte, 0, hdrlen)

	binary.LittleEndian.PutUint32(buf, GetReq)
	binary.LittleEndian.PutUint32(buf, 0)
	binary.LittleEndian.PutUint32(buf, instance)
	binary.LittleEndian.PutUint32(buf, 0)

	_, err := w.Write(buf)
	return err
}

func getreq(w io.Writer, instance uint32, key []byte) error {
	if len(key) > math.MaxUint32 {
		return ErrKeyTooLong
	}

	payload := 4 + len(key)
	if payload > maxpayload {
		return ErrPayloadTooLong
	}

	buf := make([]byte, 0, hdrlen+payload)

	binary.LittleEndian.PutUint32(buf, GetReq)
	binary.LittleEndian.PutUint32(buf, uint32(payload))
	binary.LittleEndian.PutUint32(buf, instance)
	binary.LittleEndian.PutUint32(buf, 0)

	binary.LittleEndian.PutUint32(buf, uint32(len(key)))
	buf = append(buf, key...)

	_, err := w.Write(buf)
	return err
}

func getres(w io.Writer, instance uint32, val []byte) error {
	if len(val) > math.MaxUint32 {
		return ErrValueTooLong
	}

	payload := 4 + len(val)
	if payload > maxpayload {
		return ErrPayloadTooLong
	}

	buf := make([]byte, 0, hdrlen+payload)

	binary.LittleEndian.PutUint32(buf, GetRes)
	binary.LittleEndian.PutUint32(buf, uint32(payload))
	binary.LittleEndian.PutUint32(buf, instance)
	binary.LittleEndian.PutUint32(buf, 0)

	binary.LittleEndian.PutUint32(buf, uint32(len(val)))
	buf = append(buf, val...)

	_, err := w.Write(buf)
	return err
}

func hasreq(w io.Writer, instance uint32, key []byte) error {
	if len(key) > math.MaxUint32 {
		return ErrKeyTooLong
	}

	payload := 4 + len(key)
	if payload > maxpayload {
		return ErrPayloadTooLong
	}

	buf := make([]byte, 0, hdrlen+payload)

	binary.LittleEndian.PutUint32(buf, HasReq)
	binary.LittleEndian.PutUint32(buf, uint32(payload))
	binary.LittleEndian.PutUint32(buf, instance)
	binary.LittleEndian.PutUint32(buf, 0)

	binary.LittleEndian.PutUint32(buf, uint32(len(key)))
	buf = append(buf, key...)

	_, err := w.Write(buf)
	return err
}

func hasres(w io.Writer, instance uint32, found bool) error {
	payload := 4
	if payload > maxpayload { // impossible
		return ErrPayloadTooLong
	}

	buf := make([]byte, 0, hdrlen+payload)

	binary.LittleEndian.PutUint32(buf, HasReq)
	binary.LittleEndian.PutUint32(buf, uint32(payload))
	binary.LittleEndian.PutUint32(buf, instance)
	binary.LittleEndian.PutUint32(buf, 0)

	if found {
		binary.LittleEndian.PutUint32(buf, 1)
	} else {
		binary.LittleEndian.PutUint32(buf, 0)
	}

	_, err := w.Write(buf)
	return err
}

func delreq(w io.WriteCloser, instance uint32, key []byte) error {
	if len(key) > math.MaxUint32 {
		return ErrKeyTooLong
	}

	payload := 4 + len(key)
	if payload > maxpayload {
		return ErrPayloadTooLong
	}

	buf := make([]byte, 0, hdrlen+payload)

	binary.LittleEndian.PutUint32(buf, DeleteReq)
	binary.LittleEndian.PutUint32(buf, uint32(payload))
	binary.LittleEndian.PutUint32(buf, instance)
	binary.LittleEndian.PutUint32(buf, 0)

	binary.LittleEndian.PutUint32(buf, uint32(len(key)))
	buf = append(buf, key...)

	_, err := w.Write(buf)
	return err
}

func delres(w io.Writer, instance uint32) error {
	buf := make([]byte, 0, hdrlen)

	binary.LittleEndian.PutUint32(buf, DeleteRes)
	binary.LittleEndian.PutUint32(buf, 0)
	binary.LittleEndian.PutUint32(buf, instance)
	binary.LittleEndian.PutUint32(buf, 0)

	_, err := w.Write(buf)
	return err
}

func closereq(w io.WriteCloser, instance uint32) error {
	buf := make([]byte, 0, hdrlen)

	binary.LittleEndian.PutUint32(buf, CloseReq)
	binary.LittleEndian.PutUint32(buf, 0)
	binary.LittleEndian.PutUint32(buf, instance)
	binary.LittleEndian.PutUint32(buf, 0)

	_, err := w.Write(buf)
	return err
}

func closeres(w io.Writer, instance uint32) error {
	buf := make([]byte, 0, hdrlen)

	binary.LittleEndian.PutUint32(buf, CloseRes)
	binary.LittleEndian.PutUint32(buf, 0)
	binary.LittleEndian.PutUint32(buf, instance)
	binary.LittleEndian.PutUint32(buf, 0)

	_, err := w.Write(buf)
	return err
}
