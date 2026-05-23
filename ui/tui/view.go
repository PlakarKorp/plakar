package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/muesli/termenv"
)

// ── styles ────────────────────────────────────────────────────────────────────

var (
	crossMark = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).SetString("✘")
	okStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	errStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red
	dimStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // gray
	boldStyle = lipgloss.NewStyle().Bold(true)
)

// ── helpers ───────────────────────────────────────────────────────────────────

func humanDuration(d time.Duration) string {
	sec := int(d.Round(time.Second).Seconds())
	if sec < 0 {
		sec = 0
	}
	h := sec / 3600
	m := (sec % 3600) / 60
	s := sec % 60
	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

func progressBar() progress.Model {
	p := progress.New(
		progress.WithColorProfile(termenv.Ascii),
	)
	p.Full = '█'
	p.Empty = '░'
	p.ShowPercentage = false
	return p
}

func fmtETA(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
}

func formatBytes(b int64) string {
	if b <= 0 {
		return "0 B"
	}
	return humanize.IBytes(uint64(b))
}

func truncateLeft(s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxW {
		return s
	}
	if maxW <= 3 {
		return strings.Repeat(".", maxW)
	}
	rs := []rune(s)
	tail := make([]rune, 0, len(rs))
	w := 0
	for i := len(rs) - 1; i >= 0; i-- {
		rw := lipgloss.Width(string(rs[i]))
		if w+rw > maxW-3 {
			break
		}
		tail = append(tail, rs[i])
		w += rw
	}
	for i, j := 0, len(tail)-1; i < j; i, j = i+1, j-1 {
		tail[i], tail[j] = tail[j], tail[i]
	}
	return "..." + string(tail)
}

func shortenPathTailMax(path string, maxW int) string {
	if maxW <= 0 || path == "" {
		return ""
	}
	if lipgloss.Width(path) <= maxW {
		return path
	}
	sep := string(filepath.Separator)
	p := strings.ReplaceAll(strings.ReplaceAll(path, "\\", sep), "/", sep)
	vol := filepath.VolumeName(p)
	if vol != "" {
		p = strings.TrimPrefix(strings.TrimPrefix(p, vol), sep)
	}
	if len(p) > 1 {
		p = strings.TrimRight(p, sep)
	}
	parts := strings.FieldsFunc(p, func(r rune) bool { return string(r) == sep })
	if len(parts) == 0 {
		out := vol
		if out == "" {
			out = path
		}
		return truncateLeft(out, maxW)
	}
	join := func(dots bool, tail []string) string {
		body := strings.Join(tail, sep)
		if dots {
			if vol != "" {
				return vol + sep + "..." + sep + body
			}
			return "..." + sep + body
		}
		if vol != "" {
			return vol + sep + body
		}
		return body
	}
	tail := []string{parts[len(parts)-1]}
	best := join(true, tail)
	if lipgloss.Width(best) > maxW {
		file := parts[len(parts)-1]
		prefix := "..." + sep
		avail := maxW - lipgloss.Width(prefix)
		if avail <= 0 {
			return truncateLeft(prefix+file, maxW)
		}
		return prefix + truncateLeft(file, avail)
	}
	for i := len(parts) - 2; i >= 0; i-- {
		cand := join(true, append([]string{parts[i]}, tail...))
		if lipgloss.Width(cand) <= maxW {
			tail = append([]string{parts[i]}, tail...)
			best = cand
		} else {
			break
		}
	}
	if full := join(false, parts); lipgloss.Width(full) <= maxW {
		return full
	}
	return best
}

// rightAlign returns a string of exactly width w with left padded so that
// content sits at the right edge.
func rightAlign(content string, w int) string {
	cw := lipgloss.Width(content)
	if cw >= w {
		return content
	}
	return strings.Repeat(" ", w-cw) + content
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m appModel) View() string {
	state := m.application.state
	elapsed := time.Since(state.startTime)

	if m.forceQuit {
		return fmt.Sprintf("[%s] %s: aborted\n",
			humanDuration(elapsed), m.application.name)
	}

	termW := m.width
	if termW <= 0 {
		termW = 80
	}

	var s strings.Builder

	// ── progress computation ─────────────────────────────────────────────────
	//
	// Prefer byte-driven progress when fs.summary carried a total size
	// (currently populated for export/check). Falls back to object count
	// otherwise. Byte-driven progress better reflects time-to-completion
	// when item sizes vary widely (one 5 GiB file vs 10k config files).

	// object-driven done/total (PathOk already includes cached items —
	// don't add countPathCached or it double-counts).
	objDone := state.countPathOk + state.countPathError
	objTotal := state.summaryPath

	// byte-driven done/total. sourceReadBytes accumulates from
	// import.progress; sourceWriteBytes from export.progress — use
	// whichever is non-zero (only one is active per workflow).
	byteDone := state.sourceReadBytes
	if state.sourceWriteBytes > byteDone {
		byteDone = state.sourceWriteBytes
	}
	byteTotal := int64(state.summarySize)

	useBytes := state.gotSummary && byteTotal > 0
	hasSummary := useBytes || (state.gotSummary && objTotal > 0)

	var done, total uint64
	if useBytes {
		done = uint64(byteDone)
		total = uint64(byteTotal)
	} else {
		done = objDone
		total = objTotal
	}

	ratio := 0.0
	pct := 0
	if hasSummary && total > 0 {
		ratio = float64(done) / float64(total)
		if ratio < 0 {
			ratio = 0
		} else if ratio >= 1 {
			// Don't show 100% until workflow.end confirms completion —
			// fs.summary totals can be approximate and counts can exceed
			// them before all post-processing (VFS build, index, commit)
			// is complete.
			if state.finished {
				ratio = 1
			} else {
				ratio = 0.99
			}
		}
		pct = int(ratio * 100)
	}

	// ETA: derive from the same units we're using for the bar.
	etaText := ""
	if hasSummary && elapsed > 2*time.Second && total > done {
		var rate float64
		if useBytes {
			// Pick the matching byte rate.
			if state.sourceReadBytes > 0 {
				rate = state.sourceReadRate
			} else {
				rate = state.sourceWriteRate
			}
		} else {
			rate = m.rateEMA
		}
		if rate > 0 {
			remaining := float64(total - done)
			etaDur := time.Duration(remaining / rate * float64(time.Second))
			if v := fmtETA(etaDur); v != "" {
				etaText = "ETA " + v
			}
		}
	}

	// indent used on lines 2+ (aligns with the content after "[HH:MM] ")
	timerLabel := "[" + humanDuration(elapsed) + "]"
	timerW := lipgloss.Width(timerLabel)
	indent := strings.Repeat(" ", timerW+1)

	// ── line 1: [HH:MM] snapID  phase ───────────────────────────────────────

	header := boldStyle.Render(timerLabel) + " " + dimStyle.Render(state.snapshotID)
	if state.phase != "" {
		header += "  " + state.phase
	}
	fmt.Fprintln(&s, header)

	// ── line 2: path (left) + size (right) ───────────────────────────────────
	//
	// When the importer has moved past the per-file phase (lastItem is
	// cleared on snapshot.vfs.start / index.start / commit.start) we
	// show a "<phase>…" placeholder so the line isn't blank for several
	// seconds during VFS/index/commit.

	sizeText := humanize.IBytes(uint64(state.countFileSize))
	sizeW := lipgloss.Width(sizeText)

	item := state.lastItem
	if item == "" && state.phase != "" {
		item = dimStyle.Render(state.phase + "…")
	}
	availPath := termW - timerW - 1 - sizeW - 1
	if availPath < 0 {
		availPath = 0
	}
	// Only path-shorten real paths, not the dim phase placeholder.
	if state.lastItem != "" {
		item = shortenPathTailMax(item, availPath)
	}
	itemW := lipgloss.Width(item)

	pad := availPath - itemW
	if pad < 0 {
		pad = 0
	}
	fmt.Fprintf(&s, "%s%s%s%s\n", indent, item, strings.Repeat(" ", pad), sizeText)

	// ── line 3: progress bar + pct (left) + ETA (right) ─────────────────────
	//
	// When we don't yet have a total to compute progress against
	// (fs.summary not received), render nothing — an empty static bar
	// for tens of seconds is worse than no bar at all. The timer in the
	// header already shows that work is in flight.

	if hasSummary {
		pctLabel := fmt.Sprintf(" %3d%%", pct)
		etaLabel := ""
		if etaText != "" {
			etaLabel = dimStyle.Render(etaText)
		}

		pctW := lipgloss.Width(pctLabel)
		etaW := lipgloss.Width(etaLabel)
		barW := termW - timerW - 1 - pctW - etaW
		if etaW > 0 {
			barW-- // space before ETA
		}
		if barW < 4 {
			barW = 4
		}

		p := m.progress
		p.Width = barW
		bar := p.ViewAs(ratio)

		barLine := indent + bar + pctLabel
		if etaLabel != "" {
			barLineW := lipgloss.Width(barLine)
			gap := termW - barLineW - etaW
			if gap < 1 {
				gap = 1
			}
			barLine += strings.Repeat(" ", gap) + etaLabel
		}
		fmt.Fprintln(&s, barLine)
	}

	// ── line 4: nodes / objects / errors ─────────────────────────────────────
	// nodes   = directory entries (structural)
	// objects = files + symlinks + xattrs (actual data objects)

	var statParts []string

	// nodes (dirs)
	statParts = append(statParts,
		dimStyle.Render("nodes: ")+okStyle.Render(fmt.Sprintf("%d", state.countDirOk)))

	// objects: done/total or just done
	objectsDone := state.countFileOk + state.countSymlinkOk + state.countXattrOk
	if hasSummary {
		objectsTotal := state.summaryFile + state.summarySymlink + state.summaryXattr
		statParts = append(statParts,
			dimStyle.Render("objects: ")+
				okStyle.Render(fmt.Sprintf("%d", objectsDone))+
				dimStyle.Render(fmt.Sprintf("/%d", objectsTotal)))
	} else {
		statParts = append(statParts,
			dimStyle.Render("objects: ")+okStyle.Render(fmt.Sprintf("%d", objectsDone)))
	}

	// cached
	cached := state.countFileCached + state.countDirCached + state.countSymlinkCached + state.countXattrCached
	if cached > 0 {
		statParts = append(statParts,
			dimStyle.Render("cached: ")+dimStyle.Render(fmt.Sprintf("%d", cached)))
	}

	// errors
	if state.countPathError > 0 {
		statParts = append(statParts,
			dimStyle.Render("errors: ")+errStyle.Render(fmt.Sprintf("%d", state.countPathError)))
	} else {
		statParts = append(statParts, dimStyle.Render("errors: 0"))
	}

	fmt.Fprintln(&s, indent+strings.Join(statParts, dimStyle.Render("   ")))

	// ── line 5: connector and store throughput ──────────────────────────────
	//
	// For backup  (import workflow):
	//   importer  in 142 MiB/s  out 0 B/s     store  in 12 MiB/s  out 89 MiB/s
	// For restore (export workflow):
	//   exporter  in 0 B/s      out 95 MiB/s  store  in 110 MiB/s out 0 B/s
	//
	// importer-in  = import.progress  (bytes read from source filesystem)
	// exporter-out = export.progress  (bytes written to destination filesystem)
	// store-in     = store.read.progress  (bytes read from the store)
	// store-out    = store.write.progress (bytes written to the store)

	connectorLabel := "importer"
	if m.application.name == "export" {
		connectorLabel = "exporter"
	}

	fmtRate := func(r float64) string {
		return fmt.Sprintf("%s/s", humanize.IBytes(uint64(r)))
	}

	hasGranular := state.sourceReadBytes > 0 || state.sourceWriteBytes > 0 ||
		state.storeReadBytes > 0 || state.storeWriteBytes > 0

	if hasGranular {
		connIn := fmtRate(state.sourceReadRate)
		connOut := fmtRate(state.sourceWriteRate)
		storeIn := fmtRate(state.storeReadRate)
		storeOut := fmtRate(state.storeWriteRate)

		ioLine := dimStyle.Render(connectorLabel) +
			"  " + dimStyle.Render("in ") + connIn +
			"  " + dimStyle.Render("out ") + connOut +
			"     " +
			dimStyle.Render("store") +
			"  " + dimStyle.Render("in ") + storeIn +
			"  " + dimStyle.Render("out ") + storeOut

		fmt.Fprintln(&s, indent+ioLine)
	} else if m.repo != nil {
		// fallback: poll repo stats totals when granular events not yet flowing
		if time.Since(m.application.debounceStat) >= time.Second {
			io := m.repo.IOStats()
			r := io.Read.Stats()
			w := io.Write.Stats()
			m.application.lastStat = dimStyle.Render("store") +
				"  " + dimStyle.Render("in ") + humanize.IBytes(uint64(r.TotalBytes)) +
				"  " + dimStyle.Render("out ") + humanize.IBytes(uint64(w.TotalBytes))
			m.application.debounceStat = time.Now()
		}
		if m.application.lastStat != "" {
			fmt.Fprintln(&s, indent+m.application.lastStat)
		}
	}

	// ── errors (below, bounded by terminal height) ────────────────────────────

	if len(state.errors) > 0 {
		fmt.Fprintln(&s)
		used := strings.Count(s.String(), "\n") + 2
		avail := m.height - used
		if avail < 1 {
			avail = 1
		}
		if avail > len(state.errors) {
			avail = len(state.errors)
		}
		start := len(state.errors) - avail
		for _, e := range state.errors[start:] {
			fmt.Fprintln(&s, e)
		}
		if start > 0 {
			fmt.Fprintln(&s, dimStyle.Render(fmt.Sprintf(
				"  … %d more — run `plakar info -errors %s`",
				start, state.snapshotID)))
		}
	}

	return s.String()
}
