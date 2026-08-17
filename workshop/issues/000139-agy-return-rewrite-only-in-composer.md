---
id: 000139
status: working
deps: ["#140"]
github_issue:
created: 2026-08-16
updated: 2026-08-17
estimate_hours:
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
- [ ] Complete the authoritative revised plan in `## Revisions`.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only.

```estimate-superseded
model: estimate-logic-v3.1

item: issue-spec design=0.08 impl=0.02
item: smaller-go-module design=0.15 impl=0.25
total: 0.57
```

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
  `x/vt` input pipe can block `Write`. `Close` closes the emulator and joins the
  drainer, and proxy teardown always calls it. Tests must prove that query
  sequences and shutdown both complete without blocking.
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

- [ ] Reconcile and close blocking issue #140 without changing Muse's landed
      recognition contract.
- [ ] Write the revised durable implementation plan, pass plan-quality review,
      and reconcile the expanded estimate through `sdlc change-code`.
- [ ] Build the tested `x/vt` terminal wrapper with atomic snapshots, reply
      draining, reset/resize handling, and deterministic close.
- [ ] Consolidate Return customization into `harnessTTYProfile` and a pure
      fail-safe Return decision, with registry shadow-sweep tests.
- [ ] Migrate Codex and Muse recognizers with differential fixture coverage and
      no semantic change.
- [ ] Add Agy coherent-box recognition via failing geometry and lifecycle tests.
- [ ] Replay stateful real-harness fixtures through `proxy.handleChunk`, add the
      executable live conformance test, and run race/full verification.
- [ ] Update atlas documentation for the profile, terminal, routing, fixture,
      and release-conformance flows.
