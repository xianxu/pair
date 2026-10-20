package strictjson

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestKnownFieldsFollowsOwnedTags(t *testing.T) {
	type wire struct {
		Name    string `json:"name"`
		Count   int    `json:"count,omitempty"`
		Bare    string
		Skipped string `json:"-"`
		private string
	}
	_ = wire{}.private
	got := KnownFields(reflect.TypeOf(wire{}))
	want := map[string]bool{"name": true, "count": true, "Bare": true}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("KnownFields = %v, want %v", got, want)
	}
}

// The unknown members of a record must come back byte for byte after a
// rewrite: whitespace, escapes and integers beyond float64 precision included.
func TestSplitEncodeObjectPreservesUnknownBytes(t *testing.T) {
	unknownValues := map[string]string{
		"spaced":  `{ "a" :  [1, 2 ,3] }`,
		"escaped": `"café \"q\" \\ \/"`,
		"big":     `123456789012345678901234567890`,
		"nested":  `{"x":{"y":[{"z":null}]}}`,
		"null":    `null`,
	}
	var b strings.Builder
	b.WriteString(`{"name":"before"`)
	for k, v := range unknownValues {
		b.WriteString(`,"` + k + `":` + v)
	}
	b.WriteString(`}`)
	known := map[string]bool{"name": true}
	owned, unknown, err := SplitObject([]byte(b.String()), known, nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(owned["name"]) != `"before"` {
		t.Fatalf("owned name = %s", owned["name"])
	}
	owned["name"] = json.RawMessage(`"after"`)
	out, err := EncodeObject(owned, unknown, known)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(out) {
		t.Fatalf("encoded object is not valid JSON: %s", out)
	}
	_, again, err := SplitObject(out, known, nil)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range unknownValues {
		if string(again[k]) != v {
			t.Errorf("unknown %q = %s, want %s", k, again[k], v)
		}
	}
	if !bytes.Contains(out, []byte(`"name":"after"`)) {
		t.Fatalf("owned rewrite lost: %s", out)
	}
}

func TestSplitObjectStaysStrict(t *testing.T) {
	known := map[string]bool{"name": true, "when": true}
	nullable := map[string]bool{"when": true}
	for name, raw := range map[string]string{
		"duplicate owned":   `{"name":"a","name":"b"}`,
		"duplicate unknown": `{"x":1,"x":2}`,
		"duplicate nested":  `{"x":{"y":1,"y":2}}`,
		"null owned":        `{"name":null}`,
		"null object":       `null`,
		"array":             `[1]`,
		"trailing value":    `{"name":"a"} {}`,
		"malformed":         `{"name":`,
		"escaped duplicate": `{"name":"a","name":"b"}`,
	} {
		if _, _, err := SplitObject([]byte(raw), known, nullable); err == nil {
			t.Errorf("%s: SplitObject accepted %s", name, raw)
		}
	}
	if _, _, err := SplitObject([]byte(`{"when":null}`), known, nullable); err != nil {
		t.Fatalf("nullable owned member refused: %v", err)
	}
}

// An unknown member can never shadow an owned one on encode, and a corrupt
// preserved value refuses rather than writing an invalid record.
func TestEncodeObjectOwnedWinsAndRefusesCorruptUnknown(t *testing.T) {
	known := map[string]bool{"name": true}
	out, err := EncodeObject(map[string]json.RawMessage{"name": json.RawMessage(`"owned"`)},
		map[string]json.RawMessage{"name": json.RawMessage(`"shadow"`), "extra": json.RawMessage(`1`)}, known)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != `{"extra":1,"name":"owned"}` {
		t.Fatalf("encoded = %s", out)
	}
	if _, err := EncodeObject(nil, map[string]json.RawMessage{"bad": json.RawMessage(`{"a":1,"a":2}`)}, known); err == nil {
		t.Fatal("EncodeObject accepted a duplicate-key unknown value")
	}
	if _, err := EncodeObject(nil, map[string]json.RawMessage{"bad": json.RawMessage(`{`)}, known); err == nil {
		t.Fatal("EncodeObject accepted an invalid unknown value")
	}
}

func TestCloneAndMergeRawCopyBytes(t *testing.T) {
	if CloneRaw(nil) != nil {
		t.Fatal("CloneRaw(nil) must stay nil")
	}
	src := map[string]json.RawMessage{"a": json.RawMessage(`1`)}
	c := CloneRaw(src)
	c["a"][0] = '2'
	if string(src["a"]) != `1` {
		t.Fatal("CloneRaw shares bytes with its source")
	}
	m := MergeRaw(map[string]json.RawMessage{"a": json.RawMessage(`1`), "b": json.RawMessage(`2`)},
		map[string]json.RawMessage{"b": json.RawMessage(`3`)})
	if string(m["a"]) != `1` || string(m["b"]) != `3` {
		t.Fatalf("MergeRaw = %v", m)
	}
	if MergeRaw(nil, nil) != nil {
		t.Fatal("MergeRaw(nil, nil) must stay nil")
	}
}

func FuzzSplitObject(f *testing.F) {
	for _, seed := range []string{
		`{"name":"a","x":1}`,
		`{"name":"a","name":"b"}`,
		`{"x":{"y":1,"y":2}}`,
		`{"name":1}`,
		`{"name":null,"when":null}`,
		`{"x":[{"a":1},{"a":1,"a":2}]}`,
		`{"x":123456789012345678901234567890}`,
	} {
		f.Add([]byte(seed))
	}
	known := map[string]bool{"name": true, "when": true}
	nullable := map[string]bool{"when": true}
	f.Fuzz(func(t *testing.T, raw []byte) {
		owned, unknown, err := SplitObject(raw, known, nullable)
		if err != nil {
			return
		}
		if RejectDuplicateKeys(raw) != nil {
			t.Fatalf("accepted a document with duplicate keys: %s", raw)
		}
		out, err := EncodeObject(owned, unknown, known)
		if err != nil {
			t.Fatalf("re-encode of an accepted object failed: %v", err)
		}
		owned2, unknown2, err := SplitObject(out, known, nullable)
		if err != nil {
			t.Fatalf("re-split of %s failed: %v", out, err)
		}
		if !reflect.DeepEqual(rawStrings(owned), rawStrings(owned2)) || !reflect.DeepEqual(rawStrings(unknown), rawStrings(unknown2)) {
			t.Fatalf("round trip changed members: %s -> %s", raw, out)
		}
	})
}

func rawStrings(m map[string]json.RawMessage) map[string]string {
	out := map[string]string{}
	for k, v := range m {
		out[k] = string(v)
	}
	return out
}

func TestAppendMembersKeepsFormattingAndBytes(t *testing.T) {
	known := map[string]bool{"name": true}
	indented := []byte("{\n  \"name\": \"a\"\n}")
	out, err := AppendMembers(indented, map[string]json.RawMessage{
		"z": json.RawMessage(`{ "k" : 1 }`), "name": json.RawMessage(`"shadow"`),
	}, known)
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"name\": \"a\",\n  \"z\": { \"k\" : 1 }\n}"
	if string(out) != want {
		t.Fatalf("AppendMembers = %q, want %q", out, want)
	}
	if out, err := AppendMembers([]byte("{}"), map[string]json.RawMessage{"x": json.RawMessage(`1`)}, known); err != nil || string(out) != "{\n  \"x\": 1\n}" {
		t.Fatalf("empty object: %q, %v", out, err)
	}
	if same, err := AppendMembers(indented, nil, known); err != nil || string(same) != string(indented) {
		t.Fatalf("no members must return the input unchanged: %q, %v", same, err)
	}
	if _, err := AppendMembers([]byte("[1]"), map[string]json.RawMessage{"x": json.RawMessage(`1`)}, known); err == nil {
		t.Fatal("AppendMembers accepted a non-object")
	}
	if _, err := AppendMembers(indented, map[string]json.RawMessage{"x": json.RawMessage(`{"a":1,"a":2}`)}, known); err == nil {
		t.Fatal("AppendMembers accepted a duplicate-key value")
	}
}
