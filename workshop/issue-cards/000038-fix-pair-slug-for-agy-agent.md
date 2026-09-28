---
id: '000038'
status: done
created: 2026-06-01
updated: 2026-06-01
estimate_hours: 2
actual_hours: 1.5
---

# Fix pair-slug for agy agent

## Problem

`pair-slug` is used to generate a slug describing what a session is about. Each coding agent should call its own smaller model to generate the slug. Currently, `pair-slug` does not support the `agy` (Antigravity) agent, resulting in `pair-slug` not proposing slugs when running `pair agy`. Additionally, when running the `agy` agent, calling its own model should bypass tool execution/agentic workspace exploration by setting the working directory to a temporary directory.
