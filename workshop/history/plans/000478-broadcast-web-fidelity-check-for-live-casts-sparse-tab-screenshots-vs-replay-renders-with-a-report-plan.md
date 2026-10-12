# Broadcast-web fidelity check Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Every recorded cast gets a fidelity report: the viewer's renderer (#460) replays the recording at each
`frame` (the real tab's sparse low-res screenshot), #453's diff scores it, and a report per recording gives
median/max and the worst moments side by side. It runs from the CLI on any recording and automatically when a
recorded cast ends.

**Architecture:** A pure package `webcast/fidelity` owns the scoring (#453's YIQ diff, moved out of the
development tool it lives in today), the tolerance, the report and the side-by-side crops. The shell lives in
package `webcast` (`fidelity_check.go`), because it needs what only that package has: the embedded viewer modules,
`ReplayServer` (records and asset proxy for one recording) and `FindChrome`. It serves the recording on loopback,
opens a new embedded page `fidelity.html` in a headless Chrome, and for each frame calls the page's
`replayAt(t, 1000)` (the harness's real-time window, now product code in `web/replayer.js`), screenshots the
viewport, scales it with `recorder.LowRes` (the same filter the tab's frames went through) and diffs. `CastConfig`
gains a `Check` hook that `Cast` runs when a recorded cast ends; couchcmd wires it to the check. The CLI is
`couch --broadcast-web-fidelity [ID|DIR]`.

**Tech Stack:** Go, CDP over `--remote-debugging-pipe`, the viewer's ES modules (`model.js`, `render.js`), node
tests for the JS.

**What already exists (no recorder change):** the recorder (#458) already takes a frame at each segment start
(go-live, resume, navigate, tab) and every 5 s while live (`Options.ShotEvery`), writes `frames/<n>.png` +
`frames/index.jsonl` through `DirWriter.PutFrame`, never into the live stream. So the Spec's first bullet holds on
main; this issue measures it (a live blog cast) rather than changing it. Any recorder change found necessary goes
through the TL (ops:0).

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `Diff`, `DiffImages`, `ControlCorner`, `Threshold` | `cmd/internal/webcast/fidelity/diff.go` | modified (moved from `recorder/internal/fidelity/diff.go`) |
| `Tolerance` | `cmd/internal/webcast/fidelity/report.go` | new |
| `Shot`, `Report`, `Summarize` | `cmd/internal/webcast/fidelity/report.go` | modified (moved + extended: `Over`, `Worst`) |
| `DiffBox`, `SideBySide` | `cmd/internal/webcast/fidelity/crop.go` | new |
| `WriteReport` (pure render of `report.md` / `report.json` text) + `Report.Markdown` | `cmd/internal/webcast/fidelity/report.go` | new |
| `Replayer` (JS: model + renderer over a record list, `replayTo`, `replayAt`) | `cmd/internal/webcast/web/replayer.js` | new (promoted from `testdata/replay/harness.js`) |
| static renderer `Render`, `ScrollJS`, `Offsets`, `Pick`, `ReplayDPR`, `Run` | `recorder/internal/fidelity/` | deleted (superseded: the check uses the viewer's renderer) |

- **Diff** — #453's pixelmatch-style YIQ distance, share of differing pixels outside the control's corner. Moved
  as-is. **DRY:** one diff for the product check; `replay_test.go`'s channel-threshold `pixelDiff` is a different
  measure for #460's conformance test and stays test-only (noted, not merged: its tolerance was measured against it).
- **Tolerance** — 3% of pixels per frame (the viewer conformance's measured bound, 0.2–2.3% on the fixtures),
  re-measured in Task 6 on a live blog cast with #453's diff; the constant's comment carries the measured numbers.
- **Report** — per recording: `Shots` (file, t, diff %, `Over`), `Median`, `Max`, `Skipped` (frames with nothing
  live to replay), `Worst` (up to 3 highest, plus every shot over tolerance, capped at 10, each with a crop file).
  **Future:** per-URL breakdown, trend across recordings.
- **DiffBox / SideBySide** — the bounding box of differing pixels, padded 8 px and clipped; the side-by-side image
  is tab crop | replay crop | diff mask (red where pixels differ) at the frame's low resolution, so a worst moment
  is readable without the replay.
- **Replayer** — the JS replay the harness has today (`load`, `replayTo`, `join`, `replayAt`), as a class over a
  given frame, records and asset resolver. The test harness and `fidelity.js` both use it (**DRY:** one
  `replayAt`, whose real-time window is what makes transitions comparable).

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `CheckFidelity(ctx, chromeBin, dir) (fidelity.Report, error)` | `cmd/internal/webcast/fidelity_check.go` | new | headless Chrome over CDP, loopback HTTP |
| `fidelity.html` + `fidelity.js` | `cmd/internal/webcast/web/` | new | the page the check drives |
| `ReplayServer` serves `fidelity.html`/`fidelity.js`/`replayer.js` | `cmd/internal/webcast/server.go` (`assets`) | modified | — |
| `CastConfig.Check` | `cmd/internal/webcast/cast.go` | new | runs after a recorded cast ends |
| `couch --broadcast-web-fidelity [ID|DIR]` | `cmd/internal/couchcmd/fidelity.go` | new | CLI |
| couchcmd wiring of `Check` | `cmd/internal/couchcmd/webcast.go` | modified | — |

- **CheckFidelity** — serves `filepath.Dir(dir)` with `NewReplayServer` on 127.0.0.1 under a fresh token,
  launches headless Chrome (fresh temp profile, removed after), opens `r/<id>/fidelity.html`, then per frame in
  `t` order: `Emulation.setDeviceMetricsOverride` to the state's viewport at DPR 1, `fidelity.replayAt(t, 1000)`
  (returns `{live, w, h}`; not live → skipped), `Page.captureScreenshot`, `LowRes(full, 1/(2·scale))`, `Diff`
  against the frame's png; a size mismatch is a 100% shot, not an error. Writes the report (below). Forward-only:
  frames are replayed in order on one Replayer, so a recording is replayed once, not once per frame.
  **Injected into:** nothing pure — the pure pieces take its outputs. **Fake:** none needed beyond the fixtures:
  its seam is a real recording directory; it is tested live (`WEBCAST_LIVE_CHROME=1`) on the fixtures and a live
  cast, and its pure halves are unit-tested.
- **CastConfig.Check** — `func(ctx context.Context, dir string) (string, error)`. `Cast.run` calls it in a
  goroutine after the cast has ended if the recording directory holds `frames/index.jsonl`; a non-empty summary or
  the error goes to `Logf` ("Chrome cast: fidelity …"). Cast never imports the check (couchcmd wires it), so a
  `FakeBrowser` cast and tests run without Chrome. Tested with a fake Check in `cast_test.go`.

### Lifecycle (ARCH-FUNERAL / ARCH-CONSTRAINTS)

- **`<recording>/fidelity/`** (`report.md`, `report.json`, `worst-<n>.png`): created by the check (CLI or end of
  cast), needed until the operator is done with the recording, removed with the recording directory (the
  recording's own lifecycle: kept until the operator deletes it). Size: at most 10 crops of at most the frame size
  (~640×400 PNG, ≤ ~300 KB each) plus two small text files; a re-run replaces them (written to temp names and
  renamed). Not part of the record stream and not under `frames/`, so `ValidateDir` and the readers are
  unaffected; documented in webcast-format.md's directory layout as additive.
- **The headless Chrome + temp profile**: per run, killed and removed when the check returns; Chrome exits by
  itself when Couch's pipe end closes, so a Couch that exits mid-check leaves none.
- **Runtime cost:** about 1.5 s per frame (1 s real-time window + load + capture); a 30-minute cast at 5 s frames is
  ~360 frames, ~9 min in the background after the cast. Only one end-of-cast check runs at a time per Couch
  (a second waits), so back-to-back casts don't stack Chromes.

---

## Tasks

### Task 1: pure `webcast/fidelity` package (move + extend)

**Files:** create `cmd/internal/webcast/fidelity/{diff.go,report.go,crop.go,fidelity_test.go}`; delete
`cmd/internal/webcast/recorder/internal/fidelity/`; update `cmd/internal/artifactpath/manifest.go`,
`webcast/live_record_test.go` and `webcast/testdata/stress/README.md` (they point at the old tool).

- [ ] Move `diff.go` and its `TestDiff`; add failing tests: `Summarize` marks `Over` at `> Tolerance`, picks
  `Worst` (top 3 plus all over, cap 10, by diff descending), median/max as before, skipped counted;
  `DiffBox` on two images differing in a known rectangle returns it padded and clipped, and `ok=false` for equal
  images; `SideBySide` output is `3w+2·gap` wide and shows red in the mask panel at a differing pixel;
  `Report.Markdown` lists median, max, tolerance, each over-tolerance moment with its `t` and crop file.
- [ ] Implement; `go test ./cmd/internal/webcast/fidelity` green.
- [ ] Commit `#478: fidelity: pure scoring, report and crops (moved from the recorder's dev tool)`.

### Task 2: `Replayer` in the viewer (JS), harness on it

**Files:** create `web/replayer.js`; modify `testdata/replay/harness.js` to use it; node test
`tests/webcast/replayer.test.mjs` (fake frame not needed: test `replayAt`'s record scheduling with an injected
clock/feed, as `replay.test.mjs` does with fakes).

- [ ] Failing node test: `Replayer.replayAt(t, w)` feeds records `≤ t-w` at once, then the rest up to `t` in
  order at their times relative to an injected `now`/`sleep`; forward only (a second call continues from where
  the first stopped); returns `{live, w, h}` from the model.
- [ ] Implement by moving harness.js's logic; harness keeps its window API (`load`, `replayTo`, `join`,
  `replayAt`) as thin calls. `WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run LiveReplayFixtures` still
  passes (the harness refactor changes no measurement).
- [ ] Commit.

### Task 3: the fidelity page and `CheckFidelity`

**Files:** create `web/fidelity.html`, `web/fidelity.js`, `fidelity_check.go`, `fidelity_check_test.go`; modify
`server.go` (`//go:embed` list and `assets` map).

- [ ] `fidelity.html`: no chrome, `body{margin:0;overflow:hidden}`, one `iframe#page sandbox="allow-same-origin"`
  at (0,0). `fidelity.js`: fetches `records`, builds a `Replayer` resolving assets to `assets/<hash>`, sizes the
  iframe to the model's viewport, exposes `window.fidelity = {ready, replayAt}`.
- [ ] `replay_server_test.go`: `r/<id>/fidelity.html` and `replayer.js` are served with the viewer's CSP.
- [ ] `CheckFidelity` as specified above, writing the report through Task 1's `WriteReport`.
- [ ] Live test `TestLiveFidelityFixtures` (`WEBCAST_LIVE_CHROME=1`): every real fixture (stress, app, blog) is
  within `Tolerance` at median, and its report files exist in a copy of the fixture; log per-frame numbers.
- [ ] Live test `TestLiveFidelityCatchesBrokenRender`: a copy of `blog` with every `sheet` record and every
  `text/css` asset record removed (a render without its styles) reports `Max > Tolerance` and lists those
  moments as over, with crops.
- [ ] Commit.

### Task 4: run at the end of a cast

**Files:** modify `cast.go` (`CastConfig.Check`, `Cast` remembers its recording dir), `cast_test.go`,
`couchcmd/webcast.go`.

- [ ] Failing test: a cast over a `FakeBrowser` with `RecordingsDir` set and a fake `Check` that records its call:
  when the recording dir has `frames/index.jsonl` the hook runs once after `Done`, its summary reaches `Logf`;
  with no recording (never live) it doesn't run; an error reaches `Logf`.
- [ ] Implement; couchcmd wires `Check` to `CheckFidelity` with `FindChrome` and a process-wide one-at-a-time
  semaphore; summary is `fidelity median X% max Y% (N over 3%): <dir>/fidelity/report.md`.
- [ ] Commit.

### Task 5: CLI

**Files:** create `couchcmd/fidelity.go`, `couchcmd/fidelity_test.go`; modify `cli.go`, `run.go` usage.

- [ ] `couch --broadcast-web-fidelity [ID|DIR]` reuses `resolveReplay`; no target = the newest recording in the
  store (`ListRecordings`). Prints the summary and the report path. Exit 0 within tolerance, 3 when some frame is
  over, 1 on error, 2 on usage. Tests cover parsing, resolution and exit codes with an injected check function.
- [ ] Commit.

### Task 6: live blog cast + atlas

- [ ] Record the blog (`localhost:4321` dev server, `browse` scenario) with `TestLiveRecordURL`, run the CLI on it:
  report within tolerance; record the numbers in the issue Log and in `Tolerance`'s comment.
- [ ] `atlas/webcast.md`: replace the dev-tool bullet with the fidelity check (shell, page, hook, CLI);
  `atlas/webcast-format.md`: `fidelity/` in the directory layout and the report format (report.json fields,
  report.md, crops), a `## Revisions` line (additive).
- [ ] Full `make -k test` + `go test ./...`, then `sdlc close`.

## Revisions

- 2026-10-12 (close review round 1, BR-1): the retries are true replays at t±δ, in the order 0, −300, −150,
  +150, +300 (stopping at the first within tolerance). −300 seeks back through the catch-up (one segment at most,
  not the whole recording); the rest continue, and continuing resumes the animations paused at the last capture
  instead of finishing them. So "a recording is replayed once" holds for frames within tolerance. A frame over it
  costs one catch-up seek plus about 1.6 s of real-time windows.
- As built: `WriteReport` is `fidelity.Write` (plus `Report.Summary`, `Within`, `Markdown`). The hook's test is in
  `record_cast_test.go`. A segment that isn't a navigation finishes its document's animations (found on the live
  blog cast). The report keeps `offset` and `at_t`. Frame records are validated (`frameViewport`) before use.
- 2026-10-12 (after merging #472): the retries above are superseded. Frames are compared at `fidelity.Offsets` =
  t, t − 50 ms (#472's measured capture skew), through `recording.FrameTimes`. A later moment would hide a late
  recorder (pair#475 on rec-stress frame 0). The live test covers every fixture with frames and honours #472's
  `knownGaps`.

