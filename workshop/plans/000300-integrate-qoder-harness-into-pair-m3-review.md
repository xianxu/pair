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
