---
collaboration-mode: discussion
---

# Tab-bar notice removal implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 for execution strategy. Use the existing console fixtures for integration and the binary-owned close review for acceptance.

**Goal:** Give tab-bar space entirely to controls and slot chips while preserving the latest notice in the switcher and surfacing actionable asynchronous failures in the existing attention list.

**Architecture:** Remove the general feed notice from the tab-bar model and common row renderer. Keep the bounded feed as the sole history owner; project its newest retained message into the switcher without overwriting menu-operation notices. Explicit producer calls raise actionable events through the existing bounded AttentionNotify lifecycle. No automatic delivery, desktop output, background worker, durable storage or new attention kind.

**Tech stack:** Go, existing couchtty console/renderer fixtures and couchcore.AttentionList.

## Decision and approval boundary

The operator already chose removal from both layouts. TL approved this plan, including global-attention selection, in receipt db842971-53f5-4660-a836-870b0d14de1a. The audit found that the general feed is currently available through Ctrl+Y but is not drawn by RenderMenuView: merely preserving MenuState.Notice would lose the displayed exit/error text entirely.

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
| noticeAttention | cmd/internal/couchtty/notice.go | new pure helper: explicit source facts to a safe attention event |
| Feed | cmd/internal/couchtty/notice.go | retained owner: collapse, capacity, full message and Ctrl+Y history |

The renderer remains pure and receives explicit snapshots (ARCH-PURE). Persistent capability warnings, capture status, attention badge, casts, slot status summaries, hit spans and grouping are separate surfaces and remain intact. Feed owns the text once; the menu projection is derived rather than a second retained history (ARCH-DRY).

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| statusModelLocked | cmd/internal/couchtty/console_presentation.go | modified | console snapshot into pure tab-bar input |
| menuSnapshot / showMenuPublishing | cmd/internal/couchtty/console_menu.go | modified | copy latest retained feed entry under console mutex |
| Console.Warn / publishWarningLocked | cmd/internal/couchtty/console.go, console_attention.go | new thin method | feed publication plus existing AttentionList event application |
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

Global selection must not silently jump to the currently focused actor. An exited/replaced exact target must open the switcher rather than produce another invisible notice. TL approved this selection behavior. AttentionNotify selection/dismissal clears an event; the existing cap50, oldest-notify eviction and 12-hour expiry remain the owner. No live condition is invented for one-time errors (ARCH-FUNERAL).

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

## Function-level regression strategies

| Risky function | Adversarial input / mechanical guard |
|---|---|
| renderStatusCells / RenderStatusBar / prepareCollapsedStatusChips | Generate actor/width/style inputs and compare snapshots with/without a pending Feed message: identical rows, geometry and hit spans; retain independent controls as the oracle. |
| menuSnapshot / showMenuPublishing / RenderMenuView | Drive real bounded Feed updates through the stateful console fixture, with simultaneous menu-operation/copy state; assert retained full text and pure bounded viewport/selection projection rather than reproducing layout internals. |
| noticeAttention (new pure function in notice.go) | Fuzz arbitrary notice bytes and explicit source identity; enforce fixed text byte bounds and safe controls, exact supplied thread/global identity, and no inference from body wording. |
| Console.Warn / publishWarningLocked (new methods in console.go) | Use existing console and attention mutex boundaries with channel-controlled goroutines to force warning-publication-before-selection and selection-before-publication; compare final event ownership and feed state to a sequential reference model. Count one bounded attention application per explicit warning at N/2N. No sleeps as ordering proof. |
| jumpToAttention | Generate exact, replaced and absent thread identities in the stateful console fixture; action oracle is exact attached target or existing switcher opening, never another focused actor. |
| asynchronous producer callbacks in the audit table | Trigger real source transitions with their existing stateful fixtures, asserting actionable failure yields one attention event while normal completion/progress yields none; already-routed events retain a single owner. |

The new exported Console.Warn takes a Notice whose optional Thread and Label name an exact source; empty Thread explicitly means global. Routine Notify/setNotice stays feed-only. noticeAttention maps one explicitly chosen warning to existing AttentionNotify. publishWarningLocked commits feed plus attention under the existing console-to-attention lock order, without calling ObserveAttention/repaint under c.mu; the public wrapper repaints after unlock. Global selection reuses onHotkey's existing switcher opening rather than adding navigation state.

### Task 1: Remove trailing rendering and preserve switcher display

Files: reserve.go, status_layout.go, status_collapse.go, console_presentation.go, menu.go, menu_render.go and console_menu.go, with colocated render/console regression tests.

- [x] Write generated render/projection regressions for the named functions and run them red against the current behavior.
- [x] Remove StatusModel.Notice and common trailing append paths. Derive a latest-feed switcher snapshot without replacing menu-operation/copy feedback, preserving Ctrl+Y's full retained source.
- [x] Run the regressions green; reconcile obsolete tab-notice expectations while retaining independent Feed lifetime/copy tests. Commit this coherent presentation change.

### Task 2: Preserve actionable asynchronous signals

Files: notice.go, console.go, console_attention.go and the audited console/composition callbacks, with colocated producer and attention tests.

- [x] Write the function-level pure, generated and controlled-order regression strategies above; retain behavioral red evidence before implementation.
- [x] Implement noticeAttention, Console.Warn and publishWarningLocked using existing Feed/AttentionList owners. Bound text before projection, assign one console-local event ID, and preserve source-safe cast wording.
- [x] Route only actionable asynchronous transitions at explicit source sites. Leave normal completion/progress and already-covered sources on their existing paths; no English-text classification, new desktop effects, retries or native input.
- [x] Implement exact/global selection via the existing switcher/notification navigation funnels. Verify sequential-model, cap/lifetime and controlled-order invariants, then targeted race/fuzz checks; commit the complete producer routing.

### Task 3: Document and qualify

- [x] Update README.md tab-bar/switcher guidance and atlas/couch.md wrap/collapse descriptions: no trailing feed message, latest message in switcher, actionable events in attention. Keep atlas index link stable.
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


## Revisions

### 2026-10-23 — approved with explicit noise guard

TL db842971-53f5-4660-a836-870b0d14de1a approves all surfaces and global/detached selection behavior. Only events requiring the operator belong in attention; a class that fires routinely during normal operation stays feed-only. Apply this producer-level distinction without parsing English text or duplicating already-routed events. Approval supersedes the earlier pending language; implementation still waits for change-code.

### 2026-10-23 — PQ-1 function-level verification correction

Plan-quality round1 confirmed the existing-behavior claims and architecture, but required test strategies tied to named functions instead of prose case inventories. Replaced Tasks1–2's enumerations with a function-level table, named noticeAttention/Console.Warn/publishWarningLocked, and specified channel-controlled ordering at existing mutex boundaries with a sequential reference model. This changes verification precision, not the approved source classes or visible behavior.

### 2026-10-23 — collapse anchor name corrected

The strategy row now names the existing prepareCollapsedStatusChips function; collapse is a branch of RenderStatusBar, not a separate renderCollapsedStatusBar function. No behavior or strategy change.

### 2026-10-23 — TL resequences local close before stage

Authenticated TL1e4df14f-c092-4148-8ad7-6921cd401d30 reserves the next stage after #811, and requests recorded check plus local close meanwhile. Run the binary-owned local close/review before real-console acceptance; retain acceptance as the final prerequisite before landing. The issue carries four short operator steps. Pending live/landing checkboxes remain honest; no code change.

### 2026-10-23 — boundary producer-class correction

BR-1 requires all local/remote/direct terminal completion owners to route actionable diagnostics once; ordinary successful recovery guidance stays feed-only. BR-2 requires optional typed Warnf alongside informational Logf through existing cast/recorder configuration. Warnf falls back to Logf for legacy consumers, and only one callback handles each diagnostic. Healthy fidelity, recovered overflow/resync/aggregate-close and transient drive-busy reports stay informational; changed Recovery retains uncertain-input attention ownership. A transient SwitchAgentResult cleanup-warning fact may preserve the producer's exact meaning without parsing its mixed Notice string; it adds no persisted state or lifecycle. Retain real recorded-cast success/error, local queue failure, recovered overflow and transient-busy regression evidence before re-review.

### 2026-10-23 — BR-3 complete source rule and console enumeration

Rule: unsolicited failures and failed accepted asynchronous operations that require explicit operator repair use attention at their typed source. Routine progress/success, automatic recovery, expected races/cancellation, safety transitions caused by ordinary operator layout changes, and synchronous admission/key/copy feedback remain feed-only. Already represented prompts/questions/notifications retain their owner. Mixed callback families must carry a typed source distinction; a callback name or English text never grants severity. Exact source identity is preserved when available, otherwise the event is global. Local/remote/direct terminal completion owners publish once.

Authenticated TL3a019860-25e1-4345-8318-72004b300014 explicitly keeps pointer-watch stop feed-only and approves warning routing for failed accepted async capability/drive operations. This is a source-class correction within the approved noise guard, not a new queue, condition or retry design.

Complete couchtty publication enumeration (every remaining production feed call and callback, grouped by source function):

| Sources | Classification and owner |
|---|---|
| onPointerClick (copy/create/enable/disable), copyLink, copyViewLink | Direct click/copy feedback; clipboard failure also has terminalError owner |
| pointerOff / pointerOffByWatch | Manual or layout-caused routine safety stop; feed-only per explicit TL decision |
| routeNotesMenu copy/resolve branches | Direct note-menu feedback |
| applyDrive / driveNotices | State transitions; actual requests retain visible hand badge and selectable menu |
| startDriveOperation pending/admission | Routine admission feedback |
| driveResult | Typed expected stale/closed/not-live/cancel/deadline races and success quiet; unexpected accepted-operation error warns |
| onWebCapabilityClick before worker | Synchronous transition/enqueue/copy feedback |
| onWebCapabilityClick current worker completion | Failure warns after existing exact cast/generation/attempt guards; successful off/copy quiet |
| runCodesSelection | Direct annotation/code-menu failure or success feedback |
| onCodeRotated | Current guessing/lockout warnings; stale/duplicate/manual rotations suppressed |
| toggleBroadcast / startBroadcast / stopBroadcast | Configuration/refusal/cancel/progress and explicit operator stop quiet |
| broadcastStarted / broadcastEnded | Current async start/tap failure or unsolicited end warns; ordinary live/stop confirmation quiet |
| SetWebcast callbacks | Typed Warnf warns; informational Logf stays feed-only; legacy fallback at source invokes only one callback |
| toggleWebcast / startWebcast / endWebcast | Direct key refusal, progress or explicit stop quiet |
| webcastStarted / webcastEnded / webcastUpdated | Async failures and changed Recovery warn; normal open/end/hand-close quiet; Recovery is sole uncertainty alert owner |
| focusWebcast / copyWebcastLinkOf | Direct focus/copy/link-not-ready feedback |
| SetReplayShares | OnReady is successful completion only; feed-only |
| acceptContinuationRequests / finishContinuationOperation | Scan failure episode, missed continuation and nonbusy failure warn; busy retry silent |
| SetInputTrace / SetMouseTrace / SetEventTrace | Trace open failures warn |
| traceDropped | Trace-only missing-endpoint diagnostic; no repaint recursion |
| onExit | Unexpected exit and forget failure warn with exact thread; expected exits quiet |
| newestPage / reportPrevious / detachActive / relaunchActive / parkActive / onChord | Direct navigation/selection/restart-key admission feedback |
| finishOperation / lifecycleCompletionWarningSource | Typed actionable successful-result facts warn once; ordinary recovery/fallback/NextCommand quiet |
| finishLifecycleAttempt | Queue-owned terminal diagnostics warn once; remote/direct retain their separate once owners |
| toggleAttention | Empty-list feedback cannot recursively raise attention |
| Notify / setNotice / publishNotice / publishNoticeLocked | Generic feed wrappers; classification belongs to actual source above or couchcmd enumeration |
| Warn / setWarning / publishWarningLocked | Explicit warning wrappers; existing cap/expiry, lock order and shutdown guard |
| presenter.Notify in desktop_notify.go | Separate desktop protocol, not a feed publication |

Chrome callback sub-enumeration remains the preceding BR-2 revision: recovered overflow/aggregate reports, successful fidelity, transient drive503, retryable retained draft cleanup and private-submission policy fallback are informational; explicit write/open/snapshot/segment/close, failed fidelity, notes reducer and drive502/500 failures are actionable. Uncertain input uses changed Recovery once.

### 2026-10-23 — composition-root enumeration and exact capacity-latch approval

| Remaining couchcmd producers | Classification and single owner |
|---|---|
| ScratchStartup manager callbacks | Normal-temp opt-out; no RAM/space/probe/identity safe disk fallback; successful sweep and retryable retained-entry cleanup errors stay feed-only. Coding remains usable; automatic retry owns recovery. |
| Scratch startup metadata setup | Explicit MetadataDirectory/ConfigureMetadata errors use a startup-scoped typed warning sink; live-server metadata remaining on disk until reboot is expected feedback. No new couchscratch callback API. |
| lifecyclePublisher diagnostic projection | Typed failed/refused/unsupported delivery outcomes, missing verified restart-drain slot metadata and projection capacity require attention; unavailable/cancelled recipient churn and uncertain canonical-receipt polling are routine/recovering feedback. |
| Renicer | Once-per-binding unrecoverable agent priority warns with retained exact Thread/slot label. Normal descendant renice, permission-denied/table observations retain their existing facts without new attention. |
| archive scheduler | Existing notifyArchiveWarning feed callback stays feed-only; pass/retention failure has separate actionable callback. |
| notifyArchiveWarning | Connected hub uses existing exact notification; absent/disconnected hub uses the same existing notification owner with a global target. One event, feed and existing desktop effect, no Warn+Notify duplication. |
| handleNotify / prompt / question sources | Existing attention lifecycle only. |
| supervision callbacks | Unconfirmed/unavailable resend, escalation, automatic-input stall and released-but-inspect diagnostics require attention; ordinary startup/nudge sends are broker-only. |
| crash / hub startup or conflict / startup adoption | Explicit actionable failure warns; exact source when retained, otherwise global. |
| direct dispatch / local and remote lifecycle queues / failed attach rollback | Diagnostic persistence and unproven cleanup warn at their existing terminal once owner; no dispatcher double publication for queue-owned attempts. |
| CLI-only diagnostic branches | Existing stderr; no console publication to migrate. |

Authenticated TLd40218f2-3381-4372-b703-fbbf844e9c74 approves exactly one capacity latch bool under the existing publisher mutex, set at first128-hub overflow, limiting that capacity warning to once per publisher lifetime. Capacity cannot recover within that lifetime. No other new source state, map, queue, lock or retry is approved. This is the only added suppression latch in this correction; existing source once/episode guards otherwise remain the owner.

### 2026-10-23 — BR-5 publication before test input

Asynchronous warning acceptance helpers must acknowledge actual attention publication before sending the opening key; queuing or callback admission is not publication. Audit all openWarningAttention callers, including synchronous producers, and force both early-input and already-published orderings in regression coverage. Keep production empty-list behavior unchanged. Qualification remains pending a corrected complete package, recorded check, clean close and coordinated real-console acceptance.

### 2026-10-23 — BR-6 all asynchronous assertions, not only shared helper

The publication acknowledgment rule applies to every async warning assertion, including direct attention-count tests and absence controls. Pane removal, phase change and dispatcher admission are intermediate states unless their owning function proves publication completed before them. Enumerate source→completion→assertion for all affected tests; force the removal-before-warning legal interleaving and its reverse without changing production behavior. Preserve the recorded/live acceptance prerequisites.

#### Completed async assertion enumeration

| Source / assertion | Actual acknowledgment |
|---|---|
| Unexpected exit | Exact warning source/body; SetForget after Warn; controlled assertion-first and publication-first |
| Inactive child exit | Matching LatestFeed under c.mu after atomic feed/attention publication |
| Broadcast failure | Actual attention count before cause/duplicate assertions |
| Broadcast operator stop | Following terminal-command completion before absence assertion |
| Unexpected webcast failure | Actual attention count before cause assertion |
| Normal browser/console end | Following terminal command and matching terminal feed before absence |
| Guessing after stopped cast | Stop callback joined through following terminal command before stale-generation negative |
| Recorded fidelity healthy/error | cast.Done then matching feed under c.mu before attention assertion |
| Accepted drive classifications | drivePending nil is deferred until after driveResult, including negatives |
| Accepted capability classifications | Returned completion channel, including cancellation/state-change negatives |
| Local queued lifecycle failure/success | Queue result follows publication; explicit duplicate completion afterward |
| Manual code rotation | Matching retained feedback under c.mu before absence |
| Terminal migration ordering | Actual badge predicates and publication/paint channel joins |
| Command standalone/later hub failure | Failure acknowledgment and current-VT badge, both warm-start orders |
| Command queued lifecycle diagnostic | Actual Warn completion channel then current-VT badge |
| Successful startup negative | restart-drained joins background attachment before absence |
| Edited restart warning/copy | Terminal receipt, actual switcher cells, exact full OSC52 output |
| Legacy persistence negative | Actual new-event badge; service/console shutdown before disk comparison |

Remaining assertions invoke synchronous handlers before observing state: crash installation, supervision callback, explicit lifecycle completion, archive tick, renicer and independent barrier, scratch metadata setup, trace setters, cast-start completion handlers, current/stale guessing callbacks, webcast recovery updates, continuation scan, pointer visibility stop, typed result handling, direct publication/expiry/bounds/projection. No asynchronous negative relies solely on phase.

### 2026-10-23 — TL live-evidence allocation

Authenticated TL be810f58-06be-43c0-ad23-910fa12dd692 chooses live checks for tab-bar layouts, global startup-warning selection, retained switcher message/full Ctrl+Y, and attached hub notification selection. TL triggers the warnings; operator only observes/selects. No spare disposable worker is available and no new slot is created. Unexpected-exit exact-source publication and detached-target navigation use the deterministic controlled-order/race/native Console regressions, not a claimed live check. This allocates evidence without waiving those behaviors. Actual operator results and recorded full-suite pass remain pending before landing.

### 2026-10-23 — live restart-noise correction

The initial live restart FAILED: per-slot borrowed attachment ownership diagnostics and session freshness skipped checks are routine feedback, not attention obligations. This supersedes earlier producer-table classification of borrowed adoption and freshness diagnostics. Exclude copied StartResult.Record.AttachmentNotice when determining actionable successful-start facts; retain any AcceptedConfiguration publication warning, explicit cleanup and unproven-state diagnostics. Initial adoption uses the same typed distinction. Freshness-skip output uses the feed; actual stale reminders retain their existing generated-message path. Add real multi-slot borrowed reattach and captured diagnostic regressions, preserve full feed/clipboard text, and join background completion before absence assertions. No new state, queue, lock or English filtering.

TL2e2303e2 reports operator acceptance of tab bar, global selection and Ctrl+Y conditional on the noise correction, with attached-target result still to confirm. They explicitly require recorded suite, re-close and landing, with no second live check. The first live failure is retained with screenshot provenance in the issue Log and never relabeled a pass.
