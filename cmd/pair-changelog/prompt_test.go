package main

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
	if !strings.Contains(first, "full terminal transcript") {
		t.Fatalf("first-run prompt missing whole-transcript instruction:\n%s", first)
	}
	if !strings.Contains(incr, "Never\n   drop it") && !strings.Contains(incr, "Never drop it") {
		t.Fatalf("incremental prompt missing never-drop-last rule:\n%s", incr)
	}
	// shared guidance is present in both.
	if !strings.Contains(first, "CHANGE LOG") || !strings.Contains(incr, "CHANGE LOG") {
		t.Fatal("entry guidance missing from a prompt")
	}
}

func TestBuildInputSectionsIncremental(t *testing.T) {
	got := buildInput("## d\n\n- one\n", "- two\n", "new activity", false)
	for _, want := range []string{
		"ALREADY LOGGED", "- one", "CURRENT LAST ENTRY", "- two",
		"NEW TERMINAL ACTIVITY", "new activity",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("buildInput missing %q in:\n%s", want, got)
		}
	}
}

func TestBuildInputFirstRunIsRawSlice(t *testing.T) {
	if got := buildInput("", "", "the whole transcript", true); got != "the whole transcript" {
		t.Fatalf("first-run input = %q", got)
	}
}
