package testcouch

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

func TestNativeFailureEvidenceRetainsBoundedProductionOutput(t *testing.T) {
	namespace, err := couchcore.ResolveCouchNamespace(t.TempDir(), "/")
	if err != nil {
		t.Fatal(err)
	}
	store := couchcore.NewLaunchFailureStore(namespace)
	for n := 0; n < 6; n++ {
		failure := couchcore.LaunchFailure{Address: couchcore.ThreadAddress{RepoScope: "repo", Tag: "1-fixture"}, Nonce: fmt.Sprintf("failure-%d", n), FailedAt: time.Unix(int64(n), 0), Agent: "claude", Stage: "registration", LaunchError: "Pair exited before registering", Output: []byte(fmt.Sprintf("native exit detail %d", n))}
		if err := store.Save(t.Context(), failure); err != nil {
			t.Fatal(err)
		}
	}
	evidence, err := nativeFailureEvidence(namespace.Dir())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(evidence, "nonce=") != 4 || !strings.Contains(evidence, "native exit detail 5") || strings.Contains(evidence, "native exit detail 0") {
		t.Fatalf("lost bounded newest production output: %q", evidence)
	}
}
