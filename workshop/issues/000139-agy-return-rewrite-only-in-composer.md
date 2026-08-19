---
id: 000139
status: working
deps: []
github_issue:
created: 2026-08-16
updated: 2026-08-17
estimate_hours: 5.83
started: 2026-08-17T10:59:29-07:00
---

# Agy Return rewrite only in composer

## Problem

Agy currently participates in the pair Return remap convention, while overlay
handling depends on known visible prompt markers. Codex now uses a safer rule:
rewrite plain Return only when Pair positively identifies the live
composer/input box. Agy should follow the same contract so permission pickers
and future UI variants keep plain Return as confirm instead of receiving an
accidental newline.

## Spec

### Goal & Problem
Antigravity (`agy`) previously participated in un-gated Return remapping (mapping plain Enter to `\n` without checking composer presence), relying solely on `agyPickerMarkers` to catch overlay dialogs. Codex (#137) and Muse (#140) established that plain Return must rewrite to newline (`\n`) **only** when Pair positively identifies that the agent is in an active input composer with a visible cursor. If the composer state is inactive, unknown, or if an overlay is active, plain Return must pass through as bare `\r` (CR) so confirmation pickers, shortcut dialogs, and external tools receive normal confirmation instead of an unintended newline.

To preserve strict workstream isolation with open issue #140 (which tracks Muse), `agyComposerTracker` lives in `cmd/internal/wrapcmd/agy_composer.go`, following the established per-agent tracker pattern (`codex_composer.go`, `muse_composer.go`) while providing comprehensive VT escape handling and positive composer gating.

### Architecture & Components

1. **Pure Terminal State Machine (`cmd/internal/wrapcmd/agy_composer.go`)**:
   - `agyComposerTracker` maintains screen dimensions (`rows`, `cols`), cursor coordinates (`cursorRow`, `cursorCol`), cursor visibility (`cursorVisible`), and per-row painted features (`agyBorderRows`, `agyPromptRows`).
   - ANSI escape parsing handles:
     - Absolute cursor positioning: `CUP` (`\x1b[<r>;<c>H`, `\x1b[<r>;<c>f`), `CHA` (`\x1b[<c>G`), `VPA` (`\x1b[<r>d`) — parameter defaults to 1.
     - Relative cursor positioning: `CUU` (`\x1b[<n>A`), `CUD` (`\x1b[<n>B`), `CUF` (`\x1b[<n>C`), `CUB` (`\x1b[<n>D`) — parameter `<n>` defaults to 1.
     - 2-byte escapes: `RI` (`\x1bM` -> move cursor up 1 line, clamped to 1).
     - Carriage control: `\r` (col 1), `\n` (row+1, col 1).
     - Cursor visibility: `DECTCEM` (`\x1b[?25h` -> visible, `\x1b[?25l` -> hidden).
     - Screen mutation & invalidation:
       - `ED` (`\x1b[<n>J`): mode 0 (cursor to end), mode 1 (start to cursor), mode 2/3 (entire screen) clears `agyBorderRows` and `agyPromptRows` in the erased row range.
       - `EL` (`\x1b[<n>K`): erases prompt and border marks for `cursorRow`.
       - Non-border text print: printing non-border text at `cursorRow` clears `agyBorderRows[cursorRow]`.
       - `setWinsize`: prunes row entries $> rows$.
   - Chrome attribute tracking:
     - `agyBorderRows[row]`: set when Agy's horizontal border rule `─` (`\xe2\x94\x80`) is painted at `cursorRow` (requires $\ge 5$ consecutive `─` or full-line rule).
     - `agyPromptRows[row]`: set when Agy's prompt glyph `>` (`0x3e` / `\x1b[94m>`) is painted at `cursorRow` anchored at prompt column (`col <= 6`).

2. **Agy Composer Predicate**:
   - `agyComposerState.active() bool`: `cursorVisible && cursorRow > 0 && (isNearby(cursorRow) || isEnclosed(cursorRow))`, where:
     - `isNearby(r)`: checks if `agyPromptRows` or `agyBorderRows` is active on `r` or `r±1`.
     - `isEnclosed(r)`: checks if `r` is enclosed between an active top border / prompt row and bottom border row (`top <= r <= bottom` with `bottom - top <= 25`).

3. **Proxy Integration (`cmd/internal/wrapcmd/wrap.go`)**:
   - `proxy` holds `agyComposer *agyComposerTracker`.
   - `proxy.setWinsize` (at `wrap.go:2000`) calls `p.ensureAgyComposer().resize(int(ws.Rows), int(ws.Cols))`.
   - `proxy.handleChunk` feeds raw PTY stream chunk into `p.ensureAgyComposer().feed(data)`.
   - `proxy.emitPlainCR`:
     - If `pickerActive` is set $\rightarrow$ clear flag, log `adapt.Bypass` (overlay active), return bare `\r`.
     - If `agentBasename == "agy"` and `!p.agyComposerActive()` $\rightarrow$ log `adapt.Bypass` (agy composer inactive), return bare `\r`.
     - If composer is active $\rightarrow$ log `adapt.Fired` (newline remap), return `p.sendKM.plainCR` (`\n`).

4. **External Terminal Protocol Fake & Conformance (ARCH-MOCK)**:
   - Seam: `proxy.handleChunk(data, rolling)` receiving PTY output.
   - Stateful fake `agySessionFake`: models Agy UI lifecycle transitions (startup prompt, multi-line editing, overlay picker, thinking generation).
   - Integration tests replay these state transitions and verify `emitPlainCR` behavior.

### Invariants & Non-Goals
- **Precedence**: `pickerActive` from `detectAgyOverlayOpen` still takes absolute precedence over composer detection.
- **Alt+Enter**: Alt+Enter always submits bare `\r` (CR) across all agents regardless of composer state.
- **Non-blocking / Pure**: Telemetry and tracker failures are non-fatal and isolated.
- **Workstream Isolation**: Codex and Muse trackers remain untouched; no dependency on open issue #140.

## Done when

- `agyComposerTracker` encapsulates VT escape sequences, relative movements, and screen mutation invalidations cleanly (ARCH-DRY, ARCH-PURE).
- Agy plain Return rewrites to LF (`\n`) only inside a positively detected composer with visible cursor.
- Agy permission prompts/pickers still receive bare CR (`\r`).
- Inactive, hidden-cursor, or unknown Agy UI state receives bare CR (`\r`).
- Stateful fake and regression tests in `cmd/internal/wrapcmd/agy_return_test.go` and `cmd/internal/wrapcmd/agy_composer_test.go` pass (ARCH-MOCK).
- Existing Codex and Muse return remapping tests pass unchanged.
- `atlas/architecture.md` and `atlas/how-to-bring-up-a-new-harness-cli.md` updated with Agy positive composer gating details.

## Plan

- Historical: implement `agyComposerTracker` in `cmd/internal/wrapcmd/agy_composer.go` with relative/absolute cursor movements and mutation invalidation rules.
- Historical: add unit tests in `cmd/internal/wrapcmd/agy_composer_test.go`.
- Historical: wire `agy` composer tracking into `proxy` in `cmd/internal/wrapcmd/wrap.go` (in `setWinsize`, `handleChunk`, and `emitPlainCR`).
- Historical: add return routing and stateful fake replay tests in `cmd/internal/wrapcmd/agy_return_test.go`.
- Historical: run full test suite (`go test ./...`) to ensure zero regressions across all agents.
- Historical: update `atlas/architecture.md` and `atlas/how-to-bring-up-a-new-harness-cli.md`.
- [x] Complete the authoritative revised plan in `## Revisions`.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.

```text
superseded-model: estimate-logic-v3.1
item: issue-spec design=0.08 impl=0.02
item: smaller-go-module design=0.15 impl=0.25
total: 0.57
```

Re-derived after the unified plan cleared plan-quality. Produced via
`brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. `sdlc estimate-source` reports the
calibration source as stale, so the number is provisional but uses the required
method. The `x/vt` and `creack/pty` libraries halve design for the terminal and
PTY integration primitives; implementation remains at v3.1's 40% scale.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.30 impl=0.12
item: greenfield-go-module design=0.20 impl=0.32
item: smaller-go-module design=0.06 impl=0.20
item: api-integration design=0.30 impl=0.60
item: tui-screen design=0.40 impl=0.40
item: cross-cutting-refactor design=0.20 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: api-integration design=0.30 impl=0.60
item: real-api-discovery design=0.00 impl=0.24
item: real-api-discovery design=0.00 impl=0.24
item: real-api-discovery design=0.00 impl=0.24
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.04 impl=0.20
design-buffer: 0.15
total: 5.83
```

The two integrations are distinct: bounded PTY capture/lifecycle and fixture
replay/live conformance through the production proxy seam. The three real-API
discovery rows cover Agy, Codex, and Muse. The two smaller modules cover the
harness profile/router and the stateful fixture/fake support.

## Log

### 2026-08-16

- Follow-up from pair#137: Codex now rewrites Return only under positive live
  composer detection; agy should adopt the same safety model.
- Updated after pair#142 design review: Codex's cursor/paint detector is
  Codex-specific; agy should reuse the positive-detection contract, not the
  exact terminal heuristic.

### 2026-08-17

- Brainstormed positive composer detection for Agy. Captured raw terminal output from live `agy` CLI session.
- Identified that `agy` uses relative cursor movements (`CUU` `\x1b[2A`, `CHA` `\x1b[6G`, `RI` `\x1bM`) to position the cursor at the prompt row between horizontal rules `───` (`\xe2\x94\x80`).
- Sourced plan-quality gate findings: isolated Agy tracker ownership to `agy_composer.go` (avoiding entanglement with open issue #140), specified screen invalidation rules, named `proxy.setWinsize`, and defined stateful fake replay (ARCH-MOCK).
- Exact-window review `ab736d1^..ab736d1` returned `REWORK`; #140 was recorded
  as superseded (`d230fd8`) rather than falsely closed codecomplete, and #139
  absorbed its product findings.
- Revised plan-quality round 6 returned `CLEAN`, disposing PQ-3 and PQ-6. The
  clean pre-change baseline was `go test ./... -count=1`.
- Re-derived `estimate_hours: 5.83` with estimate-logic-v3.1 after the plan
  gate. `sdlc estimate-source` reports stale calibration; estimate-quality
  returned `INFO`, chiefly that terminal concurrency and three live harness
  captures may make the calibrated total optimistic.
- Task 1 landed as `9d3e27e`: a mutex-owned x/vt model with immutable
  snapshots, bounded stateful control observation, reply draining,
  deterministic concurrent close, and overflow-safe 4096-axis/262,144-cell
  allocation limits. Focused, race, full-package, and both fuzz targets passed.
- Task 1 fresh reviews both returned `APPROVED`. Review-driven corrections
  covered parser-state-aware C1 handling, coordinated concurrent close,
  purpose-level rather than x/vt-grid chunk equivalence, and pre-allocation
  dimension validation (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE).
- Task 2 landed as `9504da3`: one immutable-copying harness TTY profile lookup
  plus a pure Return decision. Exact profile/keymap/overlay contracts, overlay
  precedence, legacy behavior, and positive-gate fail-safe paths pass focused
  and full wrapcmd tests.
- Task 2 fresh spec and quality reviews returned `APPROVED` after reserving the
  gate-policy zero value for unknown and adding zero/out-of-range regressions.
  A full-package race probe also exposed an unrelated pre-existing
  `TestMasterPumpFlushesStdoutOnTick` bytes.Buffer race; Task 2 focused race
  tests pass and no Task 2 file appears in that trace.
- Task 2A landed as `20490b3` plus cap-test correction `197f816`. The bounded
  120x38 PTY seam owns one reader and one child wait, caps retained bytes at 1
  MiB, joins process/reader cleanup with primary errors, and passes injected
  operation failures plus real controlled-child lifecycle tests.
- Captured literal Muse `0.1.0-R708.1` startup bytes (2,198 bytes, SHA-256
  `01b4fe61b95e8e2b8563f8ae35d380e4eec09eb199e45e237fa4273d274bfa82`).
  The stable qualified evidence is a normal-intensity `⟩` at row 8 between
  faint rule cells, with a visible cursor at column 3; an old bare `›` and the
  same geometry without faint rules are both negative.
- Task 2A fresh spec and quality reviews returned `APPROVED`; the cap test
  passed 100 repetitions under concurrent load, focused race passed, and full
  wrapcmd passed. A later live rerun was blocked in approval orchestration, not
  a test process; the earlier live recapture was byte-identical to the fixture.
- Task 3 landed as `9038158` with preservation fixes `174b2cd` and `44ac8c0`.
  Codex and Muse profiles now point at pure bounded snapshot predicates while
  old trackers remain as the differential oracle. Fifteen rows finish as 12
  unchanged, three named Muse `true→false` safety corrections, and zero
  `false→true` expansions.
- Task 3 fresh spec and quality reviews returned `APPROVED` after restoring
  non-empty and cursor-adjacent Muse composers and making arbitrary snapshot
  validation overflow-safe. Focused/race/full wrapcmd tests pass; the known
  unrelated stdout batching test race remains outside this task.
- Task 4 landed as `09817d1`: Agy now has a pure coherent-box snapshot
  recognizer with 30 geometry/lifecycle cases, direct profile registration,
  and no retained evidence. The inherited untracked partial-parser prototype
  and its tests were removed after unique cases were ported.
- Task 4 fresh spec and quality reviews returned `APPROVED` with no findings;
  focused/race/full wrapcmd checks pass and the commit touches exactly the four
  planned tracked files.
- Task 5 landed as `86bf52e` with concurrency corrections `3306d73` and
  `7bedff2`. Proxy startup resolves one profile, positive gates own one terminal,
  output/resize feed that model, and plain Return delegates to the pure decision.
  The old keymap/overlay registries and Codex/Muse tracker implementations and
  tests are deleted; the frozen 15-row oracle remains.
- Task 5 fresh spec and quality reviews returned `APPROVED` after atomic overlay
  consume/re-arm, exclusive prepare/commit/abort PTY resize synchronization,
  and defer-safe detector panic recovery. Focused/scoped race/full wrapcmd and
  shadow sweeps pass; the unrelated stdout batching test race remains recorded.

## Revisions

### 2026-08-17T13:03:00-07:00 — Unify harness TTY Return handling

**Reason.** The first Agy-only implementation exposed a third copy of terminal
cursor, erase, resize, and chunk-framing behavior. Its weak prompt/border
signals could also survive stale repaints and incorrectly authorize the LF
rewrite. The operator approved a shared Return-specific design rather than
adding another independent tracker.

**Delta.** This revision supersedes the Agy-only tracker ownership and
workstream-isolation decisions above. Issue #139 now owns the shared
Return-routing substrate and its behavior-preserving adoption by the existing
Codex and Muse gates, plus the new Agy recognizer. It does not implement
Claude composer recognition; #138 remains the future opt-in consumer. It does
not broaden into a general TTY customization framework.

#### Revised architecture

1. **One harness TTY profile registry.** A `harnessTTYProfile` is the single
   source for one harness's Return keymap, overlay detector, and optional
   positive composer-recognizer factory. This replaces the parallel keymap and
   overlay registries plus agent-name branches in `emitPlainCR` (ARCH-DRY).
   Harnesses without a composer recognizer retain their current legacy Return
   behavior until their own issue opts them into positive gating.
2. **One shared terminal model.** Composer recognizers read a concurrency-safe
   current-screen view backed by the existing `github.com/charmbracelet/x/vt`
   emulator already used by `scrollbackcmd`; Pair does not grow another partial
   VT parser. A thin wrapper owns feed, resize, cursor-visibility tracking, and
   lifecycle. Terminal emulation stays deterministic and IO-free
   (ARCH-PURE).
3. **Harness-specific recognition only.** Codex recognizes its painted
   composer surface, Muse its prompt signature, and Agy one coherent current
   screen box: top border, anchored prompt row, bottom border, bounded height,
   and visible cursor inside. A lone `>`, a lone divider, distant chrome, or
   stale erased/reflowed content is never positive evidence
   (ARCH-PURPOSE).
4. **One pure Return decision.** The shared router resolves an input surface
   with strict precedence: active confirmation overlay -> bare CR; positively
   recognized composer -> the profile's `plainCR` multiline bytes; unknown,
   busy, hidden-cursor, or inactive surface -> bare CR for profiles requiring
   positive gating. Alt+Return continues to emit the profile's `altCR` submit
   bytes regardless of surface.
5. **Narrow integration shell.** `proxy.handleChunk` feeds the selected
   profile's tracker, `proxy.setWinsize` resizes it, and `emitPlainCR` asks the
   shared decision function. No other proxy behavior learns harness-specific
   composer rules.

#### Verification and conformance

- Unit-test the shared terminal wrapper against erase, scrolling, wrapping,
  resize/reflow, malformed/incomplete sequences, arbitrary chunk splits, and
  concurrent feed/state reads.
- Table-test the pure Return decision across overlay, composer, unknown, and
  legacy-profile states; preserve each harness's exact key bytes.
- Test each recognizer with literal PTY fixtures captured from that real
  harness, including negative local-evidence cases. Replay fixtures through
  the production `proxy.handleChunk` seam with a stateful session fake
  (ARCH-MOCK).
- Re-capture and compare fixtures whenever a supported harness version changes,
  and run the live conformance smoke during release validation.

#### Revised acceptance

- Adding a Return-customized harness requires one profile registration and one
  recognizer, not edits to the shared routing flow.
- Codex and Muse preserve their current Return behavior while deriving from the
  shared profile/router/terminal substrate.
- Agy rewrites plain Return only for its coherent live composer box; all weak,
  stale, or unknown evidence sends bare CR.
- Claude behavior remains unchanged and #138 can opt it into the same contract
  without changing the router.

#### Review corrections

- The terminal wrapper starts a goroutine that drains emulator-generated
  replies (DSR, device attributes, and similar queries), because an undrained
  `x/vt` input pipe can block `Write`. `Close` first closes the retained input
  pipe writer, joins the reply drainer, and only then closes the emulator so it
  cannot race `Emulator.Read`'s unsynchronized closed-state access. Proxy
  teardown always calls it. Tests must prove that query sequences and shutdown
  both complete without blocking.
- Feed, resize, reset, snapshot, and close share one wrapper lock. Recognizers
  receive one immutable `terminalSnapshot` containing dimensions, cursor,
  visibility, active-screen identity, and copied cells from one screen
  generation; they never make a series of independently locked emulator calls.
  Cursor visibility starts false and resets false on terminal reset or screen
  replacement until a show-cursor sequence is observed. Race verification runs
  `go test -race ./cmd/internal/wrapcmd` (ARCH-PURE).
- Agy recognition requires two distinct contiguous border runs of at least five
  `─` cells with overlapping horizontal spans, an anchored `>` at column 1-6
  on a row strictly between the borders, and a visible cursor strictly between
  the same borders and within their overlapping span. Border height is at most
  25 rows. Every cell and coordinate comes from the same snapshot. Negative
  tests cover a lone prompt, one border, borders without a prompt,
  non-overlapping borders, prompt/cursor outside the box, and locally weak
  evidence beside a complete but distant box (ARCH-PURPOSE).
- `harnessTTYProfile` owns the complete existing Return-related customization:
  `plainCR`, `altCR`, `altBS`, overlay detector, positive-gating policy and
  recognizer factory, plus the capability that makes Codex image capture set
  overlay state. A shadow-sweep test covers Claude, Codex, Agy, Muse, unknown
  harnesses, and `PAIR_WRAP_REMAP_RETURN=0`, and proves the old parallel
  registries and composer agent-name branches are gone (ARCH-DRY,
  ARCH-PURPOSE).
- #140 remains the owner of Muse's recognition semantics; #139 only migrates
  the already-landed Muse behavior without semantic change. #139 now records
  #140 as a blocking dependency and implementation waits until #140's artifact
  is reconciled and closed. Differential fixture tests compare Codex and Muse
  decisions before and after the shared-substrate migration.
- Checked-in conformance data lives under
  `cmd/internal/wrapcmd/testdata/tty/<agent>/<version>/` with `composer.raw`,
  `overlay.raw` when capturable, and `metadata.json` recording agent version,
  capture time, command, and SHA-256. `go test ./cmd/internal/wrapcmd -run
  TestHarnessTTYFixtureConformance -count=1` deterministically replays every
  fixture. `PAIR_LIVE_HARNESS=<agent> go test ./cmd/internal/wrapcmd -run
  TestHarnessTTYLiveConformance -count=1` checks the installed harness's initial
  composer through a real PTY; a version mismatch or recognition difference
  fails with the required recapture path. Release validation runs the live test
  for each installed supported harness (ARCH-MOCK).

#### Authoritative revised Done when

The earlier Agy-only `Done when` and `Plan` are superseded records; the rows
below are authoritative for the revised scope and the expanded estimate is
re-derived only after the revised durable plan passes plan-quality review.

- One profile registration completely describes each harness's Return-related
  keymap, overlay detection, gating policy, recognizer, and capture capability.
- One `x/vt`-backed wrapper owns terminal interpretation, atomic snapshots,
  reply draining, and shutdown; no agent-specific tracker parses ANSI/VT bytes.
- The shared pure router preserves overlay precedence, fail-safe unknown-state
  behavior, legacy Claude behavior, and unconditional Alt+Return submission.
- Codex and Muse decisions are behaviorally identical before and after
  migration; Agy requires the coherent current-screen geometry above.
- Stateful fixture replay, live initial-composer conformance, race tests, the
  full test suite, and updated atlas guidance pass.

#### Authoritative revised Plan

- [x] Reconcile and close blocking issue #140 without changing Muse's landed
      recognition contract.
- [x] Write the revised durable implementation plan, pass plan-quality review,
      and reconcile the expanded estimate through `sdlc change-code`.
- [x] Build the tested `x/vt` terminal wrapper with atomic snapshots, reply
      draining, reset/resize handling, and deterministic close.
- [x] Consolidate Return customization into `harnessTTYProfile` and a pure
      fail-safe Return decision, with registry shadow-sweep tests.
- [x] Migrate Codex and Muse recognizers with differential fixture coverage and
      no semantic change.
- [x] Add Agy coherent-box recognition via failing geometry and lifecycle tests.
- [ ] Replay stateful real-harness fixtures through `proxy.handleChunk`, add the
      executable live conformance test, and run race/full verification.
- [ ] Update atlas documentation for the profile, terminal, routing, fixture,
      and release-conformance flows.

#### Plan clarification — behavior preservation

For Codex and Muse, "behaviorally identical" means the explicit active-
composer, hidden-cursor, overlay, and unknown-state decision matrix remains
unchanged. Current-screen emulation may intentionally turn a previously stale
`true` into fail-safe `false` after overwrite, erase, scroll, resize/reflow,
alternate-screen replacement, or reset. Every such transition must be
allowlisted by a differential test and documented; no `false` to `true`
transition is permitted. This clarification serves the positive-evidence
purpose rather than preserving known stale state (ARCH-PURPOSE).

### 2026-08-17T14:17:00-07:00 — Absorb #140 after boundary rework

The exact lost-window review of Muse commit `ab736d1^..ab736d1` returned
`REWORK`. It confirmed two purpose-breaking false positives (stale evidence
after screen mutation and an unqualified prompt glyph) plus the same shared
terminal, stateful-fixture, and documentation gaps this unified issue exists to
solve. Repairing all findings inside #140 and then replacing that implementation
here would duplicate the terminal migration and its conformance work
(ARCH-DRY).

**Delta.** #139 now absorbs #140's remaining acceptance criteria and removes
the blocking dependency. #140 is recorded separately as superseded by this
issue; its landed behavior remains characterization input, not an independently
closable contract. The revised Codex/Muse differential rule already permits
two named old-`true` to new-`false` Muse safety corrections: stale evidence
after screen mutation and an unrelated `›` lacking the capture-proven composer
signature. It forbids every `false` to `true` change. This explicitly supersedes
the earlier "behaviorally identical" / "no semantic change" wording for Muse;
Codex remains preservation-only apart from named stale-state corrections.
README coverage is added to Task 7 alongside atlas coverage. Implementation may
enter `sdlc change-code --issue 139` after this revision passes fresh plan
review (ARCH-PURPOSE, ARCH-MOCK).

### 2026-08-17T15:58:00-07:00 — Plan-quality execution corrections

The next `change-code` round disposed of the stale-screen and external-protocol
findings but retained PQ-3 and raised PQ-6. The durable plan now names one
adversarial strategy and mechanical guard per risky function instead of
enumerating terminal cases. For reply shutdown, construction checks that the
`io.Writer` returned by pinned `x/vt.Emulator.InputPipe()` also implements
`io.Closer`; construction fails before starting the drainer if that capability
is absent. The model retains that checked closer, closes it before joining the
drainer, then closes the emulator. This makes the pinned-library assumption
explicit and testable rather than calling `Close` through an interface that does
not declare it (ARCH-PURE).

### 2026-08-17T17:02:00-07:00 — Match the pinned parser boundary

Task 1 quality review proved that a byte scanner built only on ESC framing
cannot match x/vt: the pinned x/ansi parser recognizes ground-state C1 CSI while
treating the same byte as data inside UTF-8 and OSC/DCS states. The visibility
observer therefore uses its own bounded x/ansi parser with CSI/ESC handlers,
sharing x/vt's state semantics rather than maintaining another partial DFA
(ARCH-DRY, ARCH-PURPOSE).

Fuzzing and a deterministic `👩‍💻` split exposed a pinned x/vt behavior:
graphemes can render differently across separate `Emulator.Write` calls because
x/vt flushes at write boundaries, even when the combined stream is valid UTF-8.
Pair will not wrap the emulator in a second grapheme/UTF-8 representation layer.
All byte streams must remain bounded, nonblocking, panic-free, bounds-safe, and
fail-closed for visibility; the observer remains chunk-equivalent. Production
fixture tests, rather than whole-grid equality, require each harness's
recognizer and Return decision to be invariant at every raw-stream split. x/vt
remains the single screen-semantics owner (ARCH-PURE, ARCH-PURPOSE).

### 2026-08-17T17:38:00-07:00 — Bound terminal allocations

Task 1 quality review found that rejecting only non-positive dimensions leaves
positive sizes free to drive unbounded x/vt screen and snapshot allocations.
Construction and resize must pass one overflow-safe pure validator before x/vt
sees the dimensions: each axis is at most 4096 cells and total screen area is at
most 262,144 cells. This is comfortably above real PTY sizes while placing a
deterministic ceiling on both emulator buffers and copied snapshots. Rejection
leaves the existing model unchanged (ARCH-PURE, ARCH-PURPOSE).

### 2026-08-17T21:04:00-07:00 — Make the gate-policy zero value fail safe

Task 2 quality review found that assigning `composerGateLegacy` to enum zero
made an absent or corrupt profile an authorization path: the pure decision
could report `Fired` and emit an empty keymap, swallowing Return. Zero is now an
explicit unknown policy. The decision switches exhaustively: only named legacy
or positive policies may reach their remap behavior; zero and every invalid
value emit bare CR with `adapt.Bypass` and composer-unknown telemetry
(ARCH-PURE, ARCH-PURPOSE).

### 2026-08-17T21:42:00-07:00 — Coordinate capture teardown

Task 2A quality review found three ways its reusable PTY helper could return
before proving cleanup: a primary capture error masked teardown errors, an
unexpected kill error skipped the final reap wait, and cancellation did not
join the reader goroutine. Capture teardown is one coordinated operation: close
the cancellation channel and PTY, signal, always continue through a bounded
wait/kill/reap sequence even when an operation fails, join the sole reader, and
combine primary plus cleanup failures with `errors.Join`. Injected operation
failures must prove every later cleanup step is still attempted
(ARCH-PURE, ARCH-MOCK, ARCH-PURPOSE).

### 2026-08-18T06:15:00-07:00 — Make overlay consume and resize fail safe

Task 5 quality review found two logical authorization races. Overlay detection
and consumption must share `overlayMu`: the production detector updates its
text carryover and arms the overlay inside the same critical section, while
Return atomically swaps the active flag and clears only the consumed tail under
that lock. A newly detected overlay can therefore never be erased by an older
Enter. Resize uses a latched synchronization transaction: validate before
mutation; prepare the model while atomically masking snapshot visibility;
resize the PTY; then commit synchronization while clearing prior visibility so
only later explicit cursor evidence can authorize Return. PTY failure leaves
the latch closed across all later output until a complete successful resize
transaction. Tests exercise deterministic overlay re-arm and the real
`setWinsize` path. Task 7 also updates the stale `doctor/README.md` registry
instructions (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE).

Overlapping resizes are serialized by an exclusive prepare token; every path
must commit or abort it, and abort leaves authorization latched closed. Overlay
detection runs inside a defer-unlocked helper so the existing `handleChunk`
panic recovery cannot strand `overlayMu`. Tests overlap two transactions and
inject a panicking detector before a subsequent usable Return.

### 2026-08-19 — Tasks 6 and 6A: conformance fixtures and Codex recognizer restoration

- Task 6 Step 4 against the three installed harnesses produced two findings that
  changed the task; both are recorded in the plan's `## Revisions`.
- **Codex recognition was dead on the installed CLI.** `codexComposerActive`
  required a `57,57,57` background band inherited from #137's retired
  `codexComposerBG`. Codex `0.147.0` paints no background at all, so the
  positive gate never fired and plain Enter could never insert a newline in the
  Codex composer. Pre-existing #137 surface; Task 6 was the first check able to
  observe it. Task 6A restores recognition from literal evidence.
- Codex `0.147.0` also blocks startup on an update interstitial. Its capture
  argv gained the documented `-c check_for_update_on_startup=false`, the
  counterpart of Muse's `MUSE_NO_AUTO_UPDATE=1`.
- **Live conformance no longer asserts byte identity.** Captures embed
  per-account and per-moment content (Agy paints the signed-in address, Codex a
  rate-limit banner and rotating tip, Muse the model name) and Agy self-updated
  `1.1.13 -> 1.1.15` between two runs minutes apart. Muse is deterministic run
  to run — three consecutive captures shared one digest — yet no longer matched
  its two-day-old fixture. The live path now asserts recognition plus the
  production `emitPlainCR` result; byte exactness moved to the frozen-fixture
  replay, which is the stronger check.
- Task 6A evidence, all captured live through the bounded PTY seam: settled
  composer, slash-command menu, composer after one LF, three-line composer, and
  the update interstitial. Sending LF into the slash menu inserted a newline and
  dismissed the popup rather than firing `/model`, so Codex distinguishes LF
  from CR itself; the remaining risk is confirm dialogs, which the picker-marker
  overlay layer already arms on.
- The recognizer keys on Codex's actual structure: a **bold** U+203A at column 0
  with the cursor reaching it through painted continuation rows only. Codex
  reuses the same glyph unemphasized as a menu marker, and parks the cursor on
  the status line mid-paint — both rejected. The bold rule alone rejects the
  update interstitial even with the cursor forced visible on a menu row.
- **Frozen Codex rows removed**, all premised on the `57,57,57` band that no
  installed Codex paints: `generated composer`, `hidden cursor`,
  `erased composer`, `composer away from cursor`, and
  `one local painted row plus distant complete evidence`. Their intent is
  preserved by `generated captured signature`, `hidden cursor`,
  `erased composer`, `cursor below composer past a blank row`, and
  `prompt beyond the composer height bound`, plus new literal-fixture,
  multi-line, empty-continuation-row, and unemphasized-marker cases.
- Retired `museComposerPrefixEnd`, the per-harness truncation scan; every
  fixture is now trimmed by the shared recognizing-prefix helper, and
  `TestMuseFixtureEvidence` asserts that shared rule and reuses
  `readHarnessTTYFixture` instead of decoding metadata a second time
  (ARCH-DRY). The Codex composer paint was duplicated across four test files
  and is now one `codexLiveComposerPaint` helper.
- Fixtures checked in with digests: `agy/1.1.15`, `codex/0.147.0`
  (composer + overlay), `muse/0.1.0-R708.1`. The fixture test replays each
  through `proxy.handleChunk` at every split from 0 to len and requires the
  recognizer, overlay arming, and `emitPlainCR` bytes to match the unsplit
  baseline.
- Verification: full `wrapcmd` package, `go test ./...`, and `make test` all
  pass. `go test -race ./cmd/internal/wrapcmd` reports exactly one failure, the
  known unrelated `TestMasterPumpFlushesStdoutOnTick` `bytes.Buffer` race; no
  file from this work appears in its trace. All three live harness checks pass:
  Agy, Codex, and Muse composers remap to `\n`, and the Codex overlay passes
  bare `\r` with the picker layer armed.
