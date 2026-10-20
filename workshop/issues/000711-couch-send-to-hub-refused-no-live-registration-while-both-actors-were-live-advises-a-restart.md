---
id: 000711
status: working
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '26da33de7d83d4e426d7ba0639fe8b4c3388e8f5' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T05:51:05-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
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
- [ ] Retain those probes as permanent regressions and implement one-listing observation,
  same-actor busy classification, and phase diagnostics; preserve negative controls.
- [ ] Run affected launcher, broker and messaging tests plus source/consumer guards;
  update atlas/lessons and file any shared lint finding in its owning repo.
- [ ] Push, record full suite, close on evidence, enter normal merge and report the landing.

## Log

### 2026-10-19

### 2026-10-20 — measured root causes

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
