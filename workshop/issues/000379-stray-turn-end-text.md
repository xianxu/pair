---
id: 000379
status: codecomplete
deps: []
github_issue:
created: 2026-10-01
updated: 2026-10-07
estimate_hours:
card_mirror: 'a21837eba0afa77921e3757a0c740c1f66983441' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-03T11:34:37-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 5.82
---

# Turn-end text flashes at focused pane cursor

Low priority. The operator is curious about the cause; Couch's screen handling is otherwise solid.

## Problem

Under Couch, Claude's turn-end line ("✻ Sautéed for 10m …") sometimes flashes
**outside the agent pane**, at the screen cursor of the focused pane (the
draft nvim or the right terminal). It stays for about a second, then a repaint
covers it. It is too brief to screenshot, and it's the only screen artifact the
operator has seen. Plain `pair` hasn't been checked; the operator now uses Couch.

Evidence so far (2026-10-01):

- Claude draws that line with **only relative** cursor motion, right after
  closing one synchronized frame, sending progress-bar escape codes and
  opening the next frame (recorded in `scrollback-couch-…-claude.raw`):
  `ESC[?2026l ESC]9;4;0;BEL ESC]9;4;3;BEL ESC[?2026h ESC[?25l ESC[2D ESC[3B \r ESC[9A ✻ Churned for 1m 45s`.
  Inside the zellij pane that is fine, because zellij tracks each pane's cursor.
- Text landing at the *focused pane's* cursor means that, between zellij's client
  output and Couch, a "move cursor to (row, col)" instruction was lost (or never
  emitted). Most likely it was the first one in a redraw, since zellij leaves
  the cursor at the focused pane between redraws.
- It is **intermittent**, but Claude sends the same bytes at every turn end, so
  the cause must depend on timing (read boundaries, or interleaved redraws),
  not on the byte content.
- `Endpoint.Feed` (`cmd/internal/terminal/endpoint.go`) hands each chunk
  directly to the `vt` emulator, and that parser keeps its state between
  chunks. A sequence split across two reads in Couch should therefore parse
  correctly; a bug inside the `third_party/vt` fork isn't ruled out.

## Spec

Own the long-running investigation of Claude turn-end text appearing in another
pane: preserve observations, audit isolation/redraw behavior, capture an actual
incident, and reduce it to a repeatable reproduction that identifies the faulty
layer. Timing, cursor state and terminal escape handling remain hypotheses,
not established causes. The early theories in Problem are historical; the Log
records subsequent tests and corrections.

Tracing implementation, reliability, tests and delivery now belong to #404.
Use that instrumentation once reliable to correlate endpoint ingress, host
output and the visible incident. Closing or shipping #404 must not close this
investigation.

## Done when

- At least one actual stray-text occurrence has been captured and analysed with evidence identifying the faulty layer, or a repeatable reproduction establishes it.
- A regression reproducer and fix, or a precise upstream report with evidence, address the established cause; instrumentation alone does not satisfy this issue.

## Plan

- [x] Preserve operator observations, isolation audits and negative replay results in the Log.
- [x] Transfer tracing implementation and its durable plan to #404 for independent delivery.
- [x] Capture and diagnose a real occurrence using reliable tracing from #404.
- [x] Reduce the incident to a regression reproducer and fix or upstream report.

## Log

### 2026-10-01

- Filed from a diagnosis conversation; hypotheses above. The operator could not
  say whether the earlier sightings coincided with typing.

### 2026-10-02 — Additional operator evidence

- Claude's turn-closing text appeared again, this time in the **right pane
  running Neovim**, with Neovim in **insert mode**.
- The operator was **not typing anything** when it appeared. Active operator
  typing is therefore not required for an occurrence; concurrent background
  redraws remain possible but were not established by this report.
- This report establishes visible stray text, not whether it entered Neovim's
  buffer. The exact text, duration, and deciding output stream were not captured
  in this report, so the responsible layer remains unconfirmed.

### 2026-10-03 — Cursor flicker during some occurrences

- In at least some occurrences of the stray turn-closing text, the operator
  also sees the cursor "flickering": its visibility changes irregularly rather
  than blinking at a regular pace.
- The operator describes the appearance as excessive redrawing. This is an
  observed visual correlation, not yet a measured redraw rate or proof of the
  cause; it has not been reported for every occurrence.

### 2026-10-03 — Pane isolation audit

Audited Pair `37af674e`, the installed Zellij 0.45.1, and the local Zellij
repository's `v0.45.1` source. This was an audit with temporary probes, not an
implementation or a reproduction of the reported misplaced text.

- **Boundary:** Zellij owns the visible agent/draft/right-pane grids and their
  parsers (`TerminalPane` owns a `Grid` and `vte_parser`). Couch's `Endpoint`
  consumes the entire Zellij client screen, not each visible pane separately.
  Normal Zellij text rendering translates pane cells to screen coordinates;
  normal Couch rendering consumes validated cells, not raw escape passthrough.
- **Redraw coupling confirmed:** `terminal.RenderWithHistory` marks a frame
  dirty for any cell/cursor change. `HistoryRender.Emit` reconstructs the whole
  viewport, hides the physical cursor, and restores its position/style/visibility
  inside a synchronized-output bracket. A production Presenter probe changed
  exactly one left-region cell while preserving right-region cells and cursor:
  it emitted 244 bytes including unchanged `RIGHT-NVIM` / `idle editor` text and
  cursor hide/show/style controls. An unchanged subsequent frame emitted nothing.
  See `cmd/internal/terminal/history_render.go` and `render.go`.
- **Isolation checks passed:** 14 Endpoint subcases (whole and bytewise delivery)
  covered out-of-bounds cursor motion, erases, alternate screen/modes, synchronized
  hold, OSC effects, cursor query, and unknown DCS. The second Endpoint's
  publication, modes, and input remained unchanged. Reproduce while temporary
  files remain with `go test -overlay /tmp/pair379-isolation-audit/overlay.json
  ./cmd/internal/terminal -run '^TestAudit379' -v -count=1`.
- **Native pane checks passed:** a disposable 80x20 two-pane Zellij 0.45.1 session
  kept the right pane focused and idle. Seven left-pane scenarios exercised the
  reported relative turn-end sequence (with a synthetic marker), cursor bounds,
  erase/scroll regions, alternate screen, reset, bytewise synchronized drawing,
  and unknown strings. Each stage showed the left stimulus, preserved the right
  pane's native dump and host marker, and never showed the left marker in the
  right region at captured read boundaries. Host output was independently parsed
  by pyte; this tests cells, not physical blink timing. Probe:
  `/tmp/pair-379-audit/native.py`; final capture:
  `/tmp/p379-r7i7hl4q/zellij.raw`. Fixture KDL syntax and pyte query-handler errors
  were corrected before the successful run; neither was a product failure.
- **Existing suites passed:** `go test ./cmd/internal/terminal
  ./cmd/internal/ptychild ./cmd/internal/termcmd ./cmd/internal/couchtty -count=1`.
- **Deliberate global effects:** bell/title/clipboard/notifications have decoded,
  policy-controlled paths through `Presenter.EmitEffects`. They are exceptions
  to "everything stays inside", but do not authorize arbitrary screen drawing.
- **Limits and next discriminator:** excessive shared repaint can explain cursor
  disturbance without escape leakage; actual host blink-timer behavior remains
  unmeasured. Misplaced text still requires a rendering/parser/stream-ordering
  fault somewhere, not merely a redraw. Compare pane-local state, Zellij client
  bytes, Couch's parsed frame, and final host rendering at an occurrence to find
  the first divergence. These finite synthetic cases do not exclude a timing-
  dependent fault. ARCH-PURPOSE / ARCH-MOCK: Endpoint isolation alone cannot prove
  the visible Zellij pane boundary; both boundaries were exercised separately.

### 2026-10-03 — Claimed in pair:2; deeper cursor/stream audit

- Operator requested claim here and deeper investigation. `sdlc claim --issue
  379` reserved the card for `pair:2`; `sdlc start-plan --issue 379` created
  `000379-stray-turn-end-text` from fresh main. Earlier evidence was restored
  after the branch transition and checkpointed with `sdlc issue sync`.
- Clarified the operator's timing hypothesis: separate virtual cursors become
  one focused cursor in the combined screen. The physical cursor temporarily
  visits all rows during painting. Wrong positioning, interleaved output, or
  parser/width disagreement could put text in another region; merely exposing
  an incomplete synchronized frame explains flicker but not text relocation.
- Exact Zellij v0.45.1 source: `output/mod.rs` positions each character chunk
  absolutely; client Render and forwarded-query instructions share the client
  loop. Startup/resize/nested-session writers emit complete control strings,
  not application text. OSC 9;4 progress is filtered as a ConEmu subcommand.
  No ordinary raw-text escape route found. `tab/mod.rs:4968` reasserts focused
  cursor visibility, position and shape when any output is dirty, so cursor
  disturbance can arise upstream of Couch as well.
- Extended native probe with right-pane window-size/color queries and cursor
  redraws passed all seven left-pane scenarios. Host replies were absent,
  exercising timeout/resume. The oracle checks marker completion byte by byte,
  so a correcting redraw later in the same read cannot hide marker leakage.
  Temporary probe: `/tmp/pair-379-audit/native-query-transient.py`; capture:
  `/tmp/p379-4yd3rbj9/zellij.raw` (44,160 bytes). This is synthetic traffic,
  not yet proof for Neovim or Ghostty.
- 500 deterministic generated ASCII/CJK editing sequences (seed 379), including
  byte-split input, cursor save/restore, erase/insert/delete/scroll, alternate
  screens and synchronized holds, matched an independent xterm/headless cell
  oracle after production `RenderWithHistory` rendering. This checks completed
  frames, not every interleaving or physical blink behavior.
- A conditional placement fault was reproduced: `👩‍👩‍👧‍👧turn closed` puts
  the text at column 2 in Endpoint's model but column 4 in the test xterm host
  (zero-based). `historyEmitter.cells` emits sequential text based on the
  assumed character widths. This establishes a width-disagreement mechanism,
  not that Ghostty or the incident had this disagreement. Installed Ghostty
  reports 1.3.1; no live-host width or blink test was performed.
- Verified deeper probes with `go test -overlay
  /tmp/pair379-isolation-audit/overlay.json ./cmd/internal/terminal
  -run '^TestAudit379(History|Width)' -v -count=1` (pass). Probe code/dependencies
  are temporary, outside the repository; no production behavior changed.
- Next decisive measurement remains the first divergence between pane-local
  state, Zellij output, Couch publication and host output at a real occurrence.
  A recorder of Zellij input alone would not distinguish Couch's parser from
  its renderer or the physical host; capture both sides and their geometry and
  ordering if instrumentation is implemented.

### 2026-10-03 — Neovim insert-mode and closing-frame experiments

- Actual right-pane `nvim --clean -n -i NONE` remained in insert mode through
  17 direct-Zellij scenarios (original seven plus ten repeated turn-end frames).
  Its timer-sampled buffer stayed `["RIGHT379"]`; the file and native pane dump
  stayed unchanged. The bytewise host marker oracle found no crossing.
  Temporary probe: `/tmp/pair-379-audit/native-nvim-dcs.py`; capture:
  `/tmp/p379-m6n9wzmv/zellij.raw` (89,116 bytes).
- Corrected another independent-oracle limitation: pyte prints unknown DCS
  payload instead of consuming it. Zellij's periodic nested-session announcement
  therefore appeared at the focused cursor in the initial probe, while the
  native right-pane dump remained correct. Its payload was base64 session
  metadata, not closing prose. The probe now discards complete DCS strings.
  Tested that captured announcement separately through production Endpoint at
  every split and bytewise: cells/cursor unchanged, no effects. Verified with
  `go test -overlay /tmp/pair-379-audit/dcs-overlay.json
  ./cmd/internal/terminal -run 'TestAudit379.*DCS' -v -count=1` (pass).
- Extracted a real 192-byte closing transition at offset 127131 of
  `repos/11873b98e33bf004/scrollback-1-pair-7-claude.raw` into
  `/tmp/pair-379-audit/claude-close.raw`. It uses RGB grey
  `ESC[38;2;153;153;153m`, not standalone faint `ESC[2m`. Neither command moves
  the cursor. The frame includes a progress-stop OSC, synchronized frame
  boundaries, relative cursor movement, the closing text, and cursor restore.
- Six actual-frame variants beside idle insert-mode Neovim passed: RGB grey,
  standalone faint instead, no SGR, no progress OSC, no synchronization markers,
  and every-byte delivery with 4ms pauses. Each starts from the same explicit
  cursor position; only `Crunched for 1s` is replaced with the left marker.
  Probe: `/tmp/pair-379-audit/native-nvim-ablation.py`; capture:
  `/tmp/p379-p79c14kh/zellij.raw`. These direct replays do not exercise Pair's
  wrapper completion/notification lifecycle or Ghostty's actual rendering.
- Closing-only correlation remains unexplained. Two hypotheses to distinguish:
  completion's frame/progress/notification transition triggers the fault, or
  subsequent silence leaves a transient fault visible that ongoing output would
  rapidly erase. Styling is not currently supported as the trigger.
- Operator clarified that the stray closing line belongs to the **currently
  visible Claude thread**, not another/background Couch thread. Prioritize
  corruption within that displayed Zellij screen over cross-thread routing.

### 2026-10-03 — Operator refines confidence and reproduction priority

- Supersedes the source-thread certainty in the preceding note: the operator
  **strongly suspects** the current Claude thread, but is less certain where the
  text originates. Do not exclude another Claude thread based on that answer.
- Stronger observations: the artifact is Claude's **turn-stop text**, occurs
  somewhat regularly around that sequence, and has not been noticed elsewhere
  or when using Codex. The operator strongly suspects the stop sequence's
  interaction with Couch/Zellij/Pair, rather than a generic redraw problem.
- Prioritize faithful Claude completion reproduction: preceding terminal state,
  exact stop-transition bytes, byte/chunk timing, and wrapper completion hooks.
  Passing short synthetic/ablated sequences does not rule out those interactions.
- Wrapped-transition probe also passed: fixture named `claude` through the
  freshly built candidate `pair-go wrap`, with a private storage root/tag and
  no-op slug helper. `wrap.log` proves progress start/stop, `SLUG-spawn`, and
  canonical OSC 777 emission; `slug.receipt` confirms helper dispatch. After
  two seconds, Neovim's insert mode, buffer/file/native pane dump and host marker
  isolation were intact. Probe:
  `/tmp/pair-379-audit/native-nvim-wrapped-receipt.py`; evidence:
  `/tmp/p379-8zqgeq7m/{wrap.log,slug.receipt,claude.raw,zellij.raw}`.
  Initial fixture attempts lacked the private `PAIR_TAG` and were rejected by
  the storage guard before the wrapper ran; those attempts are not evidence
  about terminal behavior.
- No incident reproduction yet. The native probes cover Zellij and the wrapper
  against pyte; the separate renderer probes cover Couch components against
  xterm/headless. Neither is a complete production Couch-to-Ghostty reproduction.
  Preserve an immutable full Claude capture plus matching geometry/chunk records
  before attempting faithful replay: the live `1-pair-7` raw file changed during
  this investigation, so the later prefix read cannot reconstruct the earlier
  closing occurrence. The separately extracted 192-byte close frame is retained.

### 2026-10-03 — Existing live logs: fixed geometry and split closing frame

- Operator reports dimensions stayed fixed during incidents; although draft height
  can change, they did not resize it. Treat resize as absent in the reported case.
- Current Couch mapping: pair:1 is Claude, tag `couch-9d6fdf5eecace248` in
  repo storage `68f32487e79d85bd`; pair:2 currently has Codex capture `1-pair-8`.
- Preserved pair:1 raw (7,880,514 bytes), geometry/time sidecar, and wrapper
  events (104,060,063 bytes) under `/tmp/pair379-live-1791053793/` with source
  inode/size/SHA-256 manifest. Rechecked all three live prefixes against the
  snapshot hashes: unchanged. These are local diagnostic artifacts, not committed
  transcript contents. `analyze.py` and JSON reports accompany the snapshot.
- Found at least 17 literal duration/`done` footer matches (not an exhaustive
  terminal-rendered-text count). Eight recent selected matches have hash-verified
  ingress chunks. Examples: `Brewed for 3s · done 1:41 AM` at offset 7,470,752;
  `Baked for 37s · done 2:01 AM` at offset 7,605,835, on October 3.
  Seven of those eight occur after the last recorded geometry change, all 94×39.
  Styling in sampled frames is RGB grey, not standalone SGR 2.
- Concrete split-frame case at 01:41:26: wrapper accepted a 1,024-byte write
  at .833304 containing sync-begin and the footer, but not sync-end; a 40-byte
  notification write followed at .854234; the remaining 338-byte write containing
  cursor restore and sync-end followed at .989995 (156.691 ms after first receipt).
  Both raw portions match the recorded output hashes, with full accepted lengths
  and no write errors. The insertion boundary is between ordinary text `for` and
  ` agents`, **not inside an escape command or UTF-8 character**. This is a
  concrete completion-interleaving replay candidate, not evidence that a pane
  leaked or that this particular completion was visibly faulty.
- `split-replay.json` records the two exact byte segments, preceding-state prefix
  offset and receipt timing. The notification payload itself is not captured;
  its length/hash are known, so the recipe explicitly leaves that payload unknown
  rather than substituting assumed bytes. Receipt timestamps are not host paint
  timestamps. Five selected footer-containing batches independently match raw
  concatenation hashes; the 02:01 footer arrived in a complete 2,800-byte batch.
- Logging boundary audit: raw records Claude PTY output; wrapper queue/flush
  receipts substantiate accepted writes into Zellij. Existing Couch traces do
  not persist Zellij→Endpoint or Presenter→Ghostty display bytes. The former has
  only an in-memory 128 KiB ring; the latter has no byte capture. Thus these logs
  support faithful input/chunk reproduction but cannot establish where text was
  painted in the other pane. No incident reproduction or root cause claimed.

### 2026-10-03 — Full-prefix reproduction attempt

- Recovered the previously unknown 40-byte notification: canonical
  `ESC]777;notify;pair;agent stopped working BEL`. Its SHA-256 begins
  `b85c08c341da`, exactly matching the captured write receipt. This is the
  grace/watchdog completion message in `notification_lifecycle.go`.
  Saved verified bytes as `notification.raw` and updated `split-replay.json`.
- Temporary native fixtures replay all 7,470,588 preceding Claude bytes, then
  the exact 1,024-byte close fragment and 338-byte tail separated by 156.691 ms.
  Two variants: no notification and exact notification inserted at +20.930 ms.
  Both passed: the actual `Brewed for 3s` footer remained left; focused Neovim
  stayed in insert mode with unchanged buffer, file and native pane dump.
- Actual left PTY ioctl verifies 94×39. Zellij requires a 189×39 host for this
  fixture: 94 content columns, one divider, 94 right-pane columns. Prefix used
  fixed geometry and accelerated delivery, not historical resize/timing replay.
- Scripts: `/tmp/pair-379-audit/native-nvim-full-prefix.py` and
  `native-nvim-full-prefix-notify.py`. Captures respectively:
  `/tmp/p379-56zh7mup/` and `/tmp/p379-700_spw2/`, including `zellij.raw`,
  monotonic read-chunk records, Neovim snapshots and before/after pane dumps.
  Native observation uses pyte with DCS suppression, not actual Ghostty.
- Next diagnostic boundary if recorded sequence remains isolated: capture a
  real occurrence at Zellij→Couch and Couch→host, with chunk timestamps,
  geometry, actor identity and accepted write counts. Replay must distinguish
  malformed Zellij screen state, Couch rendering/routing, and host display.
  A passing selected frame is not a reproduction of the reported incident.

- Downstream replay of both native captures through production Endpoint and
  Presenter also passed, using recorded native chunks and bytewise feeds into
  the parser. The independent xterm observer checked every actual parent write
  boundary, including effect writes and asynchronous paints: baseline 152
  boundaries / 84 footer-left matches / zero footer-right; notification variant
  157 boundaries / 87 footer-left matches / zero footer-right, in both feed modes.
  `EmitEffects` runs before `Present` with all active effect policies enabled.
  Verified the exact `agent stopped working` OSC777 occurs once in each
  notification-variant host stream and zero times in baseline host streams.
  Downstream raw/frames/boundaries/reports reside in each native capture directory.
  Temporary overlay: `/tmp/pair379-isolation-audit/replay_test.go`.
  This exercises renderer components, not full live Console scheduling or Ghostty.

### 2026-10-03 — Guarded live capture delivered

- Operator authorized instrumentation. Implemented `COUCH_CAPTURE_DIR`, requiring
  an absolute confined path under `COUCH_ISOLATED_ROOT`; off by default and cleared
  in launched child environments. Built `bin/couch`; existing sessions untouched.
- One private session JSONL stream records pre-parser Zellij bytes and applied
  geometry under Endpoint ordering, thread bindings, host geometry, presentation
  transitions, and exact host write receipts (including partial success/error).
  Schema includes sequence, wall/monotonic timing, byte chunks and build identity.
- Recorder has one asynchronous writer: 128 records / 8 MiB retained record
  budget, 256 MiB file limit, explicit incomplete ending on loss, and bounded
  two-second close. Terminal restoration precedes recorder drain. Normal exit
  and startup failure both report capture failure once. Regular-file cancellation
  limitations are explicit in the runbook; no silent resume after dropped data.
- Fresh-eyes plan review clarified applied resize followed by reply failure and
  bounded shutdown. Implementation review caught early-return error suppression
  and inherited activation; both fixed, with regression coverage. Review approved.
- Tests: full `terminal`, `ptychild`, `couchcore`, `couchcmd`, `couchtty`, and
  `terminalcapture` suites passed. Race suites passed for terminal/ptychild and
  recorder; focused race coverage passed for the real PTY observer and capture
  integration. Real PTY capture verifies the marker at both boundaries with
  identity/geometry, unchanged partial-write behavior, no default-on files, path
  guards, early launch errors, and concurrent admission/close.
- Initial real-PTY observer fixture hung joining a blocking master Read after
  close on macOS; switched the test to existing contextual ttyio transport and
  explicitly canceled/joined it. This was fixture teardown, not a capture failure.
- `make build` passed and rebuilt Couch (existing `.skip-make-build` skips the
  unrelated pair-go build). Built binary refuses capture without isolation.
  `git diff --check` passed. Artifact inventory check has pre-existing unrelated
  failures; new capture sources are classified. Baseline comparison is retained
  under `/tmp/pair379-artifact-{baseline-generated,current}.log`.
- Operator instructions and schema/extraction example:
  [isolated live capture](../../atlas/couch-live-capture.md).
  Capture implementation completes this requested step; the original occurrence
  analysis criterion remains unchecked, so #379 is not closed or claimed fixed.

### 2026-10-07 — Regular Couch opt-in correction
- 2026-10-07: closed — BR-1 addressed: unsupported file-based benchmarks removed; go test ./ansiparser -count=1 -bench . -benchtime=1x passes. No production delta since SHIP review; prior Endpoint mutation, captured replay, vt/parser race, focused observer/notification tests and Couch build evidence retained.; review verdict: SHIP
- 2026-10-07: closed — Captured incident replay: 23025 feeds, 798 left-footer observations, zero right leaks after fix. Permanent Endpoint and wrapper regressions fail against upstream and pass with repair. vt/parser race suites, full terminal tests, focused observer/notification race tests, Couch tests/build pass. Broader notification helper race reproduced unchanged on main; /tmp/p379-baseline-race.log.; review verdict: SHIP
- 2026-10-07: flow upgraded quick → full — 520 added lines in code files (limit 100)

- Removed mandatory isolated-root activation. An absolute COUCH_CAPTURE_DIR now
  enables capture for regular Couch; explicit isolated runtimes retain confinement.
  Default-off and descendant activation clearing remain unchanged.
- Regular-Couch acceptance test first failed on the old guard, then passed after
  the change. Real PTY tests cover both regular and isolated capture. Fresh-eyes
  review approved; binary rebuilt and runbook simplified for regular restart.
- Full couchcmd run encountered continuation-writer acceptance failures caused by
  repository-scope/root conflicts during helper restart, outside the capture tests.
  Targeted capture tests passed. Baseline check retained in
  `/tmp/pair379-regular-baseline.log`; no rendering fix is claimed.

### 2026-10-07 — Recording resumed with shipped tracing

- #404 landed through PR #206. Operator restarted ordinary Couch with `COUCH_CAPTURE_DIR="$HOME/.local/share/pair/captures" COUCH_CAPTURE_MAX_MIB=4096 couch`.
- New recording: `/Users/xianxu/.local/share/pair/captures/session-2616925832/events.jsonl`; started 14:00:35 PDT (21:00:35 UTC), PID 61877, clean build revision `86da11d31a2d64a91986e8848e69e00345c4efaf`, budget 4,294,967,296 bytes. At 21:05:33 UTC the readable prefix was 93,317,062 bytes, including 49,602 endpoint feeds, 892 host writes, ten endpoint opens/bindings, geometry and selection records. Partial trailing lines were excluded while the writer remained active; no capture-end appeared in this snapshot. This confirms ongoing recording, not a finalized complete capture or a reproduction.
- Another earlier capture, `session-1849422337`, used old build `0b384b45` and ended incomplete after 7.53 seconds (queue limit). Preserve as instrumentation evidence; it does not implicate the shipped recorder.
- Operator requested clearer badge emphasis. Delivered separately as #405 / PR #209: bold reverse-video capture label, existing scoped reset; then resumed this issue and merged current main. #379 remains open, awaiting an observed incident with its approximate time and thread/pane. Preserve the entire capture directory and exit Couch normally after an incident to finalize evidence before replay analysis.

### 2026-10-07 — Proactive scan without a witnessed occurrence

- Operator has not noticed another occurrence but requested checking the logs for a missed one. Inspected `session-2616925832` in full (182,541,017 bytes, 99,599 records, complete at 14:11:49 PDT) and a fixed live prefix of `session-2834647007` (878,552,357 bytes, 418,959 records through 14:53:41 PDT; started 14:12:09). New session uses clean build `6986321e216f9da591464e5991bcb03a5675fdf4`. Both streams have contiguous sequence numbers, no recorded errors, no short host writes, and constant recorded host geometry 191×54. The new session was still actively growing; no completeness claim for that live prefix.
- Replayed accepted host bytes, in recorded write order with host geometry, through the existing independent xterm/headless oracle. Checked screen rows after all 1,752 + 12,200 = 13,952 host writes. The duration-footer heuristic found matching left-half text in 309 + 2,792 = 3,101 frames, and zero right-half matches. Left-side labels included Claude’s Sautéed/Brewed/Baked/Churned/Cogitated/Cooked/Crunched and Codex’s Worked; matches are frame observations, not distinct completed turns.
- No evidence of a missed stray-text occurrence in this check. Limits: heuristic footer detection, midpoint pane classification, xterm rather than Ghostty, accelerated replay without wall-clock delays, and inspection after each host write rather than intermediate paint states within a write. These negative results do not establish absence of a transient live rendering fault or its cause.
- Private analysis scripts, extraction metadata, row-level replay results and usage skill: `/tmp/p379-logcheck/`. Source captures remain untouched. Continue waiting for an observed occurrence and approximate time to narrow a faithful replay.

### 2026-10-07 — New witnessed occurrence after capture capacity exhausted

- At approximately 19:34 PDT the operator reported another occurrence and a full REC indicator. Exact occurrence time, visible thread and footer wording have been requested and remain unconfirmed.
- `session-2834647007/events.jsonl` ended at 18:30:03.961 PDT with seq 2,015,319, status incomplete, error `terminal capture file limit reached`; size 4,294,963,014 bytes. Its 4 GiB budget lasted about 4h18m from 14:12:09. If the new occurrence was contemporaneous with the report, its Couch ingress/host-write evidence is unavailable. The earlier completed recording and this capped prefix remain intact.
- Preserved fixed-length snapshots of raw Claude output, timestamp sidecars and wrapper-event logs for the three recently active Claude sources (`11873b98e33bf004/1-pair-7`, `5749d0ffa92b055d/1-pair-6`, `2e51fcf9799b1d8f/couch-6b111ea230c149dc`) under private `/tmp/p379-incident-20261007-1934/`: nine files, 215,152,829 bytes, SHA-256 manifest with source sizes/mtimes. These source logs may narrow the emitted sequence but cannot establish its actual placement in Ghostty. No causal claim or reproduction yet.

### 2026-10-07 — Captured brain:0 occurrence and isolated parser defect

- Operator reported a new occurrence in brain:0’s right-hand nvim pane, approximately 19:58 PDT and several seconds before reporting; wording not remembered. New 16 GiB recording `session-1484782486` was active and not full. Preserved a fixed 254,011,507-byte prefix under private `/tmp/p379-incident-1958/events.jsonl`, SHA-256 `0ea9ba3bc94a91d1a31f6827f3e8aa33ea03657b78ebf46e57308b8bdcfa2e26`, through seq 113424. No sequence gaps or recorded errors in the readable prefix. Full source capture remains live; this is not a finalized capture-end claim.
- Independent xterm replay of 1,909 accepted host writes reproduces `Crunched for 12s · done 7:57 PM` in the right pane at host seq 108130, 19:57:49.201027 PDT, screen row 40 / text column 104, beside nvim line number 74. Endpoint `couch-pty-1` binds scope `2e51fcf9799b1d8f`, tag `couch-6b111ea230c149dc`; pane metadata confirms `/Users/xianxu/workspace/brain`. Source raw-output, sidecar, wrapper logs and hashes also preserved under the incident directory.
- Preceding ingress seq 108117 at 19:57:49.181598 contains exactly a synchronized-output-wrapped OSC9 notification: `ESC ] 9 ; pair: ✻ Crunched for 12s · done 7:57 PM BEL`. Separately, seq 108127 paints the legitimate left footer. Independent xterm replay of the entire brain endpoint feed shows no right-pane footer, while recorded Couch host output does. This isolates the observed corruption to Couch’s terminal processing/output boundary, not a requirement for Ghostty-specific rendering behavior.
- Reduced to a deterministic production Endpoint probe with geometry 191×53 and cursor row 40/column 103. Feed only the notification, then snapshot: ASCII `*` and `ready; café` controls leave the screen empty and emit intact notification effects; `✻` leaks the suffix onto the screen and emits a truncated replacement-character notification plus a bell. The temporary overlay regression fails as expected. Files: `minimal_test.go`, `minimal-overlay.json`, `minimal-result.log` under `/tmp/p379-incident-1958/`. Run `go test -overlay /tmp/p379-incident-1958/minimal-overlay.json ./cmd/internal/terminal -run '^TestIncident379OSCUnicode$' -count=1 -v`.
- Root mechanism independently audited: `third_party/vt/emulator.go:311` feeds raw bytes to x/ansi v0.11.7. Its `parser/transition_table.go:264–269` collects OSC bytes but overrides `0x9c` as C1 ST → Ground. UTF-8 `✻` is `e2 9c bb`: the continuation byte prematurely ends OSC; remaining notification text prints at the saved cursor, often nvim’s cursor. SGR2 and redraw timing are not necessary triggers. Cursor position controls where the leaked text appears.
- Repair scope should cover control-string parsing, not strip Claude’s marker or patch only notification handlers (ARCH-PURPOSE). Audit found the same 0x9c collision for DCS; SOS/PM/APC also inherit UTF-8 transitions that escape string state. Those related families are static findings pending runtime regression coverage. Fix must preserve valid UTF-8 payloads across chunk splits, retain intended standalone C1 handling, and honor existing bounds. No production fix has been made yet.

### 2026-10-07 — Parser repair and permanent regressions

- Added narrow licensed `third_party/vt/ansiparser` fork of x/ansi v0.11.7 streaming parser; reused upstream handler/value types and transition table. Constant-space UTF-8 prefix state preserves all five control-string families across chunks and overflow, validates first-continuation ranges, and leaves actual standalone controls/cancellation/reset semantics intact. All three raw streaming consumers migrated: vt emulator, wrapper control observer, notification-output boundary (ARCH-DRY/ARCH-PURPOSE). No notification text filtering or global table mutation.
- Permanent Endpoint regression models the recorded cursor at row 40/column 103, asserts unchanged cells and exactly one intact Claude footer notification, and covers all splits/bytewise feeds. Verified it fails with original parser via `/tmp/p379-original-parser-overlay.json` (log `/tmp/p379-endpoint-red.log`). Wrapper observer regression likewise fails against upstream, then passes after migration. Low-level tests cover all five strings, UTF-8 boundary values, malformed prefixes, termination/cancellation/reset and overflow. Retained upstream streaming tests with one correction to their formerly expected corrupt OSC payload.
- Validation: full terminal suite passes; nested vt + parser `-race ./...` passes; targeted terminal/wrapper endpoint, observer, notification-output race checks pass; Couch capture/notification tests pass; Couch build and diff check pass. Broader notification-race run found unchanged `TestCodexWorkingNotificationReachesCouchStatusAndSwitcher` helper's unjoined io.Copy vs Close race; reproduced independently with all modified consumers overlaid from origin/main 48d5c69 (`/tmp/p379-baseline-race.log`). This unrelated preexisting race remains.
- Captured brain endpoint replay through repaired production Endpoint: 23,025 feeds, 798 legitimate left-footer observations, zero right-footer occurrences (`/tmp/p379-fixed-replay.log`). Original accepted host output remains preserved showing the defect; no new live-session outcome claimed before restart.
- Upstream check requested by operator: exact issue https://github.com/charmbracelet/x/issues/848 remains open; PRs https://github.com/charmbracelet/x/pull/946 and /976 unmerged/open; current main parser still faulty. PR /886 closed unmerged and intentionally removed bare C1 ST support. Local fork provenance/retirement path documented; no external report or comment sent.
- Sibling audit: local `cmd/internal/ansi` stripping and terminal OSC8 framing do not use the faulty C1-ST rule. Local Frame's lack of DCS/SOS/PM/APC containment is an existing separate regex-equivalence limitation. Trusted ASCII SGR DecodeSequence use is not an exposed Unicode string path. Remaining work: close-time review and delivery.

## Revisions

- 2026-10-07 — Corrected the interpretation of “isolated session”: operator wants
  explicit per-launch capture on regular Couch. Mandatory isolated storage was
  an agent misunderstanding. Remove that requirement; retain opt-in/default-off
  capture and optional isolation confinement. Existing threads need only reattach
  to the capture-enabled Couch, not restart their agents.

- 2026-10-03 — Operator approved implementing live capture, guarded and only
  enabled in an isolated session. Extend the initial recorder spec to both
  Zellij→Couch and Couch→host boundaries, exact byte/timing/geometry records,
  bounded asynchronous capture with explicit incomplete status, private files,
  and isolated-root activation. Rendering fix and occurrence analysis remain
  outstanding; capture delivery alone does not close this issue.

- 2026-10-03 — Operator requested an isolation audit after reporting cursor
  flicker. Clarified the boundary and added measured shared-redraw evidence in
  the Log. Earlier statements that misplaced text *means* a lost cursor move
  remain hypotheses, not established causes; flicker alone does not establish
  cross-pane escape leakage. No implementation or acceptance criteria changed.

### 2026-10-07 — Tracing split and live capture failure

- Operator requested #404 for tracing implementation and independent delivery; #379 remains the long-running diagnosis. The implementation plan moves to `workshop/plans/000404-couch-live-terminal-tracing-plan.md`; all historical evidence below/above remains preserved here.
- Regular capture `/Users/xianxu/.local/share/pair/captures/session-2417847440/events.jsonl` stopped after 6.88 seconds: 5,609,068 bytes, 3,280 records, `capture-end` incomplete with `terminal capture queue limit reached`. It cannot supply evidence of later incidents. This is a tracing reliability defect owned by #404, not evidence of the display bug's cause.
- Split preserves original implementation commits 9ca17478 and 0b384b45 in a backup branch. The independent implementation branch is based on main; this branch retains investigation evidence only.

## Revisions — scope separation

- 2026-10-07 — Operator requested independent tracing delivery. Current Spec, Done when and Plan now cover diagnosis only; tracing and its outstanding queue-overflow/visibility/retention work move to #404. Historical Log entries and prior revision records are retained, including earlier instrumentation acceptance decisions that this revision supersedes.

- 2026-10-07 — Actual incident and deterministic reduction supersede the timing-only hypothesis: notification UTF-8 is misparsed as a control terminator in Couch’s terminal backend. The remaining step is a parser-level regression and repair covering related control-string families; instrumentation is sufficient.

- 2026-10-07 — Operator authorized parser repair and sibling-path audit. Implementation design: `workshop/plans/000379-stray-turn-end-text-plan.md`. Share a narrow licensed streaming-parser fork across vt and wrapper observers; preserve valid UTF-8 control-string data and standalone controls, enforce existing bounds, and test all five families across chunk splits. Corrected audit: local `cmd/internal/ansi` owns stripping, so it is not automatically implicated by x/ansi’s Strip implementation.

- 2026-10-07 — Close review SHIP, BR-1 minor: two imported benchmarks referenced absent upstream fixture files. Removed those unsupported benchmark entrypoints and their unused os import; retained all compatibility tests and self-contained benchmark. Added lesson and provenance note; rerun nested suite and remaining benchmark before advancing close evidence.
