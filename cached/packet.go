package cached

import (
	"encoding/binary"
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
)

const hdrlen = 16

// packets layout:
//
// 32 bit type tag +
// 32 bit payload length +
// 32 bit instance number +
// 32 bit gap +
// payload bytes packet specific fields.
//
// strings: pascal-style 2 bytes for length + data
// []byte: 4 bytes length + data
//
// OpenReq: name, repoid, scheme, origin string, ondelete
// OpenRes: handle int64
//
// GetReq: key []byte
// GetRes: val []byte
//
// HasReq: key []byte
// HasRes: 32byte "boolean"

func getmessage(r io.Reader) (Op, uint32, []byte, error) {
	var hdr [hdrlen]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return 0, 0, nil, err
	}

	op := binary.BigEndian.Uint32(hdr[0:4])
	instance := binary.BigEndian.Uint32(hdr[4:8])
	payload := binary.BigEndian.Uint32(hdr[8:12])

	body := make([]byte, 0, payload)
	if _, err := io.ReadFull(r, body[:]); err != nil {
		return op, instance, nil, err
	}

	return op, instance, body, nil
}

func openreq(w io.Writer, name, repoid, scheme, origin string, delete bool) error {
	if len(name) > math.MaxUint16 || len(repoid) > math.MaxUint16 ||
		len(scheme) > math.MaxUint16 || len(origin) > math.MaxUint16 {
		return fmt.Errorf("names too long!")
	}

	payload := 4*2 + len(name) + len(repoid) + len(scheme) + len(origin) + 1
	if payload > math.MaxUint32 { // impossible
		return fmt.Errorf("message too long")
	}

	buf := make([]byte, 0, hdrlen+payload)

	binary.BigEndian.PutUint32(buf, OpenReq)
	binary.BigEndian.PutUint32(buf, uint32(payload))
	binary.BigEndian.PutUint64(buf, 0)

	binary.BigEndian.PutUint16(buf, uint16(len(name)))
	buf = append(buf, []byte(name)...)

	binary.BigEndian.PutUint16(buf, uint16(len(repoid)))
	buf = append(buf, []byte(repoid)...)

	binary.BigEndian.PutUint16(buf, uint16(len(scheme)))
	buf = append(buf, []byte(scheme)...)

	binary.BigEndian.PutUint16(buf, uint16(len(origin)))
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

	binary.BigEndian.PutUint32(buf, GetReq)
	binary.BigEndian.PutUint32(buf, 0)
	binary.BigEndian.PutUint32(buf, instance)
	binary.BigEndian.PutUint32(buf, 0)

	_, err := w.Write(buf)
	return err
}

func getreq(w io.Writer, instance uint32, key []byte) error {
	if len(key) > math.MaxUint32 {
		return fmt.Errorf("key too long!")
	}

	payload := 4 + len(key)
	if payload > math.MaxUint32 { // impossible
		return fmt.Errorf("message too long")
	}

	buf := make([]byte, 0, hdrlen+payload)

	binary.BigEndian.PutUint32(buf, GetReq)
	binary.BigEndian.PutUint32(buf, uint32(payload))
	binary.BigEndian.PutUint32(buf, instance)
	binary.BigEndian.PutUint32(buf, 0)

	binary.BigEndian.PutUint32(buf, uint32(len(key)))
	buf = append(buf, key...)

	_, err := w.Write(buf)
	return err
}

func getres(w io.Writer, instance uint32, val []byte) error {
	if len(val) > math.MaxUint32 {
		return fmt.Errorf("value too long!")
	}

	payload := 4 + len(val)
	if payload > math.MaxUint32 { // impossible
		return fmt.Errorf("message too long")
	}

	buf := make([]byte, 0, hdrlen+payload)

	binary.BigEndian.PutUint32(buf, GetRes)
	binary.BigEndian.PutUint32(buf, uint32(payload))
	binary.BigEndian.PutUint32(buf, instance)
	binary.BigEndian.PutUint32(buf, 0)

	binary.BigEndian.PutUint32(buf, uint32(len(val)))
	buf = append(buf, val...)

	_, err := w.Write(buf)
	return err
}

func hasreq(w io.Writer, instance uint32, key []byte) error {
	if len(key) > math.MaxUint32 {
		return fmt.Errorf("value too long!")
	}

	payload := 4 + len(key)
	if payload > math.MaxUint32 { // impossible
		return fmt.Errorf("message too long")
	}

	buf := make([]byte, 0, hdrlen+payload)

	binary.BigEndian.PutUint32(buf, HasReq)
	binary.BigEndian.PutUint32(buf, uint32(payload))
	binary.BigEndian.PutUint32(buf, instance)
	binary.BigEndian.PutUint32(buf, 0)

	binary.BigEndian.PutUint32(buf, uint32(len(key)))
	buf = append(buf, key...)

	_, err := w.Write(buf)
	return err
}

func hasres(w io.Writer, instance uint32, found bool) error {
	payload := 4
	if payload > math.MaxUint32 { // impossible
		return fmt.Errorf("message too long")
	}

	buf := make([]byte, 0, hdrlen+payload)

	binary.BigEndian.PutUint32(buf, HasReq)
	binary.BigEndian.PutUint32(buf, uint32(payload))
	binary.BigEndian.PutUint32(buf, instance)
	binary.BigEndian.PutUint32(buf, 0)

	if found {
		binary.BigEndian.PutUint32(buf, 1)
	} else {
		binary.BigEndian.PutUint32(buf, 0)
	}

	_, err := w.Write(buf)
	return err
}
