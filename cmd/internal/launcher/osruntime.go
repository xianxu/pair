package launcher

import (
	"bufio"
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

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
	cmd := exec.Command("zellij",
		"--config-dir", configDir,
		"--new-session-with-layout", layout,
		"--session", session)
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
// and returns the chosen line, or "" on abort.
func (OSRuntime) PickFromList(header string, options []string, height int) string {
	cmd := exec.Command("fzf", "--read0",
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
