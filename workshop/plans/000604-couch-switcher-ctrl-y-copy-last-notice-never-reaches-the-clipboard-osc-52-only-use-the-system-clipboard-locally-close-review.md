# Boundary Review — pair#604 (whole-issue close)

| field | value |
|-------|-------|
| issue | 604 — Couch switcher: Ctrl+Y 'copy last notice' never reaches the clipboard (OSC 52 only); use the system clipboard locally |
| repo | pair |
| issue file | workshop/issues/000604-couch-switcher-ctrl-y-copy-last-notice-never-reaches-the-clipboard-osc-52-only-use-the-system-clipboard-locally.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9fc035e718b1d2f4810b426fb6dfa9297b4c746d..cd7beba6916499e28d64600662660ecd000ea3d9 |
| command | sdlc close --issue 604 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-16T16:33:52-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The clipboard implementation and relevant tests pass, including race checks. Two requirement/documentation gaps remain: the promised OSC 52 diagnosis is unresolved, and README guidance omits the changed clipboard behavior. No executable correctness defect was found.

```findings
findings:
  - id: new
    severity: Important
    family: spec-deliverable-traceability
    title: |
      ARCH-PURPOSE: The promised OSC 52 diagnosis remains incomplete
    detail: |
      The issue's Spec at lines 36–37 requires identifying why OSC 52 fails here, but atlas/couch.md:1451–1455 explicitly says the loss layer was not established. The complete family in this window is that Spec requirement, the checked investigation/documentation Plan item at issue line 49, and the unresolved atlas account. The acceptance revision at issue line 94 defers live clipboard confirmation, without explicitly revising this diagnostic requirement. Complete the diagnosis with evidence, or obtain and document an explicit scope revision reconciling the Spec, Plan, and atlas.
  - id: new
    severity: Important
    family: user-surface-documentation
    title: |
      README update appears missing for changed Ctrl+Y clipboard behavior
    detail: |
      console_menu.go:443–452 changes Ctrl+Y to attempt the local system clipboard and distinguish confirmed copying from unconfirmed terminal fallback, but README.md:1428–1430 remains unchanged. This is the only changed user-facing surface in the window; its related tool selection, SSH exclusion, timeout, and fallback are documented only in atlas/couch.md:1441–1447. Update the existing README paragraph with local copying and the meaning of unconfirmed fallback feedback, satisfying the requested README gate.
```

1. **Strengths**

   - Reuses the existing clipboard tool registry and preserves the legacy optional-copy contract (`runtime.go:58–63`).
   - Bounds the local attempt and performs clipboard IO outside the console mutex (`console_menu.go:437–446`).
   - Tests verify exact multiline/Unicode delivery, repeated copies, and absence of OSC 52 after local success.
   - Atlas accurately distinguishes observed evidence from an unproven transport diagnosis.

2. **Critical findings:** None.

3. **Important findings:** The two findings above require reconciliation before close.

4. **Minor findings:** None.

5. **Test coverage notes:** Both complete affected packages passed. Focused race tests passed for local copying, fallback, empty history, and focused-child routing. The recorded suite log’s SHA-256 matches the issue; subsequent changes are prose only. Live acceptance remains explicitly deferred.

6. **Architectural notes:** **ARCH-DRY: pass**—shared registry and executor. **ARCH-PURE: pass**—IO stays in the console/runtime boundary, exercised through private executables. **ARCH-PURPOSE: flag**—clipboard functionality is delivered, but the separate diagnostic Spec clause remains unresolved. No Core concepts table is present.

7. **Plan revision recommendation:** Add a `## Revisions` entry explicitly resolving the OSC 52 diagnostic scope, then align the Spec and checked Plan item with that resolution. Documentation-only corrections need no wording-presence tests.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"f77ae22b-b527-4d8d-910b-fcbebe870aea","repository":"github.com/xianxu/pair","issue_id":"000604","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/77e37eafd05f3cdd","reviewed_head":"cd7beba6916499e28d64600662660ecd000ea3d9","slot":"pair:7","requested_at":"2026-10-16T23:31:59.388014Z"},"phase":"finished","pid":17178,"started_at":"2026-10-16T23:31:59.515623Z","finished_at":"2026-10-16T23:33:50.452753Z","session_id":"01a1470e-e0c7-7d10-889d-c23ee9d5bb49","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.5855925s
head: "cd7beba6916499e28d64600662660ecd000ea3d9"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000410
