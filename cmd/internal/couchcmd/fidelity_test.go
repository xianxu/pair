package couchcmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/webcast/fidelity"
	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestParseFidelityCLI(t *testing.T) {
	inv, err := ParseCLI([]string{fidelityFlag, "rec"}, couchcore.Operations())
	if err != nil || inv.kind != cliWebFidelity || len(inv.args) != 1 || inv.args[0] != "rec" {
		t.Fatalf("%+v %v", inv, err)
	}
	if inv, err := ParseCLI([]string{fidelityFlag}, couchcore.Operations()); err != nil || inv.kind != cliWebFidelity {
		t.Fatalf("bare: %+v %v", inv, err)
	}
	for _, bad := range [][]string{{fidelityFlag, "--at", "1"}, {fidelityFlag, "a", "b"}} {
		if _, err := ParseCLI(bad, couchcore.Operations()); err == nil {
			t.Errorf("%v parsed", bad)
		}
	}
}

// store makes a Couch store holding the named recordings (copies of the
// segments fixture), oldest first.
func fidelityStore(t *testing.T, ids ...string) string {
	t.Helper()
	store := t.TempDir()
	for i, id := range ids {
		dir := filepath.Join(recordingsDir(store), id)
		if err := os.CopyFS(dir, os.DirFS(recording.Fixture("segments"))); err != nil {
			t.Fatal(err)
		}
		// The listing orders by the head's start.
		events := filepath.Join(dir, recording.FileEvents)
		raw, _ := os.ReadFile(events)
		head, rest, _ := bytes.Cut(raw, []byte("\n"))
		head = []byte(strings.Replace(string(head), `"started":"`, `"started":"`+string(rune('0'+i)), 1))
		os.WriteFile(events, append(append(head, '\n'), rest...), 0o600)
	}
	return store
}

func TestRunFidelity(t *testing.T) {
	store := fidelityStore(t, "older", "newer")
	chrome := func() (string, error) { return "/chrome", nil }
	var checked []string
	report := func(over int, shots int) fidelityCheck {
		return func(_ context.Context, bin, dir string) (fidelity.Report, error) {
			if bin != "/chrome" {
				t.Errorf("chrome %s", bin)
			}
			checked = append(checked, dir)
			var s []fidelity.Shot
			for i := 0; i < shots; i++ {
				d := 0.5
				if i < over {
					d = 9
				}
				s = append(s, fidelity.Shot{File: "frames/0.png", Diff: d})
			}
			return fidelity.Summarize(filepath.Base(dir), s, 0), nil
		}
	}
	for _, c := range []struct {
		name  string
		args  []string
		check fidelityCheck
		code  int
		dir   string
		out   string
	}{
		{"newest by default", nil, report(0, 3), fidelityWithin, "newer", "0 of 3 frames over 3%"},
		{"by id", []string{"older"}, report(1, 3), fidelityOver, "older", "1 of 3 frames over 3%"},
		{"nothing compared", []string{"older"}, report(0, 0), fidelityError, "older", "no frame to compare"},
	} {
		t.Run(c.name, func(t *testing.T) {
			checked = nil
			var out bytes.Buffer
			code := runFidelity(context.Background(), c.args, store, chrome, c.check, &out, io.Discard)
			if code != c.code || len(checked) != 1 || filepath.Base(checked[0]) != c.dir {
				t.Fatalf("exit %d, checked %v", code, checked)
			}
			if !strings.Contains(out.String(), c.out) || !strings.Contains(out.String(), filepath.Join(c.dir, "fidelity", "report.md")) {
				t.Errorf("printed %q", out.String())
			}
		})
	}
	var errs bytes.Buffer
	if code := runFidelity(context.Background(), []string{"nope"}, store, chrome, report(0, 1), io.Discard, &errs); code != fidelityError || !strings.Contains(errs.String(), "no recording") {
		t.Errorf("unknown id: %d %s", code, errs.String())
	}
	if code := runFidelity(context.Background(), nil, t.TempDir(), chrome, report(0, 1), io.Discard, io.Discard); code != fidelityError {
		t.Errorf("empty store: %d", code)
	}
	if code := runFidelity(context.Background(), []string{"-x"}, store, chrome, report(0, 1), io.Discard, io.Discard); code != fidelityUsage {
		t.Errorf("bad flag: %d", code)
	}
	noChrome := func() (string, error) { return "", errors.New("no Chrome") }
	if code := runFidelity(context.Background(), nil, store, noChrome, report(0, 1), io.Discard, io.Discard); code != fidelityError {
		t.Errorf("no Chrome: %d", code)
	}
	failing := func(context.Context, string, string) (fidelity.Report, error) { return fidelity.Report{}, errors.New("boom") }
	errs.Reset()
	if code := runFidelity(context.Background(), nil, store, chrome, failing, io.Discard, &errs); code != fidelityError || !strings.Contains(errs.String(), "boom") {
		t.Errorf("failing check: %d %s", code, errs.String())
	}
}

// The end-of-cast hook runs one check at a time and tells the operator
// the score and where the report is.
func TestFidelityHookOneAtATime(t *testing.T) {
	release := make(chan struct{})
	running := make(chan string, 2)
	check := func(_ context.Context, _, dir string) (fidelity.Report, error) {
		running <- dir
		<-release
		return fidelity.Summarize("r", []fidelity.Shot{{Diff: 0.2}}, 0), nil
	}
	hook := fidelityHook(func() (string, error) { return "/chrome", nil }, check)
	lines := make(chan string, 2)
	for _, d := range []string{"/a", "/b"} {
		go func() {
			line, err := hook(context.Background(), d)
			if err != nil {
				t.Error(err)
			}
			lines <- line
		}()
	}
	<-running
	select {
	case d := <-running:
		t.Fatalf("a second check (%s) ran beside the first", d)
	case <-time.After(100 * time.Millisecond):
	}
	release <- struct{}{}
	<-running
	release <- struct{}{}
	for i := 0; i < 2; i++ {
		if line := <-lines; !strings.HasPrefix(line, "fidelity median 0.20%") || !strings.Contains(line, "report.md") {
			t.Errorf("line %q", line)
		}
	}
	// A cancelled wait gives up.
	fidelitySlot <- struct{}{}
	defer func() { <-fidelitySlot }()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := hook(ctx, "/c"); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled: %v", err)
	}
}
