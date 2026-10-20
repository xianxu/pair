---
id: 000744
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '0b0fbb168e6a61e651b7fbfb402464e2c3223a43' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T14:46:17-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
---

# testing: one shared disposable-Couch harness for every live test (isolated roots, real supervisor and slots, every agent kind, exact cleanup)

## Problem

Live tests each build their own disposable Couch: pair#722's three-slot drain matrix, pair#726's held-cold resumes,
pair#740's global park, pair#741's hub-resume conformance, and pair#743's conformance in a VM. Each reinvents
isolation (state folder, socket namespace, HOME, queue root) and cleanup, and a mistake in any of them can touch the
live fleet. Operator, 10-20 (ops side doc `2026-10-20-19-conformance-structure.md`, decision 1): one shared harness.

## Spec

Build one shared `cmd/internal/testcouch` helper for the native disposable fleets needed by #722, #740 and #741. TL2d5e80a3 explicitly selects real installed Zellij topology under isolated roots and dispatches this issue independently from main. No #722 commit or drain API is a dependency. Its claim/PR remain parked until this helper lands.

The helper owns a fresh root lease, empty environment, actual Git primary/remote and N numbered workspaces, one real supervisor/steward, actual Pair/wrappers and native Zellij sessions. Real installed SDLC/Weave provision minimal repositories without Brewfiles/Makefiles; no invented setup markers or caller bindings. Public CLI runs inside the controlled agent's inherited registered environment. All mutations and cleanup target only the owned fleet. A deterministic fake-agent engine has Claude and Codex personalities; these are fixtures, not installed models or behavioral conformance to real agents. Qoder and real-model/network runs are outside this dispatch.

Normal tests qualify pure confinement/ownership/agent reducers with stateful process/filesystem doubles. A named opt-in native test consumes the shared helper, starts a genuine steward and three non-hub workers, exercises actual messaging for both supported personalities, detach/restart and exact cleanup. It is the first live-test consumer of the helper; the full #722/#740/#741 scenarios resume in their own issues. Existing bespoke tests migrate when next touched, rather than a fleet-wide refactor here. Full design: [shared native harness plan](../plans/000744-shared-native-couch-plan.md).

## Done when

- One shared helper creates only private roots/environment, actual native supervisor/Pair/wrapper/Zellij and a configurable bounded number of genuine numbered workers; provisioning and caller identity use production routes.
- Deterministic Claude/Codex fixture personalities support recognized input, controlled work/picker/idle, public CLI execution and native conversation artifacts for consumer-owned lifecycle scenarios, without credentials or network.
- Every direct child and IO pump is joined; daemon/session shutdown is fenced to the run's private namespace and observed process generations. Startup failure, cancellation and PID reuse cannot target a foreign process or erase unknown live state.
- A guard test rejects live/inherited/empty/symlink-escaping paths before effects and proves derived messaging sockets differ from the live namespace. State/operation counts at N and 2N defend bounded fanout.
- A real opt-in consumer runs successfully for Claude and Codex with a genuine steward plus three non-hub workers, public message/receipt paths, detach/restart, and verified cleanup. It emits exact-head/native-version/isolation/log evidence. Cheap tests stay in the ordinary suite.
- Atlas documents this as the shared live-fleet path; reader/artifact inventories, required recorded suite, mandatory close review and normal PR landing pass. No protected local-gate declaration or staging of operator slots is included.

## Plan

- [ ] TL reviews the durable design and state/recovery table before change-code; derive the estimate only after plan-quality.
- [ ] Implement confinement and owned lifecycle plus deterministic agent/CLI protocol with focused adversarial tests.
- [ ] Compose real provisioning/native runtime, then qualify the first shared-helper consumer and audit source-reader/artifact ownership.
- [ ] Document and run required checks, close and land. Resume #722 only after this issue is on main.

## Log

### 2026-10-20 14:50 PDT — native shared-harness plan ready for TL review

#722 is parked at its pushed checkpoint with claim kept. #744 claim/start-plan created this independent branch at fresh main; no #722 commits were imported. Two bounded read-only explorations verified native environment/agent reuse and real provisioning/public worker startup. The proposed helper uses real provision-workspace and recordless reboot, not extra hubs, fake workspace identities or setup markers. Durable plan records confinement/ownership state table, exact execution/cleanup seams, agent personalities, native qualification/first consumer and limits. Await TL review before change-code or implementation; no estimate derived yet.


### 2026-10-20


## Revisions

### 2026-10-20 — approved base plan; PQ-1 isolation correction pending

Verified submitted TLcf61df78 approved the base native plan and change-code. Full-flow plan-quality then found that directory-wide dead-wrapper collection at message listener startup can touch foreign residue in the shared per-UID parent even with disjoint addresses. No implementation started. The durable plan now proposes extending existing COUCH_ISOLATED_ROOT authority to an owned message parent, with actual-listener foreign-residue regression and compatibility checks. This narrow transport seam corrects the original no-production-change assumption; request TL approval before code, then rerun the remembered plan gate.

### 2026-10-20 14:47 PDT — dispatched native prerequisite for #722

Verified submitted TL2d5e80a3: park #722 with claim, commit/push and next step; take #744 from main on its own branch, plan first, land it, then resume #722 over landed main. Native Zellij topology approved; scope limited to #722/#740/#741 needs. The original broader Spec/Done when/Plan are retained below; the active contract above narrows agent coverage to Claude/Codex and makes the initial native helper qualification its first live consumer. Qoder is a future adapter, not a promised result now.

### Original Spec — historical

One pair test helper package for a disposable Couch fleet:
- private `COUCH_ISOLATED_ROOT`, store, identity, Pair data, HOME, XDG, Zellij socket/config/runtime and sdlc queue
  roots, under TestScratch when present (TMPDIR otherwise); refuses empty, live or inherited paths before starting;
- a real supervisor plus N throwaway worker slots in real disposable checkouts, launched with genuine caller
  identity (no synthesized bindings);
- agent fixtures for **each agent kind** (Claude and Codex, Qoder when available) and a deterministic fake agent;
- exact cleanup: joins every child and removes only the processes, sessions and sockets it created;
- a guard test proving it can never see the live Couch's socket, store or HOME.
Existing live tests migrate to it as they're next touched; new ones must use it.

### Original Done when — historical

- The helper exists with the guard test, and pair#741 (or the first live test to land next) uses it.
- `atlas/` documents it as the one way to write a live test.

### Original Plan — historical

- [ ]

### 2026-10-20 — TL3d5e6782 approves transport isolation correction

The PQ-1 scope correction is approved, conditional on unset-isolation path/sweep compatibility, refusal before effects for invalid/overlong paths, and actual-listener foreign-residue coverage. Rerun the remembered full-flow plan gate. No code has been written.
