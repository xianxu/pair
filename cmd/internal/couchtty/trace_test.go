package couchtty

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/hostty"
	"github.com/xianxu/pair/cmd/internal/ptychild"
)

// The COUCH_TRACE timing trace (pair#206 Task 11).

// One event is one line of four tab-separated fields. The expectations are
// literals: a table that built them with the formatter would assert nothing.
func TestATraceEventIsOneLineOfFourTabSeparatedFields(t *testing.T) {
	at := time.UnixMilli(1757600000123)
	thread := couchcore.ThreadAddress{RepoScope: "816fc349d3faebf8", Tag: "couch-first"}
	for _, tc := range []struct {
		name    string
		event   string
		address couchcore.ThreadAddress
		detail  string
		want    string
	}{
		{"a thread and a detail", traceReattachStart, thread, "attempt=3", "1757600000123\treattach-start\t816fc349d3faebf8/couch-first\tattempt=3"},
		{"neither", traceStartup, couchcore.ThreadAddress{}, "", "1757600000123\tstartup\t-\t-"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatTraceEvent(at, tc.event, tc.address, tc.detail); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// An attempt's end records a code, never an error's text: the trace holds no
// content.
func TestAPassAttemptEndsAsOkItsDiagnosticOrError(t *testing.T) {
	for _, tc := range []struct {
		success    bool
		diagnostic couchcore.ResumeDiagnosticCode
		want       string
	}{
		{true, "", "ok"},
		{true, "session-gone", "ok"},
		{false, "session-gone", "session-gone"},
		{false, "", "error"},
	} {
		if got := reattachDoneDetail(tc.success, tc.diagnostic); got != tc.want {
			t.Errorf("reattachDoneDetail(%v, %q) = %q, want %q", tc.success, tc.diagnostic, got, tc.want)
		}
	}
}

// Off is silent. A trace that was asked for but cannot open says so, through
// the tracer's error and the console's notice: an empty trace must never read
// as "nothing happened".
func TestTheTimingTraceIsSilentWhenOffAndSaysSoWhenItCannotStart(t *testing.T) {
	if tracer, err := newEventTracer(""); tracer != nil || err != nil {
		t.Fatalf("tracing off produced tracer %v err %v", tracer, err)
	}
	var off *eventTracer
	off.record(time.Now(), traceStartup, couchcore.ThreadAddress{}, "")

	unwritable := filepath.Join(t.TempDir(), "no-such-dir", "trace.tsv")
	if tracer, err := newEventTracer(unwritable); tracer != nil || err == nil || !strings.Contains(err.Error(), "COUCH_TRACE") {
		t.Fatalf("an unopenable path gave tracer %v err %v; want no tracer and an error naming COUCH_TRACE", tracer, err)
	}
	con := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	t.Cleanup(con.Stop)
	if err := con.SetEventTrace(unwritable, time.Now()); err == nil {
		t.Fatal("SetEventTrace hid the failure from its caller")
	}
	var bodies []string
	for _, message := range con.feed.Messages() {
		bodies = append(bodies, message.Body)
	}
	if !slices.ContainsFunc(bodies, func(body string) bool { return strings.Contains(body, "COUCH_TRACE") }) {
		t.Fatalf("the console kept a dead trace and said nothing: %q", bodies)
	}
}

// The trace, end to end through the console: the startup stamp it was handed,
// the first paint, the inventory that seeds the pass, then each attempt's start
// and end, in that order and each on its own thread. couch-first reattaches;
// couch-second fails with no diagnostic.
//
// The order holds by construction, not by timing:
//   - SetEventTrace writes startup before Run.
//   - Run paints once before its loop, and inventories land only on the loop.
//   - The seeding is traced before the effects it produces are dispatched.
//   - An attempt's end is traced before the reduction that starts the next.
//
// Inventories are the exception: they also land while an attempt runs, and a
// refresh can land before the provider is set. So they are checked apart from
// the sequence. The one that seeds is the line immediately before pass-seeded.
func TestTheConsoleTracesStartupThePassAndEachAttempt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.tsv")
	f := newFixtureBeforeRun(t, 24, 100, func(con *Console) {
		if err := con.SetEventTrace(path, time.UnixMilli(1757600000000)); err != nil {
			t.Fatal(err)
		}
	})
	root := consoleThread(f, "c1")
	resumeSucceeds(t, f, "couch-first")
	f.con.ArmReattachPass(root)
	f.con.SetActionableProvider(detachedPair(root))
	waitUpTo(t, 2*time.Second, "the pass to finish", func() bool {
		return f.con.menuSnapshot().Reattach.Phase == ReattachDone
	})

	lines := traceLines(t, path)
	if lines[0] != "1757600000000\tstartup\t-\t-" {
		t.Fatalf("first line = %q, want the startup event stamped with the time it was handed", lines[0])
	}
	where := func(address couchcore.ThreadAddress) string { return address.RepoScope + "/" + string(address.Tag) }
	first, second := where(menuAddress("couch-first")), where(menuAddress("couch-second"))
	attempt := regexp.MustCompile(`^attempt=[0-9]+$`)
	var got []string
	seedingInventory := ""
	for i, line := range lines {
		fields := strings.Split(line, "\t")
		if len(fields) != 4 {
			t.Fatalf("line %q has %d fields, want 4", line, len(fields))
		}
		if _, err := strconv.ParseInt(fields[0], 10, 64); err != nil {
			t.Fatalf("line %q does not start with a unix-ms stamp", line)
		}
		switch fields[1] {
		case traceInventory:
			continue
		case tracePassSeeded:
			seedingInventory = strings.Join(strings.Split(lines[i-1], "\t")[1:], " ")
		case traceReattachStart:
			if !attempt.MatchString(fields[3]) {
				t.Fatalf("reattach-start detail %q, want attempt=<n>", fields[3])
			}
			fields[3] = "attempt"
		}
		got = append(got, strings.Join(fields[1:], " "))
	}
	want := []string{
		"startup - -",
		"first-frame " + where(root) + " -",
		"pass-seeded " + where(root) + " pending=2",
		"reattach-start " + first + " attempt",
		"reattach-done " + first + " ok",
		"reattach-start " + second + " attempt",
		"reattach-done " + second + " error",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("trace events:\n  got  %q\n  want %q", got, want)
	}
	if seedingInventory != "inventory - rows=3" {
		t.Fatalf("the line before pass-seeded is %q, want the three-row inventory that seeded it", seedingInventory)
	}
}

func traceLines(t *testing.T, path string) []string {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSuffix(string(body), "\n"), "\n")
}

func countTraceEvents(lines []string, event string) int {
	n := 0
	for _, line := range lines {
		if fields := strings.Split(line, "\t"); len(fields) == 4 && fields[1] == event {
			n++
		}
	}
	return n
}

// The first frame is traced once, however often the console paints after it.
// The pass test above paints only once, so a tracer firing on every paint
// passed it.
func TestTheFirstFrameIsTracedOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.tsv")
	con := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	t.Cleanup(con.Stop)
	if err := con.SetEventTrace(path, time.UnixMilli(1757600000000)); err != nil {
		t.Fatal(err)
	}
	con.showMenu()
	con.showMenu()
	if got := countTraceEvents(traceLines(t, path), traceFirstFrame); got != 1 {
		t.Fatalf("first-frame traced %d times over two paints, want once", got)
	}
}

// Only a pass attempt's end is traced as reattach-done. An operator's own
// resume finishes through the same function, and counting it would corrupt the
// per-attempt timings the trace exists to measure.
func TestOnlyAPassAttemptsEndIsTraced(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.tsv")
	con := New(hostty.NewFakeHost(ptychild.Size{Rows: 24, Cols: 80}), nil)
	t.Cleanup(con.Stop)
	if err := con.SetEventTrace(path, time.UnixMilli(1757600000000)); err != nil {
		t.Fatal(err)
	}
	refused := errors.New("refused")
	con.finishOperation(operationCompletion{name: "resume", err: refused,
		origin: MenuOperationOrigin{Operation: "resume", Attempt: 1, Address: menuAddress("couch-operator")}})
	con.finishOperation(operationCompletion{name: "resume", err: refused,
		origin: MenuOperationOrigin{Operation: "resume", Attempt: 2, Address: menuAddress("couch-pass"), Background: true}})
	var done []string
	for _, line := range traceLines(t, path) {
		if fields := strings.Split(line, "\t"); fields[1] == traceReattachDone {
			done = append(done, fields[2])
		}
	}
	pass := menuAddress("couch-pass")
	if len(done) != 1 || done[0] != pass.RepoScope+"/"+string(pass.Tag) {
		t.Fatalf("reattach-done traced for %q, want only the pass attempt", done)
	}
}

// An unarmed console traces its inventories but never a seeding: only a start
// arms the pass. The console is stopped and Run joined before the trace is
// read, so the refresh that wrote the inventory has finished writing
// everything it was going to.
func TestAnUnarmedConsoleTracesNoSeeding(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.tsv")
	f := newFixtureBeforeRun(t, 24, 100, func(con *Console) {
		if err := con.SetEventTrace(path, time.UnixMilli(1757600000000)); err != nil {
			t.Fatal(err)
		}
	})
	f.con.SetActionableProvider(detachedPair(consoleThread(f, "c1")))
	waitUpTo(t, 2*time.Second, "an inventory in the trace", func() bool {
		body, _ := os.ReadFile(path)
		return strings.Contains(string(body), "\t"+traceInventory+"\t")
	})
	f.con.Stop()
	select {
	case <-f.done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after Stop")
	}
	if lines := traceLines(t, path); countTraceEvents(lines, tracePassSeeded) != 0 {
		t.Fatalf("an unarmed console traced a seeding: %q", lines)
	}
}

// Every trace event constant is documented in the atlas (pair#265 BR-8).
//
// The enumeration went stale three times in one issue -- each time it changed,
// some durable restatement of it did not. A list a reader trusts is a claim the
// tree can check, so this checks it.
func TestAtlasNamesEveryTraceEvent(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "atlas", "couch.md"))
	if err != nil {
		t.Fatal(err)
	}
	atlas := string(raw)
	for _, event := range []string{
		traceStartup, traceFirstFrame, traceInventory,
		tracePassSeeded, traceReattachStart, traceReattachDone, traceNoDestination,
	} {
		if !strings.Contains(atlas, "`"+event+"`") {
			t.Errorf("atlas/couch.md does not document the %q trace event", event)
		}
	}
}
