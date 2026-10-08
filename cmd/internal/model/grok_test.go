package model

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A fake grok that behaves like 1.0.46 headless: it ignores stdin, reads the
// prompt file, and persists a session dir plus prompt_history.jsonl under
// $HOME/.grok/sessions/<url-encoded resolved cwd>/ (measured live).
func fakeGrok(t *testing.T) (dir string) {
	t.Helper()
	dir = t.TempDir()
	script := strings.Join([]string{
		"#!/bin/sh",
		"printf '%s\\n' \"$@\" > '" + filepath.Join(dir, "args") + "'",
		"cwd=$(pwd -P)",
		"printf '%s' \"$cwd\" > '" + filepath.Join(dir, "cwd") + "'",
		"cat > '" + filepath.Join(dir, "stdin") + "'",
		"while [ $# -gt 0 ]; do if [ \"$1\" = --prompt-file ]; then cp \"$2\" '" + filepath.Join(dir, "prompt") + "'; fi; shift; done",
		"enc=$(printf '%s' \"$cwd\" | sed 's|/|%2F|g')",
		"mkdir -p \"$HOME/.grok/sessions/$enc/01a11a54-4fcc-74f1-9f75-1a214eeba2fe\"",
		"echo '{}' > \"$HOME/.grok/sessions/$enc/prompt_history.jsonl\"",
		"printf 'grok slug\\n'",
	}, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(dir, "grok"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return dir
}

func TestRunGrokDispatchesThroughAPromptFileAndLeavesNoResidue(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	fake := fakeGrok(t)
	// An unrelated grok session that must survive the cleanup.
	keep := filepath.Join(home, ".grok", "sessions", "%2Frepo", "keep")
	if err := os.MkdirAll(keep, 0o700); err != nil {
		t.Fatal(err)
	}

	got, err := Run(Request{Agent: "grok", Prompt: "prompt text", Input: "input text"})
	if err != nil || got != "grok slug\n" {
		t.Fatalf("Run = %q, %v", got, err)
	}
	read := func(name string) string {
		b, err := os.ReadFile(filepath.Join(fake, name))
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	args := strings.Split(strings.TrimSpace(read("args")), "\n")
	if len(args) != 7 || args[0] != "--prompt-file" || args[2] != "-m" || args[3] != DefaultGrokModel || args[4] != "--max-turns" || args[5] != "1" || args[6] != "--no-subagents" {
		t.Fatalf("grok argv = %q", args)
	}
	// grok -p ignores stdin (measured), so instructions and input both travel
	// in the prompt file.
	if prompt := read("prompt"); !strings.Contains(prompt, "prompt text") || !strings.Contains(prompt, "input text") {
		t.Fatalf("prompt file = %q", prompt)
	}
	cwd := read("cwd")
	if !strings.Contains(filepath.Base(cwd), "pair-model-grok-") {
		t.Fatalf("grok ran in %q, want a pair-owned temp dir", cwd)
	}
	if _, err := os.Stat(cwd); !os.IsNotExist(err) {
		t.Fatalf("temp dir survived: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".grok", "sessions", url.PathEscape(cwd))); !os.IsNotExist(err) {
		t.Fatalf("grok session residue survived: %v", err)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("cleanup touched another session: %v", err)
	}
}

// The cleanup only ever removes the entry for a directory pair itself created,
// and refuses anything that is not a plain directory directly under the
// sessions root.
func TestRemoveGrokSessionResidueIsConfined(t *testing.T) {
	home := t.TempDir()
	root := filepath.Join(home, ".grok", "sessions")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	victim := t.TempDir()
	if err := os.WriteFile(filepath.Join(victim, "precious"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	cwd := "/private/tmp/pair-model-grok-1"
	target := filepath.Join(root, url.PathEscape(cwd))

	if err := removeGrokSessionResidue(home, cwd); err != nil {
		t.Fatalf("absent residue: %v", err)
	}
	if err := os.Symlink(victim, target); err != nil {
		t.Fatal(err)
	}
	if err := removeGrokSessionResidue(home, cwd); err == nil {
		t.Fatal("symlinked residue was not refused")
	}
	if _, err := os.Stat(filepath.Join(victim, "precious")); err != nil {
		t.Fatalf("cleanup followed a symlink: %v", err)
	}
	os.Remove(target)
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := removeGrokSessionResidue(home, cwd); err == nil {
		t.Fatal("non-directory residue was not refused")
	}
	for _, bad := range []string{"relative/dir", "/"} {
		if err := removeGrokSessionResidue(home, bad); err == nil {
			t.Errorf("cwd %q was not refused", bad)
		}
	}
}

func TestRunGrokLiveConformance(t *testing.T) {
	if os.Getenv("PAIR_LIVE_GROK_MODEL") != "1" {
		t.Skip("set PAIR_LIVE_GROK_MODEL=1 to exercise the installed grok CLI")
	}
	out, err := Run(Request{
		Agent:  "grok",
		Prompt: "What is the secret word in the input? Answer with one word.",
		Input:  "The secret word is BANANA.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "BANANA") {
		t.Fatalf("grok live output = %q, want the word BANANA from the input", out)
	}
}
