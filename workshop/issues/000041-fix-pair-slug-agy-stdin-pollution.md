---
id: 000041
status: working
deps: []
github_issue:
created: 2026-06-01
updated: 2026-06-01
estimate_hours: 0.5
---

# Fix pair-slug agy stdin transcript pollution

## Problem

When the `pair-slug` summarizer runs in the background for `agy` (Antigravity CLI), it invokes `agy -p <prompt>` and pipes the conversation transcript into `agy`'s `stdin` via `cmd.Stdin = strings.NewReader(input)`. 

Because `agy`'s interactive/non-interactive print mode consumes `stdin` as prompts when `stdin` is redirected, the background `agy` process ignores the summary prompt and instead re-executes all the lines in the transcript as new prompts in a fresh database. This fresh database then becomes the most recently modified database under `~/.gemini/antigravity-cli/conversations/`, causing the active `pair` launcher to resume it on the next launch and pollute the session history with duplicate prompts (e.g. `CCC/BBB/HELLO`).

## Spec

Remove the `cmd.Stdin` assignment from `runAgyModel` in `cmd/pair-slug/main.go` so that the background `agy` process sees an empty `stdin` (EOF) and correctly executes only the `-p` summary prompt, without reading or executing any transcript lines.

## Done when

- [x] `cmd.Stdin` assignment is removed from `runAgyModel` in `cmd/pair-slug/main.go`.
- [x] Unit and integration tests (`make test`) pass successfully.
- [x] Staged and committed changes on branch `000041-fix-pair-slug-agy-stdin-pollution`.

## Plan

- [x] Modify `cmd/pair-slug/main.go` to remove `cmd.Stdin = strings.NewReader(input)` in `runAgyModel`.
- [x] Run `make test` to ensure it compiles and tests are green.
- [x] Commit the changes and close the issue.

## Log

### 2026-06-01

- Removed `cmd.Stdin = strings.NewReader(input)` from `runAgyModel` in `cmd/pair-slug/main.go`.
- Verified that all unit tests (`make test`) and rename integration tests (`tests/pair-rename.sh`) are green.

