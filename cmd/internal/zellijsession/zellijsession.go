// Package zellijsession classifies `zellij list-sessions --no-formatting` rows.
// It is pure and dependency-free so both the launcher and the title poller
// (which the launcher imports) share one row scan (ARCH-DRY, pair#420). The
// `--short` listing cannot serve liveness: it also names EXITED (resurrectable)
// sessions, which is how title pollers outlived their sessions for days.
package zellijsession

import "strings"

// RowState scans list-sessions output for the row whose first field is
// session, reporting whether it is present and whether it is an EXITED
// (resurrectable) row. Absent → (false, false); a running/detached row →
// (true, false); an EXITED row → (true, true).
func RowState(raw, session string) (present, exited bool) {
	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || fields[0] != session {
			continue
		}
		return true, strings.Contains(line, "EXITED")
	}
	return false, false
}

// Live reports whether session is present and not EXITED: a server is running
// it, so its pollers still have a session to serve.
func Live(raw, session string) bool {
	present, exited := RowState(raw, session)
	return present && !exited
}
