---
collaboration-mode: discussion
---

# Proven previous-incarnation orphan reaping implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3. Use executing-plans for the safety-sensitive integration; bounded independent tests may use subagents. SDLC owns the plan-quality and close reviews.

**Goal:** Let the owning hub or local operator end the exact proven old-incarnation processes beside a live slot, with no signal to a live or unproven process.

**Architecture:** Reuse the monitor's provenance classifier and existing confirmation command. The CLI coordinates the existing bounded TERM/3s/KILL/2s cadence over an immutable exact set; a short supervisor request re-proves each target and sends each signal. Neither this path nor its executor expands descendants. Existing unattributed-nvim and teardown-registry behavior remains supported.

**Tech stack:** Go, existing Unix message transport, native process table, launcher reaper, stateful process fakes, shared disposable-Couch harness.

## Scope and approved contract

The incident is the six proven processes left by echo:2's operator restart, recorded in #792. `--reap` correctly refuses the current live slot; the resource monitor already remembers which exact processes belonged to its earlier incarnation. Do not weaken `--reap`, infer ownership from a process name, or create an orphan registry for evidence already retained by the sampler.

Hub approval 941a9381-a841-44ca-98b0-7e10a90936b3 pins authority: a request carrying caller credentials must authenticate the current connected hub, and that hub must own the target's canonical slot. Workers and other hubs are refused. No special primary-hub or machine-steward role is added. An entirely unbound CLI uses the existing UID-private, owner-only namespace socket as the operator boundary; partial or stale credentials never fall back to it.

`couch --reap-orphans [--confirm] [--json]` retains its argv. Preview names the proven group and signals nothing. Confirm obtains a fresh supervisor classification. Rows outside the caller's ownership are explicit refusals, not silently omitted. An authenticated hub may finish its eligible rows while reporting refused foreign rows; any refusal/failure makes the command nonzero.

The new group uses `ResourceView.Orphans` with `Orphan && OrphanKnown`, nonempty exact identity and verified slot/workspace provenance. Unknown/incomplete ownership is never a kill permit. Reuse `SelectResourceProcesses` for use-time checks, including remembered ownership transfer to a current reachable root. Thread inequality alone is insufficient: an old wrapper can remain after a relaunch of the same conversation.

## Alternatives

- **Selected: exact set plus supervisor-side effect requests.** Keeps the UI thread free and the existing 2-second request budget; authority stays beside the signal. It adds a small private protocol operation and an exact-set execution entry point.
- Returning a fresh list and calling `TreeReaper` in the CLI is unsafe: it expands unproved descendants and cannot veto a later hub/ownership change at the effect owner.
- Running the whole waiting reap inside one message handler would exceed the transport's 2-second contract. A persistent operation/receipt queue would solve a larger problem and add recovery state unnecessarily.

## Core concepts

| Entity | Kind | Location | Change |
| --- | --- | --- | --- |
| ProvenOrphanTarget; target selection/use-time decision | PURE | cmd/internal/couchcore/resource_reap.go | new |
| ResourceProvenance and SelectResourceProcesses | PURE | cmd/internal/couchcore/resource.go | reuse unchanged |
| ExactReapOutcome / exact batch transitions | PURE | cmd/internal/launcher/exact_reap.go | new |
| Reaper.ReapExact execution | INTEGRATION | cmd/internal/launcher/exact_reap.go | new; reuse configured cadence |
| Proven-orphan signal handler | INTEGRATION | cmd/internal/couchcmd/reap_proven.go | new; native table, inventory, authority, signal |
| Orphan CLI coordinator and renderers | INTEGRATION | cmd/internal/couchcmd/reap_orphans.go | modified |
| Strict request/response shapes | boundary | cmd/internal/couchmessage/protocol.go | additive |

An exact target contains the process key plus its original root, slot and thread witnesses. Many targets belong to one former incarnation; process identity is the deduplication key. All fields received from a CLI are claims until matched to supervisor-owned provenance. The target never authorizes a process group or a descendant search.

The executor receives an existing ProcessTable, an injected signal-effect function and controllable waits. Production effects go through the supervisor handler; its fake has process generations, parents, ownership changes, signal effects/errors and lifecycle barriers. The outcome reducer is tested without IO. No durable state or persisted schema is created; the request, exact plan and partial results die with the invocation. Current orphan evidence expires through the existing resource sampler/history/registry rules.

## Effect path and safety invariants

1. CLI obtains a fresh confirmed classification, carrying `callerIdentity` when any caller fields exist. The supervisor validates either all required credentials or the entirely unbound form. It checks bound caller role before returning an actionable proven group. Preview remains read-only and may show refusals.
2. Freeze the admitted exact keys and provenance witnesses for this invocation. Deduplicate overlaps with `LeftRunning` so a proven target cannot also take the legacy expanding-tree path. Keep unrelated legacy families under their existing contract; do not use them as fallback for a refused proven row.
3. Add one strict private `reap-orphan-signal` request: confirmed, complete optional caller credentials, exact target, and only TERM or KILL. It cannot choose arbitrary signals, groups, or an unauthenticated slot string. Response identifies the same target and distinguishes sent, skipped/gone-or-changed, refused and failed/uncertain. These are operation results, not persisted lifecycle states.
4. At each effect, get a current process table and resource inventory, then run the existing provenance classifier against supervisor-owned prior provenance. Match the frozen target exactly, require current positive orphan proof and veto current/live ownership, unknown identity, unknown census, conflicts and PID reuse. No `Usage`, argv, cwd, environment or shell resolver calls are needed for this check.
5. Re-authenticate the bound hub's current incarnation and ownership. Hold the existing hub ownership read lock across its final ownership decision and the short native signal so a hub move cannot commit between them. Do not hold it across process discovery, waits or transport calls. Reject unverified implicit hub configurations rather than guessing a hub role.
6. Check context and re-read the target's nonempty start identity immediately before the native single-PID signal. Signal errors are returned, not swallowed. There is no group kill and no descendant expansion. Native read/kill retains the existing OS check-to-signal limitation; this plan adds no claim of an atomic Darwin PID handle.
7. Send TERM once per permitted exact target, wait a shared 3-second phase, then KILL once per still-permitted survivor and wait 2 seconds. A fresh effect request repeats all vetoes before KILL. Use current complete snapshots for absence; unreadable evidence is uncertainty, not successful termination. New children not present in the frozen set are never added.
8. Every admitted target gets one final existing CLI status: reaped, skipped, refused or failed, with signal/error/uncertainty detail. Existing per-command 2-minute budget remains. Stop admitting signals when canceled; preserve partial outcomes for rows already processed. Lost signal replies are uncertain, never an automatic resend or an authorization to escalate that row. The operator inspects resources before explicitly confirming another attempt.

No lifetime lease can freeze these old and current processes: the slot must stay live, so acquiring its exclusive lifetime lease would refuse the intended operation. Thread gates also do not freeze OS ancestry or cross-thread hub moves. Fresh effect proof and the exact nonempty identity fence are therefore mandatory; the hub lock only serializes hub ownership changes.

## State and operator actions

Actions: preview, confirm, inspect resources, explicitly confirm again after inspection.

| Reachable state/event | Visible result | Recovery | Possible loss |
| --- | --- | --- | --- |
| No confirmation | Refused preview, zero signals | Confirm after inspection | None |
| Current proof and authority, target exits | Reaped | Inspect resources | Confirmed old process work ends |
| Target already gone or PID replaced before signal | Skipped, replacement untouched | Inspect resources | None to replacement |
| Ownership becomes current/unknown; caller replaced; hub moved | Refused with reason, remaining unsafe effects stopped | Inspect resources | Earlier admitted signals remain effective |
| Signal/probe fails, survivor remains, deadline or cancellation | Failed with per-row progress | Inspect before explicit retry | Some old processes may have ended |
| Reply lost after a signal | Failed/uncertain; no resend/escalation for that row | Inspect before explicit retry | Signal may already have happened |
| Supervisor/CLI exits mid-command | No success assertion; old evidence remains in monitor | Inspect with new invocation | Partial termination is possible |

Cancellation cannot undo earlier signals. The command creates no detached worker: waits/effects join the invocation's scope. One request can still have an uncertain outcome on transport loss; use the normal transport bound and never assert rollback.

## Operating envelope and cost

This is an explicitly confirmed manual batch, not a keystroke or admission hot path. Preserve ResourceMaxProcesses=8192, ResourceMaxTableRows=65536, ResourceMaxSlots=256 and the 2-minute CLI / 2-second exchange bounds. Exceeding evidence limits refuses affected rows and prints coverage; it never widens authority or hides partial results. No deadline tuning, installations or per-process subprocesses.

Let K be frozen candidates, P host table rows and S slots. Each of at most 2K signal requests performs one native table snapshot and bounded current inventory/provenance work: O(K*(P+S)) plus classifier ordering, explicitly not a claimed linear fleet-wide scan. This repeated observation buys the required per-effect veto without a persistent permit protocol. Batch wait snapshots are shared rather than per-row waits. At K=8 and16, tests count at most 2K signal requests/table reads, zero usage/argv/cwd/environment reads and zero descendant additions; independently double P and S and count the classifier work to reject worse growth than the declared factors. Native conformance records the six-process incident-sized command's calls and duration; materially exceeding the bounds returns to planning.

## Implementation and verification tasks

### 1. Pure target proof and protocol

Files: new resource_reap.go / resource_reap_test.go, protocol.go / protocol_test.go; existing resource.go/resource_test.go as fixtures, without changing orphan classification rules.

- [x] Write `TestProvenOrphanTargetsUseExactPriorOwnership` and `TestProvenOrphanSignalDecision` first. Use production SelectResourceProcesses over old+live generations, same-thread relaunch, reparented survivor, current-root ownership transfer, empty/duplicate/reused identity, incomplete census and overlapping registry entries. Assert the exact allowed set and zero current keys.
- [x] Add strict protocol tests for complete/unbound/partial caller forms, malformed target, nonconfirmed request and signals other than TERM/KILL. Confirm old Op+Confirmed callers retain the existing families and do not gain a new client-side signalling capability.
- [x] Implement the narrow types/pure decisions and request schema; verify red then green with `go test ./cmd/internal/couchcore -run 'TestProvenOrphan' -count=1` and the named couchmessage selection. Register new production sources in the exhaustive artifact inventory.

### 2. Exact executor and supervisor effect owner

Files: new launcher/exact_reap.go / exact_reap_test.go; new couchcmd/reap_proven.go / reap_proven_test.go; message_service.go and the existing hub/authority seams.

- [x] Write `TestExactReapNeverExpandsDescendants`, `TestExactReapEscalationAndUncertainty` and stateful service tests before implementation. Inject ownership/caller/hub changes before TERM and between TERM/KILL; reused/empty identity; permission error; canceled context; lost reply; surviving SIGKILL. Include a live child beneath an old root and an old child reparented to PID1. Assert no signal outside frozen eligible keys and one outcome per target.
- [x] Implement ReapExact without changing existing Reap(tree) semantics; obtain its timings from the same Reaper configuration. Implement supervisor reproof/signal requests with no wait in the handler. Exercise the real handler through its stateful table/inventory seam, not a stub asserting request spelling.
- [x] Count K/P/S growth at8/16 and bound all waits. Run `go test ./cmd/internal/launcher -run 'TestExactReap|TestReap' -count=1`, the named service selection and corresponding `-race` tests.

### 3. CLI, output and public-command conformance

Files: reap_orphans.go / reap_orphans_test.go, messages.go; new reap_proven_live_test.go beside the existing reap_orphans_live_test.go; reuse cmd/internal/testcouch, no private fleet harness.

- [x] Write preview/confirmed text+JSON tests carrying both the new proven group and legacy groups, duplicate keys, mixed own/foreign hubs, unsupported older supervisor and partial/uncertain results. Test authentic owning hub, worker, different hub, stale binding and unbound operator through real service composition. Preserve output sanitization, coverage and existing return codes.
- [x] Wire the CLI to ReapExact and supervisor effects; keep legacy families working and prevent overlap from reaching the expanding-tree path. Preview names each proven target and never signals. Refusals never invoke fallback signal code.
- [x] Native public command on a disposable Couch: keep a current slot/wrapper/agent alive, retain known previous-incarnation fixture processes, invoke preview then confirmed reap, assert only owned old identities end and the current slot still accepts a message. Cover Claude and Codex fixture kinds. Use acknowledged lifetime barriers, exact cleanup and PID-incarnation checks rather than sleep-based readiness. Never use the live fleet as a fixture. Ask the hub for the required stage/live acceptance window separately.

### 4. Documentation and gates

- [x] Update README, embedded Couch skill, atlas/couch.md resource/reap sections and index: authority, exact-set scope, no descendant discovery, partial outcomes and explicit recovery after uncertainty. Update the issue's final acceptance and Log without rewriting prior evidence.
- [ ] Run the complete artifactpath guard and affected-reader inventory audit before the recorded check. Classify any new reader using the per-reader regenerate command; no new quarantine or exemption. Run relevant public-command and identity/authority regressions, then the declared pushed-head `sdlc local-check run --name affected` (full fallback when selected).
- [ ] Complete required live acceptance agreed with the hub, SDLC close review, fix findings, land through sdlc merge, and report the PR and exact evidence. No stage on :0 without hub permission.

## Architectural decisions

ARCH-DRY/PURE: reuse SelectResourceProcesses, existing caller/hub facts and Reaper cadence; target membership and outcome decisions are pure. ARCH-PURPOSE: include every proven process kind, not just an editor/agent subset, without touching the current incarnation. ARCH-MOCK: stateful tables and public disposable-Couch conformance model interactions and races. ARCH-CONSTRAINTS: budgets, caps and counted K/P/S costs are explicit. ARCH-SECURE: supervisor treats caller-supplied target bytes as claims; authority and identity are checked at each effect. ARCH-STATE: exact frozen set plus per-row progress preserves refusal/uncertainty and forbids resend/escalation after a lost reply. ARCH-FUNERAL: invocation-scoped plans only, no new persistent residue. ARCH-EVOLVE: no persisted schema change; wire fields are additive and operation shapes remain strict, with explicit unsupported handling for older supervisors.

## Revisions

- 2026-10-22 — Hub approval cea188f7-9b5c-4999-879d-59404fd1971a accepts the full design and selects disposable public-command conformance with Claude and Codex as live acceptance. No operator/:0 stage is needed; this supersedes the proposed stage-window request in tasks 3/4. The first real post-landing use belongs to ops-echo against echo:2's still-proven leftovers, not this test invocation.

### 2026-10-22 — Plan-quality refinement (authoritative delta)

Reason: PQ-1 requires the complete detached-root veto, PQ-2 requires a structurally enforced transition function, and PQ-3 requires function-level test strategies. The following replaces the active proof/cost details and case inventories above; the incident, authority contract, exact signal scope, cadence and acceptance choice stay unchanged. Earlier text remains as the design history.

#### Complete ownership proof: PQ-1

Extract the sampler's common native ownership stage into `resourceSampler.collectOwnership(ctx)`: read the bounded table and current slot inventory, enrich it through existing `DetachedSlotRoots` with the generation-cached `inheritedThread` reader, then call `SelectResourceProcesses` with the sampler's prior provenance. Both `pass` and an effect-proof request use this stage. This is an integration shell, not a second classifier. It excludes usage/argv/cwd probes and persistence but **includes required detached-root environment evidence**; the earlier zero-environment-read claim is superseded.

The existing sampler worker remains the sole owner of provenance and the thread cache. Extend its existing request channel to distinguish full sweep from `prove-reap`; the signal handler requests a typed `ProvenOrphanProof` from that worker and never reads its mutable state or substitutes a cached ResourceView for a fresh proof. The reply includes the table/selection and inventory-completeness facts for the requested exact target. If the worker is busy or cannot finish inside the existing exchange deadline, refuse without a signal. It creates no detached task or durable operation state.

Preserve three outcomes of a detached-thread read: known matching thread, known absence of thread fields, and unknown/unreadable evidence. Cache them by exact generation under the existing lifecycle. For effect proof, unknown candidate or relevant-ancestor enrichment is a veto rather than a negative match; excluded infrastructure remains excluded by the existing predicate. In particular, a platform shell's withheld environment cannot justify killing it or an otherwise ambiguous subtree. Missing current inventory or incomplete table is also a veto. Fresh positive current ownership always wins over old provenance.

`ProvenOrphanTargets` and `DecideProvenOrphanSignal` consume this complete proof and reuse the existing classifier result. They do not call the raw classifier over unenriched roots. Keep the worker read stage outside the hub lock; reacquire current hub authority under its read lock, apply the final target decision and nonempty identity check, then send the single native PID signal while that lock remains held. An identity/ownership change detected before that final decision refuses/skips as appropriate.

Cost now names E, the gated detached candidates: each proof costs one table read plus at most E environment reads for previously unobserved generations, with cached known/unknown results reused and transient identity/cancellation failures not cached. The command bound remains 2 minutes and each exchange 2 seconds. Count K=8/16 effect requests, P=8/16 table rows and S=8/16 slots independently, plus cold/warm E=8/16 cache behavior. Require at most 2K effect requests/table reads; environment work follows unique gated generations, never a per-target unconditional subprocess. Unknown evidence may reduce eligible cleanup; it must not reduce protection.

#### Authoritative invocation transitions: PQ-2

Add private `ExactReapState`, tagged `ExactReapPhase`, `ExactReapRowState`, `ExactReapEvent` and `ExactReapEffect` in `launcher/exact_reap_model.go`. Export only the execution/outcome API needed by the CLI. `AdvanceExactReap(state,event)` is the sole writer of per-row progress and the sole producer of signal/wait/probe effects. Reaper.ReapExact is an interpreter: it runs those effects through injected IO and feeds their outcomes back as events; it cannot assign authoritative row state or synthesize a final success.

The immutable key/provenance set is fixed by Begin. Row states are pending-term, sending-term, waiting-term, pending-kill, sending-kill, waiting-kill and done. Phase states are TERM dispatch, TERM wait, KILL dispatch, KILL wait and done. Effects are dispatched sequentially; there is no concurrent outstanding signal and no spawned lifetime. All timers/observations enter as events; no clock or IO is read by the reducer.

| State/event | Next state / permitted effect |
| --- | --- |
| pending-term + next | sending-term; emit exactly one guarded TERM |
| sending-term + acknowledged sent | waiting-term |
| sending-term + refused, changed/gone or error/lost reply | done with refused, skipped or failed/uncertain; never KILL |
| All TERM replies resolved | TERM wait; emit bounded shared wait/probe effects |
| waiting-term + positive exact absence | done/reaped |
| waiting-term + known same-generation survivor AND TERM deadline reached | pending-kill |
| waiting-term + unknown probe | done/failed; never KILL |
| pending-kill + next | sending-kill; emit exactly one newly guarded KILL |
| sending-kill + acknowledged sent | waiting-kill |
| sending-kill + refused, changed/gone or error/lost reply | done with explicit result; no repeat |
| waiting-kill + positive exact absence | done/reaped |
| waiting-kill + known survivor at KILL deadline or unknown probe | done/failed |
| Any unfinished row + cancellation/deadline | done/failed with retained progress; emit no new signal |
| Duplicate/out-of-order reply or foreign key | no new effects; finished state never reopens |

Skipping after a changed identity never means the replacement ended. Positive absence after an acknowledged signal means reaped; uncertainty cannot produce that result. The public CLI statuses remain the existing four outcomes; these invocation-local tags are not new durable/user-actionable operation states. Hold state fields private so the executor cannot bypass `AdvanceExactReap`. Generate reproducible seeded event sequences plus exhaustive short sequences over all event categories; assert the frozen-key invariant, no post-cancel signal, at-most-once TERM/KILL, and KILL only after acknowledged TERM, deadline and positive survivor proof. Also replay the interpreter's effects against the stateful process world so pure-model coverage cannot hide a wiring bypass.

#### Active production-function test strategies: PQ-3

These strategies replace tasks 1–3's enumerated case inventories. Keep their file ownership, verification commands and public-command acceptance, using `exact_reap_model.go` for the pure model and `resources.go` for the shared ownership stage.

| Production function | Adversarial strategy and mechanical guard |
| --- | --- |
| `ProvenOrphanTargets` | Start from monitor-produced mixed prior/current provenance; falsify one eligibility witness per generated row. Compare exact keys to an independent expected set, with immutable input and overlap-dedup assertions. |
| `DecideProvenOrphanSignal` | Replay a frozen target through changing complete ownership proofs. Positive current/detached ownership dominates old provenance; absent/changed and unknown are distinct. Every allowed decision must match all original witnesses. |
| `resourceSampler.collectOwnership` | Reuse the real detached-suite reparenting production predicate and gated stateful reader. Compare sample and effect-proof selection on the same world; blocked/incomplete/unknown evidence cannot return an actionable proof. Count cold/warm cache calls at E and 2E. |
| `AdvanceExactReap` | Generated event sequences enforce the tagged transition table and signal invariants above; no mocks or IO. |
| `Reaper.ReapExact` | Stateful process-world interpreter conformance, including unrelated current descendants and generation changes at effect barriers. Assert every observed signal was an emitted permitted model effect and no descendant discovery added a key. |
| `authorizeProvenOrphan` | Mutate authenticated caller generation, hub role and current slot claims across an admitted target. Only entirely unbound operator or the current owning hub passes; partial credentials never become operator. |
| `handleProvenOrphanSignal` | Drive the actual handler with the worker proof seam and signal world. Change ownership/authority at barriers between TERM and KILL, inject effect/result loss, and verify a vetoed effect causes zero native signals. Hold a hub move at the read-lock barrier to prove serialization. |
| `ValidateRequest` | Mutate each allowed request field and closed signal value; strict shape rejection precedes proof/effect. Retain compatibility coverage for older supervisors and legacy calls. |
| `runReapOrphansCLI` / rendering | Exercise real service composition with mixed new/legacy rows and typed effect outcomes. Compare exact text/JSON row identity and multiplicity, return code, coverage and uncertainty; a refused row never reaches legacy fallback. |

No implementation starts until these refinements pass plan quality and the hub accepts the complete-ownership/cost clarification. Native acceptance remains the approved shared disposable-Couch public command with both Claude and Codex, never the real fleet.

### 2026-10-22 — Native acceptance refinement (approved)

Public live reboot returned its exact confirmation offer but failed at the acknowledged stop with `ownership unresolved: unknown pane owner: pane has no unambiguous actual command`, before orphan-reap effects. This independently tracked product defect is #797, filed unclaimed per the hub; it was previously only a Log mention in #791. Approval d2356d3f-461c-4576-9ceb-a71acb4179f6 accepts a fixture-only exact-generation crash of the already-proven old worker cohort, explicitly excluding the frozen six descendants and the steward. Then public reboot runs on a genuinely dead session, followed by public preview, confirmed reap and delivery to the current Claude/Codex worker. The shared disposable harness owns every fixture process and cleanup. No recovery validator, native fleet, production signal scope or deadline changes.

The startup ready-file miss observed before fixture setup stays unclassified, alongside #773's related evidence; later startup success does not establish its cause or register a flake. Native-test oracle repairs decode the public flat ResourceView, require a complete typed intentional preview refusal, and follow the public recovery confirmation envelope without weakening uncertainty checks.

### 2026-10-22 — Final implementation traceability and verification structure

Approval8c0e8477-dcd4-495d-a6b9-ea4f95285957 selects one focused Go invocation per Claude/Codex personality, with the unchanged96MiB write cap and all existing time/frame/output bounds. The earlier combined command is retained as failed evidence; the Codex missed-ready startup remains unclassified alongside #773 and the safety test is not registered flaky.

Final source locations clarify the earlier concept table: ExactReapOutcome and the integration interpreter live in launcher/exact_reap.go; ExactReapState and AdvanceExactReap live in launcher/exact_reap_model.go. State fields, exactReapPhase, exactReapRowState, exactReapEvent and exactReapEffect are private; the exported reducer has package-private event/effect types. The protocol tests live in couchmessage/reap_protocol_test.go. The production function strategies remain the governing acceptance, using systematic witness mutation rows for target selection and generated/exhaustive event sequences for the reducer.

Counted cost proof belongs to deterministic tests: independently vary K/P/S at8/16, require one table and inventory read for each of2K proof collections, and measure cold/warm E at8/16. Executor tests independently bound effect emissions and shared wait snapshots at K8/16. The shared classifier's existing ordering is reused unchanged; these counters bound added observation work, not an invented classifier-comparison metric. Native qualification measures exact six-key outcomes, elapsed time, current-worker delivery, cleanup and total write bytes; it does not instrument internal native syscall counts. This clarifies the measurement boundary without changing any runtime bound or allowed work.

### 2026-10-22 — BR-1 complete overlap protection before projection

The close review reproduced an unsafe information-loss boundary: independently bounded orphan and legacy sections can omit the orphan witness but retain the same PID as a legacy tree-reap candidate. ARCH-PURPOSE/SECURE/CONSTRAINTS require omission never to widen authority. Apply overlap exclusion to both legacy families using every orphan PID in the complete supervisor ResourceView before its first lossy projection, including cached sampler projection. Preserve the CLI exclusion as defense against incomplete visible witnesses. No new protocol, state, signal scope or bounds. Add a regression through actual response projection and CLI execution with an omitted overlapping orphan, preserving unrelated legacy rows and source immutability. Retain the omission notice; omitted targets are not acted on. Repeat final native qualification and recorded verification on the corrected source.

### 2026-10-22 — BR-2 approved re-plan: lifetime refusal after provenance loss

Authenticated hub pin08398b83-dc96-4884-8c39-48c93a17f479 changes the prior legacy-compatibility constraint: safety wins over preserving legacy tree execution after incomplete evidence. One sampler-worker-owned in-memory sticky flag records any non-authoritative ownership collection since this Couch process started. The sampler is constructed once by the process service owner; the flag is never persisted and only a fresh Couch process clears it. No separate exclusion memory, changed cap or durable schema.

A pure completeness decision checks table/inventory completeness and omission/cap bounds, classifier errors/truncation/conflicts, duplicate or missing live identities and unknown detached ownership. Every shared collection, including effect proofs, can set the flag; no collection clears it. The existing worker reply carries its current refusal fact to legacy admission, without a second authoritative flag or mutable reader. Once sticky, confirmed orphan classification returns an explicit provenance-incomplete code and the bounded exact proven group, with no actionable legacy rows. Older CLIs refuse that non-ok code; the new CLI can still execute independently guarded exact targets and reports a nonzero legacy refusal with the exact wording: provenance incomplete; use the exact proven reap or restart Couch. Legacy tree signals never follow that refusal. Normal complete-process legacy behavior and all exact effect vetoes/cadence remain unchanged.

This is an intentional conservative process-lifetime refusal, even after a later scan looks healthy; returning capacity is not proof that forgotten generations are gone. It closes table/slot/selection loss, unknown/conflicting classification, prior-provenance replacement and downstream projection as one omission class. Keeping only the old bounded prior list plus a temporary refusal is rejected: newly seen owned rows can exceed that list, then lose their witness on another scan. A larger exclusion store or persisted latch is unnecessary and unapproved.

Actions remain inspect resources, exact confirmed reap, and restart Couch when the operator chooses. Visible legacy state after any loss is refused; a later healthy pass keeps it refused; exact candidates still require their full current proof; a fresh process starts with an unset flag but its first incomplete collection immediately sets it. No earlier signals are rolled back and there is no automatic restart/retry.

Tests: the real sampler/service/CLI sequence covers overflow, repeated overflow then healthy, conflicting ownership then healthy, incomplete census, unknown identity/detached and table/slot bounds. Both legacy families must remain inert after the flag sets, including subsequent healthy passes, while a fresh isolated service starts clean and exact proven targets remain usable. Pure completeness tests and counted N/2N loops guard added bounded linear work. Repeat focused native Claude/Codex, recorded verification and full close on the corrected source. BR-3 independently fixes the RPC fixture's long-TMPDIR socket root using the existing short-root pattern; retain long-path red/green evidence.

### 2026-10-22 — Approved initial seed ordering

Pin dc2b6d09-7e95-437f-bf1c-0f77972be5c0 distinguishes cold unobserved startup from an incomplete observation. The inventory and sampler currently start concurrently; the initial inventory default is unknown, so sampling it immediately would make the new sticky refusal universal. One owned completion barrier orders the sampler after the existing inventory seed attempt, with cancellation inherited from the service. No new wait duration, retry or deadline. Resource requests remain unavailable under their existing context until the sampler starts. Failed/incomplete seed results are observed normally and set the sticky flag; healthy seeds allow ordinary legacy admission. Test delayed, healthy and failed seed completion and cancellation. The barrier is transient process startup coordination, not persisted state or a reset for the refusal flag.

### 2026-10-22 — Current-proof contract and typed native partial result

A deterministic actual-handler regression found a distinction between useful monitor evidence and destructive proof: positive death of an old root can keep monitor orphan provenance known while a current owning root is unconfirmed. The effect reply now additionally requires ResourceOwnershipComplete for the current collection; it does not substitute the historical sticky flag. Unknown current ownership therefore refuses both TERM and KILL even after old-root death. This enforces the already approved full current-proof contract with the same pure completeness source, without another state, protocol or bound.

The first sticky native run reported all six exact outcomes reaped plus the documented legacy_refusal/exit1; the old exit0 assertion stopped before current-delivery verification, so that command remains failed evidence. The acceptance oracle now permits only joined exit0/no-refusal or joined exit1/the exact documented legacy-refusal value. It still requires six unique exact reaped rows, independent native absence, current generation preservation, public delivery and complete cleanup; truncation, uncertain cleanup, unrelated legacy outcomes or any other failure remain failures. No deadline/frame/output/write limit changes.
