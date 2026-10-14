package couchcore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/orientation"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

func TestSwitchContextUsesExactOutgoingOwner(t *testing.T) {
	t.Parallel()

	record := verifiedResumeThread(t)
	dataDir := t.TempDir()
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: dataDir, RepoScope: record.Address.RepoScope, Tag: string(record.Address.Tag)})
	if err != nil {
		t.Fatal(err)
	}
	native := filepath.Join(t.TempDir(), "root.jsonl")
	for _, path := range []string{paths.Log(), native} {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("evidence"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	calls := 0
	resolver := OSSwitchContextResolver{DataDir: dataDir, HomeDir: t.TempDir(), Renderer: "/bin/pair", Query: func(_ context.Context, _ sessioninventory.Runtime, scope, tag string, agent sessioninventory.Agent) (sessioninventory.SessionQuery, error) {
		calls++
		if scope != record.Address.RepoScope || tag != string(record.Address.Tag) || agent != "codex" {
			t.Fatalf("wrong owner %s %s %s", scope, tag, agent)
		}
		return sessioninventory.SessionQuery{Status: sessioninventory.BindingEstablished, Root: &sessioninventory.Node{NativeID: "source-native", Artifacts: []sessioninventory.Artifact{{StorageRoot: "codex-sessions", RelativePath: "root.jsonl", Kind: sessioninventory.ArtifactTranscript}}}}, nil
	}, NativePath: func(_ sessioninventory.Runtime, artifact sessioninventory.Artifact) (string, error) {
		if artifact.RelativePath != "root.jsonl" {
			t.Fatal(artifact)
		}
		return native, nil
	}}
	got, err := resolver.Resolve(context.Background(), record)
	if err != nil {
		t.Fatal(err)
	}
	if got.SourceSession != "source-native" || !reflect.DeepEqual(got.NativeTranscripts, []string{native}) || got.PairLog != paths.Log() {
		t.Fatalf("context = %+v", got)
	}
	if calls != 1 {
		t.Fatalf("native query calls=%d", calls)
	}
}

func TestSwitchContextMissingAndAmbiguousSourcesStayUnavailable(t *testing.T) {
	t.Parallel()

	record := verifiedResumeThread(t)
	resolver := OSSwitchContextResolver{DataDir: t.TempDir(), HomeDir: t.TempDir(), Query: func(context.Context, sessioninventory.Runtime, string, string, sessioninventory.Agent) (sessioninventory.SessionQuery, error) {
		return sessioninventory.SessionQuery{Status: sessioninventory.BindingAmbiguous, Root: &sessioninventory.Node{NativeID: "must-not-use"}}, nil
	}}
	got, err := resolver.Resolve(context.Background(), record)
	if err != nil {
		t.Fatal(err)
	}
	if got.SourceSession != "" || len(got.NativeTranscripts) != 0 || len(got.Unavailable) == 0 {
		t.Fatalf("guessed context: %+v", got)
	}
	got.TargetAgent = "claude"
	body, err := orientation.BuildPrompt(got)
	if err != nil || !strings.Contains(body, "unavailable") {
		t.Fatalf("missing context prevented prompt: %v %s", err, body)
	}
}

func TestOrientationStatusRequiresExactReadyIdentity(t *testing.T) {
	t.Parallel()

	address := verifiedResumeThread(t).Address
	dataDir := t.TempDir()
	paths, _ := artifactpath.Resolve(artifactpath.Address{DataDir: dataDir, RepoScope: address.RepoScope, Tag: string(address.Tag)})
	readyPath, _ := paths.AgentReadyChecked("codex")
	if err := os.MkdirAll(filepath.Dir(readyPath), 0700); err != nil {
		t.Fatal(err)
	}
	ready := readiness.ReadyRecord{Tag: string(address.Tag), Agent: "codex", Session: "pair-exact", Nonce: "attempt-1", PID: os.Getpid()}
	reader := OSReadyReader{DataDir: dataDir, Session: func(ThreadAddress) (PairSessionBinding, error) {
		return PairSessionBinding{Name: "pair-exact", Present: true}, nil
	}, Proc: OSProcOps{}}
	if registered, err := reader.Registered(context.Background(), address, "codex", "attempt-1"); err != nil || registered {
		t.Fatalf("a missing ready record registered: %v %v", registered, err)
	}
	for _, change := range []string{"valid", "nonce", "session", "tag", "agent", "oversized"} {
		candidate := ready
		switch change {
		case "oversized":
			candidate.Tag = strings.Repeat("x", 20*1024)
		case "nonce":
			candidate.Nonce = "other"
		case "session":
			candidate.Session = "other"
		case "tag":
			candidate.Tag = "other"
		case "agent":
			candidate.Agent = "claude"
		}
		raw, err := readiness.Encode(candidate)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(readyPath, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		registered, err := reader.Registered(context.Background(), address, "codex", "attempt-1")
		if change == "valid" {
			if err != nil || !registered {
				t.Fatalf("valid ready record = %v %v", registered, err)
			}
		} else if err == nil && registered {
			t.Fatalf("accepted obsolete %s", change)
		}
		if change == "oversized" && !strings.Contains(err.Error(), "size limit") {
			t.Fatalf("unbounded ready read: %v", err)
		}
	}
}

func TestFreshRegistrationAcceptsTheExactReadyRecord(t *testing.T) {
	t.Parallel()

	record := verifiedResumeThread(t)
	paths, _ := artifactpath.Resolve(artifactpath.Address{DataDir: t.TempDir(), RepoScope: record.Address.RepoScope, Tag: string(record.Address.Tag)})
	path, _ := paths.AgentReadyChecked("codex")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	ready := readiness.ReadyRecord{Tag: string(record.Address.Tag), Agent: "codex", Session: "exact-session", Nonce: "attempt", PID: os.Getpid()}
	raw, err := readiness.Encode(ready)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	reader := OSReadyReader{DataDir: filepath.Dir(filepath.Dir(paths.ScopeDir())), Session: func(ThreadAddress) (PairSessionBinding, error) {
		return PairSessionBinding{Name: "exact-session", Present: true}, nil
	}, Proc: OSProcOps{}}
	registered, err := reader.Registered(context.Background(), record.Address, "codex", "attempt")
	if err != nil || !registered {
		t.Fatalf("exact child readiness rejected: %v %v", registered, err)
	}
}

func TestFreshRegistrationWaitsForOldReadyReplacement(t *testing.T) {
	t.Parallel()

	record := verifiedResumeThread(t)
	dataDir := t.TempDir()
	paths, _ := artifactpath.Resolve(artifactpath.Address{DataDir: dataDir, RepoScope: record.Address.RepoScope, Tag: string(record.Address.Tag)})
	path, _ := paths.AgentReadyChecked("codex")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	reader := OSReadyReader{DataDir: dataDir, Session: func(ThreadAddress) (PairSessionBinding, error) {
		return PairSessionBinding{Name: "session", Present: true}, nil
	}, Proc: OSProcOps{}}
	for _, nonce := range []string{"old-attempt", "new-attempt"} {
		raw, err := readiness.Encode(readiness.ReadyRecord{Tag: string(record.Address.Tag), Agent: "codex", Session: "session", Nonce: nonce, PID: os.Getpid()})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		registered, err := reader.Registered(context.Background(), record.Address, "codex", "new-attempt")
		if err != nil || registered != (nonce == "new-attempt") {
			t.Fatalf("%s registered=%v err=%v", nonce, registered, err)
		}
	}
}

func TestSwitchContextRejectsNonRegularEvidenceWithoutBlocking(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("data"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if readableSwitchFile(link) {
		t.Fatal("symlink accepted as exact archive evidence")
	}
	fifo := filepath.Join(dir, "fifo")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	done := make(chan bool, 1)
	go func() { done <- readableSwitchFile(fifo) }()
	select {
	case readable := <-done:
		if readable {
			t.Fatal("FIFO accepted")
		}
	case <-testwait.After(t):
		t.Fatal("context lookup blocked on FIFO")
	}
}

func TestContinuationGenerationReceiptSurvivesTargetDeath(t *testing.T) {
	t.Parallel()

	record := verifiedResumeThread(t)
	data := t.TempDir()
	paths, _ := artifactpath.Resolve(artifactpath.Address{DataDir: data, RepoScope: record.Address.RepoScope, Tag: string(record.Address.Tag)})
	path, _ := paths.AgentReadyChecked("codex")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	ready := readiness.ReadyRecord{Tag: string(record.Address.Tag), Agent: "codex", Session: "exact-session", Nonce: "attempt", PID: 42, LaunchOrdinal: 17}
	raw, err := readiness.Encode(ready)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	reader := OSReadyReader{DataDir: data, Session: func(ThreadAddress) (PairSessionBinding, error) {
		return PairSessionBinding{Name: "exact-session", Present: false}, nil
	}, Proc: NewFakeProcOps()}
	generation, err := reader.Generation(context.Background(), record.Address, "codex", "attempt")
	if err != nil || generation == nil || generation.LaunchOrdinal != 17 || generation.Attempt != "attempt" {
		t.Fatalf("lost durable correlation: %+v %v", generation, err)
	}
	registered, err := reader.Registered(context.Background(), record.Address, "codex", "attempt")
	if err != nil || registered {
		t.Fatalf("dead receipt became live proof: %v %v", registered, err)
	}
	foreign, err := reader.Generation(context.Background(), record.Address, "codex", "obsolete")
	if err != nil || foreign != nil {
		t.Fatalf("foreign attempt correlated: %+v %v", foreign, err)
	}
}

func TestOrientationSessionObservationHonorsCallerDeadline(t *testing.T) {
	t.Parallel()

	for _, operation := range []string{"generation", "registered"} {
		t.Run(operation, func(t *testing.T) {
			record := verifiedResumeThread(t)
			data := t.TempDir()
			paths, _ := artifactpath.Resolve(artifactpath.Address{DataDir: data, RepoScope: record.Address.RepoScope, Tag: string(record.Address.Tag)})
			path, _ := paths.AgentReadyChecked("codex")
			if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
				t.Fatal(err)
			}
			raw, err := readiness.Encode(readiness.ReadyRecord{Tag: string(record.Address.Tag), Agent: "codex", Session: "exact-session", Nonce: "attempt", PID: 42, LaunchOrdinal: 17})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
				t.Fatal(err)
			}
			// The deadline lapses when the observer has the context in hand,
			// not 20ms after creation: a loaded run used to spend it reading
			// the ready file, before the observer was ever asked (#480).
			budget := newSpendableBudget()
			ctx, cancel := budget.bind(context.Background())
			defer cancel()
			observed := false
			reader := OSReadyReader{DataDir: data, Session: func(ThreadAddress) (PairSessionBinding, error) {
				t.Fatal("legacy observer bypassed caller context")
				return PairSessionBinding{}, nil
			}, SessionContext: func(got context.Context, _ ThreadAddress) (PairSessionBinding, error) {
				observed = true
				wantDeadline, _ := ctx.Deadline()
				gotDeadline, _ := got.Deadline()
				if !gotDeadline.Equal(wantDeadline) {
					t.Fatalf("deadline lost: %v want %v", gotDeadline, wantDeadline)
				}
				budget.spend()
				<-got.Done()
				return PairSessionBinding{}, got.Err()
			}}
			switch operation {
			case "generation":
				_, err = reader.Generation(ctx, record.Address, "codex", "attempt")
			case "registered":
				_, err = reader.Registered(ctx, record.Address, "codex", "attempt")
			}
			if !observed || !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("observed=%v err=%v", observed, err)
			}
		})
	}
}

// RecordedSession reads only the ready file: it never asks the session
// observer, whose production form runs the ps/zellij ownership probe (#365).
func TestRecordedSessionReadsReadyFileWithoutSessionObservation(t *testing.T) {
	t.Parallel()

	address := verifiedResumeThread(t).Address
	dataDir := t.TempDir()
	paths, _ := artifactpath.Resolve(artifactpath.Address{DataDir: dataDir, RepoScope: address.RepoScope, Tag: string(address.Tag)})
	readyPath, _ := paths.AgentReadyChecked("codex")
	if err := os.MkdirAll(filepath.Dir(readyPath), 0700); err != nil {
		t.Fatal(err)
	}
	reader := OSReadyReader{DataDir: dataDir, Session: func(ThreadAddress) (PairSessionBinding, error) {
		t.Fatal("RecordedSession observed the Pair session")
		return PairSessionBinding{}, nil
	}, Proc: OSProcOps{}}
	if _, ok, err := reader.RecordedSession(context.Background(), address, "codex", "attempt-1"); ok || err != nil {
		t.Fatalf("missing ready file: ok=%v err=%v", ok, err)
	}
	raw, err := readiness.Encode(readiness.ReadyRecord{Tag: string(address.Tag), Agent: "codex", Session: "pair-exact", Nonce: "attempt-1", PID: os.Getpid()})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(readyPath, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	if session, ok, err := reader.RecordedSession(context.Background(), address, "codex", "attempt-1"); !ok || err != nil || session != "pair-exact" {
		t.Fatalf("recorded = %q %v %v", session, ok, err)
	}
	if _, ok, err := reader.RecordedSession(context.Background(), address, "codex", "attempt-2"); ok || err != nil {
		t.Fatalf("another nonce's ready file vouched: ok=%v err=%v", ok, err)
	}
}
