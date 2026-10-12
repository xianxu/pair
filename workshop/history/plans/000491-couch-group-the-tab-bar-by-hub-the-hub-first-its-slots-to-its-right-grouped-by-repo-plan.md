# pair#491 plan: group Couch's tab bar by hub

**Goal:** the tab bar reads `[ops]  :1  ariadne  :1  :2  blog  pair  :1  :2`: the hub first and bracketed, then its slots grouped by repository. The design leaves room for several hub sections later, but builds only one now.

**What already exists:** the repository grouping. `PresentThreads` (`couchtty/thread_presentation.go`) orders groups alphabetically by repository name, and orders slots by number within a group. `RenderStatusRow` (`reserve.go`) already draws a group's later members as `:N`, so `pair  :1  :2` is today's rendering. What's missing is the hub section, a bracket that means "hub", and a narrow-width policy. Today the row simply clips at the right edge, so a red chip past the edge disappears.

**One source of order (ARCH-DRY):** `PresentThreads` already orders both the tab bar and the switcher. The hub ordering goes there, so clicks, the switcher and the tab bar all agree without a second sort.

## Design

### 1. The hub reaches the Console
- couchcmd already computes the hub, `hubWorktree(os.Getwd, …)`, which is the worktree root Couch started in (#433/#463). `startMessageService` hands the same value to the Console through `console.SetHub(root)`. The value is fixed for the process lifetime, so it is set once and never refreshed. A Couch started outside a checkout has `""`: no hub, and today's layout.
- It is stored as `MenuState.Hub`, a path, because `PresentThreads` already has `MenuState` at both of its call sites.

### 2. Order: `PresentThreads(rows, git, hub)`
- The hub's row is the one whose presentation `Path` equals `hub`: a slot's worktree root, or the primary's root.
- **Section order:** first the hub's repository group, then the remaining groups alphabetically, as today. Within the hub's group the hub row comes first, then the rest by slot number. Example, hub = `ops:1`: `ops:1, ops, ops:2`.
- `ThreadPresentation.Hub bool` marks the hub row and is carried to `StatusActor.Hub`.
- The switcher uses the same order, so the hub's group is listed first there too. The switcher's look is otherwise unchanged; it gets no brackets.
- The reattach inventory (`menu_reattach.go`) passes `""`, because its order is not a display order.

### 3. Rendering the row
- **The hub chip** is drawn bracketed with its full label: `[ops]`, or `[ops:1]` when the hub is a numbered slot. The chips after it in its group follow today's rule: the repository name stands for `:0`, and `:N` for a numbered slot. So hub `ops:1` gives `[ops:1]  ops  :2`, and hub `ops` gives `[ops]  :1  :2`.
- **The active chip.** Today the active chip is shown by `[…]`, which collides with the hub's brackets. **Proposal:** draw the active chip in reverse video (`\x1b[7m`) instead of brackets, keeping every other cue: red behind a picker, amber bell, fade, placeholder. When the hub is the active chip it is bracketed *and* reversed. (Alternative if the operator would rather keep `[…]` for active: give the hub a distinct mark, e.g. `⟦ops⟧` or `ops▸`. This is the one choice I'd like ops:0 / the operator to make.)
- **Sections, for later:** a section is a hub chip plus the chips that follow it up to the next hub chip. With one hub, every slot is in its section. The `:N` shortening already restarts at a new group, and a new hub chip will restart it too. The renderer needs nothing more for several hubs. **Membership rule when there are several hubs (not built):** a slot belongs to the hub that most recently dispatched work to it (the last `--send-to` from a hub). A slot no hub has messaged belongs to the first hub. The rule lives in `PresentThreads`' section assignment, which today has one section.
- The broadcast, Chrome, capture and notice cells keep their places. The broadcast cells and the capture badge lead (they are indicators that must never be pushed off the row), and the notice trails. Note: the Spec says the broadcast cells are "at the end", but they lead today, and #395 requires that; I'm keeping them where they are.

### 4. Narrow widths: `fitChips(actors, budget) []cell` (pure)
After the leading cells, the chips get the remaining width, and the notice gets only what the chips leave. Chips are *protected* if they are the hub, active, blocked (red) or belled. If the full layout doesn't fit, it degrades in these steps, stopping at the first that fits:
1. **Full:** `[ops]  ariadne  :1  :2  blog  pair  :1  :2`.
2. **Collapse numbered slots into a count**, one group at a time from the rightmost: the group's unprotected `:N` chips become one `+N` cell. `pair  :1  :2` → `pair +2`, while a red `:2` stays drawn: `pair  :2 +1`.
3. **Collapse whole groups**, from the rightmost: an unprotected group (leader included) becomes part of one trailing `+N` cell counting every hidden slot. A group with a protected chip keeps that chip, drawn with its full label (`pair:2`), because its leader is gone.
4. **Clip**, as today. This only happens when the protected chips alone don't fit. The hub and red chips come first in the protected order, so they are the last to go.

A `+N` cell is not a click target. Clicking it does nothing, as clicking bare row does today, and the switcher (Ctrl-Space…) lists everything. Chip spans still come from the same pass that clips (the `RenderedStatusRow` contract), so a click maps to what was actually drawn.

## ARCH notes
- **ARCH-DRY:** one order, in `PresentThreads`, and one fitting function. Spans are recorded by the pass that draws.
- **ARCH-PURE:** `PresentThreads` and `fitChips` are pure functions of their inputs and are tested without a terminal.
- **ARCH-FUNERAL:** this creates nothing durable. `MenuState.Hub` is an in-memory string set once per process.
- **ARCH-CONSTRAINTS:** `fitChips` is O(chips × groups) per render over at most a few dozen chips, and does no IO.
- **Layer direction:** couchcmd hands couchtty a path. Nothing new reaches pair or sdlc.

## Tests (each fails if its code is reverted)
- `PresentThreads`: the hub's group goes first, with the hub row first in it, for a hub that is `:0` and for one that is `:N`. No hub gives today's order. Hub flags appear only on the hub row.
- `RenderStatusRow`:
  - `[ops]  :1  ariadne  :1  :2  blog  pair  :1  :2`, exact.
  - A numbered hub gives `[ops:1]  ops  :2`.
  - Missing slots aren't drawn.
  - The active chip is reverse video and not bracketed; the active hub is both.
  - The precedence test is kept: red > bell > fade, placeholder.
- `fitChips` / narrow widths:
  - Each degradation step happens at the width where the previous step stops fitting.
  - A red `:2` survives steps 2 and 3, drawn as `:2` and then as `pair:2`.
  - The hub survives to the end.
  - Chip spans map each column to the drawn thread, and `+N` maps to nobody.
- The switcher lists the hub's group first.
- couchcmd: `startMessageService` hands the Console the hub root.

## Steps
- [ ] `SetHub` / `MenuState.Hub` wiring, with a test.
- [ ] `PresentThreads` hub ordering and the `Hub` flag, with tests.
- [ ] Hub bracket and active reverse video in `RenderStatusRow` (pending the marker decision), with tests.
- [ ] `fitChips` degradation and spans, with tests.
- [ ] Atlas (couch tab bar), README tab-bar colours (also folds in #465's advisory about the red state), couch SKILL if it describes the row.
- [ ] Full suite, then a live check in a scratch Couch at several widths. Ask ops:0 to arrange the operator's smoke test, then close.

## Revisions
- 2026-10-12, operator review of the first proposal. The hub's repository is not grouped with it. The hub is a section header standing alone, and its section's slots follow in repository groups:
  `[ops]  ariadne  :1  :2  blog  pair  :1  :2  [ops:1]  ariadne:3  :4  pair:3  :4`.
  This replaces §2's "the hub's repository group first" and §3's `[ops:1]  ops  :2`.
  - **Order (supersedes §2):** each section is its hub row first, then its member rows in repository groups, alphabetical, slots by number, as today. A group whose `:0` is not in the section leads with its full label (`ariadne:3`), the rule the renderer already follows. A non-hub slot of the hub's own repository (`ops:1` while only `ops` is a hub) is an ordinary member in its alphabetical group, drawn `ops:1`.
  - **The hub concept (supersedes §3's membership rule):**
    - A hub is a slot *designated* as one. Today that is only the slot Couch started from, which is implicit and can't be changed.
    - A slot is *assigned* to one hub.
    - A newly started slot inherits the hub of the slot it was started from: starting `pair:10` from `pair:1`, which reports to hub1, makes `pair:10` report to hub1, so no slot is ever without a hub.
    - Designating and assigning become Couch UI actions in a later issue.
  - **Built now:** one implicit hub. The data path is shaped for more: `PresentThreads` takes a membership function `hubOf(row) → hub path`, which today returns the single hub for every row. Sections are rendered from the hub flags, so a second hub needs only a designation source and an assignment source. With no hub (Couch started outside a checkout) there is one section with no header, as today.
  - The hub's special display is the brackets. The active-chip marker question stands (reverse video proposed).
- 2026-10-12, operator, second review: **no special marking for a hub.** A hub is shown by position only. This supersedes the brackets in §3 and the previous revision, and closes the active-marker question: `[…]` stays the active chip's marker, unchanged.
  - **Tab bar:** each section starts with its hub, followed by its members grouped by repository: `ops  ariadne  :1  :2  blog  pair  :1  :2`. A separator (` │ `) goes between two sections; with one hub there is none. With no hub the row is today's.
  - **Switcher:** the hub is the first line, at indent 0. Every other row of its section, including the notification lines under it, is indented one more level (2 spaces). Repository primaries go to 2, slots to 4, and their message lines follow `entry.Indent`, so they shift with them. With no hub, today's indentation.
  - **Mechanics:** `ThreadPresentation` gains `Hub bool`. The extra indent is added in `PresentThreads`, the one place that sets `Indent`. `orderedMenuInventory` (switcher order) and both render paths pass `MenuState.Hub`.
  - **Narrow widths (§4)** are unchanged. The hub stays protected so it remains leftmost, along with the active, red and belled chips.
- 2026-10-12, close review (round 1) fixes:
  - **Membership:** there is no `hubOf(row)` function. With one hub nothing would vary it, so the rule ("every row belongs to the one hub") lives inside `hubSection`. A second hub's assignment source replaces that rule there.
  - **§4 step 2:** a collapsing group folds all but its *first member* into `+N`, not its "unprotected `:N` chips". A group with no `:0` (`ops:1  ops:2  ops:3` while `ops` is the hub) otherwise drew a bare `+2` after the previous repository, which read as that repository's count. Now it reads `ops:1  +2`.
  - A `+N` that doesn't fit whole is dropped instead of being clipped to a bare `+`. Drawing and measuring share `cellGap`.
  - The hub's handoff is tested at both ends: `messageService.showHub` (couchcmd) and the Console keeping it across the menu's lazy first build (`ensureMenuLocked`).
