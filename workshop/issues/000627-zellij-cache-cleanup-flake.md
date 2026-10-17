---
id: 000627
status: open
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: 'e5d89d56424ea85d73df3d7de09cb901a8942677' # card fields mirrored from issue-cards; edit via sdlc
---

# Live shortcut fixture races Zellij cache removal during TempDir cleanup

## Problem

First observed 2026-10-17T00:28:29Z on pair#604 PR #381 integrated head f8e3e1c7970f0ab6505478e74ec823007d2c3a5e, GitHub Actions run 38986306428, conformance job 117258474451. TestAgentShortcutInputConformanceLive passed every input subtest, then failed during Go TempDir cleanup: unlinkat .../002/Library/Caches/org.Zellij-Contributors.Zellij: directory not empty.

Failed log: /tmp/pair604-integrated-conformance-failure.log. Durable CI evidence: https://github.com/xianxu/pair/actions/runs/38986306428/job/117258474451 . Previous PR-head conformance run 38983105918 passed unchanged clipboard implementation. No repeat of the failed job yet.

## Spec

Investigate and eliminate the isolated real-Zellij fixture cleanup race. TestAgentShortcutInputConformanceLive installs a private HOME and registers fixture.Close, currently ignoring its error. A surviving Zellij writer racing TempDir removal is a hypothesis, not measured causality. Preserve input assertions; do not hide cleanup errors or increase timeouts without evidence. Independent from #604 clipboard behavior; no test fix folded into it.

## Done when

- A reproducible cleanup regression identifies the resource still writing and passes after the fixture owns its teardown.
- Repeated isolated/headless runs retain all shortcut assertions and clean their private cache reliably.

## Plan

- [ ]

## Log

All 13 shortcut input subtests passed; other live launcher/core tests passed before this teardown failure. The landing CI gate correctly refused #604. No retry or bypass has been used for this failure.
