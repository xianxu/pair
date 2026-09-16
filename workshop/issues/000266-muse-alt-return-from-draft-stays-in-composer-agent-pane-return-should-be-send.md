---
id: 000266
status: open
deps: []
github_issue:
created: 2026-09-15
updated: 2026-09-15
estimate_hours:
---

# muse Alt+Return from draft stays in composer; agent pane Return should be Send

## Problem

When `muse` is the active harness under pair, `Alt+Return` (Alt+Enter) submit from the nvim draft pane does not send. The authored text lands in the Muse composer buffer and sits there — no turn is opened.

Separately, when the Muse agent pane itself has focus, bare `Return` currently inserts a newline instead of sending. Pair's convention for coding harnesses (claude/codex/agy/muse) is `Return = Send, Alt+Return = newline` when the composer is active and no overlay/picker is open — the agent pane should behave the same. Muse currently violates that: draft-originated `Alt+Return` fails to become a send, and agent-pane `Return` fails to be a send.

This is the exact gap pair's return-remap seam exists to close (`cmd/internal/wrapcmd/harness_tty.go: harnessTTYProfiles["muse"]`, `museComposerActive`, `overlayDetectorByAgent["muse"]`, `sendKeymapByAgent["muse"]` with `plainCR=\n / altCR=\r`). The draft send path itself uses `zellij action send-keys 'Alt Enter'` (`nvim/draft_send.lua:17`), which relies on the wrap proxy translating `Alt Enter` → bare `CR` for the harness. Something in that chain is not closing for muse.

## Spec

- `Alt+Return` from the draft nvim must reliably submit the draft to Muse (same as claude/codex/agy): authored text is written to the agent pane, a submission `CR` reaches the harness, a turn opens, focus returns to draft.
- When the Muse agent pane has focus and the composer is active (no picker/overlay), bare `Return` must **Send**, and `Alt+Return` must insert a **newline** — matching the pair harness convention. Overlay/picker path remains `Return = confirm` (bypass remap).
- No harness-specific workaround in nvim/viewer layer; fix lives in the shared wrap/proxy seam (`harnessTTYProfiles`, composer recognizer, overlay detector, keymap) and the draft send translation that feeds it.
- Preserve existing Muse recognizer contract (`museComposerActive` = non-faint `⟩` at col 0 inside faint `─` rules, cursor anywhere in box) unless evidence shows the composer shape changed; if it changed, update recognizer + frozen `testdata/tty/muse/*/composer.raw` accordingly.
- Add regression coverage for both entry points: draft-originated `Alt+Return` and agent-pane `Return` under Muse.

## Done when

- With `muse` as `pair` agent: `Alt+Return` from the draft sends the draft to Muse and opens a turn (composer clears, agent output follows); text no longer sits idle in the composer.
- With the Muse agent pane focused and composer active (no overlay): `Return` sends, `Alt+Return` inserts newline. With a picker/overlay active: `Return` confirms the overlay (no newline leak).
- No regression for claude/codex/agy return remap; existing `TestEmitPlainCR_*` suites still pass.
- Tests cover the Muse draft-submit path and the agent-pane plain-Return/Alt-Return routing.

## Estimate

```estimate
# refined estimate pending plan approval
```

## Plan

- [ ] Reproduce: `pair muse` → draft vs agent-pane key probes; capture `wrapcmd` proxy decisions (`plainCR`/`altCR` bytes, `museComposerActive` verdict, `pickerActive`/overlay detector) on the failing paths.
- [ ] Trace `nvim/draft_send.lua:17 send-keys 'Alt Enter'` → `wrapcmd` proxy → harness: verify translation to bare `CR` for Muse and whether composer gate declines.
- [ ] Fix shared seam (keymap / recognizer / overlay) so Muse satisfies the convention; keep fix generic (no nvim-only shim).
- [ ] Add tests: `muse_return_test.go` draft-submit case + agent-pane `Return`/`Alt+Return` matrix; frozen fixture if composer shape changed.
- [ ] Manual smoke: `pair muse` Alt+Return from draft + agent-pane Return/Alt-Return, plus overlay case.

## Log

### 2026-09-15

- Recorded from operator report: `pair muse` — `Alt+Return` from draft stays in composer (not sent); agent-pane `Return` should be Send per harness convention. Created as `000266`.
