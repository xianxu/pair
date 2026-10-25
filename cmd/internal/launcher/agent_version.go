package launcher

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// agentVersionProbeTimeout bounds the read-only `<agent> --version` probe; a
// wedged install must never delay a launch past it.
const agentVersionProbeTimeout = 3 * time.Second

// codexNoDaemonVersion is the first Codex CLI that runs a shared background
// server and offers --no-daemon (pair#891).
var codexNoDaemonVersion = [3]int{0, 164, 0}

var agentVersionPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?$`)

// parseAgentVersion reads the version token from a `<agent> --version` line
// (`codex-cli 0.164.0`); anything else is "".
func parseAgentVersion(out string) string {
	fields := strings.Fields(out)
	if len(fields) == 0 {
		return ""
	}
	last := fields[len(fields)-1]
	if !agentVersionPattern.MatchString(last) {
		return ""
	}
	return last
}

// versionAtLeast compares numeric release components; a pre-release suffix
// sorts with its release. Unparseable input is never at least anything, so an
// unknown CLI does not gain a flag it may not support.
func versionAtLeast(version string, want [3]int) bool {
	release, _, _ := strings.Cut(version, "-")
	release, _, _ = strings.Cut(release, "+")
	parts := strings.Split(release, ".")
	if len(parts) != 3 {
		return false
	}
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return false
		}
		if n != want[i] {
			return n > want[i]
		}
	}
	return true
}

func argumentPresent(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}
