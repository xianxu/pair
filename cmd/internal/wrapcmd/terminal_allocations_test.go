package wrapcmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

type controlAllocationSample struct {
	PID                  int
	Small, Large, Double float64
}

func sampleControlObserverAllocations() controlAllocationSample {
	allocations := func(stream []byte) float64 {
		return testing.AllocsPerRun(10, func() {
			var observer terminalControlObserver
			observer.Feed(stream)
			if observer.visible {
				panic("OSC payload authorized visibility")
			}
		})
	}
	return controlAllocationSample{
		PID:    os.Getpid(),
		Small:  allocations([]byte("\x1b]0;payload\x9b?25h\x07")),
		Large:  allocations([]byte("\x1b]0;" + strings.Repeat("x", 1<<20) + "\x9b?25h\x07")),
		Double: allocations([]byte("\x1b]0;" + strings.Repeat("x", 2<<20) + "\x9b?25h\x07")),
	}
}

func isolatedControlObserverAllocations(t *testing.T) controlAllocationSample {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestControlObserverAllocationHelper$")
	cmd.Env = setEnv(os.Environ(), "PAIR_724_ALLOCATION_HELPER", "1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("allocation helper: %v\n%s", err, output)
	}
	var sample controlAllocationSample
	if err := json.NewDecoder(bytes.NewReader(output)).Decode(&sample); err != nil {
		t.Fatalf("allocation helper result: %v\n%s", err, output)
	}
	if sample.PID != cmd.Process.Pid {
		t.Fatalf("allocation sample did not come from the measuring child: sample=%d child=%d", sample.PID, cmd.Process.Pid)
	}
	return sample
}

// Only this self-exec helper samples process-global allocation counters. Other
// package tests and their background goroutines stay in the parent process.
func TestControlObserverAllocationHelper(t *testing.T) {
	if os.Getenv("PAIR_724_ALLOCATION_HELPER") != "1" {
		return
	}
	if err := json.NewEncoder(os.Stdout).Encode(sampleControlObserverAllocations()); err != nil {
		t.Fatal(err)
	}
}

func TestTerminalModelControlObserverBoundsStringStorage(t *testing.T) {
	sample := isolatedControlObserverAllocations(t)
	if sample.PID == os.Getpid() || sample.PID <= 0 {
		t.Fatalf("process-global allocation oracle was not isolated: probe PID=%d test PID=%d", sample.PID, os.Getpid())
	}
	if sample.Large > sample.Small+1 || sample.Double > sample.Small+1 {
		t.Fatalf("OSC allocations small=%.0f 1MiB=%.0f 2MiB=%.0f; parser storage is not bounded", sample.Small, sample.Large, sample.Double)
	}
	t.Logf("isolated allocation oracle: small=%.0f 1MiB=%.0f 2MiB=%.0f", sample.Small, sample.Large, sample.Double)
}
