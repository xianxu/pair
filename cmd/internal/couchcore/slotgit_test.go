package couchcore

import (
	"context"
	"strings"
	"testing"

	"github.com/xianxu/pair/cmd/internal/terminal"
)

func TestSlotGitChangedFileCountEvidence(t *testing.T) {
	if (SlotGitStatus{}).ChangedFiles != nil {
		t.Fatal("zero-value status invented a known file count")
	}
	head := "# branch.head main\n"
	ordinary := "1 .M N... 100644 100644 100644 a b dir/file with spaces.go\n"
	rename := "2 R. N... 100644 100644 100644 a b R100 new.go\told.go\n"
	unmerged := "u UU N... 100644 100644 100644 100644 a b c conflict.go\n"
	untracked := "? new.txt\n"
	for _, test := range []struct {
		name string
		out  string
		want int
	}{
		{"clean", head, 0},
		{"ignored", head + "! ignored/\n", 0},
		{"ordinary", head + ordinary, 1},
		{"rename counts destination once", head + rename, 1},
		{"unmerged", head + unmerged, 1},
		{"untracked", head + untracked, 1},
		{"mixed", head + ordinary + rename + unmerged + untracked + "! ignored\n", 4},
	} {
		t.Run(test.name, func(t *testing.T) {
			status, err := ParseSlotGitStatus(test.out)
			if err != nil || status.ChangedFiles == nil || *status.ChangedFiles != test.want || status.Dirty != (test.want > 0) {
				t.Fatalf("status=%+v count=%v err=%v; want %d changed files", status, status.ChangedFiles, err, test.want)
			}
		})
	}
}

func TestSlotGitUncountableDirtyEntriesKeepCountUnknown(t *testing.T) {
	for _, entry := range []string{
		"1 .M",
		"2 R.",
		"u UU",
		"1 .M N... 100644 100644 100644 a b ",
		"2 R. N... 100644 100644 100644 a b R100 \told.go",
		"u UU N... 100644 100644 100644 100644 a b c ",
		"? untracked-directory/",
		"? \"untracked directory/\"",
	} {
		for _, prefix := range []string{"", "? counted.txt\n"} {
			status, err := ParseSlotGitStatus("# branch.head main\n" + prefix + entry)
			if err != nil || !status.Dirty || status.ChangedFiles != nil {
				t.Fatalf("uncountable dirty entry %q became known count: %+v %v", entry, status, err)
			}
		}
	}
	for _, out := range []string{"", "? x", "# branch.head main\nZ unknown", "# branch.head main\n# branch.ab +x -0"} {
		status, err := ParseSlotGitStatus(out)
		if err == nil || status.ChangedFiles != nil {
			t.Fatalf("parse failure invented count: %+v %v", status, err)
		}
	}
}

// TestSlotGlyphParts is a truth table over the two independent parts: branch
// position/divergence, then `*` for a dirty tree on any branch (pair#319).
func TestSlotGlyphParts(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		s    SlotGitStatus
		want string
	}{
		{"detached dirty", SlotGitStatus{Detached: true, Dirty: true}, slotGlyphBranch + "*"},
		{"detached clean", SlotGitStatus{Detached: true}, slotGlyphBranch},
		{"off resting dirty", SlotGitStatus{Branch: "000317-x", Dirty: true, HasUpstream: true, Ahead: 2}, slotGlyphBranch + "*"},
		{"off resting clean", SlotGitStatus{Branch: "000317-x"}, slotGlyphBranch},
		{"dirty and ahead", SlotGitStatus{Branch: "main-slot1", Dirty: true, HasUpstream: true, Ahead: 2}, "+*"},
		{"dirty and behind", SlotGitStatus{Branch: "main-slot1", Dirty: true, HasUpstream: true, Behind: 2}, "-*"},
		{"dirty only", SlotGitStatus{Branch: "main-slot1", Dirty: true, HasUpstream: true}, "*"},
		{"dirty without upstream", SlotGitStatus{Branch: "main-slot1", Dirty: true, Ahead: 1}, "*"},
		{"ahead", SlotGitStatus{Branch: "main-slot1", HasUpstream: true, Ahead: 1}, "+"},
		{"behind", SlotGitStatus{Branch: "main-slot1", HasUpstream: true, Behind: 4}, "-"},
		{"diverged", SlotGitStatus{Branch: "main-slot1", HasUpstream: true, Ahead: 3, Behind: 34}, "±"},
		{"dirty and diverged", SlotGitStatus{Branch: "main-slot1", Dirty: true, HasUpstream: true, Ahead: 3, Behind: 34}, "±*"},
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
	t.Parallel()

	if got := RestingBranch(0); got != "main" {
		t.Fatalf("RestingBranch(0) = %q", got)
	}
	if got := RestingBranch(3); got != "main-slot3" {
		t.Fatalf("RestingBranch(3) = %q", got)
	}
}

func TestParseSlotGitStatus(t *testing.T) {
	t.Parallel()

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
		// Count evidence has its own table above; retain the original complete
		// truth table for branch, divergence and cleanliness independently.
		got.ChangedFiles = nil
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
	t.Parallel()

	git := NewFakeGit(map[GitCall]string{
		{Dir: "/w/pair-slot2/pair", Args: "--no-optional-locks status --porcelain=v2 --branch"}: "# branch.head main-slot2\n? x\n",
	})
	got, err := ProbeSlotGit(context.Background(), git, "/w/pair-slot2/pair")
	if err != nil || got.ChangedFiles == nil || *got.ChangedFiles != 1 || len(git.Ops) != 1 {
		t.Fatalf("probe did not derive count from its single status observation: %+v %v calls=%v", got, err, git.Ops)
	}
	got.ChangedFiles = nil
	if err != nil || got != (SlotGitStatus{Branch: "main-slot2", Dirty: true}) {
		t.Fatalf("probe = %+v, %v", got, err)
	}
	if failed, err := ProbeSlotGit(context.Background(), git, "/w/pair-slot1/pair"); err == nil || failed.ChangedFiles != nil {
		t.Fatal("failed probe invented status/count for another checkout")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if failed, err := ProbeSlotGit(ctx, git, "/w/pair-slot2/pair"); err == nil || failed.ChangedFiles != nil {
		t.Fatal("cancelled probe invented status/count")
	}
}

func TestSlotGitChangedPaths(t *testing.T) {
	out := "# branch.oid abc\n# branch.head main-slot1\n" +
		"1 .M N... 100644 100644 100644 aaa aaa .gitattributes\n" +
		"1 M. N... 100644 100644 100644 aaa bbb dir/with space.go\n" +
		"2 R. N... 100644 100644 100644 aaa aaa R100 new.go\told.go\n" +
		"u UU N... 100644 100644 100644 100644 aaa bbb ccc both.go\n" +
		"? generated.txt\n" +
		"! ignored.log\n"
	got := strings.Join(SlotGitChangedPaths(out), "|")
	if want := ".gitattributes|dir/with space.go|new.go|both.go|generated.txt"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if status, err := ParseSlotGitStatus(out); err != nil || !status.Dirty {
		t.Fatalf("the parse must agree the same output is dirty: %+v %v", status, err)
	}
}

// Exercise every independent checkout observation, deriving the compact oracle
// from the observation rather than parsing the full glyph the same way twice.
func TestCompactSlotGlyphObservedStates(t *testing.T) {
	t.Parallel()
	for bits := 0; bits < 32; bits++ {
		for ahead := 0; ahead < 2; ahead++ {
			for behind := 0; behind < 2; behind++ {
				status := SlotGitStatus{Branch: "main-slot1", Dirty: bits&1 != 0, Detached: bits&2 != 0, HasUpstream: bits&4 != 0, Ahead: ahead, Behind: behind}
				if bits&8 != 0 {
					status.Branch = "000819-work"
				}
				if bits&16 != 0 {
					status.Branch = ""
				}
				want := "·"
				switch {
				case status.Dirty:
					want = "*"
				case status.Detached || status.Branch != "main-slot1":
					want = ""
				case status.HasUpstream && ahead > 0 && behind > 0:
					want = "±"
				case status.HasUpstream && ahead > 0:
					want = "+"
				case status.HasUpstream && behind > 0:
					want = "-"
				}
				full := SlotGlyph(status, "main-slot1")
				if got := CompactSlotGlyph(full); got != want {
					t.Fatalf("status %+v, full %q: compact %q, want %q", status, full, got, want)
				}
			}
		}
	}
}

func TestCompactSlotGlyphRejectsMalformedEvidence(t *testing.T) {
	t.Parallel()
	for _, malformed := range []string{"unknown", "working", "idle", "**", "*+", "+-", "±+", "±", "**", "foo*", "*foo", " +", "+ ", "\x1b[31m*", "*\x1b[0m", "\n*", "*\r", "\t", "\x00*", "\xff", "é", "🛋️", "+\u0301", "·*"} {
		if got := CompactSlotGlyph(malformed); got != "·" {
			t.Errorf("malformed %q: compact %q, want neutral dot", malformed, got)
		}
	}
}

func TestCompactSlotGlyphAlphabetOccupiesOneTerminalCell(t *testing.T) {
	t.Parallel()
	for _, input := range []string{"", "*", "±", "±*", "+", "+*", "-", "-*", "", "*", "unknown"} {
		got := CompactSlotGlyph(input)
		switch got {
		case "·", "*", "±", "+", "-", "":
		default:
			t.Fatalf("input %q: unexpected marker %q", input, got)
		}
		cells, err := terminal.StyledRows(got+"X", 3, 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(cells) != 3 || cells[0].Content != got || cells[0].Width != 1 || cells[1].Content != "X" {
			t.Fatalf("input %q: marker %q occupies unexpected terminal cells %+v", input, got, cells)
		}
	}
}
