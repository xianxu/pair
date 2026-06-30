# Fix Alt Return Draft Submit Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Alt+Return from the draft pane submit the text to the agent instead of only inserting it into the agent composer.

**Architecture:** Treat modified key chords as semantic zellij key events, not raw terminal bytes. Keep the change at the existing nvim/zellij integration boundary: `send_to_agent` and `pair_poke` still own delivery, but submit uses `zellij action send-keys ... "Alt Enter"` so zellij emits the intended chord to pair-wrap (`ARCH-PURPOSE`, `ARCH-DRY`).

**Tech Stack:** Lua, headless Neovim tests with fake `zellij`, existing shell test harness.

---

## Core Concepts

### Pure Entities

| Name | Lives in | Status |
|------|----------|--------|
| `draftSendCommands` | `nvim/init.lua` | new |
| `pair_poke._cmds` | `nvim/pair_poke.lua` | modified |

- **draftSendCommands** — pure argv builder for the draft-pane zellij action sequence.
  - **Relationships:** 1:1 with one draft send; consumed by `send_to_agent`; exposed as `_G.PairDraftSendCommands` for headless tests.
  - **DRY rationale:** Keeps submit vs append-only command selection testable without duplicating expectations inside the IO path.
  - **Future extensions:** If another draft action needs the same delivery sequence, widen the builder rather than adding ad hoc zellij commands.

- **pair_poke._cmds** — pure argv builder for review-pane agent pokes.
  - **Relationships:** 1:1 with one review poke; consumed by `pair_poke.send`.
  - **DRY rationale:** The review-poke submit command should use the same semantic zellij action as the draft send path.
  - **Future extensions:** If zellij changes key naming, update the single command builder and matching draft path.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `draft send_to_agent` | `nvim/init.lua` | modified | `zellij action` |
| `review poke send` | `nvim/pair_poke.lua` | modified | `zellij action --pane-id` |
| `queue-send fake zellij assertion` | `tests/queue-send-test.sh` | modified | fake zellij process |
| `review-poke fake zellij assertion` | `tests/review-poke-test.sh` | modified | fake zellij process |

- **draft send_to_agent** — gets commands from `draftSendCommands`, executes them only when a UI is attached, and redacts body writes in the existing trace path. Submit uses semantic `send-keys "Alt Enter"`; `no_submit` remains raw `write 13` because that path intentionally asks pair-wrap for insert-newline.
  - **Injected into:** Existing draft `<M-CR>` and `<S-M-CR>` maps.
  - **Future extensions:** Any other modified-key submit should use `send-keys`.

- **review poke send** — writes directly to the agent pane id and submits using `send-keys --pane-id <id> "Alt Enter"` without changing focus.
  - **Injected into:** Review workbench poke flow.
  - **Future extensions:** Can share a tiny helper if a third nvim integration needs the same submit action.

## Chunk 1: Semantic Submit

### Task 1: Pin draft submit command

**Files:**
- Modify: `tests/queue-send-test.sh`
- Modify: `nvim/init.lua`

- [x] **Step 1: Write failing fake-zellij assertion**

Extend `tests/queue-send-test.sh` so its headless driver calls `_G.PairDraftSendCommands` and records the returned argv:

- straight submit asserts `zellij action send-keys Alt Enter`.
- append-only asserts `zellij action write 13` and asserts no `zellij action send-keys Alt Enter`.

- [x] **Step 2: Run test to verify it fails**

Run: `bash tests/queue-send-test.sh`

Expected: fail because `_G.PairDraftSendCommands` does not exist yet. After adding the seam but before changing submit, the straight-submit assertion should fail on `write 27 13`, while the append-only assertion should pass.

- [x] **Step 3: Change draft submit implementation**

In `nvim/init.lua`, add a pure `draftSendCommands(body, no_submit)` builder that returns labeled command records for focus up, write body, submit/newline, and focus down. Expose it as `_G.PairDraftSendCommands`. Then replace the submit command with:

```lua
PairZellijTrace.action('draft.send.submit', { 'zellij', 'action', 'send-keys', 'Alt Enter' })
```

Leave the `no_submit` newline path as `write 13`. Update `send_to_agent` to execute the builder's command records through `PairZellijTrace.action`.

- [x] **Step 4: Run test to verify it passes**

Run: `bash tests/queue-send-test.sh`

Expected: pass.

### Task 2: Pin review poke submit command

**Files:**
- Modify: `tests/review-poke-test.sh`
- Modify: `nvim/pair_poke.lua`

- [x] **Step 1: Update failing review-poke assertions**

Change the pure `_cmds` assertion and fake-zellij log assertion to expect `send-keys --pane-id 7 "Alt Enter"`.

- [x] **Step 2: Run test to verify it fails**

Run: `bash tests/review-poke-test.sh`

Expected: fail because `pair_poke._cmds` still returns `write --pane-id 7 27 13`.

- [x] **Step 3: Change review poke implementation**

In `nvim/pair_poke.lua`, replace the submit command with:

```lua
{ 'zellij', 'action', 'send-keys', '--pane-id', tostring(agent_id), 'Alt Enter' }
```

- [x] **Step 4: Run test to verify it passes**

Run: `bash tests/review-poke-test.sh`

Expected: pass.

### Task 3: Verify scope

- [x] Run `bash tests/queue-send-test.sh` and confirm both submit and append-only zellij command assertions pass.
- [x] Run `bash tests/review-poke-test.sh`.
- [x] Run `make test-lua`.
- [x] Run `git diff --check`.
- [x] Update #86 checkboxes/log and close with `--no-atlas` if no architectural docs changed.
