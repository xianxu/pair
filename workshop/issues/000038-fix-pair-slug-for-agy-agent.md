---
id: 000038
status: working
deps: []
github_issue:
created: 2026-06-01
updated: 2026-06-01
estimate_hours: 2
---

# Fix pair-slug for agy agent

## Problem

`pair-slug` is used to generate a slug describing what a session is about. Each coding agent should call its own smaller model to generate the slug. Currently, `pair-slug` does not support the `agy` (Antigravity) agent, resulting in `pair-slug` not proposing slugs when running `pair agy`. Additionally, when running the `agy` agent, calling its own model should bypass tool execution/agentic workspace exploration by setting the working directory to a temporary directory.

## Spec

1. **Transcript Resolution**: Add a case for `agy` to `resolveTranscript` in `cmd/pair-slug/main.go` pointing to:
   `~/.gemini/antigravity-cli/brain/<session_id>/.system_generated/logs/transcript.jsonl`
2. **Transcript Parsing**: Add a `parseAgy` function to `cmd/pair-slug/slug.go` that parses the `transcript.jsonl` JSONL format. Extract user turns (from `USER_INPUT` entries, parsing out `<USER_REQUEST>` text) and assistant turns (from `PLANNER_RESPONSE` entries).
3. **Model Execution**: Add `runAgyModel` to `cmd/pair-slug/main.go` that executes `agy -p <prompt>` with the transcript context fed on stdin. To prevent `agy` from performing agentic workspace tool exploration, set the execution directory `Dir` of `exec.Command` to a temporary directory (`os.TempDir()`).
4. **Session Watcher**: Update `bin/pair-session-watch.sh` to support the `agy` agent.
   - Watch directory: `$HOME/.gemini/antigravity-cli/brain`
   - Find args: `-type f -name 'transcript.jsonl' -path '*/.system_generated/logs/*'`
   - ID extraction: Extract the UUID folder name from the path.
5. **Launcher Integration**: Update `bin/pair` to support `agy` for `--conversation` flag checks and restart loops, stripping the flag from saved args, and including `agy` in the rename paths list.
6. **Tests**: Add unit test for `parseAgy` in `cmd/pair-slug/slug_test.go`.

## Done when

- `go test ./cmd/pair-slug/...` runs successfully.
- `pair-slug` correctly resolves, parses, and runs the `agy` model.
- `bin/pair-session-watch.sh` correctly parses the `agy` session ID.
- `pair rename` supports `agy` suffixes.

## Plan

- [x] Edit `cmd/pair-slug/slug.go` to add `agy` transcript parsing.
- [x] Edit `cmd/pair-slug/main.go` to add transcript resolution and non-interactive, isolated model execution for `agy`.
- [x] Edit `cmd/pair-slug/slug_test.go` to add test cases for `agy` transcript parsing.
- [x] Edit `bin/pair-session-watch.sh` to configure session watching for `agy`.
- [x] Edit `bin/pair` to configure flags, resume handling, and renaming for `agy`.
- [x] Run `go test` and verify all tests pass.
- [x] Close the issue via `sdlc close`.

## Log

### 2026-06-01

Created issue.

### 2026-06-02

Completed all implementations and verified with both Go and Neovim/Lua test suites:
- Registered `agy` transcript parsing (from JSONL `transcript.jsonl` under `~/.gemini/antigravity-cli/brain`).
- Resolved transcripts of `agy` sessions and ran `agy` CLI in print mode in an isolated `os.TempDir()` directory to completely bypass agentic tool explore hooks.
- Configured `bin/pair-session-watch.sh` to discover and monitor the `agy` session ID.
- Fully integrated `agy` flags, resume actions, and renames in `bin/pair`.
- Staged, committed, and compiled successfully. All tests are green.
