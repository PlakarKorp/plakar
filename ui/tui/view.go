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

func noColor() bool { return os.Getenv("NO_COLOR") != "" }

func init() {
	if noColor() {
		lipgloss.SetColorProfile(termenv.Ascii)
	}
}

var (
	crossMark = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).SetString("✘")
	okStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	errStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	dimStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	valueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	titleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	rateStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	spinStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	doneStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	downStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	upStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

	barFillStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	barErrStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	barEmptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
)

var statsSep = "  " + lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("│") + "  "

// alignColumns joins each row's cells with sep, padding every non-trailing cell
// to the widest cell in its column so separators line up vertically.
func alignColumns(rows [][]string, sep string) []string {
	widths := map[int]int{}
	for _, r := range rows {
		for i, cell := range r {
			if i == len(r)-1 {
				continue
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

// renderBar draws a progress bar of the given cell width. ratio is the overall
// fill; errRatio is the fraction of that fill painted red. NO_COLOR falls back
// to a plain ASCII bar.
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

// compactBytes formats a byte count compactly for tight lines: 1023 -> "1023B",
// 6_800_000_000 -> "6.3G". Binary (1024) units.
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
	for i, j := 0, len(tail)-1; i < j; i, j = i+1, j-1 {
		tail[i], tail[j] = tail[j], tail[i]
	}
	return "..." + string(tail)
}

// shortenPathTailMax keeps as many whole trailing path components as fit within
// maxW, prefixing with ".../" when leading components are dropped. Directory
// names are never truncated; only the file name is, as a last resort.
func shortenPathTailMax(path string, maxW int) string {
	if maxW <= 0 || path == "" {
		return ""
	}
	if lipgloss.Width(path) <= maxW {
		return path
	}

	sep := string(filepath.Separator)

	p := path
	p = strings.ReplaceAll(p, "\\", sep)
	p = strings.ReplaceAll(p, "/", sep)

	vol := filepath.VolumeName(p)
	if vol != "" {
		p = strings.TrimPrefix(p, vol)
		p = strings.TrimPrefix(p, sep)
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
		candidateTail := append([]string{parts[i]}, tail...)
		candidate := join(true, candidateTail)
		if lipgloss.Width(candidate) <= maxW {
			tail = candidateTail
			best = candidate
			continue
		}
		break
	}

	full := join(false, parts)
	if lipgloss.Width(full) <= maxW {
		return full
	}

	return best
}

// ioSummary describes the read/write sides of the active workflow: backup reads
// the source and writes the store, restore reads the store and writes the
// target, sync reads the source and writes the peer, check only reads.
type ioSummary struct {
	readLabel, writeLabel string
	readTotal, writeTotal int64
	readOnly              bool
}

func (m appModel) ioSummary() (ioSummary, bool) {
	if m.repo == nil {
		return ioSummary{}, false
	}
	switch m.application.state.workflow {
	case "export":
		var w int64
		if m.repo.ExportStats != nil {
			w = m.repo.ExportStats.Write.TotalBytes()
		}
		return ioSummary{
			readLabel: "store", writeLabel: "target",
			readTotal: m.repo.IOStats().Read.TotalBytes(), writeTotal: w,
		}, true
	case "synchronize":
		st := m.application.state
		return ioSummary{
			readLabel: "source", writeLabel: "peer",
			readTotal: st.sourceRead, writeTotal: st.storeWrite,
		}, true
	case "check":
		return ioSummary{
			readLabel: "store",
			readTotal: m.repo.IOStats().Read.TotalBytes(),
			readOnly:  true,
		}, true
	default:
		var r int64
		if m.repo.ImportStats != nil {
			r = m.repo.ImportStats.Read.TotalBytes()
		}
		return ioSummary{
			readLabel: "source", writeLabel: "store",
			readTotal: r, writeTotal: m.repo.IOStats().Write.TotalBytes(),
		}, true
	}
}

// readTotal returns the bytes read from the workflow's read side, 0 if no repo.
func (m appModel) readTotal() int64 {
	if io, ok := m.ioSummary(); ok {
		return io.readTotal
	}
	return 0
}

// progressBytes is the logical payload processed so far; it shares units with
// summarySize so the bar reaches 100%.
func (m appModel) progressBytes() uint64 {
	return m.application.state.processedBytes()
}

func (m appModel) termWidth() int {
	if m.width > 0 {
		return m.width
	}
	return 80
}

// narrow reports terminals below 80 columns, where labels and suffixes shrink.
func (m appModel) narrow() bool { return m.termWidth() < 80 }

// clampLine truncates a line so its visible width never exceeds w.
func clampLine(s string, w int) string {
	if w <= 0 || lipgloss.Width(s) <= w {
		return s
	}
	return truncateLeft(s, w)
}

func (m appModel) View() string {
	state := m.application.state
	w := m.termWidth()

	// Once finished we paint nothing: the final summary is printed once by
	// newApplication.
	if m.finished {
		return ""
	}

	if m.forceQuit {
		return spinStyle.Render("✘") + " " +
			titleStyle.Render(m.application.name) + " " +
			dimStyle.Render(humanDuration(time.Since(state.startTime))+" "+state.snapshotID) + "  " +
			errStyle.Render("aborted") + "\n"
	}

	var s strings.Builder
	bytesDone := m.progressBytes()
	elapsed := time.Since(state.startTime)
	done := state.phase == "completed"

	// line 1: [spinner] workflow  clock  snapshot  path/phase
	lead := spinStyle.Render(string(spinnerFrame(elapsed)))
	if done {
		lead = doneStyle.Render("✓")
	}
	header := lead + " " + titleStyle.Render(m.application.name) +
		" " + dimStyle.Render(humanDuration(elapsed)) +
		" " + dimStyle.Bold(true).Render(state.snapshotID)
	// Current path while files stream, else the phase label.
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

	// line 2: bar  %  size  ETA (size then ETA drop first when too tight)
	if state.gotSummary && state.summarySize > 0 {
		total := state.summarySize
		ratio := float64(bytesDone) / float64(total)
		if ratio < 0 {
			ratio = 0
		} else if ratio > 1 {
			ratio = 1
		}

		pct := valueStyle.Render(fmt.Sprintf("%3.0f%%", ratio*100))

		sizeSeg := valueStyle.Render(humanize.IBytes(bytesDone))
		if !done {
			sizeSeg += dimStyle.Render("/" + compactBytes(int64(total)))
		}

		// ETA = remaining bytes ÷ whole-run average rate. Progress is the
		// furthest-along signal (processed payload or source reads, whichever is
		// greater); the average rather than an instantaneous rate keeps the ETA
		// from lurching between seconds and hours.
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
		optional := []string{sizeSeg, etaSeg}
		barLine := func(segs []string) (string, int) {
			tail := " " + pct
			for _, seg := range segs {
				if seg != "" {
					tail += "   " + seg
				}
			}
			return tail, w - 2 - lipgloss.Width(tail)
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

		// Paint the errored fraction of the bar red, by the share of failed items.
		errRatio := 0.0
		if okc := state.countPathOk + state.countPathError; okc > 0 {
			errRatio = float64(state.countPathError) / float64(okc)
		}
		fmt.Fprintln(&s, "  "+renderBar(barW, ratio, errRatio)+tail)
	}

	// lines 3 & 4: a shared 3-column grid, │ separators aligned.
	//   tree X/Y   │  items X/Y   │  errors N
	//   <src> ↓ B  │  <dst> ↑ B   │  savings %
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

		row1 := []string{
			labelStyle.Render("tree") + " " + fmtNewReuse(state.countDirOk, nodesTotal, state.gotSummary),
			labelStyle.Render(itemLabel) + " " + fmtNewReuse(itemsDone, leavesTotal, state.gotSummary),
		}
		if state.countPathError != 0 {
			row1 = append(row1, labelStyle.Render("errors")+" "+err(state.countPathError))
		}

		var rows [][]string
		rows = append(rows, row1)

		// row 2: I/O + savings. Each side shows its whole-run average rate while
		// transferring, else a status word: read "(stalled)", write "(wait)".
		// Nothing shows before a side's first byte or once the run completes.
		if io, ok := m.ioSummary(); ok {
			now := time.Now()
			readAct, writeAct := state.sampleIO(io.readTotal, io.writeTotal, now)
			state.readRates.observe(io.readTotal, now)
			state.writeRates.observe(io.writeTotal, now)
			ioSeg := func(scope, arrow string, arrowStyle lipgloss.Style, total int64, act ioActivity, isRead bool) string {
				out := labelStyle.Render(scope) + " " + arrowStyle.Render(arrow) + " " +
					valueStyle.Render(compactBytes(total))
				if m.narrow() && !isRead {
					return out
				}
				switch {
				case done || act == ioNone:
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
			row2 := []string{ioSeg(io.readLabel, "↓", downStyle, io.readTotal, readAct, true)}
			if !io.readOnly {
				row2 = append(row2, ioSeg(io.writeLabel, "↑", upStyle, io.writeTotal, writeAct, false))
				// savings: processed payload vs bytes written to the store.
				processed := int64(bytesDone)
				if state.workflow == "import" && io.writeTotal > 0 && io.writeTotal < processed {
					savings := 100 * (1 - float64(io.writeTotal)/float64(processed))
					row2 = append(row2, labelStyle.Render("savings")+" "+doneStyle.Render(fmt.Sprintf("%.0f%%", savings)))
				}
			}
			rows = append(rows, row2)
		}

		for _, line := range alignColumns(rows, statsSep) {
			fmt.Fprintln(&s, clampLine("  "+line, w))
		}
	}

	if m.debug {
		m.writeDebug(&s)
	}

	if len(state.logs) != 0 {
		fmt.Fprintln(&s)
		fmt.Fprintln(&s, clampLine("  "+dimStyle.Render(state.logs[len(state.logs)-1]), w))
	}

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
// the current workflow. Toggled with 'd'. It samples cumulative totals, so it
// stays correct for bursty writers and cached reads alike.
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
