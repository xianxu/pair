package sessioninventory_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

func TestTitleProjectionWarmPollDoesNotReadHistory(t *testing.T) {
	at := time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)
	rt, transcript, _ := proofBackedCodexFixtureWithGeneration(t, "019d1111-1111-7111-8111-111111111111", &at, "")
	p := sessioninventory.NewTitleProjection(rt, "scope", "work", sessioninventory.AgentCodex, 1)
	if got, err := p.Poll(); err != nil || got.Awaiting || got.Activity != at {
		t.Fatalf("admit=%+v %v", got, err)
	}
	before := rt.OperationCount(sessioninventorytest.OperationReadAt, transcript.StorageRoot+":"+transcript.RelativePath)
	for i := 0; i < 3; i++ {
		if _, err := p.Poll(); err != nil {
			t.Fatal(err)
		}
	}
	if after := rt.OperationCount(sessioninventorytest.OperationReadAt, transcript.StorageRoot+":"+transcript.RelativePath); after != before {
		t.Fatalf("unchanged title reread full transcript: before=%d after=%d", before, after)
	}
}

func TestTitleProjectionSplitAppendAndTerminalMutation(t *testing.T) {
	at := time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)
	rt, a, _ := proofBackedCodexFixtureWithGeneration(t, "019d1111-1111-7111-8111-111111111111", &at, "")
	p := sessioninventory.NewTitleProjection(rt, "scope", "work", sessioninventory.AgentCodex, 1)
	if _, err := p.Poll(); err != nil {
		t.Fatal(err)
	}
	rt.AppendFile(a, []byte(`{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":198000}}}}`), "partial")
	if got, err := p.Poll(); err != nil || got.HasUsage {
		t.Fatalf("partial=%+v %v", got, err)
	}
	rt.AppendFile(a, []byte("\n"), "complete")
	if got, err := p.Poll(); err != nil || !got.HasUsage || got.Usage.InputTokens != 198000 {
		t.Fatalf("complete=%+v %v", got, err)
	}
	rt.TruncateFile(a, 0, "shrink")
	if _, err := p.Poll(); err == nil {
		t.Fatal("shrink did not stop title")
	}
	if _, err := p.Poll(); err == nil {
		t.Fatal("stopped title retried admission")
	}
}

type titleRead struct{ offset, limit int64 }
type titleObservedRuntime struct {
	*sessioninventorytest.FakeRuntime
	transcript sessioninventory.Artifact
	reads      []titleRead
	afterRead  func(int64, int64)
}

func (r *titleObservedRuntime) ReadAt(a sessioninventory.Artifact, offset, limit int64) ([]byte, bool, error) {
	raw, eof, err := r.FakeRuntime.ReadAt(a, offset, limit)
	if a.StorageRoot == r.transcript.StorageRoot && a.RelativePath == r.transcript.RelativePath {
		r.reads = append(r.reads, titleRead{offset, limit})
		if r.afterRead != nil {
			r.afterRead(offset, limit)
		}
	}
	return raw, eof, err
}

func TestTitleProjectionInitialProofAndAppendWork(t *testing.T) {
	at := time.Now()
	fake, a, _ := proofBackedCodexFixtureWithGeneration(t, "019d1111-1111-7111-8111-111111111111", &at, "")
	rt := &titleObservedRuntime{FakeRuntime: fake, transcript: a}
	p := sessioninventory.NewTitleProjection(rt, "scope", "work", sessioninventory.AgentCodex, 1)
	if _, err := p.Poll(); err != nil {
		t.Fatal(err)
	}
	// The proof-only admission requires one initialization read; the other read
	// captures the boundary window. No occupancy scan duplicates the full proof.
	if len(rt.reads) != 2 {
		t.Fatalf("admission reads=%+v, want full+boundary only", rt.reads)
	}
	old, _ := fake.StatFile(a)
	rt.reads = nil
	fake.AppendFile(a, []byte("{\"type\":\"event_msg\",\"payload\":{\"type\":\"token_count\",\"info\":{\"last_token_usage\":{\"input_tokens\":17}}}}\n"), "append")
	got, err := p.Poll()
	if err != nil || got.Usage.InputTokens != 17 {
		t.Fatalf("tail=%+v %v", got, err)
	}
	if len(rt.reads) != 2 || rt.reads[1].offset != old.Size {
		t.Fatalf("warm reads=%+v, want boundary+new suffix", rt.reads)
	}
}

func TestTitleProjectionAwaitsExactInvocationAndRejectsOwnerChanges(t *testing.T) {
	for _, mode := range []string{"delayed", "wrong ordinal", "superseded", "conflict"} {
		t.Run(mode, func(t *testing.T) {
			at := time.Now()
			rt, a, ledger := proofBackedCodexFixtureWithGeneration(t, "019d1111-1111-7111-8111-111111111111", &at, "")
			original, err := rt.ReadFile(ledger, -1)
			if err != nil {
				t.Fatal(err)
			}
			ordinal := uint64(1)
			if mode == "wrong ordinal" {
				ordinal = 4
			}
			p := sessioninventory.NewTitleProjection(rt, "scope", "work", sessioninventory.AgentCodex, ordinal)
			if mode == "delayed" {
				split := bytes.IndexByte(original, '\n') + 1
				entry, _ := rt.StatFile(ledger)
				rt.PutFile(entry, original[:split])
				before := rt.OperationCount(sessioninventorytest.OperationReadAt, a.StorageRoot+":"+a.RelativePath)
				for i := 0; i < 3; i++ {
					got, err := p.Poll()
					if err != nil || !got.Awaiting {
						t.Fatalf("await=%+v %v", got, err)
					}
				}
				if after := rt.OperationCount(sessioninventorytest.OperationReadAt, a.StorageRoot+":"+a.RelativePath); after != before {
					t.Fatal("awaiting performed native proof work")
				}
				rt.PutFile(entry, original)
				if got, err := p.Poll(); err != nil || got.Awaiting {
					t.Fatalf("delayed admission=%+v %v", got, err)
				}
				return
			}
			if mode == "wrong ordinal" {
				if _, err := p.Poll(); err == nil {
					t.Fatal("adopted wrong invocation")
				}
				return
			}
			if _, err := p.Poll(); err != nil {
				t.Fatal(err)
			}
			var appended string
			if mode == "superseded" {
				appended = `{"v":1,"kind":"launch","scope_key":"scope","tag":"work","agent":"codex","pair_log_offset":0,"native_watermarks":[]}`
			} else {
				appended = `{"v":1,"kind":"binding","scope_key":"scope","tag":"work","agent":"codex","launch_ordinal":1,"root_native_id":"another-root"}`
			}
			rt.AppendFile(ledger, []byte(appended+"\n"), sessioninventory.MutationToken(mode))
			if _, err := p.Poll(); err == nil {
				t.Fatalf("%s retained authority", mode)
			}
		})
	}
}

func TestTitleProjectionHashFenceBeforeEveryChunk(t *testing.T) {
	at := time.Now()
	fake, a, _ := proofBackedCodexFixtureWithGeneration(t, "019d1111-1111-7111-8111-111111111111", &at, "")
	rt := &titleObservedRuntime{FakeRuntime: fake, transcript: a}
	p := sessioninventory.NewTitleProjection(rt, "scope", "work", sessioninventory.AgentCodex, 1)
	if _, err := p.Poll(); err != nil {
		t.Fatal(err)
	}
	entry, _ := fake.StatFile(a)
	initial := entry.Size
	// Valid small records span several reads, avoiding a single giant-record case.
	fake.AppendFile(a, []byte(strings.Repeat("{\"type\":\"event_msg\"}\n", 10000)), "append")
	mutated := false
	rt.afterRead = func(offset, limit int64) {
		if !mutated && offset == initial && limit == 64<<10 {
			mutated = true
			current, _ := fake.StatFile(a)
			raw, _ := fake.ReadFile(a, -1)
			raw[initial+limit-2] = 'x' // corrupt the boundary already consumed by chunk1
			fake.PutFile(current, raw)
		}
	}
	rt.reads = nil
	if _, err := p.Poll(); err == nil {
		t.Fatal("between-chunk boundary mutation was accepted")
	}
	if !mutated {
		t.Fatal("mutation hook did not run")
	}
	for _, read := range rt.reads {
		if read.offset == initial+(64<<10) {
			t.Fatal("read second tail chunk after boundary mutation")
		}
	}
}

func TestTitleProjectionAdmissionRejectsSnapshotDrift(t *testing.T) {
	at := time.Now()
	fake, a, _ := proofBackedCodexFixtureWithGeneration(t, "019d1111-1111-7111-8111-111111111111", &at, "")
	rt := &titleObservedRuntime{FakeRuntime: fake, transcript: a}
	// Full proof read is64KiB; the next read initializes the boundary. Mutating
	// after that read must not pin a snapshot different from the accepted proof.
	rt.afterRead = func(_, limit int64) {
		if limit < 64<<10 {
			fake.AppendFile(a, []byte("{}\n"), "race")
			rt.afterRead = nil
		}
	}
	p := sessioninventory.NewTitleProjection(rt, "scope", "work", sessioninventory.AgentCodex, 1)
	if _, err := p.Poll(); err == nil {
		t.Fatal("admitted boundary from changed proof snapshot")
	}
}

func TestTitleProjectionRejectsTailContradictions(t *testing.T) {
	for _, mode := range []string{"replace", "rewrite", "malformed", "hash mutation", "partial overflow"} {
		t.Run(mode, func(t *testing.T) {
			at := time.Now()
			rt, a, _ := proofBackedCodexFixtureWithGeneration(t, "019d1111-1111-7111-8111-111111111111", &at, "")
			p := sessioninventory.NewTitleProjection(rt, "scope", "work", sessioninventory.AgentCodex, 1)
			if _, err := p.Poll(); err != nil {
				t.Fatal(err)
			}
			entry, _ := rt.StatFile(a)
			raw, _ := rt.ReadFile(a, -1)
			switch mode {
			case "replace":
				entry.StableFileID = "another-inode"
				rt.PutFile(entry, raw)
			case "rewrite":
				entry.MutationToken = "rewrite"
				rt.PutFile(entry, raw)
			case "malformed":
				rt.AppendFile(a, []byte("{broken}\n"), "bad")
			case "hash mutation":
				raw[len(raw)-2] = 'x'
				rt.PutFile(entry, raw)
				rt.AppendFile(a, []byte("{}\n"), "append")
			case "partial overflow":
				rt.AppendFile(a, []byte(strings.Repeat(" ", (32<<20)+1)), "overflow")
			}
			if _, err := p.Poll(); err == nil {
				t.Fatalf("%s did not stop", mode)
			}
		})
	}
}

func titleProviderFixture(t *testing.T, agent sessioninventory.Agent) (*sessioninventorytest.FakeRuntime, sessioninventory.Artifact, sessioninventory.Artifact, []byte) {
	t.Helper()
	const id = "019d1111-1111-7111-8111-111111111111"
	rt := sessioninventorytest.NewFakeRuntime()
	at := time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)
	var a sessioninventory.Artifact
	var initial, tail string
	switch agent {
	case sessioninventory.AgentCodex:
		a = sessioninventory.Artifact{StorageRoot: "codex-sessions", RelativePath: "2026/10/16/rollout-test-" + id + ".jsonl"}
		initial = `{"timestamp":"2026-10-16T00:00:00Z","type":"session_meta","payload":{"id":"` + id + `","parent_thread_id":null,"source":"cli"}}`
		tail = `{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":201000}}}}`
	case sessioninventory.AgentClaude, sessioninventory.AgentQoder:
		name := "claude-projects"
		if agent == sessioninventory.AgentQoder {
			name = "qoder-projects"
		}
		a = sessioninventory.Artifact{StorageRoot: name, RelativePath: "project/" + id + ".jsonl"}
		initial = `{"type":"user","sessionId":"` + id + `","timestamp":"2026-10-16T00:00:00Z"}`
		tail = `{"type":"assistant","sessionId":"` + id + `","message":{"model":"real","usage":{"input_tokens":200000,"cache_creation_input_tokens":900,"cache_read_input_tokens":100}}}`
	case sessioninventory.AgentMuse:
		a = sessioninventory.Artifact{StorageRoot: "muse-sessions", RelativePath: "2026/10/16/" + id + "/session.jsonl"}
		initial = `{"payload_type":"runtime.session","payload":{"kind":"run","run_id":"` + id + `","event":{"kind":"started"}}}`
		tail = `{"payload_type":"runtime.session","payload":{"kind":"run","run_id":"` + id + `","event":{"kind":"completed"}},"usage":{"input_tokens":201000}}`
	case sessioninventory.AgentGrok:
		a = sessioninventory.Artifact{StorageRoot: "grok-sessions", RelativePath: "%2Frepo/" + id + "/updates.jsonl"}
		initial = string(grokRecord(id, "user_message_chunk"))
		tail = `{"timestamp":1791400001,"method":"session/update","params":{"sessionId":"` + id + `","update":{"sessionUpdate":"agent_message_chunk"},"_meta":{"totalTokens":201000}}}`
	case sessioninventory.AgentAgy:
		var database, transcript sessioninventory.FileEntry
		rt, database, transcript = incrementalAgyFixture(id)
		a = transcript.Artifact
		database.StableFileID = "database"
		database.MutationToken = "initial"
		database.ModTime = &at
		rt.PutFile(database, []byte("SQLite format 3\x00fixture"))
		rt.AddRoot(sessioninventory.StorageRoot{Agent: agent, Name: database.Artifact.StorageRoot, Path: "/native/database"})
		initial = `{"type":"USER_INPUT","content":"prompt"}`
		tail = `{"type":"PLANNER_RESPONSE","content":"answer","usage":{"input_tokens":201000}}`
	default:
		t.Fatalf("unsupported fixture %s", agent)
	}
	rt.AddRoot(sessioninventory.StorageRoot{Agent: agent, Name: a.StorageRoot, Path: "/native/transcript"})
	rt.PutFile(sessioninventory.FileEntry{Artifact: a, StableFileID: "transcript", MutationToken: "initial", ModTime: &at}, []byte(initial+"\n"))
	root := sessioninventory.StorageRoot{Name: "pair-data", Path: "/pair/scope"}
	rt.SetPairDataRoot(root)
	ledger := sessioninventory.Artifact{StorageRoot: root.Name, RelativePath: "ledger-work.jsonl"}
	launch, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 3, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: string(agent), RequestedNativeID: id, RequestOrigin: sessionledger.RequestOriginChosen, BaselineComplete: true, LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{}})
	if err != nil {
		t.Fatal(err)
	}
	binding, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 3, Kind: sessionledger.RecordBinding, ScopeKey: "scope", Tag: "work", Agent: string(agent), LaunchOrdinal: 1, RootNativeID: id, ConfirmationReason: sessionledger.ConfirmationChosen})
	if err != nil {
		t.Fatal(err)
	}
	rt.PutFile(sessioninventory.FileEntry{Artifact: ledger, StableFileID: "ledger", MutationToken: "initial"}, append(append(launch, '\n'), append(binding, '\n')...))
	return rt, a, ledger, []byte(tail + "\n")
}

func TestTitleProjectionProviderAppendParity(t *testing.T) {
	for _, agent := range sessioninventory.SupportedAgents() {
		t.Run(string(agent), func(t *testing.T) {
			rt, a, _, tail := titleProviderFixture(t, agent)
			p := sessioninventory.NewTitleProjection(rt, "scope", "work", agent, 1)
			first, err := p.Poll()
			if err != nil || first.Awaiting {
				t.Fatalf("initial=%+v %v", first, err)
			}
			rt.AppendFile(a, tail, "append")
			got, err := p.Poll()
			if err != nil {
				t.Fatal(err)
			}
			all, err := rt.ReadFile(a, -1)
			if err != nil {
				t.Fatal(err)
			}
			usage, ok := sessioninventory.TokenUsageFromJSONL(agent, all)
			if got.HasUsage != ok || got.Usage != usage {
				t.Fatalf("usage parity got=%+v want=%+v,%v", got, usage, ok)
			}
			if agent == sessioninventory.AgentCodex || agent == sessioninventory.AgentClaude || agent == sessioninventory.AgentGrok {
				if !ok || usage.InputTokens != 201000 {
					t.Fatalf("usage fixture disconnected: %+v %v", usage, ok)
				}
			} else if got.HasUsage {
				t.Fatal("invented unsupported usage")
			}
			// Every provider must reject malformed complete suffixes.
			rt.AppendFile(a, []byte("{malformed}\n"), "invalid")
			if _, err := p.Poll(); err == nil {
				t.Fatal("malformed provider tail accepted")
			}
		})
	}
	unknown := sessioninventory.NewTitleProjection(sessioninventorytest.NewFakeRuntime(), "scope", "work", sessioninventory.Agent("unknown"), 1)
	if _, err := unknown.Poll(); err == nil {
		t.Fatal("unknown provider admitted")
	}
}

func TestTitleProjectionAgyDatabaseMutationIsTerminal(t *testing.T) {
	rt, _, _, _ := titleProviderFixture(t, sessioninventory.AgentAgy)
	p := sessioninventory.NewTitleProjection(rt, "scope", "work", sessioninventory.AgentAgy, 1)
	if _, err := p.Poll(); err != nil {
		t.Fatal(err)
	}
	root := rt.NativeRoots(sessioninventory.AgentAgy)[0]
	files, err := rt.ListFiles(root)
	if err != nil || len(files) != 1 {
		t.Fatalf("db fixture %v", err)
	}
	rt.AppendFile(files[0].Artifact, []byte("changed"), "db mutation")
	if _, err := p.Poll(); err == nil {
		t.Fatal("nonappend-only database mutation retained authority")
	}
}

func TestTitleProjectionRetainedStateIndependentOfHistory(t *testing.T) {
	rt, a, _, tail := titleProviderFixture(t, sessioninventory.AgentClaude)
	p := sessioninventory.NewTitleProjection(rt, "scope", "work", sessioninventory.AgentClaude, 1)
	if _, err := p.Poll(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 200; i++ {
		rt.AppendFile(a, bytes.Repeat(tail, 100), sessioninventory.MutationToken(fmt.Sprintf("append-%d", i)))
		if _, err := p.Poll(); err != nil {
			t.Fatal(err)
		}
		state := reflect.ValueOf(p).Elem()
		if n := state.FieldByName("boundary").Cap(); n > 4096 {
			t.Fatalf("retained boundary capacity=%d", n)
		}
		if n := state.FieldByName("frame").FieldByName("IncompleteTail").Len(); n != 0 {
			t.Fatalf("completed record tail retained %d bytes", n)
		}
		// The filesystem double intentionally retains the native file contents;
		// measure the projection's own retained state independently of that IO.
		var retained func(reflect.Value) int
		retained = func(v reflect.Value) int {
			switch v.Kind() {
			case reflect.Struct:
				n := 0
				for j := 0; j < v.NumField(); j++ {
					name := v.Type().Field(j).Name
					if name != "runtime" && name != "ownerRuntime" {
						n += retained(v.Field(j))
					}
				}
				return n
			case reflect.Pointer:
				if !v.IsNil() {
					return retained(v.Elem())
				}
			case reflect.String:
				return v.Len()
			case reflect.Slice:
				n := v.Cap() * int(v.Type().Elem().Size())
				for j := 0; j < v.Len(); j++ {
					n += retained(v.Index(j))
				}
				return n
			}
			return int(v.Type().Size())
		}
		if n := retained(state); n > 64<<10 {
			t.Fatalf("projection retained %d bytes after append %d", n, i)
		}
	}
}

func TestTitleProjectionReusesFullAdmissionRecords(t *testing.T) {
	at := time.Now()
	fake, a, _ := proofBackedCodexFixtureWithGeneration(t, "019d1111-1111-7111-8111-111111111111", &at, "")
	// Drift without generation takes the existing authorized full-proof fallback.
	fake.AppendFile(a, []byte("{\"type\":\"event_msg\"}\n"), "before-start")
	rt := &titleObservedRuntime{FakeRuntime: fake, transcript: a}
	p := sessioninventory.NewTitleProjection(rt, "scope", "work", sessioninventory.AgentCodex, 1)
	if _, err := p.Poll(); err != nil {
		t.Fatal(err)
	}
	if len(rt.reads) != 2 {
		t.Fatalf("full proof initialization duplicated transcript scan: %+v", rt.reads)
	}
}

func FuzzTitleProjectionSplitUsageParity(f *testing.F) {
	for _, step := range []uint16{1, 17, 64, 256} {
		f.Add(step)
	}
	f.Fuzz(func(t *testing.T, step uint16) {
		at := time.Now()
		rt, a, _ := proofBackedCodexFixtureWithGeneration(t, "019d1111-1111-7111-8111-111111111111", &at, "")
		p := sessioninventory.NewTitleProjection(rt, "scope", "work", sessioninventory.AgentCodex, 1)
		if _, err := p.Poll(); err != nil {
			t.Fatal(err)
		}
		row := func(n int) string {
			return fmt.Sprintf(`{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":%d}}}}`, n)
		}
		tail := []byte(row(17) + "\r\n" + row(19) + "\n" + row(-1) + "\n")
		stride := 1 + int(step)%len(tail)
		var got sessioninventory.TitleProjectionResult
		for pos := 0; pos < len(tail); pos += stride {
			rt.AppendFile(a, tail[pos:min(len(tail), pos+stride)], sessioninventory.MutationToken(fmt.Sprintf("append-%d", pos)))
			var err error
			got, err = p.Poll()
			if err != nil {
				t.Fatal(err)
			}
		}
		want, ok := sessioninventory.TokenUsageFromJSONL(sessioninventory.AgentCodex, tail)
		if got.Usage != want || got.HasUsage != ok {
			t.Fatalf("split usage=%+v want=%+v,%v", got, want, ok)
		}
	})
}

func TestTitleProjectionRejectsOwnerSupersessionDuringTail(t *testing.T) {
	at := time.Now()
	fake, a, ledger := proofBackedCodexFixtureWithGeneration(t, "019d1111-1111-7111-8111-111111111111", &at, "")
	rt := &titleObservedRuntime{FakeRuntime: fake, transcript: a}
	p := sessioninventory.NewTitleProjection(rt, "scope", "work", sessioninventory.AgentCodex, 1)
	if _, err := p.Poll(); err != nil {
		t.Fatal(err)
	}
	entry, _ := fake.StatFile(a)
	fake.AppendFile(a, []byte("{\"type\":\"event_msg\",\"payload\":{\"type\":\"token_count\",\"info\":{\"last_token_usage\":{\"input_tokens\":17}}}}\n"), "tail")
	superseded := false
	rt.afterRead = func(offset, _ int64) {
		if offset == entry.Size && !superseded {
			superseded = true
			fake.AppendFile(ledger, []byte("{\"v\":1,\"kind\":\"launch\",\"scope_key\":\"scope\",\"tag\":\"work\",\"agent\":\"codex\",\"pair_log_offset\":0,\"native_watermarks\":[]}\n"), "superseded")
		}
	}
	if got, err := p.Poll(); err == nil || got.HasUsage || !got.Activity.IsZero() {
		t.Fatalf("published superseded tail result=%+v err=%v", got, err)
	}
	if !superseded {
		t.Fatal("supersession hook disconnected")
	}
}

// Admission retains the full validator's complete-snapshot requirement. Partial
// framing is permitted only after a successful admission of existing authority.
func TestTitleProjectionAdmissionPreservesFullValidatorRules(t *testing.T) {
	at := time.Now()
	rt, a, ledger := proofBackedCodexFixtureWithGeneration(t, "019d1111-1111-7111-8111-111111111111", &at, "")
	raw, err := rt.ReadFile(ledger, -1)
	if err != nil {
		t.Fatal(err)
	}
	current, ok := sessionledger.CurrentLaunch(sessionledger.ParseLedger(raw).Records, sessionledger.Owner{ScopeKey: "scope", Tag: "work", Agent: "codex"})
	if !ok || current.Binding == nil || current.Binding.AuthorizationProof == nil {
		t.Fatal("missing proof control")
	}
	rt.AppendFile(a, []byte(`{"type":"event_msg"}`), "partial-at-admission")
	_, _, oracleErr := sessioninventory.NewIncrementalInventory(rt, sessioninventory.Catalog{Version: sessioninventory.CatalogVersion}).ValidateBindingProof(sessioninventory.AgentCodex, *current.Binding.AuthorizationProof)
	if oracleErr == nil {
		t.Fatal("full-validator control unexpectedly accepts partial snapshot")
	}
	p := sessioninventory.NewTitleProjection(rt, "scope", "work", sessioninventory.AgentCodex, 1)
	if _, err := p.Poll(); err == nil {
		t.Fatal("title weakened initial authorization")
	}
	rt.AppendFile(a, []byte("\n"), "complete-admission-tail")
	if _, err := p.Poll(); err == nil {
		t.Fatal("stopped admission retried proof")
	}
	if _, err := sessioninventory.NewTitleProjection(rt, "scope", "work", sessioninventory.AgentCodex, 1).Poll(); err != nil {
		t.Fatalf("fresh complete admission control: %v", err)
	}
}

func TestCurrentTitleLaunchExactCachedOwner(t *testing.T) {
	rt, ledger := launchCacheFixture()
	got, ok, err := sessioninventory.CurrentTitleLaunch(context.Background(), rt, "scope", "work", sessioninventory.AgentCodex)
	if err != nil || !ok || got.Ordinal != 1 || got.Agent != sessioninventory.AgentCodex {
		t.Fatalf("current=%+v %v %v", got, ok, err)
	}
	reads := rt.OperationCount(sessioninventorytest.OperationReadAt, "pair-data:ledger-work.jsonl")
	if _, ok, err := sessioninventory.CurrentTitleLaunch(context.Background(), rt, "scope", "work", sessioninventory.AgentCodex); err != nil || !ok {
		t.Fatalf("cached=%v %v", ok, err)
	}
	if n := rt.OperationCount(sessioninventorytest.OperationReadAt, "pair-data:ledger-work.jsonl"); n != reads {
		t.Fatalf("cached read %d -> %d", reads, n)
	}
	rt.AppendFile(ledger, []byte(strings.ReplaceAll(v1Launch, "codex", "claude")), "new-agent")
	if got, ok, _ := sessioninventory.CurrentTitleLaunch(context.Background(), rt, "scope", "work", sessioninventory.AgentCodex); ok {
		t.Fatalf("stale launch accepted: %+v", got)
	}
	if got, ok, err := sessioninventory.CurrentTitleLaunch(context.Background(), rt, "scope", "work", sessioninventory.AgentClaude); err != nil || !ok || got.Ordinal != 2 {
		t.Fatalf("new agent=%+v %v %v", got, ok, err)
	}
	rt.AppendFile(ledger, []byte("{}\n"), "untyped")
	if got, ok, _ := sessioninventory.CurrentTitleLaunch(context.Background(), rt, "scope", "work", sessioninventory.AgentClaude); ok {
		t.Fatalf("untyped latest accepted: %+v", got)
	}
}

func TestTitleProjectionPollContextCancellationResumes(t *testing.T) {
	at := time.Now()
	fake, a, _ := proofBackedCodexFixtureWithGeneration(t, "019d1111-1111-7111-8111-111111111111", &at, "")
	rt := &titleObservedRuntime{FakeRuntime: fake, transcript: a}
	p := sessioninventory.NewTitleProjection(rt, "scope", "work", sessioninventory.AgentCodex, 1)
	ctx, cancel := context.WithCancel(context.Background())
	rt.afterRead = func(_, _ int64) { cancel() }
	if _, err := p.PollContext(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("admission cancellation=%v", err)
	}
	rt.afterRead = nil
	if _, err := p.Poll(); err != nil {
		t.Fatalf("resume admission=%v", err)
	}
	fake.AppendFile(a, []byte(`{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"input_tokens":117}}}}`+"\n"), "append")
	ctx, cancel = context.WithCancel(context.Background())
	rt.afterRead = func(offset, _ int64) {
		if offset > 0 {
			cancel()
		}
	}
	if _, err := p.PollContext(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("tail cancellation=%v", err)
	}
	rt.afterRead = nil
	if got, err := p.Poll(); err != nil || !got.HasUsage || got.Usage.InputTokens != 117 {
		t.Fatalf("resume tail=%+v %v", got, err)
	}
}

type titleSQLiteRuntime struct {
	*sessioninventorytest.FakeRuntime
	entered chan struct{}
}

func (r *titleSQLiteRuntime) QuerySQLiteContext(ctx context.Context, a sessioninventory.Artifact, q string, n int64) (sessioninventory.SQLiteResult, error) {
	if r.entered != nil {
		close(r.entered)
		<-ctx.Done()
		return sessioninventory.SQLiteResult{}, ctx.Err()
	}
	return r.FakeRuntime.QuerySQLite(a, q, n)
}
func TestTitleProjectionPollContextCancelsSQLiteAdmission(t *testing.T) {
	fake, _, _, _ := titleProviderFixture(t, sessioninventory.AgentAgy)
	rt := &titleSQLiteRuntime{FakeRuntime: fake, entered: make(chan struct{})}
	p := sessioninventory.NewTitleProjection(rt, "scope", "work", sessioninventory.AgentAgy, 1)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := p.PollContext(ctx); done <- err }()
	select {
	case <-rt.entered:
	case <-time.After(2 * time.Second):
		cancel()
		t.Fatal("SQLite context seam not called")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("SQLite cancellation=%v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("SQLite cancellation blocked")
	}
	rt.entered = nil
	if got, err := p.Poll(); err != nil || got.Awaiting {
		t.Fatalf("SQLite resume=%+v %v", got, err)
	}
}

func TestCurrentTitleLaunchRejectsConflictingGeneration(t *testing.T) {
	rt, _, ledger, _ := titleProviderFixture(t, sessioninventory.AgentCodex)
	binding, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 3, Kind: sessionledger.RecordBinding, ScopeKey: "scope", Tag: "work", Agent: "codex", LaunchOrdinal: 1, RootNativeID: "019d2222-2222-7222-8222-222222222222", ConfirmationReason: sessionledger.ConfirmationChosen})
	if err != nil {
		t.Fatal(err)
	}
	rt.AppendFile(ledger, append(binding, '\n'), "conflict")
	if got, ok, err := sessioninventory.CurrentTitleLaunch(context.Background(), rt, "scope", "work", sessioninventory.AgentCodex); err != nil || ok {
		t.Fatalf("conflict=%+v %v %v", got, ok, err)
	}
}
func TestCurrentTitleLaunchDoesNotFallBackToEstablishedGeneration(t *testing.T) {
	rt, _, ledger, _ := titleProviderFixture(t, sessioninventory.AgentCodex)
	launch, err := sessionledger.EncodeRecord(sessionledger.Record{Version: 3, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "codex", RequestedNativeID: "019d2222-2222-7222-8222-222222222222", RequestOrigin: sessionledger.RequestOriginChosen, BaselineComplete: true, LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{}})
	if err != nil {
		t.Fatal(err)
	}
	rt.AppendFile(ledger, append(launch, '\n'), "new-fresh")
	got, ok, err := sessioninventory.CurrentTitleLaunch(context.Background(), rt, "scope", "work", sessioninventory.AgentCodex)
	if err != nil || !ok || got.Ordinal != 3 {
		t.Fatalf("fresh current=%+v %v %v", got, ok, err)
	}
	if n := rt.OperationCount(sessioninventorytest.OperationListFiles, "codex-sessions"); n != 0 {
		t.Fatalf("current launch scanned native storage %d times", n)
	}
}

func TestTitleProjectionPollContextRejectsNilRuntime(t *testing.T) {
	p := sessioninventory.NewTitleProjection(nil, "scope", "work", sessioninventory.AgentCodex, 1)
	if _, err := p.PollContext(context.Background()); err == nil {
		t.Fatal("nil runtime accepted")
	}
}
