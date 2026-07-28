# Boundary Review — pair#118 (whole-issue close)

| field | value |
|-------|-------|
| issue | 118 — Rename terminal tabs in the pane frame |
| repo | pair |
| issue file | `workshop/issues/000118-rename-terminal-tabs-in-the-pane-frame.md` |
| boundary | whole-issue close |
| window | `1245357..HEAD` |
| reviewer | codex |
| timestamp | 2026-07-27 |
| verdict | SHIP |

## Summary

The final close review accepted the frame-title rename implementation and found
no Critical, Important, or Minor findings. Earlier rework during close addressed
the `Alt+t` stale viewport residue, rename target drift across tab removal,
rename-title preservation during tab lifecycle cleanup, and malformed CSI/SS3
decoder handling.

## Strengths

- `RenameEditor` remains a pure rune/cursor state machine.
- `RenameDecoderState` cleanly separates streaming byte decoding from editor
  transitions.
- Rename sessions carry the target tab ID through begin/refresh/finish.
- `terminalMux` tracks the active rename preview so async tab lifecycle events
  preserve the frame-title editor.
- README and atlas document the new `Alt+r` frame-title behavior.

## Prior Rework Resolved

- `Alt+t` now redraws the new active tab immediately so old-tab viewport content
  is cleared before child output arrives.
- Rename commit/cancel uses the captured tab ID, not the active tab at finish
  time.
- Tab removal during rename preserves the visible rename field and suppresses
  active viewport redraw.
- Malformed SGR-like and unknown CSI/SS3 controls consume through the terminal
  final byte and preserve following printable input.

## Verification

- `go test ./cmd/internal/termcmd -count=1`
- `go test ./... -count=1`
- `make test-lua`
- `bash tests/term-pane-shortcuts-test.sh`
- `bash tests/review-toggle-test.sh`
- `zellij --config-dir zellij setup --check`
- `git diff --check`
- Live temporary `./bin/pair term` smoke verified `Alt+t` new tab clears the
  old-tab marker residue before child output.

## Close Trailers

```text
Review-Verdict: SHIP
Review-Window: 1245357..HEAD
```
