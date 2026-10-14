package orientation

import (
	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"strings"
	"testing"
)

func TestBuildPromptNamesOnlyTheOutgoingSessionAndWaits(t *testing.T) {
	body, err := BuildPrompt(OrientationContext{
		Tag: "work", WorkingPath: "/repo", SourceAgent: "codex", SourceSession: "source-2", TargetAgent: "claude",
		PairLog: "/data/log-work.md", ScrollbackRaw: "/data/parked-source.raw", ScrollbackEvents: "/data/parked-source.events.jsonl",
		NativeTranscripts: []string{"/native/source-2.jsonl"}, Renderer: "/pair/bin/pair",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"source-2", "/data/parked-source.raw", "/native/source-2.jsonl", `"/pair/bin/pair","scrollback","render","--plain"`, "fresh conversation", "wait for the operator", "historical context"} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q in %s", want, body)
		}
	}
	if strings.Contains(body, "write a continuation") {
		t.Fatal("source must not prepare a continuation")
	}
}

func TestBuildPromptMissingContextStillOrients(t *testing.T) {
	body, err := BuildPrompt(OrientationContext{Tag: "work", WorkingPath: "/repo", SourceAgent: "muse", TargetAgent: "agy"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "unavailable") || !strings.Contains(body, "do not invent") {
		t.Fatal(body)
	}
}

func TestRendererReceivesExplicitSourceOwner(t *testing.T) {
	owner, err := artifactpath.NewStorageOwner("/data", "scope", "tag")
	if err != nil {
		t.Fatal(err)
	}
	body, err := BuildPrompt(OrientationContext{Tag: "tag", WorkingPath: "/repo", TargetAgent: "codex", Renderer: "/bin/pair", ScrollbackRaw: "/data/repos/scope/capture.raw", Owner: &owner})
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{`"--owner-dir","/data/repos/scope"`, `"--owner-scope","scope"`, `"--owner-tag","tag"`} {
		if !strings.Contains(body, value) {
			t.Fatalf("owner missing: %s", body)
		}
	}
}
