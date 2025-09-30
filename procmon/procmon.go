package procmon

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/PlakarKorp/kloset/events"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/shirou/gopsutil/v3/process"
)

type Sample struct {
	TS         time.Time
	CPUPercent float64
	RSSBytes   uint64
}

type Marker struct {
	Label string
	TS    time.Time
	Color string // optional (e.g. "red", "green", "blue", "orange", "purple", "grey")
}

const (
	sampleInterval = 100 * time.Millisecond
	maxSamples     = 10_000 // ring cap (keeps memory bounded)
	maxMarkers     = 10_000
)

var (
	mu       sync.RWMutex
	samples  = make([]Sample, 0, 1024)
	markers  = make([]Marker, 0, 64)
	running  bool
	stopOnce sync.Once
	stopCh   chan struct{}
)

// Start begins sampling the *current process*. Call the returned function to stop.
func Start(ctx *appcontext.AppContext) func() {
	if running {
		// already sampling; return a no-op stopper
		return func() {}
	}
	running = true
	stopCh = make(chan struct{})

	// Prepare process handle and prime CPU% baseline
	var proc *process.Process
	if p, err := process.NewProcess(int32(os.Getpid())); err == nil {
		proc = p
		_, _ = proc.Percent(0)
	}

	go func() {
		eventslistener := ctx.Events().Listen()
		for {
			select {
			case <-stopCh:
				return
			case evt := <-eventslistener:
				switch evt.(type) {
				case events.Start, events.Done, events.StartImporter, events.DoneImporter:
					MarkAt(time.Now(), fmt.Sprintf("%T", evt))

				default:
					//	fmt.Println("procmon: unhandled event type:", fmt.Sprintf("%T", evt)) // DEBUG
				}
			}
		}
	}()

	go func() {

		t := time.NewTicker(sampleInterval)
		defer t.Stop()
		for {
			select {
			case <-stopCh:
				return

			case now := <-t.C:
				var cpu float64
				var rss uint64
				if proc != nil {
					if v, err := proc.Percent(0); err == nil {
						cpu = v
					}
					if mi, err := proc.MemoryInfo(); err == nil && mi != nil {
						rss = mi.RSS
					}
				}
				s := Sample{
					TS:         now.UTC(),
					CPUPercent: cpu,
					RSSBytes:   rss,
				}
				mu.Lock()
				if len(samples) < maxSamples {
					samples = append(samples, s)
				} else {
					copy(samples, samples[1:])
					samples[len(samples)-1] = s
				}
				mu.Unlock()

				if hub != nil { // NEW: push live sample
					hub.broadcast("sample", s)
				}
			}
		}
	}()

	return func() {
		stopOnce.Do(func() {
			close(stopCh)
			running = false
		})
	}
}

// Samples returns a copy of all collected samples so far.
func Samples() []Sample {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Sample, len(samples))
	copy(out, samples)
	return out
}

// ---- simple marker API (instant-only) ----

// MarkNow records a marker at the current time (absolute).
func MarkNow(label string, color ...string) {
	MarkAt(time.Now().UTC(), label, color...)
}

// MarkAt records a marker at an explicit absolute time.
func MarkAt(ts time.Time, label string, color ...string) {
	m := Marker{
		Label: label,
		TS:    ts.UTC(),
	}
	if len(color) > 0 {
		m.Color = color[0]
	}
	mu.Lock()
	if len(markers) < maxMarkers {
		markers = append(markers, m)
	} else {
		copy(markers, markers[1:])
		markers[len(markers)-1] = m
	}
	mu.Unlock()
	if hub != nil { // NEW: push live marker (if you later add markers to the UI)
		hub.broadcast("mark", m)
	}
}

// Markers returns a copy of all instant markers so far.
func Markers() []Marker {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Marker, len(markers))
	copy(out, markers)
	return out
}
