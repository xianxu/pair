# Boundary Review — pair#590 (whole-issue close)

| field | value |
|-------|-------|
| issue | 590 — Diagnose concurrent launch diagnostics lock timeout under load |
| repo | pair |
| issue file | workshop/issues/000590-launch-diagnostics-lock-flake.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1a1047c89126c68fa1f9c2651bff36663f6ab656..a2a2337bced071aa144bc18e25b45dae944d87bb |
| command | sdlc close --issue 590 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-16T01:40:52-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned changes satisfy the issue’s Spec and Plan. Caller-context admission replaces the store-owned timeout, cleanup precedes capture publication, and existing durability and retention protections remain intact. No blocking findings.

```findings
{}
```

1. **Strengths**
   - Admission checks cancellation before acquisition and after obtaining the lease, with explicit lock ownership and release (`launchfailure_store.go:102–121`).
   - Capture clones evidence before cleanup and preserves launch, cleanup, and persistence error chains (`launchfailure_capture.go:110–130`).
   - Controlled tests exercise real flock contention, release, cancellation, deadline expiry, and refusal without late writes.
   - Atlas documents admission policy and the accepted crash-loss window.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes:** Focused launch-failure tests passed in couchcore and couchcmd, including under `-race`. Existing exact newest-sixteen retention and unsafe/corrupt publication coverage remain. Independently inspected the recorded passing broad-suite log and verified its SHA256; its tested source differs from the pinned head only in the issue log. `git diff --check` passed.

6. **Architectural notes**
   - **ARCH-DRY — pass:** Reuses existing cloning, history normalization, atomic publication, and unlock primitives.
   - **ARCH-PURE — pass:** History logic remains pure; filesystem admission stays in the storage shell with an injected wait seam.
   - **ARCH-PURPOSE — pass:** Addresses both contention and cleanup priority across the affected callers. No new CLI/configuration surface requires a README update; Atlas covers the changed internal contract.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"d27864f2-fe89-4282-88ec-6772ca1b4303","repository":"github.com/xianxu/pair","issue_id":"000590","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/b15efb8edefbc736","reviewed_head":"a2a2337bced071aa144bc18e25b45dae944d87bb","slot":"pair:3","requested_at":"2026-10-16T08:38:34.100187Z"},"phase":"finished","pid":90463,"started_at":"2026-10-16T08:38:34.46111Z","finished_at":"2026-10-16T08:40:43.179149Z","session_id":"01a143dc-eec9-7b50-93b4-556bd03c1546","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.244737625s
head: "a2a2337bced071aa144bc18e25b45dae944d87bb"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
