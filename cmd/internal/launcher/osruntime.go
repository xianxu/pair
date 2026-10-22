package launcher

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/storagegc"
	"golang.org/x/sys/unix"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/continuationcmd"
	"github.com/xianxu/pair/cmd/internal/osfs"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
	"github.com/xianxu/pair/cmd/internal/sessionwatch"
	"github.com/xianxu/pair/cmd/internal/zellijquery"
	"github.com/xianxu/pair/cmd/internal/zellijreply"
)

// OSRuntime is the concrete create-flow Runtime: real zellij/fzf/cmux/tty/exec
// calls (#99 M2). It embeds osfs.FS for the filesystem primitives (FSOps) and
// carries DataDir/PairHome so the seams that need them don't depend on env-set
// ordering. Read-only zellij queries go through a 5s-timeout wrapper (zj) — the
// shell's watchdog against a wedged daemon socket; the blocking launch is
// deliberately un-timed.
type OSRuntime struct {
	osfs.FS
	DataDir       string
	GlobalDataDir string
	PairHome      string
	Processes     procutil.Source
	// nameQueryRun owns the name probe process; nil uses the OS collector.
	nameQueryRun func(*exec.Cmd) error

	sessionQuiescence  sessionQuiescenceOps
	sessionQuiesceWait time.Duration
	sessionQuiescePoll time.Duration
}

// NewOSRuntime builds the concrete runtime for a resolved data dir + asset root.
func NewOSRuntime(dataDir, pairHome string) *OSRuntime {
	return NewScopedOSRuntime(dataDir, dataDir, pairHome)
}

func NewScopedOSRuntime(globalDataDir, dataDir, pairHome string) *OSRuntime {
	return &OSRuntime{DataDir: dataDir, GlobalDataDir: globalDataDir, PairHome: pairHome}
}

// Touch ensures an ordinary draft/marker exists without opening a FIFO or
// following a substituted symlink. It preserves existing regular-file bytes.
func (r OSRuntime) Touch(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	fd, err := unix.Open(path, unix.O_WRONLY|unix.O_CREAT|unix.O_NONBLOCK|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0644)
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(fd), path)
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return &osfs.InvalidMetadataError{Path: path, Kind: osfs.MetadataNotRegular}
	}
	return file.Close()
}

func (r OSRuntime) StartProofMigration() {
	go func() { _ = sessionwatch.MigrateProoflessBindings(os.Getenv("HOME"), r.DataDir) }()
}

func (r OSRuntime) EnsureThreadAddress(scope RepoScope, tag string, couchOwned bool) error {
	return EnsureThreadAddressForPair(r.GlobalDataDir, scope, tag, couchOwned)
}

func (r OSRuntime) RegisterExistingCouchThread(scope RepoScope, tag string) error {
	return RegisterExistingCouchThread(r.GlobalDataDir, scope, tag)
}

const zjTimeout = 5 * time.Second

// zj runs a read-only zellij query under a hard timeout, returning combined
// stdout (stderr discarded) or "" on any error/timeout — the callers treat an
// empty result the way the shell's `|| true` pipelines do.
func zj(args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), zjTimeout)
	defer cancel()
	out, err := (zellijquery.Source{}).Output(ctx, args...)
	if err != nil {
		return ""
	}
	return string(out)
}

// --- ZellijOps -------------------------------------------------------------

func (OSRuntime) Sessions() ([]Session, error) { return ZellijSource{}.Snapshot() }

func (OSRuntime) SessionLiveness() ([]Session, error) {
	return (OSRuntime{}).SessionLivenessContext(context.Background())
}

func (OSRuntime) SessionLivenessContext(ctx context.Context) ([]Session, error) {
	return ZellijSource{}.LivenessContext(ctx)
}

func (r OSRuntime) SessionBlocksReuse(session string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), zjTimeout)
	defer cancel()
	raw, err := (ZellijSource{}).runContext(ctx, "list-sessions", "--no-formatting")
	if err != nil {
		return true
	}
	present, _, err := sessionListingState(raw, session)
	if err != nil {
		return true
	}
	// A saved layout remains evidence. Reuse never deletes it implicitly;
	// legacy recovery creates a new terminal name after resolving ownership.
	if present {
		return true
	}
	// Native discovery may silently omit a still-live exact server. Listing
	// absence cannot release its name without independent process evidence.
	servers, err := r.liveSessionServers(ctx, session)
	return err != nil || ctx.Err() != nil || len(servers) != 0
}

// liveSessionServers is the independent process evidence that a listing's
// omission of session is not absence: an OS process scan, no native query.
func (r OSRuntime) liveSessionServers(ctx context.Context, session string) ([]SessionServerIdentity, error) {
	ops := r.sessionQuiescence
	if ops == nil {
		ops = newOSSessionQuiescenceOps(r.Processes)
	}
	return ops.SessionServers(ctx, session)
}

// ProbeSessionName asks zellij whether it accepts session's length, at the cost
// of one list-clients. AssignSessionName skips it for a name a live session
// already runs under (acceptingLiveNames, pair#228): that session is the proof.
func (r OSRuntime) ProbeSessionName(session string) error {
	ctx, cancel := zellijreply.AdmissionContext(context.Background())
	defer cancel()
	return r.ProbeSessionNameContext(ctx, session)
}

func (r OSRuntime) ProbeSessionNameContext(ctx context.Context, session string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	result, readErr := (zellijquery.Source{Run: r.nameQueryRun}).ReadContext(ctx, "--session", session, "action", "list-clients")
	stdout, stderr := bytes.NewBuffer(result.Stdout), bytes.NewBuffer(result.Stderr)
	err := result.ProcessErr
	if ctx.Err() != nil {
		return fmt.Errorf("could not ask zellij about session name %q: %w", session, errors.Join(err, ctx.Err()))
	}
	if err != nil {
		// Diagnostic evidence may consume only a pure subprocess exit. Joined
		// cancellation/collection failures cannot teach either length bound.
		if exit, pureExit := err.(*exec.ExitError); pureExit {
			if sessionNameRejected(session, exit.ExitCode(), stdout.String(), stderr.String()) {
				return fmt.Errorf("%w: %s", errSessionNameTooLong, session)
			}
			// Clap validates --session before action dispatch. These exact
			// post-parse absence diagnostics prove length acceptance only.
			if exit.ExitCode() == 1 && stdout.Len() == 0 &&
				sessionNameAbsentDiagnostic(session, stderr.String()) {
				return nil
			}
		}
		return fmt.Errorf("could not ask zellij about session name %q: %w", session, err)
	}
	if readErr != nil {
		return fmt.Errorf("could not ask zellij about session name %q: %w", session, readErr)
	}
	// Empty success after an early unblock is unanswered. Neither failed IO
	// nor unrelated stderr may teach the assignment's monotone length bounds.
	if len(bytes.TrimSpace(stdout.Bytes())) == 0 {
		return fmt.Errorf("could not ask zellij about session name %q: %w", session, ErrZellijNoReply)
	}
	if stderr.String() == "Session '"+session+"' not found. The following sessions are active:\n" {
		if !sessionNameAbsentListing(session, stdout.String()) {
			return fmt.Errorf("could not validate zellij name probe: invalid absent-session listing")
		}
		return nil
	}
	_, parseErr := zellijreply.ParseClients(stdout.Bytes())
	if parseErr != nil {
		return fmt.Errorf("could not validate zellij name probe: %w", parseErr)
	}
	return nil
}

// LaunchSession is the BLOCKING fork+wait handoff (NOT syscall.Exec): the tty is
// passed straight through and the Go launcher regains control on pane exit.
// Cancelling ctx ends the client (#288): see killOnCancel.
func (r OSRuntime) LaunchSession(ctx context.Context, session, configDir, layout string) (int, error) {
	return r.runTerminalHandoff(ctx, session, configDir, layout, true)
}

func (r OSRuntime) LaunchSessionContext(lifetime, startup, admission context.Context, session, configDir, layout string) (int, error) {
	return r.runTerminalHandoffContext(lifetime, startup, admission, session, configDir, layout, true)
}

// clientKillGrace is how long a cancelled client gets to exit on SIGTERM
// before SIGKILL. A zellij client hung on a dead-at-birth server exits about
// 50 ms after SIGTERM (measured, #288).
const clientKillGrace = 2 * time.Second

// killOnCancel makes a CommandContext handoff end gently when its ctx does:
// SIGTERM, then SIGKILL after clientKillGrace. Run returns only once the child
// is reaped, so a cancelled launch leaves no process holding the terminal --
// under Couch, a child left behind would keep the helper's pty open and the
// helper would read as alive.
func killOnCancel(cmd *exec.Cmd) *exec.Cmd {
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	cmd.WaitDelay = clientKillGrace
	return cmd
}

func (OSRuntime) ProbeLiveLayout(session string) (LayoutMode, error) {
	ctx, cancel := zellijreply.AdmissionContext(context.Background())
	defer cancel()
	read := func(ctx context.Context) ([]byte, error) {
		return (ZellijSource{}).runContext(ctx, "--session", session, "action", "list-panes", "--json", "--command", "--state")
	}
	out, err := zellijreply.Confirm(ctx, read, projectLiveLayoutReply)
	if err != nil {
		return "", err
	}
	panes, err := zellijreply.ParsePanes(out, zellijreply.PaneRequirements{Command: true, Floating: true})
	if err != nil {
		return "", err
	}
	mode, ok := ClassifyLiveLayout(panes)
	if !ok {
		return "", fmt.Errorf("unrecognized pane signature")
	}
	return mode, nil
}

// projectLiveLayoutReply retains precisely the command/title/identity/floating
// facts used by ClassifyLiveLayout; geometry and output cannot identify layout.
func projectLiveLayoutReply(raw []byte) (string, error) {
	panes, err := zellijreply.ParsePanes(raw, zellijreply.PaneRequirements{Command: true, Floating: true})
	if err != nil {
		return "", err
	}
	if _, ok := ClassifyLiveLayout(panes); !ok {
		return "", fmt.Errorf("unrecognized pane signature")
	}
	type fact struct {
		ID, Command, Title string
		Plugin, Floating   bool
	}
	facts := make([]fact, 0, len(panes))
	for _, p := range panes {
		facts = append(facts, fact{p.ID, p.TerminalCommand, p.Title, p.IsPlugin, p.IsFloating})
	}
	sort.Slice(facts, func(i, j int) bool {
		if facts[i].Plugin != facts[j].Plugin {
			return !facts[i].Plugin
		}
		return facts[i].ID < facts[j].ID
	})
	data, err := json.Marshal(facts)
	return string(data), err
}

// runBlockingHandoff runs cmd with the tty passed straight through (NOT
// syscall.Exec) and maps its result to an exit code — the shared contract of the
// create + attach zellij handoffs (#99 M3), so the Go launcher regains control
// afterward for cleanup + the restart loop.
//
// The child's own status wins whenever it ran (-1 when a signal ended it). A
// cancelled CommandContext child that still exits 0 makes Run return ctx.Err()
// -- os/exec's contract -- but what the launcher acts on is how the client
// ended; the birth watch knows about the cancel itself (#288). Only a child
// that never ran reports the error.
func runBlockingHandoff(cmd *exec.Cmd) (int, error) {
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	err := cmd.Run()
	if cmd.ProcessState != nil {
		return cmd.ProcessState.ExitCode(), nil
	}
	return 1, err
}

// --- SnapshotOps -----------------------------------------------------------

func (r OSRuntime) ScanHistory(base string, cutoff time.Time) ([]HistoricalTag, error) {
	return HistorySource{DataDir: r.DataDir, LegacyDataDir: r.globalDataDir()}.Scan(base, cutoff)
}

// --- ListOps ---------------------------------------------------------------

// ListSessions gathers the `pair list`/`ls` rows: each Pair session's
// reuse state (EXITED vs live), its live client count, and the agent it was last
// paired with (shell 228-306). Agent resolution reuses InferAgent (current
// ledger ownership, then agent-<tag>), but it does NOT replicate the retired
// shell's pgrep backfill
// of agent-<tag> from a live pair-wrap's env for pre-agent-tracking sessions, so
// such a legacy session shows "?" for its agent. Accepted gap (ancient,
// display-only) — a follow-up could re-add the pair-wrap env scan if it recurs.
func (r OSRuntime) ListSessions() ([]ListRow, error) {
	if _, err := exec.LookPath("zellij"); err != nil {
		return nil, fmt.Errorf("zellij not found on PATH") // shell 231-234
	}
	source := ZellijSource{}
	raw, err := source.listings(context.Background())
	if err != nil {
		return nil, err
	}
	states, err := sessionListingStates(raw)
	if err != nil {
		return nil, err
	}
	var names []string
	for name := range states {
		if isPairSessionName(name) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	index, err := r.ReadSessionNameIndex()
	if err != nil {
		return nil, fmt.Errorf("read session-name index: %w", err)
	}
	// One list-clients per pair session, deliberately: each row renders
	// "attached (N clients)", so every count IS the output: a justified full
	// scan, like bare pair's picker (pair#228).
	var queryErr error
	rows := buildListRowsForScope(names, states, index, scopeKeyFromDataDir(r.GlobalDataDir, r.DataDir), r.InferAgent, func(session string) int {
		if queryErr != nil {
			return 0
		}
		var count int
		count, queryErr = source.ClientCountContext(context.Background(), session)
		return count
	})
	if queryErr != nil {
		return nil, queryErr
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
	} else {
		return "", "", false
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

func (OSRuntime) ConfirmLayoutChange(tag string, from, to LayoutMode) bool {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return false
	}
	defer tty.Close()
	fmt.Fprintf(tty, "pair: switch tag %q from %s to %s? This restarts the whole workbench and loses user-terminal state. [y/N]: ", tag, from, to)
	line, _ := bufio.NewReader(tty).ReadString('\n')
	fmt.Fprintln(tty)
	answer := strings.TrimSpace(line)
	return answer == "y" || answer == "Y"
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

// runningPairExe resolves the running `pair` executable for the self-exec
// sidecar spawns (#104 M2 folded pair-title/pair-session-watch into `pair title`
// / `pair session-watch`). The launcher is always `pair`, so os.Executable() is
// the pair binary directly — this works in the copied/Homebrew layout where
// $PAIR_HOME/bin holds no `pair` (pair is never in its own runtime bundle).
// (cf. clipcmd.siblingPairExe, which may run under a helper's name and so
// resolves the pair sibling instead.) Falls back to $PAIR_HOME/bin/pair only if
// os.Executable() is unavailable.
func runningPairExe(pairHome string) string {
	if exe, err := os.Executable(); err == nil && exe != "" {
		return exe
	}
	return filepath.Join(pairHome, "bin", "pair")
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

// spawnDetached backgrounds argv[0] with stdio to /dev/null. Direct Pair puts
// it in its own session (the historical disown behavior); Couch-launched Pair
// leaves it in the actor-owned process group until ownership handoff.
func spawnDetached(argv []string, extraEnv []string) {
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		return
	}
	defer devNull.Close()
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Env = childEnviron(extraEnv)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = devNull, devNull, devNull
	cmd.SysProcAttr = sidecarProcessAttributes(os.Getenv("COUCH_THREAD_SCOPE"), os.Getenv("COUCH_THREAD_TAG"))
	if err := cmd.Start(); err != nil {
		return
	}
	go func() { _ = cmd.Wait() }() // reap our bookkeeping when the sidecar exits.
}

// childEnviron is a sidecar's environment: everything this process has, then
// what the spawn hands it explicitly. The explicit entries come LAST because
// os/exec keeps the last value of a duplicate key -- that is what lets a
// launch contract (titlepoller.SessionEnv) override a stale inherited value.
// TestChildEnvironLetsTheContractBeatAnInheritedKey checks it against a real
// child rather than assuming it.
func childEnviron(extraEnv []string) []string {
	return append(os.Environ(), extraEnv...)
}

// A Couch-launched Pair already runs in an actor-owned process group. Its
// pre-handoff sidecars must remain in that group so Couch can quiesce the whole
// incarnation on a failed start. Direct Pair sidecars retain the historical
// detached-session behavior.
func sidecarProcessAttributes(couchScope, couchTag string) *syscall.SysProcAttr {
	if couchScope != "" && couchTag != "" {
		return nil
	}
	return &syscall.SysProcAttr{Setsid: true}
}

// --- EnvOps ----------------------------------------------------------------

func (OSRuntime) SetEnv(key, value string) { _ = os.Setenv(key, value) }

// InZellijPane walks the PPID chain looking for a zellij ancestor — the ground
// truth (env vars leak across cmux panes, so $ZELLIJ can't be trusted).
func (r OSRuntime) InZellijPane() bool {
	rows, err := r.Processes.Table()
	return err == nil && hasProcessAncestor(rows, os.Getpid(), "zellij")
}

func hasProcessAncestor(rows []procutil.Process, pid int, name string) bool {
	byPID := make(map[int]procutil.Process, len(rows))
	for _, row := range rows {
		byPID[row.PID] = row
	}
	seen := make(map[int]bool)
	for pid > 1 && !seen[pid] {
		seen[pid] = true
		row, ok := byPID[pid]
		if !ok {
			return false
		}
		if row.Name == name {
			return true
		}
		pid = row.PPID
	}
	return false
}

func (OSRuntime) CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func (r OSRuntime) RecordOuterTTY(tag string, couch bool) {
	paths, err := artifactpath.ResolveScoped(r.DataDir, tag)
	if err != nil {
		return
	}
	path := paths.OuterTTY()
	cmd := exec.Command("tty")
	cmd.Stdin = os.Stdin
	out, _ := cmd.Output()
	outer := strings.TrimSpace(string(out))
	if strings.HasPrefix(outer, "/dev/") {
		_ = r.WriteAtomic(path, EncodeOuterRecord(OuterRecord{TTY: outer, Couch: couch}))
	} else {
		r.Remove(path)
	}
}

func (r OSRuntime) OuterPresenter(tag string) (bool, error) {
	return ReadOuterPresenter(r.DataDir, tag)
}

// ReadOuterPresenter reports whether the client that last attached to tag was
// presented by Couch -- what Alt+h reads to decide whether Couch's keys belong
// on the page (#282). No record (never attached from a tty, or quit) is "no".
func ReadOuterPresenter(dataDir, tag string) (bool, error) {
	paths, err := artifactpath.ResolveScoped(dataDir, tag)
	if err != nil {
		return false, err
	}
	raw, err := os.ReadFile(paths.OuterTTY())
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	record, err := DecodeOuterRecord(string(raw))
	return record.Couch, err
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
	exec.Command("cmux", "rename-workspace", title).Run()
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

// --- ReadinessOps ----------------------------------------------------------

func (OSRuntime) MintLaunchNonce() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return ""
	}
	return fmt.Sprintf("%x", b[:])
}

func (r OSRuntime) WaitReadyRecord(expect ReadyExpectation, timeout time.Duration) (readiness.ReadyRecord, error) {
	deadline := time.Now().Add(timeout)
	path := AgentReadyPath(r.DataDir, expect.Tag, expect.Agent)
	for {
		if raw, err := r.ReadFile(path); err == nil {
			record, err := readiness.Decode(raw)
			if err == nil {
				if matchErr := MatchReadyRecord(expect, record, r.PIDAlive); matchErr == nil {
					return record, nil
				}
			}
		}
		if !time.Now().Before(deadline) {
			return readiness.ReadyRecord{}, fmt.Errorf("timed out waiting for %s readiness", expect.Agent)
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func (OSRuntime) PIDAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

// --- AgentDefaultOps -------------------------------------------------------

func (r OSRuntime) ReadAgentDefault(agent string) (AgentDefault, bool) {
	raw, err := r.ReadFile(AgentDefaultPath(r.DataDir, agent))
	if err != nil {
		return AgentDefault{}, false
	}
	d, err := ParseAgentDefault(agent, raw)
	if err != nil {
		return AgentDefault{}, false
	}
	return d, true
}

func (r OSRuntime) WriteAgentDefault(agent string, args []string) error {
	raw, err := BuildAgentDefault(agent, args)
	if err != nil {
		return err
	}
	return r.WriteAtomic(AgentDefaultPath(r.DataDir, agent), raw)
}

// InferAgent reads current ledger ownership first, then the live agent record.
// Compatibility config filenames are saved launch parameters, not authority.
func (r OSRuntime) InferAgent(tag string) string {
	paths, pathErr := artifactpath.ResolveScoped(r.DataDir, tag)
	if pathErr != nil {
		return ""
	}
	// Agent inference is presentation metadata, not native resume authority.
	// Read only local bounded records; resolving typed native targets here
	// would make an optional context meter delay a selected terminal attach.
	if raw, err := osfs.ReadRegularMetadata(context.Background(), paths.Ledger(), 4*1024*1024); err == nil {
		entries := parseLedger(string(raw), func(sessionledger.Current) sessioninventory.ResumeTarget { return sessioninventory.ResumeTarget{} })
		if latest, ok := LatestLedgerEntry(entries); ok && latest.Agent != "" {
			return latest.Agent
		}
	}
	if raw, err := osfs.ReadRegularMetadata(context.Background(), paths.Agent(), 4096); err == nil {
		if a := strings.TrimSpace(string(raw)); a != "" {
			return a
		}
	}
	return ""
}

func (r OSRuntime) ReadLedger(tag string) ([]LedgerEntry, error) {
	paths, err := artifactpath.ResolveScoped(r.DataDir, tag)
	if err != nil {
		return nil, err
	}
	raw, err := r.ReadFile(paths.Ledger())
	if err != nil {
		return nil, err
	}
	type namespace struct {
		native sessioninventory.OSRuntime
		err    error
	}
	namespaces := map[string]namespace{}
	lookup := func(agent string) namespace {
		if found, ok := namespaces[agent]; ok {
			return found
		}
		native, err := r.nativeInventory(agent, "")
		found := namespace{native: native, err: err}
		namespaces[agent] = found
		return found
	}
	entries := parseLedger(raw, func(current sessionledger.Current) sessioninventory.ResumeTarget {
		found := lookup(current.Launch.Agent)
		if found.err != nil {
			return sessioninventory.ResumeTarget{Status: sessioninventory.BindingUnbound}
		}
		return sessioninventory.ResumeTargetForRuntimeLaunch(found.native, current)
	})
	// Check every represented agent, including compatibility-only rows that
	// never invoke the typed projection. Namespace failures block that agent's
	// resume authority without discarding other agents or inventing entries.
	for index := range entries {
		if lookup(entries[index].Agent).err != nil {
			entries[index].SessionID = ""
			entries[index].ResumeBlocked = true
		}
	}
	return entries, nil
}

func (r OSRuntime) AppendLedger(tag string, entry LedgerEntry) error {
	paths, err := artifactpath.ResolveScoped(r.DataDir, tag)
	if err != nil {
		return err
	}
	line, err := BuildLedgerLine(entry)
	if err != nil {
		return err
	}
	store := sessionledger.LedgerStore{Runtime: sessionledger.OSRuntime{}}
	ordinal, err := store.AppendLegacy(paths.Ledger(), []byte(line))
	if sessionledger.AppendOutcomeOf(err) != sessionledger.AppendIndeterminate {
		return err
	}
	reconcileErr := store.ReconcileLegacy(paths.Ledger(), ordinal, []byte(line))
	if reconcileErr == nil || sessionledger.AppendOutcomeOf(reconcileErr) == sessionledger.AppendCommitted {
		return reconcileErr
	}
	return errors.Join(err, reconcileErr)
}

func (r OSRuntime) PrepareSessionLaunch(scopeKey, tag, agent, resumeNativeID string, origin sessionledger.RequestOrigin) (uint64, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return 0, err
	}
	prepared, err := sessionwatch.PrepareOSLaunchRequest(home, r.DataDir, sessionledger.Owner{ScopeKey: scopeKey, Tag: tag, Agent: agent}, resumeNativeID, origin)
	return prepared.Launch.Ordinal, err
}

// pair:m5-concept integration
func (r OSRuntime) ReadSessionNameIndex() (SessionNameIndex, error) {
	return readSessionNameIndexes(r.globalDataDir(), r.DataDir, r.ReadFile)
}

func (r OSRuntime) AppendSessionNameIndex(entry SessionNameEntry) error {
	owner, err := storagegc.SelectedOwner(r.DataDir, "", entry.Tag)
	if err != nil {
		return err
	}
	coordinator, err := storagegc.NewCoordinator(owner.DataDir)
	if err != nil {
		return err
	}
	return coordinator.WithLock(context.Background(), func(held *storagegc.Locked) error {
		return r.appendSessionNameIndex(entry)
	})
}

func (r OSRuntime) appendSessionNameIndex(entry SessionNameEntry) error {
	scope, err := artifactpath.ResolveSelectedScope(r.DataDir)
	if err != nil {
		return err
	}
	path := scope.SessionBindings()
	var raw string
	if existing, err := r.ReadFile(path); err == nil {
		raw = existing
	}
	line, err := BuildSessionNameIndexLine(entry)
	if err != nil {
		return err
	}
	if raw != "" && !strings.HasSuffix(raw, "\n") {
		raw += "\n"
	}
	raw += line + "\n"
	return r.WriteAtomic(path, raw)
}

func (r OSRuntime) globalDataDir() string {
	if r.GlobalDataDir != "" {
		return r.GlobalDataDir
	}
	return r.DataDir
}

func (r OSRuntime) AgentSessionExists(agent, sid, cwd string) bool {
	if sid == "" {
		return false
	}
	runtime, err := r.nativeInventory(agent, cwd)
	if err != nil {
		// Unknown namespace cannot prove absence.
		return true
	}
	nativeAgent := sessioninventory.Agent(agent)
	if scope, err := artifactpath.ResolveSelectedScope(r.DataDir); err == nil {
		if catalog, err := (sessioninventory.CatalogStore{Runtime: sessioninventory.CatalogOSRuntime{}}).Read(scope.SessionInventoryCatalog()); err == nil {
			inventory := sessioninventory.NewIncrementalInventory(runtime, catalog)
			snapshot := inventory.Observe(nativeAgent)
			if inventory.CatalogSessionCandidateExists(snapshot, nativeAgent, sid) {
				return true
			}
			return sessioninventory.NativeSessionCandidateExistsFromObservations(runtime, nativeAgent, sid, snapshot.Observations)
		}
	}
	inventory := sessioninventory.NewIncrementalInventory(runtime, sessioninventory.Catalog{Version: sessioninventory.CatalogVersion})
	snapshot := inventory.Observe(nativeAgent)
	return sessioninventory.NativeSessionCandidateExistsFromObservations(runtime, nativeAgent, sid, snapshot.Observations)
}

func (r OSRuntime) EstablishedSessionID(scopeKey, tag, agent string) (string, sessioninventory.BindingStatus) {
	runtime, err := r.nativeInventory(agent, "")
	if err != nil {
		return "", sessioninventory.BindingUnbound
	}
	query, err := sessioninventory.QueryResumeTarget(runtime, scopeKey, tag, sessioninventory.Agent(agent))
	if err != nil {
		return "", query.Status
	}
	return query.NativeID, query.Status
}

// --- LifecycleOps ----------------------------------------------------------

// AttachSession is the BLOCKING attach twin of LaunchSession.
func (r OSRuntime) AttachSession(session, configDir string) (int, error) {
	return r.runTerminalHandoff(context.Background(), session, configDir, "", false)
}

func (r OSRuntime) AttachSessionContext(lifetime, startup, admission context.Context, session, configDir string) (int, error) {
	return r.runTerminalHandoffContext(lifetime, startup, admission, session, configDir, "", false)
}

// restartMarkerPath and quitMarkerPath keep compatibility cache paths behind
// the artifactpath authority while preserving the launcher's fail-closed bool
// seam for an invalid HOME or session name.
func restartMarkerPath(session string) (string, bool) {
	paths, err := artifactpath.ResolvePairCache(os.Getenv("HOME"))
	if err != nil {
		return "", false
	}
	path, err := paths.Restart(session)
	return path, err == nil
}

func quitMarkerPath(session string) (string, bool) {
	paths, err := artifactpath.ResolvePairCache(os.Getenv("HOME"))
	if err != nil {
		return "", false
	}
	path, err := paths.Quit(session)
	return path, err == nil
}

func (r OSRuntime) TakeQuitMarker(session string) bool {
	path, ok := quitMarkerPath(session)
	if !ok {
		return false
	}
	if !fileExists(path) {
		return false
	}
	r.Remove(path)
	return true
}

func (OSRuntime) RestartMarkerPresent(session string) bool {
	path, ok := restartMarkerPath(session)
	if !ok {
		return false
	}
	return fileExists(path)
}

// WriteRestartMarker + TouchQuitMarker are the in-session compaction write twins
// (#99 M5b, shell 1052-1058); WriteAtomic/Touch MkdirAll the cache dir.
func (r OSRuntime) WriteRestartMarker(session string, m RestartMarker) error {
	return r.writeRestartMarker(session, m)
}

func (r OSRuntime) TouchQuitMarker(session string) {
	path, ok := quitMarkerPath(session)
	if !ok {
		return
	}
	_ = r.Touch(path)
}

// ExecKillSession execs `${PAIR_KILL_CMD:-zellij kill-session} <session>` and
// does NOT return on success (syscall.Exec replaces the process — the compaction
// pane dies, the outer bin/pair regains the tty). A missing binary falls through
// so the caller isn't wedged (shell 1060).
func (OSRuntime) ExecKillSession(session string) error {
	argv := []string{"zellij", "kill-session", session}
	if kc := os.Getenv("PAIR_KILL_CMD"); kc != "" {
		argv = append(strings.Fields(kc), session)
	}
	path, err := exec.LookPath(argv[0])
	if err != nil {
		return err
	}
	return syscall.Exec(path, argv, os.Environ())
}

// DeleteSession returns success only after the authorized server generations
// and live session are observed absent. Saved layouts are retained; their
// existence is not evidence of a running terminal.
func (r OSRuntime) DeleteSession(session string) error {
	return r.DeleteSessionContext(context.Background(), session)
}

// ReapNvim ends this tag's embedded nvim processes, and only on verified
// identity (#583): a pid record names a pid, the process table says what that
// pid is running now, and the kernel start stamp is re-read before the signal.
// There is no pattern kill here — the sweep that used to follow the pidfiles
// could reach any process whose argv mentioned the draft's path.
func (r OSRuntime) ReapNvim(tag string) {
	paths, err := artifactpath.ResolveScoped(r.DataDir, tag)
	if err != nil {
		return
	}
	_, _ = reapNvimSidecars(context.Background(), r.processTable(), r, nvimSidecarsFor(paths, tag, paths.ScopeDir(), false))
}

// SweepOrphanNvim reaps embedded nvim processes whose Pair session is not live,
// one pid at a time and only on verified identity (#583). Candidates come from
// the nvim-pid sidecars, the changelog viewer's artifact-keyed record, and argv
// that carries a scope path; the decision is per pid, so an embed is never
// reached through a name it happens to match.
func (r OSRuntime) SweepOrphanNvim(liveTags []string) {
	_, _ = sweepOrphanNvim(context.Background(), r.processTable(), r, r.DataDir, liveTags)
}

// ParkScrollback preserves the .raw/.events.jsonl scrollback to a timestamped
// parked-scrollback-<tag>-<ts> base (move on quit, copy on compaction) and
// touches parked-<tag> (shell 696-708). The timestamp is taken here (time.Now is
// live in OSRuntime, unlike a pure decider).
func (r OSRuntime) ParkScrollback(tag, agent string, move bool) (string, bool) {
	return r.ParkScrollbackContext(context.Background(), tag, agent, move)
}
func (r OSRuntime) ParkScrollbackContext(ctx context.Context, tag, agent string, move bool) (string, bool) {
	if ctx == nil || ctx.Err() != nil {
		return "", false
	}
	lease, err := storagegc.AcquireSelectedProcess(ctx, func(key string) string {
		switch key {
		case "PAIR_DATA_DIR":
			return r.DataDir
		case "PAIR_TAG":
			return tag
		}
		return ""
	}, "capture-producer")
	if err != nil || lease == nil {
		return "", false
	}
	defer lease.Close()
	paths, err := artifactpath.ResolveScoped(lease.Owner.Directory(), tag)
	if err != nil {
		return "", false
	}
	scrollback, err := paths.ScrollbackArtifacts(agent)
	if err != nil {
		return "", false
	}
	if size, ok := r.FileSize(scrollback.Raw); !ok || size == 0 {
		return "", false
	}
	// Capture provenance must be checked against the source generation before
	// transfer changes its file IDs. Missing historical evidence stays legacy.
	witness, err := storagegc.ReadCaptureWitness(ctx, scrollback.Raw, scrollback.Events)
	skipEvents := false
	if err != nil {
		// Optional unreadable events must not discard usable raw history. Keep
		// that capture explicitly uncertified; malformed authority still refuses.
		var pathError *os.PathError
		var invalid *osfs.InvalidMetadataError
		unreadableType := errors.As(err, &invalid) && invalid.Kind == osfs.MetadataNotRegular
		if !errors.As(err, &pathError) && !unreadableType {
			return "", false
		}
		witness = nil
		skipEvents = true
	}
	if witness != nil && (witness.ScopeKey != lease.Owner.RepoScope || witness.Tag != tag || witness.Agent != agent) {
		return "", false
	}
	// Exclusive destination creation protects earlier captures, including concurrent
	// parks and copies in the same second. The suffix remains a family component.
	capturedAt := time.Now()
	stamp := capturedAt.Format("20060102T150405")
	for collision := 0; ; collision++ {
		if ctx.Err() != nil {
			return "", false
		}
		token := stamp
		if collision > 0 {
			token = fmt.Sprintf("%s-%d", stamp, collision)
		}
		parked, err := paths.ParkedScrollbackArtifacts(token)
		if err != nil {
			return "", false
		}
		// An orphan sidecar must not become metadata for this new raw capture.
		if _, err := os.Lstat(parked.Metadata); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", false
		}
		if _, err := os.Lstat(parked.Events); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", false
		}
		var rawID, eventsID *artifactpath.CaptureFileID
		if witness != nil {
			rawID = &witness.RawFile
			eventsID = &witness.EventsFile
		}
		err = transferFileContext(ctx, scrollback.Raw, parked.Raw, move, rawID)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return "", false
		}
		// Preserve the selected directory spelling for lifecycle token extraction;
		// only retention metadata uses canonical physical paths.
		selectedBase := filepath.Join(r.DataDir, filepath.Base(parked.Base))
		// Events are optional; failed copies remove their incomplete destination.
		if !skipEvents {
			if err := transferFileContext(ctx, scrollback.Events, parked.Events, move, eventsID); err != nil {
				witness = nil
			}
		}
		member, err := artifactpath.MatchArtifact(parked.Raw, []artifactpath.StorageOwner{lease.Owner}, nil)
		if err != nil {
			return selectedBase, false
		}
		capture, err := artifactpath.ParseParkedCapture(member, time.UTC)
		if err != nil {
			return selectedBase, false
		}
		if err := lease.Coordinator.PublishCaptureMetadata(ctx, lease.Owner, capture, capturedAt, witness); err != nil {
			return selectedBase, false
		}
		_ = r.Touch(paths.Parked())
		return selectedBase, true
	}
}

// transferFile streams to an exclusive destination and only removes the source
// after the complete capture has been synced and closed.
func transferFile(src, dst string, move bool, expected ...*artifactpath.CaptureFileID) error {
	return transferFileContext(context.Background(), src, dst, move, expected...)
}
func transferFileContext(ctx context.Context, src, dst string, move bool, expected ...*artifactpath.CaptureFileID) error {
	if ctx == nil {
		return errors.New("capture transfer requires context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(expected) > 1 {
		return errors.New("multiple expected capture identities")
	}
	source, err := os.OpenFile(src, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return err
	}
	defer source.Close()
	before, err := source.Stat()
	if err != nil {
		return err
	}
	native, ok := before.Sys().(*syscall.Stat_t)
	if !ok || !before.Mode().IsRegular() {
		return errors.New("capture source must be regular")
	}
	if len(expected) == 1 && expected[0] != nil && *expected[0] != (artifactpath.CaptureFileID{Device: uint64(native.Dev), Inode: uint64(native.Ino)}) {
		return errors.New("capture source generation changed before transfer")
	}
	destination, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, copyErr := io.CopyN(destination, captureContextReader{ctx: ctx, reader: source}, before.Size())
	if copyErr == nil {
		copyErr = ctx.Err()
	}
	if copyErr == nil {
		after, e := source.Stat()
		current, e2 := os.Lstat(src)
		if e != nil || e2 != nil {
			copyErr = errors.Join(e, e2)
		} else if !current.Mode().IsRegular() || !os.SameFile(before, current) || !os.SameFile(before, after) || before.Size() != after.Size() || before.Size() != current.Size() || !before.ModTime().Equal(after.ModTime()) || !before.ModTime().Equal(current.ModTime()) {
			copyErr = errors.New("capture source changed during transfer")
		}
	}
	if copyErr == nil {
		copyErr = destination.Sync()
	}
	closeErr := destination.Close()
	if err := errors.Join(copyErr, closeErr); err != nil {
		_ = os.Remove(dst)
		return err
	}
	if move {
		return os.Remove(src)
	}
	return nil
}

type captureContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r captureContextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
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

// KillTitlePoller remains on the launcher Runtime for lifecycle compatibility.
// A pidfile grants no signal or removal authority: the wrapper closes its own
// child's lease, and that generation owns its metadata. Legacy processes use
// the separately sampled, identity-fenced cleanup path.
func (r OSRuntime) KillTitlePoller(tag string) {}

func (r OSRuntime) PairOwnsCmuxWorkspace(tag string) bool {
	wsID := os.Getenv("CMUX_WORKSPACE_ID")
	if wsID == "" {
		return false
	}
	raw, err := r.ReadFile(filepath.Join(r.DataDir, "cmux-owner-"+wsID))
	fields := strings.Fields(strings.TrimSpace(raw))
	return err == nil && len(fields) > 0 && fields[0] == tag
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
