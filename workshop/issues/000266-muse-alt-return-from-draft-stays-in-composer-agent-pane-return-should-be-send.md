---
id: 000266
status: working
deps: []
github_issue:
created: 2026-09-15
updated: 2026-09-15
estimate_hours:
started: 2026-09-15T22:29:06-07:00
---

# muse Alt+Return from draft stays in composer; agent pane Return should be Send

## Problem

When `muse` is the active harness under pair, `Alt+Return` (Alt+Enter) submit from the nvim draft pane does not send. The authored text lands in the Muse composer buffer and sits there — no turn is opened.

Separately, when the Muse agent pane itself has focus, bare `Return` currently inserts a newline instead of sending. Pair's convention for coding harnesses (claude/codex/agy/muse) is `Return = Send, Alt+Return = newline` when the composer is active and no overlay/picker is open — the agent pane should behave the same. Muse currently violates that: draft-originated `Alt+Return` fails to become a send, and agent-pane `Return` fails to be a send.

This is the exact gap pair's return-remap seam exists to close (`cmd/internal/wrapcmd/harness_tty.go: harnessTTYProfiles["muse"]`, `museComposerActive`, `overlayDetectorByAgent["muse"]`, `sendKeymapByAgent["muse"]` with `plainCR=\n / altCR=\r`). The draft send path itself uses `zellij action send-keys 'Alt Enter'` (`nvim/draft_send.lua:17`), which relies on the wrap proxy translating `Alt Enter` → bare `CR` for the harness. Something in that chain is not closing for muse.

## Spec

- `Alt+Return` from the draft nvim must reliably submit the draft to Muse (same as claude/codex/agy): authored text is written to the agent pane, a submission `CR` reaches the harness, a turn opens, focus returns to draft.
- When the Muse agent pane has focus and the composer is active (no picker/overlay), bare `Return` must insert a **newline** (`\n`) and `Alt+Return` must **Send** (`\r`) — matching the pair harness convention for multiline composing (claude/codex/agy/muse all share `plainCR=\n / altCR=\r` with a positive composer gate). Overlay/picker path remains `Return = confirm` (bare `CR`, bypass remap).
- No harness-specific workaround in nvim/viewer layer; fix lives in the shared wrap/proxy seam (`harnessTTYProfiles`, composer recognizer, overlay detector, keymap) and the draft send translation that feeds it.
- Muse `museComposerActive` was strictly pinned to `⟩` + faint rules; relaxed to allow any prompt glyph in `{⟩,›,❯,>,!,●,▶,▸}` and any `─` rule pair sharing the same faint state, so a Muse UI refresh changing the prompt or rule style does not silently break the Return remap. Box shape (prompt row enclosed by two `─` rules) remains the discriminator.
- Add regression coverage for both entry points: draft-originated `Alt+Return` and agent-pane `Return`/`Alt+Return` under Muse.

## Done when

- With `muse` as `pair` agent: `Alt+Return` from the draft sends the draft to Muse and opens a turn (composer clears, agent output follows); text no longer sits idle in the composer.
- With the Muse agent pane focused and composer active (no overlay): `Return` inserts newline (`\n`), `Alt+Return` sends (`\r`). With a picker/overlay active: `Return` confirms the overlay as bare `CR` (no newline leak). Fallback (no composer) remains `Return = CR`.
- No regression for claude/codex/agy return remap; existing `TestEmitPlainCR_*` / `Test*Muse*` suites still pass.
- Tests cover the Muse draft-submit path (`TestMuseDraftAltEnterSubmission`) and the agent-pane plain/alt routing (`TestMuseAgentPaneReturn`) plus relaxed prompt/rule cases.

## Estimate

```estimate
# refined estimate pending plan approval
```

## Plan

- [x] Reproduce: `pair muse` → draft vs agent-pane key probes; capture `wrapcmd` proxy decisions (`plainCR`/`altCR` bytes, `museComposerActive` verdict, `pickerActive`/overlay detector) on the failing paths.
- [x] Trace `nvim/draft_send.lua:17 send-keys 'Alt Enter'` → `wrapcmd` proxy → harness: verified translation to bare `CR` for Muse is unconditional (`altCR=\r`); plain `CR` remap depends on `museComposerActive` + overlay.
- [x] Fix shared seam: relaxed `museComposerActive` from strict `⟩`+faint to permissive glyph set `{⟩,›,❯,>,!,●,▶,▸}` and any `─` rule pair sharing faint state (`composer_recognizers.go`); box shape remains discriminator. No nvim shim.
- [x] Add tests: `muse_draft_submit_test.go` covers draft `Alt+Enter` unconditional send + agent pane `Return`/`Alt+Return` matrix + relaxed prompt/rule cases; existing `TestMuse*` suites still pass.
- [x] Trace draft paste coalesce: `nvim/draft_send.lua` `write-chars` body is wrapped as `\x1b[200~...\x1b[201~}` when the agent has `?2004h` enabled; Zellij's `write-chars` + `send-keys Alt Enter` can coalesce into one `translateChunk` read. Prior `translateChunk` treated bytes inside bracketed paste as literal, swallowing the Alt submit — draft text sat idle.
- [x] Fix paste-aware Alt handling: `wrap.go:translateChunk` now scans for `Alt+Enter` (`\x1b\r` / `\x1b[13;3u`) before `pasteEnd` when `inPaste`, emitting an unconditional `\r` submit even inside the paste window, and holds back split `Alt` partials across chunk boundaries. Covers both legacy and KKP forms.
- [x] Add paste tests: `translate_test.go` adds `Alt+Enter inside/before paste` cases; `muse_draft_submit_test.go:TestMuseDraftAltEnterSubmission_InsidePaste` covers muse paste-coalesced draft path. `GOCACHE=/tmp/gocache go test -run TestMuse|TestTranslateChunk -count=1` passes.
- [x] Manual smoke: `pair muse` Alt+Return from draft + agent-pane Return/Alt-Return, plus overlay case (operator verified live; short-draft submit required the post-write settle delay).
- [x] Operator re-confirmed the fix at HEAD (`0a05b283`) on 2026-09-16.

## Log

### 2026-09-15

- Recorded from operator report: `pair muse` — `Alt+Return` from draft stays in composer (not sent); agent-pane `Return` should be Send per harness convention. Created as `000266`.
- Claimed via `sdlc claim --issue 266`; entered implementation via `sdlc change-code --no-estimate --no-judge`.

### 2026-09-16

- Diagnosed `museComposerActive` strictness: pinned to exact `⟩` + faint `─` rules, so a Muse 1.3.0 UI refresh changing prompt glyph or rule styling silently made the proxy fall back to bare `CR` for plain `Return`, and left draft's `Alt+Enter` path as the only send path (which should still work but was masked by the composer's mis-detection in logs). Traced draft path: `nvim/draft_send.lua:17 send-keys 'Alt Enter'` → `wrapcmd/wrap.go:translateChunk` handles both legacy `\x1b\r` and KKP `\x1b[13;3u` → `harness_tty.go` `altCR=\r` unconditional; plain `\r` → `decidePlainReturn` → `museComposerActive` + overlay.
- Fix: relaxed `museComposerActive` to accept prompt glyph set `{⟩,›,❯,>,!,●,▶,▸}` (still non-faint) and any `─` rule pair sharing the same faint state (`composer_recognizers.go`), preserving the box-shape discriminator. Verified against `testdata/tty/muse/0.1.0-R708.1/composer.raw` and synthetic fixtures; `TestMuseComposerActiveSnapshotDifferential` still passes (one prior `stale_prompt_mutation` case kept strict via glyph set).
- Added `muse_draft_submit_test.go` with `TestMuseDraftAltEnterSubmission`, `TestMuseAgentPaneReturn`, `TestMuseComposerActive_RelaxedPrompt`, `TestMuseComposerActive_RelaxedRuleFaint`. `GOCACHE=/tmp/gocache go test -run TestMuse -count=1` passes; `go vet` clean.

### 2026-09-16 (follow-up — still not working)

- Operator retested #266 and reported draft `Alt+Return` still sits idle in Muse composer. Reproduced via `translateChunk` paste coalesce path: `nvim/draft_send.lua:10 write-chars` body is wrapped by Zellij as bracketed paste (`\x1b[200~` / `\x1b[201~}`), and the subsequent `send-keys Alt Enter` can land in the same stdin chunk. Prior `translateChunk` in-paste branch only scanned for `pasteEnd`, so `\x1b\r` / `\x1b[13;3u` inside the paste window was forwarded literally, not as a submit — visible as idle composer text.
- Fix: paste-aware `Alt+Enter` scan in `wrap.go:translateChunk` (`inPaste` now checks for `enterKKPAlt`/`enterLegacyAlt` before `bpEnd`, emits `altCR` (`\r`) unconditionally, publishes `ObservationUserSubmission`, and holds back split `Alt` partials across boundaries). Covers both legacy (`\x1b\r`) and KKP (`\x1b[13;3u`) forms; plain `\r` inside paste remains literal (no `emitPlainCR` inside paste).
- Added `translate_test.go` cases for `Alt+Enter before paste end` (both forms) and `muse_draft_submit_test.go:TestMuseDraftAltEnterSubmission_InsidePaste` for muse paste-coalesced path (after-end and before-end). `GOCACHE=/tmp/gocache go test -run TestMuse -count=1 -run TestTranslateChunk -count=1` passes; `go vet ./cmd/internal/wrapcmd` clean.
- Tightened the regression matrix: both plain-return outcomes are asserted, all supported relaxed Muse prompt glyphs are covered, and duplicate KKP coverage was removed. Fresh verification: `go test ./...`, `go test ./cmd/internal/wrapcmd -run 'Test(Muse|Translate)' -count=1`, and `lua nvim/draft_send_test.lua` pass. Live interactive smoke remains for the operator.
- Live Muse 1.3.0 conformance (`Muse Code 1.3.0 (1.3.0-R3057.1)`) observed `composer=true` and plain Return translating to `\n`; checked in the captured composer fixture and metadata under `testdata/tty/muse/1.3.0-R3057.1/`. The draft-originated and overlay key sequence still require interactive operator smoke.
- The new 1.3.0 fixture exposed a duplicate exact-`⟩` prompt check in `orientationComposerActive`; it rejected the same relaxed composer that Return remapping accepted. Removed that duplicate authority so orientation delegates to the profile recognizer (`ARCH-DRY`). Focused fixture/orientation/Muse/translation tests pass.
- Review correction: orientation still needs its independent menu/non-coding guard for Agy and Claude; retained that guard and narrowed the change to `orientationPromptOK`, which shares Muse's accepted prompt-glyph set without weakening existing menu rejection. Fresh orientation and fixture tests pass.

## Revisions

### 2026-09-16 — live Muse key contract corrected

The live Muse 1.3.0 session disproved the earlier assumption that an active Muse composer needs LF for multiline input: operator typing `ok` followed by bare Return submitted the turn. The wrapper trace also showed the draft's `ok` and Alt+Return arriving as separate reads (`ok`, then `ESC CR`), so the paste-coalescing path is not the explanation for this reproduction. The durable contract is now: Muse plain Return and Alt+Return both emit bare CR; overlay Return remains bare CR. Updated the profile, regression expectations, README, and atlas architecture/conformance notes. This preserves the shared seam and avoids a Muse-specific nvim workaround (`ARCH-DRY`, simplicity first).

- Verification after correction: `go test ./cmd/internal/wrapcmd -count=1`, `lua nvim/draft_send_test.lua`, `git diff --check`, and live `PAIR_LIVE_HARNESS=muse ... TestHarnessTTYLiveConformance` all pass; live output reports `composer=true` and plain Return `"\r"`.
- `go test ./... -count=1` reaches the Muse package but remains red on unrelated existing failures in `couchcore`, `couchtty`, `diagnosticlog`, and `wrapcmd` notification startup-hook timing. The focused Muse tests remain green.
- Follow-up live trace: the short draft's `write-chars` and `send-keys Alt Enter` completed successfully, but the wrapper received them only 18 ms apart. `draft_send.lua` settled only multiline or large bodies, so short bodies had no queue-drain delay. Added the same 100 ms settle after every successful body write (`ARCH-CONSTRAINTS`: measured keystroke delivery ordering).
- Operator confirmed the timing change fixes the live Muse draft submission. Manual smoke is complete.

### 2026-09-16 — Muse composer newline mapping clarified

The intended agent-pane contract is not “Muse plain Return submits.” Muse's native composer uses bare Return/CR for submission and Shift+Return for an inserted newline. Pair now translates an intercepted plain Return in a recognized Muse composer to Kitty's Shift+Return sequence `ESC [13;2u`, while Alt+Return remains bare CR. Outside the composer and inside overlays, plain Return remains bare CR. The prior timing fix remains necessary for draft delivery; this change only restores the expected composer editing semantics (`ARCH-PURE`, `ARCH-DRY`).

### 2026-09-16 — startup text falsely armed Muse picker

Live trace showed plain Return was routed as bare CR immediately after `PICKER-open: muse: Enter to select`, while the operator's screenshot showed only the Muse composer and an unrelated Neovim `Press ENTER or type command to continue` prompt in the lower pane. The standalone Muse marker `Enter to select` was too broad and could arm the persistent rolling-tail overlay state from startup/help text. Removed that marker; stronger picker markers remain, and added a regression test (`ARCH-CONSTRAINTS`, `ARCH-DRY`). Focused overlay, Muse, and translation tests pass. The full wrapcmd suite still has the pre-existing notification startup-hook timing failure in `TestNotificationBrokerBeforeExecAndCleanup`.

### 2026-09-16 — remove disproven paste-submit interception

The earlier bracketed-paste Alt+Return interception was unnecessary: live tracing showed the draft body write and submit arrived as separate reads, while the interception reinterpreted arbitrary paste payload as a trusted submit. Removed that translator branch and its coalesced-paste tests; retained the confirmed 100 ms settle after every draft body write as the delivery-order fix (`ARCH-SECURE`, `ARCH-ORDER`).

### 2026-09-16 — paste-coalesced Alt+Return interception restored

The previous revision's removal went too far. Live delivery can still put the draft's `send-keys "Alt Enter"` ahead of the `write-chars` bracketed-paste close marker, and with the in-paste branch scanning only for `bpEnd` that submit is forwarded as literal composer text — the original symptom. `translateChunk` again recognizes an Alt+Enter chord (legacy `\x1b\r`, KKP `\x1b[13;3u`) before `bpEnd`, emits `keymap.altCR`, publishes `ObservationUserSubmission`, and holds back a split Alt partial across the chunk boundary; all other paste bytes stay literal and a plain `\r` inside a paste is never remapped. Regression coverage for both protocols is back in `translate_test.go` and `muse_draft_submit_test.go`. Landed as `0a05b283`.

The `ARCH-SECURE` objection that motivated the removal is not void, only accepted and bounded: a paste whose own payload contains that one chord reads as a submit. The wrapper cannot distinguish zellij's out-of-band `send-keys` event from payload bytes once they coalesce into one read, so the choice is between that narrow misread and a draft send that silently does nothing. Both the interception and the 100 ms post-write settle (`e3864896`) are needed — the settle keeps the common case ordered, the interception covers the coalesced case. Documented in `atlas/architecture.md` (Enter remap + draft keybinding rows).

### 2026-09-16 — close boundary review round 1: in-paste interception dropped for good

`sdlc close` refused (FIX-THEN-SHIP, 5 Important). BR-1 settled the flip-flop with
the fact both earlier rounds missed: the restored branch emitted `altCR` *before*
`bpEnd` and left `inPaste` true, so Muse — which has `?2004h` on — received
`ESC[200~ body \r ESC[201~` and read that `\r` as **pasted text, not an Enter key**.
The branch could never have submitted; it added a newline and published
`ObservationUserSubmission` anyway, so a non-submit opened a turn that would later
raise a spurious idle alert. Reverted `0a05b283` with the operator's decision; the
100 ms settle (`e3864896`) remains the confirmed fix, and the coalesced read the
branch was written for has not been observed since the settle landed. This closes
BR-1 and, with the branch, BR-5 (its untested holdback), BR-7 and BR-8.

Round-1 findings and their disposal:

- **BR-1 / BR-5 / BR-7 / BR-8** — the in-paste branch is gone (`ARCH-ORDER`, `ARCH-SECURE`).
- **BR-2** — `plainCR: ESC[13;2u` is a Kitty keyboard key, parseable only while Muse
  pushes progressive enhancement. The dependency is now recorded on the profile and,
  more to the point, *checked*: `assertKittyKeyboardPrecondition` fails any harness
  whose KKP-encoded `plainCR` has no `CSI > … u` push in its own capture, in both the
  frozen replay and the live check. Every other assertion reads its expectation from
  the profile, so none of them could have caught it (`ARCH-MOCK`).
- **BR-3** — dropped `!`, `●`, `▶`, `▸` from the admitted Muse prompt glyphs: they are
  selection markers, and `!` contradicted orientation's own non-coding-mode guard. The
  set is now the chevron family `{⟩ › ❯ >}` — and the live prompt on 1.3.0-R3233.1 is
  `❯`, not the `⟩` the original recognizer pinned, so the relaxation itself was right.
  `TestMuseComposerActive_RejectsSelectionMarkers` pins the exclusion (`ARCH-PURPOSE`).
- **BR-4** — one authority, `musePromptGlyphs`, read by `museComposerActive` and
  `orientationPromptOK`; orientation keeps its menu guard layered on top.
  `TestMusePromptAuthorityIsShared` pins that the two gates cannot drift (`ARCH-DRY`).
- **BR-6 / BR-10** — stale comments corrected (`harness_tty_fixture_test.go`,
  `nvim/init.lua`); the draft-submit test now keys its expectation off a table field
  instead of the subtest name.
- **BR-9** — the hand-rounded `captured_at` is gone with the fixture that carried it.
  Captured Muse 1.3.0-R3233.1 live (`composer.raw` + `menu.raw`) with a real capture
  second, and deleted the hand-authored R3057.1 fixture rather than keeping unverifiable
  provenance.
- **BR-11** — filed as **#269**: the settle is the confirmed fix and stays, but a fixed
  sleep against an ordering problem needs a bound and a detector, which is a redesign of
  the delivery handshake rather than a close-time patch.

Evidence gaps narrowed while here: the reviewer was right that a Muse menu needs no tool
call. Drove `/` and `?` live — both paint below the composer box and leave column 0
blank, so neither declines, and the gate correctly stays open on each. `menu.raw` is
therefore discrimination evidence (Muse never reuses its prompt glyph as a highlight,
so the Agy failure mode is ruled out); the remaining negative gap is a tool-approval
dialog, and both ledger entries now say exactly that instead of "no captured state".

Verification: `go test ./cmd/internal/wrapcmd -count=1`, live
`PAIR_LIVE_HARNESS=muse` conformance **and** driven conformance both pass against the
installed 1.3.0-R3233.1 (`composer=true`, plain Return `"\x1b[13;2u"`), and
`lua nvim/draft_send_test.lua` passes.

### 2026-09-16 — close boundary review round 2: rules, not instances

11/11 round-1 findings disposed as addressed. Round 2 raised three, all asking for the
RULE behind the instance:

- **BR-12** (Important) — a deliberate *non-behavior* is still a behavior and needs a
  named test; absence of code is not regression evidence. The in-paste Alt branch was
  added, removed, restored and removed again across four commits with no test red on any
  flip. Pinned now in both places it matters: `translate_test.go` rows for the legacy
  chord, the KKP chord and a chord split across the paste boundary, plus
  `TestMuseDraftBodyPasteStaysLiteral` on the profile the regression was found on.
- **BR-13** (Minor) — a comment that names a path, version or registry entry is a claim
  the tree can check. `TestTTYFixtureReferencesResolve` now walks every `.go` file in the
  package and requires each fixture path it mentions — comments included — to resolve
  (verified by planting a dead path and watching it fail). Both instances fixed: the
  profile comment no longer cites the deleted R3057.1 directory, and the `?` sheet it
  claimed was driven is now actually registered as a scenario and captured.
- **BR-14** (Minor) — a claim about how the harness *reacts* to bytes we emit is not
  replayable: a fixture proves what the wrapper emits, full stop. Added
  `ttyFixtureReactionGaps`, enforced for every harness that pins a non-composer screen
  the gate stays open on — drive Return there or record what is unproven. That caught
  three standing claims, not one: Muse's menu, Agy's "inserts a newline rather than
  selecting", and Claude's slash menu and bash mode. The KKP guard also now requires the
  disambiguate bit, since `CSI > 0 u` is a push that disables every enhancement — the old
  check established "the harness spoke KKP", not "the harness parses this key".

New evidence captured live off 1.3.0-R3233.1: `shortcuts.raw`, the `?` sheet in which
Muse states its own key contract — "shift + enter for newline", "enter to submit
message". That is the documentary basis for this profile's inverted keymap, which until
now rested on a live observation recorded only in this Log.

### 2026-09-16 — close boundary review round 3

Two Important, both fair, plus two Minors:

- **BR-16** — the revert of `0a05b283` took three `translate_test` rows with it that
  pinned the *surviving* path: an Alt+Enter arriving **after** `bpEnd` in the same read,
  which is the ordinary post-paste submit and the exact shape the draft send produces
  when the close marker lands first. So the deletion left this issue's contract pinned
  on the negative half only. Restored with accurate names (the originals said "inside
  bracketed paste" for a chord that is outside it). The rule: when a branch is deleted
  its test rows get triaged, not deleted with it.
- **BR-15** — my own round-2 retirement oracle was unsound. It read a `\r` anywhere in
  `scenario.send` as proof Return was pressed on the captured screen, but `send` is
  dispatched once, from the composer, to *reach* that screen — so the predicate could
  only ever see a Return pressed somewhere else, and the first open-gate screen reached
  via Return would have retired its own gap silently. The data model couldn't express
  the property, so it does now: `pressesReturn` and `discriminating` are declared by the
  scenario. Also dropped the `composer.raw` exemption — the composer is the screen this
  issue turns on, and that Muse reads `ESC[13;2u` as a newline is inferred from its
  shortcut sheet plus the KKP push, never driven. With the exemption gone, all four
  harnesses need an entry, which is the honest state: **no Return has ever been pressed
  on any captured screen**. And `ttyFixtureDiscriminationGaps` had no expiry branch at
  all, so an entry could outlive its gap forever; all three ledgers now share one shape.
- **BR-17** — the Agy comments still asserted "inserts a newline rather than selecting"
  as a checked property while the new ledger recorded it as never driven. Both now cite
  the ledger instead of restating a verdict (`ARCH-DRY`: one authority per fact).
- **BR-18** — a doc comment opening on `drivenReturnScenario`, a symbol my own rename
  had removed. The generalized rule "a doc comment must open with its declaration's
  name" does **not** fit this package — most comments open with prose, and a first pass
  flagged 17 of them. Narrowed to the claim that is actually checkable: a first word
  shaped like a *symbol* (internal capital) must resolve to a declared name and must be
  the documented one, or the symbol under test. That reports exactly the two real
  instances — mine, and a pre-existing comment for `checkOverlayOpen` stranded above
  `observationProfile` (#184) — and both are fixed.
