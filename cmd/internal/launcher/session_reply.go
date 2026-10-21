package launcher

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Zellij 0.45.1 sessions::print_sessions emits this humantime 2.1.0 grammar.
// We validate its shape, not the elapsed time's value: it conveys no authority.
var sessionCreatedDuration = regexp.MustCompile(`^(?:[0-9]+(?:years?|months?|days?|h|m|s|ms|us|ns))(?: [0-9]+(?:years?|months?|days?|h|m|s|ms|us|ns))*$`)

// validateSessionListing checks the expected no-formatting reply kind before
// one-shot callers infer absence. Empty input is allowed ONLY because the IO
// boundary separately proves Zellij's exact no-active-sessions diagnostic;
// successful empty stdout must already have been rejected there.
//
// Native names may contain spaces/tabs, so full rows are delimited by their
// creation metadata rather than Fields[0]. Short output is inherently ambiguous:
// native names can themselves look like JSON or a client-table header. Those
// recognizable other reply kinds fail closed here, even if used as real names.
// Newlines/control characters are likewise not safe evidence in this line format.
// pair:m5-concept pure
func validateSessionListing(raw []byte, short bool) error {
	if len(raw) == 0 {
		return nil
	}
	if !utf8.Valid(raw) {
		return fmt.Errorf("session listing is not UTF-8")
	}
	seen := make(map[string]bool)
	for i, row := range strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n") {
		name := row
		if short {
			trimmed := strings.TrimSpace(row)
			fields := strings.Fields(row)
			if strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "{") ||
				strings.Contains(row, " [Created ") ||
				(len(fields) >= 3 && fields[0] == "CLIENT_ID" && fields[1] == "ZELLIJ_PANE_ID" && fields[2] == "RUNNING_COMMAND") ||
				trimmed == "No active zellij sessions found." || strings.HasPrefix(trimmed, "Error occurred:") {
				return fmt.Errorf("session listing row %d has another reply kind", i+1)
			}
		} else {
			delimiter := strings.LastIndex(row, " [Created ")
			if delimiter < 0 {
				return fmt.Errorf("session listing row %d lacks creation metadata", i+1)
			}
			name = row[:delimiter]
			metadata := row[delimiter+len(" [Created "):]
			duration, suffix, ok := strings.Cut(metadata, " ago]")
			if !ok || !sessionCreatedDuration.MatchString(duration) ||
				(suffix != "" && suffix != " " && suffix != " (current)" && suffix != " (EXITED - attach to resurrect)") {
				return fmt.Errorf("session listing row %d has invalid creation metadata or suffix", i+1)
			}
		}
		if strings.TrimSpace(name) == "" || name == "." || name == ".." || strings.Contains(name, "/") ||
			strings.IndexFunc(row, func(r rune) bool { return unicode.IsControl(r) && r != '\t' }) >= 0 {
			return fmt.Errorf("session listing row %d has an unsafe session name", i+1)
		}
		if seen[name] {
			return fmt.Errorf("session listing row %d repeats a session name", i+1)
		}
		seen[name] = true
	}
	return nil
}

// sessionListingStates derives exact identity and state only after validating
// the complete full reply. Session names are opaque; neither whitespace nor
// EXITED within a name is metadata.
func sessionListingStates(raw []byte) (map[string]bool, error) {
	if err := validateSessionListing(raw, false); err != nil {
		return nil, err
	}
	states := make(map[string]bool)
	if len(raw) == 0 {
		return states, nil
	}
	for _, row := range strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n") {
		delimiter := strings.LastIndex(row, " [Created ")
		states[row[:delimiter]] = strings.HasSuffix(row[delimiter:], " (EXITED - attach to resurrect)")
	}
	return states, nil
}

func sessionListingState(raw []byte, name string) (present, exited bool, err error) {
	states, err := sessionListingStates(raw)
	if err != nil {
		return false, false, err
	}
	exited, present = states[name]
	return present, exited, nil
}

// parseColoredSessionListing decodes only print_sessions' native 0.45.1 SGR
// envelope. It is not an ANSI scrubber: unrelated escapes and partially colored
// rows remain invalid, and the existing whole-list validator owns names/state.
func parseColoredSessionListing(raw []byte) (map[string]bool, error) {
	if len(raw) == 0 || !utf8.Valid(raw) {
		return nil, fmt.Errorf("invalid colored session listing")
	}
	var plain strings.Builder
	for _, row := range strings.Split(strings.TrimSuffix(string(raw), "\n"), "\n") {
		name, rest, ok := strings.Cut(strings.TrimPrefix(row, "\x1b[32;1m"), "\x1b[m [Created \x1b[35;1m")
		if !strings.HasPrefix(row, "\x1b[32;1m") || !ok {
			return nil, fmt.Errorf("invalid colored session name envelope")
		}
		duration, suffix, ok := strings.Cut(rest, "\x1b[m ago]")
		if !ok {
			return nil, fmt.Errorf("invalid colored session duration envelope")
		}
		// Native exited rows color exactly EXITED; a live/current row has no more SGR.
		switch suffix {
		case " ", " (current)":
		case " (\x1b[31;1mEXITED\x1b[m - attach to resurrect)":
			suffix = " (EXITED - attach to resurrect)"
		default:
			return nil, fmt.Errorf("invalid colored session suffix envelope")
		}
		plain.WriteString(name + " [Created " + duration + " ago]" + suffix + "\n")
	}
	return sessionListingStates([]byte(plain.String()))
}
