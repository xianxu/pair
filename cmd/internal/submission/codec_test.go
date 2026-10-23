package submission

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func extensionManifest(t *testing.T) []byte {
	t.Helper()
	b, err := json.Marshal(sketchSubmission(t))
	if err != nil {
		t.Fatal(err)
	}
	var tree map[string]any
	if err = json.Unmarshal(b, &tree); err != nil {
		t.Fatal(err)
	}
	tree["future"] = json.RawMessage(`{"integer":90071992547409931234567890,"ref":"../unknown-image"}`)
	sketch := tree["sketches"].([]any)[0].(map[string]any)
	sketch["future_sketch"] = map[string]any{"a": true}
	sketch["source"].(map[string]any)["future_source"] = "original"
	sketch["author"].(map[string]any)["future_author"] = "author"
	sketch["image"].(map[string]any)["future_image"] = "image"
	plan := sketch["image"].(map[string]any)["plan"].(map[string]any)
	plan["future_transform"] = "transform"
	path := plan["paths"].([]any)[0].(map[string]any)
	path["future_path"] = "path"
	path["at"].(map[string]any)["future_anchor"] = "anchor"
	b, err = json.Marshal(tree)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func TestPreservingCodecKeepsNestedExtensionsAndExactNumbers(t *testing.T) {
	data := extensionManifest(t)
	s, err := DecodePreserving(data)
	if err != nil {
		t.Fatal(err)
	}
	s.Delivery = Delivering
	s.Superseded = true
	b, err := EncodePreserving(s)
	if err != nil {
		t.Fatal(err)
	}
	for _, literal := range []string{"90071992547409931234567890", "future_sketch", "future_source", "future_author", "future_image", "future_transform", "future_path", "future_anchor"} {
		if !bytes.Contains(b, []byte(literal)) {
			t.Fatalf("lost %s: %s", literal, b)
		}
	}
	s, err = DecodePreserving(b)
	if err != nil {
		t.Fatal(err)
	}
	s.Superseded = false
	b, err = EncodePreserving(s)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(b, []byte(`"superseded"`)) {
		t.Fatalf("known omitted field survived: %s", b)
	}
}
func TestPreservingCodecRejectsAmbiguityAndKnownTypes(t *testing.T) {
	valid := string(extensionManifest(t))
	for _, bad := range []string{
		strings.Replace(valid, `"version":3`, `"version":3,"version":3`, 1),
		strings.Replace(valid, `"future_path":"path"`, `"future_path":{"x":1,"x":2}`, 1),
		strings.Replace(valid, `"next_sequence":5`, `"next_sequence":"5"`, 1),
		strings.Replace(valid, `"next_sequence":5`, `"Next_sequence":5`, 1),
		strings.Replace(valid, `"width":54`, `"width":"54"`, 1),
		valid + ` {}`,
	} {
		if _, err := DecodePreserving([]byte(bad)); err == nil {
			t.Fatalf("accepted invalid manifest: %s", bad)
		}
	}
}
func TestPreservingCodecDoesNotMoveUnknownArrayOwnership(t *testing.T) {
	s, err := DecodePreserving(extensionManifest(t))
	if err != nil {
		t.Fatal(err)
	}
	s.Sketches[0].ID = "different"
	if _, err = EncodePreserving(s); err == nil {
		t.Fatal("extensions silently reassigned to another sketch")
	}
}

func TestPreservingCodecUnknownNumberHasNoFloatDomain(t *testing.T) {
	data := bytes.Replace(extensionManifest(t), []byte("90071992547409931234567890"), []byte("1e10000"), 1)
	s, err := DecodePreserving(data)
	if err != nil {
		t.Fatal("valid unknown JSON number rejected", err)
	}
	encoded, err := EncodePreserving(s)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(encoded, []byte("1e10000")) {
		t.Fatalf("unknown number changed: %s", encoded)
	}
	data = bytes.Replace(data, []byte(`"next_sequence":5`), []byte(`"next_sequence":1e10000`), 1)
	if _, err = DecodePreserving(data); err == nil {
		t.Fatal("out-of-domain known uint64 accepted")
	}
}

func TestPreservingCodecRequiresExactPressureAndGeometryTuples(t *testing.T) {
	valid := extensionManifest(t)
	for _, pair := range [][2]string{{`[20,20,0.2]`, `[20,20]`}, {`[20,20,0.2]`, `[20,20,0.2,999]`}, {`"scroll":[0,0]`, `"scroll":[0]`}, {`"box":[0,0,100,100]`, `"box":[0,0,100,100,999]`}} {
		bad := bytes.Replace(valid, []byte(pair[0]), []byte(pair[1]), 1)
		if bytes.Equal(bad, valid) {
			t.Fatalf("fixture lacked %s", pair[0])
		}
		if _, err := DecodePreserving(bad); err == nil {
			t.Fatalf("accepted malformed fixed tuple %s", pair[1])
		}
	}
}

func FuzzPreservingManifestExtensions(f *testing.F) {
	for _, seed := range [][]byte{[]byte(`{"future":[90071992547409931234567890,1e10000]}`), []byte(`{"x":1,"x":2}`), []byte(`null`), []byte(`"<untrusted>"`)} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, extension []byte) {
		if len(extension) > 32<<10 || !json.Valid(extension) {
			return
		}
		original := extensionManifest(t)
		var root map[string]json.RawMessage
		if err := json.Unmarshal(original, &root); err != nil {
			t.Fatal(err)
		}
		root["fuzz_extension"] = bytes.Clone(extension)
		data, err := json.Marshal(root)
		if err != nil {
			t.Fatal(err)
		}
		sub, err := DecodePreserving(data)
		if err != nil {
			return
		} // Ambiguous keys remain a refusal.
		sub.Delivery = Delivering
		encoded, err := EncodePreserving(sub)
		if err != nil {
			t.Fatal(err)
		}
		var after map[string]json.RawMessage
		if err = json.Unmarshal(encoded, &after); err != nil {
			t.Fatal(err)
		}
		canonical := func(raw []byte) []byte {
			decoder := json.NewDecoder(bytes.NewReader(raw))
			decoder.UseNumber()
			var value any
			if err := decoder.Decode(&value); err != nil {
				t.Fatal(err)
			}
			b, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			return b
		}
		if !bytes.Equal(canonical(extension), canonical(after["fuzz_extension"])) {
			t.Fatal("extension changed")
		}
		files, err := EvidenceFiles(sub)
		if err != nil || len(files) != 1 || files[0] != "sketch-0.png" {
			t.Fatalf("extension gained file authority: %v %v", files, err)
		}
		decoded, err := DecodePreserving(encoded)
		if err != nil {
			t.Fatal(err)
		}
		repeated, err := EncodePreserving(decoded)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(encoded, repeated) {
			t.Fatal("canonical rewrite unstable")
		}
	})
}
