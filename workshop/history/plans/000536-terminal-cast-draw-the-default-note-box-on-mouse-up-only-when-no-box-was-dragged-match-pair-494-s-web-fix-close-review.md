# Boundary Review — pair#536 (whole-issue close)

| field | value |
|-------|-------|
| issue | 536 — terminal cast: draw the default note box on mouse up only when no box was dragged (match pair#494's web fix) |
| repo | pair |
| issue file | workshop/issues/000536-terminal-cast-draw-the-default-note-box-on-mouse-up-only-when-no-box-was-dragged-match-pair-494-s-web-fix.md |
| boundary | whole-issue close |
| milestone | — |
| window | 126dc2035fd05b797346c266d440012dfc19c621..a6b46e0903a5e8fb3b9d0480b80a64f78eaab7b0 |
| command | sdlc close --issue 536 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T16:23:54-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned change fulfills the gesture behavior in the Spec. Both viewers use shared drag tracking, terminal bounds and pin guards remain intact, and web/operator regressions pass. No blocking findings.

1. **Strengths**
   - Shared `noteGesture` centralizes threshold crossing and monotonic drag latching (`cmd/internal/broadcast/web/collab.js:162`).
   - Terminal preview suppression reuses existing geometry and preserves cancellation and guard checks (`cmd/internal/broadcast/web/viewer.js:480`).
   - Six regression cases cover mouse/touch clicks, exact thresholds, and drag-back behavior (`tests/broadcast-viewer/pointermode.test.mjs:349`).
   - Atlas documentation accurately describes the changed behavior (`atlas/broadcast.md:519`). No new command/configuration surface requires a README update.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Independently ran all **244 viewer tests: passed**.
   - Loaded the base terminal handler in memory: **all six new regressions failed**, confirming they detect the correction.
   - Twelve direct checks passed across axes, directions, and mouse/touch/pen thresholds.
   - Inspected full-suite logs: two failures in unchanged runner/wrapper code; isolated rerun logs show each passing three times. Did not independently rerun the full suite.
   - Checkout remains unchanged.

6. **Architectural notes**
   - **ARCH-DRY: pass** — terminal and web share tracking; operator uses the shared annotation mount.
   - **ARCH-PURE: pass** — deterministic gesture logic is separated from event/rendering adapters.
   - **ARCH-PURPOSE: pass** — click timing, pixel thresholds, drag latching, and existing consumer behavior are covered.

7. **Plan revision recommendations:** None.

```findings
{}
```
