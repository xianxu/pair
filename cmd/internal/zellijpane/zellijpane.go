// Package zellijpane parses `zellij action list-panes --json` into a flat pane
// list, so the recursive-descent walk that several pair helpers need (the
// copy-on-select focused-pane pick, the clipboard-to-pane nvim-pane pick, and
// opener's agent-pane pick) lives in one place rather than being re-open-coded
// per consumer (ARCH-DRY, #93 M4). Callers filter the []Pane by predicate.
package zellijpane

import (
	"encoding/json"
	"sort"
	"strconv"
)

// Pane is the subset of a zellij pane manifest the pair helpers key off. Title
// supports title-keyed role fallbacks, and geometry supports workbench layout
// decisions when callers request `list-panes --geometry`.
type Pane struct {
	ID              string
	TerminalCommand string
	Title           string
	IsFocused       bool
	IsPlugin        bool
	IsFloating      bool
	X               int
	Y               int
	Columns         int
	Rows            int
}

// Parse decodes the `list-panes --json` output and returns every pane object it
// finds, in a deterministic depth-first order (map children visited in
// sorted-key order — jq's `..` uses document order, but that only matters when
// >1 pane matches a predicate, which Pair's one-pane-per-role invariant rules out; the
// selectors in clipcmd pick a unique pane). Invalid JSON yields nil.
func Parse(data []byte) []Pane {
	var root interface{}
	if json.Unmarshal(data, &root) != nil {
		return nil
	}
	var out []Pane
	walk(root, &out)
	return out
}

func walk(v interface{}, out *[]Pane) {
	switch t := v.(type) {
	case map[string]interface{}:
		if p, ok := paneFrom(t); ok {
			*out = append(*out, p)
		}
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			walk(t[k], out)
		}
	case []interface{}:
		for _, child := range t {
			walk(child, out)
		}
	}
}

// paneFrom treats an object as a pane iff it carries an id (`.id // .pane_id`,
// what the shell selectors extract) together with at least one pane-defining
// marker — so tab/layout wrapper objects (which have neither a terminal_command
// nor the is_* flags) are skipped. This accepts panes from both the `--command`
// and bare `list-panes --json` shapes (the flag varies across callers).
func paneFrom(m map[string]interface{}) (Pane, bool) {
	id := idString(m["id"])
	if id == "" {
		id = idString(m["pane_id"])
	}
	if id == "" {
		return Pane{}, false
	}
	tc, hasTC := m["terminal_command"].(string)
	focused, hasFocused := m["is_focused"].(bool)
	plugin, hasPlugin := m["is_plugin"].(bool)
	floating, hasFloating := m["is_floating"].(bool)
	if !hasTC && !hasFocused && !hasPlugin && !hasFloating {
		return Pane{}, false
	}
	title, _ := m["title"].(string)
	return Pane{
		ID:              id,
		TerminalCommand: tc,
		Title:           title,
		IsFocused:       focused,
		IsPlugin:        plugin,
		IsFloating:      floating,
		X:               intNumber(m["pane_x"]),
		Y:               intNumber(m["pane_y"]),
		Columns:         intNumber(m["pane_columns"]),
		Rows:            intNumber(m["pane_rows"]),
	}, true
}

// idString renders a pane id, which zellij emits as either a JSON number or a
// string, into the bare string form the `zellij action` commands accept.
func idString(v interface{}) string {
	switch n := v.(type) {
	case float64:
		return strconv.Itoa(int(n))
	case string:
		return n
	}
	return ""
}

func intNumber(v interface{}) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return 0
	}
}
