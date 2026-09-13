package couchcore

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/orientation"
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
	"github.com/xianxu/pair/cmd/internal/readiness"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func TestSwitchContextUsesExactOutgoingOwnerAndArchive(t *testing.T) {
	record := verifiedResumeThread(t)
	descriptor := &pairlifecycle.PreservedScrollback{Agent: "codex", Token: "exact-capture", Events: true}
	record.VerifiedPark.Scrollback = descriptor
	dataDir := t.TempDir()
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: dataDir, RepoScope: record.Address.RepoScope, Tag: string(record.Address.Tag)})
	if err != nil {
		t.Fatal(err)
	}
	archive, _ := paths.ParkedScrollbackArtifacts(descriptor.Token)
	native := filepath.Join(t.TempDir(), "root.jsonl")
	for _, path := range []string{archive.Raw, archive.Events, paths.Log(), native} {
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
	if err := resolver.ResolveArchive(record, &got); err != nil {
		t.Fatal(err)
	}
	if got.ScrollbackRaw != archive.Raw || got.ScrollbackEvents != archive.Events || got.Renderer != "/bin/pair" || calls != 1 {
		t.Fatalf("archive lost or native rescanned: %+v calls=%d", got, calls)
	}
	record.VerifiedPark.Scrollback = nil
	if err := resolver.ResolveArchive(record, &got); err != nil {
		t.Fatal(err)
	}
	if got.ScrollbackRaw != "" || got.ScrollbackEvents != "" {
		t.Fatal("stale archive retained", got)
	}
}

func TestSwitchContextMissingAndAmbiguousSourcesStayUnavailable(t *testing.T) {
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
	if err := resolver.ResolveArchive(record, &got); err != nil {
		t.Fatal(err)
	}
	got.TargetAgent = "claude"
	body, err := orientation.BuildPrompt(got)
	if err != nil || !strings.Contains(body, "unavailable") {
		t.Fatalf("missing context prevented prompt: %v %s", err, body)
	}
}

func TestOrientationStatusRequiresExactReadyIdentity(t *testing.T) {
	address := verifiedResumeThread(t).Address
	dataDir := t.TempDir()
	paths, _ := artifactpath.Resolve(artifactpath.Address{DataDir: dataDir, RepoScope: address.RepoScope, Tag: string(address.Tag)})
	readyPath, _ := paths.AgentReadyChecked("codex")
	if err := os.MkdirAll(filepath.Dir(readyPath), 0700); err != nil {
		t.Fatal(err)
	}
	ready := readiness.ReadyRecord{Tag: string(address.Tag), Agent: "codex", Session: "pair-exact", Nonce: "attempt-1", PID: os.Getpid(), Orientation: &orientation.DeliveryState{Phase: orientation.DeliverySubmitted, BodyWritten: true}}
	reader := OSOrientationStatusReader{DataDir: dataDir, Session: func(ThreadAddress) (PairSessionBinding, error) {
		return PairSessionBinding{Name: "pair-exact", Present: true}, nil
	}, Proc: OSProcOps{}}
	state, err := reader.Read(context.Background(), address, "codex", "attempt-1")
	if err != nil || state.Phase != orientation.DeliveryWaiting {
		t.Fatalf("missing status = %+v %v", state, err)
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
		state, err := reader.Read(context.Background(), address, "codex", "attempt-1")
		if change == "valid" {
			if err != nil || state.Phase != orientation.DeliverySubmitted {
				t.Fatalf("valid status=%+v %v", state, err)
			}
		} else if err == nil {
			t.Fatalf("accepted obsolete %s", change)
		}
		if change == "oversized" && !strings.Contains(err.Error(), "size limit") {
			t.Fatalf("unbounded ready read: %v", err)
		}
	}
}

func TestFreshRegistrationAcceptsReadyBeforeOrientation(t *testing.T) {
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
	reader := OSOrientationStatusReader{DataDir: filepath.Dir(filepath.Dir(paths.ScopeDir())), Session: func(ThreadAddress) (PairSessionBinding, error) {
		return PairSessionBinding{Name: "exact-session", Present: true}, nil
	}, Proc: OSProcOps{}}
	registered, err := reader.Registered(context.Background(), record.Address, "codex", "attempt")
	if err != nil || !registered {
		t.Fatalf("exact child readiness rejected: %v %v", registered, err)
	}
	state, err := reader.Read(context.Background(), record.Address, "codex", "attempt")
	if err != nil || state.Phase != orientation.DeliveryWaiting {
		t.Fatalf("orientation not waiting: %+v %v", state, err)
	}
}

func TestReadOrientationStatusRejectsReplacedIncarnation(t *testing.T) {
	store, _, record := createControllerThread(t)
	proc := NewFakeProcOps()
	proc.Set(42, "pair-helper")
	calls := 0
	couch := &Couch{Threads: store, Proc: proc, OrientationStatus: func(context.Context, ThreadAddress, string, string) (orientation.DeliveryState, error) {
		calls++
		return orientation.DeliveryState{Phase: orientation.DeliverySubmitted, BodyWritten: true}, nil
	}}
	if _, err := couch.ReadOrientationStatus(context.Background(), record.Address, "codex", "attempt"); err != nil || calls != 1 {
		t.Fatalf("valid incarnation rejected: %v", err)
	}
	proc.Set(42, "replacement")
	if _, err := couch.ReadOrientationStatus(context.Background(), record.Address, "codex", "attempt"); err == nil || calls != 1 {
		t.Fatal("read status from replaced process")
	}
	proc.Set(42, "pair-helper")
	if _, err := couch.ReadOrientationStatus(context.Background(), record.Address, "claude", "attempt"); err == nil || calls != 1 {
		t.Fatal("read status from different agent")
	}
}

func TestFreshRegistrationWaitsForOldReadyReplacement(t *testing.T) {
	record := verifiedResumeThread(t)
	dataDir := t.TempDir()
	paths, _ := artifactpath.Resolve(artifactpath.Address{DataDir: dataDir, RepoScope: record.Address.RepoScope, Tag: string(record.Address.Tag)})
	path, _ := paths.AgentReadyChecked("codex")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	reader := OSOrientationStatusReader{DataDir: dataDir, Session: func(ThreadAddress) (PairSessionBinding, error) {
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
	case <-time.After(time.Second):
		t.Fatal("context lookup blocked on FIFO")
	}
}
