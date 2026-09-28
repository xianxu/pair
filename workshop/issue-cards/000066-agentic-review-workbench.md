---
id: '000066'
status: done
started: 2026-06-21T19:33:22-07:00
created: 2026-06-18
updated: 2026-06-22
estimate_hours: 30
actual_hours: 12.87
---

# Agentic memory-backed review as a document workbench in pair

## Problem

parley's `review` is one-shot and amnesiac: it assembles a prompt, forces a single
`propose_edits` tool call, and stops. No agentic loop, no transcript, no cross-repo
memory discovery — the review tip floats over a corpus (brain, pensives, the repos)
it can't reach. Meanwhile pair already runs the structure that fixes this: a
persistent `claude --resume` session paired with nvim document surfaces. So the goal
is to make review a memory-backed agentic loop **hosted in pair**, with an embedded
nvim review pane as the document workbench — not to build a new harness.

Design rationale lives in `workshop/pensive/2026-06-18-01-pensive-agentic-review-workbench.md`.
