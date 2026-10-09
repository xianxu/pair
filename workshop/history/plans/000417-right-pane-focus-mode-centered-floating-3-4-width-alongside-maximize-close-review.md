# Boundary Review — pair#417 (whole-issue close)

| field | value |
|-------|-------|
| issue | 417 — Right pane focus mode: centered floating ~3/4 width, alongside maximize |
| repo | pair |
| issue file | workshop/issues/000417-right-pane-focus-mode-centered-floating-3-4-width-alongside-maximize.md |
| boundary | whole-issue close |
| milestone | — |
| window | 566cd1ecef3245d5638c36b1f2d3f715cbf6db33..114366c8efe47cc406895c11649e5d71d07aae91 |
| command | sdlc close --issue 417 |
| reviewer | claude |
| timestamp | 2026-10-09T11:15:52-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The cycle itself is well built. `ObserveRightPaneMode` reads the mode from zellij's pane report alone, `PlanRightPane` turns that into an ordered step list, and the executor runs the steps without contradicting the observed state. The live probes are written into the plan, and the `restoreTiling` swap-layout re-tile is grounded in those probes and backed by fake and live tests. `layoutcmd`, the dim tests and the launcher classifier pass locally. The `termcmd`/`artifactpath` failures come from the sandbox (`ptychild` "operation not permitted") or from tracked-but-unlisted working-tree files, not from this diff. Four things stand between this and SHIP:
1. The dimmer can inject an SGR into the middle of an agent CSI sequence when the dim flips, and a test locks that behavior in.
2. The dim observer runs `zellij action` with no timeout on wrap's signal goroutine and at startup.
3. The test the plan promised for the order of the SIGWINCH path and for signalling the child was not delivered.
4. README.md still describes Alt+Shift+Return as a fullscreen toggle.

1. **Strengths**
   - `rightpane.go:157-191`: `PlanRightPane` is pure and gives one ordered step list per transition. It removes the 5-phase `FullscreenTransition` machine without losing the "stop at the first failure, never run an inverse effect" rule (`rightpane.go:380-382`).
   - `ExpandRecord` rides the existing `FullscreenReturnStore` line. A bare `<id>` written by an older binary still decodes, and unknown tokens are ignored. That avoids a new artifact family (ARCH-FUNERAL) and keeps old binaries compatible.
   - Wrap reuses `layoutcmd.FocusModeActive` for its dim decision, so the cycle and the dim share one predicate (ARCH-DRY).
   - `restoreTiling` is bounded (`maxSwapCycle`), and a failed re-tile only degrades the layout: the press is logged and continues.
   - `ClassifyLiveLayout` was made simpler (`tiledTerminal || floatingTerminal`), and the simpler condition still covers the pre-pivot signature.

2. **Critical**
   - None.

3. **Important**
   - **The dim flip can split a CSI sequence** (`cmd/internal/wrapcmd/dim.go:36-45`). Two cases:
     - **On→off:** the held tail is flushed raw and then `ESC[22m` is emitted. A tail like `ESC[3` is aborted by the injected ESC, and the agent's `1m…` prints as literal text.
     - **Off→on:** while off, the dimmer doesn't track sequences, so `ESC[2m` can land inside an agent CSI that straddles the chunk boundary.

     `TestSGRDimmerTransitions` asserts this broken output (`"\x1b[2ma\x1b[3\x1b[22m1mb"`).

     Fix sketch:
     - On on→off, emit `22m` *before* the pending tail, i.e. `22m + pending + data`.
     - Run `ansi.Frame` tail-tracking even while off, so a transition prefix is only ever emitted at a sequence boundary.
     - Flip the test's expectation.
   - **Blocking IO on wrap's signal goroutine** (`wrap.go:2924`, `dim.go:116-128`; ARCH-CONSTRAINTS). `refreshDim` shells out to `zellij action list-panes` (`exec.Command` with no context or timeout):
     - **Startup:** it runs before the initial `setWinsize`.
     - **Signal loop:** it runs inside the single loop that also serves SIGUSR1/SIGUSR2.
     - **Cost:** if zellij is slow or hung, both resizing and the capture/re-exec signals stall, all for a cosmetic dim.

     Fix: put a short timeout on the observer (e.g. 250ms via `exec.CommandContext`). Ideally also move the observation off the signal loop, or coalesce it.
   - **Promised test missing** (`wrap.go:2920-2934`). Plan Task 5 promised a proxy test with an injected `signalChild` seam. That test would prove two things:
     - the dim is set before `setWinsize`;
     - a flip with an unchanged size signals the child.

     The code calls `syscall.Kill(-pid, …)` inline, and `TestRefreshDimSignalsChildOnlyOnChange` covers `refreshDim` only. Neither the ordering nor the child signal is pinned. Extract the handler body into a method that takes the signal function, and test that method.
   - **README not updated** for the changed keybinding behavior. `README.md:30`, `:148` and `:191` still say Alt+Shift+Return "toggles right-terminal fullscreen". The new shortcut help text and the atlas were updated, so the README row should describe the normal → focus → maximize cycle and the dim too.

4. **Minor**
   - In the drift case where one right terminal floats and another tiled one is fullscreen, `ObserveRightPaneMode` returns Focus and runs Embed then Fullscreen. That toggles a second pane's fullscreen state without first collapsing the existing one. It's rare, but it is not the "converges to one well-defined state" the Spec asks for.
   - Focus→Maximize leaves `swap=`/`order=` tokens in the record (documented in the plan's Revisions as harmless). Fine, but `Maximize→Normal` could clear just those tokens for hygiene.
   - `restoreTiling` handles split order only when there are exactly 2 halves. Splits with 3 or more halves are silently not reordered. That's acceptable given Pair's layouts, but worth a comment.
   - Spec bullet 1 said "dropping SGR 1/22"; the implementation appends faint after every SGR instead. The plan already supersedes the Spec here, so this is not drift, just untidy wording in the Spec.

5. **Test coverage notes**
   - The plan logic has thorough fake tests: the runtime cycle, failure stops, show exit-2, and the legacy record format.
   - The live tests (chord, conformance, rungs) cover the real zellij 0.45.1.
   - Wrap's gaps are the transition-boundary cases (finding 1) and the ordering of the signal handler (finding 3).

6. **Architecture**
   - **ARCH-DRY: pass.** Shared predicate, and selection is reused through `PlanFullscreen`.
   - **ARCH-PURE: pass** for layoutcmd. Wrap's observer is injected, but the child-signal effect is not (finding 3).
   - **ARCH-PURPOSE: pass.** Every Done-when item is delivered and smoke-tested by the operator.
   - **ARCH-MOCK: pass.** The stateful fake models float, swap and order, and the live conformance tests check it against real zellij.
   - **ARCH-CONSTRAINTS: flag.** The unbounded zellij exec on the signal and startup path (finding 2).
   - **ARCH-SECURE: pass.**
     - The pid file is parsed as a positive integer.
     - The record decode is total.
     - The `current-tab-info` JSON parse errors are logged and degrade only the restore.
   - **ARCH-ORDER: mostly pass.**
     - The mode is derived from observation, not stored, and the step list replaces the phase machine.
     - The dimmer's state across chunks (`on` plus `pending`) mishandles the transition boundary (finding 1).
     - The handler ordering is not testable (finding 3).
   - **ARCH-FUNERAL: pass.** No new durable artifact; the existing record is cleared on return to Normal.

7. **Plan revision recommendations**
   - Add a Revisions entry noting that the `signalChild` seam and proxy test from Task 5 were not delivered (or deliver them).
   - Note the timeout or bound on the wrap focus observer once it is added.

```findings
findings:
  - id: new
    severity: Important
    family: stream-rewrite-sequence-boundary
    title: |
      sgrDimmer injects SGR inside a split CSI on dim transitions; test pins the corrupt output
    detail: |
      dim.go:36-45 flushes the pending CSI tail and then emits ESC[22m (on to off), and while off it does not track sequences, so ESC[2m can land mid-CSI (off to on). The agent's SGR is aborted and its remainder prints as literal text. TestSGRDimmerTransitions asserts this. Fix: emit 22m before the pending tail, frame-track while off, and only emit transition prefixes at sequence boundaries.
  - id: new
    severity: Important
    family: optional-io-on-critical-path
    title: |
      Dim observer runs zellij exec with no timeout on wrap's signal goroutine and at startup
    detail: |
      refreshDim (wrap.go:2911, 2924) calls exec.Command zellij list-panes with no context or timeout, before setWinsize and inside the loop that also handles SIGUSR1/SIGUSR2. A hung zellij stalls resizes and the capture/re-exec signals for a cosmetic feature (ARCH-CONSTRAINTS). Bound it with a short exec.CommandContext timeout and/or move it off the signal loop.
  - id: new
    severity: Important
    family: plan-promised-test-missing
    title: |
      SIGWINCH handler ordering and child re-signal on a dim flip are untested; the planned signalChild seam is absent
    detail: |
      Plan Task 5 promised a proxy test showing the dim is set before setWinsize and that a flip with an unchanged size signals the child. The handler calls syscall.Kill inline (wrap.go:2930), and only refreshDim is tested.
  - id: new
    severity: Important
    family: readme-user-surface-sync
    title: |
      README still describes Alt+Shift+Return as a fullscreen toggle
    detail: |
      README.md:30, :148 and :191 should describe the normal, focus (centered, agent dimmed) and maximize cycle that the help text and atlas now document.
  - id: new
    severity: Minor
    family: observed-state-convergence
    title: |
      A floating terminal plus a different fullscreen tiled terminal plans embed+fullscreen without collapsing the existing fullscreen
  - id: new
    severity: Minor
    family: record-hygiene
    title: |
      Focus to Maximize leaves stale swap/order tokens in the return record until the collapse clears it
```

---

## Re-review — 2026-10-09T11:23:04-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 417 — Right pane focus mode: centered floating ~3/4 width, alongside maximize |
| repo | pair |
| issue file | workshop/issues/000417-right-pane-focus-mode-centered-floating-3-4-width-alongside-maximize.md |
| boundary | whole-issue close |
| milestone | — |
| window | 566cd1ecef3245d5638c36b1f2d3f715cbf6db33..ec51c47747ee97fbf2e19999def4fff201f9ec70 |
| command | sdlc close --issue 417 |
| reviewer | claude |
| timestamp | 2026-10-09T11:23:04-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

All four Important findings from round 1 are fixed, and each has a test that would fail without its fix. The dimmer now tracks escape sequences whether or not it is dimming. It never holds back a byte, and it only switches the dim at the first sequence boundary. A new test splits the stream at every possible point, so a transition landing inside the agent's own CSI is covered for every split and both modes. The zellij query behind the dim now has a 1-second deadline. The SIGWINCH handling is now its own function, `handleWinch`, with a `signalChild` seam, and a test checks the order of events. The README describes the new cycle. BR-5 (floating plus a fullscreen half) is now refused, with a test. Only BR-6, which is Minor, is still open. Tests pass at HEAD `ec51c477` for `wrapcmd` (dim/winch tests) and for all of `layoutcmd`.

1. **Strengths**
   - `TestSGRDimmerChunkInvariance` (`dim_test.go`) checks the general rule, not one case. Output must not depend on where reads split. A flip must land at the first boundary of the stream as a whole, which `tokenEdges` works out independently of the code under test.
   - `dimFrame` keeps OSC/DCS/APC open until BEL or ST, so a split OSC 52 clipboard write is no longer cut off by a transition. That gap was real and the fix is narrow.
   - `Feed` never delays a byte (`dim.go:40-47`). Holding output stays with the notification pump, so the dimmer can't hide a split sequence from it.
   - `handleWinch` and `signalChildWinch` make the ordering testable without a real child process (`dim.go:185-210`).

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - The dimmer now parses every escape sequence even while it isn't dimming. Before, that path returned the data unchanged. Output is still byte-identical (the invariance test checks `on=false`), but this is the agent's hot output path. Worth a benchmark if output-heavy agents ever feel slower.
   - No test checks the 1-second deadline itself; it would need a fake zellij that hangs. The deadline is just `CommandContext` wiring, so this is acceptable.
   - BR-6 is still open: the return record keeps stale swap/order tokens.

5. **Test coverage:** the dimmer now has a property-style test, and the SIGWINCH ordering and the re-signal on a flip are covered through the seam. ARCH-ORDER: the dim state is a single atomic boolean plus the `sgrDimmer` state, and its transitions are tested across every split point.

6. **Architecture**
   - **ARCH-DRY:** pass. `dimFrame` reuses `ansi.Frame` and `ansi.OSCEnd`.
   - **ARCH-PURE:** pass. `sgrDimmer` is pure; the IO goes through `observeFocus` and `signalChild`.
   - **ARCH-PURPOSE:** pass.
   - **ARCH-MOCK:** pass. The observer is injected.
   - **ARCH-CONSTRAINTS:** pass. The query is bounded, and the remembered tail is capped at 64 KiB.
   - **ARCH-SECURE:** pass. Malformed strings fall back to the standard framing.
   - **ARCH-ORDER:** pass.
   - **ARCH-FUNERAL:** pass. The expand record rides the existing return record and is cleared by `StepClear`.

7. **Plan revisions:** none needed.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Feed frames in both modes, never delays a byte, and switches only at a sequence boundary; TestSGRDimmerTransitions plus TestSGRDimmerChunkInvariance fail against the old dim.go.
  - id: BR-2
    disposition: addressed
    note: |
      zellijFocusObserver uses ListPanesJSONContext with a 1s focusObserveTimeout (dim.go, layoutcmd.go execZellijContext).
  - id: BR-3
    disposition: addressed
    note: |
      handleWinch plus the signalChild seam; TestHandleWinchOrdersDimBeforeResizeAndSignalsOnFlip pins observe, then resize, then a signal only on a flip.
  - id: BR-4
    disposition: addressed
    note: |
      README.md:30, :148 and :191 now describe the focus to maximize to normal cycle.
  - id: BR-5
    disposition: addressed
    note: |
      ObserveRightPaneMode refuses floating plus fullscreen; the new case in TestObserveRightPaneMode fails without the guard.
  - id: BR-6
    disposition: not-addressed
    note: |
      Minor and still open: Focus to Maximize carries the swap/order tokens until StepClear. Harmless, because the collapse clears the record.
```
