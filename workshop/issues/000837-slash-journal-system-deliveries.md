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

- [x] Add the retained delayed restart envelope and pure RED regressions for arrival before/after the baseline and command, pending-to-submitted receipt order, duplicate commands/system IDs, authored prose, forged/unknown envelopes, wrong full identities, and journal mutation/truncation.
- [x] Implement the test-only pure oracle and thin public receipt adapter; preserve current proof, namespace, timeout, byte and cleanup bounds; prove stable-receipt lookup counts at8/16.
- [x] Run focused oracle/cmd-couch and audited reader inventory checks, then one disposable public qualification per Claude/Codex personality with exact cleanup; record each result without blind retries.
- [x] Prepare the pushed-source recorded affected check/full fallback and mandatory close handoff; record check, review and normal landing outcomes in the Log.

## Log


- 2026-10-23: closed — Test-only journal hardening; immutable baseline and every observed prefix; exact OriginSystem/Submitted envelope and full captured hub/worker bindings; spoofed/unknown/authored additions, duplicate commands/system IDs and row mutation/truncation fail. RED legacy oracle and no-op receipt adapter regressions turn GREEN; deterministic captured205-byte delayed restart replay and8/16 receipt lookup counts pass. Complete cmd/couch PASS19.408s; repolint14.108s, artifactpath13.668s, termcmd6.700s; make test-affected-inventory PASS immediately before close, two named reader audits only. On111be6e29, one disposable Claude PASS55.970s and one Codex PASS36.869s with exact owned cleanup; Codex exercised actual late system delivery b7628390-d192-41ba-9390-bebd737b3f97 through public message-status Submitted proof after command acceptance. Logs and SHA256s in issue Log. Atlas updated. Required affected check selected full fallback lease-2535690368 queued on7cf0dcf13; later commits prose only. Landing waits for its pass; no deadline, readiness, production, protocol or trust change. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP
2026-10-23: #162 source96abb5a7a4dee4c49e9a56afd8c6410c16ba605e. One disposable Claude run accepted /model once (receipt663d25eb-e060-43f0-8186-8415c53f7006), then observed the ordinary prose draft in the live pane. Journal contained /model and a Couch system restart delivery a2965831-4006-4231-910e-52b5c2a34aab; no ordinary prose entry. Frozen-baseline oracle failed at slash_keys_acceptance_test.go:187. Package34.886s, subtest27.13s. No retry and no flake registration. Evidence /Volumes/TestScratch/tmp/pair-slot2/pair162-review-regressions.pykj07ci/public-claude.log SHA256 a6269662c6e86f6ff9a18fb3fa27cd966a131391572cd48074556f6cab7b6b9d. Ops:0 deliveryc64e1031 accepts the measured behavior evidence for #162 and directs this separate unclaimed hardening issue.

## Revisions

### 2026-10-23 — Receipt-proved classification selected over a quiet baseline

The measured producer can deliver after wrapper Idle, so a timing-only baseline does not exclude it. Proposed scope uses current broker-origin/identity receipts to classify exact appended system envelopes, while preserving full observed-prefix immutability and exactly-once authored command/prose checks. No production change. Plan is proposed to ops:0 and implementation awaits its decision.

### Planning checkpoint

2026-10-23: claimed837 and started its own main-based branch000837-slash-journal-system-deliveries at a14c4a341a051b3819bef3ecfb95ec8ab5123007. Catch-up rebuilt local substrate tools; seeded workflow regeneration was explicitly deferred, with no workflow edit. Existing measured log rehashed to a6269662c6e86f6ff9a18fb3fa27cd966a131391572cd48074556f6cab7b6b9d. Source census found the sole frozen-baseline oracle in slash_keys_acceptance_test.go; broker Status already exposes immutable server origin and full binding to the authenticated sender/recipient. No implementation or native run has started. Quick flow fits: test-only source changes, no Mx, design below500 lines. Next: send this plan to the hub and wait for approval, then change-code and RED/GREEN.


### 2026-10-23 — Approved quick flow; oracle and adapter RED/GREEN

Authenticated ops:0 delivery92e81965 approves the receipt-proved design exactly: immutable full prefix, OriginSystem/Submitted/exact envelope/full captured bindings, unknown/spoofed/duplicate/authored additions fail, known in-flight stays pending within120s. sdlc change-code passed inferred quick flow; no estimate or production changes.

Pure oracle helper/regressions live in slash_journal_oracle_test.go. Behavioral RED using the old length-only rule reproduced a late proved system notice and same-length prefix mutation; current stateful regressions pass across missing command, receipts Queued/Held/Delivering→Submitted, before/after command and baseline, repeated IDs, duplicate /model, unknown/prose input, every prefix field, truncation and input-alias mutation. The literal205-byte delayed restart envelope was extracted from the original measured Go diagnostic (source log hash above), without changing its delivery ID or paragraph; SHA256ee27b17dcf2ca813951d11af96a740c95753090019e0b09ee2f8d70d7931206d. Fixture provenance explicitly distinguishes extracted tool-output bytes from the removed original accepted.jsonl and typed fake broker proofs.

The thin receipt adapter initially ran as a no-op; its RED tests failed with zero lookups at8/16, missing pending receipt proof and accepted unproved terminal replies. Current adapter tests pass: N first lookups at8/16, zero repeated lookups for Submitted evidence, pending requery after transition, no cache admission for wrong origin/body/ID/identities or terminal failure, and propagated lookup errors. Native assertion wiring uses the same pure oracle and adapter for command, prose and recovery; in-flight evidence polls only the existing waitFleet/context, with no replayed keys. Previously captured hub/worker full Actors bindings are validated and pinned.

Focused TestSlashJournal PASS0.204s; complete cmd/couch PASS19.408s. Supporting preflight: repolint14.108s, artifactpath13.668s, termcmd6.700s. Reader inventory first named the changed native acceptance reader and then the new oracle reader after staging. Audited the existing fleet/public native tools and accepted journal inputs, and the oracle's sole os.ReadFile input testdata/slash-journal/delayed-restart-envelope.txt. Recorded only those two per-reader certificates as non_go_full_fallback; final inventory PASS. No bulk hash refresh and no local-gates configuration change.

Next: push this checkpoint, run one disposable native public qualification per Claude/Codex personality on this head, retain exact outcomes/cleanup, then required recorded affected/full fallback and close. No native invocation has been run yet for837; no queued suite or merge runner exists for this item. No operator stage or trust mutation is authorized.


### 2026-10-23 — Both public qualifications pass; actual late delivery proved

One invocation per personality on111be6e29dac54fc0778cbd868c1ecffcc03ac14, no retry. Claude PASS55.970s (subtest32.32s), receipt2195a905-a4fd-4b6c-92be-b2f6629f6517, /model accepted once, ordinary prose retained, recovery and exact owned generation/root cleanup with direct IO joined. The restart notice arrived before the baseline in this run; no late-receipt lookup claim. Log /Volumes/TestScratch/tmp/pair-slot2/pair837-native.2z48hu_p/claude.log SHA256167f23e4c7430fc7876d6c292a3afcb9e64078cb0858d471506c0b6f6ae85e87.

Codex PASS36.869s (subtest28.56s), command receipt217e9283-81ba-4cc8-84cf-141f37b1a6d2, exact cleanup. Crucially, system deliveryb7628390-d192-41ba-9390-bebd737b3f97 arrived after /model acceptance; the real public message-status returned Submitted, and the shared pure proof validated its system origin, exact envelope and full captured bindings before the later journal checks passed. This qualifies the formerly failing ordering through the real adapter, not just the fake. Log /Volumes/TestScratch/tmp/pair-slot2/pair837-native.2z48hu_p/codex.log SHA256ce7ec27438f0b374cc1a5357f523d3d97f80fe9f4a5ab2564f87a59b68ef08aa.

PR566 is open for CI overlap; its local-gates guard passed. Atlas/couch-testing.md maps the test-only oracle and opt-in command. No production change, live fleet contact, actual model UI invocation, trust write, deadline change or operator stage. Lookup-growth tests bound RPCs by distinct receipts; per-poll validation remains linear in journal records and envelope bytes within existing128-entry/frame/body bounds. Next: required recorded affected/full fallback on the pushed documentation descendant, inventory immediately before mandatory close, then normal landing gates. The actual-hours exception for legacy162 does not apply to modern837; use normal measured close without typed hours.

### 2026-10-23 — Gate checklist bookkeeping

The last Plan row originally read: "Push the tested source, run the required recorded affected check/full fallback, inventory immediately before close, the binary-owned close review and normal SDLC landing gates." Clarify it as preparation for these binary-owned gates, recording their real outcomes in the Log, so close does not require falsely checking review/landing before their verbs execute. Product Done-when, scope and verification requirements are unchanged.


### 2026-10-23 — Required recorded fallback queued; close handoff

sdlc local-check run --name affected on pushed7cf0dcf13 selected full coverage because of the audited captured-input reader cmd/couch/slash_journal_oracle_test.go, then started the declared full-suite fallback. Retain terminal64772, lease-2535690368, checkout /Volumes/TestScratch/local-checks/sdlc-local-1489521246, inflight log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/inflight/61b1ee187cbb8a3e5ac919be0bc4f3ff/run.log. Initial FIFO8 behind legitimate ariadne:4; no test result or pass claimed. Later commits are documentation only; code and native qualification source remain111be6e29dac54fc0778cbd868c1ecffcc03ac14. Preserve this runner and FIFO, with no duplicate check or continuation/restart while it owns the run.

Inventory preflight PASS again after both per-reader audits. All active Plan deliverables and close preparation are checked; actual recorded check/review/landing outcomes remain pending here. Next: mandatory binary-owned close review may overlap the queued recorded check per tl-worker; use normal measured actuals, and wait for this fallback pass before landing. If review changes test code, checkpoint the correction and replace only its stale owned check through supported SDLC controls. No manual main integration, configuration label or gate waiver is authorized.
