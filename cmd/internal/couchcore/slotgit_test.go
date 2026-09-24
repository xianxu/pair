package couchcore

import (
	"context"
	"testing"
)

func TestSlotGlyphPrecedence(t *testing.T) {
	for _, tc := range []struct {
		name string
		s    SlotGitStatus
		want string
	}{
		{"detached", SlotGitStatus{Detached: true, Dirty: true}, slotGlyphBranch},
		{"off resting beats dirty", SlotGitStatus{Branch: "000317-x", Dirty: true, HasUpstream: true, Ahead: 2}, slotGlyphBranch},
		{"dirty beats ahead", SlotGitStatus{Branch: "main-slot1", Dirty: true, HasUpstream: true, Ahead: 2}, "*"},
		{"ahead", SlotGitStatus{Branch: "main-slot1", HasUpstream: true, Ahead: 1}, "+"},
		{"behind", SlotGitStatus{Branch: "main-slot1", HasUpstream: true, Behind: 4}, "-"},
		{"diverged", SlotGitStatus{Branch: "main-slot1", HasUpstream: true, Ahead: 3, Behind: 34}, "±"},
		{"dirty hides divergence", SlotGitStatus{Branch: "main-slot1", Dirty: true, HasUpstream: true, Ahead: 3, Behind: 34}, "*"},
		{"off resting hides divergence", SlotGitStatus{Branch: "000319-x", HasUpstream: true, Ahead: 3, Behind: 34}, slotGlyphBranch},
		{"no upstream is no evidence", SlotGitStatus{Branch: "main-slot1", Ahead: 3, Behind: 2}, ""},
		{"clean", SlotGitStatus{Branch: "main-slot1", HasUpstream: true}, ""},
	} {
		if got := SlotGlyph(tc.s, "main-slot1"); got != tc.want {
			t.Errorf("%s: SlotGlyph = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestRestingBranch(t *testing.T) {
	if got := RestingBranch(0); got != "main" {
		t.Fatalf("RestingBranch(0) = %q", got)
	}
	if got := RestingBranch(3); got != "main-slot3" {
		t.Fatalf("RestingBranch(3) = %q", got)
	}
}

func TestParseSlotGitStatus(t *testing.T) {
	head := "# branch.oid ad5c735d925a3095c5632d9a99603d33e490e19a\n# branch.head main-slot1\n"
	upstream := "# branch.upstream origin/main\n"
	for _, tc := range []struct {
		name string
		out  string
		want SlotGitStatus
	}{
		{"clean with upstream", head + upstream + "# branch.ab +0 -0", SlotGitStatus{Branch: "main-slot1", HasUpstream: true}},
		{"behind", head + upstream + "# branch.ab +0 -4", SlotGitStatus{Branch: "main-slot1", HasUpstream: true, Behind: 4}},
		{"ahead", head + upstream + "# branch.ab +3 -0", SlotGitStatus{Branch: "main-slot1", HasUpstream: true, Ahead: 3}},
		{"diverged", head + upstream + "# branch.ab +3 -34", SlotGitStatus{Branch: "main-slot1", HasUpstream: true, Ahead: 3, Behind: 34}},
		{"no upstream", head, SlotGitStatus{Branch: "main-slot1"}},
		{"detached", "# branch.oid ad5c735d925a3095c5632d9a99603d33e490e19a\n# branch.head (detached)", SlotGitStatus{Detached: true}},
		{"ordinary change", head + "1 .M N... 100644 100644 100644 a b f.go", SlotGitStatus{Branch: "main-slot1", Dirty: true}},
		{"rename", head + "2 R. N... 100644 100644 100644 a b R100 new\told", SlotGitStatus{Branch: "main-slot1", Dirty: true}},
		{"unmerged", head + "u UU N... 100644 100644 100644 100644 a b c f.go", SlotGitStatus{Branch: "main-slot1", Dirty: true}},
		{"untracked", head + "? notes.md", SlotGitStatus{Branch: "main-slot1", Dirty: true}},
		{"ignored is not dirty", head + "! build/", SlotGitStatus{Branch: "main-slot1"}},
		{"unknown header ignored", head + "# branch.future x", SlotGitStatus{Branch: "main-slot1"}},
	} {
		got, err := ParseSlotGitStatus(tc.out)
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s: got %+v, want %+v", tc.name, got, tc.want)
		}
	}
	for _, bad := range []string{
		"",
		upstream + "# branch.ab +0 -0",
		head + "# branch.ab +x -0",
		head + "# branch.ab +1",
		head + "# branch.ab -1 -0",
		head + "# branch.ab +1 -0 extra",
		head + "# branch.ab +1 --1",
		head + "# branch.ab ++1 -0",
		head + "# branch.ab +1 -0\n# branch.ab +2 -0",
		"# branch.head ",
		"# branch.head",
		head + "# branch.head another",
		head + "# branch.upstream ",
		head + upstream + upstream,
		head + "Z what",
		head + "1",
	} {
		if _, err := ParseSlotGitStatus(bad); err == nil {
			t.Errorf("accepted malformed output %q", bad)
		}
	}
}

func TestProbeSlotGitRunsInDirWithNoOptionalLocks(t *testing.T) {
	git := NewFakeGit(map[GitCall]string{
		{Dir: "/w/pair-slot2/pair", Args: "--no-optional-locks status --porcelain=v2 --branch"}: "# branch.head main-slot2\n? x\n",
	})
	got, err := ProbeSlotGit(context.Background(), git, "/w/pair-slot2/pair")
	if err != nil || got != (SlotGitStatus{Branch: "main-slot2", Dirty: true}) {
		t.Fatalf("probe = %+v, %v", got, err)
	}
	if _, err := ProbeSlotGit(context.Background(), git, "/w/pair-slot1/pair"); err == nil {
		t.Fatal("probe in another dir answered from the wrong checkout")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ProbeSlotGit(ctx, git, "/w/pair-slot2/pair"); err == nil {
		t.Fatal("cancelled probe succeeded")
	}
}
