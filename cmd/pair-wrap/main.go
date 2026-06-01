// pair-wrap — transparent PTY proxy around a TUI coding agent.
//
// Installed at bin/pair-wrap and invoked by zellij/layouts/main.kdl on
// pair startup. (Originally ported from a Python prototype, #000011; the
// Python original was retired in #000019.)
//
// What it does:
//   - Spawns the agent in a fresh pty so the wrapper sees the raw output.
//   - Forwards stdin → agent and agent → stdout transparently.
//   - On agent OSC 9 / OSC 777 (and optional bare BEL), writes OSC 9
//     directly to pair's recorded outer-TTY — bypassing zellij, which
//     would otherwise eat the OSC.
//   - Per-agent notify mode: native (forward agent's OSC), idle (after
//     no output for IDLE_S), or marker (on first sighting of an
//     end-of-turn regex over extracted colored spans).
//   - SGR span extraction: per-foreground-color byte-level state machine
//     building an LRU of 1000 unique colored spans, written atomically
//     to agent-output-<tag> for nvim's autocomplete pickup.
//   - Optional --scrollback-log <path> tee with .events.jsonl sidecar
//     recording resize events keyed by byte offset — feeds Alt+/.
//   - Image-paste capture: SIGUSR1 arms a 0.9s window that buffers agent
//     output, writes image-capture-<tag>, touches .done sentinel.
//   - Startup banner; per-feature debug log via PAIR_WRAP_LOG.
//
// Failure mode: any error in detection / emission / capture is logged
// (when PAIR_WRAP_LOG is set) and swallowed. The proxy never blocks the
// agent on a logging hiccup.
package main

import (
	"container/list"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// ----- Tunables ---------------------------------------------------------------

const (
	rateLimitS          = 500 * time.Millisecond
	slugDebounceS       = 1 * time.Second // min gap between pair-slug spawns (#000027)
	agentOutputSpansMax = 1000
	agentSpanMax        = 512
	rollingTailLen      = 512
	pendingMax          = 64 // cap on incomplete-escape carryover
	readBufSize         = 4096
)

var (
	// IDLE_S default — overridable via PAIR_WRAP_IDLE_S. 0 disables.
	defaultIdleS = 60 * time.Second

	// Image capture window — overridable via PAIR_WRAP_CAPTURE_S.
	defaultCaptureWindow = 900 * time.Millisecond
)

// Per-agent notify mode (mode that owns emit_outer for that agent run).
// Anything not listed here uses notifyModeDefault.
var notifyMode = map[string]string{
	"claude": "marker",
}

const notifyModeDefault = "native"

// Per-agent end-of-turn pattern, applied only in "marker" notify mode.
// Matched against finalized colored spans (post-SGR-stripping by the
// span extractor). The Python regex (raw bytes form):
//
//	rb"^\xe2\x9c\xbb\s*[A-Za-z]+\s+for\s+\d+[hms](?:\s+\d+[hms])*"
//
// ✻ = U+273B = 0xE2 0x9C 0xBB in UTF-8. Anchored on ✻ so the
// quoted-history form ("> ✻ Churned for 21s", different color) won't
// double-emit. Durations accept multiple `\d+[hms]` parts: 1m 52s, 2h 13m 4s, etc.
var endOfTurnByAgent = map[string]*regexp.Regexp{
	"claude": regexp.MustCompile(`^\x{273B}\s*[A-Za-z]+\s+for\s+\d+[hms](?:\s+\d+[hms])*`),
}

// Agents we trust the colored-span extractor to handle. Outside this set,
// span extraction is disabled (autocomplete falls back to draft alone,
// and "marker" notify mode becomes a no-op for that agent — a caller
// config error worth logging but not crashing on).
var spanExtractionAgents = map[string]bool{
	"claude": true,
}

// Per-agent stdin keymap. The pair-managed nvim draft uses
//
//	Enter      = insert newline
//	Alt+Enter  = send to agent
//
// but the agent's native TUI typically uses Enter = send. That mismatch
// is jarring when the user moves between panes. When PAIR_WRAP_REMAP_RETURN
// isn't "0", pair-wrap rewrites stdin so the agent receives the inverted
// mapping: incoming Enter becomes the agent's "insert newline" sequence,
// incoming Alt+Enter becomes a plain Enter (send).
//
//   - plainCR:   bytes emitted when the user hits Enter alone (\r)
//   - altCR:     bytes emitted when the user hits Alt+Enter (\x1b\r)
//
// Claude reads "\<Enter>" (backslash + CR) as a newline regardless of
// terminal keyboard-protocol support — this is the documented portable
// path. Other agents need their own probing; leave them out of the table
// to fall through to no-rewrite (today's pass-through behavior).
type sendKeymap struct {
	plainCR, altCR []byte
}

var sendKeymapByAgent = map[string]sendKeymap{
	"claude": {
		// Claude reads `\<Enter>` as newline regardless of terminal
		// keyboard-protocol level — the documented portable path.
		plainCR: []byte{'\\', '\r'},
		altCR:   []byte{'\r'},
	},
	"codex": {
		// Codex follows the textbook chat-UI convention: Enter = send,
		// Shift+Enter = newline. Under Ghostty's KKP level-1
		// negotiation, Shift+Enter comes through as a literal LF
		// (\n) and plain Enter stays as \r. Probed via
		// PAIR_WRAP_LOG=… PAIR_WRAP_REMAP_RETURN=0.
		plainCR: []byte{'\n'},
		altCR:   []byte{'\r'},
	},
	"gemini": {
		// Same Enter/Shift+Enter convention as codex. Gemini explicitly
		// disables KKP at startup (\x1b[?0u) so all special keys arrive
		// as legacy bytes; the row is identical to codex's.
		plainCR: []byte{'\n'},
		altCR:   []byte{'\r'},
	},
}

type overlayDetector func(*proxy, []byte, []byte) (bool, string)

var overlayDetectorByAgent = map[string]overlayDetector{
	"claude": detectClaudeOverlayOpen,
	"codex":  detectCodexOverlayOpen,
}

// ----- Compiled regexes (byte-mode) -------------------------------------------

var (
	sgrRe         = regexp.MustCompile(`\x1b\[([0-9;]*)m`)
	otherEscRe    = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]|\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)|\x1b[()*+][@-~]|\x1b[@-Z\\-_]`)
	imageMarkerRe = regexp.MustCompile(`\[Image[ #][^\]]+\]`)
	oscRe         = regexp.MustCompile(`\x1b\](\d+);([^\x07\x1b]*)(?:\x07|\x1b\\)`)
)

// ----- State ------------------------------------------------------------------

// proxy holds all mutable wrapper state. Fields touched only from the main
// loop don't need locking; the few touched from signal goroutines (capture
// window, notify-mode flags) are guarded explicitly.
type proxy struct {
	// CLI / config
	scrollbackLog string
	agentBasename string
	debugLogPath  string
	bellFallback  bool

	// Resolved paths (empty when env didn't provide PAIR_TAG)
	outerTTYFile    string
	agentOutputFile string
	captureOutPath  string
	captureDonePath string
	capturePIDPath  string
	agentPIDPath    string

	// PTY
	ptmx *os.File
	cmd  *exec.Cmd

	// Notify
	notifyModeActive string
	endOfTurnRe      *regexp.Regexp
	idleS            time.Duration

	// Stdin Return-key remap. Zero-value (empty plainCR + altCR) means
	// pass-through. Populated from sendKeymapByAgent unless the user
	// opts out via PAIR_WRAP_REMAP_RETURN=0.
	sendKM sendKeymap

	// pickerActive is set when the active agent's output stream signals
	// that a blocking overlay / picker opened. While set,
	// translateChunk emits a bare \r for the user's plain Enter
	// instead of the textarea-aware remap, so the overlay confirms.
	// The flag clears after the first plain Enter is consumed —
	// restoring normal remap for the next Enter, which is back in the
	// textarea. Set from masterPump (handleChunk), read+cleared from
	// translateChunk → atomic.
	pickerActive atomic.Bool

	// Codex does not expose a dedicated overlay OSC today, so its
	// detector watches newly arrived visible text plus this carryover for
	// split picker labels. Keeping it separate from the OSC rolling tail
	// avoids re-detecting stale picker text after Enter clears
	// pickerActive.
	overlayMu       sync.Mutex
	overlayTextTail string

	// Scrollback log (-1 / nil when disabled)
	scrollbackFD    *os.File
	eventsFD        *os.File
	scrollbackBytes int64

	// OSC rate limiting
	lastEmit time.Time
	// pair-slug spawn debounce (#000027)
	lastSlug time.Time

	// Span LRU. spans maps key="<color>\t<text>" → *spanEntry; order keeps
	// insertion order, oldest at Front, newest at Back. Move-to-back on
	// re-emission. Cap by popping from Front when size > limit.
	spans     map[string]*spanEntry
	spanOrder *list.List

	// Byte-level SGR state
	agentFG      []byte // nil = default fg; "34" / "5;75" / "2;R;G;B"
	agentSpanBuf []byte
	agentSpanFG  []byte
	agentPending []byte // carry-over for split escapes between chunks

	// Image capture (signal-driven; guarded)
	captureMu       sync.Mutex
	captureActive   bool
	captureBuffer   []byte
	captureDeadline time.Time
	captureWindow   time.Duration
}

type spanEntry struct {
	color []byte
	text  []byte
	count int
	elem  *list.Element // pointer into spanOrder for O(1) move/remove
}

// ----- Path resolution --------------------------------------------------------

// dataDir returns $PAIR_DATA_DIR or the XDG default. Mirrors the Python.
func dataDir() string {
	if d := os.Getenv("PAIR_DATA_DIR"); d != "" {
		return d
	}
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, "pair")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "pair")
}

func (p *proxy) resolvePaths() {
	tag := os.Getenv("PAIR_TAG")
	if tag == "" {
		return
	}
	dir := dataDir()
	p.outerTTYFile = filepath.Join(dir, "outer-tty-"+tag)
	if spanExtractionAgents[p.agentBasename] {
		p.agentOutputFile = filepath.Join(dir, "agent-output-"+tag)
	}
	p.captureOutPath = filepath.Join(dir, "image-capture-"+tag)
	p.captureDonePath = p.captureOutPath + ".done"
	p.capturePIDPath = filepath.Join(dir, "pair-wrap-pid-"+tag)
	p.agentPIDPath = filepath.Join(dir, "agent-pid-"+tag)
}

// ----- Debug log --------------------------------------------------------------

// debug appends a one-line forensic record when PAIR_WRAP_LOG is set.
// Safe to call always; the gate is one map lookup. Snippets are escape-
// printed (Go's %q) and capped at 240 chars to match the Python's repr-
// style entries.
func (p *proxy) debug(label, ctx string) {
	if p.debugLogPath == "" {
		return
	}
	f, err := os.OpenFile(p.debugLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	if len(ctx) > 240 {
		ctx = ctx[:240]
	}
	fmt.Fprintf(f, "[%s] %s: %q\n", time.Now().Format("15:04:05"), label, ctx)
}

// ----- Outer-TTY OSC emit -----------------------------------------------------

// emitOuter writes \x1b]9;<msg>\x07 to the path recorded in outerTTYFile.
// maybeSpawnSlug fires pair-slug in the background to refresh the orientation
// slug (#000027 M3). Debounced by slugDebounceS so closely-spaced turn-end
// signals don't double-spawn. pair-slug self-gates (no-op without PAIR_TAG)
// and is non-fatal, so this is fire-and-forget. PAIR_AGENT tells it which
// session-file format to parse; cwd is inherited (the agent's repo → branch).
//
// Cost note: this runs once per turn-end, and pair-slug must call the small
// model before it can know the answer is KEEP — so steady-state cost is ~one
// haiku call per agent turn. The 1s debounce only collapses bursts, not the
// per-turn baseline; that's the accepted price of an always-current slug.
func (p *proxy) maybeSpawnSlug() {
	now := time.Now()
	if !p.lastSlug.IsZero() && now.Sub(p.lastSlug) < slugDebounceS {
		return
	}
	p.lastSlug = now
	p.debug("SLUG-spawn", "agent="+p.agentBasename)
	go func() {
		cmd := exec.Command("pair-slug")
		cmd.Env = append(os.Environ(), "PAIR_AGENT="+p.agentBasename)
		_ = cmd.Run()
	}()
}

// Rate-limited: any call within rateLimitS of the last successful emit is
// silently dropped. All errors are swallowed — never blocks the proxy.
func (p *proxy) emitOuter(msg string) {
	if msg == "" {
		msg = "agent attention"
	}
	// Turn-end is also when the orientation slug should refresh (#000027).
	// This is pair's agent-agnostic notify sink (marker/idle/native all land
	// here), so it works for claude/codex/gemini alike — no claude Stop hook.
	p.maybeSpawnSlug()
	now := time.Now()
	if !p.lastEmit.IsZero() && now.Sub(p.lastEmit) < rateLimitS {
		p.debug("EMIT-skip", fmt.Sprintf("rate-limited (%.2fs since last)", now.Sub(p.lastEmit).Seconds()))
		return
	}
	if p.outerTTYFile == "" {
		p.debug("EMIT-skip", "no outer-tty file resolved")
		return
	}
	pathBytes, err := os.ReadFile(p.outerTTYFile)
	if err != nil {
		p.debug("EMIT-fail", fmt.Sprintf("%s: %v", p.outerTTYFile, err))
		return
	}
	path := strings.TrimSpace(strings.SplitN(string(pathBytes), "\n", 2)[0])
	if path == "" {
		p.debug("EMIT-skip", "outer-tty file empty")
		return
	}
	// O_NONBLOCK so a stuck reader on the other end can't wedge us.
	fd, err := unix.Open(path, unix.O_WRONLY|unix.O_NONBLOCK, 0)
	if err != nil {
		p.debug("EMIT-fail", fmt.Sprintf("%s: %v", path, err))
		return
	}
	defer unix.Close(fd)
	osc := fmt.Sprintf("\x1b]9;%s\x07", msg)
	if _, err := unix.Write(fd, []byte(osc)); err != nil {
		p.debug("EMIT-fail", fmt.Sprintf("%s: %v", path, err))
		return
	}
	p.lastEmit = now
	p.debug("EMIT", "wrote OSC 9 to "+path)
}

// pickerOpenOSCBody is the OSC 777 body claude emits when a blocking
// overlay (AskUserQuestion picker or tool-permission prompt) opens.
// Distinct from the end-of-turn body "Claude is waiting for your
// input" — only this variant means "Enter routes to the overlay, not
// the textarea." Used to suspend the textarea-aware Enter remap.
const pickerOpenOSCBody = "notify;Claude Code;Claude needs your permission"

var codexPickerMarkers = []string{
	// Codex 0.134.0 resume-CWD picker. Both labels are visible in the
	// overlay; either is enough to know Enter should select, not insert
	// a textarea newline.
	"Use session directory (",
	"Use current directory (",

	// Generic picker footer observed in Codex blocking prompts. Keep as
	// a fallback for picker variants that do not include cwd choices.
	"Press enter to continue",

	// Codex ask-user-question choices render the recommended option label
	// inline, without the resume-CWD labels above.
	"(Recommended)",
}

func detectClaudeOverlayOpen(_ *proxy, _ []byte, rolling []byte) (bool, string) {
	matches := oscRe.FindAllSubmatch(rolling, -1)
	for _, m := range matches {
		if len(m) >= 3 && string(m[1]) == "777" && string(m[2]) == pickerOpenOSCBody {
			return true, string(m[2])
		}
	}
	return false, ""
}

func detectCodexOverlayOpen(p *proxy, data, _ []byte) (bool, string) {
	visible := stripTerminalControls(data)
	if p != nil {
		p.overlayMu.Lock()
		defer p.overlayMu.Unlock()
		visible = p.overlayTextTail + visible
		p.overlayTextTail = textSuffix(visible, rollingTailLen)
	}
	return detectCodexOverlayText(visible)
}

func detectCodexOverlayText(visible string) (bool, string) {
	for _, marker := range codexPickerMarkers {
		if strings.Contains(visible, marker) {
			return true, marker
		}
	}
	return false, ""
}

func stripTerminalControls(raw []byte) string {
	stripped := otherEscRe.ReplaceAll(raw, nil)
	stripped = bytesReplaceAll(stripped, '\r')
	return string(stripped)
}

func textSuffix(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[len(s)-max:]
}

// isActionableOSC decides whether an OSC <ps>;<body> should be forwarded.
// Skip 0/1/2 (title sets — claude updates every second with a spinner),
// 9;4;... (iTerm progress codes), and 1337 (iTerm proprietary). Forward
// 777 (urxvt notification) and 9 with non-"4;" body (iTerm notification).
func isActionableOSC(ps, body []byte) bool {
	switch string(ps) {
	case "777":
		return true
	case "9":
		return !strings.HasPrefix(string(body), "4;")
	}
	return false
}

// ----- Span LRU + agent-output file -------------------------------------------

// pushSpan adds (color, text) to the LRU. Returns true iff this is a new
// span (count == 1 after the call); existing entries get count++ and a
// move-to-back. The caller's distinction between "new" and "re-emission"
// drives both endOfTurn matching and the decision to flush the file.
func (p *proxy) pushSpan(color, text []byte) (isNew bool, entry *spanEntry) {
	key := string(color) + "\t" + string(text)
	if e, ok := p.spans[key]; ok {
		e.count++
		p.spanOrder.MoveToBack(e.elem)
		return false, e
	}
	e := &spanEntry{color: color, text: text, count: 1}
	e.elem = p.spanOrder.PushBack(key)
	p.spans[key] = e
	for len(p.spans) > agentOutputSpansMax {
		front := p.spanOrder.Front()
		if front == nil {
			break
		}
		oldKey := front.Value.(string)
		delete(p.spans, oldKey)
		p.spanOrder.Remove(front)
	}
	return true, e
}

// flushAgentFile atomically writes the LRU snapshot to agentOutputFile.
// Format: `<color>\t<count>\t<text>\n` per line, oldest first (front of
// list) — nvim ranks newer (later lines) higher. Errors are swallowed.
func (p *proxy) flushAgentFile() {
	if p.agentOutputFile == "" {
		return
	}
	tmp := p.agentOutputFile + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return
	}
	for el := p.spanOrder.Front(); el != nil; el = el.Next() {
		key := el.Value.(string)
		e := p.spans[key]
		fmt.Fprintf(f, "%s\t%d\t%s\n", e.color, e.count, e.text)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return
	}
	_ = os.Rename(tmp, p.agentOutputFile)
}

// finalizeSpan trims the in-progress span and pushes it on the LRU.
// Returns true iff a new (not-seen-before) span was added. On a new
// claude marker match, also fires emitOuter("<text>").
func (p *proxy) finalizeSpan() bool {
	text := bytesTrimSpace(p.agentSpanBuf)
	color := p.agentSpanFG
	p.agentSpanBuf = nil
	p.agentSpanFG = nil
	if len(text) == 0 || color == nil {
		return false
	}
	isNew, _ := p.pushSpan(color, text)
	if !isNew {
		return false
	}
	if p.endOfTurnRe != nil && p.endOfTurnRe.Match(text) {
		msg := string(text)
		p.debug("END-OF-TURN", msg)
		p.emitOuter(msg)
	}
	return true
}

// bytesTrimSpace is bytes.TrimSpace (avoiding import noise; pure ASCII path).
func bytesTrimSpace(b []byte) []byte {
	i, j := 0, len(b)
	for i < j && asciiSpace(b[i]) {
		i++
	}
	for j > i && asciiSpace(b[j-1]) {
		j--
	}
	return b[i:j]
}
func asciiSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\v' || c == '\f'
}

// extractFG applies an SGR parameter list and returns the new FG color id.
// Returns nil for default FG; otherwise one of:
//   - "34"…"97" for the 16-color codes
//   - "5;N" for 256-color (CSI 38;5;N)
//   - "2;R;G;B" for truecolor (CSI 38;2;R;G;B)
//
// Background, bold/underline, and other attributes are ignored.
func extractFG(params, current []byte) []byte {
	if len(params) == 0 {
		// "\x1b[m" — full reset.
		return nil
	}
	parts := splitBytes(params, ';')
	fg := current
	for i := 0; i < len(parts); i++ {
		p := parts[i]
		n, err := strconv.Atoi(string(p))
		if err != nil && len(p) > 0 {
			continue
		}
		if len(p) == 0 {
			n = 0
		}
		switch {
		case n == 0 || n == 39:
			fg = nil
		case (n >= 30 && n <= 37) || (n >= 90 && n <= 97):
			fg = []byte(strconv.Itoa(n))
		case n == 38 && i+1 < len(parts):
			if string(parts[i+1]) == "5" && i+2 < len(parts) {
				fg = append([]byte("5;"), parts[i+2]...)
				i += 2
			} else if string(parts[i+1]) == "2" && i+4 < len(parts) {
				fg = append(append(append(append(append(
					[]byte("2;"), parts[i+2]...), ';'),
					parts[i+3]...), ';'), parts[i+4]...)
				i += 4
			}
		}
	}
	return fg
}

func splitBytes(b []byte, sep byte) [][]byte {
	var out [][]byte
	start := 0
	for i := 0; i < len(b); i++ {
		if b[i] == sep {
			out = append(out, b[start:i])
			start = i + 1
		}
	}
	out = append(out, b[start:])
	return out
}

// updateAgentOutput walks a chunk byte-by-byte, capturing per-color FG runs
// as spans. State machine mirrors the Python:
//
//   - \x1b → consume an SGR (updates agentFG; finalize current span if
//     FG changed mid-span) or any other complete escape (skipped). Split
//     escapes at chunk end carry over via agentPending (capped at 64B
//     so a malformed stream can't grow it unbounded).
//   - \n / \r → finalize current span (line breaks don't span runs).
//   - any other byte → append to span buffer iff a color is active. Auto-
//     finalize past agentSpanMax to bound memory on pathological streams.
func (p *proxy) updateAgentOutput(data []byte) {
	if p.agentOutputFile == "" {
		return
	}
	if len(p.agentPending) > 0 {
		data = append(p.agentPending, data...)
		p.agentPending = nil
	}
	newSpans := 0
	i, n := 0, len(data)
	for i < n {
		b := data[i]
		if b == 0x1b {
			if loc := sgrRe.FindSubmatchIndex(data[i:]); loc != nil && loc[0] == 0 {
				newFG := extractFG(data[i+loc[2]:i+loc[3]], p.agentFG)
				if !bytesEqual(newFG, p.agentFG) && len(p.agentSpanBuf) > 0 {
					if p.finalizeSpan() {
						newSpans++
					}
				}
				p.agentFG = newFG
				i += loc[1]
				continue
			}
			if loc := otherEscRe.FindIndex(data[i:]); loc != nil && loc[0] == 0 {
				// Cursor-positioning escapes inside an active colored
				// run mean the agent skipped one or more cells without
				// repainting (typically blanks). Claude's TUI in
				// particular paints inline code char-by-char and uses
				// CUF to jump over spaces, so without this we'd merge
				// `make nous-install` into the unusable autocomplete
				// candidate `makenous-install`. Drop in a single-space
				// placeholder to preserve word boundaries. Mirrors the
				// fix in bin/pair-wrap.py (949aeec).
				if len(p.agentSpanBuf) > 0 && p.agentSpanBuf[len(p.agentSpanBuf)-1] != ' ' {
					p.agentSpanBuf = append(p.agentSpanBuf, ' ')
				}
				i += loc[1]
				continue
			}
			// Incomplete escape at chunk end — carry over if remaining
			// tail is plausibly an escape; else drop the lone ESC byte.
			if n-i < pendingMax {
				p.agentPending = append([]byte(nil), data[i:]...)
				break
			}
			i++
			continue
		}
		if b == '\n' || b == '\r' {
			if len(p.agentSpanBuf) > 0 {
				if p.finalizeSpan() {
					newSpans++
				}
			}
			i++
			continue
		}
		if p.agentFG != nil {
			if len(p.agentSpanBuf) == 0 {
				p.agentSpanFG = p.agentFG
			}
			p.agentSpanBuf = append(p.agentSpanBuf, b)
			if len(p.agentSpanBuf) > agentSpanMax {
				if p.finalizeSpan() {
					newSpans++
				}
			}
		}
		i++
	}
	if newSpans > 0 {
		p.flushAgentFile()
	}
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// ----- Image-paste capture ----------------------------------------------------

// armCapture is called from the SIGUSR1 handler goroutine. Coalesces
// repeated signals into a single window by extending the deadline.
// Clears any stale .done sentinel so nvim doesn't read a previous result.
func (p *proxy) armCapture() {
	if p.captureOutPath == "" {
		return
	}
	p.captureMu.Lock()
	defer p.captureMu.Unlock()
	_ = os.Remove(p.captureDonePath)
	deadline := time.Now().Add(p.captureWindow)
	if p.captureActive {
		p.captureDeadline = deadline
		p.debug("CAPTURE-extend", fmt.Sprintf("window now ends at %v", deadline.Format("15:04:05.000")))
		return
	}
	p.captureBuffer = nil
	p.captureActive = true
	p.captureDeadline = deadline
	p.debug("CAPTURE-start", fmt.Sprintf("window %.3fs", p.captureWindow.Seconds()))
}

// finalizeCapture writes the buffered bytes and touches the .done sentinel.
// Idempotent — guarded so the early-finalize fast path and the deadline
// expiry both reach a clean no-op after the first write. Without this the
// second call would clobber the buffered file with empty bytes and race
// nvim's read of the first write.
func (p *proxy) finalizeCapture() {
	p.captureMu.Lock()
	if !p.captureActive {
		p.captureMu.Unlock()
		return
	}
	buf := p.captureBuffer
	p.captureActive = false
	p.captureBuffer = nil
	out, done := p.captureOutPath, p.captureDonePath
	p.captureMu.Unlock()
	if err := os.WriteFile(out, buf, 0644); err != nil {
		p.debug("CAPTURE-write-fail", err.Error())
		return
	}
	// Touch sentinel last — nvim polls for it and ordering guarantees the
	// buffer file is fully written by the time the sentinel exists.
	if f, err := os.Create(done); err == nil {
		f.Close()
	}
	p.debug("CAPTURE-done", fmt.Sprintf("%d bytes", len(buf)))
}

// maybeFinalizeEarly: if the agent's image-marker is visible in the
// captured buffer, finalize now rather than waiting out the rest of
// captureWindow. Drops Alt+i tail latency from ~0.9s to ~(agent render
// time + one nvim poll tick). Buffer is typically <1 KB during a window
// so the strip+regex pass is cheap.
func (p *proxy) maybeFinalizeEarly() {
	p.captureMu.Lock()
	active := p.captureActive
	buf := p.captureBuffer
	p.captureMu.Unlock()
	if !active {
		return
	}
	stripped := otherEscRe.ReplaceAll(buf, nil)
	stripped = bytesReplaceAll(stripped, '\r')
	if imageMarkerRe.Match(stripped) {
		p.debug("CAPTURE-early", fmt.Sprintf("marker visible after %d bytes", len(buf)))
		p.finalizeCapture()
	}
}

// bytesReplaceAll removes all occurrences of c from b. (Avoids the dep on
// bytes.ReplaceAll for a single-byte case; the buffer is small here.)
func bytesReplaceAll(b []byte, c byte) []byte {
	out := b[:0:len(b)]
	for _, x := range b {
		if x != c {
			out = append(out, x)
		}
	}
	return out
}

// ----- Stdin Return-key remap -------------------------------------------------

// Bracketed-paste markers. Modern terminals wrap pasted text in
// ESC[200~ ... ESC[201~ when DECSET 2004 is active. Claude (and most
// modern TUIs) enables it. We MUST NOT rewrite \r bytes inside a paste
// — those are literal newlines from the source content, not user
// keystrokes that mean "send."
var (
	bpStart = []byte("\x1b[200~")
	bpEnd   = []byte("\x1b[201~")
)

// Enter / Alt+Enter byte sequences across the two protocols modern
// terminals use:
//
//   - Legacy ("cooked"): plain Enter = \r, Alt+Enter = \x1b\r.
//   - Kitty keyboard protocol (KKP): plain Enter = \x1b[13u (or the
//     explicit-no-modifier form \x1b[13;1u), Alt+Enter = \x1b[13;3u
//     (modifier param 3 = alt). Claude enables KKP when it starts; if
//     the host terminal supports it (Ghostty, kitty, WezTerm, recent
//     iTerm) the user's keystrokes arrive in this form instead of the
//     legacy bytes.
//
// pair-wrap must recognize both — the user can't be expected to know
// which protocol their terminal is negotiating. Matching is greedy on
// the longer KKP forms first so e.g. \x1b[13;3u doesn't get partially
// matched as \x1b[13u.
var (
	enterLegacyPlain = []byte("\r")
	enterLegacyAlt   = []byte("\x1b\r")
	enterKKPPlain    = []byte("\x1b[13u")
	enterKKPPlainExp = []byte("\x1b[13;1u") // explicit-no-modifier form
	enterKKPAlt      = []byte("\x1b[13;3u")
)

// holdbackPatterns lists every multi-byte marker the translator might
// need to complete across a chunk boundary. If a chunk ends with bytes
// that form a strict prefix of any pattern here, hold those bytes back
// to the next read.
var holdbackPatterns = [][]byte{
	bpStart, bpEnd,
	enterKKPPlain, enterKKPPlainExp, enterKKPAlt,
	enterLegacyAlt,
}

// pendingFlushAfter is the timeout for held-back bytes that haven't
// completed into a known marker. Real terminals dispatch chorded
// keystrokes (Alt+Enter, KKP CSI sequences) in microseconds, so 30 ms
// safely catches a split chord. A standalone ESC (e.g. nvim's
// send_esc_to_agent writes a lone \x1b for "interrupt the agent") waits
// at most this long before being flushed verbatim to the child.
const pendingFlushAfter = 30 * time.Millisecond

// translateStdin replaces the io.Copy(ptmx, os.Stdin) pass-through with
// a byte-stream translator that rewrites Return / Alt+Return per the
// resolved per-agent sendKM, while honoring bracketed-paste mode so
// pasted multi-line text passes through unchanged.
//
// Pipeline:
//   - a reader goroutine pumps stdin chunks into a channel
//   - the main select-loop combines each chunk with any pending bytes
//     from a partial sequence held over from the previous chunk, runs
//     translateChunk over the combined slice, writes the output to
//     the pty master, and stashes any new leftover as pending
//   - if pending is non-empty, a timer is armed; the timer firing
//     means "no continuation byte arrived within pendingFlushAfter —
//     this is a standalone sequence, flush it." Resets on every read.
//
// State machine (per translateChunk):
//   - in paste mode: scan for bpEnd, forward bytes verbatim
//   - otherwise: look for bpStart, KKP / legacy Alt+Enter, KKP plain
//     Enter, plain \r. Anything else passes through.
//   - chunk-tail that's a strict prefix of any known marker → held
//     over to the next read.
func (p *proxy) translateStdin() {
	p.translateStdinFrom(os.Stdin, p.ptmx, pendingFlushAfter)
}

// translateStdinFrom is the testable inner loop. Takes an explicit
// reader / writer / flush-timeout so tests can inject pipes and a
// shortened timeout. Production binds to os.Stdin + p.ptmx via the
// thin wrapper above.
func (p *proxy) translateStdinFrom(stdin io.Reader, out io.Writer, flushAfter time.Duration) {
	type readEv struct {
		data []byte
		err  error
	}
	ch := make(chan readEv, 4)
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdin.Read(buf)
			if n > 0 {
				cp := make([]byte, n)
				copy(cp, buf[:n])
				ch <- readEv{data: cp}
			}
			if err != nil {
				ch <- readEv{err: err}
				close(ch)
				return
			}
		}
	}()

	var pending []byte
	inPaste := false

	// Timer for flushing pending. Starts in a stopped+drained state so
	// the select can wait on it without an immediate spurious fire.
	flushTimer := time.NewTimer(time.Hour)
	if !flushTimer.Stop() {
		<-flushTimer.C
	}
	timerArmed := false
	armTimer := func() {
		if timerArmed {
			if !flushTimer.Stop() {
				select {
				case <-flushTimer.C:
				default:
				}
			}
		}
		flushTimer.Reset(flushAfter)
		timerArmed = true
	}
	disarmTimer := func() {
		if !timerArmed {
			return
		}
		if !flushTimer.Stop() {
			select {
			case <-flushTimer.C:
			default:
			}
		}
		timerArmed = false
	}

	flushPending := func() {
		if len(pending) == 0 {
			return
		}
		_, _ = out.Write(pending)
		pending = nil
		disarmTimer()
	}

	for {
		select {
		case ev, ok := <-ch:
			if !ok || ev.err != nil {
				// EOF / read error: flush whatever was held over —
				// nothing more is coming to complete the sequence.
				flushPending()
				return
			}
			data := ev.data
			if len(pending) > 0 {
				data = append(pending, data...)
				pending = nil
			}
			outBytes, leftover, newInPaste := p.translateChunk(data, inPaste)
			inPaste = newInPaste
			pending = leftover
			if len(outBytes) > 0 {
				if _, werr := out.Write(outBytes); werr != nil {
					return
				}
			}
			if len(pending) > 0 {
				armTimer()
			} else {
				disarmTimer()
			}
		case <-flushTimer.C:
			timerArmed = false
			flushPending()
		}
	}
}

// checkOverlayOpen flips pickerActive when the current agent's output
// indicates that a blocking overlay opened. Idempotent — repeated
// rerenders within one overlay don't re-debug-log.
func (p *proxy) checkOverlayOpen(data, rolling []byte) {
	detect, ok := overlayDetectorByAgent[p.agentBasename]
	if !ok {
		return
	}
	open, reason := detect(p, data, rolling)
	if !open {
		return
	}
	if !p.pickerActive.Load() {
		p.pickerActive.Store(true)
		p.debug("PICKER-open", p.agentBasename+": "+reason)
	}
}

// emitPlainCR appends bytes for a user "plain Enter" event, honoring
// the overlay-active state. While pickerActive is set, Enter goes
// through as a bare \r so the overlay confirms — and the flag clears,
// restoring the textarea-aware plainCR remap for the next Enter.
// See the pickerActive field doc for the open/close protocol.
func (p *proxy) emitPlainCR(out []byte) []byte {
	if p.pickerActive.Load() {
		p.pickerActive.Store(false)
		p.overlayMu.Lock()
		p.overlayTextTail = ""
		p.overlayMu.Unlock()
		return append(out, '\r')
	}
	return append(out, p.sendKM.plainCR...)
}

// translateChunk walks `data` and returns (rewritten bytes, leftover to
// carry over, new bracketed-paste state). `leftover` is non-nil only
// when the chunk ends mid-escape that could still resolve into bpStart,
// bpEnd, or an Alt+Enter — the caller prepends it to the next read.
func (p *proxy) translateChunk(data []byte, inPaste bool) ([]byte, []byte, bool) {
	out := make([]byte, 0, len(data))
	i := 0
	for i < len(data) {
		if inPaste {
			// Scan for end-of-paste marker. Anything before it is
			// literal pasted content — forward verbatim.
			if idx := indexOfSubseq(data[i:], bpEnd); idx >= 0 {
				out = append(out, data[i:i+idx+len(bpEnd)]...)
				i += idx + len(bpEnd)
				inPaste = false
				continue
			}
			// Marker not in this chunk. Forward everything but hold back
			// a trailing partial ESC[201~ in case it splits the boundary.
			tail := trailingPartial(data[i:], bpEnd)
			out = append(out, data[i:len(data)-tail]...)
			leftover := append([]byte(nil), data[len(data)-tail:]...)
			return out, leftover, true
		}

		b := data[i]
		// Outside paste: scan for the multi-byte markers and the
		// single-byte plain Enter. Longer KKP forms come first so a
		// 7-byte \x1b[13;3u doesn't get partially matched as the
		// 5-byte \x1b[13u.
		if b == 0x1b {
			if startsWith(data[i:], bpStart) {
				out = append(out, bpStart...)
				i += len(bpStart)
				inPaste = true
				continue
			}
			// KKP Alt+Enter: \x1b[13;3u → send.
			if startsWith(data[i:], enterKKPAlt) {
				out = append(out, p.sendKM.altCR...)
				i += len(enterKKPAlt)
				continue
			}
			// KKP plain Enter, explicit-no-modifier form: \x1b[13;1u.
			if startsWith(data[i:], enterKKPPlainExp) {
				out = p.emitPlainCR(out)
				i += len(enterKKPPlainExp)
				continue
			}
			// KKP plain Enter: \x1b[13u.
			if startsWith(data[i:], enterKKPPlain) {
				out = p.emitPlainCR(out)
				i += len(enterKKPPlain)
				continue
			}
			// Legacy Alt+Enter: \x1b\r.
			if startsWith(data[i:], enterLegacyAlt) {
				out = append(out, p.sendKM.altCR...)
				i += len(enterLegacyAlt)
				continue
			}
			// Could the chunk-tail still grow into one of our markers
			// on the next read? Hold back only if data[i:] is a strict
			// prefix of *some* known pattern — unrelated escapes (arrow
			// keys, CSI sequences, etc.) pass through.
			held := false
			for _, pat := range holdbackPatterns {
				if isPrefixOf(data[i:], pat) {
					held = true
					break
				}
			}
			if held {
				return out, append([]byte(nil), data[i:]...), false
			}
			// Lone trailing ESC could be the first byte of an Alt+Enter
			// arriving across a chunk boundary; hold it back.
			if i == len(data)-1 {
				return out, append([]byte(nil), data[i:]...), false
			}
			// Some other escape — pass the ESC byte through and let
			// the next iteration handle the rest of the sequence
			// naturally (each byte after ESC isn't special to us).
			out = append(out, b)
			i++
			continue
		}
		if b == '\r' {
			out = p.emitPlainCR(out)
			i++
			continue
		}
		out = append(out, b)
		i++
	}
	return out, nil, inPaste
}

// startsWith reports whether b starts with prefix.
func startsWith(b, prefix []byte) bool {
	if len(b) < len(prefix) {
		return false
	}
	for i := range prefix {
		if b[i] != prefix[i] {
			return false
		}
	}
	return true
}

// isPrefixOf reports whether short is a (possibly equal) prefix of long.
// Used to decide whether a chunk-tail could grow into a known marker
// (bpStart/bpEnd) on the next read, vs being some unrelated escape.
func isPrefixOf(short, long []byte) bool {
	if len(short) > len(long) {
		return false
	}
	for i := range short {
		if short[i] != long[i] {
			return false
		}
	}
	return true
}

// indexOfSubseq returns the index of the first occurrence of needle in
// haystack, or -1.
func indexOfSubseq(haystack, needle []byte) int {
	if len(needle) == 0 {
		return 0
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// trailingPartial returns the count of bytes at the tail of b that form
// a strict prefix of needle. Used to hold back potentially-split
// markers across chunk boundaries.
func trailingPartial(b, needle []byte) int {
	maxK := len(needle) - 1
	if maxK > len(b) {
		maxK = len(b)
	}
	for k := maxK; k > 0; k-- {
		tail := b[len(b)-k:]
		match := true
		for i := 0; i < k; i++ {
			if tail[i] != needle[i] {
				match = false
				break
			}
		}
		if match {
			return k
		}
	}
	return 0
}

// ----- Scrollback log ---------------------------------------------------------

// logScrollbackEvent writes one JSON record keyed by current
// scrollbackBytes offset (so the renderer can split feed-segments by
// resize boundary). No-op when --scrollback-log isn't enabled.
func (p *proxy) logScrollbackEvent(typ string, fields map[string]any) {
	if p.eventsFD == nil {
		return
	}
	payload := map[string]any{"type": typ, "offset": p.scrollbackBytes}
	for k, v := range fields {
		payload[k] = v
	}
	line, _ := json.Marshal(payload)
	line = append(line, '\n')
	if _, err := p.eventsFD.Write(line); err != nil {
		p.debug("EVENT-write-fail", err.Error())
	}
}

// setWinsize copies stdin's window size to the master ptm and emits a
// resize event into the scrollback sidecar. Called once at startup and
// again on each SIGWINCH.
func (p *proxy) setWinsize() {
	ws, err := pty.GetsizeFull(os.Stdin)
	if err != nil {
		return
	}
	if err := pty.Setsize(p.ptmx, ws); err != nil {
		return
	}
	p.logScrollbackEvent("resize", map[string]any{
		"cols": int(ws.Cols),
		"rows": int(ws.Rows),
	})
}

// ----- Startup banner ---------------------------------------------------------

// writeStartupBanner paints a one-line inverse-video banner before the
// pty fork so it's the first byte the pane sees. Most TUI agents clear
// or enter alt-screen on startup so the banner only persists briefly —
// long enough to flag that the user is inside pair-wrap.
func writeStartupBanner() {
	var cols int
	if ws, err := pty.GetsizeFull(os.Stdout); err == nil {
		cols = int(ws.Cols)
	}
	if cols == 0 {
		cols = 80
	}
	text := " ⚙  pair-wrapped · Alt+h for help · Happy coding! "
	if len(text) > cols {
		text = text[:cols]
	}
	if pad := cols - len(text); pad > 0 {
		text += strings.Repeat(" ", pad)
	}
	// Trailing CRLF puts cursor on row 2, blank line, agent starts row 3.
	os.Stdout.WriteString("\x1b[7m" + text + "\x1b[27m\r\n\r\n")
}

// ----- Main -------------------------------------------------------------------

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "pair-wrap: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	p := &proxy{
		spans:         make(map[string]*spanEntry),
		spanOrder:     list.New(),
		idleS:         envDuration("PAIR_WRAP_IDLE_S", defaultIdleS),
		captureWindow: envDuration("PAIR_WRAP_CAPTURE_S", defaultCaptureWindow),
		debugLogPath:  os.Getenv("PAIR_WRAP_LOG"),
		bellFallback:  envFlag("PAIR_WRAP_BELL_FALLBACK"),
	}

	// Argv: strip our own flags before resolving the command. argparse
	// would be heavier than needed; this matches the Python loop shape.
	argv := os.Args[1:]
	for len(argv) > 0 && strings.HasPrefix(argv[0], "-") {
		switch {
		case argv[0] == "--scrollback-log" && len(argv) > 1:
			p.scrollbackLog = argv[1]
			argv = argv[2:]
		case argv[0] == "--":
			argv = argv[1:]
			goto argsDone
		default:
			return fmt.Errorf("unknown flag %q", argv[0])
		}
	}
argsDone:
	if len(argv) == 0 {
		return errors.New("usage: pair-wrap [--scrollback-log <path>] <command> [args...]")
	}

	p.agentBasename = filepath.Base(argv[0])
	p.resolvePaths()

	// Open scrollback log (truncate) + matching .events.jsonl sidecar.
	// Disable scrollback entirely on any open failure; never block startup.
	if p.scrollbackLog != "" {
		eventsPath := strings.TrimSuffix(p.scrollbackLog, ".raw") + ".events.jsonl"
		if !strings.HasSuffix(p.scrollbackLog, ".raw") {
			eventsPath = p.scrollbackLog + ".events.jsonl"
		}
		f, err := os.OpenFile(p.scrollbackLog, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			p.debug("SCROLLBACK-open-fail", fmt.Sprintf("%q: %v", p.scrollbackLog, err))
		} else {
			p.scrollbackFD = f
			p.debug("SCROLLBACK-open", p.scrollbackLog)
			ef, err := os.OpenFile(eventsPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
			if err != nil {
				p.debug("EVENTS-open-fail", fmt.Sprintf("%q: %v", eventsPath, err))
			} else {
				p.eventsFD = ef
				p.debug("EVENTS-open", eventsPath)
			}
		}
	}

	// Pick notify mode + per-agent end-of-turn regex.
	if m, ok := notifyMode[p.agentBasename]; ok {
		p.notifyModeActive = m
	} else {
		p.notifyModeActive = notifyModeDefault
	}
	p.debug("NOTIFY-mode", fmt.Sprintf("%s=%s", p.agentBasename, p.notifyModeActive))
	if p.notifyModeActive == "marker" {
		if re, ok := endOfTurnByAgent[p.agentBasename]; ok {
			p.endOfTurnRe = re
		} else {
			p.debug("MARKER-missing", p.agentBasename+" has no endOfTurnByAgent entry")
		}
	}
	if p.notifyModeActive != "idle" {
		p.idleS = 0
	}

	// Resolve the per-agent stdin Return-key keymap unless the user has
	// disabled the rewrite via PAIR_WRAP_REMAP_RETURN=0. Empty struct
	// means pass-through.
	if os.Getenv("PAIR_WRAP_REMAP_RETURN") != "0" {
		if km, ok := sendKeymapByAgent[p.agentBasename]; ok {
			p.sendKM = km
			p.debug("REMAP-return", fmt.Sprintf(
				"%s: Enter→%q  Alt+Enter→%q",
				p.agentBasename, string(km.plainCR), string(km.altCR)))
		}
	}

	writeStartupBanner()

	// Spawn child in a fresh PTY.
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Env = os.Environ()
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return fmt.Errorf("cannot exec %s: %w", argv[0], err)
	}
	p.cmd = cmd
	p.ptmx = ptmx
	defer ptmx.Close()

	// Drop the agent's PID so pair-session-watch.sh can bind discovery to
	// this specific child (lsof -p <pid>) instead of racing peers in the
	// shared session dir. Best-effort: a failed write only degrades the
	// session-id capture for codex/gemini; claude doesn't need it.
	if p.agentPIDPath != "" {
		if err := os.WriteFile(p.agentPIDPath, []byte(strconv.Itoa(cmd.Process.Pid)), 0644); err != nil {
			p.debug("AGENT-PID-write-fail", err.Error())
			p.agentPIDPath = ""
		}
	}

	// Initial winsize copy + SIGWINCH handler.
	p.setWinsize()

	// Image-capture wiring. Drop the pidfile so nvim's Alt+i knows where
	// to send SIGUSR1; only enabled when PAIR_TAG/PAIR_DATA_DIR resolved
	// a valid output path.
	if p.captureOutPath != "" {
		if err := os.WriteFile(p.capturePIDPath, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
			p.debug("CAPTURE-arm-fail", err.Error())
			p.captureOutPath = "" // disable; armCapture bails on empty
		} else {
			p.debug("CAPTURE-arm",
				fmt.Sprintf("pid=%d window=%.3fs", os.Getpid(), p.captureWindow.Seconds()))
		}
	}

	// Raw mode on stdin: without it the kernel does canonical line discipline
	// (line buffering + echo + signal interpretation) on our input, which
	// double-echoes keystrokes and corrupts terminal escape responses bound
	// for the child. The child's slave pty has its own (raw) mode set by
	// the wrapped TUI; this only affects OUR view of stdin.
	var oldState *term.State
	if isTTY(os.Stdin) {
		s, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err == nil {
			oldState = s
		}
	}
	defer func() {
		if oldState != nil {
			_ = term.Restore(int(os.Stdin.Fd()), oldState)
		}
		if p.scrollbackFD != nil {
			_ = p.scrollbackFD.Close()
		}
		if p.eventsFD != nil {
			_ = p.eventsFD.Close()
		}
		if p.capturePIDPath != "" {
			// Drop pidfile so a future Alt+i doesn't signal a stale pid.
			// image-capture-* files are intentionally left alone here —
			// bin/pair's cleanup_quit_marker handles them with the rest
			// of $DATA_DIR on Alt+x.
			_ = os.Remove(p.capturePIDPath)
		}
		if p.agentPIDPath != "" {
			_ = os.Remove(p.agentPIDPath)
		}
	}()

	// Signal handling.
	sigCh := make(chan os.Signal, 4)
	signal.Notify(sigCh, syscall.SIGWINCH, syscall.SIGUSR1)
	go func() {
		for s := range sigCh {
			switch s {
			case syscall.SIGWINCH:
				p.setWinsize()
			case syscall.SIGUSR1:
				p.armCapture()
			}
		}
	}()

	// Main loop. One goroutine per direction; everything else is in
	// the master-reader goroutine (the only place per-chunk processing
	// happens, mirroring the Python's serial loop).
	stdinDone := make(chan struct{})
	go func() {
		// stdin → master. EOF on stdin doesn't kill the proxy — the child
		// may still be producing output. We just stop forwarding.
		if p.sendKM.plainCR == nil && p.sendKM.altCR == nil {
			// No remap configured. Pass-through, but log raw chunks
			// when PAIR_WRAP_LOG is set — useful for probing what
			// bytes a terminal sends for a given keystroke (e.g.
			// figuring out a new agent's send/newline encoding).
			if p.debugLogPath != "" {
				buf := make([]byte, 4096)
				for {
					n, err := os.Stdin.Read(buf)
					if n > 0 {
						p.debug("STDIN", fmt.Sprintf("%q", buf[:n]))
						if _, werr := ptmx.Write(buf[:n]); werr != nil {
							break
						}
					}
					if err != nil {
						break
					}
				}
			} else {
				_, _ = io.Copy(ptmx, os.Stdin)
			}
		} else {
			p.translateStdin()
		}
		close(stdinDone)
	}()

	p.masterPump()

	// Wait for the child and propagate its exit code.
	werr := cmd.Wait()
	if exitErr, ok := werr.(*exec.ExitError); ok {
		os.Exit(exitErr.ExitCode())
	}
	if werr != nil {
		return werr
	}
	return nil
}

// masterPump runs the per-chunk processing loop on data from the master
// PTY: tee to stdout + scrollback log, span extraction, OSC detection,
// idle timer, image-capture buffer + early-finalize. Replaces Python's
// select() loop with a Go select on:
//   - a read channel fed by a tiny reader goroutine
//   - the idle timer (only the main goroutine touches it → no races)
//   - a 50 ms capture-deadline tick
//
// Returns when the master pipe closes (child exited).
func (p *proxy) masterPump() {
	type readEv struct {
		data []byte
		err  error
	}
	ch := make(chan readEv, 4)
	// Reader goroutine. Copies into a fresh slice each time so the receiver
	// can hang onto the bytes without racing the next read.
	go func() {
		buf := make([]byte, readBufSize)
		for {
			n, err := p.ptmx.Read(buf)
			if n > 0 {
				cp := make([]byte, n)
				copy(cp, buf[:n])
				ch <- readEv{data: cp}
			}
			if err != nil {
				ch <- readEv{err: err}
				close(ch)
				return
			}
		}
	}()

	// Idle timer is owned by the main goroutine — set up stopped if not in
	// "idle" notify mode (p.idleS == 0), otherwise armed for p.idleS.
	idleTimer := time.NewTimer(time.Hour)
	if !idleTimer.Stop() {
		<-idleTimer.C
	}
	idleFired := false
	if p.idleS > 0 {
		idleTimer.Reset(p.idleS)
	}

	captureTick := time.NewTicker(50 * time.Millisecond)
	defer captureTick.Stop()

	rolling := make([]byte, 0, rollingTailLen*2)

	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return
			}
			if ev.err != nil {
				// EIO on master read after slave close is the normal
				// child-exit path; treat as quiet EOF. Same for any
				// PathError wrapping syscall.EIO.
				if errors.Is(ev.err, io.EOF) || isEIO(ev.err) {
					return
				}
				p.debug("MASTER-read-fail", ev.err.Error())
				return
			}
			p.handleChunk(ev.data, &rolling)
			if p.idleS > 0 {
				// Stop+drain+reset is safe here because only this
				// goroutine ever reads idleTimer.C.
				if !idleTimer.Stop() {
					select {
					case <-idleTimer.C:
					default:
					}
				}
				idleTimer.Reset(p.idleS)
				idleFired = false
			}
		case <-idleTimer.C:
			if p.idleS > 0 && !idleFired {
				p.debug("IDLE", fmt.Sprintf("no agent output for %.0fs", p.idleS.Seconds()))
				p.emitOuter("agent idle")
				idleFired = true
			}
		case <-captureTick.C:
			p.captureMu.Lock()
			due := p.captureActive && !time.Now().Before(p.captureDeadline)
			p.captureMu.Unlock()
			if due {
				p.finalizeCapture()
			}
		}
	}
}

// handleChunk owns the per-chunk work that used to live inline in the
// Python loop's `if master in r:` branch. Order matters:
//
//  1. capture-buffer tee + early-finalize (so even a brief window snags
//     the very next byte)
//  2. stdout tee (user sees the bytes ASAP)
//  3. scrollback log tee
//  4. span extraction
//  5. OSC/BEL detection on a rolling tail
//
// Each step is wrapped so a single failure can't take down the proxy —
// matches the Python's try/except pattern.
func (p *proxy) handleChunk(data []byte, rolling *[]byte) {
	p.captureMu.Lock()
	active := p.captureActive
	if active {
		p.captureBuffer = append(p.captureBuffer, data...)
	}
	p.captureMu.Unlock()
	if active {
		p.maybeFinalizeEarly()
	}

	_, _ = os.Stdout.Write(data)

	if p.scrollbackFD != nil {
		if wn, err := p.scrollbackFD.Write(data); err == nil {
			p.scrollbackBytes += int64(wn)
		} else {
			p.debug("SCROLLBACK-write-fail", err.Error())
		}
	}

	func() {
		defer func() {
			if r := recover(); r != nil {
				p.debug("AGENT-OUT-fail", fmt.Sprintf("%v", r))
			}
		}()
		p.updateAgentOutput(data)
	}()

	func() {
		defer func() {
			if r := recover(); r != nil {
				p.debug("DETECT-fail", fmt.Sprintf("%v", r))
			}
		}()
		*rolling = append(*rolling, data...)
		if len(*rolling) > rollingTailLen {
			*rolling = (*rolling)[len(*rolling)-rollingTailLen:]
		}
		p.checkOverlayOpen(data, *rolling)
		matches := oscRe.FindAllSubmatchIndex(*rolling, -1)
		if len(matches) > 0 {
			last := matches[len(matches)-1]
			actioned := false
			for _, m := range matches {
				ps := (*rolling)[m[2]:m[3]]
				body := (*rolling)[m[4]:m[5]]
				if isActionableOSC(ps, body) {
					if p.notifyModeActive == "native" {
						p.debug("OSC"+string(ps), string(truncate(body, 80)))
						if !actioned {
							p.emitOuter("")
							actioned = true
						}
					} else {
						p.debug("OSC"+string(ps)+"-swallow", string(truncate(body, 80)))
					}
				} else {
					p.debug("OSC"+string(ps)+"-skip", string(truncate(body, 80)))
				}
			}
			*rolling = (*rolling)[last[1]:]
			return
		}
		if idx := indexByte(data, 0x07); idx >= 0 {
			start := idx - 16
			if start < 0 {
				start = 0
			}
			end := idx + 16
			if end > len(data) {
				end = len(data)
			}
			snippet := string(data[start:end])
			if p.bellFallback && p.notifyModeActive == "native" {
				p.debug("BEL", snippet)
				p.emitOuter("")
			} else {
				p.debug("BEL-skip", snippet)
			}
		}
	}()
}

func isEIO(err error) bool {
	var pathErr *os.PathError
	if errors.As(err, &pathErr) && errors.Is(pathErr.Err, syscall.EIO) {
		return true
	}
	return errors.Is(err, syscall.EIO)
}

// ----- Small helpers ----------------------------------------------------------

func envDuration(key string, dflt time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return time.Duration(f * float64(time.Second))
		}
	}
	return dflt
}

func envFlag(key string) bool {
	v := os.Getenv(key)
	return v != "" && v != "0"
}

func isTTY(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}

func truncate(b []byte, n int) []byte {
	if len(b) > n {
		return b[:n]
	}
	return b
}

func indexByte(b []byte, c byte) int {
	for i, x := range b {
		if x == c {
			return i
		}
	}
	return -1
}
