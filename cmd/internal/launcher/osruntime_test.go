package launcher

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/contextcmd"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
	"github.com/xianxu/pair/cmd/internal/titlepoller"
)

func TestOSRuntimeStartProofMigrationUpgradesPersistedOwner(t *testing.T) {
	home := t.TempDir()
	dataDir := t.TempDir()
	t.Setenv("HOME", home)
	sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
	dir := filepath.Join(home, ".codex", "sessions", "2026", "08", "29")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	rawSession := []byte(`{"timestamp":"2026-08-29T01:00:00Z","type":"session_meta","payload":{"id":"` + sid + `","parent_thread_id":null,"source":"cli"}}` + "\n" + `{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"legacy"}]}}` + "\n" + `{"type":"response_item","payload":{"type":"function_call"}}` + "\n")
	if err := os.WriteFile(filepath.Join(dir, "rollout-test-"+sid+".jsonl"), rawSession, 0o600); err != nil {
		t.Fatal(err)
	}
	ledgerPath := filepath.Join(dataDir, "ledger-work.jsonl")
	store := sessionledger.LedgerStore{Runtime: sessionledger.OSRuntime{}}
	owner := sessionledger.Owner{ScopeKey: "scope", Tag: "work", Agent: "codex"}
	launch, err := store.Append(ledgerPath, sessionledger.Record{Version: 1, Kind: sessionledger.RecordLaunch, ScopeKey: owner.ScopeKey, Tag: owner.Tag, Agent: owner.Agent})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AppendBindingIfCurrent(ledgerPath, owner, launch.Ordinal, sid); err != nil {
		t.Fatal(err)
	}
	(OSRuntime{DataDir: dataDir}).StartProofMigration()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		raw, readErr := os.ReadFile(ledgerPath)
		if readErr == nil {
			current, ok := sessionledger.CurrentLaunch(sessionledger.ParseLedger(raw).Records, owner)
			if ok && current.Binding != nil && current.Binding.AuthorizationProof != nil {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("launcher startup did not durably migrate proofless owner")
}

// ResolveContinuationDoc / ScanContinuations do real glob+read IO the fake can't
// exercise. Critically, "newest doc wins" is `matches[len-1]` after sort — if it
// were `matches[0]` every fake-driven test still passes but the wrong doc seeds
// the draft, so pin it against real files in a non-git temp cwd (git rev-parse
// fails there → continuationDirPath falls back to cwd) (#99 M5b review, Important).
func TestOSRuntimeResolveContinuation(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)
	cdir := filepath.Join(dir, "workshop", "continuation")
	if err := os.MkdirAll(cdir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, agent, next string) {
		body := "---\nagent: " + agent + "\nissues: [#99]\n---\n## NEXT ACTION\n" + next + "\n"
		if err := os.WriteFile(filepath.Join(cdir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("20260101T000000-demo.md", "claude", "the old one")
	write("20260202T000000-demo.md", "codex", "the newest one") // newest by timestamp name
	write("20260101T000000-other.md", "claude", "other work")

	rt := NewOSRuntime(dir, "/pair")
	path, agent, ok := rt.ResolveContinuationDoc("demo")
	if !ok {
		t.Fatal("demo should resolve")
	}
	if filepath.Base(path) != "20260202T000000-demo.md" {
		t.Fatalf("newest-wins failed (got %s) — matches[0] would pick the 2026-01-01 doc", filepath.Base(path))
	}
	if agent != "codex" {
		t.Fatalf("agent = %q, want codex (from the newest doc)", agent)
	}
	if _, _, ok := rt.ResolveContinuationDoc("missing"); ok {
		t.Fatal("a missing slug must not resolve")
	}

	rows, gotDir := rt.ScanContinuations()
	if !strings.HasSuffix(gotDir, filepath.Join("workshop", "continuation")) {
		t.Fatalf("scan dir = %q", gotDir)
	}
	if len(rows) != 3 { // two demo docs + one other
		t.Fatalf("rows = %d (%+v), want 3", len(rows), rows)
	}
	var demoRows int
	for _, r := range rows {
		if r.Slug == "demo" {
			demoRows++
			if r.Issues != "[#99]" {
				t.Fatalf("issues = %q", r.Issues)
			}
		}
	}
	if demoRows != 2 {
		t.Fatalf("demo rows = %d, want 2", demoRows)
	}
}

// The OSRuntime lifecycle methods that do real filesystem IO (marker read-clear,
// scrollback park, cmux ownership, pidfile reaping) exercised against temp dirs —
// the process-level coverage the fake-Runtime loop tests can't give (#99 M3; the
// M2 review's lesson: don't ship OSRuntime IO untested). The exec-only seams
// (zellij attach/create/delete-session, the ps orphan sweep) are exercised by the
// M3 boundary smoke against a stub zellij (attach → cleanup → in-process re-create)
// — a one-time end-to-end verification recorded in the issue Log, not a committed
// unit test (the real zellij interaction has no in-test home).

func mkCacheDir(t *testing.T) (home, cacheDir string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	cacheDir = filepath.Join(home, ".cache", "pair")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return home, cacheDir
}

func TestOSRuntimeProbeLiveLayoutUsesSessionScopedPaneReport(t *testing.T) {
	bin := t.TempDir()
	argsLog := filepath.Join(bin, "args")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + argsLog + "\n" +
		"printf '%s\\n' '{\"0\":[" +
		"{\"id\":0,\"title\":\"codex\",\"terminal_command\":\"pair wrap codex\"}," +
		"{\"id\":1,\"title\":\"draft\",\"terminal_command\":\"nvim /data/draft-work.md\"}," +
		"{\"id\":3,\"title\":\"terminal\",\"terminal_command\":\"pair term\"}" +
		"]}'\n"
	zellij := filepath.Join(bin, "zellij")
	if err := os.WriteFile(zellij, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	mode, err := (OSRuntime{}).ProbeLiveLayout("pair-work")
	if err != nil || mode != Layout3 {
		t.Fatalf("ProbeLiveLayout = (%q,%v), want (layout3,nil)", mode, err)
	}
	got, err := os.ReadFile(argsLog)
	if err != nil {
		t.Fatal(err)
	}
	want := "--session\npair-work\naction\nlist-panes\n--json\n--command\n--state\n"
	if string(got) != want {
		t.Fatalf("zellij argv = %q, want %q", got, want)
	}
}

func TestOSRuntimeQuitMarker(t *testing.T) {
	_, cacheDir := mkCacheDir(t)
	rt := NewOSRuntime(t.TempDir(), "/pair")

	if rt.TakeQuitMarker("pair-x") {
		t.Fatal("absent quit marker should read false")
	}
	marker := filepath.Join(cacheDir, "quit-pair-x")
	if err := os.WriteFile(marker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if !rt.TakeQuitMarker("pair-x") {
		t.Fatal("present quit marker should read true")
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("TakeQuitMarker must clear the marker")
	}
	if rt.TakeQuitMarker("pair-x") {
		t.Fatal("a cleared marker should read false")
	}
}

func TestOSRuntimeRestartMarker(t *testing.T) {
	_, cacheDir := mkCacheDir(t)
	rt := NewOSRuntime(t.TempDir(), "/pair")
	marker := filepath.Join(cacheDir, "restart-pair-x")
	if err := os.WriteFile(marker, []byte("tag=x\nagent=codex\nnew_session=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Peek must NOT clear (park-nudge skip reads it before acknowledgment).
	if !rt.RestartMarkerPresent("pair-x") {
		t.Fatal("RestartMarkerPresent should see the marker")
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatal("RestartMarkerPresent must not clear the marker")
	}

	m, ok, readErr := rt.ReadRestartMarker("pair-x")
	if readErr != nil || !ok || m.Tag != "x" || m.Agent != "codex" || !m.NewSession {
		t.Fatalf("TakeRestartMarker = %+v ok=%v", m, ok)
	}
	if err := rt.AcknowledgeRestartMarker("pair-x", m); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("acknowledgment must clear matching marker")
	}
	if _, ok, err := rt.ReadRestartMarker("pair-x"); ok || err != nil {
		t.Fatal("a cleared restart marker should read false")
	}
}

func TestOSRuntimeParkScrollbackMove(t *testing.T) {
	dataDir := t.TempDir()
	rt := NewOSRuntime(dataDir, "/pair")
	raw := filepath.Join(dataDir, "scrollback-work-claude.raw")
	if err := os.WriteFile(raw, []byte("captured"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "scrollback-work-claude.events.jsonl"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	base, ok := rt.ParkScrollback("work", "claude", true)
	if !ok {
		t.Fatal("park should succeed with a non-empty raw")
	}
	if _, err := os.Stat(raw); !os.IsNotExist(err) {
		t.Fatal("move mode must remove the original raw")
	}
	for _, suffix := range []string{".raw", ".events.jsonl"} {
		if _, err := os.Stat(base + suffix); err != nil {
			t.Fatalf("parked %s missing: %v", suffix, err)
		}
	}
	if _, err := os.Stat(filepath.Join(dataDir, "parked-work")); err != nil {
		t.Fatal("ParkScrollback must touch the parked-<tag> marker")
	}
}

func TestOSRuntimeParkScrollbackCopyKeepsOriginal(t *testing.T) {
	dataDir := t.TempDir()
	rt := NewOSRuntime(dataDir, "/pair")
	raw := filepath.Join(dataDir, "scrollback-c-claude.raw")
	if err := os.WriteFile(raw, []byte("live bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	base, ok := rt.ParkScrollback("c", "claude", false) // copy (compaction path)
	if !ok {
		t.Fatal("copy park should succeed")
	}
	if _, err := os.Stat(raw); err != nil {
		t.Fatal("copy mode must leave the original raw in place")
	}
	if _, err := os.Stat(base + ".raw"); err != nil {
		t.Fatal("copy park should still write the parked raw")
	}
}

func TestOSRuntimeParkScrollbackEmpty(t *testing.T) {
	dataDir := t.TempDir()
	rt := NewOSRuntime(dataDir, "/pair")
	if err := os.WriteFile(filepath.Join(dataDir, "scrollback-work-claude.raw"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := rt.ParkScrollback("work", "claude", true); ok {
		t.Fatal("an empty raw should not park")
	}
}

func TestOSRuntimeCmuxOwnership(t *testing.T) {
	dataDir := t.TempDir()
	rt := NewOSRuntime(dataDir, "/pair")
	t.Setenv("CMUX_WORKSPACE_ID", "ws1")
	owner := filepath.Join(dataDir, "cmux-owner-ws1")
	if err := os.WriteFile(owner, []byte("work\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !rt.PairOwnsCmuxWorkspace("work") {
		t.Fatal("owner-file == tag should own")
	}
	if err := os.WriteFile(owner, []byte("work\tpair-work\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !rt.PairOwnsCmuxWorkspace("work") {
		t.Fatal("owner-file tag+session should own")
	}
	if rt.PairOwnsCmuxWorkspace("other") {
		t.Fatal("owner-file mismatch should not own")
	}
	rt.ClearCmuxOwner()
	if _, err := os.Stat(owner); !os.IsNotExist(err) {
		t.Fatal("ClearCmuxOwner must remove the owner file")
	}

	t.Setenv("CMUX_WORKSPACE_ID", "")
	if rt.PairOwnsCmuxWorkspace("work") {
		t.Fatal("outside cmux (no CMUX_WORKSPACE_ID) nothing is owned")
	}
}

func TestOSRuntimeInferAgentPrefersLedger(t *testing.T) {
	dataDir := t.TempDir()
	rt := NewOSRuntime(dataDir, "/pair")
	entry := LedgerEntry{
		Agent:      "codex",
		SessionID:  "SID",
		LastActive: timeUnix(40),
	}
	line, err := BuildLedgerLine(entry)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "ledger-work.jsonl"), []byte(line+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "agent-work"), []byte("claude\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := rt.InferAgent("work"); got != "codex" {
		t.Fatalf("InferAgent = %q, want ledger codex", got)
	}
}

func TestOSRuntimeInferAgentDoesNotTreatConfigFilenameAsAuthority(t *testing.T) {
	dataDir := t.TempDir()
	rt := NewOSRuntime(dataDir, "/pair")
	if err := os.WriteFile(filepath.Join(dataDir, "config-work-claude.json"), []byte(`{"agent":"claude","args":[],"session_id":"stale"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := rt.InferAgent("work"); got != "" {
		t.Fatalf("InferAgent = %q, want no authority from compatibility config", got)
	}
}

func TestOSRuntimeAgentSessionExistsFindsNestedCodexRollout(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sid := "12345678-1234-1234-1234-123456789abc"
	path := filepath.Join(home, ".codex", "sessions", "2026", "07", "07", "rollout-2026-07-07T21-00-00-"+sid+".jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	first := fmt.Sprintf(`{"type":"session_meta","payload":{"id":%q,"parent_thread_id":null,"source":"cli"}}`+"\n", sid)
	if err := os.WriteFile(path, []byte(first), 0o644); err != nil {
		t.Fatal(err)
	}

	if !(OSRuntime{}).AgentSessionExists("codex", sid, "/repo") {
		t.Fatal("AgentSessionExists(codex) did not find nested rollout file")
	}
	parent := "019e8178-79c2-7862-91db-e8fa1be3b162"
	subagent := fmt.Sprintf(`{"type":"session_meta","payload":{"id":%q,"parent_thread_id":%q,"source":{"subagent":{}}}}`+"\n", sid, parent)
	if err := os.WriteFile(path, []byte(subagent), 0o644); err != nil {
		t.Fatal(err)
	}
	if (OSRuntime{}).AgentSessionExists("codex", sid, "/repo") {
		t.Fatal("AgentSessionExists(codex) accepted a real subagent rollout")
	}
}

func TestOSRuntimeAgentSessionExistsFindsQoderTranscript(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sid := "12345678-1234-1234-1234-123456789abc"
	path := filepath.Join(home, ".qoder", "projects", "-repo", sid+".jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	first := fmt.Sprintf(`{"type":"user","timestamp":"2026-08-28T09:01:00.000Z","message":{"role":"user","content":"sanitized"},"isSidechain":false,"sessionId":%q}`+"\n", sid)
	if err := os.WriteFile(path, []byte(first), 0o644); err != nil {
		t.Fatal(err)
	}

	if !(OSRuntime{}).AgentSessionExists("qoder", sid, "/repo") {
		t.Fatal("AgentSessionExists(qoder) did not find the transcript")
	}
	t.Setenv("HOME", t.TempDir())
	if (OSRuntime{}).AgentSessionExists("qoder", sid, "/repo") {
		t.Fatal("AgentSessionExists(qoder) accepted an empty native root")
	}
}

// Grok's transcript is <url-encoded cwd>/<uuid>/updates.jsonl under
// ~/.grok/sessions; a session directory with only summary.json (a stub) is not
// a conversation.
func TestOSRuntimeAgentSessionExistsFindsGrokTranscript(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sid := "12345678-1234-4234-8234-123456789abc"
	dir := filepath.Join(home, ".grok", "sessions", "%2Frepo", sid)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "summary.json"), []byte(`{"info":{"id":"`+sid+`"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if (OSRuntime{}).AgentSessionExists("grok", sid, "/repo") {
		t.Fatal("AgentSessionExists(grok) accepted a stub session with no updates.jsonl")
	}
	first := fmt.Sprintf(`{"timestamp":1791400000,"method":"session/update","params":{"sessionId":%q,"update":{"sessionUpdate":"user_message_chunk","content":{"type":"text","text":"sanitized"}}}}`+"\n", sid)
	if err := os.WriteFile(filepath.Join(dir, "updates.jsonl"), []byte(first), 0o644); err != nil {
		t.Fatal(err)
	}
	if !(OSRuntime{}).AgentSessionExists("grok", sid, "/repo") {
		t.Fatal("AgentSessionExists(grok) did not find the transcript")
	}
	t.Setenv("HOME", t.TempDir())
	if (OSRuntime{}).AgentSessionExists("grok", sid, "/repo") {
		t.Fatal("AgentSessionExists(grok) accepted an empty native root")
	}
}

func TestOSRuntimeSessionNameIndexStore(t *testing.T) {
	dataDir := t.TempDir()
	rt := NewOSRuntime(dataDir, "/pair")
	entry := SessionNameEntry{
		SessionName: "📁pair-work",
		ScopeKey:    "scope1",
		RepoRoot:    "/repo",
		RepoName:    "pair",
		Tag:         "work",
	}
	if err := rt.AppendSessionNameIndex(entry); err != nil {
		t.Fatalf("AppendSessionNameIndex: %v", err)
	}
	index, err := rt.ReadSessionNameIndex()
	if err != nil {
		t.Fatalf("ReadSessionNameIndex: %v", err)
	}
	if len(index.Entries) != 1 || index.Entries[0] != entry {
		t.Fatalf("index = %#v, want one appended entry", index)
	}
}

func TestOSRuntimeSessionNameIndexUsesSelectedScopeDir(t *testing.T) {
	globalDir := t.TempDir()
	scopedDir := t.TempDir()
	rt := NewScopedOSRuntime(globalDir, scopedDir, "/pair")
	entry := SessionNameEntry{SessionName: "📁pair-work", ScopeKey: "scope1", Tag: "work"}
	if err := rt.AppendSessionNameIndex(entry); err != nil {
		t.Fatalf("AppendSessionNameIndex: %v", err)
	}
	if _, ok := rt.FileSize(filepath.Join(scopedDir, "session-names.jsonl")); !ok {
		t.Fatalf("session index was not written under selected scope dir")
	}
	if _, ok := rt.FileSize(filepath.Join(globalDir, "session-names.jsonl")); ok {
		t.Fatalf("session index escaped to global data dir")
	}
}

func TestOSRuntimeSessionNameIndexReadsLegacyGlobalEntriesAlongsideScopedEntries(t *testing.T) {
	globalDir := t.TempDir()
	scopedDir := t.TempDir()
	scope := RepoScope{Key: "scope1", Root: "/repo", DisplayName: "pair"}
	legacy := SessionNameEntry{SessionName: "📁pair-work", ScopeKey: scope.Key, RepoRoot: scope.Root, RepoName: scope.DisplayName, Tag: "work"}
	line, err := BuildSessionNameIndexLine(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(globalDir, "session-names.jsonl"), []byte(line+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	rt := NewScopedOSRuntime(globalDir, scopedDir, "/pair")
	index, err := rt.ReadSessionNameIndex()
	if err != nil {
		t.Fatalf("ReadSessionNameIndex: %v", err)
	}
	name, _, err := AssignSessionName(index, []Session{{Name: legacy.SessionName, State: SessionDetached}}, scope, legacy.Tag, acceptAllSessionNames)
	if err != nil {
		t.Fatalf("AssignSessionName: %v", err)
	}
	if name != legacy.SessionName {
		t.Fatalf("legacy live session assigned %q, want existing %q", name, legacy.SessionName)
	}

	scoped := SessionNameEntry{SessionName: "📁pair-next", ScopeKey: scope.Key, RepoRoot: scope.Root, RepoName: scope.DisplayName, Tag: "next"}
	if err := rt.AppendSessionNameIndex(scoped); err != nil {
		t.Fatal(err)
	}
	index, err = rt.ReadSessionNameIndex()
	if err != nil {
		t.Fatalf("ReadSessionNameIndex after scoped append: %v", err)
	}
	if len(index.Entries) != 2 || index.Entries[0] != legacy || index.Entries[1] != scoped {
		t.Fatalf("merged index = %#v, want legacy then scoped", index)
	}
}

func TestOSRuntimeSessionNameIndexRejectsMalformedDurableRows(t *testing.T) {
	globalDir := t.TempDir()
	scopedDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(scopedDir, "session-names.jsonl"), []byte("not-json\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewScopedOSRuntime(globalDir, scopedDir, "/pair").ReadSessionNameIndex(); err == nil {
		t.Fatal("ReadSessionNameIndex accepted a malformed durable row")
	}
}

func TestOSRuntimeSessionNameIndexRejectsStructurallyIncompleteRows(t *testing.T) {
	globalDir := t.TempDir()
	scopedDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(scopedDir, "session-names.jsonl"), []byte(`{"session_name":"📁repo-work","scope_key":"0123456789abcdef","tag":"work"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewScopedOSRuntime(globalDir, scopedDir, "/pair").ReadSessionNameIndex(); err == nil {
		t.Fatal("ReadSessionNameIndex accepted a structurally incomplete durable row")
	}
}

func timeUnix(sec int64) time.Time { return time.Unix(sec, 0).UTC() }

func TestOSRuntimeReapAndPollerRemovePidfiles(t *testing.T) {
	dataDir := t.TempDir()
	rt := NewOSRuntime(dataDir, "/pair")
	// A syntactically-valid but non-existent pid: kill(2) returns ESRCH, so this
	// can never signal a real process; the assertion is on the pidfile removal.
	const deadPid = "2147483646"
	for _, name := range []string{"nvim-pid-work-draft", "nvim-pid-work-scrollback", "title-pid-work"} {
		if err := os.WriteFile(filepath.Join(dataDir, name), []byte(deadPid+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	rt.ReapNvim("work")
	for _, name := range []string{"nvim-pid-work-draft", "nvim-pid-work-scrollback"} {
		if _, err := os.Stat(filepath.Join(dataDir, name)); !os.IsNotExist(err) {
			t.Fatalf("ReapNvim should clear the %s pidfile", name)
		}
	}
	rt.KillTitlePoller("work")
	if _, err := os.Stat(filepath.Join(dataDir, "title-pid-work")); !os.IsNotExist(err) {
		t.Fatal("KillTitlePoller should clear the title pidfile")
	}
}

// The poller's contract only wins because os/exec keeps the LAST value of a
// duplicate key and childEnviron puts the contract last. Asserted against a real
// child rather than assumed (close review BR-4, ARCH-MOCK): the fake's overlay
// encodes the same rule, so without this the double and the dependency could
// disagree with nothing to say so.
func TestChildEnvironLetsTheContractBeatAnInheritedKey(t *testing.T) {
	t.Setenv(contextcmd.EnvScopeKey, "stale-inherited")
	cmd := exec.Command("/bin/sh", "-c", `printf %s "$`+contextcmd.EnvScopeKey+`"`)
	cmd.Env = childEnviron(titlepoller.NewSessionEnv("/data/repos/k", "contract").Environ())
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "contract" {
		t.Fatalf("the child saw %s=%q, want the contract's %q -- an inherited value won", contextcmd.EnvScopeKey, out, "contract")
	}
}

// The sidecar spawn argv must self-exec the single `pair` binary as a
// subcommand — #104 M2 folded pair-title/pair-session-watch into `pair title` /
// `pair session-watch`. spawnDetached swallows a start error, so a regression in
// the argv shape would fail silently at runtime. Pin the subcommand + the title
// poller's "<…>/pair title <tag> <agent>" prefix the single-instance guard matches.
func TestSidecarSpawnArgvSelfExecsPair(t *testing.T) {
	const exe = "/pair/bin/pair"
	contract := titlepoller.NewSessionEnv("/data/repos/k", "k")
	tp, tpEnv := titlePollerSpawn(exe, "work", "claude", "📁pair-work", contract)
	wantTP := []string{exe, "title", "work", "claude", "📁pair-work"}
	if !reflect.DeepEqual(tp, wantTP) {
		t.Fatalf("title poller argv = %v, want %v", tp, wantTP)
	}
	if !reflect.DeepEqual(tpEnv, contract.Environ()) {
		t.Fatalf("title poller env = %v, want its contract %v -- a dropped env is silent, exactly as pair#183 was", tpEnv, contract.Environ())
	}

	// Guard the invariant explicitly: no sidecar target is a standalone helper
	// binary or a .sh shim — it self-execs `pair` with a subcommand.
	for _, argv := range [][]string{tp} {
		if strings.HasSuffix(argv[0], ".sh") || strings.HasSuffix(argv[0], "pair-title") || strings.HasSuffix(argv[0], "pair-session-watch") {
			t.Fatalf("sidecar spawn target must self-exec pair, not a standalone binary: %q", argv[0])
		}
	}
}

func TestSidecarProcessOwnershipFollowsCouchIncarnation(t *testing.T) {
	if got := sidecarProcessAttributes("0123456789abcdef", "couch-0001020304050607"); got != nil {
		t.Fatalf("Couch sidecar escaped actor process group: %+v", got)
	}
	for _, tc := range []struct{ scope, tag string }{{"", ""}, {"scope", ""}, {"", "tag"}} {
		got := sidecarProcessAttributes(tc.scope, tc.tag)
		if got == nil || !got.Setsid {
			t.Fatalf("direct Pair sidecar attributes for %q/%q = %+v", tc.scope, tc.tag, got)
		}
	}
}

func TestSpawnDetachedKeepsCouchSidecarInOwnedProcessGroup(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "sidecar-pgid")
	t.Setenv("COUCH_THREAD_SCOPE", "0123456789abcdef")
	t.Setenv("COUCH_THREAD_TAG", "couch-0001020304050607")
	spawnDetached([]string{os.Args[0], "-test.run=^TestSidecarProcessGroupProbe$"}, []string{"PAIR_TEST_SIDECAR_PG_MARKER=" + marker})

	deadline := time.Now().Add(2 * time.Second)
	var fields []string
	for time.Now().Before(deadline) {
		raw, err := os.ReadFile(marker)
		if err == nil {
			fields = strings.Fields(string(raw))
			if len(fields) == 2 {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	if len(fields) != 2 {
		t.Fatalf("sidecar process-group probe = %v", fields)
	}
	pid, _ := strconv.Atoi(fields[0])
	pgid, _ := strconv.Atoi(fields[1])
	t.Cleanup(func() {
		if process, err := os.FindProcess(pid); err == nil {
			_ = process.Kill()
		}
	})
	if pgid != syscall.Getpgrp() {
		t.Fatalf("Couch sidecar pgid = %d, want inherited actor pgid %d", pgid, syscall.Getpgrp())
	}
}

func TestSidecarProcessGroupProbe(t *testing.T) {
	marker := os.Getenv("PAIR_TEST_SIDECAR_PG_MARKER")
	if marker == "" {
		return
	}
	if err := os.WriteFile(marker, []byte(fmt.Sprintf("%d %d\n", os.Getpid(), syscall.Getpgrp())), 0o600); err != nil {
		os.Exit(2)
	}
	time.Sleep(30 * time.Second)
}

func TestParkScrollbackRepeatedCopiesNeverOverwrite(t *testing.T) {
	dataDir := t.TempDir()
	rt := NewOSRuntime(dataDir, "/pair")
	raw := filepath.Join(dataDir, "scrollback-work-claude.raw")
	seen := map[string]string{}
	for i := 0; i < 5; i++ {
		content := fmt.Sprintf("capture %d", i)
		if err := os.WriteFile(raw, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		base, ok := rt.ParkScrollback("work", "claude", false)
		if !ok {
			t.Fatal("park failed")
		}
		if _, exists := seen[base]; exists {
			t.Fatal("archive overwritten", base)
		}
		seen[base] = content
	}
	for base, want := range seen {
		got, err := os.ReadFile(base + ".raw")
		if err != nil || string(got) != want {
			t.Fatalf("archive %s: %q %v", base, got, err)
		}
	}
}

func TestParkScrollbackEventsFailureRetainsRawWithoutEvents(t *testing.T) {
	dataDir := t.TempDir()
	rt := NewOSRuntime(dataDir, "/pair")
	raw := filepath.Join(dataDir, "scrollback-work-codex.raw")
	events := filepath.Join(dataDir, "scrollback-work-codex.events.jsonl")
	if err := os.WriteFile(raw, []byte("raw survives"), 0600); err != nil {
		t.Fatal(err)
	}
	// An unreadable-as-file sidecar deterministically fails transfer even as root.
	if err := os.Mkdir(events, 0700); err != nil {
		t.Fatal(err)
	}
	base, ok := rt.ParkScrollback("work", "codex", true)
	if !ok {
		t.Fatal("optional events failure rejected raw capture")
	}
	if got, err := os.ReadFile(base + ".raw"); err != nil || string(got) != "raw survives" {
		t.Fatalf("raw = %q %v", got, err)
	}
	if _, err := os.Stat(base + ".events.jsonl"); !os.IsNotExist(err) {
		t.Fatalf("failed sidecar still looks present: %v", err)
	}
}

func TestScrollbackTransferCannotOverwriteDestination(t *testing.T) {
	dir := t.TempDir()
	source, destination := filepath.Join(dir, "source"), filepath.Join(dir, "destination")
	if err := os.WriteFile(source, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := transferFile(source, destination, true); !errors.Is(err, os.ErrExist) {
		t.Fatalf("expected exclusive collision: %v", err)
	}
	for path, want := range map[string]string{source: "new", destination: "old"} {
		got, err := os.ReadFile(path)
		if err != nil || string(got) != want {
			t.Fatalf("%s changed: %q %v", path, got, err)
		}
	}
}

func TestParkScrollbackSkipsOrphanedEventsArchive(t *testing.T) {
	dataDir := t.TempDir()
	rt := NewOSRuntime(dataDir, "/pair")
	raw := filepath.Join(dataDir, "scrollback-work-codex.raw")
	if err := os.WriteFile(raw, []byte("new raw"), 0600); err != nil {
		t.Fatal(err)
	}
	// Cover the clock boundary while forcing the first family candidate to collide.
	now := time.Now()
	for offset := -2; offset <= 2; offset++ {
		token := now.Add(time.Duration(offset) * time.Second).Format("20060102T150405")
		orphan := filepath.Join(dataDir, "parked-scrollback-work-"+token+".events.jsonl")
		if err := os.WriteFile(orphan, []byte("unrelated events"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	base, ok := rt.ParkScrollback("work", "codex", true)
	if !ok {
		t.Fatal("park failed")
	}
	if _, err := os.Stat(base + ".events.jsonl"); !os.IsNotExist(err) {
		t.Fatalf("adopted unrelated events: %s %v", base, err)
	}
}

func TestParkScrollbackPublishesProducerClock(t *testing.T) {
	dir := t.TempDir()
	rt := NewOSRuntime(dir, "/pair")
	if err := os.WriteFile(filepath.Join(dir, "scrollback-work-codex.raw"), []byte("capture"), 0600); err != nil {
		t.Fatal(err)
	}
	before := time.Now()
	base, ok := rt.ParkScrollback("work", "codex", false)
	after := time.Now()
	if !ok {
		t.Fatal("park failed")
	}
	physical, _ := filepath.EvalSymlinks(dir)
	o, _ := artifactpath.NewStorageOwner(physical, "", "work")
	physicalBase := filepath.Join(physical, filepath.Base(base))
	m, _ := artifactpath.MatchArtifact(physicalBase+".raw", []artifactpath.StorageOwner{o}, nil)
	cap, err := artifactpath.ParseParkedCapture(m, time.UTC)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(cap.Metadata)
	if err != nil {
		t.Fatal(err)
	}
	meta, err := artifactpath.DecodeCaptureMetadata(data, cap)
	if err != nil {
		t.Fatal(err)
	}
	if meta.CapturedAt.Before(before) || meta.CapturedAt.After(after) || meta.EventsIdentity != nil {
		t.Fatalf("bad producer evidence %+v", meta)
	}
}

func TestParkScrollbackPreservesSelectedAliasInReturnedBase(t *testing.T) {
	root := t.TempDir()
	alias := filepath.Join(t.TempDir(), "selected")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	paths, _ := artifactpath.ResolveScoped(alias, "work")
	live, _ := paths.ScrollbackArtifacts("codex")
	if err := os.WriteFile(live.Raw, []byte("capture"), 0600); err != nil {
		t.Fatal(err)
	}
	base, ok := NewOSRuntime(alias, "/pair").ParkScrollback("work", "codex", false)
	if !ok || filepath.Dir(base) != alias {
		t.Fatalf("returned %q %v, expected selected directory %q", base, ok, alias)
	}
	if _, err := os.Stat(base + ".capture.json"); err != nil {
		t.Fatal(err)
	}
}

// Exercise the native metadata adapter, ledger reader and Alt+n marker together.
func TestOSLedgerChosenRestartUsesOnlyMaterializedRoot(t *testing.T) {
	const id = "11111111-1111-4111-8111-111111111111"
	nativePath := map[string]func(home string) string{
		"claude": func(home string) string { return filepath.Join(home, ".claude", "projects", "-repo", id+".jsonl") },
		"qoder":  func(home string) string { return filepath.Join(home, ".qoder", "projects", "-repo", id+".jsonl") },
		"grok": func(home string) string {
			return filepath.Join(home, ".grok", "sessions", "%2Frepo", id, "updates.jsonl")
		},
	}
	for agent, nativeAt := range nativePath {
		t.Run(agent, func(t *testing.T) {
			home, data := t.TempDir(), t.TempDir()
			t.Setenv("HOME", home)
			launch, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 3, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: agent, RequestedNativeID: id, RequestOrigin: sessionledger.RequestOriginChosen, BaselineComplete: true})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(data, "ledger-work.jsonl"), append(launch, '\n'), 0600); err != nil {
				t.Fatal(err)
			}
			native := nativeAt(home)
			for _, present := range []bool{false, true} {
				if present {
					if err := os.MkdirAll(filepath.Dir(native), 0700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(native, []byte("unknown future format"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				entries, err := (OSRuntime{DataDir: data}).ReadLedger("work")
				if err != nil {
					t.Fatal(err)
				}
				rt := newFakeRuntime()
				rt.inferAgent["work"] = agent
				rt.ledger["work"] = entries
				var stderr strings.Builder
				if code := runRestart(rt, LaunchArgs{}, "📁work", "work", false, &stderr); code != 0 {
					t.Fatal(stderr.String())
				}
				marker := rt.writtenMarkers["📁work"]
				want := ""
				if present {
					want = id
				}
				if marker.SessionID != want {
					t.Fatalf("present=%v marker=%+v", present, marker)
				}
				if !present {
					plan := planRestart(marker, "work", agent, savedConfig{Agent: agent, SessionID: id, Args: []string{"--session-id", id}})
					if !plan.DropConfig {
						t.Fatal("did not drop stale config")
					}
					fresh := newFakeRuntime()
					fresh.uuids = []string{"new-Y"}
					if code, err := run(t, baseOpts(plan.Args), fresh); err != nil || code != 0 {
						t.Fatalf("fresh launch=%d,%v", code, err)
					}
					if fresh.env["PAIR_SESSION_ID"] != "new-Y" || !strings.Contains(launchArgsText(t, fresh.env), "--session-id new-Y") {
						t.Fatalf("did not mint new UUID: %v", fresh.env)
					}
				}
			}
		})
	}
}

func TestChosenUnknownMetadataRefusesRestartAndConfigPicker(t *testing.T) {
	home, data := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	// A rejected native entry produces an incomplete metadata listing on every OS,
	// even when tests execute with permissions that bypass chmod-based fixtures.
	root := filepath.Join(home, ".claude", "projects")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/unavailable", filepath.Join(root, "rejected")); err != nil {
		t.Fatal(err)
	}
	raw := `{"v":3,"kind":"launch","scope_key":"scope","tag":"work","agent":"claude","pair_log_offset":0,"artifact_boundaries":[],"requested_native_id":"11111111-1111-4111-8111-111111111111","request_origin":"chosen-id","baseline_complete":true}` + "\n"
	if err := os.WriteFile(filepath.Join(data, "ledger-work.jsonl"), []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	entries, err := (OSRuntime{DataDir: data}).ReadLedger("work")
	if err != nil {
		t.Fatal(err)
	}
	rt := newFakeRuntime()
	rt.inferAgent["work"] = "claude"
	rt.ledger["work"] = entries
	var stderr strings.Builder
	if code := runRestart(rt, LaunchArgs{}, "📁work", "work", false, &stderr); code != 1 || len(rt.writtenMarkers) != 0 || len(rt.killed) != 0 {
		t.Fatalf("restart destroyed unknown session: code=%d markers=%v killed=%v stderr=%s", code, rt.writtenMarkers, rt.killed, stderr.String())
	}
	rt.files["/data/config-work-claude.json"] = `{"agent":"claude","args":[],"session_id":"X"}`
	code, err := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work"}), rt)
	if err != nil || code != 1 || rt.launched != "" || rt.files["/data/config-work-claude.json"] == "" {
		t.Fatalf("createflow did not preserve/refuse unknown: code=%d err=%v launched=%q", code, err, rt.launched)
	}
}

func TestConfigPickerReusesTypedResumeProjection(t *testing.T) {
	rt := newFakeRuntime()
	rt.ledger["work"] = []LedgerEntry{{Agent: "claude", Typed: true, SessionID: "chosen-X"}}
	rt.files["/data/config-work-claude.json"] = `{"agent":"claude","args":[],"session_id":"chosen-X"}`
	// The fake's separate EstablishedSessionID probe has no answer. Calling it
	// again would lose X and remove the config despite the observed typed target.
	saved, _, err := readSavedConfigForTag(rt, "/data/config-work-claude.json", "scope", "work", "claude")
	if err != nil || saved.SessionID != "chosen-X" || rt.files["/data/config-work-claude.json"] == "" {
		t.Fatalf("lost typed projection: %+v err=%v", saved, err)
	}
}
