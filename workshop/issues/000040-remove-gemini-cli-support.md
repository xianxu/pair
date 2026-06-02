---
id: 000040
status: working
deps: []
github_issue:
created: 2026-06-01
updated: 2026-06-01
estimate_hours: 1.5
---

# Remove gemini CLI support

## Problem

The old standalone `gemini` CLI is deprecated in favor of `agy` (Antigravity). We need to remove all deprecated `gemini` CLI integration logic, session watchers, keymap overrides, parsers, and tests from the codebase, while keeping `agy` fully intact.

## Spec

Complete the following removals:
1. **Launcher (`bin/pair`)**: Remove `gemini` from help text, rename-suffixes list, recovery flags, and explicit resume check blocks.
2. **Session Watcher (`bin/pair-session-watch.sh`)**: Remove `gemini` watch directories (`~/.gemini/tmp`), file matcher patterns (`session-*.json`), and watcher logic. Ensure `agy` watcher is NOT touched.
3. **Cmux Title Wrapper (`bin/pair-cmux-title.sh`)**: Remove `gemini` case from the title resolver.
4. **PTY Proxy (`cmd/pair-wrap/`)**: Remove the `gemini` row from the keymap overrides, remove `TestTranslateChunk_GeminiKeymap`, and clean up any references.
5. **Slug Summarizer (`cmd/pair-slug/`)**: Remove `gemini` case and the `geminiFile` parser from the Go transcript summarizer logic, and remove its test cases in `slug_test.go`.
6. **Neovim Configs (`nvim/init.lua`, `nvim/scrollback.lua`)**: Remove `gemini` search patterns (`^ >`) and configuration logic.
7. **Scrollback Test Suite (`nvim/scrollback_test.lua`)**: Remove `gemini` search pattern test cases.
8. **Documentation**: Tidy up general mentions of the deprecated gemini agent where appropriate.

## Done when

- [ ] All `gemini` CLI mentions and logic blocks are removed from `bin/pair`, `bin/pair-session-watch.sh`, and `bin/pair-cmux-title.sh`.
- [ ] PTY proxy `cmd/pair-wrap/` has `gemini` removed from keymaps and tests.
- [ ] Transcript summarizer `cmd/pair-slug/` has `gemini` parser and tests deleted.
- [ ] Neovim Lua files (`nvim/init.lua`, `nvim/scrollback.lua`) and tests (`scrollback_test.lua`) have `gemini` patterns and tests removed.
- [ ] Main test suite (`make test`) is fully green.

## Plan

- [ ] Remove `gemini` from `bin/pair` launcher code.
- [ ] Remove `gemini` from `bin/pair-session-watch.sh` and `bin/pair-cmux-title.sh`.
- [ ] Remove `gemini` keymap overrides and test cases from `cmd/pair-wrap/`.
- [ ] Remove `gemini` transcript parser and tests from `cmd/pair-slug/`.
- [ ] Remove `gemini` from `nvim/init.lua`, `nvim/scrollback.lua`, and `nvim/scrollback_test.lua`.
- [ ] Run `make test` to verify the codebase is healthy and fully green.

## Log

### 2026-06-01

- Created issue ticket for gemini CLI removal.

