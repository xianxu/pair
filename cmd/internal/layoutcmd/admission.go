package layoutcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
	"github.com/xianxu/pair/cmd/internal/zellijreply"
)

// ConfirmWorkbenchPanesContext binds the decision to the independently known invoking
// terminal. Focus flags on other tabs are valid and do not select this context.
func ConfirmWorkbenchPanesContext(ctx context.Context, rt interface {
	ListPanesJSONContext(context.Context) ([]byte, error)
}, caller string, validate ...func([]zellijpane.Pane) error) ([]zellijpane.Pane, int, int, error) {
	return confirmWorkbenchPanesContext(ctx, rt, caller, nil, validate...)
}

func confirmWorkbenchPanesContext(ctx context.Context, rt interface {
	ListPanesJSONContext(context.Context) ([]byte, error)
}, caller string, needGeometry func([]zellijpane.Pane) bool, validate ...func([]zellijpane.Pane) error) ([]zellijpane.Pane, int, int, error) {
	var chosen []zellijpane.Pane
	var tabID, position int
	_, err := zellijreply.Confirm(ctx, func(ctx context.Context) ([]byte, error) { return rt.ListPanesJSONContext(ctx) }, func(raw []byte) (string, error) {
		panes, err := zellijreply.ParsePanes(raw, zellijreply.PaneRequirements{Command: true, State: true, Geometry: needGeometry == nil, Tab: true})
		if err != nil {
			return "", err
		}
		invoking, ok := findFullscreenPane(panes, caller)
		if !ok {
			return "", fmt.Errorf("invoking terminal absent")
		}
		tabID, position = *invoking.TabID, *invoking.TabPosition
		chosen = nil
		for _, p := range panes {
			if *p.TabID == tabID {
				if *p.TabPosition != position {
					return "", fmt.Errorf("contradictory tab position")
				}
				chosen = append(chosen, p)
			}
		}
		for _, check := range validate {
			if err := check(chosen); err != nil {
				return "", err
			}
		}
		projection := append([]zellijpane.Pane(nil), chosen...)
		geometry := needGeometry == nil || needGeometry(chosen)
		if geometry && needGeometry != nil {
			if _, err := zellijreply.ParsePanes(raw, zellijreply.PaneRequirements{Command: true, State: true, Geometry: true, Tab: true}); err != nil {
				return "", err
			}
		}
		for i := range projection {
			// Native foreground command/cwd are not layout decision facts.
			projection[i].PaneCommand = ""
			projection[i].PaneCwd = ""
			if !geometry {
				projection[i].X = 0
				projection[i].Y = 0
				projection[i].Columns = 0
				projection[i].Rows = 0
			}
		}
		projected, err := json.Marshal(projection)
		return string(projected), err
	})
	return chosen, tabID, position, err
}

func confirmLayoutTab(ctx context.Context, rt FullscreenRuntime, id, position int) (zellijreply.Tab, error) {
	var tab zellijreply.Tab
	_, err := zellijreply.Confirm(ctx, func(ctx context.Context) ([]byte, error) { return rt.CurrentTabJSONContext(ctx) }, func(raw []byte) (string, error) {
		var err error
		tab, err = zellijreply.ParseTab(raw)
		if err != nil {
			return "", err
		}
		if tab.ID == nil || tab.Position == nil || *tab.ID != id || *tab.Position != position {
			return "", fmt.Errorf("current tab does not match invoking terminal")
		}
		b, err := json.Marshal(tab)
		return string(b), err
	})
	return tab, err
}
