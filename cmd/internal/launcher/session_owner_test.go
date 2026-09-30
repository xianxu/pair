package launcher

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/artifactpath"
	"github.com/xianxu/pair/cmd/internal/zellijpane"
)

func TestClassifySessionOwnerUsesActualExactArguments(t *testing.T) {
	root := "/tmp/Pair data"
	scope := "0123456789abcdef"
	owner, _ := artifactpath.NewStorageOwner(root, scope, "1-repo-1")
	p, _ := artifactpath.Resolve(artifactpath.Address{DataDir: root, RepoScope: scope, Tag: owner.Tag})
	foreign, _ := artifactpath.Resolve(artifactpath.Address{DataDir: root, RepoScope: scope, Tag: "1-repo-10"})
	for _, tc := range []struct {
		name  string
		panes []zellijpane.Pane
		want  SessionOwnerState
	}{
		{"draft", []zellijpane.Pane{{PaneCommand: "nvim '" + p.Draft() + "'"}}, SessionOwnerOwned},
		{"wrapper", []zellijpane.Pane{{PaneCommand: "/bin/pair-wrap --scrollback-log '" + p.ScrollbackRaw("codex") + "' codex"}}, SessionOwnerOwned},
		{"multicall wrapper", []zellijpane.Pane{{PaneCommand: "/path/pair wrap --scrollback-log '" + p.ScrollbackRaw("codex") + "' codex"}}, SessionOwnerOwned},
		{"agent argument is not wrapper flag", []zellijpane.Pane{{PaneCommand: "/path/pair wrap codex --scrollback-log '" + p.ScrollbackRaw("codex") + "'"}}, SessionOwnerUnknown},
		{"prefix foreign", []zellijpane.Pane{{PaneCommand: "nvim '" + foreign.Draft() + "'"}}, SessionOwnerForeign},
		{"template is not evidence", []zellijpane.Pane{{TerminalCommand: "nvim '" + p.Draft() + "'"}}, SessionOwnerUnknown},
		{"echo is not editor", []zellijpane.Pane{{PaneCommand: "echo '" + p.Draft() + "'"}}, SessionOwnerUnknown},
		{"editor config is not draft", []zellijpane.Pane{{PaneCommand: "nvim -u '" + p.Draft() + "' /tmp/other.md"}}, SessionOwnerUnknown},
		{"argument prefix", []zellijpane.Pane{{PaneCommand: "nvim '" + p.Draft() + ".bak'"}}, SessionOwnerUnknown},
		{"unclosed quote", []zellijpane.Pane{{PaneCommand: "nvim '" + p.Draft()}}, SessionOwnerUnknown},
		{"double quote preserves non-special escape", []zellijpane.Pane{{PaneCommand: "nvim \"" + strings.Replace(p.Draft(), "Pair", "\\Pair", 1) + "\""}}, SessionOwnerUnknown},
		{"conflicting owners", []zellijpane.Pane{{PaneCommand: "nvim '" + p.Draft() + "'"}, {PaneCommand: "nvim '" + foreign.Draft() + "'"}}, SessionOwnerUnknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifySessionOwner(owner, tc.panes)
			if got.State != tc.want {
				t.Fatalf("got %+v want %v", got, tc.want)
			}
		})
	}
}

type ownerWorld struct {
	generation    SessionServerIdentity
	present       bool
	panes         []byte
	changeOnQuery bool
}

func (w *ownerWorld) SessionServers(context.Context, string) ([]SessionServerIdentity, error) {
	if !w.present {
		return nil, nil
	}
	return []SessionServerIdentity{w.generation}, nil
}
func (w *ownerWorld) SessionPresent(context.Context, string) (bool, error) { return w.present, nil }
func (w *ownerWorld) SessionPanes(context.Context, string) ([]byte, error) {
	if w.changeOnQuery {
		w.generation.Identity = "replaced"
	}
	return w.panes, nil
}

func TestSessionOwnerProbeRevalidatesServerGeneration(t *testing.T) {
	w := &ownerWorld{present: true, generation: SessionServerIdentity{PID: 42, Identity: "first", Session: "📁1-2"}, panes: []byte(`[{"id":1,"pane_command":"nvim /tmp/data/repos/0123456789abcdef/draft-1-repo-1.md"}]`)}
	p := SessionOwnerProbe{IO: w}
	got, err := p.Probe(context.Background(), "📁1-2", "/tmp/data", "0123456789abcdef", "1-repo-1")
	if err != nil || got.State != SessionOwnerOwned {
		t.Fatalf("%+v %v", got, err)
	}
	w.generation.Identity = "second"
	if err := p.Revalidate(context.Background(), got); err == nil {
		t.Fatal("reused PID accepted")
	}
	w.changeOnQuery = true
	got, err = p.Probe(context.Background(), "📁1-2", "/tmp/data", "0123456789abcdef", "1-repo-1")
	if err == nil && got.State != SessionOwnerUnknown {
		t.Fatalf("changing server accepted: %+v", got)
	}
	w.present = false
	got, err = p.Probe(context.Background(), "📁1-2", "/tmp/data", "0123456789abcdef", "1-repo-1")
	if err != nil || got.State != SessionOwnerAbsent {
		t.Fatalf("absent: %+v %v", got, err)
	}
}

func TestSessionOwnerPresenceIgnoresExitedRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "zellij")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nprintf '%s\\n' '📁1-2 [Created 1h ago] (EXITED - attach to resurrect)'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", filepath.Dir(path)+string(os.PathListSeparator)+os.Getenv("PATH"))
	present, err := (osSessionOwnerIO{}).SessionPresent(context.Background(), "📁1-2")
	if err != nil || present {
		t.Fatalf("exited record is live: %v %v", present, err)
	}
}
