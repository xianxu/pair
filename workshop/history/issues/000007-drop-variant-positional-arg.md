---
id: 000007
status: done
deps: [000001, 000002]
created: 2026-05-02
updated: 2026-05-02
---

# drop the variant positional arg

## Problem

`bin/pair` accepts a second positional arg `<variant>` (e.g., `pair claude work` → session `pair-claude-work`). With the create-flow naming prompt (#000002), this is now redundant — the user can name the session anything at the prompt. Variant only biases the default name shown in the prompt, which is a marginal save over typing one word.

User feedback: "I don't think we need it ... we should remove this option."

## Spec

- Drop second positional arg parsing in `bin/pair`. AGENT becomes the only positional arg.
- BASE_TAG / PAIR_TAG default to AGENT (no `-${VARIANT}` suffix).
- Remove `pair <agent> <variant>` line from `--help` USAGE.
- Update README Usage section.
- Update atlas/architecture.md.
- Naming prompt still works as today: user can type `claude-work` (or any custom name) at the prompt to get `pair-claude-work` if they want.

If non-interactive naming becomes a real need later, file a separate issue for `-n NAME` flag. Don't conflate.

## Plan

- [x] Remove `VARIANT="${2:-}"` and the `if [ -n "$VARIANT" ]; then ... -${VARIANT} ...` block from `bin/pair`.
- [x] Update `--help` USAGE to drop the variant line.
- [x] Update README Usage section.
- [x] Update atlas/architecture.md.
- [x] `bash -n` clean.

## Log

### 2026-05-02

Filed after user pointed out variant became redundant once #000002 (naming prompt) shipped.
