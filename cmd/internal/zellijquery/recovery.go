package zellijquery

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/pair/cmd/internal/zellijreply"
)

// Temporary Zellij 0.45.1 workaround: stale numeric client cleanup can deliver
// another connection's reply or exit successfully before its Log arrives.
// Remove when our supported native release includes upstream PR5630:
// https://github.com/zellij-org/zellij/pull/5630
// Only these idempotent reads may be reissued, at most three total attempts
// within the original query deadline. Never replay a mutation or collector error.
var ErrCrossedReply = errors.New("Zellij query received another reply kind")

type ReplyKind uint8

const (
	UnknownReply ReplyKind = iota
	SessionsReply
	ClientsReply
	PanesReply
	TabReply
)

func ReadKind(args []string) ReplyKind {
	if len(args) == 2 && args[0] == "list-sessions" && args[1] == "--no-formatting" {
		return SessionsReply
	}
	if len(args) >= 2 && args[0] == "--session" && args[1] != "" {
		args = args[2:]
	}
	if len(args) < 2 || args[0] != "action" {
		return UnknownReply
	}
	if len(args) == 2 && args[1] == "list-clients" {
		return ClientsReply
	}
	if len(args) == 3 && args[1] == "current-tab-info" && args[2] == "--json" {
		return TabReply
	}
	if args[1] != "list-panes" {
		return UnknownReply
	}
	seen := map[string]bool{}
	for _, flag := range args[2:] {
		switch flag {
		case "--json", "--command", "--state", "--geometry", "--tab", "--all":
		default:
			return UnknownReply
		}
		if seen[flag] {
			return UnknownReply
		}
		seen[flag] = true
	}
	if seen["--json"] {
		return PanesReply
	}
	return UnknownReply
}

// Malformed same-kind data is left to existing strict caller validation. A
// replay requires a complete validated different kind, not an arbitrary parse
// error. Same-kind misrouting still requires the existing independent agreement.
func CrossedReply(want ReplyKind, raw []byte) bool {
	if want == UnknownReply || want > TabReply || len(bytes.TrimSpace(raw)) == 0 {
		return false
	}
	valid := []bool{false, ValidateSessionListing(raw, false) == nil, false, false, false}
	_, err := zellijreply.ParseClients(raw)
	valid[ClientsReply] = err == nil
	_, err = zellijreply.ParsePanes(raw, zellijreply.PaneRequirements{})
	valid[PanesReply] = err == nil
	_, err = zellijreply.ParseTab(raw)
	valid[TabReply] = err == nil
	if valid[want] {
		return false
	}
	for _, ok := range valid {
		if ok {
			return true
		}
	}
	return false
}

// ErrSessionLookupMiss is read-retry eligibility, never session authority.
var ErrSessionLookupMiss = errors.New("Zellij query falsely missed a live session")

func validSessionLookupName(name string) bool {
	return utf8.ValidString(name) && strings.TrimSpace(name) != "" && name != "." && name != ".." && filepath.Base(name) == name && strings.IndexFunc(name, unicode.IsControl) < 0
}

// RetryableSessionLookupMiss admits only the two observed native lookup failures.
// The caller separately guarantees successful collection (or a direct exit),
// while the existing read allowlist/deadline/attempt budget owns any reissue.
func RetryableSessionLookupMiss(args []string, stdout, stderr []byte, exit int, socketPresent bool) bool {
	kind := ReadKind(args)
	if (kind != ClientsReply && kind != PanesReply) || len(args) < 4 || args[0] != "--session" || !validSessionLookupName(args[1]) {
		return false
	}
	name := args[1]
	if exit == 1 {
		return len(stdout) == 0 && string(stderr) == "There is no active session!\n" && socketPresent
	}
	if exit != 0 || string(stderr) != fmt.Sprintf("Session '%s' not found. The following sessions are active:\n", name) {
		return false
	}
	states, err := SessionListingStates(stdout)
	if err != nil {
		states, err = ParseColoredSessionListing(stdout)
	}
	if err != nil {
		return false
	}
	exited, present := states[name]
	return present && !exited
}

// SessionSocketPresent reads one exact native path; it never enumerates sessions
// or infers ownership. A symlink/non-socket/unreadable path remains unknown.
func SessionSocketPresent(name string) bool {
	return (Source{}).sessionSocketPresent(name)
}

// A custom child environment must never borrow the live caller's namespace as
// retry evidence. Missing explicit/fallback roots remain an unavailable probe.
func (s Source) sessionSocketPresent(name string) bool {
	if !validSessionLookupName(name) {
		return false
	}
	getenv, tempDir := os.Getenv, os.TempDir()
	if s.Env != nil {
		env := make(map[string]string)
		for _, entry := range s.Env {
			if key, value, ok := strings.Cut(entry, "="); ok {
				env[key] = value
			}
		}
		getenv = func(key string) string { return env[key] }
		tempDir = getenv("TMPDIR")
	}
	root, err := ResolveZellijSocketRoot(getenv("ZELLIJ_SOCKET_DIR"), tempDir, getenv("XDG_RUNTIME_DIR"), runtime.GOOS, os.Getuid())
	if err != nil {
		return false
	}
	// The kernel must resolve symlink/parent segments exactly as native Zellij
	// does. filepath.Join would clean the root into a different socket identity.
	path := AppendNativeSocketPath(root, "contract_version_1/"+name)
	info, err := os.Lstat(path)
	return err == nil && info != nil && info.Mode()&os.ModeSocket != 0
}
