# Boundary Review — pair#118 (whole-issue close)

| field | value |
|-------|-------|
| issue | 118 — Rename terminal tabs in the pane frame |
| repo | pair |
| issue file | `workshop/issues/000118-rename-terminal-tabs-in-the-pane-frame.md` |
| boundary | whole-issue close |
| window | `1245357..HEAD` |
| reviewer | codex |
| timestamp | 2026-07-27T11:48:57-07:00 |
| verdict | FIX-THEN-SHIP |

## Summary

The boundary review found no Critical issues. It accepted the core implementation:
the pure rename editor/decoder, frame-title rendering, pane-id-targeted
`rename-pane`, live Neovim-child smoke evidence, and docs/atlas updates.

## Finding

- Important: the issue and plan claimed production-stream coverage for
  `edit+Escape+suffix`, but bare Escape cancellation is timeout-driven. The
  reviewed code covered `edit+Enter+suffix`, timer Escape cancel without suffix,
  and decoder-level Escape suffix consumption, but not the production boundary
  where only a later stdin read may resume child forwarding.

## Resolution

- Added `TestPumpStdinRenameEscapeTimeoutThenNextReadForwards` in
  `cmd/internal/termcmd/run_test.go`.
- Clarified the issue Done-when and durable plan: same-read Escape suffix bytes
  are still rename-mode input until the 50ms timer fires; child forwarding resumes
  only on a subsequent read after timeout cancellation.

## Verification

- `go test ./cmd/internal/termcmd ./cmd/internal/workbenchshortcut -count=1`
- `go test ./... -count=1`
- `make test-lua`
- `bash tests/term-pane-shortcuts-test.sh`
- `bash tests/review-toggle-test.sh`
- `make runtimebundle-drift-check`
- `zellij --config-dir zellij setup --check`
- `git diff --check`

## Close Trailers

```text
Review-Verdict: FIX-THEN-SHIP
Review-Window: 1245357..HEAD
```
