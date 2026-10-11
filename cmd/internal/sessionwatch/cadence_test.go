package sessionwatch

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/adapt"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

func TestScanDelayFollowsLatestSendNotLaunch(t *testing.T) {
	t.Parallel()
	opts := Options{Timeout: time.Minute, Poll: 100 * time.Millisecond, ActivePoll: time.Second, ActiveWindow: 30 * time.Minute, SlowPoll: time.Minute}
	start := time.Unix(1000, 0)
	for _, tc := range []struct {
		name     string
		now      time.Time
		lastSend time.Time
		want     time.Duration
	}{
		{"startup window", start.Add(30 * time.Second), time.Time{}, opts.Poll},
		{"startup window wins over a send", start.Add(30 * time.Second), start.Add(20 * time.Second), opts.Poll},
		{"no send after startup stays slow", start.Add(10 * time.Minute), time.Time{}, opts.SlowPoll},
		{"send after startup is active", start.Add(10 * time.Minute), start.Add(9 * time.Minute), opts.ActivePoll},
		{"a send inside startup still arms the window after it", start.Add(20 * time.Minute), start.Add(50 * time.Second), opts.ActivePoll},
		{"window expired falls back to slow", start.Add(2 * time.Hour), start.Add(10 * time.Minute), opts.SlowPoll},
	} {
		if got := scanDelay(tc.now, start, tc.lastSend, opts); got != tc.want {
			t.Errorf("%s: scanDelay = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// #316: the operator sends only after the startup window has closed. The round
// completing must bind within one ActivePoll, not after a SlowPoll.
func TestRunBindsPromptlyWhenFirstSendFollowsStartupWindow(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	native := sessioninventorytest.NewFakeRuntime()
	root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentClaude, Name: "claude-projects", Path: "/home/.claude/projects"}
	native.AddRoot(root)
	sid := "d08af0b0-b6c1-4d7d-bb3d-f20ddffbe384"
	relative := "-Users-x-workspace-pair/" + sid + ".jsonl"
	artifact := sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: relative, Kind: sessioninventory.ArtifactTranscript}
	text := "why can't I alt+n on a new pair thread?"
	native.SetProcess("1234", "native-identity", nil, []string{filepath.Join(root.Path, filepath.FromSlash(relative))})

	paths := mustScopedPaths(t, dataDir, "work")
	runtime := newWatcherRuntime(native)
	runtime.files[paths.Ledger()] = mustLaunchRecord(t, sessionledger.Record{Version: 2, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "claude", LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{}})
	runtime.files[paths.AgentPID()] = []byte("1234\n")
	runtime.modTimes[paths.AgentPID()] = runtime.now
	runtime.identities["1234"] = "pair-identity"

	opts := Options{Agent: "claude", Tag: "work", ScopeKey: "scope", LaunchOrdinal: 1, Home: "/home", DataDir: dataDir, PIDWait: time.Second,
		Timeout: time.Second, Poll: 100 * time.Millisecond, ActivePoll: time.Second, ActiveWindow: 30 * time.Minute, SlowPoll: time.Minute}
	// Off the slice grid, and stamped when it happens: a fake sleep is atomic,
	// so an event it crosses occurred at sendAt, not at the wake-up.
	sendAt := runtime.now.Add(opts.Timeout + 5500*time.Millisecond)
	runtime.onSleep = func() {
		if runtime.now.Before(sendAt) {
			return
		}
		runtime.files[paths.Log()] = []byte("## 2026-09-23 21:48:01\n\n" + text + "\n\n---\n\n")
		runtime.modTimes[paths.Log()] = sendAt
		native.PutFile(sessioninventory.FileEntry{Artifact: artifact, StableFileID: "stable", MutationToken: "ctime:1"}, []byte(
			`{"type":"user","sessionId":"`+sid+`","isSidechain":false,"message":{"role":"user","content":[{"type":"text","text":"`+text+`"}]}}`+"\n"+
				`{"type":"assistant","sessionId":"`+sid+`","message":{"role":"assistant","content":[{"type":"text","text":"because"}]}}`+"\n"))
		runtime.onSleep = nil
	}
	if err := Run(opts, runtime); err != nil {
		t.Fatal(err)
	}
	if len(runtime.store.records) != 1 || runtime.store.records[0].RootNativeID != sid || runtime.store.records[0].AuthorizationProof == nil {
		t.Fatalf("records=%#v", runtime.store.records)
	}
	if lag := runtime.now.Sub(sendAt); lag > opts.ActivePoll {
		t.Fatalf("bound %v after the round completed, want within ActivePoll %v", lag, opts.ActivePoll)
	}
}

func TestIdleExpiredCountsFromLaterOfLaunchAndSend(t *testing.T) {
	t.Parallel()
	start := time.Unix(1000, 0)
	day := 24 * time.Hour
	for _, tc := range []struct {
		name     string
		now      time.Time
		lastSend time.Time
		want     bool
	}{
		{"fresh launch", start.Add(time.Hour), time.Time{}, false},
		{"a day with no send", start.Add(day), time.Time{}, true},
		{"a recent send keeps it", start.Add(day + time.Hour), start.Add(2 * time.Hour), false},
		{"a day after the last send", start.Add(2*day + time.Hour), start.Add(day + time.Hour), true},
	} {
		if got := idleExpired(tc.now, start, tc.lastSend, day); got != tc.want {
			t.Errorf("%s: idleExpired = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// #420: an unbound watcher whose thread is never sent to must not walk or
// validate the native store, which other sessions keep growing, and it must
// end after IdleExpiry instead of polling for the agent's whole life.
func TestRunUnsentWatcherSkipsNativeStoreAndExpires(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	native := sessioninventorytest.NewFakeRuntime()
	root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentClaude, Name: "claude-projects", Path: "/home/.claude/projects"}
	native.AddRoot(root)
	paths := mustScopedPaths(t, dataDir, "work")
	runtime := newWatcherRuntime(native)
	runtime.files[paths.Ledger()] = mustLaunchRecord(t, sessionledger.Record{Version: 2, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "claude", LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{}})
	runtime.files[paths.AgentPID()] = []byte("1234\n")
	runtime.modTimes[paths.AgentPID()] = runtime.now
	runtime.identities["1234"] = "pair-identity"
	// Another session's transcript, written after this launch.
	foreign := sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "-Users-x-other/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa.jsonl", Kind: sessioninventory.ArtifactTranscript}
	stamp := runtime.now
	native.PutFile(sessioninventory.FileEntry{Artifact: foreign, StableFileID: "foreign", MutationToken: "ctime:1", ModTime: &stamp}, []byte(`{"type":"user","sessionId":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","message":{"role":"user","content":"hi"}}`+"\n"))

	start := runtime.now
	opts := Options{Agent: "claude", Tag: "work", ScopeKey: "scope", LaunchOrdinal: 1, Home: "/home", DataDir: dataDir, PIDWait: time.Second,
		Timeout: time.Second, Poll: 100 * time.Millisecond, ActivePoll: time.Second, ActiveWindow: 30 * time.Minute, SlowPoll: time.Minute, IdleExpiry: 2 * time.Hour}
	if err := Run(opts, runtime); err != nil {
		t.Fatal(err)
	}
	if elapsed := runtime.now.Sub(start); elapsed < opts.IdleExpiry || elapsed > opts.IdleExpiry+2*opts.SlowPoll {
		t.Fatalf("watcher exited after %v, want about IdleExpiry %v", elapsed, opts.IdleExpiry)
	}
	if got := native.OperationCount(sessioninventorytest.OperationListFiles, root.Name); got != 0 {
		t.Fatalf("unsent watcher walked the native store %d times", got)
	}
	key := root.Name + ":" + foreign.RelativePath
	if got := native.OperationCount(sessioninventorytest.OperationReadFile, key) + native.OperationCount(sessioninventorytest.OperationReadAt, key); got != 0 {
		t.Fatalf("unsent watcher read another session's transcript %d times", got)
	}
	if !runtime.hasLog(adapt.NearMiss) {
		t.Fatalf("logs=%#v, want the idle-expiry near miss", runtime.logs)
	}
}

// #420: once a turn is pending, a transcript last written before the watcher's
// last quiet moment cannot hold the round, so it is never validated; the
// session's own fresh transcript still binds.
func TestRunPendingTurnSkipsCandidatesOlderThanQuietFloor(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	native := sessioninventorytest.NewFakeRuntime()
	root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentClaude, Name: "claude-projects", Path: "/home/.claude/projects"}
	native.AddRoot(root)
	sid := "d08af0b0-b6c1-4d7d-bb3d-f20ddffbe384"
	relative := "-Users-x-workspace-pair/" + sid + ".jsonl"
	artifact := sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: relative, Kind: sessioninventory.ArtifactTranscript}
	text := "first turn after a long idle"
	native.SetProcess("1234", "native-identity", nil, []string{filepath.Join(root.Path, filepath.FromSlash(relative))})

	paths := mustScopedPaths(t, dataDir, "work")
	runtime := newWatcherRuntime(native)
	runtime.files[paths.Ledger()] = mustLaunchRecord(t, sessionledger.Record{Version: 2, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "claude", LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{}})
	runtime.files[paths.AgentPID()] = []byte("1234\n")
	runtime.modTimes[paths.AgentPID()] = runtime.now
	runtime.identities["1234"] = "pair-identity"
	foreign := sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "-Users-x-other/aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa.jsonl", Kind: sessioninventory.ArtifactTranscript}
	foreignStamp := runtime.now.Add(time.Minute)
	native.PutFile(sessioninventory.FileEntry{Artifact: foreign, StableFileID: "foreign", MutationToken: "ctime:1", ModTime: &foreignStamp}, []byte(`{"type":"user","sessionId":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","message":{"role":"user","content":"hi"}}`+"\n"))

	opts := Options{Agent: "claude", Tag: "work", ScopeKey: "scope", LaunchOrdinal: 1, Home: "/home", DataDir: dataDir, PIDWait: time.Second,
		Timeout: time.Second, Poll: 100 * time.Millisecond, ActivePoll: time.Second, ActiveWindow: 30 * time.Minute, SlowPoll: time.Minute}
	sendAt := runtime.now.Add(10*time.Minute + 500*time.Millisecond)
	runtime.onSleep = func() {
		if runtime.now.Before(sendAt) {
			return
		}
		runtime.files[paths.Log()] = []byte("## 2026-09-23 21:48:01\n\n" + text + "\n\n---\n\n")
		runtime.modTimes[paths.Log()] = sendAt
		stamp := sendAt
		native.PutFile(sessioninventory.FileEntry{Artifact: artifact, StableFileID: "stable", MutationToken: "ctime:1", ModTime: &stamp}, []byte(
			`{"type":"user","sessionId":"`+sid+`","isSidechain":false,"message":{"role":"user","content":[{"type":"text","text":"`+text+`"}]}}`+"\n"+
				`{"type":"assistant","sessionId":"`+sid+`","message":{"role":"assistant","content":[{"type":"text","text":"ok"}]}}`+"\n"))
		runtime.onSleep = nil
	}
	if err := Run(opts, runtime); err != nil {
		t.Fatal(err)
	}
	if len(runtime.store.records) != 1 || runtime.store.records[0].RootNativeID != sid {
		t.Fatalf("records=%#v", runtime.store.records)
	}
	key := root.Name + ":" + foreign.RelativePath
	if got := native.OperationCount(sessioninventorytest.OperationReadFile, key) + native.OperationCount(sessioninventorytest.OperationReadAt, key); got != 0 {
		t.Fatalf("validated a transcript older than the quiet floor %d times", got)
	}
}

// #420 review: a fresh Claude launch is chosen-id, so an idle chosen watcher
// must not walk the native store each SlowPoll. It walks once, then again only
// after the agent pane writes output, and its first round still binds through
// the filename handshake.
func TestRunIdleChosenWatcherWalksOnlyAfterAgentOutput(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	native := sessioninventorytest.NewFakeRuntime()
	root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentClaude, Name: "claude-projects", Path: "/home/.claude/projects"}
	native.AddRoot(root)
	const sid = "11111111-1111-4111-8111-111111111111"
	paths := mustScopedPaths(t, dataDir, "work")
	runtime := newWatcherRuntime(native)
	runtime.files[paths.Ledger()] = mustLaunchRecord(t, sessionledger.Record{Version: 3, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "claude", RequestedNativeID: sid, RequestOrigin: sessionledger.RequestOriginChosen, BaselineComplete: true})
	runtime.files[paths.AgentPID()] = []byte("1234\n")
	runtime.modTimes[paths.AgentPID()] = runtime.now
	runtime.identities["1234"] = "pair-identity"
	runtime.modTimes[paths.ScrollbackRaw("claude")] = runtime.now

	opts := Options{Agent: "claude", Tag: "work", ScopeKey: "scope", LaunchOrdinal: 1, Home: "/home", DataDir: dataDir, PIDWait: time.Second,
		Timeout: time.Second, Poll: 100 * time.Millisecond, ActivePoll: time.Second, ActiveWindow: 30 * time.Minute, SlowPoll: time.Minute}
	typedAt := runtime.now.Add(3 * time.Hour)
	walksBeforeOutput := -1
	runtime.onSleep = func() {
		if runtime.now.Before(typedAt) {
			return
		}
		// The operator types straight into the agent pane: no Pair send, but
		// the agent answers and creates its chosen transcript.
		walksBeforeOutput = native.OperationCount(sessioninventorytest.OperationListFiles, root.Name)
		runtime.modTimes[paths.ScrollbackRaw("claude")] = typedAt
		native.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "-repo/" + sid + ".jsonl"}}, []byte("{}\n"))
		runtime.onSleep = nil
	}
	if err := Run(opts, runtime); err != nil {
		t.Fatal(err)
	}
	if walksBeforeOutput != 1 {
		t.Fatalf("idle chosen watcher walked the native store %d times in 3h, want once", walksBeforeOutput)
	}
	if len(runtime.store.records) != 1 || runtime.store.records[0].ConfirmationReason != "chosen-id" {
		t.Fatalf("records=%+v", runtime.store.records)
	}
}

// #420 review: agent output counts as activity for the idle expiry, so a pane
// someone types into directly keeps its watcher.
func TestRunAgentOutputDefersIdleExpiry(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	native := sessioninventorytest.NewFakeRuntime()
	native.AddRoot(sessioninventory.StorageRoot{Agent: sessioninventory.AgentClaude, Name: "claude-projects", Path: "/home/.claude/projects"})
	paths := mustScopedPaths(t, dataDir, "work")
	runtime := newWatcherRuntime(native)
	runtime.files[paths.Ledger()] = mustLaunchRecord(t, sessionledger.Record{Version: 2, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "claude", LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{}})
	runtime.files[paths.AgentPID()] = []byte("1234\n")
	runtime.modTimes[paths.AgentPID()] = runtime.now
	runtime.identities["1234"] = "pair-identity"
	start := runtime.now
	runtime.modTimes[paths.ScrollbackRaw("claude")] = start
	outputAt := start.Add(20 * time.Hour)
	runtime.onSleep = func() {
		if runtime.now.Before(outputAt) {
			return
		}
		runtime.modTimes[paths.ScrollbackRaw("claude")] = outputAt
		runtime.onSleep = nil
	}
	opts := Options{Agent: "claude", Tag: "work", ScopeKey: "scope", LaunchOrdinal: 1, Home: "/home", DataDir: dataDir, PIDWait: time.Second,
		Timeout: time.Second, Poll: 100 * time.Millisecond, ActivePoll: time.Second, ActiveWindow: 30 * time.Minute, SlowPoll: time.Minute, IdleExpiry: 24 * time.Hour}
	if err := Run(opts, runtime); err != nil {
		t.Fatal(err)
	}
	if elapsed := runtime.now.Sub(start); elapsed < 44*time.Hour || elapsed > 44*time.Hour+2*opts.SlowPoll {
		t.Fatalf("watcher exited after %v, want a day after the 20h output (about 44h)", elapsed)
	}
}

// #420 review: an epoch that never completes (the store stays unreadable)
// still ends at IdleExpiry; every sleeping unbound pass shares the funeral.
func TestRunIncompleteEpochStillExpires(t *testing.T) {
	t.Parallel()
	dataDir := t.TempDir()
	native := sessioninventorytest.NewFakeRuntime()
	root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentClaude, Name: "claude-projects", Path: "/home/.claude/projects"}
	native.AddRoot(root)
	native.SetError(sessioninventorytest.OperationListFiles, root.Name, errors.New("EIO"))
	paths := mustScopedPaths(t, dataDir, "work")
	runtime := newWatcherRuntime(native)
	runtime.files[paths.Ledger()] = mustLaunchRecord(t, sessionledger.Record{Version: 3, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "claude", RequestedNativeID: "11111111-1111-4111-8111-111111111111", RequestOrigin: sessionledger.RequestOriginChosen, BaselineComplete: false})
	runtime.files[paths.AgentPID()] = []byte("1234\n")
	runtime.modTimes[paths.AgentPID()] = runtime.now
	runtime.identities["1234"] = "pair-identity"
	start := runtime.now
	opts := Options{Agent: "claude", Tag: "work", ScopeKey: "scope", LaunchOrdinal: 1, Home: "/home", DataDir: dataDir, PIDWait: time.Second,
		Timeout: time.Second, Poll: 100 * time.Millisecond, ActivePoll: time.Second, ActiveWindow: 30 * time.Minute, SlowPoll: time.Minute, IdleExpiry: 2 * time.Hour}
	if err := Run(opts, runtime); err != nil {
		t.Fatal(err)
	}
	if elapsed := runtime.now.Sub(start); elapsed < opts.IdleExpiry || elapsed > opts.IdleExpiry+2*opts.SlowPoll {
		t.Fatalf("incomplete-epoch watcher exited after %v, want about IdleExpiry %v", elapsed, opts.IdleExpiry)
	}
}
