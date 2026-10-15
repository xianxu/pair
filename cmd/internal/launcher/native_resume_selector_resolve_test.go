package launcher

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

func TestNativeResumeResolutionPinsOnceAcrossNewerArrival(t *testing.T) {
	home := t.TempDir()
	cwd := filepath.Join(home, "work")
	projects := filepath.Join(home, ".claude", "projects", "-work")
	for _, p := range []string{cwd, projects} {
		if err := os.MkdirAll(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	write := func(id string, at time.Time) {
		t.Helper()
		path := filepath.Join(projects, id+".jsonl")
		raw := fmt.Sprintf("{\"type\":\"user\",\"sessionId\":%q,\"isSidechain\":false,\"cwd\":%q,\"timestamp\":\"2026-08-28T10:00:00Z\",\"message\":{\"role\":\"user\",\"content\":\"hello\"}}\n", id, cwd)
		if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, at, at); err != nil {
			t.Fatal(err)
		}
	}
	first := "11111111-1111-4111-8111-111111111111"
	write(first, time.Unix(100, 0))
	rt := sessioninventory.NewOSRuntime(home, filepath.Join(home, "pair"))
	selector, err := ParseNativeResumeSelector("claude", []string{"--continue", "--model", "m", "--", "literal --continue"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := ResolveNativeResume(t.Context(), rt, NativeResumeRequest{Agent: "claude", CWD: cwd, Selector: selector})
	if err != nil || result.Status != sessioninventory.NativeSelectorResolved || result.NativeID != first {
		t.Fatalf("result=%+v %v", result, err)
	}
	write("22222222-2222-4222-8222-222222222222", time.Unix(200, 0))
	pinned, err := ParseNativeResumeSelector("claude", result.Args)
	if err != nil || pinned.Kind != NativeResumeExplicit || pinned.NativeID != first {
		t.Fatalf("pinned=%+v %v", pinned, err)
	}
	checked, err := ResolveNativeResume(t.Context(), rt, NativeResumeRequest{Agent: "claude", CWD: cwd, Selector: pinned})
	if err != nil || checked.NativeID != first {
		t.Fatalf("explicit ID changed after arrival: %+v %v", checked, err)
	}
	if len(result.Args) < 3 || result.Args[len(result.Args)-1] != "literal --continue" {
		t.Fatalf("prompt changed: %v", result.Args)
	}
}
