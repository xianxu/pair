package launcher

import (
	"sort"
	"strconv"
	"strings"
)

// Pure parsers for zellij list-sessions / list-clients output (#99 M2). The
// OSRuntime IO methods (SessionBlocksReuse, ShowFamilyExisting, ProbeSessionName)
// exec zellij then delegate the classification here so the historically
// bug-prone #54/#67 logic is unit-tested without a live daemon (ARCH-PURE) and
// the row scan isn't duplicated across those methods (ARCH-DRY). The single-row
// classification lives in zellijsession, shared with the title poller (#420).

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

// sessionNameRejected decodes Zellij 0.45.1's complete native Clap validator
// rejection for this exact request. Only a pure exit's caller may use it; a
// collector/cancellation failure cannot establish a socket-name length bound.
func sessionNameRejected(session string, status int, stdout, stderr string) bool {
	if status != 2 || stdout != "" {
		return false
	}
	const suffix = " characters\n\nFor more information, try '--help'.\n"
	prefix := "error: invalid value '" + session + "' for '--session <SESSION>': session name must be less than "
	limit, ok := strings.CutPrefix(stderr, prefix)
	if !ok {
		return false
	}
	limit, ok = strings.CutSuffix(limit, suffix)
	if !ok {
		return false
	}
	value, err := strconv.ParseUint(limit, 10, 64)
	return err == nil && strconv.FormatUint(value, 10) == limit
}

// sessionNameAbsentDiagnostic recognizes post-parse action diagnostics from
// Zellij 0.45.1. Its --session Clap validator runs before these branches.
func sessionNameAbsentDiagnostic(session, stderr string) bool {
	diagnostic := strings.TrimSpace(stderr)
	if diagnostic == "There is no active session!" {
		return true
	}
	heading := "Session '" + session + "' not found. The following sessions are active:\n"
	active, ok := strings.CutPrefix(diagnostic, heading)
	return ok && active != "" && !strings.ContainsAny(active, "\r\n") &&
		active != "No active zellij sessions found." && !strings.HasPrefix(active, "Error occurred:")
}

// pairSessionNames extracts the Pair session names from `zellij
// list-sessions --short` output — sorted + deduped, matching the shell's
// `awk '/^pair-/' | sort` (list flow, shell 1235/232), widened to pair's
// ownership predicate so both name schemes are listed (#130).
func pairSessionNames(short string) []string {
	seen := map[string]bool{}
	var names []string
	for _, line := range strings.Split(short, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || !isPairSessionName(fields[0]) || seen[fields[0]] {
			continue
		}
		seen[fields[0]] = true
		names = append(names, fields[0])
	}
	sort.Strings(names)
	return names
}
