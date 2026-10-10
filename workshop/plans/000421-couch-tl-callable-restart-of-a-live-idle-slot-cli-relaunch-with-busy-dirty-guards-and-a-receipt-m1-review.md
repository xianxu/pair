# Boundary Review — pair#421 (milestone M1)

| field | value |
|-------|-------|
| issue | 421 — couch: TL-callable restart of a live idle slot (CLI relaunch with busy/dirty guards and a receipt) |
| repo | pair |
| issue file | workshop/issues/000421-couch-tl-callable-restart-of-a-live-idle-slot-cli-relaunch-with-busy-dirty-guards-and-a-receipt.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | fa6c2152494d727d2354286af20d0158a317ec0a..f4d9e5f3f931d78f496eea72dfec632a88c37ff9 |
| command | sdlc milestone-close --issue 421 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-10-09T23:23:28-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Reviewing the M1 diff; I'm checking whether the turn-end path can flip `Active` false without output (which would leave the settle check stale).

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

M1 delivers the wrapper-side half of the design. There is a pure `wrapperSettled` rule, a sequence-guarded settle timer with an injected clock, and per-connection `hello-v2` negotiation that never shows an old peer an unknown field. The registry keeps a Liveness snapshot that tells "unknown" apart from "busy". It matches the plan's M1 Revisions: `Settled`/`Build` sit on `SessionFrame`, not `Binding`/`Observation`, and the `--actors` column was dropped. One gap should be fixed before M2 builds admission on this signal. The settle check is re-armed only by PTY output and input, but turn state also changes on lifecycle events that carry no PTY bytes: watchdog/grace expiry, transcript completion, and transcript start. So an idle slot can stay "unsettled" indefinitely, and in the other direction a turn can open while the wrapper still says settled.

1. **Strengths**
   - `peer_settle.go:13` is a pure, exhaustively tested rule. It reuses `peerComposerState`, so "empty composer" means one thing (ARCH-DRY).
   - In `settleFired` (`peer_settle.go:56`), sequence tokens before and after the probe make a stale timer harmless. `TestSettleTimerInterleavedActivity` fires stopped timers on purpose to exercise the case where Stop loses the race.
   - Negotiation is placed carefully. `session_transport.go:136` ends a legacy session that claims Settled. The registry keeps `settled` only when `build != nil`, and a new incarnation starts at nil, so it never inherits an old Settled.
   - `Registry.Liveness` drops ambiguous slots instead of guessing (`registry.go:346`).
   - Atlas is updated in the same range (`atlas/couch.md`).

2. **Critical:** none.

3. **Important**
   - `notification_lifecycle.go:244` / `peer_settle.go`: `turnActive` is mirrored into an atomic, but a lifecycle transition never re-arms or unsettles the settle check.
     - `ObservationWatchdogExpired` (60s timer), `ObservationGraceExpired` and `ObservationTranscriptCompletion` clear `Active` without PTY output. A 3s check that saw `Active=true` therefore leaves the slot unsettled until the operator next touches it, and M2 would refuse `busy` on an idle slot.
     - The reverse case is worse. `ObservationTranscriptStarted` can open a turn while `settled=true` stands, and only the PTY output that usually follows corrects it.
     - Fix: in `processLifecycleObservation`, when `state.Active` changes, call into `peerDelivery`. Unsettle at once when a turn opens; re-arm the check when it closes. Add a test for each direction (ARCH-ORDER).

4. **Minor**
   - `peer_runtime.go:158` hashes `os.Executable()` by path, not the image that is running. On macOS, a `make build` landing between exec and this hash records the new binary's hash, which would make the M2 freshness rule look stale. The new `executable` variable also shadows the function's `executable` parameter.
   - `sessionHandshake` wraps an ack-read *timeout* as `errNoAck`. A slow new broker therefore downgrades that connection to legacy hello, and Settled is lost until the next reconnect.
   - The wrapper hashes the whole executable synchronously in `startPeerRuntime`, which is on the wrapper startup path. That costs tens of milliseconds per launch and has no measurement behind it (ARCH-CONSTRAINTS).
   - `publishUnsettled` takes a `changed` flag; inlining the flag at its 4 call sites would read more simply.

5. **Test coverage**
   - Covered: a pure rule table; timer interleavings through a fake clock; the old/new negotiation pairings and frame validation; registry Liveness.
   - Not covered: lifecycle-driven settle transitions (the Important finding above).
   - The wrapcmd tests pass. `TestSessionClientReconnectsAfterBrokerRestart` and `TestSessionServerCapacity` failed in this sandbox with a `/tmp` mkdir "operation not permitted", which is an environment limit rather than a regression.

6. **Architecture**
   - **ARCH-DRY:** pass. The composer recognizer is reused.
   - **ARCH-PURE:** pass. The rule is pure, and the probe and timer are injected.
   - **ARCH-PURPOSE:** pass for M1's scope.
   - **ARCH-MOCK:** pass. The in-memory transport and a fake timer are used.
   - **ARCH-CONSTRAINTS:** minor flag on the startup hash.
   - **ARCH-SECURE:** pass. `BuildIdentity.Validate` bounds the hash and revision fields, and frames decode strictly.
   - **ARCH-ORDER:** flag. Settled is derived from lifecycle state but does not subscribe to that state's transitions.
   - **ARCH-FUNERAL:** pass. Everything is in memory, and a session's state dies with it.

   For M2: decide whether `busy-unknown` versus `busy` should also depend on how old the last Settled report is. Today a stale `true` from before an unobserved turn-open would be trusted.

7. **Plan revisions:** none needed for M1. The M1 implementation deltas already record the field placement and the dropped `--actors` column.

```findings
findings:
  - id: new
    severity: Important
    family: derived-state-misses-source-transitions
    title: |
      Settle check is not re-armed or unsettled on lifecycle turn transitions that emit no PTY bytes
    detail: |
      Watchdog, grace and transcript completion clear Active without output, so an idle slot stays unsettled (false busy). Transcript start can open a turn while settled=true stands. processLifecycleObservation should unsettle on open and re-arm on close, with a test for each direction.
  - id: new
    severity: Minor
    family: build-identity-from-path-not-image
    title: |
      peer_runtime.go hashes os.Executable() by path, so a rebuild between exec and the hash reports the new binary
    detail: |
      It also shadows the executable parameter. This matters for M2's stale-binary rule.
  - id: new
    severity: Minor
    family: negotiation-fallback-on-timeout
    title: |
      An ack-read timeout is treated as errNoAck, so a slow new broker downgrades the connection to legacy hello
  - id: new
    severity: Minor
    family: startup-path-unmeasured-work
    title: |
      The full sha256 of the executable runs synchronously in startPeerRuntime on wrapper startup, with no measured budget
```

---

## Re-review — 2026-10-09T23:27:35-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 421 — couch: TL-callable restart of a live idle slot (CLI relaunch with busy/dirty guards and a receipt) |
| repo | pair |
| issue file | workshop/issues/000421-couch-tl-callable-restart-of-a-live-idle-slot-cli-relaunch-with-busy-dirty-guards-and-a-receipt.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | fa6c2152494d727d2354286af20d0158a317ec0a..ef0e61f28c9909674768fce988a66a9412ecd63e |
| command | sdlc milestone-close --issue 421 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-10-09T23:27:35-07:00 |
| verdict | SHIP |

## Review

Focus on the last fix commit to verify dispositions.

Sandbox /tmp failures; rerun with TMPDIR set.

```verdict
verdict: SHIP
confidence: medium
```

All five prior findings are dealt with at this boundary.

- **BR-1:** the superseding revisions are now marked inline on the M1 and M3 rows (plan:104 and plan:124).
- **BR-2:** fixed. `processLifecycleObservation` now uses `turnActive.Swap` to detect a turn opening or closing and calls `peerDelivery.lifecycleTurnChanged`. Opening unsettles at once; closing re-arms the check. A new test covers both directions.
- **BR-3 and BR-5:** the executable is hashed once, first thing in `run`, before anything is spawned. The variable shadowing is gone, and the race window is down to the time between exec and the first statement.
- **BR-4:** only EOF, ErrUnexpectedEOF or ECONNRESET now counts as an old broker. A new test shows that a timeout no longer falls back to plain hello. With the old code that test would fail, because every read error became `errNoAck`.

One new Minor finding: the settle check is matched to activity by sequence number, and a silent turn change does not advance it. A check already in flight can therefore overwrite the unsettle with `Settled=true` for up to one 3-second interval. It doesn't block the gate, but M2 should close it before it relies on Settled to admit a restart.

**Tests:** wrapcmd and couchmessage tests that create directories under a hardcoded `/tmp` fail in the sandbox with "operation not permitted". This is the sandbox, not a regression; those `/tmp` paths already existed before this range. I did not re-run those tests unsandboxed.

1. **Strengths**
   - `wrapperSettled` is one pure rule (`peer_settle.go:18`), and the live probe that feeds it is a thin shell.
   - Stale checks are discarded by sequence number, and the timer is injected through `afterFunc`, so tests control when it fires.
   - Old wrappers and old brokers are both handled: `session_transport.go:293-311` redials with plain hello and never sends Settled on that connection, and the negotiation tests cover both mixes.
   - The new lesson (lessons.md:790) states the general rule, not just the one site.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `peer_settle.go:100` `lifecycleTurnChanged` re-arms the check but does not advance `d.sequence` or a separate settle counter. A `settleFired` that has already run its probe can still land `Settled=true` after a silent turn opened. Fix sketch: add a settle-only counter that every source of Settled bumps (output, input, lifecycle), and have `settleFired` check that counter instead of `sequence`.
   - `notification_lifecycle.go:246`: the line that calls `lifecycleTurnChanged` from `processLifecycleObservation` is untested. The new test calls the method directly.

5. **Test coverage notes:** the negotiation-timeout test would fail without the fix. The BR-2 test covers the method's behavior in both directions, but checks are fired one after another, so it never sees the in-flight race above.

6. **Architecture**
   - **ARCH-DRY:** pass. Unsettling goes through the one `publishUnsettled` helper.
   - **ARCH-PURE:** pass. The rule is pure and the probe is a thin shell.
   - **ARCH-PURPOSE:** pass for M1.
   - **ARCH-MOCK:** pass. Tests use a real `net.Pipe` and a unix-socket broker rather than mocks.
   - **ARCH-CONSTRAINTS:** pass. The roughly 10ms hash is stated, though not measured in a test.
   - **ARCH-SECURE:** pass. hello-v2 frames are validated with size bounds.
   - **ARCH-ORDER:** flagged (the Minor above). The settle check's matching ignores lifecycle transitions.
   - **ARCH-FUNERAL:** pass. Nothing new is durable: timers die with the delivery.

7. **Plan revisions:** none needed.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      plan:104 and plan:124 now carry inline superseded pointers to PQ-2 / PQ-4.
  - id: BR-2
    disposition: addressed
    note: |
      notification_lifecycle.go:246 Swap + lifecycleTurnChanged; TestSettleFollowsSilentTurnTransitions covers open and close.
  - id: BR-3
    disposition: addressed
    note: |
      selfBuildIdentity runs first in run() (wrap.go:2693), before anything is spawned; the shadowing is removed.
  - id: BR-4
    disposition: addressed
    note: |
      Only EOF/UnexpectedEOF/ECONNRESET map to errNoAck; TestNegotiationTimeoutDoesNotFallBack would fail on the old code.
  - id: BR-5
    disposition: addressed
    note: |
      The hash runs once at run() entry with a stated ~10ms cost; it is no longer in startPeerRuntime.
findings:
  - id: new
    severity: Minor
    family: derived-state-misses-source-transitions
    title: |
      A silent turn change does not advance the settle generation, so an in-flight check can re-assert Settled=true
    detail: |
      This is the 2nd finding in this family. The rule: every source transition of Settled (output, input, lifecycle open/close) must both re-arm the check AND invalidate any check already in flight. lifecycleTurnChanged re-arms but leaves d.sequence alone, so a settleFired whose probe ran before turnActive.Swap still passes its seq check and publishes Settled=true during an open turn, for up to SettleInterval. Fix: give settle its own counter that every source bumps, and check it in settleFired.
```
