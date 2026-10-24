# Boundary Review — pair#849 (whole-issue close)

| field | value |
|-------|-------|
| issue | 849 — Cast notes: DOM submission loops on 'Feedback changed' and never submits |
| repo | pair |
| issue file | workshop/issues/000849-cast-notes-dom-submission-loops-on-feedback-changed-and-never-submits.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4ffd05c54f8a7fbcf32f13bb95952b0d637cb59a..da2535cc4f4909d97b5d18c92ae759acc9b4d6d4 |
| command | sdlc close --issue 849 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-23T22:55:58-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Still reviewing — checking whether operator.js's per-record visibility invalidation is actually filtered for selection records.

Checking the live tests' assertions and whether the observer dedupe has a non-live test.

Running the Chrome-gated live tests at head to see if they can execute here.

Chrome can't launch in this sandbox (GPU process fatal), so the live tests can't run here. Checking whether a JS harness for observer.js exists, to see how cheap a non-live dedupe test would be.

Atlas is updated well. Now checking for dead freeze/preview-read code left behind by the removed gate.

Found agent-facing guidance in `submission/model.go` that may now describe the wrong provenance; checking it.

The `ReadingGuide` sent to the agent with every submission still describes the old visible-only, preview-gated contract. Sweeping the repo for other copies of that wording.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The code change fixes the problem described in pair#849. The press no longer depends on the preview. It takes `SubmissionSelection().PressSelection(delivered)` at the press (`cmd/internal/couchtty/console_submission.go:143-147`). The only refusal tied to feedback is an unsaved note. The other refusals are concurrency guards for a cast change, a completed delivery or a job already in progress, and each now has its own message. The panel is display-only and gets marked "out of date". Duplicate selection records are dropped in the recorder and are no longer turned into operator views. Build, vet and the affected Go and Node suites all pass: couchtty, submission, webcast, webcast/board, webcast/recorder and `capture-dialog.test.mjs`.

What keeps this from SHIP is that the rest of the contract was not updated to match. `ReadingGuide`, the guidance stored with every submission and sent to the agent, still describes the old rules. So do the bundled couch SKILL.md and README.md. They say only feedback visible on the console is selected, that a stale visible selection refuses, and that new bundles use `replay-rendered-current-view` with `sketch_page.current`. New press bundles now use `replay-rendered` with no `current`, and they include offscreen feedback. The Chrome live tests could not run in this sandbox because Chrome dies with "GPU process isn't usable". So the Done-when claim "old code refuses, new code submits" on a page changing every 50 ms is unverified here.

1. **Strengths**
   - `board.Selection.PressSelection` (`cmd/internal/webcast/board/submission.go:484`) is pure and small. `submission_press_test.go` tests it directly: delivered notes and sketches left out, oldest 8 first, the remainder counted, and the live view cleared.
   - The manager's prepare path releases the lock for the delivered-versions read and the unsaved-note read. It then re-checks the job, cast, delivery generation, guard and an existing identity, each with its own error. Before, about a dozen causes all came out as one `errSubmissionPreview`.
   - Removing `PreviewPrepare`, `PreviewFreezeRejected`, `PreviewRefuse`, `PreviewFreezeProposal` and `samePreparedFeedback` from the core state machine leaves no orphan consumers. The worker's remaining transitions are unchanged.
   - `TestCastPressSelectionRendersFromTheRecording` shows a press renders `replay-rendered` from the recorded finish, even when the input had a live view.
   - The atlas (`atlas/webcast-submission.md`) is fully rewritten for the press-time contract, including a note that older `replay-rendered-current-view` bundles stay readable.

2. **Critical findings:** none.

3. **Important findings**
   - **The agent's `ReadingGuide` is out of date** (ARCH-PURPOSE). At `cmd/internal/submission/model.go:133-134` it still tells the agent "Only feedback actually visible on the current console is selected… A stale visible selection refuses preparation…". It also says "New sketch_page provenance replay-rendered-current-view… Read sketch_page.current". Since `cast_submission.go:424-428`, a press always produces `replay-rendered` with `Current == nil`. This text is an executable contract stored in every bundle, so the agent is misled about what it received.
     - Fix: rewrite both sentences to match atlas lines 1-16 and 66-77. Note `replay-rendered-current-view` only as the legacy form, and update `model_test.go` to match.
   - **SKILL.md and README are out of date in the same way** (ARCH-PURPOSE, docs gate).
     - `cmd/internal/couchcmd/skills/couch/SKILL.md:721,758,776-779` describe the old provenance, visible-only eligibility and the stale-selection refusal.
     - `README.md:1549` says "It submits only feedback actually visible on the current console".
     - Fix: sweep all three in the same round as the `ReadingGuide` fix.

4. **Minor findings**
   - **The scope of the unsaved-note refusal changed without a Revisions entry.** Done-when and Plan say an "open edit" refuses, with "note N is being edited". The code refuses only an edit whose text changed, and the draft message doesn't name anything. This was deliberate and is recorded in the Log, but there is no Revisions entry.
   - **The observer dedupe has no test that runs without Chrome.** The Plan claims "unit tests for… the observer dedupe". In fact the only check is in a Chrome-gated live test (`selections > 3`). The resync reset (`lastSel = null`) is not tested at all.
   - **The unsaved-note check passes silently when there is no operator scope.** `Chrome.SubmissionUnsavedNote` returns `""` when `p.scope == nil`, and the type assertion in the manager also skips silently. Failing open here is probably fine, but it isn't documented.
   - **`refreshOperatorLocked` now returns early on selection records.** That also skips `pruneOperatorLocked`, so on a page sending only selection records, stroke pruning waits for the next non-selection record. This is harmless.
   - **The live tests could not run here.** `WEBCAST_LIVE_CHROME=1` tests fail at launch in this sandbox ("GPU process isn't usable").

5. **Test coverage**
   - Deterministic tests cover these well: press without a preview, press ignoring page changes, captured note revisions, refusal of an unsaved note, the delivery-generation re-check, the out-of-date panel never gating, `PressSelection`, the operator view ignoring selection records, the dialog's `unsaved` getter, and replay rendering.
   - A cheap regression check: on the old code, `TestSubmissionPressNeedsNoPreview` would have hit `errSubmissionPreview`, because no preview run existed.
   - Two items have live coverage only, and that coverage could not be verified here: the 50 ms changing-page press and the end-to-end unsaved-draft CDP evaluation.

6. **Architecture notes**
   - **Pass:** ARCH-DRY, ARCH-PURE (`PressSelection` is pure and the IO sits in a thin seam), ARCH-MOCK (`submissionPreviewFake.unsaved` is behind the same interface), ARCH-CONSTRAINTS (one bounded CDP evaluation per press using `callTimeout`), ARCH-SECURE (the page result is parsed into a typed struct and an exception is reported as an error), ARCH-FUNERAL (nothing new persists; `previewProblem` lives in memory), ARCH-EVOLVE (no new format; the old provenance still validates in `sketch_page.go:43`).
   - **ARCH-ORDER: pass.** Removing the gate takes away states the manager had to coordinate. `lastSel` resets on resync and when recording is off.
   - **ARCH-PURPOSE: flag.** The contract consumers (`ReadingGuide`, SKILL.md, README) were not updated with the atlas.

7. **Plan revision recommendations**
   - Add a "## Revisions" entry saying the refusal covers a new unsaved draft or an edit with changed text, and that an untouched open edit does not refuse. The Done-when and Plan wording should match `collab.js` `unsaved`.
   - Either add a Node unit test for the observer dedupe and its resync reset, or revise the Plan's "unit tests for… the observer dedupe" to say the coverage is live only.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: contract-consumers-not-swept
    title: |
      Agent ReadingGuide still describes the visible-only gated contract and replay-rendered-current-view
    detail: |
      cmd/internal/submission/model.go:133-134 (stored in every bundle, sent to the agent) says only console-visible feedback is selected, a stale visible selection refuses, and new sketch_page uses replay-rendered-current-view with sketch_page.current; after this change presses emit replay-rendered with no current (cast_submission.go:424-428) and include offscreen feedback. Rewrite to match atlas/webcast-submission.md and update model_test.go.
  - id: new
    severity: Important
    kind: defect
    family: contract-consumers-not-swept
    title: |
      couch SKILL.md and README.md still describe the old visible-selection submission contract
    detail: |
      cmd/internal/couchcmd/skills/couch/SKILL.md:721,758,776-779 and README.md:1549 claim visible-only eligibility, stale-selection refusal and replay-rendered-current-view provenance; sweep them alongside the ReadingGuide fix.
  - id: new
    severity: Minor
    kind: defect
    family: plan-claims-match-code
    title: |
      Unsaved-note refusal is narrower than Done-when/Plan (untouched open edit does not refuse); no Revisions entry
    detail: |
      Plan says an open edit refuses with "note N is being edited"; collab.js unsaved refuses only a changed edit or a draft. The choice is in the Log but the Done-when and Plan were not revised.
  - id: new
    severity: Minor
    kind: defect
    family: plan-claims-match-code
    title: |
      Observer selection dedupe has no non-live unit test despite the Plan claiming one
    detail: |
      Only the WEBCAST_LIVE_CHROME-gated TestSubmissionManagerLiveSubmitsOnAChangingPage counts selection records; the lastSel reset on resync is untested. A vm-based test like tests/webcast/observer-sketch-exclusion.test.mjs could cover it.
  - id: new
    severity: Minor
    kind: unverifiable
    family: live-chrome-evidence
    title: |
      Live Chrome tests (50 ms changing page, unsaved draft CDP eval) could not run in the review sandbox
    detail: |
      Ran WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/couchtty -run TestSubmissionManagerLive; Chrome exited at launch with "GPU process isn't usable". A host run of these tests plus cmd/internal/webcast TestOperatorFeedbackLivePrepareAndStalePanel, ideally also failing on base 4ffd05c5, would answer it.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"19c11fcc-7572-4279-8992-ced14070256c","repository":"github.com/xianxu/pair","issue_id":"000849","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/193cdffb14b6c59c","reviewed_head":"da2535cc4f4909d97b5d18c92ae759acc9b4d6d4","slot":"pair:6","requested_at":"2026-10-24T05:51:59.757327Z"},"phase":"finished","pid":11504,"started_at":"2026-10-24T05:51:59.866344Z","finished_at":"2026-10-24T05:55:57.918377Z","session_id":"9357dd91-d710-4563-b6c2-8ad2f9fc0aca","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.13983275s
head: "da2535cc4f4909d97b5d18c92ae759acc9b4d6d4"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411

---

## Re-review — 2026-10-23T23:03:33-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 849 — Cast notes: DOM submission loops on 'Feedback changed' and never submits |
| repo | pair |
| issue file | workshop/issues/000849-cast-notes-dom-submission-loops-on-feedback-changed-and-never-submits.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4ffd05c54f8a7fbcf32f13bb95952b0d637cb59a..b858551ce706ac16f45d3e96069847db5c33e5c2 |
| command | sdlc close --issue 849 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-23T23:03:33-07:00 |
| verdict | SHIP |

## Review

Checking the prior-round fixes against the code: scene selection matches the new guide wording. Next I'm checking the clipping-refusal claim and the selection path.

Live Chrome still fails to launch here (same GPU error), so that finding stays open. Now running the non-live test suites for the packages this change touches.

Tests pass apart from git-init sandbox refusals in couchcmd that have nothing to do with this change. Last step: a pass over the prepare path in `console_submission.go`.

```verdict
verdict: SHIP
confidence: medium
```

The round-1 fixes check out against the code. The agent ReadingGuide, the couch skill, the README and the atlas now describe the press-time contract that `console_submission.go` and `PressSelection` actually implement. A tree-wide grep finds no leftover text from the old visible-only contract. The only remaining `replay-rendered-current-view` mentions are the legacy writer branch, codec and validator, and wording that says older bundles may carry it. The Plan and Done-when now match the unsaved-note refusal and the live-only observer test. Non-live tests pass for `submission`, `webcast/board`, `couchtty`, `webcast`, the couchcmd `Skill` tests and the two `.mjs` suites. The full `couchcmd` package failed only in its git-fixture tests, because the sandbox refuses `git init` outside the reviewer's directories. That has nothing to do with this diff. One thing is still unproven: the live Chrome tests, which carry the "old code refuses, new code submits on a 50 ms page" Done-when evidence, still can't run here. That finding (BR-5) is Minor and doesn't block.

1. **Strengths**
   - `board/submission.go:76` `PressSelection` is a small pure function that picks from the board selection minus delivered versions, with the 8-sketch cap and a count of the rest. `submission_press_test.go` tests it directly with no IO.
   - The prepare path (`console_submission.go:173-222`) does its reads outside the lock, then re-checks the job, cast, delivery generation and guard. Each refusal now gives its own reason, replacing the single catch-all `errSubmissionPreview`.
   - The noise is fixed where it starts: the observer dedupes repeated selections and resets on resync and turn-off (`observer.js:88,226`), and `refreshOperatorLocked` ignores `Selection` records (`operator_view.go:64`).
   - The fallback scene (newest finish by `RecordIndex`, `cast_submission.go:384`) matches the new guide text "the viewport where the newest selected sketch finished".

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - SKILL.md says "an edit not saved" refuses. The code refuses only an edit whose text changed, so the Done-when wording is more precise. Not raised as a finding.
   - `lastSel` reset on resync and turn-off has no test, as the plan revision now admits.

5. **Test coverage notes:** The `ReadingGuide` and skill-privacy tests now pin the new contract terms. The unsaved-note and 50 ms changing-page tests exist, but only behind `WEBCAST_LIVE_CHROME`. Locally, Chrome dies at launch with "GPU process isn't usable", so the "fails on base" half of the Done-when has no evidence from this environment.

6. **Architecture**
   - **ARCH-DRY:** pass. The JS unsaved check lives once in `collab.js`, which broadcast and webcast share.
   - **ARCH-PURE:** pass. Selection rules are pure; the CDP evaluation is the thin seam.
   - **ARCH-PURPOSE:** pass. The shadow-sweep found no leftover old-contract text in any consumer.
   - **ARCH-MOCK:** pass. Tests use the existing cast and Chrome seams with fixtures, and live conformance is kept behind the env gate.
   - **ARCH-CONSTRAINTS:** pass. There is one CDP evaluation per press, bounded by `callTimeout`, and selection noise is cut at the source.
   - **ARCH-SECURE:** pass. `SubmissionUnsavedNote` decodes into a typed struct, and an exception gives a visible error rather than an empty "nothing unsaved".
   - **ARCH-ORDER:** pass with a note. Re-checks after the unlocked reads cover the job, cast, delivery generation and guard, but the 50 ms interleaving is only sampled live.
   - **ARCH-FUNERAL:** pass. No new durable artifacts; the dedupe should only reduce how fast the recording grows.
   - **ARCH-EVOLVE:** pass. The validator and codec still accept the legacy `replay-rendered-current-view` provenance with its `current` field, and the new writer emits only `replay-rendered`.

7. **Plan revisions:** none needed. The 2026-10-23 close-review Revisions entry covers the open-edit and observer-test changes.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      model.go:133-134 now describes the press selection (offscreen included, 8 oldest, only a note being written refuses) and replay-rendered from the newest finish, which matches cast_submission.go:380-428; model_test.go and sketch_page_test.go pin the new terms and pass.
  - id: BR-2
    disposition: addressed
    note: |
      SKILL.md:721-725,757-760,775-783 and README.md:1549 now state the press-time contract; a grep finds no "actually visible", "stale visible selection" or "current recorded prefix" left outside the legacy-bundle notes; skill_privacy_test pins the new phrases and passes.
  - id: BR-3
    disposition: addressed
    note: |
      Done-when bullet 2 and the Plan now say "a new note not saved, or an edit whose text changed and is not saved", with a 2026-10-23 close-review Revisions entry, matching collab.js noteDialog.unsaved.
  - id: BR-4
    disposition: addressed
    note: |
      The Plan and Revisions now say the observer dedupe is tested live only; the existing vm harness (observer-sketch-exclusion) stubs no DOM, so the plan's claim now matches the code.
  - id: BR-5
    disposition: not-addressed
    note: |
      No dispute or host evidence recorded; WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/couchtty -run TestSubmissionManagerLive still fails at Chrome launch with "GPU process isn't usable". A host run of these tests plus TestOperatorFeedbackLivePrepareAndStalePanel, ideally failing on base 4ffd05c5, would answer it.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"a76f1f17-726b-4131-b600-b37f8fa8072a","repository":"github.com/xianxu/pair","issue_id":"000849","recipe":"milestone-review","round":2,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/2077636fdb72f489","reviewed_head":"b858551ce706ac16f45d3e96069847db5c33e5c2","slot":"pair:6","requested_at":"2026-10-24T05:59:37.224601Z"},"phase":"finished","pid":94584,"started_at":"2026-10-24T05:59:37.32469Z","finished_at":"2026-10-24T06:03:32.936203Z","session_id":"45b2d198-d51f-4328-882b-9689d66e4f4c","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.169402541s
head: "b858551ce706ac16f45d3e96069847db5c33e5c2"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-23T23:25:22-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 849 — Cast notes: DOM submission loops on 'Feedback changed' and never submits |
| repo | pair |
| issue file | workshop/issues/000849-cast-notes-dom-submission-loops-on-feedback-changed-and-never-submits.md |
| boundary | whole-issue close |
| milestone | — |
| window | 2db9843cb64a116c19245895c9a3120e0f1453c6..f18b57703a6a24792522ce52633687bcc97f7a04 |
| command | sdlc close --issue 849 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-23T23:25:22-07:00 |
| verdict | SHIP |

## Review

Retrying the live Chrome tests without `timeout`, using go test's own timeout flag.

```verdict
verdict: SHIP
confidence: high
```

This window is only bookkeeping since the close review. There is no product code in it: two re-hashed affected-reader inventory records, one new `workshop/lessons.md` entry, the generated close-gate and close-review plan records, and issue frontmatter and Log updates. The issue's `## Spec`, `## Revisions` and `## Done when` are unchanged, so scope did not change after close. The inventory fix is correct, and I checked it as a class rather than just the two files: every record under `tests/affected-readers/` now matches the sha256 of its file at head. Both touched packages pass their non-live tests. The only open prior finding is BR-5 (Minor, live Chrome evidence). It is still unverifiable here: Chrome fails at launch with the same "GPU process isn't usable" error, and no dispute with host evidence was filed. It doesn't block.

1. **Strengths**
   - `tests/affected-readers/cmd/internal/submission/{model_test,sketch_page_test}.go.json` now hold the real head hashes (`93c1b100…`, `cbf919df…`). The test files changed in 04c96ae58, before the window, and their inventory records had gone stale. I re-hashed every inventory record against its file and found no other stale or missing ones.
   - `workshop/lessons.md:2851` turns the BR-1/BR-2 family (`contract-consumers-not-swept`) into a rule rather than a fix for one site. It lists every agent-facing copy of the contract (`ReadingGuide`, `SKILL.md`, README, atlas), says to grep the key phrases before close, and links it to #802 BR-1. That is the class-level answer ARCH-PURPOSE asks for.
   - The issue's Log entry names the specific tests and mutation checks behind the close, and records the flow upgrade from quick to full and why.

2. **Critical:** none.
3. **Important:** none.
4. **Minor:** BR-5 stays open (see dispositions).
5. **Test coverage notes:** `go test ./cmd/internal/submission ./cmd/internal/couchtty` passes at head. `WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/couchtty -run TestSubmissionManagerLive` fails at Chrome launch because the sandbox has no GPU process. That is an environment limit, not a product failure.
6. **Architectural notes**
   - ARCH-DRY: pass, no duplicated logic in the window.
   - ARCH-PURE: pass, no code.
   - ARCH-PURPOSE: pass. The inventory sweep and the lessons rule cover the whole class, not one site.
   - ARCH-EVOLVE: pass. The inventory records keep the same `version: 1` schema and change only the hash value.
7. **Plan revision recommendations:** none.

```findings
dispose:
  - id: BR-5
    disposition: not-addressed
    note: |
      Still unverifiable here. Re-ran WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/couchtty -run TestSubmissionManagerLive at f18b57703 and Chrome exited at launch ("GPU process isn't usable", exit_code=6). No dispute with host evidence was filed. The Log claims a host run but gives no captured output. Minor, does not block.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"1b66477f-a4fc-4491-8f96-e0ba34461036","repository":"github.com/xianxu/pair","issue_id":"000849","recipe":"fix-review","round":3,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/4159f53a3b2c260e","reviewed_head":"f18b57703a6a24792522ce52633687bcc97f7a04","slot":"pair:6","requested_at":"2026-10-24T06:21:38.445141Z"},"phase":"finished","pid":2954,"started_at":"2026-10-24T06:21:38.660887Z","finished_at":"2026-10-24T06:25:14.661083Z","session_id":"2a5927e0-06bb-44a5-91e1-e1368d7da0b1","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 4.139515666s
head: "f18b57703a6a24792522ce52633687bcc97f7a04"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
