# Three-panel Pair Layout Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Support both the original two-pane Pair workbench (default) and an
explicit/recorded three-pane workbench with a user terminal and pane-local
shortcuts.

**Architecture:** Keep two explicit Zellij KDL assets behind a pure
`LayoutMode` selector: unrecorded tags choose layout 2, while explicit or
recorded layout 3 adds the layered user terminal. In layout 3, pane-local
shortcuts live in focused pane processes: `pair wrap` handles agent chords,
draft nvim handles draft chords, and `pair term` handles terminal chords. Reuse
`cmd/internal/zellijpane` for focused/live-pane discovery (`ARCH-DRY`), keep
shortcut and topology decisions pure with thin PTY/filesystem/Zellij shells
(`ARCH-PURE`), and carry the selected topology through every launch lifecycle
consumer (`ARCH-PURPOSE`).

**Tech Stack:** Go 1.x, `github.com/creack/pty`, `golang.org/x/term`, zellij KDL/actions 0.44.3, shell fake integration tests, runtime-bundle generator.

## Revisions

### 2026-07-24 — Reconcile the final local-tab implementation record

The close review read the pre-implementation `HEAD` and surfaced stale plan
rows from the abandoned outer-Zellij-tab design. The delivered terminal uses
Pair-owned local PTY tabs inside `pair term`, as already specified by the
revised issue and later execution-state note. Update the Core Concepts table to
the actual `run.go` runtime seam and `main-3.kdl` terminal asset; layout 2
deliberately has no terminal. The same review found a real stream-framing gap:
terminal input now consumes shortcut and SGR-mouse prefixes while preserving
coalesced payload bytes, with red/green regressions for both shapes.

### 2026-07-24 — Preserve pane identity during width-toggle collapse

Live smoke testing showed that the initial floating-overlay implementation
removed the terminal frame and restored the three-pane shape with
`override-layout`, which could recreate panes and lose the running agent and
Neovim processes. The toggle now keeps the overlay framed and collapses by
embedding the same terminal pane, then uses pane-targeted resize actions until
the existing left and right panes are balanced. No layout is reapplied.

### 2026-07-24 — Preserve the tiled split tree in both directions

The embed-based collapse preserved pane identity but Zellij reinserted the
terminal at a different node in the split tree, leaving Neovim across the
bottom and agent/terminal across the top. Expansion and collapse now both keep
the terminal tiled and resize only its left boundary, targeting 2/3 and 1/2
width respectively. This preserves topology as well as pane identity.

### 2026-07-24 — Reconcile to the closest reachable width

Smoke testing showed that the original 5% acceptance tolerance stopped
collapse one Zellij resize step before the balanced split. Reconciliation now
uses a 1% target tolerance and, when the next discrete resize step is worse,
reverses that step so the pane remains at the closest reachable width.

### 2026-07-24 — Layer the terminal over a tiled filler

Operator testing confirmed that iterative tiled resizing was both slow and
semantically wrong: expansion must cover part of the left stack rather than
shrink it. The base tiled tree now contains the half-width agent/draft stack
plus an inert borderless filler. The terminal remains permanently floating
above the filler and toggles between exact 50% and 67% coordinates with one
`change-floating-pane-coordinates` action.

### 2026-07-24 — Use live filler geometry as the normal-state anchor

The live pane report can mark both the tiled draft and floating terminal as
focused, and percentage rounding placed the terminal one column over the left
frame. Terminal discovery now scans all focused panes for the floating terminal
instead of stopping at the draft. At startup and on collapse, normal geometry
anchors to the filler's reported `pane_x`, preserving the left frame exactly.

### 2026-07-24 — Restart only the supervised agent for context refresh

The user-owned terminal makes whole-session restart an unsafe routine mechanism:
Pair cannot reconstruct arbitrary shell, process, or local-tab state. Rebind
Alt+Shift+N to confirm and signal the stable `pair wrap` supervisor. The wrapper
replaces only its agent child, preserving the Zellij pane and every other pane.
The replacement command reuses user-authored arguments through the launcher's
canonical persistence transform while dropping `resume`, `--resume`,
`--session-id`, and agent-equivalent restoration bindings. Alt+N remains the
explicit whole-workbench reload path.

### 2026-07-24 — Make the pane frame the sole terminal-tab chrome

The in-pane inverse-video tab strip duplicates the floating pane title and
forces SGR mouse reporting across plain shells. Remove the strip, its reserved
PTY row, click ranges, forced mouse modes, and resize redraw. Local tabs remain
named in the Zellij frame and switch through Alt+Left/Alt+Right. Applications
that request mouse input receive it normally; plain shells leave wheel handling
to Zellij, preventing resize-time device/mouse response chunks from leaking as
literal shell input.

### 2026-07-24 — Make two- and three-pane topology selectable

The three-pane workbench must coexist with the original two-pane workbench.
`pair` defaults to layout 2 for a tag with no topology record; `--layout2` and
`--layout3` are Pair-owned flags accepted on either side of the agent name and
before `--`, and are never forwarded to the agent. A tag records its selected
topology in `workbench-layout-<tag>` (distinct from the existing
`layout-mode-<tag>` draft-height diagnostic). An omitted flag reuses that
record; an explicit flag wins. When the selected tag is already live and the
explicit topology conflicts, Pair asks for confirmation and then uses its
existing whole-workbench restart loop to recreate the same tag with the new
layout. This is intentionally destructive to user-terminal state and is never
performed for an implicit request.

This revision applies `ARCH-PURE` by making argument parsing, precedence, asset
selection, and conflict decisions deterministic functions; `ARCH-DRY` by using
one `LayoutMode` model and one tag-scoped record across create, attach, restart,
rename, and cleanup; and `ARCH-PURPOSE` by covering every consumer that can
select or carry workbench topology rather than changing only the initial launch.

---

## Core Concepts

> **Execution state:** Chunks 1–4 and automated steps 1–4 of Chunk 5 were
> implemented and verified before the selectable-topology revision. Their
> checked boxes below are historical evidence, not a current implementation
> prescription; their abandoned outer/inner-Zellij tab experiments are
> superseded by the revision and issue Log recording Pair-owned local terminal
> tabs. Chunk 6 is the only implementation work remaining; its asset split
> promotes the completed three-pane `main.kdl` to `main-3.kdl` and restores
> layout 2 as the default.

### Pure Entities

| Name | Lives in | Status |
|------|----------|--------|
| `PaneRole` | `cmd/internal/workbenchshortcut/shortcut.go` | new |
| `ShortcutAction` | `cmd/internal/workbenchshortcut/shortcut.go` | new |
| `ShortcutDecision` | `cmd/internal/workbenchshortcut/shortcut.go` | new |
| `LastLeftPaneStore` | `cmd/internal/workbenchshortcut/shortcut.go` | new |

- **PaneRole** — classifies a focused pane as left agent, left draft, right terminal, or other.
  - **Relationships:** 1:1 with a focused `zellijpane.Pane`; role is derived from `title`/`terminal_command`, not filesystem cwd.
  - **DRY rationale:** Reuses `cmd/internal/zellijpane.Parse` rather than duplicating list-panes JSON traversal.
  - **Future extensions:** Additional Pair-owned panes can add roles without changing the parser.

- **ShortcutAction** — the workbench action requested by a recognized chord.
  - **Relationships:** N:1 from terminal input sequences to one action; actions map to zellij commands only in the IO shell.
  - **DRY rationale:** One registry for the right-terminal and future pane wrappers.
  - **Future extensions:** Add more pane-local commands without changing PTY byte pumps.

- **ShortcutDecision** — pure decision result: pass bytes through, swallow, or run a zellij action.
  - **Relationships:** Depends on `PaneRole`, current last-left-pane id, and decoded input sequence.
  - **DRY rationale:** Keeps pane gating testable without spawning zellij or shells.
  - **Future extensions:** Persist more navigation state if Pair adds multiple user panes.

- **LastLeftPaneStore** — path contract and atomic read/write helpers for the last focused left Pair pane id.
  - **Relationships:** 1:1 with a Pair tag under `$PAIR_DATA_DIR/last-left-pane-<tag>`. Written by agent/draft local shortcut handlers before moving right; read by `pair term` before moving left.
  - **DRY rationale:** One shared sidecar path prevents `pair wrap`, nvim, and `pair term` from inventing incompatible state keys.
  - **Future extensions:** If Pair tracks more pane focus history, the file can become JSON with versioned fields.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `pair term` | `cmd/internal/termcmd/run.go` | new | PTY child shell + stdin/stdout |
| `TermRuntime` | `cmd/internal/termcmd/run.go` | new | zellij IPC + process exec |
| `pair wrap` shortcut hook | `cmd/internal/wrapcmd/wrap.go` | modified | agent-pane PTY stdin |
| `draft nvim` shortcut maps | `nvim/init.lua` | modified | draft-pane key mappings |
| `main-3.kdl` terminal pane | `zellij/layouts/main-3.kdl` | new | zellij layout |
| `config.kdl` global unbinds | `zellij/config.kdl` | modified | zellij global keybinds |

- **pair term** — transparent wrapper for the right user shell. It starts `$SHELL` (fallback `/bin/sh`), forwards bytes, recognizes only the agreed workbench chords, and leaves all other shell/nvim input untouched.
  - **Injected into:** zellij layout command for the right terminal pane.
  - **Future extensions:** More pane-local helpers can be added to the pure shortcut registry.

- **TermRuntime** — side-effect seam for listing panes, focusing panes, and running zellij tab actions.
  - **Injected into:** `Run` so tests can use fakes for zellij and a fake child command.
  - **Future extensions:** Process-level fake can grow to cover live rename prompt behavior.

- **pair wrap shortcut hook** — recognizes only left-agent workbench chords before forwarding stdin to the agent PTY.
  - **Injected into:** existing `translateStdin` path in `wrapcmd`.
  - **Future extensions:** Any new left-agent shortcut should go through the shared pure shortcut decision layer, not ad hoc byte checks.

- **draft nvim shortcut maps** — local draft-pane mappings for `Alt+j`, `Alt+k`, `Alt+/`, `Alt+Shift+C` / `Ctrl+Alt+c`, and no-op `Alt+t/w/r`.
  - **Injected into:** `nvim/init.lua` alongside existing Pair layout/restart mappings.
  - **Future extensions:** If nvim Lua grows too large, extract a `nvim/workbench.lua` helper.

- **main-3.kdl terminal pane** — layered zellij geometry: a half-width
  agent/draft stack and inert filler form the tiled base; a floating
  `pair term` pane covers the filler at normal width and part of the left stack
  at expanded width. `main-2.kdl` deliberately retains only agent/draft.
  - **Injected into:** launcher `--new-session-with-layout` flow through the
    pure `LayoutMode` asset selector.
  - **Future extensions:** Width percentage can be tuned without changing command wrappers.

- **config.kdl global unbinds** — zellij must release pane-local chords so the focused pane process can decide whether to handle or ignore them.
  - **Injected into:** zellij session config.
  - **Future extensions:** If zellij adds conditional pane binds later, this layer can own gating directly.

---

## Chunk 1: Pure Shortcut Model and Dispatcher Surface

### Task 1: Add `termcmd` pure shortcut decisions

**Files:**
- Create: `cmd/internal/workbenchshortcut/shortcut.go`
- Test: `cmd/internal/workbenchshortcut/shortcut_test.go`

- [x] **Step 1: Write failing tests for role classification**

Cover:
- `pair wrap` command => left agent role.
- `nvim -u .../nvim/init.lua ...draft...` => left draft role.
- `pair term` command or `title == "terminal"` => right terminal role.
- focused floating/plugin panes => other role.

Run: `go test ./cmd/internal/workbenchshortcut -run TestPaneRole -count=1`
Expected: fails because package does not exist.

- [x] **Step 2: Implement role classification**

Use `zellijpane.Pane` as input. Match `terminal_command` for stable commands, with title only as a fallback for the new terminal pane name. Avoid cwd-sensitive title matching.

- [x] **Step 3: Write failing tests for shortcut decisions**

Cover:
- Right terminal: `Alt+t` => new tab, `Alt+w` => close tab, `Alt+r` => rename-tab prompt, `Alt+j` => swallow/no-op, `Alt+k` => focus last left pane or draft fallback.
- Left agent/draft: `Alt+j` toggles vertical focus, `Alt+k` records the focused left pane id and focuses right terminal, `Alt+t/w/r` are swallowed no-ops, `Alt+Shift+C`/`Alt+/` are allowed only left.
- Other/review/floating panes: `Alt+r` is not captured.

Run: `go test ./cmd/internal/workbenchshortcut -run TestShortcutDecision -count=1`
Expected: fails until decisions exist.

- [x] **Step 4: Implement `ShortcutDecision`**

Represent decisions without IO: `Pass`, `Swallow`, `FocusPane(id)`, `MoveFocus(direction)`, `RecordLastLeftAndFocusTerminal`, `FocusLastLeftOrDraft`, `NewTab`, `CloseTab`, `RenameTabPrompt`, `OpenScrollback`, `Compact`. The decision receives current last-left id as input; persistence is owned by `LastLeftPaneStore`.

- [x] **Step 5: Add sidecar path/read-write tests**

Cover:
- `LastLeftPath("/data", "work") == "/data/last-left-pane-work"`;
- atomic write then read returns the pane id;
- empty/missing file returns no id.

Run: `go test ./cmd/internal/workbenchshortcut -run TestLastLeft -count=1`
Expected: PASS after helpers exist.

- [x] **Step 6: Run focused tests**

Run: `go test ./cmd/internal/workbenchshortcut -count=1`
Expected: PASS.

### Task 2: Register the `pair term` command

**Files:**
- Modify: `cmd/internal/dispatcher/dispatcher.go`
- Modify: `cmd/pair-go/main.go`
- Test: `cmd/internal/dispatcher/dispatcher_test.go`

- [x] **Step 1: Add failing dispatcher tests**

Extend `TestDispatchNamesDeriveFromImplementedStatus` and `TestStreamingFlags` to include `term`.

Run: `go test ./cmd/internal/dispatcher -run 'TestDispatchNamesDeriveFromImplementedStatus|TestStreamingFlags' -count=1`
Expected: fails because `term` is not registered.

- [x] **Step 2: Register `term` as an implemented streaming command**

Add `{Name: "term", Summary: "user terminal pane wrapper", Status: "implemented", Streaming: true}` and wire `runStreamingSubcommand("term", ...)` to `termcmd.Run`.

- [x] **Step 3: Run dispatcher tests**

Run: `go test ./cmd/internal/dispatcher ./cmd/pair-go -count=1`
Expected: PASS.

---

## Chunk 2: Terminal Wrapper IO

### Task 3: Implement transparent terminal wrapper

**Files:**
- Create: `cmd/internal/termcmd/run.go`
- Create: `cmd/internal/termcmd/runtime.go`
- Test: `cmd/internal/termcmd/run_test.go`

- [x] **Step 1: Write failing run-level tests with fakes**

Use a fake runtime that records zellij actions and a fake child command. Cover:
- no args starts `$SHELL`, fallback `/bin/sh`;
- unrecognized input bytes pass through to child;
- right-terminal `Alt+t/w/r` records `NewTab`, `CloseTab`, and an internal rename prompt followed by `RenameTab(name)`;
- right-terminal `Alt+j` is swallowed;
- right-terminal `Alt+k` focuses the recorded last-left pane, falling back to draft.
- agent-recorded and draft-recorded last-left ids are visible to a later terminal shortcut through the shared `$PAIR_DATA_DIR/last-left-pane-<tag>` sidecar.

Run: `go test ./cmd/internal/termcmd -run TestRun -count=1`
Expected: fails until runtime exists.

- [x] **Step 2: Implement PTY wrapper**

Mirror the existing `wrapcmd`/`scribecmd` PTY pattern: `pty.Start`, raw stdin when stdin is an `*os.File`, SIGWINCH size propagation, stdin byte pump, stdout byte pump, child exit code propagation. Keep only shortcut decoding in the stdin pump; do not inspect or mutate child stdout.

- [x] **Step 3: Implement zellij runtime actions**

Commands:
- list panes: `zellij action list-panes --json --command`
- focus pane: `zellij action focus-pane-id <id>` then `terminal_<id>` fallback if needed
- new tab: `zellij action new-tab`
- close tab: `zellij action close-tab`
- rename tab: `pair term` temporarily enters an internal line prompt in the terminal pane, then calls `zellij action rename-tab <name>` when the user presses Enter. `Esc` or an empty name cancels and restores normal pass-through. This avoids global KDL `TabNameInput` so review-pane `Alt+r` remains local.

- [x] **Step 4: Run focused tests**

Run: `go test ./cmd/internal/termcmd -count=1`
Expected: PASS.

---

## Chunk 3: Zellij Layout and Key Routing

### Task 4: Update main layout to three panes

**Files:**
- Modify: `zellij/layouts/main.kdl`
- Generated: `cmd/internal/runtimebundle/assets/runtime/files/zellij/layouts/main.kdl`

- [x] **Step 1: Update layout**

Default geometry:
- root `split_direction="vertical"`
- left child `size="50%"` or equivalent containing the existing horizontal agent/draft stack
- right child `name="terminal"` running `sh -c 'exec pair term'`
- keep `focus=true` on draft for startup

Swap layouts:
- update `exact_panes=3`
- preserve right terminal leaf in every swap layout
- only vary draft height inside the left stack

- [x] **Step 2: Validate layout**

Run: `zellij setup --dump-layout zellij/layouts/main.kdl`
Expected: exit 0.

### Task 5: Update config keybinds

**Files:**
- Modify: `zellij/config.kdl`
- Generated: `cmd/internal/runtimebundle/assets/runtime/files/zellij/config.kdl`
- Test: existing review tests plus any new shell test

- [x] **Step 1: Release pane-local chords from global KDL**

Remove global Pair/zellij handling for pane-local chords that must be owned by focused pane processes: `Alt+j`, `Alt+k`, `Alt+t`, `Alt+w`, `Alt+r`, `Alt+/`, and `Alt C` / `Ctrl Alt c`. Keep non-pane-local Pair safety flows such as detach/quit/restart routed through draft nvim if they are still intended to work from every pane.

- [x] **Step 2: Add agent-pane shortcut handling in `pair wrap`**

Intercept recognized left-agent chords before forwarding to the wrapped agent:
- `Alt+j`: focus draft.
- `Alt+k`: record the agent pane id as last-left, then focus right terminal.
- `Alt+/`: open scrollback viewer.
- `Alt C` / `Ctrl Alt c`: focus draft and invoke `PairConfirmCompact()`.

All other bytes pass through exactly as today.

- [x] **Step 3: Add wrapcmd shortcut tests**

Extend `cmd/internal/wrapcmd/translate_test.go` or add a focused test file. Cover:
- recognized workbench chords are intercepted even when `sendKM` is empty, so no-Return-remap agents still get pane-local shortcuts;
- `Alt+t`, `Alt+w`, and `Alt+r` in the agent pane are swallowed no-ops and are not forwarded to the agent;
- ordinary text and existing Return/Alt+Return remaps still pass existing tests;
- split escape handling still holds back partial shortcut sequences just like the existing Alt+Enter handling.

Run: `go test ./cmd/internal/wrapcmd -run 'TestTranslate|TestWorkbenchShortcut' -count=1`
Expected: PASS.

- [x] **Step 4: Add draft-pane shortcut mappings**

In `nvim/init.lua`, map:
- `Alt+j`: focus agent.
- `Alt+k`: record the draft pane id as last-left, then focus right terminal.
- `Alt+/`: open scrollback viewer.
- `Alt+Shift+C` / `Ctrl+Alt+c`: existing compaction confirm.
- `Alt+t`, `Alt+w`, `Alt+r`: no-op in the draft pane, so right-terminal tab helpers do not leak left. Do not change `nvim/review.lua`'s `Alt+r` reject mapping.

- [x] **Step 5: Keep right-terminal tab helpers inside `pair term`**

Ensure `Alt+t`, `Alt+w`, and `Alt+r` are only decoded by `pair term`. The review pane must keep its own `Alt+r` reject behavior because zellij no longer owns a global `Alt+r`.

- [x] **Step 6: Validate config**

Run: `zellij --config-dir zellij setup --check`
Expected: exit 0.

---

## Chunk 4: Tests, Docs, Runtime Bundle

### Task 6: Add integration smoke test for pane gating

**Files:**
- Create: `tests/term-pane-shortcuts-test.sh`
- Modify: `Makefile.local`

- [x] **Step 1: Write fake-zellij shell test**

Use the real `bin/pair` and fake `zellij` output like `tests/copy-on-select-test.sh`. Assert:
- focused terminal + `pair term --test-shortcut Alt+t` records new-tab;
- focused terminal + `Alt+r` records rename prompt path;
- focused review/floating pane + `Alt+r` records no tab rename;
- focused terminal + `Alt+j` records no movement;
- `Alt+k` returns to the last left pane recorded by an agent/draft-side action and falls back to draft when the sidecar is missing.

If direct keystroke driving is awkward, add a test-only `pair term --test-shortcut <name>` path that exercises the same pure decision + runtime action path without starting a PTY.

- [x] **Step 2: Add Make target**

Add `test-term-pane-shortcuts` and include it in `test`.

- [x] **Step 3: Run focused integration test**

Run: `bash tests/term-pane-shortcuts-test.sh`
Expected: PASS.

### Task 7: Update atlas and runtime bundle

**Files:**
- Modify: `atlas/architecture.md`
- Modify: `atlas/index.md` if a new section/link is needed
- Generated: `cmd/internal/runtimebundle/assets/runtime/**`

- [x] **Step 1: Update architecture docs**

Replace the two-pane invariant text with the new three-pane workbench: left Pair stack, right user terminal, pane-local shortcuts, and right-terminal tab helper limits.

- [x] **Step 2: Regenerate runtime bundle assets**

Run: `make runtimebundle-generate`
Expected: generated KDL/runtime files match source changes.

- [x] **Step 3: Runtime bundle drift check**

Run: `make runtimebundle-drift-check`
Expected: PASS.

---

## Chunk 5: Final Verification

### Task 8: Verify build and behavior

**Files:**
- Modify issue checklist/log after verification.

- [x] **Step 1: Run focused Go tests**

Run: `go test ./cmd/internal/workbenchshortcut ./cmd/internal/termcmd ./cmd/internal/wrapcmd ./cmd/internal/dispatcher ./cmd/pair-go ./cmd/internal/zellijpane -count=1`
Expected: PASS.

- [x] **Step 2: Run affected integration tests**

Run: `bash tests/term-pane-shortcuts-test.sh`
Expected: PASS.

Run: `bash tests/copy-on-select-test.sh`
Expected: PASS.

Run: `bash tests/review-toggle-test.sh`
Expected: PASS, including `Alt+r` not globally stolen.

- [x] **Step 3: Validate zellij assets**

Run: `zellij --config-dir zellij setup --check`
Expected: PASS.

Run: `zellij setup --dump-layout zellij/layouts/main.kdl`
Expected: PASS.

- [x] **Step 4: Run broader tests**

Run: `go test ./... -count=1`
Expected: PASS.

Run: `make test-lua`
Expected: PASS.

- [x] **Step 5: Manual smoke — superseded by Task 13 Step 4**

This original single-topology smoke contract is retired by the
2026-07-24 selectable-topology revision. Do not execute the obsolete command or
expectation list below; Task 13 Step 4 is the sole current operator smoke.

- [x] **Step 6: Update issue and close — superseded by Task 13**

Closing is intentionally deferred until Chunk 6 passes automated verification,
the operator completes Task 13 Step 4, and the operator separately authorizes
commit/landing.

---

## Chunk 6: Selectable Workbench Topology

### Core concept addendum

#### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `LayoutMode` | `cmd/internal/launcher/layout.go` | new |
| `LayoutRequest` | `cmd/internal/launcher/layout.go` | new |
| `LayoutResolution` | `cmd/internal/launcher/layout.go` | new |

- **`LayoutMode`** — the closed topology choice `layout2` or `layout3`.
  - **Relationships:** one selected mode per Pair tag; one mode maps to one KDL
    asset.
  - **DRY rationale:** parsing, persistence, restart, and asset selection must
    not each invent strings or defaults.
  - **Future extensions:** a new topology adds one validated value and one asset
    mapping.
- **`LayoutRequest`** — a requested mode plus whether the operator supplied it
  explicitly.
  - **Relationships:** one request per launcher invocation; explicitness is
    preserved independently from the mode so an omitted default cannot be
    mistaken for an override.
  - **DRY rationale:** centralizes the user-intention rule used by both create
    and attach decisions.
- **`LayoutResolution`** — the pure result of explicit request, recorded mode,
  and live state: selected mode plus whether a confirmed restart is required.
  - **Relationships:** consumes one request and zero-or-one recorded mode.
  - **DRY rationale:** one precedence table serves launch, attach, and restart
    tests.

#### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| Workbench layout record | `cmd/internal/launcher/layout.go`, `cmd/internal/launcher/createflow.go` | new | `workbench-layout-<tag>` filesystem sidecar |
| Live layout-change confirmation | `cmd/internal/launcher/runtime.go`, `cmd/internal/launcher/osruntime.go` | new | `/dev/tty` operator prompt |
| Live topology probe | `cmd/internal/launcher/layoutflow.go`, `cmd/internal/launcher/osruntime.go` | new | session-scoped Zellij pane report |
| Layout assets | `zellij/layouts/main-2.kdl`, `zellij/layouts/main-3.kdl` | new | Zellij KDL |

- **Workbench layout record** — thin read/write helpers around the tag-owned
  topology record. Missing or malformed content resolves to layout 2 without
  rewriting until create has passed every preflight and is ready to hand off.
  The record is durable resumable-tag state: ordinary quit and restart preserve
  it; only explicit topology selection, tag rename, or deliberate tag-state
  deletion changes it.
  - **Injected into:** `runOnce`/`runCreate` through the existing `Runtime`
    filesystem seam.
  - **Future extensions:** migration can recognize retired mode spellings in
    the pure decoder.
- **Live layout-change confirmation** — asks before an explicit conflicting
  request destroys and recreates a live session. Decline/EOF leaves the session
  and record untouched.
  - **Injected into:** the attach branch only after pure conflict detection.
  - **Future extensions:** the prompt can enumerate topology-specific state at
    risk without changing the decision model.
- **Live topology probe** — classifies a pre-record rollout session from its
  actual panes: the terminal filler/floating terminal signature is layout 3;
  the original agent/draft pair is layout 2. A process-level fake supplies the
  same JSON as Zellij in integration tests.
  - **Injected into:** attach conflict resolution only when the durable record
    is missing or malformed; recorded sessions never pay for the probe.
  - **Future extensions:** new modes add a pure pane-signature classifier.
- **Layout assets** — preserve the original agent/draft workbench as layout 2
  and the current layered terminal workbench as layout 3.
  - **Injected into:** the existing `LaunchSession(..., layout)` boundary via a
    pure mode-to-path helper.
  - **Future extensions:** structural tests remain the enforcement point for
    shared agent/draft behavior across the two unavoidable static KDL products.

### Task 9: Parse Pair-owned layout flags

**Files:**
- Create: `cmd/internal/launcher/layout.go`
- Create: `cmd/internal/launcher/layout_test.go`
- Modify: `cmd/internal/launcher/args.go`
- Modify: `cmd/internal/launcher/args_test.go`
- Modify: `cmd/internal/launcher/help.go`

- [x] **Step 1: Write failing table tests for layout values and precedence**

Cover:
- no request + no record → layout 2;
- no request + recorded layout 3 → layout 3;
- explicit layout 2 + recorded layout 3 → layout 2 with conflict;
- explicit layout 3 + recorded layout 2 → layout 3 with conflict;
- same explicit and recorded mode → no conflict;
- malformed record → layout 2.

Run: `go test ./cmd/internal/launcher -run 'TestResolveLayout|TestParseLayoutMode' -count=1`
Expected: FAIL because the model does not exist.

- [x] **Step 2: Implement the pure layout model**

Define validated constants, `LayoutRequest{Mode, Explicit}`,
`ResolveLayout(request, recorded)`, record decoding, and the KDL basename
mapping in `layout.go`. Keep all defaulting here.

- [x] **Step 3: Write failing argument-parser permutations**

Cover at minimum:

```text
pair
pair --layout2
pair --layout3 codex
pair codex --layout2
pair claude --layout2 -- --other-claude-flags other-claude-params
pair --layout3 claude -- --layout2
pair resume tag --layout3
pair --layout2 resume tag
```

Assert Pair consumes flags only before `--`, forwards every token after `--`
verbatim, rejects duplicate conflicting Pair layout flags, and preserves the
explicit bit. Reject layout flags on non-launch lifecycle verbs (`list`,
`rename`, `restart`, and `quit`) rather than silently accepting a value those
commands cannot honor; `resume` and `continue <slug>` remain launch forms and
accept them.

Run: `go test ./cmd/internal/launcher -run 'TestParseArgs.*Layout' -count=1`
Expected: FAIL against the current parser.

- [x] **Step 4: Refactor parsing around a pre-separator Pair-option pass**

Consume layout flags anywhere before `--` without treating them as an agent.
Pass the remaining command/agent tokens into the existing verb-specific parsing,
then attach the single parsed `LayoutRequest` to `LaunchArgs`. Do not teach
agent-specific code about Pair layout flags.

- [x] **Step 5: Update native help and run parser tests**

Document defaulting, persistence, explicit override, destructive live-switch
confirmation, and the `--` boundary.

Run: `go test ./cmd/internal/launcher -run 'TestParseArgs|TestUsageText|TestResolveLayout' -count=1`
Expected: PASS.

### Task 10: Persist and select topology on create

**Files:**
- Modify: `cmd/internal/launcher/createflow.go`
- Create: `cmd/internal/launcher/layoutflow.go`
- Create: `cmd/internal/launcher/layoutflow_test.go`
- Modify: `cmd/internal/launcher/createflow_test.go`
- Modify: `cmd/internal/launcher/rename.go`
- Modify: `cmd/internal/launcher/rename_test.go`

- [x] **Step 1: Write failing create-flow tests**

Using the existing fake runtime, assert:
- an unrecorded tag launches `main-2.kdl` and records `layout2`;
- a recorded layout 3 tag launches `main-3.kdl` when no flag is passed;
- explicit layout 2/3 wins and updates the record only as part of create;
- failed preflight/ledger/session-index work does not mutate the layout record;
- record write failure aborts before Zellij launch;
- an immediate `LaunchSession` error restores the prior record bytes (or removes
  a newly created record), while a successful handoff keeps the selection.

Run: `go test ./cmd/internal/launcher -run 'TestRunLaunch.*Layout' -count=1`
Expected: FAIL because create always chooses `main.kdl`.

- [x] **Step 2: Thread one resolved mode through the create boundary**

Read `workbench-layout-<tag>` after the tag is final, resolve it with
`LaunchArgs.Layout`, export `PAIR_WORKBENCH_LAYOUT`, and pass the pure asset path
to `LaunchSession`. After every existing fallible preflight succeeds and
immediately before the blocking handoff, atomically write the selected record.
If that write fails, abort without launching. If `LaunchSession` returns an
immediate invocation error, atomically restore the previous bytes or remove a
previously absent record. A normal blocking handoff return keeps the record,
regardless of the eventual session exit code.

- [x] **Step 3: Add rename and persistence coverage**

Add `workbench-layout-<tag>` to exact-name rename enumeration. Assert that full
quit, Alt+N, compaction, and agent-only restart do not remove or rewrite it, and
that omitted-flag resume after a full quit selects the durable mode. Keep
`layout-mode-<tag>` unchanged because it remains the draft-height rung
diagnostic.

Run: `go test ./cmd/internal/launcher -run 'Test(Rename|RunCleanup|RunLaunch).*Layout' -count=1`
Expected: PASS.

### Task 11: Override a live topology safely

**Files:**
- Modify: `cmd/internal/launcher/runtime.go`
- Modify: `cmd/internal/launcher/osruntime.go`
- Modify: `cmd/internal/launcher/createflow.go`
- Modify: `cmd/internal/launcher/layoutflow.go`
- Modify: `cmd/internal/launcher/layoutflow_test.go`

- [x] **Step 1: Write failing attach conflict tests**

Cover:
- omitted flag attaches without prompting and leaves the record unchanged;
- explicit same mode attaches without prompting;
- a live pre-record layout-2 or layout-3 session is classified from a
  session-scoped pane report before conflict resolution;
- explicit conflicting mode prompts with the tag, old/new modes, and warning
  that terminal state will be lost;
- decline/EOF exits without attach, kill, marker, or record mutation;
- accept performs one nonterminal teardown transition, then the outer launcher
  loop recreates the same tag with the requested asset;
- restart failure does not pre-write a false topology record.

Run: `go test ./cmd/internal/launcher -run 'TestRunLaunch.*LiveLayout' -count=1`
Expected: FAIL because attach ignores topology.

- [x] **Step 2: Add confirmation and live-probe seams**

Add `ConfirmLayoutChange(tag, from, to string) bool` to `UIOps` and its
`/dev/tty` implementation. Add `ProbeLiveLayout(session string)` to the Zellij
runtime seam; invoke
`zellij --session <session> action list-panes --json --command --state`, reuse
`zellijpane.Parse`, and classify its command/floating signature with a pure
helper. Missing/malformed records use this probe only for live sessions; a probe
failure aborts an explicit override rather than guessing.
Change the existing `DeleteSession` effect to return an error: ordinary
post-handoff cleanup may report and continue as today, while a pre-attach
topology switch must refuse to retry when teardown was not confirmed.

- [x] **Step 3: Add a nonterminal relaunch transition**

Extend `launchStep` with a relaunch outcome that the outer `RunLaunch` loop
handles before the `handedOff`/cleanup path. Before `runAttach`, resolve the live
tag's topology. On confirmed conflict:

1. delete the live Zellij session without writing quit/restart markers;
2. re-query sessions and require that the deleted session is absent;
3. reap only the tag's embedded Neovim process and stop its title poller;
4. return `relaunch=true` with the same forced tag and explicit layout request;
5. have the outer loop immediately re-run create for that tag.

This transition preserves draft, config, ledger, layout record, terminal capture,
and other resumable tag state until the new create reaches its atomic layout
record commit point. On ordinary attach, perform no topology IO. If teardown
fails, abort rather than entering a retry loop.

- [x] **Step 4: Preserve layout across every restart flavor**

Assert and implement:
- Alt+N whole-workbench restart uses the recorded mode;
- Alt+Shift+N agent-only restart never touches topology;
- compaction and rename restart preserve the record;
- only a confirmed explicit live override changes topology.
- a deletion error or still-present session aborts without entering relaunch.

Run: `go test ./cmd/internal/launcher -run 'TestRunLaunch.*(LiveLayout|Restart.*Layout|Quit.*Layout)' -count=1`
Expected: PASS.

### Task 12: Split and package both KDL assets

**Files:**
- Create: `zellij/layouts/main-2.kdl`
- Create: `zellij/layouts/main-3.kdl`
- Delete: `zellij/layouts/main.kdl`
- Modify: `cmd/internal/entrypoint/asset_root.go`
- Modify: `cmd/internal/entrypoint/asset_root_test.go`
- Modify: `cmd/internal/runtimebundle/embed_test.go`
- Modify: `cmd/pair-go/main_test.go`
- Modify: `tests/pair-embedded-runtime-test.sh`
- Modify: `tests/term-pane-shortcuts-test.sh`
- Modify: `zellij/config.kdl`
- Generated: `cmd/internal/runtimebundle/assets/runtime/**`

- [x] **Step 1: Add failing asset-contract tests**

Require both layout files in source and embedded runtime. Assert layout 2 has
exactly agent+draft and preserves all draft rungs; assert layout 3 has
agent+draft+filler plus the floating terminal and its coordinate contract.
Make the asset-root marker require both files, reporting the missing basename.

- [x] **Step 2: Restore layout 2 and rename the current topology**

Restore `main-2.kdl` from the parent of the first three-pane commit
(`git show 7a552be2^:zellij/layouts/main.kdl`). Move the current layered
implementation to `main-3.kdl`. Update `zellij/config.kdl` comments and static
tests to address the correct asset. Add a structural comparison test for the
shared agent/draft launch commands and draft rung definitions so changes cannot
silently drift between static products (`ARCH-DRY` enforcement).

- [x] **Step 3: Validate both layouts with installed Zellij**

Run:

```bash
zellij setup --dump-layout zellij/layouts/main-2.kdl
zellij setup --dump-layout zellij/layouts/main-3.kdl
zellij --config-dir zellij setup --check
```

Expected: all exit 0.

- [x] **Step 4: Regenerate and verify the embedded runtime**

Run:

```bash
make runtimebundle-generate
make runtimebundle-drift-check
bash tests/pair-embedded-runtime-test.sh
```

Expected: PASS and both KDL assets are present in the extracted runtime.

### Task 13: Verify both modes and await operator smoke

**Files:**
- Modify: `atlas/architecture.md`
- Modify: `atlas/go-migration-inventory.md`
- Modify: `workshop/issues/000116-three-panel-user-terminal.md`

- [x] **Step 1: Update architecture and issue records**

Document `LayoutMode`, `workbench-layout-<tag>`, precedence, explicit live
override semantics, and the two KDL assets. Update every stale assertion that
Pair has only one topology (`ARCH-PURPOSE` shadow sweep).

- [x] **Step 2: Run automated verification**

Run:

```bash
go test ./cmd/internal/launcher ./cmd/internal/entrypoint ./cmd/internal/runtimebundle ./cmd/pair-go -count=1
go test ./... -count=1
make test-lua
bash tests/term-pane-shortcuts-test.sh
bash tests/pair-embedded-runtime-test.sh
git diff --check
```

Expected: PASS.

- [x] **Step 3: Install for live testing**

Run `make install`, then verify the running `pair` resolves to the newly
installed binary per `workshop/lessons.md`.

- [x] **Step 4: Operator smoke test — do not commit or land before approval**

Ask the operator to verify:
- a new `pair` session is the original two-pane workbench;
- `pair --layout3 codex` creates the layered three-pane workbench;
- detaching and invoking `pair resume <tag>` preserves the recorded mode;
- `pair claude --layout2 -- --other-claude-flags other-claude-params` consumes
  the Pair flag and forwards only the post-`--` values;
- an explicit conflicting layout prompts, decline is inert, accept recreates
  the same tag in the requested topology;
- Alt+N preserves topology, Alt+Shift+N restarts only the agent;
- layout 3 terminal tabs and the 50%↔67% overlay toggle still work.

Stop after reporting automated evidence. Commit and landing require a separate,
explicit operator signal after this smoke test.
