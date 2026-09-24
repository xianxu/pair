# Boundary Review — pair#311 (whole-issue close)

| field | value |
|-------|-------|
| issue | 311 — Click right-pane tab to switch tabs |
| repo | pair |
| issue file | workshop/issues/000311-right-pane-tab-click-switch.md |
| boundary | whole-issue close |
| milestone | — |
| window | 92df14c8c96dcbbac4f36080f4569710fba81800..c09a426b03c34173da2fe4e10f62916395fdeac5 |
| command | sdlc close --issue 311 |
| reviewer | codex |
| timestamp | 2026-09-23T23:45:11-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation correctly reuses renderer-emitted spans, shared tab switching, and the AnyMotion policy. However, active-tab clicks are documented and tested as harmless but still invoke `Presenter.Select` and `renamePane`, which can cancel child mouse gestures and repaint unnecessarily. README documentation for the new click affordance is also missing.

```findings
findings:
  - id: new
    severity: Critical
    family: active-click-must-be-noop
    title: |
      Active-tab clicks still perform a full selection
    detail: |
      `clickStrip` routes active-chip clicks through `switchTab` (`cmd/internal/termcmd/presentation.go:238-270`), which calls `Presenter.Select` and `renamePane`; `Presenter.Select` cancels drag state (`cmd/internal/terminal/presenter.go:401`). Make active-chip clicks consumed but return without selecting or retitling, and add a regression test proving no selection effects occur.
  - id: new
    severity: Important
    family: user-facing-behavior-docs
    title: |
      README does not document clickable right-terminal tabs
    detail: |
      The Layout 3 README text describes the tab strip and keyboard shortcuts (`README.md:13-28, 127-133`) but not that visible chips are clickable, including shell-tab mouse behavior. Add the user-facing interaction and pass-through/empty-space behavior.
```

1. Strengths

- Hit testing uses `RenderedStrip.Spans` and display columns, including wide and clipped names.
- `switchTab` centralizes keyboard and mouse tab selection.
- Parent AnyMotion and child-facing tracking remain separated.
- Atlas documentation was updated with the new architecture and routing flow.
- Targeted Go tests passed for `termcmd`, `terminal`, `couchtty`, and `terminalqualify`.

2. Critical findings

See `active-click-must-be-noop` above. Enumerated family: the active-chip path through `clickStrip` → `switchTab` → `selectLocked`/`Presenter.Select`/`renamePane`.

3. Important findings

See `user-facing-behavior-docs` above.

4. Minor findings

None.

5. Test coverage notes

- Targeted packages pass.
- `git diff --check` passes.
- New span and routing tests cover wide glyphs, clipping, separators, empty space, and mouse-event filtering.
- The repository-wide `go test ./...` run produced no failure output but did not complete within the review window and was terminated.

6. Architectural notes

- ARCH-DRY: pass. Shared AnyMotion policy, shared switch path, and renderer-owned spans avoid duplicate behavior.
- ARCH-PURE: pass. `ColumnToTab` is pure; IO remains in the pump/presenter boundary.
- ARCH-PURPOSE: mostly pass, but the active-click contract is not fully delivered until the unnecessary selection side effect is removed.

7. Plan revision recommendations

None required; the plan already states that active-tab clicks must be no-ops.
