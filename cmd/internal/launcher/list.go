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
	out := padDisplay("SESSION", 30) + " " + padDisplay("AGENT", 10) + " STATUS\n"
	for _, r := range rows {
		agent := r.Agent
		if agent == "" {
			agent = "?"
		}
		out += padDisplay(r.Session, 30) + " " + padDisplay(agent, 10) + " " + listStatus(r) + "\n"
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
			tag, _ = TagForSessionName(index, name)
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

// padDisplay left-aligns s in a field of w TERMINAL COLUMNS.
//
// Go's %-30s pads by rune count, which the 📁 prefix breaks (#130): it is one
// rune but two columns wide, so every new-format row would sit a column left of
// its header. Same unit confusion as the rune-vs-byte truncation bug this issue
// fixes, one layer up — three units are in play (bytes for zellij's socket
// budget, runes for Go's string ops, columns for the terminal) and each belongs
// to a different question.
func padDisplay(s string, w int) string {
	if pad := w - displayWidth(s); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}

// displayWidth counts terminal columns. Only the wide case that actually occurs
// here is modeled: emoji and other East-Asian-Wide runes take two columns.
func displayWidth(s string) int {
	w := 0
	for _, r := range s {
		if isWideRune(r) {
			w += 2
			continue
		}
		w++
	}
	return w
}

// isWideRune covers the double-width ranges a session name can contain — the
// emoji planes plus the CJK/symbol blocks that render wide in a terminal.
func isWideRune(r rune) bool {
	switch {
	case r >= 0x1100 && r <= 0x115F, // Hangul Jamo
		r >= 0x2E80 && r <= 0xA4CF, // CJK radicals .. Yi
		r >= 0xAC00 && r <= 0xD7A3, // Hangul syllables
		r >= 0xF900 && r <= 0xFAFF, // CJK compatibility ideographs
		r >= 0xFE30 && r <= 0xFE6F, // CJK compatibility forms
		r >= 0xFF00 && r <= 0xFF60, // fullwidth forms
		r >= 0xFFE0 && r <= 0xFFE6,
		r >= 0x1F300 && r <= 0x1F64F, // misc symbols + pictographs, emoticons
		r >= 0x1F900 && r <= 0x1F9FF, // supplemental symbols + pictographs
		r >= 0x20000 && r <= 0x3FFFD:
		return true
	}
	return false
}
