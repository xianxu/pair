---
id: 000726
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '94cde688d9fa5567a1768960be8f0b3f933e6461' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T07:54:36-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# Hub lifecycle ops time out client-side (resume pair:6, relaunch parley.nvim:4: 'context deadline exceeded') with no diagnosis

## Problem

After the 10-20 07:49 Couch restart (supervisor on pair:0 staged f3c597f8, which includes #714 and #721), slot
lifecycle operations from the hub CLI end in a client-side timeout with no diagnosis:

- `couch --resume pair:6` → "context deadline exceeded … outcome uncertain" (4 times since 10-19 23:00, across
  three supervisor builds). `--resume-candidates pair:6` says: "Previous screen unavailable: lstat
  …/scrollback-1-pair-59-codex.raw: no such file", "native transcript incomplete: session inventory read exceeds
  limit", "Candidate scan incomplete", yet still offers "Resume from disk (confirmed)". `--reboot pair:6 --confirm`
  also timed out (10-19 23:05).
- `couch --relaunch parley.nvim:4 --confirm` (idle, load ~16) → "context deadline exceeded"; the slot was NOT
  relaunched (wrapper Start still 10-19 20:12).

The CLI's deadline expires while the supervisor-side operation is presumably still working (or wedged), and the user
gets "verify with couch --recover-plan-from-sdlc", which itself fails to resolve the fleet from a non-git cwd.

## Spec

- Find where the lifecycle operation spends its time (pair:6: likely the native inventory read that "exceeds limit";
  parley.nvim:4: unknown) and bound it inside the supervisor, so the CLI gets a typed outcome (refused with reason,
  or running with an operation id to poll) instead of a client deadline.
- `--resume-candidates` must not offer "Resume from disk (confirmed)" when the resume path will fail on the same
  oversized inventory; either the resume path tolerates it (the binding is established) or the menu says why not.
- `--recover-plan-from-sdlc` must work when run from a non-repo cwd (or say which directory to run it from).

## Done when

- A fixture with an oversized native inventory: resume of an established binding completes or refuses with a reason
  within the CLI budget (test). A relaunch of an idle slot completes within the budget (test with a stateful fake).
- pair:6 resumes (or refuses with a reason) live after the fix.

## Plan

### Diagnosis (2026-10-20, pair:2, measured)

- **Both failures are the admission call, not the operation.** `couchmessage.AdmissionTimeout` is 2 s per exchange (`couchmessage/broker.go`, `transport.go`). The CLI's `exchangeSlotOperation` admits once, then polls `operation-status` for up to 5 minutes. "context deadline exceeded … outcome uncertain" can only come from the first admit exchange (`couchcmd/messages.go`). When the handler finishes late, `serve` drops the reply without writing anything, so the CLI cannot tell whether anything was queued.
- **Admission runs the hub guard synchronously** (`handleHubSlotOperation` → `guard`, `hub_guards.go`). The guard resolves the target through `WorkspaceReferencePath` → `OSSlotCatalog.Discover`: `sdlc` on the primary, `git worktree list`, `git for-each-ref`, then per slot `sdlc workspace` plus 2× `git rev-parse`, all serial and uncached.
- **Measured** on this machine (load 12–18), calling the real `Discover` on the live repositories: pair takes **3.0–3.4 s** (27 commands, 8 slots), and parley.nvim takes **2.6–2.7 s** (15 commands, 4 slots). Discover alone is over the 2 s budget, so every hub lifecycle op on these repositories fails admission, whatever the target's state. That covers both the pair:6 resume and the idle parley.nvim:4 relaunch.
- **The admitted job then repeats the guard twice** (queue prepare, and the hub effect gate). For `--resume` and `--reboot` it runs the full native recovery scan (`executePreparedRecovery` → `observe` → `CollectRecoveryEvidence`) even for a proven resume, which uses only the branch, issue, screen and capture fields. That scan walks all of `~/.codex/sessions` (6.6 GB, about 3,900 files) or `~/.claude/projects` (about 11,000 entries, over the 10,000-entry cap), so it always ends incomplete. It reads matching transcripts in full at 64 KiB per reopen, up to 30 s, while holding the one-wide hub effect gate. pair:6's root transcript is 56.8 MB. This is the next bottleneck once admission passes, and the source of "session inventory read exceeds limit".

### Design (for hub approval)

- [ ] M1 — admission returns a typed outcome inside the budget
  - **Admission reads a cached slot catalog; authority is still proven fresh before any effect.** The guard already re-runs in the queue's prepare step and again at the effect gate, both on fresh discovery. So admission may resolve the hub and target from a per-repository `Discover` snapshot: memoized, refreshed off the request path, and given a bounded age. A cold or expired snapshot runs one bounded refresh. If that refresh cannot finish inside the admission budget, admission refuses with a typed reason ("slot catalog is warming; nothing was queued; retry in a few seconds") and keeps warming.
  - **Discover runs its per-slot probes concurrently** (bounded fan-out), cutting the cold cost for admission, prepare and the effect gate alike. Results stay identical and ordered.
  - **Slot-operation admission has its own budget, below the transport's 2 s.** On overrun it answers `refused` with the phase that overran, never silence. `serve` dropping a late reply stays as it is for the other ops.
  - **Tests (stateful fakes, no live Couch):** a slow catalog fake that takes longer than the budget must give a typed refusal inside the budget, with nothing queued. A warm snapshot must admit with zero `git` or `sdlc` processes spawned, counted through the ProvisionIO seam (hub requirement). A stale snapshot must never authorize an effect: prepare re-proves on fresh discovery and refuses a target that moved. Discover must run its probes concurrently with counted commands and identical results, measured at N and 2N slots.
- [ ] M2 — a proven resume does not scan native candidates
  - `RecoverSelection(Proven)` needs no candidate list. The observation request skips `CollectRecoveryEvidence`'s transcript scan for proven and fresh selections. It keeps the screen, capture, branch and issue reads, which are bounded by their own caps. The scan stays for keyed (inferred) selection, where it picks the candidate.
  - `--resume-candidates` keeps offering "Resume from disk (confirmed)" (the binding is established), and says that an incomplete scan affects only the inferred choices.
  - **Tests:** a fixture with an oversized native inventory (over the entry cap, plus a transcript over the read budget). A proven resume completes with zero transcript reads, counted. An inferred resume still scans and reports incomplete.
  - I'll ask pair:3 (recovery owner) to check the M2 boundary before coding.
- [ ] M3 — `--recover-plan-from-sdlc` from a directory outside any repository: resolve the fleet from Couch's own scope list, or refuse naming the directory to run from. A test covers it.
- [ ] Live check (hub): `couch --resume pair:6` and `couch --relaunch parley.nvim:4 --confirm`. Each should be admitted, or refused with a typed reason, inside the budget; then complete or refuse with a reason.

State design (M1). Actions: run a lifecycle op, `--operation-status ID`, retry.

| Internal state / failure | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Warm snapshot, guard passes | accepted with an operation ID, polled to an outcome | none | none |
| Cold or expired snapshot, refresh finishes in budget | accepted as above | none | none |
| Refresh exceeds the admission budget | refused: "slot catalog is warming; nothing was queued; retry" | retry after a few seconds | none |
| Snapshot stale (slot moved), admitted | prepare re-proves on fresh discovery and refuses with a reason | fix ownership, rerun | none (no effect ran) |
| Any other admission overrun | refused, naming the phase that overran | retry; report if persistent | none |

ARCH notes: ARCH-ORDER (fresh authority before the effect; admission is a cheap gate); ARCH-CONSTRAINTS (Discover cost is linear in slots and now concurrent; admission does zero probes when warm); ARCH-MOCK (stateful catalog fake with a clock); ARCH-FUNERAL (the snapshot has a bounded age and is replaced, never accumulated).

## Log

### 2026-10-20
