# Claude Return Rewrite Only in Composer Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move Claude off `composerGateLegacy` onto #139's positive composer gate, so plain Return rewrites only when Pair can see Claude's live composer on screen.

**Architecture:** #139 built every generic part — the `harnessTTYProfiles` registry, the pure `decidePlainReturn`, the shared x/vt `terminalModel`, overlay/composer precedence, and the fixture + live conformance harness. Claude needs one recognizer and one profile flip. Captured evidence shows Claude's composer is the same *shape* as Muse's — a prompt glyph at column 0 as the first row inside a pair of rule rows — so rather than write a third near-copy, this plan extracts that shape into one shared predicate parameterised by a per-harness spec, migrates Muse onto it (its frozen differential rows are the no-change oracle), and adds Claude as the second consumer.

**Tech Stack:** Go, `github.com/charmbracelet/ultraviolet` (x/vt terminal model), `github.com/creack/pty` (bounded capture seam).

---

## Blast radius — read this before starting

Claude is **pair's default agent** (`atlas/architecture.md`). This flip changes
the failure mode for every Claude session, and it changes it in the expensive
direction:

| | today (`composerGateLegacy`) | after the flip |
|---|---|---|
| recognizer wrong in a menu | remap fires; verified inert — Claude inserts a newline and dismisses the menu | same |
| recognizer wrong in a **real composer** | cannot happen — the gate always fires | bare CR → **Claude submits the half-written message** |

So this issue trades a mild, verified-inert failure for a rarer but costly one.
Every false negative is a lost draft. Two consequences the plan is built around:

1. The recognizer must be biased against false negatives — it must match every
   composer variant, not just the default one (see the mode evidence below).
2. Frozen-fixture replay at a fixed 120x38 **cannot** observe this class of
   problem. Task 5 adds a live dogfood step in a real `pair` session before
   close, per the repo's dogfood-live value.

---

## Captured evidence (Claude Code 2.1.236)

Captured live through the Task 2A bounded PTY seam at 120x38. Task 4 re-captures
these as checked-in fixtures; nothing here is hand-authored.

**Composer, idle.** Cursor `(2,6)`. Rule `─` fg `rgb(136,136,136)` at rows 5 and
7; `❯` at column 0 row 6, **default foreground**.

**Composer, two lines** (sent `alpha`, then Pair's own `plainCR` = `\` `CR`, then
`beta`). Cursor `(6,7)`; rules move to 5 and 8; `❯ alpha` at row 6, `  beta` at
row 7 indented to column 2. Pair's remap already produces a real newline; the
box grows exactly like Muse's.

**Slash menu.** Cursor `(3,6)`; box intact at rows 5–7 with `❯ /`; the command
list renders *below* the box from row 9, each entry starting at column 2. Claude
does **not** paint column 0 in the menu, so it does not reuse the prompt glyph as
a selection marker — the trap Agy fell into (pair#139 I1).

**Slash menu + Pair's `plainCR`.** Cursor `(2,7)`; menu closes, box grows to rows
5–8. Claude treats `\` `CR` as "insert newline" and dismisses the menu — no
backslash typed, no command fired, no submit. Verified with a control run
sending a printable `X` (which lands as `❯ /X` and filters the menu), because an
earlier attempt silently failed to deliver the keystroke and would otherwise have
"proved" the opposite.

**Mode-switched composers — the load-bearing finding.**

| mode | trigger | column-0 prompt | rule foreground |
|---|---|---|---|
| default | — | `❯` default fg | `rgb(136,136,136)` |
| memory | type `#` | `❯` default fg | `rgb(136,136,136)` |
| plan/manual | shift+tab | `❯` default fg | `rgb(136,136,136)` |
| **bash** | type `!` | **`!`** fg `rgb(253,93,177)` | **`rgb(253,93,177)`** |

Bash mode changes **both** the glyph and the rule colour. A recognizer pinned to
`❯` + grey would decline there, and per the blast-radius table that means Return
submits a half-written shell command. This is why the spec below anchors on
**structure**, not on a glyph/colour allowlist: an allowlist is a false-negative
generator every time Claude adds a mode.

Note the two rules always match *each other's* colour within a mode. That is the
invariant the spec uses in place of a fixed colour.

---

## Core concepts

### Pure entities (the conceptual core)

| Name | Lives in | Status |
|------|----------|--------|
| `ruledBoxComposerSpec` | `cmd/internal/wrapcmd/composer_recognizers.go` | new |
| `ruledBoxComposerActive` | `cmd/internal/wrapcmd/composer_recognizers.go` | new |
| `claudeComposerActive` | `cmd/internal/wrapcmd/composer_recognizers.go` | new |
| `museComposerActive` | `cmd/internal/wrapcmd/composer_recognizers.go` | modified |
| `museComposerBottomRule` | `cmd/internal/wrapcmd/composer_recognizers.go` | deleted |
| `ttyFixtureReturnExpectation` | `cmd/internal/wrapcmd/harness_tty_fixture_test.go` | modified |

- **ruledBoxComposerSpec** — the per-harness parameters of a ruled-box composer:
  a predicate qualifying the column-0 prompt cell, a predicate matching a rule
  row's column-0 cell, whether the box's two rules must match each other, the
  height bound, and the minimum cursor column.
  - **Relationships:** 1:1 with a harness profile using the ruled-box shape;
    N:1 into `ruledBoxComposerActive`.
  - **DRY rationale:** Muse and Claude paint the identical structure. Without
    this, Claude's recognizer is a near-verbatim copy of Muse's — including the
    two defects pair#139's reviews already found and fixed there (an unqualified
    prompt glyph accepted; a box that could only ever be one line tall). Copying
    the code copies the footguns.
  - **Future extensions:** a harness whose chrome is vertical bars widens the
    rule predicate; a prompt at a column other than 0 adds a column field.

- **ruledBoxComposerActive** — pure `(terminalSnapshot, ruledBoxComposerSpec) bool`.
  Walks up from the cursor for a qualifying prompt immediately preceded by a rule
  row, then confirms the box's closing rule sits at or below the cursor.
  - **DRY rationale:** one implementation of "cursor is inside a ruled box", so
    the multi-line and cursor-on-rule fixes live in one place.

- **claudeComposerActive** — Claude's spec. Anchors on structure: a `─` rule row
  at column 0 above and below, a single non-blank non-`─` glyph at column 0 on
  the prompt row, and the two rules carrying the *same* foreground as each other.
  Deliberately glyph- and colour-agnostic so bash mode and future modes match.

- **museComposerActive** — reimplemented on the shared predicate. **Behavior must
  not change**; the existing rows in `TestMuseComposerActiveSnapshotDifferential`
  and the checked-in Muse fixture are the differential oracle.

- **ttyFixtureReturnExpectation** — currently returns only *whether* a fixture
  must remap; it must also yield *what bytes* that means, read from the harness
  profile's `keymap.plainCR`. See Task 3.

### Integration points (where pure meets the world)

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `harnessTTYProfiles["claude"]` | `cmd/internal/wrapcmd/harness_tty.go` | modified | Return routing policy |
| `testdata/tty/claude/<version>/` | `cmd/internal/wrapcmd/testdata/tty/` | new | literal Claude PTY bytes |
| `harnessTTYDrivenScenarios["claude"]` | `cmd/internal/wrapcmd/harness_tty_live_test.go` | new | live Claude CLI |

- **harnessTTYProfiles["claude"]** — flips `composerGate` to
  `composerGatePositive` and registers `recognize`. `keymap.plainCR` stays
  `{'\\', '\r'}` and `overlay` stays `detectClaudeOverlayOpen`: this issue
  changes *when* the remap fires, not what it emits or what suspends it.
  - **Injected into:** `decidePlainReturn`, unchanged.

- **testdata/tty/claude/<version>/** — literal captured bytes plus
  `metadata.json` (agent, exact `--version`, RFC3339 capture time, argv, SHA-256
  per raw file), replayed by `TestHarnessTTYFixtureConformance` through the
  production proxy at every split point.
  - **Fake/conformance shape (ARCH-MOCK):** the frozen fixtures are the stateful
    double — production and test flow share `proxy.handleChunk`. The live check
    against the installed CLI is the drift detector; it is manual/opt-in via
    `PAIR_LIVE_HARNESS` and nothing schedules it, as `atlas/architecture.md`
    already states.

- **harnessTTYDrivenScenarios["claude"]** — drives Claude one keystroke past
  startup to reach non-composer and mode-switched screens for fixtures.

### Task ordering constraint (why the tasks are in this order)

`newHarnessTTYLiveClassifier` fatals when `profile.recognize == nil`, so the live
capture path **cannot run until Claude's recognizer exists and its profile is
flipped**. The recognizer must therefore be designed from the raw evidence above
(already captured to the session scratchpad) and land *before* the fixture
capture. This is the same chicken-and-egg pair#139 hit with Codex. Concretely:
Task 2 writes the recognizer against generated snapshots only, Task 3 fixes the
replay's keymap assumption, Task 4 flips the profile and captures the literal
fixtures, and only then do literal-fixture rows get added.

---

## Chunk 1: Share the ruled-box shape, then add Claude

### Task 1: Extract the shared ruled-box predicate and migrate Muse

**Files:**
- Modify: `cmd/internal/wrapcmd/composer_recognizers.go`
- Test: `cmd/internal/wrapcmd/composer_recognizers_test.go`

- [ ] **Step 1: Freeze the Muse oracle**

Run: `go test ./cmd/internal/wrapcmd -run 'MuseComposerActiveSnapshotDifferential' -count=1 -v`

Record every row's result. These rows — including the two-line, three-line, and
height-ceiling rows added by pair#139 — are the contract the refactor must
preserve exactly. No row may be edited in this task.

- [ ] **Step 2: Add the spec type and shared predicate**

```go
// ruledBoxComposerSpec parameterises the composer shape Claude and Muse share:
// a prompt glyph at column 0 forming the first row inside a pair of rule rows.
type ruledBoxComposerSpec struct {
	promptOK   func(uv.Cell) bool
	ruleAt     func(terminalSnapshot, int) bool
	rulesMatch func(top, bottom uv.Cell) bool // nil = no cross-rule constraint
	maxRows    int
	minCursorX int
}

// ruledBoxComposerActive reports whether the cursor rests inside a ruled box.
// The prompt sits at or above the cursor — except when the cursor rests on the
// box's own top rule — and the closing rule must sit at or below the cursor,
// which is what lets the composer grow past one line.
func ruledBoxComposerActive(snapshot terminalSnapshot, spec ruledBoxComposerSpec) bool {
	if !snapshot.CursorVisible || !snapshotCoordinatesValid(snapshot) ||
		snapshot.Cursor.X < spec.minCursorX {
		return false
	}
	for promptY := snapshot.Cursor.Y + 1; promptY >= 0 && snapshot.Cursor.Y-promptY < spec.maxRows; promptY-- {
		if promptY >= snapshot.Height || promptY-1 < 0 {
			continue
		}
		cell := snapshot.CellAt(0, promptY)
		if cell == nil || !spec.promptOK(*cell) || !spec.ruleAt(snapshot, promptY-1) {
			continue
		}
		bottom, ok := ruledBoxBottomRule(snapshot, spec, promptY)
		if !ok || bottom < snapshot.Cursor.Y {
			continue
		}
		if spec.rulesMatch != nil {
			top, bot := snapshot.CellAt(0, promptY-1), snapshot.CellAt(0, bottom)
			if top == nil || bot == nil || !spec.rulesMatch(*top, *bot) {
				continue
			}
		}
		return true
	}
	return false
}

// ruledBoxBottomRule finds the first row below the prompt that paints column 0
// and reports whether it is the box's closing rule.
func ruledBoxBottomRule(snapshot terminalSnapshot, spec ruledBoxComposerSpec, promptY int) (int, bool) {
	for y := promptY + 1; y < snapshot.Height && y-promptY <= spec.maxRows; y++ {
		cell := snapshot.CellAt(0, y)
		if cell == nil || strings.TrimSpace(cell.Content) == "" {
			continue
		}
		return y, spec.ruleAt(snapshot, y)
	}
	return 0, false
}
```

- [ ] **Step 3: Reimplement Muse on the shared predicate**

```go
func museComposerActive(snapshot terminalSnapshot) bool {
	return ruledBoxComposerActive(snapshot, ruledBoxComposerSpec{
		promptOK: func(c uv.Cell) bool {
			return c.Content == "⟩" && c.Style.Attrs&uv.AttrFaint == 0
		},
		ruleAt:  func(s terminalSnapshot, y int) bool { return faintRuleAt(s, 0, y) },
		maxRows: museComposerMaxRows,
		// Column 0 is the prompt, column 1 its trailing space.
		minCursorX: 2,
	})
}
```

Delete `museComposerBottomRule`, superseded by `ruledBoxBottomRule`.

- [ ] **Step 4: Verify no Muse behavior changed**

```bash
go test ./cmd/internal/wrapcmd -run 'MuseComposerActiveSnapshotDifferential' -count=1 -v
go test ./cmd/internal/wrapcmd -run 'TestHarnessTTYFixtureConformance' -count=1
```

Expected: every row from Step 1 has the same result and the Muse fixture replays
identically at every split. A single changed row means the abstraction is wrong —
fix the predicate, never the row.

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/wrapcmd/composer_recognizers.go cmd/internal/wrapcmd/composer_recognizers_test.go
git commit -m "wrapcmd: #138: share the ruled-box composer predicate" -m "Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

### Task 2: Claude recognizer (generated snapshots only)

**Files:**
- Modify: `cmd/internal/wrapcmd/composer_recognizers.go`
- Test: `cmd/internal/wrapcmd/composer_recognizers_test.go`

Literal-fixture rows are deliberately **not** added here — the fixtures do not
exist until Task 4, per the ordering constraint above.

- [ ] **Step 1: Write failing differential rows**

Add `TestClaudeComposerActiveSnapshotDifferential` in the Codex/Muse shape.
Strategy rather than an enumeration: for each risky behavior of
`claudeComposerActive`, add the row that would catch its regression.

- *Multi-line growth* (the pair#139 C1/C2 class, and the expensive direction
  here): cursor on a painted continuation row; on an *empty* continuation row;
  with a blank line between prompt row and cursor.
- *Mode coverage* (the false-negative class this issue's blast radius makes
  costly): the default `❯`/grey box **and** the bash `!`/pink box, both
  recognized, proving the predicate is not colour- or glyph-pinned.
- *Box integrity*: prompt with no rule above; prompt with no rule below; two
  rules whose colours differ from each other (must decline — that is the
  `rulesMatch` invariant); cursor resting on the closing rule (must recognize).
- *Fail-closed basics*: hidden cursor; erased screen; cursor before
  `minCursorX`; a draft taller than the height ceiling, pinned with a comment
  naming it a deliberate ceiling per pair#139 I2.

Also add `claude` to the recognizer list in
`TestComposerRecognizersRejectAdversarialSnapshotsWithoutBlocking` — that
mechanical guard covers the malformed-snapshot space that prose rows cannot.

- [ ] **Step 2: Verify RED**

Run: `go test ./cmd/internal/wrapcmd -run 'ClaudeComposerActive' -count=1`
Expected: FAIL — `claudeComposerActive` undefined.

- [ ] **Step 3: Implement the spec**

```go
// claudeComposerMaxRows bounds the box height. Inherited from the box-structure
// derivation rather than measured against Claude; the enclosing rules already
// prevent pairing distant chrome.
const claudeComposerMaxRows = 20

// claudeComposerActive reports whether the cursor rests inside Claude's live
// composer: a single glyph at column 0 between two matching rule rows.
//
// Deliberately glyph- and colour-agnostic. Claude repaints both the prompt
// glyph and the rule colour per input mode — bash mode is "!" with pink rules
// where the default is "❯" with grey — so an allowlist would decline in any
// mode it had not enumerated, and for Claude a decline means the next Return
// submits a half-written draft. The two rules must match each other's
// foreground, which is what keeps unrelated chrome from forming a box.
func claudeComposerActive(snapshot terminalSnapshot) bool {
	return ruledBoxComposerActive(snapshot, ruledBoxComposerSpec{
		promptOK: func(c uv.Cell) bool {
			return strings.TrimSpace(c.Content) != "" && c.Content != claudeComposerRule
		},
		ruleAt: func(s terminalSnapshot, y int) bool {
			cell := s.CellAt(0, y)
			return cell != nil && cell.Content == claudeComposerRule
		},
		rulesMatch: func(top, bottom uv.Cell) bool {
			return sameForeground(top.Style.Fg, bottom.Style.Fg)
		},
		maxRows:    claudeComposerMaxRows,
		minCursorX: 2,
	})
}
```

`claudeComposerRule` is `"─"`; `sameForeground` is a small pure helper comparing
two `color.Color` values (both nil counts as equal).

- [ ] **Step 4: Verify GREEN**

Run: `go test ./cmd/internal/wrapcmd -run 'ComposerActive|Adversarial' -count=1 -v`
Expected: all Claude rows PASS; every Codex/Muse/Agy row unchanged.

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/wrapcmd/composer_recognizers.go cmd/internal/wrapcmd/composer_recognizers_test.go
git commit -m "wrapcmd: #138: recognize Claude's ruled composer box" -m "Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

## Chunk 2: Teach the harness Claude's keymap, flip the gate, pin the evidence

### Task 3: Derive the fixture replay's expected Return from the profile

**Files:**
- Modify: `cmd/internal/wrapcmd/harness_tty_fixture_test.go`

`replayHarnessTTYFixture` hardcodes `"\n"` as the composer Return and `"\r"`
otherwise. That holds for Codex, Muse, and Agy but is **wrong for Claude**,
whose `plainCR` is `{'\\', '\r'}`. Left alone, Claude's fixture would fail for a
reason that has nothing to do with recognition.

- [ ] **Step 1: Write the failing assertion**

Add a table test asserting the expected composer Return bytes per harness, read
from `profileForHarness(harness, true).keymap.plainCR`: `"\n"` for codex/muse/agy
and `"\\\r"` for claude.

- [ ] **Step 2: Verify RED, then derive instead of hardcode**

Replace the literal with the profile lookup, so the expectation is one
source-of-truth derivation rather than a restatement (ARCH-DRY):

```go
profile, ok := profileForHarness(harness, true)
if !ok {
	t.Fatalf("no profile for %s", harness)
}
wantReturn := "\r"
if wantComposer {
	wantReturn = string(profile.keymap.plainCR)
}
```

- [ ] **Step 3: Verify GREEN and commit**

```bash
go test ./cmd/internal/wrapcmd -run 'TestHarnessTTYFixtureConformance' -count=1
git add cmd/internal/wrapcmd/harness_tty_fixture_test.go
git commit -m "test: #138: derive fixture Return expectation from the profile keymap" -m "Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

### Task 4: Flip the gate and capture Claude fixtures

**Files:**
- Modify: `cmd/internal/wrapcmd/harness_tty.go`
- Modify: `cmd/internal/wrapcmd/harness_tty_test.go`
- Modify: `cmd/internal/wrapcmd/harness_tty_live_test.go`
- Modify: `cmd/internal/wrapcmd/harness_tty_fixture_test.go`
- Modify: `cmd/internal/wrapcmd/composer_recognizers_test.go`
- Create: `cmd/internal/wrapcmd/testdata/tty/claude/<version>/composer.raw` + `metadata.json`
- Create when captured: `bash-mode.raw`, `overlay.raw`

- [ ] **Step 1: Flip the profile and verify RED**

Set `composerGate: composerGatePositive` and `recognize: claudeComposerActive`,
and update the `harness_tty_test.go` characterization pinning Claude as legacy.

Run: `go test ./cmd/internal/wrapcmd -run 'TestHarnessTTYFixtureConformance' -count=1`
Expected: FAIL — `required positive-gated fixtures missing: claude`.

- [ ] **Step 2: Capture the composer fixture**

Add `claude` to the live test's `commands` map with the argv Pair launches, then:

```bash
PAIR_LIVE_HARNESS=claude \
PAIR_LIVE_CAPTURE_OUT=cmd/internal/wrapcmd/testdata/tty/claude/<version>/composer.raw \
  go test ./cmd/internal/wrapcmd -run '^TestHarnessTTYLiveConformance$' -count=1 -v
```

Write `metadata.json` with the exact `claude --version`, that argv, an RFC3339
capture time, and a SHA-256 per raw file. Never hand-author bytes.

- [ ] **Step 3: Capture the bash-mode composer as a second positive**

Add a `harnessTTYDrivenScenarios["claude"]` entry sending `!`, expectation
`wantComposer: true`, captured as `bash-mode.raw`. Register that filename in
`ttyFixtureExpectation["claude"]` as `true`. This is the mode whose glyph and
rule colour differ, so it is the fixture that would catch a future
colour-pinned regression.

- [ ] **Step 4: Attempt a declining state — and distinguish the two outcomes**

Add a scenario driving Claude to its permission prompt (ask it to run a shell
command in a checkout where permissions are not pre-approved). There are three
distinct outcomes and they are **not** interchangeable:

1. **Reached, and the gate declines** → capture as `overlay.raw`, expectation
   `false`. Claude now has a discriminating negative; no ledger entry.
2. **Reached, and the gate *accepts* it** → this is a **blocking defect**, not an
   evidence gap. Plain Return would leak a newline into a permission dialog.
   Fix `claudeComposerActive` (most likely by requiring the prompt row to be the
   only painted column-0 row inside the box) and re-run. Do **not** record this
   in `ttyFixtureDiscriminationGaps`.
3. **Not reachable** (e.g. the child inherits a bypass-permissions setting) →
   record the blocker in **both** ledgers, because Claude would then have no
   captured declining state at all: `ttyFixtureNegativeGaps` (no declining state
   exists) *and* `ttyFixtureDiscriminationGaps` (nothing proves the gate
   separates a composer from a same-shaped picker), each naming exactly what is
   missing and the command that would capture it, as `agy` and `muse` already
   are. Do not invent a fixture.

- [ ] **Step 5: Verify GREEN across the whole conformance surface**

```bash
go test ./cmd/internal/wrapcmd -count=1
go test -race ./cmd/internal/wrapcmd -count=1
go test ./... -count=1
make test
PAIR_LIVE_HARNESS=claude go test ./cmd/internal/wrapcmd -run '^TestHarnessTTYLive' -count=1 -v
```

`-race` is expected to report only the pre-existing
`TestMasterPumpFlushesStdoutOnTick` `bytes.Buffer` race; confirm no file from
this issue appears in its trace.

- [ ] **Step 6: Commit**

```bash
git add cmd/internal/wrapcmd/harness_tty.go cmd/internal/wrapcmd/harness_tty_test.go cmd/internal/wrapcmd/harness_tty_live_test.go cmd/internal/wrapcmd/harness_tty_fixture_test.go cmd/internal/wrapcmd/composer_recognizers_test.go cmd/internal/wrapcmd/testdata/tty/claude
git commit -m "wrapcmd: #138: gate Claude Return on positive composer detection" -m "Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

## Chunk 3: Dogfood, document, close

### Task 5: Live dogfood in a real pair session

The fixture replay runs at a fixed 120x38 over frozen bytes and cannot observe a
false negative caused by a real terminal size, a user theme, or a mode this plan
did not capture. Claude is the default agent, so this step is not optional.

- [ ] **Step 1: Run a real session and exercise each composer state**

Launch `pair` on Claude and confirm, in the agent pane:

1. plain Return inserts a newline in an empty composer, and again on the second
   and third lines;
2. plain Return still inserts a newline after typing `!` (bash mode) and `#`
   (memory mode);
3. Alt+Return submits in every one of those states;
4. plain Return **confirms** a real permission prompt rather than inserting;
5. the draft pane's Alt+Return path is unchanged.

- [ ] **Step 2: Read the telemetry rather than trusting the eye**

`$PAIR_DATA_DIR/adapt-<tag>.jsonl` records one `return-remap` line per Enter.
Confirm `fired` in composer states and `bypass` with reason
`composer inactive` or `overlay active` in the picker. A `bypass` in a real
composer is the regression this whole step exists to catch — record the
snapshot and fix before closing.

- [ ] **Step 3: Record the result in the issue Log**

Name the terminal size, theme, and Claude version, since the recognizer's
colour-matching invariant is theme-sensitive.

### Task 6: Documentation and close

**Files:**
- Modify: `atlas/architecture.md`
- Modify: `atlas/how-to-bring-up-a-new-harness-cli.md`
- Modify: `README.md`
- Modify: `doctor/README.md`
- Modify: `workshop/issues/000138-claude-return-rewrite-only-in-composer.md`

- [ ] **Step 1: Update the docs that currently single Claude out**

- `atlas/architecture.md` calls Claude the legacy-gated harness and names the
  other three recognizers — update both, and while in that paragraph fix the
  **stale Muse sentence** that still says `museComposerActive` requires the
  prompt "within one row of the cursor", which pair#139's enclosing-rule
  implementation superseded.
- `README.md`'s Return row says the positive gate applies to `codex`, `muse`,
  and `agy` and that `claude` rewrites unconditionally. That becomes wrong the
  moment Task 4 lands and must change in the same window.
- `atlas/architecture.md`'s Conformance paragraph says `composer.raw` "must
  remap to LF" — true only for the three harnesses that existed when it was
  written. Task 3 makes the expectation profile-derived, so this sentence must
  say the fixture must remap to the harness's own `plainCR`.
- `doctor/README.md`'s `composer inactive` row should note Claude now has a
  recognizer.
- Add Claude's signal — structural, glyph- and colour-agnostic, and *why* — to
  the bring-up guide's per-harness list.

- [ ] **Step 2: Record what deliberately did not change**

In the issue Log: `keymap.plainCR` remains `{'\\', '\r'}` and
`detectClaudeOverlayOpen` is unchanged. This issue narrows *when* the remap
fires; the OSC 777 signal remains Claude's picker defense, and broadening
Claude's overlay markers is separate work.

- [ ] **Step 3: Full verification and close**

```bash
git diff --check
go test ./... -count=1
make test
sdlc close --issue 138 --verified '<behavioral evidence incl. the Task 5 dogfood>'
```

Follow the verdict protocol: on FIX-THEN-SHIP fix findings before committing and
bundle them with the close mutations into one commit; do not re-run close. Then
`sdlc pr` and `sdlc merge`.

---

## Risks and non-goals

- **Non-goal: changing what Claude's remap emits.** `plainCR` stays `{'\\', '\r'}`.
- **Non-goal: replacing `detectClaudeOverlayOpen`.** The overlay layer keeps
  absolute precedence; this issue adds the second layer, it does not swap the
  first.
- **Risk: the Muse refactor is behavior-visible.** Mitigated by treating the
  frozen rows and Muse fixture as an unmodifiable oracle (Task 1 Step 4).
- **Risk: theme dependence.** The recognizer no longer pins a specific colour,
  but it does require the two rules to share one — a theme that renders them
  differently would decline. Task 5 records the theme under test; the live
  conformance check is the ongoing detector.
- **Risk: an unenumerated composer mode.** Structure-anchoring is the mitigation,
  and bash mode is pinned as a fixture precisely because it is the known variant.
  A mode that abandons the ruled box entirely would still decline; Task 5 step 2's
  telemetry read is how that gets caught in practice.
- **Risk: a Claude picker that paints column 0 between rules** would be accepted.
  Claude's slash menu demonstrably does not, but Task 4 Step 4 outcome 2 is the
  explicit branch for discovering otherwise.
