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

func TestNativeFailureEvidenceBoundsNativeQueryOutput(t *testing.T) {
	namespace, err := couchcore.ResolveCouchNamespace(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	store := couchcore.NewLaunchFailureStore(namespace)
	for n := 0; n < 6; n++ {
		failure := couchcore.LifecycleFailure{Target: "fixture:3", Nonce: fmt.Sprintf("native-%d", n), FailedAt: time.Unix(int64(n+1), 0), ZellijExecutable: "/fixture/zellij", ZellijStdout: []byte(strings.Repeat("\x00", 5000)), ZellijStderr: []byte(strings.Repeat("\xff", 5000))}
		if err := store.Save(t.Context(), failure); err != nil {
			t.Fatal(err)
		}
	}
	out, err := nativeFailureEvidence(namespace.Dir())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(out, "nonce=") != 4 || !strings.Contains(out, "nonce=native-5") || strings.Contains(out, "nonce=native-1") {
		t.Fatalf("wrong selection: %s", out)
	}
	for _, stream := range []string{"stdout", "stderr"} {
		if strings.Count(out, "zellij_"+stream+"_bytes=5000") != 4 || strings.Count(out, "zellij_"+stream+"_truncated=true") != 4 {
			t.Fatalf("lost stream bounds: %s", out)
		}
	}
	if strings.Count(out, `\x00`) != 4*4096 || strings.Count(out, `\xff`) != 4*4096 {
		t.Fatal("did not retain exact production stream prefixes")
	}
	if len(out) > 4*(2*4096*4+1024) {
		t.Fatalf("escaped evidence exceeded bounded stream and metadata budget: %d", len(out))
	}
}
