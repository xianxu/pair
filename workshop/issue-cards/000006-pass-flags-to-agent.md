---
id: '000006'
status: done
created: 2026-05-02
updated: 2026-05-02
actual_hours: N/A
---

# pass flags to agent

## Problem

`bin/pair` currently exposes only `<agent>` and `<variant>` as positional args. The layout invokes the agent as `exec ${PAIR_AGENT:-claude}` with no way to pass extra flags through. So `pair claude --resume`, `pair claude -- --model haiku-4-5`, `pair codex -- -p "say hi"` — none of those work.
