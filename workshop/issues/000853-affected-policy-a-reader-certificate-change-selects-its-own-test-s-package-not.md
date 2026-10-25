---
id: 000853
status: blocked
deps: [ariadne#598]
github_issue:
created: 2026-10-23
updated: 2026-10-24
estimate_hours:
card_mirror: '8aad738d64b9ae14af32c09deb8c6141f87dc5e0' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T10:26:38-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# Affected policy: a reader-certificate change selects its own test's package, not the full suite

## Problem

## Spec

**pair's affected-test policy falls back to the full suite whenever a reader certificate changes (found on the first
day of affected boarding, 10-23).** Several changes today ran a full suite under `--pre-close` only because they
touched a file under `tests/affected-readers/…` (the per-test reader certificates): pair#825 (its
`request_lifetime_test.go.json`), pair#679 and pair#739. Any test fix that changes what a test reads updates its
certificate, so most test changes get the full suite, which defeats the point of boarding on the affected check.

Fix: in pair's affected policy (`.sdlc/affected.json` and its selector), map a reader-certificate change to the
package (or test group) whose certificate it is, plus whatever that certificate's reads select, instead of selecting
everything. Keep the full-suite fallback for changes to the policy itself, the selector, or shared harness code.

## Done when

- A change to one test's reader certificate selects that test's package (and the reads it declares), not the full
  suite (test with a real certificate change, e.g. pair#825's).
- Changes to the policy, selector or shared harness still select the full suite (test).

## Plan

- [ ]

## Log


- 2026-10-25: unclaimed: TL-directed handoff to pair:6 (delivery 8574f573-f5e1-40b6-b447-be6ab09aaa92). Preserve plan-wait commit 2522724fa851d612fb4222da91b60b0c2d35a5fe and existing Log; no implementation or review was started. ariadne:5 reports ariadne#598 M1 landed as PR435 merge 2001041f06cd576094266cc4b562c09cec7d0858: declared input ownership (owners/full excepts/sidecars with reads, read from main) is available after rebuilt ariadne:0 and weave refresh. Next owner verifies current capability and dependency, resumes planning for reader certificates plus cmd/couch test-only changes, sends plan to hub before change-code. pair:1 remains on #880 waiting for #857 landing.
### 2026-10-23


### 2026-10-24 — provider prerequisite; paused before implementation

Claimed and entered planning on an independent branch from main. The blocker is structural: `.sdlc/affected.json` has full overrides `tests/**` and `cmd/couch/**`; removing the former leaves certificates unclassified. Ariadne `affected.Plan` unions native Graph.Owners with group paths, but Group.Go only triggers groups and cannot add packages. Current certificates contain disposition/hash, not declared reads. Reverse imports do not express runtime executable/fixture reads. ARCH-DRY: keep selection in the provider, rather than duplicating it in Pair or creating hundreds of test groups.

Ops approved the provider split (delivery e3fddf94-32aa-4044-99a5-c68e0cd84276). Filed **ariadne#598**, unclaimed, `workshop/issues/000598-affected-declared-input-ownership.md`; ariadne:5 takes it after #592 lands. Contract: declared non-Go path ownership adds Go packages through the existing union/closure, certificates record audited reads, malformed/unknown/deleted mappings fail conservatively, and policy/selector/shared harness fallbacks remain full. Set this issue's deps accordingly. No production or test code changed; no tests or passing measurement claimed.

**NEXT ACTION:** after ariadne#598 lands, resume this branch, integrate main and the provider, audit all certificate readers, finalize the consumer plan and obtain TL approval before change-code. Include #825's request_lifetime certificate and cmd/couch main_test.go/editorial_acceptance_test.go exact selections, unknown/deleted/renamed cases, explicit full precedence and actual native fixture reads. Local-gates configuration needs operator verification. In the meantime TL explicitly directed pair:1 to work #844 on its own branch from main; no stacking.

## Revisions

- 2026-10-24: planning found a missing provider capability. At ops:0 direction, block on ariadne#598 and defer the consumer design until its contract lands; #844 proceeds independently meanwhile. Original Done-when remains in force, plus the TL's cmd/couch test-file scope.
