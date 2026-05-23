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

var (
	crossMark = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).SetString("✘")
	okStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	errStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red
	dimStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // gray (optional)
)

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

func progressBar() progress.Model {
	p := progress.New(
		progress.WithColorProfile(termenv.Ascii),
	)

	// Make it ASCII-ish
	p.Full = '*' // #
	p.Empty = ' '
	p.ShowPercentage = true

	return p
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

func fmtNewReuse(okCount, total uint64, progress bool) string {
	okS := okStyle.Render(fmt.Sprintf("%d", okCount))
	totalS := dimStyle.Render(fmt.Sprintf("%d", total))

	if !progress {
		return fmt.Sprintf("%s", okS)
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

func (m appModel) View() string {
	state := m.application.state

	// fast exit
	if m.forceQuit {
		return fmt.Sprintf("[%s] %s: aborted !\n", humanDuration(time.Since(state.startTime)), m.application.name)
	}

	var s strings.Builder
	done := state.countPathOk + state.countPathError

	// --- summaries (unchanged logic) ---
	writeProcessedSummary := func() {
		nodesTotal := state.countDir
		leavesTotal := state.countFile + state.countSymlink + state.countXattr
		if state.gotSummary && state.summaryPath > 0 {
			leavesTotal = max(leavesTotal, state.summaryFile+state.summarySymlink+state.summaryXattr)
			nodesTotal = max(state.countDir, state.summaryDirectory)
		}

		indent := strings.Repeat(" ", len(humanDuration(time.Since(state.startTime))))
		fmt.Fprintf(&s, "%s   %s:", indent, m.application.name)

		fmt.Fprintf(&s, " nodes=%s", fmtNewReuse(state.countDirOk, nodesTotal, state.gotSummary))
		fmt.Fprintf(&s, ", objects=%s", fmtNewReuse(state.countFileOk+state.countSymlinkOk+state.countXattrOk, leavesTotal, state.gotSummary))

		if state.countPathError != 0 {
			fmt.Fprintf(&s, ", errors=%s", err(state.countPathError))
		}

		fmt.Fprintf(&s, "\n")
	}

	writeStoreSummary := func() {
		if m.repo == nil {
			return
		}

		if time.Since(m.application.debounceStat) >= 1*time.Second {
			ioStats := m.repo.IOStats()
			indent := strings.Repeat(" ", len(humanDuration(time.Since(state.startTime))))
			r := ioStats.Read.Stats()
			w := ioStats.Write.Stats()

			m.application.lastStat = fmt.Sprintf(
				"%s    store: read=%s, write=%s\n",
				indent,
				formatBytes(r.TotalBytes),
				formatBytes(w.TotalBytes),
			)

			m.application.debounceStat = time.Now()
		}

		fmt.Fprint(&s, m.application.lastStat)
	}

	// --- shared line writer: prefix + item + right-aligned tail ---
	writeLine := func(prefix, item, tail string) {
		// If we don't know width yet, just print plainly.
		if m.width <= 0 {
			fmt.Fprintf(&s, "%s %s %s\n", prefix, item, tail)
			return
		}

		availableW := m.width - lipgloss.Width(prefix) - lipgloss.Width(tail) - 2 // spaces around item
		if availableW < 0 {
			availableW = 0
		}

		item = shortenPathTailMax(item, availableW)

		pad := availableW - lipgloss.Width(item)
		if pad < 0 {
			pad = 0
		}

		fmt.Fprintf(&s, "%s %s%s %s\n", prefix, item, strings.Repeat(" ", pad), tail)
	}

	// count visual lines (good enough if you don't have ANSI newlines in single lines)
	countLines := func(str string) int {
		if str == "" {
			return 0
		}
		// number of '\n' == number of lines (since you always end lines with \n)
		return strings.Count(str, "\n")
	}

	writeLastErrors := func(maxLines int) {
		if maxLines <= 0 || len(state.errors) == 0 {
			return
		}
		maxLines -= 3

		if maxLines > len(state.errors) {
			maxLines = len(state.errors)
		}
		start := len(state.errors) - maxLines
		for i := start; i < len(state.errors); i++ {
			fmt.Fprintf(&s, "%s\n", state.errors[i])
		}

		if maxLines < len(state.errors) {
			fmt.Fprintf(&s, "\nerrors list truncated, run `plakar info -errors %s` for full list\n", state.snapshotID)
		}
	}

	// --- first line always shows last item + right-aligned size ---
	sizeText := humanize.IBytes(uint64(state.countFileSize))

	// Progress mode: we have a total and can show bar + ETA on bar line
	if state.gotSummary && state.summaryPath > 0 {
		total := state.summaryPath

		// ratio clamped to [0,1]
		ratio := 0.0
		if total > 0 {
			ratio = float64(done) / float64(total)
			if ratio < 0 {
				ratio = 0
			} else if ratio > 1 {
				ratio = 1
			}
		}

		// First line: prefix + last item + size (size right aligned)
		prefix := fmt.Sprintf("[%s] %s %s", humanDuration(time.Since(state.startTime)), state.snapshotID, state.phase)
		writeLine(prefix, state.lastItem, sizeText)

		// ETA (to be printed on the progress bar line)
		etaText := ""
		if m.rateEMA > 0 && done > 10 && time.Since(state.startTime) > 2*time.Second && total >= done {
			remaining := float64(total - done)
			etaDur := time.Duration(remaining / m.rateEMA * float64(time.Second))
			if v := fmtETA(etaDur); v != "" {
				etaText = "ETA " + v
			}
		}

		// Progress bar line: bar left, ETA right (ETA right-aligned)
		p := m.progress
		if m.width > 0 {
			barW := m.width
			if etaText != "" {
				// " " between bar and ETA
				barW = m.width - lipgloss.Width(etaText) - 1
				if barW < 10 {
					barW = 10
				}
			}
			p.Width = barW
		}
		bar := p.ViewAs(ratio)

		if m.width > 0 && etaText != "" {
			// right-align ETA by padding between bar and ETA
			pad := m.width - lipgloss.Width(bar) - lipgloss.Width(etaText) - 1
			if pad < 0 {
				pad = 0
			}
			fmt.Fprintf(&s, "%s%s %s\n", bar, strings.Repeat(" ", pad), etaText)
		} else if etaText != "" {
			fmt.Fprintf(&s, "%s %s\n", bar, etaText)
		} else {
			fmt.Fprintf(&s, "%s\n", bar)
		}

		writeProcessedSummary()
		writeStoreSummary()

		if len(state.logs) != 0 {
			fmt.Fprintf(&s, "\n%s\n", state.logs[len(state.logs)-1])
		}

		if len(state.errors) != 0 {
			fmt.Fprintf(&s, "\n")

			if m.height > 0 {
				used := countLines(s.String())
				remaining := m.height - used
				// If you add a separator line above, subtract 1 more here.
				writeLastErrors(remaining)
			} else {
				// fallback: show a small tail
				writeLastErrors(5)
			}
		}

		return s.String()
	}

	// Non-progress mode: same first line, no bar
	prefix := fmt.Sprintf("[%s] %s %s", humanDuration(time.Since(state.startTime)), state.snapshotID, state.phase)
	writeLine(prefix, state.lastItem, sizeText)

	writeProcessedSummary()
	writeStoreSummary()

	if len(state.logs) != 0 {
		fmt.Fprintf(&s, "\n%s\n", state.logs[len(state.logs)-1])
	}

	if len(state.errors) != 0 {
		fmt.Fprintf(&s, "\n")

		if m.height > 0 {
			used := countLines(s.String())
			remaining := m.height - used
			// If you add a separator line above, subtract 1 more here.
			writeLastErrors(remaining)
		} else {
			// fallback: show a small tail
			writeLastErrors(5)
		}
	}

	return s.String()
}
