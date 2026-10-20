package launcher

import (
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

// Zellij's multi-session not-found branch calls list_sessions, which exits
// zero before the caller can exit one. Accept only its complete listing kind;
// this teaches name acceptance, never session liveness or ownership (#672).
func sessionNameAbsentListing(session, stdout string) bool {
	if stdout == "" || !strings.HasSuffix(stdout, "\n") {
		return false
	}
	// Only the SGR sequences emitted by 0.45.1's print_sessions are metadata.
	// Other escapes remain and fail the existing strict row decoder.
	plain := strings.NewReplacer("\x1b[32;1m", "", "\x1b[35;1m", "", "\x1b[31;1m", "", "\x1b[m", "").Replace(stdout)
	states, err := sessionListingStates([]byte(plain))
	if err != nil || len(states) == 0 {
		return false
	}
	_, present := states[session]
	return !present
}
