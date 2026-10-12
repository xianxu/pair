package couchcmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchtty"
	"github.com/xianxu/pair/cmd/internal/reviewcmd"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

func TestNotePanesRolesFilesAndFloatingFirst(t *testing.T) {
	panes := []zellijpane.Pane{
		{ID: "1", TerminalCommand: "pair wrap claude", X: 0, Y: 0, Columns: 80, Rows: 18},
		{ID: "2", TerminalCommand: "nvim -u /p/nvim/init.lua /d/draft.md", X: 0, Y: 18, Columns: 80, Rows: 5},
		{ID: "3", IsPlugin: true, Columns: 80, Rows: 1},
		{ID: "4", TerminalCommand: "nvim -u /p/nvim/review.lua /docs/my plan.md", IsFloating: true, X: 0, Y: 0, Columns: 80, Rows: 23},
		{ID: "5", TerminalCommand: "zsh", X: 60, Y: 0, Columns: 20, Rows: 18},
	}
	files := map[string]string{"/d/draft.md": "# Draft\nhello", "/docs/my plan.md": strings.Repeat("x", maxNoteFile+1)}
	read := func(p string) ([]byte, error) {
		if s, ok := files[p]; ok {
			return []byte(s), nil
		}
		return nil, os.ErrNotExist
	}
	got := notePanes(panes, []string{"5"}, "/d/draft.md", read)
	want := []struct {
		role, file string
		lines      int
	}{
		{couchtty.PaneReview, "/docs/my plan.md", 0}, // floating first; too big to read
		{couchtty.PaneAgent, "", 0},
		{couchtty.PaneDraft, "/d/draft.md", 2},
		{couchtty.PaneTerminal, "", 0},
	}
	if len(got) != len(want) {
		t.Fatalf("panes %+v", got)
	}
	for i, w := range want {
		if got[i].Role != w.role || got[i].File != w.file || len(got[i].Lines) != w.lines {
			t.Errorf("pane %d: %s %q %d lines, want %s %q %d", i, got[i].Role, got[i].File, len(got[i].Lines), w.role, w.file, w.lines)
		}
	}
}

func TestReviewFile(t *testing.T) {
	// The command reviewcmd spawns the pane with, read back.
	spawned := strings.Join([]string{"nvim", "-u", "/home/p/" + reviewcmd.ReviewScript, "/docs/spawned.md"}, " ")
	if got := reviewFile(spawned); got != "/docs/spawned.md" {
		t.Errorf("reviewcmd's own command: %q", got)
	}
	for cmd, want := range map[string]string{
		"nvim -u /home/p/nvim/review.lua /docs/a.md":          "/docs/a.md",
		"nvim -u /home/p/nvim/review.lua /docs/with space.md": "/docs/with space.md",
		"nvim -u /home/p/nvim/init.lua /d.md":                 "",
		"":                                                    "",
	} {
		if got := reviewFile(cmd); got != want {
			t.Errorf("reviewFile(%q) = %q, want %q", cmd, got, want)
		}
	}
}

// The producer asks the slot's own session and fails visibly: a missing
// binding or a failed query is an error, never a guessed layout.
func TestPaneRectsOfAsksTheSlotsSession(t *testing.T) {
	f := focusArtifacts{t.TempDir()}
	a := couchcore.ThreadAddress{RepoScope: "0123456789abcdef", Tag: "work"}
	var asked []string
	rects := paneRectsOf(f, func(_ context.Context, session string) ([]byte, error) {
		asked = append(asked, session)
		return []byte(`[{"id":1,"is_plugin":false,"terminal_command":"pair wrap claude","pane_x":0,"pane_y":0,"pane_columns":80,"pane_rows":23}]`), nil
	}, readNoteFile)
	got, err := rects(context.Background(), a)
	if err != nil || len(got) != 1 || got[0].Role != couchtty.PaneAgent || len(asked) != 1 || asked[0] != "exact-session" {
		t.Fatalf("%+v %v asked %v", got, err, asked)
	}
	failing := paneRectsOf(f, func(context.Context, string) ([]byte, error) { return nil, errors.New("no server") }, readNoteFile)
	if _, err := failing(context.Background(), a); err == nil {
		t.Error("a failed query must be an error")
	}
	empty := paneRectsOf(f, func(context.Context, string) ([]byte, error) { return []byte("not json"), nil }, readNoteFile)
	if _, err := empty(context.Background(), a); err == nil {
		t.Error("an unreadable report must be an error")
	}
	if _, err := paneRectsOf(nil, nil, readNoteFile)(context.Background(), a); err == nil {
		t.Error("no artifacts must be an error")
	}
}

func TestReadNoteFileIsBounded(t *testing.T) {
	p := filepath.Join(t.TempDir(), "big.md")
	if err := os.WriteFile(p, []byte(strings.Repeat("y", maxNoteFile+10)), 0o600); err != nil {
		t.Fatal(err)
	}
	data, err := readNoteFile(p)
	if err != nil || len(data) != maxNoteFile+1 {
		t.Errorf("read %d bytes, %v", len(data), err)
	}
}
