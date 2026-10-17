---
id: 000630
status: open
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: 'e0c580ce0a5cf8e009c17707366024cf7ed21493' # card fields mirrored from issue-cards; edit via sdlc
---

# couch session-sync: carry tl-lead's stale-section list to the TL when no event would show it

## Problem

## Spec

Drafted by ariadne:4 (from ariadne#414), filed by the TL.

ariadne#414 makes tl-lead stamp each status-file section and adds `tl-lead status check FILE`, which lists stale sections: Threads and Slots older than 2 h or than the newest landing; Quota older than 12 h; Needs you when unstamped. The TL sees that list only when it runs a tl-lead status verb, which happens on an event. In a quiet stretch, or while the operator holds the lock and writes are queued, nothing prompts it. That is how Threads went 20 h stale on 10-16.

For a registered session-sync file whose owner is a hub, Couch runs `tl-lead status check FILE` at most every 30 min while the file has stale lines and the owner is idle. It sends the output to the owner as one message, and none when the output is fresh. It skips this while the operator holds the lock, then sends once after their submit. It never writes the file.

## Done when

- A fixture file with a stale Threads produces one message to its owner per interval, and none when fresh.
- A held lock suppresses it.
- tl-lead missing or failing produces no message and logs the reason once.

## Plan

- [ ]

## Log

### 2026-10-16
