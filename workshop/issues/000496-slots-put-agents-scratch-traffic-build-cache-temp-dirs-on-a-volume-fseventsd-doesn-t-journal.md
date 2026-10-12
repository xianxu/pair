---
id: 000496
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '65a32d3cd72b43fc6da52772c01c2e5d67f977b2' # card fields mirrored from issue-cards; edit via sdlc
---

# slots: put agents' scratch traffic (build cache, temp dirs) on a volume fseventsd doesn't journal

## Problem

10-12: fseventsd (macOS's file-change journal, feeding Spotlight, Time Machine and file watchers) grew to gigabytes of memory and over 100% CPU under agent churn: Go's build cache (~86 GB; ~18,000 files changed per hour) and agent and test temp space (~11,000 per hour; test suites create and delete thousands of temp git repos). It slowed the machine and Couch; the operator restarted it by hand. File watching is tuned for human-scale change, not a fleet of agents.

## Spec

Keep the churn where nothing watches it:
- a dedicated APFS volume (e.g. `/Volumes/AgentScratch`, case-sensitive optional) with `.fseventsd/no_log` at its root and excluded from Spotlight (`mdutil -i off`);
- slots point `GOCACHE`, `TMPDIR` (and the test temp roots sdlc and pair use) at it, through the slot environment Couch/pair sets for a wrapper (or weave's compiled environment), never by editing the operator's shell profile;
- measure before and after (fseventsd CPU and memory, files changed per hour on the main volume) during a normal multi-slot workload;
- document setup and teardown in the atlas.
**Operator permission (the operator, 10-12):** every operation that changes this machine (creating or mounting a volume, moving or deleting caches or logs, changing launch settings, renicing processes) is proposed first, with the exact command, and runs only after the operator approves it, through the TL. Code changes and tests in the repo don't need that; anything outside the repo does.

## Done when

- The volume exists (created with the operator's approval), slots use it, and a multi-slot workload shows fseventsd's CPU and memory staying low, with the measurements in the Log.
- Setup and teardown documented; nothing outside the repo changed without the operator's approval.

## Plan

- [ ]

## Log

### 2026-10-12
