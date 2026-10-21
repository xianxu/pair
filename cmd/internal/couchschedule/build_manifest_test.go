package couchschedule

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestBuildManifestIncludesCapturedBytesAndRecipe(t *testing.T) {
	first := manifestFor("toolchain1", []buildInput{{"root", "b", []byte("second")}, {"root", "a", []byte("first")}})
	reordered := manifestFor("toolchain1", []buildInput{{"root", "a", []byte("first")}, {"root", "b", []byte("second")}})
	if first.Key != reordered.Key {
		t.Fatal("enumeration order changed build key")
	}
	changed := manifestFor("toolchain1", []buildInput{{"root", "a", []byte("edited")}, {"root", "b", []byte("second")}})
	if first.Key == changed.Key {
		t.Fatal("changed source retained build key")
	}
	tool := manifestFor("toolchain2", []buildInput{{"root", "a", []byte("first")}, {"root", "b", []byte("second")}})
	if first.Key == tool.Key {
		t.Fatal("changed toolchain retained build key")
	}
}

func TestBuildManifestKnownFieldsStayStrict(t *testing.T) {
	m := manifestFor("tool", []buildInput{{"root", "main.go", []byte("source")}})
	m.BinarySHA256 = digestBuild([]byte("binary"))
	raw, e := encodeBuildManifest(m)
	if e != nil {
		t.Fatal(e)
	}
	var fields map[string]json.RawMessage
	if e = json.Unmarshal(raw, &fields); e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"Version", "Key", "BinarySHA256", "Toolchain", "Inputs"} {
		t.Run(name, func(t *testing.T) {
			copy := map[string]json.RawMessage{}
			for k, v := range fields {
				copy[k] = v
			}
			copy[name] = json.RawMessage("null")
			bad, _ := json.Marshal(copy)
			var got BuildManifest
			if e = json.Unmarshal(bad, &got); e == nil {
				t.Fatal("null known field accepted")
			}
			delete(copy, name)
			bad, _ = json.Marshal(copy)
			if e = json.Unmarshal(bad, &got); e == nil {
				t.Fatal("missing known field accepted")
			}
		})
	}
	var got BuildManifest
	bad := bytes.Replace(raw, []byte(`"Path":"main.go"`), []byte(`"Path":4`), 1)
	if e = json.Unmarshal(bad, &got); e == nil {
		t.Fatal("wrong input type accepted")
	}
	bad = append([]byte(`{"Version":1,"Version":1,`), raw[1:]...)
	if e = json.Unmarshal(bad, &got); e == nil {
		t.Fatal("duplicate owned member accepted")
	}
}
