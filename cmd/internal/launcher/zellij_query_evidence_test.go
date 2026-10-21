package launcher

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/zellijtest"
)

func evidenceVersionShell(t *testing.T) string {
	t.Helper()
	// A version reply is deliberately rejected by the panes decoder in the
	// producer catalog; it must never become pane ownership evidence.
	return zellijtest.ShellCase(t, "launcher-evidence-version")
}

func TestNativeQueryEvidenceBoundedClones(t *testing.T) {
	exit := 7
	raw := NativeQueryEvidence{Args: []string{strings.Repeat("a", 2000), "ignored"}, Executable: strings.Repeat("p", 2000), Stdout: []byte(strings.Repeat("o", 5000)), Stderr: []byte(strings.Repeat("e", 5000)), ExitCode: &exit, Version: strings.Repeat("v", 300)}
	got := raw.Bounded()
	if len(got.Stdout) != 4096 || len(got.Stderr) != 4096 || got.StdoutBytes != 5000 || got.StderrBytes != 5000 || !got.StdoutTruncated || !got.StderrTruncated || len(got.Executable) > 1024 || len(strings.Join(got.Args, "")) > 1024 || len(got.Version) > 256 {
		t.Fatalf("unbounded evidence: %+v", got)
	}
	got.Stdout[0] = 'x'
	got.Args = append(got.Args, "x")
	*got.ExitCode = 1
	if raw.Stdout[0] != 'o' || raw.Args[0] == "x" || *raw.ExitCode != 7 {
		t.Fatal("evidence aliases caller")
	}
	if (NativeQueryEvidence{}).Bounded().ExitCode != nil {
		t.Fatal("invented unavailable exit")
	}
}

func FuzzNativeQueryEvidenceBounded(f *testing.F) {
	f.Add([]byte("\x00\xffraw"), "☃/zellij", "--session")
	f.Add([]byte(strings.Repeat("x", 5000)), strings.Repeat("☃", 500), strings.Repeat("a", 2000))
	f.Add([]byte{}, strings.Repeat("p", 1024), "")
	f.Fuzz(func(t *testing.T, raw []byte, executable, arg string) {
		code := 0
		input := NativeQueryEvidence{Executable: executable, Args: []string{arg, "tail"}, Stdout: raw, Stderr: raw, ExitCode: &code, Version: arg}
		got := input.Bounded()
		want := raw[:min(len(raw), 4096)]
		if string(got.Stdout) != string(want) || string(got.Stderr) != string(want) || got.StdoutBytes != len(raw) || got.StderrBytes != len(raw) || got.StdoutTruncated != (len(raw) > 4096) || got.StderrTruncated != (len(raw) > 4096) {
			t.Fatal("raw prefix/count invariant failed")
		}
		if len(got.Executable)+len(strings.Join(got.Args, "")) > 1024 || len(got.Version) > 256 || !utf8.ValidString(got.Executable) || !utf8.ValidString(got.Version) {
			t.Fatal("metadata exceeded budget or lost UTF8")
		}
		for _, value := range got.Args {
			if !utf8.ValidString(value) {
				t.Fatal("invalid UTF8 argument")
			}
		}
		if len(got.Stdout) > 0 {
			before := raw[0]
			got.Stdout[0] ^= 1
			if raw[0] != before {
				t.Fatal("raw alias")
			}
		}
		if len(got.Args) > 0 {
			got.Args[0] = "changed"
		}
		*got.ExitCode = 9
		if input.Args[0] != arg || code != 0 {
			t.Fatal("metadata alias")
		}
	})
}

func TestSessionOwnerFailureEvidenceChainsObserver(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zellij")
	script := "#!/bin/sh\ncase \"$1\" in --version) " + evidenceVersionShell(t) + ";; *) " + zellijtest.ShellCase(t, "launcher-evidence-chain-malformed") + ";; esac\n"
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	world := &queryObservedOwnerWorld{&ownerWorld{present: true, generation: SessionServerIdentity{PID: 11, Identity: "original", Session: "pair-fixture", Socket: "/fixture/socket"}}}
	var events []ZellijQueryObservation
	ctx := WithZellijQueryObserver(t.Context(), func(e ZellijQueryObservation) { events = append(events, e) })
	got, err := (SessionOwnerProbe{IO: world}).Probe(ctx, "pair-fixture", "/tmp/data", "0123456789abcdef", "1-repo-1")
	if err != nil || got.State != SessionOwnerUnknown || got.Diagnostic != "invalid live pane JSON" {
		t.Fatalf("classification changed: %+v %v", got, err)
	}
	var carrier interface{ NativeQueryEvidence() NativeQueryEvidence }
	if !errors.As(got.DiagnosticCause, &carrier) {
		t.Fatalf("missing typed failure: %+v", got)
	}
	e := carrier.NativeQueryEvidence()
	if string(e.Stdout) != "{bad" || string(e.Stderr) != "native stderr" || e.ExitCode == nil || *e.ExitCode != 0 || e.Version != "zellij 0.45.1" || e.Executable != path {
		t.Fatalf("incorrect evidence %+v", e)
	}
	if len(events) != 1 || events[0].Executable != path {
		t.Fatalf("observer chain lost: %+v", events)
	}
	base := errors.New("concise refusal")
	wrapped := got.WithDiagnosticCause(base)
	if wrapped.Error() != "concise refusal" || !errors.As(wrapped, &carrier) || !errors.Is(wrapped, base) {
		t.Fatalf("UI/chain changed: %v", wrapped)
	}
}

func TestSessionOwnerLargeValidReplyDoesNotCapturePrefix(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zellij")
	output := filepath.Join(dir, "panes")
	marker := filepath.Join(dir, "version-spawned")
	// Use the same semantic pane serializer as the registered producer seed.
	raw := zellijtest.Reply(t, "panes", []map[string]any{{"id": 1, "pane_command": "nvim /tmp/data/repos/0123456789abcdef/draft-1-repo-1.md", "extra": strings.Repeat("x", 70<<10)}}).Stdout
	if err := os.WriteFile(output, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\nif [ \"$1\" = --version ]; then printf spawned > '"+marker+"'; exit 99; fi\n/bin/cat '"+output+"'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	world := &queryObservedOwnerWorld{&ownerWorld{present: true, generation: SessionServerIdentity{PID: 11, Identity: "original", Session: "pair-fixture", Socket: "/fixture/socket"}}}
	got, err := (SessionOwnerProbe{IO: world}).Probe(context.Background(), "pair-fixture", "/tmp/data", "0123456789abcdef", "1-repo-1")
	if err != nil || got.State != SessionOwnerOwned || got.DiagnosticCause != nil {
		t.Fatalf("valid reply became failure: %+v %v", got, err)
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("valid proof queried version: %v", err)
	}
}

func TestNativeQueryEvidenceMetadataSharesOneUTF8Budget(t *testing.T) {
	e := NativeQueryEvidence{Executable: strings.Repeat("界", 300), Args: []string{strings.Repeat("界", 100)}, Version: strings.Repeat("界", 100)}.Bounded()
	if len(e.Executable)+len(strings.Join(e.Args, "")) > 1024 || !utf8.ValidString(e.Executable) || !utf8.ValidString(strings.Join(e.Args, "")) || !utf8.ValidString(e.Version) {
		t.Fatalf("invalid metadata budget executable=%d argv=%d version=%d", len(e.Executable), len(strings.Join(e.Args, "")), len(e.Version))
	}
	if (NativeQueryEvidence{}).Bounded().Args != nil {
		t.Fatal("invented absent argv")
	}
}

func TestQueryZellijVersionCancelledCannotSpawn(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zellij")
	marker := filepath.Join(dir, "spawned")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nprintf spawned > '"+marker+"'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if got := queryZellijVersion(ctx, path); !strings.HasPrefix(got, "unavailable:") {
		t.Fatalf("missing cancelled version result: %q", got)
	}
	if _, err := os.Stat(marker); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancelled version spawned: %v", err)
	}
}

func TestNativeQueryVersionFailureUsesOriginalDeadline(t *testing.T) {
	// A version reader must not receive a fresh independent deadline after proof.
	ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
	defer cancel()
	e := nativeQueryFailureEvidence(ctx, ZellijQueryObservation{Executable: "/bin/sh", Stdout: []byte("{bad"), ExitCode: 0}, "invalid live pane JSON")
	var carrier interface{ NativeQueryEvidence() NativeQueryEvidence }
	if !errors.As(e, &carrier) || !strings.Contains(carrier.NativeQueryEvidence().Version, "context deadline exceeded") {
		t.Fatalf("original deadline not retained: %v", e)
	}
	if e.Error() != "invalid live pane JSON" {
		t.Fatalf("raw/versiondiagnostics in UI: %v", e)
	}
}

func TestSessionOwnerExhaustedQueryPreservesCauseAndEvidence(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zellij")
	if err := os.WriteFile(path, []byte("#!/bin/sh\ncase \"$1\" in --version) "+evidenceVersionShell(t)+";; *) "+zellijtest.ShellCase(t, "launcher-evidence-empty")+";; esac\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	world := &queryObservedOwnerWorld{&ownerWorld{present: true, generation: SessionServerIdentity{PID: 11, Identity: "original", Session: "pair-fixture", Socket: "/fixture/socket"}}}
	_, err := (SessionOwnerProbe{IO: world}).Probe(t.Context(), "pair-fixture", "/tmp/data", "0123456789abcdef", "1-repo-1")
	var carrier interface{ NativeQueryEvidence() NativeQueryEvidence }
	if !errors.Is(err, ErrZellijNoReply) || !errors.As(err, &carrier) {
		t.Fatalf("lost exhausted reply cause/evidence: %v", err)
	}
	e := carrier.NativeQueryEvidence()
	if e.ExitCode == nil || *e.ExitCode != 0 || len(e.Stdout) != 0 || e.Version != "zellij 0.45.1" {
		t.Fatalf("wrong final attempt: %+v", e)
	}
}

type cancelledEvidenceOwnerWorld struct {
	*ownerWorld
	cancel context.CancelFunc
	reply  []byte
}

func (w *cancelledEvidenceOwnerWorld) SessionPanes(ctx context.Context, name string) ([]byte, error) {
	return (ZellijSource{queryRun: func(cmd *exec.Cmd) error {
		_, _ = cmd.Stdout.Write(w.reply)
		w.cancel()
		return context.Canceled
	}}).runContext(ctx, "--session", name, "action", "list-panes", "--json", "--command")
}
func TestSessionOwnerCancelledQueryRetainsEvidence(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	world := &cancelledEvidenceOwnerWorld{ownerWorld: &ownerWorld{present: true, generation: SessionServerIdentity{PID: 11, Identity: "original", Session: "pair-fixture", Socket: "/fixture/socket"}}, cancel: cancel, reply: []byte(zellijtest.Case(t, "launcher-evidence-cancelled").Stdout)}
	_, err := (SessionOwnerProbe{IO: world}).Probe(ctx, "pair-fixture", "/tmp/data", "0123456789abcdef", "1-repo-1")
	var carrier interface{ NativeQueryEvidence() NativeQueryEvidence }
	if err == nil || err.Error() != context.Canceled.Error() || !errors.Is(err, context.Canceled) || !errors.As(err, &carrier) {
		t.Fatalf("cancelled evidence lost: %v", err)
	}
	e := carrier.NativeQueryEvidence()
	if string(e.Stdout) != "{unfinished" || e.ExitCode != nil || !strings.Contains(e.Version, "context canceled") {
		t.Fatalf("cancelled evidence invented exit/lost output: %+v", e)
	}
}

func TestSessionOwnerPassCancelledQueryRetainsEvidence(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	world := &cancelledEvidenceOwnerWorld{ownerWorld: &ownerWorld{present: true, generation: SessionServerIdentity{PID: 11, Identity: "original", Session: "pair-fixture", Socket: "/fixture/socket"}}, cancel: cancel, reply: []byte(zellijtest.Case(t, "launcher-evidence-cancelled").Stdout)}
	pass := NewSessionOwnerPass(t.Context(), ownerPassSourceFunc(func(context.Context) (map[string]ServerState, error) {
		return map[string]ServerState{"pair-fixture": {Server: world.generation, Verdict: ServerReachable}}, nil
	}), SessionOwnerProbe{IO: world})
	_, err := pass.Probe(ctx, "pair-fixture", "/tmp/data", "0123456789abcdef", "1-repo-1")
	var carrier interface{ NativeQueryEvidence() NativeQueryEvidence }
	if err == nil || err.Error() != context.Canceled.Error() || !errors.Is(err, context.Canceled) || !errors.As(err, &carrier) {
		t.Fatalf("owner pass discarded cancelled evidence: %v", err)
	}
}

type cancelledCreateOwnerRuntime struct {
	*couchSessionRuntime
	cancel context.CancelFunc
}

func (r *cancelledCreateOwnerRuntime) ObserveSessionOwner(context.Context, string, string, string, string) (SessionOwnerObservation, error) {
	r.cancel()
	return SessionOwnerObservation{State: SessionOwnerUnknown, Diagnostic: "invalid live pane JSON", DiagnosticCause: &nativeQueryFailure{diagnostic: "invalid live pane JSON", evidence: NativeQueryEvidence{Stdout: []byte("{bad"), Version: "unavailable"}}}, nil
}
func TestCouchSessionCreateCancellationRetainsOwnerEvidence(t *testing.T) {
	opts := managedSessionOptions("create")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	rt := &cancelledCreateOwnerRuntime{couchSessionRuntime: &couchSessionRuntime{fakeRuntime: newFakeRuntime()}, cancel: cancel}
	rt.sessions = []Session{{Name: legacySessionPrefix + opts.Args.CouchSession.Tag, State: SessionAttached}}
	err := verifyCouchSessionCreate(ctx, rt, opts.GlobalDataDir, *opts.Args.CouchSession)
	var carrier interface{ NativeQueryEvidence() NativeQueryEvidence }
	if err == nil || err.Error() != context.Canceled.Error() || !errors.Is(err, context.Canceled) || !errors.As(err, &carrier) {
		t.Fatalf("create cancellation discarded evidence: %v", err)
	}
}

func TestSessionOwnerEvidenceOwnsObserverBytes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "zellij")
	if err := os.WriteFile(path, []byte("#!/bin/sh\ncase \"$1\" in --version) "+evidenceVersionShell(t)+";; *) "+zellijtest.ShellCase(t, "launcher-evidence-observer-malformed")+";; esac\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	world := &queryObservedOwnerWorld{&ownerWorld{present: true, generation: SessionServerIdentity{PID: 11, Identity: "original", Session: "pair-fixture", Socket: "/fixture/socket"}}}
	ctx := WithZellijQueryObserver(t.Context(), func(e ZellijQueryObservation) { e.Stdout[0] = 'X'; e.Stderr[0] = 'X'; e.Args[0] = "corrupted" })
	got, err := (SessionOwnerProbe{IO: world}).Probe(ctx, "pair-fixture", "/tmp/data", "0123456789abcdef", "1-repo-1")
	var carrier interface{ NativeQueryEvidence() NativeQueryEvidence }
	if err != nil || !errors.As(got.DiagnosticCause, &carrier) {
		t.Fatalf("missing evidence: %+v %v", got, err)
	}
	e := carrier.NativeQueryEvidence()
	if string(e.Stdout) != "{bad" || string(e.Stderr) != "stderr" || e.Args[0] != "--session" {
		t.Fatalf("observer corrupted capture: %+v", e)
	}
}

func TestSessionOwnerCustomIORetainsRawWithoutInventingExit(t *testing.T) {
	world := &ownerWorld{present: true, generation: SessionServerIdentity{PID: 11, Identity: "original", Session: "pair-fixture", Socket: "/fixture/socket"}, panes: []byte(zellijtest.Case(t, "launcher-evidence-custom-raw").Stdout)}
	got, err := (SessionOwnerProbe{IO: world}).Probe(t.Context(), "pair-fixture", "/tmp/data", "0123456789abcdef", "1-repo-1")
	var carrier interface{ NativeQueryEvidence() NativeQueryEvidence }
	if err != nil || got.State != SessionOwnerUnknown || !errors.As(got.DiagnosticCause, &carrier) {
		t.Fatalf("custom IO lost raw: %+v %v", got, err)
	}
	e := carrier.NativeQueryEvidence()
	if string(e.Stdout) != "{custom raw" || e.ExitCode != nil || e.Version != "unavailable" || e.Executable != "" || len(e.Args) != 0 {
		t.Fatalf("custom IO invented native observation: %+v", e)
	}
}

type failedGenerationEvidenceWorld struct {
	*ownerWorld
	reads   int
	failure error
}

func (w *failedGenerationEvidenceWorld) SessionServers(ctx context.Context, name string) ([]SessionServerIdentity, error) {
	w.reads++
	if w.reads > 1 {
		return nil, w.failure
	}
	return w.ownerWorld.SessionServers(ctx, name)
}
func TestSessionOwnerGenerationFenceRetainsInvalidReply(t *testing.T) {
	for _, changed := range []bool{false, true} {
		t.Run(fmt.Sprint(changed), func(t *testing.T) {
			base := &ownerWorld{present: true, generation: SessionServerIdentity{PID: 11, Identity: "original", Session: "pair-fixture", Socket: "/fixture/socket"}, panes: []byte(zellijtest.Case(t, "launcher-evidence-generation-malformed").Stdout), changeOnQuery: changed}
			var world SessionOwnerIO = base
			generationErr := errors.New("generation read unavailable")
			if !changed {
				world = &failedGenerationEvidenceWorld{ownerWorld: base, failure: generationErr}
			}
			got, err := (SessionOwnerProbe{IO: world}).Probe(t.Context(), "pair-fixture", "/tmp/data", "0123456789abcdef", "1-repo-1")
			var carrier interface{ NativeQueryEvidence() NativeQueryEvidence }
			if changed {
				if err != nil || got.Diagnostic != "session server changed during observation" || !errors.As(got.DiagnosticCause, &carrier) {
					t.Fatalf("changed fence lost capture: %+v %v", got, err)
				}
			} else if !errors.Is(err, generationErr) || !errors.As(err, &carrier) {
				t.Fatalf("generation error lost capture: %+v %v", got, err)
			}
			if got.State != SessionOwnerUnknown || string(carrier.NativeQueryEvidence().Stdout) != "{invalid" {
				t.Fatalf("fence granted authority/lost raw: %+v", got)
			}
		})
	}
}
