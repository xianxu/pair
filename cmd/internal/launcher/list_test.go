package launcher

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestListStatus(t *testing.T) {
	cases := []struct {
		row  ListRow
		want string
	}{
		{ListRow{State: SessionExited}, "exited"},
		{ListRow{State: SessionExited, Clients: 3}, "exited"}, // exited wins over a stale count
		{ListRow{State: SessionDetached}, "detached"},
		{ListRow{State: SessionAttached, Clients: 1}, "attached (1 client)"},
		{ListRow{State: SessionAttached, Clients: 2}, "attached (2 clients)"},
	}
	for _, c := range cases {
		if got := listStatus(c.row); got != c.want {
			t.Errorf("listStatus(%+v) = %q, want %q", c.row, got, c.want)
		}
	}
}

func TestFormatListTableEmpty(t *testing.T) {
	if got := formatListTable(nil); got != "no pair sessions\n" {
		t.Fatalf("empty table = %q, want the no-sessions line", got)
	}
}

func TestFormatListTable(t *testing.T) {
	rows := []ListRow{
		{Session: "pair-a", Agent: "claude", State: SessionAttached, Clients: 1},
		{Session: "pair-b", Agent: "codex", State: SessionDetached},
		{Session: "pair-c", State: SessionExited}, // no agent → "?"
	}
	got := formatListTable(rows)
	for _, want := range []string{
		"SESSION", "AGENT", "STATUS",
		"pair-a", "claude", "attached (1 client)",
		"pair-b", "codex", "detached",
		"pair-c", "exited",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("table missing %q:\n%s", want, got)
		}
	}
	// A missing agent renders as "?".
	if !strings.Contains(got, "pair-c") || !strings.Contains(got[strings.Index(got, "pair-c"):], "?") {
		t.Fatalf("pair-c row should show a ? agent:\n%s", got)
	}
}

func TestRunList(t *testing.T) {
	rt := newFakeRuntime()
	rt.listRows = []ListRow{{Session: "pair-x", Agent: "claude", State: SessionDetached}}
	var out, errBuf bytes.Buffer
	if code := runList(rt, &out, &errBuf); code != 0 {
		t.Fatalf("runList code = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "pair-x") || !strings.Contains(out.String(), "detached") {
		t.Fatalf("runList output = %q", out.String())
	}
}

func TestRunListError(t *testing.T) {
	rt := newFakeRuntime()
	rt.listErr = errors.New("zellij not found")
	var out, errBuf bytes.Buffer
	if code := runList(rt, &out, &errBuf); code != 1 {
		t.Fatalf("runList code = %d, want 1 on error", code)
	}
	// The error goes to stderr, never stdout — stdout stays clean for pipes.
	if !strings.Contains(errBuf.String(), "zellij not found") {
		t.Fatalf("error should go to stderr, got stderr=%q", errBuf.String())
	}
	if out.Len() != 0 {
		t.Fatalf("stdout must stay empty on error, got %q", out.String())
	}
}

func TestBuildListRowsForScopeFiltersBySessionNameIndex(t *testing.T) {
	scope := mustScope(t, "/work/pair")
	other := mustScope(t, "/other/pair")
	index := SessionNameIndex{Entries: []SessionNameEntry{
		{SessionName: "pair-pair-work", ScopeKey: scope.Key, RepoName: "pair", Tag: "work"},
		{SessionName: "pair-pair-work-2", ScopeKey: other.Key, RepoName: "pair", Tag: "work"},
	}}

	rows := buildListRowsForScope(
		[]string{"pair-pair-work", "pair-pair-work-2", "pair-unindexed"},
		"pair-pair-work [Created]\npair-pair-work-2 [Created]\npair-unindexed [Created]\n",
		index,
		scope.Key,
		func(tag string) string {
			if tag != "work" {
				t.Fatalf("InferAgent called for tag %q; want only current-scope tag work", tag)
			}
			return "codex"
		},
		func(session string) int {
			if session == "pair-pair-work" {
				return 2
			}
			t.Fatalf("client count called for filtered session %q", session)
			return 0
		},
	)

	if len(rows) != 1 {
		t.Fatalf("rows = %+v, want one current-scope indexed row", rows)
	}
	row := rows[0]
	if row.Session != "pair-pair-work" || row.Agent != "codex" || row.State != SessionAttached || row.Clients != 2 {
		t.Fatalf("row = %+v", row)
	}
}
