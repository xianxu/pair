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

	"github.com/xianxu/pair/cmd/internal/webcast/query"
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
	s.Sketches[0].PrivacyRule = PolicyEvidence{Rule: &OriginRule{Origin: s.Sketches[0].PrivacyOrigin, Privacy: Private}}
	s.Sketches[0].Elements.Ancestors = []query.ElementFact{{Node: 2, Tag: "main"}}
	s.Sketches[0].Elements.Intersections = []query.ElementFact{{Node: 3, Tag: "span"}}
	s.Sketches[0].Elements.Bytes += 32 + len("main") + 32 + len("span")
	n.Elements = &s.Sketches[0].Elements
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

func removeManifestMember(t *testing.T, raw json.RawMessage, path scalarPath) []byte {
	t.Helper()
	if len(path) == 0 {
		t.Fatal("cannot delete whole manifest")
	}
	if len(path) == 1 {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			t.Fatal(err)
		}
		if _, present := fields[path[0]]; !present {
			t.Fatalf("fixture lacked member %v", path)
		}
		delete(fields, path[0])
		out, err := json.Marshal(fields)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	if bytes.TrimSpace(raw)[0] == '{' {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			t.Fatal(err)
		}
		fields[path[0]] = removeManifestMember(t, fields[path[0]], path[1:])
		out, err := json.Marshal(fields)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		t.Fatal(err)
	}
	i, err := strconv.Atoi(path[0])
	if err != nil {
		t.Fatal(err)
	}
	entries[i] = removeManifestMember(t, entries[i], path[1:])
	out, err := json.Marshal(entries)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
func TestPreservingCodecRequiresOriginalSourceMembers(t *testing.T) {
	for _, member := range []string{"t", "record_index", "scroll"} {
		t.Run(member, func(t *testing.T) {
			data := removeManifestMember(t, extensionManifest(t), scalarPath{"sketches", "0", "source", member})
			if sub, err := DecodePreserving(data); err == nil {
				t.Fatalf("missing source member %s fabricated evidence; Validate=%v source=%+v", member, Validate(sub), sub.Sketches[0].Source)
			}
		})
	}
}

type requiredMemberGroup struct {
	object  scalarPath
	members string
}

func requiredEvidenceGroups() []requiredMemberGroup {
	s := scalarPath{"sketches", "0"}
	p := func(tail ...string) scalarPath { return append(append(scalarPath(nil), s...), tail...) }
	groups := []requiredMemberGroup{
		{s, "id recording_id next_sequence digest author note history_t history_record_index phase source elements image privacy privacy_rule"},
		{p("author"), "id color"},
		{p("privacy_rule"), "origin privacy"},
		{p("source"), "t record_index doc tab url title vp scroll provenance"},
		{p("source", "vp"), "w h dpr"},
		{p("elements"), "anchors ancestors intersections truncated missing_layout inspected_nodes bytes"},
		{p("image"), "file provenance plan"},
		{p("image", "plan"), "sketch region scale width height clipped paths"},
		{p("image", "plan", "paths", "0"), "id at width pts truncated"},
		{p("image", "plan", "paths", "0", "at"), "node box"},
		{scalarPath{"notes", "0", "elements"}, "anchors ancestors intersections truncated missing_layout inspected_nodes bytes"},
	}
	for _, base := range []scalarPath{p("elements"), {"notes", "0", "elements"}} {
		for _, group := range []string{"anchors", "ancestors", "intersections"} {
			prefix := append(append(scalarPath(nil), base...), group, "0")
			groups = append(groups, requiredMemberGroup{prefix, "node"})
		}
	}
	return groups
}
func existingWireObject(t *testing.T, raw json.RawMessage, path scalarPath) (map[string]json.RawMessage, bool) {
	t.Helper()
	if len(path) == 0 {
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			return nil, false
		}
		return fields, fields != nil
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, false
	}
	switch trimmed[0] {
	case '{':
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(raw, &fields); err != nil {
			t.Fatal(err)
		}
		return existingWireObject(t, fields[path[0]], path[1:])
	case '[':
		var entries []json.RawMessage
		if err := json.Unmarshal(raw, &entries); err != nil {
			t.Fatal(err)
		}
		i, err := strconv.Atoi(path[0])
		if err != nil || i < 0 || i >= len(entries) {
			return nil, false
		}
		return existingWireObject(t, entries[i], path[1:])
	default:
		return nil, false
	}
}
func requiredEvidencePaths(t *testing.T, data []byte, requireAllGroups bool) []scalarPath {
	t.Helper()
	var paths []scalarPath
	for _, group := range requiredEvidenceGroups() {
		fields, present := existingWireObject(t, data, group.object)
		if !present {
			if requireAllGroups {
				t.Fatalf("full inventory fixture missing object group %v", group.object)
			}
			continue
		}
		for _, member := range strings.Fields(group.members) {
			if _, exists := fields[member]; !exists {
				t.Fatalf("required baseline fixture missing %v/%s", group.object, member)
			}
			paths = append(paths, append(append(scalarPath(nil), group.object...), member))
		}
	}
	return paths
}
func zeroDefaultSketchManifest(t *testing.T) []byte {
	t.Helper()
	s := sketchSubmission(t)
	sketch := &s.Sketches[0]
	sketch.Source.T = 0
	sketch.Source.RecordIndex = 0
	sketch.HistoryT = 0
	sketch.HistoryRecordIndex = 0
	sketch.Source.URL = ""
	sketch.Source.Title = ""
	sketch.PrivacyDecision = (OriginPolicy{}).Decide("")
	if err := Validate(s); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func TestPreservingCodecDeletesEveryRequiredEvidenceMember(t *testing.T) {
	cases := 0
	for name, data := range map[string][]byte{"full": strictCodecManifest(t), "explicit-zero-defaults": zeroDefaultSketchManifest(t)} {
		paths := requiredEvidencePaths(t, data, name == "full")
		want := 53
		if name == "full" {
			want = 67
		}
		if len(paths) != want {
			t.Fatalf("%s inventory changed: got%d want%d", name, len(paths), want)
		}
		t.Logf("%s: %d required deletion cases", name, len(paths))
		cases += len(paths)
		for _, path := range paths {
			t.Run(name+"/"+strings.Join(path, "/"), func(t *testing.T) {
				if _, err := DecodePreserving(removeManifestMember(t, data, path)); err == nil {
					t.Fatal("omitted required evidence member fabricated a Go default")
				}
			})
		}
	}
	if cases < 100 {
		t.Fatalf("required inventory coverage shrank: %d", cases)
	}
}

func TestPreservingCodecRetainsIntentionalOptionalEvidenceOmissions(t *testing.T) {
	data := strictCodecManifest(t)
	type omission struct {
		path     scalarPath
		explicit string
	}
	cases := []omission{
		{scalarPath{"sketches", "0", "reason"}, `""`},
		{scalarPath{"sketches", "0", "privacy_origin"}, `"https://example.test"`},
		{scalarPath{"sketches", "0", "author", "name"}, `"Operator"`},
		{scalarPath{"sketches", "0", "image", "limitations"}, `[]`},
		{scalarPath{"notes", "0", "elements"}, `null`},
		{scalarPath{"source", "page"}, `null`},
		{scalarPath{"notes", "0", "page"}, `null`},
		{scalarPath{"notes", "0", "crop", "settlement"}, `null`},
		{scalarPath{"notes", "0", "context", "box"}, `null`},
		{scalarPath{"notes", "0", "privacy_rule"}, `null`},
		{scalarPath{"how_to_read"}, `""`},
		{scalarPath{"superseded"}, `false`},
		{scalarPath{"notes"}, `null`},
	}
	anchor := scalarPath{"sketches", "0", "image", "plan", "paths", "0", "at"}
	for _, field := range []string{"tag", "classes", "stable", "route", "kind", "path", "sel", "text"} {
		value := `""`
		if field == "classes" || field == "path" {
			value = `[]`
		}
		if field == "stable" {
			value = `{}`
		}
		cases = append(cases, omission{append(append(scalarPath(nil), anchor...), field), value})
	}
	fact := scalarPath{"sketches", "0", "elements", "anchors", "0"}
	for _, field := range []string{"parent", "tag", "id", "data", "unavailable", "truncated"} {
		value := `""`
		if field == "parent" {
			value = `0`
		}
		if field == "truncated" {
			value = `false`
		}
		if field == "data" {
			value = `{}`
		}
		cases = append(cases, omission{append(append(scalarPath(nil), fact...), field), value})
	}
	// Ordinary notes share NoteAt but retain their historical region-only shape.
	for _, field := range []string{"node", "box"} {
		value := `null`
		if field == "node" {
			value = `0`
		}
		cases = append(cases, omission{scalarPath{"notes", "0", "at", field}, value})
	}
	t.Logf("%d intentional optional omission cases", len(cases))
	for _, test := range cases {
		t.Run(strings.Join(test.path, "/"), func(t *testing.T) {
			// Some omitempty values are absent from Go's encoding. Seed their valid
			// wire shape first, then delete; semantic dependencies remain Validate's job.
			present := replaceScalar(t, data, test.path, []byte(test.explicit))
			if _, err := DecodePreserving(removeManifestMember(t, present, test.path)); err != nil {
				t.Fatalf("optional omission lost compatibility: %v", err)
			}
		})
	}
}

func TestPreservingCodecLegacyOmissionDefaultsRemainValid(t *testing.T) {
	data, err := json.Marshal(fixture())
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []scalarPath{{"privacy_rule"}, {"how_to_read"}, {"notes", "0", "author"}, {"notes", "0", "at"}, {"notes", "0", "context"}, {"notes", "0", "box_text"}, {"notes", "0", "crop", "box"}, {"notes", "0", "crop", "captured_at"}, {"notes", "0", "crop", "document"}} {
		data = removeManifestMember(t, data, path)
	}
	sub, err := DecodePreserving(data)
	if err != nil {
		t.Fatal(err)
	}
	if err = Validate(sub); err != nil {
		t.Fatalf("legacy defaults changed: %v", err)
	}
}

func FuzzPreservingRequiredEvidenceMembers(f *testing.F) {
	for _, seed := range []uint16{0, 14, 16, 18, 24, 32, 40, 50, 66, 100} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, index uint16) {
		data := strictCodecManifest(t)
		paths := requiredEvidencePaths(t, data, true)
		path := paths[int(index)%len(paths)]
		// An unknown, arbitrarily large JSON number cannot satisfy a missing owned
		// field or gain authority while the original extension stays untouched.
		var root map[string]json.RawMessage
		if err := json.Unmarshal(data, &root); err != nil {
			t.Fatal(err)
		}
		root["future_required_probe"] = json.RawMessage(`{"huge":1e10000}`)
		data, err := json.Marshal(root)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := DecodePreserving(removeManifestMember(t, data, path)); err == nil {
			t.Fatalf("missing required %v accepted", path)
		}
	})
}

func TestPreservingCodecSketchPolicyUnionRetainsDefaultAndInvalidPolicy(t *testing.T) {
	for _, rule := range []string{`null`, `"invalid policy file: invalid JSON"`} {
		data := replaceScalar(t, strictCodecManifest(t), scalarPath{"sketches", "0", "privacy_rule"}, []byte(rule))
		sub, err := DecodePreserving(data)
		if err != nil {
			t.Fatal(err)
		}
		if err = Validate(sub); err != nil {
			t.Fatalf("valid policy union %s rejected: %v", rule, err)
		}
	}
}
func TestStoreRejectsMissingOriginalSourceMembersOnEveryRead(t *testing.T) {
	for _, member := range []string{"t", "record_index", "scroll"} {
		t.Run(member, func(t *testing.T) {
			root := tempRoot(t)
			store, err := NewStore(root)
			if err != nil {
				t.Fatal(err)
			}
			sub := sketchSubmission(t)
			files := map[string][]byte{"sketch-0.png": sketchPNG(t, sub.Sketches[0])}
			if _, err = store.Prepare(sub, files); err != nil {
				t.Fatal(err)
			}
			path := store.ManifestPath(sub.ID)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			bad := removeManifestMember(t, data, scalarPath{"sketches", "0", "source", member})
			if err = os.WriteFile(path, bad, 0600); err != nil {
				t.Fatal(err)
			}
			fresh, err := NewStore(root)
			if err != nil {
				t.Fatal(err)
			}
			readers := map[string]func() error{
				"Read": func() error { _, e := fresh.Read(sub.ID); return e }, "ReadMetadata": func() error { _, e := fresh.ReadMetadata(sub.ID); return e }, "Pending": func() error { _, e := fresh.Pending(sub.Target); return e }, "Delivered": func() error { _, e := fresh.DeliveredSketchVersions(sub.Source.RecordingID); return e }, "ObservePublication": func() error { _, _, e := fresh.ObservePublication(sub.ID); return e }, "ConfirmPrepared": func() error { _, e := fresh.ConfirmPrepared(sub, files); return e },
			}
			for name, read := range readers {
				t.Run(name, func(t *testing.T) {
					err := read()
					if err == nil {
						t.Fatal("missing original source fabricated defaults")
					}
					if !strings.Contains(err.Error(), "source") || !strings.Contains(err.Error(), member) {
						t.Fatalf("missing field diagnostic lost object path: %v", err)
					}
				})
			}
		})
	}
}
