package couchcore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/launcher"
)

func nativeQueryWireFixture(t *testing.T) map[string]any {
	t.Helper()
	raw, err := json.Marshal(failureFixture(1))
	if err != nil {
		t.Fatal(err)
	}
	var row map[string]any
	if err := json.Unmarshal(raw, &row); err != nil {
		t.Fatal(err)
	}
	row["zellij_stdout"] = bytes.Repeat([]byte{0, 255, 'x'}, 3000)
	row["zellij_stderr"] = bytes.Repeat([]byte("error"), 2000)
	row["zellij_stdout_bytes"] = 9000
	row["zellij_stderr_bytes"] = 10000
	row["zellij_exit"] = 0
	row["zellij_version"] = "zellij 0.45.1"
	row["future_native_context"] = map[string]any{"opaque": []any{1, "kept"}}
	return row
}

func TestNativeQueryFailureFieldsBoundAndPreserveRawPrefixes(t *testing.T) {
	row := nativeQueryWireFixture(t)
	raw, err := json.Marshal([]any{row})
	if err != nil {
		t.Fatal(err)
	}
	history, err := decodeLifecycleFailureHistory(raw)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := encodeLaunchFailureHistory(history)
	if err != nil {
		t.Fatal(err)
	}
	var rows []map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &rows); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"zellij_stdout", "zellij_stderr"} {
		var got []byte
		if err := json.Unmarshal(rows[0][key], &got); err != nil {
			t.Fatal(err)
		}
		want := row[key].([]byte)
		if len(got) != 4096 || !bytes.Equal(got, want[:4096]) {
			t.Fatalf("%s not a bounded exact raw prefix: got=%d want=4096", key, len(got))
		}
	}
	for _, key := range []string{"zellij_stdout_truncated", "zellij_stderr_truncated"} {
		if string(rows[0][key]) != "true" {
			t.Fatalf("%s lost truncation", key)
		}
	}
	if string(rows[0]["zellij_stdout_bytes"]) != "9000" || string(rows[0]["zellij_stderr_bytes"]) != "10000" || string(rows[0]["zellij_exit"]) != "0" || len(rows[0]["future_native_context"]) == 0 {
		t.Fatalf("metadata or unrelated unknown context lost: %s", encoded)
	}
}

func TestNativeQueryFailureKnownFieldsRemainStrict(t *testing.T) {
	for _, bad := range []struct {
		key   string
		value any
	}{
		{"zellij_stdout", 4}, {"zellij_stderr", false}, {"zellij_exit", "0"},
		{"zellij_version", 1}, {"zellij_stdout_bytes", -1}, {"zellij_stderr_bytes", -1},
		{"zellij_stdout_truncated", "yes"}, {"zellij_args", "--session"},
	} {
		t.Run(bad.key, func(t *testing.T) {
			row := nativeQueryWireFixture(t)
			row[bad.key] = bad.value
			raw, err := json.Marshal([]any{row})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := decodeLifecycleFailureHistory(raw); err == nil {
				t.Fatalf("accepted malformed known field %s", bad.key)
			}
		})
	}
}

type nativeEvidenceCause struct{ evidence launcher.NativeQueryEvidence }

func (e nativeEvidenceCause) Error() string { return "invalid live pane JSON" }
func (e nativeEvidenceCause) NativeQueryEvidence() launcher.NativeQueryEvidence {
	return e.evidence.Bounded()
}

type nativeEvidenceOwnerProbe struct {
	observation launcher.SessionOwnerObservation
}

func (p nativeEvidenceOwnerProbe) Probe(context.Context, string, string, string, string) (launcher.SessionOwnerObservation, error) {
	return p.observation, nil
}
func (nativeEvidenceOwnerProbe) Revalidate(context.Context, launcher.SessionOwnerObservation) error {
	return nil
}

func TestNativeQueryFailureFollowsOwnerRefusalIntoReopenedOperation(t *testing.T) {
	exit := 0
	evidence := launcher.NativeQueryEvidence{Args: []string{"--session", "exact", "action", "list-panes", "--json", "--command"}, Executable: "/private/bin/zellij", Stdout: []byte("\x1b[32;1mraw-secret\x00\xff"), Stderr: []byte("native-stderr"), ExitCode: &exit, Version: "zellij 0.45.1"}
	checker := NewScopedThreadArtifactCollisionChecker(t.TempDir())
	checker.OwnerProbe = nativeEvidenceOwnerProbe{launcher.SessionOwnerObservation{State: launcher.SessionOwnerUnknown, Diagnostic: "invalid live pane JSON", DiagnosticCause: nativeEvidenceCause{evidence}}}
	address := ThreadAddress{RepoScope: "repo", Tag: "1-project"}
	_, cause := checker.NamedPairSessionContext(t.Context(), address, "exact")
	if cause == nil || strings.Contains(cause.Error(), "raw-secret") || strings.Contains(cause.Error(), "native-stderr") {
		t.Fatalf("refusal missing or exposed raw evidence: %v", cause)
	}
	namespace := failureNamespace(t)
	store := NewLaunchFailureStore(namespace)
	r := NewLifecycleFailureRecorder(store, func() time.Time { return time.Unix(100, 0) }, func() (string, error) { return "native-operation", nil })
	a, err := r.Begin("leave", LifecycleFailureTarget{Address: address})
	if err != nil {
		t.Fatal(err)
	}
	_, result := a.Finish(nil, fmt.Errorf("outer operation: %w", cause))
	if result == nil || strings.Contains(OperatorError(result), "raw-secret") {
		t.Fatalf("operation error exposed raw evidence: %v", result)
	}
	history, err := NewLaunchFailureStore(namespace).Read()
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 {
		t.Fatalf("wrong operation rows: %+v", history)
	}
	got := history[0]
	if !bytes.Equal(got.ZellijStdout, evidence.Stdout) || !bytes.Equal(got.ZellijStderr, evidence.Stderr) || got.ZellijExit == nil || *got.ZellijExit != 0 || got.ZellijVersion != evidence.Version || got.Nonce != "native-operation" || len(got.Output) != 0 {
		t.Fatalf("native evidence did not survive exact operation: %+v", got)
	}
	var typed nativeEvidenceCause
	if !errors.As(result, &typed) {
		t.Fatal("typed cause lost after recording")
	}
}

func TestNativeQueryFailureCloneAndEnrichmentOwnTheirBytes(t *testing.T) {
	code := 0
	f := failureFixture(1)
	f.Output = []byte("original launch output")
	f.setNativeQueryEvidence(launcher.NativeQueryEvidence{Stdout: []byte("raw"), Stderr: []byte("err"), Args: []string{"original"}, ExitCode: &code, Version: "v1"})
	cloned := cloneLaunchFailure(f)
	cloned.ZellijStdout[0], cloned.ZellijStderr[0], cloned.ZellijArgs[0], *cloned.ZellijExit = 'X', 'X', "changed", 9
	if string(f.ZellijStdout) != "raw" || string(f.ZellijStderr) != "err" || f.ZellijArgs[0] != "original" || *f.ZellijExit != 0 {
		t.Fatal("clone aliases evidence")
	}
	h, err := (LaunchFailureHistory{}).Upsert(f)
	if err != nil {
		t.Fatal(err)
	}
	update := failureFixture(1)
	update.setNativeQueryEvidence(launcher.NativeQueryEvidence{Stdout: []byte("new raw"), Version: "v2"})
	h, err = h.Upsert(update)
	if err != nil {
		t.Fatal(err)
	}
	update.ZellijStdout[0] = 'X'
	if len(h) != 1 || string(h[0].ZellijStdout) != "new raw" || string(h[0].Output) != "original launch output" || h[0].ZellijVersion != "v2" {
		t.Fatalf("enrichment lost independent evidence: %+v", h)
	}
	other := failureFixture(2)
	other.setNativeQueryEvidence(launcher.NativeQueryEvidence{Stdout: []byte("other raw"), Version: "v3"})
	h, err = h.Upsert(other)
	if err != nil {
		t.Fatal(err)
	}
	if len(h) != 2 || h[0].Nonce == h[1].Nonce {
		t.Fatal("evidence rebound another operation")
	}
}

func TestNativeQueryFailureRetentionBoundAtNAnd2N(t *testing.T) {
	for _, count := range []int{64, 128} {
		var history LaunchFailureHistory
		for n := 1; n <= count; n++ {
			f := failureFixture(n)
			f.Output = bytes.Repeat([]byte("launch"), 4000)
			f.setNativeQueryEvidence(launcher.NativeQueryEvidence{Stdout: bytes.Repeat([]byte("out"), 4000), Stderr: bytes.Repeat([]byte("err"), 4000), Version: "zellij 0.45.1"})
			var err error
			history, err = history.Upsert(f)
			if err != nil {
				t.Fatal(err)
			}
		}
		raw, err := encodeLaunchFailureHistory(history)
		if err != nil {
			t.Fatal(err)
		}
		if len(raw) > launchFailureEncodedLimit || len(history) > launchFailureHistoryLimit || history[0].Nonce != fmt.Sprint(count) {
			t.Fatalf("unbounded/lost newest at%d: %drows %dbytes", count, len(history), len(raw))
		}
		for _, f := range history {
			if len(f.ZellijStdout) != 4096 || len(f.ZellijStderr) != 4096 || !f.ZellijStdoutTruncated || !f.ZellijStderrTruncated {
				t.Fatal("retention lost raw evidence bounds")
			}
		}
	}
}
