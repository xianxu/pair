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
	"github.com/xianxu/pair/cmd/internal/testpair"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

func fixtureConfig(t *testing.T, agent string) Config {
	t.Helper()
	root := testpair.ShortRoot(t)
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
	for _, agent := range []string{"claude", "codex"} {
		t.Run(agent, func(t *testing.T) { runQuotaExhaustion(t, agent) })
	}
}

func runQuotaExhaustion(t *testing.T, agent string) {
	t.Helper()
	config := fixtureConfig(t, agent)
	input, writer := io.Pipe()
	defer writer.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	gate := newRepaintGate()
	go func() { done <- Run(ctx, config, input, gate); close(done) }()
	awaiting := "ready.json"
	t.Cleanup(func() {
		cancel()
		gate.open()
		_ = writer.Close()
		select {
		case <-done:
		case <-time.After(4 * time.Second):
			t.Errorf("fixture failed to join during test cleanup; awaiting %s", awaiting)
		}
		if t.Failed() {
			for _, sub := range []string{"commands", "results"} {
				entries, err := os.ReadDir(filepath.Join(config.ControlDir, sub))
				t.Logf("awaiting %s; %s entries=%v error=%v", awaiting, sub, entries, err)
			}
		}
	})
	waitFixture(t, func() bool { _, e := os.Stat(filepath.Join(config.ControlDir, "ready.json")); return e == nil })
	if err := os.WriteFile(filepath.Join(config.ControlDir, "commands", ".publish-held"), []byte("publication in progress"), 0600); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < MaxRequests; i++ {
		id := fmt.Sprintf("ordinary-%03d", i)
		awaiting = id
		if err := WriteCommand(config.ControlDir, Command{ID: id, Kind: SetState, State: Idle}); err != nil {
			t.Fatal(err)
		}
		gate.wait(t, i+2)
		if i == MaxRequests-1 {
			// Hold the final repaint to reject premature completion evidence.
			if _, err := ReadResult(config.ControlDir, id); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("held final request completed early: %v", err)
			}
			entries, err := os.ReadDir(filepath.Join(config.ControlDir, "results"))
			if err != nil || countMailboxEntries(entries) != MaxRequests-1 {
				t.Fatalf("held final request results=%d error=%v", countMailboxEntries(entries), err)
			}
			select {
			case err := <-done:
				t.Fatalf("fixture exited before final repaint release: %v", err)
			default:
			}
		}
		gate.advance <- struct{}{}
		waitFixture(t, func() bool {
			result, err := ReadResult(config.ControlDir, id)
			return err == nil && result.ID == id && result.Error == "" && result.ExitCode == 0
		})
	}
	if err := WriteCommand(config.ControlDir, Command{ID: "overflow", Kind: SetState, State: Idle}); !errors.Is(err, ErrMailboxFull) {
		t.Fatalf("full result mailbox not classified: %v", err)
	}
	if err := WriteCommand(config.ControlDir, Command{ID: "shutdown", Kind: Stop}); err != nil {
		t.Fatalf("shutdown refused after %d completed ordinary requests: %v", MaxRequests, err)
	}
	awaiting = "shutdown"
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
	if err := os.RemoveAll(config.ControlDir); err != nil {
		t.Fatalf("joined quota actor left a results writer: %v", err)
	}
}

// repaintGate holds each ordinary repaint after Run's initial screen. Opening
// it releases every write, including writes a broken drain starts afterward.
type repaintGate struct {
	arrived chan int
	advance chan struct{}
	opened  chan struct{}
	once    sync.Once
	writes  int // Read only after Run has joined.
}

func newRepaintGate() *repaintGate {
	return &repaintGate{arrived: make(chan int, 1), advance: make(chan struct{}), opened: make(chan struct{})}
}

func (g *repaintGate) open() { g.once.Do(func() { close(g.opened) }) }

func (g *repaintGate) Write(b []byte) (int, error) {
	g.writes++
	if g.writes > 1 {
		select {
		case <-g.opened:
			return len(b), nil
		default:
		}
		select {
		case g.arrived <- g.writes:
		case <-g.opened:
		}
		select {
		case <-g.advance:
		case <-g.opened:
		}
	}
	return len(b), nil
}

func (g *repaintGate) wait(t *testing.T, want int) {
	t.Helper()
	select {
	case got := <-g.arrived:
		if got != want {
			t.Fatalf("repaint=%d, want %d", got, want)
		}
	case <-time.After(4 * time.Second):
		t.Fatalf("repaint %d did not reach controlled writer", want)
	}
}

func TestRunCancellationStopsMailboxSnapshot(t *testing.T) {
	for _, agent := range []string{"claude", "codex"} {
		t.Run(agent, func(t *testing.T) {
			config := fixtureConfig(t, agent)
			input, writer := io.Pipe()
			ctx, cancel := context.WithCancel(t.Context())
			gate := newRepaintGate()
			done := make(chan struct{})
			var runErr error
			go func() { runErr = Run(ctx, config, input, gate); close(done) }()
			t.Cleanup(func() {
				cancel()
				gate.open()
				_ = writer.Close()
				select {
				case <-done:
				case <-time.After(4 * time.Second):
					t.Error("canceled snapshot actor did not join after writer release")
				}
			})
			waitFixture(t, func() bool { _, err := os.Stat(filepath.Join(config.ControlDir, "ready.json")); return err == nil })
			write := func(i int) {
				t.Helper()
				if err := WriteCommand(config.ControlDir, Command{ID: fmt.Sprintf("ordinary-%03d", i), Kind: SetState, State: Idle}); err != nil {
					t.Fatal(err)
				}
			}
			write(0)
			gate.wait(t, 2)
			for i := 1; i < MaxRequests; i++ {
				write(i)
			}
			// The next repaint belongs to a fresh snapshot containing the rest.
			gate.advance <- struct{}{}
			gate.wait(t, 3)
			cancel()
			gate.open()
			select {
			case <-done:
			case <-time.After(4 * time.Second):
				t.Fatal("canceled snapshot did not join")
			}
			if runErr != nil {
				t.Fatal(runErr)
			}
			entries, err := os.ReadDir(filepath.Join(config.ControlDir, "results"))
			if err != nil {
				t.Fatal(err)
			}
			if got := countMailboxEntries(entries); got != 2 || gate.writes != 3 {
				t.Errorf("snapshot continued after cancellation: completed=%d want2, writes=%d want3", got, gate.writes)
			}
			if err := os.RemoveAll(config.ControlDir); err != nil {
				t.Fatalf("joined actor left a results writer: %v", err)
			}
		})
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
	now := time.Unix(1700000000, 0)
	if got := commandCLIDeadline(Command{Timeout: 90 * time.Second}, now); got != now.Add(90*time.Second) {
		t.Fatalf("selected deadline=%v", got)
	}
	absolute := now.Add(20 * time.Second)
	if got := commandCLIDeadline(Command{Timeout: 90 * time.Second, DeadlineUnixNano: absolute.UnixNano()}, now); got != absolute {
		t.Fatalf("admission deadline=%v", got)
	}
	marker := filepath.Join(t.TempDir(), "child-ready")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	armed := make(chan func(), 1)
	done := make(chan Result, 1)
	command := Command{ID: "bounded", Kind: Exec, Timeout: 90 * time.Second, Args: []string{"-c", `printf ready > "$1"; exec sleep 30`, "fixture", marker}}
	go func() {
		done <- runCLIWithDeadline(ctx, "/bin/sh", command, func(parent context.Context, selected time.Time) (context.Context, context.CancelFunc) {
			if selected.IsZero() {
				t.Error("request deadline not selected")
			}
			child, expire, stop := testwait.DeadlineOnCue(parent, commandCLITimeout(command))
			armed <- expire
			return child, stop
		})
		close(done)
	}()
	t.Cleanup(func() { cancel(); <-done })
	waitFixture(t, func() bool { _, err := os.Stat(marker); return err == nil })
	select {
	case result := <-done:
		t.Fatalf("request ended before expiry: %+v", result)
	default:
	}
	(<-armed)()
	select {
	case result := <-done:
		if result.Error == "" || result.CleanupUncertain {
			t.Fatalf("request expiry did not join child: %+v", result)
		}
	case <-testwait.After(t):
		t.Fatal("request expiry did not finish")
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

func TestRunRetiredResultsKeepMailboxBoundedBeyondLifetimeQuota(t *testing.T) {
	config := fixtureConfig(t, "claude")
	input, writer := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Run(ctx, config, input, io.Discard); close(done) }()
	t.Cleanup(func() { cancel(); _ = writer.Close(); <-done })
	waitFixture(t, func() bool { _, err := os.Stat(filepath.Join(config.ControlDir, "ready.json")); return err == nil })
	maxPending, maxResults := 0, 0
	// Atomic publication temporarily has both a scratch name and its linked
	// final name. Neither the capacity contract nor this oracle counts scratch.
	if err := os.WriteFile(filepath.Join(config.ControlDir, "results", ".publish-held"), []byte("in progress"), 0600); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2*MaxRequests; i++ {
		id := fmt.Sprintf("serial-%03d", i)
		if err := WriteCommand(config.ControlDir, Command{ID: id, Kind: SetState, State: Idle}); err != nil {
			t.Fatal(err)
		}
		waitFixture(t, func() bool {
			select {
			case err := <-done:
				t.Fatalf("fixture ended at cycle%d: %v", i, err)
			default:
			}
			_, err := ReadResult(config.ControlDir, id)
			return err == nil
		})
		for _, sub := range []string{"commands", "results"} {
			entries, err := filepath.Glob(filepath.Join(config.ControlDir, sub, "*.json"))
			if err != nil {
				t.Fatal(err)
			}
			if sub == "commands" {
				maxPending = max(maxPending, len(entries))
			} else {
				maxResults = max(maxResults, len(entries))
			}
			if len(entries) > 1 {
				t.Fatalf("cycle%d %s retained%d entries", i, sub, len(entries))
			}
		}
		if err := RetireResult(config.ControlDir, id); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("cycles=%d max_pending=%d max_results=%d", 2*MaxRequests, maxPending, maxResults)
}

func TestRunRefusesInFlightAndUnreadDuplicateIDs(t *testing.T) {
	config := fixtureConfig(t, "claude")
	input, writer := io.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Run(ctx, config, input, io.Discard); close(done) }()
	t.Cleanup(func() { cancel(); _ = writer.Close(); <-done })
	waitFixture(t, func() bool { _, err := os.Stat(filepath.Join(config.ControlDir, "ready.json")); return err == nil })
	marker := filepath.Join(config.Root, "running")
	release := filepath.Join(config.Root, "release")
	command := Command{ID: "duplicate", Kind: Exec, Args: []string{"-c", `printf started > "$1"; while [ ! -e "$2" ]; do sleep 0.01; done`, "fixture", marker, release}}
	if err := WriteCommand(config.ControlDir, command); err != nil {
		t.Fatal(err)
	}
	waitFixture(t, func() bool { _, err := os.Stat(marker); return err == nil })
	if err := WriteCommand(config.ControlDir, command); err == nil {
		t.Fatal("duplicate in-flight ID admitted")
	}
	if err := os.WriteFile(release, []byte("release"), 0600); err != nil {
		t.Fatal(err)
	}
	waitFixture(t, func() bool { _, err := ReadResult(config.ControlDir, command.ID); return err == nil })
	if err := WriteCommand(config.ControlDir, command); err == nil {
		t.Fatal("unread completed ID admitted")
	}
	if err := RetireResult(config.ControlDir, command.ID); err != nil {
		t.Fatal(err)
	}
	if err := WriteCommand(config.ControlDir, Command{ID: command.ID, Kind: SetState, State: Idle}); err != nil {
		t.Fatalf("retired ID remained reserved: %v", err)
	}
	waitFixture(t, func() bool { _, err := ReadResult(config.ControlDir, command.ID); return err == nil })
}

func TestWriteCommandConcurrentAdmissionCountsOnlyPublishedRequests(t *testing.T) {
	for round := 0; round < 8; round++ {
		t.Run(fmt.Sprintf("round-%d", round), func(t *testing.T) {
			config := fixtureConfig(t, "claude")
			if err := atomicJSON(filepath.Join(config.ControlDir, "ready.json"), Ready{PID: os.Getpid(), RunID: config.RunID}, false); err != nil {
				t.Fatal(err)
			}
			for i := 0; i < MaxRequests-1; i++ {
				if err := WriteCommand(config.ControlDir, Command{ID: fmt.Sprintf("filled-%03d", i), Kind: SetState, State: Idle}); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(config.ControlDir, "commands", ".publish-held"), []byte("in progress"), 0600); err != nil {
				t.Fatal(err)
			}
			start := make(chan struct{})
			outcomes := make(chan error, 32)
			for i := 0; i < 32; i++ {
				go func(i int) {
					<-start
					outcomes <- WriteCommand(config.ControlDir, Command{ID: fmt.Sprintf("racer-%03d", i), Kind: SetState, State: Idle})
				}(i)
			}
			close(start)
			accepted := 0
			for i := 0; i < 32; i++ {
				if err := <-outcomes; err == nil {
					accepted++
				} else if !errors.Is(err, ErrMailboxFull) {
					t.Fatalf("unexpected admission error: %v", err)
				}
			}
			entries, err := os.ReadDir(filepath.Join(config.ControlDir, "commands"))
			if err != nil {
				t.Fatal(err)
			}
			published := 0
			for _, entry := range entries {
				if !strings.HasPrefix(entry.Name(), ".publish-") {
					published++
				}
			}
			if accepted != 1 || published != MaxRequests {
				t.Fatalf("accepted=%d actual_pending=%d; want one admission to128", accepted, published)
			}
			t.Logf("writers=32 accepted=%d actual_pending=%d", accepted, published)
		})
	}
}
