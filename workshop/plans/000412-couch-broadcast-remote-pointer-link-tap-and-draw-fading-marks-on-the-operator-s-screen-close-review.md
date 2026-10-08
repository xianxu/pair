# Boundary Review — pair#412 (whole-issue close)

| field | value |
|-------|-------|
| issue | 412 — Couch broadcast: remote pointer link (tap and draw fading marks on the operator's screen) |
| repo | pair |
| issue file | workshop/issues/000412-couch-broadcast-remote-pointer-link-tap-and-draw-fading-marks-on-the-operator-s-screen.md |
| boundary | whole-issue close |
| milestone | — |
| window | f904c1172c6bb4e2093bfef4e494e0dc0c6c26bb..d73e5ba5a3a3a7b8c3580a4949b7e2fbc1ca6459 |
| command | sdlc close --issue 412 |
| reviewer | claude |
| timestamp | 2026-10-08T12:36:34-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

Since the M4 close (2c91478a), the window has changed only one doc comment in `cmd/internal/broadcast/records.go` (d73e5ba5). The three #415 commits touch only issue files, and that path is excluded from this review. I re-checked every open finding against the current tree. BR-4, BR-7, BR-8 and BR-9 are now fixed: the plan revision, an on-grid filter with its test, the atlas budget text, and pointer stop/generation handling with its tests. BR-1, BR-5, BR-6, BR-11, BR-12 and BR-16 are still open, but all are Minor and none blocks the gate. Nothing new turned up: the delta is one comment, and a spot check of the point POST path, `ParsePointBatch`, `acceptPoint` and the Console broadcast lifecycle found no new defects.

1. **Strengths**
   - `Marks.Add` now drops off-grid points before any line is drawn (`marks.go`, `onGrid`), so its cost no longer depends on an upstream validator. `TestMarksAddIgnoresHugeCoordinates` pins this.
   - `PointerState` now has a `stopped` flag and a `gen` counter. With `offIfGeneration`, a late "pointer hidden" report can no longer turn off a pointing session the operator re-enabled. Covered by `TestPointerAfterStop` and `TestPointerLateHiddenReportIgnored`.
   - `acceptPoint` checks a batch again against the hub's current grid, frame class and whether 👆 is shown. `applyPoints` then re-checks the phase on the Run loop, so stale batches are dropped twice.
   - The POST handler makes its cheap checks before reading the body (pointing on, content type, in-flight slot, rate limit), then reads under a deadline and a size limit.

2. **Critical:** none.
3. **Important:** none.
4. **Minor:** the six open items above, disposed in the block below.
5. **Test coverage notes:** pointer state transitions and Marks bounds are well covered. The gaps are the Console-level wiring: SetBlend (BR-16) and the generation binding of the callbacks (BR-11), plus the viewer loading the Unicode 11 add-on (BR-6).
6. **Architecture:**
   - ARCH-DRY: pass.
   - ARCH-PURE: pass (Marks, ParsePointBatch and RateLimit are pure and take a clock from the caller).
   - ARCH-PURPOSE: pass.
   - ARCH-MOCK: pass (the cloudflared seam is unchanged).
   - ARCH-CONSTRAINTS: pass (the rate, in-flight, body and grid caps are enforced where the work is done).
   - ARCH-SECURE: pass (strict parse, constant-time token match).
   - ARCH-ORDER: one Minor, BR-11. Its session-layer sibling, BR-9, is fixed; the Console callbacks still lack the binding.
   - ARCH-FUNERAL: one Minor, BR-12.
7. **Plan revisions:** none needed beyond the existing revision at plan line ~334.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Tasks 1.3/2.2 still enumerate test cases in prose alongside the one-line Test strategy; Minor, doc-only.
  - id: BR-4
    disposition: addressed
    note: |
      Plan no longer names Marks.Expired; revision near plan line 334 records Live/NextChange/SetBlend (same fix as BR-14).
  - id: BR-5
    disposition: not-addressed
    note: |
      marks_test.go:216 still ends with var _ = terminal.FramePrivate.
  - id: BR-6
    disposition: not-addressed
    note: |
      server_test checks only that addon-unicode11.js is served; nothing exercises viewer.js:204 loading globalThis.Unicode11Addon.
  - id: BR-7
    disposition: addressed
    note: |
      marks.go Add drops off-grid points (onGrid) before line(); TestMarksAddIgnoresHugeCoordinates pins it with a 1<<30 point.
  - id: BR-8
    disposition: addressed
    note: |
      atlas/broadcast.md:233 now says a 2s read deadline, matching pointReadBudget; the "failed ping ends that viewer" line covers the write-deadline effect.
  - id: BR-9
    disposition: addressed
    note: |
      pointer.go set refuses on && stopped; pointerHidden uses offIfGeneration(gen); tests TestPointerAfterStop and TestPointerLateHiddenReportIgnored.
  - id: BR-11
    disposition: not-addressed
    note: |
      console_broadcast.go:117-122 still binds c.onPoints/c.pointerOffByWatch without capturing the attempt; applyPoints and pointerOffByWatch never check the generation.
  - id: BR-12
    disposition: not-addressed
    note: |
      endBroadcastForShutdown (console_broadcast.go:238) and the SetTap-failure branch of broadcastStarted still skip detachBroadcastScreen.
  - id: BR-16
    disposition: not-addressed
    note: |
      console_pointer_test.go has no blend/truecolor assertion; SetBlend at console_pointer.go:169 stays unpinned.
```
