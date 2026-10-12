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

---

## Re-review — 2026-10-12T07:22:30-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 477 — broadcast-web: replay inside a running Couch, for remote viewers (mount ReplayServer on the shared Host) |
| repo | pair |
| issue file | workshop/issues/000477-broadcast-web-replay-inside-a-running-couch-for-remote-viewers-mount-replayserver-on-the-shared-host.md |
| boundary | whole-issue close |
| milestone | — |
| window | ab270639baac60b954b18c4684c00983a2f426a4..d159ddfaa3d8fed112faee674cb6ce8eaa9ceefa |
| command | sdlc close --issue 477 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T07:22:30-07:00 |
| verdict | REWORK |

## Review

Checking a failing test that's outside the review window, to see whether the interdiff is to blame.

I've finished reviewing. The merged head fails four Go tests, which blocks shipping.

```verdict
verdict: REWORK
confidence: high
```

This window is the post-close interdiff, which merges main into the #477 branch. It touches one test file (whose conflict was resolved by hand), two workshop ledger/review files, and a reordering in `lessons.md`. The hand-resolved conflict in `cmd/internal/webcast/replay_live_test.go` is good: it keeps both sides' intent. The merge still leaves the head broken. Main's #474 (`bb95a35af`) retired the old prototype fixtures `app`, `blog` and `stress`. The resolution moved only `replay_live_test.go` to the new fixtures (`rec-app`). Three other #477 test sites still copy the deleted `blog` fixture, so `go test ./cmd/internal/webcast/` fails at the pinned head:

```
--- FAIL: TestRecordingServerServesOnlyItsRecording   replay_server_test.go:235: stat .: no such file or directory
--- FAIL: TestReplaySharesShareOneRecordingPerToken   shares_test.go:62:        stat .: no such file or directory
--- FAIL: TestReplaySharesCloseWhileOpening           shares_test.go:171:       stat .: no such file or directory
--- FAIL: TestReplaySharesSharedAndOnReady            shares_test.go:244:       stat .: no such file or directory
FAIL	github.com/xianxu/pair/cmd/internal/webcast
```

`replayRoot` (`replay_server_test.go:18-27`) runs `os.CopyFS` on `recording.Fixture("blog")`, and that directory no longer exists on main. The other `webcast/...` subpackages pass. BR-15 is still not fixed: `replay.js` is not in this window.

1. **Strengths**
   - `replay_live_test.go:130-140`: the resolution takes main's approach. Check times come from the `rec-app` recording's own Note/NoteState/Stroke/Seg records, so a re-recorded fixture keeps working. The branch's checks for painted strokes, card placement and resize while paused are all kept.
   - `replay_live_test.go:203-213`: the old hard-coded `#t=19100` (tap) and `#t=12400` (resize) are now derived (`withNote`, `withNote+1`). The test fails loudly (`no moment shows a note`) instead of silently checking nothing if a fixture loses its notes. This follows the PURPOSE lens.
   - The comment at `:167-168` now names `noteBox` instead of the removed `drawMarks`, which matches BR-13's correction.

2. **Critical findings**
   - `cmd/internal/webcast/shares_test.go:62,171,244` and `replay_server_test.go:235` still use the `blog` fixture that main's #474 deleted, so four tests fail at head. The rule: when a merge brings in a retired name, grep every use of that name across the tree in the same resolution, not just the files that conflicted textually. The sweep is `grep -rnE '(Fixture|replayRoot|sharesOn)\([^)]*"(app|blog|stress)"' cmd`, which today finds exactly these four. Fix: switch them to `rec-blog`, including the URL paths at `shares_test.go:77,81` and `replay_server_test.go:242-243`. Then confirm `go test ./cmd/internal/webcast/...` passes.

3. **Important findings:** none.

4. **Minor findings**
   - `workshop/lessons.md`: the moved #477 bullets now sit directly before the #480 bullet with no blank line between them, unlike the neighbouring entries.
   - BR-15 is still open (see the disposition below).

5. **Test coverage notes**
   - I ran `go vet` and `go test ./cmd/internal/webcast/...`. The only failures are the four above, all caused by the missing fixture.
   - I did not run the live Chrome test `TestLiveReplayServerMarks` (it needs `WEBCAST_LIVE_CHROME=1`). I read the resolved file and it compiles under vet. Using `withNote+1` for the resize check is fine: the canvas-versus-box check doesn't depend on which moment is showing.

6. **Architectural notes**
   - **ARCH-DRY: pass.** The resolution adds no duplicated logic.
   - **ARCH-PURE: pass.** The window changes only tests and docs.
   - **ARCH-PURPOSE: flag.** The conflict resolution fixed the one file that conflicted, not every reference to the retired fixtures (the Critical finding above).

7. **Plan revision recommendations:** none.

```findings
dispose:
  - id: BR-15
    disposition: not-addressed
    note: |
      replay.js is not in this window; replay.js:482 still passes win.devicePixelRatio once, at mount, while annotate.js reads it on every paint. Minor, non-blocking.
findings:
  - id: new
    severity: Critical
    family: merge-sweep-retired-references
    title: |
      Merge with main leaves four #477 tests on the retired "blog" fixture; go test ./cmd/internal/webcast fails at head
    detail: |
      Main's #474 (bb95a35af) deleted the app/blog/stress fixtures. The resolution moved only replay_live_test.go to rec-app; shares_test.go:62,171,244 and replay_server_test.go:235 still call replayRoot/sharesOn with "blog", and os.CopyFS fails with "stat .: no such file or directory" (TestReplaySharesShareOneRecordingPerToken, TestReplaySharesCloseWhileOpening, TestReplaySharesSharedAndOnReady, TestRecordingServerServesOnlyItsRecording). Rule: when a merge brings in a retired name, sweep the whole tree for it (grep -rnE '(Fixture|replayRoot|sharesOn)\([^)]*"(app|blog|stress)"' cmd), not just the files that conflicted. Fix: switch to rec-blog, including the /r/blog/ URL paths at shares_test.go:77,81 and replay_server_test.go:242-243.
  - id: new
    severity: Minor
    family: doc-formatting
    title: |
      lessons.md: the moved #477 bullets have no blank line before the #480 bullet
```

---

## Re-review — 2026-10-12T07:48:02-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 477 — broadcast-web: replay inside a running Couch, for remote viewers (mount ReplayServer on the shared Host) |
| repo | pair |
| issue file | workshop/issues/000477-broadcast-web-replay-inside-a-running-couch-for-remote-viewers-mount-replayserver-on-the-shared-host.md |
| boundary | whole-issue close |
| milestone | — |
| window | ab270639baac60b954b18c4684c00983a2f426a4..763ec0464fdac28feb9c6db35c2f8ed7797a6416 |
| command | sdlc close --issue 477 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T07:48:02-07:00 |
| verdict | SHIP |

## Review

Webcast tests pass. The live Chrome tests fail before any test logic runs, and the cause is this environment. Writing up the verdict now.

```verdict
verdict: SHIP
confidence: medium
```

This window repairs the merge with main and fixes nothing else. The conflict in `replay_live_test.go` is resolved and keeps both sides' intent. From main (#474): the `rec-app` fixture, mark times read from the recording, and a pin needs its region, element or box. From #477: the comment saying the replay uses the live viewer's modules. The tap and resize checks used fixed times (`19100`, `12400`) tied to the retired `app` fixture. They now use `withNote`, the first moment that shows a note, and fail loudly if no such moment exists.

BR-16 is fixed for the whole class, not just the cited site. Grepping all of `cmd/` for `(Fixture|replayRoot|sharesOn)("app"|"blog"|"stress")` finds nothing left. `go test ./cmd/internal/webcast` and `go vet` pass at the head, and no conflict markers remain under `cmd/`. Nothing in the window changes Spec, Revisions or Done when.

I could not run the live Chrome tests here. With `WEBCAST_LIVE_CHROME=1`, every live test fails inside `headlessPage` with `Emulation.setDeviceMetricsOverride: cdp: read: EOF`. That includes `TestLiveReplayServerDrivers`, which this window doesn't touch, so the cause is this machine's Chrome, not the diff. The rewritten `TestLiveReplayServerMarks` needs a live-Chrome run before merge. That is why confidence is medium.

1. **Strengths**
   - `replay_live_test.go:203-214`: the tap check now takes its moment from the recording (`withNote`) and stops with an error if none exists. This applies the #474 lesson that a re-recorded fixture keeps the test working.
   - `replay_server_test.go:242-243` and `shares_test.go:81`: the path-escape checks were updated along with the fixture name (`/tok/r/rec-blog/`, `segments%2F..%2Frec-blog`). The 404 checks still test real paths, not names that no longer exist.
   - The conflict resolution merges the two comments into one and drops neither side's meaning.
2. **Critical:** none.
3. **Important:** none.
4. **Minor**
   - BR-17 is still open: `workshop/lessons.md` has no blank line between the "#477 M2" bullet ("…the fix removed).") and the "#480" bullet.
   - BR-15 is still open. Nothing in this window touches `replay.js`.
   - `shares_test.go:79` still prints the failure label `"blog: %+v"`. It's cosmetic.
5. **Test coverage:** The four tests BR-16 named now pass. Before the fix they failed on a missing fixture, so they are their own regression evidence. The live-Chrome marks test could not be run here (see above).
6. **Architecture:** All three apply only to this window. ARCH-DRY passes: there's no new logic. ARCH-PURE passes: the changes are tests only. ARCH-PURPOSE passes: the sweep covered every retired fixture name, not only the sites BR-16 cited.
7. **Plan revisions:** none.

```findings
dispose:
  - id: BR-15
    disposition: not-addressed
    note: |
      replay.js is not in this window; devicePixelRatio is still read once at mount.
  - id: BR-16
    disposition: addressed
    note: |
      Grep of cmd/ for retired fixture names in Fixture/replayRoot/sharesOn finds nothing; /r/blog/ paths moved to rec-blog; go test ./cmd/internal/webcast passes at head.
  - id: BR-17
    disposition: not-addressed
    note: |
      lessons.md still has no blank line between the #477 M2 bullet and the #480 bullet.
```
