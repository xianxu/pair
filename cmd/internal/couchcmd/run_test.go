package couchcmd

import (
	"bytes"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
)

type testRT struct{ dir string }

func (t testRT) Getenv(string) string { return "" }
func (t testRT) StoreDir() string     { return t.dir }

func run(t *testing.T, dir string, args ...string) (string, string, int) {
	t.Helper()
	var out, errw bytes.Buffer
	code := RunWithRuntime(args, strings.NewReader(""), &out, &errw, testRT{dir: dir})
	return out.String(), errw.String(), code
}

// TestDispatchTableIsIdenticalToTheDeclaredOperationSet is the audit.
//
// It asserts IDENTITY, not overlap with a hand-written list: an operation
// reachable from the CLI but never declared would be invisible to the
// advisor, and a hand-written expected list would not catch it because the
// list is written by the same person who forgot to declare it.
func TestDispatchTableIsIdenticalToTheDeclaredOperationSet(t *testing.T) {
	var reachable []string
	for name := range Dispatch() {
		reachable = append(reachable, name)
	}
	sort.Strings(reachable)
	if declared := couchcore.OperationNames(); !reflect.DeepEqual(reachable, declared) {
		t.Fatalf("CLI reaches %v, declared %v", reachable, declared)
	}
}

func TestEveryOperationHasASummaryAndDescribedArgs(t *testing.T) {
	for _, op := range couchcore.Operations() {
		if op.Summary == "" {
			t.Errorf("%s: empty summary -- the advisor needs it to choose", op.Name)
		}
		for _, a := range op.Args {
			if a.Summary == "" {
				t.Errorf("%s: arg %q has no summary", op.Name, a.Name)
			}
		}
		if op.Invoke == nil {
			t.Errorf("%s: declared but not invocable", op.Name)
		}
	}
}

func TestOperationArityMatchesExpectation(t *testing.T) {
	// Declared in the test rather than read from the operation itself, so
	// this cannot degrade into asserting X == X.
	want := map[string]int{"start": 2, "list": 0, "show": 1, "stop": 1, "name": 2, "describe": 2}
	for _, op := range couchcore.Operations() {
		if got := len(op.Args); got != want[op.Name] {
			t.Errorf("%s has %d args, want %d", op.Name, got, want[op.Name])
		}
	}
}

func TestListOnEmptyRegistry(t *testing.T) {
	out, errw, code := run(t, t.TempDir(), "list")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errw)
	}
	if !strings.Contains(out, "no trees") {
		t.Fatalf("out = %q", out)
	}
}

func TestUnknownOperationIsNonZeroAndListsWhatExists(t *testing.T) {
	out, errw, code := run(t, t.TempDir(), "frobnicate")
	if code == 0 {
		t.Fatal("unknown operation must be non-zero")
	}
	if !strings.Contains(errw, "unknown operation") || !strings.Contains(errw, "start") {
		t.Fatalf("stderr = %q; the error should name what does exist", errw)
	}
	_ = out
}

func TestMissingRequiredArgumentIsRejectedBeforeAnyWork(t *testing.T) {
	_, errw, code := run(t, t.TempDir(), "show")
	if code == 0 {
		t.Fatal("a missing required argument must be non-zero")
	}
	if !strings.Contains(errw, "missing required argument") {
		t.Fatalf("stderr = %q", errw)
	}
}

func TestHelpListsEveryDeclaredOperation(t *testing.T) {
	out, _, code := run(t, t.TempDir(), "--help")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	for _, name := range couchcore.OperationNames() {
		if !strings.Contains(out, name) {
			t.Errorf("help omits %q", name)
		}
	}
}

func TestBindArgsAcceptsFlagsAndPositionals(t *testing.T) {
	var start couchcore.Operation
	for _, op := range couchcore.Operations() {
		if op.Name == "start" {
			start = op
		}
	}
	got, err := bindArgs(start, []string{"../pair", "--same-tree"})
	if err != nil {
		t.Fatalf("bindArgs: %v", err)
	}
	if got["path"] != "../pair" || got["same-tree"] != "true" {
		t.Fatalf("bound = %v", got)
	}
}

func TestListShowsANamedTreeWithNoAgent(t *testing.T) {
	// The forgetting case: a tree that was named and then parked has no actor,
	// but it is exactly the thread the operator loses track of. It must be a
	// visible row, not filtered out.
	dir := t.TempDir()
	if _, errw, code := run(t, dir, "name", "../..", "the pair tree"); code != 0 {
		t.Fatalf("name failed: %s", errw)
	}
	out, _, code := run(t, dir, "list")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out, "the pair tree") {
		t.Fatalf("out = %q; a named tree must appear even with no agent", out)
	}
	if !strings.Contains(out, "(no agent running)") {
		t.Fatalf("out = %q; the absence of an agent must be stated", out)
	}
}

func TestShowResolvesANameToItsTreePath(t *testing.T) {
	dir := t.TempDir()
	if _, errw, code := run(t, dir, "name", "../..", "pairtree"); code != 0 {
		t.Fatalf("name failed: %s", errw)
	}
	out, errw, code := run(t, dir, "show", "pairtree")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errw)
	}
	if !strings.Contains(out, "/pair") {
		t.Fatalf("out = %q; show must print the tree path", out)
	}
}

func TestRenderedOutputHasNoANSIWhenNotATerminal(t *testing.T) {
	// A bytes.Buffer is not a terminal, so dimming must be suppressed --
	// otherwise piped or captured output carries escape codes.
	dir := t.TempDir()
	_, _, _ = run(t, dir, "name", "../..", "plain")
	out, _, _ := run(t, dir, "list")
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("ANSI leaked into non-terminal output: %q", out)
	}
}
