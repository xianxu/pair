# Boundary Review — pair#412 (milestone M2)

| field | value |
|-------|-------|
| issue | 412 — Couch broadcast: remote pointer link (tap and draw fading marks on the operator's screen) |
| repo | pair |
| issue file | workshop/issues/000412-couch-broadcast-remote-pointer-link-tap-and-draw-fading-marks-on-the-operator-s-screen.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 6fafa5b1c88a421b8faf3f78541a5f57a6ec519e..f18a08fa600ccd71b0aec9ca11a7c706446c809e |
| command | sdlc milestone-close --issue 412 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-10-08T10:04:34-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

M2 delivers what the plan claims for this boundary.

- **Point parsing and rate limit:** `PointBatch`/`ParsePointBatch` is a strict parser that echoes nothing, and `RateLimit` is a pure token bucket with an injected clock.
- **Hub pointer watch:** it has its own grace clock and never ends the hub. `Current()` reports the operator's latest frame.
- **Session:** a separate 256-bit pointer token, minted once and reused across off/on. `acceptPoint` checks pointing, grid size, frame class and the active `👆` marker.
- **Server:** exactly one body-reading route (`POST /<pointer-token>/point`), with the order 403 → 415 → in-flight 429 → rate 429 → read deadline → `MaxBytesReader` 413 → parse 400. `caps` goes only to pointer streams, after `theme`, and on every flip.

`go test -race ./cmd/internal/broadcast` passes (73s). `TestProductionArtifactReferencesAreExactlyClassified` in `artifactpath` fails, but only on unrelated files (`wrapcmd/peer_*`, `nvim/review*`). The two files this diff adds (`point.go`, `pointer.go`) are classified correctly, so I take it as pre-existing on main. Nothing blocks SHIP. What's left is one atlas sentence that disagrees with the code and a narrow ordering gap in `PointerState`.

1. **Strengths**
   - `server.go:76-83`: the token is matched before the method check, and only `POST` + pointer token + `point` reaches a body read. Every other path keeps #395's "GET only, no body read". `TestPointerRouteTable` pins this across methods × routes × {view, pointer, wrong} tokens.
   - `pointer.go`: `PointerState` is a leaf lock. Callbacks run after it is released (`TestPointerCallbackMayReenter`). The `changed` channel is closed and replaced on each flip, which gives a race-free signal for `caps`.
   - `hub_test.go` `hubInterleaving`: the random property test now covers arm/disarm/fire on an independent clock. It checks a model invariant (the callback fires iff armed and the marker is hidden; the hub never ends), which is stronger than replaying the implementation.
   - The hardening fixed a real #395 gap: a viewer that stops reading is dropped by a write deadline (`TestServerDropsStalledViewer`). A slow body can't hold an in-flight slot either (`TestPointerSlowBodyDoesNotStarve`). There is also a fuzz target for the parser.
   - `session.go` `OnPointerHidden = go s.pointerHidden()` keeps the hub goroutine from blocking or re-entering the hub.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - **Atlas disagrees with the code:** `atlas/broadcast.md` step 5 says "a 5s read deadline", but `pointReadBudget` is 2s (`point.go:20`), per the security-review revision. The atlas also doesn't mention the new 2×ping write deadline on event streams.
   - **`PointerState` has no stopped or generation state (ARCH-ORDER):**
     - `EnablePointer` after `Stop` mints a token and returns a link to a server that is shutting down.
     - A late `pointerHidden` goroutine (the watch fires, then the operator clicks off and on before that goroutine runs) turns off a newly re-enabled pointing session while the hub watch stays armed.
     - Both are practically unreachable today: the first depends on M3's live-phase guard, the second on two clicks within goroutine-scheduling latency.
     - The rule: each pointer transition should carry the state it was decided against, either a stopped flag that refuses `set(true)` or a generation counter checked by `pointerHidden`.
   - **BR-1 still open:** the plan's tasks still list test cases in prose. It doesn't block anything.

5. **Test coverage**
   - Every Done-when item that M2 owns has a named test:
     - drops for stale size, private frame and hidden marker (`TestPointerPostDropped`)
     - 403 while off, and still a view link (`TestPointerPostWhileOff`)
     - `caps` on join and on flip, never on a view stream (`TestPointerCapsEvents`)
     - both links end with the broadcast (`TestPointerLinkEndsWithBroadcast`)
     - limits (`TestPointerPostRejects`, `TestPointerRateLimit`, `TestPointerContentTypeParsing`)
   - No test covers `EnablePointer` after `Stop`, or the late-`pointerHidden` ordering (see the Minor above).

6. **Architecture**
   - **ARCH-DRY: pass.** `splitPath` replaces `authorized`; the pointer watch reuses the `watch` enum. Its arm/disarm code repeats LIVE's in parallel, which is acceptable at two instances.
   - **ARCH-PURE: pass.** The parser, `RateLimit` and `PointerShown` are pure; the server and session are thin.
   - **ARCH-PURPOSE: pass** for M2's scope.
   - **ARCH-MOCK: pass.** It uses an in-process `httptest`/hub with manual clocks; nothing external is new.
   - **ARCH-CONSTRAINTS: pass.** The envelope is enforced: 4 KB, 64 points, 30/s, 4 in flight, 2s read, 8 KB headers, 60s idle.
   - **ARCH-SECURE: pass.** Constant-time compare on both tokens, typed parse at the boundary, fixed error text, and a fuzz target.
   - **ARCH-ORDER: minor flag** (above). Otherwise the hub watch states are explicit, and the property test exercises interleavings.
   - **ARCH-FUNERAL: pass.** The token and state live in memory and die with the session; nothing is persisted.
   - For M3: the Console must re-check its phase and the frame class under `marksMu` before `Marks.Add`. The plan's Revisions already records this as the L1 contract. Keep that guard in M3's tests in both orders.

7. **Plan revisions:** none needed. The code matches the Core-concepts rows that M2 claims. Fix the atlas's 5s to 2s as a prose correction.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      Plan tasks 1.3/2.2/2.3/3.2 still enumerate test cases in prose; Minor, non-blocking.
findings:
  - id: new
    severity: Minor
    family: docs-match-code
    title: |
      atlas/broadcast.md says the POST read deadline is 5s; code (pointReadBudget) is 2s
    detail: |
      The security-hardening revision cut the budget to 2s but the atlas step 5 still says 5s; the new 2x-ping event-stream write deadline is also undocumented there.
  - id: new
    severity: Minor
    family: stale-observation-acts-on-new-generation
    title: |
      PointerState has no stopped/generation state: EnablePointer after Stop mints a link, and a late pointerHidden can turn off a re-enabled pointing
    detail: |
      ARCH-ORDER. pointerHidden runs in a spawned goroutine and applies set(false) to whatever generation is current; EnablePointer after Stop succeeds against a shutting-down server. Practically unreachable today (needs M3 phase guard / sub-ms double click); fix by refusing set(true) once stopped and tagging the hub fire with an arm generation.
```
