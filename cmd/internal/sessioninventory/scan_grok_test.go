package sessioninventory_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/sessioninventory"
	"github.com/xianxu/pair/cmd/internal/sessioninventorytest"
)

const grokFixtureID = "11111111-1111-4111-8111-111111111111"

// The fixture is a sanitized real grok 1.0.46 TUI turn plus a tool-using turn
// re-keyed to the same session, a stub session (summary.json and no
// updates.jsonl, as `grok login` leaves behind), and the sibling files a real
// session directory carries. Only the transcript is an artifact; the stub and
// the siblings scan silently.
func TestScanGrokV1(t *testing.T) {
	t.Parallel()
	runtime := sessioninventorytest.NewFakeRuntime()
	loadNativeFixture(t, runtime, sessioninventory.AgentGrok, "grok-sessions", filepath.Join("testdata", "native", "grok", "v1", "grok-sessions"))
	got := inventoryFromScan(sessioninventory.ScanGrok(runtime))

	if len(got.Forests) != 1 || len(got.Forests[0].Roots) != 1 {
		t.Fatalf("forests = %#v", got.Forests)
	}
	root := got.Forests[0].Roots[0]
	if root.NativeID != grokFixtureID || !root.Resumable {
		t.Fatalf("root = %#v", root)
	}
	if root.Time == nil || root.Time.Source != sessioninventory.TimeSourceMetadata || !root.Time.Value.Equal(time.Unix(1791400000, 0).UTC()) {
		t.Fatalf("root chronology = %#v, want the first record's timestamp", root.Time)
	}
	if len(got.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want the stub and sibling files to scan clean", got.Diagnostics)
	}
}

func grokEntry(relative string) sessioninventory.FileEntry {
	return sessioninventory.FileEntry{Artifact: sessioninventory.Artifact{StorageRoot: "grok-sessions", RelativePath: relative}}
}

func grokRecord(sessionID, kind string) []byte {
	return []byte(`{"timestamp":1791400000,"method":"session/update","params":{"sessionId":"` + sessionID + `","update":{"sessionUpdate":"` + kind + `","content":{"type":"text","text":"x"}}}}`)
}

// Every record carries the session id; a record naming another session, or one
// that is not the envelope, disputes the transcript.
func TestValidateGrokDelta(t *testing.T) {
	t.Parallel()
	entry := grokEntry("%2Frepo/" + grokFixtureID + "/updates.jsonl")
	state, _, err := sessioninventory.ValidateGrokDelta(entry, nil, []sessioninventory.FramedJSONLRecord{{Bytes: grokRecord(grokFixtureID, "user_message_chunk")}})
	if err != nil || state.Disputed || !state.FirstRecordValidated || state.Role != sessioninventory.RoleRoot {
		t.Fatalf("state=%#v err=%v", state, err)
	}
	for name, record := range map[string][]byte{
		"foreign session id": grokRecord("99999999-9999-4999-8999-999999999999", "agent_message_chunk"),
		"malformed json":     []byte(`{"timestamp":1791400000,"method":`),
		"missing session id": []byte(`{"timestamp":1791400000,"method":"session/update","params":{"update":{"sessionUpdate":"plan"}}}`),
		"unknown method":     []byte(`{"timestamp":1791400000,"method":"session/request","params":{"sessionId":"` + grokFixtureID + `","update":{"sessionUpdate":"plan"}}}`),
	} {
		got, diagnostics, err := sessioninventory.ValidateGrokDelta(entry, &state, []sessioninventory.FramedJSONLRecord{{Bytes: record}})
		if err != nil || !got.Disputed || len(diagnostics) == 0 {
			t.Errorf("%s: state=%#v diagnostics=%#v err=%v", name, got, diagnostics, err)
		}
	}
	for _, relative := range []string{
		"%2Frepo/" + grokFixtureID + "/subagents/" + grokFixtureID + "/updates.jsonl",
		"%2Frepo/not-a-uuid/updates.jsonl",
		"repo/" + grokFixtureID + "/updates.jsonl", // cwd dir must be an encoded absolute path
		"%2Frepo/" + grokFixtureID + "/events.jsonl",
	} {
		if _, _, err := sessioninventory.ValidateGrokDelta(grokEntry(relative), nil, nil); err == nil {
			t.Errorf("%s: accepted as a grok transcript", relative)
		}
	}
}

// The chronology path is bounded like qoder's (BR-19): an epoch-seconds value
// outside 2000-01-01..9999-12-31 disputes the record instead of fabricating a
// time the rest of the pipeline cannot marshal.
func TestGrokTimestampBounds(t *testing.T) {
	t.Parallel()
	entry := grokEntry("%2Frepo/" + grokFixtureID + "/updates.jsonl")
	for _, ts := range []string{"9223372036854775807", "-62135596800", "1.5", `"2026-10-08"`} {
		record := []byte(`{"timestamp":` + ts + `,"method":"session/update","params":{"sessionId":"` + grokFixtureID + `","update":{"sessionUpdate":"plan"}}}`)
		state, _, err := sessioninventory.ValidateGrokDelta(entry, nil, []sessioninventory.FramedJSONLRecord{{Bytes: record}})
		if err != nil || !state.Disputed {
			t.Errorf("timestamp %s: state=%#v err=%v, want disputed", ts, state, err)
		}
	}
}
