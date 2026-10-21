package couchcmd

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/procutil"
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
			processes := &focusProcessWorld{args: []string{"pair", "term"}}
			role, err := shortcutFocusProbeWithProcesses(a, queryShortcutClients, procutil.NewSource(processes))(context.Background(), address)
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
