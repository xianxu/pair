//go:build darwin && cgo

package procutil

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"syscall"
	"testing"
	"time"
)

func TestResourceOwnedChild(t *testing.T) {
	if os.Getenv("PAIR_RESOURCE_CHILD") == "1" {
		buf := make([]byte, 16<<20)
		for i := range buf {
			buf[i] = byte(i)
		}
		f, err := os.CreateTemp(os.Getenv("PAIR_RESOURCE_DIR"), "resource-")
		if err != nil {
			panic(err)
		}
		defer os.Remove(f.Name())
		defer f.Close()
		fmt.Println("ready")
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			var before syscall.Rusage
			if err := syscall.Getrusage(syscall.RUSAGE_SELF, &before); err != nil {
				panic(err)
			}
			var total uint64
			for {
				for _, b := range buf {
					total += uint64(b)
				}
				var progress syscall.Rusage
				if err := syscall.Getrusage(syscall.RUSAGE_SELF, &progress); err != nil {
					panic(err)
				}
				if progress.Utime.Nano()+progress.Stime.Nano()-before.Utime.Nano()-before.Stime.Nano() >= int64(30*time.Millisecond) {
					break
				}
			}
			if _, err := f.Write(buf); err != nil {
				panic(err)
			}
			if err := f.Sync(); err != nil {
				panic(err)
			}
			runtime.KeepAlive(buf)
			var after syscall.Rusage
			if err := syscall.Getrusage(syscall.RUSAGE_SELF, &after); err != nil {
				panic(err)
			}
			cpuNS := after.Utime.Nano() + after.Stime.Nano() - before.Utime.Nano() - before.Stime.Nano()
			runtime.KeepAlive(total)
			fmt.Println(cpuNS)
		}
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestResourceOwnedChild$")
	cmd.Env = append(os.Environ(), "PAIR_RESOURCE_CHILD=1", "PAIR_RESOURCE_DIR="+t.TempDir())
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		stdin.Close()
		if err := cmd.Wait(); err != nil {
			t.Error(err)
		}
	}()
	scanner := bufio.NewScanner(stdout)
	if !scanner.Scan() {
		t.Fatal("child readiness missing")
	}
	r := NativeResourceReader{}
	rows, n, err := r.Table(ctx)
	if err != nil || n > 6 {
		t.Fatal(n, err)
	}
	var p Process
	for _, row := range rows {
		if row.PID == cmd.Process.Pid {
			p = row
		}
	}
	if p.Identity == "" {
		t.Fatal("child absent")
	}
	info, n, err := r.Command(ctx, p)
	if err != nil || n != 2 || info.Executable == "" {
		t.Fatal(info, n, err)
	}
	before, n, err := r.Usage(ctx, p)
	if err != nil || n != 2 || before.RSSBytes == 0 {
		t.Fatal(before, n, err)
	}
	fmt.Fprintln(stdin, "work")
	if !scanner.Scan() {
		t.Fatal("child work incomplete")
	}
	referenceCPU, err := strconv.ParseUint(scanner.Text(), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	after, n, err := r.Usage(ctx, p)
	if err != nil || n != 2 || after.CPUTimeNS-before.CPUTimeNS < uint64(10*time.Millisecond) || after.RSSBytes == 0 || after.DiskWriteBytes < before.DiskWriteBytes {
		t.Fatal(before, after, n, err)
	}
	delta := after.CPUTimeNS - before.CPUTimeNS
	if delta < referenceCPU/2 || delta > referenceCPU*2 {
		t.Fatalf("rusage CPU=%d getrusage reference=%d", delta, referenceCPU)
	}
	t.Logf("SDK timebase=%d/%d independent getrusage CPU=%d", resourceTimebase.numer, resourceTimebase.denom, referenceCPU)
	t.Logf("owned child CPU delta=%dns RSS=%d physical disk write delta=%d", after.CPUTimeNS-before.CPUTimeNS, after.RSSBytes, after.DiskWriteBytes-before.DiskWriteBytes)
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	a, n, err := r.SelfCPU(ctx)
	if err != nil || n != 1 {
		t.Fatal(a, n, err)
	}
	var total uint64
	for i := uint64(0); i < 1000000; i++ {
		total += i
	}
	runtime.KeepAlive(total)
	b, n, err := r.SelfCPU(ctx)
	if err != nil || n != 1 || b < a {
		t.Fatal(a, b, n, err)
	}
}

// BenchmarkResourceNative exercises the actual bounded observer calls on the
// benchmark's owned process, without inspecting commands of unrelated rows.
func BenchmarkResourceNative(b *testing.B) {
	r := NativeResourceReader{}
	ctx := context.Background()
	rows, _, err := r.Table(ctx)
	if err != nil {
		b.Fatal(err)
	}
	var p Process
	for _, row := range rows {
		if row.PID == os.Getpid() {
			p = row
		}
	}
	if p.Identity == "" {
		b.Fatal("self absent")
	}
	b.Run("Table", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, n, err := r.Table(ctx)
			if err != nil || n > 6 {
				b.Fatal(n, err)
			}
		}
	})
	b.Run("Usage", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, n, err := r.Usage(ctx, p)
			if err != nil || n != 2 {
				b.Fatal(n, err)
			}
		}
	})
	b.Run("Command", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, n, err := r.Command(ctx, p)
			if err != nil || n != 2 {
				b.Fatal(n, err)
			}
		}
	})
	b.Run("SelfCPU", func(b *testing.B) {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, n, err := r.SelfCPU(ctx)
			if err != nil || n != 1 {
				b.Fatal(n, err)
			}
		}
	})
}

// TestResourceNativeCost is opt-in because it enumerates the host process table.
// It enriches only this owned test process, never all host rows.
func TestResourceNativeCost(t *testing.T) {
	if os.Getenv("PAIR_RESOURCE_COST") != "1" {
		t.Skip("set PAIR_RESOURCE_COST=1 for native cost conformance")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	r := NativeResourceReader{}
	ctx := context.Background()
	var sum, maxCPU uint64
	const passes = 3
	for pass := 0; pass < passes; pass++ {
		start, n, err := r.SelfCPU(ctx)
		if err != nil {
			t.Fatal(err)
		}
		calls := n
		wall := time.Now()
		rows, n, err := r.Table(ctx)
		calls += n
		if err != nil {
			t.Fatal(err)
		}
		var owned Process
		eligible := 0
		for _, p := range rows {
			if p.PID == os.Getpid() {
				owned = p
			}
			if p.PPID == 1 && !p.Zombie {
				eligible++
			}
		}
		if owned.Identity == "" {
			t.Fatal("owned process missing")
		}
		_, n, err = r.Usage(ctx, owned)
		calls += n
		if err != nil {
			t.Fatal(err)
		}
		// Exercise the identical native command/fence path on an owned generation.
		// PPID1 filtering is counted above but grants no authority to inspect peers.
		_, n, err = r.Command(ctx, owned)
		calls += n
		if err != nil {
			t.Fatal(err)
		}
		end, n, err := r.SelfCPU(ctx)
		calls += n
		if err != nil {
			t.Fatal(err)
		}
		cpu := end - start
		sum += cpu
		if cpu > maxCPU {
			maxCPU = cpu
		}
		t.Logf("pass=%d rows=%d PPID1_non_zombie=%d selected_usage=1 command_candidates=1 calls=%d cpu_ms=%.3f wall_ms=%.3f sixty_second_cpu_pct=%.6f", pass+1, len(rows), eligible, calls, float64(cpu)/1e6, float64(time.Since(wall))/1e6, float64(cpu)/float64(60*time.Second)*100)
	}
	t.Logf("passes=%d CPU_mean_ms=%.3f CPU_max_ms=%.3f sixty_second_mean_cpu_pct=%.6f calibration_calls=1 (process initialization)", passes, float64(sum)/passes/1e6, float64(maxCPU)/1e6, float64(sum)/passes/float64(60*time.Second)*100)
}
