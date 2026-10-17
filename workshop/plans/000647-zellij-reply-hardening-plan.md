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

### 2026-10-17 — M2 source-context refinement approved

Read-only native source audit found that copy_command receives only selection stdin, without a selecting pane/client token. An inherited ZELLIJ_PANE_ID is not per-selection provenance. Native list-panes includes all tabs, where several panes may legitimately be focused; current-tab-info selects the last keyboard-active client and cannot identify the copying client. Do not infer source from a globally unique focused pane or the copy hook's inherited pane environment.

TL approved the concrete rule (authenticated delivery 39354779-66f0-4d49-9411-f7ad46f22145): auto-quote observes strict list-clients and list-panes together, requires one attached-client focus identity (or all attached clients agreeing on the same terminal pane), correlates that source's role/tab and the unique draft in that context, and confirms the combined decision projection. Client disagreement or missing correlation visibly refuses auto-quote before flash, stage, focus or trigger; the completed OS copy remains successful. Multiple tabs and clients sharing focus remain supported. Additional client queries belong only to auto-quote, under its existing single five-second admission budget; measure healthy latency rather than increasing that budget.

This confirms the observed context, not the historical origin of the selection. Focus can move before the detached orchestrator observes it; identical correlated stale replies remain the previously acknowledged protocol limit. Stronger attribution requires native per-copy source identity, which is outside this issue. Explicitly admitted source IDs are passed to flash, with no new source lookup. ARCH-SECURE/ORDER: carry the admitted facts into effects rather than replacing missing provenance with an unrelated native context.

Native source references in the existing clean checkout /Users/xianxu/workspace/zellij: zellij-server/src/tab/copy_command.rs:21–34, tab/clipboard.rs:18–26 and tab/mouse_handler.rs:1033–1065; route.rs:2417–2430 and screen.rs:5373–5408. No native build or upstream publication was performed.

### 2026-10-17 — M1 review corrections: complete the existing authority rule

M1 round1 returned REWORK (BR-1/BR-2/BR-3). These corrections enforce the already approved terminal-unknown rule without new query pairs, state, deadlines, retries or transport. ARCH-PURPOSE/ORDER: enumerate the whole lifecycle read class now: launcher verifyCouchSessionAttach, terminal handoff Check/Admit/requireListedLiveTerminal, SessionOwnerProbe/Revalidate and name admission, and Couch listedTerminal/attachListedTerminal/confirmStillDetached. Native-read errors terminate at every fence; the cold handoff's local readiness-file polling and genuine helper-start retry remain separate policies, never native-query recovery. Add the real launcher attach-path regression, not only a helper test.

BR-2's agreement rule applies equally to direct and cached short/full listing pairs: both validate and describe the same exact native session-name set before any live/attached/detached projection or client query. A missing/extra/replaced row is unknown, with no retry or partial authority. Derive full-row state only from its validated metadata suffix and preserve exact names; whitespace trimming or the word EXITED inside a name cannot create a different identity/state. Sweep strict one-shot full-row consumers (owner presence, quiescence, reuse and terminal handoff) onto the same typed row facts; retained cosmetic consumers keep their existing parser/cadence. Test native and cached contradictions, positive agreement with reordered rows, exact names/suffixes, and no attach/launch/native effects after refusal. The listing poller's cadence/cache owner is unchanged.

BR-3 upgrades the existing shared terminal registration fixture to actual short/full listing forms and typed owner panes, retaining its original lifecycle assertions; verify every consumer, including composed production Pair registration and listed detached attach. Correct the Core concepts location record: the Confirm helper is implemented in cmd/internal/zellijreply/reply.go, not the historical proposed confirm.go. No milestone is closed until SDLC disposes the findings.

### 2026-10-17 — R3 launcher budget re-plan proposed; decision pending

The M1 class sweep found that R2's total budget is not yet implemented across launcher attach. RunLaunch and LaunchOptions have no enclosing context. Three verifyCouchSessionAttach fences call context-free SessionLiveness; OSRuntime starts Background liveness, so six native listing calls can each receive a fresh five seconds. Terminal handoff Check also runs before CoordinateTerminalHandoff creates its bounded Admit observer. A timeout created separately inside verify would repeat the same error and is rejected as the simplest inadequate alternative.

Proposed warm-attach correction: start a single selected-operation query context once the intent is known, propagate it through context-aware private attach/liveness seams and native handoff Check, and retain a separate original lifetime context for Start/Wait. Do not start a timer before human picker/name prompts, impose an admission lifetime on the interactive client, or change periodic listing cadence/cache policy. Add controlled shared-deadline, earlier-caller, expiry-before-effect and uncapped-child-lifetime regressions through production paths.

Cold creation needs an explicit TL phase decision before this plumbing is implemented: either the same wall-clock five-second cap includes existing birth/readiness waiting (earlier cold refusal), or existing pre-create and post-birth admission phases receive distinct caps while the established birth bound stays unchanged. No pausable clock, deadline environment protocol, new transport, increased deadline or guessed implicit exemption is proposed. Authenticated pending request b33692e2-b561-4b99-b22a-b45526ae8611 states both choices. Budget-dependent implementation/re-review waits for that answer; the independent BR-1/BR-2/BR-3 corrections continue under the already approved uncertainty rule.


### 2026-10-17 — R3 approved: bound query decisions, preserve client and birth lifetimes

Authenticated TL delivery 7c0d5560-3f2d-40ec-b9d8-7984c09ead76 approves the explicit context plumbing. Warm attach creates one operation query context once intent is known, carries it through context-aware liveness and attach seams and handoff Check, and keeps interactive Start/Wait on their separate original lifetime. The selected-operation fences consume the same remaining budget, capped by an earlier caller deadline; no timer spans human selection.

Cold creation has distinct phases: pre-create admission owns one five-second query budget; birth/readiness retains its existing bound; post-birth admission owns a separate five-second query budget beginning after readiness. A slow healthy birth must not cause a new early refusal. The rule is query work bounded per decision, not launch completion within five seconds. No retry, sleep, longer deadline or new transport is introduced (ARCH-ORDER/FUNERAL).

Implement RED-first production-path regressions for shared warm deadlines and cancellation before effects, earlier caller cancellation, interactive-client lifetime independent of admission cancellation, and cold readiness followed by a fresh bounded post-birth query phase. Then re-run the lifecycle controls and M1 boundary review. M2/M3 remain behind their existing boundaries. Requeue #640's already closed merge when the registered checkout is free, as TL directed.


### 2026-10-17 — R3 implementation map and function strategies

The explicit launcher seams are SessionLivenessContext, ProbeSessionNameContext, AttachSessionContext and LaunchSessionContext. LaunchOptions.Context supplies the original caller lifetime; its private queryContext is created only for the selected decision. runOnce, attachWithRetention and launchWatchedContext carry those two lifetimes without encoding deadlines in environment variables. Legacy concrete OS entry points remain compatibility wrappers, while Runtime's blocking handoff contract requires the explicit contexts.

| Changed integration | Verification strategy |
|---|---|
| runOnce / attachWithRetention / context-aware liveness and attach | Actual launcher orchestration records the same bounded context at every native fence and handoff; an earlier caller deadline remains exact, cancellation refuses before effects, and query completion does not cancel the caller/client lifetime. Existing strict native CLI refusal controls remain in the path. |
| assignLaunchSessionNamesContext / assignCouchSessionNameContext / verifyCouchSessionCreate / ProbeSessionNameContext | Record the shared pre-create context across listing, name validation and owner observations through RunLaunch; cancellation during positive foreign-owner observation must still refuse. Retain the existing name-validator subprocess controls. |
| launchWatchedContext / runTerminalHandoffContext / coordinateTerminalHandoff | Preserve the existing birth watch and joined child/observer ownership; controlled channels prove Check cannot admit after cancellation, warm completion rechecks the query budget, and established clients survive admission cancellation. |
| admitTerminalAfterReady | Controlled readiness completion invalidates its own wait context; post-birth native admission must still obtain a fresh bounded context under the original caller, while cancellation of that original caller prevents the query. Existing readiness-file polling cadence and birth bound are retained. |

These thin context plumbing helpers add no artifact source or lifecycle state. ARCH-ORDER/FUNERAL: the owner that cancels a query is distinct from the owner of an interactive child. R3 does not alter retained periodic-observer cadence or give a failed query another attempt.


### 2026-10-17 — M1 round2 revision: diagnostics cannot erase collector failures

Round2 disposed BR-1/BR-2/BR-3 and opened BR-4 (Critical, unknown-observation-is-terminal). Both production consumers of zellijexec.Run—ZellijSource.runContext and OSRuntime.ProbeSessionNameContext—use errors.As to find an ExitError inside a joined failure, then normalize its diagnostic to successful absence/name-acceptance. That discards the runner's independent cancel/collection failure.

Rule: diagnostic normalization may consume only a pure subprocess exit failure. Cancellation, pipe/collection errors and joined failures remain terminal unknown even when stdout/stderr also contains an established absence or validator diagnostic. The simplest correction is a direct ExitError classification at both normalization boundaries; a joined/wrapped multi-cause error cannot qualify. A new result state machine or expanded runner API is unnecessary because the current runner returns the direct subprocess error only when cancellation/collection succeeded. Keep ordinary valid exit1 diagnostics and strict client tables compatible. Add injected collector-failure regressions for both consumers plus pure-exit positive controls, and audit every zellijexec.Run use before M1 round3 (ARCH-PURPOSE/ORDER/FUNERAL).

First checkpoint the review and requeue #640 from the registered checkout as TL directed. While that checkout is held by landing, BR-4 code/tests may be prepared in a disposable independent Git clone at /Users/xianxu/workspace/worktree/pair-slot1/pair647-query-fix; no SDLC gate runs there. Import its narrow diff into the owned #647 branch after #640's landing releases the registered checkout, verify and commit there, then submit the remaining M1 boundary round. No M2 edits occur before M1 closes.


### 2026-10-17 — BR-4 class sweep includes negative name bounds and cancellation causes

The injected collector controls reproduced six failures across empty inventory, accepted absent-name diagnostic and rejected-too-long validator diagnostic, each with collection failure alone or collection plus caller cancellation. The latter cases also lost the collector cause by replacing the error with ctx.Err. Extend the same approved rule to all three diagnostic branches: only direct subprocess exit errors may teach either name-length bound or successful empty inventory, and cancellation joins rather than replaces independent collector failures. Ordinary successful client tables follow their parser, never a validator substring. Per-instance private runner seams execute the real collector before deterministic EIO injection; no global test hook or new process protocol is needed.


### 2026-10-17 — Workflow revision approved: M1 review in independent clone

The independent clone passed the binary's read-only milestone preflight after fetching issue-tracker and configuring main's publication upstream. Its measured-actual provenance omits the primary checkout's suite-queue evidence, so it cannot silently replace that calibration input. Authenticated TL receipt 0019655b-05f9-460a-aba8-ce05a2d9026f explicitly authorizes M1 round3 there with --no-actual and reason: “M1 reviewed from an independent clone while the registered checkout lands #640; clone lacks the primary checkout's queue evidence”. This replaces the earlier implementation choice to wait/import before M1 review. Do not wait for #640 to dispatch this review. Later boundaries and landing return to the registered checkout once #640 releases it. No review or actual-hours guard is otherwise waived.


### 2026-10-17 — R4 approved in M2: exhaustive diagnostic authority matrix

M1 round3 finalized SHIP with BR-1/BR-2/BR-3/BR-4 disposed; evidence commit a237be890c228e07b4c7445301a573cdbd8beb8c. The initial push lease was stale because the ordinary clone's remote-tracking ref came from the primary local branch; fetching showed the actual remote was an ancestor, and a normal fast-forward push published the milestone. No force push was used.

A separate read-only overlay control, without changing the locked M1 candidate, reproduced pre-existing name-probe acceptance of contradictory stdout plus a validator substring: /tmp/pair647-validator-stdout-proof.log. Installed0.45.1 parse-only control (no server/session connection) retained /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/p647-validator-wwqysydw/manifest.json: exit2, empty stdout, exact requested-name-bearing Clap error and help footer. No native build or operator session was touched.

Authenticated TL approval c8f8d425-c96f-4305-8e85-b0ec8abe100a folds R4 into M2. This is the second exit-diagnostic change: implement one exhaustive matrix for both consumers rather than another isolated check. A blanket unknown for every validator would break the existing native name-budget search, so retain its positive/negative evidence under an exact pure decoder.

| Boundary / outcome | Admissible evidence |
|---|---|
| Inventory, clean exit0 | Existing strict nonempty expected listing body; empty/unrelated bodies remain unknown. |
| Inventory, pure exit1 | Empty stdout plus exact native empty-inventory diagnostic only; wrong status/kind, mixed stdout or extra diagnostic is unknown. |
| Name probe, clean exit0 | Strict valid client table, including header-only zero; malformed/empty body is unknown. Validator-looking text in a successful client command must not teach a negative bound. |
| Name probe, pure exit2 | Empty stdout plus exact native validator grammar containing the requested name and a canonical numeric limit; wrong target/header/footer/status, fragment/extra body or contradictory stdout is unknown. Valid native rejection still teaches the existing negative length bound. |
| Name probe, pure exit1 | Existing exact post-parse absence diagnostics with empty stdout retain limited name-acceptance meaning; another command's empty-inventory diagnostic does not. |
| Either, joined/collector/cancel failure | No diagnostic may teach any authority; preserve all error causes, including simultaneous cancellation and cleanup. |

Pure decoder tests cover the grammar/status/target/body matrix; production subprocess controls prove both consumers and the monotone name-budget caller preserve behavior. Retain real collector failure injection, native parse-only conformance and original positive domains. No retry, sleep, deadline increase, transport or new result state machine. ARCH-PURE/SECURE/PURPOSE/ORDER.

M2 code may be prepared in this independent clone while #640 holds the registered checkout. M2/M3 reviews, whole-issue close and landing return to the registered checkout when it is free. Import the issue branch and refs/sdlc/reviewed/000647/M1 there after #640 finishes; never change its checkout while its merge is active.

### 2026-10-17 — M2 measured interactive cost accepted; action-specific projection

Reason: initial healthy native A/B measured confirmation overhead on Alt+Shift+Return. Authenticated TL approval 683cc32b-b71b-484d-be31-ec48e5318967 keeps correctness, permits one return-to-Normal narrowing experiment and prohibits parallelizing query connections. Return compares identity/role/state/tab; geometry remains required for Normal-entry split-order capture and actual restoration moves. The native experiment passed all existing conformance assertions, median return 294 ms versus original 192 ms in the same declared window; this small host-sensitive sample does not isolate a causal optimization effect. Keep the cost, document it in atlas/README, and do not raise deadlines.

The existing first-unknown contract applies to selector ambiguity as well as schema errors. The scoped confirmation helper accepts the caller's pure selector validation so required destination ambiguity/missing targets terminate before a second query, sidecar writes or input effects. Preserve native pane order where the established target picker uses it; canonical sorting must not change that selected target. No new retry, state, transport or timer.

### 2026-10-17 — R5 approved: editor decision admission phases

Authenticated TL approval e1ea8c04-2d35-4b70-87c2-5aeb66b4ff25: DOM retains its pin as candidate evidence and starts one five-second native admission budget at the first pre-write validation. Preserve the existing 36-second transaction/preparation/receipt lifetime and recipient/pane checks before write and submit; do not add nested pairs. Review restore preserves existing birth/readiness/activation bounds, then starts one five-second visibility-decision budget after activation. Recheck exact activation and context before show/hide; completed activation survives a later visibility refusal. Shared Lua validation rejects duplicate JSON keys and preserves the entire shared corpus. M3 code remains behind the M2 boundary.

### 2026-10-17 — Native clipboard fixture re-plan proposed; awaiting TL

The first fixture failed KDL parsing then hung joining blocking FIFO reads; the changed fixture fixed KDL and made cleanup finish, but nonblocking FIFO reads returned EAGAIN before native admission. Installed Go os/file_unix.go explicitly disables Darwin FIFO runtime polling. Existing nonnative controls proved teardown but did not assert real byte delivery, so were insufficient. Stop further FIFO-wait deltas.

Proposed smallest alternative: one private Unix-stream listener and exactly two tiny native pane receiver connections. A first byte identifies source/draft readiness; subsequent bytes prove actual pane input. Hold listener and accepted connections for fixture lifetime; cleanup closes listener, joins accept owner, closes tracked connections and joins reader owners. Bounded two connections; direct Zellij CLI and production code unchanged. Nonnative controls must prove real byte delivery, no-client cleanup, idle-reader cancellation and unread-buffer teardown under race. Then one changed-fixture native timing attempt under unchanged admission/fixture bounds. No polling retry, sleeps or longer deadlines. This is private test instrumentation, not a Zellij CLI transport/proxy. Await TL approval before code.

### 2026-10-17 — Native fixture re-plan approved

Authenticated TL approval f82d7d85-64bf-4632-9d5e-27f99baf9d37 accepts the proposed replacement exactly: one private Unix-stream listener, two receiver connections, role/readiness first byte followed by actual pane-input bytes. Cleanup closes listener, joins accept owner, closes tracked connections and joins readers. Preserve explicit read failures and prove actual data plus no-client/idle-reader/unread-buffer teardown under normal/race before one changed-fixture native attempt. The direct native CLI, production code and time bounds remain unchanged.

### 2026-10-17 — M2 clone review exception approved

Authenticated TL approval 829d7622-04fb-474b-876c-301c5ae26051 authorizes M2 milestone-close from the independent clone with --no-actual and the exact proposed reason: M2 reviewed from an independent clone while the registered checkout lands #640; clone lacks the primary checkout queue evidence. Do not wait for #640; M3 starts after M2 SHIP. Preserve the queued registered merge and import the reviewed branch/pins when it releases.

### 2026-10-17 — R6 proposed: complete M2 REWORK plan; approval required before code

**Reason and boundary.** M2 review returned REWORK on reviewed head bf0524122b46540e2db6d2e0f80d6503e4fbf1d1, with BR-5/6/7/8. Its sidecar and ledger are retained in commit 75b3f75ccf33b94f0a5397ec4eb16ab82e79399a. BR-6 is the third finding in the unknown-observation-is-terminal family: stop incremental patches and replace the scattered effect checks with one enforced rule. Authenticated hub instruction e803df50-f010-4772-8934-1fca435a33b8 requires this complete plan before implementation. M2 remains open; M3 has not started.

#640 landed as PR410, merge bcf5163bdfac06a3fc51021ddb635ce52fa755a8. The clean registered pair:1 checkout now holds the exact clone branch at 75b3f75ccf33b94f0a5397ec4eb16ab82e79399a and M1 reviewed pin a237be890c228e07b4c7445301a573cdbd8beb8c. Subsequent source edits, boundaries and landing return here. The previous TTY capture is unavailable; committed source, gate artifacts and authenticated receipts supply continuity. No review-fix code or new native run preceded this plan.

#### Core concepts and integration changes

| Name | Lives in | Status | Contract / integration |
|---|---|---|---|
| Pre-effect admission gate | cmd/internal/zellijreply/effect.go | new | Thin synchronous integration: Effect(ctx, invoke) checks the existing token immediately before invoking exactly one new effect; no new context or timer. |
| Guarded M2 dispatch | layoutcmd/layoutcmd.go, rightpane.go; termcmd/run.go; draftroute/route.go; clipcmd/run.go; couchtty/console.go | modified | Every classified native action, authoritative sidecar write or query-derived UI publication uses that gate. |
| Temporary flash cleanup obligation | clipcmd/run.go, runtime.go | modified | Once color application is attempted, one reset scheduling attempt is owed independently of admission cancellation. Apply/reset errors become observable. |
| Native note enrichment adapter | couchcmd/note_panes.go and run.go wiring | deleted | Existing frames cannot correlate native client/tab observations; retain original screen anchors instead of inventing semantic authority. |
| Asynchronous note pane resolver | couchtty/console_noteanchor.go | deleted in part | Remove PaneRects/SetPaneRects and query/job owners once their only production provider is removed; retain synchronous frame capture and screen/status/tab-bar anchoring. |
| Couch restart query owner | couchcmd/shortcut_focus.go | modified | Caller-owned buffer plus existing zellijexec.Run replaces cmd.Output; existing command arguments and one-second enclosing budget remain. |
| Cancellation coverage manifest | cmd/internal/zellijreply/effect_coverage_test.go | new | Test-only enumeration of actual protected sinks and explicit exemptions, with structural rejection of unguarded/unclassified effects. |

No new pure observation schema, durable artifact family, native protocol, credential, retry or deadline is introduced. Existing pure AnchorFor/ScreenLayout algorithms remain distinct from the removed unproven native adapter; existing parser, selector and planner tests remain. Tests do not treat preserved pure geometry examples as proof of native frame provenance. ARCH-PURE/DRY.

#### One cancellation rule, not two more checks (BR-6)

Use the already owned operation context as the single cancellation token. Implement the synchronous primitive:

```go
func Effect(ctx context.Context, invoke func() error) error {
    if err := ctx.Err(); err != nil { return err }
    return invoke()
}
```

Each invocation encloses one actual effect, not a whole sequence or loop. Local evidence is not a cancellation exemption: after sidecar/path/registry reads and pure selection, the actual dispatch still passes the gate. Nested operations retain the same context. A cancellation observed before dispatch forbids that new effect; cancellation inside an already dispatched effect permits that effect to finish under its existing lifetime, then forbids the next one. This is not an atomic transaction with cancellation racing between the check and external dispatch, and it does not undo possibly completed effects.

| Owner | Complete authority-to-effect enumeration |
|---|---|
| FocusRightTerminalContext / SwitchRightTerminalTabContext | Explicit pane focus; addressed terminal-tab chord write, including positive sidecar-only resolution. |
| RunToggleFocusedContext / restore helpers | Return-record write and clear; float/embed; coordinates; show visibility; fullscreen; focus; wrapper nudge; every swap-layout advance; final pane move. Clearing a return record publishes authority and is not cleanup. |
| Terminal shortcut executor / split | Last-left and last-terminal record writes; selected pane focus; actual terminal split; nested layout/draft operations retain the same token and guard their own dispatches. |
| RouteLuaToPaneContext | Optional draft focus and each of the four addressed input actions independently. |
| Clipboard orchestrator / paste / flash | Actual color application, actual quote-file write after quote-path resolution, destination focus and Ctrl-_ trigger. |
| Couch restart routing | Observed-role chord forwarding or opening relaunch confirmation, after final local endpoint/focus checks. Retain the original one-second token until dispatch completes instead of cancelling it immediately after the probe. Later human confirmation retains its separate existing lifecycle operation. |
| Note semantic publication | Retired by the fail-closed BR-5 change below; no SetNoteAnchor enrichment survives outside a guarded owner. Ordinary synchronous screen-note creation remains independent of native queries. |

Explicit exemptions: collection of an owned query group; release/unlock; already-owed flash reset; refusal presentation/diagnostics. Initial OS clipboard mirroring and its detached orchestrator spawn, ordinary terminal input/scroll/local tabs, and startup registration do not derive authority from these protected observations; preserve their existing ownership. No admission expiry kills an interactive child or native mutator already issued.

**Simplest alternatives rejected.** Two ctx.Err checks at the reported layout sites leave the third-round class unresolved. Context-bound runtime adapters would duplicate wrappers for actions, stores, files and Couch callbacks, and risk suppressing cleanup or changing in-flight mutation lifetimes. A lock cannot make an independently cancelling context and external Zellij dispatch atomic; holding one adds serialization without the missing guarantee. The shared gate plus an enforced sink enumeration is the chosen mechanism. ARCH-PURPOSE/ORDER/CONSTRAINTS.

**One exhaustive coverage test.** A single manifest names the protected production functions and every sink above (including layout step variants and draft input positions), plus justified exemptions. A scoped Go-AST check requires actual sinks to be directly enclosed in Effect with the enclosing admission token, rejects unclassified/raw sinks and stale manifest entries, and rejects effect loops enclosed by one guard. The manifest is not the behavioral oracle: package stateful fixtures separately cancel in the last evidence getter while returning valid authority, and inside effect N before return, asserting no next effect. Include native-observed and zero-query local paths, live-token healthy controls, partial-progress records and cleanup exemptions. Removing any guard must fail the structural test; disabling the shared cancellation check must fail behavioral controls.

#### Note anchors: admit the provenance gap and fail closed (BR-5)

Source audit: terminal.Frame contains endpoint, frame generation, geometry epoch, cells and cursor, but no native client/tab identity. Couch's hosted pane has thread/process/PTY ownership, also no native client/tab identity. list-panes includes all tabs, and notePanes discards that identity. A new list-clients/current-tab-info observation, unanimous clients or matching cells cannot establish which tab produced a previously captured frame; a tab switch may precede repaint, and different tabs can display identical text.

**Chosen proposal:** remove automatic native semantic enrichment for hosted notes. Preserve the original note box, exact opening text and Couch slot label, and all Couch-owned status/tab-bar anchors. Native pane-role, filename, line and section attribution becomes unavailable. Explain that fallback visibly once per newly opened note batch using the existing session/note bookkeeping lifetime, without a worker or native query. This is a deliberate user-visible reduction requiring TL approval, not a claimed positive correlation fix.

Remove the adapter's only production root at couchcmd/run.go SetPaneRects, its native query/file-read/parser helpers, and the now-providerless asynchronous PaneRects jobs/pending/running resolver. Retain synchronous public-frame capture and AnchorFor/ScreenLayout logic. Delete the obsolete adapter tests/live geometry-only acceptance; replace them with production composition/broadcast refusal tests. Do not retain unreachable native query code or introduce an unused provenance API. Future enrichment requires independently supplied captured-frame/client/tab provenance and a separately reviewed design, not more snapshots in this revision. ARCH-SECURE/PURPOSE/FUNERAL.

Tests first reproduce the hidden-review-tab attribution against the current production composition, then require the opening screen anchor to survive overlapping panes, tab changes, multiple clients, identical text and repeated valid native rectangles. Assert zero note-related native queries, file reads or semantic SetNoteAnchor updates; preserve note text/box/slot and owned status/tab-bar positive controls. Keep tests proving pure AnchorFor behavior separate from native acceptance. Update README and atlas/broadcast.md plus the caller map with this explicit limitation.

#### Owned Couch queries (BR-7)

The native note query is removed with its consumer, structurally eliminating one bypass. Route the remaining queryShortcutClients through the existing zellijexec.Run with a caller-owned stdout buffer. Preserve argv, output/error semantics and the existing collector's one-second pipe-drain cap; do not add a query supervisor or change native mutation ownership.

Use the existing collector test's independently observed process-group/descendant pattern through the actual production queryShortcutClients entrypoint. A private PATH shim and local Go helper own readiness/release IPC and redirected descendants. Cover parent success, parent exit7, cancellation after readiness and inherited-output drain cutoff; require descendants collected before return and original errors retained. Join fixture goroutines and clean only identity-fenced owned processes. No production injection seam is needed. A scoped source check rejects reintroduced raw Output/Run calls at protected query boundaries. ARCH-DRY/MOCK/ORDER/FUNERAL.

#### Owed cleanup survives admission cancellation (BR-8)

All three color paths—auto-quote, queried no-argument flash and explicit-ID zero-query flash—share flashAdmitted. Guard the actual color attempt. Once attempted, schedule the existing detached reset exactly once before returning apply/cancellation errors, even when color application reports failure: an error does not prove no color was applied. Never gate owed reset on the admission token. Then return any cancellation and stop subsequent staging/focus/input; preserve already completed effects.

Make SetPaneColor and ResetPaneColorAfter return errors, and let private startDetached return its start error while SpawnDetached retains its existing best-effort hook contract. Join apply/reset/cancellation failures for the existing visible refusal path. The guarantee is a reset scheduling attempt; a failed spawn cannot guarantee actual cleanup and must be reported as incomplete. Existing configured delay, explicit target, detached child lifetime and reap ownership remain. No retry or synchronous reset fallback is added.

Stateful deterministic controls enumerate cancellation before apply (no color/reset), during apply (one reset; no paste), after apply/during reset scheduling (reset survives; no paste), apply failure (reset still attempted; error preserved), reset-start failure (visible incomplete cleanup; no paste), and healthy auto-quote/queried/explicit paths with unchanged targets, delays and query counts. Production OS shim tests cover apply exit and reset spawn failure without touching the real clipboard or native sessions. ARCH-ORDER/FUNERAL.

#### Execution and qualification, after approval only

- [ ] Add and run RED regressions for BR-5/6/7/8 against this checkpoint; retain logs and name each failure. Add the shared guard unit/coverage tests before migrating consumers.
- [ ] Implement the shared effect gate and sweep every enumerated boundary, with package behavioral cancellation controls. Remove native note enrichment and its orphaned async owner; verify screen/status/tab-bar behavior and visible fallback.
- [ ] Adopt the existing collector at the remaining Couch query boundary; verify process-lifetime matrix normal and race. Implement owed flash cleanup/error reporting and its complete matrix normal and race.
- [ ] Update workshop/lessons.md, README and per-surface atlas records; append dispositions with tests to the issue without editing the gate's finding records by hand.
- [ ] Run `go test ./cmd/internal/zellijreply ./cmd/internal/layoutcmd ./cmd/internal/termcmd ./cmd/internal/draftroute ./cmd/internal/clipcmd ./cmd/internal/couchcmd ./cmd/internal/couchtty -count=1` plus focused `-race` runs for the new ordering/process/cleanup fixtures. Generate existing runtime assets first when required.
- [ ] Classify the review's three broader Couch listing-fixture failures by exact test names and controlled source/base runs. The pinned review base includes our M1 code: inherited from that base is not proof of a pre-existing main defect. Fix branch regressions here; report independent failures with evidence under the fleet flake rule, never use a blind green retry.
- [ ] Use the TL-owned native lane only for bounded disposable qualification on installed0.45.1. Current lane grant is authenticated receipt 0a7eb7b6-1705-49f9-ac54-f2d997a44daf; no native run has started. Re-run affected healthy layout/clipboard controls and record observed latency without increasing budgets or concurrent native reads; note anchoring now has a zero-query refusal oracle. Report lane release when done. No operator sessions, OS clipboard, local Zellij build or :0 changes.
- [ ] Commit/push the qualified source on the registered branch and run M2 milestone-close with its fresh-context review. M3 starts only after SHIP. Full recorded suite/whole-issue close/native misrouting acceptance remain the existing later requirements, not waived by targeted M2 checks.

Operating envelope: existing five-second per-decision budgets (Couch restart one second) and original interactive/birth lifetimes remain; no new timers or background owners. Cleanup uses the already existing detached reset lifetime. Removed note query workers reduce work/residue rather than introduce caches. Native request-nonce limitations and identical correlated stale-reply limitations remain explicitly unchanged. Await hub approval of this entire revision before implementation.
