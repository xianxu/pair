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
- Inventory is deterministically withheld after the header paints; the shared
  pre-Enter predicate rejects that state, and a header-only mutation fails.
- Both origins retain resume/attach and registered-live assertions.

## Plan

- [x] Gate inventory in the existing switcher fixture and share the row predicate.
- [x] Prove header-only predicate red, restored predicate green; qualify 20 runs and race.
- [x] Run build and targeted qualification; reserve full suite for the recorded merge gate.

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

### 2026-10-14 — deterministic guard and verification

- TL approved the coverage/closure plan (receipt aea0f01a-7eb3-4f40-a648-b707e94eb84c).
- Configure the provider before Console.Run starts, so no ungated inventory
  request can race setup. The wrapper retains the production provider and
  returns on console lifetime cancellation; the failing mutation also proves
  cleanup exits while the barrier is still closed.
- Baseline both origins: 20 passes (45.254s). Barrier both origins: 20 passes
  (54.867s); natural host load was 29.77 before qualification and 13.10 near
  its completion. No artificial load was generated.
- Header-only mutation via Go overlay failed in 1.087s with exactly
  `Enter readiness accepted the header before inventory was released`.
  Restored predicate and both origins passed; focused race passed (4.944s).
  `make build` passed. Logs: /tmp/pair423-{baseline20,green20,red,race,build}.log.
- CI history: `gh run list --created '>=2026-10-11' --status failure --limit 1000`
  returned 23 failures, nine after #436's merge 80cc80afdb130a02bde57def3b537ec915dfab61
  (2026-10-11 09:54:02Z). Fetched failed logs for those nine plus the adjacent
  pre-merge run: no mention of this test. This is **no observed CI recurrence**,
  not proof of absence: couch-zellij-conformance selects live tests and excludes
  this cold-resume test; local-gates-guard is metadata-only. The known original
  report was local. CI list and fetched logs are /tmp/pair423-ci-*.
- Full `make -k test` is queued behind the machine's two-suite gate; all
  prerequisites and Go targets will be allowed to finish before closure.

### 2026-10-14 — verification sequence revision

- TL approved cancelling only the still-queued pre-close suite (receipt
  d0e3d4f9-4322-4a31-a834-5a8529e2b0d9), closing on targeted evidence and
  using the merge gate's recorded suite as the one full run, without a bypass.
  This supersedes the pre-close suite sequence above and avoids duplicate
  full runs for this test-only guard. Close/refusal must retain the queued run
  if a full suite is required there.
