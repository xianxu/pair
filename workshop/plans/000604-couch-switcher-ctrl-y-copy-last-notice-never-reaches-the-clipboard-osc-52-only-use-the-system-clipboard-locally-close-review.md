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

---

## Re-review — 2026-10-16T16:45:04-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 604 — Couch switcher: Ctrl+Y 'copy last notice' never reaches the clipboard (OSC 52 only); use the system clipboard locally |
| repo | pair |
| issue file | workshop/issues/000604-couch-switcher-ctrl-y-copy-last-notice-never-reaches-the-clipboard-osc-52-only-use-the-system-clipboard-locally.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9fc035e718b1d2f4810b426fb6dfa9297b4c746d..b0a05d4aba95172bf272930c74280409f928706e |
| command | sdlc close --issue 604 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-16T16:45:04-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The implementation satisfies the revised contract. Both prior findings are addressed, and no new defects were found. Local clipboard delivery and fallback behavior pass verification; live clipboard confirmation remains explicitly deferred until the next natural restart.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Issue #604:38–54 and its Revisions entry at lines 119–121 explicitly supersede the original diagnostic requirement and reconcile the Spec, acceptance criteria, and checked Plan. atlas/couch.md:1459–1463 records the same bounded scope and unknown loss layer. This is a prose-only scope correction.
  - id: BR-2
    disposition: addressed
    note: |
      README.md:1430–1435 now explains local clipboard copying, the command budget, SSH/tool-failure fallback, and unconfirmed feedback. These claims match console_menu.go:443–452 and clipcmd/runtime.go:70–97. No executable behavior changed in this correction.
```

1. **Strengths**

   - Reuses the existing clipboard registry and preserves legacy optional-copy behavior (`runtime.go:58–63`).
   - Bounds clipboard subprocess work and performs it outside the console mutex (`console_menu.go:437–446`).
   - Tests assert exact multiline/Unicode delivery, repeated tool invocation, and no OSC 52 after local success.
   - README and atlas consistently distinguish confirmed copying from unconfirmed fallback.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes:** `go test ./cmd/internal/clipcmd ./cmd/internal/couchtty` passed. Coverage includes missing/failed/timed-out tools, all three SSH markers, cancellation, expired history, empty history, and focused-child routing. The recorded full-suite log hash matches the tracker; subsequent changes contain no production or test changes.

6. **Architectural notes:** **ARCH-DRY: pass**—shared registry and executor. **ARCH-PURE: pass**—clipboard IO remains in the runtime/console boundary, tested through private executables. **ARCH-PURPOSE: pass**—local copying is delivered, and the diagnostic limitation is explicitly reconciled across the revised contract. No Core concepts table is present.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"6efe05d8-8f2a-478e-a2a9-e14876e6f418","repository":"github.com/xianxu/pair","issue_id":"000604","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/3c79b691f1fe8332","reviewed_head":"b0a05d4aba95172bf272930c74280409f928706e","slot":"pair:7","requested_at":"2026-10-16T23:39:55.781586Z"},"phase":"finished","pid":33494,"started_at":"2026-10-16T23:39:55.919398Z","finished_at":"2026-10-16T23:45:03.728943Z","session_id":"01a14716-25c3-79f1-9035-e4c87fb164e2","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.209594167s
head: "b0a05d4aba95172bf272930c74280409f928706e"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
