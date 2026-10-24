# Boundary Review — pair#846 (whole-issue close)

| field | value |
|-------|-------|
| issue | 846 — Couch: count-agnostic lifecycle drain diagnostic |
| repo | pair |
| issue file | workshop/issues/000846-couch-count-agnostic-lifecycle-drain-diagnostic.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3ede7c644d36dbcddf16a0b26b89785eda2c3f31..01c56b37d69c2ee1e46cd525efc38503dab790d2 |
| command | sdlc close --issue 846 |
| reviewer | claude (anthropic; author anthropic) |
| isolation | fenced |
| timestamp | 2026-10-23T15:18:06-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

This change matches the Spec. One format string in `cmd/internal/couchcmd/lifecycle_notices.go:179` changed to `Lifecycle: metadata unverified for %d slot(s) at restart drain`, and the exact-match pin at `lifecycle_notices_test.go:684` changed with it. Nothing else in the window changed. I checked the tree for the old wording: the only `metadata unverified` matches are the producer, its test, and the issue file. No atlas page, README or other message mentions it. `go test ./cmd/internal/couchcmd/ -run Lifecycle -count=1` passes. The new string has no plural that can disagree with its count, at N=1 or any other value. Nothing blocks SHIP.

1. **Strengths**
   - The diff is two lines and stays within its scope: one producer line and its pin, as the Spec says.
   - Because the test compares the whole message, the old string fails the new pin. That is real regression evidence, not a substring match.
   - The issue log explains why the plan uses `slot(s)` rather than `slots`: `for 1 slots` would still be a disagreement.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - The test pins only N=1. The format has no count-dependent branch, so N>1 cannot behave differently, but there is no fixture for it. An optional test with two missing slots would cover it.

5. **Test coverage notes**
   - **"The drain diagnostic never renders a plural that disagrees with its count":** the code meets this because `slot(s)` doesn't depend on the count. The test pins N=1, the case that was broken.
   - **"The exact-match test pins the new string; the couchcmd package is green; no other message is touched":** I confirmed all three parts. I ran the focused `Lifecycle` tests rather than the whole package. The tree search showed no other `metadata unverified` message.

6. **Architectural notes**
   - **ARCH-DRY: pass.** The string has one producer and no copy elsewhere in the tree.
   - **ARCH-PURE: pass.** No new logic or IO.
   - **ARCH-PURPOSE: pass.** The fix delivers the whole purpose; no other place uses the old message.
   - **ARCH-EVOLVE: pass.** This is diagnostic display text, not a persisted format.

7. **Plan revision recommendations:** none.

```findings
findings:
  - id: new
    severity: Minor
    kind: defect
    family: count-dependent-message-coverage
    title: |
      Drain diagnostic pin exercises only N=1
    detail: |
      TestLifecycleDrainReportsOnlyUntargetedFacts (lifecycle_notices_test.go:684) pins only the one-missing-slot case. The format has no count-dependent branch, so this is safe today. A fixture with two or more missing slots would pin the count-agnostic Done-when clause at another count. This is the only count-rendering message in the window.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"5ce979ef-db46-47ee-b3d6-6b027e5216e9","repository":"github.com/xianxu/pair","issue_id":"000846","recipe":"small-diff-review","round":1,"attempt":1,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/808621e2bc5ffd6a","reviewed_head":"01c56b37d69c2ee1e46cd525efc38503dab790d2","slot":"pair:8","requested_at":"2026-10-23T22:14:07.03303Z"},"phase":"finished","pid":5935,"started_at":"2026-10-23T22:14:07.19022Z","finished_at":"2026-10-23T22:17:33.172927Z","session_id":"dc3dde45-a3f5-4cbf-a52a-202ece93e526","outcome":"returned"},{"version":1,"intent":{"disposable_root":true,"id":"0087f90f-9ea0-4f79-a5c8-11ea14fec2de","repository":"github.com/xianxu/pair","issue_id":"000846","recipe":"small-diff-review","round":1,"attempt":2,"agent":"claude","root":"/Volumes/TestScratch/review-checkouts/808621e2bc5ffd6a","reviewed_head":"01c56b37d69c2ee1e46cd525efc38503dab790d2","slot":"pair:8","requested_at":"2026-10-23T22:17:39.972568Z"},"phase":"finished","pid":87033,"started_at":"2026-10-23T22:17:40.117265Z","finished_at":"2026-10-23T22:18:05.000167Z","session_id":"8a98adcf-6090-4001-a4fb-59367153b8bc","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.84672875s
head: "01c56b37d69c2ee1e46cd525efc38503dab790d2"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"
Preparation attempt 2; duration 2.923800291s
head: "01c56b37d69c2ee1e46cd525efc38503dab790d2"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"
Protocol attempt 1
reason: "missing valid findings block"
output: "```verdict\nverdict: SHIP\nconfidence: high\n```\n\nThis is a two-line change and it matches the Spec. `cmd/internal/couchcmd/lifecycle_notices.go:179` now prints `Lifecycle: metadata unverified for %d slot(s) at restart drain`, so at N=1 the count and the noun no longer disagree. The exact-match test at `lifecycle_notices_test.go:684` was updated to the new string. I searched the whole tree for the old wording and for \"metadata unverified\". The only live hits are the producer and its test. The rest are in `workshop/history`, which records the past and shouldn't change. The other three writers of `p.diagnostics` (lines 102, 232, 448) are untouched, which matches \"no other message is touched\". The focused test `TestLifecycleDrain*` passes and `go vet` is clean. The full package run here had four failures, all from this sandbox refusing `git init` outside the reviewer's allowed directories (sdlc #339). None of them touches this code.\n\n1. **Strengths**\n   - Fixes the cause directly: `slot(s)` reads correctly at every count, and the Log explains why a plain `for N slots` would still break at N=1.\n   - The test still checks the full string, so the count of 1 is still pinned.\n   - Status text is cut at 40 cells (#789). The new string still shows the count before the cut: `Lifecycle: metadata unverified for 1 sl…`.\n\n2. **Critical:** none.\n3. **Important:** none.\n4. **Minor:** none.\n5. **Test coverage notes:** Only N=1 is pinned. That is the case that was broken, and the format string has no branches, so testing N>1 adds little. In the full package run, these four tests failed only because the sandbox refused `git init`:\n   - `TestContinuationWriterPublishesExactCheckpointAcrossWorktrees`\n   - `TestHubWorktreeIsTheStartupCheckout`\n   - `TestRecoveryMenuReachesTerminalAfterActualHelperDeath`\n   - `TestRecoveryNativeOnlyProductionComposition`\n\n   They have nothing to do with this diff; the implementer's run on the real host is the evidence for the package being green.\n6. **Architectural notes:**\n   - **ARCH-DRY: pass.** The string is built in one place and appears nowhere else.\n   - **ARCH-PURE: pass.** The diff only changes a format literal inside existing code.\n   - **ARCH-PURPOSE: pass.** The whole purpose (no plural that disagrees with its count) is delivered, and no other live copy of the old string exists.\n   - **ARCH-EVOLVE: pass.** The string is shown on screen and never saved, so no stored format changes.\n7. **Plan revision recommendations:** none.\n\n```findings\n```"

truncated: false
retained-bytes: 0000003281
