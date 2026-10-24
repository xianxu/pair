---
id: 000864
status: wontfix
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: '781966f55e7a5483bb351831d108eecd85fca51b' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T14:53:16-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flake_tests: [github.com/xianxu/pair/cmd/internal/couchcmd.TestHubStartupLaterFailureLeavesFirstUsable]
---

# flake: TestHubStartupLaterFailureLeavesFirstUsable

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/couchcmd.TestHubStartupLaterFailureLeavesFirstUsable: failed in attempt 4fed2c456da3773658b2bd4328454d6a, passed on the same head in attempt 5bafd7c76410f108c1e6ee72e3e78952

## Spec

Retire the stale registered failure after proving the existing landed fixes cover both recorded variants. No new product behavior or timeout increase is justified: the Oct22 visibility failure required a 45-cell ASCII prefix from chrome capped at 40 cells, and #812 already replaced that oracle with the real attention panel. The original Oct16 nil-refusal registration predates #697's path-scoped failure injection. Verify both startup orderings and the publication/current-screen helper before close unregisters this test. ARCH-PURPOSE: distinguish recorded failures by pinned source rather than patching a shared test name; ARCH-DRY: reuse the existing production-path regressions.

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.

## Plan

- [ ] Confirm pinned failure provenance, existing fix ancestry and both startup-order regressions.
- [ ] Run targeted startup, attention-publication and delta-screen regressions sequentially; no stress or timeout changes.
- [ ] Record the existing-fix proof, run required recorded pre-close and inventory, then close to retire registration through SDLC.


## Log

### 2026-10-24
- 2026-10-24: abandoned (wontfix): fixed by landed pair#697 (warm-worker injection) and pair#812 (stale capped-chrome oracle replaced); both orderings pass on main. Exact pinned logs, 45-cell versus 40-cell cause, fix ancestry and targeted count3 PASS3.710s are recorded above; no runtime change or timeout increase required.

### 2026-10-24 — cause before code; plan sent for approval

Claimed and started from fresh origin/main56986a072ef70b7ee0d00c317395088b2bb44915, independently of waiting #844. Only targeted tests run. Both later-failure startup orderings passed count3 in3.710s on the unmodified main code.

Pinned recorded evidence:
- Original registered fail4fed2c456da3773658b2bd4328454d6a and same-head pass5bafd7c76410f108c1e6ee72e3e78952 ran Oct16 on61066dc88570e5b010925faf1ef02fcefc028231. Failure was injected refusal returning nil. It predates #697b75a3d1ddea2126f7d0d4982576665e765d4d6b5: the warm worker could consume FailNextStart; existing path-targeted refusal and forced warm-first regression address that ordering.
- Later failf9b3e2af722df79bce43bf1cf59e89dd ran Oct22 on352190cbc25d79b54e6ad240f6bcfe9e64af8415. Both orderings passed their injected-error assertion and then failed line230 after30.09/30.19s: timed out waiting for visible hub refusal and retry action. That source required literal 'hub failed; open it in the switcher to retry:' (45 ASCII cells, measured), but its chrome_status.go caps display at40 including ellipsis. The old visible value is 'hub failed; open it in the switcher to …'. This is a deterministic stale display oracle, not evidence of load latency; a longer wait cannot help.
- #8129bfe7f5a010736512f8fb1801b8560fd4eee2388 replaces that exact wait with attention publication and openWarningAttention. Current helper replays terminal cells, waits for publication before input and has controlled publication-before/after plus dismissed-history cases. Git confirms both fixes are ancestors of this branch. Independent read-only peer confirmed both diagnoses.

Plan to hub: evidence-only retirement using landed fixes and existing regressions, not another runtime change or redundant test; targeted startup/helper/delta-screen proof followed by required SDLC verification. Await hub response before change-code. No implementation edits made. #844 remains on its pushed branch waiting for combined live acceptance.
