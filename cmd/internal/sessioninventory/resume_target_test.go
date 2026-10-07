package sessioninventory_test

import (
	"errors"
	"fmt"
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

type incompleteNativeListing struct {
	*sessioninventorytest.FakeRuntime
	partial bool
}

func (r incompleteNativeListing) ListFiles(root sessioninventory.StorageRoot) ([]sessioninventory.FileEntry, error) {
	files, err := r.FakeRuntime.ListFiles(root)
	if root.Name != "claude-projects" {
		return files, err
	}
	if r.partial {
		return []sessioninventory.FileEntry{{Artifact: sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "project/22222222-2222-4222-8222-222222222222.jsonl"}}}, errors.New("EIO: partial listing")
	}
	return nil, errors.New("EIO: failed listing")
}
func TestChosenTargetListingFailureIsUnknown(t *testing.T) {
	for _, partial := range []bool{false, true} {
		t.Run(fmt.Sprint(partial), func(t *testing.T) {
			rt := sessioninventorytest.NewFakeRuntime()
			root := sessioninventory.StorageRoot{Name: "claude-projects", Agent: sessioninventory.AgentClaude, Path: "/native"}
			rt.AddRoot(root)
			rt.SetPairDataRoot(sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair"})
			rt.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: "pair-data", RelativePath: "ledger-work.jsonl"}}, []byte(`{"v":3,"kind":"launch","scope_key":"scope","tag":"work","agent":"claude","pair_log_offset":0,"artifact_boundaries":[],"requested_native_id":"11111111-1111-4111-8111-111111111111","request_origin":"chosen-id","baseline_complete":true}`+"\n"))
			rt.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "project/11111111-1111-4111-8111-111111111111.jsonl"}}, []byte("native body"))
			got, err := sessioninventory.QueryResumeTarget(incompleteNativeListing{rt, partial}, "scope", "work", sessioninventory.AgentClaude)
			if err != nil || got.NativeID != "" || got.FreshRequired || len(got.Diagnostics) == 0 {
				t.Fatalf("unknown probe treated as usable: %+v %v", got, err)
			}
			rt.SetError(sessioninventorytest.OperationListFiles, root.Name, errors.New("EIO: unrelated entry unreadable"))
			got, err = sessioninventory.QueryResumeTarget(rt, "scope", "work", sessioninventory.AgentClaude)
			if err != nil || got.NativeID != "11111111-1111-4111-8111-111111111111" || got.FreshRequired {
				t.Fatalf("observed root lost to unrelated diagnostic: %+v %v", got, err)
			}
		})
	}
}

// pair#214 D1: an in-pane fresh restart whose agent never took a turn does not
// hide the conversation it replaced. A complete listing that proves the chosen
// file absent makes the owner query fall back to the previous generation.
func TestAnUnturnedFreshLaunchDoesNotHideThePreviousConversation(t *testing.T) {
	const old = "9a99ff57-cb0f-4590-8ea7-d684f2ff5a3f"
	const fresh = "11111111-1111-4111-8111-111111111111"
	setup := func(t *testing.T) (*sessioninventorytest.FakeRuntime, sessioninventory.StorageRoot, sessioninventory.Artifact) {
		rt := sessioninventorytest.NewFakeRuntime()
		pair := sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair"}
		rt.SetPairDataRoot(pair)
		root := sessioninventory.StorageRoot{Name: "claude-projects", Path: "/native", Agent: sessioninventory.AgentClaude}
		rt.AddRoot(root)
		ledger := sessioninventory.Artifact{StorageRoot: pair.Name, RelativePath: "ledger-work.jsonl"}
		rows := `{"v":3,"kind":"launch","scope_key":"scope","tag":"work","agent":"claude","pair_log_offset":0,"artifact_boundaries":[],"requested_native_id":"` + old + `","request_origin":"chosen-id","baseline_complete":true}` + "\n" +
			`{"v":3,"kind":"binding","scope_key":"scope","tag":"work","agent":"claude","launch_ordinal":1,"root_native_id":"` + old + `","confirmation_reason":"chosen-id"}` + "\n" +
			`{"v":3,"kind":"launch","scope_key":"scope","tag":"work","agent":"claude","pair_log_offset":0,"artifact_boundaries":[],"requested_native_id":"` + fresh + `","request_origin":"chosen-id","baseline_complete":true}` + "\n"
		rt.PutFile(sessioninventory.FileEntry{Artifact: ledger}, []byte(rows))
		rt.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "project/" + old + ".jsonl"}}, []byte("old conversation"))
		return rt, root, ledger
	}

	t.Run("fresh file proven absent: resume the replaced conversation", func(t *testing.T) {
		rt, _, _ := setup(t)
		got, err := sessioninventory.QueryResumeTarget(rt, "scope", "work", sessioninventory.AgentClaude)
		if err != nil || got.Status != sessioninventory.BindingEstablished || got.NativeID != old || got.FreshRequired || got.FellBackFrom != 3 {
			t.Fatalf("target = %+v, %v; want the old conversation, established, fallen back from launch 3", got, err)
		}
	})
	t.Run("fresh agent took a turn: its own conversation", func(t *testing.T) {
		rt, root, _ := setup(t)
		rt.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "project/" + fresh + ".jsonl"}}, []byte("fresh turn"))
		got, err := sessioninventory.QueryResumeTarget(rt, "scope", "work", sessioninventory.AgentClaude)
		if err != nil || got.NativeID != fresh || got.FellBackFrom != 0 {
			t.Fatalf("target = %+v, %v; want the fresh conversation", got, err)
		}
	})
	t.Run("incomplete listing: no fallback on missing evidence", func(t *testing.T) {
		rt, _, _ := setup(t)
		got, err := sessioninventory.QueryResumeTarget(incompleteNativeListing{FakeRuntime: rt, partial: true}, "scope", "work", sessioninventory.AgentClaude)
		if err != nil || got.Status != sessioninventory.BindingProvisional || got.FellBackFrom != 0 {
			t.Fatalf("target = %+v, %v; want provisional, no fallback", got, err)
		}
	})
	t.Run("no earlier conversation: still fresh-required", func(t *testing.T) {
		rt := sessioninventorytest.NewFakeRuntime()
		pair := sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair"}
		rt.SetPairDataRoot(pair)
		rt.AddRoot(sessioninventory.StorageRoot{Name: "claude-projects", Path: "/native", Agent: sessioninventory.AgentClaude})
		rt.PutFile(sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: pair.Name, RelativePath: "ledger-work.jsonl"}},
			[]byte(`{"v":3,"kind":"launch","scope_key":"scope","tag":"work","agent":"claude","pair_log_offset":0,"artifact_boundaries":[],"requested_native_id":"`+fresh+`","request_origin":"chosen-id","baseline_complete":true}`+"\n"))
		got, err := sessioninventory.QueryResumeTarget(rt, "scope", "work", sessioninventory.AgentClaude)
		if err != nil || !got.FreshRequired || got.FellBackFrom != 0 {
			t.Fatalf("target = %+v, %v; want fresh-required, no fallback", got, err)
		}
	})
}
