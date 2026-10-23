package couchcmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
	"github.com/xianxu/pair/cmd/internal/testwait"
)

// Publication discovers this legacy source only after wireResolver's actual
// attach succeeds. Holding that Git boundary distinguishes publication from
// the earlier lifecycle operation and from terminal installation itself.
type publicationBarrierGit struct {
	couchcore.GitRunner
	path    string
	entered chan struct{}
	release chan struct{}
	once    sync.Once
}

func (g *publicationBarrierGit) RunContext(ctx context.Context, dir string, args ...string) (string, error) {
	if dir == g.path && reflect.DeepEqual(args, []string{"rev-parse", "--git-common-dir"}) {
		g.once.Do(func() { close(g.entered) })
		select {
		case <-g.release:
			return "/repo/.git", nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	return g.GitRunner.RunContext(ctx, dir, args...)
}

func TestRepositoryPublicationKeepsConsoleResponsiveAndRemotePending(t *testing.T) {
	rt := newRT(t, "/repo")
	rt.runner = couchcore.NewFakeRunner()
	c, err := rt.NewCouch()
	if err != nil {
		t.Fatal(err)
	}
	actor, handle, err := c.Spawn(couchcore.StartArgs{Worktree: "/repo", Stack: "codex", ExtraArgs: []string{"--model", "accepted"}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { rt.runner.SetExited(handle.ID(), 0) })
	start, _ := (couchcore.StartResult{Record: actor, Handle: handle}).Started()
	if start.Configuration == nil {
		t.Fatal("explicit launch lost configuration acceptance")
	}
	git := &publicationBarrierGit{GitRunner: c.Git, path: "/publication-legacy", entered: make(chan struct{}), release: make(chan struct{})}
	c.Git = git
	var release sync.Once
	unblock := func() { release.Do(func() { close(git.release) }) }
	t.Cleanup(unblock)

	// Install the old on-disk format after launch, so only adoption discovers
	// it. Use its real content-addressed filename, as the reader validates it.
	legacy := couchcore.PathLaunchPreference{SchemaVersion: couchcore.PathLaunchPreferenceSchemaVersion, RepoIdentity: "/repo/.git", PhysicalPath: git.path, LastAgent: "codex", ArgvByAgent: map[string][]string{"codex": {"--model", "legacy"}}, Revision: 1}
	raw, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte(legacy.RepoIdentity + "\x00" + legacy.PhysicalPath))
	legacyPath := filepath.Join(c.Namespace.Dir(), "threadstore", "path-preferences", fmt.Sprintf("%x.json", digest))
	if err := os.MkdirAll(filepath.Dir(legacyPath), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyPath, raw, 0600); err != nil {
		t.Fatal(err)
	}

	host := hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 100})
	reader, input := io.Pipe()
	t.Cleanup(func() { _ = reader.Close(); _ = input.Close() })
	console := couchtty.New(host, reader)
	initial := ptychild.NewFakeChild(nil)
	console.Attach("initial-console-pane", "initial", initial)
	t.Cleanup(func() { initial.Exit(0) })
	wireResolver(console, c)
	// Inventory is irrelevant to this test and must not independently discover
	// the same legacy source: entering the barrier must certify publication.
	console.SetActionableProvider(func(context.Context, []couchcore.LiveTTYObservation) ([]couchcore.ActionableThreadSummary, error) {
		return nil, nil
	})
	actual := console.Ops()
	console.SetOperationDispatcher(func(call couchcore.OperationCall) (any, error) {
		if call.Name == "resume" {
			return start, nil // Isolate completion ownership from resume admission.
		}
		return actual(call)
	})
	done := make(chan int, 1)
	go func() { done <- console.Run() }()
	t.Cleanup(func() {
		unblock()
		console.Stop()
		_ = input.Close()
		select {
		case <-done:
		case <-testwait.After(t):
			t.Error("Console did not join publication")
		}
	})
	finished := make(chan error, 2)
	if err := console.EnqueueRemoteOperation("publication-held", "resume", func(ctx context.Context) (couchcore.OperationCall, error) {
		return couchcore.OperationCall{Name: "resume", Context: ctx}, nil
	}, func() {}, func(_ any, err error) { finished <- err }); err != nil {
		t.Fatal(err)
	}
	select {
	case <-git.entered:
	case <-testwait.After(t):
		t.Fatal("publication did not reach legacy discovery after actual attach")
	}
	assertPending := func() {
		t.Helper()
		select {
		case err := <-finished:
			t.Fatalf("remote callback preceded publication: %v", err)
		default:
		}
	}
	assertPending()
	if _, err := input.Write([]byte("publication-input")); err != nil {
		t.Fatal(err)
	}
	testwait.Until(t, "ordinary input while publication is held", func() bool {
		return bytes.Contains(bytes.Join(initial.Writes(), nil), []byte("publication-input"))
	})
	// Deliver requires an acknowledgment from Run's chunk case; observing the
	// resulting frame proves the event loop still reduces and renders output.
	initial.SetSink(func(ctx context.Context, batch ptychild.OutputBatch) error {
		bounded, cancel := context.WithTimeout(ctx, testwait.Budget(t))
		defer cancel()
		return console.Deliver(bounded, "initial-console-pane", batch)
	})
	initial.Feed([]byte("publication-render"))
	testwait.Until(t, "render while publication is held", func() bool { return strings.Contains(host.Written(), "publication-render") })
	assertPending()
	unblock()
	select {
	case err := <-finished:
		if err != nil {
			t.Fatal(err)
		}
	case <-testwait.After(t):
		t.Fatal("remote callback did not follow publication")
	}
	raw, err = os.ReadFile(filepath.Join(c.Namespace.Dir(), "threadstore", "repo-preferences.json"))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := couchcore.DecodeRepoPreferences(raw)
	if err != nil {
		t.Fatal(err)
	}
	want := doc.Repositories["/repo/.git"].Agents["codex"]
	if !want.Confirmed || !reflect.DeepEqual(want.Argv, []string{"--model", "accepted"}) || start.Warning() != "" {
		t.Fatalf("publication incomplete at callback: preference=%+v warning=%q", want, start.Warning())
	}
}
