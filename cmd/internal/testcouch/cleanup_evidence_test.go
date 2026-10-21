package testcouch

import (
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
