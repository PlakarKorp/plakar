package httpd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/storage"
	"github.com/PlakarKorp/plakar/network"
)

type server struct {
	store    storage.Store
	ctx      context.Context
	noDelete bool
}

func (s *server) openRepository(w http.ResponseWriter, r *http.Request) {
	var reqOpen network.ReqOpen
	if err := json.NewDecoder(r.Body).Decode(&reqOpen); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	serializedConfig, err := s.store.Open(s.ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var resOpen network.ResOpen
	resOpen.Configuration = serializedConfig
	resOpen.Err = ""
	if err := json.NewEncoder(w).Encode(resOpen); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// states
func (s *server) getStates(w http.ResponseWriter, r *http.Request) {
	var reqGetIndexes network.ReqGetStates
	if err := json.NewDecoder(r.Body).Decode(&reqGetIndexes); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resGetStates network.ResGetStates
	indexes, err := s.store.GetStates(r.Context())
	if err != nil {
		resGetStates.Err = err.Error()
	} else {
		resGetStates.MACs = indexes
	}
	if err := json.NewEncoder(w).Encode(resGetStates); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) putState(w http.ResponseWriter, r *http.Request) {
	var reqPutState network.ReqPutState
	if err := json.NewDecoder(r.Body).Decode(&reqPutState); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resPutIndex network.ResPutState
	data := reqPutState.Data
	_, err := s.store.PutState(r.Context(), reqPutState.MAC, bytes.NewBuffer(data))
	if err != nil {
		resPutIndex.Err = err.Error()
	}
	if err := json.NewEncoder(w).Encode(resPutIndex); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) getState(w http.ResponseWriter, r *http.Request) {
	var reqGetState network.ReqGetState
	if err := json.NewDecoder(r.Body).Decode(&reqGetState); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resGetState network.ResGetState
	rd, err := s.store.GetState(r.Context(), reqGetState.MAC)
	if err != nil {
		resGetState.Err = err.Error()
	} else {
		defer rd.Close()

		data, err := io.ReadAll(rd)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		resGetState.Data = data
	}
	if err := json.NewEncoder(w).Encode(resGetState); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) deleteState(w http.ResponseWriter, r *http.Request) {
	if s.noDelete {
		http.Error(w, fmt.Errorf("not allowed to delete").Error(), http.StatusForbidden)
		return
	}

	var reqDeleteState network.ReqDeleteState
	if err := json.NewDecoder(r.Body).Decode(&reqDeleteState); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resDeleteState network.ResDeleteState
	err := s.store.DeleteState(r.Context(), reqDeleteState.MAC)
	if err != nil {
		resDeleteState.Err = err.Error()
	}
	if err := json.NewEncoder(w).Encode(resDeleteState); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// packfiles
func (s *server) getPackfiles(w http.ResponseWriter, r *http.Request) {
	var reqGetPackfiles network.ReqGetPackfiles
	if err := json.NewDecoder(r.Body).Decode(&reqGetPackfiles); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resGetPackfiles network.ResGetPackfiles
	packfiles, err := s.store.GetPackfiles(r.Context())
	if err != nil {
		resGetPackfiles.Err = err.Error()
	} else {
		resGetPackfiles.MACs = packfiles
	}
	if err := json.NewEncoder(w).Encode(resGetPackfiles); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) putPackfile(w http.ResponseWriter, r *http.Request) {
	var reqPutPackfile network.ReqPutPackfile
	if err := json.NewDecoder(r.Body).Decode(&reqPutPackfile); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resPutPackfile network.ResPutPackfile
	_, err := s.store.PutPackfile(r.Context(), reqPutPackfile.MAC, bytes.NewBuffer(reqPutPackfile.Data))
	if err != nil {
		resPutPackfile.Err = err.Error()
	}
	if err := json.NewEncoder(w).Encode(resPutPackfile); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) getPackfile(w http.ResponseWriter, r *http.Request) {
	var reqGetPackfile network.ReqGetPackfile
	if err := json.NewDecoder(r.Body).Decode(&reqGetPackfile); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resGetPackfile network.ResGetPackfile
	rd, err := s.store.GetPackfile(r.Context(), reqGetPackfile.MAC)
	if err != nil {
		resGetPackfile.Err = err.Error()
	} else {
		defer rd.Close()
		data, err := io.ReadAll(rd)
		if err != nil {
			resGetPackfile.Err = err.Error()
		} else {
			resGetPackfile.Data = data
		}
	}
	if err := json.NewEncoder(w).Encode(resGetPackfile); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) GetPackfileBlob(w http.ResponseWriter, r *http.Request) {
	var reqGetPackfileBlob network.ReqGetPackfileBlob
	if err := json.NewDecoder(r.Body).Decode(&reqGetPackfileBlob); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resGetPackfileBlob network.ResGetPackfileBlob
	rd, err := s.store.GetPackfileBlob(r.Context(), reqGetPackfileBlob.MAC, reqGetPackfileBlob.Offset, reqGetPackfileBlob.Length)
	if err != nil {
		resGetPackfileBlob.Err = err.Error()
	} else {
		data, err := io.ReadAll(rd)
		rd.Close()

		if err != nil {
			resGetPackfileBlob.Err = err.Error()
		} else {
			resGetPackfileBlob.Data = data
		}
	}
	if err := json.NewEncoder(w).Encode(resGetPackfileBlob); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) deletePackfile(w http.ResponseWriter, r *http.Request) {
	if s.noDelete {
		http.Error(w, fmt.Errorf("not allowed to delete").Error(), http.StatusForbidden)
		return
	}

	var reqDeletePackfile network.ReqDeletePackfile
	if err := json.NewDecoder(r.Body).Decode(&reqDeletePackfile); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resDeletePackfile network.ResDeletePackfile
	err := s.store.DeletePackfile(r.Context(), reqDeletePackfile.MAC)
	if err != nil {
		resDeletePackfile.Err = err.Error()
	}
	if err := json.NewEncoder(w).Encode(resDeletePackfile); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) getLocks(w http.ResponseWriter, r *http.Request) {
	var req network.ReqGetLocks
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	locks, err := s.store.GetLocks(r.Context())
	res := network.ResGetLocks{
		Locks: locks,
	}
	if err != nil {
		res.Err = err.Error()
	}
	if err := json.NewEncoder(w).Encode(&res); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) putLock(w http.ResponseWriter, r *http.Request) {
	var req network.ReqPutLock
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var res network.ResPutLock
	if _, err := s.store.PutLock(r.Context(), req.Mac, bytes.NewReader(req.Data)); err != nil {
		res.Err = err.Error()
	}

	if err := json.NewEncoder(w).Encode(&res); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) getLock(w http.ResponseWriter, r *http.Request) {
	var req network.ReqGetLock
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var res network.ResGetLock
	rd, err := s.store.GetLock(r.Context(), req.Mac)
	if err != nil {
		res.Err = err.Error()
	} else {
		defer rd.Close()
		data, err := io.ReadAll(rd)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Data = data
	}

	if err := json.NewEncoder(w).Encode(&res); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *server) deleteLock(w http.ResponseWriter, r *http.Request) {
	var req network.ReqDeleteLock
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var res network.ResDeleteLock
	if err := s.store.DeleteLock(r.Context(), req.Mac); err != nil {
		res.Err = err.Error()
	}

	if err := json.NewEncoder(w).Encode(&res); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func Server(ctx context.Context, repo *repository.Repository, addr string, noDelete bool) error {
	s := server{
		store:    repo.Store(),
		ctx:      ctx,
		noDelete: noDelete,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", s.openRepository)

	mux.HandleFunc("GET /states", s.getStates)
	mux.HandleFunc("PUT /state", s.putState)
	mux.HandleFunc("GET /state", s.getState)
	mux.HandleFunc("DELETE /state", s.deleteState)

	mux.HandleFunc("GET /packfiles", s.getPackfiles)
	mux.HandleFunc("PUT /packfile", s.putPackfile)
	mux.HandleFunc("GET /packfile", s.getPackfile)
	mux.HandleFunc("GET /packfile/blob", s.GetPackfileBlob)
	mux.HandleFunc("DELETE /packfile", s.deletePackfile)

	mux.HandleFunc("GET /locks", s.getLocks)
	mux.HandleFunc("PUT /lock", s.putLock)
	mux.HandleFunc("GET /lock", s.getLock)
	mux.HandleFunc("DELETE /lock", s.deleteLock)

	server := &http.Server{Addr: addr, Handler: mux}
	go func() {
		<-repo.AppContext().Done()
		server.Shutdown(repo.AppContext().Context)
	}()

	return server.ListenAndServe()
}
