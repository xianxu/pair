---
gate: boundary-review
issue: 311
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-23T23:45:11-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Active-tab clicks still perform a full selection
          detail: '`clickStrip` routes active-chip clicks through `switchTab` (`cmd/internal/termcmd/presentation.go:238-270`), which calls `Presenter.Select` and `renamePane`; `Presenter.Select` cancels drag state (`cmd/internal/terminal/presenter.go:401`). Make active-chip clicks consumed but return without selecting or retitling, and add a regression test proving no selection effects occur.'
          family: active-click-must-be-noop
          round: 1
        - id: BR-2
          severity: Important
          title: README does not document clickable right-terminal tabs
          detail: The Layout 3 README text describes the tab strip and keyboard shortcuts (`README.md:13-28, 127-133`) but not that visible chips are clickable, including shell-tab mouse behavior. Add the user-facing interaction and pass-through/empty-space behavior.
          family: user-facing-behavior-docs
          round: 1
      recipe: small-diff-review
      blocked: true
    - "n": 2
      timestamp: "2026-09-23T23:55:05-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: '`TestPresentationActiveStripClickHasNoSelectionEffects` verifies active clicks cause no repaint or pane retitle; `clickStrip` rejects the active index before selection.'
          round: 2
        - id: BR-2
          disposition: addressed
          note: README.md now documents clickable labels, shell-tab reporting, active/separator/empty-space behavior, and child routing.
          round: 2
      recipe: small-diff-review
      blocked: false
---

# Gate ledger — pair#311 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-23T23:45:11-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `active-click-must-be-noop` Active-tab clicks still perform a full selection
  `clickStrip` routes active-chip clicks through `switchTab` (`cmd/internal/termcmd/presentation.go:238-270`), which calls `Presenter.Select` and `renamePane`; `Presenter.Select` cancels drag state (`cmd/internal/terminal/presenter.go:401`). Make active-chip clicks consumed but return without selecting or retitling, and add a regression test proving no selection effects occur.
- **BR-2** [Important] `user-facing-behavior-docs` README does not document clickable right-terminal tabs
  The Layout 3 README text describes the tab strip and keyboard shortcuts (`README.md:13-28, 127-133`) but not that visible chips are clickable, including shell-tab mouse behavior. Add the user-facing interaction and pass-through/empty-space behavior.

## Round 2 — 2026-09-23T23:55:05-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — `TestPresentationActiveStripClickHasNoSelectionEffects` verifies active clicks cause no repaint or pane retitle; `clickStrip` rejects the active index before selection.
- BR-2 — addressed — README.md now documents clickable labels, shell-tab reporting, active/separator/empty-space behavior, and child routing.

## Open findings

(none — every finding has been disposed)
