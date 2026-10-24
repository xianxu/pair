# Fresh explicit-hub startup implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Use superpowers-executing-plans for the session-warm startup integration. Steps use checkbox syntax.

**Goal:** Start a genuinely new configured fleet normally, with hub automation off, without mistaking an absent startup child for a failed attachment.

**Architecture:** Preserve the existing read-only startup observation before any new startup launch. Its pure classification decides whether configured hubs may start; existing lifecycle launch/adoption paths own the effects. Known shutdown and unknown evidence remain conservative. There is no speculative broker-registration change.

**Tech stack:** Go, existing Couch core/console, shared testcouch owner and native Claude/Codex fixtures.

## Evidence and alternatives

Main99846a5015 single-hub production form gesture admits the primary within2s. Exact two-hub PRIMARY WORKER1 run also admits the primary (observation0), while reproducing the extra hub's empty-result attachment error. Source traces start-hub -> StartInteractive returning zero StartResult,nil -> Started() unconditionally true -> adoption identity refusal. Pair:6's persistent refusal remains unclassified; its repeated-Enter helper is under correction, not evidence for a registry fix.

Selected: classify fresh once before effects, then use existing launch machinery. Rejected: keep fresh fleets parked and teach every harness to compensate (contradicts operator intent); globally make StartInteractive launch whenever absent (would auto-resume/start after a known shutdown); add registration retries (no reproduced registration defect).

## Core concepts

| Pure entity | Lives in | Status |
|---|---|---|
| StartupKind / ClassifyStartup | cmd/internal/couchcore/startup_policy.go | modified: fresh is proven zero history, not absence after history |
| StartResult.Started | cmd/internal/couchcore/ops.go | modified: empty result has no child; real handle preserves normal adoption |
| start-form admission | cmd/internal/couchtty/menu.go and menu_operations.go | existing owner; qualify repeated Enter before selecting any fix |
| Options.Hubs | cmd/internal/testcouch/fleet.go | modified: ordered explicit worker directories, same #830 contract |

One StartupObservation belongs to one console startup and is never recomputed after its own launches. Fresh requires zero retained records and no unreadable/unresolved evidence; known records absent mean full-shutdown, positive survivors mean normal, uncertainty means unknown. Existing parked/scope rules remain. Hubs use the same observation, not independent clocks or per-hub scans (ARCH-PURE, ARCH-DRY).

| Integration | Lives in | Status | Wraps |
|---|---|---|---|
| startup orchestration | cmd/internal/couchcmd/run.go, hubs.go | modified | initial primary attach and serial extra-hub lifecycle queue |
| StartFreshHub / start-hub dispatch | cmd/internal/couchcore/startup.go and operationdispatch.go | modified | one exact target resolution, existing ordinary Open / numbered Fresh path |
| shared fleet acceptance | cmd/couch/fresh_hub_registration_live_test.go | new | real private supervisor, native fixtures, inherited caller CLI |

Fresh launch permission is internal startup policy from the recorded observation; no automatic agent message is authorized by it. Launch the primary through the existing explicit open path after observation, before initial adoption; queue other configured hubs serially through the existing lifecycle workers with the same fresh permission. For normal/full-shutdown/unknown, retain surviving-only StartInteractive. An empty surviving-only result is a successful no-child result, never adopted or aborted. Automation initialization consumes the original observation, so newly launched hubs cannot turn it on.

## Transition and operator projection

| Observation/event | Permitted effect | Operator sees / recovery |
|---|---|---|
| Proven fresh | Start configured hubs once, primary selected; automation off | startup: fresh; hub automation off; say turn it on |
| Normal, selected hub survived | Reattach exact surviving sessions; per-hub automation follows existing survival rule | normal restart, missing hubs retain explicit recovery advice |
| Prior records absent | No cold launch or resume | full-shutdown; Resume from disk / Ctrl+Space Start |
| Unknown evidence | No speculative launch | unknown plus diagnostic and explicit recovery action |
| Surviving-only start returns empty | No adoption effect | empty hub stays available in switcher, no false identity-failure notice |
| Start form admitted, Enter arrives before completion | At most one launch for that form | existing operation progress; later Enter cannot create another start |
| Launch/attach failure or cancellation | Existing owned cleanup and row result; no retry | one result with existing recovery action |

No new executor, queue, retry, store, protocol or agent-message producer. Existing lifecycle reservations continue through adoption; late completion keeps existing focus rules (ARCH-ORDER).

## Operating envelope and trust

Startup only: reuse one inventory/session observation and existing worker cap/serial hub queue. Startup latency uses #773's caller-owned budget; no timeout increases. Native qualification is bounded at8 minutes per command with private roots, one run per personality after deterministic tests. Real CLI caller lookup is checked separately from fixture readiness. Existing exact-generation cleanup owns descendants; evidence is retained outside removed private roots. No live fleet/pair:0 involvement and no real agent credentials. No persisted format changes (ARCH-EVOLVE); StartupKind is ephemeral output. Existing thread and diagnostic retention owns production residue; no new durable runtime artifact.

## Implementation

### Task1: pure startup and child-result boundaries

Files: startup_policy.go/startup_policy_test.go; ops.go and colocated adoption tests.

- [x] ClassifyStartup: generate incomplete and contradictory snapshot/presence combinations; an independent predicate forbids fresh whenever history or uncertainty exists, and permutation checks forbid input-order dependence. Confirm red on readable zero-history input before changing classification.
- [x] StartResult.Started: vary the record and handle independently, with handle possession as the independent adoption-authority oracle; confirm the empty result fails before the fix and inspect sibling Started implementations for the same contract.
- [x] Run targeted tests, confirm intended failures, implement minimal classifier and no-child correction; rerun startup/adoption tests.
- [x] Commit with #863 and model trailers.

### Task2: orchestrate only proven fresh launches

Files: couchcmd/run.go, hubs.go, automation.go consumers and hubs_startup_test.go; couchcore/operationdispatch.go.

- [x] runConsole/startExplicitHubs: use the existing stateful fake process/store boundary with held launch/adoption and cancellation; count effects against the immutable initial observation, independently forbidding cold effects without fresh evidence. startupAutomationOn: mutate subsequent live inventory while retaining the initial observation and require automation to stay off for fresh startup.
- [x] Observe before new launches; route permitted fresh primary/extra effects through existing explicit-open/lifecycle owners. Keep surviving-only paths conservative.
- [x] Update startup summary for fresh (not missing/resume-from-disk language for never-started hubs); keep real launch failure recovery advice.
- [x] Run core/cmd/tty startup and automation targeted tests; commit.

### Task3: qualify form duplicate admission

Files: couchtty/menu start tests / console lifecycle tests; production reducer only if a failing test proves a gap.

- [x] reduceStartKey/dispatchMenuOperation: hold effect completion and generate repeated Enter plus duplicate/late result events; independently count start effects per form identity and require at most one. Current source synchronously closes admitted form; do not add redundant state if the invariant already holds.
- [x] Ask pair:6's corrected native run to retain whether a second actual launch occurred. If reproduced, fix in this issue with one consumed-form identity until outcome; no new retry. If it needs another mechanism, stop and report to hub before expanding.
- [x] Record production versus harness classification with evidence.

### Task4: native acceptance and close

Files: shared testcouch Hubs option/tests; fresh_hub_registration_live_test.go; relevant atlas startup/hub docs and issue Log.

- [x] Diagnostic Hubs argv input red then green (same #830 contract), exact two-hub main caller admission measured.
- [x] Convert diagnostic into durable Claude/Codex fresh two-hub acceptance: Fleet.Start returns with actual live callers, both hubs registered, no false hub-failed notice, automation off, no automatic hub input. Keep explicit known-shutdown gesture test where appropriate; no duplicate fleet owner.
- [x] Run one bounded native qualification per personality. On failure retain full wrapper and console facts, classify before another run.
- [x] Coordinate #830 integration; whoever lands second takes identical Hubs contract. Preserve #830 T6 skip until its owner verifies it.
- [x] Document fresh vs known shutdown, update atlas and issue Log, audit reader inventory, run make test-affected-inventory.
- [ ] Push; run sdlc local-check run --pre-close, close with actual evidence and --no-project; on SHIP merge main then sdlc merge --yes. Report real merge/archive state to hub. No live stage requested.

## Revisions

2026-10-24 — Hub563a3e8a: zero-history is fresh and hubs start, automation off; no blanket relaxation after known shutdown. Hub c1133e63 authorizes only the small shared Hubs harness option before #830 lands. Hub bc2786cd requires duplicate Enter production qualification if the peer run proves two launches. Measurement currently reproduces empty adoption, not persistent registration failure; no registry fix planned.

2026-10-24 — PQ-1: compress test inventories into named risky-function strategies, with generated contradictory evidence, independent adoption authority, held effects and at-most-one launch invariants. Architecture and selected behavior unchanged.

2026-10-24 — Native qualification confirmed Open intentionally cannot create a never-used managed slot (resume-no-survivor). Hub7cf1b120 approves the existing Fresh path for this zero-history-only permission. Use StartFresh for initial configured hubs; known-shutdown/unknown remain untouched. Original evidence still owns automation. Retained Zellij v0.45.1 RemoveClient unwrap panic is an external lifecycle fault, separately recorded, not a registry patch or timeout change.

2026-10-24 — Hub9a53d319 approves consolidation after the second path correction: core StartFreshHub resolves the exact target once with Open intent, uses StartFreshSlot only for a numbered target, otherwise shares the existing resolution-based ordinary open implementation. Both initial primary and extra-hub startup consume this helper. Real OS SlotCatalog-backed tests for both target kinds red on missing helper, then green; no relaxed generic StartInteractive semantics. Stop and re-plan if a third path correction is needed.

2026-10-24 — Implementation and native/deterministic qualification complete. The final checkbox is post-implementation gate procedure: final recorded check is queued, close review overlaps under tl-worker PERMISSIONS, and no landing precedes final-head recorded pass. No unimplemented product behavior is waived.
