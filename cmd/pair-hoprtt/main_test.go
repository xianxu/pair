package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// build produces the probe binary once per test run. The pipe mode re-execs
// os.Args[0], so it needs a real binary rather than the test harness.
func build(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "pair-hoprtt")
	out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	return bin
}

func run(t *testing.T, bin string, args ...string) []float64 {
	t.Helper()
	out, err := exec.Command(bin, args...).Output()
	if err != nil {
		t.Fatalf("run %v: %v", args, err)
	}
	var med, p90, p99 float64
	var n int
	if _, err := fmtSscan(string(out), &med, &p90, &p99, &n); err != nil {
		t.Fatalf("parse %q: %v", out, err)
	}
	if n == 0 {
		t.Fatalf("no samples: %q", out)
	}
	return []float64{med, p90, p99}
}

// THE positive control, and the reason this file exists before any other code.
//
// The bug it pins: a timing harness that shells out to read a clock measures the
// clock process, not the command. An earlier attempt did exactly that and read
// 18.7 ms for /usr/bin/true against #201's measured 1.9 ms.
//
// The band is deliberately WIDE. This runs inside `make test`'s parallel
// `go test ./... -count=1`, where a 1.5 ms baseline can reach ~4.5 ms under the
// suite's own spawn load. A 15 ms ceiling still catches an 18.7 ms harness bug
// without going red on a busy machine.
func TestSpawnTimerMeasuresTheCommandNotTheHarness(t *testing.T) {
	bin := build(t)
	if _, err := os.Stat("/usr/bin/true"); err != nil {
		t.Skip("/usr/bin/true unavailable")
	}
	got := run(t, bin, "-spawn", "20", "--", "/usr/bin/true")
	if med := got[0]; med < 0.05 || med > 15.0 {
		t.Fatalf("fork+exec median %.3f ms is outside the plausible band; "+
			"the harness is measuring itself, not the command", med)
	}
}

// A pipe hop is a scheduler wake-up: microseconds. Milliseconds would mean we
// timed process startup instead -- the same failure from the other direction.
func TestPipeRoundTripIsMicrosecondsNotMilliseconds(t *testing.T) {
	bin := build(t)
	got := run(t, bin)
	if med := got[0]; med <= 0 || med > 2.0 {
		t.Fatalf("pipe round-trip median %.3f ms; a scheduler wake-up is "+
			"microseconds, so this is timing startup, not a hop", med)
	}
}

// The two modes must not be the same measurement. A pipe hop is orders of
// magnitude cheaper than fork+exec; if they converge, one of them is wrong.
func TestPipeHopIsFarCheaperThanSpawn(t *testing.T) {
	bin := build(t)
	hop := run(t, bin)[0]
	spawn := run(t, bin, "-spawn", "20", "--", "/usr/bin/true")[0]
	if hop >= spawn {
		t.Fatalf("pipe hop %.3f ms >= fork+exec %.3f ms; the modes are not "+
			"measuring different things", hop, spawn)
	}
}

func TestSummaryOrdersUnsortedSamples(t *testing.T) {
	med, p90, p99 := summary([]float64{5, 1, 4, 2, 3})
	if med != 3 || p90 < 4 || p99 < 4 {
		t.Fatalf("summary = %v/%v/%v; want a sorted median of 3", med, p90, p99)
	}
}

func TestUsageErrorsRatherThanSilentlyMeasuringNothing(t *testing.T) {
	bin := build(t)
	for _, args := range [][]string{
		{"-spawn"},
		{"-spawn", "0", "--", "/usr/bin/true"},
		{"-spawn", "5", "--"},
	} {
		if err := exec.Command(bin, args...).Run(); err == nil {
			t.Fatalf("args %v accepted; a probe that measures nothing must not exit 0", args)
		}
	}
}

// fmtSscan is fmt.Sscan behind a name, so the test's parse failure message can
// say what it was parsing.
func fmtSscan(s string, a ...any) (int, error) { return fmt.Sscan(s, a...) }
