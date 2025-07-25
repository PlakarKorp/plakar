package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/PlakarKorp/kloset/objects"
	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/kloset/snapshot/header"
	"github.com/PlakarKorp/kloset/snapshot/vfs"
	"github.com/PlakarKorp/kloset/storage"
)

type RepositoryInfoSnapshots struct {
	Total           int     `json:"total"`
	StorageSize     int64   `json:"storage_size"`
	LogicalSize     int64   `json:"logical_size"`
	Efficiency      float64 `json:"efficiency"`
	SnapshotsPerDay []int   `json:"snapshots_per_day"`
}

type RepositoryInfoResponse struct {
	Location      string                  `json:"location"`
	Snapshots     RepositoryInfoSnapshots `json:"snapshots"`
	Configuration storage.Configuration   `json:"configuration"`
}

func getNSnapshotsPerDay(repo *repository.Repository, ndays int) ([]int, error) {
	nSnapshotsPerDay := make([]int, ndays)
	for snapshotID := range repo.ListSnapshots() {
		snap, err := snapshot.Load(repo, snapshotID)
		if err != nil {
			continue
		}
		if !snap.Header.Timestamp.Before(repo.Configuration().Timestamp.AddDate(0, 0, -ndays)) {
			dayIndex := time.Since(snap.Header.Timestamp).Hours() / 24
			if dayIndex < float64(ndays) {
				nSnapshotsPerDay[(ndays-1)-int(dayIndex)]++
			}
		}
		snap.Close()
	}

	return nSnapshotsPerDay, nil
}

func (ui *uiserver) repositoryInfo(w http.ResponseWriter, r *http.Request) error {
	nSnapshots, logicalSize, err := snapshot.LogicalSize(ui.repository)
	if err != nil {
		return fmt.Errorf("unable to calculate logical size: %w", err)
	}

	nSnapshotsPerDay, err := getNSnapshotsPerDay(ui.repository, 30)
	if err != nil {
		return fmt.Errorf("unable to calculate snapshots per day: %w", err)
	}

	efficiency := float64(0)
	storageSize := ui.repository.StorageSize()
	if storageSize == -1 || logicalSize == 0 {
		efficiency = -1
	} else {
		usagePercent := (float64(storageSize) / float64(logicalSize)) * 100
		if usagePercent <= 100 {
			savings := 100 - usagePercent
			efficiency = savings
		} else {
			increase := usagePercent - 100
			if increase > 100 {
				efficiency = -1
			} else {
				efficiency = -1 * increase
			}
		}
	}

	return json.NewEncoder(w).Encode(Item[RepositoryInfoResponse]{Item: RepositoryInfoResponse{
		Location: ui.repository.Location(),
		Snapshots: RepositoryInfoSnapshots{
			Total:           nSnapshots,
			StorageSize:     int64(ui.repository.StorageSize()),
			LogicalSize:     logicalSize,
			Efficiency:      efficiency,
			SnapshotsPerDay: nSnapshotsPerDay,
		},
		Configuration: ui.config,
	}})
}

func (ui *uiserver) repositorySnapshots(w http.ResponseWriter, r *http.Request) error {
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

	var sinceTime time.Time
	since, _, err := QueryParamToString(r, "since")
	if err != nil {
		return err
	} else {
		sinceTime, err = time.Parse(time.RFC3339, since)
		if err != nil && since != "" {
			return &ApiError{
				HttpCode: http.StatusBadRequest,
				ErrCode:  "invalid_argument",
				Message:  "Invalid 'since' parameter format. Expected RFC3339 format.",
			}
		}
	}

	sortKeys, err := QueryParamToSortKeys(r, "sort", "Timestamp")
	if err != nil {
		return err
	}

	ui.repository.RebuildState()

	snapshotIDs, err := ui.repository.GetSnapshots()
	if err != nil {
		return err
	}

	totalSnapshots := int(0)
	headers := make([]header.Header, 0, len(snapshotIDs))
	for _, snapshotID := range snapshotIDs {
		snap, err := snapshot.Load(ui.repository, snapshotID)
		if err != nil {
			return err
		}

		if importerType != "" && strings.ToLower(snap.Header.GetSource(0).Importer.Type) != strings.ToLower(importerType) {
			snap.Close()
			continue
		}

		if since != "" && snap.Header.Timestamp.Before(sinceTime) {
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

func (ui *uiserver) repositoryStates(w http.ResponseWriter, r *http.Request) error {
	states, err := ui.repository.GetStates()
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

func (ui *uiserver) repositoryState(w http.ResponseWriter, r *http.Request) error {
	stateBytes32, err := PathParamToID(r, "state")
	if err != nil {
		return err
	}

	_, rd, err := ui.repository.GetState(stateBytes32)
	if err != nil {
		return err
	}

	defer rd.Close()
	if _, err := io.Copy(w, rd); err != nil {
		log.Println("write failed:", err)
	}

	return nil
}

func (ui *uiserver) repositoryImporterTypes(w http.ResponseWriter, r *http.Request) error {
	ui.repository.RebuildState()

	snapshotIDs, err := ui.repository.GetSnapshots()
	if err != nil {
		return err
	}

	importerTypesMap := make(map[string]struct{})
	for _, snapshotID := range snapshotIDs {
		snap, err := snapshot.Load(ui.repository, snapshotID)
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

func (ui *uiserver) repositoryLocatePathname(w http.ResponseWriter, r *http.Request) error {
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

	importerDirectory, _, err := QueryParamToString(r, "importerDirectory")
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

	ui.repository.RebuildState()

	snapshotIDs, err := ui.repository.GetSnapshots()
	if err != nil {
		return err
	}

	totalSnapshots := int(0)
	locations := make([]TimelineLocation, 0, len(snapshotIDs))
	for _, snapshotID := range snapshotIDs {
		snap, err := snapshot.Load(ui.repository, snapshotID)
		if err != nil {
			return err
		}

		if importerType != "" && !strings.EqualFold(snap.Header.GetSource(0).Importer.Type, importerType) {
			snap.Close()
			continue
		}

		if importerOrigin != "" && !strings.EqualFold(snap.Header.GetSource(0).Importer.Origin, importerOrigin) {
			snap.Close()
			continue
		}

		if importerDirectory != "" && !strings.EqualFold(snap.Header.GetSource(0).Importer.Directory, importerDirectory) {
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

	sortFunc := func(a, b TimelineLocation) int {
		if a.Snapshot.Timestamp.Before(b.Snapshot.Timestamp) {
			return -1
		}
		if a.Snapshot.Timestamp.After(b.Snapshot.Timestamp) {
			return 1
		}
		return 0
	}

	if len(sortKeys) > 0 {
		switch sortKeys[0] {
		case "-Timestamp":
			sortFunc = func(a, b TimelineLocation) int {
				if a.Snapshot.Timestamp.After(b.Snapshot.Timestamp) {
					return -1
				}
				if a.Snapshot.Timestamp.Before(b.Snapshot.Timestamp) {
					return 1
				}
				return 0
			}
		}
	}

	slices.SortFunc(locations, sortFunc)

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
