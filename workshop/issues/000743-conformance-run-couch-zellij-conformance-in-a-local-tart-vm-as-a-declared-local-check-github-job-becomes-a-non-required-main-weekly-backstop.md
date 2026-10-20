---
id: 000743
status: working
deps: [ariadne#509]
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours: 5.06
collaboration-mode: collaboration
card_mirror: '128120ffd724f84fe7916c022ed10ed1ae096acf' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T14:15:16-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
---

# conformance: run couch-zellij-conformance in a local Tart VM as a declared local check; GitHub job becomes a non-required main/weekly backstop

## Problem

`couch-zellij-conformance` (real Zellij 0.45.1, nvim and PTYs on a clean macOS runner) is a required PR check for
Couch, terminal, launcher and Zellij paths. On 10-20 GitHub's macOS arm64 capacity cancelled it three times before
any runner started ("not acquired by Runner of type hosted"), gating pair landings (#726, #732, #735) on GitHub's
capacity instead of on our tests. The operator's rule is local-first verification.

## Spec

- **Run it in a Tart VM locally** (pair's `.tart/Makefile` already boots `pair-test` headless and clones the repo in):
  a non-interactive `make conformance-vm` boots the VM, a `.tart/vm-hooks.d/` hook installs the pinned Zellij 0.45.1
  (checksum-verified, as the workflow does) and nvim, runs `make -f Makefile.local test-native-terminal-ci` and
  `test-couch-zellij-live`, copies the logs out (including Zellij's), stops the VM, and exits with the tests' status.
  Refresh the `pair-test` image first (last used four months ago).
- **Declare it as a local check** `conformance` in `.sdlc/local-checks.json`, required only when the diff touches the
  workflow's path list (one source for the list, shared with the workflow), heavy weight (a VM plus the tests), recorded
  on the PR head like `suite`.
- **GitHub job demoted (operator-approved 10-20):** keep it on main pushes and weekly as a clean-machine backstop,
  but no longer required to merge.
- Guard: the VM run can never touch the live Couch: no shared Zellij socket, HOME or Couch store with the host.

## Done when

- `make conformance-vm` passes on current main and fails on a known-bad fixture; its isolation is asserted (no host
  socket, HOME or store visible in the VM).
- `sdlc local-check run --name conformance` records on a PR that touches the paths, and is skipped on one that doesn't.
- Local-gates change: the operator verifies and applies `local-gates-verified` (never the worker).

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only. Calibration is marked stale by `sdlc estimate-source`; values are provisional.*

Primitive mapping, in block order: issue/spec; pure request/transition/claim module; Tart/guest transport and image provisioning; COW snapshot/independent Git adapter; public dispatcher; strict native evidence integration; nightly command/declaration; atlas/docs; one real Tart discovery budget; M1/M2/M3/close reviews. M1's two greenfield concerns use upper implementation anchors (0.8h each before scaling); Tart integration uses 1.5h. Smaller mirrored pieces use 0.4–0.5h anchors. Each review uses 0.3h. Implementation values are scaled once by v3.1's 0.40. Familiarity 1.5 reflects a bounded but unfamiliar VM/guest runtime; no separate cross-repo multiplier because #509/#737/#744 code is outside this issue.

Library check: Tart/guest-agent supplies virtualization and transport; native filesystem/Git supplies COW/snapshot primitives. Halve those two design anchors before the thorough-plan ×0.2 discount. Other design anchors receive ×0.2 for the accepted plan. Design subtotal 0.90h, +15% = 1.035h; scaled implementation subtotal 2.68h ×1.5 = 4.02h; total 5.055h, recorded as 5.06h. Seven-night elapsed observation and external dependency/CI waits are outside focused ship-time. Working range roughly 3.5–7.5h given the stale calibration and native runtime uncertainty.

```estimate
model: estimate-logic-v3.1
familiarity: 1.5
item: issue-spec design=0.20 impl=0.08
item: greenfield-go-module design=0.20 impl=0.32
item: api-integration design=0.20 impl=0.60
item: greenfield-go-module design=0.10 impl=0.32
item: skill-or-dispatcher design=0.10 impl=0.16
item: smaller-go-module design=0.04 impl=0.20
item: smaller-go-module design=0.04 impl=0.20
item: atlas-docs design=0.02 impl=0.08
item: real-api-discovery design=0.00 impl=0.24
item: milestone-review design=0.00 impl=0.12
item: milestone-review design=0.00 impl=0.12
item: milestone-review design=0.00 impl=0.12
item: milestone-review design=0.00 impl=0.12
design-buffer: 0.15
total: 5.06
```

## Plan

Full-flow durable design: [local VM conformance plan](../plans/000743-local-vm-conformance-plan.md). Awaiting hub approval before implementation.

- [ ] M1 — Refreshed image and generic isolated runner.
- [ ] M2 — Strict Pair conformance and runnable named check.
- [ ] M3 — Nightly burn-in handoff and initial infrastructure landing.

## Log

### 2026-10-20

- Claimed in pair:3 and entered planning. Read-only audit confirms Tart 2.32.1; stopped pair-test last accessed 2026-06-09, four CPUs/8 GiB, 50 GiB virtual disk. No VM was started or modified during planning.
- ARCH-DRY: interactive `.tart` is inherited, ignored dev infrastructure with best-effort hooks/final SSH status; the CI runner must have a strict command/cleanup boundary. The generic runner owns two fixed machine positions, not a second scheduling queue.
- SDLC lacks train-scoped/path-conditional checks and every local check currently takes a suite slot. Filed ariadne#509 unclaimed; it owns the post-suite train/single-landing hook, conditional any-car union and external resource separation. Do not emulate that contract in pair.
- Current native targets exercise real terminal/nvim/Zellij machinery but use synthetic agents; #744 owns genuine disposable-Couch/Claude/Codex cases and is not assumed available. Required-case JSON validation must also cover the lifecycle population, where plain Go test currently permits skips.
- Hub approved the initial rollout split in submitted receipt e41577c8-f467-40e7-baf7-c5d5cbea53da: land image/runner/runnable nightly first, keep landing activation off and GitHub PR conformance required through seven clean nights, then promote separately after ariadne#509. Durable plan records failure/uncertainty, resource lifetime, isolation, bounded residue and exact verification; no implementation has begun.
- Filed #745 unclaimed for the explicit promotion/cutover and seven-night evidence; its dependencies are #743 and ariadne#509. #743 depends on ariadne#509 for supported resource-aware named-check acceptance, while M1 image/runner work is independent.
- One advisory fresh-context plan review found four concrete gaps, all accepted: persist startup uncertainty before spawning so parent death cannot admit a second VM before running-state publication; admit provision/qualification through the ad-hoc lane; reserve bounded nightly evidence separately from ordinary-run pruning; and bootstrap the new named local check after its declaration reaches main (direct real-VM command qualifies before landing). No duplicate plan-approval review was run; change-code remains the mandatory plan-quality owner.
- Plan-quality round 1: one Important finding PQ-1 (`test-strategy-contract`); architecture otherwise executable. Added a canonical named risky-function strategy table and compressed M1/M2/M3 test inventories to references. Existing #721 lesson covers this class. No implementation or machine/image changes while the gate is unresolved.
- Plan-quality subsequently CLEAN, PQ-1 addressed; estimate-quality INFO and `change-code` admitted implementation. The 5.06h estimate includes stateful fake work and native qualification/repair inside the integration/snapshot primitives; `real-api-discovery` is only the external API discovery slice. Upper native correctness/qualification anchors remain an uncertainty, not an allowance to omit verification.
- Image approval correction: the 14:26 dispatch explicitly called refresh/install operator-approved with prior notice, so a notice was submitted and Tart pull began. The hub's 14:33 hold superseded that: interrupted our exact Tart child (PID 60664), which exited nonzero after its last 77% report with decompression interrupted. No VM was cloned or booted; existing local/cached images remained stopped. Hold all further image/provision/install actions pending the operator's explicit decision. Submitted hold report 31f2918d-e41a-4cf2-bbea-68c84828c0d5; do not resume or delete partial cache as an implicit retry.
- M1 partial: request parser and additive image codec passed delegated spec/quality checks. Runner transition and additive launch-claim tests passed; launch-claim fuzz executed 278,607 inputs in 3s. Kernel position IO and guest bootstrap are in progress, entirely in portable fixtures; no real VM execution is claimed.

- M1 checkpoint 14:56: kernel lane positions, additive claim publication and embedded private guest bootstrap now pass controlled subprocess/fixture tests, with prior spec/quality reviews complete. Snapshot targeted tests passed (1.588s), and the full vmcmd package passed race detection (3.014s). No native Tart action was performed. These are partial component checks, not M1 or real-VM qualification.
- Snapshot fresh review found two accepted material gaps before runtime/CLI integration: (1) current tracked-content budgets omit HEAD Git objects, so a large committed file deleted locally can bypass the cap; bound exact HEAD objects and reserve metadata capacity before a pinned transfer, report metadata bytes, and add the deleted-blob regression; (2) Git's surviving descendants are killed only on cancellation, so parent exit can leave work behind; add barrier-controlled success/error descendant fixtures and join/clean the owned process group before return. Snapshot code is checkpointed incomplete and is not wired into a public command. These fixes are the NEXT ACTION before runtime orchestration. Exhaustive artifact-source guard passed after adding all new sources including the embedded script (1.483s). The affected-reader guard passed all eight tests (23.907s); guest/position/snapshot reader sites access only private fixtures or literal invalid paths and are inventoried accordingly.
- The approved tracked-only EXDEV/unsupported-filesystem fallback and embedded guest-script placement are captured in the durable plan's 14:53 revision. Pair issue filing/main publication was held for train 197, then reopened by authenticated hub receipt 40831bf5-2c90-4de8-bd36-19f9ba4437ee after landing. Native image approval is still pending.

- Resumed snapshot corrections: deleted/truncated 2 MiB HEAD blobs reproduced the omitted-metadata budget failure. The snapshot now enumerates the pinned commit/tree/blob closure, sizes it before mutation, reserves pack/index/scaffolding capacity, caps the pack stream, and reports actual independent metadata bytes. Added shallow/history/source-removal and exact transfer-cap checks. Fresh review identified replacement-ref interpretation; a real replaced-tree regression failed before disabling replacement interpretation consistently (also refuse lazy promisor fetching).
- Controlled Git descendants reproduced survival after both success and error exits. Every exit now signals the private group and confirms disappearance under a one-second cleanup cap; unknown/expired confirmation is an operation failure. A held confirmation probe prevents claiming completion on signal delivery alone. Snapshot subset passed (13.102s); final package verification follows after concurrent Run implementation. Fresh snapshot re-review CLEAN; reviewer could not independently rerun while Run TDD temporarily left the package uncompilable.
- Run orchestration and stateful fake now follow Advance with cancellation/error barriers and a shared ten-second detached cleanup budget. Targeted Run race tests passed (11.231s); a premature-release mutant failed before restoring the passing implementation. Native adapter and CLI composition remain outstanding. Hub approved the timeout envelope in authenticated receipt d13df378-0a0a-4040-98a3-c20e29d43dbc; durable-plan revision owns the change. Native image hold remains unchanged.

- Snapshot/Run checkpoint verification: full vmcmd race run passed (16.064s), artifact source guard passed (2.585s), affected-reader guard passed all eight tests (121.855s). Fresh Run spec review caught a fake-only gap: cleanup expiry retained the local lease instead of only durable occupancy. Added lease-closed assertion first (red), corrected unconditional local-handle closure separately from destructive cleanup, then TestRun race passed (11.264s). Native driver and host ownership composition are next; no public route/real VM pass is claimed.

## Revisions

- 2026-10-20 13:30 (TL, from the operator's decisions in ops side doc `2026-10-20-19-conformance-structure.md`):
  - **Tier 3 gate:** the VM conformance runs **once per landing train** when any car touches the paths, not per
    close; it starts as a **nightly** run (off-hours) and is promoted to blocking after about a week clean. Any flake
    is fixed at its root, never retried.
  - **One VM, via a machine-wide lock:** macOS allows two VMs; tier 3 takes one through a lock like the suite lock,
    leaving the other for ad-hoc runs (which take the lock's other position).
  - **Every agent kind:** agent-facing cases run against Claude and Codex (Qoder when it has credits).
  - **Pair first** (operator); ariadne and parley later.
  - Uses the shared disposable-Couch harness (pair#744) for its live cases.
- 2026-10-20 14:00 (TL, operator decision in the same side doc): **the VM is machine-wide and Couch-provided**, not
  pair's per-repo `.tart/` setup. Build first: one provisioned image (pinned Zellij, nvim, Go, agent CLIs; refreshed
  on purpose) and one small runner (e.g. `pair vm run --checkout PATH -- <command>`: APFS-clone the checkout in, run
  with a clean HOME, copy results and logs out, stop) that owns the two-VM lock (one position for the landing train,
  one for ad-hoc). Any repo can use it; pair's conformance is its first user. The per-repo `.tart/` stays for
  interactive dev.
- 2026-10-20 14:15 (TL, operator agreed): no second queue. The landing train stays the only queue; the VM is a
  machine-wide resource pool (like the suite slots) that a train acquires at its tier-3 step, **after** its suite
  passes and only if a car touches the conformance paths. One VM position for trains and the nightly burn-in, one for
  ad-hoc runs. The machine plan (ariadne#475) counts VM positions once a second repo uses the VM.
- 2026-10-20 14:19 (TL, authenticated rollout approval): initial #743 acceptance is the refreshed shared image, generic isolated runner, strict runnable local conformance and nightly handoff with landing activation **off**. Keep required GitHub PR conformance during seven clean nights. A separate promotion issue owns conditional local landing activation (after ariadne#509) and GitHub PR-trigger removal; retain main/weekly backstop there. #744 is unowned, so do not assume or duplicate its genuine fleet/agent harness. This phases the original Done-when local-gate/path-skipping and GitHub-demotion requirements into promotion without weakening verification meanwhile.
- 2026-10-20 14:26 (TL, authenticated full-flow approval): plan approved with the scheduler correction recorded in the durable plan's Revisions. No launchd/cron job: M3 delivers the nightly command plus the task declaration for ops's `scripts/scheduled/`, ready for #737's one scheduling mechanism. Hub/operator runs and records nights by hand until #737 lands. Operator approves image refresh and machine installs; tell the hub immediately before doing them. All other approved isolation, two-lane resource, nightly evidence and promotion boundaries stand.
- 2026-10-20 14:33 (TL hold, authenticated receipt 2e85e5fa-caec-45f5-a905-7d7455af0f1f): do not run/resume Tart pull pending explicit operator approval; hub is asking. Continue runner code and fake-VM tests. This supersedes the prior image-approval wording; native image qualification remains outstanding.
