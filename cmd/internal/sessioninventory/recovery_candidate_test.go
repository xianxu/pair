package sessioninventory

import (
	"strings"
	"testing"
	"time"
)

const recoveryTestID = "019d1111-1111-7111-8111-111111111111"
const recoveryTestText = "one two three four five six seven eight nine ten eleven twelve thirteen fourteen fifteen sixteen seventeen eighteen"

func recoveryRecords(lines ...string) []FramedJSONLRecord {
	var out []FramedJSONLRecord
	for _, line := range lines {
		out = append(out, FramedJSONLRecord{Bytes: []byte(line)})
	}
	return out
}
func recoveryCodexEntry() FileEntry {
	return FileEntry{Artifact: Artifact{RelativePath: "2026/08/28/rollout-root-" + recoveryTestID + ".jsonl"}, StableFileID: "inode", GenerationToken: "generation", MutationToken: "stamp"}
}
func recoveryCodexRecords(source string) []FramedJSONLRecord {
	return recoveryRecords(`{"timestamp":"2026-08-28T10:00:00Z","type":"session_meta","payload":{"id":"`+recoveryTestID+`","source":`+source+`,"cwd":"/repo","originator":"codex-tui"}}`, `{"timestamp":"2026-08-28T10:02:00Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"`+recoveryTestText+`"}]}}`)
}
func TestRecoveryCodexEligibility(t *testing.T) {
	for _, tc := range []struct {
		source string
		yes    bool
	}{{`"cli"`, true}, {`"exec"`, false}, {`null`, false}} {
		t.Run(tc.source, func(t *testing.T) {
			c, _, err := ParseRecoveryCandidate(AgentCodex, recoveryCodexEntry(), recoveryCodexRecords(tc.source), nil)
			if err != nil {
				t.Fatal(err)
			}
			if (c.Interactive.State == InteractiveYes) != tc.yes {
				t.Fatalf("interactive=%+v", c.Interactive)
			}
			if c.CWD != "/repo" || len(c.Messages) != 1 || c.Messages[0].Text != recoveryTestText || c.SemanticLast.IsZero() {
				t.Fatalf("candidate=%+v", c)
			}
		})
	}
}
func TestRecoveryCodexRejectsMismatchMalformedAndConflicts(t *testing.T) {
	for _, change := range []func([]FramedJSONLRecord) []FramedJSONLRecord{
		func(r []FramedJSONLRecord) []FramedJSONLRecord {
			r[0].Bytes = []byte(strings.ReplaceAll(string(r[0].Bytes), recoveryTestID, "019d2222-2222-7222-8222-222222222222"))
			return r
		},
		func(r []FramedJSONLRecord) []FramedJSONLRecord {
			return append(r, FramedJSONLRecord{Bytes: []byte(`{"type":`)})
		},
		func(r []FramedJSONLRecord) []FramedJSONLRecord { return append(r, recoveryCodexRecords(`"exec"`)[0]) },
		func(r []FramedJSONLRecord) []FramedJSONLRecord {
			r[0].Bytes = []byte(strings.Replace(string(r[0].Bytes), `"source":"cli"`, `"source":"cli","source":"exec"`, 1))
			return r
		},
	} {
		if _, _, err := ParseRecoveryCandidate(AgentCodex, recoveryCodexEntry(), change(recoveryCodexRecords(`"cli"`)), nil); err == nil {
			t.Fatal("accepted malformed/contradictory authority")
		}
	}
}
func TestRecoveryClaudeTailCWDAndSource(t *testing.T) {
	entry := FileEntry{Artifact: Artifact{RelativePath: "-repo/" + recoveryTestID + ".jsonl"}}
	line := func(cwd, source string) string {
		return `{"type":"assistant","sessionId":"` + recoveryTestID + `","timestamp":"2026-08-28T10:02:00Z","isSidechain":false,"entrypoint":"` + source + `","cwd":"` + cwd + `","message":{"role":"assistant","content":[{"type":"text","text":"` + recoveryTestText + `"}]}}`
	}
	c, _, err := ParseRecoveryCandidate(AgentClaude, entry, recoveryRecords(line("/old", "cli"), line("/repo", "cli")), nil)
	if err != nil {
		t.Fatal(err)
	}
	if c.CWD != "/repo" || len(c.Messages) != 1 || c.Interactive.State != InteractiveYes {
		t.Fatalf("tail=%+v", c)
	}
	for _, source := range []string{"sdk-cli", ""} {
		c, _, err := ParseRecoveryCandidate(AgentClaude, entry, recoveryRecords(line("/repo", source)), nil)
		if err != nil {
			t.Fatal(err)
		}
		if c.Interactive.State == InteractiveYes {
			t.Fatal("unsupported entrypoint admitted")
		}
	}
}
func TestRecoveryGrokUnknownIsNotInteractive(t *testing.T) {
	e := FileEntry{Artifact: Artifact{RelativePath: "%2Frepo/" + recoveryTestID + "/updates.jsonl"}}
	records := recoveryRecords(`{"timestamp":1787911200,"method":"session/update","params":{"sessionId":"` + recoveryTestID + `","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"hello"}}}}`)
	for _, summary := range []string{`{"info":{"cwd":"/repo"}}`, `{"info":{"cwd":"/repo","session_kind":"headless"}}`} {
		c, _, err := ParseRecoveryCandidate(AgentGrok, e, records, []byte(summary))
		if err != nil {
			t.Fatal(err)
		}
		if c.Interactive.State == InteractiveYes {
			t.Fatal("Grok inferred interactive")
		}
	}
}
func TestRecoveryRankingExplicitAndDeduplicated(t *testing.T) {
	c, _, err := ParseRecoveryCandidate(AgentCodex, recoveryCodexEntry(), recoveryCodexRecords(`"cli"`), nil)
	if err != nil {
		t.Fatal(err)
	}
	ranked := RankRecoveryCandidates([]RecoveryCandidate{c, c}, []string{recoveryTestText}, "/repo", nil)
	if len(ranked) != 1 || ranked[0].Score.Percent != 100 || !ranked[0].RequiresExplicitSelection {
		t.Fatalf("ranking=%+v", ranked)
	}
	if got := RankRecoveryCandidates([]RecoveryCandidate{c}, []string{recoveryTestText}, "/other", nil); len(got) != 0 {
		t.Fatal("wrong cwd")
	}
	other := c
	other.NativeID = "019d2222-2222-7222-8222-222222222222"
	other.SemanticLast = c.SemanticLast.Add(time.Minute)
	at := other.SemanticLast
	ranked = RankRecoveryCandidates([]RecoveryCandidate{c, other}, []string{recoveryTestText}, "/repo", &at)
	if len(ranked) != 2 || ranked[0].Candidate.NativeID != other.NativeID {
		t.Fatalf("ranking=%+v", ranked)
	}
}

func TestRecoveryCopiedSubagentAndExecNeverRank(t *testing.T) {
	records := recoveryCodexRecords(`{"subagent":{}}`)
	records[0].Bytes = []byte(strings.Replace(string(records[0].Bytes), `"source":`, `"parent_thread_id":"019d2222-2222-7222-8222-222222222222","source":`, 1))
	child, _, err := ParseRecoveryCandidate(AgentCodex, recoveryCodexEntry(), records, nil)
	if err != nil {
		child = RecoveryCandidate{}
	} // Existing validator may reject child state outright; neither outcome permits ranking.
	exec, _, err := ParseRecoveryCandidate(AgentCodex, recoveryCodexEntry(), recoveryCodexRecords(`"exec"`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := RankRecoveryCandidates([]RecoveryCandidate{child, exec}, []string{recoveryTestText}, "/repo", nil); len(got) != 0 {
		t.Fatalf("copied excluded source ranked: %+v", got)
	}
}
func TestRecoveryBoundsAndSnapshotOwnership(t *testing.T) {
	records := recoveryCodexRecords(`"cli"`)
	oversized := append(records, FramedJSONLRecord{Bytes: []byte(strings.Repeat(" ", RecoveryRecordLimit+1))})
	if _, _, err := ParseRecoveryCandidate(AgentCodex, recoveryCodexEntry(), oversized, nil); err == nil {
		t.Fatal("oversized accepted")
	}
	at := time.Now()
	entry := recoveryCodexEntry()
	entry.ModTime = &at
	c, _, err := ParseRecoveryCandidate(AgentCodex, entry, records, nil)
	if err != nil {
		t.Fatal(err)
	}
	at = time.Time{}
	records[1].Bytes[0] = 'x'
	if c.Entry.ModTime.IsZero() || c.Messages[0].Text != recoveryTestText {
		t.Fatal("input aliases candidate")
	}
	ranked := RankRecoveryCandidates([]RecoveryCandidate{c}, []string{recoveryTestText}, "/repo", nil)
	ranked[0].Candidate.Messages[0].Text = "changed"
	if c.Messages[0].Text != recoveryTestText {
		t.Fatal("rank aliases messages")
	}
}
func TestRecoveryClaudeMissingIdentityAndAgentID(t *testing.T) {
	entry := FileEntry{Artifact: Artifact{RelativePath: "-repo/" + recoveryTestID + ".jsonl"}}
	line := `{"type":"assistant","timestamp":"2026-08-28T10:02:00Z","isSidechain":false,"entrypoint":"cli","cwd":"/repo","message":{"role":"assistant","content":"` + recoveryTestText + `"}}`
	c, _, err := ParseRecoveryCandidate(AgentClaude, entry, recoveryRecords(line), nil)
	if err != nil {
		t.Fatal(err)
	}
	if c.RootEligible && c.Interactive.State == InteractiveYes {
		t.Fatal("missing session identity admitted")
	}
	line = strings.Replace(line, `"type":"assistant"`, `"type":"assistant","sessionId":"`+recoveryTestID+`","agentId":"worker"`, 1)
	c, _, err = ParseRecoveryCandidate(AgentClaude, entry, recoveryRecords(line), nil)
	if err != nil {
		t.Fatal(err)
	}
	if c.Interactive.State == InteractiveYes {
		t.Fatal("agentId admitted")
	}
}
func TestRecoveryTiesStableAcrossInputOrder(t *testing.T) {
	c, _, err := ParseRecoveryCandidate(AgentCodex, recoveryCodexEntry(), recoveryCodexRecords(`"cli"`), nil)
	if err != nil {
		t.Fatal(err)
	}
	d := c
	d.NativeID = "019d2222-2222-7222-8222-222222222222"
	for _, in := range [][]RecoveryCandidate{{c, d}, {d, c}} {
		r := RankRecoveryCandidates(in, []string{recoveryTestText}, "/repo", nil)
		if len(r) != 2 || r[0].Candidate.NativeID != c.NativeID || !r[1].RequiresExplicitSelection {
			t.Fatal("unstable/automatic tied choice")
		}
	}
}

func TestRecoveryCodexCWDShiftAndNonAuthoredText(t *testing.T) {
	records := recoveryCodexRecords(`"cli"`)
	records = append(records, recoveryRecords(`{"type":"turn_context","payload":{"cwd":"/new"}}`, `{"type":"response_item","payload":{"type":"function_call_output","output":"`+recoveryTestText+`"}}`, `{"type":"response_item","payload":{"type":"reasoning","text":"`+recoveryTestText+`"}}`)...)
	c, _, err := ParseRecoveryCandidate(AgentCodex, recoveryCodexEntry(), records, nil)
	if err != nil {
		t.Fatal(err)
	}
	if c.CWD != "/new" || len(c.Messages) != 0 {
		t.Fatalf("old cwd or tool/thinking text survived: %+v", c)
	}
}
func TestRecoveryGrokRejectsSummaryPathContradiction(t *testing.T) {
	e := FileEntry{Artifact: Artifact{RelativePath: "%2Frepo/" + recoveryTestID + "/updates.jsonl"}}
	records := recoveryRecords(`{"timestamp":1787911200,"method":"session/update","params":{"sessionId":"` + recoveryTestID + `","update":{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"hello"}}}}`)
	if _, _, err := ParseRecoveryCandidate(AgentGrok, e, records, []byte(`{"info":{"id":"`+recoveryTestID+`","cwd":"/other"}}`)); err == nil {
		t.Fatal("contradictory summary cwd accepted")
	}
}

func TestRecoveryMessageWindowBoundsTinyMessages(t *testing.T) {
	records := recoveryCodexRecords(`"cli"`)
	small := strings.Replace(string(records[1].Bytes), recoveryTestText, "x", 1)
	for i := 0; i < 1000; i++ {
		records = append(records, FramedJSONLRecord{Bytes: []byte(small)})
	}
	c, _, err := ParseRecoveryCandidate(AgentCodex, recoveryCodexEntry(), records, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Messages) > 256 {
		t.Fatalf("tiny messages bypassed bounded window: %d", len(c.Messages))
	}
}
