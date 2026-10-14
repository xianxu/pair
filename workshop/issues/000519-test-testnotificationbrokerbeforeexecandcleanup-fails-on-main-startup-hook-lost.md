---
id: 000519
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-13
estimate_hours:
card_mirror: '2e25a680188420fabe3fefdd4e22fbc50bc941ab' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-13T17:55:33-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# test: TestNotificationBrokerBeforeExecAndCleanup fails on main ('startup hook lost')

## Problem

`TestNotificationBrokerBeforeExecAndCleanup` fails with 'startup hook lost' on main (pair:6, 2026-10-12, while closing #483): it reproduces alone, and with every #483-changed Go source replaced by origin/main through a Go overlay, so it predates #483.

## Spec

Find when it started failing (bisect on main), confirm the cause, and fix the root, not the test's timing. Never under machine load.

## Done when

- The test passes 20 runs in a row on main; the fix fails it when reverted; the cause is in the Log.

## Plan

- [ ] Reproduce with ordered fixtures and locate introduction; compare the cited suite failures.
- [ ] Drain accepted notification work before broker/terminal shutdown, keeping existing bounds and validation.
- [ ] Prove the fix with regression red/green, 20 consecutive startup-test passes, affected-package checks and make -k test; file unrelated cancellation evidence.

## Log

### 2026-10-12

## Revisions

### 2026-10-13 — TL assigns root-cause work

- Require 20 consecutive passes; inspect the suite-only PTY cancellation and wrap startup-notification reports, folding shared causes into this issue and filing independent failures. Bisect main only if needed to locate the introduction; no timing relaxation.

## Investigation

- Source/history: broker and early-child test were introduced together by d36fef2118df5719b1a96ec9ecce90072494863f (#255 M4). The same EOF return and deferred immediate broker close remain today. Send succeeds once the datagram reaches the kernel; no delivery acknowledgement orders it before child exit. masterPump can select PTY EOF before a queued broker message, and Close can also discard a datagram before the receive goroutine runs. Two pending-work boundaries must drain.
- /tmp/pair535-make-k.log, /tmp/pair536-make-k.log and /tmp/pair533-full-suite4.log all name the exact same TestNotificationBrokerBeforeExecAndCleanup and banner-only output. The cited wrap startup failure is not a second test.
- The PTY cancellation failure is independent: it uses couchcore runner/helper process-group liveness, without notification transport or wrapcmd. Its root remains unproved; the existing log lacks group-member evidence. File separately rather than changing its timing here.
- Current load averages are above 80; honoring this issue's quiet-machine constraint, investigation is read-only so far. No reproduction/stress loop has run.

### 2026-10-13 — bounded shutdown design and test authorization

- TL confirms bounded deterministic targeted tests may run now; 20-pass qualification should run below approximately 40 load, or retain load evidence if the machine never quiets tonight.
- Design: broker Close interrupts and joins its reader, then nonblocking-drains the socket under a fixed receive budget through the same validation/enqueue logic, then closes channels/socket and removes its owned files. The wrapper closes and consumes remaining broker messages/diagnostics before finishing stdout. No sleeps, longer test deadlines, wire-format changes, or extra goroutines.
- ARCH-DRY/PURE: one datagram validator/enqueue policy and the existing single stdout owner. ARCH-PURPOSE/ORDER: cover both buffered-message and kernel-datagram loss at EOF; preserve detached reader ownership and process no datagram twice. ARCH-MOCK: real isolated Unix sockets and pipes test shutdown order. ARCH-CONSTRAINTS: nonblocking drain is bounded by QueueSize; overflow/exhaustion is diagnosed, as with normal queue overflow. ARCH-SECURE: retain socket ownership/identity checks and message sanitation. ARCH-FUNERAL: existing binding/socket cleanup remains idempotent and joined; no new durable artifact.
- Independent cancellation follow-up filed as #542. The two wrap startup reports are one test and remain in #519.
