---
collaboration-mode: discussion
---

# Zellij reply hardening implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development for bounded independent tasks, or superpowers-executing-plans for session-warm work. Steps use checkbox syntax.

**Goal:** Reject incompatible or contradictory native observations before a query-derived one-shot action, so uncertainty produces a visible refusal instead of choosing a different target or fabricating absence.

**Architecture:** Keep the existing native CLI, runtime seams, local ownership records and periodic observers. Add pure request-specific validation and a bounded two-observation admission step to one-shot consumers. Reuse positive local identity evidence; do not introduce a proxy, daemon, native build, automatic retry loop or sleep.

**Tech stack:** Go, existing Neovim Lua modules, native Zellij 0.45.1, stateful fake CLI, existing SDLC suites and disposable native fixture.

**Status:** R1 proposed; implementation awaits TL approval and then the full-flow plan-quality gate. #640 is a separate diagnosis closeout; #636 follows this issue.

## Scope and honest guarantees

The observed fault is within one Zellij server: late work addressed by a reused numeric client ID reaches another connection. It produces wrong-kind replies, early disconnects, valid same-kind replies with another tab/flag/time projection, and silently incomplete session discovery.

Two independent observations mean two completed native invocations or an existing independent OS/local identity fence, not two parsings of the same bytes or two reads of the listing cache. A second observation confirms an otherwise valid first observation; it is never an attempt to turn a failed observation into success. The first invalid/unknown result terminates admission.

The native JSON/table protocols do not carry a request nonce or consistently carry session identity. We cannot authenticate arbitrary identical, valid stale replies by comparing them twice. Do not invent a field or claim this work supplies Byzantine safety. Guarantee: malformed/wrong-kind/wrong-addressable-target replies and disagreements are unknown/refused; observable context is independently checked before effects. Document the residual indistinguishable/correlated-reply risk and do not present the synthetic 3% sample as an independent production fault probability. This limit requires explicit TL acknowledgment with this plan.

No Pair-held lock can authenticate another process's native response: native clients/background jobs do not participate in such a lock. Keep existing per-action return-store locks to serialize Pair effects, but do not add a host-wide lock or absence scan as a substitute for response validation.

## Caller classification (becomes the atlas table)

Each row names its final disposition, exact observation and effect boundary. Retained periodic/cosmetic paths are not silently included in the two-read mechanism.

| Caller | Classification | Admission/evidence rule and effect withheld |
|---|---|---|
| Message listing poller, event-driven inventory projection | Re-checked | Keep current cadence/cache/error policy; a projection alone must not authorize a cold launch, attach, reap or input. |
| Wait-watcher, PID/start liveness, quiescence | Re-checked | Existing registry/OS checks and two OS-backed absence observations remain the independent lifetime fence; no new query timer. |
| Title poller, wrap dim, opener viewport | Re-checked | Preserve existing next-tick/next-invocation/cosmetic fallback behavior; explicitly record that these are not action authority. |
| Peer viewport advisory check | Re-checked | Keep existing advisory uncertainty policy and 500ms freshness; delivery still requires its existing independent binding/composer authority. No claim that viewport checks authenticate native replies. |
| Launcher picker/list/name/layout admission; one-shot discovery | Hardened | Strict expected reply validation. Before absence enables reuse/cold launch, independently confirm no exact native server using the existing owner/quiescence seams; a listed/OS contradiction is unknown. Listing two times alone cannot establish absence. |
| Warm attach/client count; owner/quit/recovery proof | Hardened | Strict client table/pane command validation; confirm decision-relevant client/owner facts and same server generation before effects. Reuse confirmStillDetached and existing OS fences rather than nesting another confirmation pair. |
| Layout focus/toggle/split/tab forwarding; terminal shortcut fallback | Hardened | Native-derived target/state needs matching observations of selected IDs, role, relevant geometry/fullscreen/tab facts. Positive session/live-process-validated sidecar fast paths remain local authority and do not add native queries. |
| Current-tab layout capture/restoration | Hardened | Reject missing/wrong schema; bind tab identity where the pane/tab response exposes it. Confirm layout name/dirty/tab facts before a swap decision. Never substitute empty/default tab info. Preserve genuine partial progress if a later stage becomes unknown. |
| Go draft routing; Lua workbench routing/review poke/DOM pin/send | Hardened | Require a unique matching positive pane identity; confirm fallback pane selection before any focus/write/submit. Preserve DOM's recipient and final target revalidation. No positional fallback after an uncertain target. |
| Copy-on-select auto-quote, clipboard-to-pane, query-selected flash | Fail-closed | Confirm source role/ID and unique draft target before staging, flash, focus or paste trigger; remove move-focus-down fallback. Ordinary OS clipboard mirroring of the user's selection already occurs before orchestration and remains a successful copy even if auto-quote refuses. |
| Couch restart-key role probe | Hardened | Strict exactly-one-client table plus matching focus/role/context read before choosing relaunch versus routing; disagreement consumes/refuses the candidate with an explanatory notice. |
| Couch note anchors | Hardened | Confirm relevant pane geometry/role/file identity inside the same captured-frame context; mismatch retains the original screen anchor and explicitly reports why semantic anchoring was refused. |
| Review show/hide visibility decision | Hardened | Preserve exact true/false/exit forms; confirm the decision before toggling visibility, preserving activation/recipient checks. |

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| Request-specific pane/client/tab observations and mismatch error | `cmd/internal/zellijreply/reply.go`, `panes.go`, `clients.go`, `tab.go` | new |
| Pane identity and tab correlation fields | `cmd/internal/zellijpane/zellijpane.go` | modified, additive fields where exposed by native output |
| Action evidence projection | Existing caller pure planners/selectors in `layoutcmd`, `clipcmd`, `draftroute`, `couchcmd`, `launcher` | modified |
| Lua reply validation/admission helpers | `nvim/zellij_observation.lua` | new |
| Reply conformance cases | `tests/fixtures/zellij-replies.json` | new, shared by Go and Lua tests |

`zellijreply` owns strict native response contracts; the existing tolerant `zellijpane.Parse` remains available to unchanged cosmetic/periodic consumers. Unknown additional native fields are permitted, but required decision fields must be present and correctly typed. Missing is never false/zero/empty evidence. Do not replace one permissive parser globally and accidentally change the retained observers.

Clients require the exact normalized header, valid numeric client IDs and terminal/plugin IDs, nonduplicate rows, and valid row shape. A header-only table is a valid zero-client observation; an empty body or unrelated JSON is not. Pane replies require the native array/object shape supported by the actual request, typed IDs/flags and requested command/geometry fields. Tab info requires its actual tab/layout/dirty fields rather than accepting any valid JSON object. Validate an explicitly requested pane/tab and reject an explicitly contradictory identity.

Action evidence projections contain only fields on which the decision depends; sorting independent rows is allowed, discarding target/role/tab/command/state differences is not. Do not compare cosmetic titles or continuously changing output when the action does not use them. The second snapshot's admitted facts, not a newly read third snapshot, feed the existing planner.

Lua needs the same boundary policy without a new public query proxy. One shared Lua module owns its few native decode/confirmation paths; common Go/Lua fixture cases are the executable agreement about missing/wrong fields and identity. Business selectors continue to live with their existing owners. No generic validation DSL/schema engine or new transport is introduced. ARCH-DRY/ARCH-PURE.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Bounded confirmation of projected evidence | `cmd/internal/zellijreply/confirm.go` | new | Two existing caller reads, validation, projection, equality; no retry/sleep/background task |
| Native query consumers | Existing runtime methods in launcher/layoutcmd/termcmd/wrapcmd/clipcmd/couchcmd | modified | Existing CLI argv and local process/artifact proof |
| Lua admission helper | `nvim/zellij_observation.lua` with existing `zellij_trace` action seam | new | Existing synchronous/asynchronous native commands |
| Stateful fake Zellij | `tests/lib/fake-zellij.py`, fixture adapter in `cmd/internal/zellijtest/` | new | Existing PATH/subprocess boundary, private model/effect-log folder |
| Native refusal qualification | `cmd/internal/layoutcmd/fullscreen_reply_live_test.go` plus retained #640 raw reproducer | new | Existing ControlledZellij/testpair ownership; unchanged installed native binary |

The fake owns sessions, panes, clients, tab/layout/focus and an effect log; normal replies derive from that state. Fault events substitute a reply from another same-session observation or wrong kind at an explicit request ordinal, close early, mutate focus between reads, or change exact process identity. Mutations update the model and append their exact target. This is a stateful external double, not just an argv expectation queue. Use Python already required by the repository, no package installation or per-test Pair build. Go and headless Lua exercise their actual production CLI seams with isolated HOME/cache/socket/model paths. ARCH-MOCK.

## Admission and effect ordering

| State/event | Next state | Permitted effects |
|---|---|---|
| Start, valid first observation | Candidate | Second independent observation only |
| Start, malformed/wrong-kind/unknown read | Refused | Existing error/notice/log only; no second read, mutation or fabricated absence |
| Candidate, compatible matching confirmation and local identity fence | Admitted | Existing planner's explicit target effects |
| Candidate, disagreement, missing target, timeout/cancellation, session/process/tab change | Refused | Error/notice only; no fallback target or native mutation |
| Admitted, mutation acknowledgment/error/outcome unknown | Existing partial-progress state | Preserve known completed stages/records; stop subsequent unsafe stages, report uncertainty; do not retry/rollback externally applied effects |
| Finished/refused, late completion | Finished/refused | No new effect; lifetime ends with caller |

No in-flight confirmation work outlives its caller. Validate before modifying fullscreen return records, quote files, clipboard routing targets or note anchors. The existing native mutator's empty-success contract is not a read response and must not be passed through a JSON validator. This plan does not claim to authenticate effect acknowledgments, prevent every native post-admission race, or safely replay an unconfirmed mutation. ARCH-ORDER/ARCH-SECURE.

### Confirmation helper contract

The thin Go helper has this contract (the equivalent Lua helper has the same early-exit ordering). Its projection callback performs strict parsing and returns a canonical representation of the decision facts; it is not a hash or proof of native authorship. Caller-local identity checks remain in the admitted observation path.

```go
func Confirm(ctx context.Context, read func(context.Context) ([]byte, error),
    project func([]byte) (string, error)) ([]byte, error) {
    first, err := read(ctx)
    if err != nil { return nil, err }
    candidate, err := project(first)
    if err != nil { return nil, err } // never query again after invalid evidence
    if err := ctx.Err(); err != nil { return nil, err }
    second, err := read(ctx)
    if err != nil { return nil, err }
    confirmed, err := project(second)
    if err != nil { return nil, err }
    if candidate != confirmed { return nil, ErrMismatch }
    if err := ctx.Err(); err != nil { return nil, err }
    return second, nil
}
```

A matching result is only one admission ingredient; expected session/target/local owner facts must also match. The callback cannot execute a mutation or write an authoritative record. Existing warm/DOM transactions that already have this ordering get strict parsing and projections at their two reads instead of calling this helper around an existing two-read operation.

## Operating envelope and lifecycle

- Interactive one-shot work: at most two reads for each newly protected decision. Reuse existing warm/DOM confirmations, do not multiply them to four. A multi-stage restore has separate decision points bounded by its existing four swap advances. No queries per ordinary key/output chunk.
- Keep existing operation/context deadlines and the CLI's existing five-second query cap; do not extend them to absorb the second read. Couch restart-role admission remains within its current one-second caller budget. Measure healthy/failed-path latency; a bounded refusal under load must be visible, and systematic healthy-path refusal requires redesign rather than increasing the deadline. Native performance figures in old comments are not a new measured budget. ARCH-CONSTRAINTS.
- No new workers, persistent cache, lock family, protocol endpoint, token/nonce file, production telemetry stream or credential. Projected evidence dies with the call; existing failure logs retain their existing owner/retention policy. Fake roots are test-owned TempDirs; native roots/server/client/noise producers are canceled and joined before removal. ARCH-FUNERAL.
- Existing session/pane IDs and supported context attributes correlate requests where available. No shell interpolation; all native calls remain argv arrays. Do not promise a native nonce that does not exist. Sensitive prompt/screen/clipboard contents do not go into refusal diagnostics.

## Alternatives considered

1. **Chosen:** narrow validation plus fixed independent confirmation and existing local identity fences. Fits operator option(c), exposes uncertainty and preserves the native transport.
2. **Schema validation alone:** cheaper but accepts same-kind inconsistent target/focus/owner facts and does not catch false absence.
3. **Serialize or retry all CLI calls:** rejected. The #640 sequential control also misroutes; native writers ignore a Pair lock, and retries can conceal uncertainty or duplicate effects. A proxy/native rebuild is explicitly outside scope.

## Implementation and verification

Use @superpowers-test-driven-development per task. Commit red evidence/results to the issue log and checkpoint each coherent green change. SDLC owns all milestone/close reviews; do not add a duplicate review boundary.

### M1 — Native observation contracts and lifecycle admission

**Create:** `cmd/internal/zellijreply/{reply,panes,clients,tab,confirm}.go` with colocated unit tests; `tests/fixtures/zellij-replies.json`; `tests/lib/fake-zellij.py`; `cmd/internal/zellijtest/fake.go` and tests.

**Modify:** `cmd/internal/launcher/zellij.go`, `zellijparse.go`, `session_owner_os.go`, `osruntime.go`, `session_handoff_os.go`, `layoutflow.go`; `cmd/internal/couchcore/artifactcollision.go`, `resume.go`, and their relevant existing tests. Change one-shot absence admission at its effect owner, not periodic `ProjectSessionPresence` merely to make a projection look stable.

- [ ] Add pure fixtures for empty/JSON null/wrong kind, malformed/missing required fields, valid header-only zero clients, duplicate IDs, mismatched requested IDs/tab and valid native controls. Tests expect typed unknown/mismatch, never empty-success fabrication.
- [ ] Add fake-driven lifecycle tests: a valid foreign-kind client reply must not prove detached; two different valid same-kind client tables must refuse attach; conflicting owner pane command/scope facts must refuse; a successful listing omission with an exact live server must not authorize cold launch/name reuse/recovery. Assert zero attach/launch/kill/native-binding mutation effects. Preserve valid owned/absent controls.
- [ ] Run targeted tests RED before production changes: `go test ./cmd/internal/launcher ./cmd/internal/couchcore -run 'Test.*(Zellij|SessionOwner|Warm|Detached|Reply)' -count=1` (narrow to the new names during iteration).
- [ ] Implement pure contracts and bounded confirmation, then wire these lifecycle owners. Reuse existing OS generation and detached revalidation rather than duplicating those fences. Preserve underlying errors/unknown outcomes; no automatic retry on mismatch.
- [ ] Re-run the targeted cases GREEN and a targeted `-race` pass for the fake/stateful adapters; verify no additional periodic queries. Commit and run `sdlc milestone-close --issue 647 --milestone M1` with evidence and updated atlas surface.

### M2 — Go one-shot action and clipboard safety

**Modify:** `cmd/internal/layoutcmd/{layoutcmd,rightpane}.go`, `cmd/internal/termcmd/run.go`, `cmd/internal/draftroute/route.go`, `cmd/internal/clipcmd/{run,runtime}.go`, `cmd/internal/couchcmd/{shortcut_focus,note_panes}.go`, `cmd/internal/couchtty/console_noteanchor.go` where refusal presentation requires it; colocated existing/new tests.

- [ ] Add one fake integration test per hardened row/caller, with both wrong-kind and valid same-kind displaced observation. Cover fault on first read, on confirmation, focus/tab/pane change between reads, and the unchanged healthy control. Assertions inspect actual model/effect log and existing sidecars, not just return codes.
- [ ] Clipboard RED: uncertain source must not flash/hand off; uncertain/missing/disagreeing destination must not stage quote, focus, move-focus or write Ctrl-_. Valid copy mirroring remains intact; terminal/nvim source skip behavior remains intact. Explicitly test focus failure prevents paste trigger.
- [ ] Layout RED: no native mutation/return record change on incompatible preflight. Confirm current-tab layout before decisions; mismatches after a completed stage preserve/report partial progress instead of continuing a guessed restore. Test same-kind different tab/target and valid matching snapshots.
- [ ] Restart/note RED: mismatch cannot select relaunch or emit terminal input; note mismatch keeps the screen anchor rather than inventing semantic file/line context.
- [ ] Implement using shared strict validators and projected admission. Keep positive local sidecar fast paths. Remove query-failure positional fallbacks; target writes explicitly where the native CLI supports it.
- [ ] Run `go test ./cmd/internal/layoutcmd ./cmd/internal/draftroute ./cmd/internal/clipcmd ./cmd/internal/termcmd ./cmd/internal/couchcmd ./cmd/internal/couchtty -run 'Test.*(Reply|CopyOnSelect|Clipboard|Fullscreen|Toggle|Focus|Route|NoteAnchor)' -count=1` plus targeted race tests and latency/call-budget controls. Commit, map changed behavior in atlas, and close M2 through SDLC.

### M3 — Editor admission, native refusal proof and complete caller map

**Create:** `nvim/zellij_observation.lua`; `tests/zellij-observation_test.lua`; `cmd/internal/layoutcmd/fullscreen_reply_live_test.go`.

**Modify:** `nvim/workbench_route.lua`, `pair_poke.lua`, `dom_transport.lua`, `review/restore_client.lua`; `Makefile.local` test-lua dependency for the new headless tests; `atlas/architecture.md`, new `atlas/zellij-reply-safety.md`, `atlas/index.md`, and applicable README/key help notes for visible refusals.

- [ ] Add Go/Lua shared-fixture contract tests and fake-backed headless tests for every modified editor caller. Wrong-kind or same-kind conflicting pane/tab/visibility observations must not produce native focus/write/submit/hide/show. Preserve DOM current-recipient revalidation and cached draft session/live-process checks.
- [ ] Implement the shared Lua admission helper through the existing action seam; validate before effect/staging and expose refusal through existing user-visible channels. No new generic CLI proxy or transport.
- [ ] Run `make test-lua` and the targeted Go packages; keep every original actual-chord input/geometry/focus assertion and positive success control.
- [ ] Native acceptance uses one disposable real0.45.1 session and the retained #640 raw ConnStatus/ListPanes noise producer, with an explicit finite sample budget and joined lifetime. Exercise the hardened production preflight/action path against known owned pane IDs; independently record actual target/input/state and refusals. Require observed bad native evidence to become refusals/unknown with no wrong target or guessed absence. If the fixed sample produces no fault, record inconclusive rather than claiming qualification or blindly repeating until green. Retain any partial-progress failure and classify it before continuing. Do not change native version, deadlines, test assertions, or use a proxy.
- [ ] Coordinate the native window with pair:6. Operator-real-setup smoke (focus toggle, valid paste, refused uncertain routing/restart) is requested through TL using the approved staging path only; never touch :0 autonomously. Record the native/real-setup limits explicitly.
- [ ] Finish the atlas caller table with exact disposition, validator/admission owner, fake test and recheck policy for every audited row. Verify no orphaned direct one-shot query consumer remains via `rg 'list-panes|list-clients|current-tab-info|list-sessions|dump-screen|are-floating-panes-visible' cmd/internal nvim` and review each production hit. Retained advisory/cosmetic callers remain named exceptions, not omissions.
- [ ] Push the branch with `sdlc pr`, run the required `sdlc local-check run --name suite`, and close M3 followed by the whole issue with measured actuals and verification. Enter `sdlc merge --yes` on SHIP; the landing turn owns integration. No manual main-chasing suites.

## Acceptance matrix used at each hardened caller

| Fake/native event | Required result |
|---|---|
| Wrong kind, empty success, truncated/malformed response | Unknown/refused before action; no extra read to obtain a preferred answer |
| Valid same kind, different addressable target/tab/role/owner/focus | Refuse incompatible context or differing projection; effects/records unchanged |
| Valid same kind on first read, current matching-kind facts on second but different decision | Refuse disagreement; no automatic replan/mutation within this request |
| Matching compatible observations, expected local identity | Existing correct behavior, explicit correct target and bounded call count |
| Cancellation/process/session/pane change between observations | Refuse, join child work, preserve preexisting artifacts |
| Failure after a real effect | Report incomplete/unknown stage, retain known progress, no guessed continuation/replay |
| Two indistinguishable correlated valid replies | Explicit residual protocol limit; do not claim detection or weaken the acceptance oracle to hide it |

The live native check supplements, rather than replaces, these deterministic fake cases. Neither a quiet native pass nor the upstream PR's existence closes the safety proof.

## Upstream follow-through

The issue log will link zellij-org/zellij#5270, #5382, #5546 and PR#5630. When a release includes retirement of client IDs, rerun the retained raw reproducer, native refusal/healthy controls and actual-chord conformance on that released binary before changing any compatibility assumptions. No upstream post or local native build is part of #647.

## Revisions

### 2026-10-17 — R1 approved

TL approved all three milestones and explicitly acknowledged the indistinguishable valid-stale-reply limit and the clipboard copy/auto-quote boundary. Additional acceptance emphasis: measure healthy Alt+Shift+Return and paste latency; visible slowdown returns for review instead of increasing deadlines. Proceed through change-code's plan-quality/estimate gates before production changes. The R1 proposal text above is retained as its reviewed baseline.


### 2026-10-17 — R2: executable budgets, refusal presentation and function strategies

Reason: plan-quality findings PQ-1 (function-level test strategy), PQ-2 (admission deadline enforcement), PQ-3 (refusal observability). The following is the operative refinement of R1; its original checklists and acceptance matrix above are retained as review history, not additional test-case authoring requirements. Milestone scope and boundaries remain M1/M2/M3.

**PQ-1 — named risky functions and strategy.** The implementation work is the three milestone file groups already named: build the contracts/fake and wire lifecycle owners (M1); wire Go actions and clipboard (M2); wire Lua, qualify native refusal and publish the caller map (M3). For each group, add the relevant failing regression first, implement, run the relevant package/headless tests and targeted race checks, update atlas, then close its boundary. Replace the historical enumerated case inventories with these function-level obligations:

| Risky function / owner | Adversarial class and mechanical test guard |
|---|---|
| New `zellijreply.ParseClients` | Fuzz arbitrary table/JSON bytes with captured native seeds; strict typed rows or explicit unknown, never inferred zero from foreign input. |
| New `zellijreply.ParsePanes` | Fuzz malformed, incomplete and ambiguous identity/schema input against native/explicit legacy seeds; required decision fields cannot be fabricated by decoding. |
| New `zellijreply.ParseTab` | Fuzz missing, mistyped and contradictory identity/layout facts; return typed evidence or unknown. |
| New `zellijreply.Confirm` | Controlled read/event sequences prove early termination, projection agreement, shared cancellation and zero post-refusal effects. |
| New caller-local `ProjectPaneTarget`, `ProjectClientAttachment`, `ProjectTabLayout` | Pure projection tests vary arbitrary irrelevant fields while preserving every decision-relevant target/context difference. Names may be package-local to the relevant selector, not duplicate shared implementations. |
| `SessionOwnerProbe.Probe` / `Revalidate` | Stateful owner and exact-generation changes; no destructive authority without compatible owner evidence and the existing independent OS fence. |
| Launcher `SnapshotSessionsContext`, `terminalOSAttempt.Check`, one-shot name admission | Stateful contradictory discovery/client/OS observations; effect log proves unknown cannot authorize spawn, reuse or teardown. |
| Couch `resumeContextWith`, `confirmStillDetached` | Stateful explicit-adoption evidence changes; preserve existing fences, ordinary attach semantics and query-budget controls. |
| `RunToggleFocused`, `FocusRightTerminal`, `SwitchRightTerminalTab`, split planning and `restoreTiling` | Stateful target/tab/layout drift and partial progress; effects/return records obey admitted evidence, refused actions reach the real presenter. |
| `draftroute.RouteLua`, terminal shortcut fallback | Stateful role/target drift; actual CLI effect log proves explicit admitted target or no input. |
| `RunCopyOnSelectOrchestrate`, `RunClipboardToPane` | Stateful source/destination uncertainty and failed focus; inspect clipboard mirror, staging and actual effects across the detached entry point. |
| `focusedClientPane` / restart role probe | Stateful client/context drift within the existing one-second operation; chosen routing/relaunch and notice reflect admission only. |
| `notePanes` / note-anchor integration | Stateful geometry/role changes under a captured frame; retain screen anchor and explanatory notice on refusal. |
| Lua `zellij_observation.panes`, `.tab`, `.clients`, `.confirm` | Shared Go/Lua adversarial corpus plus controlled event sequences; strict evidence, cancellation and effect exclusion agree across languages. |
| Lua `workbench_route`, `pair_poke`, `dom_transport`, `review.restore_client` entry functions | Stateful CLI fault schedules through the actual headless production modules; effect log and presenter prove refusal before input/visibility effects. |
| Fake model dispatcher and effect recorder | Model-transition tests prove normal replies derive from state and injected faults never fabricate an unrecorded effect. |

These are strategy obligations, not a prose enumeration of individual executable cases. Deterministic fakes remain the primary oracle; native qualification supplements them.

**PQ-2 — total budget and cancellation ownership.** The five-second cap in R1 belongs to Pair's `launcher/zellij.go` constant `zellijQueryTimeout`, not to the native CLI. Current layout and clipboard queries have unbounded/default-background contexts, and Lua's default `zellij_trace.action` uses synchronous `vim.fn.system`; relying on those defaults was incorrect.

Protected one-shot admission creates a single five-second context at the production operation entry, capped by any earlier caller deadline. Both observations, all query-derived decision points of that operation and OS/local identity fences use the remaining context; a second read never receives a new five seconds. The restart role probe retains its smaller one-second total. Existing DOM/action deadlines remain the smaller bound. This bounds previously unbounded paths and never extends an existing bounded operation. Mutation execution retains its existing contract; cancellation must be checked again immediately before each new effect, without pretending an already issued native mutation is rolled back.

Go runtime query methods used by protected paths accept and propagate this context to `exec.CommandContext`. Reuse the repository's synchronous owned-process cleanup seam where available; the caller owns cancel, Wait and bounded pipe draining, and returns only after its query child is collected. A query is synchronous, not a detached worker; tests independently observe process exit on cancellation and successful completion. New Lua admission records one monotonic deadline, passes remaining milliseconds to the existing `zellij_trace.action` timeout option (`vim.system(...):wait()`), and stops immediately on exhausted budget. Its test executor must honor and expose that timeout. Do not globally change retained cosmetic/periodic callers.

Auto-quote source and destination preflight share one operation context in the detached orchestrator. Extract/reuse the clipboard-to-pane admission/execution functions there rather than exec-replacing into a fresh unbounded admission; the standalone clipboard-to-pane command creates its own bounded operation. Admit the complete source/destination plan before flash or staging, preserving the separate successful OS copy. This is a local orchestration change, not new transport or a retry. Measure healthy toggle/paste latency before and after; visible slowdown or systematic healthy refusal goes back to TL, never to a larger budget.

**PQ-3 — presentation through the existing non-Zellij notification route.** Go shortcut and detached clipboard failures use the existing `notifytransport.Send` broker addressed by `PAIR_PAIR_WRAP_PID_PATH`, already used by `pair notify`; its existing 200ms delivery bound remains unchanged. Add an injected `NotifyRefusal` runtime method at these entry points; its OS implementation sends a fixed content-free message such as “Layout change refused: uncertain Zellij reply” or “Copied; auto-quote refused: uncertain Zellij reply”. Keep diagnostic stderr/log as supporting evidence, not the presentation channel. Notification is independent of native query replies and never focuses, flashes, stages or triggers a pane. Missing broker/binding is an explicit presentation-delivery failure in the retained diagnostic, never reported as a displayed notification. Test the actual CLI/detached entry with a private existing broker and capture the wrapper's presentation envelope, rather than asserting only a mocked notice call. Existing wrapper/Couch notification policy owns final terminal attention presentation; do not add a new broker, poller or UI.

Couch-owned restart and note refusals use their existing console notice result; Lua refusals use the existing editor notification surface. Tests exercise those production result/presentation paths. No refusal text includes raw clipboard, prompt, pane command or screen content.

**Source-grounded compatibility clarification.** Ordinary warm attach currently uses `listedTerminal` then `attachListedTerminal` and permits an already attached session; `confirmStillDetached` belongs to explicit `adoptWarm` continuation adoption only. Preserve `TestWarmResumeListingBudgetIsIndependentOfOtherSessions` (zero client queries, eight listings across its existing fences); do not introduce detached-client policy or nested pairs into ordinary attach. Tighten parsing at existing read fences and add confirmation only where the action lacks it. Native pane identity is `(is_plugin,id)` (plugin 0 and terminal 0 can coexist); multiple panes may be focused across tabs/clients. Native terminal_command can be null, and pane_command/pane_cwd can be absent after enrichment timeout. Require positive actual-command evidence for owner-sensitive decisions, not enrichment on every unrelated pane. Native tab info uses position, while pane rows expose tab_position. Preserve explicit legacy schemas without fabricating required decision fields.

R2 requires TL approval of the concrete admission budget, shared auto-quote orchestration and broker presentation before implementation. No production/test implementation has begun.


### 2026-10-17 — R2 approved and plan-quality clean

TL approved R2's function strategies, single five-second total admission capped by tighter callers, shared auto-quote preflight, existing broker refusal presentation and ordinary warm attach query budget (authenticated delivery 073e88c1-0bf0-4967-8e69-b37cc6758b67). The second plan-quality round disposed PQ-1/PQ-2/PQ-3 as addressed with CLEAN verdict. Deriving the estimate now; implementation remains behind change-code.


### 2026-10-17 — M1 implementation detail: shared query process ownership

R2's cancellation/collection clause is implemented by thin `cmd/internal/zellijexec.Run`, reused by native read callers while pure `zellijreply` remains free of IO. Existing process-group runners are Couch-specific and importing them from launcher would cycle. The new runner owns only synchronous read-only CLI children, uses no new transport/deadline/retry, preserves earlier pipe-drain bounds, and stops the private group on success as well as failure. This avoids repeating cancellation/cleanup in each M2 runtime (ARCH-DRY/PURE/FUNERAL). Native creation, attachment and mutator lifetimes remain with their existing owners.


### 2026-10-17 — M1 revision: reuse exact-group absence evidence

The first broader package run exposed Darwin returning EPERM for a successful query's empty process group. A fixed 600-command control reproduced it once with a valid positive PID and exit0. The existing Couch #542 helper already resolves this ambiguity through kern.proc.pgrp. Extract that unchanged OS observation to procutil.ProcessGroupMembers and delegate both existing Couch helpers and the query runner to it. Only EPERM triggers the independent lookup; an observed empty group completes cleanup, while live/zombie/unreadable groups retain the error. No retry, sleep, new deadline or normal-path process scan. The unchanged fixed control now passes600/600, and deterministic empty/live/unknown regression plus race controls pass. This narrow shared ownership extraction follows ARCH-DRY and R2's collection contract.

A separate counted guard caught an extra host table scan per ownership probe. Keep the existing before/after exact-generation fence around the complete native pair, plus effect-time revalidation; do not add a scan after each individual pane read. Native pane observations double as designed, but initial capture plus live process-table fences stay at the existing 1+2N budget.
