# Boundary Review — pair#139 (whole-issue close)

| field | value |
|-------|-------|
| issue | 139 — Agy Return rewrite only in composer |
| repo | pair |
| issue file | workshop/issues/000139-agy-return-rewrite-only-in-composer.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6130ca0f9d14cf74e6e652aa7d248f363eb8c966..HEAD |
| command | sdlc close --issue 139 |
| reviewer | claude |
| timestamp | 2026-08-19T12:32:55-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The consolidation itself is excellent: three hand-rolled terminal parsers collapse into one `x/vt`-backed model with immutable snapshots, two parallel agent-name registries collapse into one `harnessTTYProfile`, the gate enum's zero value fails closed, and the every-split fixture replay through the production proxy is a genuinely strong conformance test. What blocks SHIP is that the three composer recognizers model only the *captured startup screen* and reject ordinary multi-line composing states — the exact states "Enter inserts a newline" exists to produce. I reproduced two of these through the production path: Codex returns `false` whenever a blank line sits between the prompt row and the cursor (so `› a` / blank / `b` submits instead of inserting a newline), and Muse returns `false` for any composer that has grown past one line — which is an old-`true`→new-`false` transition against the `museComposerTracker` this window deletes, and is not among the three named safety corrections the plan's differential contract allows. A third, the Agy gate, authorizes LF on a picker-shaped screen carrying a plain `>` marker between two rules, so for Agy the positive gate currently adds no marker-independent defense — the issue's stated purpose.

## 1. Strengths

- **`terminalModel` lifecycle is carefully built.** Reply-pipe `io.Closer` capability is asserted at construction (`terminal_model.go:80-85`), close is idempotent with a shared `closeDone` result, post-close `Feed`/`Resize` return `io.ErrClosedPipe`, and `Snapshot` serves a deep-cloned final state. `go test -race` over the new paths is clean (51s, no report).
- **Fail-closed defaults are consistent and tested.** `composerGateUnknown = iota` (`harness_tty.go:8`) plus the exhaustive switch in `decidePlainReturn` means an absent/corrupt profile emits bare CR; `TestDecidePlainReturn` pins both the all-zero profile and an out-of-range policy.
- **The every-split fixture replay is the right test.** `replayHarnessTTYFixture` (`harness_tty_fixture_test.go:230-249`) establishes an unsplit baseline and requires recognizer, overlay arming, and `emitPlainCR` bytes to be identical at all `len(raw)+1` split points — PTY chunk boundaries provably cannot change what Return does.
- **Overlay consume/re-arm is now one owner.** `detectOverlayOpen` and `emitPlainCR` share `overlayMu` with `Swap`, and `TestEmitPlainCR_ConcurrentOverlayRearmRetainsNewStateAndTail` pins the interleaving; the defer-unlocked detector helper plus the injected-panic regression is a good catch to have made.
- **Live conformance correctly asserts behavior, not bytes.** The reasoning recorded in the plan revision (per-account/per-moment content, harness self-update) is right, and moving byte-exactness to the frozen fixtures is the stronger split.

Verified green: `go test ./cmd/internal/wrapcmd`, `go test ./...`, and `make test` all pass (outside the sandbox — the in-sandbox PTY failures are `operation not permitted` on `pty.StartWithSize`, not code).

## 2. Critical findings

**C1 — `codexComposerActive` rejects any composer with a blank line above the cursor; Return then submits the message.**
`cmd/internal/wrapcmd/composer_recognizers.go:44`

```go
if promptY != snapshot.Cursor.Y && !codexComposerContinuationRow(snapshot, promptY) {
    return false
}
```
A fully blank row strictly between the prompt row and the cursor row aborts the walk. The predicate cannot distinguish "blank spacer above the status line" (the negative it was written for) from "blank line inside the composer".

*Failure scenario (reproduced).* Feed `\x1b[12;1H\x1b[1m›\x1b[22m hello` then `\x1b[14;3Hworld\x1b[?25h\x1b[14;8H` — i.e. the composer holds `› hello` / *(blank)* / `world` with the cursor on `world` — and `codexComposerActive` returns `false`, so `decidePlainReturn` emits bare `\r` and Codex submits the half-written prompt. Same result for two consecutive Enters (`prompt` + cursor at `14;3H` → `false`); the existing `"cursor on empty continuation row"` case only covers *one* blank row because that row is the cursor's own.

*Fix sketch.* Keep the bound, but stop treating "blank" as "gap": walk up while rows are blank **or** continuation rows, and require the block to terminate at a bold `›` within `codexComposerMaxRows`. To retain the status-line negative, discriminate on what lies *below* the cursor (a painted status row at col ≥ 2 with a blank row above it) or on the cursor column matching `codexComposerTextColumn`, rather than on emptiness above. Add both reproductions above to `TestCodexComposerActiveSnapshotDifferential`.

**C2 — `museComposerActive` only accepts a one-line composer; the second Enter submits. This is an unnamed old-`true`→new-`false` regression.**
`cmd/internal/wrapcmd/composer_recognizers.go:84-86`

```go
if prompt != nil && prompt.Content == "⟩" && ... &&
    faintRuleAt(snapshot, 0, promptY-1) && faintRuleAt(snapshot, 0, promptY+1) {
```
Requiring a faint rule *immediately* below the prompt row means the predicate can only ever match a composer exactly one line tall — as soon as the box grows, the bottom rule moves down and `faintRuleAt(0, promptY+1)` sees the continuation text's blank column 0.

*Failure scenario (reproduced).* Rules at rows 7 and 10, `⟩ hello` at row 8, `  world` at row 9, cursor at `9;8` → `museComposerActive` = `false` → bare CR → Muse submits mid-message. The deleted `museComposerState.active()` returned `true` here (`|cursorRow-promptRow| == 1`), so this is a behavior regression introduced by this window, and it is *not* one of the three named `true→false` safety corrections the plan's differential contract permits ("zero `false→true` expansions" was checked; unnamed `true→false` was not).

*Fix sketch.* Anchor on the *enclosing* rules rather than adjacent ones: find the nearest faint `─` rule row above the prompt and the nearest below the cursor, require the non-faint `⟩` at column 0 to be the first row inside that pair, and bound the height (as `agyComposerActive` already does). Add 2-line and 3-line composer rows to `TestMuseComposerActiveSnapshotDifferential`, and re-run the frozen oracle so the regression is either eliminated or explicitly named.

## 3. Important findings

**I1 — `agyComposerActive` accepts a plain `>` selection marker, so the Agy positive gate provides no defense independent of `agyPickerMarkers` (ARCH-PURPOSE).**
`cmd/internal/wrapcmd/composer_recognizers.go:130` — `cell.Content == ">"` checks content only, no style.

*Failure scenario (reproduced through the production path).* Feed a dim rule at row 10, `> 1. Yes` at row 12, `  2. No` at row 13, a dim rule at row 14, cursor visible at `12;3` — with no registered marker text in the chunk — and `f.proxy.emitPlainCR(nil)` returns `"\n"`. `atlas/how-to-bring-up-a-new-harness-cli.md` itself documents Agy's picker as `Do you want to proceed?` / `> 1. Yes` / `2. No`, i.e. a `>` at column 0. The captured fixture shows Agy paints its real prompt as `\x1b[94m>` and its rules as `\x1b[90m─` (`testdata/tty/agy/1.1.15/composer.raw`), so the discriminating style evidence exists in-hand and is unused — the same class of discriminator the Codex recognizer correctly applies via `uv.AttrBold`. Since the issue's Problem statement is that "permission pickers and future UI variants keep plain Return as confirm", a gate that a picker satisfies does not deliver it.

*Fix sketch.* Require the prompt cell's foreground to be the captured bright-blue (or at minimum non-default) and/or require the enclosing rules to carry the captured dim style, then pin it with a captured `agy/1.1.15/overlay.raw` (see I2).

**I2 — Two of three positive-gated harnesses have no captured negative state (ARCH-MOCK).**
Only `testdata/tty/codex/0.147.0/overlay.raw` exists. `TestHarnessTTYLiveOverlayConformance` (`harness_tty_live_test.go:610-620`) registers a scenario for `codex` only and `t.Skipf`s for agy/muse. `ttyFixtureExpectation` already encodes `"overlay.raw": false`, so the harness for this exists — it is simply unpopulated for the harness this issue is *named after*. Capture Agy's permission picker and one Muse selection menu through the same bounded seam and add them; that is what would have caught I1 mechanically.

**I3 — `TestHarnessTTYFixtureConformance` requires `composer.raw` per positive-gated harness but never requires the overlay negative.**
`cmd/internal/wrapcmd/harness_tty_fixture_test.go:47-53` builds `required` from `composerGate == composerGatePositive` and only checks `composer.raw` presence (line 90). A harness can therefore ship a positive gate with zero evidence that it *declines* anything. Consider extending `required` to demand an `overlay.raw` once one exists per harness, or at minimum logging the gap loudly rather than passing silently.

**I4 — README's new Return row is inaccurate for `claude`.**
`README.md:109` says "Pair only rewrites Return while it can positively see a live composer on screen … Applies to `claude`, `codex`, `muse`, and `agy`." Claude is registered `composerGateLegacy` (`harness_tty.go:31`) and has no composer recognizer at all — it rewrites unconditionally, guarded only by the permission OSC. Split the row, or scope the positive-gate sentence to codex/muse/agy.

## 4. Minor findings

- `composer_recognizers.go:157-173` — `rowHasBackground` and `colorMatches` are dead (referenced only by each other) leftovers from the retired Codex BG gate; they are the sole reason for the `image/color` import (ARCH-DRY).
- `wrap.go:1483` — `resizeTerminal` has no production caller; only `harnessSessionFake.resize` uses it. Test-only API in production code.
- `harness_tty.go:84,93` — `returnDecision.clearOverlay` is never honored: `emitPlainCR` clears the overlay itself *before* calling `decidePlainReturn`. The pure decision advertises an output no consumer reads; only `harness_tty_test.go:122` asserts it.
- `doctor/README.md:45` — still points `codexSyncOutputMarkers` at `cmd/pair-wrap/main.go`, which no longer exists. Adjacent rows were corrected in this same diff.
- `wrap.go:1792,1810` — `translateChunk` dereferences `p.ttyProfile` with no nil guard while `emitPlainCR:1732` has one. Safe today only because `hasReturnRemap()` gates the call site; the asymmetry invites a future nil panic.
- `terminal_model.go:108-111` — when `emulator.Write` returns an error, `observer.Feed(data)` is skipped, so the observer and emulator can diverge on cursor visibility (potentially leaving a stale `visible = true`). Feeding the observer unconditionally would keep it fail-closed.
- `harness_tty.go:66-78` — `profileForHarness` copies the three keymap slices but not `overlay`/`recognize`/`composerGate`; the doc comment ("hands out an immutable copy" in the atlas) slightly overstates it. Harmless as written, but worth wording precisely.

## 5. Test coverage notes

- Coverage of the *captured* states is strong and the every-split replay is better than the bar. Coverage of *derived* states — composer after 1, 2, N newlines, blank lines inside the composer, a grown box — is absent for all three harnesses, and that is exactly where C1/C2 live. Every recognizer test either replays the startup fixture or hand-builds a snapshot in that same shape.
- `TestHarnessTTYIntegration_StatefulReturnRouting` does prove overlay-beats-composer precedence, but the overlay arms from the *marker* layer, so it cannot detect I1.
- The live tests (`PAIR_LIVE_HARNESS`) are opt-in and not wired to any schedule; `atlas/architecture.md` describes them as "scheduled/live conformance checks", but nothing schedules them. Either add a cadence (cron/routine) or soften the atlas wording.
- Adding one regression per finding — Codex blank-line-above-cursor, Muse 2-line composer, Agy plain-`>` picker — would close all three permanently.

## 6. Architectural notes

- **ARCH-DRY — pass (one nit).** Three trackers → one `terminalModel`; two registries → one `harnessTTYProfiles`; four duplicated Codex paint literals → `codexLiveComposerPaint`. The only residue is the dead `rowHasBackground`/`colorMatches` pair.
- **ARCH-PURE — pass.** Recognizers are pure `terminalSnapshot → bool`; `decidePlainReturn` is pure; `proxy` is a thin feed/resize/decide shell. Tests run the recognizers with no IO. The one seam smell is `overlayConsumeHook` on the production struct (documented, acceptable) and the unconsumed `clearOverlay`.
- **ARCH-PURPOSE — flag.** The shadow sweep is genuinely complete for the *registry* migration (no `sendKeymapByAgent` / `overlayDetectorByAgent` / `sendKM` / per-agent tracker survives in Go). But the issue's purpose — a positive gate that keeps Return as confirm in pickers *without* depending on marker strings — is not delivered for Agy (I1), and the complementary half of the contract (Return = newline while composing) fails for Codex (C1) and Muse (C2). Cited in C1, C2, I1.
- **ARCH-MOCK — partial.** `harnessSessionFake` + frozen fixtures + a bounded PTY capture seam + opt-in live conformance is a strong shape, and production and test flow share `proxy.handleChunk`. Incomplete on two axes: no negative-state fixture for agy/muse (I2/I3) and no scheduled cadence for the live check.

## 7. Plan revision recommendations

Add a `## Revisions` entry to `workshop/plans/000139-agy-return-rewrite-only-in-composer-plan.md` covering:

1. **Task 3 differential contract breach.** The plan and issue Log both claim the Muse migration produced "three named `true→false` safety corrections, and zero `false→true`". A two-line Muse composer is a fourth, unnamed `true→false` — a real regression, not a safety correction. Record it, then either fix `museComposerActive` (preferred) or name and justify it explicitly.
2. **Recognizer scope statement.** Task 6A defined the Codex recognizer from a single startup capture. State the derived-state obligation explicitly: every recognizer must be validated against composer-after-newline and composer-with-blank-line states, not only the captured startup screen, and add those rows to each `*SnapshotDifferential` table.
3. **Task 4 Agy evidence gap.** The Agy recognizer keys on unstyled `>` while the fixture shows `\x1b[94m>`; record that the style evidence exists and that an `agy/<version>/overlay.raw` is required before the Agy positive gate can be claimed as a marker-independent defense (Task 6 currently treats `overlay.raw` as optional).
4. **Live-conformance cadence.** Either name the schedule that runs `PAIR_LIVE_HARNESS` (ARCH-MOCK's "live conformance checks so drift is detected") or amend `atlas/architecture.md` to describe it as manual/opt-in.

---

## Re-review — 2026-08-19T13:05:28-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 139 — Agy Return rewrite only in composer |
| repo | pair |
| issue file | workshop/issues/000139-agy-return-rewrite-only-in-composer.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6130ca0f9d14cf74e6e652aa7d248f363eb8c966..HEAD |
| command | sdlc close --issue 139 |
| reviewer | claude |
| timestamp | 2026-08-19T13:05:28-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The consolidation is genuinely good and the three findings that blocked the previous boundary review are fixed — I reproduced each fix through the production `emitPlainCR` path: a Codex composer with a blank line inside now remaps (`"\n"`), a two- and three-line Muse composer now remaps, and an unstyled `>` between rules no longer qualifies for Agy. The shadow sweep is clean (no `sendKeymapByAgent` / `overlayDetectorByAgent` / per-agent tracker / `agentBasename == "codex"` branch survives anywhere in the tree), and `go test ./...`, `make test`, and the package suite all pass outside the sandbox (in-sandbox PTY failures are `operation not permitted` on `pty.StartWithSize`, not code). What I'd fix before crossing: the Agy negative fixture doesn't actually exercise the discriminator the Agy fix introduced — it declines on hidden cursor and cursor position — so the issue's central claim (a marker-independent picker defense) still rests on one hand-authored snapshot whose premise is untested against the real CLI; the `EVIDENCE GAP` line that is supposed to make that loud is invisible in a default `go test` run; and `codexCursorOnTrailingStatusRow` still declines a real composer whenever nothing is painted below the cursor, which Codex's full-screen-erase repaint makes momentarily reachable.

## 1. Strengths

- **The every-split fixture replay is the right test, and it runs the production seam.** `harness_tty_fixture_test.go:247` establishes an unsplit baseline and requires the recognizer, overlay arming, `emitPlainCR` bytes, *and* the decision reason to be identical at all `len(raw)+1` split points. That is a stronger guarantee than the whole-grid chunk equality the plan correctly abandoned.
- **Fail-closed defaults are consistent and pinned.** `composerGateUnknown = iota` plus the exhaustive switch in `decidePlainReturn` means an absent or corrupt profile emits bare CR; `TestDecidePlainReturn` covers both the all-zero profile and an out-of-range policy.
- **`terminalModel` lifecycle is careful.** The `io.Closer` capability assertion at construction (`terminal_model.go:80`), idempotent `Close` with a shared `closeDone` result, post-close `io.ErrClosedPipe` on `Feed`/`Resize`, deep-cloned final snapshot, and the exclusive prepare/commit/abort resize token all hold up. The only race in `go test -race ./cmd/internal/wrapcmd` is the pre-existing `TestMasterPumpFlushesStdoutOnTick` `bytes.Buffer` race in `stdout_batch_test.go`, which this window does not touch.
- **Overlay consume/re-arm is one owner, and the panic path is defused.** `detectOverlayOpen` (`wrap.go:1618`) holds `overlayMu` across the detector under `defer`, and `armCapture` releases `overlayMu` before taking `captureMu`, so there is no lock-order inversion with `handleChunk`.
- **The Codex recognizer's discriminator is backed by real bytes.** The captured update interstitial paints the same `›` unemphasized (`\x1b[22m`) while the composer paints it bold — so `uv.AttrBold` is load-bearing evidence, not a guess. That is the standard the Agy rule doesn't yet meet.

## 2. Critical findings

None.

## 3. Important findings

**I1 — The Agy `overlay.raw` fixture does not exercise the prompt-color rule, so the Agy positive gate has no captured evidence that it rejects a picker.**
`cmd/internal/wrapcmd/testdata/tty/agy/1.1.15/overlay.raw`, rule at `composer_recognizers.go:182`

I fed that fixture through `terminalModel` and dumped column 0: row 2 `─` fg 8, row 3 `>` **fg 12 (bright blue)**, row 4 `─` fg 8. It is the genuine composer box, not a picker. The gate declines it because the cursor is hidden (`\x1b[?1049h` then `\x1b[?25l`); forcing `CursorVisible = true` it still declines, because the cursor sits at `(15,5)`, below the bottom rule. Neither reason is the prompt-color discriminator.

*Failure scenario (reproduced through `emitPlainCR`).* A full-width dim-ruled box containing `\x1b[94m>` `1. Continue` / `2. Cancel`, cursor visible at the prompt row, no registered marker text in the chunk → `pickerActive=false` and plain Return emits `"\n"`. So if Agy paints its permission-picker selection marker in the same bright blue as its composer prompt, the gate accepts it and Agy's defense collapses back to `agyPickerMarkers` — the state the issue exists to fix. The only proof otherwise is the hand-authored `"unstyled picker marker between rules"` row at `composer_recognizers_test.go:62`, whose premise (Agy's picker `>` is unstyled) is untested against the real CLI.

*Fix sketch.* Capture a real Agy permission picker through the existing bounded seam as `agy/<version>/picker.raw` with expectation `false` (needs a live tool call, which the issue Log already flags as outstanding). If the marker turns out to be colored too, add a second discriminator. At minimum, correct the issue Log / `atlas/architecture.md` wording — the fixture exercises cursor position, not the marker rule, so "exercises the positive gate alone" overstates it. The same applies to Muse, which has no negative fixture at all: I verified that `faint ─` / non-faint `⟩ Select an option` / `faint ─` with a visible cursor is recognized as an active composer, so Muse's selection menus are defended only by `musePickerMarkers`.

**I2 — The `EVIDENCE GAP` line is silent in a default test run, so the gap it names cannot be noticed.**
`cmd/internal/wrapcmd/harness_tty_fixture_test.go:130`

It is a `t.Logf` on a passing test. `go test` suppresses those without `-v`. I confirmed: `go test ./cmd/internal/wrapcmd -run TestHarnessTTYFixtureConformance` prints zero occurrences of `EVIDENCE GAP`; only `-v` shows it. `atlas/architecture.md` claims the harness "is named in a loud `EVIDENCE GAP` line rather than passing silently", and the code comment says "this reports the gap loudly" — but in `make test` and `go test ./...` it is exactly a silent pass.

*Fix sketch.* Write it to `os.Stderr` (which `go test` does surface), or express it as a real subtest that fails/skips visibly, or gate the whole thing behind a `required`-style assertion once a negative exists per harness. Note that `harness_tty_fixture_test.go:65-66` builds `required` and `negatives` symmetrically but only `required` (i.e. `composer.raw`) is ever enforced — nothing demands an `overlay.raw`.

**I3 — `codexCursorOnTrailingStatusRow` declines a real composer whenever nothing is painted below the cursor, and Codex's full-screen-erase repaint makes that momentarily reachable.**
`cmd/internal/wrapcmd/composer_recognizers.go:58`

The status row is excluded by "painted row, blank row above, nothing painted below anywhere". A composer continuation row with a blank line above it satisfies the same three conditions whenever the composer is the last painted block.

*Failure scenario (reproduced through `emitPlainCR`).* Codex begins each frame with `\x1b[1;1H\x1b[J` (visible in `testdata/tty/codex/0.147.0/composer.raw`), erasing the whole screen before repainting top-down, and the status row is painted *after* the composer. Feeding `\x1b[?25h\x1b[1;1H\x1b[J\x1b[12;1H\x1b[1m›\x1b[22m alpha\x1b[14;3Hbeta\x1b[14;7H` — a frame split by a PTY read between the composer and the status line — gives `emitPlainCR` → `"\r"`, i.e. Codex submits the half-written message. Completing the frame with the status row flips it back to `"\n"`. `emitPlainCR` snapshots from the stdin goroutine while `handleChunk` feeds from the master pump, so a mid-frame snapshot is a real interleaving, and x/vt does not make `\x1b[?2026h` frames atomic.

*Fix sketch.* This is arguably the declared fail-closed direction, but the failure mode here is the expensive one (send, not miss-a-newline). At minimum pin it: add the mid-frame stream above to `TestCodexComposerActiveSnapshotDifferential` with the intended answer and a comment saying it is a deliberate ambiguity resolution, so it is a decision rather than an accident. A stronger rule would key the status row on something positive (e.g. it is separated from the *bottom* of a bold-`›`-anchored block) rather than on "nothing painted below".

**I4 — `TestHarnessTTYFixtureConformance` is 10.9s of a 13.4s package and grows superlinearly with fixture bytes.**
`cmd/internal/wrapcmd/harness_tty_fixture_test.go:230-249`

12.7 KB of fixtures costs 10.9s, because each of the `len(raw)+1` splits builds a fresh proxy, re-feeds the entire stream, and takes two full-grid snapshots (120×38 = 4560 cell clones each). `atlas/how-to-bring-up-a-new-harness-cli.md` now instructs every new harness to add fixtures, and #138 will add Claude — so the next bring-up multiplies this. Consider reusing one terminal across splits, or bounding the split set (all splits for the first ~1 KB plus a strided sample beyond) and `t.Logf`-ing what was skipped per the plan's own "no silent caps" rule.

## 4. Minor findings

- `composer_recognizers.go:75` and `:85` — `codexComposerRowPaintsLeftEdge` and `snapshotRowPainted` are the same loop with different `x` bounds; one `rowPainted(snapshot, y, x0, x1)` helper (ARCH-DRY).
- `terminal_model.go:226` — `o.csiBytes == 5` is redundant with the `len(params)==1 && param==25 && !hasMore` checks except that it rejects zero-padded canonical forms (`\x1b[?025h`), which would then fail closed permanently. Drop it or document the intent.
- `terminal_model.go:110-118` — on `emulator.Write` error the observer is fed (good) but `m.altScreen` is skipped, so `AltScreen` can go stale on exactly the path the comment says must not diverge.
- `wrap.go:1466-1469` — `configureHarnessTTY` clears `p.ttyProfile` on the not-ok path but never `p.terminal`; a second call after a successful one leaves a live terminal paired with a nil profile.
- `harness_tty_live_test.go` — `harnessTTYRecaptureDestination` and `writeLiteralCapture`'s temp prefix always name `composer.raw`, even when `TestHarnessTTYLiveOverlayConformance` is writing an overlay capture.
- `doctor/README.md:40` — the `overlay-detect/near-miss` row lists `codexPickerMarkers` / `agyPickerMarkers` but not `musePickerMarkers`, which exists.
- `harness_tty.go:95` — a `composerGateLegacy` profile with an empty `plainCR` would report `adapt.Fired` while emitting zero bytes (swallowing Enter). Unreachable with the current registry, but the enum got zero-value hardening and the keymap didn't.
- `wrap.go` — three test-only seams now live on the production `proxy` struct (`getWinsize`, `setPTYWinsize`, `overlayConsumeHook`). Each is documented; worth watching that the list stops growing.

## 5. Test coverage notes

- Derived-state coverage is the thing that was missing last round and is now present for Codex (blank line inside, two consecutive newlines, empty continuation row) and Muse (two- and three-line composers). I independently confirmed a realistic full-width two-line Agy composer is accepted, so Agy's growth case works even though its table row uses a hand-built snapshot.
- No fixture captures a *composing* state for any harness — all five are startup/overlay screens. The issue Log says multi-line Codex states were observed live during Task 6A but only the startup screen was checked in, so the every-split replay protects the startup path only. One captured multi-line composer per harness would make the derived-state coverage evidence-backed rather than hand-authored.
- Muse has no negative fixture (I1) and Agy's negative doesn't reach the discriminator (I1). Codex is the only harness with a genuine, discriminator-exercising negative.
- The `TestHarnessTTYCapture*` family requires PTY allocation; it fails wholesale under a sandbox with `operation not permitted`. Not a defect, but worth knowing if this package ever runs somewhere without `/dev/ptmx`.

## 6. Architectural notes

- **ARCH-DRY — pass.** Three hand-rolled VT parsers collapse into one `x/vt`-backed model; two parallel registries collapse into `harnessTTYProfiles`; duplicated Codex/Agy paint literals collapse into `codexLiveComposerPaint` / `agyLiveComposerPaint`; the per-harness prefix scan collapses into `firstRecognizedHarnessTTYPrefix`. The only residue is the two near-identical row-paint loops noted in Minor.
- **ARCH-PURE — pass.** Recognizers are pure `terminalSnapshot → bool` and are tested with hand-built snapshots that touch no IO; `decidePlainReturn` is pure; the proxy is a thin feed/resize/decide shell. `snapshotCoordinatesValid` reusing `validateTerminalDimensions` keeps the recognizer and model bounds in one place.
- **ARCH-PURPOSE — flag (I1).** The single-source migration is complete: I ran the shadow sweep and no consumer of the retired registries or trackers survives anywhere in the repo, and Claude's legacy behavior is preserved through the same router. But the issue's stated purpose — "permission pickers and future UI variants keep plain Return as confirm" — is delivered with proof only for Codex. For Agy the discriminating rule has no captured negative, and for Muse there is no negative at all, so on those two harnesses the positive gate contributes no verified defense beyond the marker layer it was meant to backstop.
- **ARCH-MOCK — partial (I1, I2).** The shape is right: production and test flow share `proxy.handleChunk`, fixtures are literal bytes captured through a bounded PTY seam with per-file SHA-256, and live conformance correctly asserts behavior rather than byte identity. Incomplete on the negative-state axis, and the mechanism meant to keep that gap visible is silent by default. The `c872b91` correction — describing the live checks as manual/opt-in rather than scheduled — is honest and the right call; `sdlc`-side or release-runbook scheduling remains the open half.

## 7. Plan revision recommendations

Add a `## Revisions` entry to `workshop/plans/000139-agy-return-rewrite-only-in-composer-plan.md` covering:

1. **Agy negative-fixture characterization (I1).** The 2026-08-19 revision records `agy/1.1.15/overlay.raw` as closing I2/I3 and states the gate "declines it with the overlay layer *not* armed, so it exercises the positive gate alone". Correct this: the capture is the composer box with the cursor hidden and parked below the bottom rule, so it exercises cursor visibility and position, not the bright-blue prompt rule that fixes I1. Record that a real Agy permission-picker capture is required before the Agy positive gate can be claimed as a marker-independent defense, and that the same is true for Muse.
2. **EVIDENCE GAP visibility (I2).** Task 6's design says the gap is "named in a loud line rather than passing silently"; as implemented it is a `t.Logf` invisible without `-v`. State the reporting channel explicitly (stderr, or a visible subtest) so the claim and the mechanism agree.
3. **Codex status-row ambiguity (I3).** Record that the status-row exclusion resolves "painted row, blank above, nothing below" as *not a composer*, that Codex's full-screen-erase repaint makes that state reachable mid-frame, and that the chosen resolution is deliberate — with a pinning test row so a future edit cannot flip it silently.
4. **Fixture replay cost (I4).** Task 6 Step 5 mandates every split from 0 to `len(raw)` for every fixture. Note the measured cost (12.7 KB → 10.9s, ~80% of the package) and decide the policy for the next harness before #138 adds one.

---

## Re-review — 2026-08-19T14:11:42-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 139 — Agy Return rewrite only in composer |
| repo | pair |
| issue file | workshop/issues/000139-agy-return-rewrite-only-in-composer.md |
| boundary | whole-issue close |
| milestone | — |
| window | 596b1b31dadac65fc24aede0f20435d900125510..HEAD |
| command | sdlc close --issue 139 |
| reviewer | claude |
| timestamp | 2026-08-19T14:11:42-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The three findings that blocked the previous two rounds are genuinely fixed, and I reproduced each through the production path rather than taking the Log's word: a Codex composer with a blank line inside it recognizes, a two- and three-line Muse composer recognizes, and an unstyled `>` between rules no longer qualifies for Agy. The shadow sweep is completely clean — no `sendKeymapByAgent`, `overlayDetectorByAgent`, `sendKM`, per-agent tracker, `resizeTerminal`, `clearOverlay`, `rowHasBackground`/`colorMatches`, or `agentBasename == "<harness>"` composer branch survives anywhere outside `workshop/` history. `go test ./...`, `make test`, and the focused package all pass; `-race` on `wrapcmd` reports exactly one failure, the pre-existing `TestMasterPumpFlushesStdoutOnTick` `bytes.Buffer` race in a file this window does not touch. What I'd fix before crossing is evidence, not shipped defects: `ttyFixtureNegativeGaps` reports Agy as covered by a fixture the team's own Log already says declines on cursor state rather than on any composer-vs-picker discriminator, so the one harness this issue is named after still has no captured state where its gate refuses — and I confirmed all three recognizers silently flip to "submit" once a draft exceeds ~20 rows, with only Codex pinning that boundary.

## 1. Strengths

- **The every-split fixture replay runs the real seam and is the strongest test here.** `harness_tty_fixture_test.go:247` establishes an unsplit baseline per fixture, then requires the recognizer result, overlay arming, emitted Return bytes, *and* the decision reason to be identical at every split — so PTY chunk boundaries provably cannot change what Return does. The 1024-byte exhaustive prefix + strided tail with a logged skip count is a good answer to the cost problem, and it honors the plan's own no-silent-caps rule.
- **`ttyFixtureNegativeGaps` is the right shape for the "loud warning" problem.** Moving from a `t.Logf` (invisible in a passing package) to a check that *fails* for any positively gated harness neither covered nor acknowledged — and fails again when an acknowledgment outlives its gap (`harness_tty_fixture_test.go:127-135`) — converts a silent pass into a real gate. My only complaint is what it currently counts as coverage (I1).
- **Fail-closed defaults are consistent and pinned end to end.** `composerGateUnknown = iota` plus the exhaustive switch in `decidePlainReturn`, the empty-`plainCR` guard, and the `csiBytes == 5` canonical-spelling pin all fail toward bare CR; `TestDecidePlainReturn` covers the all-zero profile and an out-of-range policy explicitly.
- **The Agy documentation correction is exactly the right instinct.** `composer_recognizers.go:149-155` states in code that the bright-blue rule is a *necessary* condition and **not** a picker discriminator, naming `menu.raw` as the counter-evidence. Pinning `menu.raw` with expectation `true` records the accepted risk as a checked property instead of an assumption — that is better engineering than a comment claiming a defense that isn't there.
- **`terminalModel` lifecycle holds up under adversarial reading.** Reply-pipe `io.Closer` capability asserted at construction, idempotent `Close` with a shared `closeDone` result, post-close `io.ErrClosedPipe`, deep-cloned final snapshot, and the exclusive prepare/commit/abort resize token. Teardown ordering in `run()` is correct: the signal defer is registered last, so signal delivery is stopped and joined before `closeTerminal()`.

## 2. Critical findings

None.

## 3. Important findings

**I1 — Agy's positive gate has no captured state where it declines *as a gate*, and `ttyFixtureNegativeGaps` reports it as covered.**
`cmd/internal/wrapcmd/harness_tty_fixture_test.go:139` (and the `negatives[metadata.Agent] = true` bookkeeping at line 96)

The guard counts any `overlay.raw` as a captured declining state. I replayed `testdata/tty/agy/1.1.15/overlay.raw` through `terminalModel`: it lands on the **alternate screen with the cursor hidden at (15,5)**, below the bottom rule of a box whose column 0 still holds `─` (fg 8) / `>` (fg 12) / `─` (fg 8). It declines on cursor visibility and position — never on the prompt-color rule, and never on anything distinguishing a picker from a composer. The issue Log already says this in the FIX-THEN-SHIP entry, but the guard does not, so `ttyFixtureNegativeGaps` names only `muse` while Agy's equally real hole reads as closed. Meanwhile the one captured Agy screen that *is* a menu (`menu.raw`) is pinned `true` — no marker matches it and the gate accepts it.

*Failure scenario.* Agy renames its permission-picker wording — the exact #000042 drift the `overlay-detect/near-miss` signal exists to fingerprint. `agyPickerMarkers` stops matching, the positive gate accepts the screen (as it demonstrably does for the slash menu, whose selection marker is painted in the same `fg 12` bright blue at column 0), and plain Return leaks a newline into a permission dialog instead of confirming. Nothing in the suite fails, and the negative-gap ledger says Agy is covered. For the harness this issue is named after, "permission pickers keep plain Return as confirm" is still delivered entirely by `agyPickerMarkers` — the marker layer the positive gate was written to backstop (ARCH-PURPOSE, ARCH-MOCK).

*Fix sketch.* Two parts, both cheap. (a) Add `agy` to `ttyFixtureNegativeGaps` with the honest reason ("the captured `overlay.raw` declines on hidden cursor and cursor position, not on any composer-vs-picker rule; a permission-picker capture is outstanding") so the ledger stops over-reporting. (b) The capture is reachable: `harnessTTYDrivenScenarios["agy"]` passes `--dangerously-skip-permissions`, which is precisely what suppresses the picker — add a scenario without that flag that drives one tool call, capture it as `picker.raw` with expectation `false`, and let the fixture test prove the rejection. The same applies to Muse, which is at least correctly acknowledged today.

**I2 — All three recognizers flip to "not a composer" past a fixed height, and Return then submits the partly-written draft; only Codex pins its boundary.**
`composer_recognizers.go:23` (`codexComposerMaxRows = 20`), `:98` (`museComposerMaxRows = 20`), `:157` (`maxBoxHeight = 25`)

Reproduced by feeding synthetic-but-structurally-faithful composers through `terminalModel` and calling the production recognizers on a 120×38 screen:

| harness | lines in draft | `recognize` |
|---|---|---|
| codex | 19, 20 | `true` |
| codex | 21, 22, 25, 30 | **`false`** |
| muse | 19, 20 | `true` |
| muse | 21, 22, 25 | **`false`** |
| agy | 23, 24 | `true` |
| agy | 25, 26, 30 | **`false`** |

*Failure scenario.* A user writing a long prompt in a ~38-row agent pane reaches line 21 (Muse/Codex) or line 25 (Agy); the composer is fully visible and unambiguous, but the gate declines and the next Enter emits bare CR, submitting the half-written message. This is the same failure class as C1 and C2 from the previous two rounds — the expensive direction, not the fail-closed one — at a different threshold. Codex pins its boundary (`"prompt beyond the composer height bound"` in `TestCodexComposerActiveSnapshotDifferential`); **Muse and Agy have no boundary row at all**, so a future edit could move either bound with nothing failing.

I verified the recognizer behavior, not the harnesses' willingness to grow a composer that tall, so reachability depends on whether each CLI expands past ~20 rows before scrolling the prompt off — I could not capture that here.

*Fix sketch.* Add a boundary row to `TestMuseComposerActiveSnapshotDifferential` and `TestAgyComposerActive` recording the intended answer and its consequence, the way Codex's row does. Then consider raising the bounds: for Muse and Agy the *enclosing-rule* structure already prevents pairing unrelated distant chrome, so the extra height cap buys little and costs a premature submit.

## 4. Minor findings

- `wrap.go:1619` — `checkOverlayOpen` now returns early when `p.ttyProfile == nil`, which is the case under `PAIR_WRAP_REMAP_RETURN=0`. Overlay detection (and its `overlay-detect` `fired`/`near-miss` telemetry) previously ran regardless, keyed only on agent name. Functionally harmless — `pickerActive` has no consumer when the remap is off — but it silences the drift fingerprint `doctor/README.md` leans on, and the scope change is undeclared in the plan, issue, README, or atlas.
- `atlas/how-to-bring-up-a-new-harness-cli.md:66` names `TestHarnessTTYLiveOverlayConformance`; no such test exists — it is `TestHarnessTTYLiveDrivenConformance`, and the command block above it names `TestHarnessTTYLiveConformance`, which does not capture overlays.
- `doctor/README.md:41` says a `composer unknown` reason "means no profile matched". When no profile matches, `emitPlainCR` returns bare CR at `wrap.go:1739` and logs *nothing*. `composer unknown` actually means a positive-gated profile with no snapshot or no registered recognizer.
- `wrap.go:1481` — if `releaseTerminal()` errors, the freshly constructed `terminalModel` (goroutine + emulator) is neither assigned nor closed, and the proxy is left with a positive-gated profile and a nil terminal. Unreachable from the single startup call site, where `p.terminal` is always nil.
- `harness_tty_fixture_test.go:229` — `ttyFixtureExpectation` is keyed by filename globally rather than per harness, so a future harness whose `menu.raw` must *decline* would be forced to `true`.
- `harness_tty_live_test.go` — the `PAIR_LIVE_CAPTURE_OUT` write block is duplicated verbatim between `TestHarnessTTYLiveConformance` and `TestHarnessTTYLiveDrivenConformance` (ARCH-DRY nit).
- `harness_tty.go` — `decidePlainReturn`'s `composerGateLegacy` branch and its positive-success branch construct identical `returnDecision` values.
- `composer_recognizers_test.go:349` — local variable named `new`, shadowing the builtin.
- `wrap.go` — three test-only seams now live on the production `proxy` struct (`getWinsize`, `setPTYWinsize`, `overlayConsumeHook`). Each is documented; worth watching that the list stops growing.

## 5. Test coverage notes

- Derived-state coverage — the gap that produced C1/C2 — is now real for Codex (blank line inside, two consecutive newlines, empty continuation row, mid-frame vs. completed frame) and Muse (two- and three-line composers). Agy's growth case is covered only by hand-built snapshots; I confirmed independently that a realistic full-width multi-line Agy box is accepted.
- The tall-draft boundary (I2) is the one derived-state axis still unpinned for Muse and Agy.
- No fixture captures a *composing* state for any harness — all six are startup, menu, or overlay screens. The Log says multi-line Codex states were driven live during Task 6A but only the startup screen was checked in, so the every-split replay protects the startup path only. One captured multi-line composer per harness would make the derived-state coverage evidence-backed rather than hand-authored.
- Codex is still the only harness with a negative fixture that exercises its discriminator (the update interstitial paints `›` unemphasized while the composer paints it bold — genuinely load-bearing evidence). Agy's is I1; Muse has none.
- The `TestHarnessTTYCapture*` family needs PTY allocation and fails wholesale under a sandbox with `operation not permitted` on `pty.StartWithSize`. Not a defect, but relevant if this package ever runs where `/dev/ptmx` is unavailable.

## 6. Architectural notes

- **ARCH-DRY — pass.** Three hand-rolled VT parsers collapse into one `x/vt`-backed model; two parallel registries collapse into `harnessTTYProfiles`; `rowPaintedBetween`, `firstRecognizedHarnessTTYPrefix`, `codexLiveComposerPaint`, and `agyLiveComposerPaint` each replace a duplicated block flagged in earlier rounds; `validateTerminalDimensions` is shared by the model and `snapshotCoordinatesValid`. Only the two test-side nits above remain.
- **ARCH-PURE — pass.** Recognizers are pure `terminalSnapshot → bool` and `TestAgyComposerActive` runs them on hand-built snapshots with no IO at all; `decidePlainReturn` is pure; the proxy is a thin feed/resize/decide shell. The differential tables read `testdata` fixtures as input data, not as mocks standing in for behavior. Cursor-visibility evidence correctly comes from a bounded `x/ansi` parser sharing x/vt's state semantics rather than a second partial DFA.
- **ARCH-PURPOSE — flag (I1).** The single-source migration is complete and verified: the shadow sweep is clean and Claude's legacy behavior routes through the same profile registry. But the issue's stated point — "permission pickers and future UI variants keep plain Return as confirm" — is delivered with proof only for Codex. For Agy the discriminator is documented as insufficient with no captured picker, and for Muse there is none; on both, the positive gate contributes no verified defense beyond the marker layer it was meant to backstop.
- **ARCH-MOCK — partial (I1).** Production and test flow share `proxy.handleChunk`; fixtures are literal bytes captured through a bounded PTY seam with per-file SHA-256 and metadata; live checks correctly assert behavior over byte identity, and `c872b91`'s correction to describe them as manual/opt-in rather than scheduled is the honest call. Incomplete on the negative-state axis, and nothing schedules the live conformance — so harness drift is caught when someone remembers to run it.

## 7. Plan revision recommendations

Add a `## Revisions` entry to `workshop/plans/000139-agy-return-rewrite-only-in-composer-plan.md` covering:

1. **Negative-evidence ledger accuracy (I1).** The 2026-08-19 FIX-THEN-SHIP entry correctly retracts the claim that `agy/1.1.15/overlay.raw` "exercises the positive gate alone", but `ttyFixtureNegativeGaps` was not updated to match — it still reports Agy as covered. Record that `agy` belongs in the ledger until a real permission-picker capture exists, and that the capture is reachable by dropping `--dangerously-skip-permissions` from the Agy driven-scenario argv.
2. **Composer height bounds (I2).** Record the measured thresholds (Codex 20 rows prompt-to-cursor, Muse 20, Agy 25), that exceeding them emits bare CR and submits the draft, and that only Codex pins its boundary today. Require a boundary row for Muse and Agy, and state whether the bounds are intentional ceilings or an artifact carried from the startup-screen derivation.
3. **`PAIR_WRAP_REMAP_RETURN=0` scope change.** Overlay detection and its `overlay-detect` telemetry now stop entirely when the remap is opted out; previously they ran regardless of the opt-out. Declare it as intended (and update README/`doctor/README.md`) or restore the prior behavior.
4. **Doc pointer corrections.** `atlas/how-to-bring-up-a-new-harness-cli.md` names a nonexistent `TestHarnessTTYLiveOverlayConformance`, and `doctor/README.md`'s `composer unknown` mapping is wrong (no-profile logs nothing at all).
