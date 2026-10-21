---
id: 000769
status: codecomplete
created: 2026-10-21
updated: 2026-10-21
estimate_hours: 1.861
github_issue:
started: 2026-10-21T06:43:45-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
actual_hours: 1.18
tracker:
    version: 1
    completion:
        token: close-3c1c7766ea4a
        repository: github.com/xianxu/pair
        reviewed_head: dfc195a1b30268c577cbe3e8d2e21123f60c19b3
        evidence_commit: 59483b7405db5b2ebe06731e22d453b36ced9c58
---

# Preserve wrap Darwin PTY final output without changing termination

## Problem

#768 audited wrapcmd.runProxy/startNativePTY and confirmed a generic-command PTY owner exposed to Darwin last-slave-close final-output loss. #768 fixes scribe only; this owner needs its own lifetime design.

Native planning probe from #768: openpty/fork, setsid/TIOCSCTTY, child writes final-marker and closes all slave stdio, then acknowledges out-of-band before parent first read. Baseline lost the bytes; child /dev/tty open/close and parent retained-slave variants each preserved them. Retained variant reached EOF and exited7. Additional descendant probe kept an inherited slave open in a descendant ignoring SIGHUP: Darwin still delivered exact marker and EOF on leader exit while descendant was alive. These are feasibility probes, not qualification of this owner. Sources: https://bugs.ruby-lang.org/issues/20682 and Apple XNU tty_tty.c/kern_exit.c.

Termination-semantics risk: Master EOF invokes ownedGroup.Wait and stops a still-live process group. Retaining the slave delays this transition when the target closes stdio but keeps running; a generic retention change could leave such a target running indefinitely.
