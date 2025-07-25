/*
 * Copyright (c) 2021 Gilles Chehade <gilles@poolp.org>
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

package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/storage"
	"github.com/PlakarKorp/plakar/network"
)

type Store struct {
	config     storage.Configuration
	Repository string
	location   string
}

func init() {
	storage.Register("http", 0, NewStore)
	storage.Register("https", 0, NewStore)
}

func NewStore(ctx context.Context, proto string, storeConfig map[string]string) (storage.Store, error) {
	return &Store{
		location: storeConfig["location"],
	}, nil
}

func (s *Store) Location() string {
	return s.location
}

func (s *Store) sendRequest(method string, requestType string, payload interface{}) (*http.Response, error) {
	requestBody, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, s.location+requestType, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	return client.Do(req)
}

func (s *Store) Create(ctx context.Context, config []byte) error {
	return nil
}

func (s *Store) Open(ctx context.Context) ([]byte, error) {
	s.Repository = s.location
	r, err := s.sendRequest("GET", "/", network.ReqOpen{
		Repository: "",
	})
	if err != nil {
		return nil, err
	}

	var resOpen network.ResOpen
	if err := json.NewDecoder(r.Body).Decode(&resOpen); err != nil {
		return nil, err
	}
	if resOpen.Err != "" {
		return nil, fmt.Errorf("%s", resOpen.Err)
	}
	return resOpen.Configuration, nil
}

func (s *Store) Close() error {
	return nil
}

func (s *Store) Mode() storage.Mode {
	return storage.ModeRead | storage.ModeWrite
}

func (s *Store) Size() int64 {
	return -1
}

// states
func (s *Store) GetStates() ([]objects.MAC, error) {
	r, err := s.sendRequest("GET", "/states", network.ReqGetStates{})
	if err != nil {
		return nil, err
	}

	var resGetStates network.ResGetStates
	if err := json.NewDecoder(r.Body).Decode(&resGetStates); err != nil {
		return nil, err
	}
	if resGetStates.Err != "" {
		return nil, fmt.Errorf("%s", resGetStates.Err)
	}

	ret := make([]objects.MAC, len(resGetStates.MACs))
	for i, MAC := range resGetStates.MACs {
		ret[i] = MAC
	}
	return ret, nil
}

func (s *Store) PutState(MAC objects.MAC, rd io.Reader) (int64, error) {
	data, err := io.ReadAll(rd)
	if err != nil {
		return 0, err
	}

	r, err := s.sendRequest("PUT", "/state", network.ReqPutState{
		MAC:  MAC,
		Data: data,
	})
	if err != nil {
		return 0, err
	}

	var resPutState network.ResPutState
	if err := json.NewDecoder(r.Body).Decode(&resPutState); err != nil {
		return 0, err
	}
	if resPutState.Err != "" {
		return 0, fmt.Errorf("%s", resPutState.Err)
	}
	return int64(len(data)), nil
}

func (s *Store) GetState(MAC objects.MAC) (io.ReadCloser, error) {
	r, err := s.sendRequest("GET", "/state", network.ReqGetState{
		MAC: MAC,
	})
	if err != nil {
		return nil, err
	}

	var resGetState network.ResGetState
	if err := json.NewDecoder(r.Body).Decode(&resGetState); err != nil {
		return nil, err
	}
	if resGetState.Err != "" {
		return nil, fmt.Errorf("%s", resGetState.Err)
	}
	return io.NopCloser(bytes.NewBuffer(resGetState.Data)), nil
}

func (s *Store) DeleteState(MAC objects.MAC) error {
	r, err := s.sendRequest("DELETE", "/state", network.ReqDeleteState{
		MAC: MAC,
	})
	if err != nil {
		return err
	}

	var resDeleteState network.ResDeleteState
	if err := json.NewDecoder(r.Body).Decode(&resDeleteState); err != nil {
		return err
	}
	if resDeleteState.Err != "" {
		return fmt.Errorf("%s", resDeleteState.Err)
	}
	return nil
}

// packfiles
func (s *Store) GetPackfiles() ([]objects.MAC, error) {
	r, err := s.sendRequest("GET", "/packfiles", network.ReqGetPackfiles{})
	if err != nil {
		return nil, err
	}

	var resGetPackfiles network.ResGetPackfiles
	if err := json.NewDecoder(r.Body).Decode(&resGetPackfiles); err != nil {
		return nil, err
	}
	if resGetPackfiles.Err != "" {
		return nil, fmt.Errorf("%s", resGetPackfiles.Err)
	}

	ret := make([]objects.MAC, len(resGetPackfiles.MACs))
	for i, MAC := range resGetPackfiles.MACs {
		ret[i] = MAC
	}
	return ret, nil
}

func (s *Store) PutPackfile(MAC objects.MAC, rd io.Reader) (int64, error) {
	data, err := io.ReadAll(rd)
	if err != nil {
		return 0, err
	}
	r, err := s.sendRequest("PUT", "/packfile", network.ReqPutPackfile{
		MAC:  MAC,
		Data: data,
	})
	if err != nil {
		return 0, err
	}

	var resPutPackfile network.ResPutPackfile
	if err := json.NewDecoder(r.Body).Decode(&resPutPackfile); err != nil {
		return 0, err
	}
	if resPutPackfile.Err != "" {
		return 0, fmt.Errorf("%s", resPutPackfile.Err)
	}
	return int64(len(data)), nil
}

func (s *Store) GetPackfile(MAC objects.MAC) (io.ReadCloser, error) {
	r, err := s.sendRequest("GET", "/packfile", network.ReqGetPackfile{
		MAC: MAC,
	})
	if err != nil {
		return nil, err
	}

	var resGetPackfile network.ResGetPackfile
	if err := json.NewDecoder(r.Body).Decode(&resGetPackfile); err != nil {
		return nil, err
	}
	if resGetPackfile.Err != "" {
		return nil, fmt.Errorf("%s", resGetPackfile.Err)
	}
	return io.NopCloser(bytes.NewBuffer(resGetPackfile.Data)), nil
}

func (s *Store) GetPackfileBlob(MAC objects.MAC, offset uint64, length uint32) (io.ReadCloser, error) {
	r, err := s.sendRequest("GET", "/packfile/blob", network.ReqGetPackfileBlob{
		MAC:    MAC,
		Offset: offset,
		Length: length,
	})
	if err != nil {
		return nil, err
	}

	var resGetPackfileBlob network.ResGetPackfileBlob
	if err := json.NewDecoder(r.Body).Decode(&resGetPackfileBlob); err != nil {
		return nil, err
	}
	if resGetPackfileBlob.Err != "" {
		return nil, fmt.Errorf("%s", resGetPackfileBlob.Err)
	}
	return io.NopCloser(bytes.NewBuffer(resGetPackfileBlob.Data)), nil
}

func (s *Store) DeletePackfile(MAC objects.MAC) error {
	r, err := s.sendRequest("DELETE", "/packfile", network.ReqDeletePackfile{
		MAC: MAC,
	})
	if err != nil {
		return err
	}

	var resDeletePackfile network.ResDeletePackfile
	if err := json.NewDecoder(r.Body).Decode(&resDeletePackfile); err != nil {
		return err
	}
	if resDeletePackfile.Err != "" {
		return fmt.Errorf("%s", resDeletePackfile.Err)
	}
	return nil
}

/* Locks */
func (s *Store) GetLocks() ([]objects.MAC, error) {
	r, err := s.sendRequest("GET", "/locks", &network.ReqGetLocks{})
	if err != nil {
		return []objects.MAC{}, err
	}

	var res network.ResGetLocks
	if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
		return []objects.MAC{}, err
	}
	if res.Err != "" {
		return []objects.MAC{}, fmt.Errorf("%s", res.Err)
	}
	return res.Locks, nil
}

func (s *Store) PutLock(lockID objects.MAC, rd io.Reader) (int64, error) {
	data, err := io.ReadAll(rd)
	if err != nil {
		return 0, err
	}

	req := network.ReqPutLock{
		Mac:  lockID,
		Data: data,
	}
	r, err := s.sendRequest("PUT", "/lock", &req)
	if err != nil {
		return 0, err
	}

	var res network.ResPutLock
	if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
		return 0, err
	}
	if res.Err != "" {
		return 0, fmt.Errorf("%s", res.Err)
	}
	return int64(len(data)), nil
}

func (s *Store) GetLock(lockID objects.MAC) (io.ReadCloser, error) {
	req := network.ReqGetLock{
		Mac: lockID,
	}
	r, err := s.sendRequest("GET", "/lock", &req)
	if err != nil {
		return nil, err
	}

	var res network.ResGetLock
	if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
		return nil, err
	}

	if res.Err != "" {
		return nil, fmt.Errorf("%s", res.Err)
	}

	return io.NopCloser(bytes.NewReader(res.Data)), nil
}

func (s *Store) DeleteLock(lockID objects.MAC) error {
	req := network.ReqDeleteLock{
		Mac: lockID,
	}
	r, err := s.sendRequest("DELETE", "/lock", &req)
	if err != nil {
		return err
	}

	var res network.ResDeleteLock
	if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
		return err
	}

	if res.Err != "" {
		return fmt.Errorf("%s", res.Err)
	}
	return nil
}
