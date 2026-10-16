---
id: 000586
status: working
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: '4be64b089094a743b85489a37e34befc5368248e' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T19:17:48-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# Diagnose intermittent review restoration observation mismatch

## Problem

`tests/review-observation-test.sh` intermittently refuses its repeated branch restoration during a full suite, with insufficient evidence to distinguish a stale observation, a timed-out observation, and an actual branch mismatch. This blocks unrelated #571 verification; the adapter changes none of the review observation/controller/test code.

## Spec

Reproduce and diagnose the first restore identity check without weakening authorization. The test's `resolve()` forwards the readiness JSON without asserting its status. The readiness CLI can return invalid status with exit zero, and identity resolution has a shared two-second Git budget. Under load an incomplete wanted observation may reach the controller, whose fresh observation then reports a generic mismatch. This is an unproven hypothesis: the failing log retains neither identity. Capture both wanted and observed status/identity/diagnostic, and distinguish malformed/incomplete evidence from genuine changed-branch refusal. No sleep or larger timeout as a substitute for root cause.

## Done when

- A deterministic regression identifies the failing observation boundary and exercises incomplete/stale input.
- The test validates resolver status before forwarding identity and retains actionable failure evidence.
- Real branch changes still refuse restoration; repeated isolated and loaded suite checks pass with measured results.

## Plan

Proposed quick-flow investigation and fix, pending hub approval. Keep #586 separate from #591 until retained evidence demonstrates a shared cause; continue #589, #591 and #590 in the agreed order after this item. Expected small boundary change; if demonstrated scope leaves the shell, revise and use a durable plan before implementation.

- [ ] Make the fixture fail diagnostically at its observation boundary: validate resolver exit, JSON shape, resolved status and complete identity before forwarding; preserve wanted and fresh status/repo/branch/head/file/diagnostic, command phase and completion ordering on refusal. Reuse the existing isolated fixture wrapper; bounded metadata only, no copied snapshots (ARCH-DRY/CONSTRAINTS).
- [ ] Establish deterministic RED with controlled invalid/incomplete wanted and fresh observations at that wrapper/controller seam. Preserve genuine changed-branch refusal and a positive same-identity restoration. Require actionable evidence and no activation on refused observations; no sleeps or altered assertions (ARCH-PURE/ORDER/SECURE).
- [ ] Reproduce under the suite's parallel load while changing one variable at a time. Record both observations before selecting the fix. In particular, distinguish the fixture forwarding invalid exit-zero JSON from actual branch movement and the resolver's shared two-second Git deadline. Append a Revision with demonstrated cause and the minimal owning-boundary change; ask the hub if choosing a new fixture time authority or broader production scope is required. Do not turn a timeout into successful identity, retry until green, or enlarge any budget (ARCH-PURPOSE/MOCK).
- [ ] Implement the demonstrated fix with regression evidence, preserving production authorization and bounded process ownership. Use event-driven completion or an injected fixture clock if scheduling is the cause. Temporary evidence dies with the existing fixture owner; dump relevant diagnostics before cleanup (ARCH-FUNERAL). Run targeted/loaded checks, make -k build, required recorded suite, atlas update as applicable, then mandatory close review and SHIP landing.

## Log

### 2026-10-15 — first observed during #571 unchanged rerun

- Exact tested head `8bd4baa138c63cda357c3a447ddb2cba294f1df7`; run `119840be5020067f99a3413fd951e608`.
- Log `/Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/119840be5020067f99a3413fd951e608.log`, SHA256 prefix `58ed1e657115`; suite stopped at test-review after 4m33s, before Go packages. A keep-going run was started to complete remaining checks.
- Python line55 in the repeated checkout/request loop: `AssertionError: {'error': 'branch changed during restoration; invoke Alt+C again', 'ok': False}`.
- Read-only source audit locates the first gate at `nvim/review/restore_controller.lua:137-139`, before activation/confirmation. A controller invalid observation normally preserves its own diagnostic; the generic text is consistent with an incomplete wanted observation but does not prove it.
- Unchanged isolated `bash tests/review-observation-test.sh` passed exit0 on the same code (subsequent branch changes are issue documents only). Earlier full suite run `b2dd3d164d5caad0fd4f624661697ade` passed this test and failed only the independent #585 broadcast test.
- Filed unassigned as independent verification followup; do not fold a review-controller fix into #571.

### 2026-10-15 — claimed after #585 landed

#585 landed via PR #357 at git-verified merge27a12fec10e7df7df86e5a5fb7a4f2a2f655b199; hub outcome receipt5a62e02b-0d83-46dc-bdaf-8549cb39b517 is submitted. Started #586 from main archive6f0e8c656bcfebc2051e85a8d0eb1c8e7366825b; claim and start-plan succeeded. No implementation changes or new test run for #586 yet.

Read-only audit: Python resolve() decodes check_output without checking status; RunReadinessCLI --resolve returns exit0 for a successfully encoded invalid observation. resolveIdentityWithSnapshot shares a real two-second context across root/index/branch/HEAD/history/integrity operations. C:request checks only wanted's table/repo shape before resolving again; its generic changed-branch refusal can compare a partial wanted identity against a fresh resolved identity. The observed historical generic message is consistent with that ordering, but neither original observation was retained, so it remains a hypothesis. Both #585 source suites passed the unchanged observation fixture; those green samples do not refute the recorded intermittent failure.

Alternatives: immediately tightening only the wanted-status check improves evidence but does not establish or eliminate the scheduling flake; replaying a fake resolved identity would remove the production Git deadline but risks bypassing the behavior under investigation. Recommend diagnostics first, deterministic invalid/stale refusal cases, then a measured Revision at the actual time/observation boundary. Keep authorization fail-closed and use the fixture's actual state rather than a larger wall-clock budget.
