# `pair keys` — real keybind help behind Alt+h

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `Alt+h` list the actual in-session keybindings, sourced so it cannot silently rot the way it did after #99 M5c.

**Architecture:** A new pure `cmd/internal/keyhelp` package renders the help text. Binding *wording* is derived from the sources that already own it — `desc = 'pair: …'` on every `vim.keymap.set` in `nvim/init.lua`, read from the embedded runtime bundle at runtime, and from the working tree in tests (the assets dir is gitignored and `go test` never regenerates it) — while *inclusion, grouping and order* come from a small curated table, because full auto-inclusion would publish editor-internals like `autopair` and `jump over` as user help. Bidirectional drift tests make silent rot impossible: an undocumented new key fails the build, and a documented key that no longer exists fails too. `pair keys` prints it; `bin/pair-help` (Alt+h) pages `pair keys` instead of `pair -h`.

**Tech Stack:** Go (stdlib only), Lua/KDL as parsed *inputs* (never re-authored), existing `runtimebundle.EmbeddedAsset` seam.

---

## Scope note: four sources, not one

The issue's Spec assumed `cmd/internal/workbenchshortcut` "holds the chord registry" and could be the single source. It can't, on its own — verified during design:

| Source | Owns | Count | Carries help text? |
|---|---|---|---|
| `nvim/init.lua` `vim.keymap.set` `desc` | draft/agent keys (`<M-CR>`, `<M-q>`, `<C-c>`, `<M-BS>`, `<M-Left>`…) | 34 calls, ~30 distinct descs | **yes** — already good prose |
| `workbenchshortcut.globalBindings` | 8 global chords (`Alt+d/x/n/c`, `Ctrl+Alt+n`, `Alt+Shift+N`, `Alt+↑/↓`) | 8 | **no** — no label field exists |
| `workbenchshortcut.Decide` role switch → new `roleBindings` table | terminal-pane-local chords (`Alt+t/w/r`, `Alt+Shift+D`, `Alt+Shift+⏎`) | ~6 | **no** — added by Task 4b |
| `zellij/config.kdl` `Run` binds | zellij-level actions nvim never sees — `Alt+h`, `Alt+l` | 2 `Run`, of **20** `bind` entries (plus 38 `unbind`); the other 18 are `WriteChars` or `Write <n>;` plumbing | **no** |

The 8 global chords are applied to nvim via the *generated* `nvim/workbench_actions.lua`, not literal `vim.keymap.set` calls — so parsing `init.lua` alone silently misses them. That asymmetry is the trap this plan is built around.

**A key's meaning is role-dependent, so the join key is (key, context) — never key alone.** Two verified cases forced this:

- `<M-t>/<M-w>/<M-r>` have nvim keymaps whose desc is `right-terminal tab helper disabled in draft` (`init.lua:3653-3658`) — they are deliberate **no-ops in the draft**. Their real behaviour (new/close/rename tab) lives in `Decide`'s terminal branch (`shortcut.go:153-158`). Deriving their wording from the nvim desc would ship "disabled in draft" as the user-facing description of a working feature.
- `<S-M-CR>` means *append to agent, no send* in the draft (`init.lua:3632`) and *toggle focused layout* in the terminal (`shortcut.go:176`). One row per key cannot express that.

So each catalog row **names its wording source explicitly** (`SourceNvim` / `SourceGlobal` / `SourceRole` / `SourceZellij`). There is no "whichever source has prose wins" fallback — that rule is what produced the misleading-wording bug above. A dual-meaning key gets **two rows**, in different sections, each naming its own source.

## Non-goals (explicit)

- **`nvim/init.lua:2137` `PAIR_CHEATS`** is a hand-maintained key+label list and today's *display-spelling* authority (`Alt+⏎`, `Alt+x`, `Alt+d`). This plan **reuses its spellings** for the keys it covers (Task 6's display mapping) so the statusline and the help agree, but **unifying the statusline and the help catalog into one table is a non-goal** — the statusline needs priority-ordered drop-out behaviour the help does not, and merging them would drag statusline layout into this change.
- **`workbenchshortcut.ChordName` (`shortcut.go:302`) must NOT be used for display.** It looks display-shaped (`"Alt+j"`), but it is a **routing** name round-tripped by `ChordFromName` (`wrapcmd/wrap.go:1502`) — a wire format. Coupling help display to it would mean a cosmetic rewording silently breaks chord routing. Stated as a non-goal so a future reader does not "DRY" these together.
- The Homebrew formula caveat lives in the separate `homebrew-pair` repo — #131's fix, not this branch's.
- Migrating `Decide`'s role switch to be fully table-driven: Task 4b adds a parallel `roleBindings` table for *wording and enumerability* only. Rewriting the dispatch logic to consume it is out of scope.

**Deliberate Done-when revision (record in the issue's `## Revisions`).** The issue's Done-when says the list must derive "with no second edit." Taken literally that means auto-including every keymap, which publishes `autopair`, `jump over `, `quit blocked`, `cycle completion or insert tab` and `smart-delete empty pair` as workbench help — worse than the bug. This plan keeps wording derived (no second edit to reword anything) but makes *inclusion* explicit, and enforces it with a test so the property that actually matters — **cannot rot unnoticed** — holds. Classifying a new key is a one-line edit the test demands; it is not optional and cannot be forgotten.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Binding` | `cmd/internal/keyhelp/keyhelp.go` | new |
| `Section` | `cmd/internal/keyhelp/keyhelp.go` | new |
| `KeymapScan` | `cmd/internal/keyhelp/parse.go` | new |
| `ParseNvimKeymaps` | `cmd/internal/keyhelp/parse.go` | new |
| `ParseZellijRunBinds` | `cmd/internal/keyhelp/parse.go` | new |
| `Catalog` | `cmd/internal/keyhelp/catalog.go` | new |
| `Render` | `cmd/internal/keyhelp/render.go` | new |
| `GlobalBinding` | `cmd/internal/workbenchshortcut/shortcut.go` | modified |
| `roleBindings` | `cmd/internal/workbenchshortcut/shortcut.go` | new |
| `UsageText` | `cmd/internal/launcher/format.go`¹ | modified |

¹ `UsageText` lives wherever it is today (`grep -rn "func UsageText"`); modify in place, do not relocate.

- **Binding** — one help row: `Key` (display form, e.g. `Alt+⏎`), `Desc` (the derived sentence), `Context` (`ContextDraft`/`ContextTerminal`/`ContextGlobal`), `Group`, `Order`.
  - **Relationships:** N:1 with `Section`. 1:1 with exactly one upstream source row, joined on **(key, context)** — see the role-dependence note above; a key with two meanings produces two Bindings.
  - **DRY rationale:** The single shape every source normalizes into, so `Render` has one input type instead of three. Without it, rendering would branch per source and each branch would drift.
  - **Future extensions:** A `Since` field if the changelog ever wants "new in" markers; a `Modes` field if draft-vs-terminal context needs showing per row.

- **Section** — a titled, ordered group of `Binding`s (`Draft`, `Panes & layout`, `Session`, `Terminal tabs`).
  - **Relationships:** 1:N with `Binding`; the ordered slice of Sections *is* the document.
  - **DRY rationale:** Grouping and ordering live here only — not duplicated between renderer and catalog.

- **ParseNvimKeymaps(luaSrc string) KeymapScan** — pure extraction from `vim.keymap.set` calls whose desc starts `pair: `. Returns a `KeymapScan{Resolved, Dynamic, Unresolved []NvimKeymap}` — a **three-way** classification, not a flat slice, because the lhs argument comes in three syntactic forms and conflating them is how a parser lies about its own coverage.

  **Input classes (all verified in the real file — do not trust this list without re-checking):**

  | Form | Example | Site | Handling |
  |---|---|---|---|
  | quoted literal | `'<M-CR>'` | `:3629` | `Resolved` |
  | interpolated literal | `'<M-' .. i .. '>'` | `:3913` | `Dynamic` (expanded to `<M-1>`…`<M-9>` by the catalog, or documented as a range) |
  | **unquoted expression** | `open`, `close`, `tostring(i)` | `:3872`, `:3877`, `:3930` | `Unresolved` — **never** guessed |

  The naive rule "the second quoted argument is the lhs" is **wrong and dangerously silent**: at `:3872` the second quoted string in the call is the *desc*, so that rule yields `Key: "pair: autopair "` — a garbage row that looks plausible — while a `count == grep -c "desc = 'pair: "` assertion still passes, because the count is right and only the *assignment* is wrong. The parser must therefore key off **argument position** (arg 2 of the call), and emit `Unresolved` whenever that argument is not a quoted literal.

  **Invariants (assert all three; they are what make the drift tests trustworthy):**
  1. No `Resolved.Key` begins with `pair: ` — a direct trap for the misparse above.
  2. Every `Resolved.Key` is a quoted literal starting `<` or a bare printable key (`z=`, `ZZ`); anything else belongs in `Dynamic`/`Unresolved`.
  3. **Reconciliation:** `len(Resolved)+len(Dynamic)+len(Unresolved)` equals the raw `desc = 'pair: ` occurrence count, AND `Unresolved` matches a named allowlist of exactly the three known sites. A *new* unquoted-lhs keymap therefore fails the build instead of vanishing — separating "skipped by design" from "misparsed", which a bare count cannot do.

  **Property strategy:** a table-driven property over generated call text — vary whitespace, mode forms (`'n'` vs `{ 'n', 'i' }`), desc quoting, and arg-2 form — asserting invariants 1 and 2 hold for every generated input. Cheap, and it pins the contract rather than three happy-path examples.

  Must also tolerate multi-line calls (desc on a later line than the lhs — the common case, `:3629-3630`).
  - **DRY rationale:** The wording is authored once, in the editor config that already needed it. Zero duplication of help prose.
  - **Future extensions:** If a `desc` convention ever adds a group hint (`pair[draft]: …`), the group could migrate here and shrink `Catalog`.

- **ParseZellijRunBinds(kdlSrc string) []ZellijBind** — pure extraction of `bind "<key>" { Run … }` from `config.kdl`, ignoring `WriteChars` binds (plumbing whose user-facing meaning belongs to the nvim keymap it feeds).
  - **DRY rationale:** Distinguishes zellij-level actions from pass-throughs in one place, so the catalog needn't hardcode which is which.

- **Catalog** — the curated inclusion/order/group table plus an `internalOnly` set. Maps a source key to its `Group`+`Order`; anything present in a source and in neither list is a **failure**, not a silent omission.
  - **Relationships:** references source rows by key; owns no prose.
  - **DRY rationale:** The one place inclusion is decided. It holds no wording, so it cannot drift from the sources' text.
  - **Future extensions:** Per-group notes/footers; a `Hidden` reason string alongside `internalOnly` entries.

- **Render(sections []Section) string** — deterministic plain-text rendering, column-aligned on the widest key in each section. No IO, no width probing (`bin/pair-help` already centers and pages).
  - **DRY rationale:** One formatter for `pair keys`, `Alt+h`, and any future surface.

- **GlobalBinding (modified)** — gains `Help string`. One of exactly two new places wording is authored, because the 8 global chords have no existing description anywhere.

- **roleBindings (new, `cmd/internal/workbenchshortcut/shortcut.go`)** — a table of the terminal-pane-local chords (`Alt+t/w/r`, `Alt+Shift+D`, `Alt+Shift+⏎`, `Alt+k`) carrying `Chord`, `PaneRole` and `Help`. The second and last place wording is authored.
  - **Why it must exist:** these chords' behaviour lives in a `switch` (`shortcut.go:150-180`), which is neither enumerable nor documentable, and their *nvim* descs describe the draft **no-op** (`right-terminal tab helper disabled in draft`) — so there is no honest existing source for their user-facing wording. Authoring it beside the switch keeps the prose next to the behaviour it describes.
  - **Scope guard:** the table is for wording + enumerability only; `Decide` keeps its switch (see Non-goals). A test asserts every chord the switch handles for `PaneRoleRightTerminal` appears in the table, so the two cannot diverge.

- **UsageText (modified)** — drops the circular final sentence, points at `pair keys`.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `SourceReader` | `cmd/internal/keyhelp/sources.go` | new | `runtimebundle.EmbeddedAsset` |
| `keys` command | `cmd/internal/keyscmd/keyscmd.go` | new | stdout |
| `bin/pair-help` | `bin/pair-help` | modified | `less` pager |

- **SourceReader** — fetches `nvim/init.lua` and `zellij/config.kdl` as bytes. Default impl delegates to `runtimebundle.EmbeddedAsset` (already embeds both — see `runtimebundle/embed_test.go`), so an installed `pair` needs no files on disk.
  - **Injected into:** `keyscmd`, which passes the bytes to the pure parsers. All parsing stays testable on string literals with no IO.
  - **Future extensions:** a repo-dir impl for a `--from-tree` debugging flag.
  - **No new fake needed** (`ARCH-MOCK`): this wraps an in-binary embed, not an external binary or service. Tests feed the parsers strings directly.
  - **CRITICAL — which copy each test reads.** `cmd/internal/runtimebundle/assets/` is **gitignored** (`.gitignore:34`) and regenerated only by `runtimebundle-generate`, which `go test` does **not** trigger. So a drift test reading the *embedded* copy would validate a stale snapshot: add a keymap, run `go test ./cmd/internal/keyhelp/`, get green, ship undocumented. The split is therefore mandatory:
    - **Classification/drift tests read the TREE copies** (`../../../nvim/init.lua`, `../../../zellij/config.kdl`) — always current, no build step required.
    - **One separate test asserts embedded == tree** for both assets, so a stale bundle fails loudly and by name instead of silently weakening every other test.
    - **The render test reads the EMBEDDED copy**, preserving shipped-binary fidelity — that is the only duty the embedded read keeps.

- **keys command** — `pair keys` writes the rendered help to stdout, exit 0. Registered in the dispatcher table (`cmd/internal/dispatcher/dispatcher.go:50-66`) as `{Name: "keys", Summary: "in-session keybindings", Status: "implemented"}`.

- **bin/pair-help** — one-line change: `help="$(pair -h)"` → `help="$(pair keys)"`. Centering, `less`, and the `q`/`Esc` lesskey bindings are already correct and stay untouched.

---

## Chunk 1: pure core

### Task 1: `Binding`, `Section`, `Render`

**Files:**
- Create: `cmd/internal/keyhelp/keyhelp.go`, `cmd/internal/keyhelp/render.go`
- Test: `cmd/internal/keyhelp/render_test.go`

- [x] **Step 1: Write the failing test**

```go
package keyhelp

import "testing"

func TestRenderAlignsWithinSection(t *testing.T) {
	got := Render([]Section{{
		Title: "Draft",
		Bindings: []Binding{
			{Key: "Alt+⏎", Desc: "send buffer + clear"},
			{Key: "Shift+Alt+⏎", Desc: "append, no send"},
		},
	}})
	want := "Draft\n" +
		"  Alt+⏎        send buffer + clear\n" +
		"  Shift+Alt+⏎  append, no send\n"
	if got != want {
		t.Fatalf("Render =\n%q\nwant\n%q", got, want)
	}
}

// Alignment is per-section, so a long key in one section must not indent another.
func TestRenderAlignsPerSectionNotGlobally(t *testing.T) {
	got := Render([]Section{
		{Title: "A", Bindings: []Binding{{Key: "Alt+h", Desc: "help"}}},
		{Title: "B", Bindings: []Binding{{Key: "Ctrl+Alt+n", Desc: "reload"}}},
	})
	want := "A\n  Alt+h  help\n\nB\n  Ctrl+Alt+n  reload\n"
	if got != want {
		t.Fatalf("Render =\n%q\nwant\n%q", got, want)
	}
}

// Width is measured in DISPLAY columns, not bytes: "Alt+⏎" is 5 runes / 8 bytes,
// and byte-length padding would visibly misalign every CJK-wide or multibyte key.
func TestRenderPadsByDisplayWidthNotBytes(t *testing.T) {
	got := Render([]Section{{Title: "T", Bindings: []Binding{
		{Key: "Alt+⏎", Desc: "a"},
		{Key: "Alt+x", Desc: "b"},
	}}})
	want := "T\n  Alt+⏎  a\n  Alt+x  b\n"
	if got != want {
		t.Fatalf("Render =\n%q\nwant\n%q", got, want)
	}
}
```

- [x] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/internal/keyhelp/ -run TestRender -v`
Expected: FAIL — `undefined: Render`, `undefined: Section`, `undefined: Binding`.

- [x] **Step 3: Write minimal implementation**

`keyhelp.go`:

```go
// Package keyhelp renders Pair's in-session keybinding help (#132).
//
// Wording is NOT authored here. It is derived from the sources that already own
// it — `desc = 'pair: …'` on each vim.keymap.set, and workbenchshortcut's
// GlobalBinding.Help — so the help cannot drift from the bindings the way it did
// when #99 M5c dropped bin/pair-shell's KEYBINDINGS section and left Alt+h
// pointing at a CLI usage block that said "keybindings are on Alt+h".
package keyhelp

// Binding is one rendered help row.
type Binding struct {
	Key   string // display form, e.g. "Alt+⏎"
	Desc  string // derived from the source, never authored here
	Group string
	Order int
}

// Section is a titled, ordered group of bindings.
type Section struct {
	Title    string
	Bindings []Binding
}
```

`render.go`:

```go
package keyhelp

import (
	"strings"

	"github.com/mattn/go-runewidth" // ONLY if already in go.mod; see Step 3a
)

func Render(sections []Section) string {
	var b strings.Builder
	for i, s := range sections {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(s.Title)
		b.WriteString("\n")
		w := 0
		for _, bind := range s.Bindings {
			if n := displayWidth(bind.Key); n > w {
				w = n
			}
		}
		for _, bind := range s.Bindings {
			b.WriteString("  ")
			b.WriteString(bind.Key)
			b.WriteString(strings.Repeat(" ", w-displayWidth(bind.Key)+2))
			b.WriteString(bind.Desc)
			b.WriteString("\n")
		}
	}
	return b.String()
}
```

- [x] **Step 3a: Resolve the width dependency BEFORE writing `displayWidth`**

Run: `grep -n "runewidth\|go-runewidth" go.mod; grep -rn "func.*[Dd]isplayWidth\|runewidth\." --include='*.go' . | grep -v _test | head`

`launcher/list.go:88` already solves exactly this for `pair list` ("pads by display width, not runes — 📁 is one rune and two columns", #130). **Reuse that helper** rather than adding a dependency or a second implementation (`ARCH-DRY`). If it is unexported, either export it or move it to a shared home and update `list.go` to call it — do not copy it. Only if no helper exists and `go-runewidth` is absent from `go.mod`, write a minimal local `displayWidth` counting East-Asian-wide runes as 2, and note the choice in the issue `## Log`.

- [x] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/internal/keyhelp/ -run TestRender -v`
Expected: PASS (3 tests).

- [x] **Step 5: Commit**

```bash
git add cmd/internal/keyhelp/
git commit -m "#132: keyhelp render — sections, per-section display-width alignment"
```

### Task 2: `ParseNvimKeymaps`

**Files:**
- Create: `cmd/internal/keyhelp/parse.go`
- Test: `cmd/internal/keyhelp/parse_test.go`

- [x] **Step 1: Write the failing test**

Cases are taken from real `init.lua` shapes — check each against the file before trusting this plan's rendition of it.

```go
package keyhelp

import "testing"

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

// An interpolated lhs must be REPORTED, not dropped: Alt+1..9 completion picks
// are real user-facing keys, and silently losing them is the #132 bug in
// miniature (init.lua:3913).
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
```

- [x] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/internal/keyhelp/ -run TestParseNvim -v`
Expected: FAIL — `undefined: ParseNvimKeymaps`.

- [x] **Step 3: Write the implementation**

Shape: scan for `vim.keymap.set(`, then from that index take the text up to the matching close of the call (or a bounded lookahead of ~6 lines — whichever the real file needs), extract the **second** quoted argument as the lhs and the `desc = '…'` payload. A `..` inside the lhs quotes means `Dynamic: true` and the literal text is kept for reporting. Return `[]NvimKeymap{Key, Desc, Dynamic}` with `pair: ` stripped and the remainder `strings.TrimSpace`d (several real descs have trailing spaces, e.g. `'pair: autopair '`).

Keep it a pure string function — no `os`, no regexp compiled at call time (hoist any `regexp.MustCompile` to a package var).

- [x] **Step 4: Run test to verify it passes, then run it against the REAL file**

Run: `go test ./cmd/internal/keyhelp/ -run TestParseNvim -v`
Expected: PASS (4 tests).

- [x] **Step 4a: Reconcile against reality — a bare count is NOT sufficient**

Add a permanent test (not temporary — this is the parser's contract) reading the **tree** copy `../../../nvim/init.lua`:

```go
// A count alone cannot catch a MISASSIGNMENT: at init.lua:3872 the second quoted
// string in the call is the desc, so a "second quoted arg is the lhs" parser
// produces Key == "pair: autopair " and the count still reconciles. This test
// pins the three-way split and the shape invariants instead (#132 PQ-1).
func TestParseNvimKeymapsReconcilesAgainstRealFile(t *testing.T) {
	src := mustReadTreeSource(t, "nvim/init.lua")
	scan := ParseNvimKeymaps(src)
	raw := strings.Count(src, "desc = 'pair: ")

	if got := len(scan.Resolved) + len(scan.Dynamic) + len(scan.Unresolved); got != raw {
		t.Fatalf("accounted for %d keymaps, file has %d — the parser is dropping rows", got, raw)
	}
	for _, km := range scan.Resolved {
		if strings.HasPrefix(km.Key, "pair: ") {
			t.Errorf("Key %q is a desc string — arg-position parsing is broken", km.Key)
		}
	}
	// Exactly the three known unquoted-lhs sites. A NEW one must fail here rather
	// than be silently absorbed as "skipped by design".
	wantUnresolved := 3 // init.lua:3872 (open), :3877 (close), :3930 (tostring(i))
	if len(scan.Unresolved) != wantUnresolved {
		t.Errorf("unresolved lhs count = %d, want %d — a new unquoted-lhs keymap needs classifying", len(scan.Unresolved), wantUnresolved)
	}
}
```

Run: `go test ./cmd/internal/keyhelp/ -run Reconciles -v`
Expected: PASS. Re-derive `raw` and `wantUnresolved` from the file if they disagree — but investigate *why* before changing a number.

- [x] **Step 5: Commit**

```bash
git add cmd/internal/keyhelp/parse.go cmd/internal/keyhelp/parse_test.go
git commit -m "#132: parse nvim keymap descs — the wording source, dynamic lhs reported not dropped"
```

### Task 3: `ParseZellijRunBinds`

**Files:**
- Modify: `cmd/internal/keyhelp/parse.go`
- Test: `cmd/internal/keyhelp/parse_test.go`

- [x] **Step 1: Write the failing test**

```go
// Only `Run` binds are user-facing zellij-level actions. A WriteChars bind is
// plumbing: it forwards an escape sequence to nvim, whose keymap desc already
// documents the behaviour — documenting both would double-list every key.
func TestParseZellijRunBindsIgnoresWriteChars(t *testing.T) {
	src := `keybinds {
    shared_except "locked" {
        bind "Alt h" {
            Run "pair-help" {
                floating true
            }
        }
        bind "Alt N" { WriteChars "\u{1b}[78;4u"; }
    }
}`
	got := ParseZellijRunBinds(src)
	if len(got) != 1 || got[0].Key != "Alt h" {
		t.Fatalf("got %+v", got)
	}
}
```

- [x] **Step 2–4:** Run (expect FAIL: undefined), implement, re-run (expect PASS). Then verify against the real file: `ParseZellijRunBinds` over `zellij/config.kdl` must return exactly **2** binds (`Alt h`, `Alt l`) — cross-check with `grep -B1 -E "^\s+Run " zellij/config.kdl`.

- [x] **Step 5: Commit**

```bash
git commit -am "#132: parse zellij Run binds (zellij-level actions, not WriteChars plumbing)"
```

---

## Chunk 2: sources, catalog, and the anti-rot tests

### Task 4: `GlobalBinding.Help` — wording for the 8 chords that have none

**Files:**
- Modify: `cmd/internal/workbenchshortcut/shortcut.go:103-120`
- Test: `cmd/internal/workbenchshortcut/shortcut_test.go`

- [x] **Step 1: Write the failing test**

```go
// Every global chord must carry help text. This is the "no second edit" property
// for the 8 chords whose wording lives nowhere else: adding a row to
// globalBindings without a Help string fails here, so it cannot reach a release
// undocumented (#132).
func TestEveryGlobalBindingHasHelp(t *testing.T) {
	for _, b := range GlobalBindings() {
		if strings.TrimSpace(b.Help) == "" {
			t.Errorf("chord %v (%s) has no Help text", b.Chord, b.NvimKey)
		}
	}
}
```

- [x] **Step 2: Run to verify it fails**

Run: `go test ./cmd/internal/workbenchshortcut/ -run TestEveryGlobalBindingHasHelp -v`
Expected: FAIL — 8 errors (no `Help` field yet; this will be a compile error first, which counts as red).

- [x] **Step 3: Add the field and fill it**

Add `Help string` as the last field of `GlobalBinding`, then fill each row. Wording must match observed behaviour — read each `LuaFunction` in `nvim/init.lua` before writing its sentence rather than guessing from the chord name:

```go
var globalBindings = []GlobalBinding{
	{ChordAltD, ActionConfirmDetach, "PairConfirmDetach", "<M-d>", true, "detach from the session (re-attach with `pair`)"},
	{ChordAltX, ActionConfirmQuit, "PairConfirmQuit", "<M-x>", true, "full quit — kill the session and drop it from the resurrect list"},
	{ChordAltN, ActionRestartPair, "PairConfirmRestart", "<M-n>", true, "reload pair — kill and re-launch the workbench in place"},
	{ChordCtrlAltN, ActionRestartPair, "PairConfirmRestart", "<C-M-n>", true, "reload pair (same as Alt+n)"},
	{ChordAltShiftN, ActionRestartAgent, "PairConfirmAgentRestart", "<M-N>", true, "restart only the agent conversation, keeping the workbench"},
	{ChordAltUp, ActionGrowDraft, "PairLayoutBigger", "<M-Up>", false, "grow the draft pane along the height ladder"},
	{ChordAltDown, ActionShrinkDraft, "PairLayoutSmaller", "<M-Down>", false, "shrink the draft pane along the height ladder"},
	{ChordAltC, ActionToggleReview, "PairReviewToggle", "<M-c>", false, "open/show/hide the review pane"},
}
```

Note: adding a positional field breaks any other unkeyed composite literal of `GlobalBinding`. Run `go build ./...` and fix; consider converting these to keyed literals if the compiler complains anywhere else.

- [x] **Step 4: Run to verify it passes**

Run: `go test ./cmd/internal/workbenchshortcut/ -v`
Expected: PASS, including the pre-existing generator tests (`RenderLuaGlobalMaps` must be unchanged — the new field is not rendered into Lua).

- [x] **Step 5: Verify the generated Lua did NOT change**

Run: `go run ./cmd/internal/workbenchshortcut/generatecmd 2>/dev/null || true; git diff --stat nvim/workbench_actions.lua`
Expected: no diff. If there is one, `Help` leaked into the generator — revert that.

- [x] **Step 6: Commit**

```bash
git commit -am "#132: GlobalBinding carries Help; test refuses an undocumented chord"
```

### Task 4b: `roleBindings` — an honest wording home for the pane-local chords

**Files:**
- Modify: `cmd/internal/workbenchshortcut/shortcut.go` (add table near `globalBindings:111`)
- Test: `cmd/internal/workbenchshortcut/shortcut_test.go`

These chords have **no honest existing source**: their behaviour is in `Decide`'s
`switch` (not enumerable) and their nvim descs describe the deliberate draft
**no-op** (`right-terminal tab helper disabled in draft`). Deriving from nvim here
would ship "disabled in draft" as the description of a working feature — the exact
class of wrong-but-plausible help this issue is fixing.

- [x] **Step 1: Write the failing test**

```go
// The table must cover every chord Decide actually handles for the right
// terminal, or the help silently omits a working key (#132 PQ-2).
func TestRoleBindingsCoverTerminalSwitch(t *testing.T) {
	for _, chord := range []Chord{ChordAltT, ChordAltW, ChordAltR, ChordAltShiftD, ChordAltShiftEnter, ChordAltK} {
		in := ShortcutInput{Role: PaneRoleRightTerminal, Chord: chord}
		if Decide(in).Disposition == DispositionPassthrough {
			continue // not handled for this role — nothing to document
		}
		found := false
		for _, rb := range RoleBindings() {
			if rb.Chord == chord {
				found = true
				if strings.TrimSpace(rb.Help) == "" {
					t.Errorf("chord %v is handled for the right terminal but has no Help", chord)
				}
			}
		}
		if !found {
			t.Errorf("chord %v is handled for the right terminal but is absent from roleBindings", chord)
		}
	}
}
```

- [x] **Step 2:** Run → FAIL (`undefined: RoleBindings`).
- [x] **Step 3:** Add `type RoleBinding struct { Chord Chord; Role PaneRole; Help string }`, the `roleBindings` var, and `RoleBindings()`/`RoleBindingKeys()` accessors. Read `Decide`'s terminal branch (`shortcut.go:150-180`) for each chord's real behaviour before writing its sentence — `Alt+t` new tab, `Alt+w` close tab, `Alt+r` rename tab, `Alt+Shift+D` split terminal down, `Alt+Shift+⏎` toggle focused-side width, `Alt+k` jump back to the left pane.
- [x] **Step 4:** Run → PASS. Confirm `Decide`'s behaviour is untouched: `go test ./cmd/internal/workbenchshortcut/` fully green (the switch is the authority; this table only describes it).
- [x] **Step 5: Commit**

```bash
git commit -am "#132: roleBindings — wording for the pane-local chords, beside the behaviour"
```

### Task 5: `SourceReader` + `Catalog` + the bidirectional drift tests

**Files:**
- Create: `cmd/internal/keyhelp/sources.go`, `cmd/internal/keyhelp/catalog.go`
- Test: `cmd/internal/keyhelp/drift_test.go`

This is the task that makes #132 unrepeatable. Write the drift tests FIRST.

- [x] **Step 1: Write the failing drift tests**

```go
package keyhelp

import "testing"

// EVERY pair: keymap in the real init.lua must be classified — either surfaced in
// the catalog or explicitly marked internal. A new user-facing key therefore
// cannot reach a release undocumented, and an internal one cannot leak into user
// help. This is the anti-rot invariant #132 exists to install; #99 M5c had no
// such test, which is why the help silently emptied.
func TestEveryNvimKeymapIsClassified(t *testing.T) {
	for _, km := range ParseNvimKeymaps(mustReadTreeSource(t, "nvim/init.lua")).Resolved {
		if !Catalog.Includes(km.Key) && !Catalog.IsInternal(km.Key) {
			t.Errorf("keymap %q (%q) is neither in the help catalog nor marked internal — classify it in catalog.go", km.Key, km.Desc)
		}
	}
}

// The reverse direction: a catalog entry whose source row is gone is stale help.
func TestEveryCatalogEntryStillExists(t *testing.T) {
	live := map[string]bool{}
	for _, km := range ParseNvimKeymaps(mustReadTreeSource(t, "nvim/init.lua")).Resolved {
		live[km.Key] = true
	}
	for _, b := range GlobalBindingKeys() {
		live[b] = true
	}
	for _, b := range RoleBindingKeys() {
		live[b] = true
	}
	for _, z := range ParseZellijRunBinds(mustReadTreeSource(t, "zellij/config.kdl")) {
		live[z.Key] = true
	}
	for _, key := range Catalog.Keys() {
		if !live[key] {
			t.Errorf("catalog documents %q but no source defines it any more — remove or repoint it", key)
		}
	}
}

// Both zellij-level Run binds must be documented; they are invisible to nvim, so
// nothing else would ever surface them. Alt+h documenting ITSELF is the point.
// The classification tests above read the TREE copies on purpose. This one is the
// only thing tying the shipped bundle to them: assets/ is gitignored and `go test`
// never regenerates it, so without this a stale embedded snapshot would silently
// weaken every classification assertion (#132 PQ-3).
func TestEmbeddedSourcesMatchTree(t *testing.T) {
	for _, path := range []string{"nvim/init.lua", "zellij/config.kdl"} {
		tree := mustReadTreeSource(t, path)
		embedded, err := runtimebundle.EmbeddedAsset(path)
		if err != nil {
			t.Fatalf("%s missing from the runtime bundle: %v", path, err)
		}
		if string(embedded) != tree {
			t.Errorf("%s: embedded bundle is stale — run `make build` to regenerate", path)
		}
	}
}

func TestZellijRunBindsAreDocumented(t *testing.T) {
	for _, z := range ParseZellijRunBinds(mustReadTreeSource(t, "zellij/config.kdl")) {
		if !Catalog.Includes(z.Key) {
			t.Errorf("zellij Run bind %q is undocumented", z.Key)
		}
	}
}
```

- [x] **Step 2: Run to verify they fail**

Run: `go test ./cmd/internal/keyhelp/ -run Drift -v` (or `-run 'Classified|StillExists|Documented'`)
Expected: FAIL — undefined `Catalog`, `mustReadSource`, `GlobalBindingKeys`.

- [x] **Step 3: Implement `sources.go`, then `catalog.go`**

`sources.go` — `SourceReader` interface with an `EmbeddedAsset`-backed default (used by `pair keys` at runtime and by the render test), plus **two distinct test helpers**, matching the CRITICAL split above:
- `mustReadTreeSource(t, path)` — reads `../../../<path>` from the working tree. **All classification/drift tests use this**, because the embedded copy is gitignored and never regenerated by `go test`.
- `mustReadEmbeddedSource(t, path)` — reads the embedded asset; used only by the render/fidelity test and by `TestEmbeddedSourcesMatchTree`.

```go
type SourceReader interface {
	Read(path string) ([]byte, error)
}

type embeddedSources struct{}

func (embeddedSources) Read(path string) ([]byte, error) {
	return runtimebundle.EmbeddedAsset(path)
}

func DefaultSources() SourceReader { return embeddedSources{} }
```

Verify the asset paths first — `grep -n "zellij/layouts/main-2.kdl\|nvim/init.lua" cmd/internal/runtimebundle/embed_test.go` shows the manifest spelling; use exactly those strings (`nvim/init.lua`, `zellij/config.kdl`).

`catalog.go` — the curated table. Populate it from the **real** parsed output (Task 2 Step 4a's log), not from this plan's memory of it. Every `Desc` stays empty here; wording is looked up from the source at build time.

```go
// Catalog decides INCLUSION, GROUP and ORDER. It deliberately holds no wording:
// descriptions come from the sources (nvim desc / GlobalBinding.Help), so help
// text cannot drift from behaviour. Anything in a source and in neither list
// fails TestEveryNvimKeymapIsClassified.
var Catalog = catalog{
	include: []catalogEntry{
		{Key: "Alt h", Group: "Session", Order: 10, Help: "show this keybinding help"},
		{Key: "<M-CR>", Group: "Draft", Order: 10},
		// … one row per user-facing key, from the real inventory
	},
	internal: []string{
		"<Tab>", "<S-Tab>", "<CR>", "<BS>", "<LeftMouse>", "z=", "ZZ", "ZQ",
		// autopair/completion/spell mechanics — editor internals, not workbench keys
	},
}
```

Zellij `Run` binds have no upstream prose, so those two rows carry `Help` here (the one exception, called out in the comment).

- [x] **Step 4: Run the drift tests until green — by CLASSIFYING, not by weakening**

Run: `go test ./cmd/internal/keyhelp/ -v`
Expected: every real keymap named in a failure gets a decision. If tempted to loosen an assertion to get green, stop: that is exactly the regression this task prevents.

- [x] **Step 5: Commit**

```bash
git add cmd/internal/keyhelp/
git commit -m "#132: catalog + bidirectional drift tests — undocumented or stale keys now fail the build"
```

### Task 6: assemble `Sections()`

**Files:**
- Modify: `cmd/internal/keyhelp/keyhelp.go`
- Test: `cmd/internal/keyhelp/keyhelp_test.go`

- [x] **Step 1: Write the failing test**

```go
// The composed document: wording comes from the sources, grouping/order from the
// catalog, and Alt+h must document itself — the discovery path the statusline
// promises (nvim/init.lua PAIR_CHEATS keeps Alt+h last-to-drop).
func TestSectionsDeriveWordingFromSources(t *testing.T) {
	secs, err := Sections(DefaultSources())
	if err != nil {
		t.Fatal(err)
	}
	find := func(key string) (Binding, bool) {
		for _, s := range secs {
			for _, b := range s.Bindings {
				if b.Key == key {
					return b, true
				}
			}
		}
		return Binding{}, false
	}
	send, ok := find("Alt+⏎")
	if !ok {
		t.Fatal("Alt+⏎ missing from help")
	}
	if send.Desc != "send buffer + clear" { // the desc authored in init.lua
		t.Errorf("Alt+⏎ desc = %q, want the init.lua wording", send.Desc)
	}
	if _, ok := find("Alt+h"); !ok {
		t.Error("Alt+h must document itself — it is the advertised discovery path")
	}
	if _, ok := find("Alt+x"); !ok {
		t.Error("Alt+x (quit) missing — GlobalBinding.Help not wired in")
	}
}

// Regression pin for the original bug in its exact form.
func TestHelpNeverTellsYouToPressAltH(t *testing.T) {
	secs, _ := Sections(DefaultSources())
	out := Render(secs)
	if strings.Contains(out, "keybindings are on Alt+h") {
		t.Error("the help must not refer the reader back to the help key (#132)")
	}
	if !strings.Contains(out, "send buffer") {
		t.Error("the help must contain real bindings, not a CLI synopsis (#132)")
	}
}
```

- [x] **Step 1a: Add the test that pins the join rule where it actually bites**

```go
// Alt+t/w/r have nvim keymaps describing the draft NO-OP ("right-terminal tab
// helper disabled in draft", init.lua:3654). Their user-facing wording must come
// from roleBindings. If the join ever falls back to "whichever source has prose",
// this test fails — which is the whole reason rows name their source (#132 PQ-2).
func TestRoleLocalWordingComesFromRoleTableNotNvimNoOp(t *testing.T) {
	secs, err := Sections(DefaultSources())
	if err != nil {
		t.Fatal(err)
	}
	out := Render(secs)
	if strings.Contains(out, "disabled in draft") {
		t.Error("help shows the draft no-op desc as a feature description")
	}
	if !strings.Contains(out, "new tab") { // from roleBindings' Help for Alt+t
		t.Error("Alt+t wording did not come from roleBindings")
	}
}

// A dual-meaning key renders TWICE, in different sections, each with its own
// wording: <S-M-CR> appends-no-send in the draft and toggles layout in the
// terminal (init.lua:3632 vs shortcut.go:176).
func TestDualMeaningKeyRendersInBothContexts(t *testing.T) {
	secs, _ := Sections(DefaultSources())
	var seen []string
	for _, sec := range secs {
		for _, b := range sec.Bindings {
			if b.Key == "Shift+Alt+⏎" {
				seen = append(seen, sec.Title+": "+b.Desc)
			}
		}
	}
	if len(seen) != 2 {
		t.Fatalf("Shift+Alt+⏎ should appear once per context, got %v", seen)
	}
}
```

- [x] **Step 2–4:** Run (FAIL), implement `Sections(SourceReader) ([]Section, error)` — parse all sources, then join **each catalog row to the source it names** (`SourceNvim` / `SourceGlobal` / `SourceRole` / `SourceZellij`). There is **no** "whichever source has prose wins" fallback: a row whose named source lacks wording is an error, not an occasion to borrow the wrong sentence. Sort by Group order then `Order`; map source key spellings to display forms (`<M-CR>` → `Alt+⏎`, `Alt h` → `Alt+h`), reusing `PAIR_CHEATS`' spellings where it covers the key. Then run again (PASS).

  The key→display mapping is itself a small pure function; give it a test with the awkward cases (`<S-M-BS>` → `Shift+Alt+⌫`, `<C-M-c>` → `Ctrl+Alt+c`, `<M-Left>` → `Alt+←`).

- [x] **Step 5: Commit**

```bash
git commit -am "#132: compose help sections — source wording, catalog grouping, Alt+h documents itself"
```

---

## Chunk 3: surfaces

### Task 7: `pair keys`

**Files:**
- Create: `cmd/internal/keyscmd/keyscmd.go`
- Modify: `cmd/internal/dispatcher/dispatcher.go` (command table, ~line 50-66)
- Modify: wherever subcommands are routed (`grep -rn '"context"' cmd/ | grep -v _test` finds the dispatch switch)
- Test: `cmd/internal/keyscmd/keyscmd_test.go`, plus the dispatcher's existing table test

- [x] **Step 1: Write the failing test** — `Run(w io.Writer) int` writes non-empty output containing a known binding and returns 0.
- [x] **Step 2:** Run → FAIL.
- [x] **Step 3:** Implement as a thin wrapper: `Sections(DefaultSources())` → `Render` → `fmt.Fprint(w, …)`. Add `--center <cols>` (pads each line using the shared display-width helper). **Always exit 0**, even on a source error, printing `keybind help unavailable: <err>` as the body — `bin/pair-help` runs under `set -euo pipefail`, so a non-zero exit kills the pane before the pager opens. Test both: the happy path and a failing `SourceReader` returning exit 0 with the diagnostic line.
- [x] **Step 4:** Run → PASS. Then `go build ./... && ./bin/pair keys` — eyeball the real output.
- [x] **Step 5:** Check the dispatcher help/table test expectations updated (`go test ./cmd/internal/dispatcher/`).
- [x] **Step 6: Commit**

```bash
git commit -am "#132: pair keys — print in-session keybindings"
```

### Task 8: point Alt+h at it, and de-circularize `pair -h`

**Files:**
- Modify: `bin/pair-help` (the `help="$(pair -h)"` line)
- Modify: `UsageText` (drop "In-session keybindings are on Alt+h.")
- Test: `cmd/internal/launcher/format_test.go` (or wherever `UsageText` is tested)

- [x] **Step 1: Write the failing test**

```go
// The circular sentence is the bug. Usage may POINT at the keybind surface but
// must never send the reader back to the key they just pressed (#132).
func TestUsageTextIsNotCircular(t *testing.T) {
	got := UsageText()
	if strings.Contains(got, "keybindings are on Alt+h") {
		t.Error("usage still refers keybindings back to Alt+h")
	}
	if !strings.Contains(got, "pair keys") {
		t.Error("usage should point at `pair keys` for keybindings")
	}
}
```

- [x] **Step 2:** Run → FAIL.
- [x] **Step 3:** Replace the final sentence with: ``In-session keybindings: run `pair keys` (or Alt+h inside a session).`` Add `pair keys` to the USAGE list as `pair keys                     in-session keybindings`.
- [x] **Step 4:** Run → PASS.
- [x] **Step 5: `bin/pair-help` — two latent bugs the plan must not inherit**

Replace `help="$(pair -h)"` with `help="$(pair keys --center "$(tput cols 2>/dev/null || echo 80)")"` and **delete** the awk/sed centering block (`bin/pair-help:30-33`).

Why, precisely — both verified:
  1. `awk '{ if (length > m) m = length }'` measures **bytes**, so glyph keys
     (`Alt+⏎`, `Alt+←`, `Alt+⌫`) inflate the measured width and skew the centering.
     The plan previously called this centering "already correct"; it is not, and it
     gets worse with the new content. Centering in Go reuses the one display-width
     helper (Task 1 Step 3a) instead of adding a second wrong measure.
  2. `set -euo pipefail` (`bin/pair-help:7`) means a **non-zero `pair keys` aborts
     the script before `less` runs** — the pane would flash and close. So
     `pair keys` MUST exit 0 even when a source read fails, printing a one-line
     `keybind help unavailable: <err>` as its body. A dead help key is the bug being
     fixed; it must not be replaced with a dead pane. Correct the Task 7 Step 3
     wording accordingly.

Then: `grep -n "pair -h" bin/pair-help` must return nothing.
- [x] **Step 6: Commit**

```bash
git commit -am "#132: Alt+h pages pair keys; usage stops pointing at itself"
```

### Task 9: docs, bundle, and live verification

**Files:**
- Modify: `README.md` (keybinding/help mentions), `atlas/architecture.md` (+ `atlas/index.md` if a new file is listed)

- [x] **Step 1: Sweep the rendered shape, not just symbols** (the #133 lesson)

Run: `grep -rn --no-ignore-files "pair --help\|pair -h\|keybindings are on" README.md atlas/ zellij/ bin/ nvim/ Makefile* | grep -v workshop`

`nvim/` is in the list because `PAIR_CHEATS` and the statusline strings live there and reference the help path.

Every hit claiming `--help` shows keybindings must be repointed at `pair keys`. Note the Homebrew formula caveat ("Run `pair --help` for keybindings") lives in the **separate `homebrew-pair` repo**, not here — it is #131's fix; record that split in the issue `## Log` rather than leaving Done-when's "no text anywhere" ambiguous.

- [x] **Step 2: Atlas** — record that `Alt+h` → `bin/pair-help` → `pair keys` → `keyhelp`, that wording derives from nvim `desc` + `GlobalBinding.Help`, and that the drift tests are what keep it honest.

- [x] **Step 3: Regenerate + verify the embedded bundle**

Run: `make build && grep -rn --no-ignore-files "keybindings are on Alt+h" cmd/internal/runtimebundle/assets/ ; echo "(empty = clean)"`

The assets dir is gitignored; an ignore-respecting grep reports a false zero (#133).

- [x] **Step 4: Live check**

`make build`, then in a live session press `Alt+h`: real bindings, `q`/`Esc` dismisses. Read the pane titles back mechanically rather than by eyeball where possible — `zellij action list-panes --json` needs the sandbox disabled (see the #133 log).

- [x] **Step 5: Full suite**

Run: `env -u PAIR_SESSION_ID -u PAIR_TAG -u PAIR_PANE_CWD make test`
Expected: exit 0. `termcmd`/`wrapcmd` pty tests need the sandbox disabled.

- [x] **Step 6: Commit**

```bash
git commit -am "#132: docs + atlas for the keyhelp surface"
```

---

## Done when (revised — see the Scope note)

- `Alt+h` in a live session lists the workbench chords, dismissable with `q`/`Esc`.
- Help **wording** derives from the sources that own it (nvim `desc`, `GlobalBinding.Help`); no binding description is authored twice anywhere.
- Adding a keymap or global chord without classifying it **fails a test**; documenting one that no longer exists also fails. Silent rot is impossible — this replaces the original "no second edit" wording, which would have published editor internals as user help.
- No text **in this repo** claims `pair --help` shows keybindings. The Homebrew formula's caveat is #131's, in another repo.
- `make test` exit 0.

## Risks

- **The nvim parser silently under-reads.** Highest risk: a missed keymap looks like success. Mitigated by Task 2 Step 4a (parsed count must equal `grep -c "desc = 'pair: "`) and by `TestEveryNvimKeymapIsClassified` failing loudly on anything unclassified.
- **Positional-literal breakage** from the new `GlobalBinding` field — caught by `go build ./...`, and Task 4 Step 5 pins that generated Lua is unaffected.
- **Display-width duplication.** `launcher/list.go:88` already pads by display width; Step 3a forces reuse instead of a second implementation.
- **`Alt+l` surprise.** The zellij scan will surface `Alt+l` (changelog), which the old shell help never listed. Document it — it is user-facing and this is the moment it becomes discoverable.
