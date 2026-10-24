package submission

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sharedPageWire(t *testing.T) map[string]any {
	t.Helper()
	b, err := json.Marshal(sketchSubmission(t))
	if err != nil {
		t.Fatal(err)
	}
	var root map[string]any
	if err = json.Unmarshal(b, &root); err != nil {
		t.Fatal(err)
	}
	sk := root["sketches"].([]any)[0].(map[string]any)
	image := sk["image"].(map[string]any)
	plan := image["plan"].(map[string]any)
	sk["paths"] = plan["paths"]
	sk["index"] = float64(1)
	delete(sk, "image")
	src := sk["source"].(map[string]any)
	finish := map[string]any{}
	for k, v := range src {
		finish[k] = v
	}
	finish["t"] = sk["history_t"]
	finish["record_index"] = sk["history_record_index"]
	sk["finish"] = finish
	pageSource := map[string]any{}
	for k, v := range finish {
		if k != "provenance" {
			pageSource[k] = v
		}
	}
	pageSource["recording_id"] = "rec"
	root["sketch_page"] = map[string]any{"file": "sketch-page.png", "provenance": "replay-rendered", "source": pageSource, "region": []any{0, 0, 100, 100}, "scale": 1, "width": 100, "height": 100, "clipped": false, "privacy": sk["privacy"], "privacy_origin": sk["privacy_origin"], "privacy_rule": sk["privacy_rule"]}
	return root
}
func decodePageWire(t *testing.T, root map[string]any) (Submission, error) {
	t.Helper()
	b, e := json.Marshal(root)
	if e != nil {
		t.Fatal(e)
	}
	return DecodePreserving(b)
}
func TestSharedSketchPageManifest(t *testing.T) {
	s, err := decodePageWire(t, sharedPageWire(t))
	if err != nil {
		t.Fatal(err)
	}
	if err = Validate(s); err != nil {
		t.Fatal(err)
	}
	files, err := EvidenceFiles(s)
	if err != nil || len(files) != 1 || files[0] != "sketch-page.png" {
		t.Fatalf("inventory=%v err=%v", files, err)
	}
}

func TestSharedPageStrictRequiredAndForbiddenFields(t *testing.T) {
	for _, where := range []string{"page", "source", "sketch", "finish", "vp"} {
		var fields []string
		switch where {
		case "page":
			fields = []string{"file", "provenance", "source", "region", "scale", "width", "height", "clipped", "privacy", "privacy_rule"}
		case "source":
			fields = []string{"recording_id", "t", "record_index", "doc", "tab", "url", "title", "vp", "scroll"}
		case "finish":
			fields = []string{"t", "record_index", "doc", "tab", "url", "title", "vp", "scroll", "provenance"}
		case "sketch":
			fields = []string{"index", "paths", "finish"}
		case "vp":
			fields = []string{"w", "h", "dpr"}
		}
		for _, field := range fields {
			for _, null := range []bool{false, true} {
				if null && field == "privacy_rule" {
					continue
				}
				t.Run(where+"/"+field+map[bool]string{false: "/missing", true: "/null"}[null], func(t *testing.T) {
					root := sharedPageWire(t)
					page := root["sketch_page"].(map[string]any)
					sk := root["sketches"].([]any)[0].(map[string]any)
					obj := page
					switch where {
					case "source":
						obj = page["source"].(map[string]any)
					case "finish":
						obj = sk["finish"].(map[string]any)
					case "sketch":
						obj = sk
					case "vp":
						obj = page["source"].(map[string]any)["vp"].(map[string]any)
					}
					if null {
						obj[field] = nil
					} else {
						delete(obj, field)
					}
					_, err := decodePageWire(t, root)
					if err == nil {
						t.Fatal("accepted malformed required field")
					}
				})
			}
		}
	}
	for _, field := range []string{"index", "paths", "finish"} {
		for _, value := range []any{nil, 0} {
			b, _ := json.Marshal(sketchSubmission(t))
			var root map[string]any
			json.Unmarshal(b, &root)
			root["sketches"].([]any)[0].(map[string]any)[field] = value
			if _, err := decodePageWire(t, root); err == nil {
				t.Fatalf("legacy accepted %s=%v", field, value)
			}
		}
	}
	root := sharedPageWire(t)
	root["sketches"].([]any)[0].(map[string]any)["image"] = nil
	if _, err := decodePageWire(t, root); err == nil {
		t.Fatal("shared accepted image:null")
	}
}
func TestSharedPageSemanticWitnesses(t *testing.T) {
	for name, change := range map[string]func(map[string]any){
		"wrong index": func(r map[string]any) { r["sketches"].([]any)[0].(map[string]any)["index"] = 2 },
		"empty paths": func(r map[string]any) { r["sketches"].([]any)[0].(map[string]any)["paths"] = []any{} },
		"changed pressure": func(r map[string]any) {
			p := r["sketches"].([]any)[0].(map[string]any)["paths"].([]any)[0].(map[string]any)
			p["pts"].([]any)[0].([]any)[2] = 1.1
		},
		"finish history": func(r map[string]any) { r["sketches"].([]any)[0].(map[string]any)["finish"].(map[string]any)["t"] = 19 },
		"page ordinal": func(r map[string]any) {
			r["sketch_page"].(map[string]any)["source"].(map[string]any)["record_index"] = 2
		},
		"page title": func(r map[string]any) {
			r["sketch_page"].(map[string]any)["source"].(map[string]any)["title"] = "later title"
		},
		"page view": func(r map[string]any) {
			r["sketch_page"].(map[string]any)["source"].(map[string]any)["scroll"] = []any{1, 0}
		},
		"page scale":    func(r map[string]any) { r["sketch_page"].(map[string]any)["scale"] = 2 },
		"v2":            func(r map[string]any) { r["version"] = 2 },
		"no sketches":   func(r map[string]any) { r["sketches"] = nil },
		"page filename": func(r map[string]any) { r["sketch_page"].(map[string]any)["file"] = "../sketch-page.png" },
	} {
		t.Run(name, func(t *testing.T) {
			r := sharedPageWire(t)
			change(r)
			s, err := decodePageWire(t, r)
			if err == nil {
				err = Validate(s)
			}
			if err == nil {
				t.Fatal("accepted forged page witness")
			}
		})
	}
}

func TestSharedPageStoreLifecyclePreservesExtensionsAndInventory(t *testing.T) {
	root := sharedPageWire(t)
	root["future_root"] = json.Number("90071992547409931234567890")
	page := root["sketch_page"].(map[string]any)
	page["future_page"] = map[string]any{"n": json.Number("1e10000")}
	page["source"].(map[string]any)["future_source"] = true
	sk := root["sketches"].([]any)[0].(map[string]any)
	sk["finish"].(map[string]any)["future_finish"] = "retained"
	sk["paths"].([]any)[0].(map[string]any)["future_path"] = "ink"
	s, err := decodePageWire(t, root)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(tempRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	var pngData bytes.Buffer
	if err = png.Encode(&pngData, image.NewRGBA(image.Rect(0, 0, 100, 100))); err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{"sketch-page.png": pngData.Bytes()}
	if _, err = store.Prepare(s, files); err != nil {
		t.Fatal(err)
	}
	if _, err = store.ConfirmPrepared(s, files); err != nil {
		t.Fatal(err)
	}
	if _, err = store.Prepare(s, files); err != nil {
		t.Fatal("same ID retry", err)
	}
	for _, event := range []Event{Begin, Ack, Complete} {
		if _, err = store.Transition(s.ID, s.Target, event); err != nil {
			t.Fatal(err)
		}
		read, err := store.Read(s.ID)
		if err != nil {
			t.Fatal(err)
		}
		wire, err := json.Marshal(read)
		if err != nil {
			t.Fatal(err)
		}
		for _, value := range []string{"90071992547409931234567890", "1e10000", "future_source", "future_finish", "future_path"} {
			if !bytes.Contains(wire, []byte(value)) {
				t.Fatalf("lost %s: %s", value, wire)
			}
		}
		versions, err := store.DeliveredSketchVersions("rec")
		if err != nil {
			t.Fatal(err)
		}
		want := event != Begin
		if (len(versions) == 1) != want {
			t.Fatalf("event %v suppressed versions %v", event, versions)
		}
	}
	if err = os.Remove(filepath.Join(filepath.Dir(store.ManifestPath(s.ID)), "sketch-page.png")); err != nil {
		t.Fatal(err)
	}
	if _, err = store.ReadMetadata(s.ID); err == nil {
		t.Fatal("missing page PNG accepted")
	}
}
func TestSharedPageStoreRefusesWrongPNGAndImmutableContext(t *testing.T) {
	s, err := decodePageWire(t, sharedPageWire(t))
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(tempRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.Prepare(s, map[string][]byte{"sketch-page.png": pngBytes(t)}); err == nil {
		t.Fatal("wrong page dimensions accepted")
	}
	var pngData bytes.Buffer
	png.Encode(&pngData, image.NewRGBA(image.Rect(0, 0, 100, 100)))
	files := map[string][]byte{"sketch-page.png": pngData.Bytes()}
	if _, err = store.Prepare(s, files); err != nil {
		t.Fatal(err)
	}
	s.Sketches[0].Note = "changed after freeze"
	if _, err = store.ConfirmPrepared(s, files); err == nil {
		t.Fatal("changed frozen note accepted")
	}
	s.Sketches[0].Note = "shape note"
	s.SketchPage.Limitations = []string{"new limitation"}
	if _, err = store.ConfirmPrepared(s, files); err == nil {
		t.Fatal("changed page metadata accepted")
	}
}
func TestLegacySketchEncodingAndPageDecoderFailClosed(t *testing.T) {
	legacy := sketchSubmission(t)
	wire, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(wire, []byte("sketch_page")) || bytes.Contains(wire, []byte(`"finish"`)) || bytes.Contains(wire, []byte(`"index"`)) {
		t.Fatalf("legacy wire changed: %s", wire)
	}
	read, err := DecodePreserving(wire)
	if err != nil {
		t.Fatal(err)
	}
	again, err := json.Marshal(read)
	if err != nil || !bytes.Equal(wire, again) {
		t.Fatalf("legacy wire not stable err=%v", err)
	}
	// Frozen pre-page required-image contract refuses the new image-less sketch.
	root := sharedPageWire(t)
	sk := root["sketches"].([]any)[0]
	b, _ := json.Marshal(sk)
	old := *submissionSchema.fields["sketches"].element
	old.forbidden = nil // pre-revision decoder had no new variant fields
	if _, _, err = filterManifest(b, &old); err == nil {
		t.Fatal("old decoder accepted shared sketch without required image")
	}
}

func TestSharedPageBatchUsesLatestCompatibleFinishAndOneFile(t *testing.T) {
	root := sharedPageWire(t)
	first := root["sketches"].([]any)[0]
	b, _ := json.Marshal(first)
	var second map[string]any
	json.Unmarshal(b, &second)
	second["id"] = "second"
	second["index"] = 2
	second["history_t"] = 30
	second["history_record_index"] = 4
	finish := second["finish"].(map[string]any)
	finish["t"] = 30
	finish["record_index"] = 4
	finish["title"] = "Latest finish"
	// Paths retain their original stroke IDs; they are scoped to their sketch.
	root["sketches"] = []any{first, second}
	page := root["sketch_page"].(map[string]any)
	source := page["source"].(map[string]any)
	source["t"] = 30
	source["record_index"] = 4
	source["title"] = "Latest finish"
	s, err := decodePageWire(t, root)
	if err != nil {
		t.Fatal(err)
	}
	if err = Validate(s); err != nil {
		t.Fatal(err)
	}
	files, err := EvidenceFiles(s)
	if err != nil || len(files) != 1 {
		t.Fatalf("repeated page inventory %v err=%v", files, err)
	}
	finish["scroll"] = []any{3, 0}
	s, err = decodePageWire(t, root)
	if err == nil {
		err = Validate(s)
	}
	if err == nil {
		t.Fatal("incompatible finish view accepted")
	}
}

func TestSharedPageCompletionMustRemainInOriginalDocument(t *testing.T) {
	for _, field := range []string{"doc", "tab"} {
		t.Run(field, func(t *testing.T) {
			root := sharedPageWire(t)
			sk := root["sketches"].([]any)[0].(map[string]any)
			finish := sk["finish"].(map[string]any)
			pageSource := root["sketch_page"].(map[string]any)["source"].(map[string]any)
			value := any(2)
			if field == "tab" {
				value = "other-tab"
			}
			finish[field] = value
			pageSource[field] = value
			s, err := decodePageWire(t, root)
			if err == nil {
				err = Validate(s)
			}
			if err == nil {
				t.Fatal("completion crossed original document/tab")
			}
		})
	}
}

func TestSharedPageStoreKeepsPrivateFilesAndRefusesSymlink(t *testing.T) {
	s, err := decodePageWire(t, sharedPageWire(t))
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(tempRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	var data bytes.Buffer
	png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 100, 100)))
	if _, err = store.Prepare(s, map[string][]byte{"sketch-page.png": data.Bytes()}); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Dir(store.ManifestPath(s.ID))
	file := filepath.Join(dir, "sketch-page.png")
	for _, path := range []string{file, store.ManifestPath(s.ID)} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm()&0077 != 0 {
			t.Fatalf("nonprivate evidence mode %v", info.Mode())
		}
	}
	read, err := store.Read(s.ID)
	if err != nil {
		t.Fatal(err)
	}
	if read.Sketches[0].Paths[0].Pts[0][2] != 0.2 {
		t.Fatal("pressure changed in shared evidence")
	}
	if err = os.Rename(file, file+".real"); err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink("sketch-page.png.real", file); err != nil {
		t.Fatal(err)
	}
	if _, err = store.ReadMetadata(s.ID); err == nil {
		t.Fatal("metadata accepted symlink page image")
	}
	if _, err = store.Read(s.ID); err == nil {
		t.Fatal("read accepted symlink page image")
	}
}

func TestSharedPageAcceptsExplicitZeroFalseAndEmptyFacts(t *testing.T) {
	root := sharedPageWire(t)
	sk := root["sketches"].([]any)[0].(map[string]any)
	page := root["sketch_page"].(map[string]any)
	for _, source := range []map[string]any{sk["source"].(map[string]any), sk["finish"].(map[string]any), page["source"].(map[string]any)} {
		source["t"] = 0
		source["record_index"] = 0
		source["title"] = ""
		source["url"] = ""
	}
	sk["history_t"] = 0
	sk["history_record_index"] = 0
	sk["privacy_origin"] = ""
	page["privacy_origin"] = ""
	s, err := decodePageWire(t, root)
	if err != nil {
		t.Fatal(err)
	}
	if err = Validate(s); err != nil {
		t.Fatal(err)
	}
}

func TestReadingGuideExplainsSharedPageIndicesAndPrivacy(t *testing.T) {
	guide := ReadingGuide("/recording")
	for _, want := range []string{"sketch-page.png", "sketch_page", "index", "finish", "full-page privacy", "replay-rendered reconstructs", "replay-rendered-current-view"} {
		if !strings.Contains(guide, want) {
			t.Errorf("reading guide missing %q", want)
		}
	}
}

func currentPageWire(t *testing.T) map[string]any {
	root := sharedPageWire(t)
	page := root["sketch_page"].(map[string]any)
	sk := root["sketches"].([]any)[0].(map[string]any)
	page["provenance"] = "replay-rendered-current-view"
	source := page["source"].(map[string]any)
	source["doc"] = float64(3)
	source["record_index"] = source["record_index"].(float64) + 10
	source["t"] = source["t"].(float64) + 10
	path := sk["paths"].([]any)[0].(map[string]any)
	at := map[string]any{"node": float64(99), "box": path["at"].(map[string]any)["box"]}
	page["current"] = map[string]any{"projections": []any{map[string]any{"sketch_id": sk["id"], "path_id": path["id"], "at": at, "clip": []any{0, 0, 100, 100}}}}
	return root
}
func TestCurrentSharedPagePreservesOriginalEvidence(t *testing.T) {
	root := currentPageWire(t)
	s, err := decodePageWire(t, root)
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(s); err != nil {
		t.Fatal(err)
	}
	if s.SketchPage.Current == nil || s.Sketches[0].Source.Doc == s.SketchPage.Source.Doc || s.Sketches[0].Finish.Doc == s.SketchPage.Source.Doc {
		t.Fatal("current view overwrote historical facts")
	}
	for _, mutate := range []func(map[string]any){
		func(p map[string]any) { delete(p, "current") }, func(p map[string]any) { p["current"] = nil }, func(p map[string]any) { p["provenance"] = "replay-rendered" },
		func(p map[string]any) { p["current"].(map[string]any)["projections"] = []any{} },
	} {
		root := currentPageWire(t)
		mutate(root["sketch_page"].(map[string]any))
		s, err := decodePageWire(t, root)
		if err == nil && Validate(s) == nil {
			t.Fatal("mixed/missing current context accepted")
		}
	}
}

func TestCurrentPageStrictRequiredProjectionFields(t *testing.T) {
	for _, group := range []string{"current", "projection", "anchor", "clip", "box"} {
		names := map[string][]string{"current": {"projections"}, "projection": {"sketch_id", "path_id", "at", "clip"}, "anchor": {"node", "box"}, "clip": {"0", "1", "2", "3"}, "box": {"0", "1", "2", "3"}}[group]
		for _, name := range names {
			for _, bad := range []string{"missing", "null", "wrong"} {
				t.Run(group+"/"+name+"/"+bad, func(t *testing.T) {
					root := currentPageWire(t)
					current := root["sketch_page"].(map[string]any)["current"].(map[string]any)
					projection := current["projections"].([]any)[0].(map[string]any)
					if group == "clip" || group == "box" {
						index := int(name[0] - '0')
						var tuple []any
						if group == "clip" {
							tuple = projection["clip"].([]any)
						} else {
							tuple = projection["at"].(map[string]any)["box"].([]any)
						}
						if bad == "missing" {
							if group == "clip" {
								projection["clip"] = tuple[:3]
							} else {
								projection["at"].(map[string]any)["box"] = tuple[:3]
							}
						} else if bad == "null" {
							tuple[index] = nil
						} else {
							tuple[index] = "0"
						}
					} else {
						obj := current
						if group == "projection" {
							obj = projection
						} else if group == "anchor" {
							obj = projection["at"].(map[string]any)
						}
						if bad == "missing" {
							delete(obj, name)
						} else if bad == "null" {
							obj[name] = nil
						} else {
							obj[name] = false
						}
					}
					if s, err := decodePageWire(t, root); err == nil && Validate(s) == nil {
						t.Fatal("fabricated required projection value")
					}
				})
			}
		}
	}
}
func TestCurrentPageStorePreservesProjectionExtensions(t *testing.T) {
	root := currentPageWire(t)
	current := root["sketch_page"].(map[string]any)["current"].(map[string]any)
	current["future_context"] = json.Number("1e10000")
	projection := current["projections"].([]any)[0].(map[string]any)
	projection["future_projection"] = json.Number("90071992547409931234567890")
	projection["at"].(map[string]any)["future_anchor"] = true
	s, err := decodePageWire(t, root)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(tempRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	var pixels bytes.Buffer
	png.Encode(&pixels, image.NewRGBA(image.Rect(0, 0, 100, 100)))
	if _, err := store.Prepare(s, map[string][]byte{"sketch-page.png": pixels.Bytes()}); err != nil {
		t.Fatal(err)
	}

	for _, event := range []Event{Begin, Ack, Complete} {
		if _, err := store.Transition(s.ID, s.Target, event); err != nil {
			t.Fatal(err)
		}
		got, err := store.Read(s.ID)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := json.Marshal(got)
		if err != nil {
			t.Fatal(err)
		}
		for _, needle := range []string{"future_context", "1e10000", "90071992547409931234567890", "future_anchor"} {
			if !bytes.Contains(raw, []byte(needle)) {
				t.Fatalf("lost %s", needle)
			}
		}
	}
}
