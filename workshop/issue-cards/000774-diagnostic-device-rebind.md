---
id: 000774
status: working
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
github_issue:
severity: high
started: 2026-10-21T08:53:21-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
---

# diagnostic checkpoints survive device remounts

## Problem

Couch supervision logging failed at fresh startup after remount/reboot because persisted st_dev is transient. At the operator-coordinated 2026-10-21 08:44 restart, resources reported unavailable, offered=77 written=0 dropped=77, last write never; open error: diagnostic reservation identity or size changed. Instrumentation from #736 M1/PR502 exposed the actual case.

Read-only bracketed metadata observations were stable: supervision.jsonl actual device16777229, inode728006625, size26061670, mtime_ns1792450564739652454. state.current records device16777232 with identical inode, size and nanosecond mtime. A retained old reservation describes the previous committed append: before size26055958 plus length5712 equals current26061670. No payload contents were read or live files/processes changed.

recoverReservedAppend first rejects checkpoint matching because of device alone, then attempts recovery using the already-committed old reservation and refuses. Saving the current writer registration before recovery explains the fresh PID/state.json timestamp despite Open failure. This is not measured truncation, unknown tail or AMFI.
