// Package hoprttcmd measures the two latencies a performance capture needs and
// shell cannot time honestly.
//
//	pair hoprtt              # pipe round-trip: ONE scheduler wake-up, isolated
//	pair hoprtt -spawn N -- cmd args...   # fork+exec+run of cmd, N times
//
// Both print "median p90 p99 samples" in milliseconds.
//
// Why this is a binary rather than a shell loop: a pipe hop is ~7 microseconds.
// Shell cannot time that without spawning a clock process per sample, and doing
// so measures the clock. An earlier attempt at exactly this read 18.7 ms for
// /usr/bin/true against a known 1.9 ms -- it was timing python startup. One
// in-process timer serves both modes so there is no second implementation to
// drift (ARCH-DRY).
//
// It is a SUBCOMMAND OF pair rather than its own binary, and that is the whole
// point: `pair` is the one binary that always exists. A separate pair-hoprtt
// shipped only via `make install` -- the Homebrew formula builds just
// ./cmd/pair-go, and PAIR_HOME at runtime is the extracted bundle root, which
// carries no helper binaries. So the probe would have been permanently n/a for
// every installed pair, which is exactly the operator this capture is for.
package hoprttcmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"time"
)

// child echoes bytes forever. It is the far side of the pipe hop: the process
// that must be woken to make progress.
func child() {
	buf := make([]byte, 1)
	for {
		if _, err := os.Stdin.Read(buf); err != nil {
			return
		}
		if _, err := os.Stdout.Write(buf); err != nil {
			return
		}
	}
}

// summary reduces samples to median/p90/p99. Sorting in place is fine; the
// caller has no further use for the order.
func summary(samples []float64) (med, p90, p99 float64) {
	// Empty input returns zeros rather than indexing into nothing. A probe whose
	// samples all failed reaches here, and panicking would take the whole
	// capture down at the one moment it is needed.
	if len(samples) == 0 {
		return 0, 0, 0
	}
	sort.Float64s(samples)
	at := func(q float64) float64 { return samples[int(q*float64(len(samples)-1))] }
	return at(0.5), at(0.9), at(0.99)
}

// pipeRTT round-trips a byte through a child process n times.
//
// The warmup is not decoration: the first iterations pay page faults and
// dynamic linking, which are startup costs rather than the scheduling cost this
// probe exists to report.
func pipeRTT(n int) ([]float64, error) {
	// Re-exec THIS binary as `pair hoprtt -child`. os.Args[0] is pair, so the
	// subcommand token has to be replayed -- a bare "-child" would land in
	// pair's own dispatcher.
	cmd := exec.Command(os.Args[0], "hoprtt", "-child")
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	defer func() { _ = in.Close(); _ = cmd.Wait() }()

	one, buf := []byte{'x'}, make([]byte, 1)
	for i := 0; i < 50; i++ {
		if _, err := in.Write(one); err != nil {
			return nil, err
		}
		if _, err := out.Read(buf); err != nil {
			return nil, err
		}
	}
	samples := make([]float64, 0, n)
	for i := 0; i < n; i++ {
		start := time.Now()
		if _, err := in.Write(one); err != nil {
			break
		}
		if _, err := out.Read(buf); err != nil {
			break
		}
		samples = append(samples, msSince(start))
	}
	if len(samples) == 0 {
		return nil, fmt.Errorf("no samples")
	}
	return samples, nil
}

// spawnRTT times n full fork+exec+run cycles of argv, and reports how many of
// them FAILED.
//
// The failure count is not bookkeeping. A command that exits immediately with an
// error is fast, so discarding the error reports a broken dependency as healthy
// latency -- `zellij_action_ms=2.1` reads as "zellij is quick" when it actually
// means "zellij is gone". The caller degrades to n/a instead.
func spawnRTT(n int, argv []string) (samples []float64, failed int) {
	samples = make([]float64, 0, n)
	for i := 0; i < n; i++ {
		c := exec.Command(argv[0], argv[1:]...)
		start := time.Now()
		err := c.Run()
		samples = append(samples, msSince(start))
		if err != nil {
			failed++
		}
	}
	return samples, failed
}

func msSince(t time.Time) float64 { return float64(time.Since(t).Microseconds()) / 1000.0 }

// report prints "median p90 p99 samples [failed]". The trailing count appears
// only when something failed, so a healthy line keeps its four-field shape and
// a caller that ignores the field cannot mistake a broken probe for a fast one.
func report(w io.Writer, samples []float64, failed int) {
	med, p90, p99 := summary(samples)
	if failed > 0 {
		fmt.Fprintf(w, "%.3f %.3f %.3f %d %d\n", med, p90, p99, len(samples), failed)
		return
	}
	fmt.Fprintf(w, "%.3f %.3f %.3f %d\n", med, p90, p99, len(samples))
}

// Run is the subcommand entrypoint. `rest` excludes the "hoprtt" token.
func Run(rest []string, stdout, stderr io.Writer) int {
	args := rest
	if len(args) > 0 && args[0] == "-child" {
		child()
		return 0
	}
	if len(args) > 0 && args[0] == "-spawn" {
		var n int
		if len(args) < 3 {
			fmt.Fprintln(stderr, "usage: pair hoprtt -spawn N -- cmd [args...]")
			return 2
		}
		if _, err := fmt.Sscanf(args[1], "%d", &n); err != nil || n < 1 {
			fmt.Fprintln(stderr, "pair hoprtt: -spawn needs a positive count")
			return 2
		}
		argv := args[2:]
		if len(argv) > 0 && argv[0] == "--" {
			argv = argv[1:]
		}
		if len(argv) == 0 {
			fmt.Fprintln(stderr, "pair hoprtt: -spawn needs a command")
			return 2
		}
		samples, failed := spawnRTT(n, argv)
		if failed == len(samples) {
			fmt.Fprintf(stderr, "pair hoprtt: every invocation of %v failed\n", argv)
			return 1
		}
		report(stdout, samples, failed)
		return 0
	}
	samples, err := pipeRTT(500)
	if err != nil {
		fmt.Fprintln(stderr, "pair hoprtt:", err)
		return 1
	}
	report(stdout, samples, 0)
	return 0
}
