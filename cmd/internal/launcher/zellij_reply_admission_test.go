package launcher

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// A nonempty reply is not proof that it answers list-clients. In particular,
// pane/tab JSON used to be counted as a header-only detached client table.
func TestZellijClientCountRejectsNonemptyForeignReply(t *testing.T) {
	for _, raw := range []string{`[{"id":1,"is_plugin":false}]`, `{"active_swap_layout_name":null,"is_swap_layout_dirty":false}`} {
		t.Run(raw, func(t *testing.T) {
			f := newEmptyReplyFake(t,
				emptyReplyFrame{Args: []string{"list-sessions", "--no-formatting"}, Stdout: "pair-a [Created 1s ago]\n"},
				emptyReplyFrame{Args: []string{"--session", "pair-a", "action", "list-clients"}, Stdout: raw},
			)
			got, err := f.source.SnapshotSessionsContext(context.Background(), []string{"pair-a"})
			if err == nil || got != nil {
				t.Errorf("foreign reply granted detached authority: %+v, %v", got, err)
			}
			if calls := f.calls(t); calls != 2 {
				t.Errorf("calls=%d; want2, no retry of unknown", calls)
			}
		})
	}
}

type changingPaneOwnerWorld struct {
	ownerWorld
	next []byte
}

func (w *changingPaneOwnerWorld) SessionPanes(ctx context.Context, name string) ([]byte, error) {
	raw, err := w.ownerWorld.SessionPanes(ctx, name)
	if w.panesCalls > 1 {
		return w.next, err
	}
	return raw, err
}

func TestSessionOwnerProbeRefusesChangedPaneOwner(t *testing.T) {
	w := &changingPaneOwnerWorld{ownerWorld: ownerWorld{
		present:    true,
		generation: SessionServerIdentity{PID: 42, Identity: "same-server", Session: "pair-a"},
		panes:      []byte(`[{"id":1,"is_plugin":false,"pane_command":"nvim /tmp/data/repos/0123456789abcdef/draft-1-repo-1.md"}]`),
	}, next: []byte(`[{"id":1,"is_plugin":false,"pane_command":"nvim /tmp/data/repos/0123456789abcdef/draft-1-repo-2.md"}]`)}
	p := SessionOwnerProbe{IO: w}
	got, err := p.Probe(t.Context(), "pair-a", "/tmp/data", "0123456789abcdef", "1-repo-1")
	if err == nil && got.State != SessionOwnerUnknown {
		t.Fatalf("conflicting owner observations granted authority: %+v", got)
	}
	if err := p.Revalidate(t.Context(), got); err == nil {
		t.Fatal("refused observation was revalidated")
	}
	if w.panesCalls != 2 {
		t.Fatalf("pane reads=%d want2 independent observations", w.panesCalls)
	}
}

func TestSessionNameOmissionCannotOverrideLiveServer(t *testing.T) {
	f := newEmptyReplyFake(t, emptyReplyFrame{Args: []string{"list-sessions", "--no-formatting"}, Stdout: "pair-other [Created 1s ago]\n"})
	t.Setenv("PATH", filepath.Dir(f.source.Path)+string(os.PathListSeparator)+os.Getenv("PATH"))
	ops := &statefulSessionQuiescenceOps{serverAlive: true, serverPID: 42}
	if !(OSRuntime{sessionQuiescence: ops}).SessionBlocksReuse("pair-a") {
		t.Fatal("well-formed listing omission freed a live server name")
	}
	if len(ops.killed) != 0 {
		t.Fatalf("read-only admission killed servers: %v", ops.killed)
	}
	if f.calls(t) != 1 {
		t.Fatal("unknown listing must not be retried")
	}
}

func TestListedTerminalUnknownNeverRetries(t *testing.T) {
	calls := 0
	err := requireListedLiveTerminal(t.Context(), "pair-a", func(context.Context) ([]byte, error) {
		calls++
		if calls == 1 {
			return []byte(`{"tab_id":1}`), nil
		}
		return []byte("pair-a [Created 1s ago]\n"), nil
	}, noLiveServers)
	if err == nil || calls != 1 {
		t.Fatalf("unknown listing was retried into attach authority: calls=%d err=%v", calls, err)
	}
}

func TestWrongKindFullListingStopsBeforeFurtherQuery(t *testing.T) {
	f := newEmptyReplyFake(t, emptyReplyFrame{Args: []string{"list-sessions", "--no-formatting"}, Stdout: `[{"id":1,"is_plugin":false}]`})
	got, err := f.source.SnapshotSessionsContext(t.Context(), []string{"pair-a"})
	if err == nil || got != nil || f.calls(t) != 1 {
		t.Fatalf("foreign first listing continued: got=%v err=%v calls=%d", got, err, f.calls(t))
	}
}

func TestFullListingConsumersKeepExactIdentityAndState(t *testing.T) {
	for _, tc := range []struct {
		name, session, raw string
		present, live      bool
	}{
		{"foreign leading space", "pair-a", " pair-a [Created 1s ago]\n", false, false},
		{"EXITED text in live name", "pair-EXITED", "pair-EXITED [Created 1s ago]\n", true, true},
		{"actual exited suffix", "pair-a", "pair-a [Created 1s ago] (EXITED - attach to resurrect)\n", true, false},
	} {
		for _, consumer := range []string{"owner", "quiescence", "handoff", "reuse"} {
			t.Run(tc.name+"/"+consumer, func(t *testing.T) {
				f := newEmptyReplyFake(t, emptyReplyFrame{Args: []string{"list-sessions", "--no-formatting"}, Stdout: tc.raw})
				t.Setenv("PATH", filepath.Dir(f.source.Path)+string(os.PathListSeparator)+os.Getenv("PATH"))
				switch consumer {
				case "owner":
					got, err := (osSessionOwnerIO{}).SessionPresent(t.Context(), tc.session)
					if err != nil || got != tc.live {
						t.Fatalf("live=%v err=%v want=%v", got, err, tc.live)
					}
				case "quiescence":
					got, err := (osSessionQuiescenceOps{}).SessionPresent(t.Context(), tc.session)
					if err != nil || got != tc.live {
						t.Fatalf("live=%v err=%v want=%v", got, err, tc.live)
					}
				case "handoff":
					err := requireListedLiveTerminal(t.Context(), tc.session, listTerminalSessions, noLiveServers)
					if (err == nil) != tc.live {
						t.Fatalf("handoff err=%v want live=%v", err, tc.live)
					}
				case "reuse":
					got := (OSRuntime{sessionQuiescence: &statefulSessionQuiescenceOps{}}).SessionBlocksReuse(tc.session)
					if got != tc.present {
						t.Fatalf("blocks reuse=%v want=%v", got, tc.present)
					}
				}
				if f.calls(t) != 1 {
					t.Fatalf("full listing was retried: calls=%d", f.calls(t))
				}
			})
		}
	}
}
