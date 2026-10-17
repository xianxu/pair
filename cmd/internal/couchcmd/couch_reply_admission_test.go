package couchcmd

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
	"github.com/xianxu/pair/cmd/internal/zellijtest"
)

func installCouchReplyFixture(t *testing.T) *zellijtest.Fixture {
	t.Helper()
	f := zellijtest.New(t)
	for _, entry := range f.Env() {
		k, v, _ := strings.Cut(entry, "=")
		t.Setenv(k, v)
	}
	return f
}

type changingFocusArtifacts struct {
	focusArtifacts
	final       error
	revalidated int
}

func (a *changingFocusArtifacts) RevalidatePairSession(ctx context.Context, _ couchcore.PairSessionBinding) error {
	a.revalidated++
	return errors.Join(a.final, ctx.Err())
}

func TestCouchRestartReplyAdmission(t *testing.T) {
	for _, mode := range []string{"healthy", "wrong-first", "wrong-confirmation", "focus-change", "generation-change"} {
		t.Run(mode, func(t *testing.T) {
			f := installCouchReplyFixture(t)
			a := &changingFocusArtifacts{focusArtifacts: focusArtifacts{t.TempDir()}}
			address := couchcore.ThreadAddress{RepoScope: "0123456789abcdef", Tag: "work"}
			paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: a.dir, RepoScope: address.RepoScope, Tag: "work"})
			if err != nil {
				t.Fatal(err)
			}
			if err := (workbenchshortcut.TerminalPaneRegistry{DataDir: paths.ScopeDir(), Tag: "work"}).Register("4", os.Getpid()); err != nil {
				t.Fatal(err)
			}
			f.SetModel(zellijtest.Model{Sessions: map[string]zellijtest.Session{"exact-session": {Clients: []zellijtest.Client{{ID: 1, PaneID: "terminal_4", RunningCommand: "zsh"}}}}})
			ordinal := 2
			reply := `{"tab_id":0}`
			if mode == "wrong-first" {
				ordinal = 1
			}
			if mode == "focus-change" {
				reply = "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n1 terminal_2 nvim -u /p/nvim/init.lua draft.md\n"
			}
			if mode != "healthy" && mode != "generation-change" {
				f.SetFaults(zellijtest.Fault{Ordinal: ordinal, Reply: &reply})
			}
			if mode == "generation-change" {
				a.final = errors.New("server generation changed")
			}
			role, err := shortcutFocusProbe(a, queryShortcutClients)(context.Background(), address)
			if mode == "healthy" {
				if err != nil || role != workbenchshortcut.PaneRoleRightTerminal {
					t.Fatalf("role=%v error=%v", role, err)
				}
			} else if err == nil {
				t.Fatalf("fault %s granted role %v", mode, role)
			}
			want := 2
			if mode == "wrong-first" {
				want = 1
			}
			if len(f.Queries()) != want {
				t.Fatalf("queries=%d want=%d", len(f.Queries()), want)
			}
			wantValidation := 0
			if mode == "healthy" || mode == "generation-change" {
				wantValidation = 1
			}
			if a.revalidated != wantValidation {
				t.Fatalf("generation checks=%d want=%d", a.revalidated, wantValidation)
			}
			if len(f.Effects()) != 0 {
				t.Fatal("focus observation mutated native state")
			}
		})
	}
}

func TestCouchNoteReplyAdmission(t *testing.T) {
	for _, mode := range []string{"healthy", "wrong-first", "wrong-confirmation", "geometry-change", "tab-change", "role-change", "file-change", "missing-geometry", "generation-change"} {
		t.Run(mode, func(t *testing.T) {
			f := installCouchReplyFixture(t)
			a := &changingFocusArtifacts{focusArtifacts: focusArtifacts{t.TempDir()}}
			pane := zellijtest.DefaultPane(1, "nvim /d/draft.md")
			pane["terminal_command"] = "nvim -u /p/nvim/init.lua /d/draft.md"
			if mode == "file-change" {
				pane["terminal_command"] = "nvim -u /p/nvim/review.lua /docs/a.md"
			}
			f.SetModel(zellijtest.Model{Sessions: map[string]zellijtest.Session{"exact-session": {Panes: []zellijtest.Pane{pane}}}})
			ordinal := 2
			reply := "CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n"
			if mode == "wrong-first" {
				ordinal = 1
			}
			if mode == "geometry-change" || mode == "tab-change" || mode == "missing-geometry" || mode == "role-change" || mode == "file-change" {
				if mode == "role-change" {
					pane["terminal_command"] = "pair wrap claude"
				}
				if mode == "file-change" {
					pane["terminal_command"] = "nvim -u /p/nvim/review.lua /docs/b.md"
				}
				if mode == "geometry-change" {
					pane["pane_x"] = 5
				}
				if mode == "tab-change" {
					pane["tab_id"] = 2
				}
				if mode == "missing-geometry" {
					delete(pane, "pane_columns")
					ordinal = 1
				}
				raw, e := json.Marshal([]zellijtest.Pane{pane})
				if e != nil {
					t.Fatal(e)
				}
				reply = string(raw)
			}
			if mode != "healthy" && mode != "generation-change" {
				f.SetFaults(zellijtest.Fault{Ordinal: ordinal, Reply: &reply})
			}
			if mode == "generation-change" {
				a.final = errors.New("server generation changed")
			}
			reads := 0
			got, err := paneRectsOf(a, queryPaneGeometry, func(string) ([]byte, error) { reads++; return []byte("draft"), nil })(context.Background(), couchcore.ThreadAddress{RepoScope: "0123456789abcdef", Tag: "work"})
			if mode == "healthy" {
				if err != nil || len(got) != 1 || reads != 1 {
					t.Fatalf("panes=%v reads=%d error=%v", got, reads, err)
				}
			} else if err == nil || reads != 0 {
				t.Fatalf("fault %s read files or admitted: panes=%v reads=%d error=%v", mode, got, reads, err)
			}
			wantValidation := 0
			if mode == "healthy" || mode == "generation-change" {
				wantValidation = 1
			}
			if a.revalidated != wantValidation {
				t.Fatalf("generation checks=%d want=%d", a.revalidated, wantValidation)
			}
			if len(f.Effects()) != 0 {
				t.Fatal("note observation mutated native state")
			}
		})
	}
}
