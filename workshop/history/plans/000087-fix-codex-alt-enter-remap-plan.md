# Fix Codex Alt Enter Remap Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make draft Alt+Return submit in current Codex sessions by forwarding Codex's Alt+Enter submit chord through pair-wrap instead of collapsing it to bare Enter.

**Architecture:** Keep the fix in the existing pure keymap table and translator tests: pair-wrap already owns the user-key to agent-key adaptation, so Codex's changed submit contract belongs there (`ARCH-PURPOSE`). Do not add nvim timing delays or Zellij workarounds because the live trace proves those boundaries delivered the expected event (`ARCH-DRY`, `ARCH-PURE`).

**Tech Stack:** Go pair-wrap translator, existing Go unit tests, Markdown atlas docs.

---

## Core Concepts

### Pure Entities

| Name | Lives in | Status |
|------|----------|--------|
| `sendKeymapByAgent["codex"]` | `cmd/pair-wrap/main.go` | modified |
| `translateChunk` | `cmd/pair-wrap/main.go` | modified contract via keymap data |

- **sendKeymapByAgent["codex"]** — the per-agent byte contract for translating pair's Enter/Alt+Enter convention into the wrapped agent's expected stdin bytes.
  - **Relationships:** 1:1 with agent basename `codex`; consumed by each `proxy` at startup.
  - **DRY rationale:** One table row feeds both legacy and KKP input shapes through the existing translator, avoiding duplicate special cases.
  - **Future extensions:** If Codex exposes a different submit chord later, update this row and the registry tests.

- **translateChunk** — pure byte-stream translator that recognizes plain Enter, legacy Alt+Enter, and KKP Alt+Enter.
  - **Relationships:** N:1 input chunks to output chunks; receives `sendKM` from the proxy.
  - **DRY rationale:** The translator already handles protocol shape; the Codex-specific output should remain data-driven.
  - **Future extensions:** Add new input protocols here only if Zellij/terminals emit a new shape.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| pair-wrap stdin pump | `cmd/pair-wrap/main.go` | unchanged | stdin and child PTY |
| draft nvim submit | `nvim/init.lua` | unchanged | `zellij action send-keys "Alt Enter"` |

- **pair-wrap stdin pump** — reads Zellij-delivered bytes, calls `translateChunk`, and writes translated bytes to the Codex PTY.
  - **Injected into:** `translateChunk` via the `sendKM` field set at startup.
  - **Future extensions:** If live instrumentation needs richer byte labels, add trace metadata without changing translation semantics.

- **draft nvim submit** — sends the modified key event to pair-wrap. It remains unchanged because live trace proves it now emits the expected Alt+Enter input.
  - **Injected into:** Zellij/pair-wrap through the existing action path.
  - **Future extensions:** None for this issue.

## Chunk 1: Codex Submit Chord

### Task 1: Pin the Codex keymap contract

**Files:**
- Modify: `cmd/pair-wrap/keymap_registry_test.go`
- Modify: `cmd/pair-wrap/translate_test.go`

- [x] **Step 1: Write failing tests**

Update `TestSendKeymapByAgent_RegistrationTable` so the Codex row expects:

```go
"codex": {[]byte{'\n'}, []byte{'\x1b', '\r'}, ctrlU},
```

Update the Codex subtest in `TestTranslateChunk` so:

- `[]byte("hi\x1b\r")` translates to `[]byte("hi\x1b\r")`.
- `[]byte("hi\x1b[13;3u")` translates to `[]byte("hi\x1b\r")`.
- plain `[]byte("hi\r")` still translates to `[]byte("hi\n")`.

- [x] **Step 2: Run tests to verify RED**

Run:

```bash
go test ./cmd/pair-wrap
```

Expected: FAIL on Codex `altCR` and Codex Alt+Enter translation, because production still emits bare `CR`.

- [x] **Step 3: Implement minimal keymap change**

In `cmd/pair-wrap/main.go`, change only the Codex row:

```go
"codex": {
  plainCR: []byte{'\n'},
  altCR:   []byte{'\x1b', '\r'},
  altBS:   []byte{0x15},
},
```

Do not change Claude or agy.

- [x] **Step 4: Run tests to verify GREEN**

Run:

```bash
go test ./cmd/pair-wrap
```

Expected: PASS.

### Task 2: Sync documentation

**Files:**
- Modify: `atlas/architecture.md`
- Modify: `atlas/how-to-bring-up-a-new-harness-cli.md`
- Modify: comments in `cmd/pair-wrap/main.go`, `cmd/pair-wrap/keymap_registry_test.go`, `cmd/pair-wrap/translate_test.go`

- [x] **Step 1: Update stale text**

Replace statements that say Codex Alt+Enter maps to bare `CR` with the current contract: plain Enter maps to LF newline; Alt+Enter is forwarded as the modified submit chord (`ESC CR`).

- [x] **Step 2: Verify no stale Codex keymap docs remain**

Run:

```bash
rg -n "codex.*Alt\\+Enter|Codex.*Alt\\+Enter|Alt\\+Enter.*codex|bare `?\\\\r|bare CR|plain Enter" cmd/pair-wrap atlas -S
```

Expected: Any remaining hits should either describe Claude/overlay behavior or the new Codex contract.

### Task 3: Final verification and close

**Files:**
- Modify: `workshop/issues/000087-fix-codex-alt-enter-remap.md`
- Modify: `workshop/plans/000087-fix-codex-alt-enter-remap-plan.md`

- [x] Run:

```bash
go test ./cmd/pair-wrap
bash tests/queue-send-test.sh
bash tests/review-poke-test.sh
git diff --check
sdlc issue validate workshop/issues/000087-fix-codex-alt-enter-remap.md
```

- [x] Mark plan and issue checkboxes, log RED/GREEN evidence, close through `sdlc close`, then ship via `sdlc pr` / `sdlc merge`.
