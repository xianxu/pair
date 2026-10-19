# Boundary Review — pair#677 (whole-issue close)

| field | value |
|-------|-------|
| issue | 677 — diagnostic logs: untie from backup, enforce the per-file cap, keep ~7 days (48 GB, +1.5 GB/day) |
| repo | pair |
| issue file | workshop/issues/000677-diagnostic-logs-untie-from-backup-enforce-the-per-file-cap-keep-7-days-48-gb-1-5-gb-day.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9950fa540423ab382e682437fc0564fadc104f29..687e59af511ceca9516f0755171c2876277d5de2 |
| command | sdlc close --issue 677 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-18T19:11:07-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The shared retention policy and diagnostic-only scope are well covered, and the saved live evidence supports removal of 18 payloads totaling 12,734,567,104 bytes. Shipping is blocked by incomplete reader protection during deletion recovery, scheduled traversal stopping at stale declarations, inaccurate apply reporting, and a failing artifact-classification guard.

```findings
findings:
  - id: new
    severity: Critical
    family: exact-target-deletion-proof
    title: |
      Writer-side deletion replay does not inspect the segment being deleted
    detail: |
      cmd/internal/diagnosticlog/collect.go:192 removes the backup fence from recoverDeletion, but writer.go:231-236 still authorizes replay using proof of the current path only; Write and maintain have the same pattern. After interruption at delete-intent, a reader can open the closed segment and a subsequent writer reopen can unlink it without inspecting that holder. Enforce exact-target proof across every recovery caller, including ordinary writers without ReaderProof, and add held-segment replay regressions. ARCH-ORDER, ARCH-SECURE.
  - id: new
    severity: Critical
    family: diagnostic-traversal-blocker-isolation
    title: |
      A missing-parent declaration terminates scheduled diagnostic collection
    detail: |
      registry.go:150 now permits structurally valid missing-parent declarations, but cmd/internal/gcruntime/schedule.go:90-97 propagates their canonicalization error. storagegc.ScheduleWorker terminates on that error and preserves its prior cursor, so subsequent eligible paths remain unreachable across restarts. Isolate expected per-path blockers, preserve them visibly, and advance the bounded cursor without recreating or deleting the stale declaration. Test a stale declaration followed by an eligible path through the scheduler. ARCH-PURPOSE, ARCH-FUNERAL.
  - id: new
    severity: Critical
    family: core-concept-contract-accuracy
    title: |
      The Core concepts tables contradict the implemented boundaries and locations
    detail: |
      workshop/plans/000677-diagnostic-log-retention-plan.md:34 classifies diagnostic-only CLI scope as PURE, although runWithService performs service/filesystem IO and its tests require temporary directories and injected services. Line 43 claims modified measurement tests in wrap_events_test.go and diagnostic_buffer_test.go, neither changed in this range; the new implementation is diagnostic_turn_budget_test.go. Promote CLI scope to INTEGRATION and correct the measurement location/status in a Revisions entry. ARCH-PURE.
  - id: new
    severity: Important
    family: diagnostic-result-completeness
    title: |
      Apply omits managed blockers and drops completed work when a page errors
    detail: |
      cmd/internal/diagnosticlog/collect.go:145-187 skips noneligible managed rows and returns only removals, so ApplyDiagnostics can report no blockers while managed payloads remain blocked. Separately, gcruntime/runtime.go:314-317 discards rows returned alongside an error, losing already-completed deletions from byte accounting. Preserve retained/blocked results and completed effects independently of errors; test mixed managed rows and failure after one completed deletion. ARCH-PURPOSE, ARCH-ORDER.
  - id: new
    severity: Important
    family: artifact-source-classification
    title: |
      The required artifact ownership guard fails on both changed source surfaces
    detail: |
      go test ./cmd/internal/artifactpath fails TestProductionArtifactReferencesAreExactlyClassified: diagnosticlog/proof.go remains a NonArtifactSource despite its new artifactpath import, and diagnosticlog/sweep.go is absent from the exhaustive inventory. Update cmd/internal/artifactpath/manifest.go with accurate source classifications and rerun the guard; do not expand the known-violations allowance. ARCH-PURPOSE.
```

**1. Strengths**

- One pure age/size selector serves managed and legacy generations; boundary tests cover seven-day expiry and young oversized files.
- Stateful diagnostic-only tests verify fresh preview/apply observations and zero session-clock effects.
- Live evidence hashes match their manifest. All 18 removed payload identities are absent from the after-inventory, supporting the recorded byte total.

**2. Critical findings**

The three Critical findings above require exact-target recovery protection, scheduled progress past stale declarations, and corrected Core concepts tables.

**3. Important findings**

Apply reporting needs complete blocker and partial-progress accounting. The artifact source inventory also needs updating before its existing guard passes.

**4. Minor findings**

None.

**5. Test coverage notes**

Passed: diagnosticlog, gcruntime, gccmd and wrapcmd suites. Artifactpath failed with the two classification violations above. `git diff --check` passed.

Missing regressions cover held-segment recovery through writer entry points, scheduled traversal past stale declarations, managed blocker reporting, and partial-page accounting.

**6. Architectural notes**

- **ARCH-DRY — pass:** shared eligibility policy and existing deletion journal reused.
- **ARCH-PURE — flag:** CLI classification contradicts its IO boundary.
- **ARCH-PURPOSE — flag:** scheduled progress and complete reporting remain incomplete.
- **ARCH-MOCK — pass:** portable storage, stateful inspection doubles and native conformance coverage exist.
- **ARCH-CONSTRAINTS — pass:** bounded pages/output and counted workload tests are present.
- **ARCH-SECURE — flag:** replay checks the wrong payload’s holders.
- **ARCH-ORDER — flag:** recovery authorization and partial-progress reporting have gaps.
- **ARCH-FUNERAL — flag:** stale declarations can prevent automatic cleanup of later logs.

README and atlas updates cover the new configuration, CLI scope and retention behavior.

**7. Plan revision recommendations**

Add `## Revisions` entries correcting the concept classifications and measurement path, enumerating every deletion-recovery caller, defining scheduled blocker progression, and requiring complete apply results across partial failures.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"98845410-e886-4819-8666-cf53bf2967ba","repository":"github.com/xianxu/pair","issue_id":"000677","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/1326dc8662ea9d21","reviewed_head":"687e59af511ceca9516f0755171c2876277d5de2","slot":"pair:3","requested_at":"2026-10-19T02:07:23.288681Z"},"phase":"finished","pid":59117,"started_at":"2026-10-19T02:07:23.585476Z","finished_at":"2026-10-19T02:11:05.675623Z","session_id":"01a151e9-df70-7791-b337-aa02e2d8a952","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.326571917s
head: "687e59af511ceca9516f0755171c2876277d5de2"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
