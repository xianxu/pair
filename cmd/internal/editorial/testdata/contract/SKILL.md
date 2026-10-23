---
name: xx-universal-review
description: Use when a message reads `submitted: PATH` (or `review request: PATH`) for a markdown document — the operator's Alt+Return in parley — or when asked to review a document using its editorial settings, including legacy `process-instruction:`.
---

# xx-universal-review — answer a document review request

The request carries only the document's path. Everything else comes from the
document and the files beside this one, so read them each round rather than
from memory:

1. **The round:** read [round.md](round.md) and follow it. It reads the
   document's `processing-mode:` and `processing-instruction:` (and legacy aliases), resolves the modes in [modes/](modes/),
   and says how to treat the 🤖 markers. Its §4 routes each ask by the
   section's collaboration mode and independent processing/presentation settings, from
   [collaboration-modes.md](collaboration-modes.md): read that too.
2. **The grammar:** [review-convention.md](review-convention.md) is the marker
   grammar. Read it before writing any marker.
3. **The adapter:** you edit the file yourself, so follow
   [adapters/file-edit.md](adapters/file-edit.md). For a Couch-tracked
   document it goes through the operator's lock and a compare-and-swap apply;
   never edit one directly.

`export.json` lists these files with their hashes, for parley's standalone
copy; it is generated, so don't edit it by hand.
