# Boundary Review — pair#300 (milestone M3)

| field | value |
|-------|-------|
| issue | 300 — integrate qoder harness into pair |
| repo | pair |
| issue file | workshop/issues/000300-integrate-qoder-harness-into-pair.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 301c53815433ccbc16f2cdf33b43ea4f3ded0e4b..1b1977c46650c7c6da496a872dd403c018ccc3c9 |
| command | sdlc milestone-close --issue 300 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-09-21T14:35:41-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M3 delivers what the plan claims: `qoder` is positively gated with live-captured fixtures, and the fixtures are machine-neutral and hash-match `metadata.json`. There is a recognizer, both picker families are marked, and `launcher.MintsSessionID` is a single source for the `--session-id` pin. The fixture conformance test, the Qoder recognizer tests and the launcher mint tests pass on an export of the head commit. The failures I saw there are sandbox `operation not permitted` errors on pty and `/tmp`.

One defect blocks. `detectQoderOverlayOpen` scans the raw rolling buffer, and nothing clears that buffer when the confirming Enter is consumed. In a scratch copy, a picker paint, then Enter, then one tiny follow-up chunk re-arms `pickerActive` on both `overlay.raw` and `selection.raw`. The next composer Enter then passes a bare CR and submits a draft the user meant to continue on a new line. This contradicts the atlas line "stale picker text cannot re-arm the flag".

## Architecture pass/flag

| Marker | Result |
|---|---|
| ARCH-DRY | **Flag** — findings 2 and 3 |
| ARCH-PURE | Pass — the recognizer is a pure function over `terminalSnapshot`, and the detector follows the sibling pattern under `overlayMu` |
| ARCH-PURPOSE | **Flag** — finding 5; I swept the claude-only `--session-id` sites and only `createflow.go:596` is unpinned |
| ARCH-MOCK | Pass — the captured fixtures are the stateful double, with opt-in live and driven conformance. The `--session-id` honoring has only a one-time manual live check |
| ARCH-CONSTRAINTS | Pass — the rolling scan is bounded at 512 bytes per chunk. `TestHarnessTTYFixtureConformance` takes about 28s at head, so fixture size is the cost to watch |
| ARCH-SECURE | Pass — 0 home-path hits in the raw fixtures, and the hashes match `metadata.json` |
| ARCH-ORDER | **Flag** — finding 1 |
| ARCH-FUNERAL | Pass — one fixture version directory is retained per the atlas policy, and trimming shrinks driven captures |

## Strengths
- The fixture replay at every byte split caught a real chunk-boundary bug in the marker detection (`selection.raw` split at 6426). This is the oracle working as designed.
- `trimmedHarnessTTYCapture` (`harness_tty_live_test.go`) checks the cut by replaying it, and keeps the full capture when the decision differs. `TestTrimmedHarnessTTYCapture` pins the accept, no-block and decides-differently branches.
- `TestShouldMintSessionID` ranges `AgentInventory()`, so a new agent must declare which side of the mint set it is on. `TestFreshPinAgentsMintSessionIDButKeepRecoveryProvisional` covers both fresh-launch mint sites for qoder except the create-path collision probe.
- `TestQoderComposerActiveSnapshotDifferential` has real negative rows: chevron between rules, separator rows without a glyph, cursor parked past the closing rule, and an erased composer. The gap ledgers say what the captures leave unproven and do not overclaim.

## Critical findings
1. **Stale re-arm of the overlay flag** (`wrap.go`, `detectQoderOverlayOpen`), family `overlay-flag-rearmed-from-stale-input`, ARCH-ORDER.
   - Reproduced at head: `handleChunk(overlay.raw)` arms the flag, `emitPlainCR` consumes it (bare CR, flag false), then `handleChunk("\x1b[1G\x1b[2K")` leaves it true. The result is identical for `selection.raw`.
   - Cause: `emitPlainCR` clears only `overlayTextTail`. The `rolling` slice is loop-local and still holds the marker for up to 512 raw bytes. The claude/codex OSC consumers trim `rolling` past a match at `wrap.go:3115`, and codex's text path never reads `rolling`. Qoder's raw text scan gets neither protection.
   - `TestCheckOverlayOpen_CodexDoesNotRedetectStalePickerText` (`overlay_test.go:253`) pins the invariant for codex, and qoder has no counterpart.
   - Rule for the family: a detector input that outlives the consumption of the flag must be reset at consumption. Give the raw window a proxy-owned tail that `emitPlainCR` clears beside `overlayTextTail`, or advance past the matched marker like the OSC path.
   - Regression test: paint, then `emitPlainCR`, then a small chunk, then assert `pickerActive` is false. Run it over both `overlay.raw` and `selection.raw`.

## Important findings
2. **`qoderComposerActive` is a near-copy of `ruledBoxComposerActive`** (`composer_recognizers.go`, ~40 lines), family `qoder-branch-copies-claude`, ARCH-DRY.
   - It duplicates the `Cursor.Y+1` prompt scan and the bottom-rule scan (`ruledBoxBottomRule`).
   - The atlas paragraph this diff edits says "add a spec rather than a fourth near-copy", and plan Task 9 Step 2(b) orders the same.
   - The real differences are two spec fields: `promptCol` and `requireVisibleCursor`. Rule: a harness that paints a ruled box registers a `ruledBoxComposerSpec` and no other function owns that loop.
3. **Qoder's prompt column and glyph set are restated in two places** (`composer_recognizers.go` `promptCol = 1` with `">"`/`"*"`, and `orientation.go` `promptCol` plus `orientationPromptOK`), family `hand-restated-registry`, ARCH-DRY.
   - Muse's precedent is `musePromptGlyphs`, shared "so the two gates cannot disagree". Rule: one `qoderPromptCol` and one `qoderPromptGlyphs` authority, read by both gates.
4. **The `claudeComposerRule` skip in `orientationComposerActive` applies to every agent** (`orientation.go:190`), family `refactor-changes-sibling-agent-behavior`, ARCH-PURPOSE.
   - Only qoder needs it: without it `composer.raw` fails, because the hidden cursor parks on the closing rule at (2,18).
   - Measured with a claude box and the cursor on the closing rule: `orientationComposerActive` goes from false at base to true at head. No test pins claude, codex, agy or muse.
   - Rule: sibling behavior changes only through a per-profile field (the prompt column and rule tolerance), with a negative row per sibling.
5. **Qoder's create-path mint is unpinned** (`createflow.go:596`), family `agent-dispatch-registration-gap`, the 7th in this family.
   - `TestShouldMintSessionID` covers the predicate only. I reverted `rt.AgentSessionExists(agent, …)` to the literal `"claude"` and the launcher tests stayed green apart from the sandbox failures.
   - Rule: a site that branches on agent identity is either derived from a registry predicate or has a `runCreate` test per registry-true agent. The fake is already keyed `agent|sid`, so a wrong-agent probe is observable.
   - Test: `runCreate` for qoder with `agentSessions["qoder|MINTED-1"]=true` should mint `MINTED-2` and append `--session-id MINTED-2`.

## Minor findings
- `ttyFixtureExpectation` comment (`harness_tty_fixture_test.go`): says `overlay.raw` and `selection.raw` "both take the shared declining default", but `selection.raw` has an explicit `false` row and only `overlay.raw` is the default.
- The "qoder prose about enter select" overlay test feeds spaced text. The atlas says Qoder paints body rows glued, so prose containing "for future sessions" or "Enter select" could arm the flag through `forfuturesessions` / `Enterselect`. The test does not model this, and `forfuturesessions` is generic.
- `atlas/architecture.md:1203` still says "For claude … `--session-id` is deterministic". Qoder now pins too.
- The plan Goal still says v1.1.59 while the fixtures are 1.1.60.

## Test coverage notes
- Only the empty idle composer is captured. Multi-line drafts are synthetic, and the parked hidden cursor is a paint artifact rather than the caret.
- No test presses Return on any Qoder screen, which the ledger discloses honestly.
- The `--session-id` honoring has no durable live check. The unsandboxed live test is worth adding as a follow-up under ARCH-MOCK.

## Architectural notes for upcoming work
- M4 adds glyph consumers (`scrollback.lua`, `distill.go`). Take them from the single `qoderPromptGlyphs` authority (finding 3) rather than a fourth restatement.
- `pickerActive` re-arming is a temporal-state class. Consider one consume-and-reset entry point that clears all detector carryover (text tail and raw tail) so future detectors cannot forget it.

## Plan revision recommendations
- Add a Revisions entry: `orientation.go` (per-agent prompt column and glyph set) landed in M3 Task 9 and not in M4 Task 14. Task 14's `orientationPromptOK` map row is superseded. Also record the shared-loop rule skip and its sibling-behavior pin once finding 4 is fixed.
- Correct the Goal line to v1.1.60.

```findings
findings:
  - id: new
    severity: Critical
    family: overlay-flag-rearmed-from-stale-input
    title: |
      Qoder raw rolling-buffer scan re-arms pickerActive after the confirming Enter (ARCH-ORDER)
    detail: |
      Reproduced at head: paint overlay.raw or selection.raw, emitPlainCR consumes the flag, then one small chunk re-arms it from stale rolling bytes; the next composer Enter passes a bare CR and submits the draft. emitPlainCR clears only overlayTextTail; rolling is loop-local. Claude/codex OSC paths trim rolling at wrap.go:3115 and codex text never reads it. Codex has TestCheckOverlayOpen_CodexDoesNotRedetectStalePickerText (overlay_test.go:253); qoder has no counterpart. Rule: detector input that outlives flag consumption is reset at consumption. Fix by giving the raw window a proxy-owned tail cleared in emitPlainCR, or advance past the matched marker; add a paint, Enter, small-chunk test over both fixtures.
  - id: new
    severity: Important
    family: qoder-branch-copies-claude
    title: |
      qoderComposerActive re-implements ruledBoxComposerActive instead of adding a spec (ARCH-DRY)
    detail: |
      composer_recognizers.go duplicates the Cursor.Y+1 prompt scan and the ruledBoxBottomRule scan. The atlas paragraph edited in this diff says add a spec rather than a fourth near-copy, and plan Task 9 Step 2(b) orders the same. The real differences are promptCol and requireVisibleCursor spec fields. Rule: a ruled-box harness registers a ruledBoxComposerSpec and no other function owns that loop.
  - id: new
    severity: Important
    family: hand-restated-registry
    title: |
      Qoder prompt column 1 and glyph set >/* are restated in the recognizer and in orientation.go
    detail: |
      qoderComposerActive hard-codes promptCol=1 and ">"/"*"; orientationComposerActive and orientationPromptOK restate both. Muse's precedent is musePromptGlyphs, shared so the two gates cannot disagree. Rule: one qoderPromptCol and one qoderPromptGlyphs authority read by both gates; M4's scrollback/distill glyph consumers should derive from it too.
  - id: new
    severity: Important
    family: refactor-changes-sibling-agent-behavior
    title: |
      Shared orientationComposerActive now skips column-N rule cells for every agent; sibling behavior changes unpinned
    detail: |
      orientation.go:190 skips claudeComposerRule for all agents, but only qoder needs it (composer.raw's hidden cursor parks on the closing rule; without the skip the fixture fails). Measured with a claude box and cursor on the closing rule: orientationComposerActive goes false to true versus base. No test pins claude, codex, agy or muse. Rule: sibling behavior changes only through a per-profile field, with a negative row per sibling.
  - id: new
    severity: Important
    family: agent-dispatch-registration-gap
    title: |
      Qoder create-path mint (createflow.go:596 AgentSessionExists(agent, ...)) is pinned by no test
    detail: |
      Only the shouldMintSessionID predicate is tested. Reverting the probe to the literal "claude" leaves every launcher test green except sandbox failures. Rule: an agent-identity branch is either derived from a registry predicate or has a runCreate test per registry-true agent, using the agent-keyed fake so a hard-coded sibling is observable. Test: runCreate for qoder with agentSessions["qoder|MINTED-1"]=true expects MINTED-2 and --session-id MINTED-2.
  - id: new
    severity: Minor
    family: unbacked-existing-behavior-claim
    title: |
      ttyFixtureExpectation comment says selection.raw takes the shared default, but it has its own explicit row
    detail: |
      Only overlay.raw is the shared default (""); the qoder row lists selection.raw false explicitly. Also the qoder prose-does-not-open-overlay test feeds spaced text while Qoder paints body rows glued, so it does not model the real false-positive risk for glued markers like forfuturesessions and Enterselect.
  - id: new
    severity: Minor
    family: plan-prose-restates-diff
    title: |
      Plan and atlas lag M3: Task 14 still lists orientation.go, Goal cites 1.1.59, architecture.md:1203 says --session-id is claude-only
    detail: |
      orientation.go's qoder branch landed in M3 Task 9, so Task 14's orientationPromptOK map row is superseded and the M3 Revisions entry omits it. The plan Goal says v1.1.59 while fixtures are 1.1.60. atlas/architecture.md:1203 still says "For claude ... --session-id is deterministic" though qoder now pins too.
```

---

## Re-review — 2026-09-21T15:03:05-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 300 — integrate qoder harness into pair |
| repo | pair |
| issue file | workshop/issues/000300-integrate-qoder-harness-into-pair.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 301c53815433ccbc16f2cdf33b43ea4f3ded0e4b..4ae4ebb5ee327260c759140c17f474051cabcdd9 |
| command | sdlc milestone-close --issue 300 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-09-21T15:03:05-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The round-1 fixes for BR-28 through BR-32 are real. I reverted each in a scratch copy of HEAD and its regression test went red. The window is now free of Critical findings. Two Important findings remain, both cheap: the Qoder raw-window scan can still miss a marker split across chunks, and `atlas/architecture.md` still omits Qoder in three places. The rest is Minor. Package tests were green except sandbox-only failures (`operation not permitted` on `/tmp` mkdir and PTY child tests), which are environmental. `go vet` output was not inspected.

```findings
dispose:
  - id: BR-28
    disposition: addressed
    note: |
      overlayRawTail is proxy-owned, mutated only under overlayMu, and cleared in emitPlainCR beside overlayTextTail. Scratch revert (drop the `p.overlayRawTail = nil` line) turns TestCheckOverlayOpen_QoderDoesNotRedetectStalePickerText red on both overlay.raw and selection.raw. The consumption sweep is complete: overlayTextTail and overlayRawTail are cleared, and the claude/codex OSC paths advance `rolling` past the last match (wrap.go:3136).
  - id: BR-29
    disposition: addressed
    note: |
      qoderComposerActive is a ruledBoxComposerSpec registration (promptCol, requireVisibleCursor). ruledBoxComposerActive owns the only ruled-box loop; the callers are muse, claude, the agy orientation fallback and qoder. The differential rows are unchanged and green.
  - id: BR-30
    disposition: addressed
    note: |
      qoderPromptCol and qoderPromptGlyphs (composer_recognizers.go) are read by the recognizer spec, by orientationComposerActive (promptCol) and by orientationPromptOK. No second restatement of column 1 or of `>`/`*` remains in Go.
  - id: BR-31
    disposition: addressed
    note: |
      The rule-cell skip is gated to qoder. With the gate removed in a scratch copy, TestOrientationRuleCellToleranceStaysPerProfile goes red for claude, muse and agy, while the qoder positive row stays true.
  - id: BR-32
    disposition: addressed
    note: |
      TestRunLaunchForcedCreateQoderMintProbesQoderSessions uses the agent-keyed fake. With the probe reverted to the literal "claude" in a scratch copy, both subtests fail (MINTED-1 instead of MINTED-2; empty PAIR_SESSION_ID).
  - id: BR-33
    disposition: addressed
    note: |
      The ttyFixtureExpectation comment now says only overlay.raw takes the shared declining default. A glued-prose negative row for "forfuturesessions" was added to TestOverlayDetectorByAgent. The class residual is raised as a new Minor below.
  - id: BR-34
    disposition: addressed
    note: |
      Task 14 is amended, the Goal cites 1.1.60, and atlas/architecture.md:1203 now names the MintsSessionID set. The stale 1.1.59 fixture paths in the Core concepts table and Task 9 are covered by the M3 Revisions entry; the sweep gap is in the atlas finding below.
findings:
  - id: new
    severity: Important
    family: detector-carry-bounded-before-scan
    title: |
      Qoder raw window is truncated to 512 bytes before it is scanned, so it is not split-proof for chunks longer than about 500 bytes
    detail: |
      detectQoderOverlayOpen (wrap.go:925-930) appends the chunk, trims to the last rollingTailLen bytes, then scans. A marker that straddles a chunk boundary inside an escape is missed whenever the second chunk carries more than about 500 bytes after the split. Measured in a scratch copy: first chunk `...Enter\x1b[2`, second chunk `3mselect·Esccancel` plus filler. Armed=true with 0 and 100 bytes of filler, armed=false with 400, 600 and 2000. The atlas and plan claim the byte-contiguous window cannot be corrupted by a split. TestHarnessTTYFixtureConformance cannot see this, because both marker paints sit within about 200 bytes of the end of their fixtures, so every replayed split leaves a short second chunk. Fix: scan stripTerminalControls(prevTail+data) and only then bound the carry. Add a test that puts the split inside the marker's escape with at least 1 KB of trailing bytes. The composer gate still forces bare CR on the captured picker shapes, which is why this is Important rather than Critical.
  - id: new
    severity: Important
    family: hand-restated-registry
    title: |
      atlas/architecture.md still enumerates profiles without Qoder at :694 (keymaps), :702 (ruled-box sharing) and :704 (conformance expectation)
    detail: |
      This is the 5th finding in family `hand-restated-registry`, and the rule matters more than this instance. Rule: when a harness registers, grep the previous newest harness (`grep -n -i muse atlas/*.md`) and extend every hit that enumerates sibling harnesses in the same commit; log the sweep. Today :694 lists keymaps for Claude/Codex/Agy/Muse only (not `\`-CR/CR/Ctrl-U for Qoder). :702 says muse and claude "share one ruledBoxComposerActive ... prompt glyph at column 0", which is now false: Qoder is a third spec, with promptCol 1 and requireVisibleCursor false. :704 lists the composer.raw keymap expectation without Qoder. :700 and :1203 were edited in this same range, two lines away. Line 903 ("Claude, Codex, Agy, and Muse record parsing") is also stale from M2; sweep it too.
  - id: new
    severity: Minor
    family: refactor-changes-sibling-agent-behavior
    title: |
      requireVisibleCursor defaults permissive, and the agy orientation fallback's hidden-cursor decline is pinned by no test
    detail: |
      This is the 5th finding in family `refactor-changes-sibling-agent-behavior`. Rule: a spec field added for one harness must default to the prior behaviour of every existing spec, so invert it to `allowHiddenCursor` and only Qoder sets it. Then no sibling needs touching and a forgotten field cannot fail open. Measured: flipping requireVisibleCursor to false on claude reddens TestClaudeComposerActiveSnapshotDifferential, and on muse it reddens TestMuseComposerActiveSnapshotDifferential and TestMuseFixtureEvidence. Flipping it on agyUncoloredOrientationComposer (orientation.go:300) turns no test red. Add a hidden-cursor negative row for the agy uncolored path.
  - id: new
    severity: Minor
    family: overlay-marker-matches-agent-prose
    title: |
      "Permission Required" is ordinary English, which the atlas rule and the dropped forfuturesessions marker say a marker must not be
    detail: |
      BR-33 fixed the instance it named, not the class of markers that agent output can produce. "Permission Required" survives the strip with real spaces and appears in any transcript, tool output or source file that mentions the phrase (this repo's own atlas and wrap.go do). It arms pickerActive, and the next composer Enter then passes a bare CR and submits a draft. The code comment defends the header as the generic marker, which is a defensible tradeoff but the opposite of the rule stated two paragraphs later. Either require co-occurrence with a body marker, or amend the atlas rule to say the header is exempt and why, and pin a spaced-prose negative row.
  - id: new
    severity: Minor
    family: agent-dispatch-registration-gap
    title: |
      orientation.go branches on p.agentBasename == "qoder" at two sites plus orientationPromptOK, instead of a per-profile orientation field
    detail: |
      Behaviour is now pinned per sibling, so this is design only. A profile-carried promptCol and ruleCellTolerant would drop the string compares and make the M4 glyph consumers derive the same way.
  - id: new
    severity: Minor
    family: hand-restated-registry
    title: |
      Qoder adds the fourth copy of the overlay tail-carry block and a third identical marker-scan loop
    detail: |
      ARCH-DRY. detectQoderOverlayText is identical to detectAgyOverlayText and detectCodexOverlayText apart from the marker slice, and the `visible = p.overlayTextTail + visible; p.overlayTextTail = textSuffix(...)` block now sits at wrap.go:792, 828, 865 and 935. A shared `firstMarker(visible, markers)` and a `p.overlayVisible(data)` helper would collapse them.
```

**Summary.** BR-28, the Critical re-arm, is fixed correctly and pinned over both frozen captures. BR-29 through BR-32 hold under mutation. What keeps this at FIX-THEN-SHIP is that the raw-window mechanism introduced to make detection split-proof is bounded before it scans (Important), and that the architecture atlas still restates per-harness facts without Qoder (Important, docs gate).

**Strengths**
- `overlayRawTail` shares one lifetime rule with `overlayTextTail` under `overlayMu`. The test drives paint, Enter and a small chunk over both fixtures (`picker_overlay_test.go:71-108`).
- `ruledBoxComposerSpec` absorbed Qoder as two orthogonal fields, and the comment says the loop is owned in one place. The existing claude and muse hidden-cursor rows do fail when their `requireVisibleCursor` flips.
- `TestOrientationRuleCellToleranceStaysPerProfile` has a negative row per sibling plus a qoder positive contrast. The claude row reproduces the measured regression.
- The create-path mint test uses an agent-keyed fake, so a hard-coded probe is observable. `TestShouldMintSessionID` ranges `AgentInventory()`, so a sixth agent must declare which side of the mint set it is on.
- `trimmedHarnessTTYCapture` verifies the trim by replaying the Return decision, and it is pinned by `TestTrimmedHarnessTTYCapture`.

**Critical findings.** None.

**Architecture pass**
- **ARCH-DRY:** BR-29 resolved; the marker-scan and tail-carry copies are flagged as Minor.
- **ARCH-PURE:** the recognizer is a pure function over a snapshot. The detector mutates proxy state under the mutex, which matches the existing pattern. Pass.
- **ARCH-PURPOSE:** the consumption-reset sweep is complete. The prose-marker class is only partly swept and is flagged as Minor.
- **ARCH-MOCK:** the fake is agent-keyed, and live conformance is manual and opt-in, as the atlas states. Pass.
- **ARCH-CONSTRAINTS:** the per-chunk cost is two bounded strips of at most 512 bytes. The truncate-before-scan finding is the flip side of that bound.
- **ARCH-SECURE:** the detector inputs are bounded and there are no credentials in the fixtures. `menu.raw` embeds a tilde-relative cwd, which the machine-neutral oracle accepts. Pass.
- **ARCH-ORDER:** the flag lifecycle is atomic, but the all-splits oracle only exercises two-chunk splits with a short second chunk (flagged as Important). The known "dismiss without Enter" edge is pre-existing.
- **ARCH-FUNERAL:** the raw tail is in-memory and capped at 512 bytes, and the fixture set is the newest version directory only. Pass.

**Test coverage notes**
- `plainCR` `\`+CR, `altCR` and `altBS` for Qoder are inferred from Claude. They are recorded honestly in `ttyFixtureReactionGaps`, and no Return has been driven on any Qoder screen. M5 smoke must press Return in the composer and in both pickers and then retire that entry.
- `TestFreshPinAgentsMintSessionIDButKeepRecoveryProvisional` hand-lists claude and qoder rather than ranging `MintsSessionID`. This is minor, because `TestShouldMintSessionID` already ranges the inventory.

**Architectural notes for M4 and M5**
- `scrollback.lua` cannot import `qoderPromptGlyphs`. Add a Go test that parses the Lua character class against the map so the two cannot drift.
- Decide whether the Task 14 distill glyph derives from the same authority.

**Plan revision recommendations**
- Add a `## Revisions` entry that says the Core concepts table row and the Task 9 `Create:` paths (`qoder/1.1.59/`) are superseded by `qoder/1.1.60/`.
- After the truncate-before-scan fix lands, amend the M3 execution-deltas entry so the "split-proof" claim is stated with its window semantics.
- Record the atlas sweep grep and its hit list in the issue Log.

---

## Re-review — 2026-09-21T15:25:08-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 300 — integrate qoder harness into pair |
| repo | pair |
| issue file | workshop/issues/000300-integrate-qoder-harness-into-pair.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 301c53815433ccbc16f2cdf33b43ea4f3ded0e4b..d2a1e81f4f2729c86939f5b12c097a173c16d3f5 |
| command | sdlc milestone-close --issue 300 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-09-21T15:25:08-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

All six open findings (BR-35..BR-40) are fixed at head `d2a1e81f`. I checked each against the pinned tree and mutation-tested the behavioural ones in a scratch copy of HEAD. I found one new finding, a Minor. It sits in code this window did not touch, and it is the same bound-before-scan class as BR-35. The Qoder-related tests (`Qoder|OrientationRuleCell|OrientationUncoloredAgy|OverlayDetectorByAgent|ComposerActive|TTYFixture|Adversarial`) pass. A full `go test` of `wrapcmd` and `launcher` in this sandbox shows 18 failures, all `mkdir /tmp/...: operation not permitted`, none from a test assertion. The issue Log records a full unsandboxed `go test ./...` at EXIT=0. I did not reproduce that run.

## Strengths
- **BR-35 regression pin** (`picker_overlay_test.go:107-129`). The split lands inside `\x1b[23m`, so the marker exists only in the concatenation. With the scan/bound order reverted in scratch, the 600 and 2000 filler rows go red and the 0-byte row stays green as the control. That is real fail-without-fix evidence.
- **BR-37 default inversion** (`composer_recognizers.go:141-145`). Only Qoder sets `allowHiddenCursor`, so a forgotten field fails closed. Setting `allowHiddenCursor: true` on the agy uncolored spec in scratch reddens `TestOrientationUncoloredAgyRequiresVisibleCursor/hidden_cursor_inside_the_composer`. That row had no coverage before, and its visible-cursor control row shows the decline comes from the cursor rather than the box shape.
- **BR-39 profile fields.** `orientationPromptCol` and `orientationRuleCellTolerant` (`harness_tty.go`) replace the `agentBasename == "qoder"` compares in the scan. `TestOrientationRuleCellToleranceStaysPerProfile` pins the sibling negatives and the Qoder positive.
- **BR-40 helpers.** `overlayVisible` and `firstMarker` collapse the four carry blocks and three loops. `overlayVisible` returns the full concatenation and bounds only the carry, so the visible path is split-proof too. It is nil-receiver safe, and Muse keeps its own folded loop with a stated reason.
- **Lock discipline.** The raw tail is touched only under `overlayMu`. The detector runs through `detectOverlayOpen` (`wrap.go:1889`), and `emitPlainCR` clears it under the same lock (`wrap.go:2001-2005`).

## Critical findings
None.

## Important findings
None. BR-35 and BR-36 are both closed.

## Minor findings
- The shared chunk pump has the BR-35 flaw, in code this window did not touch; details are in the findings block below.
- `overlayTextTail` and `overlayRawTail` must be cleared together, and that happens at a single site (`wrap.go:2004-2005`). A `resetOverlayCarry()` method would keep a third carry from being forgotten (ARCH-ORDER, cosmetic).
- `p.overlayRawTail = haystack[len-512:]` keeps the whole chunk-sized backing array alive. `append([]byte(nil), ...)` on the slice would release it. The retention is bounded to one chunk.

## Architecture pass
- ARCH-DRY: pass. BR-29 and BR-40 consolidated the ruled-box loop, the carry block and the marker loop. The remaining `orientationPromptOK` agent-keyed glyph read is a documented choice, since the glyph maps are the shared authority.
- ARCH-PURE: pass. `ruledBoxComposerActive` and `firstMarker` are pure and tested directly. The raw-tail mutation is confined to the locked detector seam.
- ARCH-PURPOSE: one flag, the new Minor. The Qoder instance is fixed, but the shared pump's `rolling` still has the same class of flaw.
- ARCH-MOCK: pass. The tests run the recognizer and detector against frozen live captures via `harnessSessionFake`, on the same seam production uses.
- ARCH-CONSTRAINTS: pass. Carries are capped at `rollingTailLen`, and the scan cost is linear in one chunk plus 512 bytes.
- ARCH-SECURE: pass. Untrusted agent output arms `pickerActive` only through exact markers. BR-38's exposure (the exact string `Permission Required` in agent prose) is now a documented, bounded exemption rather than an accident.
- ARCH-ORDER: pass. `pickerActive` is consumed only by `emitPlainCR`, and the invariant that both tails are cleared on consumption is pinned by `TestCheckOverlayOpen_QoderDoesNotRedetectStalePickerText` over both frozen captures.
- ARCH-FUNERAL: pass. The carries are in-memory, capped at 512 bytes and cleared on consumption. No durable artifact is added.

## Test coverage notes
- `TestOverlayDetectorByAgent`'s `qoder spaced prose about permissions does not open overlay` row pins only "other words, other case, non-contiguous" in one sentence. The test comment and atlas claim three separate boundaries, but one row covers them. This is acceptable for the exemption chosen, and the exact-string exposure is accepted and documented.
- I did not mutation-test the BR-39 profile fields in scratch. The Log says zeroing either one reddens a named row, and both rows exist at HEAD.

## Architectural notes for upcoming work
- M4's `scrollback.lua` `PROMPT_PATTERN_BY_AGENT` and `distill.go` should derive their glyph from `qoderPromptGlyphs`, as Task 14 already says. Lua cannot import the Go map, so the plan should name how that consumer stays in sync (a generated table or a parity test).

## Plan revision recommendations
None required; the plan's `## Revisions` entry for round 2 matches the code. Two items are owned by M5 Task 19 and must not be lost:
- `atlas/couch.md:321` still reads "selects claude, codex, agy or muse".
- `README.md:122` (Return row) and the `README.md:39/241/291/566` rosters omit Qoder's `\<CR>` remap and Qoder itself.

```findings
dispose:
  - id: BR-35
    disposition: addressed
    note: |
      wrap.go:940-946 scans stripTerminalControls(carry+data) before bounding; TestCheckOverlayOpen_QoderSplitFooterSurvivesLongSecondChunk reddens at filler=600/2000 when the order is reverted in scratch.
  - id: BR-36
    disposition: addressed
    note: |
      architecture.md :694 (Qoder keymap), :702 (three specs, promptCol/allowHiddenCursor), :704 (conformance) and :903 now name Qoder; atlas couch.md:321 is explicitly M5 Task 19's.
  - id: BR-37
    disposition: addressed
    note: |
      Field inverted to allowHiddenCursor (only Qoder sets it); TestOrientationUncoloredAgyRequiresVisibleCursor reddens when the agy uncolored spec allows a hidden cursor.
  - id: BR-38
    disposition: addressed
    note: |
      Atlas amendment option taken: how-to line 87 states the header exemption and its bound (one Enter consumes pickerActive); spaced-prose negative row added in overlay_test.go.
  - id: BR-39
    disposition: addressed
    note: |
      orientationPromptCol and orientationRuleCellTolerant now live on harnessTTYProfile; orientationComposerActive has no agentBasename compare left. orientationPromptOK keeps its agent-keyed glyph read on purpose.
  - id: BR-40
    disposition: addressed
    note: |
      overlayVisible and firstMarker replace the four carry blocks and three loops; Muse keeps its own folded loop for the reason stated at wrap.go:872.
findings:
  - id: new
    severity: Minor
    family: detector-carry-bounded-before-scan
    title: |
      Shared chunk pump trims rolling to 512 bytes before checkOverlayOpen, so Claude/Codex OSC detectors miss an OSC followed by 512+ bytes in one chunk
    detail: |
      Pre-existing and outside this window; it is the class BR-35 belongs to, and the Qoder instance was fixed while the siblings were not. wrap.go:3122-3127 appends to rolling, trims it to rollingTailLen, and only then calls checkOverlayOpen(data, *rolling) and the oscRe scan. detectClaudeOverlayOpen and detectCodexQuestionOSC read only rolling, so an OSC 777 or OSC 9 with more than 512 bytes after it in the same read is dropped before the scan. The rule is that any carry must be scanned at full carry+chunk length and bounded afterwards; here that means moving the trim below the two scans. Not shown to bite in practice (those OSCs usually arrive alone), so no gate blocks on it.
```
