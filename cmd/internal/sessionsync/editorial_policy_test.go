package sessionsync

import (
	"os"
	"reflect"
	"strings"
	"testing"
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
	base := "---\ntype: session-sync\nowner: ops:0\npresentation-mode: direct\n---\n## A\n<!-- presentation-mode: direct -->\ntext\n"
	for _, registered := range []bool{false, true} {
		for _, verb := range []string{"all", "section", "frontmatter", "apply"} {
			t.Run(verb+map[bool]string{false: "-legacy", true: "-registered"}[registered], func(t *testing.T) {
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
					scope = "A"
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
				next := strings.Replace(base, "presentation-mode: direct", "presentation-mode: highlight", 1)
				switch verb {
				case "apply":
					_, err = r.sync.Apply(ApplyRequest{File: r.p.File, IfHash: read.Hash, Edits: []EditRecord{{Old: "presentation-mode: direct", New: "presentation-mode: highlight", Occurrence: 1}}})
				case "section":
					_, err = r.sync.Write(WriteRequest{File: r.p.File, Section: "A", IfHash: read.Hash, Body: "<!-- presentation-mode: highlight -->\ntext\n"})
				case "frontmatter":
					block, e := FrontmatterBody(next)
					if e != nil {
						t.Fatal(e)
					}
					_, err = r.sync.Write(WriteRequest{File: r.p.File, Frontmatter: true, IfHash: read.Hash, Body: block})
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
	r := newFileRig(t, base)
	if err := os.WriteFile(r.path, []byte("previous round"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := r.apply(Hash(base), EditRecord{Old: "presentation-mode: direct", New: "presentation-mode: highlight", Occurrence: 1})
	wantRefusal(t, err, ExitRefused, "operator-owned")
	if r.read(r.doc) != base || r.read(r.path) != "previous round" {
		t.Fatal("standalone refusal changed artifacts")
	}
}

func TestEditorialMixedPolicyThroughWriteSeams(t *testing.T) {
	base := "---\ntype: session-sync\nowner: ops:0\npresentation-mode: direct\n---\n## Group\n\n### A\n<!-- presentation-mode: highlight -->\nold A\n### B\n<!-- presentation-mode: propose -->\nold B\n### C\nold C\n"
	clean := strings.ReplaceAll(base, "old ", "new ")
	want := strings.Replace(clean, "new B", "🤖~old~{new} B", 1)
	edits := []EditRecord{{Old: "old A", New: "new A", Occurrence: 1}, {Old: "old B", New: "new B", Occurrence: 1}, {Old: "old C", New: "new C", Occurrence: 1}}
	check := func(t *testing.T, content string, round *Round) {
		t.Helper()
		if content != want || round == nil || len(round.Spans) != 3 {
			t.Fatalf("mixed output %s round %+v", content, round)
		}
		for i, mode := range []string{"highlight", "propose", "direct"} {
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
	if len(e.Round.Spans) != 1 || e.Round.Spans[0].Presentation != "direct" {
		t.Fatalf("inserted preceding content borrowed later policy: %+v", e.Round)
	}
}

func TestEditorialReadInvalidDoesNotAcknowledge(t *testing.T) {
	r := newRig(t)
	before := r.read(r.p.Shadow)
	r.put(r.p.File, strings.Replace(statusDoc, "## Needs you\n", "## Needs you\n<!-- presentation-mode: bad -->\n", 1))
	for _, scope := range []string{"", "Needs you"} {
		if _, err := r.sync.Read(r.p.File, scope); err == nil {
			t.Fatal("read accepted invalid settings")
		}
	}
	if r.read(r.p.Shadow) != before {
		t.Fatal("invalid read acknowledged snapshot")
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
	if err != nil || fresh.Editorial.Regions[1].Effective.Presentation != "propose" {
		t.Fatalf("fresh authority: %+v %v", fresh, err)
	}
	if _, err = r.sync.Apply(ApplyRequest{File: r.p.File, IfHash: fresh.Hash, Edits: []EditRecord{{Old: "old text", New: "new text", Occurrence: 1}}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.read(r.p.File), "🤖~old~{new} text") {
		t.Fatal("fresh operator setting was not enforced")
	}
}

func TestEditorialEmptyApplyValidatesBeforeRetiringRound(t *testing.T) {
	doc := "# A\n<!-- processing-mode: bad -->\nbody\n"
	r := newFileRig(t, doc)
	oldRound := "old round bytes"
	if err := os.WriteFile(r.path, []byte(oldRound), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := r.apply(Hash(doc)); err == nil {
		t.Fatal("empty apply accepted invalid settings")
	}
	if r.read(r.path) != oldRound || r.read(r.doc) != doc {
		t.Fatal("refused empty apply had effects")
	}
}

func TestEditorialMixedPresentationUsesBasePolicy(t *testing.T) {
	base := "---\npresentation-mode: direct\n---\n# A\n<!-- presentation-mode: highlight -->\nold A\n# B\n<!-- presentation-mode: propose -->\nold B\n# C\nold C\n"
	e, err := BuildEdit(base, []EditRecord{{Old: "old A", New: "new A", Occurrence: 1}, {Old: "old B", New: "new B", Occurrence: 1}, {Old: "old C", New: "new C", Occurrence: 1}}, t0Propose, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(e.Next, "new A") || !strings.Contains(e.Next, "🤖~old~{new} B") || !strings.Contains(e.Next, "new C") {
		t.Fatalf("mixed result: %s", e.Next)
	}
	if len(e.Round.Spans) != 3 {
		t.Fatalf("spans: %+v", e.Round.Spans)
	}
	for i, want := range []string{"highlight", "propose", "direct"} {
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
	if p.Regions[1].Effective.Collaboration != "chat" || p.Regions[2].Effective.Collaboration != "collaboration" {
		t.Fatalf("routing: %+v", p.Regions)
	}
	if len(e.Round.Spans) != 2 || e.Round.Spans[0].Presentation != "direct" || e.Round.Spans[1].Presentation != "highlight" {
		t.Fatalf("round %+v", e.Round)
	}
}

func TestEditorialPendingChatAnswerCountsAsLinked(t *testing.T) {
	base := "---\ncollaboration-mode: chat\npresentation-mode: propose\n---\n- Why retry?\n"
	link := " → [answer](note.side/2026-10/2026-10-23-01-retries.md)"
	e, err := BuildEdit(base, []EditRecord{{Old: "- Why retry?", New: "- Why retry?" + link, Occurrence: 1}}, t0Propose, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(e.Next, `🤖{ → \[answer\](`) {
		t.Fatalf("answer not pending: %s", e.Next)
	}
	line := strings.Split(strings.TrimSpace(e.Next), "\n")
	last := line[len(line)-1]
	_, err = BuildEdit(e.Next, []EditRecord{{Old: last, New: last + " → [answer](other.md)", Occurrence: 1}}, t0Propose, false)
	wantRefusal(t, err, ExitRefused, "editorial-chat")
}

func TestEditorialCodeMarkerCannotApproveDirectSideDoc(t *testing.T) {
	for _, body := range []string{"```md\n🤖[question]\n```\n", "`🤖[question]`\n", "    🤖[question]\n"} {
		base := "---\npresentation-mode: propose\n---\n" + body
		e, err := BuildEdit(base, []EditRecord{{Old: "🤖[question]", New: "→ [discussion](note.side/2026-10/a.md)", Occurrence: 1}}, t0Propose, false)
		if err == nil && !strings.Contains(e.Next, "🤖~") {
			t.Fatalf("code granted approval: %s", e.Next)
		}
	}
	link := "→ [discussion](note.side/2026-10/a.md)"
	base := "---\npresentation-mode: propose\n---\n```md\n" + link + " 🤖[fold in]\na\nb\nc\nd\n```\n"
	_, err := BuildEdit(base, []EditRecord{{Old: link + " 🤖[fold in]\na\nb\nc\nd", New: "A\nB\nC\nD\nE " + link + " (folded 2026-10-23)", Occurrence: 1}}, t0Propose, false)
	wantRefusal(t, err, ExitRefused, "propose-too-large")
}
