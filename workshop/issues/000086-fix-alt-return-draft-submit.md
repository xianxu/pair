---
id: 000086
status: working
deps: []
github_issue:
created: 2026-06-29
updated: 2026-06-29
estimate_hours: 0.57
started: 2026-06-29T17:33:40-07:00
---

# Fix Alt Return draft submit

## Problem

Alt+Return from the draft nvim pane moves the draft text into the agent pane but no longer submits it. The visible symptom means focus and body insertion still work; the broken boundary is the follow-up submit action.

## Spec

Use zellij's semantic key action for modified-key submit chords instead of synthesizing Alt+Enter with raw bytes. Draft send and review-poke send should both submit through `zellij action send-keys "Alt Enter"` so zellij delivers a single modified key event to pair-wrap. The Alt+Shift+Return "append but do not submit" path should keep sending raw CR (`write 13`) because it intentionally asks pair-wrap for the insert-newline behavior.

Because `send_to_agent` intentionally short-circuits in headless nvim (`has_ui() == false`), expose a small pure draft command builder seam (`_G.PairDraftSendCommands(body, no_submit)`) for tests. The production `send_to_agent` should execute the commands from that builder only when a UI is attached.

Root-cause evidence:

- `nvim/init.lua` currently writes the body with `write-chars`, then submits with `zellij action write 27 13`.
- `nvim/pair_poke.lua` has the same raw-byte submit pattern for review workbench pokes.
- Installed zellij 0.44.3 exposes `zellij action send-keys`, documented as sending modified keys such as `Alt Shift b`. A semantic `Alt Enter` action matches the desired behavior better than byte-level `ESC` + `CR`.

## Done when

- [x] Draft Alt+Return uses `zellij action send-keys "Alt Enter"` for submit.
- [x] Review-poke submit uses `send-keys --pane-id <agent> "Alt Enter"`.
- [x] Alt+Shift+Return remains append-only and unsubmitted.
- [x] Headless tests assert the draft zellij command sequence through a pure test seam.

## Plan

- [x] Add a pure draft command builder seam in `nvim/init.lua` and headless assertions in `tests/queue-send-test.sh`: straight send must fail while draft submit still uses `write 27 13`, and append-only send must continue to include `write 13` with no `send-keys "Alt Enter"`.
- [x] Change `nvim/init.lua` draft submit builder to semantic `send-keys "Alt Enter"` and execute the builder from `send_to_agent`.
- [x] Update `tests/review-poke-test.sh` to expect `send-keys --pane-id <id> "Alt Enter"` and verify it fails before implementation.
- [x] Change `nvim/pair_poke.lua` review submit to semantic `send-keys`.
- [x] Verify `bash tests/queue-send-test.sh`, `bash tests/review-poke-test.sh`, `make test-lua`, and `git diff --check`.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: lua-neovim design=0.15 impl=0.20
item: milestone-review design=0.00 impl=0.20
design-buffer: 0.15
total: 0.57
```

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.

## Log

### 2026-06-29

User reported Alt+Return inserts draft text into the agent pane but leaves it unsubmitted. Investigation traced the failure boundary to raw-byte submit actions (`zellij action write 27 13`) after successful `write-chars`; zellij 0.44.3 exposes semantic `send-keys`, so the fix targets modified-key submit at the zellij action layer (`ARCH-PURPOSE`) without changing queue/history behavior.

Plan-quality review found the first plan under-tested the explicit Alt+Shift+Return preservation requirement. Revised the plan to require a fake-zellij assertion for the append-only path: it must record `action write 13` and no `send-keys "Alt Enter"` (`ARCH-PURPOSE`).

Second plan-quality review found the proposed `queue-send-test.sh` fake-zellij assertion could not observe submit commands because `send_to_agent` returns early when headless. Revised the plan to add a pure `_G.PairDraftSendCommands(body, no_submit)` seam and assert command construction directly (`ARCH-PURE`).

Implemented `_G.PairDraftSendCommands(body, no_submit)` and routed `send_to_agent` through it. Draft submit and review-poke submit now use zellij's semantic `send-keys "Alt Enter"` action; append-only still emits `write 13` and never emits submit. RED/GREEN evidence: `bash tests/queue-send-test.sh` failed with `C submit missing` / `C append missing` before the seam, then passed after implementation; `bash tests/review-poke-test.sh` failed on `_cmds shape` / `no pane-id submit` before the review-poke change, then passed after implementation. Verified final state with `bash tests/queue-send-test.sh`, `bash tests/review-poke-test.sh`, `make test-lua`, and `git diff --check`.
