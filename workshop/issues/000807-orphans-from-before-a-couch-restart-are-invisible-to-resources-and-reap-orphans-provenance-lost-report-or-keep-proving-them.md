---
id: 000807
status: working
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'a17d0a5a392977c03acd24e5773b92e405c6152a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T13:21:29-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# Orphans from before a Couch restart are invisible to --resources and --reap-orphans (provenance lost); report or keep proving them

## Problem

## Spec

pair#792 (landed 10-22 09:30) reaps proven orphans of a live slot's previous incarnation. Live retry by ops-echo:0 at
~10:56, on Couch built at c95f2421 (contains #792): `couch --reap-orphans` preview reports "none observed", and
`couch --resources` lists **no proven and no unattributed** processes, yet echo:2's 10-21 leftovers are alive:
zellij server 66561 (ppid 1, session 1-392), `pair wrap` 66575 and `pair term` 66576 under it, and Claude 66695 under
66575, started as an echo:2 "This slot restarted fresh" relaunch. On 10-21, before several Couch restarts,
`--resources` did list exactly these as proven orphans of echo:2.

So after a Couch restart, the previous incarnation's provenance is gone and its orphans become **invisible**: not
proven (correctly not reaped) but also not even reported as unattributed, so no one knows they're there.

Fix: either persist the bounded provenance needed to keep proving them across a Couch restart (the existing
identity fence, PID plus start time plus slot and tag), or at minimum report such processes as **unattributed**
(Couch-shaped trees: zellij servers in Couch's socket dir, pair wrap/term, agents with Couch markers) so a TL sees
them and the operator can decide. Never reap unproven processes automatically.

## Done when

- A fixture with a previous-incarnation tree surviving a Couch restart is either still proven (and reapable through
  the #792 path) or listed as unattributed, never invisible (test).

## Plan

- [ ]

## Log

### 2026-10-22


- Authenticated TL dispatch 988fcfe0-745c-418c-9482-00b3eb887790: start this item independently from main while #791's recorded check queues; poll at natural pauses and land #791 first. Claimed and started planning on main 7b9440e8c92663f8ee189f54a1260e3043a78f07. The substrate refresh generated an unrelated merge-check workflow change, preserved in the narrow stash `#807: preserve unrelated weave refresh workflow output`; it is outside this item's scope.
- Read-only root-cause investigation, corroborated by restart_orphan_discovery: message_service.go constructs an empty ResourceState after restart; resources_persist.go restores only aggregate history, explicitly never ownership. Current-slot inventory cannot reconstruct previous incarnation ancestry. UnattributedShape requires PPID 1, and ClassifyUnattributed requires known RSS above 256 MiB. A small old Zellij root and its pair/agent descendants can therefore all disappear from reports.
- Executable diagnostic model (not a native capture): overlay `/Volumes/TestScratch/tmp/pair807-repro-gz1zsunr/overlay.json`, `go test -overlay … ./cmd/internal/couchcore -run '^TestDiagnostic807RestartVisibility$' -count=1 -v`, PASS 0.268s. Modeled the reported four-PID topology, with small server and 1 GiB agent. Output: `owned=4 remembered=4 after_restart=0 unattributed=0`. This confirms the visibility gap without changing production source or signalling processes; actual native RSS/environment remain unmeasured.
- Proposed direction, not yet approved: bounded report-only discovery of unowned Couch-shaped trees into the existing Unattributed section, including small/unknown RSS. Keep report evidence out of ownership provenance and all reap authority. Persisting authoritative provenance is larger and still needs a fallback when writes fail; restoring aggregate history as authority contradicts its contract (ARCH-DRY / ARCH-PURE). Before proposing exact selectors, finish checking native socket namespace spelling and marker grammar. Do not widen UnattributedCandidate because the nvim destructive fallback also consumes it.
- 13:28 checkpoint: TL message f0f48a53-7be6-4572-bbd6-080b43fac429 requests a concrete visible-pane tab-wrap resize fixture and last-resize timing for #791 before it lands. Pausing this item's plan to handle that lead. No production edits or implementation gate for #807. Next: return after #791's bounded investigation, finish the durable report-only plan and send it to the hub before code. #791 check remains session 55235, lease-402578803, tested code ecc76e29ec0cc2d0ca5252453ebe19883bdeca86, alive/queued position 11 at the last poll; no merge runner active.
