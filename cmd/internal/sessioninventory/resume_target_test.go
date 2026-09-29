package sessioninventory_test

import (
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
	"strings"
	"testing"
)

func TestResumeTargetDoesNotNeedNativeTranscript(t *testing.T) {
	runtime := sessioninventorytest.NewFakeRuntime()
	root := sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair"}
	runtime.SetPairDataRoot(root)
	artifact := sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "ledger-work.jsonl"}
	launch := `{"v":3,"kind":"launch","scope_key":"scope","tag":"work","agent":"codex","pair_log_offset":0,"artifact_boundaries":[],"requested_native_id":"A","request_origin":"resume","baseline_complete":false}` + "\n"
	runtime.PutFile(sessioninventory.FileEntry{Artifact: artifact}, []byte(launch))
	got, err := sessioninventory.QueryResumeTarget(runtime, "scope", "work", sessioninventory.AgentCodex)
	if err != nil || got.NativeID != "A" || got.Status != sessioninventory.BindingProvisional {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	runtime.PutFile(sessioninventory.FileEntry{Artifact: artifact}, []byte(launch+`{"v":3,"kind":"binding","scope_key":"scope","tag":"work","agent":"codex","launch_ordinal":1,"root_native_id":"D","confirmation_reason":"correlation"}`+"\n"))
	got, err = sessioninventory.QueryResumeTarget(runtime, "scope", "work", sessioninventory.AgentCodex)
	if err != nil || got.NativeID != "D" || got.RequestedNativeID != "A" || got.Status != sessioninventory.BindingEstablished {
		t.Fatalf("got=%+v err=%v", got, err)
	}
	runtime.AppendFile(artifact, []byte(`{"v":3,"kind":"launch","scope_key":"scope","tag":"work","agent":"codex","pair_log_offset":0,"artifact_boundaries":[],"baseline_complete":true}`+"\n"), "new")
	got, err = sessioninventory.QueryResumeTarget(runtime, "scope", "work", sessioninventory.AgentCodex)
	if err != nil || got.NativeID != "" || got.Status != sessioninventory.BindingProvisional {
		t.Fatalf("fresh reused old target: %+v %v", got, err)
	}
}

func TestOwnerCLIEmitsProbationTargetWithoutNativeParsing(t *testing.T) {
	rt := sessioninventorytest.NewFakeRuntime()
	root := sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair"}
	rt.SetPairDataRoot(root)
	rt.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "ledger-work.jsonl"}}, []byte(`{"v":3,"kind":"launch","scope_key":"scope","tag":"work","agent":"codex","pair_log_offset":0,"artifact_boundaries":[],"requested_native_id":"A","request_origin":"resume","baseline_complete":false}`+"\n"))
	var out, errout strings.Builder
	code := sessioninventory.RunCLIWithRuntime([]string{"--agent", "codex", "--scope", "current", "--owner", "work"}, func(key string) string {
		if key == "PAIR_SCOPE_KEY" {
			return "scope"
		}
		return ""
	}, rt, &out, &errout)
	if code != 0 || out.String() != "A\n" {
		t.Fatalf("code=%d out=%q err=%q", code, out.String(), errout.String())
	}
}

func TestLegacyResumeTargetSurvivesNativeAndCatalogFailures(t *testing.T) {
	const id = "019d1111-1111-7111-8111-111111111111"
	for _, failure := range []string{"missing-transcript", "changed-device", "missing-catalog", "corrupt-catalog"} {
		t.Run(failure, func(t *testing.T) {
			rt, transcript, _ := proofBackedCodexFixture(t, id, nil)
			switch failure {
			case "missing-transcript":
				rt.DeleteFile(transcript)
			case "changed-device":
				rt.PutFile(sessioninventory.FileEntry{Artifact: transcript, StableFileID: "device-changed", MutationToken: "new"}, []byte("unfamiliar metadata"))
			case "corrupt-catalog":
				rt.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: rt.PairDataRoot().Name, RelativePath: "session-inventory-catalog.json"}}, []byte("corrupt"))
			}
			got, err := sessioninventory.QueryResumeTarget(rt, "scope", "work", sessioninventory.AgentCodex)
			if err != nil || got.NativeID != id || got.Status != sessioninventory.BindingEstablished {
				t.Fatalf("target=%+v err=%v", got, err)
			}
			if rt.OperationCountForRoot(sessioninventorytest.OperationReadAt, transcript.StorageRoot) != 0 {
				t.Fatal("target read native body")
			}
		})
	}
}

func TestChosenConfirmationCanReadOptionalParsedContent(t *testing.T) {
	const id = "019d1111-1111-7111-8111-111111111111"
	rt, _, ledger := proofBackedCodexFixture(t, id, nil)
	rt.PutFile(sessioninventory.FileEntry{Artifact: ledger}, []byte(`{"v":3,"kind":"launch","scope_key":"scope","tag":"work","agent":"codex","pair_log_offset":0,"artifact_boundaries":[],"requested_native_id":"`+id+`","request_origin":"chosen-id","baseline_complete":true}`+"\n"+`{"v":3,"kind":"binding","scope_key":"scope","tag":"work","agent":"codex","launch_ordinal":1,"root_native_id":"`+id+`","confirmation_reason":"chosen-id"}`+"\n"))
	got, err := sessioninventory.QuerySession(rt, "scope", "work", sessioninventory.AgentCodex)
	if err != nil || got.Root == nil || got.Root.NativeID != id {
		t.Fatalf("optional content unavailable: %+v %v", got, err)
	}
}

func TestUnconfirmedChosenTargetRequiresRootFilenameOnly(t *testing.T) {
	const id = "11111111-1111-4111-8111-111111111111"
	for _, agent := range []sessioninventory.Agent{sessioninventory.AgentClaude, sessioninventory.AgentQoder} {
		t.Run(string(agent), func(t *testing.T) {
			rt := sessioninventorytest.NewFakeRuntime()
			pair := sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair"}
			rt.SetPairDataRoot(pair)
			root := sessioninventory.StorageRoot{Name: string(agent) + "-projects", Path: "/native", Agent: agent}
			rt.AddRoot(root)
			ledger := sessioninventory.Artifact{StorageRoot: pair.Name, RelativePath: "ledger-work.jsonl"}
			launch := `{"v":3,"kind":"launch","scope_key":"scope","tag":"work","agent":"` + string(agent) + `","pair_log_offset":0,"artifact_boundaries":[],"requested_native_id":"` + id + `","request_origin":"chosen-id","baseline_complete":true}` + "\n"
			rt.PutFile(sessioninventory.FileEntry{Artifact: ledger}, []byte(launch))
			check := func(want string) {
				t.Helper()
				got, err := sessioninventory.QueryResumeTarget(rt, "scope", "work", agent)
				if err != nil || got.NativeID != want {
					t.Fatalf("target=%+v err=%v want=%q", got, err, want)
				}
			}
			check("")
			native := sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "project/" + id + ".jsonl"}
			rt.PutFile(sessioninventory.FileEntry{Artifact: native}, []byte("future format, no parser needed"))
			check(id)
			if rt.OperationCountForRoot(sessioninventorytest.OperationReadAt, root.Name) != 0 {
				t.Fatal("resume admission read native body")
			}
			rt.DeleteFile(native)
			rt.AppendFile(ledger, []byte(`{"v":3,"kind":"binding","scope_key":"scope","tag":"work","agent":"`+string(agent)+`","launch_ordinal":1,"root_native_id":"`+id+`","confirmation_reason":"chosen-id"}`+"\n"), "bound")
			check(id)
		})
	}
}
