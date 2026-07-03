package launcher

import (
	"sort"
	"strings"
)

// Pure parsers for zellij list-sessions / list-clients output (#99 M2). The
// OSRuntime IO methods (SessionBlocksReuse, ShowFamilyExisting, ProbeSessionName)
// exec zellij then delegate the classification here so the historically
// bug-prone #54/#67 logic is unit-tested without a live daemon (ARCH-PURE) and
// the row scan isn't duplicated across those methods (ARCH-DRY).

// sessionRowState scans `zellij list-sessions --no-formatting` output for the row
// whose first field is session, reporting whether it is present and whether it
// is an EXITED (resurrectable) row. Absent → (false, false); a running/detached
// row → (true, false); an EXITED row → (true, true).
func sessionRowState(raw, session string) (present, exited bool) {
	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || fields[0] != session {
			continue
		}
		return true, strings.Contains(line, "EXITED")
	}
	return false, false
}

// familyRows returns the list-sessions rows whose session name is exactly
// familyPrefix or begins with familyPrefix+"-" (the pair-<base>* family), in
// input order.
func familyRows(raw, familyPrefix string) []string {
	var rows []string
	for _, line := range strings.Split(raw, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if fields[0] == familyPrefix || strings.HasPrefix(fields[0], familyPrefix+"-") {
			rows = append(rows, strings.TrimSpace(line))
		}
	}
	return rows
}

// sessionNameRejected reports whether zellij's own validator rejected a session
// name as too long for the machine's socket-path budget (#54).
func sessionNameRejected(out string) bool {
	return strings.Contains(out, "session name must be less than")
}

// pairSessionNames extracts the pair-<tag> session names from `zellij
// list-sessions --short` output — sorted + deduped, matching the shell's
// `awk '/^pair-/' | sort` (list flow, shell 1235/232).
func pairSessionNames(short string) []string {
	seen := map[string]bool{}
	var names []string
	for _, line := range strings.Split(short, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || !strings.HasPrefix(fields[0], "pair-") || seen[fields[0]] {
			continue
		}
		seen[fields[0]] = true
		names = append(names, fields[0])
	}
	sort.Strings(names)
	return names
}

// clientCount counts the clients attached to a session from `zellij --session
// NAME action list-clients` output: a header line then one row per client, so
// the count is the non-empty lines after the first (shell 292: `tail -n +2 | wc
// -l`). Empty output (query failed / no session) → 0, treated as detached.
func clientCount(raw string) int {
	lines := strings.Split(strings.TrimRight(raw, "\n"), "\n")
	n := 0
	for i, line := range lines {
		if i == 0 { // header
			continue
		}
		if strings.TrimSpace(line) != "" {
			n++
		}
	}
	return n
}
