---
id: '000069'
status: done
started: 2026-06-24T14:09:23-07:00
created: 2026-06-24
updated: 2026-06-24
actual_hours: N/A
---

# Fix cmux workspace title update: poller respawn + stale ownership

## Problem

The cmux workspace title (the activity-heat prefix `🔴/🟠/🟡/🔵 pair-<tag>`
maintained by `bin/pair-cmux-title.sh`) stopped updating. Two independent
defects, surfaced while debugging a live session (tag 211) whose title was
frozen at `🔴 ♋-♋`:

1. **Poller death is not reliably healed.** The poller is a long-lived
   background process spawned only at `pair` create/attach/restart. A host
   sleep / reboot / SIGKILL kills it mid-session (all pidfiles were found
   pointing at dead PIDs), and nothing respawns it until the next entry. Worse,
   the single-instance guard used a bare `kill -0 $old_pid`: after a reboot the
   kernel can recycle the dead poller's PID onto an unrelated live process, so
   the guard reads "still alive" and **suppresses the respawn even across a pair
   restart**.

2. **Stale workspace ownership freezes the title.** `cmux_rename_workspace`
   deferred whenever a *different* tag's `pair-<owner>` zellij session existed
   *anywhere*. When a cmux workspace is reused by a new tag while the old owner
   lives on in a *different* workspace, the owner file (`cmux-owner-<wsid>`)
   goes stale and the new occupant defers to a ghost — permanently. This is why
   tag 211 showed `🔴 ♋-♋` (= `pair-pair`): a stale `pair` owner blocked it.

(Note: cmux itself is healthy — `cmux ping` → PONG. The broken-pipe seen from
the agent Bash tool is sandbox-only; the poller runs in the real pane shell.)
