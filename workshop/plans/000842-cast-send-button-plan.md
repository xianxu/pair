# Cast Send Button Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Expose the existing draft-plus-feedback action as ⏎ in Couch's cast status groups whenever that cast has eligible pending feedback.

**Architecture:** Declare the existing Shift+Ctrl+Alt+Return chord once in workbenchshortcut. A drawn status control admits that chord to the exact active actor endpoint; existing wrappers and draft routing invoke PairSendDOM, which keeps all payload, permission, acknowledgement and uncertain-write authority. Derive visibility from the existing acknowledged preview, without a second feedback store or per-paint IO.

**Tech Stack:** Go console/presenter and workbench shortcut model, generated Lua mappings, existing headless Neovim transaction fixtures.

---

## Approved boundaries

Hub ddb54395 limits this to the two Couch terminal status groups (LIVE and Chrome), not #856's browser-page column. A group's eligibility is cast-specific. Current hotkey submission supports Chrome feedback only; terminal NoteBoard notes do not qualify and cannot make Send appear. The missing terminal-sketch submission path is separately filed as #859, unclaimed. Do not fabricate a terminal eligibility source to satisfy the original broad wording.

Hub6491d0e0 approves the narrow keyboard availability expansion: declare the chord as a workbench action and reserve this one chord from fullscreen right-terminal programs. Agent, review, draft and right-terminal focus all reach the existing draft function without focus stealing. Other terminal-app shortcuts keep their policy. Document the wider availability in help and atlas. #843 tooltips remain separate.

Rejected alternative: a new Console-to-Zellij draft submission adapter would duplicate role discovery and native-effect admission. Reuse the existing chord route (ARCH-DRY). No new protocol, transaction, executor, payload assembler, draft-file reader or retry loop.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| Send feedback projection | cmd/internal/submission/preview.go | new |
| Shared send chord/action binding | cmd/internal/workbenchshortcut/shortcut.go | modified |
| Cast Send visibility and drawn spans | cmd/internal/couchtty/reserve.go | modified |

The feedback projection is a small scalar view of PreviewState: only an acknowledged Shown publication with at least one selected note or sketch qualifies. Dirty, publishing, failed, disposed and empty states do not. Include the existing host/publication witnesses; never clone ink, scan feedback or infer eligibility from NoteBoard counts. One PreviewState owns one projection. It is display evidence, never permission to submit; press-time transaction validation stays authoritative. Deferred off-view feedback alone does not qualify.

The shared binding adds ChordCtrlAltShiftEnter and ActionSendDOM, canonical Kitty bytes ESC[13;8u, LuaFunction PairSendDOM, NvimKey <C-S-M-CR>, AgentReserved true and FocusDraft false. Generate the Lua map and remove its redundant literal init.lua binding. The shared table owns encoding, routing and help (ARCH-DRY).

StatusModel has independent LIVE/Chrome Send availability; RenderedStatusRow has independent exact half-open spans. A complete glyph is drawn or its hit target is empty. Current production LIVE availability is false; render tests exercise the generic independent slots without promising unsupported terminal submission.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Preview projection publication | cmd/internal/couchtty/console_submission_preview.go | modified | existing preview worker and manager lifetime |
| Drawn Send witness and click admission | cmd/internal/couchtty/console_send.go (new), console_presentation.go, terminal.go, terminal_input.go | new/modified | accepted status frame and exact actor endpoint FIFO |
| Existing shared draft route | cmd/internal/workbenchshortcut/shortcut.go; nvim/workbench_actions.lua; nvim/init.lua | modified | wrapper/review/right-terminal role routing and PairSendDOM |

Publish an immutable scalar summary atomically from the existing serialized preview transitions, including invalidation/disposal and cast replacement. Presentation reads it in O(1); it must not acquire manager.mu while holding console.mu (the request path already takes manager then console). Keep one owner/helper for transitions and summary publication so invalidation cannot leave a stale true value. Carry the preview-run/cast identity, host and publication witness into the accepted frame. Do not add a worker or polling timer.

On an unmodified left-button press in a fully drawn Send span, capture/revalidate that accepted frame's actor, thread, endpoint and cast/preview witness against current state. A switcher panel, missing/dead endpoint, changed actor/cast/publication or ended console refuses with one notice and no write. Use Endpoint.WriteInput for the one trusted encoded chord; it admits to the exact endpoint FIFO, not whichever presenter endpoint is selected later. This bounded in-memory enqueue may occur under the short console identity lock; no native IO or waits under that lock. Never send via a generic current-focus writer after validation. Admission is not dispatch acknowledgement; only the existing transaction acknowledges and clears. On refusal or uncertainty do not retry automatically.

Stateful test seams reuse fake ptychild/terminal FIFO writers, the existing submissionCastFake preview fixture and headless nvim transaction transport. Hold publication/input completion and replace identities to prove ordering rather than using sleeps. No external service is added.

## Operator projection

| Internal condition | What the operator sees | One action |
|---|---|---|
| No eligible feedback, terminal-only notes, or unpublished preview | No ⏎ in that cast group | Finish eligible Chrome feedback |
| Acknowledged eligible Chrome feedback and live actor in view | ⏎ beside Chrome controls | Click ⏎ |
| Existing submission in progress | Existing sending status and locked draft; repeated action uses existing busy suppression | Wait for that result |
| Feedback/paint changed or an editor/drawing remains open | Existing transaction refusal; draft and feedback kept | Finish/review feedback, then click again |
| Drawn target replaced or endpoint unavailable before enqueue | Explicit target-changed/unavailable notice, no send | Select the intended live actor, then retry |
| Existing uncertain transaction outcome | Existing request-ID recovery/status behavior, never a fabricated success | Use the existing transaction recovery action |
| Confirmed acknowledgement removes the final eligible feedback | ⏎ disappears after the refreshed projection | None |

No new transaction-visible state. Paused/live Chrome eligibility follows existing hotkey rules; do not infer it from the pause icon or annotation-creation permission. The status button does not bypass draft or Board guards.

## Implementation and verification (one close boundary)

### 1. Shared shortcut, red first

Files: cmd/internal/workbenchshortcut/shortcut.go, shortcut_test.go, render_lua.go as required; generated nvim/workbench_actions.lua; nvim/init.lua; cmd/internal/termcmd/passthrough_test.go; existing wrapcmd workbench tests; nvim/dom_submission_integration_test.lua.

- [ ] Add failing encoding/role/fullscreen-policy tests for the new shared chord, with unchanged-other-chord negative controls. Verify it consumes once, names PairSendDOM and preserves focus.
- [ ] Run the focused tests and retain their expected failure; implement only the shared binding and single-chord terminal reservation, regenerate Lua and remove the manual duplicate map.
- [ ] Prove the real generated map reaches PairSendDOM in normal/insert and routed review use; existing transaction fixture checks draft bytes, selected notes/sketches, busy suppression and unchanged-draft clearing. No duplicate dispatch.
- [ ] Run go test ./cmd/internal/workbenchshortcut ./cmd/internal/wrapcmd ./cmd/internal/termcmd -run 'Workbench|Chord|Shortcut|PassesThrough' -count=1 and make test-submission-transaction. Commit the bounded routing change.

### 2. Eligibility and rendering, red first

Files: cmd/internal/submission/preview.go and preview_test.go; cmd/internal/couchtty/console_submission_preview.go and its fixture tests; reserve.go/reserve_test.go; console_presentation.go.

- [ ] Add table/transition tests: empty, note-only, sketch-only, mixed, off-view-only, dirty/publishing/failure, stale accepted publication, delivery/removal and cast disposal/replacement. Assert no terminal notes qualify.
- [ ] Add independent LIVE/Chrome rendering tests across widths, including exact fit, clipped glyph, Unicode columns, both groups and no pending feedback. Empty/clipped controls have no clickable span.
- [ ] Implement the scalar projection and atomic publication from the existing transition owner, independent group models and spans. Keep manager-before-console order; no manager lock from painting.
- [ ] Count projection/presentation work for N and 2N feedback fixtures: no new Board enumeration, native calls, filesystem reads or selection clones per paint; fixed scalar work independent of feedback count. Run go test ./cmd/internal/submission ./cmd/internal/couchtty -run 'Preview|Status|Send' -count=1. Commit.

### 3. Exact endpoint click, red first

Files: new cmd/internal/couchtty/console_send.go and console_send_test.go; console.go (stored accepted witness), terminal.go, terminal_input.go; existing terminal mouse fixture tests.

- [ ] Test the real mouse route with a published frame and fake child: one plain left press admits the exact shared chord to the original endpoint; release, right-click, drag, wheel, clipping and neighboring spans admit nothing. Preserve existing cast controls.
- [ ] Hold frame acceptance/enqueue boundaries and replace actor, endpoint, cast, host/publication or focus. Prove stale controls never send to a successor, and ended-input/queue refusal does not clear any draft/feedback or retry.
- [ ] Implement exact witness capture and bounded endpoint FIFO admission. Do not add a background route worker or bypass the existing wrapper/draft admission.
- [ ] Exercise existing stateful wrapper/draft routing and payload-equivalence tests; repeat clicks reach the existing busy guard rather than another transaction. Run go test ./cmd/internal/couchtty ./cmd/internal/terminal -run 'Send|Mouse|Status|Input' -count=1. Commit.

### 4. Documentation, acceptance and landing

Files: atlas/webcast-submission.md, atlas/broadcast.md, existing help/key catalog generated surfaces; issue Log. Update atlas/index.md only if adding a page (none planned).

- [ ] Document ⏎ visibility, wider hotkey focus availability, terminal-only limitation/#859, and unchanged transaction recovery. Keep #856 browser layout untouched.
- [ ] Audit changed reader inputs individually; update inventory certificates only for actual input changes. Run make test-affected-inventory before close.
- [ ] Push the final head, run sdlc local-check run --pre-close, and read its selected check/result. The train owns the full suite. Use sdlc close for the mandatory fresh-context boundary review; fix/dispute findings with evidence, never self-label local gates.
- [ ] Send the hub real HEAD and a brief live-check recipe: make one eligible Chrome note/sketch plus draft, click status ⏎ from an agent/right pane, verify one equivalent delivery and feedback disappearance; confirm no pending/no Send and terminal-only notes/no Send. Stage or open a visible window only when the hub authorizes it. Record actual acceptance, not a planned pass.
- [ ] On SHIP and required acceptance, use sdlc merge --yes (main integration and gates remain enabled), verify merge/tracker/archive, report PR plus real merge hash to hub and stop. #843 is the next separate issue.

## Cost and adversarial guards

Feedback-size N never enters a paint/click scan: O(1) summary, fixed-size chord, existing bounded endpoint queue. Run generated preview-transition sequences to check stale-ack/disposal invariants and render-width sweeps to prove no invisible hit target. Hold writes and replace endpoint generations in stateful fakes; assert byte count and exact destination rather than just callback count. Any routing change beyond the explicitly approved single chord requires a revision before implementation.

## Revisions

### 2026-10-23 — use #849 press selection, not preview acknowledgement

Hub7dcaf909 approves the plan with this required correction. The original preview-derived visibility design above is superseded wherever it appears (Architecture, pure/integration tables, operator projection, tasks2/3 and cost tests). #849 removes the preview gate because live pages can prevent preview completion indefinitely. An unpublished or failed preview must never hide otherwise eligible Send.

The single eligibility owner is #849's cast-scoped press-selection rule: open notes at current revisions and finished labelled sketches. Show ⏎ when that selection is non-empty. Reuse its treatment of delivered/resolved/unfinished feedback; do not recreate a competing filter in #842. Terminal-only NoteBoard notes remain ineligible. Button visibility does not require preview publication, host paint or an acknowledgement of preview; actual submission remains subject to the existing transaction's guards as changed by #849.

Replace the proposed PreviewState projection/publication with a small non-empty projection of that press-selection owner. Coordinate exact API, locking and cheap change notification with pair:6 before selecting implementation files. Preserve the no-per-paint IO rule and existing lock order; do not add an acknowledged-preview cache, host/publication witness or extra preview worker. Accepted-frame click authority still binds the original actor/thread/endpoint and exact cast lifetime; selection changes are handled by #849's press-time selection, not by a stale preview refusal introduced here.

Task2's regressions now cover empty, open notes at current revisions, finished labelled sketches, mixed selection, resolved/delivered feedback and unfinished/unlabelled sketches according to #849; add a live-page control with preview publication permanently unavailable that MUST still show Send. Eligibility changes with note edit/resolve, sketch completion/label and delivery; cast end/replacement cannot leak the old availability into a successor. N/2N controls count the chosen seam's actual work; do not claim O(1) without a measured scalar/change-driven seam. Task3 replaces host/publication replacement tests with cast-lifetime replacement and press-time feedback revision tests. The remaining rendering, exact endpoint, shared chord, transaction-equivalence and negative input tests stand.

Operator projection: eligible press selection plus a live actor shows ⏎ even when the preview never completes; empty selection hides it. The existing busy/refusal/uncertain transaction result and recovery actions stay unchanged. Document this corrected contract, not the superseded preview wording.

Implementation waits for #849 on main. No stacking or copying its unlanded source. Once pair:6 names the final seam and provides a verified landing hash, integrate main, reconcile this plan's concrete file list and run change-code/plan-quality before code. Coordination sent to pair:6 as receipt2270e3b0-cfcb-4260-9d26-b3d44009e271; awaiting reply. This is a design checkpoint, not implementation evidence.

### 2026-10-23 — exact #849 seam received; landing prerequisite remains

Pair:6 c68595da reports #849 is not on main (pushed5cd9a3e78, pre-close queued). Cast.SubmissionSelection takes an in-memory Board selection atomically: open current-revision notes plus finished labelled sketches. Selection.PressSelection(delivered) applies exact delivered versions and the oldest-eight limit, but Store.DeliveredFeedbackVersions scans manifests on disk and must never run per paint/refresh for this button. Raw selection can retain a delivered sketch until the live display mask hides it. #842 will therefore reuse/add a Board-owned read-only non-empty projection using the same exact sequence/digest feedback-display mask, so acknowledgement cannot leave a false Send. Do not duplicate a durable delivered ledger or call the store.

SubscribeSubmissionCandidates supplies a coalesced wake for note create/edit/state, sketch label and clear; delivery mask changes also need explicit qualification. Observe the projection outside Couch locks; publish/read only its small result in presentation. Exact scheduling and file names will be reconciled against landed main at change-code, reusing an existing owner where available. Count actual event-driven reads/work rather than promise an unmeasured O(1) selection copy. Tests must include a permanently unavailable preview, delivered-sketch masking, current note revisions, unfinished sketches and cast replacement. No code or #849 branch changes requested. Wait for pair:6's verified landing receipt.

### 2026-10-23 — landed integration and concrete projection owner

Verified PR578 MERGED at5d8fd7648caa64ecfcb6d9e2d20716a881d0397c, fetched origin and merged main into this issue branch (no stacking). Hubf896637c authorizes integration/change-code. Landed feedback_display.go already calls submissionChangedLocked when the delivered mask changes, so SubscribeSubmissionCandidates covers confirmation too.

Concrete files replacing the superseded preview rows: add Board.HasSubmissionFeedback in cmd/internal/webcast/board/submission.go plus colocated tests; add its thin Cast forwarding method in cmd/internal/webcast/cast_submission_candidates.go. Under Board.mu, this projection reads current open notes/current revisions and finished labelled sketch candidates, excluding exact delivered sequence/digest versions via the existing feedbackDisplay maps. Share candidate eligibility with selectionLocked where necessary rather than diverging from #849. The read performs no filesystem/browser IO and may short-circuit at the first eligible item; worst-case O(N) per candidate-change observation, not per paint.

Reuse the existing tracked cast update watcher in couchtty/console_webcast.go: subscribe to SubmissionCandidateUpdates, take an initial projection and refresh it on coalesced candidate wakes outside Console.mu, then apply the scalar through the existing runTerminalCommand path with exact cast-pointer revalidation. Store the scalar in that cast's existing webcastState; statusModelLocked reads it in O(1). Do not add a goroutine, timer, Store scan or dependency on preview progress. The cast-end path and a successor's fresh state reset the value. Tests drive a preview-unavailable cast and exact delivery-mask wake, and count projection reads over repeated paints versus candidate wakes.

The accepted render carries its actor/thread/endpoint plus cast identity into the drawn Send witness. No host/publication witness survives this revision. Click admission validates the physically confirmed frame and current cast/actor identity; press-time transaction chooses the current eligible feedback rather than pinning old note revisions from rendering. Implementation is still gated by change-code's plan-quality/estimate checks.
