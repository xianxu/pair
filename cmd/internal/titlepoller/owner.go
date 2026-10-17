package titlepoller

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	OwnerPIDEnv   = "PAIR_TITLE_OWNER_PID"
	OwnerStartEnv = "PAIR_TITLE_OWNER_START"
	LeaseFDEnv    = "PAIR_TITLE_LEASE_FD"
)

// TitleOwner identifies one wrapper incarnation; a PID alone is never an owner.
type TitleOwner struct {
	PID   int
	Start string
}

func ParseTitleOwner(pid, start string) (TitleOwner, error) {
	n, err := strconv.ParseInt(pid, 10, 32)
	sec, micro, split := strings.Cut(start, ".")
	seconds, e1 := strconv.ParseUint(sec, 10, 64)
	micros, e2 := strconv.ParseUint(micro, 10, 32)
	// Darwin's native token is seconds.microseconds; Linux uses start ticks.
	validStart := e1 == nil && seconds > 0 && ((!split) || (e2 == nil && micros < 1_000_000))
	if err != nil || n <= 0 || strconv.FormatInt(n, 10) != pid || !validStart || len(start) > 32 {
		return TitleOwner{}, fmt.Errorf("invalid title wrapper identity")
	}
	return TitleOwner{PID: int(n), Start: start}, nil
}

func (o TitleOwner) Environ() []string {
	return []string{OwnerPIDEnv + "=" + strconv.Itoa(o.PID), OwnerStartEnv + "=" + o.Start, LeaseFDEnv + "=3"}
}
