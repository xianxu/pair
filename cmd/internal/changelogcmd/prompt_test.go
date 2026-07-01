package changelogcmd

import (
	"strings"
	"testing"
)

func TestBuildSystemPromptDiffersByMode(t *testing.T) {
	first := buildSystemPrompt(true)
	incr := buildSystemPrompt(false)
	if first == incr {
		t.Fatal("first-run and incremental prompts must differ")
	}
	if !strings.Contains(first, "FULL terminal transcript") {
		t.Fatalf("first-run prompt missing whole-transcript instruction:\n%s", first)
	}
	if !strings.Contains(incr, "never drop it") {
		t.Fatalf("incremental prompt missing never-drop-last rule:\n%s", incr)
	}
	// The anti-continuation framing + change-log role are in both.
	for _, p := range []string{first, incr} {
		if !strings.Contains(p, "CHANGE LOG") || !strings.Contains(p, "DATA TO ANALYZE") {
			t.Fatalf("prompt missing distiller role / anti-continuation framing:\n%s", p)
		}
	}
}

func TestBuildInputSectionsIncremental(t *testing.T) {
	got := buildInput("## d\n\n- one\n", "- two\n", "new activity", false)
	for _, want := range []string{
		"ALREADY LOGGED", "- one", "CURRENT LAST ENTRY", "- two",
		"BEGIN TERMINAL TRANSCRIPT", "new activity", "END TERMINAL TRANSCRIPT",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("buildInput missing %q in:\n%s", want, got)
		}
	}
}

func TestBuildInputFirstRunWrapsTranscript(t *testing.T) {
	got := buildInput("", "", "the whole transcript", true)
	for _, want := range []string{"BEGIN TERMINAL TRANSCRIPT", "the whole transcript", "END TERMINAL TRANSCRIPT"} {
		if !strings.Contains(got, want) {
			t.Fatalf("first-run input missing %q in:\n%s", want, got)
		}
	}
	if got == "the whole transcript" {
		t.Fatal("first-run input should wrap the transcript in data delimiters, not pass it raw")
	}
}
