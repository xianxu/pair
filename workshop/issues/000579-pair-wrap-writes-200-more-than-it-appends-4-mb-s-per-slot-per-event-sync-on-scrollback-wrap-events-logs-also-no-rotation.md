---
id: 000579
status: working
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: 'b2a40e6a859cfbcf52526bdeb5f354a7954790a4' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T15:07:36-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# pair wrap writes ~200× more than it appends (~4 MB/s per slot): per-event sync on scrollback/wrap-events logs; also no rotation

## Problem

2026-10-15 (operator's system monitor): each `pair wrap` process (`--scrollback-log …/scrollback-1-<tag>-codex.raw`)
**writes 3.4–4.1 MB/s to disk**, about 29 MB/s total for the fleet (~2.5 TB/day of write wear). Yet the files it
writes grow only **~17 KB/s** (`wrap-events-1-<tag>.jsonl`) and **~2.3 KB/s** (`scrollback-…raw`) per active slot,
measured over 10 s. That's a **~200× write amplification**, most likely a sync (fsync/F_FULLFSYNC) or a re-open per
small event write, so every few-hundred-byte append costs whole blocks plus filesystem journal writes. It's also likely
behind the FSEvents daemon's ~200% CPU seen on 10-14.

Separately, these files are **never rotated**: wrap-events logs of **3.9 GB** (pair-8), **2.6 GB** (pair-43), 1.9 GB,
1.7 GB, 1.0 GB; the pair data dir was 57 GB on 10-14.

## Spec

1. **Measure first:** confirm the cause (count write/fsync syscalls per second for one wrapper, e.g. `fs_usage` or
   dtrace on a scratch slot), and record bytes-written vs. bytes-appended.
2. **Fix the amplification:** buffer event and scrollback writes and flush on an interval or size threshold; fsync only
   at real durability boundaries (e.g. before acknowledging a delivery, or at turn end), not per event. State what a
   crash may lose (at most the last interval's events) and why that's acceptable for each file.
3. **Retention:** rotate wrap-events and scrollback logs by size (e.g. 64 MB segments) and keep a bounded number or age
   (consistent with crash-only: what recovery and orientation actually read must remain).
4. A counted invariant (#204 style): bytes written per appended byte stays under a small bound for a synthetic event
   stream (no timing thresholds).

## Done when

- A wrapper under normal agent output writes on the order of what it appends (target: < 3× amplification), measured on a
  live slot before and after.
- Logs rotate; total size stays bounded; recovery and `couch --peek --previous` still work across a rotation.
- Tests: the amplification invariant; rotation and retention; crash-loss bound.

## Plan

- [ ]

## Log

### 2026-10-15

- Claimed and started on pair:3 from main a3beedd0074c536a49396ca68e5e038eebfb35d7. start-plan refreshed sibling Ariadne, then reran successfully.
- Measurement before design: proc_pid_rusage(pid 73688, RUSAGE_INFO_V2) over 10.006 seconds counted 7,385,088 physical disk-write bytes versus 86,077 appended raw+wrap-event bytes (85.80x). Files grew 63,591 wrap-events +22,486 raw bytes. Python ctypes called libproc directly; no root or mutation of the wrapper. Activity level affects the rate; this is one observed window, not a universal 200x.
- A 3-second sample of PID73688 found 109 sampled rename frames and 43 write frames; sampling is attribution evidence, not a syscall count. sudo -n fs_usage refused because a password is required. Hub confirmed agents cannot sudo and authorized inode-change/file-growth measurements instead (receipt ee56adcd-39bb-4351-a29b-012021701a6b).
- Read path: diagnosticlog.appendChunk publishes an intent (including base64 payload), reopens/appends the log, recoverAppend reopens/reads the tail, then republishes state. Both ordinary state publications are nondurable (no hot-path fsync), but each uses create/write/close/rename. Thus per-event atomic metadata churn, not per-event payload fsync, is the concrete hot path. Root registrations are durably republished on each Open, including unchanged entries. Existing diagnostic rotation is 64MiB/24h but refuses on incomplete writer proof and has no count cap for young generations.
- ARCH-ORDER/ARCH-SECURE: raw/event capture is an identity-bound pair, not two independent diagnostic files. Recovery, park and previous peek validate producer witness plus raw/event identities; naive copy-truncate/rename would invalidate it. Map consumers before choosing the bounded representation. No production change yet.

- Hub-approved alternative syscall evidence: 60.003 seconds, 20ms stat sampling observed **2,187 state inode changes/minute** (a lower bound because multiple replacements between samples collapse), with wrap-events growth 1,185,998 bytes and raw growth122,922 bytes. Two registry files changed once each. Main amplifier in this window is the append state publisher; registry churn is a separate redundant-write path, not observed at every event.
- Consumer exploration (read-only subagent): no full VT semantic checkpoint/restore exists. Visible-screen ANSI seeds omit hidden main/alt screen, modes, pen, margins, saved cursor, charset/tabstops and split parser/grapheme state. Bounded raw suffix eviction would silently change rendering. Exact capture park/recovery/previous-reader migration requires a certified segment/checkpoint representation. Sent scope options to hub (83dfc5b9-dcb6-4550-96f1-91f0054338af); awaiting its choice while developing the diagnostic design. Original requirements unchanged pending that decision.

- Temporary untracked Go overlay probe (no source mutation) used the real diagnostic writer with 1,000 distinct JSONL events: payload44,890 bytes, metadata1,087,924 bytes, exactly2,000 state publications; logical bytes-written/appended25.235x before APFS block/journal overhead. Probe uses existing fault checkpoints after actual metadata publications, marshals the observed disk states, and counts the payload returned by real Write. This is a code-path measurement, not a kernel syscall trace. Scratch overlay under pair579-probe-f2a4f2vd.
- Rotation blocker demonstrated through the production VerifyWriters/OSInspection seam with a read-only tracing processInspection: PID72769 (Zellij) advertises PAIR_RETENTION_PROTOCOL=1, but its PAIR_DATA_DIR points under removed birthrace-scratch-2453556978. Runtimes attempts EvalSymlinks of that irrelevant root before returning the protocol-aware row; failure becomes diagnostic writer inventory incomplete and blocks all rotation. The protocol row is later exempt from legacy-writer root comparison anyway. Proposed minimal correction: retain generation/environment fences, classify protocol participation before unnecessary root canonicalization; unknown legacy roots must still refuse. This is not authority to stop the leftover process.
