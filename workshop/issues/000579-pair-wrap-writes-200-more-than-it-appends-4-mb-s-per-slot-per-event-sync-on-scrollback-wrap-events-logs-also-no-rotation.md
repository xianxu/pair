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

Proposed durable plan: [wrapper log IO](../plans/000579-wrapper-log-io-plan.md).
Pending hub scope approval: the plan separates raw size rotation into a faithful checkpoint/segment follow-up; original Done-when remains binding until explicitly revised.

- [ ] M1 — Coalesced diagnostic IO and authority-preserving raw/event batching.
- [ ] M2 — Safe bounded diagnostic generations, compatibility and qualification.

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

- 2026-10-15T22:18:28Z: verified acceptance receipt e774d4c0-5be6-4c77-8e0a-8e25ceeb3cd4; explicit post-landing measurement added below. This is not approval of the pending scope/retention design. Production code remains unchanged.


## Revisions

### 2026-10-15T22:18:28Z — operator adds post-landing live acceptance

Reason: hub receipt e774d4c0-5be6-4c77-8e0a-8e25ceeb3cd4 conveys the operator's explicit acceptance request.
Delta: after the fix lands, measure a live slot on the landed binary and record before/after process bytes-written versus actual raw+wrap-event bytes appended, amplification ratio, and diagnostic state inode-change rate in this issue's Log. Record exact binary/commit, PID/generation, paths, sample durations and output/activity context. Inode sampling is a lower bound when replacements occur between observations. Keep the existing before numbers (85.8x over10.006s and2187 observed state replacements over60.003s); do not substitute synthetic counts for the live check. The hub will separately sample whole-disk write rate; do not attribute that aggregate to this slot without its measurement. A staged/pre-merge smoke does not satisfy this additional post-landing check.

- [ ] After landing: record live before/after bytes-written/appended and diagnostic state inode-change rate in ## Log; send results to hub for its whole-disk comparison.


### 2026-10-15T22:23:51Z — #579/#582 scope split and append-protocol revision

Reason: hub's submitted scope decision8cb950d6-7819-48dd-9813-e10321f20ad8. #582 is filed unassigned on main with the witnessed raw/events checkpoint and consumer findings. Delta: #579 now owns only diagnostic write amplification and diagnostic size/count bounds, counted regression and live before/after acceptance. Raw capture rotation/retention moves to #582; raw/event producer and readers remain unchanged here. The original raw-rotation Done-when is discharged by this authorized split, not claimed implemented.

Current Done-when for #579:
- Wrapper diagnostics coalesce writes; each flush uses one small non-payload reservation write and one checkpoint publication instead of per-event intent+commit renames and base64 payload duplication. Healthy synthetic real-writer amplification<3x includes all reservation/metadata bytes; idle and unchanged registry opens write no metadata content.
- Diagnostic generations enforce64MiB size and current+3 count policy where evidence is certified; unknown proof stops growth, preserves uncertain legacy data and reports the block. Demonstrated protocol-aware removed-root rotation refusal is fixed without weakening legacy/unknown process protection.
- Legacy pending intent migration, compact reservation crash/partial/error behavior, replacement refusal, buffer/worker lifetime, rotation/retention and cleanup have deterministic red/green regressions; mandatory suite/build/reviews pass.
- After landing, record live bytes-written/appended and state inode-change rates against the preserved85.8x and2187/minute baseline, exact build/process/activity context, and report to hub. Hub owns aggregate whole-disk sampling.

The governing protocol/tasks and material killed-partial-write tradeoff are appended to the durable plan's latest Revision. This is the proposed executable design for hub approval, not implementation. Initial raw-batching steps are superseded by this narrower scope.


### 2026-10-15T22:25:16Z — proceed on the narrowed diagnostic scope

Verified hub receipt8f573644-79af-4c74-99b6-58677b6b5137 explicitly says Go on diagnostic write amplification/bounds and the narrow protocol-aware root resolution fix; raw/events rotation is unassigned #582. Proceed through the full-flow plan gate with the latest governing protocol revision. No raw/event production or reader changes. Post-landing live measurements remain outstanding acceptance work.


### 2026-10-15T22:28:30Z — plan-quality refinements PQ-1/PQ-2

The first plan-quality review correctly found shared adaptation writers could bypass a wrapper-only option. Governing plan now persists schema-v2 DiagnosticLimits and makes every managed writer enforce them; older strict decoders refuse rather than bypass. Limits can tighten but not widen on another handle. Named-function strategy table replaces the earlier prose test inventories as the current contract. No production code changed before the gate.
