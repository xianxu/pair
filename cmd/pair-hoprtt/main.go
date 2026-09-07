// Command pair-hoprtt measures the two latencies a performance capture needs and
// shell cannot time honestly.
//
//	pair-hoprtt              # pipe round-trip: ONE scheduler wake-up, isolated
//	pair-hoprtt -spawn N -- cmd args...   # fork+exec+run of cmd, N times
//
// Both print "median p90 p99 samples" in milliseconds.
//
// Why this is a binary rather than a shell loop: a pipe hop is ~7 microseconds.
// Shell cannot time that without spawning a clock process per sample, and doing
// so measures the clock. An earlier attempt at exactly this read 18.7 ms for
// /usr/bin/true against a known 1.9 ms -- it was timing python startup. One
// in-process timer serves both modes so there is no second implementation to
// drift (ARCH-DRY).
package main

import (
	"fmt"
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
	cmd := exec.Command(os.Args[0], "-child")
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

// spawnRTT times n full fork+exec+run cycles of argv.
func spawnRTT(n int, argv []string) []float64 {
	samples := make([]float64, 0, n)
	for i := 0; i < n; i++ {
		c := exec.Command(argv[0], argv[1:]...)
		start := time.Now()
		_ = c.Run()
		samples = append(samples, msSince(start))
	}
	return samples
}

func msSince(t time.Time) float64 { return float64(time.Since(t).Microseconds()) / 1000.0 }

func report(samples []float64) {
	med, p90, p99 := summary(samples)
	fmt.Printf("%.3f %.3f %.3f %d\n", med, p90, p99, len(samples))
}

func main() {
	args := os.Args[1:]
	if len(args) > 0 && args[0] == "-child" {
		child()
		return
	}
	if len(args) > 0 && args[0] == "-spawn" {
		var n int
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, "usage: pair-hoprtt -spawn N -- cmd [args...]")
			os.Exit(2)
		}
		if _, err := fmt.Sscanf(args[1], "%d", &n); err != nil || n < 1 {
			fmt.Fprintln(os.Stderr, "pair-hoprtt: -spawn needs a positive count")
			os.Exit(2)
		}
		rest := args[2:]
		if len(rest) > 0 && rest[0] == "--" {
			rest = rest[1:]
		}
		if len(rest) == 0 {
			fmt.Fprintln(os.Stderr, "pair-hoprtt: -spawn needs a command")
			os.Exit(2)
		}
		report(spawnRTT(n, rest))
		return
	}
	n := 500
	samples, err := pipeRTT(n)
	if err != nil {
		fmt.Fprintln(os.Stderr, "pair-hoprtt:", err)
		os.Exit(1)
	}
	report(samples)
}
