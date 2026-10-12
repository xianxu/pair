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

## Open findings

(none — every finding has been disposed)
