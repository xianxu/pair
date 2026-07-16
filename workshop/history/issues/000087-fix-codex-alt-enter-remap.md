---
id: 000087
status: done
deps: []
github_issue:
created: 2026-06-29
updated: 2026-06-29
estimate_hours: 0.46
started: 2026-06-29T18:25:25-07:00
actual_hours: 0.13
---

# Fix Codex Alt Enter remap

## Problem

After #86, restarted sessions do invoke `zellij action send-keys "Alt Enter"` from the draft pane, but Codex still leaves the inserted draft text sitting in the composer. Live trace for `PAIR_TAG=2` at 2026-06-29T18:23:10-07:00 shows:

- nvim wrote the body via `draft.send.write-body` (`body_len: 41`).
- nvim invoked `draft.send.submit` as `zellij action send-keys Alt Enter`.
- pair-wrap read the body bytes, then read Alt+Enter as `ESC CR` (`raw_len: 2`, SHA `a6d286d70768`), translated it to bare `CR` (`translated_len: 1`, SHA `9d1e0e2d9459`), and wrote that to the Codex PTY.

So the Zellij action and pair-wrap Alt+Enter recognition work; the stale assumption is the Codex keymap row that collapses Alt+Enter to bare `CR`. Current Codex needs the modified Alt+Enter chord forwarded to the child PTY.

## Spec

For Codex only, pair-wrap should preserve the submit chord when it sees user Alt+Enter: legacy `ESC CR` and KKP `ESC [ 13 ; 3 u` should translate to `ESC CR` for the Codex child PTY, not bare `CR`. Plain Enter should continue translating to LF so the pair convention remains Enter = newline. Claude and agy keymaps should remain unchanged.

Docs that describe Codex's keymap should say Codex plain Enter maps to newline and Alt+Enter is forwarded as the submit chord.

## Done when

- [x] Codex Alt+Enter translation emits `ESC CR`.
- [x] Codex plain Enter still emits LF.
- [x] Claude and agy keymaps are unchanged.
- [x] Tests cover both legacy and KKP Alt+Enter inputs for Codex.
- [x] Atlas/docs no longer claim Codex Alt+Enter maps to bare `CR`.

## Plan

- [x] Update the pure keymap tests first to expect Codex `altCR == ESC CR` and to expect both legacy and KKP Codex Alt+Enter inputs to translate to `ESC CR`.
- [x] Change only `sendKeymapByAgent["codex"].altCR` in `cmd/pair-wrap/main.go`.
- [x] Update Codex keymap docs in `atlas/architecture.md`, `atlas/how-to-bring-up-a-new-harness-cli.md`, and comments/tests that state the old bare-CR contract.
- [x] Verify `go test ./cmd/pair-wrap`, focused nvim submit tests, issue validation, and whitespace.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.10 impl=0.12
item: atlas-docs design=0.00 impl=0.08
item: milestone-review design=0.00 impl=0.15
design-buffer: 0.10
total: 0.46
```

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.

## Log

### 2026-06-29
- 2026-06-29: closed — go test ./cmd/pair-wrap; bash tests/queue-send-test.sh; bash tests/review-poke-test.sh; git diff --check; sdlc issue validate workshop/issues/000087-fix-codex-alt-enter-remap.md; review verdict: SHIP

User restarted after #86 and still saw Alt+Return insert text into Codex without submitting. Investigated live traces instead of guessing. `zellij-actions-2.jsonl` confirmed the restarted draft pane is using `send-keys Alt Enter`; `wrap-events-2.jsonl` confirmed pair-wrap receives that as Alt+Enter and currently translates it to bare `CR`. Because bare `CR` reached Codex and did not submit, the stale contract is the Codex keymap's `altCR` output, not nvim focus, Zellij delivery, or pair-wrap input recognition (`ARCH-PURPOSE`).

Implemented the Codex keymap correction with TDD. RED: `go test ./cmd/pair-wrap` failed on `codex.altCR: got "\r", want "\x1b\r"` and on both legacy/KKP Codex Alt+Enter translation cases. GREEN: changed only `sendKeymapByAgent["codex"].altCR` to `ESC CR`; `go test ./cmd/pair-wrap` passed. Synced atlas/docs to distinguish Codex from agy and verified with `go test ./cmd/pair-wrap`, `bash tests/queue-send-test.sh`, `bash tests/review-poke-test.sh`, `git diff --check`, and `sdlc issue validate workshop/issues/000087-fix-codex-alt-enter-remap.md`.
