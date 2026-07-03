package launcher

import (
	"errors"
	"time"
)

// The launcher.Runtime effect seam (#99 M2). Every IO the create-flow
// orchestration performs — zellij exec/query, the fzf/vared UIs, config
// read/write, cmux, child-spawns, tty, env, uuid mint — sits behind this seam so
// RunLaunch is driven by a fake in tests and the pure deciders (createlogic.go +
// M1's agentargs/config) stay unit-tested directly. Composed from small
// sub-interfaces (ISP): a fake test can stub one concern, and each orchestration
// helper takes only the sub-seam it needs. The concrete OSRuntime (osruntime.go)
// wires the real calls, embedding osfs.FS for the filesystem primitives.

// ErrFallbackToShell signals that the native launcher declines this launch —
// the decision resolved to a non-create path (attach/pick) that M2 doesn't own
// yet. The cmd/pair-go gate falls back to exec'ing bin/pair-shell.
var ErrFallbackToShell = errors.New("launcher: not a native create path (fallback to shell)")

// ZellijOps is the zellij session boundary the create path needs.
type ZellijOps interface {
	// Sessions returns the pair-* zellij sessions with their reuse state
	// (the ZellijSource classification: attached/detached/exited).
	Sessions() ([]Session, error)
	// SessionBlocksReuse reports whether a live zellij session named session
	// blocks reuse of its tag, clearing a stale EXITED resurrect record as a
	// side effect (#67) so a create can reclaim the name.
	SessionBlocksReuse(session string) bool
	// ProbeSessionName asks zellij's own validator whether session is a legal
	// name on this machine's socket-path budget (#54); a too-long name returns
	// an error the caller surfaces as "pick a shorter tag".
	ProbeSessionName(session string) error
	// LaunchSession runs `zellij --new-session-with-layout` as a BLOCKING
	// fork+wait child with the tty passed through, returning the child's exit
	// code when the pane exits. It must NOT syscall.Exec — the Go launcher has
	// to regain control afterward for the M3 quit-cleanup / restart loop.
	LaunchSession(session, configDir, layout string) (int, error)
}

// SnapshotOps supplies the historical-tag half of the launch decision snapshot
// (the draft/log sidecar scan); the zellij half is ZellijOps.Sessions.
type SnapshotOps interface {
	ScanHistory(base string, cutoff time.Time) ([]HistoricalTag, error)
}

// ListOps gathers the `pair list`/`ls` rows (#99 M5a) — pair-<tag> sessions with
// resolved agent, reuse state, and live client count. Its own seam (ISP): the
// launch path never needs client counts, so this stays off ZellijOps.Sessions.
type ListOps interface {
	ListSessions() ([]ListRow, error)
}

// UIOps is the interactive surface — the name prompt, the config picker, and the
// terminal-title / family-existing writes that go to the real tty.
type UIOps interface {
	// ShowFamilyExisting prints the existing pair-<base>* zellij sessions
	// before the name prompt so the user sees why the prefill may carry a
	// numeric suffix (a cosmetic /dev/tty write; OSRuntime queries zellij).
	ShowFamilyExisting(familyPrefix string)
	// PromptSessionName shows an editable prompt (zsh vared) prefilled with def
	// and returns the typed value; ok=false means the user aborted (ESC/EOF).
	PromptSessionName(def string) (value string, ok bool)
	// PickFromList presents options via fzf (--read0 multi-line) under header and
	// returns the chosen line, or "" if the user aborted.
	PickFromList(header string, options []string, height int) string
	// SetTerminalTitle emits the OSC title escape for session.
	SetTerminalTitle(session string)
}

// ProcOps spawns the (already-Go) sidecar children and the dev rebuild.
type ProcOps interface {
	// SpawnSessionWatcher backgrounds bin/pair-session-watch.sh (detached) to
	// capture the async agent session id for codex/agy; a no-op-ish spawn for
	// claude (whose id is minted synchronously).
	SpawnSessionWatcher(agent, tag, cwd string, agentArgs []string)
	// SpawnTitlePoller backgrounds bin/pair-title.sh (detached), the per-tag
	// frame/cmux title singleton.
	SpawnTitlePoller(tag, agent string)
	// DevRebuild rebuilds the repo Go binaries when PAIR_DEV is set (no-op
	// otherwise) so the layout's `exec pair-wrap` resolves a fresh build (#46).
	DevRebuild(pairHome string)
}

// EnvOps covers the process environment + host guards.
type EnvOps interface {
	// SetEnv exports key=value into this process so every child (watcher,
	// poller, zellij, its panes) inherits it — the Go analogue of the shell's
	// `export`.
	SetEnv(key, value string)
	// InZellijPane reports whether this process is a descendant of a zellij
	// process (PPID ancestry, not $ZELLIJ env — cmux leaks that to siblings).
	InZellijPane() bool
	// CommandExists reports whether name resolves on PATH (`command -v`).
	CommandExists(name string) bool
	// RecordOuterTTY writes the launching tty to outer-tty-<tag> (or removes it
	// when stdin isn't a tty).
	RecordOuterTTY(tag string)
	// CmuxRename claims the cmux workspace for this tag and renames it to title
	// (with the personal emoji substitution); a no-op outside cmux.
	CmuxRename(tag, title string)
}

// IDOps mints/checks the deterministic session ids and resolves the paired agent.
type IDOps interface {
	// MintUUID returns a fresh lowercase uuid (uuidgen).
	MintUUID() string
	// AgentSessionExists reports whether the agent's native session artifact for
	// sid is on disk (claude jsonl / codex sessions glob / agy conversation db).
	AgentSessionExists(agent, sid, cwd string) bool
	// InferAgent resolves the agent a tag was last paired with — the agent-<tag>
	// record (live/detached) or, once that's cleared on Alt+x, the agent encoded
	// in a config-<tag>-<agent>.json filename. "" when neither is on disk (a
	// genuinely fresh tag); the caller then defaults to claude.
	InferAgent(tag string) string
}

// FSOps is the filesystem subset the create path uses (satisfied by osfs.FS).
type FSOps interface {
	ReadFile(path string) (string, error)
	WriteAtomic(path, data string) error
	Remove(path string)
	FileSize(path string) (int64, bool)
	Touch(path string) error
}

// LifecycleOps is the post-handoff + attach effect surface (#99 M3): the blocking
// attach, the quit/restart marker read-clears, session teardown, nvim reaping,
// the park-nudge, the title-poller kill, and the cmux-ownership reset. Embedded in
// Runtime; the fake drives the loop tests and OSRuntime wires the real
// zellij/nvim/cmux/tty calls. The markers live under ~/.cache/pair/{quit,restart}-
// <session>; parsing + the re-launch decision stay pure (markers.go).
type LifecycleOps interface {
	// AttachSession runs `zellij --config-dir CFG attach SESSION` as a BLOCKING
	// fork+wait child (tty passed through), returning the child's exit code —
	// the attach twin of LaunchSession (NOT syscall.Exec, so the loop regains
	// control for cleanup + restart).
	AttachSession(session, configDir string) (int, error)
	// TakeQuitMarker read-clears ~/.cache/pair/quit-<session>, reporting whether
	// it was present (Alt+x). Cleanup runs its body only when it was.
	TakeQuitMarker(session string) bool
	// RestartMarkerPresent PEEKS ~/.cache/pair/restart-<session> WITHOUT clearing
	// it — the park-nudge skip (a restart isn't a quit; the relaunch keeps the
	// work), shell 1553.
	RestartMarkerPresent(session string) bool
	// TakeRestartMarker read-clears the restart marker and parses it; ok=false
	// when absent (the loop terminates), shell 717-733.
	TakeRestartMarker(session string) (RestartMarker, bool)
	// DeleteSession removes the zellij session record (delete-session --force)
	// and SIGKILLs any lingering `zellij --server …/<session>` process, shell
	// 1528-1534.
	DeleteSession(session string)
	// ReapNvim kills this tag's nvim --embed children (pidfiles + pattern sweep),
	// shell 1089-1112.
	ReapNvim(tag string)
	// SweepOrphanNvim reaps nvim --embed whose pair-<tag> is not in liveTags —
	// startup hygiene for externally-killed sessions that left no quit marker,
	// shell 1117-1158.
	SweepOrphanNvim(liveTags []string)
	// ParkScrollback preserves scrollback-<tag>-<agent>.{raw,events.jsonl} to a
	// timestamped parked-scrollback-<tag>-<ts> base (move on quit, copy on
	// compaction) and touches parked-<tag>; ok=false when there's nothing to
	// park, shell 696-708.
	ParkScrollback(tag, agent string, move bool) (base string, ok bool)
	// ConfirmParkNudge shows the [y/N] preserve prompt bounded by timeoutSecs
	// (auto-declines on timeout/EOF/no), shell 1564-1565.
	ConfirmParkNudge(session string, timeoutSecs int) bool
	// IsTTY reports whether stdin is a terminal (`[ -t 0 ]`, shell 1552).
	IsTTY() bool
	// KillTitlePoller SIGTERMs + clears the title-pid-<tag> poller, shell
	// 1621-1627.
	KillTitlePoller(tag string)
	// PairOwnsCmuxWorkspace reports whether this tag owns the current cmux
	// workspace (CMUX_WORKSPACE_ID set + cmux-owner-<id> == tag), shell 902-907.
	PairOwnsCmuxWorkspace(tag string) bool
	// ClearCmuxOwner removes the cmux-owner-<CMUX_WORKSPACE_ID> record, shell 1645.
	ClearCmuxOwner()
}

// Runtime is the full launcher effect seam.
type Runtime interface {
	ZellijOps
	SnapshotOps
	ListOps
	UIOps
	ProcOps
	EnvOps
	IDOps
	FSOps
	LifecycleOps
}

// LaunchOptions are RunLaunch's post-parse inputs — the resolved argv (Args),
// the launch environment (Env), the asset root (PairHome, for zellij/nvim/bin
// paths + PAIR_HOME), and the two create-flow flags the shell reads from env.
type LaunchOptions struct {
	Args                 LaunchArgs
	Env                  Env
	PairHome             string
	ContinueDoc          string // seed the draft to read this continuation (create-only)
	CodexAltScreenOptOut bool   // PAIR_CODEX_ALT_SCREEN=1: leave codex in alt-screen
	ParkPromptTimeout    int    // PAIR_PARK_PROMPT_TIMEOUT (default 5): the quit park-nudge [y/N] bound
}
