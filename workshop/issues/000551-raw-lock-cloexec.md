---
id: 000551
status: working
deps: []
github_issue:
created: 2026-10-14
updated: 2026-10-14
estimate_hours:
card_mirror: '2ea8a08b29b28a54723027e8f29401bd83527520' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T08:51:33-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "d95d3f01", done: "c80b9267"}
---

# locks: close raw diagnostic and notification descriptors on exec

## Problem

The #545 lock-open sweep found raw descriptor opens without O_CLOEXEC in
cmd/internal/diagnosticlog/writer.go openRegular and
cmd/internal/notifytransport/address.go lockNamespace (and readPID in the same file).
Go os.NewFile does not add the atomic close-on-exec flag. These descriptors may
therefore survive unrelated execs. Both lock paths explicitly LOCK_UN, so this
is separate descriptor hygiene, not #545's pre-exec creation-lease cause.

## Spec

Add atomic close-on-exec on the raw opens in these two modules, sweeping sibling
raw opens in the same boundary. Preserve current validation and lock semantics.
No broader creation-lease or retry changes. TL requested this separate follow-up
in receipt cc82ffc3-c4f5-4c1c-9e9a-5b2fb14f5340.

## Done when

- Raw lock/binding opens in the two modules use O_CLOEXEC atomically.
- Targeted tests verify FD_CLOEXEC and existing lock/publication behavior.
- Full suite completes and the small change passes SDLC close/landing.

## Plan

- [x] Add failing kernel FD_CLOEXEC tests for diagnostic regular files and the held notification namespace lock, plus an AST guard over every production raw open in the two modules to prove atomic O_CLOEXEC rather than a later fcntl fix.
- [x] Add O_CLOEXEC at all four discovered opens: diagnosticlog.openRegular, diagnosticlog.directoryNames, notifytransport.lockNamespace and notifytransport.readPID. Preserve NOFOLLOW/NONBLOCK, validation, file modes, bounded reads and explicit lock release (ARCH-PURPOSE, ARCH-DRY).
- [x] Run both package suites and focused race tests, make -k build, and the required sdlc local-check suite on the pushed head. Close via SDLC review and land on SHIP; this descriptor bugfix creates no new architectural surface.

## Log

### 2026-10-14

Filed from #545 investigation; independent follow-up, not started.

### 2026-10-14 — pair:3 design

Hub ops:0 assigned this small fix and authorized claim, brief plan, build and landing on SHIP (receipt ca952211-058c-443e-83d8-1229b13c2e65). Claimed in pair:3. start-plan caught up the base layer and required a repeat with its rebuilt binary; the repeat entered planning. Raw-open sweep found the issue's three sites plus diagnosticlog.directoryNames in pages.go. A later CloseOnExec call would retain the fork/exec race, so add the flag to the existing opens; no new IO seam or helper is needed. Kernel flag tests plus the source inventory guard cover the actual descriptor policy and its atomic placement.

### 2026-10-14 — four-site fix verified

The kernel regressions failed on both regular-file modes and the held namespace lock; the atomic-open source guard failed at all four opens. Added O_CLOEXEC directly to the existing syscall/unix.Open flag expressions, with no other production change. Full diagnosticlog package passed (35.570s), full notifytransport package passed (3.229s), and focused race regressions passed for both packages. Existing publication, lock contention/release, PID parsing, paging and safety checks remain covered by those suites.

The namespace flag test matches its private lock inode in a bounded kernel FD scan. A first test fixture incorrectly assumed the parent's temporary directory was 0700 and /dev/fd would enumerate all macOS descriptors; corrected the fixture to let production create its private directory and use Fstat/Getrlimit instead. Both production descriptor tests then failed specifically for missing FD_CLOEXEC before the fix.

### 2026-10-14 — clean build and required suite

Pushed implementation 54aac8349cceb33280711abc1b288b29ec2cb80f. make -k -B bin/pair passed, and its embedded vcs.revision matches that commit with vcs.modified=false; SHA256 f88908cc6f024b3adbf44708e072880228e1e1b052eec8050de38262fee3b9f0. Required sdlc local-check run --name suite is queued behind earlier machine-wide requests. This is a pure flag-hygiene bugfix with no new module interface, operator flow or architectural surface; the close will acknowledge no atlas change via --no-atlas and retain this reason in verification.

### 2026-10-14 — authorized separate follow-up

After this issue lands, hub ops:0 authorized filing and fixing a separate quick-flow issue from main for the embedded Couch skill receiver-list drift (receipt 2a83ee65-f057-480a-97c4-406b81781fe4). cmd/internal/couchcmd/skills/couch/SKILL.md:502 omits Qoder support from #550. Sweep the skill and docs for other two-agent receiver-support lists, fixing the class while retaining intentional two-agent examples. Do not fold this independent work into #551 or start it on this unlanded branch.

### 2026-10-14 — full verification passed

make -k build and a final make -k -B build passed. Pair and Couch both embed fa0757f86688c04649a1b7fecf37e528e3252e2d with modified=false; that head only adds a Log entry beyond the implementation. Required sdlc local-check suite passed on 54aac8349cceb33280711abc1b288b29ec2cb80f (exit 0, 10m31s, after a 1550-second queue wait); no anchored FAIL/make Error entries. Log SHA256 fe3a000df6173b0b8f0d61381ffb313dc9d0886bd92682a062ce35b965a0c41f. Subsequent commits contain only issue Log updates.

## Revisions

### 2026-10-14 — complete raw-open family

The initial one-row plan is expanded to test/fix/verify steps. Include diagnosticlog.directoryNames discovered during the two-module sweep, alongside the filed regular-file lock, namespace lock and PID-binding read. The deliverable remains atomic descriptor hygiene without lifecycle or retry changes.
