// scrollback-render — replay a pair-wrap raw capture through a VT100
// emulator and write one ANSI-styled output line per logical scrollback row.
// Drop-in replacement for the Python+pyte renderer at bin/pair-scrollback-render.
//
// Why Go: pyte's HistoryScreen dispatches every method call through an
// __getattribute__ override that ran ~19M times for a 3 MB raw input —
// ~95% of wall time. Even after the CaptureScreen patch (3.6x speedup),
// the Python interpreter + pyte vendoring add startup cost and a private
// venv that the brew formula has to manage. A static Go binary using
// charmbracelet/x/vt replays the same stream with no runtime deps and
// stays within the pair repo's existing cmd/ layout.
//
// Pipeline:
//
//	raw bytes (.raw)              → emulator.Write(...) in segments
//	resize events (.events.jsonl) → segment boundaries with new (cols,rows)
//	final emulator state          → scrollback lines + visible buffer
//	each row                      → SGR-decorated text line written to out
//
// CLI is identical to the Python version so bin/pair-scrollback-open can
// invoke either:
//
//	scrollback-render <raw> <events.jsonl> <out.ansi>
package scrollbackcmd

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/vt"
)

// historyRows caps scrolled-out rows retained for the viewer. Matched
// to zellij/config.kdl's `scroll_buffer_size 2000` so PageUp inside the
// agent pane and Alt+/ both reach back the same distance — otherwise
// the viewer would surface lines that zellij no longer has, leaving
// the scroll-overlay logic in pair-scrollback-open unable to align
// against the agent pane's frame.
const historyRows = 2_000

const (
	defaultCols = 80
	defaultRows = 24
)

type scrollbackEvent struct {
	Type   string `json:"type"`
	Offset int64  `json:"offset"`
	Cols   int    `json:"cols"`
	Rows   int    `json:"rows"`
	Ts     string `json:"ts,omitempty"` // RFC3339 wall-clock for "time" events (#59)
}

// dateOf extracts the YYYY-MM-DD day from an RFC3339 timestamp; "" on a
// malformed value so a corrupt time event degrades to undated, never panics (#59).
func dateOf(ts string) string {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return ""
	}
	return t.Format("2006-01-02")
}

// parseEvents reads the sidecar JSONL. Empty / missing file → empty slice.
// Malformed lines are skipped so a corrupted tail doesn't abort the render —
// imperfect width tracking beats an unusable viewer.
func parseEvents(path string) ([]scrollbackEvent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return parseEventData(data), nil
}

func parseEventData(data []byte) []scrollbackEvent {
	out, _ := decodeEventData(data, false)
	return out
}

func decodeEventData(data []byte, strict bool) ([]scrollbackEvent, error) {
	var out []scrollbackEvent
	for i, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var e scrollbackEvent
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			if strict {
				return nil, fmt.Errorf("event line %d: %w", i+1, err)
			}
			continue
		}
		// Keep both known types: resize boundaries AND time stamps (#59).
		// Consumers filter by Type at their use sites.
		if e.Type == "resize" || e.Type == "time" {
			out = append(out, e)
		}
	}
	return out, nil
}

// initialSize pulls (cols, rows) from the first usable resize event, or
// falls back to 80x24 if the sidecar is empty or the first entry is
// malformed. pair-wrap always emits an initial resize at offset 0, so the
// fallback only fires on a truncated file.
func initialSize(events []scrollbackEvent) (int, int) {
	for _, e := range events {
		if e.Type == "resize" && e.Cols > 0 && e.Rows > 0 {
			return e.Cols, e.Rows
		}
	}
	return defaultCols, defaultRows
}

// dateMark records the emulator's scrollback length at a "time" event's byte
// offset → the day that applies to committed lines from that index onward (#59).
// Built during the feed (the only place that knows both byte offsets and the
// rendered line count); consumed by the pure interleaveDateMarkers.
type dateMark struct {
	line int
	date string
}

// feedSegments writes raw into the emulator as a single offset-ordered walk over
// ALL sidecar events: write everything up to event.Offset, then act — Resize on a
// resize event, or snapshot Scrollback().Len() on a time event. Returns the time
// snapshots (empty unless time events are present). The caller already set the
// initial size via initialSize; re-applying the offset-0 resize here is a harmless
// no-op (resize to the current dimensions). Walking all events — rather than
// events[1:] — means a time event in any position (incl. first) is captured, and
// an empty events slice is handled without an out-of-range slice (#59).
//
// Clamping Offset to len(raw) defends against a corrupted sidecar that records
// an offset beyond EOF (saw this once with a half-written events file after a
// hard kill); without clamping we'd panic on the slice.
func feedSegments(em *vt.Emulator, raw []byte, events []scrollbackEvent) []dateMark {
	marks, _ := feedSegmentsChecked(em, raw, events, false)
	return marks
}

func feedSegmentsChecked(em *vt.Emulator, raw []byte, events []scrollbackEvent, checked bool) ([]dateMark, error) {
	return feedSegmentsContext(context.Background(), em, raw, events, checked)
}

func feedSegmentsContext(ctx context.Context, em *vt.Emulator, raw []byte, events []scrollbackEvent, checked bool) ([]dateMark, error) {
	checkLoss := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if checked && em.HistoryCapacityLoss() != 0 {
			return fmt.Errorf("rendered history exceeds capacity; lost %d rows", em.HistoryCapacityLoss())
		}
		return nil
	}
	write := func(data []byte) error {
		for len(data) > 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
			n := len(data)
			// Stop soon after overflow rather than repeatedly evicting a large
			// history for the rest of an adversarial recording.
			if checked {
				n = min(n, 4096)
			}
			if _, err := em.Write(data[:n]); err != nil {
				return err
			}
			if err := checkLoss(); err != nil {
				return err
			}
			data = data[n:]
		}
		return nil
	}
	var cursor int64
	var marks []dateMark
	for _, e := range events {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		off := e.Offset
		if off > int64(len(raw)) {
			off = int64(len(raw))
		}
		if off > cursor {
			if err := write(raw[cursor:off]); err != nil {
				return nil, err
			}
			cursor = off
		}
		switch e.Type {
		case "resize":
			if checked {
				if err := em.ResizeChecked(e.Cols, e.Rows); err != nil {
					return nil, fmt.Errorf("resize %dx%d: %w", e.Cols, e.Rows, err)
				}
			} else {
				em.Resize(e.Cols, e.Rows)
			}
			if err := checkLoss(); err != nil {
				return nil, err
			}
		case "time":
			if d := dateOf(e.Ts); d != "" {
				marks = append(marks, dateMark{line: em.Scrollback().Len(), date: d})
			}
		}
	}
	if cursor < int64(len(raw)) {
		if err := write(raw[cursor:]); err != nil {
			return nil, err
		}
	}
	return marks, nil
}

// tsMarkerLine is the wire format the distiller parses (#59). MUST stay in sync
// with tsMarkerRe in cmd/internal/changelogcmd/distill.go — the contract is pinned by
// the render→clean→distill e2e test cmd/pair-changelog/e2e_test.go
// (TestEndToEndMarkerSurvival), which feeds real time events through both binaries.
func tsMarkerLine(date string) string {
	return "⟦pair:ts " + date + "⟧"
}

// interleaveDateMarkers inserts a tsMarkerLine immediately before the first line
// of each new date run. marks are (scrollback-line-index, date) snapshots in
// ascending index; a marker is emitted only when the applicable date *changes*
// from the running date (consecutive same-date marks collapse). Lines before the
// first mark stay undated; marks past len(lines) are ignored. Pure (#59).
func interleaveDateMarkers(lines []string, marks []dateMark) []string {
	if len(marks) == 0 {
		return lines
	}
	out := make([]string, 0, len(lines)+len(marks))
	mi := 0
	prevDate := ""
	for i := 0; i < len(lines); i++ {
		curDate := prevDate
		for mi < len(marks) && marks[mi].line <= i {
			curDate = marks[mi].date
			mi++
		}
		if curDate != "" && curDate != prevDate {
			out = append(out, tsMarkerLine(curDate))
			prevDate = curDate
		}
		out = append(out, lines[i])
	}
	return out
}

// serializeRow flattens one row into ANSI-styled text. Trims trailing
// default-styled blanks (so the viewer doesn't scroll past pad), uses
// Style.Diff so we only emit SGR codes when the style actually changes
// between cells, and terminates with \x1b[0m to keep the row's last
// style from bleeding into the next line if a viewer concatenates without
// resetting between lines.
//
// A non-default background space is treated as visible content (e.g.
// inverse-video padding). Matches what the Python renderer does.
//
// In plain mode (plain=true) no SGR is emitted at all: the row is just its
// visible content, trimmed to the last non-blank-*content* cell — a cell that
// is "visible" only via a non-default background (inverse-video padding, box
// fill) is NOT emitted in plain mode, so it must not extend the row, or a
// trailing bordered region would become space-padding toward terminal width.
func serializeRow(line uv.Line, plain bool) string {
	last := -1
	for i := range line {
		c := &line[i]
		// Continuation cells of a preceding wide grapheme are stored as
		// zero-value Cell{} per the ultraviolet convention (Width=0,
		// Content=""). They don't extend the visible row and must not
		// emit anything in the loop below.
		if c.IsZero() {
			continue
		}
		content := c.Content
		if content != "" && content != " " {
			last = i
		} else if !plain && c.Style.Bg != nil {
			last = i
		}
	}
	if last < 0 {
		return ""
	}
	var b strings.Builder
	var prev uv.Style // zero value = default; Diff vs zero emits a reset
	first := true
	for i := 0; i <= last; i++ {
		c := &line[i]
		// Skip wide-grapheme continuation cells — the wide cell already
		// emitted its full glyph; emitting anything here adds a phantom
		// space after every emoji.
		if c.IsZero() {
			continue
		}
		if !plain && (first || !c.Style.Equal(&prev)) {
			b.WriteString(c.Style.Diff(&prev))
			prev = c.Style
			first = false
		}
		if c.Content == "" {
			b.WriteByte(' ')
		} else {
			b.WriteString(c.Content)
		}
	}
	if !plain {
		b.WriteString("\x1b[0m")
	}
	return b.String()
}

// resolveMax maps a --max-lines value to a scrollback cap. <=0 means
// "uncapped" — a continuation wants the whole session, not the viewer's
// 2000-row window. Represented as a large sentinel; .raw is per-run
// O_TRUNC'd, so the practical bound is the run length.
func resolveMax(n int) int {
	if n <= 0 {
		return math.MaxInt32
	}
	return n
}

// visibleRow materializes row y of the live screen as a uv.Line. The
// emulator exposes cells one at a time via CellAt(x,y); there's no
// "give me the whole row" accessor. A missing cell (CellAt returns nil)
// becomes a zero-value Cell, which serializeRow treats as a blank.
func visibleRow(em *vt.Emulator, y, width int) uv.Line {
	row := make(uv.Line, width)
	for x := 0; x < width; x++ {
		if c := em.CellAt(x, y); c != nil {
			row[x] = *c
		}
	}
	return row
}

// RenderLines replays a recorded capture and returns it as plain text lines:
// scrollback history (oldest first, at most maxLines rows; <=0 = uncapped),
// then the visible screen, trailing blank lines trimmed. The visible screen is
// the tail, so a caller wanting "what the pane shows now" takes the last rows.
// It reads the two files and writes nothing; the caller holds any retention
// lease the capture needs.
//
// Each call parks one emulator drain goroutine for the life of the process
// (see replay), so call it from short-lived processes such as a CLI, never
// in a loop inside a long-running supervisor.
func RenderLines(rawPath, eventsPath string, maxLines int) ([]string, error) {
	r, err := replay(rawPath, eventsPath, true, maxLines)
	return r.lines, err
}

// replayed is one capture replayed through the emulator.
type replayed struct {
	lines       []string
	viewportTop int // 1-indexed line where the visible buffer starts
	marks       []dateMark
}

func render(rawPath, eventsPath, outPath, viewportPath string, plain bool, maxLines int, withTimestamps bool) error {
	r, err := replay(rawPath, eventsPath, plain, maxLines)
	if err != nil {
		return err
	}
	out, viewportTop, marks := r.lines, r.viewportTop, r.marks
	// Change-log path only: interleave day markers from the time-event snapshots
	// so the distiller can date entries by real change-time (#59). Done after the
	// trailing-blank trim so a marker never dangles past content. The scrollback
	// viewer never sets this flag → its render is byte-identical to before.
	if withTimestamps {
		out = interleaveDateMarkers(out, marks)
	}
	// Write the viewport sidecar *first*, then atomically rename the
	// .ansi into place. Order matters: scrollback.lua's BufReadPost
	// opens the .ansi and immediately reads the sidecar — flipping the
	// .ansi last guarantees the sidecar is up-to-date by the time
	// nvim sees the new content. Sidecar is best-effort: on write
	// failure, scrollback.lua falls back to its prior bottom-alignment.
	// The viewport sidecar positions the Alt+/ nvim viewer; it's meaningless
	// for the plain projection (a continuation distills the text, not a
	// scroll position), so skip it and don't litter a stray <out>.viewport.
	if !plain {
		if viewportPath == "" {
			return fmt.Errorf("viewport path is required for ANSI output")
		}
		_ = os.WriteFile(viewportPath, []byte(strconv.Itoa(viewportTop)+"\n"), 0o644)
	}

	// Atomic write so a double-tap Alt+/ can't race truncate-then-write
	// on the same path. Reader sees either the old complete file or the
	// new complete file, never a half-written one.
	tmp := outPath + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	body := strings.Join(out, "\n")
	if len(out) > 0 {
		body += "\n"
	}
	if _, err := f.WriteString(body); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, outPath)
}

func replay(rawPath, eventsPath string, plain bool, maxLines int) (replayed, error) {
	events, err := parseEvents(eventsPath)
	if err != nil {
		return replayed{}, fmt.Errorf("parse events: %w", err)
	}
	raw, err := os.ReadFile(rawPath)
	if err != nil {
		return replayed{}, fmt.Errorf("read raw: %w", err)
	}
	return replayData(raw, events, plain, resolveMax(maxLines), 0)
}

// replayData shares emulator and serialization behavior with bounded readers.
// rowLimit uses an extra history row as an overflow sentinel.
func replayData(raw []byte, events []scrollbackEvent, plain bool, historyLimit, rowLimit int) (replayed, error) {
	limits := vt.DefaultLimits()
	if rowLimit > 0 {
		limits = boundedReplayLimits(rowLimit)
	}
	return replayDataWithLimits(raw, events, plain, historyLimit, rowLimit, limits)
}

func replayDataWithLimits(raw []byte, events []scrollbackEvent, plain bool, historyLimit, rowLimit int, limits vt.Limits) (replayed, error) {
	return replayDataWithLimitsContext(context.Background(), raw, events, plain, historyLimit, rowLimit, limits)
}

func replayDataWithLimitsContext(ctx context.Context, raw []byte, events []scrollbackEvent, plain bool, historyLimit, rowLimit int, limits vt.Limits) (replayed, error) {
	if err := ctx.Err(); err != nil {
		return replayed{}, err
	}
	cols, rows := initialSize(events)
	em, err := vt.NewEmulatorWithLimits(cols, rows, limits)
	if err != nil {
		return replayed{}, fmt.Errorf("initial resize %dx%d: %w", cols, rows, err)
	}
	defer em.Close()
	em.Scrollback().SetMaxLines(historyLimit)
	// Replay has no controlling app; discard terminal replies synchronously.
	em.SetReplyWriter(io.Discard)
	marks, err := feedSegmentsContext(ctx, em, raw, events, rowLimit > 0)
	if rowLimit > 0 && em.HistoryCapacityLoss() != 0 {
		return replayed{}, fmt.Errorf("rendered history exceeds capacity (%d rows, %d cells, %d bytes); lost %d rows", rowLimit, limits.HistoryCells, limits.HistoryBytes, em.HistoryCapacityLoss())
	}
	if err != nil {
		return replayed{}, err
	}

	if rowLimit > 0 && em.Scrollback().Len() > rowLimit {
		return replayed{}, fmt.Errorf("rendered history exceeds %d rows", rowLimit)
	}

	// Scrollback lines (oldest → newest), then visible buffer top → bottom.
	// Visible buffer iterates by row index rather than dropping trailing
	// blank rows: an agent that cleared and paused mid-redraw would shift
	// every subsequent line number otherwise, and `:880` should still land
	// where zellij showed line 880.
	sb := em.Scrollback()
	viewportTop := sb.Len() + 1 // 1-indexed line where the visible buffer starts
	out := make([]string, 0, sb.Len()+em.Height())
	for i := 0; i < sb.Len(); i++ {
		if err := ctx.Err(); err != nil {
			return replayed{}, err
		}
		out = append(out, serializeRow(sb.Line(i), plain))
	}
	w := em.Width()
	for y := 0; y < em.Height(); y++ {
		if err := ctx.Err(); err != nil {
			return replayed{}, err
		}
		out = append(out, serializeRow(visibleRow(em, y, w), plain))
	}
	// Trim trailing all-blank lines: a half-empty visible buffer otherwise
	// leaves a tail of empties at EOF.
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	if rowLimit > 0 && len(out) > rowLimit {
		return replayed{}, fmt.Errorf("rendered output exceeds %d rows", rowLimit)
	}
	return replayed{lines: out, viewportTop: viewportTop, marks: marks}, nil
}

func Run(argv []string, stdout, stderr io.Writer) int {
	return RunWithEnv(argv, func(string) string { return "" }, stdout, stderr)
}

// RunWithEnv is the managed CLI entry; in-process callers may use Run when a
// surrounding lease already protects the input, or pass explicit owner flags.
func RunWithEnv(argv []string, getenv func(string) string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("pair-scrollback-render", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "usage: pair-scrollback-render [--plain] [--viewport F] [--max-lines N] [--with-timestamps] <raw> <events.jsonl> <out>\n")
	}
	ownerDir := fs.String("owner-dir", "", "exact selected storage directory")
	ownerScope := fs.String("owner-scope", "", "source repository scope")
	ownerTag := fs.String("owner-tag", "", "source owner tag")
	ownerIntent := fs.String("owner-intent", "", "source selection handoff intent")
	plain := fs.Bool("plain", false, "emit plain text (no SGR) for distillation")
	maxLines := fs.Int("max-lines", historyRows, "scrollback history rows retained; <=0 = uncapped")
	withTimestamps := fs.Bool("with-timestamps", false, "interleave ⟦pair:ts DATE⟧ day markers from time events (for the change log; #59)")
	viewport := fs.String("viewport", "", "exact viewport sidecar path (required for ANSI output)")
	if err := fs.Parse(argv); err != nil {
		return 2
	}
	args := fs.Args()
	if len(args) != 3 {
		fs.Usage()
		return 2
	}
	if *ownerDir != "" || *ownerScope != "" || *ownerTag != "" {
		original := getenv
		getenv = func(key string) string {
			switch key {
			case "PAIR_DATA_DIR":
				return *ownerDir
			case "PAIR_SCOPE_KEY":
				return *ownerScope
			case "PAIR_TAG":
				return *ownerTag
			}
			if original != nil {
				return original(key)
			}
			return ""
		}
	}
	lease, err := acquireRenderLease(getenv, args[0], args[1])
	if err != nil {
		fmt.Fprintf(stderr, "scrollback-render: retention: %v\n", err)
		return 1
	}
	defer lease.Close()
	if err := acknowledgeRenderHandoff(lease, *ownerIntent, args[0]); err != nil {
		fmt.Fprintf(stderr, "scrollback-render: retention handoff: %v\n", err)
		return 1
	}
	if err := render(args[0], args[1], args[2], *viewport, *plain, *maxLines, *withTimestamps); err != nil {
		fmt.Fprintf(stderr, "scrollback-render: %v\n", err)
		return 1
	}
	return 0
}
