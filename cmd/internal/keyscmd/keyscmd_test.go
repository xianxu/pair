package keyscmd

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchkeys"
	"github.com/xianxu/pair/cmd/internal/keyhelp"
	"github.com/xianxu/pair/cmd/internal/launcher"
)

func TestRunPrintsRealBindings(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := RunWith(nil, deps(nil, false), &stdout, &stderr); code != 0 {
		t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"Alt+⏎", "send buffer + clear", "Alt+h", "Alt+x"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
	// The bug in its exact form: the help must not point back at the help key.
	if strings.Contains(out, "keybindings are on Alt+h") {
		t.Error("help refers the reader back to Alt+h (#132)")
	}
	// Nor should it be the CLI synopsis.
	if strings.Contains(out, "pair resume <tag>") {
		t.Error("help is showing CLI usage instead of keybindings")
	}
}

func TestRunCentersWhenAsked(t *testing.T) {
	var plain, centered, stderr bytes.Buffer
	RunWith(nil, deps(nil, false), &plain, &stderr)
	RunWith([]string{"--center", "120"}, deps(nil, false), &centered, &stderr)
	if plain.String() == centered.String() {
		t.Fatal("--center had no effect")
	}
	if !strings.HasPrefix(centered.String(), " ") {
		t.Error("centered output should be indented")
	}
}

func TestRunIgnoresGarbageCenterValue(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := RunWith([]string{"--center", "not-a-number"}, deps(nil, false), &stdout, &stderr); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if stdout.Len() == 0 {
		t.Error("a bad --center must not suppress the help")
	}
}

type failingSources struct{}

func (failingSources) Read(string) ([]byte, error) { return nil, errors.New("bundle unreadable") }

// The pane must still open. bin/pair-help runs under `set -euo pipefail`: a non-zero
// exit here kills the floating pane before less opens, turning #132's useless help
// key into a dead one. So a source failure prints a diagnostic BODY and exits 0.
func TestRunExitsZeroAndExplainsWhenSourcesFail(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := RunWith(nil, Deps{Sources: failingSources{}, Getenv: env(nil), CouchPresents: presents(false, nil)}, &stdout, &stderr); code != 0 {
		t.Fatalf("exit = %d, want 0 so the pager still opens", code)
	}
	if !strings.Contains(stdout.String(), "keybind help unavailable") {
		t.Errorf("stdout = %q, want a visible diagnostic as the body", stdout.String())
	}
	if !strings.Contains(stderr.String(), "bundle unreadable") {
		t.Errorf("stderr should carry the cause, got %q", stderr.String())
	}
}

func TestRunRejectsUnknownArguments(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := RunWith([]string{"--bogus"}, deps(nil, false), &stdout, &stderr); code != 2 {
		t.Errorf("exit = %d, want 2 — a typo must not read as success", code)
	}
	if !strings.Contains(stderr.String(), "unknown argument") {
		t.Errorf("stderr = %q", stderr.String())
	}
}

func TestRunAcceptsEqualsFormCenter(t *testing.T) {
	var plain, eq, stderr bytes.Buffer
	RunWith(nil, deps(nil, false), &plain, &stderr)
	if code := RunWith([]string{"--center=120"}, deps(nil, false), &eq, &stderr); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if plain.String() == eq.String() {
		t.Error("--center=120 had no effect")
	}
}

func env(vars map[string]string) func(string) string {
	return func(k string) string { return vars[k] }
}

func presents(couch bool, err error) func() (bool, error) {
	return func() (bool, error) { return couch, err }
}

// deps is hermetic: this repo is routinely tested inside a Couch thread, whose
// env and outer-tty record must not leak into the page under test.
func deps(vars map[string]string, couch bool) Deps {
	return Deps{Sources: keyhelp.DefaultSources(), Getenv: env(vars), CouchPresents: presents(couch, nil)}
}

var couchLaunched = map[string]string{"COUCH_THREAD_SCOPE": "0123456789abcdef", "COUCH_THREAD_TAG": "couch-0123456789abcdef"}

func run(t *testing.T, d Deps) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	if code := RunWith(nil, d, &stdout, &stderr); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	return stdout.String()
}

// Standalone Pair's Alt+h is unchanged, byte for byte (#282).
func TestStandalonePageIsPairsSections(t *testing.T) {
	secs, err := keyhelp.Sections(keyhelp.DefaultSources())
	if err != nil {
		t.Fatal(err)
	}
	if got := run(t, deps(nil, false)); got != keyhelp.Render(secs) {
		t.Fatal("standalone page differs from Render(Sections)")
	}
}

// Presented by Couch: Couch's layer leads, from the table couch --help uses,
// and Pair's rows follow.
func TestCouchPresentedPageLeadsWithCouchsKeys(t *testing.T) {
	out := run(t, deps(couchLaunched, true))
	if !strings.HasPrefix(out, "Couch") {
		t.Fatalf("page does not lead with Couch:\n%s", out)
	}
	for _, b := range couchkeys.Bindings() {
		if !strings.Contains(out, b.Help) {
			t.Errorf("missing Couch's %s", b.Key)
		}
	}
	if !strings.Contains(out, "send buffer + clear") {
		t.Error("Pair's layer is missing")
	}
}

// Couch's section follows who presents the client (the attach record). Hosted
// wording follows whether Pair's Alt+n can reload: not if the session env names
// Couch (pair restart refuses) or Couch presents the client (it refuses the
// restart marker) (#282).
func TestPresenterAndHostingAreIndependent(t *testing.T) {
	for _, tc := range []struct {
		name          string
		vars          map[string]string
		couch         bool
		wantCouch     bool
		wantHostedRow bool
	}{
		{"adopted: Couch presents, env not Couch's", nil, true, true, true},
		{"Couch gone, reattached from a terminal", couchLaunched, false, false, true},
		{"Couch-launched and presented", couchLaunched, true, true, true},
		{"standalone", nil, false, false, false},
	} {
		out := run(t, deps(tc.vars, tc.couch))
		if got := strings.Contains(out, "open the Couch switcher"); got != tc.wantCouch {
			t.Errorf("%s: couch section=%v", tc.name, got)
		}
		if got := strings.Contains(out, "does not reload under Couch"); got != tc.wantHostedRow {
			t.Errorf("%s: hosted wording=%v", tc.name, got)
		}
	}
}

// A record the reader cannot trust never conjures Couch's section; Pair's page
// still renders, and the cause goes to stderr.
func TestUnreadablePresenterRendersPairsPage(t *testing.T) {
	d := deps(nil, false)
	d.CouchPresents = presents(true, errors.New("outer-tty record: unexpected shape"))
	var stdout, stderr bytes.Buffer
	if code := RunWith(nil, d, &stdout, &stderr); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if strings.Contains(stdout.String(), "open the Couch switcher") || !strings.Contains(stdout.String(), "send buffer + clear") {
		t.Fatalf("page:\n%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "unexpected shape") {
		t.Errorf("stderr = %q", stderr.String())
	}
}

// Argument errors return before reading the record.
func TestBadArgumentsNeverReadThePresenter(t *testing.T) {
	d := deps(nil, false)
	d.CouchPresents = func() (bool, error) { t.Fatal("read the record on a usage error"); return false, nil }
	var stdout, stderr bytes.Buffer
	RunWith([]string{"--bogus"}, d, &stdout, &stderr)
	RunWith([]string{"--help"}, d, &stdout, &stderr)
}

// The production wiring: Run reads the record the launcher writes, from the
// session's PAIR_DATA_DIR and PAIR_TAG (#282).
func TestRunReadsTheAttachRecord(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("COUCH_THREAD_SCOPE", "")
	t.Setenv("COUCH_THREAD_TAG", "")
	t.Setenv("PAIR_DATA_DIR", dir)
	t.Setenv("PAIR_TAG", "work")
	paths, err := artifactpath.ResolveScoped(dir, "work")
	if err != nil {
		t.Fatal(err)
	}
	record := launcher.EncodeOuterRecord(launcher.OuterRecord{TTY: "/dev/ttys006", Couch: true})
	if err := os.WriteFile(paths.OuterTTY(), []byte(record), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := Run(nil, &stdout, &stderr); code != 0 || stderr.Len() != 0 {
		t.Fatalf("exit %d stderr %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "open the Couch switcher") {
		t.Fatalf("Run ignored the record:\n%s", stdout.String())
	}
}

// `pair --help` still advertises `pair keys`; outside a session it prints
// Pair's page and nothing on stderr.
func TestRunOutsideASessionIsQuiet(t *testing.T) {
	for _, k := range []string{"COUCH_THREAD_SCOPE", "COUCH_THREAD_TAG", "PAIR_DATA_DIR", "PAIR_TAG"} {
		t.Setenv(k, "")
	}
	var stdout, stderr bytes.Buffer
	if code := Run(nil, &stdout, &stderr); code != 0 || stderr.Len() != 0 {
		t.Fatalf("exit %d stderr %q", code, stderr.String())
	}
	secs, err := keyhelp.Sections(keyhelp.DefaultSources())
	if err != nil {
		t.Fatal(err)
	}
	if stdout.String() != keyhelp.Render(secs) {
		t.Fatal("outside a session, pair keys is not Pair's standalone page")
	}
}
