package pairlifecycletest

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// StubZellij writes a `zellij` stand-in that answers the session queries from
// sessions (name -> "attached" | "detached" | "exited") and logs EVERY
// invocation, one line of arguments each. It returns the stub's path and its
// log.
//
// It exists so a count of zellij calls can be asserted at the seam rather than a
// timing (pair#228, workbench-latency's Tier 1). Launcher and couchcore tests
// share it; neither may import the other's test helpers.
//
// Anything it does not recognise is logged and fails. A test that puts the
// stub's directory first on PATH therefore also catches zellij calls from a
// route it did not name -- a `delete-session` reaching the real host is the
// case that made pair#228's plan gate insist on it.
func StubZellij(t testing.TB, sessions map[string]string) (path, log string) {
	t.Helper()
	dir := t.TempDir()
	log = filepath.Join(dir, "zellij.log")
	path = filepath.Join(dir, "zellij")

	names := make([]string, 0, len(sessions))
	for name := range sessions {
		names = append(names, name)
	}
	sort.Strings(names)

	var short, full, clients strings.Builder
	for _, name := range names {
		state := sessions[name]
		fmt.Fprintf(&short, "%s\n", name)
		suffix := ""
		if state == "exited" {
			suffix = " (EXITED - attach to resurrect)"
		}
		fmt.Fprintf(&full, "%s [Created 1s ago]%s\n", name, suffix)
		switch state {
		case "attached":
			fmt.Fprintf(&clients, "  %s) printf 'CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\\n1 terminal_0 sh\\n' ;;\n", shellQuote("--session "+name+" action list-clients"))
		case "detached":
			fmt.Fprintf(&clients, "  %s) printf 'CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\\n' ;;\n", shellQuote("--session "+name+" action list-clients"))
		case "exited":
		default:
			t.Fatalf("StubZellij: session %q has state %q; want attached, detached or exited", name, state)
		}
	}

	script := "#!/usr/bin/env bash\n" +
		"printf '%s\\n' \"$*\" >> " + shellQuote(log) + "\n" +
		"case \"$*\" in\n" +
		"  'list-sessions --short') printf %s " + shellQuote(short.String()) + " ;;\n" +
		"  'list-sessions --no-formatting') printf %s " + shellQuote(full.String()) + " ;;\n" +
		clients.String() +
		"  *) exit 1 ;;\n" +
		"esac\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path, log
}

// CountCalls reports how many logged zellij invocations contain needle.
func CountCalls(t testing.TB, log, needle string) int {
	t.Helper()
	n := 0
	for _, line := range LoggedCalls(t, log) {
		if strings.Contains(line, needle) {
			n++
		}
	}
	return n
}

// LoggedCalls is every invocation the stub has seen, in order. A missing log
// means no call was made.
func LoggedCalls(t testing.TB, log string) []string {
	t.Helper()
	raw, err := os.ReadFile(log)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
