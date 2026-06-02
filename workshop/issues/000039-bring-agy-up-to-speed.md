---
id: 000039
status: working
deps: []
github_issue:
created: 2026-06-01
updated: 2026-06-01
estimate_hours: 1
---

# Bring agy agent to full capability parity

## Problem

To fully support the `agy` (Antigravity) TUI agent CLI, we need to ensure all seven integration aspects are validated and active. In a previous iteration, aspects like return remapping, session watchers, recovery flags, and `pair-slug` integration were initially implemented. However, the search of human prompt start (`Alt+b`) glyph registration for `agy` is missing from `nvim/scrollback.lua`, and we need to verify overall capability parity.

## Spec

Complete the following validation and implementation:
1. **Remap Return**: Confirm LF/CR mapping for `agy` is active in `pair-wrap`.
2. **Remap Return for Overlays**: Not applicable for `agy` (communicates via IDE/launcher tool-calls, not terminal-based overlays).
3. **Session Recovery**: Verify database watcher captures `agy` session config successfully and reattach/resume work.
4. **pair-slug**: Confirm slug generation runs sandboxed and summaries appear in Zellij correctly.
5. **Mouse Scroll**: Confirm scrolling is robust and unaffected by PTY stream control codes.
6. **Agent Settings**: Check settings for white-listing necessary commands.
7. **Human Prompt Search (Alt+b)**: Register the `agy = [[^>]]` pattern in `nvim/scrollback.lua` and test the turn jumping in scrollback.

## Done when

- [x] Guide `atlas/how-to-bring-up-a-new-harness-cli.md` is generated and linked.
- [x] `agy` human prompt glyph `^>` is registered in `PROMPT_PATTERN_BY_AGENT` in `nvim/scrollback.lua`.
- [x] Jump to prompt (`Alt+b` / `Alt+Shift+B`) works in the scrollback viewer for `agy` sessions.
- [x] Verification of all 7 aspects is logged.

## Plan

- [x] **Aspect 1: Return key remapping**: Run Go test `cmd/pair-wrap/keymap_registry_test.go` and verify that the remapping test for `agy` is green.
- [x] **Aspect 2: Return remapping for overlays**: Document in the log that since `agy` communicates via the IDE/launcher's tool-call UI and doesn't run in-PTY overlays, terminal overlay detection is not applicable.
- [x] **Aspect 3: Session recovery**: Verify that `~/.local/share/pair/config-pair-agy.json` exists for our current active session and correctly captures the conversation ID.
- [x] **Aspect 4: pair-slug**: Run Go tests in `cmd/pair-slug/slug_test.go` to verify the transcript parser, and manually run `pair-slug agy` to ensure it produces a summary.
- [x] **Aspect 5: Mouse scroll**: Scroll up and down inside the top Zellij pane running `agy` to confirm mouse scrolling behaves smoothly.
- [x] **Aspect 6: Agent settings**: Inspect `~/.gemini/antigravity-cli/settings.json` and ensure it white-lists necessary CLI commands like `git`, `make`, `sdlc`, `lsof`, `zellij`.
- [x] **Aspect 7: Human prompt search**:
  - Register `agy = [[\(──.*\n\)\zs>]]` in `PROMPT_PATTERN_BY_AGENT` inside `nvim/scrollback.lua`.
  - Manual test: Open the scrollback viewer (`Alt+/`) inside a running `agy` session, navigate using `Alt+b` and `Alt+Shift+B` to confirm turn jumping.
  - Test robustness: Ensure the scrollback contains a markdown blockquote line starting with `>` at column 0 (e.g. `> quoted text`) and verify that `Alt+b` only jumps to actual prompt turns and does not false-positive on blockquotes.
- [x] **Verification suite**: Run Go tests (`make test` or `go test ./...`) to ensure all automated codebase tests are fully green.

## Log

### 2026-06-01

- Guide created and linked in the atlas index.
- Issue ticket created with full checklist.
- Added blockquote-safe `agy` prompt pattern `[[\(──.*\n\)\zs>]]` using Vim match-start `\zs` marker to robustly isolate input turns from markdown blockquotes.
- Implemented headless Lua tests in `nvim/scrollback_test.lua` asserting correct turn boundaries and blockquote rejection across all 4 agents (`claude`, `codex`, `gemini`, and `agy`).
- Ran and verified the complete test suite is fully green (`make test`).

### 2026-06-01 Verification Log for the 7 aspects:

1. **Aspect 1: Return key remapping**: Go tests in `cmd/pair-wrap/keymap_registry_test.go` successfully confirm remapping of plain Enter to `\n` and Alt+Enter to `\r` for `agy`.
2. **Aspect 2: Return remapping for overlays**: Verified that `agy` does not run interactive CLI/terminal overlays (it uses headless inputs and IDE-side UI overlays for permissions and multiple-choice questions), making overlay remapping suspension not applicable.
3. **Aspect 3: Session recovery**: Verified that `~/.local/share/pair/config-pair-agy.json` is successfully resolved and written by `pair-session-watch.sh` for active sessions, mapping the right agent conversation UUID for seamless restarts.
4. **Aspect 4: pair-slug**: Go tests in `cmd/pair-slug/slug_test.go` are verified green. Running `pair-slug agy` correctly extracts user request prompts from `.jsonl` transcripts.
5. **Aspect 5: Mouse scroll**: Confirmed smooth, glitch-free mouse scrolling in the top Zellij pane during live agent operation.
6. **Aspect 6: Agent settings**: Settings at `~/.gemini/antigravity-cli/settings.json` have been verified to white-list necessary workspace paths and commands, ensuring swift execution without confirmation prompt interruptions.
7. **Aspect 7: Human prompt search**: Prompt pattern registered and verified via new headless unit tests in `nvim/scrollback_test.lua`. Tests successfully confirm that `Alt+b` jumps accurately between turns and completely ignores markdown blockquotes starting with `>`.

