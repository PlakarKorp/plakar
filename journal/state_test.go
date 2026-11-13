package journal

import (
	"testing"
	"time"

	"github.com/PlakarKorp/kloset/events"
	"github.com/PlakarKorp/kloset/objects"
)

func ev(typ string, data map[string]any) *events.Event {
	return &events.Event{Type: typ, Data: data}
}

func TestUpdateCounters(t *testing.T) {
	var s State
	s.Update(ev("file", nil))
	s.Update(ev("file.ok", map[string]any{
		"fileinfo": objects.NewFileInfo("a", 100, 0, time.Time{}, 0, 0, 0, 0, 0),
	}))
	s.Update(ev("file.cached", map[string]any{
		"fileinfo": objects.NewFileInfo("b", 50, 0, time.Time{}, 0, 0, 0, 0, 0),
	}))
	s.Update(ev("directory", nil))
	s.Update(ev("directory.ok", nil))

	if s.Files.Total != 1 || s.Files.Ok != 1 || s.Files.Cached != 1 {
		t.Fatalf("files = %+v", s.Files)
	}
	if s.Files.Size != 100 || s.Files.CachedSize != 50 {
		t.Fatalf("files sizes = %+v", s.Files)
	}
	if s.Dirs.Total != 1 || s.Dirs.Ok != 1 {
		t.Fatalf("dirs = %+v", s.Dirs)
	}
}

func TestUpdatePhaseAndResult(t *testing.T) {
	var s State
	s.Update(ev("snapshot.import.start", nil))
	if s.Phase != "processing" {
		t.Fatalf("phase = %q", s.Phase)
	}

	s.Update(ev("result", map[string]any{
		"size":     uint64(2048),
		"errors":   uint64(1),
		"duration": 3 * time.Second,
	}))
	if s.Phase != "completed" {
		t.Fatalf("phase = %q", s.Phase)
	}
	if s.Result.Size != 2048 || s.Result.Errors != 1 || s.Result.Duration != 3.0 {
		t.Fatalf("result = %+v", s.Result)
	}
}

func TestUpdateIostats(t *testing.T) {
	var s State
	s.Update(ev("iostats", map[string]any{
		"scope": "source",
		"r":     map[string]any{"total": int64(1000), "overall": 2000.0, "overall_wall": 3000.0},
		"w":     map[string]any{"total": int64(0)},
	}))
	src, ok := s.IO["source"]
	if !ok {
		t.Fatal("no source scope")
	}
	if src.Read.TotalBytes != 1000 || src.Read.Overall != 2000.0 || src.Read.OverallWall != 3000.0 {
		t.Fatalf("read = %+v", src.Read)
	}

	s.Update(ev("iostats", map[string]any{"r": map[string]any{"total": int64(5)}}))
	if len(s.IO) != 1 {
		t.Fatalf("a scopeless iostats event was folded: %v", s.IO)
	}
}

// Processed prefers the per-file counters and falls back to the scan summary.
func TestProcessedFallback(t *testing.T) {
	s := State{}
	s.Files.Ok, s.Files.Size = 3, 300
	s.Files.Cached, s.Files.CachedSize = 2, 200
	if items, bytes := s.processed(); items != 5 || bytes != 500 {
		t.Fatalf("processed = %d, %d", items, bytes)
	}

	s = State{}
	s.Summary.Files, s.Summary.Size = 7, 700
	if items, bytes := s.processed(); items != 7 || bytes != 700 {
		t.Fatalf("processed = %d, %d", items, bytes)
	}
}

func TestRecentPaths(t *testing.T) {
	var s State
	for _, p := range []string{"/a", "/b", "/c"} {
		s.Update(ev("path", map[string]any{"path": p}))
	}
	s.Update(ev("path.ok", map[string]any{"path": "/a"}))
	s.Update(ev("path.error", map[string]any{"path": "/c"}))

	want := []RecentPath{
		{Path: "/b", Status: "in-progress"},
		{Path: "/c", Status: "error"},
		{Path: "/a", Status: "ok"},
	}
	if len(s.RecentPaths) != len(want) {
		t.Fatalf("recent = %+v", s.RecentPaths)
	}
	for i := range want {
		if s.RecentPaths[i] != want[i] {
			t.Fatalf("recent[%d] = %+v, want %+v", i, s.RecentPaths[i], want[i])
		}
	}

	for i := range 2 * maxRecentPaths {
		s.Update(ev("path", map[string]any{"path": string(rune('A' + i))}))
	}
	if len(s.RecentPaths) != maxRecentPaths {
		t.Fatalf("recent paths not capped: %d", len(s.RecentPaths))
	}
}

func TestCloneOwnsReferences(t *testing.T) {
	var s State
	s.Update(ev("iostats", map[string]any{"scope": "x", "r": map[string]any{}, "w": map[string]any{}}))
	s.Update(ev("path", map[string]any{"path": "/p"}))

	c := s.Clone()
	s.Update(ev("iostats", map[string]any{"scope": "y", "r": map[string]any{}, "w": map[string]any{}}))
	s.Update(ev("path.ok", map[string]any{"path": "/p"}))

	if len(c.IO) != 1 || c.RecentPaths[0].Status != "in-progress" {
		t.Fatalf("clone shares state: io=%v recent=%v", c.IO, c.RecentPaths)
	}
}
