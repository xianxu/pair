package testcouchbridge

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testcouch"
	"github.com/xianxu/pair/cmd/internal/testpair"
)

type fixtureNativeFleet struct {
	facts                     nativeFacts
	prepared, started, closed bool
	closeErr                  error
	lifetime                  context.Context
}

func (f *fixtureNativeFleet) Prepare(context.Context) error { f.prepared = true; return nil }
func (f *fixtureNativeFleet) Start(context.Context) error   { f.started = true; return nil }
func (f *fixtureNativeFleet) Done() <-chan struct{}         { return nil }
func (f *fixtureNativeFleet) Close(context.Context) error   { f.closed = true; return f.closeErr }
func (f *fixtureNativeFleet) AssertCleaned() error {
	if !f.closed || f.closeErr != nil {
		return errors.New("not joined")
	}
	return nil
}
func (f *fixtureNativeFleet) Facts() nativeFacts { return f.facts }

func nativeFixture(t *testing.T) (*NativeOwner, *fixtureNativeFleet, *int) {
	t.Helper()
	root := testpair.ShortRoot(t)
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0700); err != nil {
		t.Fatal(err)
	}
	store := filepath.Join(root, "store")
	if err := os.Mkdir(store, 0700); err != nil {
		t.Fatal(err)
	}
	f := &fixtureNativeFleet{facts: nativeFacts{root: root, namespace: store, dir: root, couch: filepath.Join(bin, "couch"), env: []string{"COUCH_ISOLATED_ROOT=" + root, "COUCH_STORE_DIR=" + store, "PATH=" + bin, "HOME=" + filepath.Join(root, "home")}}}
	candidate := t.TempDir()
	calls := 0
	deps := nativeDependencies{source: func() (string, error) { return root, nil }, build: func(context.Context, testcouch.BuildConfig) (*testcouch.CandidateOwner, error) {
		return &testcouch.CandidateOwner{Binaries: testcouch.Binaries{Dir: candidate}}, nil
	}, newFleet: func(ctx context.Context, _ testcouch.Binaries, _ testcouch.Options, _ testcouch.OwnerConfig) (nativeFleet, error) {
		f.lifetime = ctx
		return f, nil
	}, run: func(ctx context.Context, program string, args []string, dir string, env []string) ([]byte, error) {
		calls++
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 3*time.Second {
			t.Fatal("CLI call lacks its 3s sub-bound")
		}
		if program != f.facts.couch || dir != root || strings.Join(env, "\n") != strings.Join(f.facts.env, "\n") {
			t.Fatalf("CLI authority changed: %s %s %v", program, dir, env)
		}
		if args[0] == "--session-sync" {
			id := "0123456789abcdef0123456789abcdef"
			if strings.Contains(args[2], "second.md") {
				id = "fedcba9876543210fedcba9876543210"
			}
			return json.Marshal(map[string]any{"Code": "ok", "Error": "", "Registration": map[string]any{"ID": id, "Namespace": store, "File": args[2], "Owner": "fixture:0"}})
		}
		if args[0] != "--editor-endpoint" || args[1] != "lookup" {
			t.Fatalf("unexpected roundtrip %v", args)
		}
		return []byte(`{"Version":1,"Code":"ok","Error":"","EditorEndpoint":null}`), nil
	}}
	return newNativeOwner(Options{ScratchParent: root}, io.Discard, deps), f, &calls
}

func TestNativeOwnerReadinessRegistersActualParentAndSideDocuments(t *testing.T) {
	owner, f, calls := nativeFixture(t)
	if *calls != 0 || f.prepared {
		t.Fatal("constructor performed effects")
	}
	startup, cancel := context.WithCancel(t.Context())
	if err := owner.Prepare(startup); err != nil {
		t.Fatal(err)
	}
	if err := owner.Start(startup); err != nil {
		t.Fatal(err)
	}
	r, err := owner.Readiness(startup)
	if err != nil {
		t.Fatal(err)
	}
	if *calls != 4 || !f.prepared || !f.started {
		t.Fatalf("missing actual registration/lookup: calls=%d", *calls)
	}
	if err := ValidateReadiness(r); err != nil {
		t.Fatal(err)
	}
	side, err := os.ReadFile(r.Documents[1].File)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(side), "type: side-doc") || !strings.Contains(string(side), "parent: ../first.md") {
		t.Fatalf("side layout not pinned: %s", side)
	}
	cancel()
	if f.lifetime.Err() != nil {
		t.Fatal("startup context killed admitted lifetime")
	}
	if err := owner.Close(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := owner.AssertCleaned(); err != nil {
		t.Fatal(err)
	}
	if root := owner.RetainedRoot(); root != "" {
		t.Fatalf("cleaned root falsely retained: %s", root)
	}
}

func TestNativeOwnerDoesNotDeleteCandidatesWhenCleanupUncertain(t *testing.T) {
	owner, f, _ := nativeFixture(t)
	if err := owner.Prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	f.closeErr = errors.New("generation remains")
	candidate := owner.candidate.Binaries.Dir
	if err := owner.Close(t.Context()); err == nil {
		t.Fatal("uncertain cleanup passed")
	}
	if _, err := os.Stat(candidate); err != nil {
		t.Fatalf("candidate evidence deleted: %v", err)
	}
	if owner.RetainedRoot() != f.facts.root {
		t.Fatal("missing retained fleet root")
	}
}

func TestNativeOwnerRefusesUnconfirmedServiceReadiness(t *testing.T) {
	for _, tc := range []struct {
		name         string
		registration bool
		reply        string
	}{
		{"foreign-registration", true, `{"Code":"ok","Error":"","Registration":{"ID":"0123456789abcdef0123456789abcdef","Namespace":"/foreign","File":"/foreign","Owner":"fixture:0"}}`},
		{"duplicate-code", false, `{"Version":1,"Code":"unavailable","Code":"ok","Error":"","EditorEndpoint":null}`},
		{"unavailable-service", false, `{"Version":1,"Code":"unavailable","Error":"down","EditorEndpoint":null}`},
		{"missing-endpoint-confirmation", false, `{"Version":1,"Code":"ok","Error":""}`},
		{"unexpected-live-endpoint", false, `{"Version":1,"Code":"ok","Error":"","EditorEndpoint":{}}`},
		{"malformed-service", false, `not JSON`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			owner, _, _ := nativeFixture(t)
			run := owner.dependencies.run
			owner.dependencies.run = func(ctx context.Context, program string, args []string, dir string, env []string) ([]byte, error) {
				if (args[0] == "--session-sync") == tc.registration {
					return []byte(tc.reply), nil
				}
				return run(ctx, program, args, dir, env)
			}
			if err := owner.Prepare(t.Context()); err != nil {
				t.Fatal(err)
			}
			if err := owner.Start(t.Context()); err != nil {
				t.Fatal(err)
			}
			if _, err := owner.Readiness(t.Context()); err == nil {
				t.Fatal("unconfirmed readiness accepted")
			}
			if !owner.reporter.Failed() {
				t.Fatal("failure evidence not requested")
			}
			if err := owner.Close(t.Context()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestNativeOwnerPartialBuildIsClosedBySameOwner(t *testing.T) {
	owner, _, _ := nativeFixture(t)
	candidate := t.TempDir()
	failure := errors.New("offline build failed")
	owner.dependencies.build = func(context.Context, testcouch.BuildConfig) (*testcouch.CandidateOwner, error) {
		return &testcouch.CandidateOwner{Binaries: testcouch.Binaries{Dir: candidate}}, failure
	}
	if err := owner.Prepare(t.Context()); !errors.Is(err, failure) {
		t.Fatalf("unexpected build result %v", err)
	}
	if owner.RetainedRoot() != candidate {
		t.Fatal("partial build root lost")
	}
	if _, err := os.Stat(candidate); err != nil {
		t.Fatal(err)
	}
	if err := owner.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := owner.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := owner.AssertCleaned(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(candidate); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("candidate not retired: %v", err)
	}
}

func TestNativeOwnerCodexQualificationSeamDoesNotAddConsumerFlag(t *testing.T) {
	owner, _, _ := nativeFixture(t)
	owner.dependencies.agentKind = "codex"
	newFleet := owner.dependencies.newFleet
	selected := ""
	owner.dependencies.newFleet = func(ctx context.Context, b testcouch.Binaries, o testcouch.Options, c testcouch.OwnerConfig) (nativeFleet, error) {
		selected = o.AgentKind
		return newFleet(ctx, b, o, c)
	}
	if err := owner.Prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	if selected != "codex" {
		t.Fatalf("fixture kind %q", selected)
	}
	if err := owner.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func TestNativeOwnerToleratesAdditiveReplyFields(t *testing.T) {
	owner, _, _ := nativeFixture(t)
	run := owner.dependencies.run
	owner.dependencies.run = func(ctx context.Context, program string, args []string, dir string, env []string) ([]byte, error) {
		raw, err := run(ctx, program, args, dir, env)
		if err != nil {
			return raw, err
		}
		return []byte(strings.TrimSuffix(string(raw), "}") + `,"Future":{"Counter":1234567890123456789012345678901234567890}}`), nil
	}
	if err := owner.Prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := owner.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Readiness(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := owner.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
}
