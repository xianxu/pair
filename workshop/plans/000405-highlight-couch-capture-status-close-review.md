# Boundary Review — pair#405 (whole-issue close)

| field | value |
|-------|-------|
| issue | 405 — Highlight Couch capture status badge |
| repo | pair |
| issue file | workshop/issues/000405-highlight-couch-capture-status.md |
| boundary | whole-issue close |
| milestone | — |
| window | 83326ade46c839ac49fe1893c81cd8b1c4e77143..165ad55ef269c91ff8c434e5a704bcbdba63582e |
| command | sdlc close --issue 405 |
| reviewer | codex |
| timestamp | 2026-10-07T14:06:17-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation matches the Spec by inspection and the targeted tests pass. However, those tests strip styling before asserting capture output, so they do not verify the highlight—the issue’s central requirement.

1. **Strengths**
   - `reserve.go:159` reuses the existing scoped styling helper.
   - The helper resets styling after clipped text and emits nothing for an empty badge.
   - Existing coverage exercises capture states, widths 0–40, and click-span preservation.

2. **Critical findings:** None.

3. **Important findings**
   - `cmd/internal/couchtty/capture_status_test.go:73`: capture assertions discard ANSI styling. Add direct assertions that visible badges receive bold inverse styling and a reset before subsequent content. Enumerate recording, draining, closed, and all three failure labels; include disabled capture and clipped/zero-width output. The test should fail with the one-line change reverted.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Passed: `go test ./cmd/internal/couchtty -run 'Capture|RenderStatusRow|StatusRow' -count=1`.
   - Existing tests validate preserved behavior, but do not distinguish the new highlight from the base implementation.

6. **Architectural notes**
   - **ARCH-DRY: pass** — shared clipping/reset handling is reused.
   - **ARCH-PURE: pass** — styling remains in the pure renderer.
   - **ARCH-PURPOSE: pass by inspection** — all visible capture states use the highlight.
   - No new architectural surface, commands, flags, or configuration require atlas/README updates.

7. **Plan revision recommendations:** None; add verification evidence after covering the highlight.

```findings
findings:
  - id: new
    severity: Important
    family: presentation-contract-coverage
    title: |
      Capture tests do not verify the newly required highlight
    detail: |
      cmd/internal/couchtty/capture_status_test.go:73 strips ANSI before assertions, leaving reserve.go:159's new behavior untested. Cover bold inverse styling and its reset for recording, draining, closed, queue/full/IO failure badges, plus disabled and clipped/zero-width output. Assert subsequent content is outside the highlight and confirm reverting the styling change makes the regression test fail.
```

---

## Re-review — 2026-10-07T14:07:41-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 405 — Highlight Couch capture status badge |
| repo | pair |
| issue file | workshop/issues/000405-highlight-couch-capture-status.md |
| boundary | whole-issue close |
| milestone | — |
| window | 83326ade46c839ac49fe1893c81cd8b1c4e77143..c573660b8153b8438b9a2f5fc1833e665a464aa3 |
| command | sdlc close --issue 405 |
| reviewer | codex |
| timestamp | 2026-10-07T14:07:41-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The one-line change satisfies the presentation contract by inspection and preserves existing rendering behavior. BR-1 is withdrawn under the explicit session instruction against adding tests for reversible, low-impact changes; no regression-test fix is claimed.

1. **Strengths**
   - `cmd/internal/couchtty/reserve.go:159` applies bold inverse styling to every visible capture state through the existing helper.
   - `reserve.go:134–154` handles zero width, empty badges, clipping, and immediate reset before subsequent content.
   - Existing tests cover disabled, recording, draining, closed, and all three failure labels across widths 0–40, including actor click spans.

2. **Critical findings:** None.

3. **Important findings:** None remaining.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Passed: `go test ./cmd/internal/couchtty -run 'Capture|RenderStatusRow|StatusRow' -count=1`.
   - Pinned-range `git diff --check` passed.
   - Existing tests strip ANSI; highlight correctness is established by inspection, not regression coverage.

6. **Architectural notes**
   - **ARCH-DRY: pass** — reuses scoped styling and clipping.
   - **ARCH-PURE: pass** — change stays within the pure renderer.
   - **ARCH-PURPOSE: pass** — all visible capture badges receive the requested emphasis.
   - No new architectural or command/configuration surface requires atlas or README changes.

7. **Plan revision recommendations:** None.

```findings
dispose:
  - id: BR-1
    disposition: withdrawn
    note: |
      Retracted under the explicit session instruction against adding tests for reversible, low-impact changes. Inspected reserve.go:134–159 and capture.go:11–33: every visible capture state uses bold inverse styling, clipping precedes styling, reset immediately follows badge text, and disabled/zero-width output emits no highlight. Existing state, clipping, and click-span tests pass; no new regression coverage or behavior-changing correction is claimed.
```
