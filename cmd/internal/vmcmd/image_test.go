package vmcmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func validManifestJSON() string {
	return `{"version":1,"image":"pair-conformance-sonoma-v1","base":"ghcr.io/example/macos@sha256:` + strings.Repeat("a", 64) + `","created_at":"2026-10-20T12:00:00Z","tools":{"go":{"version":"go1.26.3","sha256":"` + strings.Repeat("b", 64) + `"}},"qualified":true}`
}

func TestImageManifestPreservesUnknownNestedFields(t *testing.T) {
	raw := strings.Replace(validManifestJSON(), `"qualified":true`, `"future": { "b": [1, 2] },"qualified":true`, 1)
	raw = strings.Replace(raw, `"version":"go1.26.3"`, `"future_tool": { "x": 7 },"version":"go1.26.3"`, 1)
	m, err := DecodeImageManifest([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	m.Image = "pair-conformance-updated"
	tool := m.Tools["go"]
	tool.Version = "go1.27"
	m.Tools["go"] = tool
	encoded, err := EncodeImageManifest(m)
	if err != nil {
		t.Fatal(err)
	}
	for _, preserved := range []string{`"future":{ "b": [1, 2] }`, `"future_tool":{ "x": 7 }`} {
		if !bytes.Contains(encoded, []byte(preserved)) {
			t.Fatalf("lost %s in %s", preserved, encoded)
		}
	}
	again, err := DecodeImageManifest(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if again.Image != m.Image || again.Tools["go"].Version != "go1.27" {
		t.Fatalf("mutation lost: %#v", again)
	}
}

func TestImageManifestRejectsMalformedAndInvalidFields(t *testing.T) {
	valid := validManifestJSON()
	cases := []string{"null", "[]", "{}", valid + `{}`, strings.Replace(valid, `"version":1`, `"version":1,"version":1`, 1), strings.Replace(valid, `"version":1`, `"version":2`, 1), strings.Replace(valid, `"version":1`, `"version":"1"`, 1)}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(valid), &fields); err != nil {
		t.Fatal(err)
	}
	for key := range fields {
		copyFields := make(map[string]json.RawMessage)
		for k, v := range fields {
			copyFields[k] = v
		}
		delete(copyFields, key)
		raw, _ := json.Marshal(copyFields)
		cases = append(cases, string(raw))
		copyFields[key] = json.RawMessage(`null`)
		raw, _ = json.Marshal(copyFields)
		cases = append(cases, string(raw))
	}
	for _, pair := range [][2]string{
		{`"pair-conformance-sonoma-v1"`, `"other-image"`}, {`"pair-conformance-sonoma-v1"`, `"pair-conformance-"`},
		{`"pair-conformance-sonoma-v1"`, `"pair-conformance-../escape"`},
		{`@sha256:` + strings.Repeat("a", 64), `:latest`},
		{`ghcr.io/example/macos@sha256:`, `repo///@sha256:`},
		{`ghcr.io/example/macos@sha256:`, `https://example.com/image@sha256:`},
		{`2026-10-20T12:00:00Z`, `2026-10-20T12:00:00+01:00`},
		{`2026-10-20T12:00:00Z`, `0001-01-01T00:00:00Z`},
		{`"qualified":true`, `"qualified":false`},
		{`"version":"go1.26.3"`, `"version":""`},
		{`"version":"go1.26.3"`, `"version":false`},
		{strings.Repeat("b", 64), strings.Repeat("B", 64)},
		{`"sha256":"` + strings.Repeat("b", 64) + `"`, `"sha256":null`},
		{`"version":"go1.26.3"`, `"version":"go1.26.3","version":"again"`},
	} {
		cases = append(cases, strings.Replace(valid, pair[0], pair[1], 1))
	}
	cases = append(cases, strings.Replace(valid, `"tools":{`, `"tools":{},"ignored":{`, 1))
	for i, raw := range cases {
		if m, err := DecodeImageManifest([]byte(raw)); err == nil {
			t.Errorf("case %d accepted %s: %#v", i, raw, m)
		}
	}
	m, err := DecodeImageManifest([]byte(valid))
	if err != nil {
		t.Fatal(err)
	}
	m.Tools = map[string]ToolVersion{}
	if _, err := EncodeImageManifest(m); err == nil {
		t.Error("encoded empty tools")
	}
}

func TestImageManifestEncodeValidationAndFutureTools(t *testing.T) {
	m := ImageManifest{Version: 1, Image: "pair-conformance-test", Base: "ghcr.io/example/image@sha256:" + strings.Repeat("a", 64), CreatedAt: time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC), Tools: map[string]ToolVersion{"future-tool": {Version: "1", SHA256: strings.Repeat("b", 64)}}, Qualified: true}
	raw, err := EncodeImageManifest(m)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeImageManifest(raw); err != nil {
		t.Fatal(err)
	}
	m.Qualified = false
	if _, err := EncodeImageManifest(m); err == nil {
		t.Fatal("encoded unqualified manifest")
	}
}

func TestImageManifestRejectsInvalidPreservedValuesAndProtectsKnownMembers(t *testing.T) {
	m, err := DecodeImageManifest([]byte(validManifestJSON()))
	if err != nil {
		t.Fatal(err)
	}
	m.Unknown = map[string]json.RawMessage{"version": json.RawMessage(`999`), "extra": json.RawMessage(`{"a":1,"a":2}`)}
	if _, err := EncodeImageManifest(m); err == nil {
		t.Fatal("encoded duplicate unknown member")
	}
	delete(m.Unknown, "extra")
	tool := m.Tools["go"]
	tool.Unknown = map[string]json.RawMessage{"sha256": json.RawMessage(`"wrong"`), "extra": json.RawMessage(`true false`)}
	m.Tools["go"] = tool
	if _, err := EncodeImageManifest(m); err == nil {
		t.Fatal("encoded trailing unknown tool value")
	}
	delete(tool.Unknown, "extra")
	m.Tools["go"] = tool
	raw, err := EncodeImageManifest(m)
	if err != nil {
		t.Fatal(err)
	}
	again, err := DecodeImageManifest(raw)
	if err != nil {
		t.Fatal(err)
	}
	if again.Version != 1 || again.Tools["go"].SHA256 != strings.Repeat("b", 64) {
		t.Fatalf("unknown members shadowed owned fields: %s", raw)
	}
}

func FuzzImageManifest(f *testing.F) {
	for _, raw := range []string{validManifestJSON(), `{}`, `null`, `{"version":1,"version":2}`, `{"tools":{"go":null}}`, validManifestJSON() + ` true`} {
		f.Add([]byte(raw))
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		m, err := DecodeImageManifest(raw)
		if err != nil {
			return
		}
		encoded, err := EncodeImageManifest(m)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := DecodeImageManifest(encoded); err != nil {
			t.Fatal(err)
		}
	})
}
