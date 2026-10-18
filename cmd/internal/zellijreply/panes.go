package zellijreply

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/strictjson"
	"sort"
	"strconv"
	"strings"

	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

// PaneRequirements names decision fields. State is a shorthand for all three
// state flags; individual flags avoid requiring unrelated facts for a focus read.
// ActualCommand is intentionally separate from the original terminal template.
type PaneRequirements struct{ Command, ActualCommand, State, Geometry, Tab, Focus, Floating, Fullscreen bool }

func ParsePanes(data []byte, req PaneRequirements) ([]zellijpane.Pane, error) {
	if err := strictjson.RejectDuplicateKeys(data); err != nil {
		return nil, err
	}
	var root json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("invalid pane JSON: %w", err)
	}
	root = bytes.TrimSpace(root)
	var rows []json.RawMessage
	switch {
	case len(root) > 0 && root[0] == '[':
		if err := json.Unmarshal(root, &rows); err != nil {
			return nil, err
		}
	case len(root) > 0 && root[0] == '{':
		var tabs map[string]json.RawMessage
		if err := json.Unmarshal(root, &tabs); err != nil {
			return nil, err
		}
		if len(tabs) == 0 {
			return nil, fmt.Errorf("missing pane array")
		}
		keys := make([]string, 0, len(tabs))
		for k := range tabs {
			if _, err := decimalID(k); err != nil {
				return nil, fmt.Errorf("unrecognized pane wrapper")
			}
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			b := bytes.TrimSpace(tabs[k])
			if len(b) == 0 || b[0] != '[' {
				return nil, fmt.Errorf("invalid legacy tab array")
			}
			var group []json.RawMessage
			if err := json.Unmarshal(b, &group); err != nil {
				return nil, err
			}
			rows = append(rows, group...)
		}
	default:
		return nil, fmt.Errorf("expected pane array")
	}
	out := make([]zellijpane.Pane, 0, len(rows))
	seen := map[string]bool{}
	for _, row := range rows {
		p, err := parsePane(row, req)
		if err != nil {
			return nil, err
		}
		key := strconv.FormatBool(p.IsPlugin) + ":" + p.ID
		if seen[key] {
			return nil, fmt.Errorf("duplicate pane identity")
		}
		seen[key] = true
		out = append(out, p)
	}
	return out, nil
}

func parsePane(row json.RawMessage, req PaneRequirements) (zellijpane.Pane, error) {
	var p zellijpane.Pane
	var m map[string]json.RawMessage
	if err := json.Unmarshal(row, &m); err != nil || m == nil {
		return p, fmt.Errorf("expected pane object")
	}
	plugin, err := boolField(m, "is_plugin", true)
	if err != nil {
		return p, err
	}
	p.IsPlugin = plugin
	found := false
	var id int
	for _, key := range []string{"id", "pane_id"} {
		b, ok := m[key]
		if !ok {
			continue
		}
		n, e := identity(b, plugin)
		if e != nil {
			return p, e
		}
		if found && n != id {
			return p, fmt.Errorf("contradictory pane identities")
		}
		found = true
		id = n
	}
	if !found {
		return p, fmt.Errorf("missing pane identity")
	}
	p.ID = strconv.Itoa(id)
	if p.TerminalCommand, err = stringField(m, "terminal_command", req.Command, true); err != nil {
		return p, err
	}
	if p.PaneCommand, err = stringField(m, "pane_command", req.ActualCommand, false); err != nil {
		return p, err
	}
	if req.ActualCommand && strings.TrimSpace(p.PaneCommand) == "" {
		return p, fmt.Errorf("missing actual pane command")
	}
	if p.PaneCwd, err = stringField(m, "pane_cwd", false, false); err != nil {
		return p, err
	}
	if p.Title, err = stringField(m, "title", false, false); err != nil {
		return p, err
	}
	if p.IsFocused, err = boolField(m, "is_focused", req.State || req.Focus); err != nil {
		return p, err
	}
	if p.IsFloating, err = boolField(m, "is_floating", req.State || req.Floating); err != nil {
		return p, err
	}
	if _, ok := m["is_fullscreen"]; ok || req.State || req.Fullscreen {
		v, e := boolField(m, "is_fullscreen", true)
		if e != nil {
			return p, e
		}
		p.IsFullscreen = &v
	}
	for _, f := range []struct {
		name     string
		dst      *int
		positive bool
	}{{"pane_x", &p.X, false}, {"pane_y", &p.Y, false}, {"pane_columns", &p.Columns, true}, {"pane_rows", &p.Rows, true}} {
		v, e := intField(m, f.name, req.Geometry)
		if e != nil {
			return p, e
		}
		if v != nil {
			if f.positive && *v == 0 {
				return p, fmt.Errorf("invalid pane size")
			}
			*f.dst = *v
		}
	}
	if p.TabID, err = intField(m, "tab_id", req.Tab); err != nil {
		return p, err
	}
	if p.TabPosition, err = intField(m, "tab_position", req.Tab); err != nil {
		return p, err
	}
	return p, nil
}
func identity(b json.RawMessage, plugin bool) (int, error) {
	var s string
	if json.Unmarshal(b, &s) == nil {
		if strings.Contains(s, "_") {
			n, role, e := qualifiedPaneID(s)
			if e != nil || role != plugin {
				return 0, fmt.Errorf("contradictory pane role")
			}
			return n, nil
		}
		return decimalID(s)
	}
	var n int
	if err := json.Unmarshal(b, &n); err != nil || bytes.Equal(bytes.TrimSpace(b), []byte("null")) || n < 0 {
		return 0, fmt.Errorf("invalid pane identity")
	}
	return n, nil
}
func boolField(m map[string]json.RawMessage, key string, required bool) (bool, error) {
	b, ok := m[key]
	if !ok {
		if required {
			return false, fmt.Errorf("missing %s", key)
		}
		return false, nil
	}
	var v bool
	if bytes.Equal(bytes.TrimSpace(b), []byte("null")) || json.Unmarshal(b, &v) != nil {
		return false, fmt.Errorf("invalid %s", key)
	}
	return v, nil
}
func stringField(m map[string]json.RawMessage, key string, required, nullable bool) (string, error) {
	b, ok := m[key]
	if !ok {
		if required {
			return "", fmt.Errorf("missing %s", key)
		}
		return "", nil
	}
	if bytes.Equal(bytes.TrimSpace(b), []byte("null")) {
		if nullable {
			return "", nil
		}
		return "", fmt.Errorf("invalid %s", key)
	}
	var v string
	if json.Unmarshal(b, &v) != nil {
		return "", fmt.Errorf("invalid %s", key)
	}
	return v, nil
}
func intField(m map[string]json.RawMessage, key string, required bool) (*int, error) {
	b, ok := m[key]
	if !ok {
		if required {
			return nil, fmt.Errorf("missing %s", key)
		}
		return nil, nil
	}
	var v int
	if bytes.Equal(bytes.TrimSpace(b), []byte("null")) || json.Unmarshal(b, &v) != nil || v < 0 {
		return nil, fmt.Errorf("invalid %s", key)
	}
	return &v, nil
}
