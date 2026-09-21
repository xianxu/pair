# Boundary Review — pair#300 (milestone M4)

| field | value |
|-------|-------|
| issue | 300 — integrate qoder harness into pair |
| repo | pair |
| issue file | workshop/issues/000300-integrate-qoder-harness-into-pair.md |
| boundary | milestone M4 |
| milestone | M4 |
| window | 04ff3b813f9dd5d8bfefbbfbaa3ff711ee241b22..879f8c664af3bb1f3bb850c06669b2f9a1ab771b |
| command | sdlc milestone-close --issue 300 --milestone M4 |
| reviewer | claude |
| timestamp | 2026-09-21T16:37:47-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M4 delivers what its Plan rows claim. `runQoder` and `DefaultQoderModel` are pinned by a fake-binary dispatch test plus a gated live conformance test. The Lua glyph consumer gets a derived-pattern parity test, which is the sync story M3 owed. The settings surface is recorded in the atlas, and the shared pump is fixed to scan before it bounds the carry. What keeps it from a clean SHIP:

- **Distill glyph row:** the `promptGlyphChar["qoder"] = " >"` row is unguarded, and it breaks one of the two readers of that map.
- **Headless `qoder -p`:** each call persists a session transcript, and I measured this on disk.
- **Carried item:** the footer-trim item is owned by a milestone task that doesn't list it.

I disposed of nothing: the prior findings are all already disposed, and I did not re-raise any of them.

## 1. Strengths
- `cmd/internal/wrapcmd/scrollback_glyph_parity_test.go` derives the expected Lua row from `qoderPromptGlyphs` and `qoderPromptCol` and requires exactly one qoder row in `scrollback.lua`. `scrollback_test.lua` block 4 pins the pattern's behavior with col-1 and flush-left rows. Together these answer M3's "name the sync story" note.
- `TestRunQoderDispatchesToQoderCLI` (`model_test.go`) pins argv, stdin, and the TempDir cwd through the production `Run` dispatch. `TestRunQoderLiveConformance` covers what the fake can't (ARCH-MOCK: seam plus gated live check).
- `TestHandleChunk_OscScannedBeforeCarryIsBounded` is red under the old bound-first order: 640 bytes of filler leaves only filler in a 512-byte carry. The pump side-quest is logged in the issue (line 247), and the pump's carry stays bounded on every path.
- The `../ariadne` trust decision is explicitly left open rather than invented.

## 2. Critical findings
None.

## 3. Important findings

**A. `trimLiveTail` never recognises qoder's bare input box (`distill.go:70`, `:28`).** This is the 9th finding in family `agent-dispatch-registration-gap`.
- `isFooterChrome` tests `t == glyph` where `t := strings.TrimSpace(line)`. The registered qoder value is `" >"`, so `">" == " >"` is never true.
- The map's own comment calls it "the SINGLE source for both the line-start regex and the empty-input-box detection". Only the first reader was tested (`TestScanTurnBoundaries`).
- **Rule that covers the family:** a registry row is not landed until a table test ranges the registry and drives every reader of it.
- **Fix:** add a test ranging `promptGlyphChar` over both `scanTurnBoundaries` and `trimLiveTail` with each agent's bare-box line. Normalise the glyph once (e.g. compare against `strings.TrimSpace(glyph)` in `trimLiveTail`, keep the space only for the regex).

**B. The distill qoder glyph is a hand-restated copy with no parity guard (`distill.go:28`).** This is the 7th finding in family `hand-restated-registry`.
- Plan Task 14 says "any M4 consumer derives from [`qoderPromptGlyphs`]". The Lua consumer got a parity test; distill got only a comment.
- Its only test compares the literal `" >"` against itself, so changing `qoderPromptCol` reddens nothing.
- The Lua row includes the yolo `*`, while distill drops it on the same "default-mode evidence only" basis. That is two decisions on one registry with no shared statement.
- **Rule that covers the family:** enumerate every consumer of `qoderPromptGlyphs` and `qoderPromptCol`, and require each to be derived or parity-pinned.
  - `qoderComposerActive` (recognizer): derived.
  - `orientationPromptOK`: derived.
  - `scrollback.lua`: parity-pinned.
  - `distill.go`: unguarded.
- **Fix:** export a small accessor from `changelogcmd`, e.g. `PromptGlyph(agent)`. `wrapcmd` can import `changelogcmd` without a cycle, and a test can assert `strings.Repeat(" ", qoderPromptCol)+">"` equals it. Pin the `*` omission in the same test.

**C. Every `qoder -p` slug/changelog call leaves a persisted transcript (`model.go:149`).** ARCH-FUNERAL; new family `headless-call-leaves-durable-residue`.
- I measured this: `~/.qoder/projects/-private-tmp/` holds four 8–9 KB `*.jsonl` files plus per-session dirs, and `-private-var-folders-…-T/` holds another. Both are from the A/B and live-conformance runs.
- `qoder --help` lists `--no-session-persistence`. The slug fires at turn end, so residue grows per turn with no owner and no sweep.
- **Fix:** add `--no-session-persistence` to `runQoder`'s argv and update `wantArgs`. Confirm the flag composes with `-p` in the live conformance test.
- `runClaude` and `runAgy` may share the class, but they are outside this window.

**D. The carried footer-trim item is not in Task 17's steps.** New family `deferred-work-not-in-executing-task`.
- The Revisions entry (plan line 872) says qoder's live footer matches none of the `isFooterChrome` rows, so `trimLiveTail` strips nothing, and calls that "now an M5 Task 17 scope item".
- Task 17's Steps 1–4 (plan lines 736–739) contain no Alt+l/distill check. The M5 checklist won't exercise a known #58-class degradation: anchor leak, so `FullRedistill` on every press.
- The registration ships this state now, since qoder is launchable from M1–M3 onward.
- **Fix:** add a Task 17 step to capture the settled-session footer, extend `isFooterChrome`, and verify a no-op Alt+l press.

## 4. Minor findings
- **E** (`unbacked-existing-behavior-claim`, `scrollback_glyph_parity_test.go:46`):
  - The Lua row is used by `vim.fn.search`, a Vim regex. The escape set is Lua-pattern dialect (`%]`, `%-`, `%^`) while the test comment claims it keeps the derivation total.
  - A future `-` or `]` glyph would derive a wrong class that also matches `%`, and the test would then demand it. It is dead code for `>` and `*` today. Use Vim collection escapes (`\]`, `\-`, `\^`, `\\`).
- **F** (ARCH-SECURE, new family `permission-allowlist-scope`):
  - `Bash(git:*)`, `Bash(make:*)` and `Bash(zellij:*)` are registered at user scope in `~/.qoder/settings.json`, so they apply in every repo qoder opens.
  - The claude precedent is project-scoped and narrow (`~/.claude/settings.json` has no entries; `.claude/settings.json` has verb-specific rules such as `Bash(git checkout *)`). Prefer `<repo>/.qoder/settings.local.json`.
  - The A/B ran only five simple probes. It did not check that a chained command like `git status && …` falls outside the prefix rule.
- **H** (`refactor-changes-sibling-agent-behavior`, 6th):
  - The pump change also feeds Codex's `detectCodexQuestionOSC(rolling)` and the OSC telemetry loop, but only Claude has a regression row.
  - Table-drive the pump test over every profile with an OSC overlay detector.
- The parity test reads `nvim/scrollback.lua` via `../../..`. That is fine under `go test`, but a missing file fails with `read scrollback.lua`, not a drift message.

## 5. Test coverage notes
The gaps are the ones in A, B, E and H. `go test` and `make test-lua` were green per the Log. I did not re-run the suite or mutation checks. Beyond that, the live-only claims (Alt+b on real qoder echo; `*` at col-1 possibly matching bullets) rest on M5 Task 17.

## 6. Architectural notes and the eight principles
- **ARCH-DRY: flag (B).**
- **ARCH-PURE: pass.** Distill glyph and pattern derivation are pure; the parity test's one file read is test-side only.
- **ARCH-PURPOSE: flag (B, D).** The shadow-sweep of `qoderPromptGlyphs` consumers leaves distill open.
- **ARCH-MOCK: pass.** Fake binary plus gated live check.
- **ARCH-CONSTRAINTS: pass.** The pump now scans the whole chunk instead of ≤512 bytes; that is linear and small next to the terminal model work on the same bytes, and no unmeasured performance claim is made. A 30 s timeout against ~4.6 s measured is fine.
- **ARCH-SECURE: minor (F).**
- **ARCH-ORDER: pass.** Carry ordering is preserved; sequence coverage for the pump is thin (H).
- **ARCH-FUNERAL: flag (C).**

## 7. Plan revision recommendations
- **Task 17:** add a step for the deferred footer-trim/Alt+l item (D), and record in Task 14's Revisions how distill's row is derived once B lands.
- **Task 15:** state that the allowlist was written to user scope, and why (F).

```findings
findings:
  - id: new
    severity: Important
    family: agent-dispatch-registration-gap
    title: |
      trimLiveTail's empty-box check `t == glyph` can never match qoder's space-prefixed " >" glyph
    detail: |
      distill.go:70 compares TrimSpace(line) to the raw glyph, so ">" never equals " >". The registry comment calls promptGlyphChar the single source for both the turn-boundary regex and the empty-box detection, but only the regex reader is tested. Rule: a registry row is not landed until a table test ranges the registry and drives every reader. Fix: range promptGlyphChar over scanTurnBoundaries and trimLiveTail, and normalise the glyph once.
  - id: new
    severity: Important
    family: hand-restated-registry
    title: |
      distill's qoder glyph " >" restates qoderPromptGlyphs/qoderPromptCol by hand with no parity guard
    detail: |
      Plan Task 14 requires every M4 consumer to derive from the authority. The Lua consumer got a parity test; distill.go:28 got a comment, and its own test only compares the literal to itself, so changing qoderPromptCol reddens nothing. Distill also drops yolo `*` while the Lua row keeps it, on the same evidence. Rule: enumerate consumers of qoderPromptGlyphs/Col (recognizer, orientation, Lua, distill) and require each derived or parity-pinned. Fix: export a changelogcmd accessor and assert it in a wrapcmd parity test, pinning the `*` omission.
  - id: new
    severity: Important
    family: headless-call-leaves-durable-residue
    title: |
      runQoder persists a transcript per slug/changelog call; --no-session-persistence is not passed
    detail: |
      ARCH-FUNERAL. Measured under ~/.qoder/projects/-private-tmp and the TMPDIR project dir: each `qoder -p` leaves an ~9KB jsonl plus a session dir, and `qoder --help` lists --no-session-persistence. The slug fires at turn end, so growth is per turn with no sweep. Fix: add the flag to runQoder's argv, update wantArgs, and confirm it composes with -p in the live conformance test.
  - id: new
    severity: Important
    family: deferred-work-not-in-executing-task
    title: |
      The carried qoder footer-trim item lives only in a Revisions paragraph; Task 17's steps omit it
    detail: |
      The plan Revisions entry (plan line 872) says qoder's live footer matches no isFooterChrome row, so trimLiveTail strips nothing and Alt+l anchors on volatile chrome (the #58 FullRedistill class), and it calls this "now an M5 Task 17 scope item". Task 17 Steps 1-4 (plan lines 736-739) have no Alt+l/distill step, so the M5 checklist will not exercise it although the registration ships the degraded state now. Fix: add a Task 17 step to capture the settled footer, extend isFooterChrome, and verify a no-op press.
  - id: new
    severity: Minor
    family: unbacked-existing-behavior-claim
    title: |
      Parity test's class escape uses Lua-pattern dialect but scrollback.lua patterns are Vim regex
    detail: |
      scrollback_glyph_parity_test.go:46 escapes with %], %-, %^ while PROMPT_PATTERN_BY_AGENT is consumed by vim.fn.search. The comment claims the derivation stays total for any future glyph; a `-` or `]` glyph would derive a wrong class (matching `%` too) and the test would then require it. Dead for `>` and `*` today. Use Vim collection escapes.
  - id: new
    severity: Minor
    family: permission-allowlist-scope
    title: |
      Standard-set allow rules registered at user scope, broader than claude's project-scoped precedent
    detail: |
      ARCH-SECURE. Bash(git:*), Bash(make:*), Bash(zellij:*) in ~/.qoder/settings.json apply in every repo qoder opens, while claude's equivalents are project-scoped and verb-specific. The A/B ran five simple probes and did not confirm chained commands such as `git status && ...` fall outside the prefix rule. Prefer <repo>/.qoder/settings.local.json or verify the chained-command behavior.
  - id: new
    severity: Minor
    family: refactor-changes-sibling-agent-behavior
    title: |
      Pump scan-before-bound change is pinned for Claude only; Codex's OSC detector and the OSC telemetry loop share the path
    detail: |
      TestHandleChunk_OscScannedBeforeCarryIsBounded covers Claude's picker OSC. detectCodexQuestionOSC(rolling) and the OSC telemetry loop now also see the unbounded carry+chunk with no regression row. Rule: a shared-path change is pinned over every profile that reads it. Fix: table-drive the test over each profile with an OSC overlay detector.
```

---

## Re-review — 2026-09-21T16:49:31-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 300 — integrate qoder harness into pair |
| repo | pair |
| issue file | workshop/issues/000300-integrate-qoder-harness-into-pair.md |
| boundary | milestone M4 |
| milestone | M4 |
| window | 04ff3b813f9dd5d8bfefbbfbaa3ff711ee241b22..a08e396d36ebd32667d213f1923355e31444e553 |
| command | sdlc milestone-close --issue 300 --milestone M4 |
| reviewer | claude |
| timestamp | 2026-09-21T16:49:31-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

All seven open findings (BR-42 to BR-48) are addressed, and each fix's regression test goes red when the fix is reverted. I found nothing Critical or Important. One Minor is new: the same residue class BR-44 fixed for qoder is still open in `runClaude` and `runMuse`, which this diff did not touch.

I checked the fixes this way:
- **Setup:** I ran the tests in a scratch copy of HEAD with the runtime assets copied in.
- **Mutation checks:** For each fix I reverted it in scratch and confirmed the named test fails.
- **Live run:** I ran `TestRunQoderLiveConformance` against the installed qoder CLI. It passed in 15.58s and left no new files under `~/.qoder/projects`.
- **Lua test:** `nvim/scrollback_test.lua` passes, including the new qoder block.

## Architecture
- **ARCH-DRY:** Pass. `promptGlyphChar` now feeds the boundary regex, `trimLiveTail` (normalised once) and the exported `PromptGlyph` accessor. The Lua row is derived and parity-tested.
- **ARCH-PURE:** Pass. `trimLiveTail` and `scanTurnBoundaries` are tested with no IO. The wrapcmd parity tests read only the Lua file, which is the parity fixture.
- **ARCH-PURPOSE:** Pass. The registry consumers (recognizer, orientation, Lua, distill) each derive or are parity-pinned, and `*` is pinned as a deliberate omission. The residue class outside qoder is the Minor below.
- **ARCH-MOCK:** Pass. The fake `qoder` on PATH captures argv, stdin and cwd, and the gated live conformance test passes against the real CLI.
- **ARCH-CONSTRAINTS:** Pass. The `rolling` carry is scanned first and bounded to 512 bytes afterwards. Only the two OSC readers consume `rolling`; the text detectors use their own bounded proxy-owned tails.
- **ARCH-SECURE:** Pass. Prompt goes as an argv element. The allowlist is repo-local and the chained-command probe was measured as denied. No secrets are involved.
- **ARCH-ORDER:** Pass. The pump's carry is the only state between events, and the tests drive the production `handleChunk`. Both OSC profiles fail if scan-then-bound is reverted.
- **ARCH-FUNERAL:** Pass for qoder (`--no-session-persistence`, verified live). The Minor below covers the siblings.

## Strengths
- `TestPromptGlyphRowsDriveBothReaders` (`distill_test.go`) ranges the registry over both readers. It goes red for qoder alone if `TrimSpace` is removed, so it enforces the rule rather than one row.
- The parity tests derive the expected value from `qoderPromptCol` and `qoderPromptGlyphs`. Changing the column to 0 reddens both `TestScrollbackQoderPatternTracksPromptAuthority` and `TestDistillQoderGlyphTracksPromptAuthority`.
- `TestHandleChunk_OscScannedBeforeCarryIsBounded` is table-driven over the OSC-reading profiles (claude 777, codex 9), and both rows fail under the old bound-first order.
- The BR-47 disposition is backed by measurement in the plan (repo-bound scope, chained command denied), not just a scope move.

## Test coverage notes
- `--no-session-persistence` is pinned in `wantArgs`, and the live test confirms it composes with `-p`.
- Not covered: the `Alt+l` footer shape for qoder. That is deliberately carried to Task 17 Step 3 in the plan.

## Plan revision recommendations
None. The Revisions entry for round 10 and Task 17 Step 3 match the code.

```findings
dispose:
  - id: BR-42
    disposition: addressed
    note: |
      trimLiveTail now trims the glyph once; TestPromptGlyphRowsDriveBothReaders ranges the registry over both readers. Reverting the TrimSpace makes the qoder row fail ("trimLiveTail leaves the bare input box").
  - id: BR-43
    disposition: addressed
    note: |
      changelogcmd.PromptGlyph plus TestDistillQoderGlyphTracksPromptAuthority derive from qoderPromptCol and pin the * omission. Setting qoderPromptCol to 0 reddens both parity tests.
  - id: BR-44
    disposition: addressed
    note: |
      runQoder passes --no-session-persistence and wantArgs pins it; reverting it fails TestRunQoderDispatchesToQoderCLI. The live conformance run passed and left no new files under ~/.qoder/projects.
  - id: BR-45
    disposition: addressed
    note: |
      Plan Task 17 Step 3 now owns the settled-footer capture, the isFooterChrome extension and the no-op Alt+l check.
  - id: BR-46
    disposition: addressed
    note: |
      The parity test escapes in the Vim dialect (backslash, ], -, ^), which is what vim.fn.search consumes; the sorted class still derives to ^ [*>] and matches the Lua row.
  - id: BR-47
    disposition: addressed
    note: |
      The allowlist moved to the repo-local .qoder/settings.local.json with measured repo-bound scope and a measured chained-command denial; user scope is restored.
  - id: BR-48
    disposition: addressed
    note: |
      The OSC pump test is table-driven over claude and codex; both rows fail when the bound-first order is restored in scratch.
findings:
  - id: new
    severity: Minor
    family: headless-call-leaves-durable-residue
    title: |
      runClaude and runMuse still persist a transcript per headless call, the class BR-44 fixed only for qoder
    detail: |
      This is the 3rd finding in family `headless-call-leaves-durable-residue`. The rule: every headless runner in model.go must pass its agent's no-persistence flag or carry a comment naming why it cannot, pinned by one table test over the agents Run dispatches. `claude --help` lists `--no-session-persistence` and `muse exec --help` lists `--no-session-log`; `codex exec` already passes `--ephemeral`. `~/.claude/projects` holds residue project dirs from `-private-tmp` cwds. The gap predates this diff and is outside qoder's scope, so it does not block M4. Log it as a follow-up issue rather than widening #300.
```
