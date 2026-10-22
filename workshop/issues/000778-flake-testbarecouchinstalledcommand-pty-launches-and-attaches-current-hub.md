---
id: 000778
status: open
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'bc10c44b609b2cacd1e67793faf052bcaa5c50da' # card fields mirrored from issue-cards; edit via sdlc
---

# flake: TestBareCouchInstalledCommand/pty_launches_and_attaches_current_hub

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/couch.TestBareCouchInstalledCommand/pty_launches_and_attaches_current_hub: Exact source3279e1e9ec6ccbddba0b50193f5c8f72c11f4b22: train238 suite lease-1293856745 failed cmd/couch TestBareCouchInstalledCommand/pty_launches_and_attaches_current_hub after15s with live/contested terminal 📁1-1; log .git/sdlc/test-stats/inflight/de3d7e12cd48155a874dbd8db94ac3f6/run.log. On the unchanged source, env -u PAIR_TEST_BINARY go test ./cmd/couch -run ^TestBareCouchInstalledCommand$ -count=1 -timeout=90s -v passed11.305s (PTY2.12s). This fixture has private socket roots but fixed terminal name; SessionServers filters only session basename, a lead not yet a forced reproduction. Production scribe is unchanged. Track this intermittent fixture refusal separately from #768.

## Spec

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.

## Plan

- [ ]

## Log

### 2026-10-21
