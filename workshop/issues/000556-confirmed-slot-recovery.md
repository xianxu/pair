---
id: 000556
status: working
deps: [pair#528]
github_issue:
target: slot-recovery
created: 2026-10-14
updated: 2026-10-15
estimate_hours: 6.75
card_mirror: '5ecc1a115e45b5f378d73b176e25356b1b1f7292' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T15:10:56-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
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

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only; the calibration source is marked stale by
`sdlc estimate-source`, so this is provisional. Derived after plan-quality
round 2 CLEAN, not a target chosen before decomposition.

M1 has two bounded new Go concerns (prepared selection and ephemeral acceptance),
one lifecycle integration, a schema extension and the existing menu/CLI seam.
M2 adds a reservation reducer/store and the Pair/wrapper handoff integration.
The existing Go JSON, flock, procutil, retention, queue and launch owners avoid a
novel-stack library assumption. Apply the thorough-plan design discount (0.2)
to the module/API/TUI primitive design ranges, familiarity 1.0 for this code,
v3.1's 40% implementation scale exactly once, and the 15% design buffer.
The three review chunks are M1, M2 and whole-issue close.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.3 impl=0.12
item: greenfield-go-module design=0.2 impl=0.32
item: greenfield-go-module design=0.2 impl=0.32
item: api-integration design=0.6 impl=0.6
item: smaller-go-module design=0.06 impl=0.2
item: tui-screen design=0.2 impl=0.4
item: api-integration design=0.6 impl=0.6
item: greenfield-go-module design=0.3 impl=0.32
item: cross-cutting-refactor design=0.1 impl=0.2
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0 impl=0.2
item: milestone-review design=0 impl=0.2
item: milestone-review design=0 impl=0.2
design-buffer: 0.15
total: 6.75
```

Design subtotal 2.60h × 1.15 + implementation subtotal 3.76h = **6.75h**.

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

### 2026-10-15 — M1 implementation checkpoint

Implemented the shared prepared executor and ephemeral exact-stop confirmation
owner, standalone authenticated hub confirmation, schema-4 tagged recovery with
read-only schema-3 migration, terminal attribution/rechecking, stale-start fencing,
and uncertain-launch retention. Native resume carries no Prompt; Fresh uses the
ordinary #522 launch Prompt pointing at branch/issue and available screen evidence.
The missing-record Fresh path now carries that orientation and strict readiness.

Targeted integrated checks: `go test ./cmd/internal/couchcore -run
'^TestProvenRecovery|^TestPrepared|^TestRecoveryConfirmation|^TestResumeCandidate|^TestRecoveryRetirement|^TestRecoveryFresh|^TestRecoveryStartFence'
-count=1 -timeout=90s` passed (7.124s). Surface and adapter contributors also ran
their focused regression sets; no full-suite or milestone-close claim yet.
Remaining M1 integration covers retained-screen reading, preflight/retention, the
headless matrix, documentation and acceptance script, then the recorded suite.
No live slot lifecycle effects or stage changes were performed.

### 2026-10-15 — M1 integrated headless checks

Integrated exact retention/preflight before stops, normal configuration fallback
for Fresh, safely addressed previous-screen reads after new-tag Fresh, and an
early young-Starting/native park/continuation guard. Real Git broken-host
fixtures exposed two actual failures: checkout identity preflight failed before
confirmation, and readiness attempted repair with unknown agent liveness. The
fix verifies the primary catalog read-only, then uses normal slot convergence
only after confirmed retirement/positive absence; post-repair checkout identity
must still equal the prepared common directory. Operator files survive.

Core recovery/reboot/prepared checks passed at the integration checkpoint
(15.296s); surface/CLI/menu/protocol/schema/launcher focused checks passed
(1.906s/0.304s/0.169s/0.140s/0.245s). Integrated expired-claim/young-claim/open-park
tests passed under race (1.737s). Tagged previous-screen/evidence reads passed
under race, including independent retention leases. Help/skill cost parity first
failed for the new standalone confirmation verb, then passed after recording its
heavy cost and updating the public protocol reference.

The morning script is `workshop/plans/000556-live-acceptance.md`. No live lifecycle
effects, automatic inference or stage changes ran. M1 now proceeds to the recorded
suite and its binary-owned boundary review; M2 exclusion is still outstanding.

### 2026-10-15 — recorded suite found integration regressions

The first recorded suite on `aab86bd03fb0672e9a36c4b16b63462cb1479288` failed
(exit 2, 13m54s). Retained log:
`/Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/5b9ca5e5c1cacb22853c746a916aa39e.log`
(SHA-256 prefix `cce070e42401`). No waiver is requested. It exposed exhaustive
artifact inventory omissions, stale command/declaration expectations, lost warm
resume/advice routing, strict-Fresh fixture gaps, and a regression of the existing
unreadable-primary no-stop archive escape. Continuation reboot fixtures waited
for nonexistent fresh registration and reached the core ten-minute test limit.

Corrections classify the actual new artifact consumers (no exemptions), retain
normal facade entry points while removing the obsolete alternate Reboot
stop/launch implementation, and preserve exact unreadable bytes through the
existing archive journal with expected-byte CAS. Valid future versions still
refuse mutation. The primary no-path archive escape reports honestly that no
agent started and never touches its unproved processes. Normal readable Fresh
uses only the prepared executor, including the existing explicit agent choice
resolved from normal saved configuration. No free-form recovery argv is added.

A separate private-fixture measurement found recordless Fresh inheriting ordinary
Fresh's absence gate and refusing foreign/unknown remnants. Recovery-only
admission now allocates a distinct tag in the verified physical checkout, keeps
all exact claim/repository/readiness checks, and performs no workspace
convergence beneath those remnants. It leaves/report them and says setup was
not refreshed; ordinary Fresh remains strict. The initially retained dead legacy
API test strategy was replaced with tests of the shipped shared facade rather
than adding a dead-symbol exemption.

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
