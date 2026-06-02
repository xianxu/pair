---
id: 000040
status: done
deps: []
github_issue:
created: 2026-06-01
updated: 2026-06-01
estimate_hours: 2.0
actual_hours: 1.5
---

# Remove gemini CLI support

## Problem

The old standalone `gemini` CLI is deprecated in favor of `agy` (Antigravity). We need to remove all deprecated `gemini` CLI integration logic, session watchers, keymap overrides, parsers, and tests from the codebase, while keeping `agy` fully intact. Because `agy` shares the `~/.gemini/` home namespace (under `~/.gemini/antigravity-cli/`), any deletion sweeps must be highly selective to prevent breaking the `agy` agent.

## Spec

Complete the following removals:
1. **Launcher (`bin/pair`)**: Remove `gemini` from help text, rename-suffixes list, recovery flags, and explicit resume check blocks.
2. **Session Watcher (`bin/pair-session-watch.sh`)**: Remove `gemini` watch directories (`~/.gemini/tmp`), file matcher patterns (`session-*.json`), and watcher logic. Ensure `agy` watcher is NOT touched.
3. **Cmux Title Wrapper (`bin/pair-cmux-title.sh`)**: Remove `gemini` case from the title resolver.
4. **PTY Proxy (`cmd/pair-wrap/`)**:
   - Remove `gemini` keymap overrides in `main.go`.
   - Remove `TestTranslateChunk_GeminiKeymap` in `keymap_registry_test.go`.
   - Update `picker_overlay_test.go` line 37 to replace `"gemini"` with `"agy"`.
5. **Slug Summarizer (`cmd/pair-slug/`)**: Remove `gemini` case and the `geminiFile` parser from the Go transcript summarizer logic, and remove its test cases in `slug_test.go`.
6. **Neovim Configs (`nvim/init.lua`, `nvim/scrollback.lua`)**: Remove `gemini` search patterns (`^ >`) and configuration logic.
7. **Scrollback Test Suite (`nvim/scrollback_test.lua`)**: Remove `gemini` search pattern test cases.
8. **Rename Integration Test (`tests/pair-rename.sh`)**: Update test `T8` (line 184-192) to replace `gemini` with `agy` for multi-agent rename coverage.
9. **Doc Comments (`bin/pair-scrollback-open`, `zellij/layouts/main.kdl`)**: Tidy up comments referencing `gemini`.
10. **Atlas Docs (`atlas/architecture.md`, `atlas/how-to-bring-up-a-new-harness-cli.md`, `atlas/index.md`)**: Rewrite or delete lines mentioning the deprecated `gemini` agent.

## Done when

- [x] Guide `atlas/how-to-bring-up-a-new-harness-cli.md` is updated to replace gemini references.
- [x] All `gemini` CLI mentions and logic blocks are removed from `bin/pair`, `bin/pair-session-watch.sh`, `bin/pair-scrollback-open`, and `bin/pair-cmux-title.sh`.
- [x] PTY proxy `cmd/pair-wrap/` has `gemini` removed from keymaps and test files (`keymap_registry_test.go`, `picker_overlay_test.go`).
- [x] Transcript summarizer `cmd/pair-slug/` has `gemini` parser and tests (`slug_test.go`) deleted.
- [x] Neovim Lua files (`nvim/init.lua`, `nvim/scrollback.lua`) and tests (`scrollback_test.lua`) have `gemini` patterns and tests removed.
- [x] Rename integration test (`tests/pair-rename.sh`) is refactored to use `agy` instead of `gemini`.
- [x] Zellij layout (`zellij/layouts/main.kdl`) doc comments are cleaned up.
- [x] Main test suite (`make test`) is fully green.
- [x] **Agy-Safety Verification**: Explicitly verify that `agy` session watcher and slug generation continue to function perfectly on the `~/.gemini/antigravity-cli/` path.

## Plan

- [x] **Launcher, Watcher, Cmux**: Remove `gemini` from `bin/pair`, `bin/pair-session-watch.sh`, `bin/pair-cmux-title.sh`, and `bin/pair-scrollback-open`.
- [x] **Zellij**: Remove `gemini` comments from `zellij/layouts/main.kdl`.
- [x] **PTY Wrapper**:
  - [x] Remove `gemini` from `sendKeymapByAgent` in `cmd/pair-wrap/main.go`.
  - [x] Delete `TestTranslateChunk_GeminiKeymap` from `cmd/pair-wrap/keymap_registry_test.go`.
  - [x] Replace `gemini` with `agy` in `cmd/pair-wrap/picker_overlay_test.go at TestCheckOverlayOpen_AgentsWithoutDetectorSkipped`.
- [x] **Slug**:
  - [x] Remove `geminiFile` struct and case from `cmd/pair-slug/slug.go`.
  - [x] Remove `gemini` case from `cmd/pair-slug/main.go`.
  - [x] Delete `gemini` tests from `cmd/pair-slug/slug_test.go`.
- [x] **Neovim & Scrollback**:
  - [x] Remove `gemini` pattern `^ >` from `nvim/scrollback.lua`.
  - [x] Delete `gemini` test block from `nvim/scrollback_test.lua`.
  - [x] Clean up `agent == 'gemini'` block in `nvim/init.lua`.
- [x] **Integration Tests**:
  - [x] Replace `gemini` config and tests with `agy` in `tests/pair-rename.sh` at `T8` test case.
- [x] **Atlas**:
  - [x] Clean up `gemini` mentions in `atlas/architecture.md`, `atlas/how-to-bring-up-a-new-harness-cli.md`, and `atlas/index.md`.
- [x] **Verification suite**:
  - [x] Run Go and Lua unit tests (`make test`) to ensure codebase tests are green.
  - [x] Run the rename integration test (`make test-queue` or `bash tests/pair-rename.sh`) and confirm it passes.
- [x] **Agy-Safety Check**: Assert that running `agy` launcher and slug creation works completely fine to guarantee the `~/.gemini/antigravity-cli/` path was preserved intact.

## Log


- 2026-06-01: closed — make test and tests/pair-rename.sh successfully executed and all tests passed perfectly
### 2026-06-01

- Removed all gemini CLI references and configurations.
- Verified that all unit and integration tests are fully green.
- Updated docs (README and atlas).
