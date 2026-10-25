package sessionsync

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyPreservesUnreadEditsElsewhere(t *testing.T) {
	for _, registered := range []bool{false, true} {
		t.Run(map[bool]string{false: "legacy", true: "registered"}[registered], func(t *testing.T) {
			r := newRig(t)
			paths := r.p
			if registered {
				if _, err := r.sync.Register(r.p.File, "ops:0"); err != nil {
					t.Fatal(err)
				}
				paths = registeredPaths(t, r.sync, r.p.File)
			}
			current := strings.Replace(statusDoc, "\nold\n", "\noperator unread\n", 1)
			r.put(r.p.File, current)
			result, err := r.sync.Apply(ApplyRequest{File: r.p.File, IfHash: Hash(current), Edits: []EditRecord{{Old: "trains\n", Occurrence: 1, New: "shipped\n"}}})
			if err != nil {
				t.Fatal(err)
			}
			wantDoc := Restamp(strings.Replace(current, "trains\n", "shipped\n", 1), r.now)
			wantShadow := Restamp(strings.Replace(statusDoc, "trains\n", "shipped\n", 1), r.now)
			if r.read(r.p.File) != wantDoc || r.read(paths.Shadow) != wantShadow {
				t.Fatalf("document/shadow mismatch: doc %q shadow %q", r.read(r.p.File), r.read(paths.Shadow))
			}
			if result.Warning != "" {
				t.Fatal(result.Warning)
			}
			diff := r.diffText()
			if !strings.Contains(diff, "+operator unread") || strings.Contains(diff, "+shipped") {
				t.Fatalf("lost unread change or acknowledged own edit incorrectly: %q", diff)
			}
		})
	}
}

func TestApplyShadowBatchAndSameSection(t *testing.T) {
	for _, registered := range []bool{false, true} {
		for _, broad := range []bool{false, true} {
			t.Run(fmt.Sprintf("registered=%v/broad=%v", registered, broad), func(t *testing.T) {
				r := newRig(t)
				baseline := strings.Replace(statusDoc, "trains\n", "before\ntrains\nafter\n", 1)
				r.put(r.p.File, baseline)
				r.put(r.p.Shadow, baseline)
				paths := r.p
				if registered {
					if _, err := r.sync.Register(r.p.File, "ops:0"); err != nil {
						t.Fatal(err)
					}
					paths = registeredPaths(t, r.sync, r.p.File)
				}
				current := strings.Replace(baseline, "before\n", "operator unread\n", 1)
				r.put(r.p.File, current)
				candidate := strings.Replace(strings.Replace(current, "owner: ops:0", "owner: ops:0\ndiscussions: [new]", 1), "trains\n", "shipped\n", 1)
				edits := []EditRecord{{Old: "owner: ops:0", Occurrence: 1, New: "owner: ops:0\ndiscussions: [new]"}, {Old: "trains\n", Occurrence: 1, New: "shipped\n"}}
				if broad {
					edits = []EditRecord{{Old: current, Occurrence: 1, New: candidate}}
				}
				res, err := r.sync.Apply(ApplyRequest{File: r.p.File, IfHash: Hash(current), Edits: edits})
				if err != nil {
					t.Fatal(err)
				}
				wantShadow := Restamp(strings.Replace(strings.Replace(baseline, "owner: ops:0", "owner: ops:0\ndiscussions: [new]", 1), "trains\n", "shipped\n", 1), r.now)
				if r.read(r.p.File) != Restamp(candidate, r.now) || r.read(paths.Shadow) != wantShadow || res.Warning != "" {
					t.Fatalf("doc %q shadow %q warning %q", r.read(r.p.File), r.read(paths.Shadow), res.Warning)
				}
				if round := r.round(paths.Round); round == nil || len(round.Spans) != 2 {
					t.Fatalf("round %+v", round)
				}
			})
		}
	}
}

func TestApplyRetainsUnmappableBaseline(t *testing.T) {
	for _, registered := range []bool{false, true} {
		for _, missing := range []bool{false, true} {
			t.Run(fmt.Sprintf("registered=%v/missing=%v", registered, missing), func(t *testing.T) {
				r := newRig(t)
				if missing {
					if err := os.Remove(r.p.Shadow); err != nil {
						t.Fatal(err)
					}
				}
				paths := r.p
				if registered {
					if _, err := r.sync.Register(r.p.File, "ops:0"); err != nil {
						t.Fatal(err)
					}
					paths = registeredPaths(t, r.sync, r.p.File)
				}
				shadow := r.read(paths.Shadow)
				current := strings.Replace(statusDoc, "old", "operator unread", 1)
				r.put(r.p.File, current)
				res, err := r.sync.Apply(ApplyRequest{File: r.p.File, IfHash: Hash(current), Edits: []EditRecord{{Old: "operator unread", Occurrence: 1, New: "agent changed"}}})
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(res.Warning, "previous diff baseline") || !strings.Contains(res.Warning, "review the remaining edits") || r.read(paths.Shadow) != shadow || !strings.Contains(r.read(r.p.File), "agent changed") {
					t.Fatalf("warning %q shadow %q doc %q", res.Warning, r.read(paths.Shadow), r.read(r.p.File))
				}
				diff := r.diffText()
				if !strings.Contains(diff, "+agent changed") {
					t.Fatalf("conservative diff lost changes: %q", diff)
				}
				if _, err := r.sync.Read(r.p.File, "Needs you"); err != nil {
					t.Fatal(err)
				}
				if r.read(paths.Shadow) != shadow {
					t.Fatal("section read implicitly acknowledged changes")
				}
				if _, err := r.sync.Read(r.p.File, ""); err != nil {
					t.Fatal(err)
				}
				if strings.Contains(r.diffText(), "+agent changed") {
					t.Fatal("explicit whole-file read did not acknowledge reviewed content")
				}
			})
		}
	}
}

func TestApplyShadowNoopDoesNotAcknowledgeUnread(t *testing.T) {
	r := newRig(t)
	current := strings.Replace(statusDoc, "old", "operator unread", 1)
	r.put(r.p.File, current)
	res, err := r.sync.Apply(ApplyRequest{File: r.p.File, IfHash: Hash(current), Edits: []EditRecord{{Old: current, Occurrence: 1, New: current}}})
	if err != nil || res.Warning != "" {
		t.Fatalf("%+v %v", res, err)
	}
	if r.read(r.p.Shadow) != Restamp(statusDoc, r.now) || !strings.Contains(r.diffText(), "+operator unread") {
		t.Fatal("noop acknowledged unread content")
	}
}

func TestApplyShadowWarningsCompose(t *testing.T) {
	r := newRig(t)
	current := strings.Replace(statusDoc, "old", "operator unread", 1)
	r.put(r.p.File, current)
	r.sync.RoundWriter = func(string, []byte) error {
		r.put(r.p.Lock, "holder: operator\nat: 2026-10-10T23:50\n")
		return errors.New("injected round failure")
	}
	res, err := r.sync.Apply(ApplyRequest{File: r.p.File, IfHash: Hash(current), Edits: []EditRecord{{Old: "operator unread", Occurrence: 1, New: "agent changed"}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"previous diff baseline", "round could not be saved", "operator took the lock"} {
		if !strings.Contains(res.Warning, want) {
			t.Fatalf("missing %q: %q", want, res.Warning)
		}
	}
}

func TestApplyShadowPublicationFailureIsConservative(t *testing.T) {
	r := newRig(t)
	r.deliver("d1", "report")
	current := strings.Replace(statusDoc, "old", "operator unread", 1)
	r.put(r.p.File, current)
	f, err := load(r.p.File)
	if err != nil {
		t.Fatal(err)
	}
	oldShadow := r.read(r.p.Shadow)
	f.paths.Shadow = filepath.Join(t.TempDir(), "directory")
	if err := os.Mkdir(f.paths.Shadow, 0700); err != nil {
		t.Fatal(err)
	}
	_, err = r.sync.write(f, WriteRequest{File: r.p.File, IfHash: Hash(current), Fold: []string{"d1"}, edits: []EditRecord{{Old: "trains\n", Occurrence: 1, New: "shipped\n"}}})
	if err == nil {
		t.Fatal("shadow publication succeeded")
	}
	if r.read(r.p.Shadow) != oldShadow || !strings.Contains(r.read(r.p.File), "operator unread") || !strings.Contains(r.read(r.p.File), "shipped") || len(r.owedIDs()) != 1 {
		t.Fatal("publication failure lost baseline, edits or owed event")
	}
}

func TestApplyShadowAlias(t *testing.T) {
	for _, registered := range []bool{false, true} {
		t.Run(map[bool]string{false: "legacy", true: "registered"}[registered], func(t *testing.T) {
			r := newRig(t)
			paths := r.p
			if registered {
				if _, err := r.sync.Register(r.p.File, "ops:0"); err != nil {
					t.Fatal(err)
				}
				paths = registeredPaths(t, r.sync, r.p.File)
			}
			alias := filepath.Join(filepath.Dir(r.p.File), "alias.md")
			if err := os.Symlink(r.p.File, alias); err != nil {
				t.Fatal(err)
			}
			if !registered {
				// Legacy aliases have independent sidecars; registered aliases
				// resolve to their existing canonical metadata owner.
				paths = PathsOf(alias)
				r.put(paths.Shadow, statusDoc)
			}
			current := strings.Replace(statusDoc, "old", "operator unread", 1)
			r.put(r.p.File, current)
			_, err := r.sync.Apply(ApplyRequest{File: alias, IfHash: Hash(current), Edits: []EditRecord{{Old: "trains\n", Occurrence: 1, New: "shipped\n"}}})
			if err != nil {
				t.Fatal(err)
			}
			if r.read(paths.Shadow) != Restamp(strings.Replace(statusDoc, "trains\n", "shipped\n", 1), r.now) || !strings.Contains(r.read(r.p.File), "operator unread") {
				t.Fatal("alias lost canonical baseline")
			}
		})
	}
}
