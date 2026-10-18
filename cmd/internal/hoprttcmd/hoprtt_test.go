package hoprttcmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"testing/synctest"
	"time"

	"github.com/xianxu/pair/cmd/internal/testpair"
)

// pairBin uses the REAL pair binary, because the probe is a subcommand of it.
// Testing the package function alone would not catch the thing BR-22 was about:
// whether the probe is reachable through the binary that actually ships.
func pairBin(t *testing.T) string {
	t.Helper()
	return testpair.Binary(t)
}

func runPair(t *testing.T, bin string, args ...string) []float64 {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Env = testpair.Environment(t, t.TempDir())
	out, err := cmd.Output()
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

// Command preparation is harness work; only Run belongs in each sample.
// Virtual time makes that distinction exact even when the host is overloaded.
func TestSpawnTimerMeasuresTheCommandNotTheHarness(t *testing.T) {
	for _, setup := range []time.Duration{0, 5 * time.Second, time.Hour} {
		t.Run(setup.String(), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				durations := []time.Duration{250 * time.Microsecond, 17 * time.Millisecond, 100 * time.Millisecond}
				prepared, ran := 0, 0
				samples, failed := sampleSpawns(len(durations), func() func() error {
					if prepared != ran {
						t.Fatal("prepared a new command before running the previous one")
					}
					i := prepared
					prepared++
					time.Sleep(setup)
					return func() error {
						if ran != i {
							t.Fatalf("command %d ran out of order or more than once", i)
						}
						ran++
						time.Sleep(durations[i])
						if i == 1 {
							return errors.New("controlled command failure")
						}
						return nil
					}
				})
				if prepared != len(durations) || ran != len(durations) || failed != 1 || len(samples) != len(durations) {
					t.Fatalf("prepared=%d ran=%d failed=%d samples=%v", prepared, ran, failed, samples)
				}
				for i, duration := range durations {
					if want := float64(duration/time.Microsecond) / 1000; samples[i] != want {
						t.Errorf("sample %d = %v ms, want command-only %v ms (setup %v)", i, samples[i], want, setup)
					}
				}
			})
		})
	}
}

// Retain real binary wiring and output coverage without certifying host speed.
func TestSpawnProbeIsReachableThroughTheShippedPairBinary(t *testing.T) {
	if _, err := os.Stat("/usr/bin/true"); err != nil {
		t.Skip("/usr/bin/true unavailable")
	}
	got := runPair(t, pairBin(t), "hoprtt", "-spawn", "20", "--", "/usr/bin/true")
	if !(got[0] > 0 && got[0] <= got[1] && got[1] <= got[2]) {
		t.Fatalf("spawn probe returned nonpositive or unordered measurements: %v", got)
	}
	t.Logf("native spawn median/p90/p99 milliseconds: %v", got)
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
	cmd := exec.Command(bin, "hoprtt", "-spawn", "3", "--", "/usr/bin/false")
	cmd.Env = testpair.Environment(t, t.TempDir())
	if err := cmd.Run(); err == nil {
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
