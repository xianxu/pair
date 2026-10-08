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

---

## Re-review — 2026-10-07T21:08:38-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 395 — Couch broadcast: stream the composed Couch screen, view-only, to a browser viewer |
| repo | pair |
| issue file | workshop/issues/000395-couch-broadcast-stream-the-composed-couch-screen-view-only-to-a-remote-couch-watch.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 67cccfc6102c04826c5e0adf918c0bbb6eb5000c..c4cc6aa79e277c518dafe5a56e410039cd358c57 |
| command | sdlc milestone-close --issue 395 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-10-07T21:08:38-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

All five findings from round 1 are fixed, and the fixes hold up when checked against the code. **BR-11:** a single `startClaim` compare-and-swap (CAS) now decides who owns a finished start's session. Neither side can infer the other's decision any more. The loop adopts only when the start is still current and has no error. The goroutine stops the session only if its `abandon()` wins. `go test -race -count=8 -run 'TestBroadcast|TestStartClaim' ./cmd/internal/couchtty/` passes; that same command failed in round 1. **BR-12:** both background waits (`awaitDown` and the Done watcher) now also select on `c.stop`. `teardown` closes `c.stop` before `endBroadcastForShutdown` and `workers.Wait`, so a stuck tunnel can only hold exit for `broadcastShutdownWait`. A new test using `FakeTunnel{CloseBlock}` pins this. **BR-13, BR-14 and BR-15:** the missing click test now exists, the start context is cancelled once Start returns, and the README and Log now describe the local-only link and the M3 smoke run. Two Minor issues remain, and neither blocks the boundary. In `couchcmd` the only failures I saw came from the sandbox (`mkdir /tmp/...: operation not permitted`); `TestBroadcastSettings` passes, and `broadcast` and `couchkeys` are green.

1. **Strengths**
   - `console_broadcast.go:47-61`: `startClaim` sets one rule for ownership, and its doc comment explains why neither side can trust its own view of the handover.
   - `console_broadcast.go:11-23`: the broadcast states are an explicit enum with a documented transition table, not a set of independent flags (ARCH-ORDER).
   - `endBroadcastForShutdown` moves the attempt counter forward, so a start that finishes after shutdown is never current and is always abandoned. The claim and the attempt counter work together correctly.
   - The `workshop/lessons.md` entry states the general rule ("a caller that can stop waiting must not read state written by the work it handed off"), not just this one case.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `broadcast/tunnel.go:23-25` says "Start cancels its context once it returns", but `broadcast.Start` (`session.go:51`) never cancels it; `Console.startBroadcast` does. A different caller of Start would not do this, so the contract names the wrong actor.
   - `TestStartClaimDecidesOnce` only checks that `atomic` CAS picks one winner, which mostly restates the standard library. Nothing deterministically drives the ordering that caused BR-11: `runTerminalCommand` returns early, `abandon` wins, and the loop's closure runs later.

5. **Test coverage notes:** a test now covers bounded shutdown with a stuck tunnel. Click-on-off is covered for both the phase and the `statusControl` span. The race is covered only probabilistically (`-race -count=8`); there is no way to inject the order of the loop and the goroutine.

6. **Architecture**
   - **ARCH-DRY:** pass.
   - **ARCH-PURE:** pass. The phase logic is small and lives on the loop.
   - **ARCH-PURPOSE:** pass. M3 delivered the cell, the key, wiring, options, smoke and atlas; the theme and font work was moved to a new M4 by a Revision.
   - **ARCH-MOCK:** pass. `FakeTunnel` sits behind the Tunnel seam and now models `CloseBlock`.
   - **ARCH-CONSTRAINTS:** pass. Exit is bounded at 6s and a test enforces it.
   - **ARCH-SECURE:** pass. The notice deliberately leaves the link out of broadcast frames.
   - **ARCH-ORDER:** pass with a note. The phase enum and the CAS are explicit, but the interleaving can't be injected (Minor above).
   - **ARCH-FUNERAL:** pass. The start context is released and sessions are stopped on shutdown. A session abandoned during shutdown is stopped but its teardown is not awaited, because `awaitDown` returns on `c.stop`. M5's cloudflared orphan reaping has to cover that tunnel process.

7. **Plan revisions:** none required. The M4/M5 Revision matches the code.

```findings
dispose:
  - id: BR-11
    disposition: addressed
    note: |
      startClaim CAS (console_broadcast.go:47-61,124-131,143) gives one owner; go test -race -count=8 -run TestBroadcast|TestStartClaim passes.
  - id: BR-12
    disposition: addressed
    note: |
      awaitDown and the Done watcher select on c.stop, which teardown closes before workers.Wait; TestBroadcastShutdownBoundedByStuckTunnel pins it.
  - id: BR-13
    disposition: addressed
    note: |
      TestBroadcastOffHasNoClickTarget added (console_broadcast_test.go), checking phase, session and the statusControl span.
  - id: BR-14
    disposition: addressed
    note: |
      cancel() runs right after broadcast.Start returns; the Tunnel contract now says a tunnel must outlive ctx.
  - id: BR-15
    disposition: addressed
    note: |
      README says links are local-only until M5; the issue Log records the operator's M3 local smoke at 0dbe0802.
findings:
  - id: new
    severity: Minor
    family: plan-code-drift
    title: |
      Tunnel.Open doc says Start cancels its context; the Console caller does
    detail: |
      5th in family. Rule: a contract comment names the actor that actually performs the action, checked against the code at that point. broadcast.Start (session.go:51) never cancels ctx; only Console.startBroadcast does, so say "callers may cancel ctx once Start returns".
  - id: new
    severity: Minor
    family: cross-channel-state-read
    title: |
      No deterministic test drives the late-closure-after-abandon ordering behind BR-11
    detail: |
      3rd in family. Rule: every handoff that can return early has a test that injects the order (closure after the early return). TestStartClaimDecidesOnce only exercises atomic CAS; the integration ordering is covered only by -race sampling.
```
