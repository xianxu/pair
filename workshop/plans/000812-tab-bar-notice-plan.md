---
collaboration-mode: discussion
---

# Tab-bar notice removal implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 for execution strategy. Use the existing console fixtures for integration and the binary-owned close review for acceptance.

**Goal:** Give tab-bar space entirely to controls and slot chips while preserving the latest notice in the switcher and surfacing actionable asynchronous failures in the existing attention list.

**Architecture:** Remove the general feed notice from the tab-bar model and common row renderer. Keep the bounded feed as the sole history owner; project its newest retained message into the switcher without overwriting menu-operation notices. Explicit producer calls raise actionable events through the existing bounded AttentionNotify lifecycle. No automatic delivery, desktop output, background worker, durable storage or new attention kind.

**Tech stack:** Go, existing couchtty console/renderer fixtures and couchcore.AttentionList.

## Decision and approval boundary

The operator already chose removal from both layouts. This plan still needs TL approval before code, specifically the global-attention selection behavior below. The audit found that the general feed is currently available through Ctrl+Y but is not drawn by RenderMenuView: merely preserving MenuState.Notice would lose the displayed exit/error text entirely.

Alternatives considered:

1. Clear Notice only when constructing StatusModel: smallest patch, but leaves rendering consumers able to append it and does not satisfy switcher display or actionable warning visibility.
2. Remove the rendering path, preserve the feed in the switcher, and explicitly route asynchronous actionable producers to attention (recommended).
3. Raise attention for every setNotice/publishNotice: rejects the operator's distinction between an obligation and routine progress, success or navigation feedback.

This is provisionally full flow because the producer audit crosses console and composition-root callbacks. No estimate before change-code's plan-quality gate. One atomic close; no artificial Mx boundary.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| StatusModel / renderStatusCells | cmd/internal/couchtty/reserve.go | modified: remove general Notice and its separator/overflow append path |
| RenderStatusBar / collapse planner | cmd/internal/couchtty/status_layout.go, status_collapse.go | modified: stop carrying a final-row notice |
| MenuState / RenderMenuView | cmd/internal/couchtty/menu.go, menu_render.go | modified: bounded latest-feed projection, separate from operation notice |
| bounded notice-attention projection | cmd/internal/couchtty/notice.go | new pure helper: explicit source facts to a safe attention event |
| Feed | cmd/internal/couchtty/notice.go | retained owner: collapse, capacity, full message and Ctrl+Y history |

The renderer remains pure and receives explicit snapshots (ARCH-PURE). Persistent capability warnings, capture status, attention badge, casts, slot status summaries, hit spans and grouping are separate surfaces and remain intact. Feed owns the text once; the menu projection is derived rather than a second retained history (ARCH-DRY).

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| statusModelLocked | cmd/internal/couchtty/console_presentation.go | modified | console snapshot into pure tab-bar input |
| menuSnapshot / showMenuPublishing | cmd/internal/couchtty/console_menu.go | modified | copy latest retained feed entry under console mutex |
| explicit attention-notice publisher | cmd/internal/couchtty/console.go, console_attention.go | new thin method | feed publication plus existing AttentionList event application |
| jumpToAttention | cmd/internal/couchtty/console_attention.go | modified | global/retired-target event selection opens existing switcher |
| source callbacks | cmd/internal/couchtty/console_*.go; cmd/internal/couchcmd/run.go, hubs.go, hub_move.go, lifecyclefailure_queue.go, message_service.go | modified | explicit asynchronous warning classification at its producer |

Reuse stateful console/menu fixtures and AttentionList tests. There is no new external service dependency. Do not call ObserveAttention while holding c.mu: its repaint re-enters console presentation. Prepare the bounded event under the existing owner, then apply/repaint after unlocking. Keep lock order c.mu then attention lock where current console projection already requires it; no reverse acquisition (ARCH-OWNERSHIP).

## Visible state and recovery

Actions: open switcher, inspect/copy notice, select attention, dismiss attention, retry the original operation through its existing control.

| Source / failure mode | Visible state | Recovery | Possible loss |
|---|---|---|---|
| Routine progress, success or synchronous key refusal | Latest message in switcher; absent from tab bar | Open switcher and inspect/copy | Existing bounded feed can replace/evict older text |
| Actionable asynchronous event with an exact thread | Existing bell/attention row plus retained switcher notice | Select exact thread when attached; otherwise open switcher | Existing event cap/12-hour expiry; no durable history promise |
| Global asynchronous failure without a canonical thread | Attention event labelled Couch, with no fabricated slot identity | Selecting opens switcher; inspect full text with Ctrl+Y | Same existing event cap/expiry |
| Prompt/question/hub --notify already represented | Existing attention state | Existing select/dismiss/retry behavior | Existing lifecycle unchanged |

Global selection must not silently jump to the currently focused actor. An exited/replaced exact target must open the switcher rather than produce another invisible notice. This is the explicit TL approval point. AttentionNotify selection/dismissal clears an event; the existing cap50, oldest-notify eviction and 12-hour expiry remain the owner. No live condition is invented for one-time errors (ARCH-FUNERAL).

## Producer audit and routing contract

Explicitly mark actionable warnings at their typed source, never by matching English substrings or treating every Kind=status as an error.

| Class | Executable anchors | Destination |
|---|---|---|
| Unexpected pane exit | Console.onExit / ExitNotice | exact exited thread, even after pane removal |
| Crash/capture/trace failure | installCrashReport; Console.SetInputTrace/SetMouseTrace; trace.go failure publication | global Couch event |
| Hub start, hub-move and cleanup warning | startExplicitHubs; installHubMoves; beginConsole; finishLifecycleAttempt | exact subject when provided, otherwise global |
| Stale session, supervision escalation / uncertain resend | newSessionStaleWatcher and supervision notice wiring in message_service.go | source binding when available, otherwise global |
| Continuation scan/operation failure | applyContinuationScan / finishContinuationOperation | origin address when available; retained row evidence stays |
| Async cast failure/recovery / guessing lockout | broadcastEnded; webcastUpdated/webcastEnded; cast-code rotation callback | global event with existing code-safe wording |
| Existing archive warning, hub notify, prompts/questions | notifyArchiveWarning; handleNotify; publishBlocked/publishQuestions | existing attention path only, no duplicate |
| Progress, copied confirmations, explicit cancellation, ordinary key feedback | setNotice call sites not marked above | feed/switcher only |

Audit every setNotice/Notify/publishNotice producer before implementation and record any additional actionable sibling in the Log. If classification requires new ownership/state beyond these rows, return to TL rather than silently widening the plan (ARCH-PURPOSE).

## Implementation and verification

### Task 1: Remove trailing rendering and preserve switcher display

- [ ] Add failing table tests through the console feed/status projection plus reserve_test.go/status_layout_test.go for collapse and wrap, one/two rows, wide/narrow/tiny widths, ordinary/overflow chips: a pending notice must not change tab bytes, height, chips or hit spans relative to an empty feed. Tests must still compile when StatusModel.Notice is removed. Controls and capability/capture warnings must retain their prior outputs.
- [ ] Add a console integration test using the real feed: an exit/error is absent from the tab bar, present in the switcher, and still copied fully by Ctrl+Y. Keep MenuState.Notice progress/error and copy-feedback precedence, menu bounds, selection and pointer extents correct when both notices exist.
- [ ] Run targeted tests and retain the expected red; remove StatusModel.Notice and its common renderer/layout path. Project the newest Feed.Messages entry into bounded switcher presentation. Do not clear/drain the feed or remove its independent copy semantics.
- [ ] Reconcile old status-notice expiry and overflow-order tests with the new product contract. Feed retention/expiry tests remain meaningful; no stale expectation that tab-bar output shows the notice.

### Task 2: Preserve actionable asynchronous signals

- [ ] Add failing stateful-console tests for each audit class, using exact thread identity or explicit global identity, plus negative controls for progress/success/key feedback and already-covered attention sources.
- [ ] Add a thin explicit warning publisher sharing existing feed publication. Assign one in-memory event ID per raised warning, bound body/title before applying to attention, preserve source-safe text and never include cast secrets. Use existing 50-event list/expiry owner; create no queue or timer.
- [ ] Route the audited asynchronous source callbacks. Keep desktop notification policy and source operation results independent of optional presentation; add no retries or native input.
- [ ] Test global and unavailable-thread selection opens switcher, exact attached-thread selection stays exact, dismiss/select/expiry/cap work as before, repeated updates cannot create unbounded state, and simultaneous render/publication does not deadlock.
- [ ] Run focused tests under race and bounded generated/fuzz sequences for untrusted notice text (controls, malformed UTF-8, long text), actor replacement and notice/attention/menu interleaving. Enforce hard byte bounds before rendering; do not rely on terminal width alone for zero-width combining input.

### Task 3: Document and qualify

- [ ] Update README.md tab-bar/switcher guidance and atlas/couch.md wrap/collapse descriptions: no trailing feed message, latest message in switcher, actionable events in attention. Keep atlas index link stable.
- [ ] Run complete couchtty and affected couchcmd tests, plus source inventory and make test-affected-inventory; audit each reader certificate individually if changed.
- [ ] Push the candidate, submit the declared recorded affected check (or full suite when selection requires it), and retain its exact head/result. No broad manual suite outside SDLC admission.
- [ ] Ask TL to coordinate a stage/live check in the operator's real console: collapse and wrap, synthetic/disposable exit and error, switcher copy, exact/global attention selection. Do not stage :0 or signal live slots without that go.
- [ ] After verification/live acceptance, close via the binary-owned review, resolve findings, merge and report the landed PR.

Targeted commands (refine test names when adding regressions):

```sh
go test ./cmd/internal/couchtty -run 'Test(Status|Menu|Console.*Notice|.*Attention|.*Clipboard)' -count=1
go test -race ./cmd/internal/couchtty -run 'Test.*(Notice|Attention)' -count=1
go test ./cmd/internal/couchcmd -run 'Test.*(Notice|Notify|Supervision|ArchiveWarning|Hub)' -count=1
make test-affected-inventory
```

## Cost and lifetime

Tab rendering loses work; existing actor planning remains unchanged. Latest-feed projection scans at most the existing feed capacity. Each explicit warning performs one bounded existing attention application over at most50 entries; N and2N publication tests count one projection/application per event and assert the same list cap. Text input is processed once under a hard byte cap. No additional per-frame source traversal, IO, goroutine or file family. All added values die with the console (ARCH-CONSTRAINTS, ARCH-FUNERAL).

No persisted format, core status enum, agent PTY admission or child-process lifetime changes (ARCH-EVOLVE). This is product behavior removal plus reuse of existing attention ownership, not a reason for a general notification redesign.
