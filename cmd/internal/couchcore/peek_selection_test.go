package couchcore

import (
	"reflect"
	"testing"
	"time"

	"github.com/xianxu/pair/cmd/internal/couchidentity"
	"github.com/xianxu/pair/cmd/internal/pairlifecycle"
	"github.com/xianxu/pair/cmd/internal/threadrecord"
)

func purePeekRecord() ThreadRecord {
	return ThreadRecord{
		Address:             ThreadAddress{RepoScope: "scope", Tag: "thread"},
		LatestLaunchProfile: &LaunchProfile{Agent: "agy"},
		Incarnations:        []ThreadIncarnation{{LaunchProfile: &LaunchProfile{Agent: "codex"}}},
		SessionBinding:      &couchidentity.SessionBinding{Name: "pair-current", ScopeKey: "scope", Tag: "thread", StartNonce: "current-start", Legacy: true},
		PreviousSession: &PreviousSession{
			Source: threadrecord.ParkIdentity{Address: threadrecord.Address{RepoScope: "scope", Tag: "thread"}, Nonce: "previous-park", PID: 42, ProcessIdentity: "previous-process"},
			Agent:  "claude", CapturedAt: time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC),
			Capture:       &pairlifecycle.PreservedScrollback{Agent: "claude", Token: "previous-capture", Events: true},
			Binding:       &couchidentity.SessionBinding{Name: "pair-previous", ScopeKey: "scope", Tag: "thread", StartNonce: "previous-start", Legacy: true},
			NativeSession: "legacy-native", Transcripts: []string{"/legacy/transcript"},
		},
	}
}

func TestSelectPeekSourcePureExactSelection(t *testing.T) {
	t.Parallel()
	record := purePeekRecord()
	got, err := SelectPeekSource(record, true)
	if err != nil || got != (PeekSource{Agent: "claude", Token: "previous-capture"}) {
		t.Fatalf("previous selection used current or legacy context: %+v %v", got, err)
	}
	// Selection is immutable even if the retained descriptor is later replaced.
	record.PreviousSession.Capture.Token = "changed"
	if got.Token != "previous-capture" {
		t.Fatal("previous selection aliases descriptor")
	}
	record.PreviousSession.Capture = nil
	got, err = SelectPeekSource(record, true)
	if err != nil || got != (PeekSource{Agent: "claude"}) {
		t.Fatalf("missing capture fell back to current: %+v %v", got, err)
	}
	record.PreviousSession = nil
	if got, err = SelectPeekSource(record, true); err == nil || got != (PeekSource{}) {
		t.Fatalf("missing previous descriptor fell back: %+v %v", got, err)
	}
}

func TestSelectPeekSourcePureCurrentCopiesBinding(t *testing.T) {
	t.Parallel()
	record := purePeekRecord()
	want := *record.SessionBinding
	got, err := SelectPeekSource(record, false)
	if err != nil || got.Agent != "codex" || got.Token != "" || got.Binding == nil || *got.Binding != want {
		t.Fatalf("wrong current selection: %+v %v", got, err)
	}
	got.Binding.Name = "output-edit"
	if record.SessionBinding.Name != "pair-current" {
		t.Fatal("selected binding aliases record")
	}
	got, err = SelectPeekSource(record, false)
	if err != nil {
		t.Fatal(err)
	}
	record.SessionBinding.Name = "input-edit"
	if got.Binding.Name != "pair-current" {
		t.Fatal("record binding aliases selection")
	}
	record.SessionBinding = nil
	record.Incarnations = nil
	got, err = SelectPeekSource(record, false)
	if err != nil || got != (PeekSource{Agent: "agy"}) {
		t.Fatalf("missing current binding synthesized identity: %+v %v", got, err)
	}
}

func TestSelectPeekSourcePureRejectsMalformedEvidence(t *testing.T) {
	t.Parallel()
	cases := map[string]func(*PreviousSession){
		"foreign park":               func(p *PreviousSession) { p.Source.Address.Tag = "other" },
		"missing park process":       func(p *PreviousSession) { p.Source.PID = 0 },
		"missing capture time":       func(p *PreviousSession) { p.CapturedAt = time.Time{} },
		"wrong capture agent":        func(p *PreviousSession) { p.Capture.Agent = "codex" },
		"traversing token":           func(p *PreviousSession) { p.Capture.Token = "../other" },
		"foreign legacy binding":     func(p *PreviousSession) { p.Binding.Tag = "other" },
		"relative legacy transcript": func(p *PreviousSession) { p.Transcripts = []string{"relative"} },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			record := purePeekRecord()
			mutate(record.PreviousSession)
			before := threadrecord.ClonePreviousSession(record.PreviousSession)
			got, err := SelectPeekSource(record, true)
			if err == nil || got != (PeekSource{}) {
				t.Fatalf("malformed previous evidence selected: %+v %v", got, err)
			}
			if !reflect.DeepEqual(record.PreviousSession, before) {
				t.Fatal("selection rewrote malformed evidence")
			}
		})
	}
}
