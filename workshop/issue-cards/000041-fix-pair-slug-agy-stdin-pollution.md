---
id: '000041'
status: done
created: 2026-06-01
updated: 2026-06-01
estimate_hours: 0.5
actual_hours: 0.5
---

# Fix pair-slug agy stdin transcript pollution

## Problem

When the `pair-slug` summarizer runs in the background for `agy` (Antigravity CLI), it invokes `agy -p <prompt>` and pipes the conversation transcript into `agy`'s `stdin` via `cmd.Stdin = strings.NewReader(input)`. 

Because `agy`'s interactive/non-interactive print mode consumes `stdin` as prompts when `stdin` is redirected, the background `agy` process ignores the summary prompt and instead re-executes all the lines in the transcript as new prompts in a fresh database. This fresh database then becomes the most recently modified database under `~/.gemini/antigravity-cli/conversations/`, causing the active `pair` launcher to resume it on the next launch and pollute the session history with duplicate prompts (e.g. `CCC/BBB/HELLO`).
