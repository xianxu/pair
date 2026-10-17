# Boundary Review — pair#640 (whole-issue close)

| field | value |
|-------|-------|
| issue | 640 — Native fullscreen chord fixture loses its disposable session during conformance |
| repo | pair |
| issue file | workshop/issues/000640-native-fullscreen-chord-fixture-loses-its-disposable-session-during-conformance.md |
| boundary | whole-issue close |
| milestone | — |
| window | 30e3f624ccf13df378f2dac5c5c036ebdc0015ce..da26fa0a8900192c0c95f0174e9fa34f57116163 |
| command | sdlc close --issue 640 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T12:00:44-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The range changes only issue #640. Its explicit scope revision supports closing the diagnosis while transferring implementation to #647. Retained evidence supports the documented reply misrouting, with appropriate limits on the inferred internal schedule. This verdict does not certify that native conformance is fixed.

```findings
{}
```

1. **Strengths**

   - The issue distinguishes observed crossed replies from the inferred cleanup interleaving, and avoids attributing every historical failure to that mechanism (issue file:94–110).
   - All ten manifest artifact hashes match. Independently counted replies reproduce the reported 65 parallel and 57 sequential failures.
   - The three retained server source files match the v0.45.1 release tag byte-for-byte; their allocation, routing and cleanup paths support the diagnosis.
   - Verified #647’s tracker card and published Spec in the same repository’s source checkout. The upstream reference is also accurate: [PR #5630](https://github.com/zellij-org/zellij/pull/5630) contains client-ID retirement, listed as unreleased in the [changelog](https://github.com/zellij-org/zellij/blob/main/CHANGELOG.md).

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Inspected the diagnostic source, raw reply records, protocol-control implementation and five recorded control outcomes. No runtime suite was rerun; this range changes no production behavior or existing test assertions. `git diff --check` passes. There are no prior findings requiring regression verification.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Reuses the existing controlled-session fixture.
   - **ARCH-PURE — pass:** No production logic or IO boundaries change.
   - **ARCH-PURPOSE — pass:** The explicit diagnosis-only revision matches delivery; #647 exists with the transferred acceptance criteria.
   - **ARCH-MOCK — pass:** Protocol controls exercise the native CLI through a socket fake; production hardening tests belong to #647.
   - **ARCH-CONSTRAINTS — pass:** Diagnostic workers, sample counts, deadlines and frame sizes are bounded; synthetic failure rates are clearly qualified.
   - **ARCH-SECURE — pass:** Diagnostic configuration and socket roots are private; no production trust boundary expands.
   - **ARCH-ORDER — pass:** Evidence distinguishes response completion from pending cleanup and preserves uncertainty about the exact schedule.
   - **ARCH-FUNERAL — pass:** Session cleanup uses the existing fixture; reproduction source is preserved in the issue.

   No Core concepts table requires reconciliation. No new architectural or user-facing runtime surface triggers an atlas or README update.

7. **Plan revision recommendations:** None. The existing Revisions section explicitly supersedes the historical implementation tasks.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"6f160cc8-072d-433f-9245-0c408d965768","repository":"github.com/xianxu/pair","issue_id":"000640","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/a24285a25eb3a9c8","reviewed_head":"da26fa0a8900192c0c95f0174e9fa34f57116163","slot":"pair:1","requested_at":"2026-10-17T18:58:50.023454Z"},"phase":"finished","pid":92293,"started_at":"2026-10-17T18:58:50.13061Z","finished_at":"2026-10-17T19:00:39.13894Z","session_id":"01a14b3b-2777-74e1-b91c-db21cb4b6930","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.407001875s
head: "da26fa0a8900192c0c95f0174e9fa34f57116163"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
