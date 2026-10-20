---
id: 000711
status: done
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '254e4d102ad28114cef71fe1f396ebe6d9f0f52c' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T05:51:05-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "610cc940", done: "c315e15f"}
actual_hours: 1.38
---

# couch --send-to hub refused 'no live registration' while both actors were live; advises a restart

## Problem

2026-10-19 ~13:15, pair:7 (Codex) ran `couch --send-to hub --message '<a long plan summary>'` and got exit 1 with no receipt:
`couch: unavailable: Couch cannot find a live caller or recipient registration; if the slots are running, restart Couch to
reconnect them`. Right after, `couch --actors --json` showed both pair:7 and ops:0 live with their recorded incarnations, and
a shorter message sent to `ops:0` by exact address succeeded (receipt 07decfe3…). Earlier the same day the hub hit a related
transient refusal on a send: `couch: unavailable: observe listed session: session listing observations disagree`.

Either the `hub` alias resolution fails transiently (or for this caller) while exact addressing works, or a transient
registry/listing disagreement is reported as "no live registration" and tells the slot to restart Couch. The advice is wrong
either way: restarting Couch is the most disruptive action available, and the slots were live.

## Spec

Use one validated native full session listing for each observation. The direct source,
existing listing poller, and runtime session-list consumer derive names and exited state
from that same reply. No short/full temporal pair, new retry, watcher, or executor.
Preserve exact native names, strict complete-reply validation, successful-empty uncertainty,
exact native no-sessions handling, cancellation and existing poller freshness limits.
Malformed refreshes do not replace or freshen the last good observation.

For a send whose recipient actor and both registrations are unchanged, operator input
invalidating the reservation is busy, handled by the existing exact-target queue. A real
sender/recipient disconnect or replacement still refuses; no stale reservation gains write
authority. Family routing retains its quiet-recipient requirement. The service's publication
ordering and wrapper-incarnation fencing remain unchanged.

Refusals identify caller lookup/current validation or recipient lookup/observe/reserve
phase and target where known. Preserve error classes; remove the CLI's blanket restart
advice, including for old-supervisor bare unavailable replies. Hub and exact addressing,
and short/long valid message bodies, share these semantics. No content enters diagnostics.

Architecture: ARCH-DRY reuses the full-list parser and existing exact queue; ARCH-PURE
keeps parsing/classification separate from native IO; ARCH-ORDER keeps actor identity
and lifetime checks ahead of the busy decision; ARCH-MOCK uses the existing stateful
Zellij fake and broker endpoint. No persisted format or new durable resource is introduced
(ARCH-EVOLVE, ARCH-FUNERAL). No credentials or new trust surface (ARCH-SECURE).

Operating cost: one native listing per direct read/poller refresh, zero native calls per
cached read; O(N) parsing and existing sorting for N sessions. Count at 8 and 16 sessions,
reject a second listing and per-reader native work. Existing time/queue/capacity limits
stay unchanged (ARCH-CONSTRAINTS); no timing threshold proves the fix.

Actions: inspect receipt; resend a refused request; inspect actors and report persistent refusal.

| Internal outcome | Visible state | Recovery action | Possible loss |
| --- | --- | --- | --- |
| Unrelated session churn, selected session present | Normal accepted/queued receipt | Inspect receipt | None introduced |
| Same actor, operator input invalidates reservation | Queued on existing exact path | Inspect receipt | Existing queue disconnect/uncertainty rules |
| Actual absent/replaced caller or recipient | Refusal names side and phase | Inspect actors and report persistent refusal | Request was not admitted |
| Malformed/no-reply/stale listing | Refusal names observation cause | Resend a refused request after inspecting | No authority granted from unknown evidence |
| Write/reply uncertain | Existing uncertain ID and receipt advice | Inspect receipt | Delivery may have happened; no blind retry |

Covered: deterministic ordering of input/reservation and session churn; absent, malformed,
stale and replaced negative controls; hub/exact parity. Not covered: proving which
interleaving caused the 13:15 historical incident without a retained trace, or changing
registration-publication order. The 23:40 error exactly matches the listing probe.

## Done when

- Permanent regressions retain the three measured red cases: live hub input crosses
  reservation, direct unrelated session churn, and cached unrelated session churn.
- Same-incarnation input queues; disconnect/replacement cannot admit stale work. Actual
  missing recipient and caller errors identify the side; hub/exact and body-length parity hold.
- One validated full listing supplies all listing consumers, including poller publication;
  strict malformed, duplicate, exited, empty/no-reply and freshness controls pass. Counted
  8/16-session checks prove one native query, zero per cached read, with no retry.
- Targeted tests, recorded full suite, close review and ordinary merge gates pass; hub
  decides whether acceptance rides the next natural restart. No pair:0 action without dispatch.

## Plan

- [x] Trace errors and reproduce three deterministic failures using disposable Go overlays.
- [x] Retain those probes as permanent regressions and implement one-listing observation,
  same-actor busy classification, and phase diagnostics; preserve negative controls.
- [x] Run affected launcher, broker and messaging tests plus source/consumer guards;
  update atlas/lessons and file any shared lint finding in its owning repo.
- [ ] Push, record full suite, close on evidence, enter normal merge and report the landing.

## Log

### 2026-10-19

### 2026-10-20 — measured root causes
- 2026-10-20: closed — The complete recorded suite attempt 0d6db2414b5bdc28294ef055d6fd039b on a85138a5 ran all checks and failed only four deterministic couchcore fixtures still expecting short-plus-full listings or their old fault ordinal. The two test files now preserve four independent warm-resume listing fences and all fail-closed/no-effect/capture guards with one full listing per fence; added unknown full-listing coverage. The four corrected tests passed together in 4.589s. No production code, scope, timeout or admission change. Fresh recorded full suite is running on pushed correction 623193258 and must pass before landing. Project owned by hub; final unchecked Plan row is remaining verification/landing workflow. Live check remains owed at the next natural Couch restart per hub approval; no staging. (fixed-then-ship) (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP
- 2026-10-20: closed — Round1 findings corrected with measured red/green: original sender SlotActor is captured under Send lock and checked after routing IO, reservation and Release before queue; same-binding sender reconnect at all three boundaries refuses with no delivery, ordinary unchanged-actor input still queues and submits once. Runtime row builder consumes the validated exited-state map; live EXITED-containing names and exited spaces/tabs preserve state and skip exited client queries. Full admission refusal sweep labels caller/recipient phases and preserves typed recovery classes. Corrected packages passed: couchmessage26.028s launcher65.829s couchcmd47.995s artifactpath7.139s termcmd2.546s repolint7.705s. Removed orphaned short parser; final parser/runtime checks passed1.681s. Fresh recorded suite started on a85138a58b338cdba337fbb4b2e504ebfc5efffe and must pass before landing; old queued suite cancelled, no pass claimed. Hub approved no staging and live delivery on next natural restart. --no-plan-check only final pending suite/close/land workflow row; implementation complete. --no-project respects TL ownership; README/atlas/lessons updated. Logs /Volumes/TestScratch/tmp/pair-slot7/pair711.CE03YX; lint follow-up ariadne#496. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP

At base 7d4e0c3aba6b39fb12d5125ee967dcde9e5be4be, disposable Go-overlay probes
failed as designed (no tracked source edits). After successful Reserve, one OperatorInput
left both Caller lookups valid but Send returned `recipient is not live`. Stateful Zellij
added pair-unrelated on query 2: direct and polled liveness both returned `session listing
observations disagree` while pair-a stayed present. Log:
/Volumes/TestScratch/tmp/pair-slot7/pair711.CE03YX/root-probes.log. No live process changed.

Hub evidence: 2026-10-19 ~23:40 PDT, ops:0 to ariadne:5 refused with
`couch: unavailable: observe listed session: session listing observations disagree`;
an identical fresh send seconds later succeeded (25bfb2f8-7325-484f-83d5-81948ac0ebbc).
Hub reported load around 50+; no detailed native trace retained. Alias/length are not
necessary to reproduce either mechanism. Initial `sdlc issue show` output lists headings,
not bodies; the full original Problem is present after reading the details file.

## Revisions

### 2026-10-20 05:55 PDT — root fix replaces retry acceptance (hub approved)

The original Spec asked for a bounded retry after disagreement, and Done when required
transient disagreement to deliver after an internal retry. Replace that mechanism with
one validated full snapshot: the two separate native observations caused the contradiction,
so re-querying is unnecessary and would conceal unknown evidence. The three measured
probes become permanent regressions. Separate unchanged-actor input invalidation from
actual actor replacement/disconnect, using the existing exact queue. Add phase diagnostics
and remove blanket restart advice. Approval: Couch delivery
9cbc51e5-34b5-421b-8169-55b1b18a5e33. No new retry/state/resource and no production
registration-publication rewrite. The previous contract is preserved below as superseded.

<details><summary>Superseded initial Spec / Done when / Plan</summary>

> ## Spec
> 
> - Reproduce: is it the alias (`hub`), the message length, or a transient listing disagreement? Instrument the refusal with
>   which side (caller or recipient) failed and why (no registration vs. disagreeing observations vs. alias unresolved).
> - A transient disagreement retries within the send's own bounded deadline before refusing.
> - The refusal names the failing side and the real cause, and its advice is proportionate (e.g. "retry in a few seconds;
>   if it persists, `couch --actors` and tell the operator"), never "restart Couch" unless the registration is really gone.
> 
> ## Done when
> 
> - A fixture with a transient listing disagreement delivers after an internal retry; a fixture with a missing recipient refuses
>   naming the recipient; the `hub` alias and the exact hub address behave identically in both.
> 
> ## Plan
> 
> - [ ]
> 
</details>

### 2026-10-20 — implementation and focused verification

Permanent broker red: live input failed with ErrUnavailable; sender disconnect, recipient
disconnect and same-binding reconnect controls passed. All three root regressions now
pass, with exactly one delivery after queue promotion. The listing suite additionally
proved malformed refresh must not renew the last-good age, and counts one native query
at 8/16 sessions with zero calls from repeated cached reads. Focused core packages passed
(couchmessage 0.178s, launcher 1.263s); the migrated launcher native-reply/empty/cancellation
selection passed 37.387s. Obsolete paired-list tests now test one complete reply's exact
presence/state, malformed and duplicate rejection; their old agreement expectation
encoded the removed race, not a weakened native parser.

Phase tests were red before diagnostics: 28 failure cases lacked side/phase, four healthy
hub/exact and short/maximum-body controls passed, six legacy CLI cases showed restart
advice. After error-context changes the entire matrix passed (couchcmd 0.835s), retaining
response codes, error causes and refusal/no-receipt policy. No message body is diagnostic.
The initial permanent listing red command briefly failed to compile while the parallel test
migration already used the new poller API; an overlay restoring only the old poller test
API then produced semantic red failures for direct/polled churn, 8/16 query budgets and
malformed-refresh freshness. No compiler failure is counted as causal evidence.

### 2026-10-20 — broader verification and migration audit

Affected packages passed: couchmessage 25.090s, launcher 67.510s, couchcmd 48.813s,
artifactpath 6.485s, termcmd 2.018s, internal/repolint 6.049s. The first package pass
found obsolete expectations in recovery-class diagnostic equality, pre-540 compatibility
restart advice, fleet query-attribution totals and managed-attach native ordinals. Updated
tests retain the original typed classification plus the new phase, no automatic retry,
and attribution for two native queries (one full inventory and one client query), not three.
Managed RunLaunch still has three independent admission/effect fences; exported attach
still has two. Each now uses one full query; wrong-kind replies at every fence stop all
attach/cold fallback. Replacement at the last independent fence refuses effects.

Preserved deterministic client-query order by sorting selected exact names before IO;
otherwise map iteration would add native-order nondeterminism. Listing churn/count/runtime
and reordered-query checks passed three repetitions (2.539s). The Runtime.ListSessions
consumer also derives its names from one full observation rather than a second short view.
Existing parser and empty-reply authority remains unchanged. Full-suite evidence follows.

Filed unclaimed ariadne#496 for a narrow shared lint catalog guard against recognizable
paired views of a live inventory; no peer code changes. README and atlas now describe
phase evidence and proportionate inspection rather than blanket restart advice.

### 2026-10-20 — rollout agreement

Hub approved no pair:0 staging (delivery 9de20235-7d88-4506-b408-b37fa771fd2d).
The deterministic tests qualify the races before landing. Ordinary hub and exact-address
delivery after the next natural Couch restart is the live check; the hub owns recording
that acceptance in smoke calibration. No live check is claimed here. Branch PR is #485.
Recorded suite started on implementation head eb18dbd6ebf7b205da6b7fcd4e824e1f00a8e5c4;
its pass and the close review remain mandatory before landing. The final unchecked Plan
row is this remaining verification/publication workflow, not unfinished implementation.

### 2026-10-20 — close round 1: REWORK, no close finalized

BR-1/BR-3 permanent red regressions confirm sender same-binding reconnect can authorize
an old request at catalog IO, after successful reservation, and during release before
busy-queue fallback. Both Caller lookups remain valid, yet old Send incorrectly returns
queued; four refusal paths also return bare ErrUnavailable. BR-2 red confirms runtime
listing reinterprets a live EXITED-containing name as exited and exited whitespace/tab
names as detached. This is not a claim about the historical 13:15 interleaving.
The obsolete queued suite on eb18dbd6 was cancelled through local-check cancel before
code corrections; it provides no passing evidence. No gate was waived or finalized.

### 2026-10-20 06:15 PDT — Revision: complete the existing lifetime/parser contracts

BR-1: capture the existing sender SlotActor pointer under the initial Send lock, then pass
that same lifetime resource through admit/tryAdmission. One shared locked predicate must
check exact actor identity plus connectivity at the routing-loop check, post-reservation
check, and release-to-queue fallback. Recipient pointer/reservation checks remain first
before classifying input as busy. No counter, persisted state, queue, retry or executor is
added. Historical retained receipts remain recovery evidence, not new delivery authority.

BR-2: pass the validated full-list exited-state map into buildListRowsForScope instead of
discarding its values and reparsing strings through RowState. Migrate its two pure/unit
fixture callers alongside OSRuntime.ListSessions. Runtime tests cover live names containing
EXITED plus exited names containing spaces/tabs and assert no client call for exited rows.

BR-3: the complete bare-unavailable admission sweep is Send entry, admit after family
routing IO, and release-to-queue fallback; split caller/recipient failures and label phase,
while preserving errors.Is and recovery-class semantics. Permanent tests drive all sites.
These complete the already-approved unchanged-both-registrations and one-parser contract;
visible states and acceptance arrangement remain unchanged. Hub informed before code.

### 2026-10-20 — review corrections verified

BR-1/BR-3 red tests failed before code in every enumerated reconnect/refusal phase; after
capturing the original sender actor under lock and checking it through the three yield
boundaries, all pass. Ordinary input still queues/submits once, and actual sender/recipient
loss or reconnect cannot authorize the old invocation. BR-2's three opaque-name cases
failed before code and now pass, with validated state propagated into the runtime builder
and no native client query for exited names. Focused green: couchmessage 0.174s and
launcher 0.464s; log review-fixes-green.log in the retained scratch root.

Corrected full affected-package run passed: couchmessage 26.028s, launcher 65.829s,
couchcmd 47.995s, artifactpath 7.139s, termcmd 2.546s, internal/repolint 7.705s.
Log review-affected-final.log. No source changes followed that run. These corrections
complete the existing contract rather than introduce a third mechanism or visible state.
A fresh recorded full suite and second whole-branch close review are still required.

Final consumer sweep found pairSessionNames had become production-orphaned after the
runtime migrated to the validated map. Removed that old whitespace-splitting short parser
and its obsolete deduplication unit test; strict complete-list duplicate refusal and exact
opaque-name filtering remain covered by the new production-path tests. No new mechanism
or operating contract: this retires the replaced interpretation rather than leaving a
second parser available for accidental reuse. Focused parser/runtime verification follows.

Focused parser/runtime checks after retiring the orphaned helper passed (launcher 1.681s);
log parser-retirement-final.log. Only deletion of the now-unused short parser/test followed
the six-package pass; the recorded suite will qualify this final head.

### 2026-10-20 — recorded suite exposed four remaining legacy fixture expectations

Attempt 0d6db2414b5bdc28294ef055d6fd039b ran the complete suite on a85138a5.
Its four couchcore failures are deterministic fixture drift from the approved
one-full-listing change, not flakes: TestWarmResumeListingBudgetIsIndependentOfOtherSessions
still expected eight native calls instead of four independent full-listing fences;
TestRecoverThreadNativeReplyUnknownStopsBeforeEffects and
TestRecoverThreadNativeHeaderOnlyClientsReattaches still included the deleted short query;
TestRecoveryQueryCaptureRetainsFailedRecoveryRequest injected its client failure at old
ordinal five, now the next owner query. Updated only those fixture expectations, keeping
all four warm-resume fences, all no-effect/no-retry/identity checks, and exact failure
capture. Added the full-listing unknown-reply stage alongside both owner stages and clients.
The four corrected tests pass together (couchcore 4.589s). No production changes,
no raised timeout, no waived failure. A new recorded suite and narrow post-close review
will qualify this test-only correction; the earlier SHIP does not cover these edits.
