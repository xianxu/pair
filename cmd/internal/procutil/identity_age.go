package procutil

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// linuxUserHZ is the tick rate /proc/pid/stat counts starttime in. Every linux
// target this module builds for reports 100 from sysconf(_SC_CLK_TCK); the
// kernel's own CONFIG_HZ does not change the userspace value, and pure Go cannot
// ask for it. TestStartTimeFromIdentityNamesALiveChildsBirth discriminates the
// assumption on any host that runs it: a wrong rate moves a just-started child's
// birth far outside the test's 5s window.
const linuxUserHZ = 100

// StartTimeFromIdentity turns a platform identity token — the one Table and
// Identity report — into the wall-clock birth of the incarnation it names, and
// reports false when the token is empty, malformed, or from a platform whose
// token carries no convertible time. It takes the token, not a pid, so a caller
// holding a snapshot row pays no syscall for the age. Callers gate destructive
// decisions on that age, so false has to mean "age unknown" and never "age
// zero" (#583).
func StartTimeFromIdentity(identity string) (time.Time, bool) {
	switch runtime.GOOS {
	case "darwin":
		return ParseIdentityWallClock(identity)
	case "linux":
		boot, ok := bootTime()
		if !ok {
			return time.Time{}, false
		}
		return ParseIdentityTicks(identity, boot, linuxUserHZ)
	}
	return time.Time{}, false
}

// bootTime reads the instant this linux boot began. It is read per call rather
// than cached: the read is one small file, it happens only for rows that already
// cleared the sweep's cheap gates, and a cache would survive a reboot and then
// date every process from the previous one.
func bootTime() (time.Time, bool) {
	raw, err := os.ReadFile("/proc/stat")
	if err != nil {
		return time.Time{}, false
	}
	return ParseBootTime(raw)
}

// ParseIdentityWallClock reads darwin's token, "<sec>.<usec>". The fractional
// field is an integer count of microseconds, not a decimal fraction, because
// table_darwin.go builds the token with two FormatInt calls: ".5" is 5us, and
// reading it as a fraction would date every birth up to 100ms late — an error an
// hours-wide age threshold would absorb silently.
func ParseIdentityWallClock(identity string) (time.Time, bool) {
	seconds, micros, found := strings.Cut(identity, ".")
	if !found {
		return time.Time{}, false
	}
	sec, err := strconv.ParseInt(seconds, 10, 64)
	if err != nil || sec <= 0 {
		return time.Time{}, false
	}
	usec, err := strconv.ParseInt(micros, 10, 64)
	if err != nil || usec < 0 || usec >= int64(time.Second/time.Microsecond) {
		return time.Time{}, false
	}
	return time.Unix(sec, usec*int64(time.Microsecond)), true
}

// ParseIdentityTicks reads linux's token, field 22 of /proc/pid/stat: ticks
// since boot. Both inputs the conversion needs — the boot instant and the tick
// rate — come from outside the token, which is what keeps this parse testable
// off a linux host.
func ParseIdentityTicks(identity string, boot time.Time, hz int64) (time.Time, bool) {
	if boot.IsZero() || hz <= 0 {
		return time.Time{}, false
	}
	ticks, err := strconv.ParseInt(identity, 10, 64)
	if err != nil || ticks < 0 {
		return time.Time{}, false
	}
	// Whole seconds first, then the remainder, so a year of uptime cannot
	// overflow a Duration.
	elapsed := time.Duration(ticks/hz)*time.Second + time.Duration(ticks%hz)*time.Second/time.Duration(hz)
	return boot.Add(elapsed), true
}

// ParseBootTime reads the btime line of /proc/stat. A boot instant of zero is
// refused: it is the epoch, not a boot, and dating a process from it would make
// every process look as old as the machine's clock.
func ParseBootTime(raw []byte) (time.Time, bool) {
	for _, line := range strings.Split(string(raw), "\n") {
		value, found := strings.CutPrefix(line, "btime ")
		if !found {
			continue
		}
		sec, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil || sec <= 0 {
			return time.Time{}, false
		}
		return time.Unix(sec, 0), true
	}
	return time.Time{}, false
}
