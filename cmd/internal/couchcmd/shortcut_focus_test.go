package couchcmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

type focusArtifacts struct{ dir string }

func (f focusArtifacts) PairLifecycleDataDir() string { return f.dir }
func (f focusArtifacts) PairSessionContext(ctx context.Context, a couchcore.ThreadAddress) (couchcore.PairSessionBinding, error) {
	return couchcore.PairSessionBinding{Name: "exact-session", Present: true}, ctx.Err()
}

func TestFocusProbeDoesNotInferRoleFromMissingRegistration(t *testing.T) {
	for _, contents := range []string{"missing", "garbage\n", "4 invalid-pid\n"} {
		t.Run(contents, func(t *testing.T) {
			f := focusArtifacts{t.TempDir()}
			a := couchcore.ThreadAddress{RepoScope: "0123456789abcdef", Tag: "work"}
			p, err := artifactpath.Resolve(artifactpath.Address{DataDir: f.dir, RepoScope: a.RepoScope, Tag: string(a.Tag)})
			if err != nil {
				t.Fatal(err)
			}
			reg := workbenchshortcut.TerminalPaneRegistry{DataDir: p.ScopeDir(), Tag: "work"}
			if contents != "missing" {
				if err := os.MkdirAll(filepath.Dir(reg.Path()), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(reg.Path(), []byte(contents), 0600); err != nil {
					t.Fatal(err)
				}
			}
			probe := rightTerminalFocusProbe(f, func(context.Context, string) ([]byte, error) {
				return []byte("CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n1 terminal_4 pair term\n"), nil
			})
			if right, err := probe(context.Background(), a); err == nil || right {
				t.Fatalf("missing proof: right=%v err=%v", right, err)
			}
		})
	}
}

func TestFocusProbeFollowsInnerPaneWithUnchangingOuterScreen(t *testing.T) {
	f := focusArtifacts{t.TempDir()}
	address := couchcore.ThreadAddress{RepoScope: "0123456789abcdef", Tag: "work"}
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: f.dir, RepoScope: address.RepoScope, Tag: string(address.Tag)})
	if err != nil {
		t.Fatal(err)
	}
	registry := workbenchshortcut.TerminalPaneRegistry{DataDir: paths.ScopeDir(), Tag: "work"}
	if err := registry.Register("4", os.Getpid()); err != nil {
		t.Fatal(err)
	}
	// Stateful external double: focus changes while the same client stays
	// attached. Commands/titles do not define the right terminal; its registry does.
	row := "1 terminal_2 nvim -u /pair/nvim/init.lua draft.md"
	probe := rightTerminalFocusProbe(f, func(ctx context.Context, session string) ([]byte, error) {
		if session != "exact-session" {
			t.Fatalf("session = %q", session)
		}
		return []byte("CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n" + row + "\n"), ctx.Err()
	})
	for _, step := range []struct {
		row     string
		right   bool
		failure bool
	}{
		{row, false, false},
		{"1 terminal_4 nvim", true, false},
		{"1 terminal_4 zsh", true, false},
		{row, false, false},
		{"1 terminal_4 nvim\n2 terminal_2 nvim", false, true},
		{"", false, true},
		{"1 bad nvim", false, true},
		{"1 plugin_4 plugin", false, false},
	} {
		row = step.row
		right, err := probe(context.Background(), address)
		if (err != nil) != step.failure || right != step.right {
			t.Fatalf("%q: right=%v err=%v", row, right, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := probe(ctx, address); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
}
