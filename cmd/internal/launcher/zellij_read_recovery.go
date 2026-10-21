package launcher

import (
	"errors"

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
