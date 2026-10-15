---
id: 000586
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: 'baf91ac3de1b4778dfd644c76c03ffe53464fdda' # card fields mirrored from issue-cards; edit via sdlc
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

## Log

### 2026-10-15 — first observed during #571 unchanged rerun

- Exact tested head `8bd4baa138c63cda357c3a447ddb2cba294f1df7`; run `119840be5020067f99a3413fd951e608`.
- Log `/Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/119840be5020067f99a3413fd951e608.log`, SHA256 prefix `58ed1e657115`; suite stopped at test-review after 4m33s, before Go packages. A keep-going run was started to complete remaining checks.
- Python line55 in the repeated checkout/request loop: `AssertionError: {'error': 'branch changed during restoration; invoke Alt+C again', 'ok': False}`.
- Read-only source audit locates the first gate at `nvim/review/restore_controller.lua:137-139`, before activation/confirmation. A controller invalid observation normally preserves its own diagnostic; the generic text is consistent with an incomplete wanted observation but does not prove it.
- Unchanged isolated `bash tests/review-observation-test.sh` passed exit0 on the same code (subsequent branch changes are issue documents only). Earlier full suite run `b2dd3d164d5caad0fd4f624661697ade` passed this test and failed only the independent #585 broadcast test.
- Filed unassigned as independent verification followup; do not fold a review-controller fix into #571.
