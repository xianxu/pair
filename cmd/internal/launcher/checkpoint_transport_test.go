package launcher

import (
	"bytes"
	"errors"
	"github.com/xianxu/pair/cmd/internal/checkpoint"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckpointStandaloneRetryUsesRetainedSnapshot(t *testing.T) {
	rt := newFakeRuntime()
	c := testCheckpoint(t)
	m := RestartMarker{Version: 1, Attempt: "retry-1", Tag: "work", Agent: "claude", NewSession: true, Checkpoint: c}
	rt.restartMarkers["📁work"] = m
	rt.sessionIndex = SessionNameIndex{Entries: []SessionNameEntry{{SessionName: "📁work", Tag: "work", ScopeKey: "scope"}}}
	opts := baseOpts(LaunchArgs{})
	if err := prepareContinuationRetry(&opts, rt, "work"); err != nil {
		t.Fatal(err)
	}
	code, _ := run(t, opts, rt)
	if code != 0 || rt.launchCount != 1 || len(rt.restartMarkers) != 0 || !strings.Contains(rt.files["/data/draft-work.md"], c.Body) {
		t.Fatalf("retry: code %d launches %d markers %+v", code, rt.launchCount, rt.restartMarkers)
	}
	rt.restartMarkers["📁work"] = m
	rt.sessions = []Session{{Name: "📁work", State: SessionDetached}}
	if err := prepareContinuationRetry(&opts, rt, "work"); err == nil {
		t.Fatal("retry accepted occupied session")
	}
}

func TestCheckpointMarkerFailureKeepsSourceAlive(t *testing.T) {
	rt := newFakeRuntime()
	rt.markerWriteErr = errors.New("disk full")
	code, _ := run(t, compactOpts(false, true, "pair-demo"), rt)
	if code != 1 || len(rt.killed)+len(rt.touchedQuit)+len(rt.parked) != 0 {
		t.Fatal("failed publication changed source")
	}
}

func TestCheckpointStandaloneRetryPreservesDurableAgentArgs(t *testing.T) {
	rt := newFakeRuntime()
	rt.files["/data/config-demo-claude.json"] = `{"agent":"claude","args":["--model","special model","--resume","old-native"],"session_id":"old-native"}`
	if code, _ := run(t, compactOpts(false, true, "pair-demo"), rt); code != 0 {
		t.Fatal("compaction failed")
	}
	marker := rt.writtenMarkers["pair-demo"]
	if strings.Join(marker.AgentArgs, "|") != "--model|special model" {
		t.Fatalf("lost fresh args: %+v", marker.AgentArgs)
	}
	plan := planRestart(marker, "demo", "claude", savedConfig{}) // original config is gone after failed replacement
	if strings.Join(plan.Args.AgentArgs, "|") != "--model|special model" {
		t.Fatalf("retry lost args: %+v", plan)
	}
}

func TestCheckpointKillFailureIsActionable(t *testing.T) {
	rt := newFakeRuntime()
	rt.killErr = errors.New("kill executable unavailable")
	var diagnostic bytes.Buffer
	code, _ := RunLaunch(compactOpts(false, true, "pair-demo"), rt, &diagnostic)
	if code != 1 || !strings.Contains(diagnostic.String(), "--retry demo") || len(rt.writtenMarkers) != 1 {
		t.Fatalf("kill failure: %d %s", code, diagnostic.String())
	}
}

func TestCheckpointMarkerAcknowledgmentPreservesSuccessor(t *testing.T) {
	_, cache := mkCacheDir(t)
	rt := NewOSRuntime(t.TempDir(), "/pair")
	m := RestartMarker{Version: 1, Attempt: "first", Tag: "work", Agent: "claude", NewSession: true, Checkpoint: testCheckpoint(t)}
	if err := rt.WriteRestartMarker("pair-work", m); err != nil {
		t.Fatal(err)
	}
	loaded, present, err := rt.ReadRestartMarker("pair-work")
	if err != nil || !present || !sameRestartMarker(loaded, m) {
		t.Fatalf("read %v %v %+v", err, present, loaded)
	}
	newer := m
	newer.Attempt = "second"
	if err := rt.WriteRestartMarker("pair-work", newer); err != nil {
		t.Fatal(err)
	}
	if err := rt.AcknowledgeRestartMarker("pair-work", m); err != nil {
		t.Fatal(err)
	}
	loaded, present, err = rt.ReadRestartMarker("pair-work")
	if err != nil || !present || !sameRestartMarker(loaded, newer) {
		t.Fatalf("ack erased successor: %v %+v", err, loaded)
	}
	if err := rt.AcknowledgeRestartMarker("pair-work", newer); err != nil {
		t.Fatal(err)
	}
	if _, present, err = rt.ReadRestartMarker("pair-work"); err != nil || present {
		t.Fatal("matching ack retained marker")
	}
	if err := os.WriteFile(filepath.Join(cache, "restart-pair-work"), []byte("{\"version\":1,"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := rt.ReadRestartMarker("pair-work"); err == nil {
		t.Fatal("accepted truncated marker")
	}
}

func TestCheckpointMalformedMarkerRefusesFreshFallback(t *testing.T) {
	for _, raw := range []string{"garbage\n", "new_session=1\ncontinue", "continue=demo\n", "tag=work\ntag=other\nagent=claude\n"} {
		if _, err := decodeRestartMarker(raw); err == nil {
			t.Fatalf("accepted malformed restart marker %q", raw)
		}
	}
	m := RestartMarker{Version: 1, Attempt: "attempt\nother", Tag: "work", Agent: "claude", NewSession: true, Checkpoint: testCheckpoint(t)}
	if _, err := decodeRestartMarker(serializeRestartMarker(m)); err == nil {
		t.Fatal("accepted control character in attempt")
	}
}

func TestCheckpointHostedRestartAndRenameRefuseBeforeMutation(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))
	t.Setenv("PAIR_DATA_DIR", filepath.Join(home, "data"))
	t.Setenv("COUCH_THREAD_SCOPE", "0123456789abcdef")
	t.Setenv("COUCH_THREAD_TAG", "work")
	t.Setenv("ZELLIJ_SESSION_NAME", "pair-work")
	t.Setenv("PAIR_COUCH_LAUNCH_PROFILE", "")
	for _, args := range [][]string{{"restart"}, {"restart", "--new-session"}, {"restart", "--rename-to", "new"}, {"rename", "work", "new"}} {
		var out, diagnostic bytes.Buffer
		code, err := LaunchNative(args, "/pair", &out, &diagnostic)
		if err != nil || code != 1 || !strings.Contains(diagnostic.String(), "Couch") {
			t.Fatalf("refusal %v: %d %v %s", args, code, err, diagnostic.String())
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".cache", "pair")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("refusal created markers: %v", err)
	}
}

func TestCheckpointCouchSubprocessUsesInstalledBinaryAndExactArgv(t *testing.T) {
	bin := t.TempDir()
	capture := filepath.Join(bin, "captured")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$PAIR249_CAPTURE\"\n"
	if err := os.WriteFile(filepath.Join(bin, "couch"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	t.Setenv("PAIR249_CAPTURE", capture)
	rt := NewOSRuntime(t.TempDir(), "/runtime-assets-with-no-couch-binary")
	path := "/other worktree/a $(literal) checkpoint.md"
	if err := rt.RequestCouchContinuation(path); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "--internal\nrequest-continuation\n"+path+"\n" {
		t.Fatalf("wrong argv: %q", raw)
	}
}

type checkpointClaimRuntime struct {
	*fakeRuntime
	global string
}

func (r checkpointClaimRuntime) EnsureThreadAddress(scope RepoScope, tag string, owned bool) error {
	return EnsureThreadAddressForPair(r.global, scope, tag, owned)
}
func (r checkpointClaimRuntime) RegisterExistingCouchThread(scope RepoScope, tag string) error {
	return RegisterExistingCouchThread(r.global, scope, tag)
}

func TestCheckpointFreshReplacementUsesRealEstablishedClaim(t *testing.T) {
	for _, established := range []bool{false, true} {
		t.Run(map[bool]string{false: "reserved", true: "established"}[established], func(t *testing.T) {
			global := t.TempDir()
			scope, err := ResolveRepoScope("/home/u/work")
			if err != nil {
				t.Fatal(err)
			}
			claim, err := ClaimNewThreadAddress(global, scope, "work")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = claim.Release() })
			if established {
				if err := EnsureThreadAddressForPair(global, scope, "work", true); err != nil {
					t.Fatal(err)
				}
			}
			fake := newFakeRuntime()
			rt := checkpointClaimRuntime{fake, global}
			opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work", FreshRequired: true})
			opts.ContinueCheckpoint = testCheckpoint(t)
			opts.Env.CouchThreadScope = scope.Key
			opts.Env.CouchThreadTag = "work"
			var diagnostic bytes.Buffer
			code, err := RunLaunch(opts, rt, &diagnostic)
			if err != nil || (code == 0) != established || fake.launchCount != map[bool]int{false: 0, true: 1}[established] {
				t.Fatalf("claim result %d %v launches %d: %s", code, err, fake.launchCount, diagnostic.String())
			}
		})
	}
}

func testCheckpoint(t *testing.T) checkpoint.Checkpoint {
	t.Helper()
	c, err := checkpoint.New("/other-worktree/workshop/continuation/exact.md", "---\ntype: continuation\nagent: claude\n---\n## NEXT ACTION\nEXACT-CHECKPOINT-TOKEN\n")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestCheckpointContinueArguments(t *testing.T) {
	a, err := ParseArgs([]string{"continue", "--checkpoint", "/other/exact.md"})
	if err != nil || a.ContinueCheckpoint != "/other/exact.md" || a.ContinueSlug != "" {
		t.Fatalf("exact args: %+v %v", a, err)
	}
	a, err = ParseArgs([]string{"continue", "--retry", "work"})
	if err != nil || a.ContinueRetry != "work" {
		t.Fatalf("retry args: %+v %v", a, err)
	}
	for _, args := range [][]string{{"continue", "--checkpoint"}, {"continue", "--retry"}, {"continue", "--unknown", "x"}} {
		if _, err := ParseArgs(args); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestCheckpointHostedCompactionPublishesWithoutKill(t *testing.T) {
	rt := newFakeRuntime()
	opts := compactOpts(false, true, "pair-demo")
	opts.Env.CouchThreadScope = "scope"
	opts.Env.CouchThreadTag = "demo"
	opts.ContinueCheckpoint = testCheckpoint(t)
	opts.ContinueDoc = opts.ContinueCheckpoint.SourcePath
	code, err := run(t, opts, rt)
	if err != nil || code != 0 || len(rt.couchContinuations) != 1 || rt.couchContinuations[0] != opts.ContinueDoc {
		t.Fatalf("publish %d %v %+v", code, err, rt.couchContinuations)
	}
	if len(rt.killed)+len(rt.writtenMarkers)+len(rt.touchedQuit)+len(rt.parked) != 0 {
		t.Fatal("hosted publication performed teardown")
	}
	if rt.env[checkpoint.DigestEnv] != opts.ContinueCheckpoint.Digest {
		t.Fatal("publication lost validated checkpoint digest")
	}
	rt.couchContinuationErr = errors.New("refused")
	code, _ = run(t, opts, rt)
	if code != 1 || len(rt.killed) != 0 {
		t.Fatal("request refusal killed source or reported success")
	}
}

func TestCheckpointCLIRejectsChangedCommittedBytes(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, "checkpoint.md")
	original := testCheckpoint(t)
	if err := os.WriteFile(path, []byte(strings.ReplaceAll(original.Body, "EXACT-CHECKPOINT-TOKEN", "CHANGED")), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))
	t.Setenv("PAIR_DATA_DIR", filepath.Join(home, "data"))
	t.Setenv(CouchLaunchProfileEnv, "")
	t.Setenv(checkpoint.DigestEnv, original.Digest)
	var out, diagnostic bytes.Buffer
	code, err := LaunchNative([]string{"continue", "--checkpoint", path}, "/pair", &out, &diagnostic)
	if err != nil || code != 1 || !strings.Contains(diagnostic.String(), "changed since the writer") {
		t.Fatalf("changed checkpoint %d %v %s", code, err, diagnostic.String())
	}
	if os.Getenv(checkpoint.DigestEnv) != "" {
		t.Fatal("digest leaked into subsequent unrelated launch")
	}
}

func TestCheckpointStandaloneFailedReplacementRetainsMarker(t *testing.T) {
	rt := newFakeRuntime()
	c := testCheckpoint(t)
	m := RestartMarker{Version: 1, Attempt: "attempt-1", Tag: "work", Agent: "claude", NewSession: true, Checkpoint: c}
	rt.restartMarkers["📁work"] = m
	rt.quitMarkers["📁work"] = true
	rt.launchHook = func(count int) {
		if count == 2 {
			rt.launchErr = errors.New("replacement refused")
		}
	}
	code, _ := run(t, baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work"}), rt)
	if code != 1 || rt.launchCount != 2 {
		t.Fatalf("failed replacement: code %d count %d", code, rt.launchCount)
	}
	if got, ok := rt.restartMarkers["📁work"]; !ok || !sameRestartMarker(got, m) {
		t.Fatalf("lost pending marker: %+v", got)
	}
	if draft := rt.files["/data/draft-work.md"]; !strings.Contains(draft, c.Body) || !strings.Contains(draft, c.Digest) {
		t.Fatalf("wrong seed: %q", draft)
	}
}

func TestCheckpointStandaloneDraftFailureStopsLaunch(t *testing.T) {
	rt := newFakeRuntime()
	opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "work"})
	opts.ContinueCheckpoint = testCheckpoint(t)
	rt.writeFailAt = "/data/draft-work.md"
	code, _ := run(t, opts, rt)
	if code != 1 || rt.launchCount != 0 {
		t.Fatalf("draft failure launched: %d %d", code, rt.launchCount)
	}
}

func (r checkpointClaimRuntime) RegisterFreshCouchThread(scope RepoScope, tag string) error {
	return RegisterFreshCouchThread(r.global, scope, tag)
}
