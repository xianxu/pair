---
id: 000556
status: working
deps: [pair#528]
github_issue:
target: slot-recovery
created: 2026-10-14
updated: 2026-10-15
estimate_hours:
card_mirror: '4f70a6e43e9fd6f5aefbb3b5e731b340a2e25f1d' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T15:10:56-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# Couch confirmed disk recovery and always-reachable fresh startup

## Problem

Follow-up to #528's accepted M1. Startup and Continue are accepted; the revised
M2 disk-recovery execution contract still needs a complete review and operator
acceptance. Existing ranked candidate/CLI/UI and fresh-orientation code is a
baseline, not evidence that every recovery edge is complete.

## Spec

Preserve the slot-continuity target's three outcomes. Explicit Resume from disk
uses a proven native binding or a best-supported/selected ranked candidate with
evidence. Start fresh is always reachable, including incomplete, unsupported,
weak or absent scans and missing/expired tty; disclose what context may be lost.
Use #522's ordinary launch Prompt to point at available previous-screen evidence
and branch/issue, never embed screen bytes in argv. Native resume has no prompt.

Before stopping an attributed running owner, require short-lived, non-persistent
person-or-hub confirmation naming exactly what stops. Recheck identities at both
admission and effect. Unknown/foreign remnants are never killed; leave/report
them and use a fresh name when needed. Preserve intent/evidence before cleanup;
retire only attributed owners, then clean exact dead-owned remnants/stale layouts
without force, revalidate, launch and report registration/readiness honestly.
No silent fresh replacement after an uncertain successful resume. One bounded
observation retry, then an actionable outcome rather than pending limbo.

An unfinished StartClaim with no positive listing must exit Starting when its
owner is dead or its time bound expires, through this confirmed retirement path.
Automatic inference stays OFF pending a witnessed corpus and approved threshold;
retain default plus up to four alternatives with evidence via menu and CLI.

## Done when

- Proven, best and alternate inferred resumes select the right native identity;
  no native transcript has two live owners.
- Fresh executes despite no/weak/incomplete evidence or missing tty, with truthful
  orientation from available branch/issue/screen coordinates.
- Running-owner confirmation works for person and hub; changed identities,
  cancellation, interrupted effects and uncertain launches preserve evidence.
- Foreign remnants are left intact and do not strand the row; stale Starting has
  a bounded exit. No default edits, native-file deletion, git reset or draft loss.
- Tests use measured native shapes and stateful lifecycle doubles; a recorded
  full suite and fresh-context review pass. TL schedules scratch-only operator
  acceptance across proven/inferred/no-candidate/missing-tty/owned/foreign cases.

## Plan

- [x] Inspect current owners and revise #528's M2 design before implementation;
  keep a separate review and operator acceptance boundary.
- [ ] M1 — Shared prepared recovery, confirmed Resume/Fresh, standalone hub CLI confirmation, schema-4 intent/read-only v3 migration; headless acceptance and morning scratch script; milestone-close.
- [ ] M2 — Shared native admission reservations and two-slot barrier tests; milestone-close.
- [ ] Record the full suite, close on SHIP and land; the hub owns the operator's restart and live acceptance.

## Log

### 2026-10-14

Split from #528 at operator/TL direction84fbcea7 so accepted M1 can land. Source:
#528 durable plan's revised M2 and Starting exception; no new code authorized.

### 2026-10-15 — claim and measured design audit

Claimed by pair:2 on branch `000556-confirmed-slot-recovery`, based on main
`09c5b48dd7586d6ba908a45ea86899190ad3dd4a`. Refreshed the stale ariadne
substrate before starting planning. Two independent, read-only audits covered
core execution/ownership and menu/CLI admission; no live lifecycle effects ran.

The baseline already has ranked keys, immutable capture preservation, native
owner checks, opaque exact terminal retirement, ordinary launch Prompt and
start-claim persistence. Its deliberate M1 restrictions still reject running
sources, remove Fresh on some rows, require a capture for candidate execution,
and let Starting last while its supervisor lives. Registration uncertainty also
enters generic post-ack cleanup. These are the implementation gaps, rather than
a reason to introduce another recovery queue or launch mechanism (ARCH-DRY).

There is no slot/thread lifecycle CUE model in this Pair tree; the existing
authorities are `threadrecord/record.go`, `threadrecord/recovery.go`,
`couchcore/starttransaction.go` and the park reducer. The plan extends those
owners rather than inventing a second durable lifecycle (ARCH-ORDER).

Follow-up audit found that the native owner observer explicitly does not lock
against simultaneous cross-slot creates. The plan includes bounded shared native
admission reservations at the common Pair resume boundary; the hub must review
this addition before code. The existing two ownership observations alone cannot
prove the issue's no-two-live-writers contract.

## Revisions

### 2026-10-15T05:33-07:00 — overnight M2 dispatch

The hub assigned #556 after #528 landed: plan to the hub, implement and land on
SHIP, with headless acceptance and a short live script for the operator's morning
restart. The issue now references the newer `slot-recovery` target; the earlier
`slot-continuity` commitments still constrain teardown, confirmation and Fresh.
The durable plan is awaiting the hub's approval; no implementation is started.
No messaging reconciliation (#561), wrapper endpoint reconstruction (#557),
automatic inference, live-slot experimentation or stage changes are included.

### 2026-10-15 — approved milestone split

Verified hub receipt `249a1bbd-5f2f-457a-b009-c2581eae3da9` approves the plan
with two separate milestone-close boundaries. M1 implements confirmed recovery
and schema migration, including standalone `couch --recovery-confirmation TOKEN`
for the hub and the morning scratch script. M2 separately enforces native
admission against concurrent writers. Tokens/reservations/schema are internal;
visible outcomes stay Continue / Resume from disk (confirmed) / Start fresh,
plus needs relaunch, with only the actual stop confirmation shown. Retain the
five-minute stale-Starting bound and one exact reobservation. Change-code next.
