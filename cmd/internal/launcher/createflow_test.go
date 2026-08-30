package launcher

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/commitoutcome"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"

	"github.com/xianxu/pair/cmd/internal/readiness"
)

// fakeRuntime is the in-memory create-flow seam for the RunLaunch loop tests.
// Canned inputs drive decisions; recorded outputs assert the effect sequence.
//
// mu guards the recorded-output maps and slices. EVERY accessor of f.files
// takes it, not only the one a race detector happened to flag: a concurrent
// map write is an unrecoverable Go fatal error rather than a test failure, so
// partial locking buys nothing. It is needed because
// startAgentDefaultPersistence (createflow.go:562) writes through the seam
// from its own goroutine while the main flow is still writing config, so the
// fake is genuinely concurrent even though production hits real files.
type fakeRuntime struct {
	mu                  sync.Mutex
	inPane              bool
	sessions            []Session
	historical          []HistoricalTag
	blocksReuse         map[string]bool // session -> live-blocks (default false)
	commandMissing      map[string]bool // name -> absent (default: everything exists)
	files               map[string]string
	ledger              map[string][]LedgerEntry
	sessionIndex        SessionNameIndex
	sessionIndexErr     error
	agentSessions       map[string]bool   // "agent|sid" -> native artifact exists
	establishedSessions map[string]string // "scope|tag|agent" -> established native id
	bindingStatuses     map[string]sessioninventory.BindingStatus
	uuids               []string // MintUUID pops these in order
	promptValue         string
	promptOK            bool
	maxSessionNameBytes int
	probeErr            error
	appendLedgerErr     error
	prepareLaunchErr    error
	preparedLaunches    []string
	appendIndexErr      error
	threadClaimErr      error
	threadClaims        []string
	writeFailAt         string
	inferAgent          map[string]string // tag -> paired agent (for `resume <tag>`)
	pickFunc            func(header string, options []string) string
	listRows            []ListRow // ListSessions rows (for `pair list`)
	listErr             error
	sessionsErr         error // Sessions() error (defensive exit-1 path)
	liveLayouts         map[string]LayoutMode
	liveLayoutErr       error
	confirmLayout       bool
	layoutPrompts       []string
	deleteErr           error
	keepDeletedLive     bool
	renameFailAt        string      // Rename returns an error when src == this (rollback test)
	renamed             [][2]string // {src,dst} per successful Rename
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
	launchErr      error                    // LaunchSession error
	quitMarkers    map[string]bool          // session -> Alt+x quit marker (read-cleared)
	restartMarkers map[string]RestartMarker // session -> restart marker (peek + take-once)
	cmuxOwned      map[string]bool          // tag -> PairOwnsCmuxWorkspace
	readyRecords   map[string]bool          // "tag|agent|session|nonce" -> ready
	readyPIDs      map[int]bool
	readyErr       error

	// recorded
	env             map[string]string
	launched        string // last session name handed to LaunchSession
	launchLayout    string
	launchCode      int
	launchCount     int // number of create handoffs (restart-loop iterations)
	defaultReads    int
	watchers        []string // "agent|tag|cwd|args"
	pollers         []string // "tag|agent"
	cmux            []string // "tag|title"
	ttyRecorded     []string
	titles          []string
	removed         []string
	family          []string
	devRebuilt      bool
	proofMigrations int
	attached        []string   // sessions handed to AttachSession
	deleted         []string   // sessions handed to DeleteSession
	reaped          []string   // tags handed to ReapNvim
	swept           [][]string // liveTags per SweepOrphanNvim call
	parkPrompts     []string   // sessions prompted via ConfirmParkNudge
	parked          []string   // "tag|agent|move" per ParkScrollback
	killedPollers   []string   // tags handed to KillTitlePoller
	cmuxCleared     int        // ClearCmuxOwner calls
}

func (f *fakeRuntime) StartProofMigration() { f.proofMigrations++ }

func TestRunLaunchStartsProofMigrationBeforeInteractiveWork(t *testing.T) {
	runtime := newFakeRuntime()
	runtime.sessionsErr = errors.New("stop after startup")
	_, _ = RunLaunch(LaunchOptions{Env: Env{Now: time.Now()}}, runtime, &bytes.Buffer{})
	if runtime.proofMigrations != 1 {
		t.Fatalf("proof migrations=%d", runtime.proofMigrations)
	}
}

func (f *fakeRuntime) EnsureThreadAddress(scope RepoScope, tag string, couchOwned bool) error {
	f.threadClaims = append(f.threadClaims, fmt.Sprintf("%s|%s|%t", scope.Key, tag, couchOwned))
	return f.threadClaimErr
}

func newFakeRuntime() *fakeRuntime {
	return &fakeRuntime{
		blocksReuse:         map[string]bool{},
		commandMissing:      map[string]bool{},
		files:               map[string]string{},
		ledger:              map[string][]LedgerEntry{},
		agentSessions:       map[string]bool{},
		establishedSessions: map[string]string{},
		bindingStatuses:     map[string]sessioninventory.BindingStatus{},
		inferAgent:          map[string]string{},
		promptOK:            true,
		env:                 map[string]string{},
		quitMarkers:         map[string]bool{},
		restartMarkers:      map[string]RestartMarker{},
		cmuxOwned:           map[string]bool{},
		liveLayouts:         map[string]LayoutMode{},
		readyRecords:        map[string]bool{},
		readyPIDs:           map[int]bool{},
	}
}

// ZellijOps
func (f *fakeRuntime) Sessions() ([]Session, error)           { return f.sessions, f.sessionsErr }
func (f *fakeRuntime) SessionBlocksReuse(session string) bool { return f.blocksReuse[session] }
func (f *fakeRuntime) ProbeSessionName(session string) error {
	if f.probeErr != nil {
		return f.probeErr
	}
	if f.maxSessionNameBytes > 0 && len(session) > f.maxSessionNameBytes {
		return fmt.Errorf("session name too long")
	}
	return nil
}
func (f *fakeRuntime) LaunchSession(session, configDir, layout string) (int, error) {
	f.launched = session
	f.launchLayout = layout
	f.launchCount++
	return f.launchCode, f.launchErr
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
func (f *fakeRuntime) ConfirmLayoutChange(tag string, from, to LayoutMode) bool {
	f.layoutPrompts = append(f.layoutPrompts, fmt.Sprintf("%s|%s|%s", tag, from, to))
	return f.confirmLayout
}

func (f *fakeRuntime) ProbeLiveLayout(session string) (LayoutMode, error) {
	if f.liveLayoutErr != nil {
		return "", f.liveLayoutErr
	}
	if mode, ok := f.liveLayouts[session]; ok {
		return mode, nil
	}
	return Layout2, nil
}

// ProcOps
func (f *fakeRuntime) SpawnSessionWatcher(agent, tag, scopeKey, cwd, repoRoot, repoName string, launchOrdinal uint64, agentArgs []string) {
	f.watchers = append(f.watchers, fmt.Sprintf("%s|%s|%s|%s|%s|%s|%d|%s", agent, tag, scopeKey, cwd, repoRoot, repoName, launchOrdinal, strings.Join(agentArgs, " ")))
}
func (f *fakeRuntime) SpawnTitlePoller(tag, agent, session string) {
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
func (f *fakeRuntime) EstablishedSessionID(scopeKey, tag, agent string) (string, sessioninventory.BindingStatus) {
	if status, ok := f.bindingStatuses[tag+"|"+agent]; ok {
		return f.establishedSessions[tag+"|"+agent], status
	}
	if sid := f.establishedSessions[tag+"|"+agent]; sid != "" {
		return sid, sessioninventory.BindingEstablished
	}
	return "", sessioninventory.BindingUnbound
}
func (f *fakeRuntime) InferAgent(tag string) string {
	if latest, ok := LatestLedgerEntry(f.ledger[tag]); ok && latest.Agent != "" {
		return latest.Agent
	}
	return f.inferAgent[tag]
}
func (f *fakeRuntime) ReadLedger(tag string) ([]LedgerEntry, error) {
	return append([]LedgerEntry(nil), f.ledger[tag]...), nil
}
func (f *fakeRuntime) AppendLedger(tag string, entry LedgerEntry) error {
	if f.appendLedgerErr != nil && commitoutcome.Of(f.appendLedgerErr) != commitoutcome.Committed {
		return f.appendLedgerErr
	}
	f.ledger[tag] = append(f.ledger[tag], entry)
	return f.appendLedgerErr
}
func (f *fakeRuntime) PrepareSessionLaunch(scopeKey, tag, agent, resumeNativeID string) (uint64, error) {
	f.preparedLaunches = append(f.preparedLaunches, strings.Join([]string{scopeKey, tag, agent, resumeNativeID}, "|"))
	if f.prepareLaunchErr != nil {
		if commitoutcome.Of(f.prepareLaunchErr) != commitoutcome.Committed {
			return 0, f.prepareLaunchErr
		}
		return uint64(len(f.preparedLaunches)), f.prepareLaunchErr
	}
	return uint64(len(f.preparedLaunches)), nil
}
func (f *fakeRuntime) ReadSessionNameIndex() (SessionNameIndex, error) {
	return f.sessionIndex, f.sessionIndexErr
}
func (f *fakeRuntime) AppendSessionNameIndex(entry SessionNameEntry) error {
	if f.appendIndexErr != nil {
		return f.appendIndexErr
	}
	f.sessionIndex.Entries = append(f.sessionIndex.Entries, entry)
	return nil
}

func TestAssignLaunchSessionNamesRefusesUnreadableIndex(t *testing.T) {
	rt := newFakeRuntime()
	rt.sessionIndexErr = errors.New("index unreadable")
	var stderr bytes.Buffer
	_, _, _, ok := assignLaunchSessionNames(rt, nil, "/repo", "/global", LaunchArgs{ForcedTag: "work"}, "work", &stderr)
	if ok {
		t.Fatal("assignLaunchSessionNames accepted an unreadable session binding index")
	}
	if !strings.Contains(stderr.String(), "index unreadable") {
		t.Fatalf("stderr = %q, want session binding index error", stderr.String())
	}
}

func (f *fakeRuntime) RemoveReadyRecord(tag, agent string) {
	f.Remove(AgentReadyPath("/data", tag, agent))
}
func (f *fakeRuntime) MintLaunchNonce() string { return "fake-launch-nonce" }
func (f *fakeRuntime) WaitReadyRecord(expect ReadyExpectation, timeout time.Duration) (readiness.ReadyRecord, error) {
	if f.readyErr != nil {
		return readiness.ReadyRecord{}, f.readyErr
	}
	return readiness.ReadyRecord{
		Tag:     expect.Tag,
		Agent:   expect.Agent,
		Session: expect.Session,
		Nonce:   expect.Nonce,
		PID:     123,
	}, nil
}
func (f *fakeRuntime) PIDAlive(pid int) bool { return f.readyPIDs[pid] }
func (f *fakeRuntime) ReadAgentDefault(agent string) (AgentDefault, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.defaultReads++
	raw, ok := f.files[AgentDefaultPath("/data", agent)]
	if !ok {
		return AgentDefault{}, false
	}
	d, err := ParseAgentDefault(agent, raw)
	return d, err == nil
}

func TestRequiredNativeResumeBindingAtLaunch(t *testing.T) {
	tests := []struct {
		name   string
		status sessioninventory.BindingStatus
		actual string
	}{
		{name: "provisional", status: sessioninventory.BindingProvisional},
		{name: "ambiguous", status: sessioninventory.BindingAmbiguous},
		{name: "missing", status: sessioninventory.BindingUnbound},
		{name: "established without root", status: sessioninventory.BindingEstablished},
		{name: "different established root", status: sessioninventory.BindingEstablished, actual: "native-root-2"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rt := newFakeRuntime()
			rt.bindingStatuses["work|codex"] = test.status
			rt.establishedSessions["work|codex"] = test.actual
			args := LaunchArgs{
				Agent: "codex", AgentExplicit: true, ForcedTag: "work",
				AgentArgs: []string{"--sandbox", "workspace-write"}, AgentArgsExplicit: true, AgentArgsFromCouch: true,
				ResumeRequired: true, RequiredSessionID: "native-root-1",
			}
			opts := baseOpts(args)
			scope, err := ResolveRepoScope(opts.Env.Cwd)
			if err != nil {
				t.Fatal(err)
			}
			opts.Env.CouchThreadScope, opts.Env.CouchThreadTag = scope.Key, "work"
			code, err := run(t, opts, rt)
			if err != nil || code != 1 {
				t.Fatalf("RunLaunch = %d, %v", code, err)
			}
			if rt.launchCount != 0 || len(rt.preparedLaunches) != 0 || len(rt.watchers) != 0 || len(rt.ledger["work"]) != 0 || rt.defaultReads != 0 {
				t.Fatalf("refusal effects: launches=%d prepared=%v watchers=%v ledger=%v defaultReads=%d", rt.launchCount, rt.preparedLaunches, rt.watchers, rt.ledger["work"], rt.defaultReads)
			}
		})
	}
}

func TestRequiredNativeResumeBindingLaunchesExactRootWithoutDefaults(t *testing.T) {
	rt := newFakeRuntime()
	rt.bindingStatuses["work|codex"] = sessioninventory.BindingEstablished
	rt.establishedSessions["work|codex"] = "native-root-1"
	args := LaunchArgs{
		Agent: "codex", AgentExplicit: true, ForcedTag: "work",
		AgentArgs: []string{"--sandbox", "workspace-write"}, AgentArgsExplicit: true, AgentArgsFromCouch: true,
		ResumeRequired: true, RequiredSessionID: "native-root-1",
	}
	opts := baseOpts(args)
	scope, err := ResolveRepoScope(opts.Env.Cwd)
	if err != nil {
		t.Fatal(err)
	}
	opts.Env.CouchThreadScope, opts.Env.CouchThreadTag = scope.Key, "work"
	code, err := run(t, opts, rt)
	if err != nil || code != 0 {
		t.Fatalf("RunLaunch = %d, %v", code, err)
	}
	if rt.launchCount != 1 || rt.defaultReads != 0 || len(rt.preparedLaunches) != 1 ||
		!strings.HasSuffix(rt.preparedLaunches[0], "|work|codex|native-root-1") {
		t.Fatalf("launch effects: count=%d defaults=%d prepared=%v", rt.launchCount, rt.defaultReads, rt.preparedLaunches)
	}
	if rt.env["PAIR_SESSION_ID"] != "native-root-1" || rt.env["PAIR_AGENT_ARGS"] != "resume native-root-1 --sandbox workspace-write --no-alt-screen" {
		t.Fatalf("resume env: id=%q args=%q", rt.env["PAIR_SESSION_ID"], rt.env["PAIR_AGENT_ARGS"])
	}
}

func TestRequiredNativeResumeRefusesAttachRace(t *testing.T) {
	rt := newFakeRuntime()
	rt.sessions = []Session{{Name: "📁work", Tag: "work", RepoName: "work", Agent: "codex", State: SessionDetached}}
	args := LaunchArgs{
		Agent: "codex", AgentExplicit: true, ForcedTag: "work",
		AgentArgs: []string{}, AgentArgsExplicit: true, AgentArgsFromCouch: true,
		ResumeRequired: true, RequiredSessionID: "native-root-1",
	}
	code, err := run(t, baseOpts(args), rt)
	if err != nil || code != 1 || len(rt.attached) != 0 || rt.launchCount != 0 {
		t.Fatalf("attach race = code %d err %v attached=%v launchCount=%d", code, err, rt.attached, rt.launchCount)
	}
}
func (f *fakeRuntime) WriteAgentDefault(agent string, args []string) error {
	raw, err := BuildAgentDefault(agent, args)
	if err != nil {
		return err
	}
	return f.WriteAtomic(AgentDefaultPath("/data", agent), raw)
}

// FSOps
func (f *fakeRuntime) ReadFile(path string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if v, ok := f.files[path]; ok {
		return v, nil
	}
	return "", errors.New("not found")
}
func (f *fakeRuntime) WriteAtomic(path, data string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if path == f.writeFailAt {
		return errors.New("write failed (fake)")
	}
	f.files[path] = data
	return nil
}
func (f *fakeRuntime) Remove(path string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.removed = append(f.removed, path)
	delete(f.files, path)
}
func (f *fakeRuntime) FileSize(path string) (int64, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.files[path]
	return int64(len(v)), ok
}
func (f *fakeRuntime) Touch(path string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.files[path]; !ok {
		f.files[path] = ""
	}
	return nil
}
func (f *fakeRuntime) Rename(src, dst string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
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
func (f *fakeRuntime) ReadDir(path string) ([]string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	prefix := strings.TrimSuffix(path, "/") + "/"
	var out []string
	for p := range f.files {
		if strings.HasPrefix(p, prefix) {
			out = append(out, strings.TrimPrefix(p, prefix))
		}
	}
	if len(out) == 0 {
		return nil, errors.New("not found")
	}
	return out, nil
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
func (f *fakeRuntime) DeleteSession(session string) error {
	f.deleted = append(f.deleted, session)
	if f.deleteErr != nil {
		return f.deleteErr
	}
	delete(f.blocksReuse, session) // the name is free for a restart re-create
	if !f.keepDeletedLive {
		kept := f.sessions[:0]
		for _, existing := range f.sessions {
			if existing.Name != session {
				kept = append(kept, existing)
			}
		}
		f.sessions = kept
	}
	return nil
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

func TestCreateClaimsThreadAddressBeforeDurableWrites(t *testing.T) {
	rt := newFakeRuntime()
	rt.threadClaimErr = errors.New("occupied")
	code, err := run(t, baseOpts(LaunchArgs{ForcedTag: "work"}), rt)
	if err != nil {
		t.Fatal(err)
	}
	if code != 1 || len(rt.threadClaims) != 1 {
		t.Fatalf("code=%d claims=%v", code, rt.threadClaims)
	}
	if len(rt.ledger) != 0 || len(rt.files) != 0 || rt.launched != "" {
		t.Fatalf("claim refusal wrote durable state: ledger=%v files=%v launched=%q", rt.ledger, rt.files, rt.launched)
	}
}

func TestCreateIdentifiesExactCouchOwnedClaim(t *testing.T) {
	rt := newFakeRuntime()
	opts := baseOpts(LaunchArgs{ForcedTag: "work"})
	scope, err := ResolveRepoScope(opts.Env.Cwd)
	if err != nil {
		t.Fatal(err)
	}
	opts.Env.CouchThreadScope = scope.Key
	opts.Env.CouchThreadTag = "work"
	if code, err := run(t, opts, rt); err != nil || code != 0 {
		t.Fatalf("run = %d, %v", code, err)
	}
	if got := rt.threadClaims; len(got) != 1 || !strings.HasSuffix(got[0], "|work|true") {
		t.Fatalf("claims = %v", got)
	}
}

func TestRunLaunchIgnoresCouchStateForExactTag(t *testing.T) {
	for _, tag := range []string{"compiler-fix", "couch-3dcfba1308775e82"} {
		t.Run(tag, func(t *testing.T) {
			rt := newFakeRuntime()
			rt.uuids = []string{"SID"}

			code, err := run(t, baseOpts(LaunchArgs{ForcedTag: tag}), rt)
			if err != nil || code != 0 {
				t.Fatalf("run = %d, %v; exact Pair tag must not depend on Couch state", code, err)
			}
			if rt.env["PAIR_TAG"] != tag || rt.launched == "" {
				t.Fatalf("create handoff = %q with PAIR_TAG=%q, want unchanged exact tag %q", rt.launched, rt.env["PAIR_TAG"], tag)
			}
		})
	}
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
// provisional baseline + agent record written, sidecars spawned, session handed off.
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
	if rt.launched != "📁work-bugfix" {
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
	// The STARTUP pane title (#133): the agent name and nothing more. zellij shows
	// "<session name> | <pane title>" and the session half is already 📁work-bugfix,
	// so a cwd here would name the folder twice. This is the title the tab carries
	// from launch until the poller's first pass, so it needs its own assertion —
	// the poller's steady-state title is a different code path.
	if got := rt.env["PAIR_PANE_TITLE"]; got != "claude" {
		t.Fatalf("PAIR_PANE_TITLE = %q, want bare agent name", got)
	}
	// And the pre-abbreviated cwd export is gone with it — nothing consumes one.
	if got, ok := rt.env["PAIR_PANE_CWD"]; ok {
		t.Fatalf("PAIR_PANE_CWD should no longer be exported, got %q", got)
	}
	// A freshly minted native ID is invocation authority, not recovery state.
	if cfg := rt.files["/data/config-bugfix-claude.json"]; cfg != "" {
		t.Fatalf("provisional launch wrote config = %q", cfg)
	}
	if rt.files["/data/agent-bugfix"] != "claude\n" {
		t.Fatalf("agent record = %q", rt.files["/data/agent-bugfix"])
	}
	if got := rt.preparedLaunches; len(got) != 1 || !strings.Contains(got[0], "|bugfix|claude|") {
		t.Fatalf("prepared launches = %v", got)
	}
	ledger := rt.ledger["bugfix"]
	if len(ledger) != 1 || ledger[0].Agent != "claude" || ledger[0].SessionID != "MINTED-1" {
		t.Fatalf("ledger = %+v, want claude/MINTED-1", ledger)
	}
	if got := rt.watchers; len(got) != 1 || !strings.HasPrefix(got[0], "claude|bugfix|") || !strings.Contains(got[0], "|1|") {
		t.Fatalf("watchers = %v", got)
	}
	if len(rt.pollers) != 1 || rt.pollers[0] != "bugfix|claude" {
		t.Fatalf("pollers = %v", rt.pollers)
	}
	if len(rt.titles) != 1 || len(rt.ttyRecorded) != 1 || len(rt.cmux) != 1 {
		t.Fatalf("title/tty/cmux effects missing: %v %v %v", rt.titles, rt.ttyRecorded, rt.cmux)
	}
}

func TestRunLaunchAbortsBeforeHandoffWhenNativeLaunchBoundaryFails(t *testing.T) {
	rt := newFakeRuntime()
	rt.prepareLaunchErr = errors.New("baseline unavailable")
	code, err := run(t, baseOpts(LaunchArgs{Agent: "codex", ForcedTag: "bugfix"}), rt)
	if err != nil {
		t.Fatal(err)
	}
	if code != 1 || rt.launched != "" || len(rt.watchers) != 0 || len(rt.pollers) != 0 {
		t.Fatalf("code=%d launched=%q watchers=%v pollers=%v", code, rt.launched, rt.watchers, rt.pollers)
	}
	if len(rt.preparedLaunches) != 1 {
		t.Fatalf("prepared launches = %v, want one attempted boundary", rt.preparedLaunches)
	}
	if _, ok := rt.env["PAIR_LAUNCH_ORDINAL"]; ok {
		t.Fatalf("PAIR_LAUNCH_ORDINAL exported after failed boundary: %q", rt.env["PAIR_LAUNCH_ORDINAL"])
	}
}

func TestRunLaunchForcedCreateUsesScopedSessionName(t *testing.T) {
	rt := newFakeRuntime()
	code, err := run(t, baseOpts(LaunchArgs{Agent: "codex", ForcedTag: "bugfix"}), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.launched != "📁work-bugfix" {
		t.Fatalf("launched = %q", rt.launched)
	}
	if rt.env["PAIR_TAG"] != "bugfix" {
		t.Fatalf("PAIR_TAG = %q", rt.env["PAIR_TAG"])
	}
	if len(rt.sessionIndex.Entries) != 1 {
		t.Fatalf("session index = %#v, want one entry", rt.sessionIndex)
	}
	entry := rt.sessionIndex.Entries[0]
	if entry.SessionName != "📁work-bugfix" || entry.Tag != "bugfix" || entry.RepoName != "work" {
		t.Fatalf("session index entry = %#v", entry)
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
	if rt.launched != "📁work-myproj" || rt.env["PAIR_TAG"] != "myproj" {
		t.Fatalf("launched=%q tag=%q", rt.launched, rt.env["PAIR_TAG"])
	}
	if len(rt.sessionIndex.Entries) != 1 {
		t.Fatalf("session index = %#v, want one entry", rt.sessionIndex)
	}
	entry := rt.sessionIndex.Entries[0]
	if entry.SessionName != "📁work-myproj" || entry.Tag != "myproj" || entry.RepoName != "work" {
		t.Fatalf("session index entry = %#v", entry)
	}
}

func TestRunLaunchBareIgnoresOtherRepoIndexedSessions(t *testing.T) {
	rt := newFakeRuntime()
	otherScope := mustScope(t, "/other/work")
	rt.sessions = []Session{{Name: "📁work", State: SessionDetached}}
	rt.sessionIndex = SessionNameIndex{Entries: []SessionNameEntry{{
		SessionName: "📁work",
		ScopeKey:    otherScope.Key,
		RepoRoot:    otherScope.Root,
		RepoName:    otherScope.DisplayName,
		Tag:         "work",
	}}}
	opts := baseOpts(LaunchArgs{Agent: "codex"})
	code, err := run(t, opts, rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.launched != "📁work-2" {
		t.Fatalf("launched = %q, want current repo disambiguated from indexed other repo", rt.launched)
	}
}

func TestRunLaunchBareIgnoresUnindexedLiveSessions(t *testing.T) {
	rt := newFakeRuntime()
	rt.sessions = []Session{{Name: "pair-work", State: SessionDetached}}
	rt.pickFunc = func(header string, options []string) string {
		t.Fatalf("picker should not show unindexed live sessions: %q", options)
		return ""
	}
	opts := baseOpts(LaunchArgs{Agent: "codex"})
	code, err := run(t, opts, rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if len(rt.attached) != 0 {
		t.Fatalf("attached = %v, want no unindexed attach", rt.attached)
	}
	if rt.launched != "📁work" {
		t.Fatalf("launched = %q, want current-scope create", rt.launched)
	}
}

func TestRunLaunchBareAttachesLegacyLiveSessionWithCurrentRepoPaneEvidence(t *testing.T) {
	rt := newFakeRuntime()
	rt.sessions = []Session{{Name: "pair-work", State: SessionDetached}}
	rt.inferAgent["work"] = "codex"
	rt.attachCode = 0
	rt.files["/global/pane-work-codex.json"] = `{"cwd":"/home/u/work","cwd_display":"~/work"}`
	rt.pickFunc = func(header string, options []string) string {
		for _, option := range options {
			plain := stripANSI(option)
			if plain == "work/work  codex  (detached)" {
				return plain
			}
		}
		t.Fatalf("picker options = %q, want legacy current-repo live row", options)
		return ""
	}
	opts := baseOpts(LaunchArgs{Agent: "claude"})
	opts.GlobalDataDir = "/global"
	code, err := run(t, opts, rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if !reflect.DeepEqual(rt.attached, []string{"pair-work"}) {
		t.Fatalf("attached = %v, want legacy live pair-work", rt.attached)
	}
	if len(rt.pollers) != 1 || rt.pollers[0] != "work|codex" {
		t.Fatalf("pollers = %v, want codex inferred from legacy tag", rt.pollers)
	}
}

func TestRunLaunchResumePublicSessionNameResolvesThroughIndex(t *testing.T) {
	scope := mustScope(t, "/home/u/work")
	rt := newFakeRuntime()
	rt.sessionIndex = SessionNameIndex{Entries: []SessionNameEntry{{
		SessionName: "📁work-demo",
		ScopeKey:    scope.Key,
		RepoRoot:    scope.Root,
		RepoName:    scope.DisplayName,
		Tag:         "demo",
	}}}
	rt.sessions = []Session{{Name: "📁work-demo", State: SessionDetached}}
	rt.inferAgent["demo"] = "codex"
	rt.attachCode = 0

	code, err := run(t, baseOpts(LaunchArgs{ForcedTag: "📁work-demo"}), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if !reflect.DeepEqual(rt.attached, []string{"📁work-demo"}) {
		t.Fatalf("attached = %v, want indexed public session name", rt.attached)
	}
	if len(rt.pollers) != 1 || rt.pollers[0] != "demo|codex" {
		t.Fatalf("pollers = %v, want tag/agent resolved through index", rt.pollers)
	}
}

func TestRunLaunchResumeUnindexedPublicSessionNameRefuses(t *testing.T) {
	rt := newFakeRuntime()
	var stderr bytes.Buffer
	code, err := RunLaunch(baseOpts(LaunchArgs{ForcedTag: "📁work-demo"}), rt, &stderr)
	if err != nil || code != 1 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if !strings.Contains(stderr.String(), "session name with no ledger entry") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if rt.launched != "" || len(rt.attached) != 0 {
		t.Fatalf("must not hand off an unindexed public session name: launched=%q attached=%v", rt.launched, rt.attached)
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
	rt.blocksReuse["📁work-taken"] = true
	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude"}), rt)
	if err != nil || code != 1 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.launched != "" {
		t.Fatalf("must not launch on collision: %q", rt.launched)
	}
}

func TestRunLaunchPromptRefusesNameOverDiscoveredBudget(t *testing.T) {
	rt := newFakeRuntime()
	rt.promptValue = "feature-with-many-words"
	rt.maxSessionNameBytes = 20

	var stderr bytes.Buffer
	code, err := RunLaunch(baseOpts(LaunchArgs{Agent: "claude"}), rt, &stderr)
	if err != nil || code != 1 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.launched != "" || len(rt.sessionIndex.Entries) != 0 || len(rt.ledger[rt.promptValue]) != 0 {
		t.Fatalf("over-budget prompt must not launch or persist: launched=%q index=%+v ledger=%+v", rt.launched, rt.sessionIndex, rt.ledger)
	}
	for _, want := range []string{"name '📁work-feature-with-many-words' needs", "zellij allows 20"} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("stderr missing %q: %s", want, stderr.String())
		}
	}
}

func TestRunLaunchPromptAcceptsNameUnderDiscoveredBudget(t *testing.T) {
	rt := newFakeRuntime()
	rt.promptValue = "short"
	rt.maxSessionNameBytes = 20

	var stderr bytes.Buffer
	code, err := RunLaunch(baseOpts(LaunchArgs{Agent: "claude"}), rt, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if rt.launched != "📁work-short" {
		t.Fatalf("launched = %q, want composed prompt name under budget", rt.launched)
	}
}

func TestRunLaunchFailedPreflightDoesNotAppendLedgerOrSessionIndex(t *testing.T) {
	rt := newFakeRuntime()
	rt.uuids = []string{"SID"}
	rt.probeErr = errors.New("too long")
	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "bugfix"}), rt)
	if err != nil || code != 1 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if len(rt.ledger["bugfix"]) != 0 {
		t.Fatalf("ledger should not be appended before preflight succeeds: %+v", rt.ledger["bugfix"])
	}
	if len(rt.sessionIndex.Entries) != 0 {
		t.Fatalf("session index should not be appended before preflight succeeds: %+v", rt.sessionIndex)
	}
	if len(rt.files) != 0 {
		t.Fatalf("preflight failure should not write sidecars: %+v", rt.files)
	}
	if len(rt.watchers) != 0 || len(rt.pollers) != 0 || len(rt.titles) != 0 || len(rt.cmux) != 0 || rt.devRebuilt {
		t.Fatalf("preflight failure started side effects: watchers=%v pollers=%v titles=%v cmux=%v dev=%v", rt.watchers, rt.pollers, rt.titles, rt.cmux, rt.devRebuilt)
	}
	if len(rt.env) != 1 { // PATH is set at RunLaunch entry.
		t.Fatalf("preflight failure should only set PATH env, got %+v", rt.env)
	}
}

func TestRunLaunchLedgerAppendFailureAbortsBeforeHandoff(t *testing.T) {
	rt := newFakeRuntime()
	rt.uuids = []string{"SID"}
	rt.appendLedgerErr = errors.New("ledger write failed")
	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "bugfix"}), rt)
	if err != nil || code != 1 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.launched != "" || rt.launchCount != 0 {
		t.Fatalf("must not launch when ledger append fails: launched=%q count=%d", rt.launched, rt.launchCount)
	}
	if len(rt.ledger["bugfix"]) != 0 {
		t.Fatalf("ledger append failure should not record row: %+v", rt.ledger["bugfix"])
	}
	if len(rt.files) != 0 {
		t.Fatalf("ledger append failure should not write sidecars: %+v", rt.files)
	}
	if len(rt.watchers) != 0 || len(rt.pollers) != 0 || len(rt.titles) != 0 || len(rt.cmux) != 0 || rt.devRebuilt {
		t.Fatalf("ledger append failure started side effects: watchers=%v pollers=%v titles=%v cmux=%v dev=%v", rt.watchers, rt.pollers, rt.titles, rt.cmux, rt.devRebuilt)
	}
}

func TestRunLaunchContinuesAfterCommittedCleanupWarnings(t *testing.T) {
	for _, stage := range []string{"compatibility-ledger", "native-launch"} {
		t.Run(stage, func(t *testing.T) {
			rt := newFakeRuntime()
			rt.uuids = []string{"SID"}
			warning := &commitoutcome.Error{Outcome: commitoutcome.Committed, Err: errors.New("unlock failed")}
			if stage == "compatibility-ledger" {
				rt.appendLedgerErr = warning
			} else {
				rt.prepareLaunchErr = warning
			}
			code, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "bugfix"}), rt)
			if err != nil || code != 0 || rt.launchCount != 1 || len(rt.watchers) != 1 {
				t.Fatalf("code=%d err=%v launchCount=%d watchers=%v", code, err, rt.launchCount, rt.watchers)
			}
		})
	}
}

func TestRunLaunchSessionIndexAppendFailureAbortsBeforeHandoff(t *testing.T) {
	rt := newFakeRuntime()
	rt.uuids = []string{"SID"}
	rt.appendIndexErr = errors.New("index write failed")
	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "bugfix"}), rt)
	if err != nil || code != 1 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.launched != "" || rt.launchCount != 0 {
		t.Fatalf("must not launch when session index append fails: launched=%q count=%d", rt.launched, rt.launchCount)
	}
	if len(rt.sessionIndex.Entries) != 0 {
		t.Fatalf("session index append failure should not record entry: %+v", rt.sessionIndex)
	}
	if len(rt.ledger["bugfix"]) != 0 {
		t.Fatalf("session index append failure should not leave a false ledger row: %+v", rt.ledger["bugfix"])
	}
	if len(rt.files) != 0 {
		t.Fatalf("session index append failure should not write sidecars: %+v", rt.files)
	}
	if len(rt.watchers) != 0 || len(rt.pollers) != 0 || len(rt.titles) != 0 || len(rt.cmux) != 0 || rt.devRebuilt {
		t.Fatalf("session index append failure started side effects: watchers=%v pollers=%v titles=%v cmux=%v dev=%v", rt.watchers, rt.pollers, rt.titles, rt.cmux, rt.devRebuilt)
	}
}

func TestRunLaunchPromptedTagIgnoresUnrelatedLegacySessionName(t *testing.T) {
	rt := newFakeRuntime()
	rt.promptValue = "bugfix"
	rt.blocksReuse["pair-bugfix"] = true
	rt.uuids = []string{"SID"}
	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude"}), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.launched != "📁work-bugfix" {
		t.Fatalf("launched = %q, want scoped session name despite unrelated legacy pair-bugfix", rt.launched)
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

func TestRunLaunchUsesRepoAgentDefaultWhenNoTagConfig(t *testing.T) {
	rt := newFakeRuntime()
	raw, err := BuildAgentDefault("codex", []string{"--model", "gpt-5"})
	if err != nil {
		t.Fatalf("BuildAgentDefault: %v", err)
	}
	rt.files["/data/agent-default-codex.json"] = raw

	code, err := run(t, baseOpts(LaunchArgs{Agent: "codex", ForcedTag: "cx"}), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.env["PAIR_AGENT_ARGS"] != "--model gpt-5 --no-alt-screen" {
		t.Fatalf("PAIR_AGENT_ARGS = %q", rt.env["PAIR_AGENT_ARGS"])
	}
}

func TestRunLaunchIgnoresMismatchedTagConfigWithWarning(t *testing.T) {
	rt := newFakeRuntime()
	rt.files["/data/config-cx-codex.json"] = `{"agent":"claude","args":["--old"],"session_id":"OLD"}`
	raw, err := BuildAgentDefault("codex", []string{"--model", "gpt-5"})
	if err != nil {
		t.Fatalf("BuildAgentDefault: %v", err)
	}
	rt.files["/data/agent-default-codex.json"] = raw
	pickerCalled := false
	rt.pickFunc = func(header string, options []string) string {
		pickerCalled = true
		return options[0]
	}

	var stderr bytes.Buffer
	code, err := RunLaunch(baseOpts(LaunchArgs{Agent: "codex", ForcedTag: "cx"}), rt, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if pickerCalled {
		t.Fatalf("mismatched config must not be offered in the restart picker")
	}
	if rt.env["PAIR_AGENT_ARGS"] != "--model gpt-5 --no-alt-screen" {
		t.Fatalf("PAIR_AGENT_ARGS = %q", rt.env["PAIR_AGENT_ARGS"])
	}
	if !strings.Contains(stderr.String(), `saved config agent "claude" does not match requested agent "codex"; ignoring it`) {
		t.Fatalf("stderr missing mismatch warning: %s", stderr.String())
	}
}

func TestRunLaunchLayoutOnlyNewPickUsesRepoAgentDefault(t *testing.T) {
	rt := newFakeRuntime()
	raw, err := BuildAgentDefault("claude", []string{"--model", "opus"})
	if err != nil {
		t.Fatalf("BuildAgentDefault: %v", err)
	}
	rt.files["/data/agent-default-claude.json"] = raw
	rt.historical = []HistoricalTag{{Tag: "old", MTime: time.Unix(1_700_000_000, 0), Agent: "claude"}}
	rt.pickFunc = func(header string, options []string) string {
		return "+ new work session"
	}

	opts := baseOpts(LaunchArgs{
		Agent:         "claude",
		AgentExplicit: true,
		Layout:        LayoutRequest{Mode: Layout2, Explicit: true},
	})
	code, err := run(t, opts, rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.env["PAIR_AGENT_ARGS"] != "--model opus" {
		t.Fatalf("PAIR_AGENT_ARGS = %q, want repo default", rt.env["PAIR_AGENT_ARGS"])
	}
	if got := rt.files["/data/workbench-layout-work"]; got != "layout2\n" {
		t.Fatalf("layout record = %q, want layout2", got)
	}
}

func TestRunLaunchExplicitArgsPersistRepoAgentDefaultAfterReadiness(t *testing.T) {
	rt := newFakeRuntime()
	opts := baseOpts(LaunchArgs{
		Agent:             "codex",
		ForcedTag:         "cx",
		AgentArgs:         []string{"--model", "gpt-5"},
		AgentArgsExplicit: true,
	})

	code, err := run(t, opts, rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.env["PAIR_LAUNCH_NONCE"] == "" {
		t.Fatal("PAIR_LAUNCH_NONCE was not exported")
	}
	got, err := ParseAgentDefault("codex", rt.files["/data/agent-default-codex.json"])
	if err != nil {
		t.Fatalf("ParseAgentDefault: %v", err)
	}
	if !reflect.DeepEqual(got.Args, []string{"--model", "gpt-5"}) {
		t.Fatalf("persisted default args = %#v", got.Args)
	}
}

func TestRunLaunchEmptyExplicitArgsPersistEmptyRepoAgentDefaultAfterReadiness(t *testing.T) {
	rt := newFakeRuntime()
	raw, err := BuildAgentDefault("codex", []string{"--old"})
	if err != nil {
		t.Fatalf("BuildAgentDefault: %v", err)
	}
	rt.files["/data/agent-default-codex.json"] = raw
	opts := baseOpts(LaunchArgs{
		Agent:             "codex",
		ForcedTag:         "cx",
		AgentArgsExplicit: true,
	})

	code, err := run(t, opts, rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	got, err := ParseAgentDefault("codex", rt.files["/data/agent-default-codex.json"])
	if err != nil {
		t.Fatalf("ParseAgentDefault: %v", err)
	}
	if len(got.Args) != 0 {
		t.Fatalf("persisted default args = %#v, want empty", got.Args)
	}
}

func TestRunLaunchExplicitArgsDoNotPersistRepoDefaultOnReadinessTimeout(t *testing.T) {
	rt := newFakeRuntime()
	rt.readyErr = errors.New("ready timeout")
	opts := baseOpts(LaunchArgs{
		Agent:             "codex",
		ForcedTag:         "cx",
		AgentArgs:         []string{"--model", "gpt-5"},
		AgentArgsExplicit: true,
	})

	code, err := run(t, opts, rt)
	if err != nil || code != 1 {
		t.Fatalf("code=%d err=%v, want launch failure", code, err)
	}
	if _, ok := rt.files["/data/agent-default-codex.json"]; ok {
		t.Fatalf("default persisted despite readiness failure: %q", rt.files["/data/agent-default-codex.json"])
	}
}

func TestRunLaunchExplicitArgsDoNotPersistRepoDefaultOnPreLaunchAbort(t *testing.T) {
	rt := newFakeRuntime()
	rt.commandMissing["codex"] = true
	opts := baseOpts(LaunchArgs{
		Agent:             "codex",
		ForcedTag:         "cx",
		AgentArgs:         []string{"--model", "gpt-5"},
		AgentArgsExplicit: true,
	})

	code, err := run(t, opts, rt)
	if err != nil || code != 1 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if _, ok := rt.files["/data/agent-default-codex.json"]; ok {
		t.Fatalf("default persisted despite abort: %q", rt.files["/data/agent-default-codex.json"])
	}
}

// The tag-restart config picker: a saved config offers reuse; picking "saved
// params + session" composes the resume binding.
func TestRunLaunchTagRestartPickerResume(t *testing.T) {
	rt := newFakeRuntime()
	rt.files["/data/config-cx-codex.json"] = `{"agent":"codex","args":["--search"],"session_id":"CX-9"}`
	rt.agentSessions["codex|CX-9"] = true // native session artifact exists → resumable
	rt.establishedSessions["cx|codex"] = "CX-9"
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

func TestRunLaunchSkipConfigPickerUsesRepoDefaultOverSavedConfig(t *testing.T) {
	rt := newFakeRuntime()
	rt.files["/data/config-cx-claude.json"] = `{"agent":"claude","args":["--saved"],"session_id":"OLD"}`
	rt.agentSessions["claude|OLD"] = true
	defaultRaw, err := BuildAgentDefault("claude", []string{"--model", "sonnet"})
	if err != nil {
		t.Fatalf("BuildAgentDefault: %v", err)
	}
	rt.files["/data/agent-default-claude.json"] = defaultRaw
	rt.uuids = []string{"NEW"}
	pickerCalled := false
	rt.pickFunc = func(header string, options []string) string {
		pickerCalled = true
		return options[0]
	}

	opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "cx"})
	opts.SkipConfigPicker = true
	code, err := run(t, opts, rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if pickerCalled {
		t.Fatal("saved-config picker opened despite SkipConfigPicker")
	}
	if got := rt.env["PAIR_AGENT_ARGS"]; got != "--model sonnet --session-id NEW" {
		t.Fatalf("PAIR_AGENT_ARGS = %q", got)
	}
	if !contains(rt.removed, "/data/config-cx-claude.json") || rt.files["/data/config-cx-claude.json"] != "" {
		t.Fatalf("fresh provisional launch retained config: removed=%v files=%v", rt.removed, rt.files)
	}
}

func TestRunLaunchSkipConfigPickerWithoutRepoDefaultUsesNoUserArgs(t *testing.T) {
	rt := newFakeRuntime()
	rt.files["/data/config-cx-codex.json"] = `{"agent":"codex","args":["--saved"],"session_id":"OLD"}`
	rt.agentSessions["codex|OLD"] = true
	pickerCalled := false
	rt.pickFunc = func(header string, options []string) string {
		pickerCalled = true
		return options[0]
	}

	opts := baseOpts(LaunchArgs{Agent: "codex", ForcedTag: "cx"})
	opts.SkipConfigPicker = true
	code, err := run(t, opts, rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if pickerCalled {
		t.Fatal("saved-config picker opened despite SkipConfigPicker")
	}
	if got := rt.env["PAIR_AGENT_ARGS"]; got != "--no-alt-screen" {
		t.Fatalf("PAIR_AGENT_ARGS = %q", got)
	}
}

func TestRunLaunchTagRestartPickerResumeStripsCodexResumeAfterGlobals(t *testing.T) {
	rt := newFakeRuntime()
	rt.files["/data/config-cx-codex.json"] = `{"agent":"codex","args":["--sandbox","danger-full-access","resume","CX-9","--no-alt-screen"],"session_id":"CX-9"}`
	rt.agentSessions["codex|CX-9"] = true
	rt.establishedSessions["cx|codex"] = "CX-9"
	rt.pickFunc = func(header string, options []string) string {
		return options[0] // "use saved params + session"
	}

	code, err := run(t, baseOpts(LaunchArgs{Agent: "codex", ForcedTag: "cx"}), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.env["PAIR_AGENT_ARGS"] != "resume CX-9 --sandbox danger-full-access --no-alt-screen" {
		t.Fatalf("PAIR_AGENT_ARGS = %q", rt.env["PAIR_AGENT_ARGS"])
	}
}

func TestRunLaunchTagRestartPickerWarnsWhenSavedSessionIsStale(t *testing.T) {
	rt := newFakeRuntime()
	rt.files["/data/config-cx-codex.json"] = `{"agent":"codex","args":["--search"],"session_id":"CX-9"}`
	rt.pickFunc = func(header string, options []string) string {
		for _, o := range options {
			if strings.Contains(o, "use saved params") {
				return o
			}
		}
		return ""
	}

	var stderr bytes.Buffer
	code, err := RunLaunch(baseOpts(LaunchArgs{Agent: "codex", ForcedTag: "cx"}), rt, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if rt.env["PAIR_AGENT_ARGS"] != "--search --no-alt-screen" {
		t.Fatalf("PAIR_AGENT_ARGS = %q", rt.env["PAIR_AGENT_ARGS"])
	}
	if !strings.Contains(stderr.String(), `saved session "CX-9" for codex is not available; starting fresh`) {
		t.Fatalf("stderr missing stale-session warning: %s", stderr.String())
	}
	if !slices.Contains(rt.removed, "/data/config-cx-codex.json") {
		t.Fatalf("removed = %v, want stale Codex config quarantined", rt.removed)
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
	opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work", AgentArgs: []string{"--fresh"}, AgentArgsExplicit: true})
	code, err := run(t, opts, rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if !contains(rt.removed, "/data/config-work-claude.json") {
		t.Fatalf("new should remove stale config; removed=%v", rt.removed)
	}
	if cfg := rt.files["/data/config-work-claude.json"]; cfg != "" {
		t.Fatalf("provisional launch wrote fresh config %q", cfg)
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
	opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work", AgentArgs: []string{"--resume", "EXPLICIT"}, AgentArgsExplicit: true})
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

func TestRunLaunchInPaneAllowsCouchOwnedClaimValidation(t *testing.T) {
	rt := newFakeRuntime()
	rt.inPane = true
	opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work"})
	scope, err := ResolveRepoScope(opts.Env.Cwd)
	if err != nil {
		t.Fatal(err)
	}
	opts.Env.CouchThreadScope = scope.Key
	opts.Env.CouchThreadTag = "work"

	code, err := run(t, opts, rt)
	if err != nil || code != 0 {
		t.Fatalf("Couch-owned in-pane launch = %d, %v", code, err)
	}
	if got := rt.threadClaims; len(got) != 1 || !strings.HasSuffix(got[0], "|work|true") {
		t.Fatalf("claims = %v, want exact Couch validation", got)
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

func TestRunLaunchResumeUsesLedgerAgentAndArgsWhenConfigMissing(t *testing.T) {
	rt := newFakeRuntime()
	rt.ledger["work"] = []LedgerEntry{{
		Agent:      "codex",
		Args:       []string{"--search"},
		SessionID:  "CX-9",
		LastActive: time.Unix(1_700_000_010, 0),
	}}
	rt.agentSessions["codex|CX-9"] = true
	rt.establishedSessions["work|codex"] = "CX-9"
	rt.pickFunc = func(header string, options []string) string {
		for _, o := range options {
			if strings.Contains(o, "use saved params + session") {
				return o
			}
		}
		return ""
	}

	code, err := run(t, baseOpts(LaunchArgs{ForcedTag: "work"}), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.env["PAIR_AGENT"] != "codex" {
		t.Fatalf("PAIR_AGENT = %q, want codex", rt.env["PAIR_AGENT"])
	}
	if rt.env["PAIR_AGENT_ARGS"] != "resume CX-9 --search --no-alt-screen" {
		t.Fatalf("PAIR_AGENT_ARGS = %q", rt.env["PAIR_AGENT_ARGS"])
	}
}

func TestRunLaunchRejectsInvalidLedgerCodexSession(t *testing.T) {
	rt := newFakeRuntime()
	rt.ledger["work"] = []LedgerEntry{{
		Agent:      "codex",
		Args:       []string{"--search"},
		SessionID:  "SUBAGENT",
		LastActive: time.Unix(1_700_000_010, 0),
	}}
	rt.pickFunc = func(header string, options []string) string {
		for _, o := range options {
			if strings.Contains(o, "use saved params") {
				return o
			}
		}
		return ""
	}

	var stderr bytes.Buffer
	code, err := RunLaunch(baseOpts(LaunchArgs{ForcedTag: "work"}), rt, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if strings.Contains(rt.env["PAIR_AGENT_ARGS"], "SUBAGENT") {
		t.Fatalf("PAIR_AGENT_ARGS = %q, must not resume rejected session", rt.env["PAIR_AGENT_ARGS"])
	}
	if rt.env["PAIR_AGENT_ARGS"] != "--search --no-alt-screen" {
		t.Fatalf("PAIR_AGENT_ARGS = %q", rt.env["PAIR_AGENT_ARGS"])
	}
	if !slices.Contains(rt.removed, "/data/config-work-codex.json") {
		t.Fatalf("removed = %v, want canonical config quarantine", rt.removed)
	}
	if !strings.Contains(stderr.String(), `saved session "SUBAGENT" for codex is not available; starting fresh`) {
		t.Fatalf("stderr missing stale-session warning: %s", stderr.String())
	}
}

func TestRunLaunchAltNRestartRejectsInvalidSavedCodexSession(t *testing.T) {
	rt := newFakeRuntime()
	rt.files["/data/config-cx-codex.json"] = `{"agent":"codex","args":["--search"],"session_id":"SUBAGENT"}`
	rt.restartMarkers["📁work-cx"] = RestartMarker{Tag: "cx", Agent: "codex"}

	opts := baseOpts(LaunchArgs{Agent: "codex", ForcedTag: "cx"})
	opts.SkipConfigPicker = true
	var stderr bytes.Buffer
	code, err := RunLaunch(opts, rt, &stderr)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v stderr=%s", code, err, stderr.String())
	}
	if rt.launchCount != 2 {
		t.Fatalf("launchCount = %d, want initial launch plus Alt+n relaunch", rt.launchCount)
	}
	if strings.Contains(rt.env["PAIR_AGENT_ARGS"], "SUBAGENT") {
		t.Fatalf("PAIR_AGENT_ARGS = %q, must not resume rejected session", rt.env["PAIR_AGENT_ARGS"])
	}
	if !slices.Contains(rt.removed, "/data/config-cx-codex.json") {
		t.Fatalf("removed = %v, want stale Codex config quarantined", rt.removed)
	}
	if strings.Contains(stderr.String(), "SUBAGENT") {
		t.Fatalf("removed provisional cache leaked into restart decision: %s", stderr.String())
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
	rt.blocksReuse["📁work-bugfix"] = true // forced create → no prompt collision check
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

// Agent-scoped CLI-args guard (#107): `pair codex -- <codex-only args>` must not
// route through the picker and resume an existing claude tag. Explicit
// agent+args means "create a new session for that agent".
func TestRunLaunchPickInferredAgentMustNotInheritCliArgs(t *testing.T) {
	rt := newFakeRuntime()
	// A historical claude tag (base tag for cwd /home/u/work is "work").
	rt.historical = []HistoricalTag{{Tag: "work", MTime: time.Unix(1_699_000_000, 0)}}
	rt.inferAgent["work"] = "claude"
	rt.uuids = []string{"SID"}
	rt.pickFunc = func(header string, options []string) string {
		t.Fatalf("explicit agent+args should not show picker: %q", options)
		return ""
	}
	opts := baseOpts(LaunchArgs{Agent: "codex", AgentExplicit: true, AgentArgsExplicit: true, AgentArgs: []string{"--sandbox", "danger-full-access"}})
	code, err := run(t, opts, rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.env["PAIR_AGENT"] != "codex" {
		t.Fatalf("PAIR_AGENT = %q, want codex", rt.env["PAIR_AGENT"])
	}
	if !strings.Contains(rt.env["PAIR_AGENT_ARGS"], "--sandbox") {
		t.Fatalf("codex args were not preserved: PAIR_AGENT_ARGS=%q", rt.env["PAIR_AGENT_ARGS"])
	}
	if rt.launched != "📁work-2" {
		t.Fatalf("launched = %q, want scoped next-free public session name", rt.launched)
	}
}

func TestRunLaunchPickNewDefaultUsesScopedNextFreeSessionName(t *testing.T) {
	rt := newFakeRuntime()
	scope := mustScope(t, "/home/u/work")
	rt.sessionIndex.Entries = []SessionNameEntry{{
		SessionName: "📁work",
		ScopeKey:    scope.Key,
		RepoRoot:    scope.Root,
		RepoName:    scope.DisplayName,
		Tag:         "work",
	}}
	rt.sessions = []Session{{Name: "📁work", State: SessionDetached}}
	rt.uuids = []string{"SID"}
	rt.pickFunc = func(header string, options []string) string {
		return "+ new work session"
	}

	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude"}), rt)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if rt.launched != "📁work-2" {
		t.Fatalf("launched = %q, want scoped next-free public session name", rt.launched)
	}
}
