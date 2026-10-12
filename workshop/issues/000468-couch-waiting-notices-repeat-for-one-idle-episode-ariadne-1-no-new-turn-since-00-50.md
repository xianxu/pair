---
id: 000468
status: codecomplete
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'd709df983428d17bf61f7883d5eb4584a9e82d18' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T02:01:23-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "7bf8c643", done: "f6b3be74"}
actual_hours: 0.07
---

# couch: waiting notices repeat for one idle episode (ariadne:1, no new turn since 00:50)

## Problem

pair#451 promises one waiting notice per episode. ariadne:1 finished its last turn at 00:50 on 10-12 (parked by the TL until a Couch restart) and had no turn since, yet the hub got 'slot ariadne:1 waiting 10 min idle off its resting branch' at least four times (deliveries 19729e90…, d9e1c39b…, 94e5abd7…, 900d3997…, roughly every 30 minutes). Each is noise the TL has to fold.

## Spec

Find what re-arms the episode for a slot that stays idle: an incoming message that is delivered but doesn't start a turn, a wrapper heartbeat, a settle report, or a timer. An episode ends only on real activity (a turn, operator input, a branch change). Fix at the root; keep one notice per episode.

## Done when

- A test with a slot idle off its resting branch for an hour, with periodic wrapper reports, yields one notice.
- The repeat seen on ariadne:1 is explained in the Log.

## Plan

Quick flow, no durable plan.

- [x] Find what re-arms the episode (recorded scrollback of ariadne:1 and pair:3)
- [x] Key the episode on the judged wait (reason + summary), not on `LastActivity`
- [x] Test: an hour idle off the resting branch with output every 30 min gives one notice; a turn with new words gives a second

## Log

### 2026-10-12
- 2026-10-12: closed — Cause measured from ariadne:1 recorded scrollback: Claude Code update check prints every 30 min (01:10:26, 01:40:26), moving LastActivity and re-arming the episode. Fix keys the episode on the judged wait. New test TestWaitNoticeOutputWithoutATurnKeepsTheEpisode (an hour idle off resting, output every 30 min: one notice; new words: a second) fails with the fix reverted (reproduces 2 identical notices). make -k test (short TMPDIR, unsandboxed) exit 0.; review verdict: SHIP

- **Cause, measured** from ariadne:1's recorded scrollback (`scrollback-couch-15e934537eb183bb-claude.events.jsonl` and `.raw`): after its last turn at 00:50, the pane printed about 100–160 bytes at 01:10:26 and 01:40:26 (and 00:40:26 before the turn), exactly every 30 min. The bytes draw `Checking for updates` in Claude Code's footer and then erase it: its background update check. The wrapper counts any PTY output as activity (`peerDelivery.observeOutput` moves `lastActivity`, unsettles, then re-settles 3 s later). The watcher keyed an episode on `LastActivity`, so each check started a new episode, and 10 min later came another "waiting 10 min" notice: about one every 30 min, as the hub saw. pair:3 shows the same check (00:55) plus an idle auto-compaction at 01:17 (progress OSC working, `Compacting`).
- **Fix:** an episode is one judged wait. After a new `LastActivity`, the watcher re-reads the tail. If the reason and summary are the ones it already notified, it keeps the episode and sends nothing. A turn, the operator's input or a branch change ends the episode by changing what the slot waits on, or by making it not wait. Pair still only renders; nothing classifies the update check by its text. The #451 test that called a bare `Update` a new episode encoded the bug; a new episode now means new words on screen.
- Known limit: a real turn that ends on exactly the same last prose line as the notified one stays in the episode (no second notice). An idle compaction that changes the screen notifies once more.
- Close review advisories: (1) "a not-waiting verdict drops the episode". Kept on purpose: not-waiting means a branch went back to resting, a message is in flight, or the turn ended normally. A branch change has to end the episode (Spec), and a delivered message normally starts a turn that writes new words. (2) The test doesn't model the Settled flip around each burst. Quiet is under the threshold during the flip, so the watcher never judges it. (3) Atlas reflow fixed.

