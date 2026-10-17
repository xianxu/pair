package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
	"github.com/xianxu/pair/cmd/internal/testpair"
	"github.com/xianxu/pair/cmd/internal/testwait"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

type fakeCouchTitles struct {
	pollHook func(string)
	facts    map[string]couchTitleFacts
	polls    map[string]int
	writes   []string
	fail     bool
	owner    string
	during   func()
}

func (f *fakeCouchTitles) Open(t couchcore.TitleTarget) (couchTitleSource, error) {
	return fakeCouchTitleSource{f, t.Address.RepoScope}, nil
}
func (f *fakeCouchTitles) Now() time.Time { return time.Unix(1000, 0) }
func (f *fakeCouchTitles) WorkspaceOwner(context.Context, couchcore.TitleTarget) (string, error) {
	return f.owner, nil
}
func (f *fakeCouchTitles) WriteWorkspaceOwner(_ context.Context, _ couchcore.TitleTarget, raw string) error {
	f.owner = raw
	return nil
}
func (f *fakeCouchTitles) RenamePane(_ context.Context, t couchcore.TitleTarget, pane, title string) error {
	f.writes = append(f.writes, t.Address.RepoScope+":"+t.Session+":"+pane+":"+title)
	if f.during != nil {
		f.during()
	}
	if f.fail {
		return errors.New("unknown write outcome")
	}
	return nil
}
func (f *fakeCouchTitles) RenameWorkspace(_ context.Context, title string) error {
	f.writes = append(f.writes, "workspace:"+title)
	return nil
}

type fakeCouchTitleSource struct {
	io    *fakeCouchTitles
	scope string
}

func (f fakeCouchTitleSource) Poll(ctx context.Context) (couchTitleFacts, error) {
	if err := ctx.Err(); err != nil {
		return couchTitleFacts{}, err
	}
	f.io.polls[f.scope]++
	if f.io.pollHook != nil {
		f.io.pollHook(f.scope)
	}
	return f.io.facts[f.scope], nil
}
func couchTitleRow(scope string) couchcore.ActionableThreadSummary {
	a := couchcore.ThreadAddress{RepoScope: scope, Tag: "same"}
	return couchcore.ActionableThreadSummary{Address: a, Agent: "codex", State: couchcore.ThreadLive, TitleTarget: &couchcore.TitleTarget{Address: a, Agent: "codex", Session: "📁" + scope, StartNonce: "launch", LaunchOrdinal: 1, Sessions: map[string]bool{"📁" + scope: true}}}
}
func TestCouchTitlesFailureRetriesAndScopeIsolation(t *testing.T) {
	f := &fakeCouchTitles{facts: map[string]couchTitleFacts{"a": {Ready: true, PaneID: "1", Count: "42", Activity: time.Unix(1000, 0)}, "b": {Ready: true, PaneID: "1", Count: "43", Activity: time.Unix(1000, 0)}}, polls: map[string]int{}, fail: true}
	u := newCouchTitleUpdater("unused", "unused", "", func(context.Context, couchcore.TitleTarget) bool { return true })
	u.io = f
	rows := []couchcore.ActionableThreadSummary{couchTitleRow("a"), couchTitleRow("b")}
	u.Update(context.Background(), rows)
	if len(f.writes) != 2 {
		t.Fatalf("writes=%v", f.writes)
	}
	f.fail = false
	u.Update(context.Background(), rows)
	if len(f.writes) != 4 {
		t.Fatalf("failed writes not retried: %v", f.writes)
	}
	u.Update(context.Background(), rows)
	if len(f.writes) != 4 {
		t.Fatalf("confirmed titles rewritten: %v", f.writes)
	}
	if f.writes[0] == f.writes[1] {
		t.Fatal("scope collision")
	}
	u.Update(context.Background(), nil)
	if len(u.entries) != 0 {
		t.Fatalf("retained entries=%d", len(u.entries))
	}
}
func TestCouchTitlesRejectChangedBindingAtEffect(t *testing.T) {
	f := &fakeCouchTitles{facts: map[string]couchTitleFacts{"a": {Ready: true, PaneID: "1", Activity: time.Unix(1000, 0)}}, polls: map[string]int{}}
	current := false
	u := newCouchTitleUpdater("unused", "unused", "", func(context.Context, couchcore.TitleTarget) bool { return current })
	u.io = f
	row := couchTitleRow("a")
	u.Update(context.Background(), []couchcore.ActionableThreadSummary{row})
	if len(f.writes) != 0 {
		t.Fatalf("stale writes=%v", f.writes)
	}
	current = true
	u.Update(context.Background(), []couchcore.ActionableThreadSummary{row})
	if len(f.writes) != 1 {
		t.Fatalf("new current target skipped=%v", f.writes)
	}
}
func TestCouchTitlesWorkspaceRespectsExistingListing(t *testing.T) {
	f := &fakeCouchTitles{facts: map[string]couchTitleFacts{"a": {Ready: true, PaneID: "1", Activity: time.Unix(1000, 0)}}, polls: map[string]int{}, owner: "other\tforeign\n"}
	u := newCouchTitleUpdater("unused", "unused", "workspace", func(context.Context, couchcore.TitleTarget) bool { return true })
	u.io = f
	row := couchTitleRow("a")
	row.TitleTarget.Sessions["foreign"] = true
	u.Update(context.Background(), []couchcore.ActionableThreadSummary{row})
	if len(f.writes) != 1 {
		t.Fatalf("foreign owner overwritten: %v", f.writes)
	}
	delete(row.TitleTarget.Sessions, "foreign")
	u.Update(context.Background(), []couchcore.ActionableThreadSummary{row})
	if len(f.writes) != 2 || f.writes[1] != "workspace:🔴 📁a" {
		t.Fatalf("dead owner not reclaimed: %v", f.writes)
	}
}

func TestCouchTitlesLateSuccessDoesNotConfirmOldBinding(t *testing.T) {
	f := &fakeCouchTitles{facts: map[string]couchTitleFacts{"a": {Ready: true, PaneID: "1", Activity: time.Unix(1000, 0)}}, polls: map[string]int{}}
	current := true
	u := newCouchTitleUpdater("unused", "unused", "", func(context.Context, couchcore.TitleTarget) bool { return current })
	u.io = f
	row := couchTitleRow("a")
	f.during = func() { current = false }
	u.Update(context.Background(), []couchcore.ActionableThreadSummary{row})
	current = true
	f.during = nil
	u.Update(context.Background(), []couchcore.ActionableThreadSummary{row})
	if len(f.writes) != 2 {
		t.Fatalf("late success incorrectly confirmed old binding: %v", f.writes)
	}
}
func TestCouchTitlesCanceledPassRetainsSkippedCache(t *testing.T) {
	f := &fakeCouchTitles{facts: map[string]couchTitleFacts{"a": {Ready: true, PaneID: "1", Activity: time.Unix(1000, 0)}, "b": {Ready: true, PaneID: "1", Activity: time.Unix(1000, 0)}}, polls: map[string]int{}}
	u := newCouchTitleUpdater("unused", "unused", "", func(context.Context, couchcore.TitleTarget) bool { return true })
	u.io = f
	rows := []couchcore.ActionableThreadSummary{couchTitleRow("a"), couchTitleRow("b")}
	u.Update(context.Background(), rows)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, failed := u.Update(ctx, rows)
	if !failed[rows[0].Address] || !failed[rows[1].Address] {
		t.Fatal("skipped rows not unresolved")
	}
	u.Update(context.Background(), rows)
	if len(f.writes) != 2 {
		t.Fatalf("budget skip discarded confirmed cache: %v", f.writes)
	}
}
func TestCouchTitlesWorkspaceUnknownListingDefers(t *testing.T) {
	f := &fakeCouchTitles{facts: map[string]couchTitleFacts{"a": {Ready: true, PaneID: "1", Activity: time.Unix(1000, 0)}}, polls: map[string]int{}}
	u := newCouchTitleUpdater("unused", "unused", "workspace", func(context.Context, couchcore.TitleTarget) bool { return true })
	u.io = f
	row := couchTitleRow("a")
	row.TitleTarget.Sessions = nil
	u.Update(context.Background(), []couchcore.ActionableThreadSummary{row})
	if len(f.writes) != 1 {
		t.Fatalf("unknown listing renamed workspace: %v", f.writes)
	}
}
func TestCouchTitlesEffectFairness(t *testing.T) {
	f := &fakeCouchTitles{facts: map[string]couchTitleFacts{"a": {Ready: true, PaneID: "1", Activity: time.Unix(1000, 0)}, "b": {Ready: true, PaneID: "1", Activity: time.Unix(1000, 0)}}, polls: map[string]int{}, fail: true}
	u := newCouchTitleUpdater("unused", "unused", "", func(context.Context, couchcore.TitleTarget) bool { return true })
	u.io = f
	rows := []couchcore.ActionableThreadSummary{couchTitleRow("a"), couchTitleRow("b")}
	ctx, cancel := context.WithCancel(context.Background())
	f.during = cancel
	u.Update(ctx, rows)
	if len(f.writes) != 1 {
		t.Fatalf("writes after cancellation=%v", f.writes)
	}
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()
	f.during = cancel
	u.Update(ctx, rows)
	if len(f.writes) != 2 || f.writes[0] == f.writes[1] {
		t.Fatalf("first slow effect monopolized passes: %v", f.writes)
	}
}
func TestCouchTitlesRetainedHeap1000Ticks(t *testing.T) {
	if os.Getenv("PAIR_TITLE_HEAP_CHILD") != "1" {
		cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestCouchTitlesRetainedHeap1000Ticks$", "-test.count=1", "-test.v")
		cmd.Env = append(os.Environ(), "PAIR_TITLE_HEAP_CHILD=1")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("heap subprocess: %v\n%s", err, out)
		}
		t.Logf("isolated heap regression:\n%s", out)
		return
	}
	f := &fakeCouchTitles{facts: map[string]couchTitleFacts{"a": {Ready: true, PaneID: "1", Activity: time.Unix(1000, 0)}, "b": {Ready: true, PaneID: "1", Activity: time.Unix(1000, 0)}}, polls: map[string]int{}}
	u := newCouchTitleUpdater("unused", "unused", "", func(context.Context, couchcore.TitleTarget) bool { return true })
	u.io = f
	rows := []couchcore.ActionableThreadSummary{couchTitleRow("a"), couchTitleRow("b")}
	u.Update(context.Background(), rows)

	runtime.GC()
	var before, sample runtime.MemStats
	runtime.ReadMemStats(&before)
	t.Logf("baseline tick=0 heap_alloc=%d targets=%d writes=%d", before.HeapAlloc, len(u.entries), len(f.writes))
	expectedWrites := 2
	for tick := 1; tick <= 1000; tick++ {
		turnover := tick%100 == 0
		if turnover {
			for i := range rows {
				rows[i].TitleTarget.LaunchOrdinal++
			}
			expectedWrites += len(rows)
		}
		previousWrites := len(f.writes)
		u.Update(context.Background(), rows)
		if len(u.entries) != 2 {
			t.Fatalf("tick=%d retained generations=%d", tick, len(u.entries))
		}
		if len(f.writes) != expectedWrites {
			t.Fatalf("tick=%d turnover=%t writes=%d want=%d", tick, turnover, len(f.writes), expectedWrites)
		}
		if !turnover && len(f.writes) != previousWrites {
			t.Fatalf("stable tick=%d emitted extra writes", tick)
		}
		if turnover {
			runtime.GC()
			runtime.ReadMemStats(&sample)
			t.Logf("sample tick=%d ordinal=%d heap_alloc=%d targets=%d writes=%d", tick, rows[0].TitleTarget.LaunchOrdinal, sample.HeapAlloc, len(u.entries), len(f.writes))
			if sample.HeapAlloc > before.HeapAlloc+4<<20 {
				t.Fatalf("tick=%d retained heap delta=%d", tick, sample.HeapAlloc-before.HeapAlloc)
			}
		}
	}
	runtime.GC()
	runtime.ReadMemStats(&sample)
	t.Logf("post tick=1000 heap_alloc=%d targets=%d writes=%d", sample.HeapAlloc, len(u.entries), len(f.writes))
	if sample.HeapAlloc > before.HeapAlloc+4<<20 {
		t.Fatalf("post retained heap delta=%d", sample.HeapAlloc-before.HeapAlloc)
	}

	runtime.KeepAlive(u)
}

func TestCouchTitlesWritesBeforeNextSlowProjection(t *testing.T) {
	f := &fakeCouchTitles{facts: map[string]couchTitleFacts{"a": {Ready: true, PaneID: "1", Activity: time.Unix(1000, 0)}, "b": {Ready: true, PaneID: "1", Activity: time.Unix(1000, 0)}}, polls: map[string]int{}}
	u := newCouchTitleUpdater("unused", "unused", "", func(context.Context, couchcore.TitleTarget) bool { return true })
	u.io = f
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	f.pollHook = func(scope string) {
		if scope == "b" {
			cancel()
		}
	}
	u.Update(ctx, []couchcore.ActionableThreadSummary{couchTitleRow("a"), couchTitleRow("b")})
	if len(f.writes) != 1 {
		t.Fatalf("next slow projection starved earlier useful effects: %v", f.writes)
	}
}

type couchNativeWork struct{ read, decode int64 }

func (w *couchNativeWork) ReadWork(a sessioninventory.Artifact, n int64) {
	if a.StorageRoot == "codex-sessions" {
		w.read += n
	}
}
func (w *couchNativeWork) DecodeWork(a sessioninventory.Artifact, n int64) {
	if a.StorageRoot == "codex-sessions" {
		w.decode += n
	}
}

type observedOSTitles struct {
	*osCouchTitles
	work couchNativeWork
}

func (o *observedOSTitles) Open(t couchcore.TitleTarget) (couchTitleSource, error) {
	src, err := o.osCouchTitles.Open(t)
	if err != nil {
		return nil, err
	}
	s := src.(*osCouchTitleSource)
	s.runtime = s.runtime.WithWorkObserver(&o.work)
	s.projection = sessioninventory.NewTitleProjection(s.runtime, t.Address.RepoScope, string(t.Address.Tag), sessioninventory.Agent(t.Agent), t.LaunchOrdinal)
	return s, nil
}
func titleFixtureWrite(t *testing.T, path string, raw []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
}
func TestCouchTitlesOSProjectionToScopedCommand(t *testing.T) {
	root := t.TempDir()
	testpair.ApplyEnvironment(t, root)
	home := os.Getenv("HOME")
	data := filepath.Join(root, "pair-data")
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(root, "commands")
	t.Setenv("PAIR_TITLE_COMMAND_LOG", log)
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
	if err := os.WriteFile(filepath.Join(bin, "zellij"), []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" >> \"$PAIR_TITLE_COMMAND_LOG\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	row := couchTitleRow("scope")
	row.Address.Tag = "work"
	row.TitleTarget.Address = row.Address
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: data, RepoScope: "scope", Tag: "work"})
	if err != nil {
		t.Fatal(err)
	}
	const id = "019d1111-1111-7111-8111-111111111111"
	native := filepath.Join(home, ".codex", "sessions", "2026", "10", "16", "rollout-test-"+id+".jsonl")
	titleFixtureWrite(t, native, []byte(`{"timestamp":"2026-10-16T00:00:00Z","type":"session_meta","payload":{"id":"`+id+`","parent_thread_id":null,"source":"cli"}}`+"\n"+`{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":42000}}}}`+"\n"))
	var ledger []byte
	for _, r := range []sessionledger.Record{
		{Version: 3, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "codex", RequestedNativeID: id, RequestOrigin: sessionledger.RequestOriginChosen, BaselineComplete: true, LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{}},
		{Version: 3, Kind: sessionledger.RecordBinding, ScopeKey: "scope", Tag: "work", Agent: "codex", LaunchOrdinal: 1, RootNativeID: id, ConfirmationReason: sessionledger.ConfirmationChosen},
	} {
		raw, e := sessionledger.EncodeRecord(r)
		if e != nil {
			t.Fatal(e)
		}
		ledger = append(ledger, append(raw, '\n')...)
	}
	titleFixtureWrite(t, paths.Ledger(), ledger)
	// A stale twin cannot establish selected-agent birth.
	titleFixtureWrite(t, paths.Pane("claude"), []byte(`{"pane_id":"99"}`))
	u := newCouchTitleUpdater(data, home, "", func(context.Context, couchcore.TitleTarget) bool { return true })
	observed := &observedOSTitles{osCouchTitles: u.io.(*osCouchTitles)}
	u.io = observed
	u.Update(t.Context(), []couchcore.ActionableThreadSummary{row})
	if _, e := os.Stat(log); !errors.Is(e, os.ErrNotExist) {
		t.Fatalf("unborn selected pane ran command: %v", e)
	}
	titleFixtureWrite(t, paths.Pane("codex"), []byte(`{"pane_id":"7"}`))
	_, failed := u.Update(t.Context(), []couchcore.ActionableThreadSummary{row})
	raw, e := os.ReadFile(log)
	if e != nil {
		t.Fatalf("production projection emitted no command (failed=%v): %v", failed, e)
	}
	want := []string{"--session", row.TitleTarget.Session, "action", "rename-pane", "--pane-id", "7", "codex (42k)"}
	if got := strings.Split(strings.TrimSpace(string(raw)), "\n"); !equalTitleArgs(got, want) {
		t.Fatalf("argv=%q want=%q", got, want)
	}
	if observed.work.decode == 0 {
		t.Fatal("fixture did not exercise native decoding")
	}
	before := observed.work
	for i := 0; i < 20; i++ {
		u.Update(t.Context(), []couchcore.ActionableThreadSummary{row})
	}
	if observed.work != before {
		t.Fatalf("warm source reread/decoded native history: before=%+v after=%+v", before, observed.work)
	}
	after, e := os.ReadFile(log)
	if e != nil || string(after) != string(raw) {
		t.Fatalf("warm passes repeated commands: %q %v", after, e)
	}
	// Exact-generation replacement creates a fresh compact source and write.
	row.TitleTarget.StartNonce = "replacement"
	u.Update(t.Context(), []couchcore.ActionableThreadSummary{row})
	if len(u.entries) != 1 {
		t.Fatalf("superseded projection retained: %d", len(u.entries))
	}
}
func equalTitleArgs(a, b []string) bool {
	aa, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(aa) == string(bb)
}
func TestCouchTitlesOSCommandCancellationCollectsGroup(t *testing.T) {
	root := t.TempDir()
	testpair.ApplyEnvironment(t, root)
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	pidFile := filepath.Join(root, "child.pid")
	t.Setenv("PAIR_TITLE_CHILD_PID", pidFile)
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
	script := `#!/bin/sh
/bin/sleep 30 & child=$!
printf '%s' "$child" > "$PAIR_TITLE_CHILD_PID"
wait
`
	if err := os.WriteFile(filepath.Join(bin, "zellij"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- (&osCouchTitles{}).RenamePane(ctx, *couchTitleRow("a").TitleTarget, "7", "codex") }()
	var pid int
	testwait.Until(t, "title command child started", func() bool {
		raw, err := os.ReadFile(pidFile)
		if err != nil {
			return false
		}
		pid, _ = strconv.Atoi(string(raw))
		return pid > 0
	})
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("canceled rename succeeded")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("title command did not join")
	}
	testwait.Until(t, "title command group collected", func() bool { return errors.Is(syscall.Kill(pid, 0), syscall.ESRCH) })
}
