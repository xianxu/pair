package board

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func sketchCommandSamples() map[string]string {
	meta := `"operation":"op","sequence":1,"epoch":7`
	return map[string]string{
		"sketch-begin":        `{"operation":"op","sequence":0,"epoch":7,"seg":1,"color":0}`,
		"sketch-stroke-begin": `{` + meta + `,"sketch":"s","id":"p","node":3,"box":[0,1,20,30]}`,
		"sketch-points":       `{` + meta + `,"sketch":"s","id":"p","chunk":0,"pts":[[1,2,0.5]]}`,
		"sketch-stroke-end":   `{` + meta + `,"sketch":"s","id":"p","truncated":true}`,
		"sketch-undo":         `{` + meta + `,"sketch":"s","id":"p"}`,
		"sketch-redo":         `{` + meta + `,"sketch":"s","id":"p"}`,
		"sketch-label":        `{` + meta + `,"sketch":"s","label":"Move this"}`,
		"sketch-status":       `{` + meta + `,"sketch":"s"}`,
		"sketch-source":       `{` + meta + `,"sketch":"s","source":"handle","offset":0,"length":1024}`,
		"sketch-asset":        `{` + meta + `,"sketch":"s","source":"handle","asset":"` + strings.Repeat("a", 64) + `","offset":0,"length":1024}`,
	}
}
func TestSketchCommandUnionRejectsOwnedFields(t *testing.T) {
	for kind, raw := range sketchCommandSamples() {
		t.Run(kind, func(t *testing.T) {
			cmd, err := ParseSketchCommand(kind, []byte(raw))
			if err != nil {
				t.Fatal(err)
			}
			if cmd.Kind != kind || cmd.Operation != "op" || cmd.Epoch != 7 {
				t.Fatalf("command %+v", cmd)
			}
			for _, extra := range []string{`"t":1`, `"k":"sketch-label"`, `"digest":"forged"`, `"author":{"id":"operator"}`, `"owner":"operator"`, `"at":{"node":3}`, `"future":true`} {
				bad := strings.TrimSuffix(raw, "}") + "," + extra + "}"
				if _, err := ParseSketchCommand(kind, []byte(bad)); err == nil {
					t.Errorf("accepted %s", extra)
				}
			}
			var fields map[string]json.RawMessage
			json.Unmarshal([]byte(raw), &fields)
			for key := range fields {
				if key == "truncated" {
					continue
				}
				copy := map[string]json.RawMessage{}
				for k, v := range fields {
					copy[k] = v
				}
				delete(copy, key)
				bad, _ := json.Marshal(copy)
				if _, err := ParseSketchCommand(kind, bad); err == nil {
					t.Errorf("accepted missing %s", key)
				}
			}
		})
	}
	if _, err := ParseSketchCommand("sketch-interrupt", []byte(`{}`)); err == nil {
		t.Fatal("client can interrupt")
	}
}
func TestSketchCommandUnicodeAndBounds(t *testing.T) {
	samples := sketchCommandSamples()
	bad := []struct{ kind, raw string }{
		{"sketch-begin", strings.Replace(samples["sketch-begin"], `"sequence":0`, `"sequence":1`, 1)},
		{"sketch-begin", strings.Replace(samples["sketch-begin"], `"epoch":7`, `"epoch":9007199254740992`, 1)},
		{"sketch-begin", strings.Replace(samples["sketch-begin"], `"operation":"op"`, `"operation":"\ud800"`, 1)},
		{"sketch-begin", strings.Replace(samples["sketch-begin"], `"operation":"op"`, `"operation":"`+string([]byte{0xff})+`"`, 1)},
		{"sketch-begin", strings.Replace(samples["sketch-begin"], `"seg":1`, `"seg":null`, 1)},
		{"sketch-points", strings.Replace(samples["sketch-points"], `[1,2,0.5]`, `[1,2]`, 1)},
		{"sketch-points", strings.Replace(samples["sketch-points"], `[1,2,0.5]`, `[1,2,0.5,4]`, 1)},
		{"sketch-points", strings.Replace(samples["sketch-points"], `[1,2,0.5]`, `[1,null,0.5]`, 1)},
		{"sketch-points", strings.Replace(samples["sketch-points"], `[1,2,0.5]`, `[1,2,1.1]`, 1)},
		{"sketch-stroke-begin", strings.Replace(samples["sketch-stroke-begin"], `[0,1,20,30]`, `[0,1,0,30]`, 1)},
		{"sketch-stroke-end", strings.Replace(samples["sketch-stroke-end"], `true`, `null`, 1)},
		{"sketch-stroke-end", strings.Replace(samples["sketch-stroke-end"], `true`, `1`, 1)},
		{"sketch-label", strings.Replace(samples["sketch-label"], `Move this`, strings.Repeat("a", 1025), 1)},
		{"sketch-label", strings.Replace(samples["sketch-label"], `Move this`, `\udfff`, 1)},
		{"sketch-label", strings.Replace(samples["sketch-label"], `"label":`, `"label":"first","label":`, 1)},
	}
	for _, tc := range bad {
		if _, err := ParseSketchCommand(tc.kind, []byte(tc.raw)); err == nil {
			t.Errorf("accepted malformed %s: %q", tc.kind, tc.raw)
		}
	}
	if _, err := ParseSketchCommand("sketch-label", []byte(strings.Replace(samples["sketch-label"], `Move this`, `valid \ud83d\ude00`, 1))); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseSketchCommand("sketch-label", bytes.Repeat([]byte(" "), MaxSketchBody+1)); err == nil {
		t.Fatal("body bound absent")
	}
}
func TestSketchCommandDigestNormalized(t *testing.T) {
	raw := sketchCommandSamples()["sketch-stroke-end"]
	a, err := ParseSketchCommand("sketch-stroke-end", []byte(strings.Replace(raw, `,"truncated":true`, "", 1)))
	if err != nil {
		t.Fatal(err)
	}
	b, err := ParseSketchCommand("sketch-stroke-end", []byte(strings.Replace(raw, `true`, `false`, 1)))
	if err != nil {
		t.Fatal(err)
	}
	da, _ := SketchCommandDigest(a)
	db, _ := SketchCommandDigest(b)
	if da != db {
		t.Fatal("false changed normalized digest")
	}
	c, _ := ParseSketchCommand("sketch-stroke-end", []byte(raw))
	dc, _ := SketchCommandDigest(c)
	if dc == da {
		t.Fatal("truncation missing from digest")
	}
	c.Epoch++
	changed, _ := SketchCommandDigest(c)
	if changed == dc {
		t.Fatal("epoch missing from digest")
	}
}

func TestSketchReplyClosedUnionAndBounds(t *testing.T) {
	good := `{"operation":"op","sketch":"s","sequence":1,"next_sequence":2,"outcome":"accepted","code":"","source":{"handle":"h","time":1,"length":100,"doc":1,"tab":"t","vp":{"w":100,"h":100,"dpr":1}},"status":{"phase":"drawing","reason":"","label":"","open_stroke":"p","visible_top":"","redo_top":""}}`
	if _, err := ParseSketchReply([]byte(good)); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{strings.Replace(good, `"accepted"`, `"success"`, 1), strings.Replace(good, `"sequence":1`, `"sequence":9007199254740992`, 1), strings.Replace(good, `"handle":"h"`, `"handle":"\ud800"`, 1), strings.Replace(good, `"length":100`, `"length":16777217`, 1), strings.Replace(good, `"phase":"drawing"`, `"phase":"random"`, 1), strings.Replace(good, `"reason":""`, `"reason":null`, 1), strings.TrimSuffix(good, "}") + `,"future":true}`} {
		if _, err := ParseSketchReply([]byte(bad)); err == nil {
			t.Errorf("accepted bad reply %s", bad)
		}
	}
}

func FuzzSketchCommand(f *testing.F) {
	f.Add("sketch-status", []byte(`{"operation":"begin-op","sequence":0,"epoch":7}`))
	for kind, raw := range sketchCommandSamples() {
		f.Add(kind, []byte(raw))
		f.Add(kind, []byte(strings.TrimSuffix(raw, "}")+`,"owner":"operator"}`))
		f.Add(kind, []byte(strings.Replace(raw, `"operation":"op"`, `"operation":"\ud800"`, 1)))
	}
	f.Fuzz(func(t *testing.T, kind string, raw []byte) {
		c, err := ParseSketchCommand(kind, raw)
		if err != nil {
			return
		}
		canonical, err := SketchCommandCanonical(c)
		if err != nil {
			t.Fatal(err)
		}
		var projection struct {
			Kind    string          `json:"kind"`
			Command json.RawMessage `json:"command"`
		}
		if err = json.Unmarshal(canonical, &projection); err != nil {
			t.Fatal(err)
		}
		again, err := ParseSketchCommand(projection.Kind, projection.Command)
		if err != nil {
			t.Fatal(err)
		}
		got, err := SketchCommandCanonical(again)
		if err != nil || !bytes.Equal(got, canonical) {
			t.Fatalf("command normalization drift %s -> %s (%v)", canonical, got, err)
		}
	})
}

func TestSketchCommandStatusBeforeBeginID(t *testing.T) {
	raw := []byte(`{"operation":"begin-op","sequence":0,"epoch":7}`)
	c, err := ParseSketchStatus(raw)
	if err != nil {
		t.Fatal(err)
	}
	if c.Sketch != "" || c.Sequence != 0 {
		t.Fatalf("begin lookup %+v", c)
	}
	if _, err := SketchCommandDigest(c); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{`{"operation":"op","sequence":1,"epoch":7}`, `{"operation":"op","sequence":0,"epoch":7,"sketch":null}`, `{"operation":"op","sequence":0,"epoch":7,"sketch":""}`} {
		if _, err := ParseSketchStatus([]byte(bad)); err == nil {
			t.Errorf("accepted status %s", bad)
		}
	}
}

func TestSketchCommandDigestRejectsInvalidTypedUnicode(t *testing.T) {
	c, err := ParseSketchBegin([]byte(sketchCommandSamples()["sketch-begin"]))
	if err != nil {
		t.Fatal(err)
	}
	c.Operation = string([]byte{0xff})
	if _, err := SketchCommandDigest(c); err == nil {
		t.Fatal("typed malformed Unicode silently replaced before digest")
	}
}
