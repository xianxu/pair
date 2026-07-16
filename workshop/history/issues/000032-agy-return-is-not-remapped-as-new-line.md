---
id: 000032
status: done
deps: []
estimate_hours: 0.5
created: 2026-05-31
updated: 2026-06-01
actual_hours: 0.2
---

# agy: return is not remapped as new line

## Done when

- Hitting `Enter` in the `agy` (Antigravity) terminal remaps to a newline `\n` instead of sending `\r` (unless PAIR_WRAP_REMAP_RETURN=0).
- `sendKeymapByAgent` registry in `cmd/pair-wrap/main.go` has a row for `"agy"`.
- A unit test verifies the keymap registration for `"agy"`.

## Spec

- Modify the `sendKeymapByAgent` map in the file `cmd/pair-wrap/main.go` to explicitly register a keymap for the `"agy"` (Antigravity) agent. The keymap will map a plain carriage return (Enter alone) to `\n` (newline), and Alt+Enter to `\r` (send). This is exactly the same as the `"gemini"` and `"codex"` keymaps.
- Update the unit test in `cmd/pair-wrap/keymap_registry_test.go` named `TestSendKeymapByAgent_RegistrationTable` to expect `"agy"` in the `want` map.
- Run the test suite using `go test -v ./cmd/pair-wrap/...` to verify that the keymap registration is fully correct.

## Plan

- [x] Add `"agy"` registration to `sendKeymapByAgent` in `cmd/pair-wrap/main.go`
- [x] Add `"agy"` registration to `TestSendKeymapByAgent_RegistrationTable` in `cmd/pair-wrap/keymap_registry_test.go`
- [x] Run `go test -v ./cmd/pair-wrap/...` to verify correctness

## Log


- 2026-06-01: closed — Keymap registry unit tests and Go integration tests for cmd/pair-wrap both pass cleanly
### 2026-06-01

- Analyzed codebase and identified missing keymap registration for "agy" in `cmd/pair-wrap/main.go`.
- Designed plan to add `"agy"` keymap with the same behavior as `"gemini"` and `"codex"`.
