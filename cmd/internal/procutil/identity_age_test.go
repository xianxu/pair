package procutil

import (
	"os/exec"
	"runtime"
	"strconv"
	"testing"
	"time"
)

// The orphan sweep gates on a process's age, and the only age evidence a
// snapshot carries is the identity token — so the token has to name a wall-clock
// birth on every platform that has one. That is measured against a live child
// here rather than read from a header, because the two platforms disagree on the
// token's meaning: darwin's is "<sec>.<usec>", linux's is ticks since boot
// (#583 Task 9). A platform whose token cannot become a wall clock returns
// not-ok, which leaves the age unknown, and an unknown age is never an orphan.
func TestStartTimeFromIdentityNamesALiveChildsBirth(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skipf("no native identity token on %s", runtime.GOOS)
	}
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	token := Identity(strconv.Itoa(cmd.Process.Pid))
	started, ok := StartTimeFromIdentity(token)
	if !ok {
		t.Fatalf("StartTimeFromIdentity(%q) = not ok; want this child's birth", token)
	}
	if age := time.Since(started); age < 0 || age > 5*time.Second {
		t.Fatalf("child's age = %v; want within 5s of now (token was %q)", age, token)
	}
}

// A dead pid has no birth to report, and "gone" must not become "age unknown but
// classified": the caller gates a kill on this, so a missing token is not-ok.
func TestStartTimeFromIdentityRefusesAGonePid(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skipf("no native identity token on %s", runtime.GOOS)
	}
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sleep: %v", err)
	}
	pid := strconv.Itoa(cmd.Process.Pid)
	_ = cmd.Process.Kill()
	_, _ = cmd.Process.Wait()

	token := Identity(pid)
	if started, ok := StartTimeFromIdentity(token); ok {
		t.Fatalf("StartTimeFromIdentity(%q for dead %s) = %v, ok; want not ok", token, pid, started)
	}
}

// Darwin's token is built by two FormatInt calls (table_darwin.go:30), so the
// fractional field is an integer count of microseconds, not a decimal fraction:
// ".5" is 5us. Reading it as a fraction would put every birth up to 100ms late,
// which is exactly the error an age threshold would silently absorb.
func TestParseIdentityWallClock(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		identity string
		want     time.Time
		ok       bool
	}{
		{name: "unpadded microseconds", identity: "1760000000.5", want: time.Unix(1760000000, 5_000), ok: true},
		{name: "padded microseconds", identity: "1760000000.000005", want: time.Unix(1760000000, 5_000), ok: true},
		{name: "half a second", identity: "1760000000.500000", want: time.Unix(1760000000, 500_000_000), ok: true},
		{name: "zero microseconds", identity: "1760000000.0", want: time.Unix(1760000000, 0), ok: true},
		{name: "empty", identity: ""},
		{name: "no fraction", identity: "1760000000"},
		{name: "linux ticks", identity: "4240188"},
		{name: "two fractions", identity: "1760000000.5.5"},
		{name: "negative seconds", identity: "-1760000000.5"},
		{name: "microseconds out of range", identity: "1760000000.1000000"},
		{name: "not numbers", identity: "yesterday.5"},
		{name: "empty fraction", identity: "1760000000."},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, ok := ParseIdentityWallClock(test.identity)
			if ok != test.ok || (test.ok && !got.Equal(test.want)) {
				t.Fatalf("ParseIdentityWallClock(%q) = %v, %v; want %v, %v", test.identity, got, ok, test.want, test.ok)
			}
		})
	}
}

// Linux's token is field 22 of /proc/pid/stat: ticks since boot. Converting it
// needs the boot instant and USER_HZ, and both come from outside the token, so
// the parser takes them — which is also what keeps it testable off a linux host.
func TestParseIdentityTicks(t *testing.T) {
	t.Parallel()
	boot := time.Unix(1760000000, 0)
	for _, test := range []struct {
		name     string
		identity string
		boot     time.Time
		hz       int64
		want     time.Time
		ok       bool
	}{
		{name: "ticks at 100hz", identity: "12345", boot: boot, hz: 100, want: boot.Add(123*time.Second + 450*time.Millisecond), ok: true},
		{name: "born at boot", identity: "0", boot: boot, hz: 100, want: boot, ok: true},
		{name: "a year of uptime", identity: "3153600000", boot: boot, hz: 100, want: boot.Add(365 * 24 * time.Hour), ok: true},
		{name: "empty", identity: "", boot: boot, hz: 100},
		{name: "not a number", identity: "12345.5", boot: boot, hz: 100},
		{name: "negative", identity: "-1", boot: boot, hz: 100},
		{name: "no boot instant", identity: "12345", hz: 100},
		{name: "zero hz", identity: "12345", boot: boot},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, ok := ParseIdentityTicks(test.identity, test.boot, test.hz)
			if ok != test.ok || (test.ok && !got.Equal(test.want)) {
				t.Fatalf("ParseIdentityTicks(%q, %v, %d) = %v, %v; want %v, %v", test.identity, test.boot, test.hz, got, ok, test.want, test.ok)
			}
		})
	}
}

func TestParseBootTime(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		raw  string
		want time.Time
		ok   bool
	}{
		{
			name: "a real /proc/stat head",
			raw:  "cpu  1 2 3 4 5 6 7 8 9\nintr 11 22\nctxt 33\nbtime 1760000000\nprocesses 44\n",
			want: time.Unix(1760000000, 0),
			ok:   true,
		},
		{name: "empty", raw: ""},
		{name: "no btime line", raw: "cpu 1 2 3\nctxt 33\n"},
		{name: "btime without a value", raw: "btime\n"},
		{name: "btime not a number", raw: "btime yesterday\n"},
		{name: "btime zero", raw: "btime 0\n"},
		{name: "a field that merely mentions btime", raw: "xbtime 1760000000\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, ok := ParseBootTime([]byte(test.raw))
			if ok != test.ok || (test.ok && !got.Equal(test.want)) {
				t.Fatalf("ParseBootTime(%q) = %v, %v; want %v, %v", test.raw, got, ok, test.want, test.ok)
			}
		})
	}
}
