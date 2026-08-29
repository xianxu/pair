package sessionledger

import (
	"encoding/json"
	"slices"
	"strings"
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

func TestParseLedgerClassifiesCompatibilityRowsSeparately(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"agent":"codex","args":[],"session_id":"legacy","started":"2026-08-28T00:00:00Z","last_active":"2026-08-28T00:00:00Z","repo_root":"/repo","repo_name":"pair"}` + "\n" +
		`{"v":1,"kind":"launch","scope_key":"scope","tag":"work","agent":"codex","pair_log_offset":0,"native_watermarks":[]}` + "\n" +
		"not-json\n")
	parsed := ParseLedger(raw)
	if !slices.Equal(parsed.CompatibilityOrdinals, []uint64{1}) || !slices.Equal(parsed.MalformedOrdinals, []uint64{3}) || len(parsed.Records) != 1 {
		t.Fatalf("parsed=%#v", parsed)
	}
}

func TestParseLedgerRejectsUnterminatedValidRow(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"v":1,"kind":"launch","scope_key":"scope","tag":"work","agent":"claude","pair_log_offset":0,"native_watermarks":[]}`)
	parsed := ParseLedger(raw)
	if len(parsed.Records) != 0 || !slices.Equal(parsed.MalformedOrdinals, []uint64{1}) {
		t.Fatalf("parsed=%#v", parsed)
	}
}

func TestParseLedgerCompatibilityClassificationMatrix(t *testing.T) {
	t.Parallel()
	valid := `{"agent":"claude","args":null,"session_id":"","started":"0001-01-01T00:00:00Z","last_active":"0001-01-01T00:00:00Z","repo_root":"","repo_name":""}`
	for _, test := range []struct {
		name string
		row  string
		want bool
	}{
		{"exact legacy", valid, true},
		{"exact legacy with import marker", strings.TrimSuffix(valid, "}") + `,"legacy_import":true}`, true},
		{"unsupported agent", strings.Replace(valid, `"claude"`, `"other"`, 1), false},
		{"partial", `{"agent":"claude","session_id":"x"}`, false},
		{"wrong field type", strings.Replace(valid, `"args":null`, `"args":3`, 1), false},
		{"unknown field", strings.TrimSuffix(valid, "}") + `,"extra":true}`, false},
		{"trailing value", valid + ` {}`, false},
		{"malformed", `{`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			parsed := ParseLedger([]byte(test.row + "\n"))
			got := slices.Equal(parsed.CompatibilityOrdinals, []uint64{1})
			if got != test.want || (!test.want && !slices.Equal(parsed.MalformedOrdinals, []uint64{1})) {
				t.Fatalf("parsed=%#v want compatibility=%v", parsed, test.want)
			}
		})
	}
}

func TestParseLedgerRejectsUnsupportedTypedAgentsAcrossKinds(t *testing.T) {
	t.Parallel()
	for _, row := range []string{
		`{"v":1,"kind":"launch","scope_key":"scope","tag":"work","agent":"future","pair_log_offset":0,"native_watermarks":[]}`,
		`{"v":1,"kind":"binding","scope_key":"scope","tag":"work","agent":"future","launch_ordinal":1,"root_native_id":"root"}`,
	} {
		parsed := ParseLedger([]byte(row + "\n"))
		if len(parsed.Records) != 0 || !slices.Equal(parsed.MalformedOrdinals, []uint64{1}) {
			t.Fatalf("parsed=%#v", parsed)
		}
	}
}

func TestParseLedgerRejectsDuplicateKeysAcrossFormatsAndNesting(t *testing.T) {
	t.Parallel()
	for _, row := range []string{
		`{"agent":"future","agent":"codex","args":[],"session_id":"legacy","started":"2026-08-28T00:00:00Z","last_active":"2026-08-28T00:00:00Z","repo_root":"/repo","repo_name":"pair"}`,
		`{"v":1,"kind":"launch","scope_key":"scope","tag":"work","agent":"future","agent":"codex","pair_log_offset":0,"native_watermarks":[]}`,
		`{"v":1,"kind":"launch","scope_key":"scope","tag":"work","agent":"codex","pair_log_offset":0,"native_watermarks":[{"root_native_id":"a","root_native_id":"b","event_position":1}]}`,
		`{"v":1,"kind":"launch","scope_key":"scope","tag":"work","agent":"codex","pair_log_offset":0,"native_watermarks":[{"root_native_id":"a"}]}`,
		`{"v":1,"kind":"launch","scope_key":"scope","tag":"work","agent":"codex","pair_log_offset":0,"native_watermarks":[{"root_native_id":"a","event_position":null}]}`,
		`{"v":1,"kind":"launch","scope_key":"scope","tag":"work","agent":"codex","pair_log_offset":0,"native_watermarks":[],"root_native_id":null}`,
		`{"agent":"codex","args":[],"session_id":"legacy","started":"2026-08-28T00:00:00Z","last_active":"2026-08-28T00:00:00Z","repo_root":"/repo","repo_name":"pair","legacy_import":null}`,
	} {
		parsed := ParseLedger([]byte(row + "\n"))
		if len(parsed.Records) != 0 || len(parsed.CompatibilityOrdinals) != 0 || !slices.Equal(parsed.MalformedOrdinals, []uint64{1}) {
			t.Fatalf("duplicate-key row accepted: %#v", parsed)
		}
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

func TestRecordV2LaunchArtifactBoundariesRoundTrip(t *testing.T) {
	t.Parallel()
	record := Record{
		Version: 2, Kind: RecordLaunch, ScopeKey: "scope", Tag: "work", Agent: "claude", PairLogOffset: 9,
		LaunchArtifactBoundaries: []LaunchArtifactBoundary{
			{StorageRoot: "claude-projects", RelativePath: "b.jsonl", StableFileID: "dev:1/ino:2", GenerationToken: "gen:3", MutationToken: "ctime:4", RawSize: 20},
			{StorageRoot: "claude-projects", RelativePath: "a.jsonl", StableFileID: "dev:1/ino:1", MutationToken: "ctime:2", RawSize: 10},
		},
	}
	raw, err := EncodeRecord(record)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"v":2,"kind":"launch","scope_key":"scope","tag":"work","agent":"claude","pair_log_offset":9,"artifact_boundaries":[{"storage_root":"claude-projects","relative_path":"a.jsonl","stable_file_id":"dev:1/ino:1","mutation_token":"ctime:2","raw_size":10},{"storage_root":"claude-projects","relative_path":"b.jsonl","stable_file_id":"dev:1/ino:2","generation_token":"gen:3","mutation_token":"ctime:4","raw_size":20}]}`
	if string(raw) != want {
		t.Fatalf("raw=%s\nwant=%s", raw, want)
	}
	parsed := ParseLedger(append(raw, '\n'))
	if len(parsed.Records) != 1 || len(parsed.Records[0].LaunchArtifactBoundaries) != 2 || parsed.Records[0].LaunchArtifactBoundaries[0].RelativePath != "a.jsonl" {
		t.Fatalf("parsed=%#v", parsed)
	}
}

func TestAuthorizationProofValidatesRootAndCompleteArtifacts(t *testing.T) {
	t.Parallel()
	proof := testAuthorizationProof("root-a")
	if err := ValidateAuthorizationProof(proof, "root-a"); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		mutate func(*AuthorizationProof)
	}{
		{name: "root mismatch", mutate: func(p *AuthorizationProof) { p.RootNativeID = "other" }},
		{name: "unsupported version", mutate: func(p *AuthorizationProof) { p.Version = 2 }},
		{name: "missing schema", mutate: func(p *AuthorizationProof) { p.ScannerSchema = "" }},
		{name: "invalid state", mutate: func(p *AuthorizationProof) { p.ScannerState = json.RawMessage(`null`) }},
		{name: "no artifacts", mutate: func(p *AuthorizationProof) { p.Artifacts = nil }},
		{name: "missing stable id", mutate: func(p *AuthorizationProof) { p.Artifacts[0].StableFileID = "" }},
		{name: "missing mutation", mutate: func(p *AuthorizationProof) { p.Artifacts[0].MutationToken = "" }},
		{name: "offset beyond size", mutate: func(p *AuthorizationProof) { p.Artifacts[0].ParserCompleteOffset = 11 }},
		{name: "duplicate artifact", mutate: func(p *AuthorizationProof) { p.Artifacts = append(p.Artifacts, p.Artifacts[0]) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			candidate := proof
			candidate.ScannerState = append(json.RawMessage(nil), proof.ScannerState...)
			candidate.Artifacts = append([]ArtifactProof(nil), proof.Artifacts...)
			test.mutate(&candidate)
			if err := ValidateAuthorizationProof(candidate, "root-a"); err == nil {
				t.Fatal("invalid proof accepted")
			}
		})
	}
}

func TestRecordV2BindingProofRoundTripAndStrictDecode(t *testing.T) {
	t.Parallel()
	record := Record{Version: 2, Kind: RecordBinding, ScopeKey: "scope", Tag: "work", Agent: "claude", LaunchOrdinal: 1, RootNativeID: "root-a", AuthorizationProof: ptrAuthorizationProof(testAuthorizationProof("root-a"))}
	raw, err := EncodeRecord(record)
	if err != nil {
		t.Fatal(err)
	}
	parsed := ParseLedger(append(raw, '\n'))
	if len(parsed.Records) != 1 || parsed.Records[0].AuthorizationProof == nil || parsed.Records[0].AuthorizationProof.RootNativeID != "root-a" {
		t.Fatalf("parsed=%#v", parsed)
	}
	for _, malformed := range []string{
		strings.Replace(string(raw), `"scanner_schema":"claude-v1"`, `"scanner_schema":"claude-v1","unknown":true`, 1),
		strings.Replace(string(raw), `"parser_complete_offset":10`, `"parser_complete_offset":null`, 1),
		strings.Replace(string(raw), `"root_native_id":"root-a","scanner_schema"`, `"root_native_id":"other","scanner_schema"`, 1),
	} {
		got := ParseLedger([]byte(malformed + "\n"))
		if len(got.Records) != 0 || !slices.Equal(got.MalformedOrdinals, []uint64{1}) {
			t.Fatalf("malformed v2 proof accepted: %#v", got)
		}
	}
}

func testAuthorizationProof(root string) AuthorizationProof {
	return AuthorizationProof{
		Version: 1, RootNativeID: root, ScannerSchema: "claude-v1", ScannerState: json.RawMessage(`{"version":1,"role":"root"}`),
		Artifacts: []ArtifactProof{{StorageRoot: "claude-projects", RelativePath: "a.jsonl", StableFileID: "dev:1/ino:1", GenerationToken: "gen:1", MutationToken: "ctime:1", Size: 10, ParserCompleteOffset: 10}},
	}
}

func ptrAuthorizationProof(proof AuthorizationProof) *AuthorizationProof { return &proof }

func FuzzValidateAuthorizationProof(f *testing.F) {
	f.Add(1, "root", "root", "claude-v1", []byte(`{"version":1}`), "store", "a.jsonl", "stable", "mutation", int64(10), int64(10))
	f.Add(0, "", "root", "", []byte(`null`), "", "", "", "", int64(-1), int64(2))
	f.Fuzz(func(t *testing.T, version int, proofRoot, expectedRoot, schema string, state []byte, storageRoot, relativePath, stableID, mutation string, size, offset int64) {
		if len(state) > 1<<20 {
			state = state[:1<<20]
		}
		proof := AuthorizationProof{
			Version: version, RootNativeID: proofRoot, ScannerSchema: schema, ScannerState: append(json.RawMessage(nil), state...),
			Artifacts: []ArtifactProof{{StorageRoot: storageRoot, RelativePath: relativePath, StableFileID: stableID, MutationToken: mutation, Size: size, ParserCompleteOffset: offset}},
		}
		if err := ValidateAuthorizationProof(proof, expectedRoot); err == nil {
			if version != 1 || expectedRoot == "" || proofRoot != expectedRoot || schema == "" || !json.Valid(state) || string(state) == "null" || storageRoot == "" || relativePath == "" || stableID == "" || mutation == "" || size < 0 || offset < 0 || offset > size {
				t.Fatalf("accepted incomplete proof: %#v", proof)
			}
		}
	})
}
