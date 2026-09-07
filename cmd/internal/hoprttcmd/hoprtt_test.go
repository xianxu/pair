package hoprttcmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// pairBin builds the REAL pair binary, because the probe is a subcommand of it.
// Testing the package function alone would not catch the thing BR-22 was about:
// whether the probe is reachable through the binary that actually ships.
func pairBin(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "pair")
	out, err := exec.Command("go", "build", "-o", bin, "../../pair-go").CombinedOutput()
	if err != nil {
		t.Fatalf("build pair: %v\n%s", err, out)
	}
	return bin
}

func runPair(t *testing.T, bin string, args ...string) []float64 {
	t.Helper()
	out, err := exec.Command(bin, args...).Output()
	if err != nil {
		t.Fatalf("run %v: %v", args, err)
	}
	var med, p90, p99 float64
	var n int
	if _, err := fmt.Sscan(string(out), &med, &p90, &p99, &n); err != nil {
		t.Fatalf("parse %q: %v", out, err)
	}
	if n == 0 {
		t.Fatalf("no samples: %q", out)
	}
	return []float64{med, p90, p99}
}

// BR-22: the probe must be reachable through the SHIPPED binary. A separate
// pair-hoprtt existed only after `make install` -- the Homebrew formula builds
// just ./cmd/pair-go and PAIR_HOME is the extracted bundle root, which carries
// no helper binaries. Reverting the subcommand wiring must fail here; the
// previous PATH-first fix had no test at all.
func TestProbeIsReachableThroughTheShippedPairBinary(t *testing.T) {
	bin := pairBin(t)
	got := runPair(t, bin, "hoprtt")
	if med := got[0]; med <= 0 || med > 2.0 {
		t.Fatalf("pipe round-trip via `pair hoprtt` = %.3f ms; a scheduler wake-up "+
			"is microseconds, so this timed startup rather than a hop", med)
	}
}

// THE positive control. The bug it pins: a timing harness that shells out to
// read a clock measures the clock process. An earlier attempt did exactly that
// and read 18.7 ms for /usr/bin/true against #201's measured 1.9 ms.
//
// The band is deliberately WIDE: this runs inside `make test`'s parallel suite,
// where a 1.5 ms baseline can reach ~4.5 ms under the suite's own spawn load. A
// 15 ms ceiling still catches an 18.7 ms harness bug without going red on a
// busy machine.
func TestSpawnTimerMeasuresTheCommandNotTheHarness(t *testing.T) {
	if _, err := os.Stat("/usr/bin/true"); err != nil {
		t.Skip("/usr/bin/true unavailable")
	}
	bin := pairBin(t)
	got := runPair(t, bin, "hoprtt", "-spawn", "20", "--", "/usr/bin/true")
	if med := got[0]; med < 0.05 || med > 15.0 {
		t.Fatalf("fork+exec median %.3f ms is outside the plausible band; "+
			"the harness is measuring itself, not the command", med)
	}
}

// The two modes must not converge; if they do, one is wrong.
func TestPipeHopIsFarCheaperThanSpawn(t *testing.T) {
	bin := pairBin(t)
	hop := runPair(t, bin, "hoprtt")[0]
	spawn := runPair(t, bin, "hoprtt", "-spawn", "20", "--", "/usr/bin/true")[0]
	if hop >= spawn {
		t.Fatalf("pipe hop %.3f ms >= fork+exec %.3f ms; the modes are not "+
			"measuring different things", hop, spawn)
	}
}

// A probe that discards its command's exit status reports a BROKEN dependency as
// excellent latency: `zellij action` failing instantly would read as the fastest
// number in the report.
func TestAllFailingInvocationsExitNonZeroRatherThanReportingSpeed(t *testing.T) {
	bin := pairBin(t)
	if err := exec.Command(bin, "hoprtt", "-spawn", "3", "--", "/usr/bin/false").Run(); err == nil {
		t.Fatal("a probe whose command always failed exited 0, reporting its speed as healthy")
	}
}

func TestUsageErrorsRatherThanSilentlyMeasuringNothing(t *testing.T) {
	var out, errb bytes.Buffer
	for _, args := range [][]string{
		{"-spawn"},
		{"-spawn", "0", "--", "/usr/bin/true"},
		{"-spawn", "5", "--"},
	} {
		out.Reset()
		errb.Reset()
		if code := Run(args, &out, &errb); code == 0 {
			t.Fatalf("args %v accepted; a probe that measures nothing must not exit 0", args)
		}
	}
}

func TestSummaryOrdersUnsortedSamples(t *testing.T) {
	med, p90, p99 := summary([]float64{5, 1, 4, 2, 3})
	if med != 3 || p90 < 4 || p99 < 4 {
		t.Fatalf("summary = %v/%v/%v; want a sorted median of 3", med, p90, p99)
	}
}

// BR-23's class at its own arity: empty input must not panic or fabricate.
func TestSummaryOnEmptyInputDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("summary panicked on empty input: %v", r)
		}
	}()
	med, _, _ := summary(nil)
	if med != 0 {
		t.Fatalf("empty summary median = %v; want the zero value, not a fabrication", med)
	}
}
