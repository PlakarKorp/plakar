package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
	"github.com/muesli/termenv"
)

// noColor reports whether color output is disabled, honouring the NO_COLOR
// convention (https://no-color.org): any non-empty value disables color.
func noColor() bool { return os.Getenv("NO_COLOR") != "" }

func init() {
	// Honour NO_COLOR globally: strip color from every lipgloss style, not just
	// the progress bar. The bar additionally switches to a plain ASCII glyph set
	// (see renderBar).
	if noColor() {
		lipgloss.SetColorProfile(termenv.Ascii)
	}
}

var (
	crossMark = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).SetString("✘")
	okStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	errStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red
	dimStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // gray (optional)

	// Modernized palette (256-color, degrades to the nearest basic color on
	// limited terminals via lipgloss).
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245")) // muted gray labels
	valueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252")) // near-white values
	titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	rateStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	spinStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	doneStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))  // green check
	downStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))  // ↓ read (green)
	upStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // ↑ write (amber)

	// Progress bar block styles. The fill is green; the portion representing
	// files that errored is red so problems are visible in the bar itself.
	barFillStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))  // green
	barErrStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // red
	barEmptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("238")) // dim
)

// statsSep separates fields on the stats and I/O lines.
var statsSep = "  " + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("│") + "  "

// alignColumns joins each row's cells with sep, padding every cell to the widest
// cell in its column (across all rows) so the separators line up vertically. The
// last cell of a row is not padded. Cells may contain ANSI styling; widths are
// measured with lipgloss.Width.
func alignColumns(rows [][]string, sep string) []string {
	widths := map[int]int{}
	for _, r := range rows {
		for i, cell := range r {
			if i == len(r)-1 {
				continue // don't pad trailing cell
			}
			if wdt := lipgloss.Width(cell); wdt > widths[i] {
				widths[i] = wdt
			}
		}
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		var b strings.Builder
		for i, cell := range r {
			if i > 0 {
				b.WriteString(sep)
			}
			b.WriteString(cell)
			if i < len(r)-1 {
				if pad := widths[i] - lipgloss.Width(cell); pad > 0 {
					b.WriteString(strings.Repeat(" ", pad))
				}
			}
		}
		out = append(out, b.String())
	}
	return out
}

// spinnerFrames is a braille spinner; index it by elapsed time.
var spinnerFrames = []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}

func spinnerFrame(elapsed time.Duration) rune {
	return spinnerFrames[int(elapsed/(100*time.Millisecond))%len(spinnerFrames)]
}

func err(n uint64) string { return errStyle.Render(fmt.Sprintf("%d", n)) }

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

// renderBar draws a progress bar of the given cell width. ratio in [0,1] is the
// overall fill; errRatio in [0,1] is the fraction of that fill to paint red
// (proportional to files that errored) so problems show up in the bar itself.
//
// With NO_COLOR set it falls back to the classic ASCII bar: [****    ].
func renderBar(width int, ratio, errRatio float64) string {
	if width < 1 {
		width = 1
	}
	clamp := func(f float64) float64 {
		if f < 0 {
			return 0
		}
		if f > 1 {
			return 1
		}
		return f
	}
	ratio, errRatio = clamp(ratio), clamp(errRatio)

	if noColor() {
		// Classic ASCII bar inside brackets; width includes the two brackets.
		inner := width - 2
		if inner < 1 {
			inner = 1
		}
		filled := int(ratio*float64(inner) + 0.5)
		return "[" + strings.Repeat("*", filled) + strings.Repeat(" ", inner-filled) + "]"
	}

	filled := int(ratio*float64(width) + 0.5)
	if filled > width {
		filled = width
	}
	errCells := int(errRatio*float64(filled) + 0.5)
	if errCells > filled {
		errCells = filled
	}
	okCells := filled - errCells
	empty := width - filled

	var b strings.Builder
	if okCells > 0 {
		b.WriteString(barFillStyle.Render(strings.Repeat("█", okCells)))
	}
	if errCells > 0 {
		b.WriteString(barErrStyle.Render(strings.Repeat("█", errCells)))
	}
	if empty > 0 {
		b.WriteString(barEmptStyle.Render(strings.Repeat("░", empty)))
	}
	return b.String()
}

func fmtETA(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	// keep it short
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

// compactBytes formats a byte count in a short, fixed-suffix form for tight
// lines: 1023 -> "1023B", 6_800_000_000 -> "6.3G". Uses binary (1024) units.
func compactBytes(b int64) string {
	if b <= 0 {
		return "0B"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	v := float64(b)
	suffixes := []string{"K", "M", "G", "T", "P"}
	i := -1
	for v >= unit && i < len(suffixes)-1 {
		v /= unit
		i++
	}
	if v >= 100 {
		return fmt.Sprintf("%.0f%s", v, suffixes[i])
	}
	return fmt.Sprintf("%.1f%s", v, suffixes[i])
}

func fmtNewReuse(okCount, total uint64, progress bool) string {
	okS := okStyle.Render(fmt.Sprintf("%d", okCount))
	totalS := dimStyle.Render(fmt.Sprintf("%d", total))

	if !progress {
		return okS
	}
	return fmt.Sprintf("%s/%s", okS, totalS)
}

// truncateLeft keeps the rightmost part of s, prefixing with "...".
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
	// keep tail by width (reserve 3 for "...")
	tail := make([]rune, 0, len(rs))
	w := 0
	for i := len(rs) - 1; i >= 0; i-- {
		r := rs[i]
		rw := lipgloss.Width(string(r))
		if w+rw > maxW-3 {
			break
		}
		tail = append(tail, r)
		w += rw
	}
	// reverse tail
	for i, j := 0, len(tail)-1; i < j; i, j = i+1, j-1 {
		tail[i], tail[j] = tail[j], tail[i]
	}
	return "..." + string(tail)
}

// shortenPathTailMax keeps as many *whole* trailing path components as will fit.
// - Never truncates directory names.
// - If it must drop leading components, prefixes with ".../".
// - If only the file fits, returns ".../file".
// - If even that doesn't fit, truncates the file name (left-truncate), still prefixed with ".../".
func shortenPathTailMax(path string, maxW int) string {
	if maxW <= 0 || path == "" {
		return ""
	}
	if lipgloss.Width(path) <= maxW {
		return path
	}

	sep := string(filepath.Separator)

	// Normalize separators (handles "/" and "\" inputs).
	p := path
	p = strings.ReplaceAll(p, "\\", sep)
	p = strings.ReplaceAll(p, "/", sep)

	// Extract & preserve Windows volume (e.g., "C:")
	vol := filepath.VolumeName(p)
	if vol != "" {
		p = strings.TrimPrefix(p, vol)
		p = strings.TrimPrefix(p, sep)
	}

	// Trim trailing separator (except root-ish)
	if len(p) > 1 {
		p = strings.TrimRight(p, sep)
	}

	parts := strings.FieldsFunc(p, func(r rune) bool { return string(r) == sep })
	if len(parts) == 0 {
		// could be just volume or weird root
		out := vol
		if out == "" {
			out = path
		}
		return truncateLeft(out, maxW)
	}

	// Helper to join with volume
	join := func(prefixDots bool, tail []string) string {
		body := strings.Join(tail, sep)
		if prefixDots {
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

	// If not everything fits, we will prefix with ".../".
	// But first: try to fit as many trailing *whole* components as possible.
	// Start from the last component and grow backwards.
	tail := []string{parts[len(parts)-1]}
	// Candidate when we are dropping something is ".../<tail>"
	best := join(true, tail)

	// If even ".../file" doesn't fit, truncate the filename itself.
	if lipgloss.Width(best) > maxW {
		file := parts[len(parts)-1]
		prefix := "..." + sep
		avail := maxW - lipgloss.Width(prefix)
		if avail <= 0 {
			// Can't even show the prefix fully; fall back to raw truncation.
			return truncateLeft(prefix+file, maxW)
		}
		return prefix + truncateLeft(file, avail)
	}

	// Add more directories as long as they fit
	for i := len(parts) - 2; i >= 0; i-- {
		candidateTail := append([]string{parts[i]}, tail...)
		candidate := join(true, candidateTail)
		if lipgloss.Width(candidate) <= maxW {
			tail = candidateTail
			best = candidate
			continue
		}
		break
	}

	// If by chance all components fit without dots, show full tail without prefix
	// (This can happen if original 'path' had vol/root stuff we normalized away, or width calc differs)
	full := join(false, parts)
	if lipgloss.Width(full) <= maxW {
		return full
	}

	return best
}

// ioSummary describes the two I/O sides of the active workflow for the summary
// rows. The direction meaning flips between backup and restore.
type ioSummary struct {
	readLabel, writeLabel string
	readTotal, writeTotal int64
	readScope, writeScope string // iostat scope for the wall-clock rate
}

func (m appModel) ioSummary() (ioSummary, bool) {
	if m.repo == nil {
		return ioSummary{}, false
	}
	switch m.application.state.workflow {
	case "export": // restore: read from the store, write to the target
		var w int64
		if m.repo.ExportStats != nil {
			w = m.repo.ExportStats.Write.TotalBytes()
		}
		return ioSummary{
			readLabel: "store", writeLabel: "target",
			readTotal: m.repo.IOStats().Read.TotalBytes(), writeTotal: w,
			readScope: "storage", writeScope: "destination",
		}, true
	default: // backup: read from the source, write to the store
		var r int64
		if m.repo.ImportStats != nil {
			r = m.repo.ImportStats.Read.TotalBytes()
		}
		return ioSummary{
			readLabel: "source", writeLabel: "store",
			readTotal: r, writeTotal: m.repo.IOStats().Write.TotalBytes(),
			readScope: "source", writeScope: "storage",
		}, true
	}
}

// readTotal returns the bytes read from the workflow's read side (source for
// backup, storage for restore), live from the repository trackers. 0 if no repo.
func (m appModel) readTotal() int64 {
	if io, ok := m.ioSummary(); ok {
		return io.readTotal
	}
	return 0
}

// termWidth returns the usable width, defaulting to 80 when unknown.
func (m appModel) termWidth() int {
	if m.width > 0 {
		return m.width
	}
	return 80
}

// narrow reports whether the terminal is below the clean-layout target of 80
// columns; below it we abbreviate labels and drop optional suffixes rather than
// wrap.
func (m appModel) narrow() bool { return m.termWidth() < 80 }

// clampLine truncates a rendered line so its visible width never exceeds the
// terminal, preventing wrap-induced "cut" lines. ANSI styling is applied by the
// caller after measuring, so we truncate the plain text here.
func clampLine(s string, w int) string {
	if w <= 0 || lipgloss.Width(s) <= w {
		return s
	}
	return truncateLeft(s, w)
}

func (m appModel) View() string {
	state := m.application.state
	w := m.termWidth()

	// Once finished we paint nothing: bubbletea clears its managed lines on
	// exit and the final summary is printed once by newApplication.
	if m.finished {
		return ""
	}

	// fast exit
	if m.forceQuit {
		return spinStyle.Render("✘") + " " +
			titleStyle.Render(m.application.name) + " " +
			dimStyle.Render(humanDuration(time.Since(state.startTime))+" "+state.snapshotID) + "  " +
			errStyle.Render("aborted") + "\n"
	}

	var s strings.Builder
	bytesDone := state.processedBytes()
	elapsed := time.Since(state.startTime)
	done := state.phase == "completed"

	// ── line 1: [spinner] workflow  clock  snapshot   path ─────────────────
	lead := spinStyle.Render(string(spinnerFrame(elapsed)))
	if done {
		lead = doneStyle.Render("✓")
	}
	header := lead + " " + titleStyle.Render(m.application.name) +
		" " + dimStyle.Render(humanDuration(elapsed)) +
		" " + dimStyle.Bold(true).Render(state.snapshotID)
	// Line-1 tail: the current path while files are streaming; otherwise the
	// phase label ("building VFS", "indexing", "committing", "completed"). The
	// later phases clear lastItem, so without this they would show no status.
	if done {
		if state.phase != "" {
			header += " " + doneStyle.Render(state.phase)
		}
	} else if state.lastItem != "" {
		avail := w - lipgloss.Width(header) - 2
		if avail >= 8 {
			header += "  " + valueStyle.Render(shortenPathTailMax(state.lastItem, avail))
		}
	} else if state.phase != "" {
		header += " " + spinStyle.Render(state.phase)
	}
	fmt.Fprintln(&s, clampLine(header, w))

	// ── line 2: bar  %  size  ETA ──────────────────────────────────────────
	// The bar and % always survive; size then ETA drop first when too tight.
	if state.gotSummary && state.summarySize > 0 {
		total := state.summarySize
		ratio := float64(bytesDone) / float64(total)
		if ratio < 0 {
			ratio = 0
		} else if ratio > 1 {
			ratio = 1
		}

		pct := valueStyle.Render(fmt.Sprintf("%3.0f%%", ratio*100))

		// size fraction
		sizeSeg := valueStyle.Render(humanize.IBytes(bytesDone))
		if !done {
			sizeSeg += dimStyle.Render("/" + compactBytes(int64(total)))
		}

		// ETA: remaining bytes ÷ the whole-run average rate. We measure progress
		// as the furthest-along signal — the processed payload (bytesDone) or the
		// bytes read from the source, whichever is greater — because on
		// small-file backups the payload counter advances in a late burst while
		// the source read total climbs smoothly. Using the average (progress ÷
		// elapsed) rather than the sampler's instantaneous rate keeps the ETA
		// from lurching between seconds and hours. States:
		//   - nothing measurable yet   → "ETA …"
		//   - no progress for a while   → "ETA --stalled--"
		//   - otherwise                 → "ETA <time>"
		etaSeg := ""
		if !done {
			label := labelStyle.Render("ETA") + " "

			progress := int64(bytesDone)
			if rt := m.readTotal(); rt > progress {
				progress = rt
			}
			if progress > int64(total) {
				progress = int64(total)
			}

			rate := 0.0
			if elapsed > 0 && progress > 0 {
				rate = float64(progress) / elapsed.Seconds()
			}

			switch {
			case rate <= 0 && elapsed < 10*time.Second:
				etaSeg = label + rateStyle.Render("…")
			case rate <= 0:
				etaSeg = label + errStyle.Render("--stalled--")
			case int64(total) > progress:
				etaDur := time.Duration(float64(int64(total)-progress) / rate * float64(time.Second))
				if v := fmtETA(etaDur); v != "" {
					etaSeg = label + rateStyle.Render(v)
				} else {
					etaSeg = label + rateStyle.Render("0s")
				}
			default:
				etaSeg = label + rateStyle.Render("0s")
			}
		}

		// Assemble the tail, dropping ETA then size when the bar gets too small.
		const minBar = 8
		optional := []string{sizeSeg, etaSeg} // dropped last-first: ETA, then size
		barLine := func(segs []string) (string, int) {
			tail := " " + pct
			for _, seg := range segs {
				if seg != "" {
					tail += "   " + seg
				}
			}
			return tail, w - 2 /*indent*/ - lipgloss.Width(tail)
		}
		var tail string
		var barW int
		for drop := 0; drop <= len(optional); drop++ {
			tail, barW = barLine(optional[:len(optional)-drop])
			if barW >= minBar {
				break
			}
		}
		if barW < minBar {
			barW = minBar
		}

		// Paint the errored fraction of the bar red. Approximate it by the share
		// of processed items that failed, applied to the filled portion.
		errRatio := 0.0
		if okc := state.countPathOk + state.countPathError; okc > 0 {
			errRatio = float64(state.countPathError) / float64(okc)
		}
		fmt.Fprintln(&s, "  "+renderBar(barW, ratio, errRatio)+tail)
	}

	// ── lines 3 & 4: aligned columns ───────────────────────────────────────
	// Both lines share a 3-column grid so their │ separators line up:
	//   line 3:  tree X/Y   │  items X/Y   │  errors N
	//   line 4:  <src> ↓ B  │  <dst> ↑ B   │  savings %
	{
		nodesTotal := state.countDir
		leavesTotal := state.countFile + state.countSymlink + state.countXattr
		if state.gotSummary && state.summaryPath > 0 {
			leavesTotal = max(leavesTotal, state.summaryFile+state.summarySymlink+state.summaryXattr)
			nodesTotal = max(state.countDir, state.summaryDirectory)
		}
		itemsDone := state.countFileOk + state.countSymlinkOk + state.countXattrOk
		itemLabel := "items"
		if m.narrow() {
			itemLabel = "it"
		}

		// row 1: counts
		row1 := []string{
			labelStyle.Render("tree") + " " + fmtNewReuse(state.countDirOk, nodesTotal, state.gotSummary),
			labelStyle.Render(itemLabel) + " " + fmtNewReuse(itemsDone, leavesTotal, state.gotSummary),
		}
		if state.countPathError != 0 {
			row1 = append(row1, labelStyle.Render("errors")+" "+err(state.countPathError))
		}

		var rows [][]string
		rows = append(rows, row1)

		// row 2: I/O + savings. Each side shows the whole-run average rate while
		// transferring; when it has gone idle it shows a status word instead —
		// the read side "(stalled)" (waiting on input), the write side "(wait)"
		// (idle between bursts, e.g. packfile flushes). Nothing is shown before a
		// side's first byte, nor once the run has completed.
		if io, ok := m.ioSummary(); ok {
			now := time.Now()
			readAct, writeAct := state.sampleIO(io.readTotal, io.writeTotal, now)
			// Feed the wall-clock throughput history used by the debug view.
			state.readRates.observe(io.readTotal, now)
			state.writeRates.observe(io.writeTotal, now)
			ioSeg := func(scope, arrow string, arrowStyle lipgloss.Style, total int64, act ioActivity, isRead bool) string {
				out := labelStyle.Render(scope) + " " + arrowStyle.Render(arrow) + " " +
					valueStyle.Render(compactBytes(total))
				// On narrow terminals keep only the read side's suffix.
				if m.narrow() && !isRead {
					return out
				}
				switch {
				case done || act == ioNone:
					// completed, or nothing transferred yet: no suffix.
				case act == ioMoving && elapsed > 0:
					avg := float64(total) / elapsed.Seconds()
					out += " " + rateStyle.Render(compactBytes(int64(avg))+"/s")
				case isRead:
					out += " " + dimStyle.Render("(stalled)")
				default:
					out += " " + dimStyle.Render("(wait)")
				}
				return out
			}
			row2 := []string{
				ioSeg(io.readLabel, "↓", downStyle, io.readTotal, readAct, true),
				ioSeg(io.writeLabel, "↑", upStyle, io.writeTotal, writeAct, false),
			}
			// Savings: processed payload vs bytes written to the store.
			processed := int64(bytesDone)
			if state.workflow == "import" && io.writeTotal > 0 && io.writeTotal < processed {
				savings := 100 * (1 - float64(io.writeTotal)/float64(processed))
				row2 = append(row2, labelStyle.Render("savings")+" "+doneStyle.Render(fmt.Sprintf("%.0f%%", savings)))
			}
			rows = append(rows, row2)
		}

		for _, line := range alignColumns(rows, statsSep) {
			fmt.Fprintln(&s, clampLine("  "+line, w))
		}
	}

	// ── debug: detailed iostat percentiles (toggle with 'd') ───────────────
	if m.debug {
		m.writeDebug(&s)
	}

	// ── last log line ──────────────────────────────────────────────────────
	if len(state.logs) != 0 {
		fmt.Fprintln(&s)
		fmt.Fprintln(&s, clampLine("  "+dimStyle.Render(state.logs[len(state.logs)-1]), w))
	}

	// ── error tail ───────────────────────────────────────────────────────
	if len(state.errors) != 0 {
		fmt.Fprintln(&s)
		used := strings.Count(s.String(), "\n")
		budget := 5
		if m.height > 0 {
			budget = m.height - used
		}
		m.writeErrorTail(&s, budget)
	}

	return s.String()
}

// writeErrorTail appends the most recent errors, bounded by budget lines, with a
// pointer to the full list when truncated.
func (m appModel) writeErrorTail(s *strings.Builder, budget int) {
	state := m.application.state
	if budget <= 0 || len(state.errors) == 0 {
		return
	}
	budget -= 3
	if budget > len(state.errors) {
		budget = len(state.errors)
	}
	if budget <= 0 {
		return
	}
	start := len(state.errors) - budget
	for i := start; i < len(state.errors); i++ {
		fmt.Fprintln(s, clampLine("  "+state.errors[i], m.termWidth()))
	}
	if budget < len(state.errors) {
		fmt.Fprintf(s, "\n%s\n", dimStyle.Render(
			fmt.Sprintf("  errors truncated — run `plakar info -errors %s` for the full list", state.snapshotID)))
	}
}

// writeDebug appends a wall-clock throughput distribution for the two sides of
// the current workflow. Toggled with 'd'. Figures come from kloset's iostat
// wall-clock distribution sampled by the TUI at a fixed cadence (see
// rateSampler). It is sampled from cumulative totals rather than per I/O op, so
// it stays correct for bursty writers (packfile flushes) and cached reads alike.
func (m appModel) writeDebug(s *strings.Builder) {
	if m.repo == nil {
		return
	}
	io, ok := m.ioSummary()
	if !ok {
		return
	}
	w := m.termWidth()
	state := m.application.state

	fmt.Fprintln(s)
	fmt.Fprintln(s, clampLine("  "+titleStyle.Render("debug")+
		dimStyle.Render("  wall-clock throughput distribution (press d to hide)"), w))
	fmt.Fprintln(s, clampLine("  "+dimStyle.Render(fmt.Sprintf(
		"    %-20s %9s %9s %9s %9s %9s %9s %7s",
		"side", "total", "avg", "median", "p90", "p95", "p99", "n")), w))

	rate := func(f float64) string {
		if f <= 0 {
			return "—"
		}
		return compactBytes(int64(f)) + "/s"
	}
	line := func(label string, total int64, d rateDist) {
		fmt.Fprintln(s, clampLine("  "+labelStyle.Render(fmt.Sprintf(
			"    %-20s %9s %9s %9s %9s %9s %9s %7d",
			label, compactBytes(total),
			rate(d.avg), rate(d.median), rate(d.p90), rate(d.p95), rate(d.p99), d.n)), w))
	}

	line(io.readLabel+" ↓ read", io.readTotal, state.readRates.dist())
	line(io.writeLabel+" ↑ write", io.writeTotal, state.writeRates.dist())
}
