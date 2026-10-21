package agentfixture

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func fixtureConfig(t *testing.T, agent string) Config {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{"home", "codex", "actor", "repo"} {
		if err := os.Mkdir(filepath.Join(root, dir), 0700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, OwnershipFile), []byte("fixture-run"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("COUCH_ISOLATED_ROOT", root)
	t.Setenv("HOME", filepath.Join(root, "home"))
	t.Setenv("CODEX_HOME", filepath.Join(root, "codex"))
	t.Chdir(filepath.Join(root, "repo"))
	for _, key := range []string{"COUCH_TREE", "COUCH_STORE_DIR", "COUCH_IDENTITY_DIR", "PAIR_DATA_DIR", "COUCH_PAIR_DATA_DIR", "CLAUDE_CONFIG_DIR", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME", "XDG_RUNTIME_DIR", "ZELLIJ_SOCKET_DIR", "ZELLIJ_CONFIG_DIR"} {
		t.Setenv(key, filepath.Join(root, key))
	}
	t.Setenv("COUCH_TREE", filepath.Join(root, "repo"))
	return Config{Root: root, RunID: "fixture-run", ControlDir: filepath.Join(root, "actor"), Home: filepath.Join(root, "home"), CodexHome: filepath.Join(root, "codex"), Agent: agent, CouchBinary: "/bin/sh"}
}
func waitFixture(t *testing.T, check func() bool) {
	t.Helper()
	deadline := time.Now().Add(4 * time.Second)
	for !check() {
		if time.Now().After(deadline) {
			t.Fatal("fixture observation deadline")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

type lockedOutput struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (w *lockedOutput) Write(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.Write(b)
}
func (w *lockedOutput) String() string { w.mu.Lock(); defer w.mu.Unlock(); return w.b.String() }

func TestRunSubmitTranscriptAndNativeResume(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		t.Run(agent, func(t *testing.T) {
			config := fixtureConfig(t, agent)
			if err := config.validate(); err != nil {
				t.Fatal(err)
			}
			if agent == "claude" {
				config.Args = []string{"--session-id", "11111111-1111-4111-8111-111111111111"}
			}
			input, writer := io.Pipe()
			output := &lockedOutput{}
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan error, 1)
			go func() { done <- Run(ctx, config, input, output) }()
			var ready Ready
			waitFixture(t, func() bool {
				raw, err := os.ReadFile(filepath.Join(config.ControlDir, "ready.json"))
				return err == nil && json.Unmarshal(raw, &ready) == nil
			})
			writer.Write([]byte("\x1b[200~fixture peer body\x1b[201~"))
			waitFixture(t, func() bool {
				raw, err := os.ReadFile(filepath.Join(config.ControlDir, "state.json"))
				var state AgentState
				return err == nil && json.Unmarshal(raw, &state) == nil && state.Draft == "fixture peer body"
			})
			if raw, _ := os.ReadFile(filepath.Join(config.ControlDir, "accepted.jsonl")); len(raw) != 0 {
				t.Fatal("paste counted as acceptance", string(raw))
			}
			writer.Write([]byte{'\r'})
			waitFixture(t, func() bool {
				raw, _ := os.ReadFile(filepath.Join(config.ControlDir, "accepted.jsonl"))
				return strings.Contains(string(raw), "fixture peer body")
			})
			cancel()
			writer.Close()
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			transcript, err := os.ReadFile(ready.Transcript)
			if err != nil {
				t.Fatal(err)
			}
			var records []sessioninventory.FramedJSONLRecord
			for _, line := range bytes.Split(bytes.TrimSpace(transcript), []byte{'\n'}) {
				records = append(records, sessioninventory.FramedJSONLRecord{Bytes: line})
			}
			nativeRoot := filepath.Join(config.Home, ".claude", "projects")
			if agent == "codex" {
				nativeRoot = filepath.Join(config.CodexHome, "sessions")
			}
			relative, err := filepath.Rel(nativeRoot, ready.Transcript)
			if err != nil {
				t.Fatal(err)
			}
			candidate, diagnostics, err := sessioninventory.ParseRecoveryCandidate(sessioninventory.Agent(agent), sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{RelativePath: filepath.ToSlash(relative)}}, records, nil)
			if err != nil || !candidate.RootEligible || candidate.Interactive.State != sessioninventory.InteractiveYes || candidate.CWD != ready.CWD || len(candidate.Messages) < 2 {
				t.Fatalf("candidate=%+v diagnostics=%+v err=%v transcript=%s", candidate, diagnostics, err, transcript)
			}
			if !strings.Contains(output.String(), "fixture peer body") {
				t.Fatal("actual accepted body absent from display")
			}
			config.Args = []string{"--resume", ready.NativeID}
			if agent == "codex" {
				config.Args = []string{"resume", ready.NativeID, "--no-alt-screen"}
			}
			resumed, err := openConversation(config)
			if err != nil {
				t.Fatal(err)
			}
			resumed.Close()
			again, _ := os.ReadFile(ready.Transcript)
			if !bytes.Equal(transcript, again) {
				t.Fatal("resume changed prior transcript")
			}
		})
	}
}
func TestRunConcurrentCLIUsesActualActorEnvironment(t *testing.T) {
	config := fixtureConfig(t, "claude")
	t.Setenv("COUCH_THREAD_TAG", "actual-fixture-actor")
	input, writer := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- Run(ctx, config, input, io.Discard) }()
	waitFixture(t, func() bool { _, e := os.Stat(filepath.Join(config.ControlDir, "ready.json")); return e == nil })
	release := filepath.Join(config.Root, "release")
	for _, id := range []string{"one", "two", "three"} {
		marker := filepath.Join(config.Root, id+".started")
		if err := WriteCommand(config.ControlDir, Command{ID: id, Kind: Exec, Args: []string{"-c", "printf started > \"$1\"; while [ ! -e \"$2\" ]; do sleep 0.01; done; printf '%s' \"$COUCH_THREAD_TAG\"", "fixture", marker, release}}); err != nil {
			t.Fatal(err)
		}
	}
	waitFixture(t, func() bool {
		for _, id := range []string{"one", "two", "three"} {
			if _, e := os.Stat(filepath.Join(config.Root, id+".started")); e != nil {
				return false
			}
		}
		return true
	})
	if err := os.WriteFile(release, []byte("release"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"one", "two", "three"} {
		var result Result
		waitFixture(t, func() bool { var e error; result, e = ReadResult(config.ControlDir, id); return e == nil })
		if result.ID != id || result.Stdout != "actual-fixture-actor" || result.ExitCode != 0 || result.Error != "" {
			t.Fatal(result)
		}
	}
	if err := WriteCommand(config.ControlDir, Command{ID: "state", Kind: SetState, State: Picker}); err != nil {
		t.Fatal(err)
	}
	waitFixture(t, func() bool { _, e := ReadResult(config.ControlDir, "state"); return e == nil })
	cancel()
	writer.Close()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
func TestRunRefusesUnownedOrEscapingRootsBeforeOutput(t *testing.T) {
	config := fixtureConfig(t, "claude")
	for _, mutate := range []func(*Config){func(c *Config) { c.RunID = "foreign" }, func(c *Config) { c.ControlDir = filepath.Dir(c.Root) }, func(c *Config) { c.Home = "" }} {
		c := config
		mutate(&c)
		var output bytes.Buffer
		if err := Run(context.Background(), c, io.NopCloser(strings.NewReader("")), &output); err == nil || output.Len() != 0 {
			t.Fatal("invalid roots had effects", err)
		}
	}
}

func TestRunFiniteInputDrainsBeforeEOF(t *testing.T) {
	config := fixtureConfig(t, "claude")
	if err := Run(context.Background(), config, io.NopCloser(strings.NewReader("accepted before EOF\rsecond body\r")), io.Discard); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(config.ControlDir, "accepted.jsonl"))
	if err != nil || !strings.Contains(string(raw), "accepted before EOF") {
		t.Fatalf("input lost at EOF: %s %v", raw, err)
	}
	var inputs []AcceptedInput
	for _, line := range bytes.Split(bytes.TrimSpace(raw), []byte{'\n'}) {
		var item AcceptedInput
		if err := json.Unmarshal(line, &item); err != nil {
			t.Fatal(err)
		}
		inputs = append(inputs, item)
	}
	if len(inputs) != 2 || inputs[0].Sequence != 1 || inputs[1].Sequence != 2 {
		t.Fatalf("acceptance sequence lost: %+v", inputs)
	}
}
func TestRunBoundsOutputAndJoinsCanceledCLI(t *testing.T) {
	config := fixtureConfig(t, "claude")
	input, writer := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- Run(ctx, config, input, io.Discard) }()
	waitFixture(t, func() bool { _, e := os.Stat(filepath.Join(config.ControlDir, "ready.json")); return e == nil })
	if e := WriteCommand(config.ControlDir, Command{ID: "output", Kind: Exec, Args: []string{"-c", "i=0; while [ $i -lt 12000 ]; do printf 'abcdefgh'; i=$((i+1)); done"}}); e != nil {
		t.Fatal(e)
	}
	var result Result
	waitFixture(t, func() bool { var e error; result, e = ReadResult(config.ControlDir, "output"); return e == nil })
	if len(result.Stdout) != MaxOutputBytes || !result.Truncated {
		t.Fatal("output cap failed", len(result.Stdout), result.Truncated)
	}
	marker := filepath.Join(config.Root, "child.started")
	if e := WriteCommand(config.ControlDir, Command{ID: "cancel", Kind: Exec, Timeout: time.Hour, Args: []string{"-c", "printf started > \"$1\"; sleep 30", "fixture", marker}}); e != nil {
		t.Fatal(e)
	}
	waitFixture(t, func() bool { _, e := os.Stat(marker); return e == nil })
	cancel()
	writer.Close()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("canceled CLI was not joined")
	}
}

func TestCommandRefusesStaleActorWithoutExecuting(t *testing.T) {
	config := fixtureConfig(t, "claude")
	input, writer := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer writer.Close()
	done := make(chan error, 1)
	go func() { done <- Run(ctx, config, input, io.Discard) }()
	waitFixture(t, func() bool { _, e := os.Stat(filepath.Join(config.ControlDir, "ready.json")); return e == nil })
	if err := WriteCommand(config.ControlDir, Command{ID: "old", PID: os.Getpid() + 100000, Kind: Exec, Args: []string{"-c", "printf wrong"}}); err != nil {
		t.Fatal(err)
	}
	var result Result
	waitFixture(t, func() bool { var e error; result, e = ReadResult(config.ControlDir, "old"); return e == nil })
	if result.Stdout != "" || !strings.Contains(result.Error, "incarnation") {
		t.Fatal(result)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestRunRepaintsAfterNativeResize(t *testing.T) {
	config := fixtureConfig(t, "claude")
	sizes := make(chan Size, 1)
	config.Resizes = sizes
	input, writer := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer writer.Close()
	done := make(chan error, 1)
	output := &lockedOutput{}
	go func() { done <- Run(ctx, config, input, output) }()
	waitFixture(t, func() bool { _, e := os.Stat(filepath.Join(config.ControlDir, "ready.json")); return e == nil })
	sizes <- Size{Cols: 50, Rows: 20}
	waitFixture(t, func() bool { return strings.Contains(output.String(), strings.Repeat("─", 49)+"\x1b[39m") })
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func TestRunStopAfterOrdinaryRequestQuotaExhausted(t *testing.T) {
	config := fixtureConfig(t, "claude")
	input, writer := io.Pipe()
	defer writer.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- Run(ctx, config, input, io.Discard); close(done) }()
	t.Cleanup(func() {
		cancel()
		_ = writer.Close()
		select {
		case <-done:
		case <-time.After(4 * time.Second):
			t.Error("fixture failed to join during test cleanup")
		}
	})
	waitFixture(t, func() bool { _, e := os.Stat(filepath.Join(config.ControlDir, "ready.json")); return e == nil })
	for i := 0; i < MaxRequests; i++ {
		if err := WriteCommand(config.ControlDir, Command{ID: fmt.Sprintf("ordinary-%03d", i), Kind: SetState, State: Idle}); err != nil {
			t.Fatal(err)
		}
	}
	waitFixture(t, func() bool {
		entries, e := os.ReadDir(filepath.Join(config.ControlDir, "results"))
		return e == nil && len(entries) == MaxRequests
	})
	if err := WriteCommand(config.ControlDir, Command{ID: "overflow", Kind: SetState, State: Idle}); !errors.Is(err, ErrMailboxFull) {
		t.Fatalf("full result mailbox not classified: %v", err)
	}
	if err := WriteCommand(config.ControlDir, Command{ID: "shutdown", Kind: Stop}); err != nil {
		t.Fatalf("shutdown refused after %d completed ordinary requests: %v", MaxRequests, err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(4 * time.Second):
		t.Fatal("shutdown failed to join fixture")
	}
	result, err := ReadResult(config.ControlDir, "shutdown")
	if err != nil || result.Error != "" || result.ExitCode != 0 {
		t.Fatalf("shutdown result=%+v error=%v", result, err)
	}
	entries, err := os.ReadDir(filepath.Join(config.ControlDir, "results"))
	if err != nil || len(entries) != MaxRequests {
		t.Fatalf("result count=%d error=%v", len(entries), err)
	}
}

func TestRunStaleShutdownDoesNotStopCurrentActor(t *testing.T) {
	config := fixtureConfig(t, "claude")
	input, writer := io.Pipe()
	defer writer.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- Run(ctx, config, input, io.Discard); close(done) }()
	t.Cleanup(func() {
		cancel()
		_ = writer.Close()
		select {
		case <-done:
		case <-time.After(4 * time.Second):
			t.Error("fixture failed to join during test cleanup")
		}
	})
	waitFixture(t, func() bool { _, e := os.Stat(filepath.Join(config.ControlDir, "ready.json")); return e == nil })
	for i := 0; i < 16; i++ {
		id := fmt.Sprintf("stale-stop-%02d", i)
		if err := WriteCommand(config.ControlDir, Command{ID: id, Kind: Stop, PID: os.Getpid() + 100000}); err != nil {
			t.Fatal(err)
		}
		var stale Result
		waitFixture(t, func() bool { var err error; stale, err = ReadResult(config.ControlDir, id); return err == nil })
		if !strings.Contains(stale.Error, "incarnation") {
			t.Fatalf("stale shutdown result=%+v", stale)
		}
		if _, err := ReadResult(config.ControlDir, "ordinary-pending"); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("reserved result poisoned ordinary pending observation: %v", err)
		}
		select {
		case err := <-done:
			t.Fatalf("stale shutdown stopped current actor: %v", err)
		default:
		}
	}
	entries, err := os.ReadDir(filepath.Join(config.ControlDir, "results"))
	if err != nil || len(entries) != 0 {
		t.Fatalf("stale shutdown grew ordinary results: count=%d error=%v", len(entries), err)
	}
	if err := WriteCommand(config.ControlDir, Command{ID: "current-stop", Kind: Stop}); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(4 * time.Second):
		t.Fatal("current shutdown did not join")
	}
}

func TestDecodeCommandCarriesBoundedCLITimeout(t *testing.T) {
	command, err := DecodeCommand([]byte(`{"id":"long-recovery","kind":"exec","timeout_ns":90000000000}`))
	if err != nil {
		t.Fatalf("caller deadline refused: %v", err)
	}
	encoded, err := json.Marshal(command)
	if err != nil || !bytes.Contains(encoded, []byte(`"timeout_ns":90000000000`)) {
		t.Fatalf("deadline lost: %s %v", encoded, err)
	}
	for _, raw := range []string{
		`{"id":"negative","kind":"exec","timeout_ns":-1}`,
		`{"id":"unbounded","kind":"exec","timeout_ns":86400000000001}`,
		`{"id":"state","kind":"state","state":"idle","timeout_ns":1}`,
		`{"id":"stop","kind":"stop","timeout_ns":1}`,
	} {
		if _, err := DecodeCommand([]byte(raw)); err == nil {
			t.Fatalf("accepted invalid budget: %s", raw)
		}
	}
}

func TestRunCLIUsesPerRequestBudget(t *testing.T) {
	if got := commandCLITimeout(Command{}); got != 15*time.Second {
		t.Fatalf("default=%v", got)
	}
	if got := commandCLITimeout(Command{Timeout: 90 * time.Second}); got != 90*time.Second {
		t.Fatalf("caller budget shortened: %v", got)
	}
	started := time.Now()
	result := runCLI(context.Background(), "/bin/sh", Command{ID: "bounded", Kind: Exec, Timeout: 100 * time.Millisecond, Args: []string{"-c", "printf started; exec sleep 30"}})
	if result.Error == "" {
		t.Fatalf("request budget did not cancel child: %+v", result)
	}
	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("budget cancellation did not join promptly: %v", elapsed)
	}
}

func TestWriteCommandClassifiesFullPendingMailbox(t *testing.T) {
	config := fixtureConfig(t, "claude")
	if err := atomicJSON(filepath.Join(config.ControlDir, "ready.json"), Ready{PID: os.Getpid(), RunID: config.RunID}, false); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < MaxRequests; i++ {
		if err := WriteCommand(config.ControlDir, Command{ID: fmt.Sprintf("pending-%03d", i), Kind: SetState, State: Idle}); err != nil {
			t.Fatal(err)
		}
	}
	if err := WriteCommand(config.ControlDir, Command{ID: "overflow", Kind: SetState, State: Idle}); !errors.Is(err, ErrMailboxFull) {
		t.Fatalf("full pending mailbox not classified: %v", err)
	}
}
