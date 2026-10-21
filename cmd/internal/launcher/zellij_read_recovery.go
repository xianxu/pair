package launcher

import (
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
var errZellijCrossedReply = errors.New("Zellij query received another reply kind")

type zellijReplyKind uint8

const (
	zellijUnknownReply zellijReplyKind = iota
	zellijSessionsReply
	zellijClientsReply
	zellijPanesReply
	zellijTabReply
)

func zellijReadKind(args []string) zellijReplyKind {
	if len(args) == 2 && args[0] == "list-sessions" && args[1] == "--no-formatting" {
		return zellijSessionsReply
	}
	if len(args) < 4 || args[0] != "--session" || args[1] == "" || args[2] != "action" {
		return zellijUnknownReply
	}
	if len(args) == 4 && args[3] == "list-clients" {
		return zellijClientsReply
	}
	if (len(args) == 6 || len(args) == 7 && args[6] == "--state") && args[3] == "list-panes" && args[4] == "--json" && args[5] == "--command" {
		return zellijPanesReply
	}
	return zellijUnknownReply
}

// Malformed same-kind data is left to existing strict caller validation. A
// replay requires a complete validated different kind, not an arbitrary parse
// error. Same-kind misrouting still requires the existing independent agreement.
func crossedZellijReply(want zellijReplyKind, raw []byte) bool {
	if want == 0 {
		return false
	}
	valid := []bool{false, validateSessionListing(raw, false) == nil, false, false, false}
	_, err := zellijreply.ParseClients(raw)
	valid[zellijClientsReply] = err == nil
	_, err = zellijreply.ParsePanes(raw, zellijreply.PaneRequirements{})
	valid[zellijPanesReply] = err == nil
	_, err = zellijreply.ParseTab(raw)
	valid[zellijTabReply] = err == nil
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

// errZellijSessionLookupMiss is read-retry eligibility, never session authority.
var errZellijSessionLookupMiss = errors.New("Zellij query falsely missed a live session")

func validSessionLookupName(name string) bool {
	return utf8.ValidString(name) && strings.TrimSpace(name) != "" && name != "." && name != ".." && filepath.Base(name) == name && strings.IndexFunc(name, unicode.IsControl) < 0
}

// retryableSessionLookupMiss admits only the two observed native lookup failures.
// The caller separately guarantees successful collection (or a direct exit),
// while the existing read allowlist/deadline/attempt budget owns any reissue.
func retryableSessionLookupMiss(args []string, stdout, stderr []byte, exit int, socketPresent bool) bool {
	kind := zellijReadKind(args)
	if (kind != zellijClientsReply && kind != zellijPanesReply) || len(args) < 2 || !validSessionLookupName(args[1]) {
		return false
	}
	name := args[1]
	if exit == 1 {
		return len(stdout) == 0 && string(stderr) == "There is no active session!\n" && socketPresent
	}
	if exit != 0 || string(stderr) != fmt.Sprintf("Session '%s' not found. The following sessions are active:\n", name) {
		return false
	}
	states, err := sessionListingStates(stdout)
	if err != nil {
		states, err = parseColoredSessionListing(stdout)
	}
	if err != nil {
		return false
	}
	exited, present := states[name]
	return present && !exited
}

// sessionSocketPresent reads one exact native path; it never enumerates sessions
// or infers ownership. A symlink/non-socket/unreadable path remains unknown.
func sessionSocketPresent(name string) bool {
	if !validSessionLookupName(name) {
		return false
	}
	root, err := ResolveZellijSocketRoot(os.Getenv("ZELLIJ_SOCKET_DIR"), os.TempDir(), os.Getenv("XDG_RUNTIME_DIR"), runtime.GOOS, os.Getuid())
	if err != nil {
		return false
	}
	// The kernel must resolve symlink/parent segments exactly as native Zellij
	// does. filepath.Join would clean the root into a different socket identity.
	path := root
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	path += "contract_version_1/" + name
	return ObserveSocket(os.Lstat(path)) == SocketPresent
}
