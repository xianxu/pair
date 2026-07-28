# Boundary Review — pair#124 (whole-issue close)

| field | value |
|-------|-------|
| issue | 124 — Alt+Shift+Enter re-tiles in one blind burst |
| repo | pair |
| issue file | workshop/issues/000124-alt-shift-enter-re-tiles-in-one-blind-burst.md |
| boundary | whole-issue close |
| milestone | — |
| window | bc4d05489bd38cb5a6481c65418bdfa1c3458b3b..HEAD |
| command | sdlc close --issue 124 |
| reviewer | claude |
| timestamp | 2026-07-28T08:25:15-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

All checks are done — tests pass, docs verified, one stale comment found. Here is the review.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

This boundary delivers exactly what the Spec commits to: the #123 converge loop (target/step/tolerance/step-cap/settle-delay, the stateful-resize fake mode, and `abs`) is fully deleted, replaced by a pure three-action burst planner plus a thin fire-and-forget executor. I verified the deletion is complete (no live references to the old symbols outside the issue file and history), ran `go test ./cmd/internal/layoutcmd/... ./cmd/internal/termcmd/...` (all pass; a `pty.Start: operation not permitted` failure in termcmd was sandbox-only and disappears unsandboxed) and `tests/term-pane-shortcuts-test.sh` (ok). Atlas was updated in-range and README's existing Alt+Shift+Return row still describes the behavior accurately. The one thing worth fixing before crossing: a stale test-file header comment that states the wrong step model (10%/two steps) directly above tests asserting three steps.

**1. Strengths**

- Clean ARCH-PURE split preserved through the simplification: `terminalToggleBurst` (`cmd/internal/layoutcmd/resizeplan.go:25`) is pure (geometry in → action list out), and the executor loop in `layoutcmd.go:130-135` is now genuinely trivial.
- The degradation story is designed, not accidental: refused steps are no-ops, and any starting width converges to a stable ±15% pair still classified by the 60% threshold — documented at `resizeplan.go:10-13` and in the Spec, and matching what the code actually does.
- Test suite was pruned honestly: the no-progress and step-cap tests died *with* the loop they pinned, rather than being mutated into asserting the new implementation (avoids the mock-reasserts-implementation trap).
- The burst contract is pinned at three independent layers — planner unit tests, executor tests, and the process-level shell fake (`tests/term-pane-shortcuts-test.sh:118-120`) — so a regression in any seam gets caught.
- The Log records the 10%→5% calibration correction with the *reason* for the earlier misread (reads racing application), which is exactly the evidence trail a future zellij-upgrade debugging session will need.

**2. Critical findings**

None.

**3. Important findings**

- `cmd/internal/layoutcmd/layoutcmd_test.go:10-12` — the file-header comment says "fixed two-action burst (zellij's tiled resize step is a stable 10% of the screen, so 1/2 ↔ ~2/3 is always exactly two steps)". Everything else in the diff (planner, atlas, issue Log, the tests immediately below it) says **three** steps at **5%**. This is a leftover from the pre-calibration draft the Log says was corrected; a future reader debugging a zellij step-size change will be actively misled by it. Fix: reword to "fixed three-action burst … stable 5% of the screen … exactly three steps" (one line, matches `resizeplan.go`'s header).

**4. Minor findings**

- A mid-burst `RunZellijAction` error returns 1 with the column between rungs; the next toggle self-corrects via the 60% classification — fine, but no test pins the abort-on-error path (the old loop had the same gap).
- `resizeplan.go` header hardcodes "Zellij 0.44.3" — will silently go stale on upgrade; consider "as of 0.44.3".

**5. Test coverage notes**

Direction (expand/collapse at 75, 89, 90, 105 of 150) and refusal (zero/inverted geometry) cases are covered in the planner test; the executor tests cover the burst end-to-end through the fake runtime including left-focus and missing-geometry no-ops; the shell test covers the real key-dispatch path. The removed stateful fake is a correct deletion, not lost coverage — the thing it modeled (runtime-discovered step size) no longer exists in the design. The only untested path is the mid-burst error abort (Minor above).

**6. Architectural notes** (explicit ARCH pass)

- **ARCH-DRY: pass.** One source for the burst shape (`terminalToggleSteps` + `terminalToggleBurst`); the test JSON was consolidated into `tiledWorkbenchJSON` rather than duplicated per-test.
- **ARCH-PURE: pass.** All decision logic is in the pure planner; the executor only fires actions. The deleted settle-delay was the last IO-pacing concern in this path.
- **ARCH-PURPOSE: pass.** The issue's purpose was the *deletion* of the converge loop, not just adding a fast path, and the diff delivers the full deletion — no hand-maintained remnant, no deferred "follow-up" hiding the point.
- **ARCH-MOCK: pass, with a note.** Production and tests share the `Runtime` seam, and the shell test runs the stack against the process-level fake zellij. But the 5%-per-step model has moved from runtime-discovered to a compile-time assumption whose only conformance check is manual live smoke. The Spec explicitly accepts the degradation mode (a step-size change yields a different stable width pair, still correctly classified), so this is acceptable — just be aware that a zellij upgrade changing `RESIZE_PERCENT` will alter the expanded width silently, and the Log's calibration procedure is the recovery playbook.

**7. Plan revision recommendations**

None — the issue's Spec, Plan checkboxes, and Log all match the shipped code (the Log even documents the 10%→5% correction that the stale test comment in finding 3 missed).
