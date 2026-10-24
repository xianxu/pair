package sessionsync

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollaborativeRoundNewWritesHighlightRetiredPresentation(t *testing.T) {
	for _, key := range []string{"presentation", "presentation-mode"} {
		for _, mode := range []string{"direct", "propose"} {
			for _, seam := range []string{"build", "apply", "section", "all", "frontmatter", "apply-file"} {
				t.Run(key+"/"+mode+"/"+seam, func(t *testing.T) {
					r := newRig(t)
					base := strings.Replace(statusDoc, "owner: ops:0\n", "owner: ops:0\n"+key+": "+mode+"\n", 1)
					r.put(r.p.File, base)
					r.put(r.p.Shadow, base)
					replacement := "one\ntwo\nthree\nfour\nfive\n"
					var round *Round
					var after string
					var err error
					switch seam {
					case "build":
						var e Edit
						e, err = BuildEdit(base, []EditRecord{{Old: "old\n", New: replacement, Occurrence: 1}}, r.now, true)
						round, after = &e.Round, e.Next
					case "apply":
						_, err = r.sync.Apply(ApplyRequest{File: r.p.File, IfHash: r.hash(""), Edits: []EditRecord{{Old: "old\n", New: replacement, Occurrence: 1}}})
					case "section":
						_, err = r.sync.Write(WriteRequest{File: r.p.File, Section: "Needs you", Body: replacement, IfHash: r.hash("Needs you")})
					case "all":
						_, err = r.sync.Write(WriteRequest{File: r.p.File, Body: strings.Replace(base, "old\n", replacement, 1), IfHash: r.hash("")})
					case "frontmatter":
						block, _ := FrontmatterBody(base)
						_, err = r.sync.Write(WriteRequest{File: r.p.File, Frontmatter: true, Body: strings.Replace(block, "owner: ops:0\n", "owner: ops:0\ntitle: clean\n", 1), IfHash: r.frontmatterHash()})
					case "apply-file":
						var result FileApplyResult
						dir := t.TempDir()
						result, err = ApplyFile(FileApplyRequest{File: r.p.File, IfHash: Hash(base), FlockPath: filepath.Join(dir, "flock"), RoundPath: filepath.Join(dir, "round"), Now: r.now, Edits: []EditRecord{{Old: "old\n", New: replacement, Occurrence: 1}}})
						round = result.Round
					}
					if err != nil {
						t.Fatal(err)
					}
					if seam != "build" {
						after = r.read(r.p.File)
					}
					if round == nil {
						result, e := r.sync.Round(r.p.File)
						if e != nil {
							t.Fatal(e)
						}
						round = result.Round
					}
					if round == nil || round.Presentation != "highlight" || len(round.Spans) == 0 {
						t.Fatalf("round=%+v", round)
					}
					for _, span := range round.Spans {
						if span.Presentation != "highlight" {
							t.Fatalf("span=%+v", span)
						}
					}
					if !strings.Contains(after, key+": "+mode+"\n") || strings.Contains(after, "🤖") {
						t.Fatalf("retired source changed or proposal emitted: %q", after)
					}
					if seam == "frontmatter" {
						if !strings.Contains(after, "title: clean\n") {
							t.Fatal("frontmatter edit missing")
						}
					} else if !strings.Contains(after, replacement) {
						t.Fatalf("clean replacement missing: %q", after)
					}
					if round.BaseHash != Hash(base) || round.ResultHash != Hash(after) {
						t.Fatalf("hashes=%+v", round)
					}
				})
			}
		}
	}
}

func TestCollaborativeRoundHistoricalPresentationSurvivesReadAndNoop(t *testing.T) {
	for _, mode := range []string{"direct", "propose"} {
		t.Run(mode, func(t *testing.T) {
			r := newRig(t)
			raw := `{"base_hash":"before","result_hash":"` + Hash(statusDoc) + `","at":"2026-10-10T23:43:00Z","presentation":"` + mode + `","future":{ "nested": [1e999] },"spans":[{"start":[0,0],"end":[0,1],"old":"a","new":"-","future":{ "deep": {"n":1e999} }},{"start":[0,1],"end":[0,2],"old":"b","new":"-","presentation":"highlight"}]}`
			r.put(r.p.Round, raw)
			result, err := r.sync.Round(r.p.File)
			if err != nil {
				t.Fatal(err)
			}
			if result.Round.Presentation != mode {
				t.Fatalf("changed historical round: %+v", result.Round)
			}
			body, err := RoundBody(*result.Round, statusDoc)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(body, `"presentation":"`+mode+`"`) || !strings.Contains(body, `"presentation":"highlight"`) || !strings.Contains(body, `{ "deep": {"n":1e999} }`) {
				t.Fatalf("lost record meaning/extensions: %s", body)
			}
			encoded, ok := encodeRound(*result.Round)
			if !ok || !bytes.Contains(encoded, []byte(`{ "nested": [1e999] }`)) || !bytes.Contains(encoded, []byte(`{ "deep": {"n":1e999} }`)) {
				t.Fatalf("lost nested extensions: %s", encoded)
			}
			_, err = r.sync.Write(WriteRequest{File: r.p.File, Section: "Needs you", Body: "old\n", IfHash: r.hash("Needs you")})
			if err != nil {
				t.Fatal(err)
			}
			if r.read(r.p.Round) != raw {
				t.Fatal("no-op rewrote historical round")
			}
			var artifact RoundArtifact
			if err := json.Unmarshal([]byte(`{"file":"note.md","body":"history","round":`+raw+`}`), &artifact); err != nil {
				t.Fatal(err)
			}
			saved, err := artifact.MarshalJSON()
			if err != nil || !bytes.Contains(saved, []byte(`"presentation":"`+mode+`"`)) || !bytes.Contains(saved, []byte(`{ "deep": {"n":1e999} }`)) {
				t.Fatalf("artifact rewrite lost history: %s %v", saved, err)
			}
		})
	}
}
