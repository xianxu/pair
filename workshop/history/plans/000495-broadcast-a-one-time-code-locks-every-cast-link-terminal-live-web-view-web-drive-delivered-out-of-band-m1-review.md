# Boundary Review — pair#495 (milestone M1)

| field | value |
|-------|-------|
| issue | 495 — broadcast: a one-time code locks every cast link (terminal LIVE, web view, web drive); delivered out of band |
| repo | pair |
| issue file | workshop/issues/000495-broadcast-a-one-time-code-locks-every-cast-link-terminal-live-web-view-web-drive-delivered-out-of-band.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 6212e682e076b38e47a93f8f0fd8d05294b0033e..deb579d3f5b72532de390f4ed0134e49c76b0ca1 |
| command | sdlc milestone-close --issue 495 --milestone M1 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T14:37:19-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reviewing the gate and tests now; checking webcast wiring and the M1 test coverage next.

Environment can't build couchtty (missing generated runtime assets); checking a couple of remaining edge cases.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M1 delivers what it set out to do. There is a pure `castlock` core. A single gate in `Host.ServeHTTP` stops any mount from seeing a request that has no session. The view and pointer links each get their own lock, created at the same moment as their token. Admitted requests run under a context that is cancelled on revoke, so an operator rotation ends live event streams, and a test proves it. These all pass under `-race`: `go test ./cmd/internal/castlock ./cmd/internal/broadcast ./cmd/internal/webcast`, plus 84 node tests in `tests/broadcast-viewer/*.test.mjs` and `tests/webcast/drive.test.mjs`.

Two things should be fixed before crossing the boundary, and neither blocks:
- **Cookie mode differs from the plan.** The plan's PQ-5 revision chose a `SameSite=Lax` session cookie; the code ships `Strict`.
- **Rotation callbacks can't be ordered.** They carry no generation, so a consumer can end up showing a stale code.

I could not run `cmd/internal/couchtty` or `cmd/internal/artifactpath`. This checkout is missing the generated `runtimebundle/assets/runtime/files`, so both fail to set up. The cause is the environment, not this diff. Those changed couchtty tests were read, not executed.

### 1. Strengths
- **One enforcement point (ARCH-SECURE).** `host.go:296-304` consults `LockFor` before any mount sees a request. The 401 response is the fixed text `locked`, and `TestLockedLinksRefuseEveryRouteWithoutASession` checks it on every asset and route for no session, a forged session and a revoked session. That includes a direct `GET events`.
- **Lock created with its link (PQ-1).** For the pointer link, the token and lock are minted in one critical section (`pointer.go:95-106`), and the token never changes after that. `LockFor` therefore never sees a minted token without its lock. The view lock is created before `Mount`, and `End`s if `Mount` fails (`session.go:150`).
- **Careful core.** Sessions are stored hashed, codes compared in constant time, and input is bounded (`MaxCodeInput`, a 128-byte cap on sessions, `maxUnlockBody`). Callbacks run outside the mutex. A guess-rotation keeps the existing revoke channel, so viewers stay connected.
- **Strong invariant test.** `castlock_test.go:174` runs a seeded random sequence of operations. It checks invariants stated independently of the implementation, including sessions never crossing between locks and revoked sessions never validating, and every failure names its seed.
- **Teardown order is thought through.** In `session.go:255-259` the locks end after the hub's end event, so viewers still see "Broadcast ended" rather than a bare disconnect.

### 2. Critical
None.

### 3. Important
- **The cookie is `Strict`, but the plan chose `Lax`.**
  - Where: `lockgate.go:107` and the atlas cookie paragraph.
  - The plan's Revisions PQ-5, and the plan-gate dispositions at lines 63 and 88, settle on `Lax`. The code, its test (`lockgate_test.go`, which asserts `SameSiteStrictMode`) and the atlas all say `Strict`, with the re-prompt documented as a "trade-off".
  - Effect: a viewer who re-opens the link from a chat app is asked for the code again.
  - D1 is still formally open, and the issue's `## Log` is empty, so no decision has been recorded.
  - Fix: switch to `http.SameSiteLaxMode`, update the test and the atlas, and record D1. Or, if the operator chose Strict, add a Revisions entry saying so.
- **Rotation callbacks can arrive out of order (ARCH-ORDER).**
  - Where: `castlock.go:142-147,199-201`.
  - `OnRotated` runs after the mutex is released. A guess-rotation and an operator rotation happening together can therefore deliver their `Rotation{Code}` in reverse order.
  - `Rotation` has no generation or sequence number, so the M3 Couch consumer can't detect a stale code. It would show a code that no longer works, or flash the wrong one.
  - Fix: add a monotonic `Gen uint64` to `Rotation`, stamped under the mutex. Alternatively, document that consumers must re-read `Code()` or `Session.Codes()`. Add a concurrent test.

### 4. Minor
- **Nothing shows the operator the code yet.** `Codes`, `RotateCode`, `OnCodeRotated` and webcast `WithLocks` have no production consumers. Until M3, the LIVE and pointer links are locked behind a code nothing displays, so the branch must not merge before M3.
- **Revoke behaviour differs from the plan.** The plan says a revoke sends `event: locked`; the code instead cancels the context, and the page then probes `GET session`. The atlas describes this correctly; the plan still says the old thing.
- **The planned `sessionFrom(r)` seam doesn't exist.** The cookie is read inline in `gate` (`lockgate.go:62`).
- **The fuzz target isn't the one PQ-4 named.** PQ-4 asked to fuzz the unlock *body decoder*; `FuzzUnlock` fuzzes `castlock.Unlock`, not the JSON or body path in `lockgate.go:unlock`.
- **A failed guess-rotation is silent.** If `newCode` fails, it is dropped (`castlock.go:139`); `wrong` stays at or above the budget, and every later wrong try retries the rotation. It's harmless, but should be logged or commented.
- **Wrong refusal when the token can't be generated.** `Unlock` returns `Ended`, which shows "This cast has ended."
- **A session evicted by the 64-session cap keeps its live stream until it reconnects.** Acceptable, but undocumented.
- **A test mutates global state.** `couchtty/cast_lock_helpers_test.go` sets `http.DefaultClient.Jar` package-wide in `init`. Every test in the package now shares one cookie jar, which could hide ordering dependencies.

### 5. Test coverage notes
Covered:
- Every route refused without a session.
- A direct `GET events`.
- Wrong, malformed and non-JSON unlocks.
- The rate limit, with `Retry-After`.
- A budget rotation that keeps an existing viewer.
- A rotation that cuts a live stream.
- The view session refused on the pointer link.
- The lock existing from the first request (PQ-1).
- Node tests for fragment parsing, the unlock call and the session probe.

Gaps:
- No concurrent unlock-versus-rotate test, or concurrent rotation-callback test. The single mutex makes ordering total, but the callback ordering above is untested.
- The end-to-end fragment flow in `start()`: auto-submit, then `replaceState`.
- The couchtty changes didn't build here, so they are unverified in this environment.

### 6. Architectural notes
| Principle | Result | Notes |
|---|---|---|
| ARCH-DRY | pass | `NewLinkLock` is the one constructor. The couchtty `openViewer` and `unlockCastLink` code lookups repeat each other; a small `codeFor(s, link)` helper would remove that. |
| ARCH-PURE | pass | `castlock` has no IO; its randomness and limiter are injected. The gate is a thin HTTP shell. |
| ARCH-PURPOSE | pass for M1 | The Couch display is M3, by plan. Watch that the branch doesn't merge early. |
| ARCH-MOCK | pass / N/A | No new external dependency; tests use `FakeTunnel` and real httptest servers. |
| ARCH-CONSTRAINTS | pass | The rate limit is enforced per lock. Note that a link holder can keep it saturated, so legitimate joiners get 429s. That is within PQ-3's accepted trade-off. |
| ARCH-SECURE | pass, with the cookie drift above | |
| ARCH-ORDER | flag | Rotation-callback ordering (Important above). Otherwise state changes go through one mutex, with a generation channel snapshotted at `Valid`. |
| ARCH-FUNERAL | pass | Sessions are capped in memory; locks end at teardown or on mount failure. `WithLocks` locks for webcast have no end path yet, which M2 must add. |

### 7. Plan revision recommendations
- Record D1: the operator's decision on the cookie (Lax, as PQ-5 says, or Strict). Rewrite the Shape "Session transport" body to match.
- Revocation: replace "sends `event: locked`, then closes" with "context cancelled on revoke; the page probes `GET session` (204 or 401) and reloads to the lock page."
- Either restore the `sessionFrom(r)` seam, or note that cookie reading is inlined in `gate`.
- PQ-4: name the fuzz targets that actually exist, or add one for the unlock body decoder.

```findings
findings:
  - id: new
    severity: Important
    family: spec-moves-with-decisions
    title: |
      The session cookie ships SameSite=Strict, but the plan's PQ-5 revision chose Lax (D1 not recorded)
    detail: |
      lockgate.go:107, its lockgate_test assertion and the atlas all say Strict, while plan Revisions PQ-5 and the gate dispositions say Lax. Viewers who re-open the link from chat are asked for the code again. Switch to Lax, or record the operator's D1 decision for Strict in Revisions.
  - id: new
    severity: Important
    family: unordered-async-notifications
    title: |
      castlock OnRotated callbacks run outside the mutex and carry no generation, so concurrent rotations can deliver a stale code last
    detail: |
      castlock.go:142-147 and 199-201. A guess-rotation racing an operator Rotate can call OnRotated in reverse order. The M3 Couch consumer would then show or flash a dead code. Add a monotonic Gen to Rotation, stamped under the mutex, or require consumers to re-read Code(). Add a concurrent test.
  - id: new
    severity: Minor
    family: milestone-leaves-feature-unusable
    title: |
      M1 locks the LIVE and pointer links, but nothing shows the operator the code until M3
    detail: |
      Codes, RotateCode, OnCodeRotated and WithLocks have no production consumers. Don't merge the branch before M3.
  - id: new
    severity: Minor
    family: spec-moves-with-decisions
    title: |
      The plan still says revoke sends event: locked and that a sessionFrom(r) seam exists; the code does neither
    detail: |
      The code cancels the request context and the page probes GET session. The cookie is read inline in gate.
  - id: new
    severity: Minor
    family: fuzz-target-mismatch
    title: |
      PQ-4 asked to fuzz the unlock body decoder; FuzzUnlock fuzzes castlock.Unlock instead
  - id: new
    severity: Minor
    family: silent-error-swallow
    title: |
      A failed guess-rotation newCode error is silently dropped, and a token-generation failure is reported as "cast has ended"
  - id: new
    severity: Minor
    family: test-global-state
    title: |
      couchtty test init sets http.DefaultClient.Jar package-wide, sharing one cookie jar across every test
```

---

## Re-review — 2026-10-12T14:44:53-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 495 — broadcast: a one-time code locks every cast link (terminal LIVE, web view, web drive); delivered out of band |
| repo | pair |
| issue file | workshop/issues/000495-broadcast-a-one-time-code-locks-every-cast-link-terminal-live-web-view-web-drive-delivered-out-of-band.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 6212e682e076b38e47a93f8f0fd8d05294b0033e..e137bb9d7e2e419d3505a519f4e8719ad06f4a75 |
| command | sdlc milestone-close --issue 495 --milestone M1 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T14:44:53-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

The last commit is the fix round. Let me see it and the plan.

I've read the fix commit, plan, lock core and gate. Next I'll run the tests and revert-check the claimed fixes.

Tests pass and the revert checks went red as expected. Now checking the Go version and the remaining plan-body findings.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M1 is in good shape. Every route on the terminal LIVE and pointer links now goes through a single gate in `Host.ServeHTTP`, before any mount sees the request. The `castlock` core is pure: randomness and the rate limiter are injected, and the clock is passed in as an argument. The castlock, broadcast and webcast Go tests pass, and so do the 84 node tests. I ran `FuzzUnlockBody` for 10 seconds (about 266k runs) with no failures. Of the nine prior findings, the behavioural fixes (BR-4, BR-7, BR-8, BR-9) are confirmed. I reverted the fail-closed fix and the `Failed` fix in the checkout, and both regression tests (`TestSpentCodeThatCannotRotateFailsClosed`, `TestSessionMintFailureIsNotEnded`) went red, then I restored the file. Only BR-1 and BR-6 are still open, and both are Minor. The plan's body still describes the design that was replaced, and the new decisions appear only in Revisions. One thing I could not check: `cmd/internal/couchtty` does not build in this checkout because `runtimebundle/assets` is a git-ignored generated directory that is missing here. So the couchtty test changes for BR-9 were read but not run, which is why confidence is medium.

**1. Strengths**
- **One gate:** the lock is enforced in one place, `host.go` `ServeHTTP` → `gate`. Each mount answers `LockFor` with its lock, and the locks are created before `Mount`, so an owned link can never be served unlocked (PQ-1 is delivered). `lockgate_test` checks every route with no session, a forged session and a revoked session.
- **Revocation is bounded:** `lockgate.go:69-77` runs each admitted request under a context that is cancelled on revoke. The goroutine that watches for revoke ends with the request, because `done()` is deferred in `ServeHTTP`.
- **Core rules:** `castlock.go` compares codes in constant time and bounds input size (`MaxCodeInput`, a 128-byte cap on sessions). A guess-rotation keeps existing sessions, and the code now fails closed when a spent code cannot rotate. `Gen` is stamped under the mutex, at `castlock.go:169` and `castlock.go:217`.
- **Teardown order:** `session.go` `teardown` ends the locks after the mount is unmounted, so viewers still see "Broadcast ended". A failed `Mount` ends the view lock.
- **Test client:** tests no longer set a cookie jar on `http.DefaultClient`. A grep for `DefaultClient.Jar` across `cmd/` finds nothing.

**2. Critical:** none.

**3. Important:** none new. BR-3 is addressed: plan Revisions line 139 records D1 as Strict, and `lockgate.go:121` and the atlas agree with it. The only record that the operator confirmed it is that Revisions entry; the issue's Log is empty.

**4. Minor:** BR-1 and BR-6 are still open (see the dispositions below). The test helper that unlocks a link by looking up its code is written twice in couchtty: once in `openViewer` (`console_broadcast_test.go`) and once in `unlockCastLink` (ARCH-DRY, test-only).

**5. Test coverage notes**
- No test covers `Gen` on a guess-rotation. `TestConcurrentRotationsNewestGenIsCurrentCode` only runs operator `Rotate`.
- At the gate, `castlock.Failed` maps to 500 (`lockgate.go:131`), but only the core outcome is tested; no HTTP test checks the 500.
- `FuzzUnlockBody` is unlikely to ever reach the successful path. Its seed code is `1234`, which matches the random 4-digit code about once in 10,000 runs.
- I did not run couchtty, as explained above.

**6. Architecture.** All eight markers pass.
- **ARCH-DRY:** passes, apart from the test-only helper noted in section 4.
- **ARCH-PURE:** passes. `castlock` is pure and its dependencies are injected.
- **ARCH-PURPOSE:** passes for M1's scope. The gap that M1 leaves the code invisible to the operator is now handled by the plan's "lands only after M3".
- **ARCH-MOCK:** nothing new to fake. Tests use `FakeTunnel`.
- **ARCH-CONSTRAINTS:** passes. Unlocks are rate-limited (a burst of 5, then 1/s) on top of the 8-try budget, and sessions are capped at 64, evicted oldest first.
- **ARCH-SECURE:** passes. The cookie is HttpOnly and scoped to the link's path, the unlock body is capped at 256 bytes and its content type is checked, and refusal messages are fixed text that never echoes the request.
- **ARCH-ORDER:** passes. One mutex orders unlocks against rotations, and `Gen` orders the callbacks.
- **ARCH-FUNERAL:** passes. The locks end in `teardown`, and the pointer lock ends through `endLock`.

For M3: the Couch consumer has to keep the highest `Gen` it has seen, as the plan says (line 144). When a code can't rotate (fail-closed), the operator gets no signal. That is fine on Go 1.26, where `crypto/rand` cannot return an error. If a `Rand` source that can fail is ever injected in production, M3 should surface it.

**7. Plan revisions.** Do one rewrite of the plan body that brings each paragraph in line with Revisions. The list is in BR-1's note below. The status line is also stale: it still says "draft … waiting on the operator".

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Body still states replaced designs (Host.Lock at line 10, Unlock-to-(session,ok) and Rotate(revoke) and Revoked() at 14-16, event: locked at 29-30, name on web links only at 37, inline codes and click rotation at 55-63, M3 click-to-rotate at 78, the draft status line at 3). The rule from the 2nd finding still applies: rewrite each body paragraph when a revision replaces it.
  - id: BR-2
    disposition: addressed
    note: |
      D1 is recorded as Strict with no expiry (plan line 139), so M2's credentials change is no longer waiting on a decision.
  - id: BR-3
    disposition: addressed
    note: |
      Plan Revisions line 139 records D1=Strict, superseding PQ-5. It matches lockgate.go:121, lockgate_test:156 and the atlas trade-off. The issue Log has no separate record of the operator's answer.
  - id: BR-4
    disposition: addressed
    note: |
      Rotation.Gen is stamped under the mutex (castlock.go:169,217), with the concurrent test TestConcurrentRotationsNewestGenIsCurrentCode. No test covers Gen on a guess-rotation.
  - id: BR-5
    disposition: addressed
    note: |
      Plan line 146 commits to landing the branch only after M3, the remedy the finding asked for. This is a prose-only change and correct.
  - id: BR-6
    disposition: not-addressed
    note: |
      Revisions (lines 140-143) now describe revocation as built, but body lines 29-30 still say event: locked and line 105 the sessionFrom(r) seam. Same rule as BR-1, so fix it in the same body rewrite.
  - id: BR-7
    disposition: addressed
    note: |
      FuzzUnlockBody drives the unlock HTTP handler. It ran for 10s (about 266k runs) and passed.
  - id: BR-8
    disposition: addressed
    note: |
      A spent code that cannot rotate now fails closed, and a mint failure is Failed (500). Both tests went red when I reverted the fix. The rotation error still goes unreported, which is harmless on Go 1.26, where crypto/rand cannot fail.
  - id: BR-9
    disposition: addressed
    note: |
      Tests use a package-level castClient in both packages, and no DefaultClient.Jar remains in cmd/. I could not run couchtty here because its generated runtimebundle assets are missing.
```
