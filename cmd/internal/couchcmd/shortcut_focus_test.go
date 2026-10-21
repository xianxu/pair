package couchcmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/procutil"
	"github.com/xianxu/pair/cmd/internal/workbenchshortcut"
)

type focusArtifacts struct{ dir string }

type focusProcessWorld struct {
	args        []string
	unavailable bool
}

func (*focusProcessWorld) Table() ([]procutil.Process, error) {
	return nil, errors.New("unexpected whole-host scan")
}
func (w *focusProcessWorld) Command(pid int) (procutil.CommandInfo, error) {
	if pid != os.Getpid() {
		return procutil.CommandInfo{}, os.ErrNotExist
	}
	if w.unavailable {
		return procutil.CommandInfo{}, os.ErrPermission
	}
	return procutil.CommandInfo{Args: append([]string(nil), w.args...)}, nil
}
func (*focusProcessWorld) Identity(pid int) (string, error) {
	if pid != os.Getpid() {
		return "", os.ErrNotExist
	}
	return "registered-owner", nil
}

func (f focusArtifacts) PairLifecycleDataDir() string { return f.dir }
func (f focusArtifacts) RevalidatePairSession(ctx context.Context, _ couchcore.PairSessionBinding) error {
	return ctx.Err()
}
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
			probe := shortcutFocusProbe(f, func(context.Context, string) ([]byte, error) {
				return []byte("CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n1 terminal_4 pair term\n"), nil
			})
			if right, err := probe(context.Background(), a); err == nil || right != workbenchshortcut.PaneRoleOther {
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
	processes := &focusProcessWorld{args: []string{"pair", "term"}}
	probe := shortcutFocusProbeWithProcesses(f, func(ctx context.Context, session string) ([]byte, error) {
		if session != "exact-session" {
			t.Fatalf("session = %q", session)
		}
		return []byte("CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n" + row + "\n"), ctx.Err()
	}, procutil.NewSource(processes))
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
		if (err != nil) != step.failure || (right == workbenchshortcut.PaneRoleRightTerminal) != step.right {
			t.Fatalf("%q: right=%v err=%v", row, right, err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := probe(ctx, address); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
}

func TestFocusProbeDistinguishesShellFromUnavailableOwner(t *testing.T) {
	f := focusArtifacts{t.TempDir()}
	address := couchcore.ThreadAddress{RepoScope: "0123456789abcdef", Tag: "work"}
	paths, err := artifactpath.Resolve(artifactpath.Address{DataDir: f.dir, RepoScope: address.RepoScope, Tag: "work"})
	if err != nil {
		t.Fatal(err)
	}
	reg := workbenchshortcut.TerminalPaneRegistry{DataDir: paths.ScopeDir(), Tag: "work"}
	if err := reg.Register("4", os.Getpid()); err != nil {
		t.Fatal(err)
	}
	world := &focusProcessWorld{args: []string{"pair", "term"}}
	probe := shortcutFocusProbeWithProcesses(f, func(context.Context, string) ([]byte, error) {
		return []byte("CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n1 terminal_4 zsh\n"), nil
	}, procutil.NewSource(world))
	for _, step := range []struct {
		args                   []string
		unavailable, wantRight bool
	}{
		{[]string{"pair", "term"}, false, true},
		{[]string{"sh", "-i"}, false, false},
		{[]string{"sh", "-i"}, true, true},
		{[]string{"sh", "-i"}, false, false},
	} {
		world.args, world.unavailable = step.args, step.unavailable
		role, err := probe(context.Background(), address)
		if (role == workbenchshortcut.PaneRoleRightTerminal) != step.wantRight || (err == nil) != step.wantRight {
			t.Fatalf("step=%+v role=%v err=%v", step, role, err)
		}
	}
	data, err := os.ReadFile(paths.AdaptLog())
	if err != nil || !strings.Contains(string(data), "argv-unavailable=1") {
		t.Fatalf("missing scoped unavailable reason: %s %v", data, err)
	}
}

func TestFocusProbeRecognizesReviewCommand(t *testing.T) {
	f := focusArtifacts{t.TempDir()}
	row := ""
	probe := shortcutFocusProbe(f, func(context.Context, string) ([]byte, error) {
		return []byte("CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n" + row + "\n"), nil
	})
	address := couchcore.ThreadAddress{RepoScope: "0123456789abcdef", Tag: "work"}
	for _, step := range []struct {
		command string
		role    workbenchshortcut.PaneRole
		failure bool
	}{
		{"nvim -u /pair/nvim/review.lua /tmp/review.md", workbenchshortcut.PaneRoleReview, false},
		{"nvim /tmp/review.md", workbenchshortcut.PaneRoleOther, true},
		{"nvim -u /pair/nvim/init.lua draft.md", workbenchshortcut.PaneRoleLeftDraft, false},
		{"pair wrap --agent codex", workbenchshortcut.PaneRoleLeftAgent, false},
	} {
		row = "1 terminal_9 " + step.command
		role, err := probe(context.Background(), address)
		if role != step.role || (err != nil) != step.failure {
			t.Fatalf("%q: role=%v err=%v", step.command, role, err)
		}
	}
}
