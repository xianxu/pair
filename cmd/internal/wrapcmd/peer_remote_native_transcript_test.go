package wrapcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/testcouch/agentfixture"
	"github.com/xianxu/pair/cmd/internal/testpair"
	"github.com/xianxu/pair/cmd/internal/testwait"
	"golang.org/x/term"
)

const remoteNativeProducerKey = "PAIR_TEST_REMOTE_NATIVE_PRODUCER"

var remoteNativeForbiddenEnvironment = []string{
	"PAIR_DATA_DIR", "PAIR_TAG", "PAIR_SCOPE_KEY", "PAIR_SESSION_NAME", "PAIR_LAUNCH_NONCE", "PAIR_LAUNCH_ORDINAL", "PAIR_RETENTION_START_ID", "PAIR_SCROLLBACK_EVENTS_PATH", "PAIR_WRAP_LOG", "COUCH_LIFETIME_LEASE", "COUCH_THREAD_SCOPE", "COUCH_THREAD_TAG", "COUCH_STORE_DIR", "COUCH_IDENTITY_DIR", "COUCH_PAIR_DATA_DIR", "COUCH_CAPTURE_DIR", "COUCH_SUPERVISOR_SOCKET", "COUCH_CONTROL_DIR", "COUCH_HUB_SLOT", "CF_ACCESS_JWT", "HTTP_COOKIE", "COUCH_REMOTE_PASSCODE",
}

// This child is a deterministic native-format transcript producer, not an
// installed agent. Guard the empty-environment boundary before its first write.
func TestRemoteNativeTranscriptProducerHelper(t *testing.T) {
	agent := os.Getenv(remoteNativeProducerKey)
	if agent == "" {
		t.Skip("owned subprocess fixture")
	}
	for _, key := range remoteNativeForbiddenEnvironment {
		if os.Getenv(key) != "" {
			t.Fatalf("inherited authority or credential %s crossed producer boundary", key)
		}
	}
	allowed := map[string]bool{"COUCH_TREE": true, "COUCH_ISOLATED_ROOT": true, "PAIR_TEST_ENV_SCRIPT": true, "PAIR_TESTCOUCH_ROOT": true, "PAIR_TESTCOUCH_RUN_ID": true, "PAIR_TESTCOUCH_CONTROL": true, "PAIR_TESTCOUCH_COUCH": true, remoteNativeProducerKey: true}
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if (strings.HasPrefix(key, "COUCH_") || strings.HasPrefix(key, "PAIR_")) && !allowed[key] {
			t.Fatalf("unexpected fixture authority %s before producer startup", key)
		}
	}
	config, err := agentfixture.ConfigFromEnvironment(agent, nil)
	if err != nil {
		t.Fatal(err)
	}
	state, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	defer term.Restore(int(os.Stdin.Fd()), state)
	if err := agentfixture.Run(context.Background(), config, os.Stdin, os.Stdout); err != nil {
		t.Fatal(err)
	}
}

// The production input writer sends through a real private PTY to the shared
// stateful agentfixture producer. Its actual output drives composer occupancy
// and clear confirmation. The production native reader then checks turns9/10.
// This qualifies our transport/format seams, not real Claude/Codex persistence;
// the operator's Access/iPhone/native-agent drill remains separate acceptance.
func TestRemoteNativeTranscriptContinuesBeyondCorrelationCap(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		t.Run(agent, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			home, codex, workspace, control := filepath.Join(root, "home"), filepath.Join(root, "codex"), filepath.Join(root, "repo"), filepath.Join(root, "control")
			for _, path := range []string{home, codex, workspace, control} {
				if err := os.MkdirAll(path, 0700); err != nil {
					t.Fatal(err)
				}
			}
			const runID = "remote-native-paper-trail"
			if err := os.WriteFile(filepath.Join(root, agentfixture.OwnershipFile), []byte(runID), 0600); err != nil {
				t.Fatal(err)
			}
			poison := t.TempDir()
			for _, key := range remoteNativeForbiddenEnvironment {
				t.Setenv(key, filepath.Join(poison, "NATIVE_REMOTE_SECRET_MUST_NOT_APPEAR"))
			}
			env := testpair.Environment(t, root, "HOME="+home, "CODEX_HOME="+codex, "COUCH_ISOLATED_ROOT="+root, "COUCH_TREE="+workspace, "PAIR_TESTCOUCH_ROOT="+root, "PAIR_TESTCOUCH_RUN_ID="+runID, "PAIR_TESTCOUCH_CONTROL="+control, "PAIR_TESTCOUCH_COUCH=/bin/sh", remoteNativeProducerKey+"="+agent)
			testpair.ApplyEnvironment(t, root, env...)
			exe, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(t.Context())
			command := exec.CommandContext(ctx, exe, "-test.run=^TestRemoteNativeTranscriptProducerHelper$")
			command.Dir = workspace
			command.Env = env
			master, err := pty.StartWithSize(command, &pty.Winsize{Rows: 38, Cols: 120})
			if err != nil {
				cancel()
				t.Fatal(err)
			}
			childDone := make(chan struct{})
			var childErr error
			go func() { childErr = command.Wait(); close(childDone) }()
			output := make(chan []byte, 16)
			readDone := make(chan struct{})
			go func() {
				defer close(readDone)
				buffer := make([]byte, 16384)
				for {
					n, err := master.Read(buffer)
					if n > 0 {
						select {
						case output <- append([]byte(nil), buffer[:n]...):
						case <-ctx.Done():
							return
						}
					}
					if err != nil {
						return
					}
				}
			}()
			t.Cleanup(func() {
				cancel()
				_ = command.Process.Signal(syscall.SIGTERM)
				_ = master.Close()
				<-childDone
				<-readDone
			})
			f := newHarnessSessionFake(t, agent, true)
			t.Cleanup(f.close)
			f.resize(120, 38)
			now := time.Now()
			receiver := newPeerDelivery(peerBatchTestBinding("ops:0"), func() time.Time { return now })
			receiver.binding.Agent = agent
			receiver.session = &recordingPeerSink{}
			f.proxy.peer = receiver
			logPath := filepath.Join(root, "peer-correlation.md")
			var workers sync.WaitGroup
			receiver.recordTurn, receiver.recordBatch = peerTurnRecorders(logPath, func(string) string { return "" }, func(_, detail string) { t.Error(detail) }, &workers)
			t.Cleanup(workers.Wait)
			pump := func() {
				for {
					select {
					case chunk := <-output:
						receiver.observeOutput(chunk)
						f.output(string(chunk))
						receiver.outputForwarded()
					default:
						return
					}
				}
			}
			alive := func() {
				select {
				case <-childDone:
					t.Fatalf("native producer exited: %v", childErr)
				default:
				}
			}
			actor := agentfixture.ActorDirectory(control, workspace)
			var ready agentfixture.Ready
			testwait.Until(t, "private native producer readiness", func() bool {
				alive()
				pump()
				raw, err := os.ReadFile(filepath.Join(actor, "ready.json"))
				return err == nil && json.Unmarshal(raw, &ready) == nil && f.proxy.terminal.FrameComplete()
			})
			runtime := sessioninventory.NewOSRuntime(home, "")
			if agent == "codex" {
				runtime, err = runtime.WithCodexHome(codex)
				if err != nil {
					t.Fatal(err)
				}
			}
			roots := runtime.NativeRoots(sessioninventory.Agent(agent))
			if len(roots) != 1 {
				t.Fatalf("canonical native roots: %+v", roots)
			}
			relative, err := filepath.Rel(roots[0].Path, ready.Transcript)
			if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) {
				t.Fatalf("producer transcript outside canonical root: %s %v", ready.Transcript, err)
			}
			const turns = peerTurnRecordLimit + 2
			originals := make([]couchmessage.Message, 0, turns)
			for i := 0; i < turns; i++ {
				message := couchmessage.Message{Origin: couchmessage.OriginRemote, RemoteSource: couchmessage.RemoteSource{Email: "operator@example.com", IP: "192.0.2.77"}, RemoteEdited: true, DocumentID: "0123456789abcdef0123456789abcdef", ID: fmt.Sprintf("remote-native-%02d", i+1), From: peerBatchTestBinding("pair:6"), To: receiver.binding, Body: fmt.Sprintf("submitted: %s", filepath.Join(workspace, fmt.Sprintf("session-%02d.md", i+1))), Deadline: now.Add(couchmessage.PasteTimeout)}
				originals = append(originals, message)
				if err := receiver.reserve(message.ID, receiver.sequence); err != nil {
					t.Fatal(err)
				}
				if err := receiver.enqueue(message); err != nil {
					t.Fatal(err)
				}
				testwait.Until(t, "native composer paste", func() bool {
					alive()
					pump()
					f.proxy.dispatchPeer(master)
					return receiver.receipt().Status == couchmessage.Delivering
				})
				testwait.Until(t, "actual native occupied composer", func() bool {
					alive()
					pump()
					f.proxy.dispatchPeer(master)
					receiver.mu.Lock()
					defer receiver.mu.Unlock()
					return receiver.sawOccupied
				})
				now = now.Add(PeerSubmitDelay + time.Second)
				f.proxy.dispatchPeer(master)
				testwait.Until(t, "native submit and cleared composer confirmation", func() bool {
					alive()
					pump()
					f.proxy.dispatchPeer(master)
					return receiver.receipt().Status == couchmessage.Submitted
				})
				workers.Wait()
				release, _ := json.Marshal(couchmessage.EndpointRequest{Op: "release", Binding: receiver.binding, ID: message.ID})
				if _, err := receiver.handleEndpoint(t.Context(), release); err != nil {
					t.Fatal(err)
				}
			}
			native, err := os.ReadFile(ready.Transcript)
			if err != nil {
				t.Fatal(err)
			}
			var records []sessioninventory.FramedJSONLRecord
			for _, line := range bytes.Split(bytes.TrimSpace(native), []byte{'\n'}) {
				records = append(records, sessioninventory.FramedJSONLRecord{Bytes: line})
			}
			candidate, diagnostics, err := sessioninventory.ParseRecoveryCandidate(sessioninventory.Agent(agent), sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: roots[0].Name, RelativePath: filepath.ToSlash(relative)}}, records, nil)
			if err != nil || !candidate.RootEligible {
				t.Fatalf("native reader refused: %+v %+v %v", candidate, diagnostics, err)
			}
			var userTurns []string
			for _, message := range candidate.Messages {
				if message.Role == sessioninventory.EventOperator {
					userTurns = append(userTurns, message.Text)
				}
			}
			if len(userTurns) != turns {
				t.Fatalf("native retained %d user turns, want%d", len(userTurns), turns)
			}
			for i, message := range originals {
				if !strings.Contains(userTurns[i], "[remote operator@example.com 192.0.2.77]\n"+message.Body) {
					t.Fatalf("native turn%d lost exact trusted remote header and canonical submit: %q", i+1, userTurns[i])
				}
				if userTurns[i] != couchmessage.Envelope(message) {
					t.Fatalf("native turn%d changed trusted header/body: %q", i+1, userTurns[i])
				}
			}
			correlation, err := os.ReadFile(logPath)
			if err != nil {
				t.Fatal(err)
			}
			entries := sessioninventory.ParsePairLog(correlation, 0).Entries
			if len(entries) != peerTurnRecordLimit {
				t.Fatalf("correlation cap changed: %d", len(entries))
			}
			for _, entry := range entries {
				if strings.Contains(entry.AuthoredText, originals[peerTurnRecordLimit].ID) {
					t.Fatal("turn9 entered bounded correlation aid")
				}
			}
			for _, raw := range [][]byte{native, correlation} {
				if bytes.Contains(raw, []byte("NATIVE_REMOTE_SECRET_MUST_NOT_APPEAR")) || bytes.Contains(raw, []byte("CF_ACCESS_JWT")) || bytes.Contains(raw, []byte("HTTP_COOKIE")) {
					t.Fatal("native/correlation trail leaked credentials")
				}
			}
			if entries, err := os.ReadDir(poison); err != nil || len(entries) != 0 {
				t.Fatalf("producer touched inherited identity root: %v %v", entries, err)
			}
		})
	}
}
