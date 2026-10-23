---
id: 000679
status: open
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: '9f34976b39b1d1bf4743ed903288885c4b2998f8' # card fields mirrored from issue-cards; edit via sdlc
---

# flake: changelogcmd distiller FIFO test hangs 20m in os.ReadFile

## Problem

## Spec

TestManagedDistillerLeasesBeforeReadingWithoutTouchingUse (cmd/internal/changelogcmd/retention_test.go) hung until the 20m package timeout during the #674 merge integrated suite (sdlc merge, integration b153e18c, 2026-10-18). Unrelated to #674 (test-only Python fixture change).

Stack: the distiller goroutine is blocked in a raw syscall read inside os.ReadFile(cleanedPath) at changelogcmd.go:100, on the FIFO the test made with Mkfifo; the test goroutine waits on done at retention_test.go:53. The test opens the FIFO O_WRONLY and closes it immediately after seeing the lease, so the reader should see EOF; on darwin it instead stayed blocked, so the open/close rendezvous with the reader open/read is racy.

Evidence: failed log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/5bf0374293571563365a48c27b7f733c.log (copy kept in the pair:2 scratchpad as flake-changelogcmd-674.log). Recorded sdlc-local/suite on b153e18c exit 2 in 27m3s.

## Done when

- The test cannot hang: the FIFO handoff is deterministic (for example the writer writes a byte or holds the write end open until the reader has opened), and a bounded wait replaces the bare done receive.
- 50 consecutive runs of the test pass under -count.

## Plan

- [ ]

## Log

### 2026-10-18


### 2026-10-22 — recurrence in #739 recorded verification

On exact pushed cc7892568e0e91d4cc74516729c5d2d82e8924f0, full recorded attempt7c7510bb47fa2dbaa737a27a12e3a9fc timed out in TestManagedDistillerLeasesBeforeReadingWithoutTouchingUse after20m. The stack matches this issue: test waits on done at retention_test.go:53; distiller blocks in os.ReadFile through changelogcmd.go:100. Remaining packages/scripts passed. Complete suite exit2 after26m13s; failed log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/7c7510bb47fa2dbaa737a27a12e3a9fc.log, SHA2565a13e9973c2201b82f217132c031fa46ea75417479152fb90010ead425a4c130.

No changelogcmd diff versus main. One unchanged focused repeat on that same head: go test ./cmd/internal/changelogcmd -run '^TestManagedDistillerLeasesBeforeReadingWithoutTouchingUse$' -count=1 -timeout=60s -v passed package0.268s/test0.10s. This is focused positive evidence, not a full-suite pass or a registered-flake bypass. SDLC classified the original package timeout as unattributable-failure, so named-test resume does not explain that saved result. Hub asked for the supported next full verification step; no timeout/production fix folded into #739. No process killed; the natural timeout released the suite token.
