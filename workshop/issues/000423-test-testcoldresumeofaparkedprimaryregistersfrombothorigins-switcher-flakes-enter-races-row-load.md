---
id: 000423
status: working
deps: []
github_issue:
created: 2026-10-09
updated: 2026-10-14
estimate_hours:
card_mirror: '51adfe86b94a675bd5469606e82edb0a97e87587' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-14T23:30:55-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "927c867d", done: "8abd4617"}
---

# test: TestColdResumeOfAParkedPrimaryRegistersFromBothOrigins/switcher flakes, Enter races row load

## Problem

`TestColdResumeOfAParkedPrimaryRegistersFromBothOrigins/switcher`
(`cmd/internal/couchcmd/cold_resume_origins_test.go`) fails intermittently with
`no resume and attach` after a 10s wait. A passing run takes about 0.4s.

The rig waits only until the host output contains `threads` (the switcher
header), then writes `\r`. The header renders before the inventory arrives, with
the info notice `thread inventory unavailable` (`couchtty/menu.go:485`), so
Enter can land on an empty list. The frame then shows `error: no selection`, and
the parked row (`▸ repo … parked`) renders only afterwards.

## Spec

The root race was fixed in #436 (62a91a14b1687aa5cddf846f33bb884c0b25a2ab):
wait for the rendered parked row, not the switcher header. Preserve that fix
and pin its ordering deterministically in the existing cold-resume fixture.
Gate the real actionable provider with a cancellation-aware channel, observe
the header with inventory still withheld, and assert that the same readiness
predicate used before Enter rejects that frame. Release inventory, wait for
the row, then keep the existing resume/attach and registered-live assertions.
No production behavior, sleeps, deadline increases, or weaker assertions.

ARCH-DRY/PURE: share one readiness predicate between negative and positive
checks, wrap the existing provider rather than duplicate inventory behavior.
ARCH-ORDER: explicitly force header -> withheld inventory -> row -> Enter;
a header-only predicate must fail before releasing inventory.
ARCH-MOCK: retain the existing stateful runner/artifacts and real console.
ARCH-CONSTRAINTS: bounded sequential qualification (20 runs), existing test
bounds, no synthetic machine load. ARCH-SECURE: fixture-only data and channels.
ARCH-FUNERAL: cancellation and test cleanup release blocked provider work;
no durable runtime artifacts. ARCH-PURPOSE: prove the previously flaky ordering
and both origins still register, then close the stale issue.

## Done when

- The switcher subtest waits for the parked row (or inventory-ready state)
  before pressing Enter. It passes `-count=20` under load with no failure.

## Plan

- [ ] Gate inventory in the existing switcher fixture and share the row predicate.
- [ ] Prove header-only predicate red, restored predicate green; qualify 20 runs and race.
- [ ] Run build and full make -k test, review, merge main, land on SHIP.

## Log

### 2026-10-09

- Found while closing pair#422 (pair:2), in a clean-env `go test ./...`
  (`env -i`, load average 6–7). Filed at TL ops:0's request.
- Branch 000422 (no switcher or inventory changes):
  `go test ./cmd/internal/couchcmd -run '…/switcher' -count=6` gave 3 FAIL
  (10.15s, 10.18s, 10.18s) and 3 PASS (0.41–0.44s).
- Clean origin/main worktree after `make build`: 1 of 1 PASS (1.3s).
- The `remote` subtest passed in every run.
- Failure frame excerpt: `threads` / `thread inventory unavailable` /
  `(no match)`, then `error: no selection`, then the row
  `▸ repo /…/TestColdResume… parked`.

### 2026-10-14 — investigation and plan

- Main already contains #436's exact row-wait fix; #423 remained open.
  The old failure frame and current menu reducer agree: header publication
  precedes asynchronous inventory; Enter on an empty selection is refused.
- Prefer a controlled provider barrier over an unbounded load soak or changing
  production input semantics. The barrier exercises the actual console and
  retains the production provider under the wrapper.

## Revisions

- 2026-10-14: scope is deterministic coverage and qualification of the landed
  #436 fix, not reimplementing it; discovered by inspecting main and git history.
