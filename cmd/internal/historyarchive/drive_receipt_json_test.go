package historyarchive

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func extendedReceiptJSON(t *testing.T) []byte {
	t.Helper()
	raw, err := json.Marshal(receiptFixture())
	if err != nil {
		t.Fatal(err)
	}
	raw = bytes.Replace(raw, []byte(`"size":17`), []byte(`"size":17,"object_future":{"precise":9007199254740993,"array":[null,true,"x"]}`), 1)
	raw = bytes.Replace(raw, []byte(`"size":20`), []byte(`"size":20,"label_future":[1,2]`), 1)
	return append(append([]byte(nil), raw[:len(raw)-1]...), []byte(`,"future":{"nested":[1,{"flag":true}]}}`)...)
}

func TestUploadReceiptJSONPreservesExtensions(t *testing.T) {
	raw := extendedReceiptJSON(t)
	receipt, err := DecodeUploadReceipt(raw)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{`"future":{"nested":[1,{"flag":true}]}`, `"object_future":{"precise":9007199254740993,"array":[null,true,"x"]}`, `"label_future":[1,2]`} {
		if !bytes.Contains(encoded, []byte(fragment)) {
			t.Errorf("lost raw extension %s: %s", fragment, encoded)
		}
	}
	if !MatchReceipt(receipt, receiptFixture()) {
		t.Fatal("opaque extensions changed known receipt identity")
	}
	receipt.VerifiedAt = receipt.VerifiedAt.Add(time.Hour)
	if !MatchReceipt(receipt, receiptFixture()) {
		t.Fatal("verification time changed receipt identity")
	}
	receipt.Bundle.RemoteID = "different"
	if MatchReceipt(receipt, receiptFixture()) {
		t.Fatal("different known object identity matched")
	}
}

func TestUploadReceiptJSONLegacyBytes(t *testing.T) {
	r := receiptFixture()
	// Explicit wire structs independently pin the legacy field order and encoding.
	type object struct {
		Name         string `json:"name"`
		Size         int64  `json:"size"`
		SHA256       string `json:"sha256"`
		MD5          string `json:"md5"`
		RemoteID     string `json:"remote_id"`
		Verification string `json:"verification"`
	}
	type receipt struct {
		Version     int       `json:"version"`
		Destination string    `json:"destination"`
		FolderID    string    `json:"folder_id"`
		BundleID    string    `json:"bundle_id"`
		Bundle      object    `json:"bundle"`
		Label       object    `json:"label"`
		VerifiedAt  time.Time `json:"verified_at"`
	}
	obj := func(v UploadObject) object {
		return object{v.Name, v.Size, v.SHA256, v.MD5, v.RemoteID, v.Verification}
	}
	want, err := json.Marshal(receipt{r.Version, r.Destination, r.FolderID, r.BundleID, obj(r.Bundle), obj(r.Label), r.VerifiedAt})
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("legacy bytes changed\ngot %s\nwant %s", got, want)
	}
	decoded, err := DecodeUploadReceipt(want)
	if err != nil {
		t.Fatal(err)
	}
	round, err := json.Marshal(decoded)
	if err != nil || !bytes.Equal(round, want) {
		t.Fatalf("legacy round trip changed: %s %v", round, err)
	}
}

func TestUploadReceiptJSONStrictKnownFields(t *testing.T) {
	raw := string(extendedReceiptJSON(t))
	for name, bad := range map[string]string{
		"missing nested":      strings.Replace(raw, `"size":17,`, ``, 1),
		"null nested":         strings.Replace(raw, `"size":17`, `"size":null`, 1),
		"wrong nested type":   strings.Replace(raw, `"size":17`, `"size":"17"`, 1),
		"wrong case nested":   strings.Replace(raw, `"size":17`, `"Size":17`, 1),
		"wrong case outer":    strings.Replace(raw, `"version":1`, `"Version":1`, 1),
		"duplicate nested":    strings.Replace(raw, `"size":17`, `"size":17,"SIZE":17`, 1),
		"duplicate extension": strings.Replace(raw, `"precise":9007199254740993`, `"precise":1,"PRECISE":2`, 1),
		"null outer":          strings.Replace(raw, `"version":1`, `"version":null`, 1),
		"oversize":            strings.Replace(raw, `"label_future":[1,2]`, `"label_future":"`+strings.Repeat("x", MaxUploadReceiptBytes)+`"`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeUploadReceipt([]byte(bad)); err == nil {
				t.Fatal("accepted malformed receipt")
			}
		})
	}
}

func FuzzUploadReceiptJSON(f *testing.F) {
	seed, _ := json.Marshal(receiptFixture())
	f.Add(seed)
	f.Add(bytes.Replace(seed, []byte(`"size":17`), []byte(`"size":17,"future":{"n":9007199254740993}`), 1))
	f.Add([]byte(`{"version":1,"Version":2}`))
	f.Add([]byte(`null`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		receipt, err := DecodeUploadReceipt(raw)
		if err != nil {
			return
		}
		if len(raw) > MaxUploadReceiptBytes {
			t.Fatal("accepted excessive receipt")
		}
		assertReceiptUnknownFields(t, raw, receipt)

		encoded, err := json.Marshal(receipt)
		if err != nil || len(encoded) > MaxUploadReceiptBytes {
			t.Fatalf("cannot reencode admitted receipt: %v", err)
		}
		round, err := DecodeUploadReceipt(encoded)
		if err != nil || !MatchReceipt(receipt, round) {
			t.Fatalf("unstable known fields: %v", err)
		}
		again, err := json.Marshal(round)
		if err != nil || !bytes.Equal(encoded, again) {
			t.Fatalf("unstable encoded receipt: %v", err)
		}
	})
}

func TestUploadObjectJSONCompatibility(t *testing.T) {
	raw := `{"name":"object","size":1,"sha256":"digest","md5":"digest","remote_id":"remote","verification":"backend-sha256","future":{"n":9007199254740993}}`
	var object UploadObject
	if err := json.Unmarshal([]byte(raw), &object); err != nil {
		t.Fatal(err)
	}
	if string(object.Extensions["future"]) != `{"n":9007199254740993}` {
		t.Fatal("standalone object lost extension")
	}
	encoded, err := json.Marshal(object)
	if err != nil || string(encoded) != raw {
		t.Fatalf("standalone object changed: %s %v", encoded, err)
	}
	for _, bad := range []string{
		strings.Replace(raw, `"size":1,`, ``, 1),
		strings.Replace(raw, `"size":1`, `"Size":1`, 1),
		strings.Replace(raw, `"size":1`, `"size":null`, 1),
		strings.Replace(raw, `"size":1`, `"size":"1"`, 1),
		strings.Replace(raw, `"size":1`, `"size":1,"SIZE":1`, 1),
		`null`, `[]`,
	} {
		if err := json.Unmarshal([]byte(bad), &object); err == nil {
			t.Errorf("accepted malformed standalone object %s", bad)
		}
	}
}

func TestUploadReceiptJSONExtensionValidationAndFullBound(t *testing.T) {
	for name, extensions := range map[string]map[string]json.RawMessage{
		"known collision":   {"version": json.RawMessage(`2`)},
		"case collision":    {"Version": json.RawMessage(`2`)},
		"unknown collision": {"future": json.RawMessage(`1`), "FUTURE": json.RawMessage(`2`)},
		"invalid raw":       {"future": json.RawMessage(`{`)},
		"nested duplicate":  {"future": json.RawMessage(`{"n":1,"N":2}`)},
	} {
		t.Run(name, func(t *testing.T) {
			r := receiptFixture()
			r.Extensions = extensions
			if _, err := json.Marshal(r); err == nil {
				t.Fatal("marshaled invalid receipt extension")
			}
			r = receiptFixture()
			// Use the nested object's corresponding reserved field for this check.
			if name == "known collision" || name == "case collision" {
				extensions = map[string]json.RawMessage{"Size": json.RawMessage(`2`)}
			}
			r.Bundle.Extensions = extensions
			if _, err := json.Marshal(r); err == nil {
				t.Fatal("marshaled invalid object extension")
			}
		})
	}
	r := receiptFixture()
	baseline, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	// Each nested object remains under the receipt limit, but their aggregate must fit too.
	n := MaxUploadReceiptBytes - len(baseline) - len(`,"padding":""`)
	r.Bundle.Extensions = map[string]json.RawMessage{"padding": json.RawMessage(`"` + strings.Repeat("x", n) + `"`)}
	exact, err := json.Marshal(r)
	if err != nil || len(exact) != MaxUploadReceiptBytes {
		t.Fatalf("exact receipt bound: %d %v", len(exact), err)
	}
	if _, err := DecodeUploadReceipt(exact); err != nil {
		t.Fatal(err)
	}
	r.Label.Extensions = map[string]json.RawMessage{"extra": json.RawMessage(`true`)}
	if _, err := json.Marshal(r); err == nil {
		t.Fatal("accepted combined extensions over full receipt bound")
	}
	if err := ValidateUploadReceipt(r); err == nil {
		t.Fatal("validated combined extensions over full receipt bound")
	}
}

// Compare each unknown raw value independently of the codec's round-trip result.
func assertReceiptUnknownFields(t *testing.T, raw []byte, receipt UploadReceipt) {
	t.Helper()
	check := func(raw []byte, known []string, got map[string]json.RawMessage) {
		t.Helper()
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			t.Fatal(err)
		}
		for _, key := range known {
			delete(fields, key)
		}
		if len(fields) != len(got) {
			t.Fatal("unknown field count changed")
		}
		for key, want := range fields {
			var expected, actual bytes.Buffer
			if err := json.Compact(&expected, want); err != nil {
				t.Fatal(err)
			}
			if err := json.Compact(&actual, got[key]); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(expected.Bytes(), actual.Bytes()) {
				t.Fatalf("unknown %q changed: %s -> %s", key, expected.Bytes(), actual.Bytes())
			}
		}
	}
	check(raw, []string{"version", "destination", "folder_id", "bundle_id", "bundle", "label", "verified_at"}, receipt.Extensions)
	var outer map[string]json.RawMessage
	if err := json.Unmarshal(raw, &outer); err != nil {
		t.Fatal(err)
	}
	for key, object := range map[string]UploadObject{"bundle": receipt.Bundle, "label": receipt.Label} {
		check(outer[key], []string{"name", "size", "sha256", "md5", "remote_id", "verification"}, object.Extensions)
	}
}

func TestMatchReceiptComparesKnownObservations(t *testing.T) {
	for name, mutate := range map[string]func(*UploadReceipt){
		"destination":         func(r *UploadReceipt) { r.Destination = "pair-backup:backups/another" },
		"folder":              func(r *UploadReceipt) { r.FolderID = "another-folder" },
		"bundle size":         func(r *UploadReceipt) { r.Bundle.Size++ },
		"label size":          func(r *UploadReceipt) { r.Label.Size++ },
		"bundle sha256":       func(r *UploadReceipt) { r.Bundle.SHA256 = strings.Repeat("f", 64) },
		"label sha256":        func(r *UploadReceipt) { r.Label.SHA256 = strings.Repeat("f", 64) },
		"bundle md5":          func(r *UploadReceipt) { r.Bundle.MD5 = strings.Repeat("f", 32) },
		"label md5":           func(r *UploadReceipt) { r.Label.MD5 = strings.Repeat("f", 32) },
		"bundle remote id":    func(r *UploadReceipt) { r.Bundle.RemoteID = "another-bundle" },
		"label remote id":     func(r *UploadReceipt) { r.Label.RemoteID = "another-label" },
		"bundle verification": func(r *UploadReceipt) { r.Bundle.Verification = "backend-md5" },
		"label verification":  func(r *UploadReceipt) { r.Label.Verification = "backend-md5" },
	} {
		t.Run(name, func(t *testing.T) {
			original := receiptFixture()
			changed := receiptFixture()
			mutate(&changed)
			if err := ValidateUploadReceipt(changed); err != nil {
				t.Fatal(err)
			}
			if MatchReceipt(original, changed) || MatchReceipt(changed, original) {
				t.Fatal("different known observations matched")
			}
		})
	}
	r, err := DecodeUploadReceipt(extendedReceiptJSON(t))
	if err != nil {
		t.Fatal(err)
	}
	before, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if !MatchReceipt(r, receiptFixture()) {
		t.Fatal("extensions changed identity")
	}
	after, err := json.Marshal(r)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("matching modified preserved extensions")
	}
}
