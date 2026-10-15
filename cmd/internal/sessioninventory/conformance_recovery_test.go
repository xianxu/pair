package sessioninventory

import (
	"context"
	"io/fs"
	"strings"
	"testing"
)

func TestRecoveryConformanceDetectsRecoveryOnlyDrift(t *testing.T) {
	f, _ := recoveryCollectionFixture()
	good, err := RunRecoveryConformance(context.Background(), f, AgentCodex)
	if err != nil || good.Agents[0].Status != ConformanceOK {
		t.Fatalf("good=%+v err=%v", good, err)
	}
	key := f.entries[0].Artifact.RelativePath
	f.data[key] = []byte(strings.ReplaceAll(string(f.data[key]), `"source":"cli"`, `"source":"new-cli-shape"`))
	f.entries[0].Size = int64(len(f.data[key]))
	// The inventory validator still accepts this root: only recovery provenance drifted.
	records, _, err := FrameJSONLSuffix(JSONLFrameState{}, f.data[key], RecoveryRecordLimit)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := ValidateCodexDelta(f.entries[0], nil, records); err != nil {
		t.Fatalf("inventory contract also failed: %v", err)
	}
	report, err := RunRecoveryConformance(context.Background(), f, AgentCodex)
	if err == nil || report.Agents[0].Status != ConformanceFail {
		t.Fatalf("recovery-only schema drift escaped: %+v %v", report, err)
	}
}

func TestRecoveryConformanceReportsAbsentUnsupportedAndIncomplete(t *testing.T) {
	f, _ := recoveryCollectionFixture()
	f.entries = nil
	report, err := RunRecoveryConformance(context.Background(), f, AgentCodex, AgentGrok)
	if err != nil || len(report.Agents) != 2 {
		t.Fatal(report, err)
	}
	for _, a := range report.Agents {
		if a.Status != ConformanceSkip || a.Reason == "" {
			t.Fatalf("unexplained absent/unsupported: %+v", a)
		}
	}
	f, _ = recoveryCollectionFixture()
	f.change = true
	report, err = RunRecoveryConformance(context.Background(), f, AgentCodex)
	if err == nil || report.Agents[0].Status != ConformanceFail {
		t.Fatal("moving sample accepted", report, err)
	}
}

func TestRecoveryConformanceProductionOSReader(t *testing.T) {
	runtime, _, _ := recoveryOSFixture(t)
	report, err := RunRecoveryConformance(context.Background(), runtime, AgentCodex, AgentClaude, AgentGrok)
	if err != nil {
		t.Fatal(err)
	}
	for _, agent := range report.Agents {
		if agent.Agent == AgentCodex {
			if agent.Status != ConformanceOK || agent.Interactive != 1 || agent.AuthoredTails != 1 {
				t.Fatalf("production parser not covered: %+v", agent)
			}
		} else if agent.Status != ConformanceSkip {
			t.Fatalf("absent/unsupported falsely passed: %+v", agent)
		}
	}
}

func TestRecoveryConformanceClaudeProvenanceCWDActivityAndTail(t *testing.T) {
	for _, mode := range []string{"valid", "missing-entrypoint", "changed-entrypoint", "missing-cwd", "missing-activity", "sdk-cli", "no-authored-tail"} {
		t.Run(mode, func(t *testing.T) {
			f, _ := recoveryCollectionFixture()
			entry := f.entries[0]
			entry.Artifact.StorageRoot = "claude-projects"
			entry.Artifact.RelativePath = "-repo/" + recoveryTestID + ".jsonl"
			raw := `{"type":"assistant","sessionId":"` + recoveryTestID + `","entrypoint":"cli","isSidechain":false,"cwd":"/repo","timestamp":"2026-08-28T10:00:00Z","message":{"role":"assistant","content":[{"type":"text","text":"authored tail"}]}}` + "\n"
			switch mode {
			case "missing-entrypoint":
				raw = strings.ReplaceAll(raw, `"entrypoint":"cli",`, "")
			case "changed-entrypoint":
				raw = strings.ReplaceAll(raw, `"entrypoint":"cli"`, `"entrypoint":"new-cli"`)
			case "sdk-cli":
				raw = strings.ReplaceAll(raw, `"entrypoint":"cli"`, `"entrypoint":"sdk-cli"`)
			case "missing-cwd":
				raw = strings.ReplaceAll(raw, `"cwd":"/repo",`, "")
			case "missing-activity":
				raw = strings.ReplaceAll(raw, `"timestamp":"2026-08-28T10:00:00Z",`, "")
			case "no-authored-tail":
				raw = strings.ReplaceAll(raw, `"text":"authored tail"`, `"text":""`)
			}
			entry.Size = int64(len(raw))
			f.entries = []FileEntry{entry}
			f.data = map[string][]byte{entry.Artifact.RelativePath: []byte(raw)}
			report, err := RunRecoveryConformance(context.Background(), conformanceRecoveryFake{f}, AgentClaude)
			status := report.Agents[0].Status
			switch mode {
			case "valid":
				if err != nil || status != ConformanceOK {
					t.Fatal(report, err)
				}
			case "sdk-cli", "no-authored-tail":
				if err != nil || status != ConformanceSkip {
					t.Fatal(report, err)
				}
			default:
				if err == nil || status != ConformanceFail {
					t.Fatalf("%s drift not detected: %+v %v", mode, report, err)
				}
			}
		})
	}
}

type conformanceRecoveryFake struct{ *recoveryFake }

func (f conformanceRecoveryFake) NativeRoots(agent Agent) []StorageRoot {
	return []StorageRoot{{Agent: agent, Name: "claude-projects"}}
}

func TestRecoveryConformanceRecognizedTailCannotDisappear(t *testing.T) {
	records := recoveryCodexRecords(`"cli"`)
	events, _ := NormalizeNativeEvent(AgentCodex, records[1].Bytes)
	candidate, _, err := ParseRecoveryCandidate(AgentCodex, recoveryCodexEntry(), records, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateRecoveryConformanceTail(candidate, events); err != nil {
		t.Fatal(err)
	}
	candidate.Messages = nil
	if err := validateRecoveryConformanceTail(candidate, events); err == nil {
		t.Fatal("recognized authored tail silently became absent sample")
	}
}

func TestRecoveryConformanceVanishedSampleFails(t *testing.T) {
	f, _ := recoveryCollectionFixture()
	f.readErr = fs.ErrNotExist
	report, err := RunRecoveryConformance(context.Background(), f, AgentCodex)
	if err == nil || report.Agents[0].Status != ConformanceFail {
		t.Fatal("vanished sample became absent storage", report, err)
	}
}
