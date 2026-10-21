# Boundary Review — pair#776 (whole-issue close)

| field | value |
|-------|-------|
| issue | 776 — couch detach got slower and looks sequential after the 10-21 restart: profile and fix the regression |
| repo | pair |
| issue file | workshop/issues/000776-couch-detach-got-slower-and-looks-sequential-after-the-10-21-restart-profile-and-fix-the-regression.md |
| boundary | whole-issue close |
| milestone | — |
| window | a9522724d4d84586afe222d201029d0713a89fc8..f41bac24797392cca7187a74d645e00072c8020e |
| command | sdlc close --issue 776 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T13:07:25-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The cancellation fix is narrow, preserves descendant cleanup, and passes targeted and race tests. Restoring the old unconditional-wait behavior makes the new regression fail. No production defect was found; the remaining issue is acceptance wording that does not match the revised cancellation contract.

Done-when: stale — the new regressions test group-cancellation completion and lease ownership, while the second acceptance clause still requires a detach-time bound or parallel-retirement regression.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: acceptance-contract-traceability
    title: |
      Align Done when with the approved cancellation regression contract
    detail: |
      workshop/issues/000776-couch-detach-got-slower-and-looks-sequential-after-the-10-21-restart-profile-and-fix-the-regression.md:47-48 still requires a regression that bounds detach time or asserts parallel retirement. The revised implementation instead supplies deterministic group-completion tests (provision_cancel_test.go:49,83) and a descendant-lease control (provision_subprocess_test.go:145). The existing parallel-Leave test also passes without this fix, so it cannot establish that regression requirement. Family enumeration: this second Done-when bullet is the sole stale acceptance clause in this window. Revise it explicitly to accept completion before grace on confirmed group absence, while preserving descendant cleanup and error handling.
```

1. **Strengths:** Only ESRCH permits early completion; permission and inspection failures retain escalation ([provision_io.go:109](/Volumes/TestScratch/review-checkouts/9764e9b79d971783/cmd/internal/couchcore/provision_io.go:109)). The native test verifies inherited lease ownership before cancellation and release after the owned join. Atlas documentation accurately describes the implementation. Retained profiles distinguish observed timing from unproved changed-path execution.

2. **Critical findings:** None.

3. **Important findings:** The acceptance-contract mismatch above; a documentation correction is sufficient.

4. **Minor findings:** None.

5. **Test coverage:** `TestProvision(Process|Group)` passed in 4.263s. The race run covering those tests plus `TestLeave` passed in 6.291s. A temporary Go overlay restoring baseline cancellation semantics made `whole_group_gone` fail with `[TERM,KILL]` instead of `[TERM]`; temporary files were removed. The full recorded suite remains a landing prerequisite.

6. **Architecture:** **ARCH-DRY: pass** — one shared cancellation helper serves the existing runner. **ARCH-PURE: pass** — signal and clock dependencies are injected into the cancellation seam. **ARCH-PURPOSE: pass** — the code addresses the measured shutdown wait without changing fan-out. **ARCH-EVOLVE: pass** — no persisted format changes. Atlas was updated; no new user-facing command or configuration requires a README change.

7. **Plan revision recommendation:** Add a `## Revisions` entry aligning acceptance with deterministic cancellation completion, preserved grace/escalation, and native descendant-lease protection, while retaining the measured four-slot before/after requirement.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"95b66f25-f3eb-4d86-b6b4-0cacf6679991","repository":"github.com/xianxu/pair","issue_id":"000776","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/9764e9b79d971783","reviewed_head":"f41bac24797392cca7187a74d645e00072c8020e","slot":"pair:6","requested_at":"2026-10-21T20:05:16.896795Z"},"phase":"finished","pid":32838,"started_at":"2026-10-21T20:05:17.005238Z","finished_at":"2026-10-21T20:07:16.720138Z","session_id":"01a16011-6d5b-77e2-9120-835c348f7095","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.800233041s
head: "f41bac24797392cca7187a74d645e00072c8020e"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
