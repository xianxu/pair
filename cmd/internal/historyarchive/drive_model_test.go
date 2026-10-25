package historyarchive

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func receiptFixture() UploadReceipt {
	id := strings.Repeat("a", 64)
	return UploadReceipt{Version: 1, Destination: "pair-backup:backups/pair-history", FolderID: "folder-id", BundleID: id, Bundle: UploadObject{Name: id + ".tar.zst.gpg", Size: 17, SHA256: strings.Repeat("b", 64), MD5: strings.Repeat("c", 32), RemoteID: "bundle-id", Verification: "backend-sha256"}, Label: UploadObject{Name: id + ".json", Size: 20, SHA256: strings.Repeat("d", 64), MD5: strings.Repeat("e", 32), RemoteID: "label-id", Verification: "downloaded-sha256"}, VerifiedAt: time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)}
}
func TestDriveDestination(t *testing.T) {
	d, err := ParseDriveDestination("", "")
	if err != nil || d.String() != "pair-backup:backups/pair-history" {
		t.Fatalf("defaults %v %v", d, err)
	}
	if len(d.Namespace()) != 64 {
		t.Fatal("namespace must be SHA256")
	}
	id := strings.Repeat("a", 64)
	b, l, err := d.RemoteObjectNames(id)
	if err != nil || b != id+".tar.zst.gpg" || l != id+".json" {
		t.Fatal(b, l, err)
	}
	for _, r := range []string{"--flag", "a:b", ":drive", "remote\n", "é", "a b"} {
		if _, err := ParseDriveDestination(r, "folder"); err == nil {
			t.Errorf("accepted remote %q", r)
		}
	}
	for _, p := range []string{"/abs", "../up", "a/../b", "a//b", "a/", "a\\b", "a:b", "a\x00b", "a\nb", "a/./b"} {
		if _, err := ParseDriveDestination("remote", p); err == nil {
			t.Errorf("accepted folder %q", p)
		}
	}
	if _, _, err := d.RemoteObjectNames("../x"); err == nil {
		t.Fatal("unsafe bundle ID")
	}
}
func TestUploadReceipt(t *testing.T) {
	good := receiptFixture()
	raw, _ := json.Marshal(good)
	got, err := DecodeUploadReceipt(raw)
	if err != nil || !MatchReceipt(got, good) {
		t.Fatalf("roundtrip: %v", err)
	}
	got.VerifiedAt = got.VerifiedAt.Add(time.Hour)
	if !MatchReceipt(got, good) {
		t.Fatal("timestamp should not affect identity")
	}
	mutations := []func(*UploadReceipt){func(r *UploadReceipt) { r.Version = 0 }, func(r *UploadReceipt) { r.BundleID = "../x" }, func(r *UploadReceipt) { r.Destination = "--bad:folder" }, func(r *UploadReceipt) { r.FolderID = "" }, func(r *UploadReceipt) { r.Bundle.Name = "elsewhere" }, func(r *UploadReceipt) { r.Label.Size = 0 }, func(r *UploadReceipt) { r.Bundle.Size = MaxCiphertextBytes + 1 }, func(r *UploadReceipt) { r.Label.Size = MaxMetadataBytes + 1 }, func(r *UploadReceipt) { r.Bundle.SHA256 = "bad" }, func(r *UploadReceipt) { r.Label.MD5 = "bad" }, func(r *UploadReceipt) { r.Bundle.RemoteID = "" }, func(r *UploadReceipt) { r.Label.Verification = "copy-exited" }, func(r *UploadReceipt) { r.VerifiedAt = time.Time{} }}
	for i, mutate := range mutations {
		bad := good
		mutate(&bad)
		if ValidateUploadReceipt(bad) == nil || MatchReceipt(bad, good) {
			t.Errorf("mutation %d authorized", i)
		}
	}
	for i := 0; i < len(raw); i++ {
		if _, err := DecodeUploadReceipt(raw[:i]); err == nil {
			t.Fatalf("accepted truncation %d", i)
		}
	}
	for _, bad := range []string{`null`, strings.Replace(string(raw), `"version":1`, `"version":1,"version":1`, 1), strings.Replace(string(raw), `"size":17`, `"size":17,"size":17`, 1), strings.Replace(string(raw), `"size":17`, `"size":null`, 1), strings.Replace(string(raw), `"version":1,`, "", 1), string(raw) + ` {}`, strings.Repeat(" ", 8193)} {
		if _, err := DecodeUploadReceipt([]byte(bad)); err == nil {
			t.Errorf("accepted malformed receipt %q", bad)
		}
	}
}
func TestUploadTransitions(t *testing.T) {
	s := UploadState{Destination: "dest", BundleID: "bundle", Phase: UploadInventoryBefore}
	for _, k := range []UploadEventKind{UploadPrepared, UploadBatchObserved, UploadFinalMatch, UploadRecorded} {
		var effect UploadEffect
		var err error
		s, effect, err = NextUpload(s, UploadEvent{Destination: "dest", BundleID: "bundle", Kind: k, BundleMatched: true, LabelMatched: true})
		if err != nil {
			t.Fatal(err)
		}
		if (effect == UploadRecordReceipt) != (k == UploadFinalMatch) {
			t.Fatal("receipt effect without final pair checks")
		}
	}
	if s.Phase != UploadVerified {
		t.Fatalf("phase %v", s.Phase)
	}
	initial := UploadState{Destination: "dest", BundleID: "bundle", Phase: UploadInventoryAfter}
	for _, e := range []UploadEvent{{Destination: "other", BundleID: "bundle", Kind: UploadFinalMatch, BundleMatched: true, LabelMatched: true}, {Destination: "dest", BundleID: "old", Kind: UploadFinalMatch, BundleMatched: true, LabelMatched: true}, {Destination: "dest", BundleID: "bundle", Kind: UploadFinalMatch, BundleMatched: true}} {
		if _, effect, err := NextUpload(initial, e); err == nil || effect == UploadRecordReceipt {
			t.Fatal("stale or incomplete result authorized")
		}
	}
	for _, phase := range []UploadPhase{UploadInventoryBefore, UploadPreparedForBatch, UploadInventoryAfter, UploadReadyToRecord} {
		for _, k := range []UploadEventKind{UploadUnknown, UploadCancel, UploadConflict} {
			s, effect, err := NextUpload(UploadState{Destination: "dest", BundleID: "bundle", Phase: phase}, UploadEvent{Destination: "dest", BundleID: "bundle", Kind: k})
			if err != nil || effect != UploadReport || s.Phase == UploadVerified {
				t.Fatalf("unsafe outcome %v %v %v", s, effect, err)
			}
		}
		if phase != UploadInventoryAfter {
			if _, _, err := NextUpload(UploadState{Destination: "dest", BundleID: "bundle", Phase: phase}, UploadEvent{Destination: "dest", BundleID: "bundle", Kind: UploadFinalMatch, BundleMatched: true, LabelMatched: true}); err == nil {
				t.Fatal("early final result accepted")
			}
		}
	}
}

func TestAdmitBatchPair(t *testing.T) {
	for _, tc := range []struct {
		used, b, l, limit, next int64
		admit, bad              bool
	}{
		{0, 10, 2, 12, 12, true, false}, {1, 10, 2, 12, 1, false, false}, {12, 1, 1, 12, 12, false, false},
		{-1, 1, 1, 12, 0, false, true}, {0, 0, 1, 12, 0, false, true}, {0, 1, 0, 12, 0, false, true}, {13, 1, 1, 12, 0, false, true},
		{0, MaxCiphertextBytes + 1, 1, 1 << 40, 0, false, true}, {0, 1, MaxMetadataBytes + 1, 1 << 40, 0, false, true},
		{0, 1, 1, -1, 0, false, true}, {1<<63 - 2, 1, 1, 1<<63 - 1, 1<<63 - 2, false, false},
		{0, MaxCiphertextBytes, MaxMetadataBytes, 1 << 32, MaxCiphertextBytes + MaxMetadataBytes, true, false},
	} {
		next, admit, err := AdmitBatchPair(tc.used, tc.b, tc.l, tc.limit)
		if (err != nil) != tc.bad || (!tc.bad && (next != tc.next || admit != tc.admit)) {
			t.Fatalf("%+v => %d %v %v", tc, next, admit, err)
		}
	}
}

func FuzzAdmitBatchPair(f *testing.F) {
	f.Add(int64(0), int64(10), int64(2), int64(12))
	f.Add(int64(1<<63-2), int64(1), int64(1), int64(1<<63-1))
	f.Fuzz(func(t *testing.T, used, b, l, limit int64) {
		next, admit, err := AdmitBatchPair(used, b, l, limit)
		valid := used >= 0 && limit >= 0 && used <= limit && b > 0 && b <= MaxCiphertextBytes && l > 0 && l <= MaxMetadataBytes
		if (err == nil) != valid {
			t.Fatalf("input validation %d %d %d %d: %v", used, b, l, limit, err)
		}
		if err != nil {
			return
		}
		fits := b <= limit-used && l <= limit-used-b
		if admit != fits {
			t.Fatal("admission differs from subtraction bound")
		}
		if admit {
			if next < used || next > limit || next-used != b+l {
				t.Fatal("partial pair or overflow")
			}
		} else if next != used {
			t.Fatal("refusal consumed budget")
		}
	})
}

func TestDriveJSONRejectsNestedDuplicateKeys(t *testing.T) {
	for _, raw := range []string{`{"a":[{"k":1,"k":2}]}`, `{"a":1,"a":2}`, `{"a":[1,2]`, `{"a":1} {"b":2}`} {
		var out any
		if err := decodeJSONNoDuplicateKeys([]byte(raw), &out, false); err == nil {
			t.Fatalf("accepted ambiguous JSON %s", raw)
		}
	}
	var out struct {
		A []int `json:"a"`
	}
	if err := decodeJSONNoDuplicateKeys([]byte(`{"a":[1,2],"backend_extra":true}`), &out, false); err != nil {
		t.Fatal(err)
	}
	if err := decodeJSONNoDuplicateKeys([]byte(`{"a":[1,2],"backend_extra":true}`), &out, true); err == nil {
		t.Fatal("strict decoder accepted unknown field")
	}
}

func TestUploadReceiptRejectsCaseFoldDuplicateKeys(t *testing.T) {
	raw, err := json.Marshal(receiptFixture())
	if err != nil {
		t.Fatal(err)
	}
	for name, bad := range map[string]string{
		"version":       strings.Replace(string(raw), `"version":1`, `"Version":2,"version":1`, 1),
		"object digest": strings.Replace(string(raw), `"sha256":`, `"SHA256":"`+strings.Repeat("f", 64)+`","sha256":`, 1),
	} {
		if _, err := DecodeUploadReceipt([]byte(bad)); err == nil {
			t.Errorf("accepted semantic duplicate %s", name)
		}
	}
	var out any
	if err := decodeJSONNoDuplicateKeys([]byte(`{"Hashes":{"SHA-256":"a","sha-256":"b"}}`), &out, false); err == nil {
		t.Fatal("accepted case-ambiguous backend hashes")
	}
}

func TestDriveJSONSharedScanPreservesOwnerPolicies(t *testing.T) {
	for _, raw := range []string{`{"s":1,"ſ":2}`, `{"K":1,"K":2}`, `{"x":[{"escaped":1,"\u0065scaped":2}]}`} {
		var out json.RawMessage
		if err := decodeJSONNoDuplicateKeys([]byte(raw), &out, false); err == nil {
			t.Fatalf("accepted ambiguous backend record %s", raw)
		}
	}
	for _, depth := range []int{64, 65} {
		raw := []byte(strings.Repeat("[", depth) + "0" + strings.Repeat("]", depth))
		var out json.RawMessage
		if err := decodeJSONNoDuplicateKeys(raw, &out, false); (err == nil) != (depth == 64) {
			t.Fatalf("depth %d: %v", depth, err)
		}
	}
	raw := []byte(`{"future": { "huge" : 1e9999, "precise" : 9007199254740993123456789 }}`)
	var out map[string]json.RawMessage
	if err := decodeJSONNoDuplicateKeys(raw, &out, false); err != nil {
		t.Fatal(err)
	}
	if string(out["future"]) != `{ "huge" : 1e9999, "precise" : 9007199254740993123456789 }` {
		t.Fatalf("changed unknown raw value: %s", out["future"])
	}
}
