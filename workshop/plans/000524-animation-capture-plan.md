# Stable animation capture implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development for bounded independent tasks or superpowers-executing-plans for the measured capture path. Steps use checkbox syntax for tracking.

**Goal:** Capture a settled blog card without restarting its animation, and explicitly label captures that cannot settle.

**Architecture:** Keep capture in Recorder.CaptureCrops and its existing admitted-document/CDP seam. Disable viewport-changing screenshot capture, intersect each requested crop with the current viewport, and wait boundedly for relevant finite animations before collecting box text and pixels. Do not alter animation styles/timelines or operator focus.

**Tech Stack:** Go, CDP, browser Web Animations API, existing node fixtures and opt-in temporary-profile headless Chrome tests.

## Approval point

This proposal awaits TL approval. A current-surface crop is the requested padded box intersected with the current viewport. Report the actual captured box and `clipped_to_viewport=true` if narrowed; fully offscreen boxes retain recorded context and report crop unavailable (outside viewport). The original unpadded region still defines box-text collection, whose existing contract already uses visible intersection. This changes the prior beyond-viewport pixel behavior and must be explicitly approved. No scrolling, resizing, focusing, synthetic visibility, or forced CSS state is proposed.

## Measurement and alternatives

See issue Log for production-path and direct-CDP A/B measurements. Waiting alone failed because capture restarted a completed responsive fade. With identical clip and settled card, captureBeyondViewport=true produced zero bright pixels and reset animation currentTime; false produced 19891 bright pixels and preserved the finished timeline, twice each.

1. **Recommended:** unchanged-viewport screenshot plus bounded natural settling. Preserves the live page; offscreen pixels are explicitly unavailable.
2. Force/restore final animation styles around capture: could preserve beyond-viewport reach but must suppress responsive restarts, handle pseudo-elements/shadow trees and animation callbacks, and reconcile mutations on cancellation. More invasive and risks recording synthetic DOM/style state.
3. Wait before the existing screenshot: rejected by direct measurement; the capture restarts already-finished animation.

## Core concepts

### Pure entities

| Name | Lives in | Status |
| --- | --- | --- |
| Crop evidence | cmd/internal/submission/model.go | modified |
| Crop/viewport intersection | cmd/internal/webcast/recorder/submission.go or small colocated helper | new |

Each note owns one crop. Add optional settlement evidence with an explicit outcome (`settled` or `unsettled`) and bounded reason for timeout, infinite/paused animation or inspection limit. Older manifests without it remain readable as unknown, never inferred settled. Record clipped-to-viewport separately from animation outcome and preserve existing document/time/unavailable fields. Validation owns the vocabulary; tests reject malformed outcomes/reasons. Reuse recording.Box validation and one intersection function; do not duplicate the box-text DOM traversal (ARCH-DRY, ARCH-PURE).

### Integration points

| Name | Lives in | Status | Wraps |
| --- | --- | --- | --- |
| Crop readiness probe | cmd/internal/webcast/recorder/capture_ready.js | new | viewport and browser animation observations |
| CaptureCrops | cmd/internal/webcast/recorder/submission.go | modified | existing Caller.CallSeq + document barriers |
| Crop propagation | cmd/internal/webcast/cast_submission.go | modified | captured evidence → durable manifest |

The probe observes animations whose target/ancestor intersects the crop; includes transitions, delayed animations and pseudo-element targets as exposed by Web Animations. Browser observation is integration code, not a pure model. Reuse stateful fakeChrome to control animation completion, paused/infinite results, cancellation, context loss and navigation. Real Chrome is the conformance oracle for media-query animation restart and pixels (ARCH-MOCK).

## Operating envelope and ordering

- One existing preparation, sequential notes; retain 30 s total and existing screenshot/PNG/bundle/store limits. No new process or durable artifact family.
- Proposed settling budget: at most 1500 ms per note inside the existing total budget, polling at 50 ms, at most 512 animation entries inspected per probe. These are proposed product limits, not measured performance promises. A page hitting the budget is captured as unsettled with an explicit reason; a canceled outer operation or navigation fails preparation, as before. API enumeration itself is browser-owned; never serialize an unbounded animation list over CDP.
- Scope: same top document and accessible open-shadow surface used by current text evidence; no separate-frame traversal. Unsupported inspection is explicit unsettled evidence, never empty-list success.
- Preserve exact admitted execution-context identity. Sequence: identity check → readiness/viewport observation → bounded polling → actual crop-box derivation → box text → identity check → screenshot with captureBeyondViewport=false → identity check → evidence. A viewport/scroll change across observation/capture refuses rather than pairing mismatched geometry/text/pixels.
- No persistent animation ownership or restore operation exists because the implementation observes natural progress and does not mutate animation state. Interrupted Go calls release their contexts; no browser-side async loop survives return. Existing delivery state machine is unchanged (ARCH-ORDER).
- Reasons are fixed strings/enums, bounded bytes; metadata stays under the existing 64 MiB bundle and 1 GiB retained-store cap with no silent eviction (ARCH-CONSTRAINTS, ARCH-FUNERAL). Untrusted CDP results are decoded/validated before use (ARCH-SECURE).

## Implementation and tests (single review boundary)

### 1. Regression first

- [x] Extend cmd/internal/webcast/live_submission_test.go with a small local page: intersection-triggered, media-query-scoped opacity+translate fade at DPR 2. Wait until finished, capture a small crop, and independently assert bright pixels, unchanged animation identity/startTime and viewport. Verify the old path fails. Keep the test opt-in and temporary-profile headless, bounded by its existing deadline.
- [x] Add finite delay/transition, long-running, paused/infinite, unrelated off-box animation and partially/fully offscreen cases in stateful JS/CDP fixtures. Assert explicit outcomes and no animation mutation/focus effects.
- [x] Add pure intersection tables and metadata JSON compatibility/validation tests before implementation.

### 2. Implement one capture path

- [x] Add compact readiness probe and bounded Go polling under the existing context; use the exact admitted execution context. Re-query after progress so newly started relevant animations cannot be ignored.
- [x] Use unchanged-viewport capture, actual visible clip, and propagate settlement/clipping metadata through CapturedCrop and Cast into submission.Crop. Preserve unavailable-note semantics and strict navigation refusal.
- [x] Extend stateful recorder tests for cancellation/timeout/navigation and viewport changes at every capture boundary. Mutation-check the capture option and settlement metadata wiring.
- [x] If #494 lands first, route the screenshot through its document-scoped capture suppression helper, preserving its exact context and error contract; otherwise coordinate its second-lander hook. Keep a single screenshot call site.

### 3. Verify and ship

- [x] Run GOMAXPROCS=2 go test -p 1 ./cmd/internal/submission ./cmd/internal/webcast/recorder ./cmd/internal/webcast/submission ./cmd/internal/webcast.
- [x] Run the opt-in bounded headless regression and original crop conformance once after the final behavior change; separately confirm the localhost blog with an isolated profile. Never focus/reload the operator's Chrome.
- [x] Update README and atlas/webcast-submission.md with settled/unsettled and viewport evidence semantics; update artifact inventory for any new scanned sources and generated mirrors as required.
- [x] Run the full suite serially under the machine gate and record all failures honestly: only the pre-existing pair#519 notification startup-hook failure; full evidence is in the issue Log.

## Shipment handoff

SDLC fresh close review is next. Request smoke/staging via hub only; land on authorization, report, stop. #494 has not landed at verification time; preserve its suppression helper if it lands first, otherwise its owner integrates the single screenshot call site.

## Revisions

- 2026-10-12 — Initial measured proposal. No implementation authorized yet; viewport/offscreen policy is the principal approval question.

- 2026-10-12 22:56 PDT — TL approved this proposal including viewport-only crops and all budgets (receipt 5992b8ba-8060-42e0-949a-d52fc973ae69); the approval hold above is satisfied.


### 2026-10-12 — BR-1 animation influence correction

ARCH-PURPOSE: target geometry alone does not bound animated paint. Direct targets and enclosing ancestors remain covered; zero-size/off-box targets with light children, shadow content, or generated pseudo-elements can affect overflowing pixels. Conservatively treat those active animations as potentially relevant within the existing 512-animation and 1500 ms limits. Finite animations wait naturally; paused/infinite ones remain explicitly unsettled. No per-animation descendant walk or mutation is needed. Plain off-box leaf targets without generated content remain irrelevant. Regression fixtures cover finite/paused/infinite cases and browser pixel evidence. This corrects influence detection within the approved capture contract, with no new scope or limits.


## Revisions (2026-10-13 approval)

TL approved conservative document-wide observation (receipt 3e4e0d53-b86b-4ca6-a147-d487a5cef374). Replace all target-geometry exclusions with observation of every active animation in the top document and accessible open shadow roots. Painted influence includes children, pseudo-elements, shadows, filters, outlines and other paint expansion; no CSS paint-bound approximation is attempted. Unrelated off-box animations may consume the wait or mark evidence unsettled. Unchanged 1500 ms/note, 50 ms polls, 512 animations, 20k traversal, 30 s total; crossing limits explicitly marks unsettled. This supersedes earlier claims that plain off-box leaves are safely irrelevant. #494 is now merged; CaptureCrops uses its exact-context CaptureScreenshot helper with captureBeyondViewport=false. Stage only when TL says.


## Revisions — 2026-10-13 smoke-blocking draft recovery proposal

The operator's smoke exposed a second #483 defect after Finder metadata refused preparation: the DOM controller retained the original authored body, rejected edits, and advertised no recovery. TL requested editable pre-dispatch retries plus an actionable frozen-draft retry/discard path (71346ea3-9dee-45c6-b971-fe47f1fed593), including recovery from the old code's freeze (7f4c46f7-dbd0-4a29-b74c-106a766b7b38). This revision is proposed for TL review before its larger lifecycle/API addition. The completed Finder scan fix is independent and already built at dd1279335ea07eedf21f7ecbc1a9e3d2ab951729.

### Core concepts

| Name | Lives in | Status |
| --- | --- | --- |
| Draft submission attempt | nvim/dom_submission_state.lua | modified |
| Authored send retry | nvim/submission.lua | modified |
| Delivery lifecycle | cmd/internal/submission/delivery.go | modified |

| Name | Lives in | Status | Wraps |
| --- | --- | --- | --- |
| DOM effect adapter | nvim/dom_submission.lua | modified | callbacks and submission RPC |
| Draft keybindings | nvim/init.lua | modified | Neovim draft and confirmation |
| Submission manager | cmd/internal/couchtty/console_submission.go | modified | stored bundles and exact target |
| Submission command protocol | cmd/internal/couchmessage/protocol.go, cmd/internal/couchcmd/submission.go | modified | private Couch RPC |
| Submission store | cmd/internal/submission/store.go | modified | durable terminal state |

### Behavior and invariants

ARCH-ORDER: distinguish capture/intent from a pane write. Preparation and begin RPC failures do not freeze authored text. Keep the admitted ID when an RPC outcome is ambiguous so retry can inspect the same server work, but accept new body/text and regenerate the prompt from retained capture metadata before pane dispatch. A definite failed preparation drops the attempt. In particular, an error from prepare before job admission must not retain the original body as a lock.

The generic sender reports its phase as a second return value, propagated through the init wrapper and DOM adapter. A definite start-phase failure (append/target validation before pane write) releases authored text. Written or indeterminate retains the original body, generated prompt, target and ID. A confirmed dispatch remains acknowledgment-only; retry never sends it again. Existing indeterminate-write refusal remains explicit: a partially written composer requires manual reconciliation; never paste blindly over it.

Provide a real discard operation for an idle uncertain attempt, tentatively Ctrl+Shift+Alt+Backspace. Confirmation explains that it cannot undo a possible send and the operator must clear any partial agent composer. The operation terminalizes the existing bundle as discarded, retains local evidence under the existing store cap, and leaves notes unresolved. It then releases the matching generic attachment and DOM attempt; it preserves the editable draft. Never reset an unrelated generic send, an active preparation, or confirmed dispatch awaiting acknowledgment. Discard RPC failures retain the attempt and expose retry. Duplicate discard is idempotent; a lost reply can be retried. A new preparation excludes discarded bundles. No deletion, fabricated acknowledgment, or note resolution.

Actionable uncertainty text names the DOM send chord and actual discard chord. Retry uses the saved original payload without asking the operator to reproduce it; if the visible draft has since changed, retain those edits rather than consuming them when the original is sent. Do not silently overwrite the current buffer. Exact presentation of the two actions and key awaits TL confirmation.

ARCH-DRY: the delivery transition function owns the new terminal transition, and the existing private request validator enumerates the operation once. No second delivery ledger. ARCH-FUNERAL/CONSTRAINTS: discarded bundles remain under current byte/count caps and explicit local cleanup, matching resolved evidence. Existing retry/observation deadlines remain bounded.

### Old-build recovery

The old original-body freeze lives only in each Neovim controller's Lua memory; it is not persisted in the draft file. Fresh draft startup initializes idle, preserving the operator's current text. The specific .DS_Store admission refusal created no bundle. A durable delivering manifest is different: it records intent and cannot prove whether a pane write occurred after a crash. Do not automatically release that uncertainty based only on its name. Prepared manifests are known not yet begun; a migration/recovery test must distinguish these states without treating delivering as unsent.

### Implementation and proof (single additional review boundary)

- [ ] Add failing pure-model and adapter tests: prepare error → edit → successful submit uses new body; invalid path/timeout/begin failure before pane write allows editing; stale replies cannot freeze or send a successor.
- [ ] Add generic sender phase-result tests: append/start failure allows replacement; written/indeterminate freezes exact payload; dispatched remains commit/ack only.
- [ ] Add delivery/store/manager/protocol tests for target-scoped discard, duplicate/lost reply, terminal admission exclusion, retained evidence, no note resolution, and refusal during busy/acknowledgment work.
- [ ] Implement those model transitions and thin adapter effects, with a confirmation binding and actionable messages. Preserve modified current drafts on original-payload retry.
- [ ] Extend stateful headless Neovim fixture for prepare-error/edit/resubmit, uncertain dispatch/retry-original, discard/new attempt, stale callbacks, and a freshly initialized controller with edited disk draft after an old-code admission refusal.
- [ ] Run lua nvim/dom_submission_state_test.lua, lua nvim/dom_submission_test.lua, lua nvim/submission_test.lua; run tests/submission-transaction-nvim-test.sh with bounded headless cases. Run GOMAXPROCS=2 go test -p 1 ./cmd/internal/submission ./cmd/internal/couchmessage ./cmd/internal/couchcmd ./cmd/internal/couchtty ./cmd/internal/webcast -count=1.
- [ ] Update atlas/webcast-submission.md for recovery and the real discard key; run required close review and suite, then make build in pair:0, verify actual HEAD, hand off fresh-thread smoke, and wait for acceptance before landing.


## Revisions — 2026-10-13 operator supersession policy (approved)

TL instruction d34bfd0a-9f86-45d0-b4c8-775874af760c supersedes the entire draft-recovery/discard proposal above. The operator accepts duplicate agent sends. There is no frozen draft, original-body retry, discard operation, new key, or confirmation. Every DOM send chord starts a fresh attempt from the current buffer with a new submission ID and generated bundle. Definite and uncertain failures leave the current draft editable; late callbacks cannot consume or send a newer draft.

Keep delivery truth separate from supersession: add an optional superseded marker to an existing manifest rather than inventing a delivered/aborted outcome. New same-target preparation supersedes older prepared/delivering attempts, retaining their files and original delivery state. Supersession must survive restart, and an eventual genuine acknowledgment of an older delivering attempt may still advance it and idempotently resolve the captured note versions. No inference of delivery from supersession. Existing completed/dispatched evidence remains acknowledged evidence; one bounded background operation remains the server's concurrency limit. A transient busy failure releases the draft and the next chord is another fresh attempt.

### Revised concepts and implementation

- Submission manifest (cmd/internal/submission/model.go): optional superseded marker, backward-compatible default false; preserve it across delivery transitions. Store (store.go) atomically updates manifests using the existing durability path; Pending excludes superseded attempts. Supersede only matching immutable targets and only prepared/delivering states; no deletion and no notes effects. Retain byte/count bounds.
- Submission manager (cmd/internal/couchtty/console_submission.go): admit new same-target attempts despite older pending/delivering bundles by marking them superseded; preserve current busy limit and target guards. After a restart, an old prepared/delivering bundle cannot lock the new draft. A late ack remains valid and resolves note versions idempotently.
- DOM model/adapter (nvim/dom_submission_state.lua, nvim/dom_submission.lua): each send replaces the active attempt with a fresh epoch/id/current body; no retry-body or ack-only chord interception. All failure exits release local attempt state. Acknowledgment polling may finish an active successful attempt, but stale responses cannot mutate a new one.
- Generic sender (nvim/submission.lua): DOM attachments are independent fresh transactions keyed by the new bundle ID; do not retain a failed attached body as a lock on later sends. Keep ordinary, non-DOM submission behavior outside this policy. A confirmed dispatch still records its log marker and returns success; failed marker bookkeeping must not pretend delivery failed or freeze a subsequent DOM attempt. Preserve evidence in the existing session log.
- No binding changes. The existing Ctrl+Shift+Alt+Return always submits current text as a new attempt. Failure wording says the draft can be edited and the chord pressed again; no unavailable discard action.

### Revised verification checklist

- [ ] Store regression: prepared and delivering bundles superseded by a new same-target attempt, retained and readable after restart; other target unchanged; no fabricated delivery; old actual acknowledgment remains legal.
- [ ] Manager regression: old durable uncertainty no longer blocks a new attempt; repeated acknowledgments resolve notes idempotently; existing target/async boundedness tests pass.
- [ ] Lua RED/GREEN: prepare error → edit → resubmit sends current body with fresh ID; uncertain send → edit → resubmit sends current body with fresh ID; old callbacks never send/consume current draft; failed acknowledgment never swallows the next chord.
- [ ] Generic sender tests prove fresh DOM attachment delivery after prior start/written/indeterminate failure, without changing ordinary-send protections.
- [ ] Stateful headless fixture covers both pre-dispatch and uncertain-dispatch edits, original text unavailable, new-controller recovery and repeated chords. Update atlas/webcast-submission.md and superseded old retry documentation.
- [ ] Run affected Go packages and Lua/headless suites, fresh review and required boundary checks. Commit all changes, rebuild pair:0 from clean HEAD, verify go version -m bin/couch reports matching vcs.revision and vcs.modified=false, then send hash for smoke. Do not merge before acceptance.

ARCH-ORDER: explicit supersession releases authority without fabricating delivery. ARCH-DRY: retain one durable manifest and one existing transition/write path. ARCH-CONSTRAINTS/ARCH-FUNERAL: no additional artifact family, no extra concurrency or retention budget, existing caps and cleanup remain.
