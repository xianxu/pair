---
id: 000423
status: open
deps: []
github_issue:
created: 2026-10-09
updated: 2026-10-09
estimate_hours:
card_mirror: 'b8147a24559c7c4f4ba6596588e6765b758d265c' # card fields mirrored from issue-cards; edit via sdlc
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

## Done when

- The switcher subtest waits for the parked row (or inventory-ready state)
  before pressing Enter. It passes `-count=20` under load with no failure.

## Plan

- [ ]

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
