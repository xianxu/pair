package launcher

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/continuationcmd"
	"github.com/xianxu/pair/cmd/internal/osfs"
)

// OSRuntime is the concrete create-flow Runtime: real zellij/fzf/cmux/tty/exec
// calls (#99 M2). It embeds osfs.FS for the filesystem primitives (FSOps) and
// carries DataDir/PairHome so the seams that need them don't depend on env-set
// ordering. Read-only zellij queries go through a 5s-timeout wrapper (zj) — the
// shell's watchdog against a wedged daemon socket; the blocking launch is
// deliberately un-timed.
type OSRuntime struct {
	osfs.FS
	DataDir  string
	PairHome string
}

// NewOSRuntime builds the concrete runtime for a resolved data dir + asset root.
func NewOSRuntime(dataDir, pairHome string) *OSRuntime {
	return &OSRuntime{DataDir: dataDir, PairHome: pairHome}
}

const zjTimeout = 5 * time.Second

// zj runs a read-only zellij query under a hard timeout, returning combined
// stdout (stderr discarded) or "" on any error/timeout — the callers treat an
// empty result the way the shell's `|| true` pipelines do.
func zj(args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), zjTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "zellij", args...).Output()
	if err != nil {
		return ""
	}
	return string(out)
}

// --- ZellijOps -------------------------------------------------------------

func (OSRuntime) Sessions() ([]Session, error) { return ZellijSource{}.Snapshot() }

func (OSRuntime) SessionBlocksReuse(session string) bool {
	present, exited := sessionRowState(zj("list-sessions", "--no-formatting"), session)
	if !present {
		return false // no such session — reuse is free.
	}
	if exited {
		// Stale resurrect residue (#67): delete it and report reusable. Routed
		// through zj so a wedged daemon socket can't hang the delete (shell zj).
		zj("delete-session", session, "--force")
		return false
	}
	return true // running/detached — still occupied.
}

func (OSRuntime) ProbeSessionName(session string) error {
	ctx, cancel := context.WithTimeout(context.Background(), zjTimeout)
	defer cancel()
	out, _ := exec.CommandContext(ctx, "zellij", "--session", session, "action", "list-clients").CombinedOutput()
	if sessionNameRejected(string(out)) {
		return fmt.Errorf("session name too long: %s", session)
	}
	return nil
}

// LaunchSession is the BLOCKING fork+wait handoff (NOT syscall.Exec): the tty is
// passed straight through and the Go launcher regains control on pane exit.
func (OSRuntime) LaunchSession(session, configDir, layout string) (int, error) {
	return runBlockingHandoff(exec.Command("zellij",
		"--config-dir", configDir,
		"--new-session-with-layout", layout,
		"--session", session))
}

// runBlockingHandoff runs cmd with the tty passed straight through (NOT
// syscall.Exec) and maps its result to an exit code — the shared contract of the
// create + attach zellij handoffs (#99 M3), so the Go launcher regains control
// afterward for cleanup + the restart loop.
func runBlockingHandoff(cmd *exec.Cmd) (int, error) {
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Run()
	if err == nil {
		return 0, nil
	}
	if exit, ok := err.(*exec.ExitError); ok {
		return exit.ExitCode(), nil
	}
	return 1, err
}

// --- SnapshotOps -----------------------------------------------------------

func (r OSRuntime) ScanHistory(base string, cutoff time.Time) ([]HistoricalTag, error) {
	return HistorySource{DataDir: r.DataDir}.Scan(base, cutoff)
}

// --- ListOps ---------------------------------------------------------------

// ListSessions gathers the `pair list`/`ls` rows: each pair-<tag> session's
// reuse state (EXITED vs live), its live client count, and the agent it was last
// paired with (shell 228-306). Agent resolution reuses InferAgent (agent-<tag>
// then the config-filename agent) — broader than the shell's agent-<tag>-only
// read on that axis (a cleared agent record still resolves from the saved config
// rather than "?"), but it does NOT replicate the shell's pgrep backfill of
// agent-<tag> from a live pair-wrap's env for pre-agent-tracking sessions (shell
// 246-272), so such a legacy session shows "?" where the shell resolved it. That
// backfill is a legacy-only path to reconcile when the shell retires (M5c).
func (r OSRuntime) ListSessions() ([]ListRow, error) {
	if _, err := exec.LookPath("zellij"); err != nil {
		return nil, fmt.Errorf("zellij not found on PATH") // shell 231-234
	}
	raw := zj("list-sessions", "--no-formatting")
	names := pairSessionNames(zj("list-sessions", "--short"))
	rows := make([]ListRow, 0, len(names))
	for _, name := range names {
		tag := strings.TrimPrefix(name, "pair-")
		row := ListRow{Session: name, Agent: r.InferAgent(tag), State: SessionDetached}
		if _, exited := sessionRowState(raw, name); exited {
			row.State = SessionExited
		} else if row.Clients = parseClientCount(zj("--session", name, "action", "list-clients")); row.Clients > 0 {
			row.State = SessionAttached
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// --- ContinuationOps -------------------------------------------------------

// continuationDirPath is <git-root-or-cwd>/workshop/continuation (shell 613-614);
// the dir constant is shared with continuationcmd (ARCH-DRY, #99 M5b).
func continuationDirPath() string {
	root := ""
	if out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output(); err == nil {
		root = strings.TrimSpace(string(out))
	}
	if root == "" {
		root, _ = os.Getwd()
	}
	return filepath.Join(root, continuationcmd.ContinuationDir)
}

// continuationSlug strips the timestamp prefix from a continuation basename
// (`20260101T000000-demo.md` → `demo`; shell 619's `${_cs#*-}`).
func continuationSlug(base string) string {
	s := strings.TrimSuffix(base, ".md")
	if i := strings.IndexByte(s, '-'); i >= 0 {
		return s[i+1:]
	}
	return s
}

func (r OSRuntime) ResolveContinuationDoc(slug string) (string, string, bool) {
	matches, _ := filepath.Glob(filepath.Join(continuationDirPath(), "*-"+slug+".md"))
	if len(matches) == 0 {
		return "", "", false
	}
	sort.Strings(matches)
	path := matches[len(matches)-1] // newest by timestamp-prefixed name (shell `sort | tail -1`)
	agent := ""
	if raw, err := r.ReadFile(path); err == nil {
		agent = frontmatterField(raw, "agent")
	}
	return path, agent, true
}

func (r OSRuntime) ScanContinuations() ([]ContinuationRow, string) {
	dir := continuationDirPath()
	matches, _ := filepath.Glob(filepath.Join(dir, "*.md"))
	sort.Strings(matches)
	rows := make([]ContinuationRow, 0, len(matches))
	for _, path := range matches {
		raw, err := r.ReadFile(path)
		if err != nil {
			continue
		}
		rows = append(rows, ContinuationRow{
			Slug:    continuationSlug(filepath.Base(path)),
			Issues:  frontmatterField(raw, "issues"),
			Preview: continuationcmd.NextActionPreview(raw),
		})
	}
	return rows, dir
}

// --- UIOps -----------------------------------------------------------------

func (OSRuntime) ShowFamilyExisting(familyPrefix string) {
	rows := familyRows(zj("list-sessions", "--no-formatting"), familyPrefix)
	if len(rows) == 0 {
		return
	}
	tty, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	if err != nil {
		return
	}
	defer tty.Close()
	fmt.Fprintf(tty, "Existing %s* sessions:\n", familyPrefix)
	for _, row := range rows {
		fmt.Fprintf(tty, "  %s\n", row)
	}
}

// PromptSessionName delegates the editable prefill to zsh's vared (bash 3.2 on
// macOS lacks `read -i`); with no zsh it falls back to a plain [default] prompt.
// The vared UI runs on /dev/tty; only the final value reaches stdout.
func (OSRuntime) PromptSessionName(def string) (string, bool) {
	if _, err := exec.LookPath("zsh"); err == nil {
		script := `bindkey -e; v=$1; vared -p "Session name: " v; print -r -- "$v"`
		cmd := exec.Command("zsh", "-c", script, "_", def)
		if tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0); err == nil {
			defer tty.Close()
			cmd.Stdin, cmd.Stderr = tty, tty
		}
		out, err := cmd.Output()
		if err != nil {
			return "", false // vared aborted (ESC / EOF).
		}
		return strings.TrimRight(string(out), "\n"), true
	}
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return "", false
	}
	defer tty.Close()
	fmt.Fprintf(tty, "Session name [%s]: ", def)
	line, err := bufio.NewReader(tty).ReadString('\n')
	line = strings.TrimRight(line, "\n")
	if err != nil && line == "" {
		return "", false // EOF with no input — abort (shell's `read` non-zero).
	}
	return line, true // an empty line falls through to the default in RunLaunch.
}

// PickFromList presents options (NUL-separated, fzf --read0 multi-line render)
// and returns the chosen line, or "" on abort. --ansi lets callers pass
// color-coded rows (the #99 M5a session picker greys history + ambers the queued
// badge); fzf strips the SGR codes from the returned line, so callers key their
// selection map by the plain text. Rows without codes (the config picker) are
// unaffected.
func (OSRuntime) PickFromList(header string, options []string, height int) string {
	cmd := exec.Command("fzf", "--read0", "--ansi",
		"--header", header,
		"--height", fmt.Sprintf("%d", height),
		"--no-info", "--layout", "reverse", "--no-multi")
	cmd.Stdin = strings.NewReader(strings.Join(options, "\x00"))
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.Trim(string(out), "\x00\n")
}

func (OSRuntime) SetTerminalTitle(session string) {
	fmt.Fprintf(os.Stdout, "\033]0;%s\007", session)
}

// --- ProcOps ---------------------------------------------------------------

func (r OSRuntime) SpawnSessionWatcher(agent, tag, cwd string, agentArgs []string) {
	args := append([]string{filepath.Join(r.PairHome, "bin", "pair-session-watch.sh"), agent, tag, cwd}, agentArgs...)
	spawnDetached(args, nil)
}

func (r OSRuntime) SpawnTitlePoller(tag, agent string) {
	spawnDetached([]string{filepath.Join(r.PairHome, "bin", "pair-title.sh"), tag, agent}, nil)
}

func (OSRuntime) DevRebuild(pairHome string) {
	if os.Getenv("PAIR_DEV") == "" {
		return
	}
	script := fmt.Sprintf(`. "%s/bin/lib/dev-rebuild.sh"; dev_rebuild`, pairHome)
	cmd := exec.Command("sh", "-c", script)
	cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr // rebuild chatter to stderr
	cmd.Run()
}

// spawnDetached backgrounds argv[0] in its own session (setsid) with stdio to
// /dev/null — the Go analogue of the shell's `… </dev/null >/dev/null 2>&1 &`
// + disown, so the child survives the launcher and never leaks a job line.
func spawnDetached(argv []string, extraEnv []string) {
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		return
	}
	defer devNull.Close()
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Env = append(os.Environ(), extraEnv...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = devNull, devNull, devNull
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return
	}
	go func() { _ = cmd.Wait() }() // reap our bookkeeping; the child is reparented.
}

// --- EnvOps ----------------------------------------------------------------

func (OSRuntime) SetEnv(key, value string) { _ = os.Setenv(key, value) }

// InZellijPane walks the PPID chain looking for a zellij ancestor — the ground
// truth (env vars leak across cmux panes, so $ZELLIJ can't be trusted).
func (OSRuntime) InZellijPane() bool {
	pid := os.Getpid()
	for pid > 1 {
		if base := lastPathElem(procComm(pid)); base == "zellij" {
			return true
		}
		parent := procPPID(pid)
		if parent <= 1 || parent == pid {
			return false
		}
		pid = parent
	}
	return false
}

func (OSRuntime) CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func (r OSRuntime) RecordOuterTTY(tag string) {
	path := filepath.Join(r.DataDir, "outer-tty-"+tag)
	cmd := exec.Command("tty")
	cmd.Stdin = os.Stdin
	out, _ := cmd.Output()
	outer := strings.TrimSpace(string(out))
	if strings.HasPrefix(outer, "/dev/") {
		_ = r.WriteAtomic(path, outer+"\n")
	} else {
		r.Remove(path)
	}
}

// CmuxRename claims this workspace for tag (presence beats a stale owner file)
// and renames it with the personal emoji substitution; a no-op outside cmux.
func (r OSRuntime) CmuxRename(tag, title string) {
	wsID := os.Getenv("CMUX_WORKSPACE_ID")
	if wsID == "" {
		return
	}
	if _, err := exec.LookPath("cmux"); err != nil {
		return
	}
	_ = r.WriteAtomic(filepath.Join(r.DataDir, "cmux-owner-"+wsID), tag+"\n")
	exec.Command("cmux", "rename-workspace", EmojiTitle(title)).Run()
}

// --- IDOps -----------------------------------------------------------------

// MintUUID returns a fresh lowercase RFC 4122 v4 uuid (the shell's `uuidgen |
// tr` without the subprocess).
func (OSRuntime) MintUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return ""
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// InferAgent reads the agent-<tag> record (primary) or the agent encoded in a
// config-<tag>-<agent>.json filename (fallback for Alt+x'd sessions).
func (r OSRuntime) InferAgent(tag string) string {
	if raw, err := r.ReadFile(filepath.Join(r.DataDir, "agent-"+tag)); err == nil {
		if a := strings.TrimSpace(raw); a != "" {
			return a
		}
	}
	matches, _ := filepath.Glob(filepath.Join(r.DataDir, "config-"+tag+"-*.json"))
	for _, m := range matches {
		if raw, err := r.ReadFile(m); err == nil {
			if cfg, err := parseConfig(raw); err == nil && cfg.Agent != "" {
				return cfg.Agent
			}
		}
	}
	return ""
}

func (OSRuntime) AgentSessionExists(agent, sid, cwd string) bool {
	if sid == "" {
		return false
	}
	home := os.Getenv("HOME")
	switch agent {
	case "claude":
		return fileExists(ClaudeTranscriptPath(home, cwd, sid))
	case "agy":
		return fileExists(AgyConversationPath(home, sid))
	case "codex":
		matches, _ := filepath.Glob(filepath.Join(CodexSessionsDir(home), "*"+sid+"*"))
		for _, m := range matches {
			if fileExists(m) {
				return true
			}
		}
		return false
	}
	return false
}

// --- LifecycleOps ----------------------------------------------------------

// AttachSession is the BLOCKING attach twin of LaunchSession.
func (OSRuntime) AttachSession(session, configDir string) (int, error) {
	return runBlockingHandoff(exec.Command("zellij", "--config-dir", configDir, "attach", session))
}

// pairCacheDir is where pair-restart.sh drops the {quit,restart}-<session> markers.
func pairCacheDir() string { return filepath.Join(os.Getenv("HOME"), ".cache", "pair") }

func (r OSRuntime) TakeQuitMarker(session string) bool {
	path := filepath.Join(pairCacheDir(), "quit-"+session)
	if !fileExists(path) {
		return false
	}
	r.Remove(path)
	return true
}

func (OSRuntime) RestartMarkerPresent(session string) bool {
	return fileExists(filepath.Join(pairCacheDir(), "restart-"+session))
}

func (r OSRuntime) TakeRestartMarker(session string) (RestartMarker, bool) {
	path := filepath.Join(pairCacheDir(), "restart-"+session)
	raw, err := r.ReadFile(path)
	if err != nil {
		return RestartMarker{}, false
	}
	r.Remove(path)
	return parseRestartMarker(raw), true
}

// WriteRestartMarker + TouchQuitMarker are the in-session compaction write twins
// (#99 M5b, shell 1052-1058); WriteAtomic/Touch MkdirAll the cache dir.
func (r OSRuntime) WriteRestartMarker(session string, m RestartMarker) {
	_ = r.WriteAtomic(filepath.Join(pairCacheDir(), "restart-"+session), serializeRestartMarker(m))
}

func (r OSRuntime) TouchQuitMarker(session string) {
	_ = r.Touch(filepath.Join(pairCacheDir(), "quit-"+session))
}

// ExecKillSession execs `${PAIR_KILL_CMD:-zellij kill-session} <session>` and
// does NOT return on success (syscall.Exec replaces the process — the compaction
// pane dies, the outer bin/pair regains the tty). A missing binary falls through
// so the caller isn't wedged (shell 1060).
func (OSRuntime) ExecKillSession(session string) {
	argv := []string{"zellij", "kill-session", session}
	if kc := os.Getenv("PAIR_KILL_CMD"); kc != "" {
		argv = append(strings.Fields(kc), session)
	}
	if path, err := exec.LookPath(argv[0]); err == nil {
		_ = syscall.Exec(path, argv, os.Environ())
	}
}

// DeleteSession removes the zellij session record, then SIGKILLs a lingering
// `zellij --server …/<session>` that re-registered the record on a heartbeat.
func (OSRuntime) DeleteSession(session string) {
	zj("delete-session", session, "--force")
	pkillF("zellij --server .*/" + session + "$")
}

// pkillF runs `pkill -9 -f <pattern>` (best-effort; macOS pkill -f is BRE).
func pkillF(pattern string) { _ = exec.Command("pkill", "-9", "-f", pattern).Run() }

// ReapNvim kills this tag's nvim --embed children: the deterministic pidfiles
// first, then a scoped pkill for the missing-pidfile case (shell 1089-1112).
func (r OSRuntime) ReapNvim(tag string) {
	for _, kind := range []string{"draft", "scrollback"} {
		pf := filepath.Join(r.DataDir, "nvim-pid-"+tag+"-"+kind)
		if raw, err := r.ReadFile(pf); err == nil {
			if pid := strings.TrimSpace(raw); pid != "" {
				_ = exec.Command("kill", "-9", pid).Run()
			}
			r.Remove(pf)
		}
	}
	pkillF("nvim --embed.*" + r.DataDir + "/draft-" + tag + `\.md$`)
	pkillF("nvim --embed.*" + r.DataDir + "/scrollback-" + tag + "-")
}

// SweepOrphanNvim reaps nvim --embed processes whose pair-<tag> is not live —
// candidates come from the nvim-pid-* sidecars and a full `ps` argv scan (catches
// embeds with no pidfile), shell 1117-1158.
func (r OSRuntime) SweepOrphanNvim(liveTags []string) {
	live := make(map[string]bool, len(liveTags))
	for _, t := range liveTags {
		live[t] = true
	}
	cands := map[string]bool{}
	for _, kind := range []string{"draft", "scrollback"} {
		matches, _ := filepath.Glob(filepath.Join(r.DataDir, "nvim-pid-*-"+kind))
		for _, m := range matches {
			tag := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(m), "nvim-pid-"), "-"+kind)
			if tag != "" {
				cands[tag] = true
			}
		}
	}
	if out, err := exec.Command("ps", "-ww", "-A", "-o", "args=").Output(); err == nil {
		for _, argv := range strings.Split(string(out), "\n") {
			if !strings.Contains(argv, "nvim") || !strings.Contains(argv, "--embed") {
				continue
			}
			if tag := tagFromEmbedArgv(argv, r.DataDir); tag != "" {
				cands[tag] = true
			}
		}
	}
	for tag := range cands {
		if !live[tag] {
			r.ReapNvim(tag)
		}
	}
}

// ParkScrollback preserves the .raw/.events.jsonl scrollback to a timestamped
// parked-scrollback-<tag>-<ts> base (move on quit, copy on compaction) and
// touches parked-<tag> (shell 696-708). The timestamp is taken here (time.Now is
// live in OSRuntime, unlike a pure decider).
func (r OSRuntime) ParkScrollback(tag, agent string, move bool) (string, bool) {
	base := filepath.Join(r.DataDir, "scrollback-"+tag+"-"+agent)
	if size, ok := r.FileSize(base + ".raw"); !ok || size == 0 {
		return "", false
	}
	pbase := filepath.Join(r.DataDir, "parked-scrollback-"+tag+"-"+time.Now().Format("20060102T150405"))
	if !transferFile(base+".raw", pbase+".raw", move) {
		return "", false
	}
	if _, ok := r.FileSize(base + ".events.jsonl"); ok {
		transferFile(base+".events.jsonl", pbase+".events.jsonl", move)
	}
	_ = r.Touch(filepath.Join(r.DataDir, "parked-"+tag))
	return pbase, true
}

// transferFile moves (rename, with a cross-device copy+remove fallback) or copies
// src to dst; the Go analogue of the shell's `mv`/`cp`.
func transferFile(src, dst string, move bool) bool {
	if move {
		if os.Rename(src, dst) == nil {
			return true
		}
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return false
	}
	if os.WriteFile(dst, data, 0o644) != nil {
		return false
	}
	if move {
		_ = os.Remove(src)
	}
	return true
}

// ConfirmParkNudge shows the [y/N] preserve prompt on /dev/tty, bounded by
// timeoutSecs (a non-positive bound or a timeout/EOF auto-declines), shell
// 1562-1581.
func (OSRuntime) ConfirmParkNudge(session string, timeoutSecs int) bool {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return false
	}
	defer tty.Close()
	fmt.Fprintf(tty, "pair: preserve %q scrollback to distill into a continuation later? [y/N] (%ds → N): ", session, timeoutSecs)
	if timeoutSecs <= 0 {
		fmt.Fprintln(tty)
		return false
	}
	// The reader goroutine may outlive a timeout (a /dev/tty read isn't
	// interruptible by Close), but the buffered channel keeps its send from
	// blocking, and this seam only runs at quit — the process exits (or loops to
	// the next handoff) right after, so the stray reader can't steal later input.
	ansCh := make(chan string, 1)
	go func() {
		line, _ := bufio.NewReader(tty).ReadString('\n')
		ansCh <- strings.TrimSpace(line)
	}()
	select {
	case ans := <-ansCh:
		return ans == "y" || ans == "Y"
	case <-time.After(time.Duration(timeoutSecs) * time.Second):
		fmt.Fprintln(tty) // close the un-terminated prompt line.
		return false
	}
}

func (OSRuntime) IsTTY() bool {
	fi, err := os.Stdin.Stat()
	return err == nil && (fi.Mode()&os.ModeCharDevice) != 0
}

// KillTitlePoller SIGTERMs (not -9: it self-cleans) + clears the poller pidfile.
func (r OSRuntime) KillTitlePoller(tag string) {
	pf := filepath.Join(r.DataDir, "title-pid-"+tag)
	if raw, err := r.ReadFile(pf); err == nil {
		if pid := strings.TrimSpace(raw); pid != "" {
			_ = exec.Command("kill", pid).Run()
		}
		r.Remove(pf)
	}
}

func (r OSRuntime) PairOwnsCmuxWorkspace(tag string) bool {
	wsID := os.Getenv("CMUX_WORKSPACE_ID")
	if wsID == "" {
		return false
	}
	raw, err := r.ReadFile(filepath.Join(r.DataDir, "cmux-owner-"+wsID))
	return err == nil && strings.TrimSpace(raw) == tag
}

func (r OSRuntime) ClearCmuxOwner() {
	wsID := os.Getenv("CMUX_WORKSPACE_ID")
	if wsID == "" {
		return
	}
	r.Remove(filepath.Join(r.DataDir, "cmux-owner-"+wsID))
}

// --- helpers ---------------------------------------------------------------

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func lastPathElem(s string) string {
	if i := strings.LastIndexByte(s, '/'); i >= 0 {
		return s[i+1:]
	}
	return s
}

func procComm(pid int) string {
	out, err := exec.Command("ps", "-o", "comm=", "-p", fmt.Sprintf("%d", pid)).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func procPPID(pid int) int {
	out, err := exec.Command("ps", "-o", "ppid=", "-p", fmt.Sprintf("%d", pid)).Output()
	if err != nil {
		return 0
	}
	var ppid int
	if _, err := fmt.Sscanf(strings.TrimSpace(string(out)), "%d", &ppid); err != nil {
		return 0
	}
	return ppid
}
