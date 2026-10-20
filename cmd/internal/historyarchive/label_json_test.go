package historyarchive

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func labelFixture(t *testing.T) BundleLabel { t.Helper(); return catalogFixture(t, "a").Label }

func TestLabelLegacyGolden(t *testing.T) {
	l := labelFixture(t)
	if l.ID != "1b38d6414986ba32e12a95c2ad0b4d5a1be65a332eeb64eeffeaa87069d8d543" {
		t.Fatalf("legacy manifest identity changed: %s", l.ID)
	}
	raw, err := json.Marshal(l)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"version":1,"id":"` + l.ID + `","kind":"ops-documents","from":"2026-10-01T00:00:00Z","to":"2026-10-01T00:00:00Z","file_count":1,"raw_bytes":1,"bytes":10,"compression":"zstd -19","recipients":["` + strings.Repeat("A", 40) + `"],"sha256":"` + strings.Repeat("c", 64) + `"}`
	if string(raw) != want {
		t.Fatalf("legacy bytes changed: %s", raw)
	}
	if _, err := DecodeLabel(raw); err != nil {
		t.Fatal(err)
	}
}
func TestLabelExtensionsRoundTrip(t *testing.T) {
	raw, _ := json.Marshal(labelFixture(t))
	raw = append(raw[:len(raw)-1], []byte(`,"future":{"nested":[null,123456789012345678901234567890,{"flag":true}],"value":"x"}}`)...)
	l, err := DecodeLabel(raw)
	if err != nil {
		t.Fatal(err)
	}
	original := append([]byte(nil), l.Extensions["future"]...)
	raw[0] = 'x'
	if !bytes.Equal(original, l.Extensions["future"]) {
		t.Fatal("extension aliases input")
	}
	encoded, err := json.Marshal(l)
	if err != nil {
		t.Fatal(err)
	}
	again, err := DecodeLabel(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, again.Extensions["future"]) {
		t.Fatalf("lost raw extension: %s", encoded)
	}
}
func TestLabelRejectsMalformed(t *testing.T) {
	raw, _ := json.Marshal(labelFixture(t))
	var fields map[string]json.RawMessage
	json.Unmarshal(raw, &fields)
	for key := range fields {
		t.Run("missing_"+key, func(t *testing.T) {
			var copy map[string]json.RawMessage
			json.Unmarshal(raw, &copy)
			delete(copy, key)
			b, _ := json.Marshal(copy)
			if _, err := DecodeLabel(b); err == nil {
				t.Fatal("missing field accepted")
			}
		})
		for _, value := range []string{"null", `{}`, `[]`, `true`} {
			t.Run(key+"_"+value, func(t *testing.T) {
				var copy map[string]json.RawMessage
				json.Unmarshal(raw, &copy)
				copy[key] = json.RawMessage(value)
				b, _ := json.Marshal(copy)
				if _, err := DecodeLabel(b); err == nil {
					t.Fatal("wrong type accepted")
				}
			})
		}
	}
	for _, bad := range []string{`null`, `[]`, string(raw) + ` {}`, strings.Replace(string(raw), `"version":1`, `"version":1,"version":1`, 1), strings.Replace(string(raw), `"version":1`, `"VERSION":1`, 1), strings.Replace(string(raw), `"version":1`, `"version":1,"VERSION":1`, 1), strings.TrimSuffix(string(raw), "}") + `,"future":{"A":1,"a":2}}`, strings.TrimSuffix(string(raw), "}") + `,"future":` + strings.Repeat("[", 70) + `0` + strings.Repeat("]", 70) + `}`, string(raw) + strings.Repeat(" ", MaxMetadataBytes)} {
		if _, err := DecodeLabel([]byte(bad)); err == nil {
			t.Fatalf("accepted malformed %.100s", bad)
		}
	}
}
func TestLabelExtensionCollisions(t *testing.T) {
	for _, key := range []string{"id", "ID", "verſion"} {
		l := labelFixture(t)
		l.Extensions = map[string]json.RawMessage{key: json.RawMessage(`1`)}
		if _, err := json.Marshal(l); err == nil {
			t.Fatalf("accepted collision %s", key)
		}
	}
	for _, raw := range []string{`{"a":1,"A":2}`, `1,"injected":2`, `{`, strings.Repeat("[", 70) + `0` + strings.Repeat("]", 70), `"` + strings.Repeat("x", MaxMetadataBytes) + `"`} {
		l := labelFixture(t)
		l.Extensions = map[string]json.RawMessage{"future": json.RawMessage(raw)}
		if _, err := json.Marshal(l); err == nil {
			t.Fatal("accepted invalid extension")
		}
	}
}
func TestBundleKeysCompatibility(t *testing.T) {
	m := pureManifest()
	id, _ := m.ID()
	b, l, err := BundleKeys(m.Kind, m.From, m.To, id)
	if err != nil {
		t.Fatal(err)
	}
	oldB, oldL, _ := m.Keys()
	if b != oldB || l != oldL {
		t.Fatal("keys changed")
	}
	want := "ops-documents/2026/01/ops-documents-20260102T000000.000000000Z-20260102T000000.000000000Z-" + id
	if b != want+".tar.zst.gpg" || l != want+".json" {
		t.Fatalf("keys: %s %s", b, l)
	}
	for _, bad := range []struct {
		kind     string
		from, to time.Time
		id       string
	}{{"../x", m.From, m.To, id}, {m.Kind, time.Time{}, m.To, id}, {m.Kind, m.From, m.From.Add(-time.Second), id}, {m.Kind, m.From, m.To, "bad"}} {
		if _, _, err := BundleKeys(bad.kind, bad.from, bad.to, bad.id); err == nil {
			t.Fatal("accepted invalid key inputs")
		}
	}
	m.Files = nil
	if _, _, err := m.Keys(); err == nil {
		t.Fatal("manifest validation bypassed")
	}
}
func FuzzLabel(f *testing.F) {
	f.Add([]byte(`null`))
	f.Add([]byte(`{"version":1,"id":"` + strings.Repeat("a", 64) + `","kind":"ops-documents","from":"2026-10-01T00:00:00Z","to":"2026-10-01T00:00:00Z","file_count":1,"raw_bytes":1,"bytes":10,"compression":"zstd -19","recipients":["` + strings.Repeat("A", 40) + `"],"sha256":"` + strings.Repeat("b", 64) + `","future":{"nested":[null,123456789012345678901234567890]}}`))
	f.Add([]byte(`{"version":1}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		l, err := DecodeLabel(raw)
		if err != nil {
			return
		}
		b, err := json.Marshal(l)
		if err != nil {
			t.Fatal(err)
		}
		if len(b) > MaxMetadataBytes {
			t.Fatal("unbounded output")
		}
		if _, err := DecodeLabel(b); err != nil {
			t.Fatal(err)
		}
	})
}
