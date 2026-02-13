package httpd

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/PlakarKorp/kloset/connectors/storage"
	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
)

var ErrInvalidResourceType = fmt.Errorf("Invalid resource type")
var ErrInvalidMAC = fmt.Errorf("Invalid MAC")
var ErrInvalidRange = fmt.Errorf("Invalid range")

type server struct {
	store    storage.Store
	ctx      context.Context
	noDelete bool
}

func (s *server) openRepository(w http.ResponseWriter, r *http.Request) {
	serializedConfig, err := s.store.Open(s.ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(serializedConfig)
}

func (s *server) listResource(w http.ResponseWriter, r *http.Request) {
	typ, err := getResource(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if indexes, err := s.store.List(r.Context(), typ); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(indexes)
	}
}

func (s *server) getResource(w http.ResponseWriter, r *http.Request) {
	typ, err := getResource(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mac, err := getMac(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	rg, err := getRange(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if rd, err := s.store.Get(r.Context(), typ, mac, rg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		defer rd.Close()

		data, err := io.ReadAll(rd)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(data)
	}
}

func (s *server) putResource(w http.ResponseWriter, r *http.Request) {
	typ, err := getResource(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mac, err := getMac(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if _, err = s.store.Put(r.Context(), typ, mac, r.Body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *server) deleteResource(w http.ResponseWriter, r *http.Request) {
	if s.noDelete {
		http.Error(w, fmt.Errorf("not allowed to delete").Error(), http.StatusForbidden)
		return
	}

	typ, err := getResource(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	mac, err := getMac(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.store.Delete(r.Context(), typ, mac); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

	mux.HandleFunc("GET /resources/{resource}", s.listResource)
	mux.HandleFunc("GET /resources/{resource}/{mac}", s.getResource)
	mux.HandleFunc("PUT /resources/{resource}/{mac}", s.putResource)
	mux.HandleFunc("DELETE /resources/{resource}/{mac}", s.deleteResource)

	server := &http.Server{Addr: addr, Handler: mux}
	go func() {
		<-repo.AppContext().Done()
		server.Shutdown(repo.AppContext().Context)
	}()

	return server.ListenAndServe()
}

func getResource(r *http.Request) (storage.StorageResource, error) {
	switch r.PathValue("resource") {
	case "packfiles":
		return storage.StorageResourcePackfile, nil
	case "states":
		return storage.StorageResourceState, nil
	case "locks":
		return storage.StorageResourceLock, nil
	case "eccpackfiles":
		return storage.StorageResourceECCPackfile, nil
	case "eccstates":
		return storage.StorageResourceECCState, nil
	default:
		return 0, ErrInvalidResourceType
	}
}

func getMac(r *http.Request) (objects.MAC, error) {
	mac, err := hex.DecodeString(r.PathValue("mac"))
	if err != nil {
		return objects.NilMac, ErrInvalidMAC
	}

	if len(mac) != 32 {
		return objects.NilMac, ErrInvalidMAC
	}

	return objects.MAC(mac), nil
}

// Only support fmt.Sprintf("bytes=<offset>-<offset+length>")
func getRange(r *http.Request) (*storage.Range, error) {
	var rng storage.Range
	var err error

	s := r.Header.Get("Range")
	if s == "" {
		return nil, nil
	}

	s, found := strings.CutPrefix(s, "bytes=")
	if !found {
		return nil, ErrInvalidRange
	}

	start, stop, found := strings.Cut(s, "-")
	if !found {
		return nil, ErrInvalidRange
	}

	rng.Offset, err = strconv.ParseUint(start, 10, 64)
	if err != nil {
		return nil, ErrInvalidRange
	}

	end, err := strconv.ParseUint(stop, 10, 64)
	if err != nil {
		return nil, ErrInvalidRange
	}

	if end <= rng.Offset {
		return nil, ErrInvalidRange
	}

	len := end - rng.Offset
	if len > math.MaxUint32 {
		return nil, ErrInvalidRange
	}

	rng.Length = uint32(len)

	return &rng, nil
}
