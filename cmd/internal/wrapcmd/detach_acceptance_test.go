package wrapcmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
	"github.com/xianxu/pair/cmd/internal/sessionwatch"
)

// The helper is the real watcher command in a child process. TestMain normally
// suppresses watcher spawning; only this acceptance test opts into execution.
func detachWatcherHelper() {
	if os.Getenv("PAIR_TEST_DETACH_WATCHER") == "" || len(os.Args) < 3 || os.Args[1] != "session-watch" {
		return
	}
	_ = os.WriteFile(os.Getenv("PAIR_TEST_DETACH_WATCHER"), []byte(strconv.Itoa(os.Getpid())), 0600)
	os.Exit(sessionwatch.RunCLI(os.Args[2:], os.Getenv, os.Stderr))
}

// TestDetachBeforeFirstTurnAuthorizesRelaunch composes real wrap startup, real
// watcher execution, Couch's process-group detach, warm resume eligibility, and
// its native-ledger relaunch preflight. Zellij's surviving server/agent session
// is represented by a stateful PairSessionIO fake; native agents are executable
// fixtures holding their real transcript open, with no completed round until
// after a replacement client attaches. No binding or authorization is stubbed.
func TestDetachBeforeFirstTurnAuthorizesRelaunch(t *testing.T) {
	for _, agent := range []string{"codex", "claude"} {
		t.Run(agent, func(t *testing.T) {
			isolateNotificationSockets(t)
			root := t.TempDir()
			home := filepath.Join(root, "home")
			scope, tag := "0123456789abcdef", "couch-0001020304050607"
			dataRoot := filepath.Join(root, "data")
			data := filepath.Join(dataRoot, "repos", scope)
			if err := os.MkdirAll(data, 0700); err != nil {
				t.Fatal(err)
			}
			paths, err := artifactpath.ResolveScoped(data, tag)
			if err != nil {
				t.Fatal(err)
			}
			sid := "019eff64-6ceb-7e72-9d41-a735a97029ac"
			native := filepath.Join(home, ".codex", "sessions", "2026", "09", "28", "rollout-test-"+sid+".jsonl")
			if agent == "claude" {
				native = filepath.Join(home, ".claude", "projects", "-fixture", sid+".jsonl")
			}
			if err := os.MkdirAll(filepath.Dir(native), 0700); err != nil {
				t.Fatal(err)
			}
			write := func(path, text string) {
				t.Helper()
				if err := os.WriteFile(path, []byte(text), 0600); err != nil {
					t.Fatal(err)
				}
			}
			launch, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 2, Kind: sessionledger.RecordLaunch, ScopeKey: scope, Tag: tag, Agent: agent, LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{}})
			if err != nil {
				t.Fatal(err)
			}
			write(paths.Ledger(), string(launch)+"\n")
			write(native, "")
			watcherPID := filepath.Join(root, "watcher-pid")
			stop, ready := filepath.Join(root, "stop"), filepath.Join(root, "ready")
			for key, value := range map[string]string{"HOME": home, "PAIR_DATA_DIR": data, "PAIR_SCOPE_KEY": scope, "PAIR_TAG": tag, "PAIR_LAUNCH_ORDINAL": "1", "PAIR_RETENTION_START_ID": "", "PAIR_TEST_DETACH_WATCHER": watcherPID, "PAIR_TEST_NATIVE": native, "PAIR_TEST_STOP": stop, "PAIR_TEST_READY": ready} {
				t.Setenv(key, value)
			}
			executable := filepath.Join(root, agent)
			write(executable, "#!/bin/sh\nexec 3<\"$PAIR_TEST_NATIVE\"\necho ready > \"$PAIR_TEST_READY\"\nwhile [ ! -e \"$PAIR_TEST_STOP\" ]; do sleep 0.05; done\n")
			if err := os.Chmod(executable, 0700); err != nil {
				t.Fatal(err)
			}
			// Pair clients are real process groups. Waiting in a goroutine reaps the
			// terminated child so Couch can prove exit before retiring the incarnation.
			startClient := func() *exec.Cmd {
				t.Helper()
				cmd := exec.Command("sleep", "30")
				cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
				if err := cmd.Start(); err != nil {
					t.Fatal(err)
				}
				done := make(chan struct{})
				go func() { _ = cmd.Wait(); close(done) }()
				t.Cleanup(func() { _ = cmd.Process.Kill(); <-done })
				return cmd
			}
			client := startClient()
			// Expose the actor group for mutation verification of the old ownership
			// arrangement; production watcher spawning does not consume this.
			t.Setenv("PAIR_TEST_CLIENT_PID", strconv.Itoa(client.Process.Pid))
			oldSpawn := startWatcherProcess
			startWatcherProcess = realStartWatcherProcess
			defer func() { startWatcherProcess = oldSpawn }()
			wrapDone := make(chan int, 1)
			go func() { wrapDone <- Run([]string{executable}, bytes.NewReader(nil), &bytes.Buffer{}, &bytes.Buffer{}) }()
			defer func() {
				write(stop, "stop")
				select {
				case <-wrapDone:
				case <-time.After(5 * time.Second):
					t.Error("fixture wrap did not stop")
				}
				if raw, e := os.ReadFile(watcherPID); e == nil {
					pid, _ := strconv.Atoi(string(raw))
					if pid > 0 {
						_ = syscall.Kill(pid, syscall.SIGKILL)
					}
				}
			}()
			waitDetachCondition(t, "agent startup", func() bool { _, e := os.Stat(ready); return e == nil })
			waitDetachCondition(t, "wrap-owned watcher startup", func() bool { _, e := os.Stat(watcherPID); return e == nil })
			current := func() sessionledger.Current {
				t.Helper()
				raw, e := os.ReadFile(paths.Ledger())
				if e != nil {
					t.Fatal(e)
				}
				c, ok := sessionledger.CurrentLaunch(sessionledger.ParseLedger(raw).Records, sessionledger.Owner{ScopeKey: scope, Tag: tag, Agent: agent})
				if !ok {
					t.Fatal("missing launch")
				}
				return c
			}
			if current().Binding != nil {
				t.Fatal("bound before the first turn")
			}

			proc := couchcore.OSProcOps{}
			identity, err := proc.Identity(client.Process.Pid)
			if err != nil {
				t.Fatal(err)
			}
			ns, err := couchcore.ResolveCouchNamespace(filepath.Join(root, "couch"), root)
			if err != nil {
				t.Fatal(err)
			}
			store := couchcore.NewThreadStore(ns)
			address := couchcore.ThreadAddress{RepoScope: scope, Tag: couchcore.ThreadTag(tag)}
			thread, err := store.CreateThread(couchcore.ThreadRecord{SchemaVersion: couchcore.ThreadSchemaVersion, Address: address, StartingPath: root, WorkingPath: root, CreatedAt: time.Now().UTC(), Revision: 1, LatestLaunchProfile: &couchcore.LaunchProfile{Agent: agent, Argv: []string{}}, Incarnations: []couchcore.ThreadIncarnation{{State: couchcore.IncarnationLive, PID: client.Process.Pid, Identity: identity, StartedAt: time.Now().UTC()}}})
			if err != nil {
				t.Fatal(err)
			}
			artifacts := couchcore.NewFakeThreadArtifactCollisionChecker()
			artifacts.SetPairSession(address, "fixture-session", true)
			couch := couchcore.Couch{Namespace: ns, Threads: store, Proc: proc, Artifacts: artifacts, Clock: couchcore.FixedClock{T: time.Now().UTC()}}
			thread, err = couch.Detach(context.Background(), address)
			if err != nil {
				t.Fatal(err)
			}
			if proc.Exists(client.Process.Pid) != couchcore.Dead || len(thread.Incarnations) != 0 {
				t.Fatal("detach did not retire the exited client")
			}
			if current().Binding != nil {
				t.Fatal("detach unexpectedly established a binding")
			}
			agentRaw, err := os.ReadFile(paths.AgentPID())
			if err != nil {
				t.Fatal(err)
			}
			agentPID, err := strconv.Atoi(strings.TrimSpace(string(agentRaw)))
			if err != nil || proc.Exists(agentPID) != couchcore.Live {
				t.Fatalf("detach lost the native agent: pid=%s err=%v", agentRaw, err)
			}
			if err := couchcore.CheckResumePreconditions(thread, couchcore.NativeBindingResolution{}, true); couchcore.ResumeDiagnosticOf(err) != couchcore.ResumeBindingUnbound {
				t.Fatalf("relaunch before first turn = %v, want unbound refusal", err)
			}
			if _, err := couchcore.DecideResume(couchcore.ResumeEligibilityInput{Thread: thread, WorkingPathExists: true, Detached: true}); err != nil {
				t.Fatalf("warm reattach: %v", err)
			}
			attached := startClient()
			if proc.Exists(attached.Process.Pid) != couchcore.Live {
				t.Fatal("replacement client is not attached")
			}
			// The first input and complete native answer occur only AFTER reattach.
			text := "first turn after detach and reattach"
			write(paths.Log(), "## 2026-09-28 01:00:01\n\n"+text+"\n\n---\n\n")
			round := fmt.Sprintf(`{"type":"session_meta","payload":{"id":%q,"source":"cli"}}
{"type":"event_msg","payload":{"type":"task_started","turn_id":"one"}}
{"type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":%q}]}}
{"type":"event_msg","payload":{"type":"task_complete","turn_id":"one"}}
`, sid, text)
			if agent == "claude" {
				round = fmt.Sprintf(`{"type":"user","sessionId":%q,"isSidechain":false,"message":{"role":"user","content":%q}}
{"type":"assistant","sessionId":%q,"message":{"role":"assistant","content":[{"type":"text","text":"done"}],"stop_reason":"end_turn"}}
`, sid, text, sid)
			}
			write(native, round)
			waitDetachCondition(t, "watcher binding after first post-reattach turn", func() bool { return current().Binding != nil })
			binding := current().Binding
			if binding.RootNativeID != sid || binding.AuthorizationProof == nil {
				t.Fatalf("binding lacks native authority: %+v", binding)
			}
			resolver := couchcore.SessionInventoryNativeBindingResolver{Runtime: sessioninventory.NewOSRuntime(home, data)}
			resolved, err := resolver.ResolveEstablished(context.Background(), scope, tag, agent)
			if err != nil {
				t.Fatal(err)
			}
			if err := couchcore.CheckResumePreconditions(thread, resolved, true); err != nil {
				t.Fatalf("relaunch authorization: %v", err)
			}
			if resolved.NativeID != sid {
				t.Fatalf("relaunch selected %q, want %q", resolved.NativeID, sid)
			}
		})
	}
}

func waitDetachCondition(t *testing.T, description string, ready func() bool) {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for !ready() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", description)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
