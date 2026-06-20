# Agentic Review Workbench — M3 (review window + agent poke) Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Steps use checkbox (`- [ ]`) syntax.

**Goal:** Put M1/M2's review loop into a real pair session — a `:PairReview <file>` / Alt+r review pane (floating nvim loading `nvim/review/`), the mode-aware Alt+r toggle between agent+draft and review views, and the review-view "finish human turn" gesture that pokes the agent — so a human can drive a review against a live agent. Agent *intelligence* (recognizing review/ship/chat, memory discovery) is M4.

**Architecture:** The review view is a **persistent floating zellij pane** running `nvim -u nvim/review.lua <file>` (the proven scrollback/changelog pattern, but persistent so its visibility can toggle). A `:PairReview` user command in the draft nvim (`-complete=file`, `:e`-style) opens it; a `pair-review-toggle` script bound to **Alt+r** branches on review-pane state — file-select when none is open, toggle floating-pane visibility when one is. Inside the review pane, **Alt+Return** runs M1's `human_round` (commit the human's incoming edits) then pokes the agent ("updated, please review") via the existing `send_to_agent` sequence (`move-focus up → write-chars → write 27 13 → move-focus down`). The agent pane is pair's *existing* agent — free-form chat and "ship it" work conversationally for free; the SKILL that makes those review-aware is M4.

**Tech Stack:** zellij keybinds (`zellij/config.kdl`) + shell launchers (`bin/pair-*`, template: `pair-scrollback-open`); nvim Lua (`nvim/review.lua` init dofiling `nvim/review/init.lua`; review-view keymaps; a `:PairReview` command in `nvim/init.lua`). Headless-testable: command/keymap wiring, the poke-string, marker-highlight computation, `pair-review-toggle`'s branch logic (zellij mocked). **Not** headless-testable: live zellij pane/focus/floating behavior → a manual real-session smoke checklist.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `review.markers` highlight mapping | `nvim/review/markers.lua` (extend) | modified |

- **marker→highlight mapping** — a pure function turning `markers.parse_markers(lines)` output into highlight spans `{row, col_start, col_end, hl_group}` for the `🤖`/quoted/strike/`[]`/`{}` parts (the `ParleyReview{Quoted,Strike,User,Agent}` groups, per parley's highlighter). Colocated test. (The rendering — placing the extmarks — is integration.)

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `pair-review-open` | `bin/pair-review-open` | new | zellij floating pane launch |
| `nvim/review.lua` (pane init) | `nvim/review.lua` | new | nvim bootstrap for the review pane |
| `:PairReview` command | `nvim/init.lua` | modified | draft-nvim user command (`-complete=file`) |
| `pair-review-toggle` | `bin/pair-review-toggle` | new | mode-aware Alt+r (file-select vs toggle) |
| Alt+r keybind | `zellij/config.kdl` | modified | global keybind → `pair-review-toggle` |
| review-view keymaps | `nvim/review.lua` | new | Alt+Return = human_round + poke; marker render |
| agent poke | reuse `nvim/init.lua` `send_to_agent` pattern | (reused) | zellij `write-chars`/`write` |

- **`pair-review-open <file>`** — launches the review floating pane: validates the file, writes a review pid/state file (so `pair-review-toggle` can detect "a review is open"), and execs `nvim -u "$PAIR_HOME/nvim/review.lua" <file>`. Template: `bin/pair-scrollback-open`, but the floating pane is **persistent** (not `close_on_exit`) so toggling hides rather than kills it.
- **`nvim/review.lua`** — the pane's init: `dofile` `nvim/review/init.lua`, `review.start{ file=<arg>, tag=$PAIR_TAG }`, set the review-view keymaps, render the doc + markers, write the pid file (VimEnter, like `scrollback.lua`). Self-contained (no rtp), mirroring `scrollback.lua`/`changelog.lua`.
- **`:PairReview` command** — `nvim_create_user_command('PairReview', fn, { nargs=1, complete='file' })` in the draft's `init.lua`; `fn` shells out to `pair-review-open <file>`. This is the `:e`-style entry; Alt+r-in-normal-mode just feeds `:PairReview ` into the draft command line for tab-completion.
- **`pair-review-toggle`** — the Alt+r brain: if the review pid/state file is absent or dead → focus the draft and inject `:PairReview ` (the file-select prompt); else → toggle the review pane's visibility. **Fix (M3 review): NEVER `toggle-floating-panes`** — it toggles ALL floating panes and *opens a stray one if none exist* (a footgun). Branch on `zellij action are-floating-panes-visible` + explicit `show-floating-panes` / `hide-floating-panes`. The Task 2 test asserts no `toggle-floating-panes` call.
- **review-view keymaps** (in `nvim/review.lua`): **Alt+Return** → `review.human_round(buf)` (M1: save + `docflow round --side human`) then poke the agent "updated, please review" (the `send_to_agent` sequence); **Alt+r** also bound inside the pane to toggle back (so it works whether focus is in the pane or not); never bind bare `esc`.
- **agent poke** (`nvim/pair_poke.lua`, new) — **CRITICAL fix (M3 review):** `send_to_agent`'s relative `move-focus up/down` does NOT escape a floating pane (documented in `scrollback.lua`), so the review pane must address the agent pane by **absolute id**: `zellij action list-panes --json` → find the agent (`is_floating==false && title!="draft"`, as `pair-scrollback-open` already does) → `focus-pane-id <agent>` → `write-chars <body>` → `write 27 13` → `focus-pane-id <review>` (restore). The draft's existing `send_to_agent` is left untouched (proven); `pair_poke.send(body)` is the new id-based poke the review pane uses. **Testability:** `pair_poke` must NOT early-return on `has_ui()` the way `send_to_agent` does (that's why the headless poke test can record `zellij` args) — gate UI-only behavior elsewhere, or expose a `pair_poke._cmds(body)` pure builder the test asserts on while a `$PATH` `zellij` stub records the real calls.

---

## Milestone 3 — the review window + poke

### Task 1: `:PairReview` + `pair-review-open` + `nvim/review.lua` (open the pane)

**Files:** Create `bin/pair-review-open`, `nvim/review.lua`; Modify `nvim/init.lua`; Test `tests/review-window-test.sh`

- [ ] **Step 1: failing test** (headless) — run `nvim -u nvim/review.lua <tmp-file>` headlessly and assert: `nvim/review/init.lua` loaded (a sentinel global from `review.start` is set), the buffer is the target file, and the review-view keymaps exist (`vim.fn.maparg` for the Alt+Return mapping is non-empty). Separately assert the `:PairReview` command exists with `complete=file` after sourcing the draft `init.lua` bits (or a focused require).
- [ ] **Step 2: run → fail.**
- [ ] **Step 3: implement** — `nvim/review.lua` (dofile review/init.lua, `review.start`, keymaps, pid file); `bin/pair-review-open` (template from `pair-scrollback-open`, persistent floating, writes pid/state file, `nvim -u nvim/review.lua`); `:PairReview` user command in `nvim/init.lua` shelling to `pair-review-open`. **Lifecycle fixes (M3 review):** (a) wire `VimLeave` → `review.stop(buf)` so the handoff poll timer (`vim.uv` in `handoff.watch`) + the projection autocmd are torn down when the pane nvim exits — the persistent pane makes this the M1-carried "VimLeave timer cleanup" item's home; (b) **single review pane** — a 2nd `:PairReview <other>` while a review is open *replaces* it (close the old pane / re-target), not stack; the pid/state file is the singleton guard (like scrollback's `.openlock`, but persistent).
- [ ] **Step 4: run → pass.**
- [ ] **Step 5: commit** — `#66 M3: review pane — :PairReview + pair-review-open + nvim/review.lua`.

### Task 2: Alt+r — mode-aware toggle (`pair-review-toggle` + keybind)

**Files:** Create `bin/pair-review-toggle`; Modify `zellij/config.kdl`; Test `tests/review-toggle-test.sh`

- [ ] **Step 1: failing test** — with zellij actions stubbed on `$PATH` (record args), run `pair-review-toggle` (a) with NO review state file → asserts it injects `:PairReview ` to the draft (the file-select path); (b) with a live review state file → asserts it calls `are-floating-panes-visible` then `show-floating-panes`/`hide-floating-panes`, and asserts it issues **no** `toggle-floating-panes` (the footgun is locked out). Lint `zellij/config.kdl` for the new `Alt r` bind syntax.
- [ ] **Step 2: run → fail.**
- [ ] **Step 3: implement** — `bin/pair-review-toggle` (branch on the pid/state file); `zellij/config.kdl` `bind "Alt r" { Run "pair-review-toggle" { floating true; ... } }`. Confirm Alt+r is unbound + no macOS dead-key collision (it is/none).
- [ ] **Step 4: run → pass.**
- [ ] **Step 5: commit** — `#66 M3: Alt+r — mode-aware review toggle (file-select vs visibility)`.

### Task 3: review-view "finish human turn" (Alt+Return = human_round + poke)

**Files:** Modify `nvim/review.lua`; Create `nvim/pair_poke.lua` (extracted poke); Test `tests/review-poke-test.sh`

- [ ] **Step 1: failing test** (headless, `zellij`+`docflow` stubbed on `$PATH` recording args) — drive the review pane's Alt+Return callback; assert (a) `docflow round --side human` was invoked (M1 `human_round`), and (b) the poke addressed the agent **by id**: the recorded `zellij` calls include `list-panes`, `focus-pane-id <agent>`, `write-chars` with a "please review" body, `write 27 13`, and `focus-pane-id <review>` (restore) — and crucially the call recording works because `pair_poke.send` does NOT early-return on `has_ui()`. Also unit-test `pair_poke._cmds(body)` (pure command builder) directly.
- [ ] **Step 2: run → fail.**
- [ ] **Step 3: implement** — `nvim/pair_poke.lua` (new): a `_cmds(body)` pure builder + `send(body)` that resolves the agent pane id via `list-panes --json`, `focus-pane-id` agent → `write-chars`/`write 27 13` → `focus-pane-id` review (restore). **Not** a verbatim copy of `send_to_agent` — id-based, no floating-escape assumption, no `has_ui()` short-circuit. Wire review-view Alt+Return → `review.human_round(buf)` → `pair_poke.send("updated, please review <file>")`. (Naming note: parent plan called this `review/poke.lua`; it lives at top-level `nvim/pair_poke.lua` because the draft can share it — reconciled in the parent plan's Revisions.)
- [ ] **Step 4: run → pass.**
- [ ] **Step 5: commit** — `#66 M3: review-view Alt+Return — finish human turn (human_round + poke agent)`.

### Task 4: marker rendering (highlight 🤖 in the review buffer)

**Files:** Modify `nvim/review/markers.lua` (highlight mapping) + `nvim/review.lua` (place); Test `nvim/review/markers_test.lua` (extend) + the window test

- [ ] **Step 1: failing test** — pure: `markers.highlight_spans(lines)` → spans `{row,col_start,col_end,hl_group}` for a `🤖<q>[u]{a}` fixture (Quoted/User/Agent groups, byte-accurate). Headless: placing them yields extmarks on the right rows.
- [ ] **Step 2: run → fail.**
- [ ] **Step 3: implement** — port parley's highlighter fragment (`highlighter.lua:447-484`) as a pure `highlight_spans`; in `review.lua`, define the `ParleyReview*` hl groups and place the spans (extmarks) on render + on TextChanged.
- [ ] **Step 4: run → pass.**
- [ ] **Step 5: commit** — `#66 M3: marker rendering — highlight 🤖 review-request sections`.

### Task 5: real-session smoke + manual verification

Automated tests cover the wiring; live zellij pane behavior cannot run headlessly. Add `workshop/` or in-plan **manual smoke steps**, and run them in a real pair session:

- [ ] In a real `pair` session on a git doc: **Alt+r** → file-select prompt with tab-completion → pick the doc → review pane opens, renders the doc (+ any markers).
- [ ] **Alt+r** again → view toggles to agent+draft; **Alt+r** → back to review. (Floating-pane visibility toggle, agent/draft survive.)
- [ ] Run M1's **fake agent** (or a real agent told inline to emit a handoff) → the pane applies + renders the round (M2 projection); **undo** in the live pane → decorations restore (this exercises the `TextChanged` autocmd that headless can't — closes the M2 gap).
- [ ] In the review pane, edit + **Alt+Return** → a human round commits and the agent pane receives "updated, please review".
- [ ] Switch to agent+draft, type a question → free-form chat (pair's existing agent); type "ship it" → (no-op in M3; M4 makes the agent act).
- [ ] Record the smoke result in `## Log`.

### Task 6: Milestone close

- [ ] `make test-lua` + `make test-review` (+ new window/toggle/poke tests) green; manual smoke recorded.
- [ ] Update `atlas/review-workbench.md` (the window/poke surface; M3 done).
- [ ] Close via **ariadne's** sdlc (omit `--actual` → measured) — `milestone-close --issue 66 --milestone M3 --verified '<wiring tests green + manual smoke: pane opens/toggles, human-round pokes, projection undo fires live>'`. Address the auto-dispatched review's Critical/Important; log the verdict.

---

## Open details to resolve in-milestone

- **Floating-pane visibility toggle exact zellij action** — `toggle-floating-panes` toggles *all* floating panes; usually only the review one is live (scrollback/changelog are close-on-exit), but confirm in a real session and switch to a focus-by-name approach if needed.
- **`pair-review-toggle` state detection** — pid file (review pane writes it on VimEnter, removes on exit) vs querying zellij for a pane named `review`. Pid file is simpler + matches scrollback's pattern.
- **Where `:PairReview` runs from** — the draft nvim's command line (Alt+r feeds `:PairReview `). Confirm the draft is the right host (it's the only interactive nvim) and that command-line mode doesn't disturb the draft buffer.
- **Agent-side is dumb in M3** — the poke says "please review"; without the M4 SKILL the agent won't do memory discovery or emit the records handoff cleverly. M3's automated loop uses the fake agent; the real intelligent loop is M4. Don't over-build the agent side here.
- **Ship** — `docflow ship` plumbing exists (`review.docflow.ship`), but the *trigger* ("ship it") is the agent's call (M4). M3 leaves ship conversational + unwired.

## Revisions

### 2026-06-19 — Alt+r rework (toggle-pane → draft-nvim lua)

**Reason:** the first M3 real-session smoke (Task 5) found 5 bugs, four sharing one
root cause — Task 2's design ran the Alt+r branch *inside a transient 20%×1 floating
shell pane* (`Run "pair-review-toggle"`). That intermediate floating pane caused: #1
~1s open delay (two pane spawns), #3 the review pane auto-hiding after ~1s + #5 Alt+r
in review mode mis-firing `:PairReview` (the transient floating pane confounded
`are-floating-panes-visible` / flapped the alive-state), and #4 a half-size review
pane (`pair-review-open`'s `tput cols/lines` measured the 20%×1 toggle pane). (#2, a
docflow ENOENT on VimEnter, was unrelated — fixed separately by making `docflow.lua`
resilient.)

**Delta** (supersedes Task 2's `bin/pair-review-toggle` design; Tasks 1/3/4 unchanged):
- **Alt+r bind** (`zellij/config.kdl`) now routes through the draft nvim exactly like
  Alt+d / `PairConfirmDetach`: `MoveFocus "Down"; Write 28; Write 14; WriteChars
  ":lua PairReviewToggle()"; Write 13`. No spawned shell pane → no second spawn, no
  transient floating pane.
- **`PairReviewToggle()`** moves into nvim lua, defined in **two** places:
  - draft `nvim/init.lua` — branches on review state-file liveness: not alive →
    `:PairReview ` cmdline (file-select); alive → `are-floating-panes-visible` (now
    reliable, queried from the *tiled* draft) → `show`/`hide-floating-panes`. Pure
    decision `_pair_review_toggle_action(alive, visible)`.
  - review pane `nvim/review.lua` — hide-self, for when Alt+r fires from inside the
    focused floating review pane and the bind's relative `MoveFocus Down` doesn't
    escape it (robust either way: if it *does* escape, the draft handles it via
    `are-floating-panes-visible`).
- **`pair-review-open`** spawns full-screen via percentage dims (`--width 100%
  --height 100%`, a `zellij run` feature) instead of `tput` (the half-size fix).
- **state file** simplifies to a single line (the pane nvim's pid); visibility is no
  longer tracked there (the `are-floating-panes-visible` query is now reliable).
- **`bin/pair-review-toggle` deleted.** Test `tests/review-toggle-test.sh` rewritten
  to headless-drive the lua `PairReviewToggle()` (zellij stubbed, `are-floating-panes
  -visible` answered from a file) + the pure decision + the KDL bind lint.

**docflow in M3 (open-question resolution):** render-only stays the M3 scope. `docflow`
isn't on PATH in a live session and M3 is the *window/toggle* milestone — the commit
pipeline (`DOCFLOW_BIN` resolution to the sibling ariadne, ship) is wired in M4. The
Task 5 smoke item "Alt+Return → a human round commits" relaxes to "Alt+Return pokes
the agent" (the round commit no-ops with a warning until M4).

All headless suites green after the rework (`make test-lua` + `make test-review`, 92
checks). The live re-smoke (Task 5) is the remaining gate before `milestone-close`.
