# Boundary Review — pair#410 (whole-issue close)

| field | value |
|-------|-------|
| issue | 410 — Grok harness integration for pair and couch |
| repo | pair |
| issue file | workshop/issues/000410-grok-harness-integration-for-pair-and-couch.md |
| boundary | whole-issue close |
| milestone | — |
| window | f904c1172c6bb4e2093bfef4e494e0dc0c6c26bb..f1e88e07ad1012188ebcd65cc00ee589433d59cc |
| command | sdlc close --issue 410 |
| reviewer | claude |
| timestamp | 2026-10-08T11:33:24-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The code is in good shape. BR-6 is fixed for the whole class, and the M2 additions are each tested at the level of real logic: the slug through `--prompt-file` with confined cleanup, the bell-to-attention translation, the per-agent interrupt byte, the minimal-mode prompt glyph and the context meter. One thing should change before the close is recorded. The issue's `## Done when` still promises several things that were not delivered or not shown, and the operator's decision to close without them exists only as Log prose. There is no `## Revisions` entry that changes the contract. Fixing that is a text-only change.

**Strengths**
- **`--` boundary (BR-6).** `resumeform.FlagRegion` (`cmd/internal/resumeform/resumeform.go:146-157`) is now the one rule for where flags end. `hasFlag`, `stripValuelessFlag`, `stripFlagAllForms`, `Strip`, `Extract`, `HasSessionID` and `insertBeforeDoubleDash` all go through it. `TestStripHelpersStopAtDoubleDash` (`agentargs_test.go:256-280`) now checks the readers too: `shouldMintSessionID` with prompt text after `--`, for claude, qoder and grok.
- **Slug cleanup (`runGrok` / `removeGrokSessionResidue`, `model.go`).** The deletion target is built only from the temp directory pair just created, never from grok's output. It must sit directly under the sessions root, and symlinks are refused. `TestRemoveGrokSessionResidueIsConfined` and the fake-binary dispatch test cover this, and a live conformance test backs the shape of the real binary.
- **Bell translation (`notification_rewriter.go`).** A bell becomes an attention notification only when the `outputBoundary` parser is in ground state, so a BEL that ends an OSC or sits inside a DCS/APC string is left alone. `TestNotificationRewriterBellAttentionEverySplit` checks every chunk split, which covers the ordering risk.
- **Interrupt key (`nvim/interrupt.lua`).** It is a pure module with a headless test, and it re-reads the agent file at keypress so it follows a switch-agent. It is also added to the runtime-bundle manifest (`artifactpath/manifest.go:551,1085`).
- **One glyph source.** `promptGlyphAuthorities` (`orientation.go`) replaces the per-agent `if` chain, and scrollback, distill and orientation are all held to the same authority by parity tests.

**Critical**
- None.

**Important**
- **This is the 2nd finding in family `plan-code-drift`.** `workshop/issues/000410-…md` `## Done when`, the smoke bullet. It still requires:
  - switch-agent to Grok auto-submits orientation (not met; filed as #414);
  - the thread parks and cold-resumes under Couch (the Log reports switched and live, but not park/cold-resume);
  - `doctor/doctor.sh` shows grok `return-remap`, `session-id` and `slug-parse` firing. The Log's round-1 doctor run found no `adapt-grok.jsonl`, said "the next smoke should rerun", and no rerun is logged.

  The rule behind this family: any deviation from a committed contract (Done-when, plan design or plan Durable state) gets a `## Revisions` entry when the decision is made. A Log note is not enough. The fix is to sweep every Done-when item and every plan design claim against what shipped, in one pass, rather than fixing only these:
  - Append a Revisions entry to the issue that moves the orientation item to #414 and the park/cold-resume item to verified or #414.
  - Either log a live doctor rerun or move that item out of scope.
  - Add to the plan's Revisions that cleanup refusals now go to stderr, not the planned adapt `slug-parse` near-miss (plan Durable state).
  - Record the M2-added surfaces that no Revisions entry covers: `bellAttentionHarnesses` and `nvim/interrupt.lua`.

**Minor**
- **ARCH-DRY: the grok record envelope is stated three ways.**
  - The scanner (`scan_grok.go` `applyGrokRecord`) accepts any non-empty `method`.
  - `normalizeGrokEvent` accepts `session/update` and `_x.ai/session/update`.
  - `ParseTokenUsage` accepts only `session/update`.

  These should share one accepted-methods set.
- **ARCH-DRY:** `nvim/interrupt.lua` `current_agent` repeats the agent-file read that `pair_read_saved_config` does in `nvim/init.lua:3099-3103`. The init.lua copy could call `PairInterrupt.current_agent`, or the read could move into a shared helper.
- The plan's Integration-points row for `runGrok` (plan:76) still says `grok -p <prompt>` with `cmd.Dir = os.TempDir()`. It is superseded only by a later Revisions entry, which is allowed but easy to misread.

**Test coverage notes**
- The bell, interrupt, residue-confinement, `FlagRegion` reader/editor and context-meter paths all have tests that pin real behavior.
- The Couch switch, attach and orientation path for grok has no automated coverage. That is #414's to fix.

**ARCH pass/flag**
- **ARCH-DRY:** flag (two Minor items above).
- **ARCH-PURE:** pass. `interrupt.lua`, the rewriter state and `normalizeGrokEvent` are pure, and the residue function is tested against a temp home.
- **ARCH-PURPOSE:** flag, as the Important item above. Parts of the Couch half of the Done-when were not delivered, and the contract was not revised.
- **ARCH-MOCK:** pass. Tests run against fake-runtime fixture trees and a fake grok binary, with live conformance for both the scanner and `runGrok`.
- **ARCH-CONSTRAINTS:** pass. `echo.raw` was deliberately kept out of the byte-split replay to hold the suite's run time.
- **ARCH-SECURE:** pass. No subprocess output reaches the deletion.
- **ARCH-ORDER:** pass. The rewriter's state lives in `outputBoundary`, and the test covers every chunk split.
- **ARCH-FUNERAL:** pass. Slug residue and the temp directory are removed on every call, and no new pair-owned artifact family is created.

**Plan revision recommendations**
- **Issue Revisions:** the Done-when deltas listed above (orientation and park/cold-resume → #414 or verified; the doctor item rerun or moved out of scope).
- **Plan Revisions:**
  - cleanup refusals are reported to stderr, not as an adapt near-miss;
  - bell-attention translation and the per-agent interrupt byte are added as M2 surfaces;
  - the M2 milestone closes together with the issue close (already stated in the Log; restate it in Revisions).

```findings
dispose:
  - id: BR-6
    disposition: addressed
    note: |
      hasFlag routes through resumeform.FlagRegion (agentargs.go:17-24), every argv editor uses it, and TestStripHelpersStopAtDoubleDash now asserts shouldMintSessionID ignores --fork-session/--session-id/-s after -- for claude, qoder and grok.
findings:
  - id: new
    severity: Important
    family: plan-code-drift
    title: |
      Done-when still promises orientation auto-submit, park/cold-resume and doctor firing that did not ship; no Revisions entry
    detail: |
      2nd in family. Rule: every deviation from a committed contract (Done-when, plan design or Durable state) gets a Revisions entry when decided, not just Log prose. Sweep all Done-when items and plan design claims in one pass: orientation goes to #414; park/cold-resume is verified or goes to #414; the doctor rerun is logged or moved out of scope; residue-refusal reporting changed from adapt near-miss to stderr; bellAttentionHarnesses and nvim/interrupt.lua are new M2 surfaces.
  - id: new
    severity: Minor
    family: grok-envelope-single-source
    title: |
      Accepted grok update methods differ across scanner, normalizer and ParseTokenUsage
    detail: |
      applyGrokRecord accepts any non-empty method, normalizeGrokEvent accepts session/update and _x.ai/session/update, and usage accepts only session/update. These should share one set (ARCH-DRY).
  - id: new
    severity: Minor
    family: agent-file-read-duplicated
    title: |
      interrupt.lua current_agent duplicates the agent-file read in init.lua pair_read_saved_config
    detail: |
      nvim/init.lua:3099-3103 reads PAIR_AGENT_PATH the same way; it could call PairInterrupt.current_agent, or both could use a shared helper (ARCH-DRY).
```
