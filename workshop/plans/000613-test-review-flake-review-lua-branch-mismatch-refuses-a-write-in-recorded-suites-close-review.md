# Boundary Review — pair#613 (whole-issue close)

| field | value |
|-------|-------|
| issue | 613 — test-review flake: review.lua branch mismatch refuses a write in recorded suites |
| repo | pair |
| issue file | workshop/issues/000613-test-review-flake-review-lua-branch-mismatch-refuses-a-write-in-recorded-suites.md |
| boundary | whole-issue close |
| milestone | — |
| window | d43f34027a65bcfdede3a7d22a37ef0b3aabb953..0a4877243f96e48884c6b9c0c13764e39778e259 |
| command | sdlc close --issue 613 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-16T07:21:26-07:00 |
| verdict | SHIP |

## Review

Checking how many sequential git calls a single identity resolve makes against the 2.5 s budget.

```verdict
verdict: SHIP
confidence: high
```

This fix finds a real cause and fixes it with a small change. I checked it independently instead of relying on the log. `reviewcmd.OSRuntime.GitContext` used a 50 ms `WaitDelay`. That meant a read that was already complete could fail with `exec: WaitDelay expired before I/O complete` while git's stdout was still draining or held for a moment by a child process. The review pane's `BufWritePre` then reported that failure as "branch mismatch". The new regression test, `TestOSGitContextToleratesBrieflyHeldStdout`, uses a git alias with a 0.3 s background holder of stdout. I ran it on the pinned head and it passed 3/3. I then set the delay back to 50 ms and it failed with exactly the error from the sighting. I restored the checkout afterwards. The `reviewcmd` and `couchcmd` packages pass in full. The fix also covers the same problem in two other places (the zellij `list-panes` and `list-clients` queries), and the write-refusal message now gives the real reason. Three Minor findings, none blocking:
- The new code comment states a time budget that is slightly wrong in the worst case.
- The new error message has no test.
- The issue's Plan section was never filled in.

### 1. Strengths
- **Cause measured, not guessed.** The Log uses the second sighting's own observation JSON (resolve call 39 returned `invalid` with the WaitDelay error). It also rules out CPU load alone: 0/20 failures at load 18 and at load 58.
- **The regression test fails without the fix.** At `cmd/internal/reviewcmd/identity_test.go:525` it runs the real `OSRuntime` against a real temporary git repo, with no mocks. It goes red at 50 ms and green at 1 s.
- **The whole class is fixed, not just one site.** `git grep WaitDelay` shows the two short delays that remained (100 ms in `couchcmd`) are now in line with the 1 s used everywhere else (`couchcore/runner.go:86`, `provision_io.go:72`, `historyarchive/codec.go:56`, `launcher/checkpoint_io.go:32`).
- **The error message is fixed.** At `nvim/review.lua:881-885` the message now uses the reason that `C:guard` already returns (`restore_controller.lua:84-93`). A resolver failure is no longer shown as a branch change. The `or 'review authorization refused'` fallback matches the existing wording in `C:admit`.

### 2. Critical findings
None.

### 3. Important findings
None.

### 4. Minor findings
- **The new comment overstates the time bound** (`cmd/internal/reviewcmd/runtime.go:160-164`).
  - It says 1 s "still bounds a stuck holder within the review guard's 2.5s budget". But `resolveIdentityWithSnapshot` gives the whole resolve a 2 s context (`identity.go:142`), and Go starts the `WaitDelay` timer when the process exits or when the context ends.
  - So a git call that exits or is killed near 2 s, with stdout still held, can return at about 3 s. That is past the Lua side's `wait(2500)` / `timeout=2500` (`nvim/review/identity.lua:24,29`). The old worst case was about 2.05 s.
  - The effect is limited: the guard still refuses (it fails closed), but with a generic timeout instead of the git diagnostic.
  - Fix: either correct the comment, or cap the delay against the remaining deadline, e.g. `min(1s, time.Until(deadline))`, or a 500 ms delay with a 2 s context.
- **The new refusal message is not tested** (`nvim/review.lua:884`). No test checks that the guard's reason (rather than "branch mismatch") reaches the `BufWritePre` error. A small Lua test that injects a resolver returning `invalid` would cover it.
- **The issue's `## Plan` is still an empty `- [ ]`.** The delivered work is described only in the Log, so there is nothing to tick off against the scope.

### 5. Test coverage notes
- The cause is covered by a real-IO integration test that fails without the fix (verified above).
- The `couchcmd` changes have no specific regression test. They are the same pattern, and that package's tests pass. Acceptable.
- Done-when asks for "passes repeatedly under concurrent load". The 0/20 runs under load happened before the fix and did not reproduce the flake. The evidence after the fix is the regression test that fails without it. That is reasonable, because the failing condition (stdout held past the delay) is now reproduced exactly on demand instead of by chance.

### 6. Architecture
- **ARCH-DRY: pass.** The repo has several separate `WaitDelay = time.Second` assignments. A shared constant or helper (for example `execx.DefaultWaitDelay`) would stop this drifting again, but that is not needed here.
- **ARCH-PURE: pass.** The change is confined to the thin IO layer.
- **ARCH-PURPOSE: pass.**
  - The class (a short `WaitDelay` turning complete output into a failed read) was searched across the repo and every remaining short delay was fixed.
  - The misleading message was fixed where it appears. Other guard callers (`review.lua:495,528,632,813`) already go through `guard()`, which notifies the reason.
- **ARCH-MOCK: pass.** The test drives real git in a temporary repo, not a mock.
- **ARCH-CONSTRAINTS: flag (Minor).** This is the worst-case timing overshoot described in section 4.
- **ARCH-SECURE: pass.** No new untrusted input. A resolver failure still fails closed.
- **ARCH-ORDER: pass.** The test pins the problem ordering (git exits while a descendant still holds stdout) deterministically through the alias.
- **ARCH-FUNERAL: pass.** Nothing durable is created. The background `sleep` in the test exits on its own after 0.3 s inside `t.TempDir()`.

### 7. Plan revision recommendations
- Fill in `## Plan` with what was delivered:
  - [x] `GitContext` `WaitDelay` 50 ms → 1 s
  - [x] `couchcmd` zellij queries 100 ms → 1 s
  - [x] `BufWritePre` reports the guard's reason
  - [x] regression `TestOSGitContextToleratesBrieflyHeldStdout`

```findings
findings:
  - id: new
    severity: Minor
    kind: defect
    family: exec-timeout-budget-composition
    title: |
      GitContext comment claims 1s WaitDelay stays within the 2.5s guard budget, but the worst case is about 3s
    detail: |
      resolveIdentityWithSnapshot bounds the resolve with a 2s context (identity.go:142), and Go starts the WaitDelay timer when the process exits or the context ends. A call that exits or is killed near 2s with stdout still held can return at about 3s, past the Lua wait(2500) in nvim/review/identity.lua:24. The guard still refuses (fails closed) but with a generic timeout instead of the git diagnostic. Fix the comment, or cap WaitDelay against the remaining deadline.
  - id: new
    severity: Minor
    kind: defect
    family: diagnostic-surface-untested
    title: |
      BufWritePre refusal now reports the guard reason, but no test checks it
    detail: |
      nvim/review.lua:884 changed from a fixed "branch mismatch" to the reason from guard(). No Lua or recorded test asserts that an invalid resolver's diagnostic reaches the error text, so the misleading message could come back without any test failing.
  - id: new
    severity: Minor
    kind: defect
    family: plan-not-recorded
    title: |
      Issue Plan section left as an empty checkbox
    detail: |
      The delivered steps (two WaitDelay changes, the message change, the regression test) are recorded only in the Log, so there is nothing to check the scope against.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"ca610391-5b31-4b4d-ba0e-bc122e76506e","repository":"github.com/xianxu/pair","issue_id":"000613","recipe":"milestone-review","round":1,"attempt":1,"agent":"claude","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/04df92a036749b7e","reviewed_head":"0a4877243f96e48884c6b9c0c13764e39778e259","slot":"pair:2","requested_at":"2026-10-16T14:19:05.765104Z"},"phase":"finished","pid":87204,"started_at":"2026-10-16T14:19:05.891736Z","finished_at":"2026-10-16T14:21:23.805049Z","session_id":"6f8dc521-4f73-40fb-8467-0fa18a49f599","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.726102834s
head: "0a4877243f96e48884c6b9c0c13764e39778e259"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
