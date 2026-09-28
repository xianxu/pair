---
id: '000134'
status: done
started: 2026-08-14T17:25:00-07:00
created: 2026-08-14
updated: 2026-08-19
estimate_hours: 4
actual_hours: N/A
---

# Muse harness support for pair

## Problem

Pair currently supports `claude`, `codex`, and `agy` as TUI harnesses (see `atlas/how-to-bring-up-a-new-harness-cli.md`). Meta's `muse` (`~/.local/bin/muse`, `Muse Code 0.1.0`) stores sessions at `~/.local/share/muse/sessions/YYYY/MM/DD/<uuid>/session.jsonl`, resumes via `muse resume <uuid>` / `--last`, and runs headless via `muse exec`. It needs the same 7 integration surfaces as the other agents to feel native inside the `zellij`+`nvim` workbench.
