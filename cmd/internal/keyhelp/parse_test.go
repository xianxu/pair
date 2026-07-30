package keyhelp

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestParseNvimKeymapsMultiLineCall(t *testing.T) {
	// The COMMON shape: desc sits on a later line than the lhs (init.lua:3629).
	src := `vim.keymap.set({ 'n', 'i' }, '<M-CR>', send_and_clear,
  { silent = true, desc = 'pair: send buffer + clear' })`
	got := ParseNvimKeymaps(src).Resolved
	if len(got) != 1 || got[0].Key != "<M-CR>" || got[0].Desc != "send buffer + clear" {
		t.Fatalf("got %+v", got)
	}
}

func TestParseNvimKeymapsSkipsNonPairDescs(t *testing.T) {
	src := `vim.keymap.set('n', 'gq', fmt, { desc = 'format' })`
	if got := ParseNvimKeymaps(src).Resolved; len(got) != 0 {
		t.Fatalf("non-pair desc must be ignored, got %+v", got)
	}
}

// An interpolated lhs must be REPORTED, not dropped: Alt+1..9 completion picks are
// real user-facing keys, and silently losing them is the #132 bug in miniature
// (init.lua:3913).
func TestParseNvimKeymapsReportsDynamicLhs(t *testing.T) {
	src := `for i = 1, 9 do
  vim.keymap.set('i', '<M-' .. i .. '>',
    function() end, { desc = 'pair: pick completion item ' .. i })
end`
	got := ParseNvimKeymaps(src).Dynamic
	if len(got) != 1 {
		t.Fatalf("interpolated lhs must land in Dynamic, got %+v", got)
	}
}

// The PQ-1 trap as a unit test: an unquoted lhs must land in Unresolved, and must
// NEVER produce a Resolved row whose Key is the desc string (init.lua:3872).
func TestParseNvimKeymapsUnquotedLhsIsUnresolvedNotMisassigned(t *testing.T) {
	src := `vim.keymap.set('i', open, function() return pair_insert_open(open) end,
    { silent = true, expr = true, desc = 'pair: autopair ' .. open })`
	scan := ParseNvimKeymaps(src)
	if len(scan.Unresolved) != 1 {
		t.Fatalf("unquoted lhs must be Unresolved, got %+v", scan)
	}
	for _, km := range scan.Resolved {
		if strings.HasPrefix(km.Key, "pair: ") {
			t.Fatalf("desc misassigned as Key: %q", km.Key)
		}
	}
}

func TestParseNvimKeymapsHandlesSingleModeAndTrailingSpaceDesc(t *testing.T) {
	src := `vim.keymap.set('n', '<M-BS>', del, { desc = 'pair: delete the current +N queue item' })`
	got := ParseNvimKeymaps(src).Resolved
	if len(got) != 1 || got[0].Key != "<M-BS>" {
		t.Fatalf("got %+v", got)
	}
}

// Property-shaped: whatever the whitespace, mode form or arg-2 form, the two shape
// invariants must hold. Pins the contract rather than three happy paths.
func TestParseNvimKeymapsInvariantsHoldAcrossForms(t *testing.T) {
	modes := []string{`'n'`, `'i'`, `{ 'n', 'i' }`, `{'n','i'}`}
	lhs := []string{`'<M-CR>'`, `'<M-1>'`, `open`, `tostring(i)`, `'<M-' .. i .. '>'`}
	for _, m := range modes {
		for _, l := range lhs {
			for _, sep := range []string{" ", "\n  ", "\n\t"} {
				src := "vim.keymap.set(" + m + ", " + l + "," + sep + "fn," + sep + "{ desc = 'pair: x' })"
				scan := ParseNvimKeymaps(src)
				total := len(scan.Resolved) + len(scan.Dynamic) + len(scan.Unresolved)
				if total != 1 {
					t.Errorf("src %q accounted %d, want 1", src, total)
				}
				for _, km := range scan.Resolved {
					if strings.HasPrefix(km.Key, "pair: ") {
						t.Errorf("src %q: Key %q is a desc", src, km.Key)
					}
					if !strings.HasPrefix(km.Key, "<") && !isBarePrintableKey(km.Key) {
						t.Errorf("src %q: Key %q is neither a <...> key nor a bare printable", src, km.Key)
					}
				}
			}
		}
	}
}

// Only `Run` binds are user-facing zellij-level actions. A WriteChars/Write bind is
// plumbing: it forwards an escape sequence to nvim, whose keymap desc already
// documents the behaviour — documenting both would double-list every key.
func TestParseZellijRunBindsIgnoresPassThroughBinds(t *testing.T) {
	src := `keybinds {
    shared_except "locked" {
        bind "Alt j" { Write 27; Write 106; }
        bind "Alt Up" { WriteChars "\u{1b}[1;3A"; }
        bind "Alt h" {
            Run "pair-help" {
                floating true
            }
        }
        bind "Alt l" { Run "pair" "changelog" "open"; }
    }
}`
	got := ParseZellijRunBinds(src)
	if len(got) != 2 {
		t.Fatalf("want 2 Run binds, got %+v", got)
	}
	if got[0].Key != "Alt h" || got[1].Key != "Alt l" {
		t.Fatalf("keys = %q, %q", got[0].Key, got[1].Key)
	}
}

// A `Write` bind immediately BEFORE a Run bind must not have the Run attributed to
// it — the config.kdl:157→163 shape a naive lookahead scanner gets wrong.
func TestParseZellijRunBindsAttributesRunToItsOwnBind(t *testing.T) {
	src := `bind "Alt Down" { WriteChars "\u{1b}[1;3B"; }

        bind "Alt h" {
            Run "pair-help" {
                floating true
            }
        }`
	got := ParseZellijRunBinds(src)
	if len(got) != 1 || got[0].Key != "Alt h" {
		t.Fatalf("Run attributed to the wrong bind: %+v", got)
	}
}

// --- reconciliation against the real file ----------------------------------

func mustReadTreeSource(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "..", filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// A count alone cannot catch a MISASSIGNMENT: at init.lua:3872 the second quoted
// string in the call is the desc, so a "second quoted arg is the lhs" parser
// produces Key == "pair: autopair " while the count still reconciles. This pins the
// three-way split and the shape invariants instead (#132 PQ-1).
func TestParseNvimKeymapsReconcilesAgainstRealFile(t *testing.T) {
	src := mustReadTreeSource(t, "nvim/init.lua")
	scan := ParseNvimKeymaps(src)
	raw := strings.Count(src, "desc = 'pair: ")

	// The parser keys off the exact literal `desc = 'pair: `. Counting with the SAME
	// literal would make the reconciliation blind in the same way the parser is: a
	// keymap written `desc = "pair: …"` or `desc='pair: …'` would be absent from both
	// sides, so it would never appear in `scan` and TestEveryNvimKeymapIsClassified
	// could not flag it. Counting with a tolerant regexp and requiring the two counts
	// to AGREE turns a silent blind spot into a loud failure.
	loose := len(regexp.MustCompile(`desc\s*=\s*["']pair: `).FindAllString(src, -1))
	if loose != raw {
		t.Errorf("found %d loosely-spelled `pair:` descs but %d matching the strict marker — a keymap uses a spelling the parser cannot see; widen descMarker or normalise init.lua", loose, raw)
	}

	if got := len(scan.Resolved) + len(scan.Dynamic) + len(scan.Unresolved); got != raw {
		t.Fatalf("accounted for %d keymaps, file has %d — the parser is dropping rows", got, raw)
	}
	for _, km := range scan.Resolved {
		if strings.HasPrefix(km.Key, "pair: ") {
			t.Errorf("Key %q is a desc string — arg-position parsing is broken", km.Key)
		}
	}
	// Exactly the known unquoted-lhs sites: init.lua:3872 (open), :3877 (close),
	// :3930 (tostring(i)). A NEW one must fail here rather than be absorbed as
	// "skipped by design".
	if len(scan.Unresolved) != 3 {
		var got []string
		for _, km := range scan.Unresolved {
			got = append(got, km.Raw+" → "+km.Desc)
		}
		t.Errorf("unresolved lhs count = %d, want 3 — a new unquoted-lhs keymap needs classifying: %v", len(scan.Unresolved), got)
	}
}

func TestParseZellijRunBindsAgainstRealFile(t *testing.T) {
	got := ParseZellijRunBinds(mustReadTreeSource(t, "zellij/config.kdl"))
	if len(got) != 2 {
		t.Fatalf("config.kdl should have exactly 2 Run binds (Alt h, Alt l), got %+v", got)
	}
}
