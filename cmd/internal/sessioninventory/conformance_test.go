package sessioninventory_test

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

func TestRunConformanceDistinguishesMalformedDataFromSchemaDrift(t *testing.T) {
	t.Parallel()

	runtime := sessioninventorytest.NewFakeRuntime()
	root := sessioninventory.StorageRoot{Agent: sessioninventory.AgentClaude, Name: "claude-projects", Path: "/native/claude"}
	runtime.AddRoot(root)
	runtime.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "-repo/11111111-1111-4111-8111-111111111111.jsonl"}}, []byte(`{"type":"user","timestamp":"2026-08-28T09:01:00Z","sessionId":"11111111-1111-4111-8111-111111111111","isSidechain":false}`+"\n"))
	runtime.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "-repo/11111111-1111-4111-8111-111111111111/subagents/agent-bad.jsonl"}}, []byte(
		`{"type":"user","timestamp":"2026-08-28T09:02:00Z","sessionId":"11111111-1111-4111-8111-111111111111","isSidechain":true}`+"\n"+
			`{"type":"assistant","sessionId":"22222222-2222-4222-8222-222222222222","isSidechain":true}`+"\n",
	))

	report, err := sessioninventory.RunConformance(runtime, sessioninventory.AgentClaude)
	if err != nil || report.Agents[0].Status != sessioninventory.ConformanceOK || !containsCode(report.Agents[0].Diagnostics, sessioninventory.DiagnosticNodeMalformed) {
		t.Fatalf("malformed-data report = %#v, err=%v", report, err)
	}

	runtime.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "-repo/not-a-v1-id.jsonl"}}, []byte("{}\n"))
	report, err = sessioninventory.RunConformance(runtime, sessioninventory.AgentClaude)
	if err == nil || report.Agents[0].Status != sessioninventory.ConformanceFail || !containsCode(report.Agents[0].Diagnostics, sessioninventory.DiagnosticSchemaNearMiss) {
		t.Fatalf("schema-drift report = %#v, err=%v", report, err)
	}
}

func containsCode(codes []sessioninventory.DiagnosticCode, wanted sessioninventory.DiagnosticCode) bool {
	for _, code := range codes {
		if code == wanted {
			return true
		}
	}
	return false
}
