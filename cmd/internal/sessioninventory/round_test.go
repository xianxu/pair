package sessioninventory

import (
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestQualifyTurnSequence(t *testing.T) {
	t.Parallel()
	strong := "please inspect the durable session inventory boundary"
	shortA := "check logs"
	shortB := "then fix it"
	tests := []struct {
		name   string
		pair   []PairLogFact
		native []NativeEventFact
		want   []RoundObservation
	}{
		{
			name: "one unique substantial completed round",
			pair: []PairLogFact{{Position: 10, Text: strong}},
			native: []NativeEventFact{
				{RootNodeID: "node-a", Position: 7, Event: NativeEvent{Kind: EventOperator, Text: strong}},
				{RootNodeID: "node-a", Position: 8, Event: NativeEvent{Kind: EventToolCall}},
			},
			want: []RoundObservation{{RootNodeID: "node-a", PairPositions: []uint64{10}, NativePositions: []uint64{7}, ProgressPositions: []uint64{8}}},
		},
		{
			name: "composer text without progress remains provisional",
			pair: []PairLogFact{{Position: 10, Text: strong}},
			native: []NativeEventFact{
				{RootNodeID: "node-a", Position: 7, Event: NativeEvent{Kind: EventOperator, Text: strong}},
			},
		},
		{
			name: "unknown event is not progress",
			pair: []PairLogFact{{Position: 10, Text: strong}},
			native: []NativeEventFact{
				{RootNodeID: "node-a", Position: 7, Event: NativeEvent{Kind: EventOperator, Text: strong}},
				{RootNodeID: "node-a", Position: 8, Event: NativeEvent{Kind: NativeEventKind("future")}},
			},
		},
		{
			name: "progress after next operator cannot complete earlier round",
			pair: []PairLogFact{{Position: 10, Text: strong}},
			native: []NativeEventFact{
				{RootNodeID: "node-a", Position: 7, Event: NativeEvent{Kind: EventOperator, Text: strong}},
				{RootNodeID: "node-a", Position: 8, Event: NativeEvent{Kind: EventOperator, Text: "another operator turn that is sufficiently long"}},
				{RootNodeID: "node-a", Position: 9, Event: NativeEvent{Kind: EventAssistant}},
			},
		},
		{
			name: "two consecutive short turns qualify",
			pair: []PairLogFact{{Position: 10, Text: shortA}, {Position: 20, Text: shortB}},
			native: []NativeEventFact{
				{RootNodeID: "node-b", Position: 1, Event: NativeEvent{Kind: EventOperator, Text: shortA}},
				{RootNodeID: "node-b", Position: 2, Event: NativeEvent{Kind: EventAssistant}},
				{RootNodeID: "node-b", Position: 3, Event: NativeEvent{Kind: EventOperator, Text: shortB}},
				{RootNodeID: "node-b", Position: 4, Event: NativeEvent{Kind: EventTerminal}},
			},
			want: []RoundObservation{{RootNodeID: "node-b", PairPositions: []uint64{10, 20}, NativePositions: []uint64{1, 3}, ProgressPositions: []uint64{2, 4}}},
		},
		{
			name: "native operator gap rejects pair",
			pair: []PairLogFact{{Position: 10, Text: shortA}, {Position: 20, Text: shortB}},
			native: []NativeEventFact{
				{RootNodeID: "node-b", Position: 1, Event: NativeEvent{Kind: EventOperator, Text: shortA}},
				{RootNodeID: "node-b", Position: 2, Event: NativeEvent{Kind: EventAssistant}},
				{RootNodeID: "node-b", Position: 3, Event: NativeEvent{Kind: EventOperator, Text: "unmatched gap"}},
				{RootNodeID: "node-b", Position: 4, Event: NativeEvent{Kind: EventAssistant}},
				{RootNodeID: "node-b", Position: 5, Event: NativeEvent{Kind: EventOperator, Text: shortB}},
				{RootNodeID: "node-b", Position: 6, Event: NativeEvent{Kind: EventAssistant}},
			},
		},
		{
			name: "repeated single is globally ambiguous",
			pair: []PairLogFact{{Position: 10, Text: strong}},
			native: []NativeEventFact{
				{RootNodeID: "node-a", Position: 1, Event: NativeEvent{Kind: EventOperator, Text: strong}},
				{RootNodeID: "node-a", Position: 2, Event: NativeEvent{Kind: EventAssistant}},
				{RootNodeID: "node-b", Position: 1, Event: NativeEvent{Kind: EventOperator, Text: strong}},
				{RootNodeID: "node-b", Position: 2, Event: NativeEvent{Kind: EventAssistant}},
			},
		},
		{
			name: "normalization is shared at matcher boundary",
			pair: []PairLogFact{{Position: 10, Text: "=== focus ===\r\n" + strong + "  \r\n"}},
			native: []NativeEventFact{
				{RootNodeID: "node-a", Position: 1, Event: NativeEvent{Kind: EventOperator, Text: strong}},
				{RootNodeID: "node-a", Position: 2, Event: NativeEvent{Kind: EventToolResult}},
			},
			want: []RoundObservation{{RootNodeID: "node-a", PairPositions: []uint64{10}, NativePositions: []uint64{1}, ProgressPositions: []uint64{2}}},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got := QualifyTurnSequence(test.pair, test.native)
			stripObservationFingerprints(got)
			if !slices.EqualFunc(got, test.want, equalRoundObservation) {
				t.Fatalf("QualifyTurnSequence() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestQualifyTurnSequenceExactThresholds(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		text string
		want bool
	}{
		{name: "32 bytes and five words", text: "one two three four five " + strings.Repeat("x", 8), want: true},
		{name: "31 bytes", text: "one two three four five " + strings.Repeat("x", 7)},
		{name: "four words", text: "extraordinary words remain only" + strings.Repeat("x", 8)},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			facts := []PairLogFact{{Position: 1, Text: test.text}}
			native := []NativeEventFact{
				{RootNodeID: "root", Position: 1, Event: NativeEvent{Kind: EventOperator, Text: test.text}},
				{RootNodeID: "root", Position: 2, Event: NativeEvent{Kind: EventAssistant}},
			}
			if got := len(QualifyTurnSequence(facts, native)) != 0; got != test.want {
				t.Fatalf("qualified=%v, want %v for %q (%d bytes)", got, test.want, test.text, len([]byte(test.text)))
			}
		})
	}
}

func TestRoundObservationJSONNeverCopiesText(t *testing.T) {
	t.Parallel()
	raw, err := json.Marshal(RoundObservation{RootNodeID: "root", Texts: []string{"private authored body"}, Fingerprints: []string{"safe"}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "private") || strings.Contains(string(raw), "authored") {
		t.Fatalf("observation leaked text: %s", raw)
	}
}

func FuzzQualifyTurnSequenceDeterministic(f *testing.F) {
	f.Add("check logs", "then fix it", false)
	f.Add("åtta byte", "三 words here", true)
	f.Fuzz(func(t *testing.T, first, second string, gap bool) {
		if len(first) > 256 {
			first = first[:256]
		}
		if len(second) > 256 {
			second = second[:256]
		}
		pair := []PairLogFact{{Position: 10, Text: first}, {Position: 20, Text: second}}
		native := []NativeEventFact{
			{RootNodeID: "root", Position: 1, Event: NativeEvent{Kind: EventOperator, Text: first}},
			{RootNodeID: "root", Position: 2, Event: NativeEvent{Kind: EventAssistant}},
		}
		if gap {
			native = append(native,
				NativeEventFact{RootNodeID: "root", Position: 3, Event: NativeEvent{Kind: EventOperator, Text: "gap"}},
				NativeEventFact{RootNodeID: "root", Position: 4, Event: NativeEvent{Kind: EventToolResult}})
		}
		native = append(native,
			NativeEventFact{RootNodeID: "root", Position: 5, Event: NativeEvent{Kind: EventOperator, Text: second}},
			NativeEventFact{RootNodeID: "root", Position: 6, Event: NativeEvent{Kind: EventTerminal}})
		want := QualifyTurnSequence(pair, native)
		slices.Reverse(pair)
		slices.Reverse(native)
		if got := QualifyTurnSequence(pair, native); !reflect.DeepEqual(got, want) {
			t.Fatalf("permutation changed result: got=%#v want=%#v", got, want)
		}
	})
}

func stripObservationFingerprints(observations []RoundObservation) {
	for i := range observations {
		observations[i].Fingerprints = nil
	}
}

func equalRoundObservation(a, b RoundObservation) bool {
	return a.RootNodeID == b.RootNodeID && slices.Equal(a.PairPositions, b.PairPositions) &&
		slices.Equal(a.NativePositions, b.NativePositions) && slices.Equal(a.ProgressPositions, b.ProgressPositions)
}
