# Boundary Review — pair#751 (whole-issue close)

| field | value |
|-------|-------|
| issue | 751 — cast-sketch: recording model, pressure paths and undo history |
| repo | pair |
| issue file | workshop/issues/000751-cast-sketch-recording.md |
| boundary | whole-issue close |
| milestone | — |
| window | a1630f0c07c8ac61814c74131799fa53357db8f3..19bc7a71951e1555bf8e94be342458e8e9d3954c |
| command | sdlc close --issue 751 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T22:15:11-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation delivers the main recording surfaces, with strong compatibility, ordering, reservation, and catch-up tests. Targeted checks pass. Two issues remain: signed zero makes Go/JS history accounting disagree, and the browser exposes mutable authoritative sketch state despite the plan’s encapsulation contract.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: cross-language-history-accounting
    title: |
      Signed zero produces different Go and JavaScript history_bytes
    detail: |
      cmd/internal/webcast/recording/sketch_state.go:149 counts json.Marshal(known), which preserves a float64 negative zero as -0. cmd/internal/webcast/web/sketch-model.js:229 instead counts sketchCanonical output, which normalizes -0 to 0. A legal negative-zero coordinate therefore adds one byte in Go but not JavaScript, violating the required prefix-projection agreement and producing different cap admission arithmetic. Use one defined numeric normalization for model accounting in both languages, retain exact wire accounting in SketchBudget, and add a shared prefix regression using math.Copysign(0, -1).
  - id: new
    severity: Important
    kind: defect
    family: authoritative-state-encapsulation
    title: |
      Browser callers can bypass sketch transitions through public mutable history
    detail: |
      cmd/internal/webcast/web/model.js:75 exposes the authoritative state as model.sketches, including its mutable maps, points, stacks, receipts, and sequence counters. A caller can change accepted geometry or operation authority without nextSketch validation; a direct probe changed a stored point to 999 and next_sequence to 12345, and sketchProjection reported both changes. ARCH-ORDER requires bypass prevention, and the plan explicitly promises read-only projections. Keep production history private and expose owned projections, with a regression proving callers cannot mutate authoritative state.
```

1. **Strengths**

   - Both ordered producers reuse `OrderedTime` and `SketchBudget`, keeping source fences and admission arithmetic centralized.
   - The preserving codecs retain nested unknown fields while validating known fields strictly. Legacy-reader compatibility exercises the existing decoder and validator.
   - Catch-up restoration validates receipts through the transition core. Prefix and repeated-compaction tests cover interruption, clear, navigation, and driven DOM events.
   - Sink-failure tests distinguish reducer acceptance from persistence and prevent exact retries from becoming fabricated acknowledgments.

2. **Critical findings**

   - **History accounting parity:** [sketch_state.go:149](/Volumes/TestScratch/review-checkouts/182d3d354475ba4e/cmd/internal/webcast/recording/sketch_state.go:149) and [sketch-model.js:229](/Volumes/TestScratch/review-checkouts/182d3d354475ba4e/cmd/internal/webcast/web/sketch-model.js:229). Normalize numeric values consistently before counting model bytes. The Node probe retained `history_bytes: 660` with a negative-zero coordinate; Go’s encoding adds the minus-sign byte. **ARCH-PURPOSE** is flagged because prefix agreement is an explicit Done-when requirement.

3. **Important findings**

   - **Mutable browser authority:** [model.js:75](/Volumes/TestScratch/review-checkouts/182d3d354475ba4e/cmd/internal/webcast/web/model.js:75). Make the production sketch state private; retain `sketchProjection()` as the consumer interface. **ARCH-ORDER** is flagged because callers can currently bypass the transition owner.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:

   - Go recording, sketch, board, and recorder package tests.
   - Webcast `TestNode`, `OperatorScript`, catch-up, and Hub tests.
   - All 116 selected Node tests.
   - `make test-affected-inventory` and pinned-range `git diff --check`.

   The existing signed-zero test covers canonical identity, but not history accounting. The corpus’s Go literal `-0.` does not create a negative-zero float; use `math.Copysign`. Add a regression for browser state encapsulation as well.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared producer helpers and transition-based snapshot verification.
   - **ARCH-PURE — pass:** history, geometry, and budget logic remain separate from sink/browser IO.
   - **ARCH-PURPOSE — flag:** signed-zero accounting breaks required cross-language agreement.
   - **ARCH-MOCK — pass:** stateful sinks and existing browser fakes exercise production seams.
   - **ARCH-CONSTRAINTS — pass:** explicit history/count limits, reserved finalization capacity, and counted append/stack tests.
   - **ARCH-SECURE — pass:** strict known-field decoding and atomic snapshot validation.
   - **ARCH-ORDER — flag:** browser authoritative state lacks encapsulation.
   - **ARCH-FUNERAL — pass:** records use the existing recording-directory lifecycle with bounded added growth.
   - **ARCH-EVOLVE — pass:** additive event kinds and nested unknown-field round trips are covered.

   Atlas updates cover the new format and architectural surfaces. This slice introduces no new user command, flag, or configuration requiring a README update.

7. **Plan revision recommendations**

   Add `## Revisions` entries defining consistent signed-zero accounting and the browser’s private state ownership boundary. Record the corresponding shared-prefix and mutation-isolation regression evidence.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"f17ab95c-0850-4f9d-ad1a-09152d543277","repository":"github.com/xianxu/pair","issue_id":"000751","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/182d3d354475ba4e","reviewed_head":"19bc7a71951e1555bf8e94be342458e8e9d3954c","slot":"pair:9","requested_at":"2026-10-21T05:11:42.339327Z"},"phase":"finished","pid":9808,"started_at":"2026-10-21T05:11:42.533718Z","finished_at":"2026-10-21T05:15:10.232365Z","session_id":"01a15cdf-564d-77e3-bbee-607a691a7afc","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.598363208s
head: "19bc7a71951e1555bf8e94be342458e8e9d3954c"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
