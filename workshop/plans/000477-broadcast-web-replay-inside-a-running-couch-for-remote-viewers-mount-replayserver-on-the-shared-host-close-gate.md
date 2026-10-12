---
gate: boundary-review
issue: 477
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T06:04:03-07:00"
      dispatched: "2026-10-12T05:58:03-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: ReplayShares mount-completes-after-Close unmount branch is unexercised; reverting it keeps tests green
          detail: shares.go:120-127. Disabling m.Unmount() in the closed branch passes all share tests (-count=3); CloseWhileOpening only hits the abandoned-open path via a 20ms sleep. Add an ordering seam (blockable mount/probe) and test probe-success -> Close -> release, asserting Opens==Closes.
          family: untested-interleaving-branch
          round: 1
        - id: BR-2
          severity: Important
          title: Agent-triggered replay shares publish recordings via the tunnel with no operator-visible indicator or listing
          detail: query link lets any agent open a public tunnel serving a recording until Couch exits; neither --broadcast-list nor the status bar shows active shares. Add a ready-share count to BroadcastStatus/status cell (unshare can stay a follow-up).
          family: public-exposure-without-operator-visibility
          round: 1
        - id: BR-3
          severity: Important
          title: atlas/webcast-format.md agent-queries section still says the query CLI needs no Couch and omits the link verb
          detail: 'Lines 423-433: add a link [--at T] row, note it asks the running Couch, and replace "Main is the CLI" with query.Run/Linker.'
          family: atlas-surface-not-updated
          round: 1
        - id: BR-4
          severity: Minor
          title: query link appends a tunnel-off note to the URL on stdout, breaking agents that read stdout as a URL
          detail: replay.go:392-394; put the note on stderr as runReplay does.
          family: machine-output-contract
          round: 1
        - id: BR-5
          severity: Minor
          title: ShareStatus.Local paths in runReplay and replayLinker have no couchcmd test
          family: missing-branch-test
          round: 1
        - id: BR-6
          severity: Minor
          title: query Usage link line is overlong and runs into the existing "Times are ms" sentence
          family: usage-text-formatting
          round: 1
        - id: BR-7
          severity: Minor
          title: runReplay and replayLinker each classify the same three fallback sentinels; extract a shared predicate
          family: duplicated-error-classification
          round: 1
        - id: BR-8
          severity: Minor
          title: query.Main is now used only by tests; production calls query.Run
          family: dead-production-entrypoint
          round: 1
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T06:14:18-07:00"
      dispatched: "2026-10-12T06:07:56-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Reverted m.Unmount() in shares.go:154; TestReplaySharesMountCompletingAfterClose fails 3/3 (Opens:1 Closes:0); green restored.
          round: 2
        - id: BR-2
          disposition: addressed
          note: broadcast-status carries Replays, --broadcast-list prints them, pinned in TestReplayShareThroughTheBroker; notice half unpinned (raised separately).
          round: 2
        - id: BR-3
          disposition: addressed
          note: atlas/webcast-format.md:423-432 now lists link, says it asks the running Couch, names Run/Linker.
          round: 2
        - id: BR-4
          disposition: addressed
          note: replay.go:265-267 writes tunnelOffNote to stderr; TestReplayLinkTunnelOff asserts stdout is exactly the URL.
          round: 2
        - id: BR-5
          disposition: addressed
          note: TestReplayLinkTunnelOff covers Local for both runReplay and replayLinker.
          round: 2
        - id: BR-6
          disposition: addressed
          note: query/cli.go Usage reflowed; link sentence separated from the Times sentence.
          round: 2
        - id: BR-7
          disposition: addressed
          note: cantShare predicate at replay.go:183 used by both callers.
          round: 2
        - id: BR-8
          disposition: addressed
          note: query.Main removed; tests call Run(argv, nil, ...), nil linker refuses.
          round: 2
      findings:
        - id: BR-9
          severity: Important
          title: README not updated for --broadcast-list shared-replays line, replays JSON key, and link in the query prose
          detail: 'README.md:400 and 867-870 omit the new replays output; 872 still says dom|events|notes. 2nd in family: rule is every new user-visible surface (verb, flag, output line, JSON key) is swept across README synopsis+prose, atlas, SKILL.md and Usage in the same commit; grep each surface name across all five.'
          family: atlas-surface-not-updated
          round: 2
        - id: BR-10
          severity: Minor
          title: Share-ready operator notice (console_webcast.go:258-261) has no test; disabling it keeps tests green
          detail: '2nd in family: each behavior a fix adds needs its own failing-without-it test; round 1 added listing and notice but pinned only the listing.'
          family: missing-branch-test
          round: 2
        - id: BR-11
          severity: Minor
          title: Share-ready notice says "through the broadcast tunnel" even when the share is Local (tunnel off)
          detail: console_webcast.go:259; branch on ShareStatus.Local as both CLIs do.
          family: operator-message-accuracy
          round: 2
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 3
      timestamp: "2026-10-12T06:19:47-07:00"
      dispatched: "2026-10-12T06:16:54-07:00"
      agent: claude
      dispose:
        - id: BR-9
          disposition: addressed
          note: README.md:400-401 synopsis and 868-882 prose now name the replays line, the replays JSON key and link (dom|events|notes|link, stdout-only URL); help run.go:1315 and atlas/broadcast.md:431 also swept.
          round: 3
        - id: BR-10
          disposition: addressed
          note: TestReplayShareReadyIsANotice (console_webcast_test.go:299) goes red when setNotice is removed; verified by revert in the review checkout.
          round: 3
        - id: BR-11
          disposition: addressed
          note: console_webcast.go:258-262 branches on ShareStatus.Local passed via OnReady; the LocalOnly test case goes red when the branch is disabled.
          round: 3
      boundary: M1
      recipe: milestone-review
      isolation: fenced
      reviewed: 7f856ae6f6e73f1c6c8e0478d2be8d3420566801
      blocked: false
    - "n": 4
      timestamp: "2026-10-12T06:53:53-07:00"
      dispatched: "2026-10-12T06:51:30-07:00"
      agent: claude
      findings:
        - id: BR-12
          severity: Important
          title: Replay note card is positioned only on a frame; a tap while paused shows it at the box's top-left or at the previous pin
          detail: 'replay.js:446-461 sets card text and visibility in the pinLayer tap handler, but left/top are set only inside draw(player), which runs from deps.frame(). replaySource.draw schedules frames only while playing, after a seek, or on a bar change, so on a paused replay (how a t= link opens) the card stays unplaced until the next frame. Fix: call a placeCard helper from the tap handler too, or request a frame. Make the live test assert the card sits next to the tapped pin.'
          family: ui-event-without-redraw
          round: 4
        - id: BR-13
          severity: Important
          title: atlas/webcast-format.md:417 still describes the drawMarks path M2 removed
          detail: 'This is the 3rd finding in the family. Rule: when a milestone removes or renames a symbol, or completes work the atlas describes as today or to replace, search atlas/ and README.md for the old symbol names and the issue number, and rewrite every hit in the same commit (for M2: drawMarks, STROKE_FADE, replay-layer, replay-strokes, and 477 converges). The passage should also note that note placement now follows noteBox (region first).'
          family: atlas-surface-not-updated
          round: 4
        - id: BR-14
          severity: Minor
          title: Window resize while paused leaves replay strokes stretched and pins misplaced until the next frame
          detail: The resize handler runs layout() and drawPointer() but not the marks. The old SVG layer had the same gap.
          family: ui-event-without-redraw
          round: 4
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 5
      timestamp: "2026-10-12T07:00:57-07:00"
      dispatched: "2026-10-12T06:58:39-07:00"
      agent: claude
      dispose:
        - id: BR-12
          disposition: addressed
          note: 'replay.js:455-461 now calls placeCard from the pinLayer tap handler; replay_live_test.go measures the card in the same step as the click (dx 0, dy 4), which fails if left/top are unset. Read, not run: Chrome could not start here.'
          round: 5
        - id: BR-13
          disposition: addressed
          note: atlas/webcast-format.md:417 rewritten (replayMarks, noteBox region first); atlas/webcast.md:239 updated. A grep of atlas, README and cmd finds no drawMarks, STROKE_FADE, replay-layer, replay-strokes, PIN_ROOM or "converges".
          round: 5
        - id: BR-14
          disposition: addressed
          note: replay.js:500 redraws the marks on the frame after a resize; the live test checks the canvas size against the box after exactly two frames, with the replay paused.
          round: 5
      findings:
        - id: BR-15
          severity: Minor
          title: replayMarks reads devicePixelRatio once, at mount, so the stroke canvas goes stale after a zoom or a move to another display
          detail: 'replay.js:482 passes win.devicePixelRatio as a fixed value; the live viewer reads it on every draw (annotate.js:484). Fix: pass a getter, or read it inside draw.'
          family: captured-environment-value
          round: 5
      boundary: M2
      recipe: milestone-review
      isolation: fenced
      reviewed: 70309982bc57dc87c258ea217dc5d032f6e027b6
      blocked: false
    - "n": 6
      timestamp: "2026-10-12T07:03:32-07:00"
      dispatched: "2026-10-12T07:01:19-07:00"
      agent: claude
      dispose:
        - id: BR-15
          disposition: not-addressed
          note: 'At head, replay.js:482 still passes win.devicePixelRatio once, at mount, while annotate.js:484 reads it on every paint; no later commit touches it. Minor and non-blocking: pass a getter, or read it inside draw.'
          round: 6
      recipe: milestone-review
      isolation: fenced
      reviewed: 9b9a11d67fefd9568ef15fe89bebb4428e2ddf05
      blocked: false
---

# Gate ledger — pair#477 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T06:04:03-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `untested-interleaving-branch` ReplayShares mount-completes-after-Close unmount branch is unexercised; reverting it keeps tests green
  shares.go:120-127. Disabling m.Unmount() in the closed branch passes all share tests (-count=3); CloseWhileOpening only hits the abandoned-open path via a 20ms sleep. Add an ordering seam (blockable mount/probe) and test probe-success -> Close -> release, asserting Opens==Closes.
- **BR-2** [Important] `public-exposure-without-operator-visibility` Agent-triggered replay shares publish recordings via the tunnel with no operator-visible indicator or listing
  query link lets any agent open a public tunnel serving a recording until Couch exits; neither --broadcast-list nor the status bar shows active shares. Add a ready-share count to BroadcastStatus/status cell (unshare can stay a follow-up).
- **BR-3** [Important] `atlas-surface-not-updated` atlas/webcast-format.md agent-queries section still says the query CLI needs no Couch and omits the link verb
  Lines 423-433: add a link [--at T] row, note it asks the running Couch, and replace "Main is the CLI" with query.Run/Linker.
- **BR-4** [Minor] `machine-output-contract` query link appends a tunnel-off note to the URL on stdout, breaking agents that read stdout as a URL
  replay.go:392-394; put the note on stderr as runReplay does.
- **BR-5** [Minor] `missing-branch-test` ShareStatus.Local paths in runReplay and replayLinker have no couchcmd test
- **BR-6** [Minor] `usage-text-formatting` query Usage link line is overlong and runs into the existing "Times are ms" sentence
- **BR-7** [Minor] `duplicated-error-classification` runReplay and replayLinker each classify the same three fallback sentinels; extract a shared predicate
- **BR-8** [Minor] `dead-production-entrypoint` query.Main is now used only by tests; production calls query.Run

## Round 2 — 2026-10-12T06:14:18-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Reverted m.Unmount() in shares.go:154; TestReplaySharesMountCompletingAfterClose fails 3/3 (Opens:1 Closes:0); green restored.
- BR-2 — addressed — broadcast-status carries Replays, --broadcast-list prints them, pinned in TestReplayShareThroughTheBroker; notice half unpinned (raised separately).
- BR-3 — addressed — atlas/webcast-format.md:423-432 now lists link, says it asks the running Couch, names Run/Linker.
- BR-4 — addressed — replay.go:265-267 writes tunnelOffNote to stderr; TestReplayLinkTunnelOff asserts stdout is exactly the URL.
- BR-5 — addressed — TestReplayLinkTunnelOff covers Local for both runReplay and replayLinker.
- BR-6 — addressed — query/cli.go Usage reflowed; link sentence separated from the Times sentence.
- BR-7 — addressed — cantShare predicate at replay.go:183 used by both callers.
- BR-8 — addressed — query.Main removed; tests call Run(argv, nil, ...), nil linker refuses.

### Raised

- **BR-9** [Important] `atlas-surface-not-updated` README not updated for --broadcast-list shared-replays line, replays JSON key, and link in the query prose
  README.md:400 and 867-870 omit the new replays output; 872 still says dom|events|notes. 2nd in family: rule is every new user-visible surface (verb, flag, output line, JSON key) is swept across README synopsis+prose, atlas, SKILL.md and Usage in the same commit; grep each surface name across all five.
- **BR-10** [Minor] `missing-branch-test` Share-ready operator notice (console_webcast.go:258-261) has no test; disabling it keeps tests green
  2nd in family: each behavior a fix adds needs its own failing-without-it test; round 1 added listing and notice but pinned only the listing.
- **BR-11** [Minor] `operator-message-accuracy` Share-ready notice says "through the broadcast tunnel" even when the share is Local (tunnel off)
  console_webcast.go:259; branch on ShareStatus.Local as both CLIs do.

## Round 3 — 2026-10-12T06:19:47-07:00 (claude) — passed

### Disposed

- BR-9 — addressed — README.md:400-401 synopsis and 868-882 prose now name the replays line, the replays JSON key and link (dom|events|notes|link, stdout-only URL); help run.go:1315 and atlas/broadcast.md:431 also swept.
- BR-10 — addressed — TestReplayShareReadyIsANotice (console_webcast_test.go:299) goes red when setNotice is removed; verified by revert in the review checkout.
- BR-11 — addressed — console_webcast.go:258-262 branches on ShareStatus.Local passed via OnReady; the LocalOnly test case goes red when the branch is disabled.

## Round 4 — 2026-10-12T06:53:53-07:00 (claude) — BLOCKED

### Raised

- **BR-12** [Important] `ui-event-without-redraw` Replay note card is positioned only on a frame; a tap while paused shows it at the box's top-left or at the previous pin
  replay.js:446-461 sets card text and visibility in the pinLayer tap handler, but left/top are set only inside draw(player), which runs from deps.frame(). replaySource.draw schedules frames only while playing, after a seek, or on a bar change, so on a paused replay (how a t= link opens) the card stays unplaced until the next frame. Fix: call a placeCard helper from the tap handler too, or request a frame. Make the live test assert the card sits next to the tapped pin.
- **BR-13** [Important] `atlas-surface-not-updated` atlas/webcast-format.md:417 still describes the drawMarks path M2 removed
  This is the 3rd finding in the family. Rule: when a milestone removes or renames a symbol, or completes work the atlas describes as today or to replace, search atlas/ and README.md for the old symbol names and the issue number, and rewrite every hit in the same commit (for M2: drawMarks, STROKE_FADE, replay-layer, replay-strokes, and 477 converges). The passage should also note that note placement now follows noteBox (region first).
- **BR-14** [Minor] `ui-event-without-redraw` Window resize while paused leaves replay strokes stretched and pins misplaced until the next frame
  The resize handler runs layout() and drawPointer() but not the marks. The old SVG layer had the same gap.

## Round 5 — 2026-10-12T07:00:57-07:00 (claude) — passed

### Disposed

- BR-12 — addressed — replay.js:455-461 now calls placeCard from the pinLayer tap handler; replay_live_test.go measures the card in the same step as the click (dx 0, dy 4), which fails if left/top are unset. Read, not run: Chrome could not start here.
- BR-13 — addressed — atlas/webcast-format.md:417 rewritten (replayMarks, noteBox region first); atlas/webcast.md:239 updated. A grep of atlas, README and cmd finds no drawMarks, STROKE_FADE, replay-layer, replay-strokes, PIN_ROOM or "converges".
- BR-14 — addressed — replay.js:500 redraws the marks on the frame after a resize; the live test checks the canvas size against the box after exactly two frames, with the replay paused.

### Raised

- **BR-15** [Minor] `captured-environment-value` replayMarks reads devicePixelRatio once, at mount, so the stroke canvas goes stale after a zoom or a move to another display
  replay.js:482 passes win.devicePixelRatio as a fixed value; the live viewer reads it on every draw (annotate.js:484). Fix: pass a getter, or read it inside draw.

## Round 6 — 2026-10-12T07:03:32-07:00 (claude) — passed

### Disposed

- BR-15 — not-addressed — At head, replay.js:482 still passes win.devicePixelRatio once, at mount, while annotate.js:484 reads it on every paint; no later commit touches it. Minor and non-blocking: pass a getter, or read it inside draw.

## Open findings

- **BR-15** [Minor] `captured-environment-value` replayMarks reads devicePixelRatio once, at mount, so the stroke canvas goes stale after a zoom or a move to another display
