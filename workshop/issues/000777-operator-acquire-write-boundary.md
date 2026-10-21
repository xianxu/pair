---
id: 000777
status: working
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours: 2.17
collaboration-mode: collaboration
card_mirror: 'b8b4c755376e7baf82e3dfb312aa5c8c48e756b5' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T14:41:07-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# session-sync: serialize operator acquisition with the writer commit boundary

## Problem

The operator requested closing the residual InsertEnter race after parley.nvim#404. Auditing cmd/internal/sessionsync/sync.go shows f.guard at write entry, preparation and replaceShared later, and only a post-write sidecar inspection that emits a warning. The source explicitly notes that guard and write cannot be one step against Parley, which does not take the Couch flock. An agent writer that passed the guard can commit after the operator creates the lock.

## Spec

Define and implement a single exclusion resource/linearization boundary shared by registered operator acquisition and session-sync document writes, so an acknowledged acquisition excludes earlier in-flight writes and future writes observe held protection before committing. A repeated sidecar check without shared exclusion is insufficient. Coordinate the acquisition/acknowledgment contract with Parley's InsertEnter follow-up. Preserve lock ownership, CAS, journal/recovery behavior and the ordinary registered write path; no two-phase protocol.

## Done when

- A deterministic writer pause between guard and mutation races operator acquisition and proves no writer commit crosses an acknowledged operator lock.
- Registered write/apply paths and migration protection use the intended shared exclusion contract; refusal leaves document and metadata consistent.
- Consumer-facing acquisition/acknowledgment semantics and failure recovery are documented and communicated to parley.nvim:1.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only. The named calibration is marked stale by estimate-source, so these hours are provisional. Scope is the producer plus its executable proof and documentation; parley.nvim#408's implementation is excluded.

The pure ownership model uses the greenfield-Go primitive: base design 1.0 h, halved for existing registry/ParseHolder/durablefile building blocks, then multiplied by 0.2 for the approved detailed contract = 0.10 h. Its base implementation 0.6 h becomes 0.24 h at the v3.1 40% scale. Transaction/CLI integration with uncertain retries uses API-integration base design 2.0 h × 0.2 and implementation 1.5 h × 0.4. A separate smaller module covers real executable conformance (base 0.3/0.5 h), not the integration's unit/folder tests. Spec, docs and the one close review use their own primitives below. Familiar Go/filesystem/CLI territory gives familiarity 1.0; the thorough plan gives a 15% design buffer. No new library discovery is needed: existing in-repository primitives supply the mechanism.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.10 impl=0.04
item: greenfield-go-module design=0.10 impl=0.24
item: api-integration design=0.40 impl=0.60
item: smaller-go-module design=0.06 impl=0.20
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0.02 impl=0.20
design-buffer: 0.15
total: 2.17
```

Derivation: design 0.70 × 1.15 + implementation 1.36 = 2.165 h, rounded to 2.17 h.

## Plan

- [x] Agree the pinned CLI, ownership, lost-ack, rename and release contract with parley.nvim:1 through the hub; approve the [durable plan](../plans/000777-operator-acquire-write-boundary-plan.md).
- [x] Implement pure turn/proof decisions and acquire/release inside the existing writer registry transaction, with deterministic writer/acquire ordering and uncertainty tests.
- [x] Expose strict public commands and machine-readable refusal/unavailable outcomes; prove real executable and move-copy recovery on disposable files.
- [ ] Document, run the required recorded check, close through SDLC, land on SHIP and report availability to the hub for parley.nvim#408.

## Log

Filed by parley.nvim:1 for its TL after read-only producer audit; unclaimed for producer dispatch. Operator request is Couch 61805ff7-ac98-4fcf-9a32-9a8215d5b878. Existing write path: cmd/internal/sessionsync/sync.go around lines 617, 651 and 671. parley.nvim#404 landed PR #327; its residual-race follow-up is separate.

### 2026-10-21 — producer contract and plan checkpoint

Started from fresh main after #768 landed. Inspection confirms withRegistry already encloses registered writer document/shadow/state/round effects; the missing participant is operator acquisition. ARCH-DRY: reuse that exclusion resource and existing sidecar lifetime, rather than a second guard or two-phase protocol. Proposed operator-acquire/operator-release require current FILE, registration ID and caller 32-hex turn token; the token persists in exact LockText and survives existing move journals. Retired bare release remains unchanged.

Hub receipt d359cf75-8956-45ac-b6ac-ae5957cca584 relays parley.nvim:1 agreement on argv, identity, canonical File, same-turn idempotence, success envelope and stopped-supervisor behavior. Its three added pins are in the durable plan: explicit session-sync-unavailable/7 distinct from ownership/path/identity refusals; document/turn fields recoverable from copied lock bytes and release at NEW after rename; same release envelope with released/already-released and no ownership inferred from absence. The final unavailable taxonomy and full plan await hub approval. No production code or estimate yet.

Hub approval 061be603-8e01-4db6-99ab-0572a62d93f6 accepts the full plan and relays its taxonomy to parley.nvim:1. Public-command conformance on disposable files is sufficient producer live acceptance; no :0 stage. Document the direct-sidecar-client limit in the atlas. Entering change-code; the full-flow plan-quality gate precedes estimate derivation and implementation.

Plan-quality round 1 raised PQ-1 (already-absent release retries must confirm directory durability) and PQ-2 (name production test functions and compress case inventories to adversarial strategies). Verified the durability requirement against durablefile's file-plus-directory publication contract, then clarified both success/retry paths and the fake's visible/durable states. The revised plan adds function-level strategies and removes the duplicate task inventories. No contract or scope change; rerunning the gate before implementation.

Plan-quality is CLEAN with PQ-1/PQ-2 addressed; change-code admitted full flow after the derived 2.17 h estimate (estimate-quality INFO: integration/conformance allowance is optimistic, and the review primitive also covers recorded verification/landing). First executable parser regression failed as expected on unknown operator-acquire. Pure ownership/proof tests then failed on missing model definitions and passed after the model implementation (`go test ./cmd/internal/sessionsync -run '^TestOperatorLockModel' -count=1`, 0.144 s). Transaction tests are now written and red on the missing AcquireOperator/ReleaseOperator methods; implementation follows. A bounded subagent owns only CLI parser/result wiring and its tests; the main session owns exclusion, uncertainty and migration.

### 2026-10-21 — implementation and executable acceptance

The existing registry transaction now owns operator acquire/release, exact ID/token decisions and bounded staged publication. Pure proof parsing preserves unknown lines, rejects ambiguous known fields and recovers tokens from move-journal bytes. Directory confirmation covers existing and absent retry states. Foreign protection is never adopted/deleted; registration/path refusals preserve document and metadata. Existing direct-sidecar warning remains, with its exclusion limit documented in the atlas and public Couch skill.

Deterministic Write and Apply tests pause after the guard and again at the final round publication; a contender proves real flock contention at both boundaries. Future Write/Apply under acknowledged protection refuse with unchanged document/shadow/state/round. Visible-versus-durable fake sequences cover uncertain publication, absent release retry and partial alias deletion. Real child death before and after publication proves stage reclamation and exact lost-ack recovery. Real-backend counts at 8/16 aliases prove one observation per admitted spelling, one acquire publication and linear release effects. Move-copy/rename/release-NEW and forgotten/replaced identity are exercised.

Verification returned: full sessionsync package PASS 8.619 s; targeted ownership race run PASS 1.846 s; combined operator CLI/public executable PASS couchcmd 1.469 s and cmd/couch 7.182 s; ownership-proof fuzz 108414 executions in 2 s PASS; tracked Go catalog PASS 3.970 s. Native public-command acceptance uses disposable adopted roots, proves the daemon absent before/after, discards an acquire reply then recovers exact bytes/time, refuses protected writes unchanged, moves/recover/releases the token, and distinguishes typed failures. No :0 stage.

Executable tests first exposed runtime preparation before the session-sync handler returning plain exit 1; fixed that operator-only boundary through the same unavailable/7 JSON owner. An additional red test proved invalid turn data could otherwise reach namespace-unavailable; validate identity/token before runtime IO now, and the targeted operator selection passes 0.533 s. Existing malformed argv remains usage/2; valid argv with invalid ID/token is operator-invalid/1. The clarification went through the hub in receipt 83a7a713-7bb4-4988-a501-fc1dfa81d91c.

The affected-reader audit identifies only the model test's literal invalid ../ token; it reads no repository source and its new per-reader certificate is local_or_literal_fixture, regenerated with make affected-readers. Inventory check and diff whitespace check pass. No local-gates-guard protected path is changed. Production, tests, docs and evidence are checkpointed before the required pushed-head recorded check.
