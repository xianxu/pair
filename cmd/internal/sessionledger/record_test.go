package sessionledger

import (
	"slices"
	"testing"
)

func TestParseLedgerRetainsPhysicalOrdinals(t *testing.T) {
	t.Parallel()
	raw := []byte("not-json\n" +
		`{"v":1,"kind":"launch","scope_key":"scope","tag":"work","agent":"claude","pair_log_offset":12,"native_watermarks":[{"root_native_id":"b","event_position":9},{"root_native_id":"a","event_position":4}]}` + "\n" +
		`{"v":1,"kind":"binding","scope_key":"scope","tag":"work","agent":"claude","launch_ordinal":2,"root_native_id":"native-a"}` + "\n")
	parsed := ParseLedger(raw)
	if !slices.Equal(parsed.MalformedOrdinals, []uint64{1}) || len(parsed.Records) != 2 {
		t.Fatalf("parsed=%#v", parsed)
	}
	if parsed.Records[0].Ordinal != 2 || parsed.Records[1].Ordinal != 3 {
		t.Fatalf("ordinals=%d,%d", parsed.Records[0].Ordinal, parsed.Records[1].Ordinal)
	}
	if got := parsed.Records[0].NativeWatermarks; len(got) != 2 || got[0].RootNativeID != "a" || got[1].RootNativeID != "b" {
		t.Fatalf("watermarks=%#v", got)
	}
}

func TestCurrentLaunchUsesLatestPhysicalGeneration(t *testing.T) {
	t.Parallel()
	records := []Record{
		{Ordinal: 1, Version: 1, Kind: RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "claude", PairLogOffset: 10},
		{Ordinal: 2, Version: 1, Kind: RecordBinding, ScopeKey: "scope", Tag: "work", Agent: "claude", LaunchOrdinal: 1, RootNativeID: "old-root"},
		{Ordinal: 3, Version: 1, Kind: RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "claude", PairLogOffset: 20},
		{Ordinal: 4, Version: 1, Kind: RecordBinding, ScopeKey: "other", Tag: "work", Agent: "claude", LaunchOrdinal: 3, RootNativeID: "wrong-owner"},
	}
	current, ok := CurrentLaunch(records, Owner{ScopeKey: "scope", Tag: "work", Agent: "claude"})
	if !ok || current.Launch.Ordinal != 3 || current.Binding != nil {
		t.Fatalf("current=%#v ok=%v", current, ok)
	}
	records = append(records, Record{Ordinal: 5, Version: 1, Kind: RecordBinding, ScopeKey: "scope", Tag: "work", Agent: "claude", LaunchOrdinal: 3, RootNativeID: "new-root"})
	current, ok = CurrentLaunch(records, Owner{ScopeKey: "scope", Tag: "work", Agent: "claude"})
	if !ok || current.Binding == nil || current.Binding.RootNativeID != "new-root" {
		t.Fatalf("current=%#v ok=%v", current, ok)
	}
}

func TestCurrentLaunchRejectsConflictingBindingsForGeneration(t *testing.T) {
	t.Parallel()
	records := []Record{
		{Ordinal: 1, Version: 1, Kind: RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "claude"},
		{Ordinal: 2, Version: 1, Kind: RecordBinding, ScopeKey: "scope", Tag: "work", Agent: "claude", LaunchOrdinal: 1, RootNativeID: "a"},
		{Ordinal: 3, Version: 1, Kind: RecordBinding, ScopeKey: "scope", Tag: "work", Agent: "claude", LaunchOrdinal: 1, RootNativeID: "b"},
	}
	current, ok := CurrentLaunch(records, Owner{ScopeKey: "scope", Tag: "work", Agent: "claude"})
	if !ok || !current.Conflict || current.Binding != nil || len(current.Bindings) != 2 {
		t.Fatalf("current=%#v ok=%v", current, ok)
	}
}

func FuzzParseLedgerPhysicalOrdinals(f *testing.F) {
	f.Add([]byte("not-json\n{}\n"))
	f.Add([]byte(`{"v":1,"kind":"binding"}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) > 2<<20 {
			raw = raw[:2<<20]
		}
		parsed := ParseLedger(raw)
		seen := map[uint64]bool{}
		for _, record := range parsed.Records {
			if record.Ordinal == 0 || seen[record.Ordinal] {
				t.Fatalf("invalid record ordinal %d", record.Ordinal)
			}
			seen[record.Ordinal] = true
		}
		for _, ordinal := range parsed.MalformedOrdinals {
			if ordinal == 0 || seen[ordinal] {
				t.Fatalf("invalid malformed ordinal %d", ordinal)
			}
			seen[ordinal] = true
		}
	})
}

func TestEncodeRecordCanonicalizesAndValidates(t *testing.T) {
	t.Parallel()
	record := Record{Version: 1, Kind: RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "claude", PairLogOffset: 9,
		NativeWatermarks: []NativeWatermark{{RootNativeID: "b", EventPosition: 2}, {RootNativeID: "a", EventPosition: 3}}}
	raw, err := EncodeRecord(record)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"v":1,"kind":"launch","scope_key":"scope","tag":"work","agent":"claude","pair_log_offset":9,"native_watermarks":[{"root_native_id":"a","event_position":3},{"root_native_id":"b","event_position":2}]}`
	if string(raw) != want {
		t.Fatalf("raw=%s want=%s", raw, want)
	}
	if _, err := EncodeRecord(Record{Version: 1, Kind: RecordBinding, ScopeKey: "scope", Tag: "work", Agent: "claude"}); err == nil {
		t.Fatal("invalid binding encoded")
	}
}

func TestEncodeLaunchIncludesEmptyWatermarkArray(t *testing.T) {
	t.Parallel()
	raw, err := EncodeRecord(Record{Version: 1, Kind: RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "claude"})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"v":1,"kind":"launch","scope_key":"scope","tag":"work","agent":"claude","pair_log_offset":0,"native_watermarks":[]}` {
		t.Fatalf("raw=%s", raw)
	}
}
