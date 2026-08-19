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
