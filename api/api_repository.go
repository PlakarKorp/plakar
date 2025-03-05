package api

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/snapshot"
	"github.com/PlakarKorp/plakar/snapshot/header"
)

func repositoryConfiguration(w http.ResponseWriter, r *http.Request) error {
	configuration := lrepository.Configuration()
	return json.NewEncoder(w).Encode(configuration)
}

func repositorySnapshots(w http.ResponseWriter, r *http.Request) error {
	offset, _, err := QueryParamToUint32(r, "offset")
	if err != nil {
		return err
	}
	limit, _, err := QueryParamToUint32(r, "limit")
	if err != nil {
		return err
	}

	importerType, _, err := QueryParamToString(r, "importer")
	if err != nil {
		return err
	}

	sortKeys, err := QueryParamToSortKeys(r, "sort", "Timestamp")
	if err != nil {
		return err
	}

	lrepository.RebuildState()

	snapshotIDs, err := lrepository.GetSnapshots()
	if err != nil {
		return err
	}

	totalSnapshots := int(0)
	headers := make([]header.Header, 0, len(snapshotIDs))
	for _, snapshotID := range snapshotIDs {
		snap, err := snapshot.Load(lrepository, snapshotID)
		if err != nil {
			return err
		}

		if importerType != "" && strings.ToLower(snap.Header.GetSource(0).Importer.Type) != strings.ToLower(importerType) {
			snap.Close()
			continue
		}

		headers = append(headers, *snap.Header)
		totalSnapshots++
		snap.Close()
	}

	if limit == 0 {
		limit = uint32(len(headers))
	}

	header.SortHeaders(headers, sortKeys)
	if offset > uint32(len(headers)) {
		headers = []header.Header{}
	} else if offset+limit > uint32(len(headers)) {
		headers = headers[offset:]
	} else {
		headers = headers[offset : offset+limit]
	}

	items := Items[header.Header]{
		Total: totalSnapshots,
		Items: make([]header.Header, len(headers)),
	}
	for i, header := range headers {
		items.Items[i] = header
	}

	return json.NewEncoder(w).Encode(items)
}

func repositoryStates(w http.ResponseWriter, r *http.Request) error {
	states, err := lrepository.GetStates()
	if err != nil {
		return err
	}

	items := Items[objects.MAC]{
		Total: len(states),
		Items: make([]objects.MAC, len(states)),
	}
	for i, state := range states {
		items.Items[i] = state
	}

	return json.NewEncoder(w).Encode(items)
}

func repositoryState(w http.ResponseWriter, r *http.Request) error {
	stateBytes32, err := PathParamToID(r, "state")
	if err != nil {
		return err
	}

	_, rd, err := lrepository.GetState(stateBytes32)
	if err != nil {
		return err
	}

	if _, err := io.Copy(w, rd); err != nil {
		log.Println("write failed:", err)
	}
	return nil
}

func repositoryImporterTypes(w http.ResponseWriter, r *http.Request) error {
	lrepository.RebuildState()

	snapshotIDs, err := lrepository.GetSnapshots()
	if err != nil {
		return err
	}

	importerTypesMap := make(map[string]struct{})
	for _, snapshotID := range snapshotIDs {
		snap, err := snapshot.Load(lrepository, snapshotID)
		if err != nil {
			return err
		}
		importerTypesMap[strings.ToLower(snap.Header.GetSource(0).Importer.Type)] = struct{}{}
	}

	importerTypes := make([]string, 0, len(importerTypesMap))
	for importerType := range importerTypesMap {
		importerTypes = append(importerTypes, importerType)
	}
	sort.Slice(importerTypes, func(i, j int) bool {
		return importerTypes[i] < importerTypes[j]
	})

	type Entry struct {
		Name string `json:"name"`
	}

	items := Items[Entry]{
		Total: len(importerTypes),
		Items: make([]Entry, len(importerTypes)),
	}
	for i, importerType := range importerTypes {
		items.Items[i] = Entry{Name: importerType}
	}

	return json.NewEncoder(w).Encode(items)
}
