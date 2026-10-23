package testcouch

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestNativeFailureEvidencePreservesObservedAndUnknownExits(t *testing.T) {
	namespace, err := couchcore.ResolveCouchNamespace(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	code := 23
	store := couchcore.NewLaunchFailureStore(namespace)
	for i, before := range []*int{nil, &code} {
		f := couchcore.LaunchFailure{Target: "fixture:3", Nonce: []string{"unknown", "observed"}[i], FailedAt: time.Unix(int64(i+1), 0), Stage: "attach-preflight", BeforeExit: before, Stream: "pty stdout/stderr", Output: []byte("recent\xffbytes"), LaunchError: "endpoint=pty-4 input_ended=true termination_cause=unavailable"}
		if i == 1 {
			after := 17
			f.AfterExit = &after
		}
		if err := store.Save(t.Context(), f); err != nil {
			t.Fatal(err)
		}
	}
	out, err := nativeFailureEvidence(namespace.Dir())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"nonce=unknown", "nonce=observed", "before_exit=unobserved", "before_exit=23", "after_exit=unobserved", "after_exit=17", "stage=attach-preflight", `output="recent\xffbytes"`, "endpoint=pty-4"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
}

func TestNativeFailureEvidenceRetainsNativeQuery(t *testing.T) {
	namespace, err := couchcore.ResolveCouchNamespace(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	code := 0
	raw := []byte("[{\"id\":1,\"is_held\":true}]\n\x00\xff")
	store := couchcore.NewLaunchFailureStore(namespace)
	for i, f := range []couchcore.LifecycleFailure{
		{Nonce: "absent"},
		{Nonce: "empty", ZellijExecutable: "/private/zellij", ZellijVersion: "unavailable"},
		{Nonce: "observed", ZellijExecutable: "/private/zellij", ZellijArgs: []string{"--session", "exact name", "action", "list-panes", "--json", "--command"}, ZellijVersion: "zellij fixture", ZellijExit: &code, ZellijStdout: raw, ZellijStdoutBytes: 9000, ZellijStdoutTruncated: true, ZellijStderr: []byte("native\nstderr"), ZellijStderrBytes: 13},
	} {
		f.FailedAt = time.Unix(int64(i+1), 0)
		f.Target = "fixture:3"
		if err := store.Save(t.Context(), f); err != nil {
			t.Fatal(err)
		}
	}
	out, err := nativeFailureEvidence(namespace.Dir())
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"zellij=absent", `zellij_executable="/private/zellij"`, `zellij_args=["--session" "exact name" "action" "list-panes" "--json" "--command"]`,
		`zellij_version="zellij fixture"`, "zellij_exit=0", "zellij_exit=unobserved", `zellij_stdout=""`,
		fmt.Sprintf("zellij_stdout=%q", raw), "zellij_stdout_bytes=9000", "zellij_stdout_truncated=true", `zellij_stderr="native\nstderr"`, "zellij_stderr_bytes=13", "zellij_stderr_truncated=false",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in %s", want, out)
		}
	}
}
