package api

import (
	"encoding/hex"
	"net/http"
	"strconv"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot/header"
	"github.com/PlakarKorp/plakar/locate"
	"github.com/PlakarKorp/plakar/utils"
)

// Parse a URL parameter with the format "snapshotID:path".
func SnapshotPathParam(r *http.Request, repo *repository.Repository, param string) (objects.MAC, string, error) {
	idstr, path := utils.ParseSnapshotID(r.PathValue(param))

	if idstr == "" {
		return objects.MAC{}, "", parameterError(param, MissingArgument, ErrMissingField)
	}

	mac, err := locate.LocateSnapshotByPrefix(repo, idstr)
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

func QueryParamToUint32(r *http.Request, param string, min, def uint32) (uint32, error) {
	str := r.URL.Query().Get(param)
	if str == "" {
		return def, nil
	}

	n, err := strconv.ParseInt(str, 10, 32)
	if err != nil {
		return 0, parameterError(param, BadNumber, err)
	}

	if n < 0 || uint32(n) < min {
		return 0, parameterError(param, BadNumber, ErrNumberOutOfRange)
	}

	return uint32(n), nil
}

func QueryParamToInt64(r *http.Request, param string, min, def int64) (int64, error) {
	str := r.URL.Query().Get(param)
	if str == "" {
		return def, nil
	}

	n, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, parameterError(param, BadNumber, err)
	}

	if n < min {
		return 0, parameterError(param, BadNumber, ErrNumberOutOfRange)
	}

	return n, nil
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
