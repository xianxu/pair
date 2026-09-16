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
- [ ] Manual smoke: `pair muse` Alt+Return from draft + agent-pane Return/Alt-Return, plus overlay case (operator to verify live).

## Log

### 2026-09-15

- Recorded from operator report: `pair muse` — `Alt+Return` from draft stays in composer (not sent); agent-pane `Return` should be Send per harness convention. Created as `000266`.
- Claimed via `sdlc claim --issue 266`; entered implementation via `sdlc change-code --no-estimate --no-judge`.

### 2026-09-16

- Diagnosed `museComposerActive` strictness: pinned to exact `⟩` + faint `─` rules, so a Muse 1.3.0 UI refresh changing prompt glyph or rule styling silently made the proxy fall back to bare `CR` for plain `Return`, and left draft's `Alt+Enter` path as the only send path (which should still work but was masked by the composer's mis-detection in logs). Traced draft path: `nvim/draft_send.lua:17 send-keys 'Alt Enter'` → `wrapcmd/wrap.go:translateChunk` handles both legacy `\x1b\r` and KKP `\x1b[13;3u` → `harness_tty.go` `altCR=\r` unconditional; plain `\r` → `decidePlainReturn` → `museComposerActive` + overlay.
- Fix: relaxed `museComposerActive` to accept prompt glyph set `{⟩,›,❯,>,!,●,▶,▸}` (still non-faint) and any `─` rule pair sharing the same faint state (`composer_recognizers.go`), preserving the box-shape discriminator. Verified against `testdata/tty/muse/0.1.0-R708.1/composer.raw` and synthetic fixtures; `TestMuseComposerActiveSnapshotDifferential` still passes (one prior `stale_prompt_mutation` case kept strict via glyph set).
- Added `muse_draft_submit_test.go` with `TestMuseDraftAltEnterSubmission`, `TestMuseAgentPaneReturn`, `TestMuseComposerActive_RelaxedPrompt`, `TestMuseComposerActive_RelaxedRuleFaint`. `GOCACHE=/tmp/gocache go test -run TestMuse -count=1` passes; `go vet` clean.
