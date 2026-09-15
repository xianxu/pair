package storagegc

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

// FakeProcessProbe models PID reuse, disappearance and inspection failures.
// Missing PIDs are dead; entries can be changed between collection passes.
type FakeProcessProbe struct {
	Processes map[int]string
	Unknown   map[int]bool
}

func (f *FakeProcessProbe) Inspect(p ProcessIdentity) Liveness {
	if p.PID <= 0 || f.Unknown[p.PID] {
		return ProcessUnknown
	}
	birth, ok := f.Processes[p.PID]
	if !ok {
		return ProcessDead
	}
	if birth == "" || p.Birth == "" {
		return ProcessUnknown
	}
	if birth != p.Birth {
		return ProcessDead
	}
	return ProcessAlive
}

func TestProbeConservativeErrors(t *testing.T) {
	for _, tc := range []struct {
		name    string
		pid     int
		birth   string
		killErr error
		current string
		want    Liveness
	}{
		{"zero", 0, "old", nil, "old", ProcessUnknown},
		{"negative", -1, "old", nil, "old", ProcessUnknown},
		{"missing", 42, "old", syscall.ESRCH, "", ProcessDead},
		{"permission", 42, "old", syscall.EPERM, "old", ProcessUnknown},
		{"unexpected", 42, "old", syscall.EIO, "old", ProcessUnknown},
		{"identity unavailable", 42, "old", nil, "", ProcessUnknown},
		{"identity absent", 42, "", nil, "new", ProcessUnknown},
		{"same", 42, "old", nil, "old", ProcessAlive},
		{"reused", 42, "old", nil, "new", ProcessDead},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := OSProcessProbe{kill: func(pid int, sig syscall.Signal) error {
				if pid <= 0 {
					t.Fatal("nonpositive PID reached syscall")
				}
				if sig != 0 {
					t.Fatal("nonzero signal")
				}
				return tc.killErr
			}, identity: func(int) (string, error) {
				if tc.current == "" {
					return "", errors.New("unavailable")
				}
				return tc.current, nil
			}}
			if got := p.Inspect(ProcessIdentity{PID: tc.pid, Birth: tc.birth}); got != tc.want {
				t.Fatalf("got %s want %s", got, tc.want)
			}
		})
	}
}

func TestOSProcessProbeChildConformance(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	identity, err := CurrentProcessIdentity(cmd.Process.Pid)
	if err != nil {
		t.Fatal(err)
	}
	probe := OSProcessProbe{}
	fake := &FakeProcessProbe{Processes: map[int]string{identity.PID: identity.Birth}}
	for _, p := range []ProcessProbe{probe, fake} {
		if got := p.Inspect(identity); got != ProcessAlive {
			t.Fatalf("live child = %s", got)
		}
		stale := identity
		stale.Birth += "-stale"
		if got := p.Inspect(stale); got != ProcessDead {
			t.Fatalf("reused PID = %s", got)
		}
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = cmd.Wait()
	delete(fake.Processes, identity.PID)
	for _, p := range []ProcessProbe{probe, fake} {
		if got := p.Inspect(identity); got != ProcessDead {
			t.Fatalf("reaped child = %s", got)
		}
	}
}

func TestCurrentProcessIdentity(t *testing.T) {
	for _, pid := range []int{0, -1} {
		if _, err := CurrentProcessIdentity(pid); err == nil {
			t.Fatalf("accepted PID %d", pid)
		}
	}
	a, err := CurrentProcessIdentity(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}
	b, err := CurrentProcessIdentity(os.Getpid())
	if err != nil || a != b || a.Birth == "" {
		t.Fatalf("unstable identity: %v %v %v", a, b, err)
	}
}

// The child identity remains authoritative after its supervisor exits: a dead
// supervisor must never be taken as proof that every owned writer has stopped.
func TestOSProcessProbeChildSurvivesParent(t *testing.T) {
	if os.Getenv("PAIR_GC_ORPHAN_HELPER") == "1" {
		child := exec.Command("sleep", "30")
		if err := child.Start(); err != nil {
			os.Exit(2)
		}
		fmt.Println(child.Process.Pid)
		os.Exit(0)
	}
	parent := exec.Command(os.Args[0], "-test.run=^TestOSProcessProbeChildSurvivesParent$")
	parent.Env = append(os.Environ(), "PAIR_GC_ORPHAN_HELPER=1")
	out, err := parent.Output()
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		t.Fatal(err)
	}
	child, err := os.FindProcess(pid)
	if err != nil {
		t.Fatal(err)
	}
	defer child.Kill()
	identity, err := CurrentProcessIdentity(pid)
	if err != nil {
		t.Fatal(err)
	}
	probe := OSProcessProbe{}
	if got := probe.Inspect(ProcessIdentity{PID: parent.Process.Pid, Birth: "exited"}); got != ProcessDead {
		t.Fatalf("parent = %s", got)
	}
	if got := probe.Inspect(identity); got != ProcessAlive {
		t.Fatalf("surviving child = %s", got)
	}
}
