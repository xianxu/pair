# pair atlas

`pair` is a small launcher that gives any TUI coding agent (Claude Code, Codex, Antigravity, Muse) a Neovim-backed input field, decoupling the input scroll from the output scroll inside a zellij session.

## Map

- [Architecture](architecture.md) — what the pieces are and how they fit together.
- [Session identity and storage](session-identity.md) — Pair's scoped address claims and exact artifact bindings, Couch's independent ThreadStore authority, the `📁` public zellij name scheme, and legacy flat-data recovery.
- [Go migration inventory](go-migration-inventory.md) — artifact/caller/runtime contract for the staged primary-Go-binary migration.
- [Workflow](workflow/index.md) — issue-based development loop inherited from the ariadne base layer.
- [How-to-bring-up-a-new-harness-cli](how-to-bring-up-a-new-harness-cli.md) — guide on integrating a new agent harness CLI.
- [couch](couch.md) — the session supervisor (`cmd/couch`): one leased namespace, Couch-owned composite durable threads coordinated with Pair-owned address claims, recoverable pre-exec starts, normalized fleet-policy admission, and tty routing.
- [Review workbench](review-workbench.md) — embedded nvim document-review pane (#66): agent proposes edit records, nvim applies them undo-ably + journals rounds via docflow.

## See also

- `probes/` — committed probe drivers that exercise real binaries end to end.
  `make test-smoke` runs **every** directory under `probes/`, so a new probe is
  covered by existing it rather than by remembering to add a line. A probe earns
  a place here when its output is quoted as close evidence; each explains in its
  own header what question it answers, which is where to look rather than in a
  list here that would drift.

- `doctor/README.md` — `pair-doctor`: read the adaptation flight recorder to diagnose harness integration drift (see the bring-up guide §3 for the signal registry). Primary entry is the agent-agnostic `:PairDoctor` nvim command (`nvim/doctor.lua`); the procedure is single-sourced in `doctor/SKILL.md`, optionally registerable as a Claude skill.
- `README.md` (repo root) — install and usage.
- Design pensive (sibling repo): `~/workspace/brain/docs/vision/2026-05-02-01-pensive-nvim-as-input-field-for-tui-coding-agents.md`
