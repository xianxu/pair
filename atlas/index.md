# pair atlas

`pair` is a small launcher that gives any TUI coding agent (Claude Code, Codex, Antigravity) a Neovim-backed input field, decoupling the input scroll from the output scroll inside a zellij session.

## Map

- [Architecture](architecture.md) — what the pieces are and how they fit together.
- [Go migration inventory](go-migration-inventory.md) — artifact/caller/runtime contract for the staged primary-Go-binary migration.
- [Workflow](workflow/index.md) — issue-based development loop inherited from the ariadne base layer.
- [How-to-bring-up-a-new-harness-cli](how-to-bring-up-a-new-harness-cli.md) — guide on integrating a new agent harness CLI.
- [Review workbench](review-workbench.md) — embedded nvim document-review pane (#66): agent proposes edit records, nvim applies them undo-ably + journals rounds via docflow.

## See also

- `doctor/README.md` — `pair-doctor`: read the adaptation flight recorder to diagnose harness integration drift (see the bring-up guide §3 for the signal registry). Primary entry is the agent-agnostic `:PairDoctor` nvim command (`nvim/doctor.lua`); the procedure is single-sourced in `doctor/SKILL.md`, optionally registerable as a Claude skill.
- `README.md` (repo root) — install and usage.
- Design pensive (sibling repo): `~/workspace/brain/docs/vision/2026-05-02-01-pensive-nvim-as-input-field-for-tui-coding-agents.md`
