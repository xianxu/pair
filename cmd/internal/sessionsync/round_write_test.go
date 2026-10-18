package sessionsync

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func (r *rig) round(path string) *Round {
	r.t.Helper()
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		r.t.Fatal(err)
	}
	var round Round
	if err := json.Unmarshal(raw, &round); err != nil {
		r.t.Fatal(err)
	}
	return &round
}

func TestWriteDerivesARoundWithoutReasons(t *testing.T) {
	r := newRig(t)
	res, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Needs you", Body: "new\n", IfHash: r.hash("Needs you")})
	if err != nil {
		t.Fatal(err)
	}
	round := r.round(r.p.Round)
	if round == nil || len(round.Spans) != 1 {
		t.Fatalf("round=%+v", round)
	}
	s := round.Spans[0]
	if s.Old != "old" || s.New != "new" || s.Explain != "" {
		t.Fatalf("span=%+v", s)
	}
	if round.ResultHash != Hash(r.read(r.p.File)) || round.BaseHash != Hash(statusDoc) || round.Presentation != "highlight" {
		t.Fatalf("round header %+v", round)
	}
	if res.Warning != "" {
		t.Fatalf("warning %q", res.Warning)
	}
	// A write that changes nothing (the stamp is excluded) keeps that round.
	if _, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Needs you", Body: "new\n", IfHash: r.hash("Needs you")}); err != nil {
		t.Fatal(err)
	}
	if again := r.round(r.p.Round); again == nil || again.Spans[0].New != "new" {
		t.Fatalf("no-op write replaced the round: %+v", again)
	}
}

func TestApplyAttachesReasonsAndRefusesABadBatchWholesale(t *testing.T) {
	r := newRig(t)
	_, err := r.sync.Apply(ApplyRequest{File: r.p.File, IfHash: r.hash(""), Edits: []EditRecord{
		{Old: "old", Occurrence: 1, New: "fresh", Explain: "answer"},
		{Old: "trains\n", Occurrence: 1, New: "trains\nmore\n", Explain: "add"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	round := r.round(r.p.Round)
	if round == nil || len(round.Spans) != 2 || round.Spans[0].Explain != "answer" || round.Spans[1].Explain != "add" {
		t.Fatalf("round=%+v", round)
	}
	// One bad record: nothing changes, not the file, shadow, state or round.
	snap := func() []string {
		return []string{r.read(r.p.File), r.read(r.p.Shadow), r.read(r.p.State), r.read(r.p.Round)}
	}
	before := snap()
	_, err = r.sync.Apply(ApplyRequest{File: r.p.File, IfHash: r.hash(""), Edits: []EditRecord{
		{Old: "fresh", Occurrence: 1, New: "x"}, {Old: "absent", Occurrence: 1, New: "y"},
	}})
	wantRefusal(t, err, ExitRefused, "edits-invalid")
	var refusal *Refusal
	errors.As(err, &refusal)
	if len(refusal.Invalid) != 1 || refusal.Invalid[0] != (BadEdit{2, "not found"}) {
		t.Fatalf("invalid=%+v", refusal.Invalid)
	}
	if after := snap(); strings.Join(after, "\x00") != strings.Join(before, "\x00") {
		t.Fatal("a refused batch changed state")
	}
}

func TestApplyStaleAndLockedRefuseBeforeResolving(t *testing.T) {
	r := newRig(t)
	stale := r.hash("")
	r.put(r.p.File, statusDoc+"operator edit\n")
	before := r.read(r.p.File)
	_, err := r.sync.Apply(ApplyRequest{File: r.p.File, IfHash: stale, Edits: []EditRecord{{Old: "absent", Occurrence: 1, New: "x"}}})
	wantRefusal(t, err, ExitStale, "stale")
	r.put(r.p.Lock, "holder: operator\nat: 2026-10-10T23:50\n")
	_, err = r.sync.Apply(ApplyRequest{File: r.p.File, IfHash: r.hash(""), Edits: []EditRecord{{Old: "old", Occurrence: 1, New: "x"}}})
	wantRefusal(t, err, ExitOperatorLock, "operator-lock")
	if r.read(r.p.File) != before || r.round(r.p.Round) != nil {
		t.Fatal("a refused apply wrote")
	}
}

const proposeDoc = "---\ntype: session-sync\nowner: ops:0\npresentation: propose\n---\n\n_Updated 2026-10-10 23:43._\n\n## Needs you\n\nold\n\n## Trains\n\ntrains\n"

// Under presentation: propose the file, shadow, hash and round all follow
// the written marker bytes; the agent's own markers never read as edits.
func TestProposeWritesMarkersThatTheShadowAndHashFollow(t *testing.T) {
	r := newRig(t)
	r.put(r.p.File, proposeDoc)
	r.put(r.p.Shadow, proposeDoc)
	res, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Needs you", Body: "new\n", IfHash: r.hash("Needs you")})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.read(r.p.File), "🤖~old~{new}") {
		t.Fatalf("no marker in %q", r.read(r.p.File))
	}
	if res.Hash != r.hash("Needs you") {
		t.Fatal("the returned hash does not match the written section")
	}
	if d, err := r.sync.Diff(r.p.Shadow, r.p.File); err != nil || d != "" {
		t.Fatalf("own markers show as unread edits: %q %v", d, err)
	}
	if round := r.round(r.p.Round); round == nil || round.Presentation != "propose" || round.ResultHash != Hash(r.read(r.p.File)) {
		t.Fatalf("round=%+v", round)
	}
	// A frontmatter write is never proposed and leaves no stamp span.
	block, _ := FrontmatterBody(r.read(r.p.File))
	if _, err := r.sync.Write(WriteRequest{File: r.p.File, Frontmatter: true, Body: strings.Replace(block, "presentation: propose\n", "presentation: propose\nextra: 1\n", 1), IfHash: r.frontmatterHash()}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(r.read(r.p.File), "🤖{extra") {
		t.Fatal("frontmatter write was proposed")
	}
	for _, s := range r.round(r.p.Round).Spans {
		if strings.Contains(s.New, "Updated") {
			t.Fatalf("stamp span %+v", s)
		}
	}
	// A large change refuses with nothing written.
	before := r.read(r.p.File)
	_, err = r.sync.Write(WriteRequest{File: r.p.File, Section: "Trains", Body: "a\nb\nc\nd\n", IfHash: r.hash("Trains")})
	wantRefusal(t, err, ExitRefused, "propose-too-large")
	if r.read(r.p.File) != before {
		t.Fatal("a refused proposal wrote")
	}
}

func (r *rig) frontmatterHash() string {
	r.t.Helper()
	res, err := r.sync.ReadFrontmatter(r.p.File)
	if err != nil {
		r.t.Fatal(err)
	}
	return res.Hash
}

// No stale round survives: a failed round write leaves none, and a failure to
// clear the previous one refuses before the document moves.
func TestRoundFailureNeverLeavesAStaleRound(t *testing.T) {
	r := newRig(t)
	if _, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Needs you", Body: "one\n", IfHash: r.hash("Needs you")}); err != nil {
		t.Fatal(err)
	}
	r.sync.RoundWriter = func(string, []byte) error { return errors.New("disk full") }
	res, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Needs you", Body: "two\n", IfHash: r.hash("Needs you")})
	if err != nil || res.Warning == "" || !strings.Contains(r.read(r.p.File), "two") {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if r.round(r.p.Round) != nil {
		t.Fatal("an old round survived a failed round write")
	}
	// The previous round can't be cleared (a non-empty directory sits there).
	r.sync.RoundWriter = nil
	if err := os.MkdirAll(filepath.Join(r.p.Round, "x"), 0o755); err != nil {
		t.Fatal(err)
	}
	before := r.read(r.p.File)
	_, err = r.sync.Write(WriteRequest{File: r.p.File, Section: "Needs you", Body: "three\n", IfHash: r.hash("Needs you")})
	wantRefusal(t, err, ExitRefused, "refused")
	if r.read(r.p.File) != before {
		t.Fatal("the document moved although its old round could not be cleared")
	}
}

func TestRoundSidecarFollowsRegistration(t *testing.T) {
	r := newRig(t)
	if _, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Needs you", Body: "legacy\n", IfHash: r.hash("Needs you")}); err != nil {
		t.Fatal(err)
	}
	registered, err := r.sync.Register(r.p.File, "ops:0")
	if err != nil {
		t.Fatal(err)
	}
	central := filepath.Join(r.sync.registryStore().Dir, "state", registered.Registration.ID, "round.json")
	if round := r.round(central); round == nil || round.Spans[0].New != "legacy" {
		t.Fatalf("register did not import the round: %+v", round)
	}
	if _, err := r.sync.Write(WriteRequest{File: r.p.File, Section: "Needs you", Body: "central\n", IfHash: r.hash("Needs you")}); err != nil {
		t.Fatal(err)
	}
	if round := r.round(central); round == nil || round.Spans[0].New != "central" {
		t.Fatalf("registered write round: %+v", round)
	}
	moved := filepath.Join(filepath.Dir(r.p.File), "moved.md")
	if err := os.Rename(r.p.File, moved); err != nil {
		t.Fatal(err)
	}
	if _, err := r.sync.Rename(r.p.File, moved); err != nil {
		t.Fatal(err)
	}
	if r.round(central) == nil {
		t.Fatal("rename lost the round")
	}
	if _, err := r.sync.Forget(moved); err != nil {
		t.Fatal(err)
	}
	if round := r.round(PathsOf(moved).Round); round == nil || round.Spans[0].New != "central" {
		t.Fatalf("forget did not export the round: %+v", round)
	}
}
