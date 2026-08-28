package sessioninventory_test

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

func FuzzScanClaudeV1Records(f *testing.F) {
	f.Add([]byte(`{"type":"user","timestamp":"2026-08-28T09:01:00Z","sessionId":"11111111-1111-4111-8111-111111111111","isSidechain":false}`))
	f.Add([]byte(`{"type":"future"}`))
	f.Fuzz(func(t *testing.T, record []byte) {
		runtime := sessioninventorytest.NewFakeRuntime()
		root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentClaude, Name: "claude-projects"}
		runtime.AddRoot(root)
		runtime.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "-repo/11111111-1111-4111-8111-111111111111.jsonl"}}, boundedRecord(record))
		result := sessioninventory.ScanClaude(runtime)
		assertFixedFacts(t, result.Facts, sessioninventory.AgentClaude, "11111111-1111-4111-8111-111111111111")
	})
}

func FuzzScanCodexV1Records(f *testing.F) {
	f.Add([]byte(`{"timestamp":"2026-08-28T10:00:00Z","type":"session_meta","payload":{"id":"019d1111-1111-7111-8111-111111111111","source":"cli"}}`))
	f.Add([]byte(`{"type":"session_meta","payload":{"source":"future"}}`))
	f.Fuzz(func(t *testing.T, record []byte) {
		runtime := sessioninventorytest.NewFakeRuntime()
		root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentCodex, Name: "codex-sessions"}
		runtime.AddRoot(root)
		runtime.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "2026/08/28/rollout-fuzz-019d1111-1111-7111-8111-111111111111.jsonl"}}, boundedRecord(record))
		result := sessioninventory.ScanCodex(runtime)
		assertFixedFacts(t, result.Facts, sessioninventory.AgentCodex, "019d1111-1111-7111-8111-111111111111")
	})
}

func FuzzScanAgyV1Header(f *testing.F) {
	f.Add([]byte("SQLite format 3\x00sanitized"))
	f.Add([]byte("not sqlite"))
	f.Fuzz(func(t *testing.T, content []byte) {
		runtime := sessioninventorytest.NewFakeRuntime()
		root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentAgy, Name: "agy-conversations"}
		runtime.AddRoot(root)
		runtime.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "55555555-5555-4555-8555-555555555555.db"}}, boundedBytes(content))
		result := sessioninventory.ScanAgy(runtime)
		assertFixedFacts(t, result.Facts, sessioninventory.AgentAgy, "55555555-5555-4555-8555-555555555555")
	})
}

func FuzzScanMuseV1Records(f *testing.F) {
	f.Add([]byte(`{"payload_type":"runtime.session","payload":{"kind":"run","run_id":"77777777-7777-4777-8777-777777777777","event":{"kind":"started","prompt":"sanitized"}}}`))
	f.Add([]byte(`{"payload_type":"future"}`))
	f.Fuzz(func(t *testing.T, record []byte) {
		runtime := sessioninventorytest.NewFakeRuntime()
		root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentMuse, Name: "muse-sessions"}
		runtime.AddRoot(root)
		runtime.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "2026/08/28/77777777-7777-4777-8777-777777777777/session.jsonl"}}, boundedRecord(record))
		result := sessioninventory.ScanMuse(runtime)
		assertFixedFacts(t, result.Facts, sessioninventory.AgentMuse, "77777777-7777-4777-8777-777777777777")
	})
}

func boundedRecord(record []byte) []byte {
	record = boundedBytes(record)
	return append(record, '\n')
}

func boundedBytes(content []byte) []byte {
	if len(content) > 2<<20 {
		content = content[:2<<20]
	}
	return append([]byte(nil), content...)
}

func assertFixedFacts(t *testing.T, facts []sessioninventory.Fact, agent sessioninventory.Agent, nativeID string) {
	t.Helper()
	for _, fact := range facts {
		if fact.Agent != agent || fact.NativeID != nativeID {
			t.Fatalf("scanner invented fact %#v", fact)
		}
	}
}
