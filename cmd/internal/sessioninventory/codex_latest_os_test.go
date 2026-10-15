package sessioninventory_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	si "github.com/xianxu/pair/cmd/internal/sessioninventory"
)

type codexLatestFixture struct {
	t                *testing.T
	home, state, cwd string
	rt               si.OSRuntime
}

func newCodexLatestFixture(t *testing.T) *codexLatestFixture {
	t.Helper()
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 unavailable")
	}
	home, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	f := &codexLatestFixture{t: t, home: home, state: filepath.Join(home, ".codex"), cwd: filepath.Join(home, "work")}
	for _, p := range []string{filepath.Join(f.state, "sessions", "2026", "08", "28"), f.cwd, filepath.Join(home, "other")} {
		if err := os.MkdirAll(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	f.rt = si.NewOSRuntime(home, filepath.Join(home, "pair"))
	f.sql("CREATE TABLE threads(id TEXT,rollout_path TEXT,updated_at_ms INTEGER,archived INTEGER,preview TEXT,cwd TEXT,source TEXT,model_provider TEXT)")
	return f
}
func (f *codexLatestFixture) sql(s string) {
	f.t.Helper()
	if out, err := exec.Command("sqlite3", filepath.Join(f.state, "state_5.sqlite"), s).CombinedOutput(); err != nil {
		f.t.Fatalf("SQLite: %v %s", err, out)
	}
}
func (f *codexLatestFixture) add(n, ms int, source, cwd, tail string) string {
	f.t.Helper()
	id := fmt.Sprintf("019d1111-1111-7111-8111-%012d", n)
	p := filepath.Join(f.state, "sessions", "2026", "08", "28", "rollout-test-"+id+".jsonl")
	raw := codexLatestValidMeta(id, cwd) + tail
	if err := os.WriteFile(p, []byte(raw), 0600); err != nil {
		f.t.Fatal(err)
	}
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
	f.sql(fmt.Sprintf("INSERT INTO threads VALUES(%s,%s,%d,0,'visible',%s,%s,'openai')", quote(id), quote(p), ms, quote(cwd), quote(source)))
	return id
}
func (f *codexLatestFixture) query(mode si.CodexInvocationMode, all bool, cwds ...string) si.NativeSelectorResolution {
	f.t.Helper()
	q := si.NativeSelectorRequest{Agent: si.AgentCodex, CWD: f.cwd, Latest: true, All: all, Config: si.NativeSelectorConfig{CodexScope: &si.CodexSelectionScope{CodexHome: f.state, SQLiteHome: f.state, Provider: "openai", CWD: f.cwd, CWDs: cwds, Mode: mode}}}
	r, err := si.ResolveNativeSelector(f.t.Context(), f.rt, q)
	if err != nil {
		f.t.Fatal(err)
	}
	return r
}
func TestCodexLatestTUIAndExecHaveDifferentCWDSources(t *testing.T) {
	f := newCodexLatestFixture(t)
	other := filepath.Join(f.home, "other")
	wantTUI := f.add(1, 100, "cli", f.cwd, "")
	wantExec := f.add(2, 200, "app-server", other, codexLatestValidTurn(f.cwd))
	for _, tc := range []struct {
		mode si.CodexInvocationMode
		want string
	}{{si.CodexModeTUI, wantTUI}, {si.CodexModeExec, wantExec}} {
		r := f.query(tc.mode, false, f.cwd)
		if r.Status != si.NativeSelectorResolved || r.NativeID != tc.want {
			t.Fatalf("mode=%s: %+v want %s", tc.mode, r, tc.want)
		}
	}
}
func TestCodexLatestTUIWorktreeAndAllScope(t *testing.T) {
	f := newCodexLatestFixture(t)
	other := filepath.Join(f.home, "other")
	f.add(1, 100, "cli", f.cwd, "")
	want := f.add(2, 200, "vscode", other, "")
	for _, all := range []bool{false, true} {
		r := f.query(si.CodexModeTUI, all, f.cwd, other)
		if r.Status != si.NativeSelectorResolved || r.NativeID != want {
			t.Fatalf("all=%v: %+v", all, r)
		}
	}
}
func TestCodexLatestDoesNotSkipUnusableTopOrTies(t *testing.T) {
	for _, mode := range []si.CodexInvocationMode{si.CodexModeTUI, si.CodexModeExec} {
		for _, kind := range []string{"path", "identity", "tie", "tail"} {
			t.Run(string(mode)+kind, func(t *testing.T) {
				f := newCodexLatestFixture(t)
				f.add(1, 100, "cli", f.cwd, "")
				id := f.add(2, 200, "cli", f.cwd, "")
				switch kind {
				case "path":
					f.sql("UPDATE threads SET rollout_path='/outside/rollout.jsonl' WHERE updated_at_ms=200")
				case "identity":
					p := filepath.Join(f.state, "sessions", "2026", "08", "28", "rollout-test-"+id+".jsonl")
					if err := os.WriteFile(p, []byte("{}\n"), 0600); err != nil {
						t.Fatal(err)
					}
				case "tie":
					f.sql("UPDATE threads SET updated_at_ms=200")
				case "tail":
					if mode == si.CodexModeTUI {
						t.Skip("TUI does not inspect latest turn cwd")
					}
					p := filepath.Join(f.state, "sessions", "2026", "08", "28", "rollout-test-"+id+".jsonl")
					h, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0600)
					if err != nil {
						t.Fatal(err)
					}
					_, err = h.WriteString(`{"type":"turn_context","payload":{"cwd":`)
					h.Close()
					if err != nil {
						t.Fatal(err)
					}
				}
				if r := f.query(mode, false, f.cwd); r.Status == si.NativeSelectorResolved {
					t.Fatalf("uncertain winner resolved: %+v", r)
				}
			})
		}
	}
}

func TestCodexLatestExecRejectsTieBeforeMatchingCWD(t *testing.T) {
	f := newCodexLatestFixture(t)
	f.add(1, 200, "cli", filepath.Join(f.home, "other"), "")
	f.add(2, 200, "cli", f.cwd, "")
	if r := f.query(si.CodexModeExec, false, f.cwd); r.Status == si.NativeSelectorResolved {
		t.Fatalf("tie authorized after cwd filtering: %+v", r)
	}
}
func TestCodexLatestExecPagesAndUsesLastTurn(t *testing.T) {
	f := newCodexLatestFixture(t)
	other := filepath.Join(f.home, "other")
	for n := 1; n <= 101; n++ {
		f.add(n, 1000-n, "cli", other, "")
	}
	tail := codexLatestValidTurn(other) + codexLatestValidWorldState(strings.Repeat("x", 256<<10)) + codexLatestValidTurn(f.cwd)
	want := f.add(102, 1, "exec", other, tail)
	if r := f.query(si.CodexModeExec, false, f.cwd); r.Status != si.NativeSelectorResolved || r.NativeID != want {
		t.Fatalf("paged latest: %+v want %s", r, want)
	}
}
func TestCodexLatestExecAllBypassesTurnCWD(t *testing.T) {
	f := newCodexLatestFixture(t)
	want := f.add(1, 100, "exec", filepath.Join(f.home, "other"), "")
	if r := f.query(si.CodexModeExec, true, f.cwd); r.Status != si.NativeSelectorResolved || r.NativeID != want {
		t.Fatalf("all latest: %+v", r)
	}
}

func TestCodexLatestScopedHomesUseOnlySelectedNamespace(t *testing.T) {
	f := newCodexLatestFixture(t)
	f.add(1, 999, "cli", f.cwd, "")
	originalState := f.state
	f.state = filepath.Join(f.home, "custom-codex")
	if err := os.MkdirAll(filepath.Join(f.state, "sessions", "2026", "08", "28"), 0700); err != nil {
		t.Fatal(err)
	}
	f.sql("CREATE TABLE threads(id TEXT,rollout_path TEXT,updated_at_ms INTEGER,archived INTEGER,preview TEXT,cwd TEXT,source TEXT,model_provider TEXT)")
	want := f.add(2, 100, "cli", f.cwd, "")
	scope := si.CodexSelectionScope{CodexHome: f.state, SQLiteHome: f.state, Provider: "openai", CWD: f.cwd, CWDs: []string{f.cwd}, Mode: si.CodexModeTUI}
	scoped, err := f.rt.WithCodexScope(scope)
	if err != nil {
		t.Fatal(err)
	}
	f.rt = scoped
	if r := f.query(si.CodexModeTUI, false, f.cwd); r.Status != si.NativeSelectorResolved || r.NativeID != want {
		t.Fatalf("custom namespace: %+v", r)
	}
	outside := filepath.Join(originalState, "sessions", "2026", "08", "28", "rollout-test-019d1111-1111-7111-8111-000000000001.jsonl")
	f.sql("UPDATE threads SET rollout_path='" + outside + "'")
	if r := f.query(si.CodexModeTUI, false, f.cwd); r.Status == si.NativeSelectorResolved {
		t.Fatalf("other namespace authorized: %+v", r)
	}
}

type codexLatestChangingRuntime struct {
	si.OSRuntime
	beforeVisit func()
}

func (r codexLatestChangingRuntime) VisitRecoveryFiles(ctx context.Context, root si.StorageRoot, limit int, visit func(si.FileEntry) error) (int, error) {
	r.beforeVisit()
	return r.OSRuntime.VisitRecoveryFiles(ctx, root, limit, visit)
}
func TestCodexLatestCannotAuthorizeTranscriptChangedAfterCWDSelection(t *testing.T) {
	f := newCodexLatestFixture(t)
	want := f.add(1, 100, "exec", f.cwd, "")
	scope := si.CodexSelectionScope{CodexHome: f.state, SQLiteHome: f.state, Provider: "openai", CWD: f.cwd, CWDs: []string{f.cwd}, Mode: si.CodexModeExec}
	rt, err := f.rt.WithCodexScope(scope)
	if err != nil {
		t.Fatal(err)
	}
	changed := false
	mutating := codexLatestChangingRuntime{OSRuntime: rt, beforeVisit: func() {
		if changed {
			return
		}
		changed = true
		p := filepath.Join(f.state, "sessions", "2026", "08", "28", "rollout-test-"+want+".jsonl")
		h, err := os.OpenFile(p, os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			t.Fatal(err)
		}
		_, err = h.WriteString(codexLatestValidTurn(filepath.Join(f.home, "other")))
		h.Close()
		if err != nil {
			t.Fatal(err)
		}
	}}
	q := si.NativeSelectorRequest{Agent: si.AgentCodex, CWD: f.cwd, Latest: true, Config: si.NativeSelectorConfig{CodexScope: &scope}}
	r, err := si.ResolveNativeSelector(t.Context(), mutating, q)
	if err != nil {
		t.Fatal(err)
	}
	if r.Status == si.NativeSelectorResolved {
		t.Fatalf("changed cwd snapshot authorized: %+v", r)
	}
}

func TestCodexLatestCannotIgnoreHigherCandidateChangedAfterCWDSelection(t *testing.T) {
	f := newCodexLatestFixture(t)
	higher := f.add(1, 200, "exec", filepath.Join(f.home, "other"), "")
	f.add(2, 100, "exec", f.cwd, "")
	scope := si.CodexSelectionScope{CodexHome: f.state, SQLiteHome: f.state, Provider: "openai", CWD: f.cwd, CWDs: []string{f.cwd}, Mode: si.CodexModeExec}
	rt, err := f.rt.WithCodexScope(scope)
	if err != nil {
		t.Fatal(err)
	}
	changed := false
	mutating := codexLatestChangingRuntime{OSRuntime: rt, beforeVisit: func() {
		if changed {
			return
		}
		changed = true
		p := filepath.Join(f.state, "sessions", "2026", "08", "28", "rollout-test-"+higher+".jsonl")
		h, err := os.OpenFile(p, os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			t.Fatal(err)
		}
		_, err = h.WriteString(codexLatestValidTurn(f.cwd))
		h.Close()
		if err != nil {
			t.Fatal(err)
		}
	}}
	q := si.NativeSelectorRequest{Agent: si.AgentCodex, CWD: f.cwd, Latest: true, Config: si.NativeSelectorConfig{CodexScope: &scope}}
	r, err := si.ResolveNativeSelector(t.Context(), mutating, q)
	if err != nil {
		t.Fatal(err)
	}
	if r.Status == si.NativeSelectorResolved {
		t.Fatalf("changed higher candidate authorized: %+v", r)
	}
}

type codexLatestQueryRuntime struct {
	si.OSRuntime
	beforeQuery func()
}

func (r codexLatestQueryRuntime) QueryNativeSelectorSQLite(ctx context.Context, a si.Artifact, query string, limit int64) (si.SQLiteResult, error) {
	r.beforeQuery()
	return r.OSRuntime.QueryNativeSelectorSQLite(ctx, a, query, limit)
}
func TestCodexLatestExecCursorSurvivesDatabaseChangesBetweenPages(t *testing.T) {
	for _, mutation := range []string{"delete-higher", "arrive-higher", "delete-and-arrive"} {
		t.Run(mutation, func(t *testing.T) {
			f := newCodexLatestFixture(t)
			other := filepath.Join(f.home, "other")
			for n := 1; n <= 100; n++ {
				f.add(n, 1000-n, "exec", other, "")
			}
			want := f.add(101, 100, "exec", f.cwd, "")
			f.add(102, 50, "exec", f.cwd, "")
			scope := si.CodexSelectionScope{CodexHome: f.state, SQLiteHome: f.state, Provider: "openai", CWD: f.cwd, CWDs: []string{f.cwd}, Mode: si.CodexModeExec}
			native, err := f.rt.WithCodexScope(scope)
			if err != nil {
				t.Fatal(err)
			}
			queries := 0
			rt := codexLatestQueryRuntime{OSRuntime: native, beforeQuery: func() {
				queries++
				if queries != 2 {
					return
				}
				switch mutation {
				case "delete-higher":
					f.sql("DELETE FROM threads WHERE updated_at_ms=999")
				case "arrive-higher":
					f.add(999, 9999, "exec", f.cwd, "")
				case "delete-and-arrive":
					f.sql("DELETE FROM threads WHERE updated_at_ms>=998")
					f.add(999, 9999, "exec", f.cwd, "")
				}
			}}
			q := si.NativeSelectorRequest{Agent: si.AgentCodex, CWD: f.cwd, Latest: true, Config: si.NativeSelectorConfig{CodexScope: &scope}}
			got, err := si.ResolveNativeSelector(t.Context(), rt, q)
			if err != nil {
				t.Fatal(err)
			}
			if queries < 2 || got.Status != si.NativeSelectorResolved || got.NativeID != want {
				t.Fatalf("queries=%d got=%+v want=%s", queries, got, want)
			}
		})
	}
}
func TestCodexLatestExecCursorDoesNotSkipBoundaryTies(t *testing.T) {
	f := newCodexLatestFixture(t)
	other := filepath.Join(f.home, "other")
	for n := 1; n <= 100; n++ {
		f.add(n, 1000-n, "exec", other, "")
	}
	f.add(101, 900, "exec", f.cwd, "")
	f.add(102, 50, "exec", f.cwd, "")
	if got := f.query(si.CodexModeExec, false, f.cwd); got.Status == si.NativeSelectorResolved {
		t.Fatalf("boundary tie authorized: %+v", got)
	}
}

func codexLatestValidMeta(id, cwd string) string {
	return fmt.Sprintf("{\"timestamp\":\"2026-08-28T10:00:00Z\",\"type\":\"session_meta\",\"payload\":{\"id\":%q,\"timestamp\":\"2026-08-28T10:00:00Z\",\"cwd\":%q,\"originator\":\"codex_cli_rs\",\"cli_version\":\"0.162.1\",\"source\":\"cli\"}}\n", id, cwd)
}
func codexLatestValidTurn(cwd string) string {
	return fmt.Sprintf("{\"timestamp\":\"2026-08-28T10:00:01Z\",\"type\":\"turn_context\",\"payload\":{\"cwd\":%q,\"approval_policy\":\"never\",\"sandbox_policy\":{\"type\":\"read-only\"},\"model\":\"test-model\"}}\n", cwd)
}
func codexLatestMutatedRecord(t *testing.T, raw, field string, value any, remove bool, envelope bool) string {
	t.Helper()
	var row map[string]any
	if err := json.Unmarshal([]byte(raw), &row); err != nil {
		t.Fatal(err)
	}
	target := row
	if !envelope {
		target = row["payload"].(map[string]any)
	}
	if remove {
		delete(target, field)
	} else {
		target[field] = value
	}
	out, err := json.Marshal(row)
	if err != nil {
		t.Fatal(err)
	}
	return string(out) + "\n"
}
func TestCodexLatestRejectsNativeInvalidTurnContext(t *testing.T) {
	cases := []struct {
		name, field      string
		value            any
		remove, envelope bool
	}{
		{name: "missing-approval", field: "approval_policy", remove: true},
		{name: "missing-sandbox", field: "sandbox_policy", remove: true},
		{name: "missing-model", field: "model", remove: true},
		{name: "missing-timestamp", field: "timestamp", remove: true, envelope: true},
		{name: "null-timestamp", field: "timestamp", envelope: true},
		{name: "negative-ordinal", field: "ordinal", value: -1, envelope: true},
		{name: "unknown-approval", field: "approval_policy", value: "future-approval"},
		{name: "malformed-approval", field: "approval_policy", value: map[string]any{"granular": map[string]any{"rules": true}}},
		{name: "unknown-sandbox", field: "sandbox_policy", value: map[string]any{"type": "future-sandbox"}},
		{name: "malformed-sandbox", field: "sandbox_policy", value: map[string]any{"type": "workspace-write", "network_access": "yes"}},
		{name: "malformed-model", field: "model", value: 123},
		{name: "malformed-optional-plugins", field: "disabled_plugin_ids", value: []any{1}},
		{name: "malformed-optional-permissions", field: "permission_profile", value: map[string]any{"garbage": true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newCodexLatestFixture(t)
			f.add(1, 100, "exec", f.cwd, "")
			tail := codexLatestMutatedRecord(t, codexLatestValidTurn(f.cwd), tc.field, tc.value, tc.remove, tc.envelope)
			f.add(2, 200, "exec", filepath.Join(f.home, "other"), tail)
			if got := f.query(si.CodexModeExec, false, f.cwd); got.Status == si.NativeSelectorResolved {
				t.Fatalf("native-invalid turn selected guarded ID: %+v", got)
			}
		})
	}
}
func TestCodexLatestRejectsNativeInvalidSessionMeta(t *testing.T) {
	for _, mode := range []si.CodexInvocationMode{si.CodexModeTUI, si.CodexModeExec} {
		for _, field := range []string{"timestamp", "cwd", "originator", "cli_version", "session_id", "source", "runtime_workspace_roots", "base_instructions", "history_mode", "git"} {
			t.Run(string(mode)+field, func(t *testing.T) {
				f := newCodexLatestFixture(t)
				f.add(1, 100, "cli", f.cwd, "")
				id := f.add(2, 200, "cli", f.cwd, "")
				remove := field == "timestamp" || field == "cwd" || field == "originator" || field == "cli_version"
				var value any = 123
				if field == "source" {
					value = "custom"
				}
				if field == "history_mode" {
					value = "future-history"
				}
				if field == "base_instructions" || field == "git" {
					value = map[string]any{"garbage": true}
				}
				raw := codexLatestMutatedRecord(t, codexLatestValidMeta(id, f.cwd), field, value, remove, false)
				p := filepath.Join(f.state, "sessions", "2026", "08", "28", "rollout-test-"+id+".jsonl")
				if err := os.WriteFile(p, []byte(raw), 0600); err != nil {
					t.Fatal(err)
				}
				if got := f.query(mode, false, f.cwd); got.Status == si.NativeSelectorResolved {
					t.Fatalf("native-invalid session meta selected guarded ID: %+v", got)
				}
			})
		}
	}
}

func codexLatestValidWorldState(text string) string {
	return fmt.Sprintf("{\"timestamp\":\"2026-08-28T10:00:02Z\",\"type\":\"world_state\",\"payload\":{\"full\":true,\"state\":{\"text\":%q}}}\n", text)
}
