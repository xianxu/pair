package sessioninventory

import (
	"testing"

	"github.com/xianxu/pair/cmd/internal/sessionledger"
)

func TestOfflineRecoveryUsesOnlyPostLaunchSuffixes(t *testing.T) {
	t.Parallel()
	text := "please preserve this completed native session round"
	old := "## 2026-08-28 01:00:00\n\nold text\n\n---\n\n"
	current := "## 2026-08-28 01:00:01\n\n" + text + "\n\n---\n\n"
	launch := sessionledger.Record{Ordinal: 2, Version: 1, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "claude", PairLogOffset: uint64(len(old)),
		NativeWatermarks: []sessionledger.NativeWatermark{{RootNativeID: "native-a", EventPosition: 10}}}
	result := OfflineRecovery(bindingTestInventory(), OfflineRecoveryInput{
		ScopeKey: "scope", Tag: "work", Agent: AgentClaude, Log: []byte(old + current), Current: sessionledger.Current{Launch: launch},
		NativeEvents: []NativeEventFact{
			{RootNodeID: "root-a", Position: 9, Event: NativeEvent{Kind: EventOperator, Text: text}},
			{RootNodeID: "root-a", Position: 10, Event: NativeEvent{Kind: EventAssistant}},
			{RootNodeID: "root-b", Position: 1, Event: NativeEvent{Kind: EventOperator, Text: text}},
			{RootNodeID: "root-b", Position: 2, Event: NativeEvent{Kind: EventToolCall}},
		},
	})
	binding := onlyBinding(t, result)
	if binding.Status != BindingProvisional || binding.RootNodeID == nil || *binding.RootNodeID != "root-b" {
		t.Fatalf("binding=%#v", binding)
	}
}

func TestOfflineRecoveryPrefersJoinedLedgerBinding(t *testing.T) {
	t.Parallel()
	launch := sessionledger.Record{Ordinal: 2, Version: 1, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "claude"}
	bindingRecord := sessionledger.Record{Ordinal: 3, Version: 1, Kind: sessionledger.RecordBinding, ScopeKey: "scope", Tag: "work", Agent: "claude", LaunchOrdinal: 2, RootNativeID: "native-a"}
	result := OfflineRecovery(bindingTestInventory(), OfflineRecoveryInput{ScopeKey: "scope", Tag: "work", Agent: AgentClaude, Current: sessionledger.Current{Launch: launch, Binding: &bindingRecord}})
	binding := onlyBinding(t, result)
	if binding.Status != BindingEstablished || binding.RootNodeID == nil || *binding.RootNodeID != "root-a" {
		t.Fatalf("binding=%#v", binding)
	}
}

func TestOfflineRecoveryMalformedPairSuffixFailsClosed(t *testing.T) {
	t.Parallel()
	launch := sessionledger.Record{Ordinal: 1, Version: 1, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "claude"}
	result := OfflineRecovery(bindingTestInventory(), OfflineRecoveryInput{ScopeKey: "scope", Tag: "work", Agent: AgentClaude, Log: []byte("truncated"), Current: sessionledger.Current{Launch: launch}})
	binding := onlyBinding(t, result)
	if binding.RootNodeID != nil || binding.Status != BindingProvisional {
		t.Fatalf("binding=%#v", binding)
	}
	found := false
	for _, diagnostic := range result.Diagnostics {
		found = found || diagnostic.Code == DiagnosticPairRecordMalformed
	}
	if !found {
		t.Fatalf("diagnostics=%#v", result.Diagnostics)
	}
}

func TestOfflineRecoveryWithoutLaunchIgnoresHistoricalRounds(t *testing.T) {
	t.Parallel()
	text := "please preserve this completed native session round"
	log := []byte("## 2026-08-28 01:00:01\n\n" + text + "\n\n---\n\n")
	result := OfflineRecovery(bindingTestInventory(), OfflineRecoveryInput{
		ScopeKey: "scope", Tag: "work", Agent: AgentClaude, Log: log,
		NativeEvents: []NativeEventFact{
			{RootNodeID: "root-a", Position: 1, Event: NativeEvent{Kind: EventOperator, Text: text}},
			{RootNodeID: "root-a", Position: 2, Event: NativeEvent{Kind: EventAssistant}},
		},
	})
	binding := onlyBinding(t, result)
	if binding.Status != BindingUnbound || binding.RootNodeID != nil || len(binding.Evidence) != 0 {
		t.Fatalf("binding=%#v", binding)
	}
}

func TestOfflineRecoveryLedgerConflictCannotFallThroughToRounds(t *testing.T) {
	t.Parallel()
	text := "please preserve this completed native session round"
	launch := sessionledger.Record{Ordinal: 1, Version: 1, Kind: sessionledger.RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "claude"}
	bindings := []sessionledger.Record{
		{Ordinal: 2, Version: 1, Kind: sessionledger.RecordBinding, ScopeKey: "scope", Tag: "work", Agent: "claude", LaunchOrdinal: 1, RootNativeID: "native-a"},
		{Ordinal: 3, Version: 1, Kind: sessionledger.RecordBinding, ScopeKey: "scope", Tag: "work", Agent: "claude", LaunchOrdinal: 1, RootNativeID: "native-b"},
	}
	result := OfflineRecovery(bindingTestInventory(), OfflineRecoveryInput{
		ScopeKey: "scope", Tag: "work", Agent: AgentClaude,
		Log:     []byte("## 2026-08-28 01:00:01\n\n" + text + "\n\n---\n\n"),
		Current: sessionledger.Current{Launch: launch, Bindings: bindings, Conflict: true},
		NativeEvents: []NativeEventFact{
			{RootNodeID: "root-a", Position: 1, Event: NativeEvent{Kind: EventOperator, Text: text}},
			{RootNodeID: "root-a", Position: 2, Event: NativeEvent{Kind: EventAssistant}},
		},
	})
	binding := onlyBinding(t, result)
	if binding.Status != BindingAmbiguous || binding.RootNodeID != nil || len(binding.Candidates) != 2 {
		t.Fatalf("binding=%#v", binding)
	}
}

func TestRoundsAfterLaunchUsesTranscriptByteBoundary(t *testing.T) {
	text := "please preserve this completed native session round"
	log := []byte("## 2026-08-28 01:00:01\n\n" + text + "\n\n---\n\n")
	inv := bindingTestInventory()
	artifact := Artifact{StorageRoot: "root", RelativePath: "transcript", Kind: ArtifactTranscript}
	inv.Forests[0].Roots[0].Artifacts = []Artifact{artifact}
	launch := sessionledger.Record{Version: 2, Ordinal: 1, LaunchArtifactBoundaries: []sessionledger.LaunchArtifactBoundary{{StorageRoot: "root", RelativePath: "transcript", RawSize: 100}}}
	for _, tc := range []struct {
		name  string
		start uint64
		want  int
	}{{"historical", 10, 0}, {"partial prelaunch record", 99, 0}, {"post launch", 100, 1}} {
		t.Run(tc.name, func(t *testing.T) {
			events := []NativeEventFact{{RootNodeID: "root-a", Position: ((tc.start + 1) << 8), Event: NativeEvent{Kind: EventOperator, Text: text}}, {RootNodeID: "root-a", Position: ((120 + 1) << 8), Event: NativeEvent{Kind: EventAssistant}}}
			rounds, _ := RoundsAfterLaunch(inv, "scope", "tag", AgentClaude, log, launch, events)
			if len(rounds) != tc.want {
				t.Fatalf("rounds=%#v want %d", rounds, tc.want)
			}
		})
	}
}

func TestRoundsAfterLaunchIncompleteBaselineCannotConfirm(t *testing.T) {
	text := "please preserve this completed native session round"
	log := []byte("## 2026-08-28 01:00:01\n\n" + text + "\n\n---\n\n")
	events := []NativeEventFact{{RootNodeID: "root-a", Position: 256, Event: NativeEvent{Kind: EventOperator, Text: text}}, {RootNodeID: "root-a", Position: 512, Event: NativeEvent{Kind: EventAssistant}}}
	rounds, _ := RoundsAfterLaunch(bindingTestInventory(), "scope", "tag", AgentClaude, log, sessionledger.Record{Version: 3, Ordinal: 1}, events)
	if len(rounds) != 0 {
		t.Fatalf("incomplete baseline confirmed: %#v", rounds)
	}
}
