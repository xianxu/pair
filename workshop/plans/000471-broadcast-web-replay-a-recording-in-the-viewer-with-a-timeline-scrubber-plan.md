# broadcast-web replay with a timeline scrubber: Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Open a recording directory in the #460 viewer and play it with a timeline scrubber (seek, play, pause, speed), with pauses shown as gaps, notes and strokes at their moments, who drove, and `#t=` links to a moment.

**Architecture:** The viewer keeps its model (`model.js`) and renderer (`render.js`) unchanged. A new **replay source** (`web/replay.js`) stands in for the live event stream behind the same surface `connect()` returns (`{model, renderer, onRecord}`). It seeks with the format's catch-up rule: the state at t is the catch-up of the records up to t, reduced, so a seek replays at most one segment. A new Go `webcast.ReplayServer` (a `broadcast.Mountable`) serves a recordings root: a listing, each recording's merged records, its assets through #459's proxy, and the same viewer modules. For now `couch --broadcast-web-replay [ID|DIR] [--at T]` serves it on loopback and prints the link. Mounting it on the shared Host inside a running Couch is a TL-filed follow-up after #458.

**Tech Stack:** Go (net/http, embed), ES modules (no build), node `--test`, headless Chrome over CDP for the live checks.

**Decisions (TL ops:0, 2026-10-12):** recordings live at `<couch store>/webcast/recordings/<id>/` (#458 sets `cfg.RecordingsDir`); loopback CLI entry point now, Host mount later. #470 (pair:6) edits `viewer.js`, `index.html` and `viewer.css` in parallel, so this plan's edits to those three files stay small and additive: replay mode behind **one** branch point in `mount`, the timeline as its own element, its styles in a new `replay.css`. Whoever lands second merges main and resolves.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `catchUp` | `cmd/internal/webcast/web/replay.js` | new |
| `timeline` | `cmd/internal/webcast/web/replay.js` | new |
| `placer` | `cmd/internal/webcast/web/replay.js` | new |
| `parseMoment` / `momentHash` | `cmd/internal/webcast/web/replay.js` | new |
| `Player` | `cmd/internal/webcast/web/replay.js` | new |
| `assets.Reopen` | `cmd/internal/webcast/assets/reopen.go` | new |
| `ReplayLink` | `cmd/internal/webcast/replay.go` | new |

- **catchUp(recs)**: a port of `recording.CatchUp` (head and assets before the latest seg, the bare seg the document began in, the notes, note states and strokes since then, the latest ctl before the latest seg, everything from the latest seg on).
  - **Relationships:** the Go function is the rule's owner. This port is held to it by the format's own stated invariant, and the live check also compares it with Go's output (below).
  - **DRY rationale:** the seek rule *is* the catch-up rule ("seek by reducing from the latest seg before t", the reducer's rule; webcast-format.md "Joining mid-cast"). The replay needs it on the client because scrubbing must not round-trip per seek. It is written once in JS and tested against Go's. `model.js` is the precedent for this kind of held port. ARCH-DRY: a server `catchup?t=` route would keep one source but make each scrub a network fetch over the tunnel (the follow-up's Host mount). It was rejected for smoothness. The parity test makes the second copy safe.
- **timeline(recs)** gives `{start, end, segs:[{seg, why, t0, t1}], gaps:[{t0, t1}], notes:[{t, id, author, text}], strokes:[{t, id, color, dur}], drives:[{t0, t1, driver}], acts:[{t, by, k}]}`.
  - A segment runs from its `seg` record to the next `stop` (or to the end).
  - A gap runs from a `stop pause` to the next `seg`.
  - `drives` are the intervals where a `ctl` record's state names a driver.
  - `acts` are the records carrying `by`.
  - It is pure over the merged records.
- **placer(tl, width)** maps time to x and x back to time, with every gap drawn at a fixed share (`GAP_SHARE`, 3% of the width, at most the gap's real share). An hour's pause therefore doesn't flatten the segments. The two maps are monotonic and inverse on the segments. A point inside a gap maps to the next segment's start.
- **parseMoment(hash)** reads `#t=1530` (ms; integer ≥ 0, else null). **momentHash(t)** writes it. Agents link with this exact form.
- **Player**: the seek and play state machine over one recording.
  - **State:**
    - `recs`: merged records.
    - `next`: the index of the first record not yet applied.
    - `t`: the playhead.
    - `playing`: a bool.
    - `speed`: one of 0.5, 1, 2 or 4.
    - `model` and `renderer` come from the factories.
  - **Transitions:**
    - `seek(t)`. Forward within the current document stretch: if no `seg` record lies in `recs[next..]` up to t, apply forward. Otherwise rebuild: a fresh model and renderer, then `catchUp(recs up to t)` applied. A seek backward always rebuilds.
    - `tick(dtMs)`. While playing, it advances `t` by `dt·speed`. A playhead that lands in a gap jumps to the next segment's start, so playback skips pauses. At the end it stops playing.
    - `play()` and `pause()`. `play()` at the end restarts from 0.
    - `setSpeed(s)`.
  - It holds no timers. The shell drives `tick` from requestAnimationFrame, so the state machine is testable without a clock.
  - Refusal: a record the model refuses during replay is a broken recording. The player stops, keeps the last good state, and reports the error to the bar (`status: 'Recording error'`). It never resyncs, because there is no stream to resync from.
- **assets.Reopen(rec)** is a read-only `assets.Served` over a finished recording.
  - It is the existing `Recorded`, populated from the recording's `asset` records: hash to MIME, URL to binding. It reads bodies through `rec.Asset`.
  - `Put` refuses with `ErrReadOnly`.
  - It lets the existing `Proxy` serve a replay with the same CSS rewrite and the same MIME allow-list (ARCH-DRY: one serving path).
- **ReplayLink(base, id, t)** returns `base/r/<id>/#t=<t>`, the agent-facing URL of a moment. The CLI prints it.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `ReplayServer` | `cmd/internal/webcast/replay.go` | new | the recordings root on disk, HTTP |
| replay mode in `mount` | `cmd/internal/webcast/web/viewer.js` | modified | `fetch`, `location.hash`, rAF |
| `replaySource` | `cmd/internal/webcast/web/replay.js` | new | the records URL, DOM of the timeline bar |
| `couch --broadcast-web-replay` | `cmd/internal/couchcmd/replay.go` | new | loopback listener, signals |

- **ReplayServer(token, root)** is a `broadcast.Mountable`. It is GET only, under the token, and uses the viewer's CSP. Its routes:
  - `<token>/` lists the recordings in root as server-rendered HTML (`html/template`): id, started, duration, segments, notes. Each row links to `r/<id>/`.
  - `<token>/r/<id>/` serves the viewer's `index.html`.
  - `<token>/r/<id>/<module>` serves the viewer's modules and `replay.css`, from the same embedded FS as `Server`.
  - `<token>/r/<id>/records` serves the merged records as JSON lines (`recording.ReadDir`, then `Encode`).
  - `<token>/r/<id>/assets/<hash>` goes through an `assets.Proxy` bound to `assets.Reopen(rec)`.
  - `<id>` must match `^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$` and name a directory directly under root. Anything else is a 404, so no traversal.
  - A recording is read per request for `records` (a recording is about 1.4 MB/min; ARCH-CONSTRAINTS: bounded by the file size, no cache). The asset proxy for an id is built once and kept in a map bounded by the number of directories (a few). A directory that disappears is answered 404 on its next read.
  - The replay page is told it is a replay by the path: `mount` sees `/r/` in `location.pathname`. Nothing is templated into `index.html`.
  - **Injected into:** nothing pure. The pure parts it uses (`ReadDir`, `Merge`, `Reopen`, `Proxy`) are already tested with fakes.
  - **Future extensions:** the Host mount inside Couch (TL follow-up). A retention sweep comes with the first listing feature (webcast-format.md, Lifecycle).
- **replaySource(deps)** returns the same surface as `connect` (`{model, renderer, onRecord}`) plus `player`. deps:
  - `fetchRecords()`.
  - `renderer(model)`.
  - `show(view)`.
  - `frame()`.
  - `schedule(fn)`.
  - `now()`.
  - `bar`, the timeline element.
  - `hash()` and `setHash(h)` for `#t=`.
  - Like `connect`'s, the deps are injected so node tests drive it with fakes.
- **Notes and strokes in replay** are drawn by `replay.js` into its own layer inside `#overlay`:
  - A note shows as a pin at its node's `renderer.boxOf(at.node)`, else at `at.box`. It is shown while open, from its `t`, on its document, which is `model.notes` as the player has reduced them.
  - A stroke is a polyline, shown from its `t` through its last point plus a 1.5 s fade.
  - #470 builds the live notes and pointer UI in its own modules. When both have landed, replay should switch to #470's drawing. That is a follow-up logged at close, so this issue does not duplicate #470's UI beyond a pin and a line.
- **couch --broadcast-web-replay [ID|DIR] [--at T]**:
  - With no argument it serves the root `<store>/webcast/recordings`. An ID resolves under the root. A DIR (a path with a separator, or one that exists) is served as root=parent, id=base.
  - It listens on `127.0.0.1:0` with a fresh token (`broadcast.NewToken`). It prints the listing URL, or `ReplayLink` for an ID with `--at`. It serves until SIGINT or SIGTERM.
  - Creates nothing durable (ARCH-FUNERAL): it only reads recordings, and its listener and token die with the process.

### Lifecycle (ARCH-FUNERAL), state (ARCH-STATE)

- Nothing new is persisted. The player's model and renderer live as long as the page. A rebuild replaces both, and `revokeAll` frees the old drawing's blob: URLs, as `connect`'s `fresh` does.
- Player states: `(playing ∈ {true,false}) × (t ∈ [start,end])`, plus `error` (terminal, stops playing). `tick` is the only clock input. `seek` from any state is legal, and it leaves `error` only by a reload.

## Files

- Create: `cmd/internal/webcast/web/replay.js`, `cmd/internal/webcast/web/replay.css`, `tests/webcast/replay.test.mjs`
- Create: `cmd/internal/webcast/replay.go`, `cmd/internal/webcast/replay_server_test.go` (`replay_test.go` is #460's fixture replay), `cmd/internal/webcast/replay_live_test.go`
- Create: `cmd/internal/webcast/assets/reopen.go`, `reopen_test.go`
- Create: `cmd/internal/couchcmd/replay.go`, `replay_test.go`
- Modify (additive):
  - `web/viewer.js`: one branch point in `mount`.
  - `web/index.html`: `<footer id="timeline" hidden>` and a `replay.css` link.
  - `server.go`: add `replay.js`/`replay.css` to the embed and the module table.
  - `couchcmd/cli.go` + `run.go`: the flag and its usage line.
  - `cmd/internal/artifactpath/manifest.go`: classify the new production files (lesson #460).
  - `atlas/webcast.md`: the replay section.

## Tasks

- [ ] **1. catchUp + timeline + placer + moment (pure JS, TDD).** `tests/webcast/replay.test.mjs`:
  - For every fixture and every cut index i, reducing `catchUp(recs[0..i])` gives the same `canonical()`, `docSeg`, current-document notes (id, state) and `assets` as reducing `recs[0..i]`. This is the format's own stated catch-up invariant, independent of the implementation. Run it against a deliberately broken catchUp (assets dropped) first and see it fail.
  - Timeline on `segments`: the segs and gaps match the fixture's seg and stop times, written by hand from `fixturegen -segments` output.
  - Timeline on `driven`: the drive intervals and the `by` acts, against a hand-written expectation.
  - Placer: monotonic, inverse on segments, gap share capped, a point in a gap goes to the next seg.
  - `parseMoment`: `#t=1530` gives 1530; `#t=`, `#t=-1`, `#t=1.5` and `#x=1` give null.
- [ ] **2. Player (pure JS, TDD).** With a fake renderer and the real Model:
  - `seek(p.t)` for every fixture pin gives the pinned sha256, visited forward, backward and in a seeded shuffle, so seeking matches reducing from scratch.
  - A forward seek within a segment does not rebuild (the renderer factory is called once). A seek across a `seg` or backward does rebuild.
  - `tick` advances by speed, skips a gap (the `segments` fixture's pause), and stops at the end.
  - A refused record puts the player in `error` and keeps the last state.
- [ ] **3. replaySource + shell branch (JS).** `replaySource` builds the bar (play/pause button, speed select, track with segs, gaps, note ticks, stroke ticks, drive bands labelled with the driver's name, a playhead, a time label). Pointer drag and click on the track seek through the placer; the keyboard does too (Space, ←/→ ±1 s).
  - `#t=` on load seeks there paused. A seek updates the hash (`history.replaceState`, so no history spam), and `hashchange` seeks.
  - The bar shows status `Replay`, the title and the URL from the model.
  - In `mount`, one branch: `const v = replayPath(win.location.pathname) ? replaySource({...}) : connect({...})`.
  - Node tests with fakes: the hash seeks on load; a track click seeks to `placer.timeAt(x)`; the hash follows the playhead.
- [ ] **4. assets.Reopen (Go, TDD).** Reopen a fixture: `TypeOf` and `Lookup` agree with its asset records, `Get` reads bodies, `Put` gives `ErrReadOnly`, and `Proxy` bound to it serves a CSS body rewritten as the live proxy does (reuse `proxy_test` helpers).
- [ ] **5. ReplayServer (Go, TDD).** `replay_server_test.go` on a temp root holding copies of two fixtures:
  - The listing names both.
  - `r/<id>/records` equals the encoded `ReadDir` records.
  - `r/<id>/assets/<hash>` serves.
  - Modules are served.
  - A traversal id (`..`, `a/b`, `%2e%2e`), a file instead of a dir, and an unknown id all 404.
  - Non-GET gives 405.
  - The CSP header is set.
  - Another token gives 404.
- [ ] **6. CLI (Go, TDD).** `couchcmd/replay.go`:
  - Argument resolution covers no argument, ID, DIR, and a missing one (an error naming the root).
  - `--at` without an id is refused.
  - The printed link has the form `http://127.0.0.1:<port>/<token>/r/<id>/#t=<T>`.
  - The server starts and stops on context cancel (a test with a cancelled context).
  - Add the usage line to the `run.go` help and classify the file in the manifest.
- [ ] **7. Live check in headless Chrome (reuse #460's harness).** `replay_live_test.go` (`WEBCAST_LIVE_CHROME=1`):
  - Serve a temp root of every fixture through `ReplayServer`.
  - For each fixture pin, open `r/<id>/#t=<pin>` and await `couchViewer.ready()`. `couchViewer.canonical()` must equal `recording.Reduce(...,pin).Canonical()`, which covers the `#t=` link.
  - Then, in one page, `player.seek` across all pins in shuffled order with the same check, which covers scrubbing.
  - JS `catchUp` output must equal Go `CatchUp` encoded, at every pin.
  - The `driven` fixture's timeline shows drive bands.
  - On `app`, notes are drawn as pins at the time of the note and are absent before it. Strokes are drawn at their `t`.
  - The page requests nothing outside the server.
- [ ] **8. Atlas + lessons + close.**
  - `atlas/webcast.md`: a "Replay (#471)" section covering routes, the seek rule, the CLI, and the follow-ups (the Host mount; adopting #470's notes and pointer drawing).
  - `atlas/webcast-format.md` Lifecycle: the "seek from latest seg" note is now done, in the viewer, by catch-up.
  - Full `make -k test` + `go test ./...`. The live run is unsandboxed.
  - `sdlc close`, then report to the TL. The operator's smoke test is the TL's to schedule.

## Verification

- `go test ./cmd/internal/webcast/... ./cmd/internal/couchcmd/ ./cmd/internal/artifactpath/` (TestNode runs `tests/webcast/*.test.mjs`).
- `WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run 'Live' -v`, unsandboxed.
- Manual: `couch --broadcast-web-replay <fixture dir> --at <pin>` opens at the moment, scrubbing works, and pauses show as gaps.
