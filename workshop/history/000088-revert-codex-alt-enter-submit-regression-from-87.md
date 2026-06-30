---
id: 000088
status: done
deps: []
github_issue:
created: 2026-06-29
updated: 2026-06-29
estimate_hours:
started: 2026-06-29T18:54:30-07:00
actual_hours: N/A
---

# Revert codex Alt Enter submit regression from 87

## Problem

After #87, Codex submit is broken for every Return modifier — the user can no
longer submit at all from the Codex pane. #87 (authored by Codex) changed
`sendKeymapByAgent["codex"].altCR` from a bare `\r` to `\x1b\r` (ESC CR),
theorizing Codex needed the "modified submit chord." That diagnosis was wrong:

- Codex's input parser reads a lone `\r` as the **Enter** key. #31/#34 proved
  this — overlay-active emits a bare `\r` and Codex confirms pickers with it.
- `\x1b\r` parses as **Alt+Enter** (ESC = Alt prefix), which Codex does **not**
  bind to submit. So after #87 Alt+Enter inserts nothing / no-ops and the
  composer text just sits there.
- `altCR: \r` was the stable mapping from 2026-05-10 → 2026-06-29 (7 weeks of
  working Codex submit) and still matches `agy`'s convention.

#87 misattributed the **draft-pane** submit failure (#86's domain — nvim writes
body then sends a separate `send-keys "Alt Enter"`) to the byte value, and
"fixed" it by corrupting the universal submit byte, breaking direct Alt+Enter
submit too.

## Spec

Revert #87's behavioral change: Codex `altCR` back to bare `\r`. Plain Enter
stays LF (newline). Legacy `\x1b\r` and KKP `\x1b[13;3u` Alt+Enter both
translate to `\r` for the Codex child PTY. Restore the atlas/doc text #87 edited.
agy is unchanged (already `\r`).

## Done when

- [x] Codex `altCR == \r`; both legacy and KKP Alt+Enter translate to `\r`.
- [x] Codex plain Enter still emits LF; claude/agy keymaps unchanged.
- [x] Tests revert to expect `\r` and pass.
- [x] Atlas/docs no longer claim Codex Alt+Enter is `ESC CR`.
- [x] Live verify: restart Codex pane on the rebuilt binary; Alt+Enter submits.

## Plan

- [x] Revert `sendKeymapByAgent["codex"].altCR` to `\r` in `cmd/pair-wrap/main.go`.
- [x] Revert `keymap_registry_test.go` + `translate_test.go` codex expectations to `\r`.
- [x] Revert codex passages in `atlas/architecture.md` + `atlas/how-to-bring-up-a-new-harness-cli.md`.
- [x] `go test ./cmd/pair-wrap`; `make pair-wrap`; `git diff --check`.
- [x] Live verify with user after Codex-pane restart.

## Log

### 2026-06-29
- 2026-06-29: closed — User reports Claude-corrected Codex Alt+Enter fix works after restart; fresh go test ./cmd/pair-wrap passes, make pair-wrap is current, and git diff --check is clean.; review verdict: not-run

- Investigated live traces (`wrap-events-2.jsonl`, tag 2): user hit KKP Alt+Enter
  (`\x1b[13;3u`, sha 185fa23f768a) → pair-wrap emitted `\x1b\r` (sha a6d286d70768)
  → Codex did not submit. Decoded byte SHAs to confirm.
- Root cause: #87's `altCR: \r → \x1b\r`. `git log -S altCR` shows `\r` stable
  for 7 weeks; #31/#34 prove bare `\r` is Codex's Enter/confirm. ESC CR = Alt+Enter,
  unbound in Codex. (`ARCH-PURPOSE`: fix the layer that's actually wrong.)
- Reverted code + tests + atlas docs. `go test ./cmd/pair-wrap` green; rebuilt
  `bin/pair-wrap`; `git diff --check` clean.
- User reported Claude corrected Codex's mistake and #88 is ready to close.
