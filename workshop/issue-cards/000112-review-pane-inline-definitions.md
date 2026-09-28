---
id: '000112'
status: done
started: 2026-07-08T14:24:21-07:00
created: 2026-07-08
updated: 2026-07-08
estimate_hours: 3.30
actual_hours: 1.72
---

# review pane inline definitions

## Problem

Pair's review pane is a good place to read and revise unfamiliar documents, but
there is no lightweight way to ask "what does this selected phrase mean here?"
without leaving the document flow. Parley solved the same user need in
parley.nvim#161 as an inline definition, then improved it in parley.nvim#166 and
parley.nvim#167 by persisting definitions as markdown footnotes and highlighting
only the selected term/reference span.

Pair should bring that behavior to the review workbench, adapted to pair's
architecture: the review pane should not embed its own LLM client. It should use
the existing pair agent via the same poke/file-seam style that review handoffs
already use.
