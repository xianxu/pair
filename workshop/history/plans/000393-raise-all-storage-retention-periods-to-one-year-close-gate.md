---
gate: boundary-review
issue: 393
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-05T12:26:18-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Minor
          title: diagnosticlog RetainedReason recomputes days inline instead of sharing storagegc.Days
          detail: cmd/internal/diagnosticlog/writer.go:29 duplicates the duration-to-days conversion in cmd/internal/storagegc/policy.go:21 (ARCH-DRY). These are the only two conversion sites in the window; move Days to one place both packages can use, or have diagnosticlog own a matching helper.
          family: single-source-derived-text
          round: 1
        - id: BR-2
          severity: Minor
          title: Test name still says SevenDays after the capture period became 365 days
          detail: cmd/internal/storagegc/transaction_test.go:304 TestSessionRetirementLeavesYoungCaptureDiscoverableUntilSevenDays; rename to something period-neutral, e.g. ...UntilCaptureRetention. It is the only test name in the tree with a leftover period word.
          family: stale-period-literal-in-names
          round: 1
        - id: BR-3
          severity: Minor
          title: gc render's retained-vs-blocked classification of debug logs has no test
          detail: cmd/internal/gccmd/run.go:189-193 decides retained or blocked by comparing against diagnosticlog.RetainedReason, an exported mutable var, and no gccmd test checks it. Consider making it a const or a typed state, and add a render test that checks one young diagnostic is reported as retained.
          family: untested-reason-string-classification
          round: 1
      recipe: small-diff-review
      blocked: false
---

# Gate ledger — pair#393 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-05T12:26:18-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `single-source-derived-text` diagnosticlog RetainedReason recomputes days inline instead of sharing storagegc.Days
  cmd/internal/diagnosticlog/writer.go:29 duplicates the duration-to-days conversion in cmd/internal/storagegc/policy.go:21 (ARCH-DRY). These are the only two conversion sites in the window; move Days to one place both packages can use, or have diagnosticlog own a matching helper.
- **BR-2** [Minor] `stale-period-literal-in-names` Test name still says SevenDays after the capture period became 365 days
  cmd/internal/storagegc/transaction_test.go:304 TestSessionRetirementLeavesYoungCaptureDiscoverableUntilSevenDays; rename to something period-neutral, e.g. ...UntilCaptureRetention. It is the only test name in the tree with a leftover period word.
- **BR-3** [Minor] `untested-reason-string-classification` gc render's retained-vs-blocked classification of debug logs has no test
  cmd/internal/gccmd/run.go:189-193 decides retained or blocked by comparing against diagnosticlog.RetainedReason, an exported mutable var, and no gccmd test checks it. Consider making it a const or a typed state, and add a render test that checks one young diagnostic is reported as retained.

## Open findings

- **BR-1** [Minor] `single-source-derived-text` diagnosticlog RetainedReason recomputes days inline instead of sharing storagegc.Days
- **BR-2** [Minor] `stale-period-literal-in-names` Test name still says SevenDays after the capture period became 365 days
- **BR-3** [Minor] `untested-reason-string-classification` gc render's retained-vs-blocked classification of debug logs has no test
