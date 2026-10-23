package testcouchbridge

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/testcouch"
	"github.com/xianxu/pair/cmd/internal/testpair"
	"github.com/xianxu/pair/internal/reporoot"
)

// TestNativeCouchBridge qualifies the production bridge reducer and pipe owner
// with real shared-fleet provisioning. The consumer's editor-registration probe
// remains parley.nvim#405; this check proves the producer's advertised documents,
// selected native editor, actual endpoint and joined EOF/TERM ownership.
func TestNativeCouchBridge(t *testing.T) {
	if os.Getenv("PAIR_NATIVE_COUCH_BRIDGE") != "1" {
		t.Skip("opt-in native Couch bridge qualification")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 630*time.Second)
	defer cancel()
	source, err := reporoot.Current()
	if err != nil {
		t.Fatal(err)
	}
	source, err = filepath.EvalSymlinks(source)
	if err != nil {
		t.Fatal(err)
	}
	// ShortRoot selects the supported physical socket parent. Each actual fleet
	// still allocates and owns its own distinct generation through NewOwned.
	scratch := filepath.Dir(testpair.ShortRoot(t))
	selectedRoot := testpair.ShortRoot(t)
	explicit := filepath.Join(selectedRoot, "couch-candidate")
	for i, kind := range []string{"claude", "codex"} {
		if !t.Run(kind, func(t *testing.T) {
			var captured bytes.Buffer
			diagnostics := NewDiagnostics(&captured)
			options := Options{ScratchParent: scratch}
			if i == 1 {
				options.Couch = explicit
			}
			owner := NewNativeOwner(options, diagnostics)
			owner.dependencies.source = func() (string, error) { return source, nil }
			owner.dependencies.agentKind = kind
			if i == 0 {
				build := owner.dependencies.build
				owner.dependencies.build = func(ctx context.Context, cfg testcouch.BuildConfig) (*testcouch.CandidateOwner, error) {
					candidate, err := build(ctx, cfg)
					if err != nil {
						return candidate, err
					}
					if err = copyQualifiedCouch(candidate.Binaries.Couch, explicit); err != nil {
						return candidate, err
					}
					return candidate, nil
				}
			}
			runNativeBridgeCohort(t, ctx, owner, diagnostics, &captured, i == 1)
		}) {
			return
		}
	}
}

func copyQualifiedCouch(source, destination string) (result error) {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { result = errors.Join(result, in.Close()) }()
	out, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0700)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	return errors.Join(err, out.Close())
}

func runNativeBridgeCohort(t *testing.T, parent context.Context, owner *NativeOwner, diagnostics *Diagnostics, captured *bytes.Buffer, term bool) {
	t.Helper()
	ctx, cancel := context.WithCancel(parent)
	input, control, err := os.Pipe()
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	observed, output, err := os.Pipe()
	if err != nil {
		cancel()
		_ = input.Close()
		_ = control.Close()
		t.Fatal(err)
	}
	signals := make(chan os.Signal, 1)
	pipes, err := NewPipeIO(input, output, signals)
	if err != nil {
		cancel()
		_ = control.Close()
		_ = observed.Close()
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- RunBridge(ctx, owner, pipes) }()
	type readResult struct {
		line []byte
		err  error
	}
	readiness := make(chan readResult, 1)
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		line, err := bufio.NewReaderSize(observed, ReadinessLimit).ReadSlice('\n')
		readiness <- readResult{append([]byte(nil), line...), err}
	}()
	joined := false
	defer func() {
		_ = control.Close()
		cancel()
		if !joined {
			select {
			case err := <-done:
				if err != nil {
					t.Errorf("bridge stop: %v", err)
				}
			case <-time.After(StopBudget + 3*time.Second):
				t.Errorf("bridge did not join; retained root %s", owner.RetainedRoot())
			}
		}
		_ = observed.Close()
		<-readDone
		if err := diagnostics.Finish(owner.RetainedRoot()); err != nil {
			t.Errorf("diagnostic finalization: %v", err)
		}
		t.Logf("native bridge diagnostics:\n%s", captured.String())
	}()
	var line []byte
	select {
	case result := <-readiness:
		if result.err != nil {
			t.Fatalf("readiness: %v", result.err)
		}
		line = result.line
	case err := <-done:
		joined = true
		t.Fatalf("bridge exited before readiness: %v", err)
	case <-ctx.Done():
		t.Fatalf("native readiness observation: %v", ctx.Err())
	}
	if len(line) > ReadinessLimit || bytes.Count(line, []byte{'\n'}) != 1 {
		t.Fatal("readiness exceeded one bounded JSON line")
	}
	var ready Readiness
	if err := json.Unmarshal(line, &ready); err != nil {
		t.Fatal(err)
	}
	if err := ValidateReadiness(ready); err != nil {
		t.Fatal(err)
	}
	root := ready.Environment["COUCH_ISOLATED_ROOT"]
	env := make([]string, 0, len(ready.Environment))
	for key, value := range ready.Environment {
		env = append(env, key+"="+value)
	}
	couch := filepath.Join(root, "bin", "couch")
	for _, doc := range ready.Documents {
		call, stop := context.WithTimeout(ctx, 3*time.Second)
		raw, err := nativeCommand(call, couch, []string{"--editor-endpoint", "lookup", "--namespace", ready.Namespace, "--document-id", doc.ID, "--json"}, root, env)
		stop()
		if err != nil {
			t.Fatalf("public endpoint roundtrip: %v", err)
		}
		var reply struct {
			Version        int
			Code, Error    string
			EditorEndpoint json.RawMessage
		}
		if err := json.Unmarshal(raw, &reply); err != nil || reply.Version != 1 || reply.Code != "ok" || reply.Error != "" || string(reply.EditorEndpoint) != "null" {
			t.Fatalf("endpoint reply %s (%v)", raw, err)
		}
	}
	// The exact advertised environment replaces ambient HOME/COUCH/PATH. No
	// native editor survives this foreground invocation when EOF/TERM is sent.
	nvim := filepath.Join(root, "bin", "nvim")
	lua := `local a=vim.fn.argv(); assert(#a==2); local side=table.concat(vim.fn.readfile(a[2]),"\n"); assert(side:find("type: side-doc",1,true)); assert(side:find("parent: ../first.md",1,true)); io.stdout:write(vim.json.encode({Parent=vim.fn.fnamemodify(a[1],":p"),Side=vim.fn.fnamemodify(a[2],":p")}).."\n"); vim.cmd("qa!")`
	editor, stopEditor := context.WithTimeout(ctx, 15*time.Second)
	native, err := nativeCommand(editor, nvim, []string{"--headless", "--clean", "-u", "NONE", "-i", "NONE", "-n", ready.Documents[0].File, ready.Documents[1].File, "-c", "lua " + lua}, root, env)
	stopEditor()
	if err != nil {
		t.Fatalf("selected native nvim: %v", err)
	}
	var docs struct {
		Parent, Side string
	}
	if err := json.Unmarshal(bytes.TrimSpace(native), &docs); err != nil || docs.Parent != ready.Documents[0].File || docs.Side != ready.Documents[1].File {
		t.Fatalf("native document recognition %s (%v)", native, err)
	}
	if term {
		signals <- syscall.SIGTERM
	} else {
		if err := control.Close(); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case err := <-done:
		joined = true
		if err != nil {
			t.Fatalf("native joined stop: %v", err)
		}
	case <-ctx.Done():
		t.Fatalf("native stop observation: %v", ctx.Err())
	}
	tail, err := io.ReadAll(io.LimitReader(observed, ReadinessLimit+1))
	if err != nil || len(tail) != 0 {
		t.Fatalf("unexpected extra stdout %q (%v)", tail, err)
	}
	if err := owner.AssertCleaned(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(root); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("fleet root remains %s: %v", root, err)
	}
	if _, err := os.Stat(owner.candidate.Binaries.Dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("candidate root remains: %v", err)
	}
	t.Logf("native bridge: kind=%s explicit_candidate=%v namespace=%s socket=%s docs=%s,%s joined=true root_absent=true", owner.dependencies.agentKind, term, ready.Namespace, ready.Socket, ready.Documents[0].ID, ready.Documents[1].ID)
	if !strings.HasPrefix(nvim, root+string(filepath.Separator)) {
		t.Fatal("native editor escaped private PATH")
	}
}
