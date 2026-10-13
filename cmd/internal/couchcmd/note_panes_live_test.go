package couchcmd

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/pairlifecycletest"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

// The live check behind notePanes (#488): a real zellij reports each pane's
// geometry in the client's cells, in the shape the note anchor relies on
// (an agent pane above a draft pane, both inside the 80×24 client), and
// the role classifier recognizes them from their commands. Run on zellij
// upgrades:
//
//	PAIR_LIVE_ZELLIJ=1 go test ./cmd/internal/couchcmd -run '^TestNotePanesZellijConformance$' -count=1
func TestNotePanesZellijConformance(t *testing.T) {
	if os.Getenv("PAIR_LIVE_ZELLIJ") != "1" {
		t.Skip("set PAIR_LIVE_ZELLIJ=1 for a disposable zellij session")
	}
	base := t.TempDir()
	t.Setenv("TMPDIR", "/tmp") // Darwin's Unix socket pathname limit.
	for _, key := range []string{"HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "XDG_RUNTIME_DIR"} {
		t.Setenv(key, t.TempDir())
	}
	t.Setenv("SHELL", "/bin/sh")
	for _, key := range []string{"ZELLIJ", "ZELLIJ_SESSION_NAME", "ZELLIJ_PANE_ID"} {
		os.Unsetenv(key)
	}
	config := filepath.Join(base, "config.kdl")
	if err := os.WriteFile(config, []byte("show_startup_tips false\nshow_release_notes false\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Commands that classify as Pair's agent and draft without running them.
	layout := filepath.Join(base, "layout.kdl")
	body := `layout {
  pane name="agent" command="/bin/sh" { args "-c" "sleep 600" "pair wrap"; }
  pane name="draft" size=6 borderless=true command="/bin/sh" { args "-c" "sleep 600" "/x/nvim/init.lua"; }
}
`
	if err := os.WriteFile(layout, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	var entropy [8]byte
	rand.Read(entropy[:])
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	fixture, err := pairlifecycletest.StartControlledZellijWithOptions(ctx, "pair-notes-"+hex.EncodeToString(entropy[:]), pairlifecycletest.ZellijOptions{ConfigFile: config, LayoutFile: layout})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = fixture.Close() })

	var agent, draft *couchtty.PaneRect
	for agent == nil || draft == nil {
		raw, err := queryPaneGeometry(ctx, fixture.Session)
		if err != nil {
			t.Fatalf("list-panes: %v", err)
		}
		var mine []zellijpane.Pane
		for _, p := range zellijpane.Parse(raw) {
			if p.Title == "agent" || p.Title == "draft" {
				mine = append(mine, p) // the layout's tab, not the session's first
			}
		}
		for _, r := range notePanes(mine, nil, "/x/draft.md", func(string) ([]byte, error) { return []byte("x"), nil }) {
			switch r.Role {
			case couchtty.PaneAgent:
				agent = &r
			case couchtty.PaneDraft:
				draft = &r
			}
		}
		if agent == nil || draft == nil {
			select {
			case <-ctx.Done():
				t.Fatalf("the layout's panes never classified: %s", raw)
			case <-time.After(100 * time.Millisecond):
			}
		}
	}
	for _, r := range []*couchtty.PaneRect{agent, draft} {
		if r.X < 0 || r.Y < 0 || r.W <= 0 || r.H <= 0 || r.X+r.W > 80 || r.Y+r.H > 24 {
			t.Errorf("%s pane %+v is not inside the 80×24 client", r.Role, *r)
		}
	}
	t.Logf("agent %+v, draft at %d,%d %dx%d", *agent, draft.X, draft.Y, draft.W, draft.H)
	if draft.Y < agent.Y+agent.H-1 || draft.File != "/x/draft.md" || len(draft.Lines) != 1 {
		t.Errorf("draft %+v must sit below agent %+v with its file", *draft, *agent)
	}
}
