---
id: 000837
status: working
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'a614d2ddaf93746908b1476c705629b70daf5a42' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T12:08:30-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "d0534500", done: "cb2fcab9"}
---

# Fence slash qualification journal against system deliveries

## Problem

The #162 disposable slash public check freezes an accepted-input journal baseline after current wrapper Idle, then expects exactly one appended /model entry. A delayed legitimate Couch restart notice can arrive after that baseline and make the later prose assertion fail even when /model was accepted exactly once and prose remained a draft. This is measured startup-notice interference, not evidence prose submitted.

## Spec

Harden cmd/couch/slash_keys_acceptance_test.go against asynchronous system deliveries. Select a causal baseline or identify legitimate system deliveries separately without hiding duplicate slash submissions, ordinary prose submission or journal mutation. Reuse the existing owned disposable fleet and current proof; preserve runtime/output/cleanup bounds. No production readiness/deadline change or blind retry. The TL suggested ignoring or waiting out system deliveries before baseline; settle the exact oracle with adversarial regression evidence before code.


### Proposed correction — 2026-10-23

Test-only, quick-flow scope. Keep the current wrapper Idle admission proof and freeze the full accepted journal as before, but do not interpret every later entry as the authored slash command. A pure journal oracle requires an unchanged prefix for every previously observed row, exactly one new /model, and no other authored input. An appended system entry is tolerated only when its exact envelope matches a real public broker receipt with OriginSystem, Submitted status, and the full captured fixture:0 sender and fixture:1 receiver bindings. A header or matching prose alone is never authority. Reject duplicate system delivery IDs, unknown additions and conflicting/terminal non-submitted receipts; known Queued/Held/Delivering receipts remain pending within the existing context.

The thin test adapter reads Accepted(1), resolves candidate IDs through the owned fleet's ActorCLI --message-status ID --json, and passes typed receipt evidence into the pure oracle. Cache only proved Submitted receipts for this subtest, at most the existing128 accepted-entry bound. Advance the immutable observed prefix on each valid observation, including a pending one, so late mutation or truncation of previously tolerated entries still fails. This changes neither delivery nor the acceptance journal's format.

Alternatives: waiting for Idle or a quiet interval is not a completion barrier for the asynchronous startup producer; adding a sleep repeats the measured race. A producer barrier or suppression switch changes production/harness lifecycle unnecessarily. Broad header filtering could hide spoofed or unexpected input. Use existing broker provenance instead (ARCH-DRY, ARCH-SECURE, ARCH-ORDER).

| Concept | Kind | Location | Change |
|---|---|---|---|
| Journal oracle and system-envelope candidate parser | PURE | cmd/couch/slash_journal_oracle_test.go | new test-only helpers and adversarial tests |
| Public qualification journal adapter | INTEGRATION | cmd/couch/slash_keys_acceptance_test.go | replace local length-only assertion; reuse Fleet.Accepted, ActorCLI and typed receipts |
| Retained delayed restart envelope | evidence | cmd/couch/testdata/slash-journal/ | faithfully extracted literal envelope plus provenance from the retained failed run |

Oracle state is scoped to one subtest: baseline, last observed prefix and receipt cache die with it. No new goroutine, persistent state, API, production source or cleanup owner. The existing disposable fleet owns actors/files and joins teardown (ARCH-PURE, ARCH-FUNERAL, ARCH-EVOLVE). A lock cannot make an asynchronous system producer finish before this read; its independent accepted input must be classified, not prohibited.

Actions: poll existing proof/journal, stop on a mismatch, inspect retained diagnostics.

| Observation | Test result | Next action | Evidence retained |
|---|---|---|---|
| No /model yet, or exact known system receipt still in flight | pending | poll within existing context | full observed prefix remains pinned |
| One /model and only receipt-proved system additions | ready | continue original prose/recovery assertions | full observed rows and proved receipts |
| Duplicate command, unexpected body, wrong origin/binding, duplicate system ID, prefix edit/truncation, failed receipt lookup | failure | stop and inspect | existing bounded diagnostics; no passing retry |
| Context expires while pending | failure | stop and inspect | unchanged timeout and cleanup ownership |

Bounds: retain setup180s, scenario120s, cleanup15s, current100ms wait cadence, existing2x8192-byte diagnostic capture, and all frame/output/fixture128-entry caps. For P polls and N journal rows, comparisons/classification are O(P*N); stable Submitted system IDs cause no repeated RPC. Count public lookup calls at8/16 distinct system IDs, requiring N first lookups and zero additional lookups on unchanged Submitted evidence. No deadline tuning, new polling worker or command replay (ARCH-CONSTRAINTS).

## Done when

- A delayed system restart notice before or after the baseline does not cause false command/prose failures.
- Duplicate /model, submitted ordinary prose, changed pre-existing journal entries and unrecognized additions still fail.
- Qualification uses the existing bounded disposable public command harness and exact owned cleanup.

## Plan

- [ ] Add the retained delayed restart envelope and pure RED regressions for arrival before/after the baseline and command, pending-to-submitted receipt order, duplicate commands/system IDs, authored prose, forged/unknown envelopes, wrong full identities, and journal mutation/truncation.
- [ ] Implement the test-only pure oracle and thin public receipt adapter; preserve current proof, namespace, timeout, byte and cleanup bounds; prove stable-receipt lookup counts at8/16.
- [ ] Run focused oracle/cmd-couch and audited reader inventory checks, then one disposable public qualification per Claude/Codex personality with exact cleanup; record each result without blind retries.
- [ ] Push the tested source, run the required recorded affected check/full fallback, inventory immediately before close, the binary-owned close review and normal SDLC landing gates.

## Log

2026-10-23: #162 source96abb5a7a4dee4c49e9a56afd8c6410c16ba605e. One disposable Claude run accepted /model once (receipt663d25eb-e060-43f0-8186-8415c53f7006), then observed the ordinary prose draft in the live pane. Journal contained /model and a Couch system restart delivery a2965831-4006-4231-910e-52b5c2a34aab; no ordinary prose entry. Frozen-baseline oracle failed at slash_keys_acceptance_test.go:187. Package34.886s, subtest27.13s. No retry and no flake registration. Evidence /Volumes/TestScratch/tmp/pair-slot2/pair162-review-regressions.pykj07ci/public-claude.log SHA256 a6269662c6e86f6ff9a18fb3fa27cd966a131391572cd48074556f6cab7b6b9d. Ops:0 deliveryc64e1031 accepts the measured behavior evidence for #162 and directs this separate unclaimed hardening issue.

## Revisions

### 2026-10-23 — Receipt-proved classification selected over a quiet baseline

The measured producer can deliver after wrapper Idle, so a timing-only baseline does not exclude it. Proposed scope uses current broker-origin/identity receipts to classify exact appended system envelopes, while preserving full observed-prefix immutability and exactly-once authored command/prose checks. No production change. Plan is proposed to ops:0 and implementation awaits its decision.

### Planning checkpoint

2026-10-23: claimed837 and started its own main-based branch000837-slash-journal-system-deliveries at a14c4a341a051b3819bef3ecfb95ec8ab5123007. Catch-up rebuilt local substrate tools; seeded workflow regeneration was explicitly deferred, with no workflow edit. Existing measured log rehashed to a6269662c6e86f6ff9a18fb3fa27cd966a131391572cd48074556f6cab7b6b9d. Source census found the sole frozen-baseline oracle in slash_keys_acceptance_test.go; broker Status already exposes immutable server origin and full binding to the authenticated sender/recipient. No implementation or native run has started. Quick flow fits: test-only source changes, no Mx, design below500 lines. Next: send this plan to the hub and wait for approval, then change-code and RED/GREEN.
