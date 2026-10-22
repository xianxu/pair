package submission

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/webcast/recording"
)

func TestPreservingCodecRejectsNullPressureInsteadOfInventingZero(t *testing.T) {
	data := bytes.Replace(extensionManifest(t), []byte(`[20,20,0.2]`), []byte(`[20,20,null]`), 1)
	if sub, err := DecodePreserving(data); err == nil {
		t.Fatalf("fabricated pressure from null: %+v (semantic validation=%v)", sub.Sketches[0].Image.Plan.Paths[0].Pts, Validate(sub))
	}
}

func strictCodecManifest(t *testing.T) []byte {
	t.Helper()
	s := sketchSubmission(t)
	s.Notes = v3Fixture().Notes
	n := &s.Notes[0]
	box := recording.Box{0, 0, 2, 2}
	n.At = recording.NoteAt{Node: 1, Tag: "button", Classes: []string{"primary"}, Stable: map[string]string{"id": "save"}, Route: "/old", Kind: "element", Path: []string{"main", "button"}, Text: "Save", Box: &box}
	n.Author = recording.Author{ID: "author", Name: "Author", Color: "#ff0000"}
	n.Context.Node = 1
	n.Context.Landmark = 2
	n.Context.Text = "Context"
	n.Context.Box = &box
	n.Before = []string{"click save"}
	n.Revision = 1
	n.BoxText = BoxText{Region: box, VisibleText: []BoxTextEntry{{Element: "button", Text: "Save", Kind: "text", Role: "button"}}, AccessibleNames: []BoxTextEntry{{Element: "button", Text: "Save", Kind: "aria", Role: "button"}}, Truncated: true, Scope: "viewport", Skipped: []string{"frame"}}
	n.Crop.Settlement = &CaptureSettlement{Status: "settled"}
	s.Superseded = true
	n.PrivacyRule = PolicyEvidence{Rule: &OriginRule{Origin: n.PrivacyOrigin, Privacy: Private}}
	f := &s.Sketches[0].Elements.Anchors[0]
	f.Parent = 2
	f.Truncated = true
	f.Data = map[string]string{"data-action": "save"}
	s.Sketches[0].Elements.Bytes += len("data-action") + len("save")
	p := &s.Sketches[0].Image.Plan.Paths[0]
	p.At.Tag = "div"
	p.At.Classes = []string{"shape"}
	p.At.Path = []string{"main", "div"}
	p.At.Stable = map[string]string{"id": "shape"}
	p.At.Route = "/old"
	p.At.Kind = "element"
	p.At.Text = "Original"
	p.Truncated = true
	s.Sketches[0].Author.Name = "Operator"
	s.Sketches[0].Image.Limitations = []string{"recorded DOM"}
	if err := Validate(s); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

type scalarPath []string

func scalarPaths(t *testing.T, raw json.RawMessage, path scalarPath, out *[]scalarPath) {
	t.Helper()
	trim := bytes.TrimSpace(raw)
	if len(trim) == 0 || bytes.Equal(trim, []byte("null")) {
		return
	}
	switch trim[0] {
	case '{':
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			t.Fatal(err)
		}
		keys := make([]string, 0, len(fields))
		for key := range fields {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			scalarPaths(t, fields[key], append(append(scalarPath(nil), path...), key), out)
		}
	case '[':
		var elements []json.RawMessage
		if err := json.Unmarshal(raw, &elements); err != nil {
			t.Fatal(err)
		}
		for i, v := range elements {
			scalarPaths(t, v, append(append(scalarPath(nil), path...), strconv.Itoa(i)), out)
		}
	default:
		*out = append(*out, append(scalarPath(nil), path...))
	}
}
func replaceScalar(t *testing.T, raw json.RawMessage, path scalarPath, value json.RawMessage) []byte {
	t.Helper()
	if len(path) == 0 {
		return bytes.Clone(value)
	}
	var out []byte
	var err error
	if bytes.TrimSpace(raw)[0] == '{' {
		var fields map[string]json.RawMessage
		if err = json.Unmarshal(raw, &fields); err != nil {
			t.Fatal(err)
		}
		fields[path[0]] = replaceScalar(t, fields[path[0]], path[1:], value)
		out, err = json.Marshal(fields)
	} else {
		var elements []json.RawMessage
		if err = json.Unmarshal(raw, &elements); err != nil {
			t.Fatal(err)
		}
		i, parseErr := strconv.Atoi(path[0])
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		elements[i] = replaceScalar(t, elements[i], path[1:], value)
		out, err = json.Marshal(elements)
	}
	if err != nil {
		t.Fatal(err)
	}
	return out
}
func scalarAt(t *testing.T, raw json.RawMessage, path scalarPath) json.RawMessage {
	t.Helper()
	if len(path) == 0 {
		return raw
	}
	if bytes.TrimSpace(raw)[0] == '{' {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			t.Fatal(err)
		}
		return scalarAt(t, fields[path[0]], path[1:])
	}
	var elements []json.RawMessage
	if err := json.Unmarshal(raw, &elements); err != nil {
		t.Fatal(err)
	}
	i, err := strconv.Atoi(path[0])
	if err != nil {
		t.Fatal(err)
	}
	return scalarAt(t, elements[i], path[1:])
}
func TestPreservingCodecEveryKnownScalarRejectsNullAndWrongType(t *testing.T) {
	valid := strictCodecManifest(t)
	var paths []scalarPath
	scalarPaths(t, valid, nil, &paths)
	if len(paths) < 150 {
		t.Fatalf("fixture lost scalar/tuple coverage: %d", len(paths))
	}
	for _, path := range paths {
		original := bytes.TrimSpace(scalarAt(t, valid, path))
		bad := []string{"null", "{}", "[]"}
		switch original[0] {
		case '"':
			bad = append(bad, "false", "1")
		case 't', 'f':
			bad = append(bad, `"false"`, "0")
		default:
			bad = append(bad, `"0"`, "false")
		}
		for _, value := range bad {
			t.Run(strings.Join(path, "/")+"="+value, func(t *testing.T) {
				if _, err := DecodePreserving(replaceScalar(t, valid, path, []byte(value))); err == nil {
					t.Fatal("accepted invalid known scalar/tuple member")
				}
			})
		}
	}
}
func TestStoreRejectsNullKnownEvidenceOnEveryRead(t *testing.T) {
	root := tempRoot(t)
	store, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	sub := sketchSubmission(t)
	if _, err = store.Prepare(sub, map[string][]byte{"sketch-0.png": sketchPNG(t, sub.Sketches[0])}); err != nil {
		t.Fatal(err)
	}
	path := store.ManifestPath(sub.ID)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	bad := replaceScalar(t, data, scalarPath{"sketches", "0", "image", "plan", "paths", "0", "pts", "0", "2"}, []byte("null"))
	if err = os.WriteFile(path, bad, 0600); err != nil {
		t.Fatal(err)
	}
	fresh, err := NewStore(root)
	if err != nil {
		t.Fatal(err)
	}
	for name, read := range map[string]func() error{"Read": func() error { _, e := fresh.Read(sub.ID); return e }, "ReadMetadata": func() error { _, e := fresh.ReadMetadata(sub.ID); return e }, "Pending": func() error { _, e := fresh.Pending(sub.Target); return e }, "Delivered": func() error { _, e := fresh.DeliveredSketchVersions(sub.Source.RecordingID); return e }} {
		t.Run(name, func(t *testing.T) {
			if err := read(); err == nil {
				t.Fatal("wire pressure null silently became zero")
			}
		})
	}
}

func FuzzPreservingKnownScalarTypes(f *testing.F) {
	for _, seed := range []struct {
		index uint16
		value string
	}{{0, "null"}, {50, "null"}, {120, `"0"`}, {200, "false"}, {300, "{}"}, {400, "[]"}} {
		f.Add(seed.index, seed.value)
	}
	f.Fuzz(func(t *testing.T, index uint16, value string) {
		if len(value) > 128 || !json.Valid([]byte(value)) {
			return
		}
		data := strictCodecManifest(t)
		var paths []scalarPath
		scalarPaths(t, data, nil, &paths)
		path := paths[int(index)%len(paths)]
		original := bytes.TrimSpace(scalarAt(t, data, path))
		replacement := bytes.TrimSpace([]byte(value))
		invalid := bytes.Equal(replacement, []byte("null")) || replacement[0] == '{' || replacement[0] == '['
		if original[0] == '"' {
			invalid = invalid || replacement[0] != '"'
		} else if original[0] == 't' || original[0] == 'f' {
			invalid = invalid || !bytes.Equal(replacement, []byte("true")) && !bytes.Equal(replacement, []byte("false"))
		} else {
			invalid = invalid || replacement[0] == '"' || replacement[0] == 't' || replacement[0] == 'f'
		}
		if !invalid {
			return
		}
		if _, err := DecodePreserving(replaceScalar(t, data, path, replacement)); err == nil {
			t.Fatalf("accepted %s at %s", value, fmt.Sprint(path))
		}
	})
}

func TestPreservingCodecAllowsOnlyIntentionalNullableContainers(t *testing.T) {
	data := strictCodecManifest(t)
	allowed := []scalarPath{{"notes"}, {"sketches"}, {"notes", "0", "page"}, {"source", "page"}, {"notes", "0", "elements"}, {"notes", "0", "context", "box"}, {"notes", "0", "at", "box"}, {"notes", "0", "crop", "settlement"}, {"notes", "0", "before"}, {"notes", "0", "at", "classes"}, {"notes", "0", "at", "path"}, {"notes", "0", "at", "stable"}, {"privacy_rule"}, {"notes", "0", "privacy_rule"}, {"sketches", "0", "image", "plan", "paths", "0", "at", "box"}, {"sketches", "0", "image", "plan", "paths", "0", "pts"}, {"sketches", "0", "elements", "anchors", "0", "data"}}
	for _, path := range allowed {
		t.Run("nullable/"+strings.Join(path, "/"), func(t *testing.T) {
			if _, err := DecodePreserving(replaceScalar(t, data, path, []byte("null"))); err != nil {
				t.Fatalf("lost nullable pointer/slice/map compatibility: %v", err)
			}
		})
	}
	forbidden := []scalarPath{{}, {"target"}, {"source"}, {"notes", "0"}, {"notes", "0", "author"}, {"notes", "0", "box_text"}, {"notes", "0", "crop"}, {"sketches", "0"}, {"sketches", "0", "source"}, {"sketches", "0", "source", "vp"}, {"sketches", "0", "elements"}, {"sketches", "0", "image"}, {"sketches", "0", "image", "plan"}, {"sketches", "0", "image", "plan", "region"}, {"sketches", "0", "source", "scroll"}, {"sketches", "0", "image", "plan", "paths", "0"}, {"sketches", "0", "image", "plan", "paths", "0", "at"}, {"sketches", "0", "image", "plan", "paths", "0", "pts", "0"}}
	for _, path := range forbidden {
		t.Run("nonnullable/"+strings.Join(path, "/"), func(t *testing.T) {
			if _, err := DecodePreserving(replaceScalar(t, data, path, []byte("null"))); err == nil {
				t.Fatal("null erased known evidence structure")
			}
		})
	}
}

func TestPreservingCodecOptionalScalarsStrictWhenPresent(t *testing.T) {
	data := strictCodecManifest(t)
	for _, path := range []scalarPath{{"sketches", "0", "reason"}, {"sketches", "0", "elements", "anchors", "0", "unavailable"}, {"sketches", "0", "image", "plan", "paths", "0", "at", "sel"}, {"notes", "0", "at", "sel"}, {"notes", "0", "box_text", "visible_text", "0", "selector"}, {"notes", "0", "box_text", "unavailable"}, {"notes", "0", "crop", "unavailable"}} {
		if _, err := DecodePreserving(replaceScalar(t, data, path, []byte("null"))); err == nil {
			t.Fatalf("null optional scalar %v", path)
		}
	}
}
