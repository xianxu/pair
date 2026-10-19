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
	s := UploadState{Destination: "dest", BundleID: "bundle", Phase: UploadCheckingBundle}
	for _, k := range []UploadEventKind{UploadAbsent, UploadCopied, UploadMatch, UploadFinalMatch, UploadRecorded} {
		e := UploadEvent{Destination: "dest", BundleID: "bundle", Kind: k, BundleMatched: true, LabelMatched: true}
		var err error
		s, _, err = NextUpload(s, e)
		if err != nil {
			t.Fatal(err)
		}
	}
	if s.Phase != UploadVerified {
		t.Fatalf("phase %v", s.Phase)
	}
	initial := UploadState{Destination: "dest", BundleID: "bundle", Phase: UploadInventoryAfter}
	for _, e := range []UploadEvent{{Destination: "other", BundleID: "bundle", Kind: UploadFinalMatch, BundleMatched: true, LabelMatched: true}, {Destination: "dest", BundleID: "bundle", Kind: UploadFinalMatch, BundleMatched: true}} {
		if _, effect, err := NextUpload(initial, e); err == nil || effect == UploadRecordReceipt {
			t.Fatal("stale/incomplete match authorized")
		}
	}
	for _, k := range []UploadEventKind{UploadUnknown, UploadCancel, UploadConflict} {
		s, e, err := NextUpload(initial, UploadEvent{Destination: "dest", BundleID: "bundle", Kind: k})
		if err != nil || e != UploadReport || s.Phase == UploadVerified {
			t.Fatalf("unsafe outcome %v %v %v", s, e, err)
		}
	}
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
