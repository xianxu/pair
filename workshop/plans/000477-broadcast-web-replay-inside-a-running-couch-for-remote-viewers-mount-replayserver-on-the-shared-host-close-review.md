# Boundary Review — pair#477 (whole-issue close)

| field | value |
|-------|-------|
| issue | 477 — broadcast-web: replay inside a running Couch, for remote viewers (mount ReplayServer on the shared Host) |
| repo | pair |
| issue file | workshop/issues/000477-broadcast-web-replay-inside-a-running-couch-for-remote-viewers-mount-replayserver-on-the-shared-host.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5bd8db0d5db11f07e55a654168e5fedb8839ccbc..9b9a11d67fefd9568ef15fe89bebb4428e2ddf05 |
| command | sdlc close --issue 477 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T07:03:31-07:00 |
| verdict | SHIP |

## Review

Still reviewing: BR-15 looks unfixed at head. Next I'm checking whether scroll also leaves replay marks stale, then I'll run the tests.

Generating the runtime bundle so the couchcmd and couchtty tests can compile.

```verdict
verdict: SHIP
confidence: high
```

Issue #477 does what its Spec and Plan promise. Each recording is shared on the shared broadcast Host under its own token, by a `ReplayServer` that serves only that recording. The `ReplayShares` state machine has an explicit transition table and a test for each transition and interleaving. Two broker ops, `replay-share` and `replay-status`, are split so that polling never restarts a share that failed. Both CLIs share one share-and-poll function, with a loopback fallback for replay and a refusal for `query link`. The torn-last-line length advisory is fixed. In M2, the replay viewer draws notes and strokes with #470's `annotate.js`. I checked the window with stat and name-status first, then read the targeted diffs. The touched packages pass under `-race`: webcast/..., couchmessage, couchcmd and couchtty. couchcmd and couchtty only compiled after I generated the runtime bundle into this throwaway checkout (`generatecmd --repo . --out …`). That is a missing build artifact, not a code problem. I did not run the node tests or the live Chrome tests. The only open finding is BR-15, a Minor that is still not fixed at head. It does not block the gate.

1. **Strengths**
   - `cmd/internal/webcast/shares.go:37-43,104-203`: the share states are a written transition table, and the code follows it. `Share` and `Status` are split so that polling only reads. `Close` sets `closed` under the lock before cancelling and waiting, so a `wg.Add` can't race the `Wait`. A mount that completes after Close unmounts itself (`:151-156`).
   - `cmd/internal/webcast/replay.go:668-681`: a scoped server reuses the existing `recordingExists` and `only` check instead of copying `ReplayServer`. One check now covers both shapes of server (ARCH-DRY).
   - `cmd/internal/couchcmd/replay.go:392-394` and `broadcast_list.go:84-87`: `cantShare` and `predatesOp` are each one predicate, used by both CLIs and by `--broadcast-list`.
   - `cmd/internal/webcast/query/cli.go:113-137`: `query` stays independent of Couch because the `Linker` is injected. The link goes alone to stdout and the note to stderr.
   - The tests cover each ordering through `FakeTunnel` seams (`OpenDelay`, `IgnoreCancel`, `Exit`) and the `mounted` hook: one mount while opening, close while opening, close when ready, mount completing after close, and lost then reshared.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - BR-15 is still open. `replay.js:482` passes `win.devicePixelRatio || 1` once, at mount, while the live viewer reads it on every paint (`annotate.js:484`). The fix is to pass a getter, or read `win.devicePixelRatio` inside `draw`. The last two commits (`70309982b`, `9b9a11d67`) don't touch this line.

5. **Test coverage notes**
   - The state machine, the scoped server's path table, `ValidateRequest`, the CLI share-and-poll cases (opening then ready, deadline, no Couch, old Couch, DIR outside the store, tunnel off), the broker round trip and the ready notice are all covered. They all pass under `-race` here.
   - The JS mark drawing is covered by the live Chrome test `TestLiveReplayServerMarks`, which I did not run (it needs `WEBCAST_LIVE_CHROME=1`). The issue log says the resize and card-placement fixes were each mutation-checked against it.

6. **Architectural notes**
   - **ARCH-DRY: pass.** The shared helpers are `recordingExists`, `cantShare`, `predatesOp` and `serveViewerFile`, and the old `drawMarks` was removed in favour of #470's modules.
   - **ARCH-PURE: pass.** Link building (`ReplayLink`), argument parsing and `formatSharedReplays` are pure. The poll gets its clock and `messageCall` injected.
   - **ARCH-PURPOSE: pass.** Both ways of producing a link exist (the Couch CLI and the agent's query CLI), plus the advisory and the drawing. The operator's tunnel smoke test is correctly left to the TL.
   - **ARCH-MOCK: pass.** The Host's tunnel is behind the `Tunnel` seam, with a stateful `FakeTunnel` and `LocalOnly`. The broker is behind `messageCall`.
   - **ARCH-CONSTRAINTS: pass.** The poll runs every 300ms and gives up at 45s, past the 30s probe. Broker requests never wait on the tunnel.
   - **ARCH-SECURE: pass.** One token per recording, with the scoped server returning 404 for every other id, the listing, traversal and encoded slashes. The operator notice and `--broadcast-list` never print the token.
   - **ARCH-ORDER: pass.** The transitions are explicit under the mutex, and the tests control the ordering. One small gap: `onReady` runs after the lock is released (`shares.go:166`), so a notice can still show just after `Close`. That's harmless during teardown.
   - **ARCH-FUNERAL: pass.** Shares live in memory, at most one per recording asked for, and end at Couch's teardown (`console.go:995`). The tunnel's lifetime is written in the plan. There is no expiry or "unshare" verb; the plan flags that as a follow-up for the TL.

7. **Plan revision recommendations:** none. The Revisions section already records the `Share`/`Status` split, the loopback fallback for a DIR outside the store, `ShareStatus.Local` and the torn-line regex, and the code matches them.

```findings
dispose:
  - id: BR-15
    disposition: not-addressed
    note: |
      At head, replay.js:482 still passes win.devicePixelRatio once, at mount, while annotate.js:484 reads it on every paint; no later commit touches it. Minor and non-blocking: pass a getter, or read it inside draw.
```
