package sessionwatch

import (
	"path/filepath"
	"testing"
	"time"

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
