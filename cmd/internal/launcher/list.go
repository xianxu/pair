package launcher

import (
	"fmt"
	"io"
	"strings"
)

// The `pair list` / `ls` subcommand (#99 M5a, ported from bin/pair-shell 228-306).
// A read-only listing of current-scope Pair zellij sessions with their resolved
// agent and attach state. The row gather (zellij queries + agent resolution +
// client counts) is a Runtime effect (ListSessions); the table + status-string
// rendering is pure.

// listStatus renders a row's STATUS column purely from its state + client count:
// an exited resurrect record, a detached (0-client) live session, or an attached
// session with its client count (shell 283-299).
func listStatus(row ListRow) string {
	switch {
	case row.State == SessionExited:
		return "exited"
	case row.Clients <= 0:
		return "detached"
	case row.Clients == 1:
		return "attached (1 client)"
	default:
		return fmt.Sprintf("attached (%d clients)", row.Clients)
	}
}

// formatListTable renders the SESSION/AGENT/STATUS table (or the empty-set line).
// Pure over the gathered rows so the formatting is unit-tested directly.
func formatListTable(rows []ListRow) string {
	if len(rows) == 0 {
		return "no pair sessions\n"
	}
	out := fmt.Sprintf("%-30s %-10s %s\n", "SESSION", "AGENT", "STATUS")
	for _, r := range rows {
		agent := r.Agent
		if agent == "" {
			agent = "?"
		}
		out += fmt.Sprintf("%-30s %-10s %s\n", r.Session, agent, listStatus(r))
	}
	return out
}

// runList drives the `list`/`ls` subcommand: gather the rows behind the Runtime
// seam, then print the pure table to stdout. A gather error goes to stderr (shell
// 230: `>&2`) so it doesn't pollute `pair list | …`. Returns the exit code.
func runList(rt Runtime, stdout, stderr io.Writer) int {
	rows, err := rt.ListSessions()
	if err != nil {
		fmt.Fprintf(stderr, "pair: %v\n", err)
		return 1
	}
	fmt.Fprint(stdout, formatListTable(rows))
	return 0
}

func buildListRowsForScope(names []string, raw string, index SessionNameIndex, scopeKey string, inferAgent func(string) string, clientCount func(string) int) []ListRow {
	rows := make([]ListRow, 0, len(names))
	for _, name := range names {
		tag := ""
		if scopeKey != "" {
			entry, ok := index.ownerOf(name)
			if !ok || entry.ScopeKey != scopeKey {
				continue
			}
			tag = entry.Tag
		}
		if tag == "" {
			tag = tagFromPublicSessionName(name)
		}
		row := ListRow{Session: name, Agent: inferAgent(tag), State: SessionDetached}
		if _, exited := sessionRowState(raw, name); exited {
			row.State = SessionExited
		} else if row.Clients = clientCount(name); row.Clients > 0 {
			row.State = SessionAttached
		}
		rows = append(rows, row)
	}
	return rows
}

func tagFromPublicSessionName(session string) string {
	if entry, ok := strings.CutPrefix(session, "pair-"); ok {
		return entry
	}
	return session
}
