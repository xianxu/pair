---
id: '000007'
status: done
created: 2026-05-02
updated: 2026-05-02
actual_hours: N/A
---

# drop the variant positional arg

## Problem

`bin/pair` accepts a second positional arg `<variant>` (e.g., `pair claude work` → session `pair-claude-work`). With the create-flow naming prompt (#000002), this is now redundant — the user can name the session anything at the prompt. Variant only biases the default name shown in the prompt, which is a marginal save over typing one word.

User feedback: "I don't think we need it ... we should remove this option."
