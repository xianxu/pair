# Boundary Review — pair#395 (milestone M3)

| field | value |
|-------|-------|
| issue | 395 — Couch broadcast: stream the composed Couch screen, view-only, to a browser viewer |
| repo | pair |
| issue file | workshop/issues/000395-couch-broadcast-stream-the-composed-couch-screen-view-only-to-a-remote-couch-watch.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 67cccfc6102c04826c5e0adf918c0bbb6eb5000c..0dbe0802a8f258eacf31e228551d9e5b6388b599 |
| command | sdlc milestone-close --issue 395 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-10-07T21:02:31-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

**I'm returning REWORK on M3 because of one data race.** Most of the milestone is solid. The status-row control, the Ctrl+Alt+b binding, the `couchcmd` options, the README and the atlas match the plan and its M3 Revisions. The tests drive a real Console fixture, a real SSE viewer and `FakeTunnel`. The race is in the start path, `console_broadcast.go:93-102`. When I ran `go test -race -count=8 -run TestBroadcast ./cmd/internal/couchtty` at the pinned head, `TestBroadcastStoppedOnShutdown` failed with "race detected" in 1 of about 3 runs. The race report has the Run loop writing `adopted` at :97 while the start goroutine reads it at :100. Once that is fixed, the other findings are cheap.

### 1. Strengths
- **The drawer and the checker can't drift apart.** `TestStatusRowLiveCellSatisfiesIndicator` (`reserve_broadcast_test.go:44`) passes the drawn row to `broadcast.IndicatorShown`. It also covers the clipped width, at label width −1 and at exact width (ARCH-DRY).
- **The phase is a tagged enum with its transition table in the header** (`console_broadcast.go:10-28`). The `attempt` counter correctly turns away a late or superseded start (ARCH-ORDER). `TestBroadcastToggleWhileStarting` and `TestBroadcastStopIsOffTheInputPath` exercise the interleavings the operator can't block, using `FakeTunnel` delays and `CloseBlock`.
- **The link stays out of the broadcast.** It is never drawn on screen; the test checks both that the OSC 52 copy happened and that the link is absent from the screen text. End notices reuse the closed vocabulary through `EndReason` (ARCH-SECURE).
- **The options are checked before anything opens.** Settings are parsed before the capture is opened, and an unknown value is a startup error that names the variable (`couchcmd/broadcast.go`).
- **The key binding is enhanced-encoding only**, and a test pins that the legacy `ESC ^B` passes through to the child.

### 2. Critical
- **Data race on `adopted` in `startBroadcast`** (`console_broadcast.go:94-102`).
  - **Cause:** `runTerminalCommand` (`terminal.go:37-44`) can return through `<-ctx.Done()` or `<-c.stop` after the command has been queued, while the Run loop is still running the closure or runs it later. The goroutine then reads `adopted` with no happens-before edge.
  - **Consequence:** besides the `-race` flake, the goroutine can see `false` and stop a session that the loop goes on to adopt. Adopting it installs the tap, sets the phase to `live` and calls `Activate` on a session that is already stopped.
  - **Fix:** a single claim decides who disposes of the session, e.g. an `atomic.Bool` CAS that both the loop closure and the goroutine attempt. Only the claimant decides whether to adopt or stop. Alternatively, have the loop side own the stop of a session it doesn't adopt, and have the goroutine stop it only when it wins the claim.
  - **Regression test:** a test that holds the loop inside `Copy`/`SetTap` while `Stop` fires, run under `-race`.
  - **This is the 2nd finding in family `cross-channel-state-read`** (the M2 one was `Hub.Err()` being read before `done`). The rule covering both: *a caller that can return early (on ctx or stop) must not read anything written by the work it handed to another goroutine. Results must come through the edge the writer publishes, and a single claim decides ownership of any resource the work produced.*
  - **Sweep:** I checked every `runTerminalCommand` call site. `console_broadcast.go:146` captures nothing it reads afterwards, and `console.go:2032` uses only the returned error. So `:96` is the only instance.

### 3. Important
- **The 6s shutdown bound isn't enforced** (`console_broadcast.go:146-150`, `console.go:970`).
  - **What happens:** `endBroadcastForShutdown` waits up to `shutdownWait`, but `teardown` then calls `c.workers.Wait()`. That join includes the `GoTracked` watcher blocked on `<-s.Done()`, and the late-start goroutine's `<-s.Done()` at :100. So shutdown waits as long as the session takes to come down.
  - **Effect:** with `FakeTunnel{CloseBlock}`, Couch never exits. `TestBroadcastStopIsOffTheInputPath` only passes because its `defer close(release)` runs before cleanup.
  - **Fix:** have both waiters also select on `c.stop`, and add a shutdown test with a blocking `Close` that asserts the Console returns within about 6s. ARCH-ORDER (extent of spawned work), ARCH-CONSTRAINTS (stated bound not enforced).

### 4. Minor
- **`TestBroadcastOffHasNoClickTarget` is missing.** The plan's Task 3.3 lists it, there's no Revision explaining its absence, and only the pure row test covers the off span. This is the 2nd finding in `plan-test-coverage-gap`. The rule: every test named in a plan step either exists, or a Revision says why it doesn't.
- **The context is never cancelled after an adopted start or a failed `SetTap`** (`console_broadcast.go:83,120`). Each broadcast leaves one child context registered on `c.lifetime` until Couch exits. If M4's `Cloudflared` ties the process to `ctx`, document that the context must outlive the session; otherwise cancel it once the start completes.
- **The README describes a remote link that M3 doesn't produce yet.** Every broadcast is local-only until M4; worth a "(cloudflared tunnel: M4)" note or making sure M4 updates the text.
- **The operator's local smoke (Task 3.4 Step 2) isn't recorded in the issue Log at head.** Record it there, or in `--verified`, before closing.
- **Import grouping:** the `broadcast` import sits in the stdlib group in `console.go` and `reserve.go`.

### 5. Test coverage
- At the pinned head, the targeted `-race` run passes at `-count=1`, but at `-count=8` the race above fails `TestBroadcastStoppedOnShutdown` intermittently.
- No test covers shutdown while the session's teardown is blocked, or `Stop` arriving while the adoption closure is running mid-way.

### 6. Architecture
| Principle | Result |
|---|---|
| ARCH-DRY | pass (shared `LiveLabel`/`LiveSGR`/`EndReason`) |
| ARCH-PURE | pass (`RenderStatusRow`/`ColumnSpan` pure; wiring thin) |
| ARCH-PURPOSE | pass for M3 scope |
| ARCH-MOCK | pass (`FakeTunnel` behind the `Tunnel` seam) |
| ARCH-CONSTRAINTS | flag (6s bound) |
| ARCH-SECURE | pass |
| ARCH-ORDER | flag (the race; unbounded waiter extent) |
| ARCH-FUNERAL | pass, minor context residue |

For M4, `Cloudflared.Close` must stay bounded (SIGKILL), because shutdown currently waits on it.

### 7. Plan revisions
- If the M3 close doesn't add `TestBroadcastOffHasNoClickTarget`, add a Revision saying why.

```findings
findings:
  - id: new
    severity: Critical
    family: cross-channel-state-read
    title: |
      startBroadcast reads the captured adopted flag after runTerminalCommand may have returned early; reproduced -race failure
    detail: |
      console_broadcast.go:94-102. runTerminalCommand (terminal.go:37-44) returns on ctx/stop while the queued closure may still run on the loop, so the goroutine reads adopted at :100 unsynchronised with the write at :97 (go test -race -count=8 -run TestBroadcast failed TestBroadcastStoppedOnShutdown). Logically the goroutine can stop a session the loop then adopts. 2nd in family. Rule: a caller that can return early must not read state written by work it handed to another goroutine; a single claim (atomic CAS) owns disposal of produced resources. Sweep done: only this call site captures a result.
  - id: new
    severity: Important
    family: declared-bound-unenforced
    title: |
      Shutdown's 6s broadcast wait is defeated by workers.Wait on GoTracked waiters blocked on s.Done()
    detail: |
      The Done watcher (console_broadcast.go:146) and the late-start stop (:100) wait unbounded; teardown's c.workers.Wait (console.go:970) joins them after endBroadcastForShutdown's 6s, so a slow or stuck tunnel Close hangs Couch exit. Select on c.stop as well; add a shutdown test with FakeTunnel CloseBlock asserting a bounded return.
  - id: new
    severity: Minor
    family: plan-test-coverage-gap
    title: |
      TestBroadcastOffHasNoClickTarget from Task 3.3 is absent with no Revision
    detail: |
      2nd in family. Rule: every test named in a plan step exists or a Revision explains its absence.
  - id: new
    severity: Minor
    family: context-lifetime-leak
    title: |
      Start context is never cancelled after adoption or a SetTap failure
    detail: |
      Each broadcast leaves a child context registered on c.lifetime until Couch exits; cancel it once the start completes, or document why it must outlive the session (M4 cloudflared).
  - id: new
    severity: Minor
    family: plan-code-drift
    title: |
      README describes a remote link, but M3 broadcasts are local-only, and the M3 local smoke is not recorded in the Log
    detail: |
      4th in plan-code-drift; rule: docs and the Log state the milestone's actual delivered state. Add an M4 note to the README, or make sure M4 updates it; record the operator smoke in the Log or in --verified.
```
