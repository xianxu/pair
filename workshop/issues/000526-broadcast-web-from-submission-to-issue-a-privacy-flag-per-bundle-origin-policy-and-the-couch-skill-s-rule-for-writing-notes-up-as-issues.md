---
id: 000526
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours: 2.243
card_mirror: 'ff5cae15423961b289f9a13deb1adbfe8b133a02' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T23:04:21-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# broadcast-web: from submission to issue: a privacy flag per bundle (origin policy) and the couch skill's rule for writing notes up as issues

## Problem

When the operator asks an agent to "file an issue from these notes" with a DOM submission (#483), no rule explains how to externalize the evidence. A crop of private data, such as a customer's app or a logged-in page, can land in repository issue history.

## Spec

Operator requirements, 2026-10-12: issues hold evidence as text; only relevant crops go into the issue asset folder under ariadne#337. Recordings stay local (no S3). Privacy is gated rather than left to agent judgment.

1. Couch has a small operator-owned list mapping page origins to `shareable` or `private`, defaulting to `private`. For example, the operator may list `http://localhost:4321` as shareable. Each submission records `privacy: shareable|private` and the matched rule in `submission.json`. Agents read the policy; only the operator edits the list.
2. The Couch skill's "Reading a broadcast-web recording" section and the bundle's `how_to_read` require issue evidence as text: page URL, element and classes, box text, the request, and one line describing what the crop shows. Attach only relevant crops, only when the bundle says `shareable` AND the repository asset policy permits it (ariadne#337). For private origins, summarize instead of quoting box text. Name recordings as local evidence that may expire; the issue must remain understandable without them.
3. Check that typed input values are not quoted in box text for private origins; passwords are already excluded.

Coordinate with #525, which adds the page URL to the same manifest. Build on main rather than its unlanded branch.

## Done when

- Invalid origin policy does not refuse submission: bundle is private with `privacy_rule: "invalid policy file: <reason>"`, and a Couch notice names the file.

- A submission from an explicitly shareable origin records `shareable` and its matching rule; unlisted origins record `private`, as do explicit private rules.
- The Couch skill and `how_to_read` carry the text-first evidence rule, the two conditions for crop attachment, private-text summarization, and the local/expiring recording limitation.
- Tests pin the manifest privacy field, matched rule, private default, and exclusion of typed input values for private origins.


## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only; calibration is flagged stale, so provisional.

The issue/spec work uses 0.5 design + 0.2 implementation hours from v2. The pure policy module uses 1.0 design (halved for standard-library JSON/URL support, then ×0.2 for the resolved spec) + 0.6 implementation. Two existing Go seams (loader/capture) each use 0.2 design ×0.2 + 0.4 implementation; the DOM collector change uses the same bounded-refactor rates. Skill guidance uses 0.3 design ×0.2 + 0.4 implementation, docs 0.1 ×0.2 + 0.15, close review 0.1 ×0.2 + 0.4, and Chrome conformance 0 + 0.3. All implementation figures below are scaled to 40% per v3.1, familiarity 1.0; design buffer 15% for the detailed plan. Standard libraries and existing stateful fixtures avoid a new external integration. Scope includes tests, docs and one close review; #525 coordination does not implement that issue.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.5 impl=0.08
item: greenfield-go-module design=0.1 impl=0.24
item: smaller-go-module design=0.04 impl=0.16
item: smaller-go-module design=0.04 impl=0.16
item: cross-cutting-refactor design=0.04 impl=0.16
item: skill-or-dispatcher design=0.06 impl=0.16
item: atlas-docs design=0.02 impl=0.06
item: milestone-review design=0.02 impl=0.16
item: real-api-discovery design=0 impl=0.12
design-buffer: 0.15
total: 2.243
```

## Plan

- [ ] Implement and verify [the durable privacy plan](../plans/000526-submission-privacy-plan.md).
- [ ] Close through the SDLC review, open the PR, and report to hub; merge only on SHIP.

## Log

### 2026-10-12

- Published the operator contract to main at `9e9a182ec00dad07ea31dc57b7be712a7517522e`; started an issue branch from main.
- Inspected submission model/build/store, Cast preparation, recorder capture fences, DOM collector and shipped Couch skill. Input `.value` is not read, but textarea/editable descendants need regression coverage.
- Drafted `workshop/plans/000526-submission-privacy-plan.md`; policy matching is pure (ARCH-PURE), capture identity remains the authority (ARCH-SECURE), and delivery retains its existing reducer (ARCH-ORDER). No implementation or tests run yet.

## Revisions

### 2026-10-12 — hub coordination with #525

Hub confirmed top-level `privacy` and `privacy_rule`, an operator-edited local exact-origin list, default private, and no edit CLI. #525 owns the capture-bound source URL; whichever issue lands second integrates the fields, without building on the other's unlanded branch. The draft plan proposes excluding editable values for all origins to keep the protection independent of publication permission.

### 2026-10-13 — approved plan; invalid policy must not strand submission

Hub approved commit `5738858e62f3ab31716f641a7f1af8de87db4fa1` with the operator's no-limbo correction. This supersedes decision 3, the partial-JSON failure wording, and task 2's refusal expectation: missing policy remains private; unreadable, oversized or invalid policy makes every origin private, preserves submission/delivery, records `privacy_rule: "invalid policy file: <reason>"`, and emits a Couch notice naming the policy file. Reasons are bounded safe categories, never raw policy content. Valid matched rules remain objects, unmatched defaults null; the typed rule representation must support the explicit invalid-policy string variant and validate that it occurs only with private. Tests must prove successful preparation/delivery and the notice on these failure paths. No estimate until the plan-quality gate passes.

### 2026-10-13 — plan-quality revision

PQ-1 addressed with named risky-function strategies instead of prose case inventories; PQ-2 corrected operator-owned retention versus admission capacity. Full plan-quality rerun required before implementation.

### 2026-10-13 — implementation checkpoint

- Implemented pure bounded origin rules and manifest decision validation, operator-only local file loading with private fallback/notice, capture-fenced origin matching, text-first guidance, and editable-value exclusion. New policy parser and manifest tests were red before implementation; targeted submission/webcast/recorder/couchcmd/couchtty privacy, origin, guide, capture and box-text tests pass. Policy fuzz: 127446 executions in 3 seconds, passed.
- Skill baseline identified reliance on agent judgment; explicit shipped contract tests were red then green. Fresh-context five-scenario application confirmed private/legacy/invalid text-only behavior and the conjunction of shareability plus repository asset permission.
- Full `make test` attempted: stopped at known #514 (`capture took 20s over a 6s budget`); `/tmp/pair-526-full-test.log`. Running `make -k test` to collect the remaining checks and asked hub about the known baseline failure. Live Chrome remains unrun until full-suite attempt completes.
- Estimate scope clarification: issue-spec includes incurred authoring/design, policy module includes the null/object/string codec, and capture integration includes second-lander #525 reconciliation. Rates remain provisional per estimate-quality INFO.

- Live Chrome `TestLiveSubmissionCrop` passed (actual typed-input fixture); targeted race run across submission/webcast/recorder/couchcmd passed; `make build` passed. Full keep-going run exposed the exhaustive production-source inventory: registered the two new policy files beside existing Couch/submission sources; targeted artifactpath coverage now passes. Full rerun required after that correction.
- Hub authorized treating only known #514 timing as non-blocking and instructed close/land after other verification; next assignment is #525 after #526 lands.

- The keep-going full suite completed: its sole failure was the now-corrected production-source inventory; every other package/check passed (including Couch core in 232.967s). `/tmp/pair-526-full-keepgoing.log`. A final full rerun is running at corrected code HEAD `828e9471b`; no further production changes pending.
