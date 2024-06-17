package httpd

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/PlakarLabs/plakar/network"
	"github.com/PlakarLabs/plakar/storage"
	"github.com/gorilla/mux"
)

var lrepository *storage.Repository
var lNoDelete bool

func openRepository(w http.ResponseWriter, r *http.Request) {
	var reqOpen network.ReqOpen
	if err := json.NewDecoder(r.Body).Decode(&reqOpen); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	config := lrepository.Configuration()

	var resOpen network.ResOpen
	resOpen.RepositoryConfig = &config
	resOpen.Err = ""
	if err := json.NewEncoder(w).Encode(resOpen); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func closeRepository(w http.ResponseWriter, r *http.Request) {
	var reqClose network.ReqClose
	if err := json.NewDecoder(r.Body).Decode(&reqClose); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if reqClose.Uuid != lrepository.Configuration().RepositoryID.String() {
		http.Error(w, "UUID mismatch", http.StatusBadRequest)
		return
	}

	var resClose network.ResClose
	resClose.Err = ""
	if err := json.NewEncoder(w).Encode(resClose); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// snapshots
func getSnapshots(w http.ResponseWriter, r *http.Request) {
	var reqGetSnapshots network.ReqGetSnapshots
	if err := json.NewDecoder(r.Body).Decode(&reqGetSnapshots); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resGetSnapshots network.ResGetSnapshots
	snapshots, err := lrepository.GetSnapshots()
	if err != nil {
		resGetSnapshots.Err = err.Error()
	} else {
		resGetSnapshots.Snapshots = snapshots
	}
	if err := json.NewEncoder(w).Encode(resGetSnapshots); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func putSnapshot(w http.ResponseWriter, r *http.Request) {
	var reqPutSnapshot network.ReqPutSnapshot
	if err := json.NewDecoder(r.Body).Decode(&reqPutSnapshot); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resPutSnapshot network.ResPutSnapshot
	err := lrepository.PutSnapshot(reqPutSnapshot.IndexID, reqPutSnapshot.Data)
	if err != nil {
		resPutSnapshot.Err = err.Error()
	}
	if err := json.NewEncoder(w).Encode(resPutSnapshot); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getSnapshot(w http.ResponseWriter, r *http.Request) {
	var reqGetSnapshot network.ReqGetSnapshot
	if err := json.NewDecoder(r.Body).Decode(&reqGetSnapshot); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resGetSnapshot network.ResGetSnapshot
	data, err := lrepository.GetSnapshot(reqGetSnapshot.IndexID)
	if err != nil {
		resGetSnapshot.Err = err.Error()
	} else {
		resGetSnapshot.Data = data
	}
	if err := json.NewEncoder(w).Encode(resGetSnapshot); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func deleteSnapshot(w http.ResponseWriter, r *http.Request) {
	if lNoDelete {
		http.Error(w, fmt.Errorf("not allowed to delete").Error(), http.StatusForbidden)
		return
	}

	var reqDeleteSnapshot network.ReqDeleteSnapshot
	if err := json.NewDecoder(r.Body).Decode(&reqDeleteSnapshot); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resDeleteSnapshot network.ResDeleteSnapshot
	err := lrepository.DeleteSnapshot(reqDeleteSnapshot.IndexID)
	if err != nil {
		resDeleteSnapshot.Err = err.Error()
	}
	if err := json.NewEncoder(w).Encode(resDeleteSnapshot); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func commitSnapshot(w http.ResponseWriter, r *http.Request) {
	var ReqCommit network.ReqCommit
	if err := json.NewDecoder(r.Body).Decode(&ReqCommit); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var ResCommit network.ResCommit
	err := lrepository.Commit(ReqCommit.IndexID, ReqCommit.Data)
	if err != nil {
		ResCommit.Err = err.Error()
	}

	if err := json.NewEncoder(w).Encode(ResCommit); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// locks
func getLocks(w http.ResponseWriter, r *http.Request) {
	var reqGetLocks network.ReqGetLocks
	if err := json.NewDecoder(r.Body).Decode(&reqGetLocks); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resGetLocks network.ResGetLocks
	locks, err := lrepository.GetLocks()
	if err != nil {
		resGetLocks.Err = err.Error()
	} else {
		resGetLocks.Locks = locks
	}
	if err := json.NewEncoder(w).Encode(resGetLocks); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func putLock(w http.ResponseWriter, r *http.Request) {
	var reqPutLock network.ReqPutLock
	if err := json.NewDecoder(r.Body).Decode(&reqPutLock); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resPutLock network.ResPutLock
	err := lrepository.PutLock(reqPutLock.IndexID, reqPutLock.Data)
	if err != nil {
		resPutLock.Err = err.Error()
	}
	if err := json.NewEncoder(w).Encode(resPutLock); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getLock(w http.ResponseWriter, r *http.Request) {
	var reqGetLock network.ReqGetLock
	if err := json.NewDecoder(r.Body).Decode(&reqGetLock); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resGetLock network.ResGetLock
	data, err := lrepository.GetLock(reqGetLock.IndexID)
	if err != nil {
		resGetLock.Err = err.Error()
	} else {
		resGetLock.Data = data
	}
	if err := json.NewEncoder(w).Encode(resGetLock); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func deleteLock(w http.ResponseWriter, r *http.Request) {
	var reqDeleteLock network.ReqDeleteLock
	if err := json.NewDecoder(r.Body).Decode(&reqDeleteLock); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resDeleteLock network.ResDeleteLock
	err := lrepository.DeleteLock(reqDeleteLock.IndexID)
	if err != nil {
		resDeleteLock.Err = err.Error()
	}
	if err := json.NewEncoder(w).Encode(resDeleteLock); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// blobs
func getBlobs(w http.ResponseWriter, r *http.Request) {
	var reqGetBlobs network.ReqGetBlobs
	if err := json.NewDecoder(r.Body).Decode(&reqGetBlobs); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resGetBlobs network.ResGetBlobs
	checksums, err := lrepository.GetBlobs()
	if err != nil {
		resGetBlobs.Err = err.Error()
	} else {
		resGetBlobs.Checksums = checksums
	}
	if err := json.NewEncoder(w).Encode(resGetBlobs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func putBlob(w http.ResponseWriter, r *http.Request) {
	var reqPutBlob network.ReqPutBlob
	if err := json.NewDecoder(r.Body).Decode(&reqPutBlob); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resPutBlob network.ResPutBlob
	err := lrepository.PutBlob(reqPutBlob.Checksum, reqPutBlob.Data)
	if err != nil {
		resPutBlob.Err = err.Error()
	}
	if err := json.NewEncoder(w).Encode(resPutBlob); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func checkBlob(w http.ResponseWriter, r *http.Request) {
	var reqCheckBlob network.ReqCheckBlob
	if err := json.NewDecoder(r.Body).Decode(&reqCheckBlob); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resCheckBlob network.ResCheckBlob
	exists, err := lrepository.CheckBlob(reqCheckBlob.Checksum)
	if err != nil {
		resCheckBlob.Err = err.Error()
	} else {
		resCheckBlob.Exists = exists
	}
	if err := json.NewEncoder(w).Encode(resCheckBlob); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getBlob(w http.ResponseWriter, r *http.Request) {
	var reqGetBlob network.ReqGetBlob
	if err := json.NewDecoder(r.Body).Decode(&reqGetBlob); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resGetBlob network.ResGetBlob
	data, err := lrepository.GetBlob(reqGetBlob.Checksum)
	if err != nil {
		resGetBlob.Err = err.Error()
	} else {
		resGetBlob.Data = data
	}
	if err := json.NewEncoder(w).Encode(resGetBlob); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func deleteBlob(w http.ResponseWriter, r *http.Request) {
	if lNoDelete {
		http.Error(w, fmt.Errorf("not allowed to delete").Error(), http.StatusForbidden)
		return
	}

	var reqDeleteBlob network.ReqDeleteBlob
	if err := json.NewDecoder(r.Body).Decode(&reqDeleteBlob); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resDeleteBlob network.ResDeleteBlob
	err := lrepository.DeleteBlob(reqDeleteBlob.Checksum)
	if err != nil {
		resDeleteBlob.Err = err.Error()
	}
	if err := json.NewEncoder(w).Encode(resDeleteBlob); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// indexes
func getIndexes(w http.ResponseWriter, r *http.Request) {
	var reqGetIndexes network.ReqGetIndexes
	if err := json.NewDecoder(r.Body).Decode(&reqGetIndexes); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resGetIndexes network.ResGetIndexes
	indexes, err := lrepository.GetIndexes()
	if err != nil {
		resGetIndexes.Err = err.Error()
	} else {
		resGetIndexes.Checksums = indexes
	}
	if err := json.NewEncoder(w).Encode(resGetIndexes); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func putIndex(w http.ResponseWriter, r *http.Request) {
	var reqPutIndex network.ReqPutIndex
	if err := json.NewDecoder(r.Body).Decode(&reqPutIndex); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resPutIndex network.ResPutIndex
	err := lrepository.PutIndex(reqPutIndex.Checksum, reqPutIndex.Data)
	if err != nil {
		resPutIndex.Err = err.Error()
	}
	if err := json.NewEncoder(w).Encode(resPutIndex); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getIndex(w http.ResponseWriter, r *http.Request) {
	var reqGetIndex network.ReqGetIndex
	if err := json.NewDecoder(r.Body).Decode(&reqGetIndex); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resGetIndex network.ResGetIndex
	data, err := lrepository.GetIndex(reqGetIndex.Checksum)
	if err != nil {
		resGetIndex.Err = err.Error()
	} else {
		resGetIndex.Data = data
	}
	if err := json.NewEncoder(w).Encode(resGetIndex); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func deleteIndex(w http.ResponseWriter, r *http.Request) {
	if lNoDelete {
		http.Error(w, fmt.Errorf("not allowed to delete").Error(), http.StatusForbidden)
		return
	}

	var reqDeleteIndex network.ReqDeleteIndex
	if err := json.NewDecoder(r.Body).Decode(&reqDeleteIndex); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resDeleteIndex network.ResDeleteIndex
	err := lrepository.DeleteIndex(reqDeleteIndex.Checksum)
	if err != nil {
		resDeleteIndex.Err = err.Error()
	}
	if err := json.NewEncoder(w).Encode(resDeleteIndex); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// packfiles
func getPackfiles(w http.ResponseWriter, r *http.Request) {
	var reqGetPackfiles network.ReqGetPackfiles
	if err := json.NewDecoder(r.Body).Decode(&reqGetPackfiles); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resGetPackfiles network.ResGetPackfiles
	packfiles, err := lrepository.GetPackfiles()
	if err != nil {
		resGetPackfiles.Err = err.Error()
	} else {
		resGetPackfiles.Checksums = packfiles
	}
	if err := json.NewEncoder(w).Encode(resGetPackfiles); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func putPackfile(w http.ResponseWriter, r *http.Request) {
	var reqPutPackfile network.ReqPutPackfile
	if err := json.NewDecoder(r.Body).Decode(&reqPutPackfile); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resPutPackfile network.ResPutPackfile
	err := lrepository.PutPackfile(reqPutPackfile.Checksum, reqPutPackfile.Data)
	if err != nil {
		resPutPackfile.Err = err.Error()
	}
	if err := json.NewEncoder(w).Encode(resPutPackfile); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getPackfile(w http.ResponseWriter, r *http.Request) {
	var reqGetPackfile network.ReqGetPackfile
	if err := json.NewDecoder(r.Body).Decode(&reqGetPackfile); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resGetPackfile network.ResGetPackfile
	data, err := lrepository.GetPackfile(reqGetPackfile.Checksum)
	if err != nil {
		resGetPackfile.Err = err.Error()
	} else {
		resGetPackfile.Data = data
	}
	if err := json.NewEncoder(w).Encode(resGetPackfile); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func getPackfileSubpart(w http.ResponseWriter, r *http.Request) {
	var reqGetPackfileSubpart network.ReqGetPackfileSubpart
	if err := json.NewDecoder(r.Body).Decode(&reqGetPackfileSubpart); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resGetPackfileSubpart network.ResGetPackfileSubpart
	data, err := lrepository.GetPackfileSubpart(reqGetPackfileSubpart.Checksum, reqGetPackfileSubpart.Offset, reqGetPackfileSubpart.Length)
	if err != nil {
		resGetPackfileSubpart.Err = err.Error()
	} else {
		resGetPackfileSubpart.Data = data
	}
	if err := json.NewEncoder(w).Encode(resGetPackfileSubpart); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func deletePackfile(w http.ResponseWriter, r *http.Request) {
	if lNoDelete {
		http.Error(w, fmt.Errorf("not allowed to delete").Error(), http.StatusForbidden)
		return
	}

	var reqDeletePackfile network.ReqDeletePackfile
	if err := json.NewDecoder(r.Body).Decode(&reqDeletePackfile); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var resDeletePackfile network.ResDeletePackfile
	err := lrepository.DeletePackfile(reqDeletePackfile.Checksum)
	if err != nil {
		resDeletePackfile.Err = err.Error()
	}
	if err := json.NewEncoder(w).Encode(resDeletePackfile); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func Server(repository *storage.Repository, addr string, noDelete bool) error {

	lNoDelete = noDelete

	lrepository = repository
	network.ProtocolRegister()

	r := mux.NewRouter()
	r.HandleFunc("/", openRepository).Methods("GET")
	r.HandleFunc("/", closeRepository).Methods("POST")

	r.HandleFunc("/snapshots", getSnapshots).Methods("GET")
	r.HandleFunc("/snapshot", putSnapshot).Methods("PUT")
	r.HandleFunc("/snapshot", getSnapshot).Methods("GET")
	r.HandleFunc("/snapshot", deleteSnapshot).Methods("DELETE")
	r.HandleFunc("/snapshot", commitSnapshot).Methods("POST")

	r.HandleFunc("/locks", getLocks).Methods("GET")
	r.HandleFunc("/lock", putLock).Methods("PUT")
	r.HandleFunc("/lock", getLock).Methods("GET")
	r.HandleFunc("/lock", deleteLock).Methods("DELETE")

	r.HandleFunc("/blobs", getBlobs).Methods("GET")
	r.HandleFunc("/blob", putBlob).Methods("PUT")
	r.HandleFunc("/blob", getBlob).Methods("GET")
	r.HandleFunc("/blob/check", checkBlob).Methods("GET")
	r.HandleFunc("/blob", deleteBlob).Methods("DELETE")

	r.HandleFunc("/indexes", getIndexes).Methods("GET")
	r.HandleFunc("/index", putIndex).Methods("PUT")
	r.HandleFunc("/index", getIndex).Methods("GET")
	r.HandleFunc("/index", deleteIndex).Methods("DELETE")

	r.HandleFunc("/packfiles", getPackfiles).Methods("GET")
	r.HandleFunc("/packfile", putPackfile).Methods("PUT")
	r.HandleFunc("/packfile", getPackfile).Methods("GET")
	r.HandleFunc("/packfile/subpart", getPackfileSubpart).Methods("GET")
	r.HandleFunc("/packfile", deletePackfile).Methods("DELETE")

	return http.ListenAndServe(addr, r)
}
