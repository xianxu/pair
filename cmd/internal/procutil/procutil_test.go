package procutil

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestAlive(t *testing.T) {
	if Alive("") {
		t.Error("empty pid must not be alive")
	}
	self := strconv.Itoa(os.Getpid())
	if !Alive(self) {
		t.Errorf("own pid %s should be alive", self)
	}
	// A pid that (almost certainly) doesn't exist.
	if Alive("2147483646") {
		t.Error("bogus high pid should not be alive")
	}
}

func TestCommand(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native inspection unsupported")
	}
	if Command("") != "" {
		t.Error("empty pid must yield empty command")
	}
	self := strconv.Itoa(os.Getpid())
	got := Command(self)
	if got == "" {
		t.Errorf("own pid %s should have a non-empty command line", self)
	}
	// Native argv contains no output-formatting newline.
	if len(got) > 0 && got[len(got)-1] == '\n' {
		t.Errorf("command line should be newline-trimmed, got %q", got)
	}
}

func TestIdentity(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native inspection unsupported")
	}
	if Identity("") != "" {
		t.Error("empty pid must yield empty identity")
	}
	self := strconv.Itoa(os.Getpid())
	first := Identity(self)
	second := Identity(self)
	if first == "" {
		t.Fatalf("own pid %s should have a process identity", self)
	}
	if second != first {
		t.Fatalf("identity changed for same process: %q != %q", first, second)
	}
	if Identity("2147483646") != "" {
		t.Error("bogus high pid must yield empty identity")
	}
	for _, pid := range []string{"self", "thread-self", "-1", "0", "not-a-pid"} {
		if got := Identity(pid); got != "" {
			t.Errorf("Identity(%q) = %q, want empty identity for invalid pid", pid, got)
		}
	}
}

func TestPositivePID(t *testing.T) {
	for _, tc := range []struct {
		pid  string
		want bool
	}{
		{pid: "1", want: true},
		{pid: strconv.Itoa(os.Getpid()), want: true},
		{pid: "self"},
		{pid: "thread-self"},
		{pid: "-1"},
		{pid: "0"},
		{pid: "not-a-pid"},
	} {
		_, got := positivePID(tc.pid)
		if got != tc.want {
			t.Errorf("positivePID(%q) valid = %v, want %v", tc.pid, got, tc.want)
		}
	}
}

func TestDescendantPIDsIncludesNestedChildren(t *testing.T) {
	children := map[string][]string{
		"10": {"11", "12"},
		"11": {"13"},
		"13": {"14"},
	}
	got := DescendantPIDs("10", children)
	want := []string{"10", "11", "12", "13", "14"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("DescendantPIDs = %v, want %v", got, want)
	}
}

// #220 changed Alive from `kill -0` in a subprocess to a syscall, and declared
// two behaviours while doing it: a positivePID guard, and EPERM meaning alive.
// Neither was pinned by a test (BR-6). Signal 0 is the null signal, so what the
// guard prevents here is a FALSE POSITIVE — Kill(0, 0) and Kill(-1, 0) both
// succeed — which would report a live pane for a registry line reading `1 0`.
// It also stops those pid selectors reaching any future call that sends a real
// signal.
func TestAliveRefusesPIDsThatWouldBroadcastASignal(t *testing.T) {
	for _, pid := range []string{"", "0", "-1", "-12345", "abc", "1.5", " 1"} {
		if Alive(pid) {
			t.Errorf("Alive(%q) = true; Kill succeeds for these pid SELECTORS, so an unguarded check reports a pane that does not exist", pid)
		}
	}
}

func TestAliveReportsThisProcessAndNotADeadOne(t *testing.T) {
	if !Alive(strconv.Itoa(os.Getpid())) {
		t.Error("Alive(self) = false")
	}
	// PID 1 (launchd/init) exists and is NOT ours: the EPERM branch. A
	// subprocess `kill -0` got this right by accident via its exit status; the
	// syscall has to say so explicitly.
	if !Alive("1") {
		t.Error("Alive(1) = false; EPERM means the process exists but is not ours — still alive")
	}
	// A pid that cannot exist: above the platform maximum.
	if Alive("4194305") {
		t.Error("Alive(4194305) = true for a pid past the platform maximum")
	}
}

// Table's identity is the same token Identity reads, so a reaper's later
// identity re-check compares like with like; and this process's row names its
// real parent.
func TestTableMatchesIdentityAndParent(t *testing.T) {
	rows, err := Table()
	if err != nil {
		t.Fatalf("process table unavailable: %v", err)
	}
	self := os.Getpid()
	for _, r := range rows {
		if r.PID != self {
			continue
		}
		if r.Identity != Identity(strconv.Itoa(self)) || r.PPID != os.Getppid() {
			t.Fatalf("row %+v, identity %q, ppid %d", r, Identity(strconv.Itoa(self)), os.Getppid())
		}
		return
	}
	t.Fatal("this process is missing from the table")
}

// Table reads niceness (#502): a child started under `nice -n 7` shows 7
// more than this process.
func TestTableReadsNice(t *testing.T) {
	// nice(1) warns and runs the command anyway when it may not lower it.
	if out, err := exec.Command("nice", "-n", "7", "true").CombinedOutput(); err != nil || len(out) > 0 {
		t.Skipf("lowering priority is refused here (a sandbox): %v %s", err, out)
	}
	cmd := exec.Command("nice", "-n", "7", "sleep", "5")
	if err := cmd.Start(); err != nil {
		t.Skip(err)
	}
	defer func() { cmd.Process.Kill(); cmd.Wait() }()
	// nice(1) lowers itself, then execs sleep: wait for that.
	var got, own int
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); time.Sleep(20 * time.Millisecond) {
		rows, err := Table()
		if err != nil {
			t.Fatalf("process table unavailable: %v", err)
		}
		got, own = -100, -100
		for _, r := range rows {
			switch r.PID {
			case os.Getpid():
				own = r.Nice
			case cmd.Process.Pid:
				got = r.Nice
			}
		}
		if own != -100 && got == min(own+7, 20) {
			return
		}
	}
	t.Fatalf("child nice %d, want %d", got, min(own+7, 20))
}
