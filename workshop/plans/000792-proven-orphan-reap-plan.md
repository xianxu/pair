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

- [ ] Write `TestProvenOrphanTargetsUseExactPriorOwnership` and `TestProvenOrphanSignalDecision` first. Use production SelectResourceProcesses over old+live generations, same-thread relaunch, reparented survivor, current-root ownership transfer, empty/duplicate/reused identity, incomplete census and overlapping registry entries. Assert the exact allowed set and zero current keys.
- [ ] Add strict protocol tests for complete/unbound/partial caller forms, malformed target, nonconfirmed request and signals other than TERM/KILL. Confirm old Op+Confirmed callers retain the existing families and do not gain a new client-side signalling capability.
- [ ] Implement the narrow types/pure decisions and request schema; verify red then green with `go test ./cmd/internal/couchcore -run 'TestProvenOrphan' -count=1` and the named couchmessage selection. Register new production sources in the exhaustive artifact inventory.

### 2. Exact executor and supervisor effect owner

Files: new launcher/exact_reap.go / exact_reap_test.go; new couchcmd/reap_proven.go / reap_proven_test.go; message_service.go and the existing hub/authority seams.

- [ ] Write `TestExactReapNeverExpandsDescendants`, `TestExactReapEscalationAndUncertainty` and stateful service tests before implementation. Inject ownership/caller/hub changes before TERM and between TERM/KILL; reused/empty identity; permission error; canceled context; lost reply; surviving SIGKILL. Include a live child beneath an old root and an old child reparented to PID1. Assert no signal outside frozen eligible keys and one outcome per target.
- [ ] Implement ReapExact without changing existing Reap(tree) semantics; obtain its timings from the same Reaper configuration. Implement supervisor reproof/signal requests with no wait in the handler. Exercise the real handler through its stateful table/inventory seam, not a stub asserting request spelling.
- [ ] Count K/P/S growth at8/16 and bound all waits. Run `go test ./cmd/internal/launcher -run 'TestExactReap|TestReap' -count=1`, the named service selection and corresponding `-race` tests.

### 3. CLI, output and public-command conformance

Files: reap_orphans.go / reap_orphans_test.go, messages.go; new reap_proven_live_test.go beside the existing reap_orphans_live_test.go; reuse cmd/internal/testcouch, no private fleet harness.

- [ ] Write preview/confirmed text+JSON tests carrying both the new proven group and legacy groups, duplicate keys, mixed own/foreign hubs, unsupported older supervisor and partial/uncertain results. Test authentic owning hub, worker, different hub, stale binding and unbound operator through real service composition. Preserve output sanitization, coverage and existing return codes.
- [ ] Wire the CLI to ReapExact and supervisor effects; keep legacy families working and prevent overlap from reaching the expanding-tree path. Preview names each proven target and never signals. Refusals never invoke fallback signal code.
- [ ] Native public command on a disposable Couch: keep a current slot/wrapper/agent alive, retain known previous-incarnation fixture processes, invoke preview then confirmed reap, assert only owned old identities end and the current slot still accepts a message. Cover Claude and Codex fixture kinds. Use acknowledged lifetime barriers, exact cleanup and PID-incarnation checks rather than sleep-based readiness. Never use the live fleet as a fixture. Ask the hub for the required stage/live acceptance window separately.

### 4. Documentation and gates

- [ ] Update README, embedded Couch skill, atlas/couch.md resource/reap sections and index: authority, exact-set scope, no descendant discovery, partial outcomes and explicit recovery after uncertainty. Update the issue's final acceptance and Log without rewriting prior evidence.
- [ ] Run the complete artifactpath guard and affected-reader inventory audit before the recorded check. Classify any new reader using the per-reader regenerate command; no new quarantine or exemption. Run relevant public-command and identity/authority regressions, then the declared pushed-head `sdlc local-check run --name affected` (full fallback when selected).
- [ ] Complete required live acceptance agreed with the hub, SDLC close review, fix findings, land through sdlc merge, and report the PR and exact evidence. No stage on :0 without hub permission.

## Architectural decisions

ARCH-DRY/PURE: reuse SelectResourceProcesses, existing caller/hub facts and Reaper cadence; target membership and outcome decisions are pure. ARCH-PURPOSE: include every proven process kind, not just an editor/agent subset, without touching the current incarnation. ARCH-MOCK: stateful tables and public disposable-Couch conformance model interactions and races. ARCH-CONSTRAINTS: budgets, caps and counted K/P/S costs are explicit. ARCH-SECURE: supervisor treats caller-supplied target bytes as claims; authority and identity are checked at each effect. ARCH-STATE: exact frozen set plus per-row progress preserves refusal/uncertainty and forbids resend/escalation after a lost reply. ARCH-FUNERAL: invocation-scoped plans only, no new persistent residue. ARCH-EVOLVE: no persisted schema change; wire fields are additive and operation shapes remain strict, with explicit unsupported handling for older supervisors.

## Revisions

- 2026-10-22 — Hub approval cea188f7-9b5c-4999-879d-59404fd1971a accepts the full design and selects disposable public-command conformance with Claude and Codex as live acceptance. No operator/:0 stage is needed; this supersedes the proposed stage-window request in tasks 3/4. The first real post-landing use belongs to ops-echo against echo:2's still-proven leftovers, not this test invocation.
