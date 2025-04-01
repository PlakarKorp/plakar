package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"

	"github.com/PlakarKorp/plakar/cmd/plakar/utils"
	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/snapshot"
	"github.com/PlakarKorp/plakar/snapshot/header"
	"github.com/PlakarKorp/plakar/snapshot/vfs"
	"github.com/PlakarKorp/plakar/storage"
)

type RepositoryInfoSnapshots struct {
	Total int    `json:"total"`
	Size  uint64 `json:"size"`
}

type RepositoryInfoResponse struct {
	Location      string                  `json:"location"`
	Snapshots     RepositoryInfoSnapshots `json:"snapshots"`
	Configuration storage.Configuration   `json:"configuration"`
}

func repositoryInfo(w http.ResponseWriter, r *http.Request) error {

	configuration := lrepository.Configuration()

	snapshotIDs, err := utils.LocateSnapshotIDs(lrepository, nil)
	if err != nil {
		return fmt.Errorf("unable to locate snapshots: %w", err)
	}

	totalSize := uint64(0)
	for _, snapshotID := range snapshotIDs {
		snap, err := snapshot.Load(lrepository, snapshotID)
		if err != nil {
			return fmt.Errorf("unable to load snapshot: %w", err)
		}
		totalSize += snap.Header.GetSource(0).Summary.Directory.Size + snap.Header.GetSource(0).Summary.Below.Size
		snap.Close()
	}

	return json.NewEncoder(w).Encode(Item[RepositoryInfoResponse]{Item: RepositoryInfoResponse{
		Location: lrepository.Location(),
		Snapshots: RepositoryInfoSnapshots{
			Total: len(snapshotIDs),
			Size:  totalSize,
		},
		Configuration: configuration,
	}})
}

func repositorySnapshots(w http.ResponseWriter, r *http.Request) error {
	offset, err := QueryParamToUint32(r, "offset", 0, 0)
	if err != nil {
		return err
	}
	limit, err := QueryParamToUint32(r, "limit", 1, 50)
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

type TimelineLocation struct {
	Snapshot header.Header `json:"snapshot"`
	Entry    vfs.Entry     `json:"vfs_entry"`
}

func repositoryLocatePathname(w http.ResponseWriter, r *http.Request) error {
	offset, err := QueryParamToUint32(r, "offset", 0, 0)
	if err != nil {
		return err
	}
	limit, err := QueryParamToUint32(r, "limit", 1, 50)
	if err != nil {
		return err
	}

	importerType, _, err := QueryParamToString(r, "importerType")
	if err != nil {
		return err
	}

	importerOrigin, _, err := QueryParamToString(r, "importerOrigin")
	if err != nil {
		return err
	}

	resource, _, err := QueryParamToString(r, "resource")
	if err != nil {
		return err
	}

	sortKeys, err := QueryParamToSortKeys(r, "sort", "Timestamp")
	if err != nil {
		return err
	}
	_ = sortKeys

	lrepository.RebuildState()

	snapshotIDs, err := lrepository.GetSnapshots()
	if err != nil {
		return err
	}

	totalSnapshots := int(0)
	locations := make([]TimelineLocation, 0, len(snapshotIDs))
	for _, snapshotID := range snapshotIDs {
		snap, err := snapshot.Load(lrepository, snapshotID)
		if err != nil {
			return err
		}

		if importerType != "" && strings.ToLower(snap.Header.GetSource(0).Importer.Type) != strings.ToLower(importerType) {
			snap.Close()
			continue
		}

		if importerOrigin != "" && strings.ToLower(snap.Header.GetSource(0).Importer.Origin) != strings.ToLower(importerOrigin) {
			snap.Close()
			continue
		}

		path := snap.Header.GetSource(0).Importer.Directory
		if path != "/" {
			path = path + "/"
		}
		if !strings.HasPrefix(resource, path) {
			snap.Close()
			continue
		}

		pvfs, err := snap.Filesystem()
		if err != nil {
			snap.Close()
			continue
		}

		entry, err := pvfs.GetEntry(resource)
		if err != nil {
			snap.Close()
			continue
		}

		locations = append(locations, TimelineLocation{
			Snapshot: *snap.Header,
			Entry:    *entry,
		})
		totalSnapshots++
		snap.Close()
	}

	if limit == 0 {
		limit = uint32(len(locations))
	}

	sort.Slice(locations, func(i, j int) bool {
		return locations[i].Snapshot.Timestamp.Before(locations[j].Snapshot.Timestamp)
	})

	if offset > uint32(len(locations)) {
		locations = []TimelineLocation{}
	} else if offset+limit > uint32(len(locations)) {
		locations = locations[offset:]
	} else {
		locations = locations[offset : offset+limit]
	}

	items := Items[TimelineLocation]{
		Total: totalSnapshots,
		Items: make([]TimelineLocation, 0, len(locations)),
	}
	items.Items = append(items.Items, locations...)

	return json.NewEncoder(w).Encode(items)
}
