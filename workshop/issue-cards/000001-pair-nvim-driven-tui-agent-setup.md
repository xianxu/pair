---
id: '000001'
status: done
created: 2026-05-02
updated: 2026-05-02
actual_hours: N/A
---

# pair — nvim-driven TUI coding agent setup

## Problem

The input box in every TUI coding agent (Claude Code, Codex, Gemini CLI) is cramped, lacks editing power (no real undo, no search/replace, no snippets, no syntax highlighting), and conflates the input affordance with the output affordance in the same scrollable region. Composing non-trivial prompts there is friction; reviewing output while drafting the next message is friction. The TUI vendors will not fix this — it's not their layer to fix.
