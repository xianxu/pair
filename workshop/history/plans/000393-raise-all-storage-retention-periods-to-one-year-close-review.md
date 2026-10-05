# Boundary Review — pair#393 (whole-issue close)

| field | value |
|-------|-------|
| issue | 393 — Raise all storage retention periods to one year |
| repo | pair |
| issue file | workshop/issues/000393-raise-all-storage-retention-periods-to-one-year.md |
| boundary | whole-issue close |
| milestone | — |
| window | a73a85b0f3e8c9b0e06da536ed145ef47735b6fa..3d2986ea93fba1a195333717a30e537d237b3a4e |
| command | sdlc close --issue 393 |
| reviewer | claude |
| timestamp | 2026-10-05T12:26:18-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

All three retention periods are now 365 days. Every hard-coded "60 days" and "7 days" string in reasons, `pair gc` output, README and the atlas either comes from the constants now or was rewritten. Tests that used hand-typed day offsets now use the constants. I searched the whole tree at the head commit (excluding `workshop/`) for `60-day`, `60 days`, `7 days`, `seven-day` and `sixty` and found nothing left.

Tests: storagegc, diagnosticlog and gccmd pass. gcruntime has one failure, `TestCouchReferencesLocalArchiveLocatorRoundTrip` ("missing slot Couch metadata"). It also fails at base `a73a85b0`, run from a clean `git archive` copy with the session env cleared, so it existed before this change and isn't caused by it. It does mean the Done-when line "`go test` passes for … gcruntime" isn't literally true in this environment. That should go in `--verified`.

Nothing blocks SHIP. There are only Minor findings.

1. **Strengths**
   - `gccmd/run.go:191` classifies diagnostics as retained or blocked by comparing against `diagnosticlog.RetainedReason`. Before, that was a string literal copied in three places (`legacy.go`, `pages.go`, `run.go`), so a period change would have silently turned every retained log into "blocked". Now there is one source for it (ARCH-DRY).
   - `storagegc.Days` keeps the reasons and CLI text tied to the constants (`policy.go:21`, `capture.go:34-36`, `run.go:107,167,201`).
   - The boundary tests still hold their exact edges with the new values: `writer_test.go:172` uses `RetentionPeriod - time.Nanosecond`, and `references_test.go:157-162` checks both sides of the boundary.
   - `apply_acceptance_test.go:32-40` computes "older than every period" instead of hard-coding 90 days, so it will still hold if one period changes again.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - **ARCH-DRY:** `diagnosticlog/writer.go:29` recomputes days inline (`int(RetentionPeriod/(24*time.Hour))`), duplicating `storagegc.Days` (`policy.go:21`). These are the only two places that convert a duration to days.
   - **Stale test name:** `storagegc/transaction_test.go:304`, `TestSessionRetirementLeavesYoungCaptureDiscoverableUntilSevenDays`, still says "SevenDays". It's the only test name with a leftover period word.
   - **No test for the retained/blocked classification:** `gccmd/run.go:189-193` decides "retained" vs "blocked" for debug logs, and no gccmd test pins it. Also, `RetainedReason` is an exported mutable `var`.

5. **Test coverage notes**
   - The migration from hand-typed offsets is complete in the diff's packages. The remaining literal `365 * 24 * time.Hour` lines in `gcruntime/references_test.go:155,212` are outside the diff, and their behavior doesn't depend on the period value.
   - In `transaction_test.go:326`, the capture is written at now−1d, so at now+`CaptureRetentionPeriod` its age is 366 days, past the limit as intended.

6. **Architectural notes**
   - **ARCH-DRY:** pass, apart from the `Days` duplication above.
   - **ARCH-PURE:** pass. `Decide` and `DecideCapture` are still pure, and the formatting is deterministic.
   - **ARCH-PURPOSE:** pass. All three periods and all user-facing text are covered, as the operator decision required.
   - The accepted #376 risk (logs can now grow for a year) is recorded in the Spec.

7. **Plan revisions:** none, though the Plan checkboxes still need ticking at close.

```findings
findings:
  - id: new
    severity: Minor
    family: single-source-derived-text
    title: |
      diagnosticlog RetainedReason recomputes days inline instead of sharing storagegc.Days
    detail: |
      cmd/internal/diagnosticlog/writer.go:29 duplicates the duration-to-days conversion in cmd/internal/storagegc/policy.go:21 (ARCH-DRY). These are the only two conversion sites in the window; move Days to one place both packages can use, or have diagnosticlog own a matching helper.
  - id: new
    severity: Minor
    family: stale-period-literal-in-names
    title: |
      Test name still says SevenDays after the capture period became 365 days
    detail: |
      cmd/internal/storagegc/transaction_test.go:304 TestSessionRetirementLeavesYoungCaptureDiscoverableUntilSevenDays; rename to something period-neutral, e.g. ...UntilCaptureRetention. It is the only test name in the tree with a leftover period word.
  - id: new
    severity: Minor
    family: untested-reason-string-classification
    title: |
      gc render's retained-vs-blocked classification of debug logs has no test
    detail: |
      cmd/internal/gccmd/run.go:189-193 decides retained or blocked by comparing against diagnosticlog.RetainedReason, an exported mutable var, and no gccmd test checks it. Consider making it a const or a typed state, and add a render test that checks one young diagnostic is reported as retained.
```
