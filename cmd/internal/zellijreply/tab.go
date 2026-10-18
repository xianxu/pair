package zellijreply

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/strictjson"
)

// Tab preserves absent legacy identity separately from native zero identity.
// Callers correlating panes and tabs must require ID and Position themselves.
type Tab struct {
	ID, Position         *int
	ActiveSwapLayoutName *string
	IsSwapLayoutDirty    bool
}

func ParseTab(data []byte) (Tab, error) {
	if err := strictjson.RejectDuplicateKeys(data); err != nil {
		return Tab{}, err
	}
	var t Tab
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil || m == nil {
		return t, fmt.Errorf("expected tab object")
	}
	s, err := stringField(m, "active_swap_layout_name", true, true)
	if err != nil {
		return t, err
	}
	if !bytes.Equal(bytes.TrimSpace(m["active_swap_layout_name"]), []byte("null")) {
		t.ActiveSwapLayoutName = &s
	}
	if t.IsSwapLayoutDirty, err = boolField(m, "is_swap_layout_dirty", true); err != nil {
		return t, err
	}
	if t.ID, err = intField(m, "tab_id", false); err != nil {
		return t, err
	}
	if t.Position, err = intField(m, "position", false); err != nil {
		return t, err
	}
	legacy, err := intField(m, "tab_position", false)
	if err != nil {
		return t, err
	}
	if legacy != nil {
		if t.Position != nil && *legacy != *t.Position {
			return t, fmt.Errorf("contradictory tab position")
		}
		if t.Position == nil {
			t.Position = legacy
		}
	}
	return t, nil
}
