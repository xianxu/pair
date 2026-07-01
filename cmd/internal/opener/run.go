package opener

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// Options are a viewer launcher's inputs after CLI/env resolution.
type Options struct {
	Tag       string
	Agent     string
	DataDir   string
	PairHome  string // nvim lua root + the pair binary the detached distiller execs
	SessionID string // PAIR_SESSION_ID (changelog per-session keying)
	Jump      string // --jump prev|next (scrollback)
}

// Runtime is the IO/process boundary for the viewer launchers. Pure decisions
// live in opener.go; everything here that touches zellij/nvim/exec/detach/fs sits
// behind this seam so the orchestration is unit-testable with a fake.
type Runtime interface {
	Sleep(time.Duration)
	Getpid() string
	ProcessAlive(pid string) bool

	ReadFile(path string) (string, error)
	WriteFile(path, data string) error
	WriteAtomic(path, data string) error // temp + rename (for the .viewport a live viewer may re-read)
	Remove(path string)
	FileSize(path string) (int64, bool) // for `[ -s FILE ]` guards
	Touch(path string) error            // `[ -f LOG ] || : > LOG`
	Executable(path string) bool        // `[ -x FILE ]`

	RenderScrollback(raw, events, ansi string) error // in-process scrollbackcmd.Run (sync)
	AgentPaneID() string                             // zellij list-panes → agent pane id, "" if none
	DumpScreen(paneID string) (string, error)        // zellij dump-screen
	// StartDetached launches `sh -c script` in its own session (setsid) with
	// extraEnv, stderr → statusPath, detached from this process; returns its pid.
	StartDetached(script string, extraEnv []string, statusPath string) (string, error)
	// RunViewer execs nvim (-u luaPath file) with extraEnv as a HELD child,
	// returning when the user quits.
	RunViewer(luaPath, file string, extraEnv []string) error
}

func missingEnv(opts Options) bool {
	return opts.DataDir == "" || opts.Tag == "" || opts.Agent == ""
}

// RunScrollback renders the agent pane's captured scrollback to ANSI, overlays
// the user's scroll position, and opens the read-only nvim viewer (Alt+/).
func RunScrollback(opts Options, rt Runtime, stderr io.Writer) int {
	if missingEnv(opts) {
		fmt.Fprintf(stderr, "pair-scrollback-open: missing PAIR_DATA_DIR / PAIR_TAG / PAIR_AGENT\n")
		fmt.Fprintf(stderr, "  This is meant to run inside a pair session.\n")
		rt.Sleep(3 * time.Second)
		return 1
	}
	sb := opts.DataDir + "/scrollback-" + opts.Tag + "-" + opts.Agent
	lock := sb + ".openlock"

	// Re-entrancy: a second Alt+/ while a viewer is up exits (focus falls back).
	if raw, err := rt.ReadFile(lock); err == nil {
		if other := strings.TrimSpace(raw); rt.ProcessAlive(other) {
			return 0
		}
	}

	raw, events, ansi := sb+".raw", sb+".events.jsonl", sb+".ansi"
	if sz, ok := rt.FileSize(raw); !ok || sz == 0 {
		fmt.Fprintf(stderr, "pair-scrollback-open: no scrollback yet for %s/%s\n", opts.Tag, opts.Agent)
		fmt.Fprintf(stderr, "  (capture starts when the agent pane begins emitting output.)\n")
		rt.Sleep(3 * time.Second)
		return 0
	}
	if err := rt.RenderScrollback(raw, events, ansi); err != nil {
		fmt.Fprintf(stderr, "pair-scrollback-open: scrollback-render failed: %v\n", err)
		rt.Sleep(5 * time.Second)
		return 1
	}

	overlayViewport(opts, rt, sb, ansi)

	env := []string{
		"PAIR_NVIM_PID_FILE=" + opts.DataDir + "/nvim-pid-" + opts.Tag + "-scrollback",
		"PAIR_SCROLLBACK_JUMP=" + opts.Jump,
	}
	_ = rt.WriteFile(lock, rt.Getpid()+"\n")
	defer rt.Remove(lock)
	_ = rt.RunViewer(opts.PairHome+"/nvim/scrollback.lua", ansi, env)
	return 0
}

// overlayViewport matches zellij's actual visible content onto the rendered
// .ansi to record the line the user is looking at. Best-effort: any seam failure
// leaves the renderer's own .viewport in place.
func overlayViewport(opts Options, rt Runtime, sb, ansi string) {
	paneID := rt.AgentPaneID()
	if paneID == "" {
		return
	}
	dump, err := rt.DumpScreen(paneID)
	if err != nil || dump == "" {
		return
	}
	content, err := rt.ReadFile(ansi)
	if err != nil {
		return
	}
	var ansiLines []string
	for _, l := range strings.Split(content, "\n") {
		ansiLines = append(ansiLines, stripSGR(l))
	}
	if line, ok := matchViewport(strings.Split(dump, "\n"), ansiLines); ok {
		// Atomic (temp + rename), like the shell's `> .tmp && mv -f`: a live
		// viewer's `G` refresh may re-read .viewport concurrently.
		_ = rt.WriteAtomic(sb+".viewport", strconv.Itoa(line)+"\n")
	}
}

// RunChangelog resolves the per-session change log, launches a DETACHED
// render+distill build (survives the viewer closing), and opens the nvim watcher
// (Alt+l).
func RunChangelog(opts Options, rt Runtime, stderr io.Writer) int {
	if missingEnv(opts) {
		fmt.Fprintf(stderr, "pair-changelog-open: missing PAIR_DATA_DIR / PAIR_TAG / PAIR_AGENT\n")
		fmt.Fprintf(stderr, "  This is meant to run inside a pair session.\n")
		rt.Sleep(3 * time.Second)
		return 1
	}

	sid := opts.SessionID
	if sid == "" {
		if cfg, err := rt.ReadFile(opts.DataDir + "/config-" + opts.Tag + "-" + opts.Agent + ".json"); err == nil {
			sid = resolveSessionID("", []byte(cfg))
		}
	}
	base := changelogBase(opts.DataDir, opts.Tag, opts.Agent, sid)
	sb := opts.DataDir + "/scrollback-" + opts.Tag + "-" + opts.Agent
	raw, events := sb+".raw", sb+".events.jsonl"
	log, anchor, cleaned := base+".md", base+".anchor", base+".cleaned"
	openlock, dlock, status := base+".openlock", base+".distill.lock", base+".status"

	// Viewer re-entrancy.
	if r, err := rt.ReadFile(openlock); err == nil {
		if other := strings.TrimSpace(r); rt.ProcessAlive(other) {
			return 0
		}
	}
	_ = rt.WriteFile(openlock, rt.Getpid()+"\n")
	defer rt.Remove(openlock)

	_ = rt.Touch(log)

	// Launch the detached distiller unless one is already running, RAW has
	// content, and the pair binary is built.
	distillerRunning := false
	if r, err := rt.ReadFile(dlock); err == nil {
		if p := strings.TrimSpace(r); rt.ProcessAlive(p) {
			distillerRunning = true
		}
	}
	bin := opts.PairHome + "/bin/pair"
	if sz, ok := rt.FileSize(raw); !distillerRunning && ok && sz > 0 && rt.Executable(bin) {
		_ = rt.WriteFile(status, "")
		env := distillerEnv(bin, raw, events, cleaned, log, anchor, opts.Agent)
		if pid, err := rt.StartDetached(distillerInner, env, status); err == nil {
			_ = rt.WriteFile(dlock, pid+"\n")
		}
	}

	env := []string{
		"PAIR_CHANGELOG_LOG=" + log,
		"PAIR_CHANGELOG_DLOCK=" + dlock,
		"PAIR_CHANGELOG_STATUS=" + status,
	}
	_ = rt.RunViewer(opts.PairHome+"/nvim/changelog.lua", log, env)
	return 0
}
