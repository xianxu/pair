# Boundary Review — pair#395 (milestone M2)

| field | value |
|-------|-------|
| issue | 395 — Couch broadcast: stream the composed Couch screen, view-only, to a browser viewer |
| repo | pair |
| issue file | workshop/issues/000395-couch-broadcast-stream-the-composed-couch-screen-view-only-to-a-remote-couch-watch.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 2992e81d93829fd43e4f77e685501f9c278393d0..e67af5892393ccc0f725d61d3c12918a60488792 |
| command | sdlc milestone-close --issue 395 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-10-07T19:24:50-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M2 delivers the four plan tasks: xterm.js 5.5.0 vendored and pinned with sha512/sha256 records, the viewer page with a pure `nextFontSize` and its node tests, the GET-only SSE server behind a token, and `Session` over a `Tunnel` seam with a stateful `FakeTunnel`. Tests cover the method and token gate, path and traversal edge cases, headers on every response, same-origin page assets, SSE streaming checked through the emulator, stop, cancel and tunnel exit, and the no-persistence walk. `go test -race ./cmd/internal/broadcast` passes and `go vet` is clean. Two Important problems need fixing first. One is a real race I reproduced: the server reads the hub's end reason before that reason is published, so viewers can be told the wrong reason and `TestSessionTunnelExitEndsSession` is flaky. The other is that the viewer freezes silently when `EventSource` closes for good.

### 1. Strengths
- `server.go:53-71`: the method check comes before the token check, so a non-GET request gets 405 on any path and can't be used to test tokens. The token check uses `subtle.ConstantTimeCompare`. A fixed `assets` map means no file-system path is built from the request. `TestServerRefusesWrongTokenAndPaths` covers `..`, `./`, `//`, prefix, case and trailing-slash variants.
- `tunnel.go`: the `Tunnel` interface owns both `Listen` and `Open`. That decision is correct, and it gives M4's unix socket a clean slot. `FakeTunnel` records state (opens, closes, double closes, `IgnoreCancel` for a slow tunnel that finishes after cancel), so it meets ARCH-MOCK for the M3 Console tests.
- `session.go:76-86`: when a tunnel finishes opening after its start was cancelled, it is closed, not adopted. `TestSessionCancelDuringStart` tests both branches and checks that the listener stopped accepting.
- `TestViewerPageLoadsOnlySameOrigin` statically checks the "no other origin, no storage" Done-when items. The CSP choice is backed by a recorded real-browser check.
- `TestSessionPersistsNoFrameData` walks HOME, TMPDIR, the XDG directories and the working directory for a marker. That is real evidence for render-and-forget.

### 2. Critical findings
None.

### 3. Important findings
- **`server.go:131-133`: the end reason is read before the hub publishes it.** `Hub.end` closes every `sub.c` inside the loop. `h.done` is closed only afterwards, in `run()`. The SSE handler can see the closed channel first. It then calls `Hub.Err()`, which returns nil because `done` is still open, and `endReason(nil)` gives "the operator stopped broadcasting". I reproduced this with `go test -count=300 -cpu=1,2,8 -run 'TestServerSSEStreamsFrames|TestSessionTunnelExitEndsSession'`, which failed once: `TestSessionTunnelExitEndsSession` got `{"reason":"the operator stopped broadcasting"}`. Viewers would see the wrong reason for a tunnel exit or a hidden indicator.
  - Fix: `<-s.opts.Hub.Done()` before `Err()`. This is safe because the only other path that closes the channel is the handler's own `sub.Close`. The alternative is to carry the reason in a final `Message`, which is what the plan said.
  - Rule: any consumer that reacts to a closed channel must read the ending state through an edge that orders it after that state is written.
- **`viewer.js:83-87`: the viewer freezes silently once `EventSource` closes.** Under the HTML spec, a non-200 response (Cloudflare 502/530 during an edge hiccup, 503 when there are too many viewers, 410/404 after the end) closes the `EventSource` for good, with no reconnect. `onerror` then does nothing because `readyState === CLOSED`. The result depends on when it happens:
  - A viewer already watching keeps a frozen screen with no status, and it looks live.
  - A viewer turned away at the cap sees "Connecting…" forever.

  This breaks the spec's "Reconnecting…" contract. The viewer's states (connecting, live, reconnecting, gone, ended) are left implicit (ARCH-ORDER). Fix: in the CLOSED branch, reset or dim the screen and show "Disconnected". Optionally retry with a new `EventSource` and backoff, since the first message after a reconnect is a full render.

### 4. Minor findings
- `session.go:70`: the error from `go srv.Serve(l)` is dropped. If `Serve` dies unexpectedly, not with `ErrServerClosed`, the session still reports itself live.
- The plan's Task 2.2 says to add the node test to the Makefile's test target. Instead it runs through Go's `TestViewerFit`, which skips silently when `node` is missing. The result is equivalent, but no Revision records the change. This is the 2nd `plan-code-drift` finding. Rule: when the implementation departs from a plan step, record it in `## Revisions` in the same commit.
- `viewer.js` fit loop: it assumes xterm updates `.xterm-screen`'s rect synchronously after `options.fontSize`. If it didn't, the loop would compound the scale. The operator's Chrome check suggests it does, but nothing tests it.

### 5. Test coverage notes
- The end-reason tests exist, but only one interleaving is observable. They pass about 299 times in 300, so a green run doesn't prove the reason is right. After the fix, add a test that makes the handler run first, for example by closing the subscription channel before `done` through a hook.
- `viewer.js`'s event handling (`end`, `onerror`, the CLOSED state) has no tests. A small node test with a fake `EventSource`, `Terminal` and `document` would catch the freeze above.

### 6. Architectural notes for upcoming work
- ARCH-DRY: pass. `assets` is the single route table, and `endReason` is shared.
- ARCH-PURE: pass. `nextFontSize` is pure and tested in node; the server and session are thin IO.
- ARCH-PURPOSE: pass for M2's scope. Each Done-when item for this milestone has a test.
- ARCH-MOCK: pass. `FakeTunnel` sits behind the same seam production uses. The live `cloudflared` conformance check is correctly deferred to M4.
- ARCH-CONSTRAINTS: pass. The 16-viewer cap returns 503, `ReadHeaderTimeout` is 10s, the shutdown budget is 2s, and the probe gives up after 30s.
- ARCH-SECURE: pass. Requests are never reflected, the token is never in a body, and the CSP is strict for scripts. One note: `probe` uses the default transport, which honours `HTTPS_PROXY` (loopback is exempt). That is acceptable, but M4 should know it.
- ARCH-ORDER: flagged twice, for the end-reason race and the viewer's implicit connection state above. Session `Stop` and `watch` ordering through `stopOnce` is sound.
- ARCH-FUNERAL: pass. M2 creates nothing durable. The watch goroutine exits on `hub.Done`, and the listener closes on teardown or abandon. The M4 socket directory and pidfile are already planned.
- For M3: the Console should treat `Session.Err()` as the reason shown in its notice. Fixing the race keeps the viewer's reason and the operator's notice in agreement.

### 7. Plan revision recommendations
- Add a `## Revisions` entry: the Hub ends a viewer by closing its channel, not with `Message{End: true}` as Task 1.3 said, and the server reads the reason after `Hub.Done()`.
- Add a `## Revisions` entry: the node fit test runs through Go's `TestViewerFit`, not through a Makefile target, and skips when `node` is missing.

```findings
findings:
  - id: new
    severity: Important
    family: cross-channel-state-read
    title: |
      SSE end reason read via Hub.Err() before hub.done closes; viewers can be told the wrong reason (reproduced flake)
    detail: |
      server.go:131-133 reacts to the closed sub channel, which Hub.end closes inside the loop before run() closes h.done, so Hub.Err() can return nil and the reason becomes "the operator stopped broadcasting". go test -count=300 -cpu=1,2,8 failed TestSessionTunnelExitEndsSession once with exactly that. Fix: wait on <-Hub.Done() before Err(), or carry the reason in a final Message. Rule: a consumer reacting to a closed channel reads ending state through an edge ordered after its write.
  - id: new
    severity: Important
    family: viewer-liveness-visible
    title: |
      viewer.js shows nothing when EventSource closes for good, leaving a frozen screen that looks live
    detail: |
      A non-200 response (Cloudflare 502/530, 503 at the viewer cap, 410/404 after the end) closes EventSource without a reconnect. onerror skips the CLOSED state, so a watching viewer keeps a stale screen with no status, and a viewer turned away at the cap sees "Connecting…" forever. In the CLOSED branch, reset or dim the screen and show "Disconnected", optionally retrying with backoff. Add a node test with a fake EventSource.
  - id: new
    severity: Minor
    family: plan-code-drift
    title: |
      Node fit test runs through Go's TestViewerFit, not the planned Makefile target, and no Revision records it
    detail: |
      2nd finding in this family. Rule: when the implementation departs from a plan step, record it in Revisions in the same commit. Also unrecorded: the hub ends viewers by closing the channel, not with Message End as Task 1.3 said.
  - id: new
    severity: Minor
    family: silent-error-swallow
    title: |
      session.go drops the error from go srv.Serve(l), so a server that dies unexpectedly leaves the session reporting live
```

---

## Re-review — 2026-10-07T19:33:25-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 395 — Couch broadcast: stream the composed Couch screen, view-only, to a browser viewer |
| repo | pair |
| issue file | workshop/issues/000395-couch-broadcast-stream-the-composed-couch-screen-view-only-to-a-remote-couch-watch.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 2992e81d93829fd43e4f77e685501f9c278393d0..042c401ede3aaa337dcdd536089671779bdb160c |
| command | sdlc milestone-close --issue 395 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-10-07T19:33:25-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All four findings from the prior round are fixed, and I checked each against the code rather than the commit messages.
- **BR-5:** the end reason is now stored atomically before any subscriber queue closes (`hub.go:360-370`). With a scratch mutant that ties `Err()` back to `Done`, `TestHubReasonVisibleWhenQueueCloses` fails on iteration 0. With the fix, `TestSession|TestServer` passes 200 times at `-cpu=1,2,8`; the old flake was reproduced at 1 in 300.
- **BR-6:** the viewer's connection handling is now an injected state machine (`viewer.js:42-81`). Node tests cover every terminal state.
- **BR-7:** both departures from the plan are now recorded in Revisions.
- **BR-8:** the session now watches the HTTP server's exit, and `TestSessionEndsWhenServerFails` pins it.

Two new things remain:
- **Important:** when the server fails, the end reason sent to remote viewers includes the raw OS error text, which contains local socket details.
- **Minor:** the atlas still names a test that was renamed.

Neither blocks shipping; both are cheap to fix.

**Strengths**
- `Hub.Err` publishes the reason through an `atomic.Value` before anything is closed (`hub.go:199-207`, `:365`). That fixes the rule, not just the one site: whoever reacts first reads the real state.
- `connect(deps)` keeps the viewer's logic separate from the browser APIs (`viewer.js:42`). The fake EventSource test drives every state: transient error, closed for good, backoff exhausted, turned away at the cap, backoff reset, and end (`connect.test.mjs`).
- The `Tunnel`/`Handle` seam has a stateful `FakeTunnel` (`tunnel_fake.go`), with `IgnoreCancel`, `CloseBlock` and `BreakListener`. These exercise the hard paths: a cancel during start, and the server dying.
- The server is read-only by construction: GET only, a closed asset map, the token compared in constant time, and CSP, `no-store` and `no-referrer` on every response (`server.go:52-96`).
- `TestSessionPersistsNoFrameData` checks the "writes nothing to disk" claim directly.

**Critical:** none.

**Important**
- `session.go:141` wraps the raw `Serve` error as `fmt.Errorf("%w: %v", ErrServerFailed, err)`. Then `server.go:162` (`endReason`) sends `err.Error()` to every remote viewer. The text looks like `the viewer server stopped: accept tcp 127.0.0.1:53211: use of closed network connection`. In M4, with a cloudflared unix socket, it would include a filesystem path, likely with the username in it.
  - This contradicts the comment at `server.go:156-157` ("comes from our own errors").
  - **Fix:** make `endReason` a closed mapping from sentinel to fixed text (`errors.Is` against `ErrIndicatorHidden`, `ErrTunnelExited`, `ErrServerFailed`, `ErrHubClosed`), with a generic fallback.
  - **Test:** end a session with a wrapped error and assert the `end` event's reason is exactly the fixed string (ARCH-SECURE).

**Minor**
- `atlas/broadcast.md:69` says "node-tested via `TestViewerFit`", but the test is now `TestViewerNode` (`viewer_test.go:12`).
  - This is the 3rd finding in family `plan-code-drift`.
  - **Rule:** a rename or departure greps the old identifier across `atlas/`, the plan and lessons in the same commit. The existing lesson (`lessons.md`, last bullet) covers Revisions but not atlas references. Extend it to say "every prose reference to a renamed identifier", and check with `git grep <old-name>` before committing.

**Test coverage**
- I mutation-checked BR-5 (above). BR-8's test can only pass through the new `served` case.
- `TestViewerNode` skips when node isn't installed. That's acceptable, but CI without node would silently skip the BR-6 coverage.

**Architecture**
| Principle | Result | Notes |
|---|---|---|
| ARCH-DRY | pass | `fakeHandle` embeds `localHandle`. |
| ARCH-PURE | pass | `nextFontSize` and `connect` are pure; the server and session are thin shells. |
| ARCH-PURPOSE | pass | Delivers M2's scope. |
| ARCH-MOCK | pass | The live cloudflared conformance check is rightly deferred to M4. |
| ARCH-CONSTRAINTS | pass | Viewer cap, queue depth, 15s ping, 2s shutdown budget, `ReadHeaderTimeout`. |
| ARCH-SECURE | flag | The end-reason leak above. |
| ARCH-ORDER | pass | The hub's watch is an explicit enum. The session's state is just `stopOnce` + `reason` + `done`. The BR-5 test loops 2000 times over the real scheduler. |
| ARCH-FUNERAL | pass | Nothing durable is written. The `watch` goroutine ends whenever the hub ends, and `Stop` always ends the hub. |

**For upcoming work**
- A viewer whose connection drops exactly when the broadcast ends never sees `end`. EventSource can't see the 410 status, so the viewer retries for about 60s and ends on "Disconnected", with the last frame left dimmed rather than reset. That's acceptable under "never looks live". M3/M4 could add a cheap `GET /<token>/status` probe, or reset the screen once backoff is exhausted, so stale content doesn't stay up.
- In M4, the cloudflared `Handle.Close` must also remove the unix socket: the `Tunnel` contract says Close "removes whatever Listen created".

**Plan revisions:** none needed beyond the atlas fix.

```findings
dispose:
  - id: BR-5
    disposition: addressed
    note: |
      Reason stored atomically before queues close (hub.go:365); TestHubReasonVisibleWhenQueueCloses fails at iteration 0 against an Err-gated-on-done overlay mutant; 200x -cpu=1,2,8 session/server stress green.
  - id: BR-6
    disposition: addressed
    note: |
      connect(deps) state machine dims + shows Disconnected/retry on CLOSED (viewer.js:60-78); connect.test.mjs drives CLOSED, cap-refusal, backoff exhaustion via a fake EventSource, run by TestViewerNode.
  - id: BR-7
    disposition: addressed
    note: |
      Plan Revisions 2026-10-07 records the Go-driven node tests; the End-message-to-channel-close departure is recorded at plan line 777.
  - id: BR-8
    disposition: addressed
    note: |
      session.watch selects on served and stops with ErrServerFailed; TestSessionEndsWhenServerFails (BreakListener) passes only through that case.
findings:
  - id: new
    severity: Important
    family: viewer-text-closed-set
    title: |
      endReason forwards the raw wrapped Serve error (local addr/socket path) to remote viewers
    detail: |
      session.go:141 wraps the OS error text into ErrServerFailed, and server.go:162 sends err.Error() in the SSE end event, exposing 127.0.0.1:port today and a unix-socket path (username) with cloudflared in M4. Map sentinels to fixed strings via errors.Is with a generic fallback, and test that a wrapped error yields the fixed text (ARCH-SECURE).
  - id: new
    severity: Minor
    family: plan-code-drift
    title: |
      atlas/broadcast.md:69 still cites TestViewerFit after the rename to TestViewerNode
    detail: |
      3rd finding in plan-code-drift. Rule: a rename or departure greps the old identifier across atlas, plan and lessons in the same commit; extend the lessons.md Revisions bullet to cover atlas references.
```

---

## Re-review — 2026-10-07T19:35:47-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 395 — Couch broadcast: stream the composed Couch screen, view-only, to a browser viewer |
| repo | pair |
| issue file | workshop/issues/000395-couch-broadcast-stream-the-composed-couch-screen-view-only-to-a-remote-couch-watch.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 2992e81d93829fd43e4f77e685501f9c278393d0..c4c000ad1f3aba5035033422482417be5007e218 |
| command | sdlc milestone-close --issue 395 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-10-07T19:35:47-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Verdict: fix then ship (high confidence).** The one blocking issue, BR-9, is fixed: the reason text sent to remote viewers now comes from a fixed list of strings. A regression test catches the old behaviour, so BR-9 can stay addressed. BR-10 is still open: the atlas still names the old test `TestViewerFit` at `atlas/broadcast.md:69`, but the test is now `TestViewerNode` (`cmd/internal/broadcast/viewer_test.go:12`). Since this is the 4th repeat in the `plan-code-drift` family, the fix should be the general rule, not just this one line. BR-10 is Minor and doesn't block. `go test ./cmd/internal/broadcast/` passes at HEAD.

**1. Strengths**
- **The end-reason list:** `server.go:156-181` turns the reason into a fixed list. Wrapped errors are matched with `errors.Is`, and anything unknown gets the generic "the broadcast ended". Error text never leaves the machine (ARCH-SECURE).
- **The BR-9 test:** `TestEndReasonIsAClosedVocabulary` (`server_test.go:330`) covers wrapped and unrecognised errors that contain a username path and `127.0.0.1:port`. It also checks for leaks directly, not just the expected string. Without the fix (`TrimPrefix(err.Error(), …)`), the `ErrServerFailed` case would return the secret path and the test would go red. So this is real regression evidence.
- **The viewer's display:** it renders the reason with `textContent` (`viewer.js:91`), not `innerHTML`, which is a second layer of protection.
- **The lessons rule:** the new entry in `workshop/lessons.md` states the general rule ("text that crosses a trust boundary is a closed vocabulary"), not just the one place it broke.

**2. Critical findings**
None.

**3. Important findings**
None new.

**4. Minor findings**
- **BR-10 is not fixed.** `atlas/broadcast.md:69` still says "node-tested via `TestViewerFit`". The rule from the last round wasn't applied either: the lessons.md bullet ("Record a departure from the plan…") still doesn't mention renames or atlas references. Fix: extend that bullet to say that a rename greps the old name across `atlas/`, the plan and `lessons.md` in the same commit, run that grep for `TestViewerFit`, and update line 69.
- **Stray files (outside the review range, noting only):** the working tree has untracked `hub.go` and `o.json` at the repo root. Make sure neither gets committed with the fix.

**5. Test coverage notes**
- BR-9's coverage is adequate, as described above.
- `TestViewerNode` still skips when `node` isn't installed (already discussed in a prior round, not raised again).

**6. Architectural notes for upcoming work**
- **ARCH-DRY: pass.** The reason list is the single source for viewer-facing text.
- **ARCH-PURE: pass.** `endReason` is a pure function with a table-driven test.
- **ARCH-PURPOSE: pass for BR-9.** I swept every place server.go writes text to a viewer (the `http.Error` calls at lines 60, 79, 112 and 115). They're all fixed literals, so no other channel leaks error text.
- **ARCH-MOCK: pass.** The tunnel still goes through its own seam (`tunnel.go`) with a stateful fake (`tunnel_fake.go`).
- **ARCH-CONSTRAINTS: no change this round.**
- **ARCH-SECURE: pass.**
- **ARCH-ORDER: no change this round.** BR-5's ordering fix stands.
- **ARCH-FUNERAL: no change.** Nothing durable was added this round.
- **For M4:** when cloudflared adds new error types, add each one to `endReasons`. Until then, the generic fallback is a safe default.

**7. Plan revision recommendations**
None. The plan already records the `TestViewerNode` change (plan line 789); only the atlas is out of date.

```findings
dispose:
  - id: BR-9
    disposition: addressed
    note: |
      server.go:156-181 closed endReasons table via errors.Is plus generic fallback; TestEndReasonIsAClosedVocabulary feeds wrapped secret-bearing errors and would fail under the old TrimPrefix(err.Error()) path; other viewer-facing writes (http.Error at server.go:60,79,112,115) are fixed literals.
  - id: BR-10
    disposition: not-addressed
    note: |
      atlas/broadcast.md:69 still reads "node-tested via TestViewerFit" at c4c000ad; lessons.md Revisions bullet was not extended to cover renames/atlas references (4th plan-code-drift instance, fix the rule not the site).
```
