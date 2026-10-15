---
id: 000556
status: open
deps: [pair#528]
github_issue:
target: slot-continuity
created: 2026-10-14
updated: 2026-10-14
estimate_hours:
card_mirror: 'bbfae38f89795c43ebeb471b6b43bb2b99b4c269' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ] Inspect current owners and revise #528's M2 design before implementation;
  keep a separate review and operator acceptance boundary.

## Log

### 2026-10-14

Split from #528 at operator/TL direction84fbcea7 so accepted M1 can land. Source:
#528 durable plan's revised M2 and Starting exception; no new code authorized.
