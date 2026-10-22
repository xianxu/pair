---
id: 000778
status: working
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '9d8f9b1d406856908db6dee43a406e6e15e5833e' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T18:23:34-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flake_tests: [github.com/xianxu/pair/cmd/couch.TestBareCouchInstalledCommand]
flow: {kind: quick, provenance: inferred, spec: "6653e200", done: "855df120"}
---

# flake: TestBareCouchInstalledCommand/pty_launches_and_attaches_current_hub

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/couch.TestBareCouchInstalledCommand/pty_launches_and_attaches_current_hub: Exact source3279e1e9ec6ccbddba0b50193f5c8f72c11f4b22: train238 suite lease-1293856745 failed cmd/couch TestBareCouchInstalledCommand/pty_launches_and_attaches_current_hub after15s with live/contested terminal 📁1-1; log .git/sdlc/test-stats/inflight/de3d7e12cd48155a874dbd8db94ac3f6/run.log. On the unchanged source, env -u PAIR_TEST_BINARY go test ./cmd/couch -run ^TestBareCouchInstalledCommand$ -count=1 -timeout=90s -v passed11.305s (PTY2.12s). This fixture has private socket roots but fixed terminal name; SessionServers filters only session basename, a lead not yet a forced reproduction. Production scribe is unchanged. Track this intermittent fixture refusal separately from #768.

## Spec

Fresh private identity authorities each start at C=1; their first terminal is therefore the same 📁1-1 even when socket roots differ. The launcher deliberately uses a host-wide name census. Preserve that refusal and give fixture authorities distinct high-range store IDs instead (ARCH-PURPOSE). Both installedEnv and testcouch.allocateRoots construct this class of authority; initialize both through one testpair helper (ARCH-DRY).

The helper creates a new private registry exclusively, seeds its first C from the live test-process PID plus a bounded atomic fixture sequence, and leaves enrollment and N/M allocation to IdentityStore. Each fixture owns one store. PID supplies disjoint concurrent-process namespaces; sequence distinguishes fixtures in one process. Reject invalid PID, sequence exhaustion and existing registry rather than overwrite. This is concurrent-fixture uniqueness, not permanent identity: crashed residue/PID reuse still encounters the unchanged fail-closed census. Existing high-range fixture precedent stays disjoint. Do not weaken launch admission, add retries or change readiness budgets.

Alternatives: socket-only isolation is insufficient because census is host-wide; serializing suites would hide the shared-name defect and reduce capacity. Decimal temp suffixes alone do not guarantee uniqueness across temp parents. The OS process incarnation owns the namespace for the lifetime of its fixtures, whose existing cleanups join children before releasing their roots (ARCH-ORDER/FUNERAL). The private seed file is one bounded record in the already-owned identity root and is removed with that root; no new shared state or production format changes. Work is constant per fixture: one sequence reservation and one exclusive seed write; normal allocation is unchanged (ARCH-CONSTRAINTS/PURE/EVOLVE). Existing stateful native fixture and shared disposable harness remain the external seam (ARCH-MOCK).

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.
- Two actual independent installed environments allocate distinct C/session/tag through the real allocator; first N/M=1 and later N/M advance under stable C. Equivalent shared-fleet authorities also remain distinct.
- The existing installed PTY launch passes with an expectation derived from its allocated C. Seed validation, no-overwrite and concurrent fixture coverage pass; an AST catalog rule catches omitted seeding at both constructors.
- Recorded full suite and fresh close review pass; retain original failures and subsequent passes without treating a single native pass as proof of absence.

## Plan

- [ ] Reproduce independent installed-environment name equality deterministically; retain the saved #775/#757 failure references and source trace.
- [ ] Add one exclusive fixture-authority initializer, apply it to installedEnv and testcouch roots, derive the installed launch expectation, and enforce both constructors in Pair's lint catalog with positive/negative tests.
- [ ] Run targeted allocator/constructor/PTY/lint checks and inventory preflight, record the full suite, close, land, and retire the flake registration through close.

## Log

### 2026-10-21

### 2026-10-21 — recovered evidence and bounded plan

#775 landed in train254 through PR534; its retained installed-fixture-778.log and peer757-installed-fixture-778.log under workshop/evidence/000775 preserve the original fixed-name refusal. Peer #757 unchanged-head exact-package rerun passed7.556s (receipt4a5871e0); original failure and pass remain distinct evidence. Read-only tracing confirms private host authorities default NextC=1, first M=1, and native census selects matching basename across all socket roots. The old failure does not identify its competing process; the new deterministic allocation regression will establish the collision precondition without depending on scheduler timing. Bounded independent audit found the same constructor class in testcouch and the cross-parent weakness of decimal suffixes. Quick-flow fixture repair, no public product state or gates change; post plan to hub and proceed unless objected under tl-worker rules.

## Revisions

### 2026-10-21 — concrete fixture-isolation acceptance

Expand the original root-cause Done when into deterministic allocation, preserved admission, both constructor consumers, exclusive seed safety, lint enforcement and recorded verification. Preserve the known-flake history; closing retires its existing parent registration.
