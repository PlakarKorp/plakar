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
	okStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))  // green
	errStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))  // red
	dimStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))  // bright-black / gray
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
	p.ShowPercentage = false // we render percentage ourselves
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

// truncateLeft keeps the rightmost part of s, prefixing with "…".
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

// shortenPathTailMax keeps as many whole trailing path components as fit.
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
		prefix := ""
		if dots {
			prefix = "..."
			if vol != "" {
				prefix = vol + sep + prefix
			}
			return prefix + sep + body
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

// padRight pads or truncates s to exactly w visible characters.
func padRight(s string, w int) string {
	sw := lipgloss.Width(s)
	if sw >= w {
		return s
	}
	return s + strings.Repeat(" ", w-sw)
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m appModel) View() string {
	state := m.application.state
	elapsed := time.Since(state.startTime)

	if m.forceQuit {
		return dimStyle.Render(fmt.Sprintf("  %s  %s: aborted\n",
			humanDuration(elapsed), m.application.name))
	}

	var s strings.Builder
	w := m.width
	if w <= 0 {
		w = 80 // safe default before first WindowSizeMsg
	}

	// ── computed values ──────────────────────────────────────────────────────

	done := state.countPathOk + state.countPathError + state.countPathCached
	total := state.summaryPath
	hasSummary := state.gotSummary && total > 0

	ratio := 0.0
	pct := 0
	if hasSummary && total > 0 {
		ratio = float64(done) / float64(total)
		if ratio < 0 {
			ratio = 0
		} else if ratio > 1 {
			ratio = 1
		}
		pct = int(ratio * 100)
	}

	etaText := ""
	if hasSummary && m.rateEMA > 0 && done > 10 &&
		elapsed > 2*time.Second && total >= done {
		remaining := float64(total - done)
		etaDur := time.Duration(remaining / m.rateEMA * float64(time.Second))
		if v := fmtETA(etaDur); v != "" {
			etaText = "ETA " + v
		}
	}

	// ── line 1: timer · bar · progress fraction · size · ETA ────────────────
	//
	//   02:14  ████████████░░░░  74%  8,432/9,721  1.2 GiB  ETA 32s
	//
	// Fixed-width tokens go on the outside; the bar stretches to fill the gap.

	timer := boldStyle.Render(humanDuration(elapsed))

	// Right side: "done/total  size  ETA" or just "size" when no summary
	var rightTokens []string
	if hasSummary {
		fraction := fmt.Sprintf("%d/%d", done, total)
		if state.countPathError > 0 {
			fraction += "  " + errStyle.Render(fmt.Sprintf("%d err", state.countPathError))
		}
		rightTokens = append(rightTokens, dimStyle.Render(fraction))
	}
	if state.countFileSize > 0 {
		rightTokens = append(rightTokens, humanize.IBytes(uint64(state.countFileSize)))
	}
	if etaText != "" {
		rightTokens = append(rightTokens, dimStyle.Render(etaText))
	}
	right := strings.Join(rightTokens, dimStyle.Render("  ·  "))

	// Bar with percentage label embedded at the right end
	pctLabel := ""
	if hasSummary {
		pctLabel = fmt.Sprintf(" %3d%%", pct)
	}

	// Calculate bar width: total - timer - spaces - pctLabel - right
	timerW := lipgloss.Width(timer)
	rightW := lipgloss.Width(right)
	pctW := lipgloss.Width(pctLabel)

	// layout: "  {timer}  {bar}{pct}  {right}\n"
	// fixed overhead: 2+2+2 = 6 spaces
	overhead := 2 + timerW + 2 + pctW + 2 + rightW
	barW := w - overhead
	if barW < 4 {
		barW = 4
	}

	bar := ""
	if hasSummary {
		p := m.progress
		p.Width = barW
		bar = p.ViewAs(ratio) + dimStyle.Render(pctLabel)
	} else {
		// Spinner-style placeholder when we don't know the total yet
		bar = dimStyle.Render(strings.Repeat("░", barW))
	}

	line1 := fmt.Sprintf("  %s  %s", timer, bar)
	if right != "" {
		// pad so right sits at the terminal edge
		line1W := lipgloss.Width(line1)
		gap := w - line1W - rightW - 2
		if gap < 1 {
			gap = 1
		}
		line1 += strings.Repeat(" ", gap) + right
	}
	fmt.Fprintln(&s, line1)

	// ── line 2: current path (dim, indented) ─────────────────────────────────

	if state.lastItem != "" {
		indent := "  " + strings.Repeat(" ", timerW+2) // align under the bar
		avail := w - lipgloss.Width(indent)
		if avail < 1 {
			avail = 1
		}
		path := dimStyle.Render(shortenPathTailMax(state.lastItem, avail))
		fmt.Fprintln(&s, indent+path)
	} else if state.phase != "" {
		indent := "  " + strings.Repeat(" ", timerW+2)
		fmt.Fprintln(&s, indent+dimStyle.Render(state.phase+"…"))
	}

	// ── line 3: stats row (dirs · cached · store I/O) ────────────────────────

	var statTokens []string

	if state.countDirOk > 0 {
		statTokens = append(statTokens,
			dimStyle.Render("dirs ")+okStyle.Render(fmt.Sprintf("%d", state.countDirOk)))
	}
	if state.countFileCached+state.countDirCached+state.countSymlinkCached+state.countXattrCached > 0 {
		cached := state.countFileCached + state.countDirCached + state.countSymlinkCached + state.countXattrCached
		statTokens = append(statTokens,
			dimStyle.Render("cached ")+dimStyle.Render(fmt.Sprintf("%d", cached)))
	}

	if m.repo != nil {
		if time.Since(m.application.debounceStat) >= time.Second {
			io := m.repo.IOStats()
			r := io.Read.Stats()
			ww := io.Write.Stats()
			m.application.lastStat = fmt.Sprintf("store ↑%s ↓%s",
				humanize.IBytes(uint64(ww.TotalBytes)),
				humanize.IBytes(uint64(r.TotalBytes)))
			m.application.debounceStat = time.Now()
		}
		if m.application.lastStat != "" {
			statTokens = append(statTokens, dimStyle.Render(m.application.lastStat))
		}
	}

	if len(statTokens) > 0 {
		indent := "  " + strings.Repeat(" ", timerW+2)
		fmt.Fprintln(&s, indent+strings.Join(statTokens, dimStyle.Render("  ·  ")))
	}

	// ── errors (tail, bounded by remaining terminal height) ──────────────────

	if len(state.errors) > 0 {
		fmt.Fprintln(&s)

		// how many lines we've used so far
		used := strings.Count(s.String(), "\n")
		avail := m.height - used - 1 // leave one line for the notice
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
				"  … %d more errors — run `plakar info -errors %s` for full list",
				start, state.snapshotID)))
		}
	}

	return s.String()
}
