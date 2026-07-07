package launcher

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

// fakeRuntime is the in-memory create-flow seam for the RunLaunch loop tests.
// Canned inputs drive decisions; recorded outputs assert the effect sequence.
type fakeRuntime struct {
	inPane         bool
	sessions       []Session
	historical     []HistoricalTag
	blocksReuse    map[string]bool // session -> live-blocks (default false)
	commandMissing map[string]bool // name -> absent (default: everything exists)
	files          map[string]string
	agentSessions  map[string]bool // "agent|sid" -> native artifact exists
	uuids          []string        // MintUUID pops these in order
	promptValue    string
	promptOK       bool
	probeErr       error
	inferAgent     map[string]string // tag -> paired agent (for `resume <tag>`)
	pickFunc       func(header string, options []string) string
	listRows       []ListRow // ListSessions rows (for `pair list`)
	listErr        error
	sessionsErr    error       // Sessions() error (defensive exit-1 path)
	renameFailAt   string      // Rename returns an error when src == this (rollback test)
	renamed        [][2]string // {src,dst} per successful Rename
	// #99 M5b compaction/continue
	writtenMarkers   map[string]RestartMarker // WriteRestartMarker by session
	touchedQuit      []string                 // TouchQuitMarker sessions
	killed           []string                 // ExecKillSession sessions
	continuationDocs map[string][2]string     // slug -> {path, agent} for ResolveContinuationDoc
	continuationRows []ContinuationRow        // ScanContinuations rows
	continuationDir  string                   // ScanContinuations dir

	// M3 lifecycle inputs
	isTTY          bool
	confirmPark    bool
	parkOK         bool                     // ParkScrollback returns ("<base>", parkOK)
	attachCode     int                      // AttachSession exit code
	quitMarkers    map[string]bool          // session -> Alt+x quit marker (read-cleared)
	restartMarkers map[string]RestartMarker // session -> restart marker (peek + take-once)
	cmuxOwned      map[string]bool          // tag -> PairOwnsCmuxWorkspace

	// recorded
	env           map[string]string
	launched      string // last session name handed to LaunchSession
	launchCode    int
	launchCount   int      // number of create handoffs (restart-loop iterations)
	watchers      []string // "agent|tag|cwd|args"
	pollers       []string // "tag|agent"
	cmux          []string // "tag|title"
	ttyRecorded   []string
	titles        []string
	removed       []string
	family        []string
	devRebuilt    bool
	attached      []string   // sessions handed to AttachSession
	deleted       []string   // sessions handed to DeleteSession
	reaped        []string   // tags handed to ReapNvim
	swept         [][]string // liveTags per SweepOrphanNvim call
	parkPrompts   []string   // sessions prompted via ConfirmParkNudge
	parked        []string   // "tag|agent|move" per ParkScrollback
	killedPollers []string   // tags handed to KillTitlePoller
	cmuxCleared   int        // ClearCmuxOwner calls
}

func newFakeRuntime() *fakeRuntime {
	return &fakeRuntime{
		blocksReuse:    map[string]bool{},
		commandMissing: map[string]bool{},
		files:          map[string]string{},
		agentSessions:  map[string]bool{},
		inferAgent:     map[string]string{},
		promptOK:       true,
		env:            map[string]string{},
		quitMarkers:    map[string]bool{},
		restartMarkers: map[string]RestartMarker{},
		cmuxOwned:      map[string]bool{},
	}
}

// ZellijOps
func (f *fakeRuntime) Sessions() ([]Session, error)           { return f.sessions, f.sessionsErr }
func (f *fakeRuntime) SessionBlocksReuse(session string) bool { return f.blocksReuse[session] }
func (f *fakeRuntime) ProbeSessionName(session string) error  { return f.probeErr }
func (f *fakeRuntime) LaunchSession(session, configDir, layout string) (int, error) {
	f.launched = session
	f.launchCount++
	return f.launchCode, nil
}

// SnapshotOps
func (f *fakeRuntime) ScanHistory(base string, cutoff time.Time) ([]HistoricalTag, error) {
	return f.historical, nil
}

// ListOps
func (f *fakeRuntime) ListSessions() ([]ListRow, error) { return f.listRows, f.listErr }

// ContinuationOps (#99 M5b)
func (f *fakeRuntime) ResolveContinuationDoc(slug string) (string, string, bool) {
	if d, ok := f.continuationDocs[slug]; ok {
		return d[0], d[1], true
	}
	return "", "", false
}
func (f *fakeRuntime) ScanContinuations() ([]ContinuationRow, string) {
	return f.continuationRows, f.continuationDir
}

// UIOps
func (f *fakeRuntime) ShowFamilyExisting(familyPrefix string) {
	f.family = append(f.family, familyPrefix)
}
func (f *fakeRuntime) PromptSessionName(def string) (string, bool) {
	if f.promptValue != "" {
		return f.promptValue, f.promptOK
	}
	return def, f.promptOK
}
func (f *fakeRuntime) PickFromList(header string, options []string, height int) string {
	if f.pickFunc == nil {
		return ""
	}
	return f.pickFunc(header, options)
}
func (f *fakeRuntime) SetTerminalTitle(session string) { f.titles = append(f.titles, session) }

// ProcOps
func (f *fakeRuntime) SpawnSessionWatcher(agent, tag, cwd string, agentArgs []string) {
	f.watchers = append(f.watchers, agent+"|"+tag+"|"+cwd+"|"+strings.Join(agentArgs, " "))
}
func (f *fakeRuntime) SpawnTitlePoller(tag, agent string) {
	f.pollers = append(f.pollers, tag+"|"+agent)
}
func (f *fakeRuntime) DevRebuild(pairHome string) { f.devRebuilt = true }

// EnvOps
func (f *fakeRuntime) SetEnv(key, value string)       { f.env[key] = value }
func (f *fakeRuntime) InZellijPane() bool             { return f.inPane }
func (f *fakeRuntime) CommandExists(name string) bool { return !f.commandMissing[name] }
func (f *fakeRuntime) RecordOuterTTY(tag string)      { f.ttyRecorded = append(f.ttyRecorded, tag) }
func (f *fakeRuntime) CmuxRename(tag, title string)   { f.cmux = append(f.cmux, tag+"|"+title) }

// IDOps
func (f *fakeRuntime) MintUUID() string {
	if len(f.uuids) == 0 {
		return ""
	}
	u := f.uuids[0]
	f.uuids = f.uuids[1:]
	return u
}
func (f *fakeRuntime) AgentSessionExists(agent, sid, cwd string) bool {
	return f.agentSessions[agent+"|"+sid]
}
func (f *fakeRuntime) InferAgent(tag string) string { return f.inferAgent[tag] }

// FSOps
func (f *fakeRuntime) ReadFile(path string) (string, error) {
	if v, ok := f.files[path]; ok {
		return v, nil
	}
	return "", errors.New("not found")
}
func (f *fakeRuntime) WriteAtomic(path, data string) error { f.files[path] = data; return nil }
func (f *fakeRuntime) Remove(path string) {
	f.removed = append(f.removed, path)
	delete(f.files, path)
}
func (f *fakeRuntime) FileSize(path string) (int64, bool) {
	v, ok := f.files[path]
	return int64(len(v)), ok
}
func (f *fakeRuntime) Touch(path string) error {
	if _, ok := f.files[path]; !ok {
		f.files[path] = ""
	}
	return nil
}
func (f *fakeRuntime) Rename(src, dst string) error {
	if f.renameFailAt != "" && src == f.renameFailAt {
		return errors.New("mv failed (fake)")
	}
	if v, ok := f.files[src]; ok {
		f.files[dst] = v
		delete(f.files, src)
	}
	f.renamed = append(f.renamed, [2]string{src, dst})
	return nil
}
func (f *fakeRuntime) WriteRestartMarker(session string, m RestartMarker) {
	if f.writtenMarkers == nil {
		f.writtenMarkers = map[string]RestartMarker{}
	}
	f.writtenMarkers[session] = m
}
func (f *fakeRuntime) TouchQuitMarker(session string) { f.touchedQuit = append(f.touchedQuit, session) }
func (f *fakeRuntime) ExecKillSession(session string) { f.killed = append(f.killed, session) }

// LifecycleOps
func (f *fakeRuntime) AttachSession(session, configDir string) (int, error) {
	f.attached = append(f.attached, session)
	return f.attachCode, nil
}
func (f *fakeRuntime) TakeQuitMarker(session string) bool {
	if f.quitMarkers[session] {
		delete(f.quitMarkers, session) // read-clear
		return true
	}
	return false
}
func (f *fakeRuntime) RestartMarkerPresent(session string) bool {
	_, ok := f.restartMarkers[session]
	return ok
}
func (f *fakeRuntime) TakeRestartMarker(session string) (RestartMarker, bool) {
	m, ok := f.restartMarkers[session]
	if ok {
		delete(f.restartMarkers, session) // read-clear (one-shot)
	}
	return m, ok
}
func (f *fakeRuntime) DeleteSession(session string) {
	f.deleted = append(f.deleted, session)
	delete(f.blocksReuse, session) // the name is free for a restart re-create
}
func (f *fakeRuntime) ReapNvim(tag string) { f.reaped = append(f.reaped, tag) }
func (f *fakeRuntime) SweepOrphanNvim(liveTags []string) {
	f.swept = append(f.swept, liveTags)
}
func (f *fakeRuntime) ParkScrollback(tag, agent string, move bool) (string, bool) {
	f.parked = append(f.parked, fmt.Sprintf("%s|%s|%t", tag, agent, move))
	return "/data/parked-scrollback-" + tag + "-TS", f.parkOK
}
func (f *fakeRuntime) ConfirmParkNudge(session string, timeoutSecs int) bool {
	f.parkPrompts = append(f.parkPrompts, session)
	return f.confirmPark
}
func (f *fakeRuntime) IsTTY() bool { return f.isTTY }
func (f *fakeRuntime) KillTitlePoller(tag string) {
	f.killedPollers = append(f.killedPollers, tag)
}
func (f *fakeRuntime) PairOwnsCmuxWorkspace(tag string) bool { return f.cmuxOwned[tag] }
func (f *fakeRuntime) ClearCmuxOwner()                       { f.cmuxCleared++ }

func baseOpts(args LaunchArgs) LaunchOptions {
	return LaunchOptions{
		Args:     args,
		Env:      Env{Home: "/home/u", Cwd: "/home/u/work", DataDir: "/data", Now: time.Unix(1_700_000_000, 0), HistoryD: 14},
		PairHome: "/pair",
	}
}

func run(t *testing.T, opts LaunchOptions, rt *fakeRuntime) (int, error) {
	t.Helper()
	var stderr bytes.Buffer
	code, err := RunLaunch(opts, rt, &stderr)
	if stderr.Len() > 0 {
		t.Logf("stderr: %s", stderr.String())
	}
	return code, err
}

// RunLaunch must front the resolved asset root's bin/ on PATH at entry (#95),
// so zellij resolves the shell shims (pair-help/pair-notify) and, in dev, `pair`.
// Since #104 M3 it also fronts the running executable's dir (where `pair` lives
// in the copied/Homebrew layout) — so the asset-root bin/ is among the first two
// front entries (the exe dir, here the go-test binary's dir, precedes it).
// Driven through the fake, so SetEnv records into f.env — no real env pollution.
func TestRunLaunchPrependsBinToPath(t *testing.T) {
	rt := newFakeRuntime()
	rt.uuids = []string{"S"}
	if _, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "x"}), rt); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	got := rt.env["PATH"]
	sep := string(os.PathListSeparator)
	parts := strings.Split(got, sep)
	front := parts
	if len(front) > 2 {
		front = front[:2]
	}
	if !containsStr(front, "/pair/bin") {
		t.Fatalf("RunLaunch did not front the asset-root bin/ on PATH: %q", got)
	}
}

// A forced-tag create with no live session: no prompt, claude mints a session id,
// config + agent record written, sidecars spawned, session handed off.
func TestRunLaunchForcedCreateClaude(t *testing.T) {
	rt := newFakeRuntime()
	rt.uuids = []string{"MINTED-1"}
	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "bugfix"}), rt)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if code != 0 {
		t.Fatalf("code = %d", code)
	}
	if rt.launched != "pair-bugfix" {
		t.Fatalf("launched = %q", rt.launched)
	}
	if len(rt.family) != 0 {
		t.Fatalf("forced create must not prompt/show family: %v", rt.family)
	}
	if rt.env["PAIR_TAG"] != "bugfix" || rt.env["PAIR_AGENT"] != "claude" || rt.env["PAIR_HOME"] != "/pair" {
		t.Fatalf("env = %+v", rt.env)
	}
	if rt.env["PAIR_SESSION_ID"] != "MINTED-1" {
		t.Fatalf("PAIR_SESSION_ID = %q", rt.env["PAIR_SESSION_ID"])
	}
	if !strings.Contains(rt.env["PAIR_AGENT_ARGS"], "--session-id MINTED-1") {
		t.Fatalf("PAIR_AGENT_ARGS = %q", rt.env["PAIR_AGENT_ARGS"])
	}
	// Config written WITHOUT the resume binding (session_id is canonical storage).
	cfg := rt.files["/data/config-bugfix-claude.json"]
	if !strings.Contains(cfg, `"session_id":"MINTED-1"`) || strings.Contains(cfg, "--session-id") {
		t.Fatalf("config = %q", cfg)
	}
	if rt.files["/data/agent-bugfix"] != "claude\n" {
		t.Fatalf("agent record = %q", rt.files["/data/agent-bugfix"])
	}
	if got := rt.watchers; len(got) != 1 || !strings.HasPrefix(got[0], "claude|bugfix|/home/u/work|") {
		t.Fatalf("watchers = %v", got)
	}
	if len(rt.pollers) != 1 || rt.pollers[0] != "bugfix|claude" {
		t.Fatalf("pollers = %v", rt.pollers)
	}
	if len(rt.titles) != 1 || len(rt.ttyRecorded) != 1 || len(rt.cmux) != 1 {
		t.Fatalf("title/tty/cmux effects missing: %v %v %v", rt.titles, rt.ttyRecorded, rt.cmux)
	}
}

// Empty-state create prompts for a name; the typed value drives the tag.
func TestRunLaunchPromptCreate(t *testing.T) {
	rt := newFakeRuntime()
	rt.uuids = []string{"S1"}
	rt.promptValue = "myproj"
	opts := baseOpts(LaunchArgs{Agent: "claude"})
	code, err := run(t, opts, rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if len(rt.family) != 1 {
		t.Fatalf("prompt path should show family: %v", rt.family)
	}
	if rt.launched != "pair-myproj" || rt.env["PAIR_TAG"] != "myproj" {
		t.Fatalf("launched=%q tag=%q", rt.launched, rt.env["PAIR_TAG"])
	}
}

// Aborting the name prompt exits 0 (handled) without launching.
func TestRunLaunchPromptAbort(t *testing.T) {
	rt := newFakeRuntime()
	rt.promptOK = false
	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude"}), rt)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if code != 0 {
		t.Fatalf("abort should exit 0, got %d", code)
	}
	if rt.launched != "" {
		t.Fatalf("must not launch on abort: %q", rt.launched)
	}
}

// A typed name that collides with a live session errors (exit 1, no launch).
func TestRunLaunchPromptCollision(t *testing.T) {
	rt := newFakeRuntime()
	rt.promptValue = "taken"
	rt.blocksReuse["pair-taken"] = true
	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude"}), rt)
	if err != nil || code != 1 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.launched != "" {
		t.Fatalf("must not launch on collision: %q", rt.launched)
	}
}

// Codex forces --no-alt-screen and its watcher gets the final args.
func TestRunLaunchCodexAltScreen(t *testing.T) {
	rt := newFakeRuntime()
	code, err := run(t, baseOpts(LaunchArgs{Agent: "codex", ForcedTag: "cx"}), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.env["PAIR_AGENT_ARGS"] != "--no-alt-screen" {
		t.Fatalf("PAIR_AGENT_ARGS = %q", rt.env["PAIR_AGENT_ARGS"])
	}
	// Codex does not mint a claude session id.
	if rt.env["PAIR_SESSION_ID"] != "" {
		t.Fatalf("codex should not mint a session id: %q", rt.env["PAIR_SESSION_ID"])
	}
	if len(rt.watchers) != 1 || !strings.HasSuffix(rt.watchers[0], "|--no-alt-screen") {
		t.Fatalf("watcher args = %v", rt.watchers)
	}
}

// The tag-restart config picker: a saved config offers reuse; picking "saved
// params + session" composes the resume binding.
func TestRunLaunchTagRestartPickerResume(t *testing.T) {
	rt := newFakeRuntime()
	rt.files["/data/config-cx-codex.json"] = `{"agent":"codex","args":["--search"],"session_id":"CX-9"}`
	rt.agentSessions["codex|CX-9"] = true // native session artifact exists → resumable
	rt.pickFunc = func(header string, options []string) string {
		return options[0] // "use saved params + session"
	}
	code, err := run(t, baseOpts(LaunchArgs{Agent: "codex", ForcedTag: "cx"}), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	// codex resume subcommand LEADS, --no-alt-screen appended idempotently.
	if rt.env["PAIR_AGENT_ARGS"] != "resume CX-9 --search --no-alt-screen" {
		t.Fatalf("PAIR_AGENT_ARGS = %q", rt.env["PAIR_AGENT_ARGS"])
	}
}

// Picking "new" drops the stale config.
func TestRunLaunchTagRestartPickerNew(t *testing.T) {
	rt := newFakeRuntime()
	rt.files["/data/config-work-claude.json"] = `{"agent":"claude","args":["--old"],"session_id":"OLD"}`
	rt.uuids = []string{"NEW-SID"}
	rt.pickFunc = func(header string, options []string) string {
		for _, o := range options {
			if strings.Contains(o, "use new params passed in") {
				return o
			}
		}
		return ""
	}
	opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work", AgentArgs: []string{"--fresh"}})
	code, err := run(t, opts, rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if !contains(rt.removed, "/data/config-work-claude.json") {
		t.Fatalf("new should remove stale config; removed=%v", rt.removed)
	}
	// The freshly-minted config replaces it (mint runs after the picker).
	if cfg := rt.files["/data/config-work-claude.json"]; !strings.Contains(cfg, "NEW-SID") {
		t.Fatalf("expected fresh minted config, got %q", cfg)
	}
}

// Aborting the config picker exits 1.
func TestRunLaunchTagRestartPickerAbort(t *testing.T) {
	rt := newFakeRuntime()
	rt.files["/data/config-cx-codex.json"] = `{"agent":"codex","args":[],"session_id":""}`
	rt.pickFunc = func(header string, options []string) string { return "" }
	code, err := run(t, baseOpts(LaunchArgs{Agent: "codex", ForcedTag: "cx"}), rt)
	if err != nil || code != 1 {
		t.Fatalf("picker abort should exit 1: code=%d err=%v", code, err)
	}
	if rt.launched != "" {
		t.Fatalf("must not launch on picker abort")
	}
}

// An explicit --resume on argv skips the picker and pre-writes the config.
func TestRunLaunchExplicitResumeSkipsPicker(t *testing.T) {
	rt := newFakeRuntime()
	rt.files["/data/config-work-claude.json"] = `{"agent":"claude","args":["--saved"],"session_id":"SAVED"}`
	pickerCalled := false
	rt.pickFunc = func(header string, options []string) string { pickerCalled = true; return options[0] }
	opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work", AgentArgs: []string{"--resume", "EXPLICIT"}})
	code, err := run(t, opts, rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if pickerCalled {
		t.Fatalf("explicit resume must skip the picker")
	}
	// Config pre-written with the explicit id, args stripped of the resume token.
	cfg := rt.files["/data/config-work-claude.json"]
	if !strings.Contains(cfg, `"session_id":"EXPLICIT"`) || strings.Contains(cfg, "--resume") {
		t.Fatalf("config = %q", cfg)
	}
	if rt.env["PAIR_SESSION_ID"] != "EXPLICIT" {
		t.Fatalf("PAIR_SESSION_ID = %q", rt.env["PAIR_SESSION_ID"])
	}
}

// A Runtime query failure (Sessions) exits 1 with a message — no shell to fall
// back to as of M5c (the path is unreachable via OSRuntime, which swallows zellij
// errors, but this pins the defensive branch).
func TestRunLaunchSessionsErrorExits(t *testing.T) {
	rt := newFakeRuntime()
	rt.sessionsErr = errors.New("zellij unreachable")
	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "x"}), rt)
	if err != nil || code != 1 {
		t.Fatalf("code=%d err=%v, want a messaged exit 1", code, err)
	}
	if rt.launched != "" {
		t.Fatal("a Sessions() error must not hand off")
	}
}

// A bare launch from inside a zellij pane (no `continue` slug → not compaction)
// is rejected natively now (#99 M5b) — a nested --session would break. It no
// longer falls back to the shell. (Attach + pick + compaction are native — see
// lifecycle_test.go, pick_test.go, compaction_test.go.)
func TestRunLaunchInPaneRejected(t *testing.T) {
	rt := newFakeRuntime()
	rt.inPane = true
	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude"}), rt)
	if err != nil {
		t.Fatalf("in-pane bare launch should be handled natively, got err %v", err)
	}
	if code != 1 {
		t.Fatalf("in-pane bare launch should exit 1, got %d", code)
	}
	if rt.launched != "" || len(rt.attached) != 0 {
		t.Fatal("in-pane rejection must not hand off")
	}
}

// A missing agent binary errors before any session work.
func TestRunLaunchAgentMissing(t *testing.T) {
	rt := newFakeRuntime()
	rt.commandMissing["claude"] = true
	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "x"}), rt)
	if err != nil || code != 1 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.launched != "" {
		t.Fatalf("must not launch with missing agent")
	}
}

// `pair resume <tag>` (agent unset) infers the paired agent from disk.
func TestRunLaunchResumeInfersAgent(t *testing.T) {
	rt := newFakeRuntime()
	rt.inferAgent["oldcx"] = "codex"
	code, err := run(t, baseOpts(LaunchArgs{ForcedTag: "oldcx"}), rt) // Agent: "" → infer
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.env["PAIR_AGENT"] != "codex" {
		t.Fatalf("inferred agent = %q, want codex", rt.env["PAIR_AGENT"])
	}
	if rt.files["/data/agent-oldcx"] != "codex\n" {
		t.Fatalf("agent record = %q", rt.files["/data/agent-oldcx"])
	}
}

// With nothing on disk to infer from, the agent defaults to claude.
func TestRunLaunchResumeDefaultsClaude(t *testing.T) {
	rt := newFakeRuntime()
	rt.uuids = []string{"SID"}
	code, err := run(t, baseOpts(LaunchArgs{ForcedTag: "brand-new"}), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.env["PAIR_AGENT"] != "claude" {
		t.Fatalf("default agent = %q, want claude", rt.env["PAIR_AGENT"])
	}
}

// A zellij name-length rejection (#54) aborts with exit 1 before the handoff.
func TestRunLaunchProbeTooLong(t *testing.T) {
	rt := newFakeRuntime()
	rt.uuids = []string{"S"}
	rt.probeErr = errors.New("session name too long")
	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "waytoolongtag"}), rt)
	if err != nil || code != 1 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.launched != "" {
		t.Fatalf("must not launch when the name probe rejects: %q", rt.launched)
	}
}

// A live session unexpectedly occupying the name at the pre-handoff guard
// (#67 TOCTOU) aborts with exit 1 rather than colliding in --new-session.
func TestRunLaunchPreHandoffCollision(t *testing.T) {
	rt := newFakeRuntime()
	rt.uuids = []string{"S"}
	rt.blocksReuse["pair-bugfix"] = true // forced create → no prompt collision check
	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "bugfix"}), rt)
	if err != nil || code != 1 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.launched != "" {
		t.Fatalf("must not launch when the name is occupied at handoff: %q", rt.launched)
	}
}

func contains(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

var ansiEscapeRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSI mimics fzf --ansi: the picker rows are ANSI-colored for display, but
// fzf returns the plain text (which buildPickRows keys byPlain on).
func stripANSI(s string) string { return ansiEscapeRE.ReplaceAllString(s, "") }

// Agent-scoped CLI-args guard (#107): `pair codex -- <codex-only args>` with
// history present routes to the session picker (DecideLaunch ignores the agent).
// Picking an existing CLAUDE tag is resume-by-name → the agent re-infers to
// claude; the codex-intended CLI args must NOT ride along onto claude (which
// would choke on them at launch). The guard drops them on agent mismatch.
func TestRunLaunchPickInferredAgentMustNotInheritCliArgs(t *testing.T) {
	rt := newFakeRuntime()
	// A historical claude tag (base tag for cwd /home/u/work is "work").
	rt.historical = []HistoricalTag{{Tag: "work", MTime: time.Unix(1_699_000_000, 0)}}
	rt.inferAgent["work"] = "claude"
	rt.uuids = []string{"SID"}
	rt.pickFunc = func(header string, options []string) string {
		return stripANSI(options[0]) // pick the historical claude row
	}
	opts := baseOpts(LaunchArgs{Agent: "codex", AgentArgs: []string{"--sandbox", "danger-full-access"}})
	code, err := run(t, opts, rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.env["PAIR_AGENT"] != "claude" {
		t.Fatalf("precondition: resume-by-name should infer claude, got %q", rt.env["PAIR_AGENT"])
	}
	// The codex-only args must NOT reach claude.
	if strings.Contains(rt.env["PAIR_AGENT_ARGS"], "--sandbox") {
		t.Fatalf("claude inherited codex CLI args: PAIR_AGENT_ARGS=%q", rt.env["PAIR_AGENT_ARGS"])
	}
}
