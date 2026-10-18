# Boundary Review — pair#647 (whole-issue close)

| field | value |
|-------|-------|
| issue | 647 — Harden one-shot zellij actions against misrouted replies (zellij 0.45.1 client-ID reuse, pair#640): unknown on mismatch, confirm before mutating, clipboard fails closed |
| repo | pair |
| issue file | workshop/issues/000647-harden-one-shot-zellij-actions-against-misrouted-replies-zellij-0-45-1-client-id-reuse-pair-640-unknown-on-mismatch-confirm-before-mutating-clipboard-fails-closed.md |
| boundary | whole-issue close |
| milestone | — |
| window | c0841e1082d39731781d8a2972eeeaed41d07ade..c06f33954f1e6962217fe27545e86329911443d3 |
| command | sdlc close --issue 647 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-18T03:24:22-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation satisfies the revised Spec and Plan. Strict observations, contextual confirmation, cancellation checks, and clipboard refusal are wired into production consumers. No blocking defects found. The retained native run supports the refusal claim; its missing final witness remains explicitly inconclusive, with clean-path qualification assigned to the next restart.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Attach stops at the first unknown listing; production and exported-entry regression cases pass.
  - id: BR-2
    disposition: addressed
    note: |
      Short/full listings require exact identity-set agreement; contradictory-listing refusal regression passes.
  - id: BR-3
    disposition: addressed
    note: |
      Registration fixtures distinguish native listing forms and supply typed evidence; fixture contract tests pass.
  - id: BR-4
    disposition: addressed
    note: |
      Diagnostic normalization requires a direct ExitError; collector and cancellation preservation regressions pass.
  - id: BR-5
    disposition: addressed
    note: |
      Native note enrichment and asynchronous setter chain are removed; composition and broadcast tests preserve Couch-owned screen anchors without native queries.
  - id: BR-6
    disposition: addressed
    note: |
      Per-effect cancellation guards cover sidecar fast paths and classified sinks; structural and behavioral cancellation tests pass.
  - id: BR-7
    disposition: addressed
    note: |
      Restart queries use zellijexec.Run and note queries are deleted; owned-descendant collection tests pass.
  - id: BR-8
    disposition: addressed
    note: |
      Every attempted flash application schedules reset independently of admission cancellation; cleanup and failure regressions pass.
  - id: BR-9
    disposition: addressed
    note: |
      Shared confirmation compares decision facts and derived roles; layout and terminal cosmetic-title regressions pass while authority changes refuse.
```

1. **Strengths**

   - `zellijreply/reply.go:23` validates before confirming, terminates on uncertainty, and returns the second snapshot.
   - `layoutcmd/admission.go:63` centralizes contextual decision projections across consumers.
   - `clipcmd/run.go:244` preserves owed cleanup and reports application/reset failures.
   - Stateful CLI fixtures share serialization and production-decoder conformance checks.
   - README and atlas document refusal behavior, partial progress, and protocol limits.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed full `make test-lua`; complete zellijreply, zellijtest, zellijexec, layoutcmd, termcmd, clipcmd, and draftroute packages; focused lifecycle/Couch and prior-finding regressions; and focused race checks for confirmation, cancellation, projections, cleanup, and partial progress.

   Inspected the saved full-suite log and confirmed subsequent changes are tracker-only. Independently inspected native events: the unknown tab reply is followed by refusal, no further mutation, and an unchanged recovery record. No native rerun performed.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Shared validators, projections, effect guard, and fixture serializer.
   - **ARCH-PURE — pass:** Parsing/selectors remain pure; process, clock, and dispatch seams are separated.
   - **ARCH-PURPOSE — pass:** Audited callers are covered, with revised note behavior explicitly documented.
   - **ARCH-MOCK — pass:** Stateful fake exercises production CLI boundaries; native qualification remains available.
   - **ARCH-CONSTRAINTS — pass:** Shared admission deadlines, bounded reads, and bounded restoration.
   - **ARCH-SECURE — pass:** External evidence is validated; context binding precedes addressed effects.
   - **ARCH-ORDER — pass:** Unknown terminates admission; cancellation stops subsequent effects while preserving completed progress.
   - **ARCH-FUNERAL — pass:** Query ownership and cleanup are explicit; no new durable production artifact family.

7. **Plan revision recommendations:** None. Existing revisions reconcile final entity locations, removals, and native-evidence limits.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"01c981e3-1bf5-4174-8459-f6337eecf357","repository":"github.com/xianxu/pair","issue_id":"000647","recipe":"milestone-review","round":8,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/b9c148d5a2eb9f9a","reviewed_head":"c06f33954f1e6962217fe27545e86329911443d3","slot":"pair:1","requested_at":"2026-10-18T10:21:50.519978Z"},"phase":"finished","pid":75483,"started_at":"2026-10-18T10:21:50.623951Z","finished_at":"2026-10-18T10:24:22.046116Z","session_id":"01a14e88-3183-7302-a16d-1d1ff7ca0d2e","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.387988041s
head: "c06f33954f1e6962217fe27545e86329911443d3"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
