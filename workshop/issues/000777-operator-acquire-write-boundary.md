---
id: 000777
status: working
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '575a1b4a08ce264372c7bf53bed00be7a9645670' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T14:41:07-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
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

## Plan

- [ ] Agree the pinned CLI, ownership, lost-ack, rename and release contract with parley.nvim:1 through the hub; approve the [durable plan](../plans/000777-operator-acquire-write-boundary-plan.md).
- [ ] Implement pure turn/proof decisions and acquire/release inside the existing writer registry transaction, with deterministic writer/acquire ordering and uncertainty tests.
- [ ] Expose strict public commands and machine-readable refusal/unavailable outcomes; prove real executable and move-copy recovery on disposable files.
- [ ] Document, run the required recorded check, close through SDLC, land on SHIP and report availability to the hub for parley.nvim#408.

## Log

Filed by parley.nvim:1 for its TL after read-only producer audit; unclaimed for producer dispatch. Operator request is Couch 61805ff7-ac98-4fcf-9a32-9a8215d5b878. Existing write path: cmd/internal/sessionsync/sync.go around lines 617, 651 and 671. parley.nvim#404 landed PR #327; its residual-race follow-up is separate.

### 2026-10-21 — producer contract and plan checkpoint

Started from fresh main after #768 landed. Inspection confirms withRegistry already encloses registered writer document/shadow/state/round effects; the missing participant is operator acquisition. ARCH-DRY: reuse that exclusion resource and existing sidecar lifetime, rather than a second guard or two-phase protocol. Proposed operator-acquire/operator-release require current FILE, registration ID and caller 32-hex turn token; the token persists in exact LockText and survives existing move journals. Retired bare release remains unchanged.

Hub receipt d359cf75-8956-45ac-b6ac-ae5957cca584 relays parley.nvim:1 agreement on argv, identity, canonical File, same-turn idempotence, success envelope and stopped-supervisor behavior. Its three added pins are in the durable plan: explicit session-sync-unavailable/7 distinct from ownership/path/identity refusals; document/turn fields recoverable from copied lock bytes and release at NEW after rename; same release envelope with released/already-released and no ownership inferred from absence. The final unavailable taxonomy and full plan await hub approval. No production code or estimate yet.

Hub approval 061be603-8e01-4db6-99ab-0572a62d93f6 accepts the full plan and relays its taxonomy to parley.nvim:1. Public-command conformance on disposable files is sufficient producer live acceptance; no :0 stage. Document the direct-sidecar-client limit in the atlas. Entering change-code; the full-flow plan-quality gate precedes estimate derivation and implementation.
