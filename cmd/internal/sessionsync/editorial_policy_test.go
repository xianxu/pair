package sessionsync

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

func TestEditorialSectionHashBindsAuthority(t *testing.T) {
	r := newRig(t)
	base := strings.Replace(statusDoc, "owner: ops:0", "owner: ops:0\npresentation-mode: direct", 1)
	r.put(r.p.File, base)
	h := r.hash("Trains")
	// An unrelated body edit retains the scope token.
	r.put(r.p.File, strings.Replace(base, "old", "operator text", 1))
	if r.hash("Trains") != h {
		t.Fatal("unrelated text staled a section")
	}
	// Identical section bytes with changed review authority must stale it.
	r.put(r.p.File, strings.Replace(base, "presentation-mode: direct", "presentation-mode: propose", 1))
	if r.hash("Trains") == h {
		t.Fatal("changed document policy retained section token")
	}
	_, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Trains", Body: "new trains", IfHash: h})
	wantRefusal(t, err, ExitStale, "stale")
}

func TestEditorialAllAgentWriteSeamsProtectSettings(t *testing.T) {
	base := "---\ntype: session-sync\nowner: ops:0\ncollaboration-mode: chat\npresentation-mode: direct\n---\n## Group\n@@chat@@\n### A\n<!-- presentation-mode: direct -->\ntext\n"
	mutations := []struct{ name, old, next string }{
		{"frontmatter setting", "presentation-mode: direct", "presentation-mode: highlight"},
		{"heading setting", "<!-- presentation-mode: direct -->", "<!-- presentation-mode: highlight -->"},
		{"chat tag", "@@chat@@", "ordinary"},
		{"tag heading association", "### A", "### B"},
		{"tag attachment transfer", "### A", "### Injected\n### A"},
		{"directive attachment transfer", "<!-- presentation-mode: direct -->", "### Injected\n<!-- presentation-mode: direct -->"},
	}
	for _, mutation := range mutations {
		for _, registered := range []bool{false, true} {
			for _, verb := range []string{"all", "section", "frontmatter", "apply"} {
				// A section does not contain document settings; a frontmatter replacement
				// cannot reach a tag or heading in the body.
				if (verb == "frontmatter" && mutation.name != "frontmatter setting") || (verb == "section" && mutation.name == "frontmatter setting") {
					continue
				}
				t.Run(mutation.name+"/"+verb+map[bool]string{false: "-legacy", true: "-registered"}[registered], func(t *testing.T) {
					r := newRig(t)
					r.put(r.p.File, base)
					paths := r.p
					if registered {
						if _, err := r.sync.Register(r.p.File, "ops:0"); err != nil {
							t.Fatal(err)
						}
						paths = registeredPaths(t, r.sync, r.p.File)
					}
					scope := ""
					if verb == "section" {
						scope = "Group"
					}
					var read ReadResult
					var err error
					if verb == "frontmatter" {
						read, err = r.sync.ReadFrontmatter(r.p.File)
					} else {
						read, err = r.sync.Read(r.p.File, scope)
					}
					if err != nil {
						t.Fatal(err)
					}
					r.put(paths.Round, "previous round")
					capture := func() []string {
						return []string{r.read(r.p.File), r.read(paths.Shadow), r.read(paths.State), r.read(paths.Round)}
					}
					before := capture()
					next := strings.Replace(base, mutation.old, mutation.next, 1)
					switch verb {
					case "apply":
						_, err = r.sync.Apply(ApplyRequest{File: r.p.File, IfHash: read.Hash, Edits: []EditRecord{{Old: mutation.old, New: mutation.next, Occurrence: 1}}})
					case "frontmatter":
						block, e := FrontmatterBody(next)
						if e != nil {
							t.Fatal(e)
						}
						_, err = r.sync.Write(WriteRequest{File: r.p.File, Frontmatter: true, IfHash: read.Hash, Body: block})
					case "section":
						body, e := SectionBody(next, scope)
						if e != nil {
							t.Fatal(e)
						}
						_, err = r.sync.Write(WriteRequest{File: r.p.File, Section: scope, IfHash: read.Hash, Body: body})
					default:
						_, err = r.sync.Write(WriteRequest{File: r.p.File, IfHash: read.Hash, Body: next})
					}
					wantRefusal(t, err, ExitRefused, "operator-owned")
					if !reflect.DeepEqual(before, capture()) {
						t.Fatal("protected refusal changed file/shadow/state/round")
					}
				})
			}
		}
		t.Run(mutation.name+"/ApplyFile", func(t *testing.T) {
			r := newFileRig(t, base)
			if err := os.WriteFile(r.path, []byte("previous round"), 0600); err != nil {
				t.Fatal(err)
			}
			_, err := r.apply(Hash(base), EditRecord{Old: mutation.old, New: mutation.next, Occurrence: 1})
			wantRefusal(t, err, ExitRefused, "operator-owned")
			if r.read(r.doc) != base || r.read(r.path) != "previous round" {
				t.Fatal("standalone refusal changed artifacts")
			}
		})
	}
}

func TestEditorialMixedPolicyThroughWriteSeams(t *testing.T) {
	base := "---\ntype: session-sync\nowner: ops:0\npresentation-mode: direct\n---\n## Group\n\n### A\n<!-- presentation-mode: highlight -->\nold A\n### B\n<!-- presentation-mode: propose -->\nold B\n### C\nold C\n"
	clean := strings.ReplaceAll(base, "old ", "new ")
	want := clean
	edits := []EditRecord{{Old: "old A", New: "new A", Occurrence: 1}, {Old: "old B", New: "new B", Occurrence: 1}, {Old: "old C", New: "new C", Occurrence: 1}}
	check := func(t *testing.T, content string, round *Round) {
		t.Helper()
		if content != want || round == nil || len(round.Spans) != 3 {
			t.Fatalf("mixed output %s round %+v", content, round)
		}
		for i, mode := range []string{"highlight", "highlight", "highlight"} {
			if round.Spans[i].Presentation != mode {
				t.Fatalf("span %d: %+v", i, round.Spans[i])
			}
		}
	}
	for _, registered := range []bool{false, true} {
		for _, verb := range []string{"all", "section", "apply"} {
			t.Run(verb+map[bool]string{false: "-legacy", true: "-registered"}[registered], func(t *testing.T) {
				r := newRig(t)
				r.put(r.p.File, base)
				if registered {
					if _, err := r.sync.Register(r.p.File, "ops:0"); err != nil {
						t.Fatal(err)
					}
				}
				scope := ""
				body := clean
				if verb == "section" {
					scope = "Group"
					var err error
					body, err = SectionBody(clean, scope)
					if err != nil {
						t.Fatal(err)
					}
				}
				h := r.hash(scope)
				var err error
				if verb == "apply" {
					_, err = r.sync.Apply(ApplyRequest{File: r.p.File, IfHash: h, Edits: edits})
				} else {
					_, err = r.sync.Write(WriteRequest{File: r.p.File, Section: scope, IfHash: h, Body: body})
				}
				if err != nil {
					t.Fatal(err)
				}
				round, err := r.sync.Round(r.p.File)
				if err != nil {
					t.Fatal(err)
				}
				check(t, r.read(r.p.File), round.Round)
				if verb == "section" && r.hash(scope) == h {
					t.Fatal("changed section retained old token")
				}
			})
		}
	}
	r := newFileRig(t, base)
	result, err := r.apply(Hash(base), edits...)
	if err != nil {
		t.Fatal(err)
	}
	check(t, r.read(r.doc), result.Round)
}

func TestEditorialInsertionBeforeHeadingUsesPrecedingPolicy(t *testing.T) {
	base := "---\npresentation-mode: direct\n---\n# A\nleft\n# B\n<!-- presentation-mode: highlight -->\nright\n"
	e, err := BuildEdit(base, []EditRecord{{Old: "# B", New: "\n# B", Occurrence: 1}}, t0Propose, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(e.Round.Spans) != 1 || e.Round.Spans[0].Presentation != "highlight" {
		t.Fatalf("inserted preceding content borrowed later policy: %+v", e.Round)
	}
}

func TestEditorialReadRetiredValuesIsNonfatal(t *testing.T) {
	r := newRig(t)
	r.put(r.p.File, strings.Replace(statusDoc, "## Needs you\n", "## Needs you\n<!-- presentation-mode: bad -->\n", 1))
	for _, scope := range []string{"", "Needs you"} {
		read, err := r.sync.Read(r.p.File, scope)
		if err != nil {
			t.Fatal(err)
		}
		if read.Editorial == nil || len(read.Editorial.Diagnostics) == 0 {
			t.Fatal("missing shared compatibility notice")
		}
		for _, region := range read.Editorial.Regions {
			if region.Effective.Presentation != "highlight" {
				t.Fatalf("retired setting active: %+v", region)
			}
		}
	}
	if r.read(r.p.Shadow) != r.read(r.p.File) {
		t.Fatal("successful read did not acknowledge snapshot")
	}
}

func TestEditorialOperatorChangeAppliesOnFreshRead(t *testing.T) {
	r := newRig(t)
	base := "---\ntype: session-sync\nowner: ops:0\n---\n## A\n<!-- presentation-mode: direct -->\nold text\n"
	r.put(r.p.File, base)
	if _, err := r.sync.Read(r.p.File, ""); err != nil {
		t.Fatal(err)
	}
	// Normal editor writes can change authority between rounds.
	r.put(r.p.File, strings.Replace(base, "presentation-mode: direct", "presentation-mode: propose", 1))
	fresh, err := r.sync.Read(r.p.File, "")
	if err != nil || fresh.Editorial.Regions[1].Effective.Presentation != "highlight" {
		t.Fatalf("fresh authority: %+v %v", fresh, err)
	}
	if _, err = r.sync.Apply(ApplyRequest{File: r.p.File, IfHash: fresh.Hash, Edits: []EditRecord{{Old: "old text", New: "new text", Occurrence: 1}}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.read(r.p.File), "new text") || strings.Contains(r.read(r.p.File), "🤖~") {
		t.Fatal("retired setting changed new highlight content")
	}
}

func TestEditorialEmptyApplyIgnoresRetiredSettings(t *testing.T) {
	doc := "# A\n<!-- processing-mode: bad -->\nbody\n"
	r := newFileRig(t, doc)
	if err := os.WriteFile(r.path, []byte("old round bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := r.apply(Hash(doc))
	if err != nil {
		t.Fatal(err)
	}
	if r.read(r.doc) != doc || result.Hash != Hash(doc) || result.Body != "" || result.Round == nil || len(result.Round.Spans) != 0 || result.Round.Presentation != "highlight" {
		t.Fatalf("empty reply changed content or policy: %+v", result)
	}
	var artifact RoundArtifact
	if err := json.Unmarshal([]byte(r.read(r.path)), &artifact); err != nil {
		t.Fatal(err)
	}
	if artifact.Body != "" || len(artifact.Round.Spans) != 0 || artifact.Round.BaseHash != Hash(doc) || artifact.Round.ResultHash != Hash(doc) || !artifact.Round.At.Equal(r.now) {
		t.Fatalf("empty reply failed to replace previous round: %+v", artifact)
	}
}

func TestEditorialMixedPresentationUsesBasePolicy(t *testing.T) {
	base := "---\npresentation-mode: direct\n---\n# A\n<!-- presentation-mode: highlight -->\nold A\n# B\n<!-- presentation-mode: propose -->\nold B\n# C\nold C\n"
	e, err := BuildEdit(base, []EditRecord{{Old: "old A", New: "new A", Occurrence: 1}, {Old: "old B", New: "new B", Occurrence: 1}, {Old: "old C", New: "new C", Occurrence: 1}}, t0Propose, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(e.Next, "new A") || !strings.Contains(e.Next, "new B") || !strings.Contains(e.Next, "new C") {
		t.Fatalf("mixed result: %s", e.Next)
	}
	if len(e.Round.Spans) != 3 {
		t.Fatalf("spans: %+v", e.Round.Spans)
	}
	for i, want := range []string{"highlight", "highlight", "highlight"} {
		if e.Round.Spans[i].Presentation != want {
			t.Errorf("span %d: %+v", i, e.Round.Spans[i])
		}
	}
}

func TestEditorialRoundCombinesChatAndCollaboration(t *testing.T) {
	base := "---\ncollaboration-mode: discussion\npresentation-mode: direct\n---\n# Questions\n<!-- collaboration-mode: chat; processing-mode: proofreading -->\n- Why retry?\n# Notes\n<!-- collaboration-mode: collaboration; presentation-mode: highlight -->\nRetries help. 🤖[Explain briefly]\n"
	link := " → [answer](note.side/2026-10/2026-10-23-01-retries.md)"
	e, err := BuildEdit(base, []EditRecord{{Old: "- Why retry?", New: "- Why retry?" + link, Occurrence: 1}, {Old: "Retries help. 🤖[Explain briefly]", New: "Retries recover transient failures.", Occurrence: 1}}, t0Propose, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(e.Next, "- Why retry?"+link) || !strings.Contains(e.Next, "Retries recover transient failures.") {
		t.Fatal(e.Next)
	}
	p, err := compileEditorial(base)
	if err != nil {
		t.Fatal(err)
	}
	chatFound := false
	for _, region := range p.Regions {
		if region.Effective.Collaboration == "chat" {
			chatFound = true
		}
	}
	if !chatFound || p.Regions[len(p.Regions)-1].Effective.Collaboration != "collaboration" {
		t.Fatalf("routing: %+v", p.Regions)
	}
	if len(e.Round.Spans) != 2 || e.Round.Spans[0].Presentation != "highlight" || e.Round.Spans[1].Presentation != "highlight" {
		t.Fatalf("round %+v", e.Round)
	}
}

func TestEditorialCleanChatAnswerCountsAsLinked(t *testing.T) {
	base := "---\ncollaboration-mode: chat\npresentation-mode: propose\n---\n- Why retry?\n"
	link := " → [answer](note.side/2026-10/2026-10-23-01-retries.md)"
	e, err := BuildEdit(base, []EditRecord{{Old: "- Why retry?", New: "- Why retry?" + link, Occurrence: 1}}, t0Propose, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(e.Next, link) || strings.Contains(e.Next, "🤖{") {
		t.Fatalf("new answer was not clean highlight content: %s", e.Next)
	}
	line := strings.Split(strings.TrimSpace(e.Next), "\n")
	last := line[len(line)-1]
	_, err = BuildEdit(e.Next, []EditRecord{{Old: last, New: last + " → [answer](other.md)", Occurrence: 1}}, t0Propose, false)
	wantRefusal(t, err, ExitRefused, "editorial-chat")
}

func TestEditorialChatProtectsCompleteFencedItem(t *testing.T) {
	for _, content := range []string{"  example\n", ""} {
		base := "---\ncollaboration-mode: chat\n---\n- Why?\n\n  ```\n" + content + "  ```\n"
		_, err := BuildEdit(base, []EditRecord{{Old: "  ```", New: "  ~~~", Occurrence: 2}}, t0Propose, false)
		wantRefusal(t, err, ExitRefused, "editorial-chat")
	}
}

func TestEditorialChatAnswerMustPreserveMarkdownStructure(t *testing.T) {
	prefix := "---\ncollaboration-mode: chat\npresentation-mode: direct\n---\n"
	for _, tc := range []struct{ name, body, tail string }{
		{"fenced", "- Why?\n\n  ```\n  example\n  ```\n", "  ```"},
		{"empty fence", "- Why?\n\n  ```\n  ```\n", "  ```"},
		{"indented code", "- Why?\n\n      example\n", "      example"},
		{"thematic break", "- Why?\n\n  ---\n", "  ---"},
		{"HTML block", "- Why?\n\n  <!-- comment -->\n", "  <!-- comment -->"},
		{"heading delimiter", "- # Ask #\n", "#"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			edit, err := BuildEdit(prefix+tc.body, []EditRecord{{Old: tc.body, New: strings.TrimSuffix(tc.body, "\n") + " → [answer](note.side/a.md)\n", Occurrence: 1}}, t0Propose, false)
			if err == nil {
				var rendered bytes.Buffer
				body := strings.TrimPrefix(edit.Next, prefix)
				if err := goldmark.Convert([]byte(body), &rendered); err != nil {
					t.Fatal(err)
				}
				t.Fatalf("answer appended to syntax tail %q; rendered=%s", tc.tail, rendered.String())
			}
			wantRefusal(t, err, ExitRefused, "editorial-chat")
		})
	}
	body := "- Why?\n\n  ```\n  example\n  ```\n\n  Last continuation.\n"
	edit, err := BuildEdit(prefix+body, []EditRecord{{Old: "Last continuation.", New: "Last continuation. → [answer](note.side/a.md)", Occurrence: 1}}, t0Propose, false)
	if err != nil {
		t.Fatal(err)
	}
	var rendered bytes.Buffer
	if err := goldmark.Convert([]byte(strings.TrimPrefix(edit.Next, prefix)), &rendered); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rendered.String(), "<pre><code>example\n</code></pre>") || !strings.Contains(rendered.String(), `<p>Last continuation. → <a href="note.side/a.md">answer</a></p>`) {
		t.Fatalf("block structure or visible answer changed: %s", rendered.String())
	}
}

func TestEditorialChatProtectsItemStructureAcrossOutsideEdits(t *testing.T) {
	prefix := "---\ncollaboration-mode: chat\npresentation-mode: direct\n---\n"
	for _, tc := range []struct{ name, body, old, next string }{
		{"code rescope", "Preamble.\n\n- Why?\n", "Preamble.", "Preamble.\n\n```"},
		{"continued after item", "- Why?\n\n", "\n\n", "\n  Changed ask continuation.\n\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			edit, err := BuildEdit(prefix+tc.body, []EditRecord{{Old: tc.old, New: tc.next, Occurrence: 1}}, t0Propose, false)
			if err == nil {
				var rendered bytes.Buffer
				if err := goldmark.Convert([]byte(strings.TrimPrefix(edit.Next, prefix)), &rendered); err != nil {
					t.Fatal(err)
				}
				t.Fatalf("changed protected item structure: %s", rendered.String())
			}
			wantRefusal(t, err, ExitRefused, "editorial-chat")
		})
	}
	base := prefix + "- Original ask 🤖[revise this ask]\n"
	edit, err := BuildEdit(base, []EditRecord{{Old: "Original ask", New: "Explicitly revised ask", Occurrence: 1}}, t0Propose, false)
	if err != nil || !strings.Contains(edit.Next, "Explicitly revised ask") {
		t.Fatalf("ready referral inside chat lost authority: %v, %+v", err, edit)
	}
}

func TestEditorialCodeMarkerWritesStayHighlight(t *testing.T) {
	for _, body := range []string{"```md\n🤖[question]\n```\n", "`🤖[question]`\n", "    🤖[question]\n"} {
		base := "---\npresentation-mode: propose\n---\n" + body
		e, err := BuildEdit(base, []EditRecord{{Old: "🤖[question]", New: "→ [discussion](note.side/2026-10/a.md)", Occurrence: 1}}, t0Propose, false)
		if err != nil {
			t.Fatal(err)
		}
		if e.Round.Presentation != "highlight" {
			t.Fatalf("code changed new presentation: %+v", e.Round)
		}
	}
	link := "→ [discussion](note.side/2026-10/a.md)"
	base := "---\npresentation-mode: propose\n---\n```md\n" + link + " 🤖[fold in]\na\nb\nc\nd\n```\n"
	edit, err := BuildEdit(base, []EditRecord{{Old: link + " 🤖[fold in]\na\nb\nc\nd", New: "A\nB\nC\nD\nE " + link + " (folded 2026-10-23)", Occurrence: 1}}, t0Propose, false)
	if err != nil {
		t.Fatal(err)
	}
	if edit.Round.Presentation != "highlight" {
		t.Fatalf("retired size limit changed policy: %+v", edit.Round)
	}
}

func TestEditorialSectionHashBindsTagOutsideBody(t *testing.T) {
	for _, taggedHeading := range []string{"# Group", "## A"} {
		t.Run(taggedHeading, func(t *testing.T) {
			r := newRig(t)
			plain := "---\ntype: session-sync\nowner: ops:0\n---\n# Other\nunrelated\n# Group\n## A\nnumbered ask prose\n"
			tagged := strings.Replace(plain, taggedHeading, "@@chat@@\n"+taggedHeading, 1)
			r.put(r.p.File, tagged)
			token := r.hash("A")
			r.put(r.p.File, strings.Replace(tagged, "unrelated", "operator unrelated prose", 1))
			if r.hash("A") != token {
				t.Fatal("unrelated prose changed section authority token")
			}
			r.put(r.p.File, plain)
			if r.hash("A") == token {
				t.Fatal("removing tag outside section body retained token")
			}
			_, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "A", IfHash: token, Body: "replacement\n"})
			wantRefusal(t, err, ExitStale, "stale")
		})
	}
}

func TestBuildEditRejectsDeclarationAttachmentTransfer(t *testing.T) {
	for _, body := range []string{"@@chat@@\n# Questions\n", "# Questions\n<!-- collaboration-mode: chat -->\n", "# Questions\n<!-- presentation-mode: direct -->\n"} {
		base := "---\ncollaboration-mode: chat\n---\n" + body + "1. Why?\n"
		old, next := "# Questions", "# Injected\n# Questions"
		if !strings.HasPrefix(body, "@@") {
			old = strings.Split(body, "\n")[1]
			next = "# Injected\n" + old
		}
		_, err := BuildEdit(base, []EditRecord{{Old: old, New: next, Occurrence: 1}}, t0Propose, false)
		wantRefusal(t, err, ExitRefused, "operator-owned")
	}
}
