package launcher

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

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
	panesErr      error
	panesCalls    int
	changeOnQuery bool
	// sockets is the socket table; a path it doesn't name reads present, so
	// the older cases keep their reachable server.
	sockets map[string]SocketState
}

func (w *ownerWorld) Socket(path string) SocketState {
	if state, ok := w.sockets[path]; ok {
		return state
	}
	return SocketPresent
}

func (w *ownerWorld) SessionServers(context.Context, string) ([]SessionServerIdentity, error) {
	if !w.present {
		return nil, nil
	}
	return []SessionServerIdentity{w.generation}, nil
}
func (w *ownerWorld) SessionPresent(context.Context, string) (bool, error) { return w.present, nil }
func (w *ownerWorld) SessionPanes(context.Context, string) ([]byte, error) {
	w.panesCalls++
	if w.changeOnQuery {
		w.generation.Identity = "replaced"
	}
	return w.panes, w.panesErr
}

// A server alive with its socket gone is an orphan, reported as a state and
// never an error (#399). Strategy: socket {present, gone, unknown} x list-panes
// {ok, error}. Only gone yields orphaned, and an orphan is never asked for panes
// (it has no socket); unknown is unknown; present keeps today's behaviour.
func TestSessionOwnerProbeNamesAnOrphanedServer(t *testing.T) {
	panes := []byte(`[{"id":1,"pane_command":"nvim /tmp/data/repos/0123456789abcdef/draft-1-repo-1.md"}]`)
	server := SessionServerIdentity{PID: 7, Identity: "t7", Session: "📁1-37", Socket: "/T/zellij-501/contract_version_1/📁1-37"}
	for _, tc := range []struct {
		socket    SocketState
		panesErr  error
		wantState SessionOwnerState
		wantErr   bool
	}{
		{SocketGone, nil, SessionOwnerOrphaned, false},
		{SocketGone, errors.New("exit status 1"), SessionOwnerOrphaned, false},
		{SocketUnknown, nil, SessionOwnerUnknown, false},
		{SocketUnknown, errors.New("exit status 1"), SessionOwnerUnknown, false},
		{SocketPresent, nil, SessionOwnerOwned, false},
		{SocketPresent, errors.New("exit status 1"), SessionOwnerUnknown, true},
	} {
		w := &ownerWorld{present: true, generation: server, panes: panes, panesErr: tc.panesErr,
			sockets: map[string]SocketState{server.Socket: tc.socket}}
		got, err := SessionOwnerProbe{IO: w}.Probe(context.Background(), "📁1-37", "/tmp/data", "0123456789abcdef", "1-repo-1")
		if (err != nil) != tc.wantErr || got.State != tc.wantState {
			t.Fatalf("socket %v, panes err %v: got %+v, %v", tc.socket, tc.panesErr, got, err)
		}
		if tc.socket != SocketPresent && w.panesCalls != 0 {
			t.Fatalf("socket %v: asked list-panes of a server with no reachable socket", tc.socket)
		}
		if tc.socket == SocketGone {
			if got.Server != server || !strings.Contains(got.Diagnostic, "server PID 7 lost its socket — Tab → recover") {
				t.Fatalf("orphan observation %+v", got)
			}
		}
	}
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

func FuzzObservedCommandQuotedArguments(f *testing.F) {
	for _, arg := range []string{"", "/tmp/Pair data/draft-1-repo-1.md", "one'two", "a\\b", "$HOME; echo other", "\"quoted\"", "line\nnext"} {
		f.Add(arg)
	}
	f.Fuzz(func(t *testing.T, arg string) {
		if !utf8.ValidString(arg) || strings.ContainsRune(arg, 0) {
			return
		}
		quoted := "'" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
		got, err := splitObservedCommand("nvim " + quoted)
		if err != nil || len(got) != 2 || got[0] != "nvim" || got[1] != arg {
			t.Fatalf("quoted roundtrip %q: %q %v", arg, got, err)
		}
	})
}
