package sessioninventory_test

import (
	"context"
	"errors"
	si "github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNativeSelectorExactRequiresCanonicalRoot(t *testing.T) {
	for _, tc := range []struct {
		agent           si.Agent
		root, id, child string
	}{
		{si.AgentClaude, "claude-projects", "11111111-1111-4111-8111-111111111111", "worker-a"},
		{si.AgentCodex, "codex-sessions", "019d1111-1111-7111-8111-111111111111", "019d2222-2222-7222-8222-222222222222"},
		{si.AgentMuse, "muse-sessions", "77777777-7777-4777-8777-777777777777", "88888888-8888-4888-8888-888888888888"},
		{si.AgentQoder, "qoder-projects", "11111111-1111-4111-8111-111111111111", "worker-a"},
	} {
		t.Run(string(tc.agent), func(t *testing.T) {
			rt := sessioninventorytest.NewFakeRuntime()
			loadNativeFixture(t, rt, tc.agent, tc.root, filepath.Join("testdata", "native", string(tc.agent), "v1", tc.root))
			result, err := si.ResolveNativeSelector(context.Background(), selectorBoundedFake{rt}, si.NativeSelectorRequest{Agent: tc.agent, NativeID: tc.id})
			if err != nil || result.Status != si.NativeSelectorResolved || result.NativeID != tc.id {
				t.Fatalf("root=%+v %v", result, err)
			}
			for _, id := range []string{tc.child, "unknown-name", "--last"} {
				result, err = si.ResolveNativeSelector(context.Background(), selectorBoundedFake{rt}, si.NativeSelectorRequest{Agent: tc.agent, NativeID: id})
				if err == nil && result.Status == si.NativeSelectorResolved {
					t.Fatalf("nonroot authorized: %+v", result)
				}
			}
		})
	}
}

func TestNativeSelectorLatestNeverChoosesFromIncompleteOrTiedEvidence(t *testing.T) {
	at := time.Unix(100, 0)
	a := si.NativeSelectorCandidate{Agent: si.AgentClaude, NativeID: "a", CWD: "/work", UpdatedAt: at, Eligible: true}
	b := a
	b.NativeID = "b"
	b.UpdatedAt = at.Add(time.Second)
	for _, tc := range []struct {
		name     string
		snapshot si.NativeSelectorSnapshot
		want     si.NativeSelectorStatus
		id       string
	}{
		{"latest", si.NativeSelectorSnapshot{Complete: true, Candidates: []si.NativeSelectorCandidate{a, b}}, si.NativeSelectorResolved, "b"},
		{"incomplete", si.NativeSelectorSnapshot{Candidates: []si.NativeSelectorCandidate{a, b}}, si.NativeSelectorIncomplete, ""},
		{"tie", si.NativeSelectorSnapshot{Complete: true, Candidates: []si.NativeSelectorCandidate{a, {Agent: a.Agent, NativeID: "b", CWD: a.CWD, UpdatedAt: at, Eligible: true}}}, si.NativeSelectorAmbiguous, ""},
		{"none", si.NativeSelectorSnapshot{Complete: true}, si.NativeSelectorNone, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := si.SelectNativeLatest(si.NativeSelectorRequest{Agent: a.Agent, CWD: a.CWD, Latest: true}, tc.snapshot)
			if r.Status != tc.want || r.NativeID != tc.id {
				t.Fatalf("result=%+v", r)
			}
		})
	}
	foreign := b
	foreign.CWD = "/other"
	wrongAgent := b
	wrongAgent.Agent = si.AgentCodex
	child := b
	child.Eligible = false
	r := si.SelectNativeLatest(si.NativeSelectorRequest{Agent: a.Agent, CWD: a.CWD, Latest: true}, si.NativeSelectorSnapshot{Complete: true, Candidates: []si.NativeSelectorCandidate{a, foreign, wrongAgent, child}})
	if r.Status != si.NativeSelectorResolved || r.NativeID != "a" {
		t.Fatalf("eligibility=%+v", r)
	}
}

// selectorBoundedFake exercises the production bounded IO extension against the
// same mutable artifact world used by the existing scanner conformance tests.
type selectorBoundedFake struct {
	*sessioninventorytest.FakeRuntime
}

func (f selectorBoundedFake) VisitRecoveryFiles(ctx context.Context, root si.StorageRoot, limit int, visit func(si.FileEntry) error) (int, error) {
	files, err := f.ListFiles(root)
	if err != nil {
		return 0, err
	}
	for i, e := range files {
		if i >= limit {
			return i, si.ErrReadLimit
		}
		if err := ctx.Err(); err != nil {
			return i, err
		}
		if err := visit(e); err != nil {
			return i + 1, err
		}
	}
	return len(files), nil
}
func (f selectorBoundedFake) StatRecoveryFile(ctx context.Context, a si.Artifact) (si.FileEntry, error) {
	if err := ctx.Err(); err != nil {
		return si.FileEntry{}, err
	}
	return f.StatFile(a)
}
func (f selectorBoundedFake) ReadRecoveryAt(ctx context.Context, e si.FileEntry, offset, limit int64) ([]byte, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	return f.ReadAt(e.Artifact, offset, limit)
}
func (f selectorBoundedFake) ResolveRecoveryCWD(ctx context.Context, cwd string) (string, error) {
	return cwd, ctx.Err()
}

func TestNativeSelectorExactGrokAndAgy(t *testing.T) {
	t.Run("grok", func(t *testing.T) {
		rt := sessioninventorytest.NewFakeRuntime()
		loadNativeFixture(t, rt, si.AgentGrok, "grok-sessions", filepath.Join("testdata", "native", "grok", "v1", "grok-sessions"))
		r, err := si.ResolveNativeSelector(t.Context(), selectorBoundedFake{rt}, si.NativeSelectorRequest{Agent: si.AgentGrok, NativeID: grokFixtureID})
		if err != nil || r.Status != si.NativeSelectorResolved {
			t.Fatalf("grok=%+v %v", r, err)
		}
	})
	t.Run("agy", func(t *testing.T) {
		id := "55555555-5555-4555-8555-555555555555"
		rt, _, _ := incrementalAgyFixture(id)
		rt.AddRoot(si.StorageRoot{Agent: si.AgentAgy, Name: "agy-conversations", Path: "/native/conversations"})
		rt.AddRoot(si.StorageRoot{Agent: si.AgentAgy, Name: "agy-brain", Path: "/native/brain"})
		r, err := si.ResolveNativeSelector(t.Context(), selectorBoundedFake{rt}, si.NativeSelectorRequest{Agent: si.AgentAgy, NativeID: id})
		if err != nil || r.Status != si.NativeSelectorResolved {
			t.Fatalf("agy=%+v %v", r, err)
		}
	})
}

func TestNativeSelectorClaudeAndQoderLatestUseNativeMTimeNotMessageTime(t *testing.T) {
	for _, agent := range []si.Agent{si.AgentClaude, si.AgentQoder} {
		t.Run(string(agent), func(t *testing.T) {
			rt := sessioninventorytest.NewFakeRuntime()
			root := si.StorageRoot{Agent: agent, Name: string(agent) + "-projects", Path: "/native/projects"}
			rt.AddRoot(root)
			ids := []string{"11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222"}
			for i, id := range ids {
				mt := time.Unix(100+int64(i), 0)
				content := []byte(`{"type":"user","sessionId":"` + id + `","isSidechain":false,"cwd":"/work","timestamp":"2026-08-28T10:00:00Z","message":{"role":"user","content":"hello"}}` + "\n")
				rt.PutFile(si.FileEntry{Artifact: si.Artifact{StorageRoot: root.Name, RelativePath: "-work/" + id + ".jsonl"}, ModTime: &mt}, content)
			}
			r, err := si.ResolveNativeSelector(t.Context(), selectorBoundedFake{rt}, si.NativeSelectorRequest{Agent: agent, CWD: "/work", Latest: true})
			if err != nil || r.Status != si.NativeSelectorResolved || r.NativeID != ids[1] {
				t.Fatalf("native latest=%+v %v", r, err)
			}
		})
	}
}

func TestNativeSelectorCodexRequiresPositiveConfigurationAndExactScope(t *testing.T) {
	rt := selectorBoundedFake{sessioninventorytest.NewFakeRuntime()}
	for _, q := range []si.NativeSelectorRequest{
		{Agent: si.AgentCodex, CWD: "/work", Latest: true},
		{Agent: si.AgentCodex, CWD: "/work", Latest: true, Config: si.NativeSelectorConfig{ProviderResolved: true}},
		{Agent: si.AgentCodex, CWD: "/work", Latest: true, Config: si.NativeSelectorConfig{SQLiteHomeResolved: true, SQLiteHome: "/state"}},
		{Agent: si.AgentCodex, CWD: "/work", Latest: true, All: true},
		{Agent: si.AgentCodex, CWD: "/work", Latest: true, Config: si.NativeSelectorConfig{Remote: true}},
	} {
		r, err := si.ResolveNativeSelector(t.Context(), rt, q)
		if err != nil || r.Status != si.NativeSelectorUnsupported {
			t.Fatalf("unmapped config=%+v %v", r, err)
		}
	}
}

type selectorCodexMetadata struct {
	selectorBoundedFake
	rows  si.SQLiteResult
	query string
}

func (f *selectorCodexMetadata) QueryNativeSelectorSQLite(ctx context.Context, a si.Artifact, query string, limit int64) (si.SQLiteResult, error) {
	f.query = query
	return f.rows, ctx.Err()
}

func TestNativeSelectorCodexPinsNativeUpdatedMillisAndAuthorizedRoot(t *testing.T) {
	rt := sessioninventorytest.NewFakeRuntime()
	root := si.StorageRoot{Agent: si.AgentCodex, Name: "codex-sessions", Path: "/native/sessions"}
	rt.AddRoot(root)
	id := "019d1111-1111-7111-8111-111111111111"
	relative := "2026/08/28/rollout-test-" + id + ".jsonl"
	rt.PutFile(si.FileEntry{Artifact: si.Artifact{StorageRoot: root.Name, RelativePath: relative}}, []byte(`{"timestamp":"2026-08-28T10:00:00Z","type":"session_meta","payload":{"id":"`+id+`","source":"vscode","cwd":"/work"}}`+"\n"))
	metadata := &selectorCodexMetadata{selectorBoundedFake: selectorBoundedFake{rt}, rows: si.SQLiteResult{Columns: []string{"id", "rollout_path", "updated_at_ms"}, Rows: [][]string{{id, root.Path + "/" + relative, "100000"}}}}
	q := si.NativeSelectorRequest{Agent: si.AgentCodex, CWD: "/work", Latest: true, Config: si.NativeSelectorConfig{Provider: "provider'quoted", ProviderResolved: true, SQLiteHome: "/native/state", SQLiteHomeResolved: true}}
	r, err := si.ResolveNativeSelector(t.Context(), metadata, q)
	if err != nil || r.Status != si.NativeSelectorResolved || r.NativeID != id {
		t.Fatalf("resolved=%+v %v", r, err)
	}
	for _, want := range []string{"updated_at_ms DESC LIMIT 2", "preview<>''", "source IN ('cli','vscode')", "provider''quoted"} {
		if !strings.Contains(metadata.query, want) {
			t.Errorf("query missing %q: %s", want, metadata.query)
		}
	}
	q.IncludeNonInteractive = true
	_, err = si.ResolveNativeSelector(t.Context(), metadata, q)
	if err != nil || !strings.Contains(metadata.query, "'exec','mcp'") {
		t.Fatalf("source kinds: %s %v", metadata.query, err)
	}
	metadata.rows.Rows[0][1] = "/foreign/rollout-" + id + ".jsonl"
	r, err = si.ResolveNativeSelector(t.Context(), metadata, q)
	if err != nil || r.Status == si.NativeSelectorResolved {
		t.Fatalf("foreign path admitted: %+v %v", r, err)
	}
}

func TestNativeSelectorExactDisputedAndLatestPartialNeverAuthorize(t *testing.T) {
	rt := sessioninventorytest.NewFakeRuntime()
	root := si.StorageRoot{Agent: si.AgentClaude, Name: "claude-projects", Path: "/native/projects"}
	rt.AddRoot(root)
	id := "11111111-1111-4111-8111-111111111111"
	mt := time.Unix(100, 0)
	entry := si.FileEntry{Artifact: si.Artifact{StorageRoot: root.Name, RelativePath: "-work/" + id + ".jsonl"}, ModTime: &mt}
	content := []byte(`{"type":"user","sessionId":"` + id + `","isSidechain":false,"cwd":"/work","timestamp":"2026-08-28T10:00:00Z","message":{"role":"user","content":"hello"}}` + "\n")
	rt.PutFile(entry, append(content, []byte(`{"type":"user","sessionId":"`+id+`","isSidechain":true,"cwd":"/work","timestamp":"2026-08-28T10:01:00Z","message":{"role":"user","content":"changed"}}`+"\n")...))
	r, err := si.ResolveNativeSelector(t.Context(), selectorBoundedFake{rt}, si.NativeSelectorRequest{Agent: si.AgentClaude, NativeID: id})
	if err != nil || r.Status == si.NativeSelectorResolved {
		t.Fatalf("disputed=%+v %v", r, err)
	}
	rt.PutFile(entry, content)
	rt.SetError(sessioninventorytest.OperationListFiles, root.Name, errors.New("partial listing"))
	r, err = si.ResolveNativeSelector(t.Context(), selectorBoundedFake{rt}, si.NativeSelectorRequest{Agent: si.AgentClaude, CWD: "/work", Latest: true})
	if err != nil || r.Status != si.NativeSelectorIncomplete {
		t.Fatalf("partial=%+v %v", r, err)
	}
}

func TestNativeSelectorCodexRejectsMetadataIDDisagreement(t *testing.T) {
	rt := sessioninventorytest.NewFakeRuntime()
	root := si.StorageRoot{Agent: si.AgentCodex, Name: "codex-sessions", Path: "/native/sessions"}
	rt.AddRoot(root)
	id := "019d1111-1111-7111-8111-111111111111"
	rt.PutFile(si.FileEntry{Artifact: si.Artifact{StorageRoot: root.Name, RelativePath: "2026/08/28/rollout-test-" + id + ".jsonl"}}, []byte(`{"timestamp":"2026-08-28T10:00:00Z","type":"session_meta","payload":{"id":"019d2222-2222-7222-8222-222222222222","source":"cli"}}`+"\n"))
	r, err := si.ResolveNativeSelector(t.Context(), selectorBoundedFake{rt}, si.NativeSelectorRequest{Agent: si.AgentCodex, NativeID: id})
	if err != nil || r.Status == si.NativeSelectorResolved {
		t.Fatalf("contradictory native ID admitted: %+v %v", r, err)
	}
}

func TestIsCanonicalCodexSessionIDUsesFullNativeIdentity(t *testing.T) {
	for id, want := range map[string]bool{"019d1111-1111-7111-8111-111111111111": true, "FFFFFFFF-FFFF-4FFF-8FFF-FFFFFFFFFFFF": true, "019d1111": false, "session-name": false, "--last": false, "": false, "019d1111-1111-7111-8111-111111111111 suffix": false} {
		if got := si.IsCanonicalCodexSessionID(id); got != want {
			t.Errorf("%q: got%v want%v", id, got, want)
		}
	}
}
