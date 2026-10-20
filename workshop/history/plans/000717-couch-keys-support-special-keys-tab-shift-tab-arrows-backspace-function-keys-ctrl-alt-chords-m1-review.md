# Boundary Review — pair#717 (milestone M1)

| field | value |
|-------|-------|
| issue | 717 — couch --keys: support special keys (tab, shift-tab, arrows, backspace, function keys, ctrl/alt chords) |
| repo | pair |
| issue file | workshop/issues/000717-couch-keys-support-special-keys-tab-shift-tab-arrows-backspace-function-keys-ctrl-alt-chords.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 0b93f91b541fdba4a7bb52196e5abd8d66bebd52..55335ce84b2411e558a2785ef9616389f6fe1e22 |
| command | sdlc milestone-close --issue 717 --milestone M1 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-20T01:27:46-07:00 |
| verdict | SHIP |

## Review

Code tests ran. The couchcmd failures come from the sandbox refusing `git init`, so next I'm running only the tests for the changed areas.

```verdict
verdict: SHIP
confidence: high
```

M1 does what it set out to do. Each caller key is now one row of `agentdrive.Steps`, and that row carries the key's bytes. The parser (`isStep`/`stepDoc`), `InputBytes`, `couch --keys --help`, `StepSyntax`, the couch skill sentence and the new wrapper test all read that one table. Refused chords are checked before the unknown-key path, and each states its own reason. The recipe's internal arrows `recipe-left`/`recipe-right` are gone; the recipe now uses the caller names `left` and `right`, with the same bytes. README, atlas and the skill are updated in the same range. Tests pass: `agentdrive` and `wrapcmd` fully, and `couchcmd` filtered to Keys/Skill/Help/Drive. The full `couchcmd` run fails only in tests that call `git init`, which the review sandbox refuses ("outside the reviewer's directories (sdlc #339)"). That failure has nothing to do with this diff. Only Minor findings remain, and none blocks the boundary.

1. **Strengths**
   - `cmd/internal/agentdrive/keys.go:68-101` puts bytes, syntax, meaning and group in one generated table. `InputBytes` (`keys.go:217-225`) is now a table lookup, which removes the old hand-written switch (ARCH-DRY holds).
   - `keys.go:200-202`: a refused chord is checked before `isStep`, and `TestRefusedStepsStateTheirReason` also asserts that no name is both refused and a step.
   - `cmd/internal/wrapcmd/agent_drive_keys_test.go` sends every non-text named key through the real wrapper writer, in batches of 64, one step at a time. That covers every `alt-*` member against the ESC-prefix handling, which is the bug this change was most likely to ship.
   - `TestStepBytes` pins the bytes with an independent table rather than reading them back from `Steps`. Its row count fails if a family or the exclusions change.
   - The help output stays short: families print once (I checked the rendered `couch --keys --help`), and a line-count bound guards against printing each member.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `keys.go:57` and `atlas/couch.md:2291-2292` tell callers to use `couch --interrupt SLOT --confirm`, but that verb arrives only in M2. If M1 ships alone, the refusal points at a command that does not exist.
   - Plan M1 says "the skill steps sentence and quick table gain the new names", but the skill quick table (`SKILL.md:21-23`) is unchanged; only the sentence was updated.
   - New rows use string literals (`"tab"`, `"shift-tab"`, …) next to the `Key*` constants for older rows. This is harmless, but the style is mixed.
   - `Steps` and `RefusedSteps` are exported mutable package variables, so any importer could change the admission vocabulary at runtime. Low risk, since the module is internal.

5. **Test coverage notes:** Bytes, refusal reasons, the unknown-name listing, help grouping, the skill sentence and the wrapper path are all covered. `TestStepsTableIsTheParsersAcceptedSet` checks that every row parses, in both directions. Old wrapper with new couch: an old wrapper's `Validate` refuses unknown kinds, so the skew fails closed. No test covers this, but it is existing behaviour.

6. **Architecture notes**
   - **ARCH-DRY: pass.** One table serves every consumer.
   - **ARCH-PURE: pass.** The table, parser and bytes are pure, and the tests need no IO except the wrapper integration test, which uses the existing fixture.
   - **ARCH-PURPOSE: pass for M1.** All consumers derive from the table (shadow sweep: parser, `InputBytes`, help, `StepSyntax`, the skill test and the wrapper test).
   - **ARCH-MOCK: pass.** The wrapper test uses the existing drive fixture; the live check is planned after landing.
   - **ARCH-CONSTRAINTS: pass.** About 86 rows scanned linearly, at most 64 steps per request.
   - **ARCH-SECURE: pass.** Decoded sequences still go through `Validate`, `InputBytes` returns nil for an unknown kind, and the refusals happen before anything is sent.
   - **ARCH-ORDER: pass.** M1 adds no state carried between events, because the only change is a static table and a stateless lookup; the admission paths are untouched.
   - **ARCH-FUNERAL: pass.** M1 creates nothing durable: it is in-memory tables only.
   - **ARCH-EVOLVE: pass.** The new key kinds are additive on the wire, and old readers refuse unknown kinds. The `recipe-left` → `left` rename is internal and never persisted (no `recipe-left` remains outside the workshop docs).
   - For M2: the interrupt's internal `\x03` kind must stay out of `Steps`, so that `Validate` keeps refusing it.

7. **Plan revisions:** add a "## Revisions" entry to either drop "quick table gain the new names" from M1 or record it as not delivered.

```findings
findings:
  - id: new
    severity: Minor
    kind: defect
    family: forward-reference-to-unshipped-surface
    title: |
      ctrl-c refusal and atlas point at couch --interrupt, which does not exist until M2
    detail: |
      keys.go:57 and atlas/couch.md:2291 direct callers to `couch --interrupt SLOT --confirm`; if M1 lands alone the refusal names a missing verb. Fine if M1 and M2 ship together.
  - id: new
    severity: Minor
    kind: defect
    family: plan-claims-undelivered-item
    title: |
      Plan M1 claims the skill quick table gained new names; SKILL.md quick table unchanged
    detail: |
      Only the --keys steps sentence was updated (SKILL.md:107-112); either add a quick-table row (e.g. shift-tab) or revise the plan item.
  - id: new
    severity: Minor
    kind: defect
    family: mixed-constant-style
    title: |
      New Steps rows use string-literal KeyKinds beside Key* constants
    detail: |
      keys.go:76-83 use "tab", "shift-tab", etc. while older rows use named constants; cosmetic inconsistency.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"46c1419d-cdb4-460f-b951-d47ecfbdd2db","repository":"github.com/xianxu/pair","issue_id":"000717","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/78598fd55144cb8c","reviewed_head":"55335ce84b2411e558a2785ef9616389f6fe1e22","slot":"pair:2","requested_at":"2026-10-20T08:25:09.609158Z"},"phase":"finished","pid":33921,"started_at":"2026-10-20T08:25:09.725163Z","finished_at":"2026-10-20T08:27:42.147863Z","session_id":"283e0eff-e05d-45ed-ac1d-65a7f24bb212","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.252091709s
head: "55335ce84b2411e558a2785ef9616389f6fe1e22"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
