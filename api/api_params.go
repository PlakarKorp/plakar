package api

import (
	"encoding/hex"
	"net/http"
	"strconv"

	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/repository"
	"github.com/PlakarKorp/plakar/snapshot/header"
)

// Parse a URL parameter with the format "snapshotID:path".
func SnapshotPathParam(r *http.Request, repo *repository.Repository, param string) (objects.MAC, string, error) {
	idstr, path := utils.ParseSnapshotID(r.PathValue(param))

	if idstr == "" {
		return objects.MAC{}, "", parameterError(param, MissingArgument, ErrMissingField)
	}

	mac, err := utils.LocateSnapshotByPrefix(repo, idstr)
	if err != nil {
		return objects.MAC{}, "", parameterError(param, InvalidArgument, err)
	}
	return mac, path, nil
}

func PathParamToID(r *http.Request, param string) (id [32]byte, err error) {
	idstr := r.PathValue(param)

	if idstr == "" {
		return id, parameterError(param, MissingArgument, ErrMissingField)
	}

	b, err := hex.DecodeString(idstr)
	if err != nil {
		return id, parameterError(param, InvalidArgument, err)
	}

	if len(b) != 32 {
		return id, parameterError(param, InvalidArgument, ErrInvalidID)
	}

	copy(id[:], b)
	return id, nil
}

func QueryParamToUint32(r *http.Request, param string) (uint32, bool, error) {
	str := r.URL.Query().Get(param)
	if str == "" {
		return 0, false, nil
	}

	n, err := strconv.ParseInt(str, 10, 32)
	if err != nil {
		return 0, true, err
	}

	if n < 0 {
		return 0, true, parameterError(param, BadNumber, ErrNegativeNumber)
	}

	return uint32(n), true, nil
}

func QueryParamToInt64(r *http.Request, param string) (int64, bool, error) {
	str := r.URL.Query().Get(param)
	if str == "" {
		return 0, false, nil
	}

	n, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, true, err
	}

	return n, true, nil
}

func QueryParamToString(r *http.Request, param string) (string, bool, error) {
	str := r.URL.Query().Get(param)
	if str == "" {
		return "", false, nil
	}

	return str, true, nil
}

func QueryParamToSortKeys(r *http.Request, param, def string) ([]string, error) {
	str := r.URL.Query().Get(param)
	if str == "" {
		str = def
	}

	sortKeys, err := header.ParseSortKeys(str)
	if err != nil {
		return []string{}, parameterError(param, InvalidArgument, err)
	}

	return sortKeys, nil
}
