---
id: 000594
status: codecomplete
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: 'd8b4fb957b386892ea07886502e9a6d57a939e3e' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T18:58:23-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "57af933c", done: "664a5825"}
actual_hours: 0.51
---

# Couch: deliveries to a newly started slot (parley.nvim:4) never land, receipts 'indeterminate' (unknown field AcceptedAt)

## Problem

2026-10-15: the operator started a new slot, parley.nvim:4 (Claude Code, ~17:12). Every `couch --send-to parley.nvim:4`
from the hub (ops:0) is accepted ("queued", ids 7d561939…, 115b6364…, 12c756aa…) but never appears in the pane, while
the operator's own typing works and deliveries to every other slot arrive. `couch --message-status ID` prints
"indeterminate" and, with `--json`, fails: `json: unknown field "AcceptedAt"`.

The hub's `couch` CLI is pair:0's staged build (6d65c4c4, main + #573 + #544, built 16:28); the running supervisor and
the new slot's wrapper may be a different build. So two problems:
1. A delivery to a live slot silently never lands, with no failure reported to the sender.
2. CLI/supervisor version skew breaks receipt decoding (strict JSON refusing a newer field), hiding the status.

## Spec

Omit unset AcceptedAt from wrapper receipts and omit that broker-only field from wrapper commit requests, preserving nonzero admission time in the broker's sender receipts. Keep strict decoding; incompatible wrapper replies identify the slot and advise restarting Couch. Never retry commit after a failed reply. Report a terminal protocol failure with honest uncertainty if input may have landed; broker release must not claim operator keystrokes. No general compatibility framework.

## Done when

- Literal pre-AcceptedAt request and reply schemas reproduce both skews before repair and decode successfully after it; broker sender receipts retain their nonzero original timestamp.
- A wrapper reply decode failure reaches the sender as a terminal receipt with the slot and restart advice; commit occurs once. Possible input effects remain indeterminate, never claimed absent. Broker release before paste is cancelled, after paste uncertain, and neither blames operator input.
- Focused tests and race regressions, build, required recorded full suite and SHIP close review pass. No production slot mutations are needed to land.

Production acceptance remains pending with the operator/hub after landing: restart Couch onto the combined stage, send one new hub message to the newly started slot and inspect its submitted receipt. TL c93fb492/1d8c54db directs tonight's landing before that restart; this deferred check is not claimed complete here.

## Plan

- [x] Omit unset AcceptedAt; prove new wrapper replies decode with the old schema.
- [x] Name wrapper reply incompatibility and broker release honestly; never retry commit or hide possible effects.
- [x] Run focused regressions, build, required recorded suite and SHIP review; land and report.

## Log

### 2026-10-15
- 2026-10-15: closed — BR-1: reconciled active Spec and both original Done when clauses with approved uncertainty and operator-owned post-landing restart, preserving superseded promises in Revisions. Documentation-only correction; production remains d1a8bbbcef7d with recorded suite PASS 9m37s, log a03eb6ac7b6b; targeted race tests, build and all CI PASS. Reviewer independently passed both affected packages with no executable defects. Live production acceptance remains explicitly pending after tonight landing per TL c93fb492/1d8c54db. No waivers or production slot effects.; review verdict: SHIP

- Read-only cause confirmed: running supervisor e1311121 (pid 40081, 14:57) predates AcceptedAt; wrapper 10627 was started 17:12 from a newer build. All five exact IDs (7d561939-1070-4287-89e9-b939084afe25, 115b6364-6b4a-4d62-91f2-f6837896fac3, 12c756aa-4756-4632-a14a-ae533e48018a, ed7175bb-c9dd-4bf9-8ce9-d829fb2dfed4, b408e185-aa67-46b5-82f5-f152168ae991) are retained in the wrapper as cancelled with zero AcceptedAt and the misleading operator-input reason. New wrapper commit replies emit that unknown zero time; supervisor decode fails, then its release marks interrupted. Reported before changes, receipt 64c033f7. No live input or lifecycle effects.
- TL approved the narrow plan (b5ab4618). Literal old-schema wrapper commit/status tests went RED on unknown AcceptedAt; protocol-error and broker-release tests also went RED on missing diagnostics/operator blame. Minimal fixes pass focused race tests (couchmessage 2.221s, wrapcmd 1.447s); full affected packages passed (6.814s, 53.602s), and make build passed. Status scans also preserve the exact decode diagnostic instead of a generic unanswered count. No commit retry; uncertain effects remain indeterminate.
- Reverse direction needs a decision before landing: new broker assigns nonzero AcceptedAt and old wrappers strictly decode requests without that field. Proposed omitting the broker-only timestamp from wrapper commit requests while keeping broker evidence; sent hub receipt 9b5ba083. Current patch only omits zero values as already approved. No unapproved reverse-wire change.

- 2026-10-15: Final required recorded suite PASS at d1a8bbbcef7d659d59eb45927d0d8164f62fa134, exit 0 in 9m37s, log SHA-256 a03eb6ac7b6b; core 323.211s, couchmessage 7.301s and couchcmd 53.006s passed. Final targeted race matrix passed (couchmessage 1.374s, wrapcmd 2.077s), including broker-to-wrapper CLI receipt timestamp preservation. Final make build PASS; all CI checks including real live conformance PASS. Log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/d1a8bbbcef7d659d59eb45927d0d8164f62fa134/suite.log. Production slot acceptance remains the operator's combined restart, as directed; it does not gate this tonight landing. No gate waivers or production slot effects. Close review follows.

## Revisions

### 2026-10-15 — measured cause and approved narrow fix

Read-only investigation proved the failure is older supervisor e1311121 decoding newer wrapper receipts, not the staged CLI decoding a newer supervisor. New Message.AcceptedAt used omitempty, which still emits a zero time.Time. All five wrapper-retained receipts were cancelled before submission, with zero AcceptedAt; broker release reused the operator-input event. The exact IDs were recovered from the hub tty capture; only read-only status requests were issued to wrapper 10627. TL approved omitzero, a literal old-schema decoder regression, slot-specific restart advice on reply decode failure, and truthful broker-release cancellation. A failed reply never retries commit; possible input effects retain uncertainty. No general compatibility framework or production lifecycle action. ARCH-ORDER preserves uncertainty; ARCH-DRY reuses transport and delivery reducer.

### 2026-10-15 — approved reverse-direction compatibility

TL approved c93fb492/1d8c54db: omit AcceptedAt from wrapper commit wire messages only; the broker retains its nonzero original and sender receipt. Literal old-request decoder regression went RED with terminal indeterminate/unknown AcceptedAt before the copy omission. The first RED attempt waited for Submitted rather than any terminal result and was interrupted as our own fixture; corrected the test to await terminal and assert Submitted, exposing the actual decode failure immediately. No production processes were signalled. No negotiation, capability framework or relaxed decoder introduced. Wrapper-supplied nonzero values still serialize, and zero timestamps omit in every receipt. The live check waits for the operator's combined restart; no slot mutations here.

### 2026-10-15 — BR-1 acceptance criteria reconciliation

Round 1 found both original Done when clauses stale: they promised an unconditionally definite delivery outcome and completed live acceptance, although TL b5ab4618 approved preserving uncertain effects and c93fb492/1d8c54db directed landing before tomorrow's live restart. Reconcile the active Spec and every Done when clause with those decisions; the old promises are recorded here as superseded, not silently erased. Automated acceptance is complete; production acceptance is explicitly pending with the operator/hub and is not a landing gate. No code changed. The reviewer independently passed both affected package suites and found no executable defect. ARCH-PURPOSE: the current acceptance contract must describe the approved deliverable, not merely leave its correction in the Log.
