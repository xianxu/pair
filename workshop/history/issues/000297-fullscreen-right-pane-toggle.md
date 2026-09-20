---
id: 000297
status: done
deps: []
github_issue:
created: 2026-09-20
updated: 2026-09-20
estimate_hours: 2.48
started: 2026-09-20T09:53:35-07:00
flow: {kind: full, provenance: operator}
actual_hours: 3.65
---

# Alt+Shift+Return globally toggles the right pane between 50/50 and fullscreen, restoring focus

## Problem

`Alt+Shift+Return` with the right pane focused re-tiles the terminal column
between half the screen and about two thirds. Two thirds is the ceiling, and it
is reached by a blind three-step resize burst calibrated to zellij's 5%-per-step
`resize increase left` (`layoutcmd/resizeplan.go:15-18`, #124).

The operator wants the right pane maximised — a full-width terminal on a
186-column screen — in preparation for the carbonyl browser tab (#292), where a
rendered page wants every column it can get. The current ladder cannot express
that, and widening it by adding resize steps would keep inheriting the burst's
fragility: the step size is a zellij constant Pair re-derives by measurement, and
`resizeplan.go` already carries the caveat that a future zellij step change
"degrades to a different stable pair of widths".

The chord is also pane-local, which makes it useless at the moment the operator
actually wants it. The workflow is: typing in the draft → maximise the right pane
and start working in it → come back to the draft and keep typing. Today that
costs an `Alt+k` before and after, and the chord does not even exist outside the
right pane.

zellij has the operation natively:

    $ zellij action toggle-fullscreen --help
    Toggle between fullscreen focus pane and normal layout
      -p, --pane-id <PANE_ID>  Target a specific pane by ID

It is per-*pane*, not per-session: the focused pane fills the tab and the other
panes are hidden. Pair already drives zellij this way — `RunToggleFocused` calls
`rt.RunZellijAction("resize", …)` today, so this is the same seam with a
different verb.

It is also strictly safer than the alternatives considered. A fullscreen pane has
no neighbour boundary to drag and no floating frame, so it carries none of the
mouse-drag exposure that made the floating right terminal untenable in #123
(zellij still has no config gate for frame-drag move — verified against 0.45.1's
`setup --dump-config`; the mouse keys added since 0.44.3 are
`mouse_scroll_resize`, `scroll_mode_sync`, `mouse_hover_tips`,
`osc133_command_selection`, `osc8_hyperlinks`, none of which gates drag).

## Spec

`Alt+Shift+Return` becomes a **global** chord that toggles the right terminal
between the layout's resting 50/50 tiling and zellij fullscreen, **carrying focus
both ways**:

- **Expand** (from any pane): record the currently focused pane, focus the right
  terminal, fullscreen it. The operator lands in the maximised pane ready to work.
- **Collapse** (from the fullscreen right terminal): leave fullscreen, restore
  focus to the recorded pane, clear the record. The operator lands back in the
  draft with the cursor where they left it.

Precision, measured: zellij restores **the tiling that was in effect**, not the
layout's declared proportions — the probe went 95 → 191 → 95 columns. Those are
the same thing only as long as nothing has dragged the split. "50/50" throughout
this issue means the resting state, not a re-tile that collapse performs; if a
true snap-back is wanted, it is extra work and is not specified here.

Two states, one press each way. The ~2/3 rung is retired — operator decision,
2026-09-20: a true toggle is worth more than a middle width, and the draft ladder
(`Alt+Up`/`Alt+Down`) already covers partial re-tiling.

### Global, not role-scoped

Move `ChordAltShiftEnter` from `roleBindings` into `globalBindings` with
`HandledInPane: true`, following the `ChordAltShiftLeft/Right/T` precedent
(#216/#243): every pane acts on it directly rather than routing a Lua call into
the draft, and the draft gets its own `NvimKey` entry for the same action.

**This absorbed #296** (closed wontfix/superseded 2026-09-20). `RightTerminalChordPassesThrough` already returns false
for any global (`!IsGlobalChord(chord)`, `shortcut.go:398-400`), so a global chord
is never forwarded to a full-screen child. The carbonyl passthrough problem #296
was filed to fix disappears as a consequence of this change rather than needing
its own reservation mechanism. Close #296 as superseded when this lands; do not
build both.

The standard global tradeoff applies and is accepted: a full-screen application
in the right pane can no longer claim this chord. That is the same bargain #258
struck for the tab chords, and it is the point — the chord must work *especially*
when carbonyl owns the screen.

### Chord encoding — absorbed from #296

`ChordAltShiftEnter` registers exactly one sequence, `\x1b[13;4u`
(`shortcut.go:437`). Two gaps come with it, and **going global raises the stakes
on both**: a chord that is dead on a given host is now dead in every pane rather
than in one.

- **The meta-family sibling is missing.** `shortcut.go:423-428` establishes the
  rule that both modifier families are registered — bit-2 "alt" reports modifier
  3/4, bit-8 "meta" reports 9/10 — precisely so a chord is not silently dead on a
  meta-style terminal, and `TestMetaSiblings` enforces it for the chords it
  covers. `\x1b[13;10u` is absent. Confirm whether it is genuinely unreachable
  for Enter or an oversight, and register it if reachable.
- **`\x1b[13;4u` is a Kitty-keyboard encoding.** Pair pushes `\x1b[>3u` to the
  host (`terminal/presenter.go:847`), so the form arrives on a KKP host such as
  Ghostty. On a host without KKP, Shift+Alt+Enter is likely indistinguishable
  from Alt+Enter at the byte level, which would make this chord host-dependent in
  the same way #232/#233 are. Establish which hosts deliver it before promising
  the behaviour in the README's *Terminal setup* table; if it is host-dependent,
  that is a documentation row, not code.

### Demote Alt+Up / Alt+Down to the draft pane

Operator decision, 2026-09-20. `ChordAltUp` / `ChordAltDown` step the draft's
height ladder and are currently **global** (`shortcut.go:183-186`), so they fire
from the right terminal and the agent pane too. They were made global for a
reason that has expired: stepping a rung re-applies a swap layout, which re-tiles
the whole workbench and therefore snaps an accidentally-dragged left/right split
back to its layout proportions. It was the quick undo for a mis-drag. With
ctrl+wheel resize filtered, the mis-drag largely stopped happening, and the
global reach is now just a chord the right pane cannot use for itself.

Make both draft-local. In every other pane they fall through to
`DispositionPass` and reach the child, which is the passthrough default.

**This dissolves unknown 2.** A draft-local rung chord is unreachable while the
right pane is fullscreen, because the draft is hidden and cannot hold focus.
"What do the rungs do to a fullscreen pane" stops being a question rather than
needing an answer.

**Do not implement this by deleting the two entries.** `RenderLuaGlobalMaps`
(`render_lua.go`) generates the draft's nvim keymaps from `GlobalBindings()`, so
removing them removes the draft binding as well — the opposite of the intent.
`_G.PairLayoutBigger` / `_G.PairLayoutSmaller` exist in `init.lua:3415,3439`, but
nothing hand-binds `<M-Up>` / `<M-Down>` to them.

Design decision for the plan, with a recommendation: add a **scope field** to the
binding (draft-local vs global) rather than a second table or a hand-written
keymap. `DecideGlobal` / `IsGlobalChord` filter on it, so the chord stops being
intercepted from other panes; `RenderLuaGlobalMaps` keeps rendering it, so the
draft keeps working; `keyhelp/catalog.go:63-64` flips from `ContextGlobal` to
`ContextDraft` off the same source. One table, one truth — the rule #257 states
and #132 paid for. A hand-written keymap in `init.lua` would work and is the
cheaper diff, but it re-creates exactly the second hand-maintained list this repo
keeps removing.

**Premise check before this lands.** "Ctrl+scroll is disabled" is true *here* and
not everywhere: couch filters ctrl+wheel (#213, done), but `mouse_scroll_resize
false` is still absent from `zellij/config.kdl` (#226, open), so a standalone
`pair` session outside couch still resizes on ctrl+wheel — and #268 (ctrl+drag
motion in pair) is open and unverified. Removing the global undo from standalone
pair while the mis-drag is still live there is the one way this change bites.
Landing #226's one-line config change alongside it makes the premise true
everywhere and closes that ticket; do that, or record why not.

### Chord retirements — two, both deliberate

`Alt+Shift+Return` currently means two other things. Operator decision,
2026-09-20: **retire both** rather than relocate them, so the chord has one
meaning everywhere.

1. **Draft: append-without-send.** `nvim/init.lua:3540` binds `<S-M-CR>` in
   normal + insert to `send_and_clear(true)` — append the buffer to the agent's
   composer with a trailing newline, do not submit. Remove the keymap. Then check
   whether anything else passes `no_submit`; if not, the parameter and its branch
   in `send_and_clear` (`init.lua:1522`) are dead and go too — a retirement that
   leaves the dead limb behind is half done.
2. **Review pane: the send menu.** `nvim/review.lua:706` binds `<M-S-CR>`
   buffer-locally to `open_mode_menu` — the mode/instruction selector, which also
   applies a pending round (#89 M3). Buffer-locality does not protect it: a Pair
   global is intercepted at the pane wrapper before nvim sees the bytes. Removing
   it leaves `<M-CR>` (`finish_human_turn`) as the review pane's only Return
   action, and `open_mode_menu` still reachable via its module export
   (`review.lua:793`). **This one was surfaced after the retirement decision was
   taken** — if the review send menu is load-bearing in practice, relocating it to
   a free chord is a cheaper change than reversing this issue's design.

Also update: `keyhelp/catalog.go:45` (the `<S-M-CR>` draft entry) and
`README.md:122`.

### Focus memory

The record of "which pane to come back to" needs a home. Reuse the existing
sidecar pattern — `workbenchshortcut`'s package doc already carves out
`LastLeftPaneStore`'s "small sidecar helpers" as the one impure corner, and
`ShortcutDecision` already carries `RecordLastLeftPaneID` /
`RecordLastTerminalPaneID`. Add the fullscreen-return record there; do not invent
a second storage mechanism.

Two reuses, not new inventions:

- **Which half to fullscreen under an `Alt+Shift+D` split:** the last-used-half
  memory `ActionFocusRightTerminal` already consults. One derivation of "the right
  terminal you mean", shared (ARCH-DRY).
- **When the recorded pane is gone:** fall back to the draft pane id, exactly as
  `ChordAltK` does (`shortcut.go:307-310`).

### Direction detection

Unlike a bare toggle, Pair must know which way it is going, because expand and
collapse do different focus work.

**Settled live, 2026-09-20: `list-panes --json` carries `is_fullscreen` per
pane.** That is the truth source — use it. The fallback this section previously
contemplated (treating the presence of the focus record as the state) is not
needed and should not be built. `zellijpane.Pane` does not parse the field today
(`zellijpane.go:17-28`, which already parses its sibling `is_floating`); adding it
is the whole change.

Measured, same session: the right terminal went 95 → 191 columns and back, rows
unchanged at 51. Note that hidden panes keep reporting their *old* geometry while
another pane is fullscreen — the agent and draft still reported cols=96 at their
original x/y — so pane geometry is not a usable signal for anything but the
target itself. `is_fullscreen` is.

### Order of operations is load-bearing

Whether zellij's fullscreen follows focus is unknown (below). Both sequences are
written so that it does not matter:

    expand:    record focused id → toggle-fullscreen --pane-id <right terminal>
    collapse:  toggle-fullscreen → focus-pane-id <recorded> → clear record

**Measured live, 2026-09-20.** `toggle-fullscreen --pane-id X` *pulls focus to X*
as a side effect, with no separate focus call: focus was on the draft before the
probe and on the right terminal during it. So the expand half of focus carry is
free — one command both maximises and lands the operator in the pane.

The collapse half is not free, and is exactly the gap this issue fills: leaving
fullscreen left focus on the right terminal. It did **not** return to the draft.
Pair must restore focus explicitly, after exiting fullscreen. Do not reorder
collapse as a "simplification" — exiting first is what keeps a focus-following
implementation from re-targeting fullscreen onto the draft.

### Unchanged

`ActionToggleFocusedLayout` and the `handleTerminalChord` seam keep their shape
(`termcmd/run.go:631`). Use `toggle-fullscreen`, not `toggle-no-ui-fullscreen` —
the goal is columns, and zellij's UI bars cost rows; say so at the call site so
the next reader does not "upgrade" it.

**This remains mostly a deletion on the layout side.** `terminalToggleBurst`,
`terminalToggleSteps`, the 60%-of-screen classification and `resizeplan_test.go`
all go, and with them Pair's dependence on zellij's resize step size. What
replaces them is focus bookkeeping, not geometry arithmetic. Note that
`RunToggleFocused` must stop keying off `focusedRightTerminal` — the chord now
fires when the right terminal is *not* focused, which is the common case.

### Unknowns to settle live before the design is fixed

Each changes the spec if it goes the wrong way. Establish them in a live session
and record the answers in `## Log`:

1. **Does fullscreen follow focus?** ~~Open~~ **half answered** (2026-09-20):
   `--pane-id` targeting pulls focus *to* the target on entry, and exiting leaves
   focus where it was. What remains untested is whether moving focus *while*
   fullscreen re-targets which pane is fullscreen — the case that would matter if
   anything can steal focus mid-fullscreen.
2. ~~**What do the swap-layout rungs do to a fullscreen pane?**~~ **Dissolved**
   by demoting `Alt+Up`/`Alt+Down` to the draft (above): a draft-local chord
   cannot fire while the draft is hidden. If that demotion is dropped from this
   issue, this unknown comes back.

3. **What happens under the `Alt+Shift+D` split?** Confirm fullscreening one half
   hides the other, and that the last-used-half memory survives the round trip.
4. **Does `pair term` re-render correctly across the geometry jump?** Measured
   95 → 191 → 95 columns live with a clean result confirmed by the operator, so
   this looks free; the tab strip's bytes were not inspected, so treat it as
   promising rather than closed. The pane The reserved tab-strip row and the
   presenter's geometry epoch already handle rung changes, but this is the largest
   single resize Pair will have made, and #223's scroll-region history says
   geometry edges are where this breaks.

## Done when

- From the draft, `Alt+Shift+Return` fullscreens the right pane **and moves focus
  into it**; pressing it again restores the 50/50 tiling **and returns focus to
  the draft**, cursor position intact.
- The same round trip works from the agent pane, and from the right pane itself
  (where "restore focus" means staying put).
- Round-tripping leaves the workbench in exactly the layout it started in — same
  rung, same pane sizes, same focus, no process restarts.
- The chord fires from every pane while a full-screen TUI (nvim, and later
  carbonyl) runs in the right pane, and the TUI receives no bytes for it.
- The right pane measures full screen width at fullscreen (186 columns on the
  operator's machine; assert against the screen width, not a literal).
- Under an `Alt+Shift+D` split, the last-used half is the one that fullscreens,
  and that memory survives the round trip.
- When the recorded pane is gone at collapse time, focus falls back to the draft
  rather than being left in the hidden pane or dropped.
- `Alt+Up`/`Alt+Down` behave sanely while fullscreen (per unknown 2 — either they
  work, or Pair exits fullscreen first; a corrupted rung ladder or a leaked focus
  record is a fail).
- `Alt+Up`/`Alt+Down` step the draft ladder from the draft, and reach the child
  as ordinary passthrough from the agent pane and the right terminal.
- The draft's `<M-Up>`/`<M-Down>` keymaps still exist after the demotion — a
  regression test proves the generated Lua table still carries them, since
  deleting the global entries is the obvious wrong implementation.
- `keyhelp` describes both as draft-scoped, derived from the routing source
  rather than restated.
- `mouse_scroll_resize false` is in `zellij/config.kdl` and #226 is closed, or
  the decision not to is recorded with its reason.
- `terminalToggleBurst`, `terminalToggleSteps` and the 60% classification are
  gone, along with `resizeplan_test.go`; no code depends on zellij's resize step
  size any more.
- The draft's append-without-send keymap is gone, and `send_and_clear`'s
  `no_submit` branch with it if nothing else calls it.
- The review pane's `<M-S-CR>` send-menu keymap is gone (or relocated, if that
  decision is revisited — see Spec).
- A test at the `layoutcmd` seam asserts the full action sequence for expand and
  for collapse, including order, without a live zellij.
- `IsGlobalChord(ChordAltShiftEnter)` is true and a guard asserts the chord
  cannot pass through to a full-screen child.
- `Alt+Shift+Enter`'s meta-family sibling is either registered or its absence
  is documented at the chord table with the reason.
- Which hosts deliver `\x1b[13;4u` is established; if the chord is
  host-dependent, the README *Terminal setup* table says so.
- #296 is closed as superseded, with the reason recorded.
- README's `Alt+Shift+Return` rows (`README.md:122-123`), `pair keys` / `Alt+h`
  help and `keyhelp/catalog.go:45` describe one global toggle; CHANGELOG entry
  written and flagged as a breaking keybinding change.
- `atlas/` updated if the layout vocabulary changes ("expanded" / "collapsed" no
  longer describe two widths).
- Operator smoke-tests it live in `~/workspace/pair`: draft → fullscreen → back
  to draft, with a shell, with nvim, and — once #292 lands — with carbonyl.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only; calibration is marked stale by estimate-source.
Derived after plan-quality accepted round 2. Native zellij actions, the existing
picker, sidecar helpers and diagnostic writer cover the external mechanisms;
the small pure transition model is the only new core. Familiar existing stack.

Decomposition: fullscreen model/executor (greenfield Go, base design 0.5 × 0.2
spec discount; impl 0.8 × 0.4); store/path lifecycle and Go routing (two smaller
Go extensions, each design 0.2 × 0.2, impl 0.5 × 0.4); editor routing (Lua,
design 1.5 × 0.2, impl 1.0 × 0.4); no-submit retirement (cross-cutting refactor,
design 0.5 × 0.2, impl 0.5 × 0.4); docs (design 0.15 × 0.2, impl 0.2 × 0.4);
one close review (design 0.1 × 0.2, impl 0.5 × 0.4); live conformance
(real-API discovery, impl 0.4 × 0.4). Thorough-plan design buffer 15%.
No additional library needed: use the already-tested native fullscreen command.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module design=0.10 impl=0.32
item: smaller-go-module design=0.04 impl=0.20
item: smaller-go-module design=0.04 impl=0.20
item: lua-neovim design=0.30 impl=0.40
item: cross-cutting-refactor design=0.10 impl=0.20
item: atlas-docs design=0.03 impl=0.08
item: milestone-review design=0.02 impl=0.20
item: real-api-discovery design=0.00 impl=0.16
design-buffer: 0.15
total: 2.48
```

## Plan

Durable implementation plan: [native fullscreen toggle](../plans/000297-fullscreen-right-pane-toggle-plan.md).
The 2026-09-20 implementation-planning revision below supersedes stale unknowns
and narrows the #226 obligation without changing the chosen native toggle.

- [x] Settle the four live unknowns in a real session; record answers in `## Log`
      before writing code. Any that goes the wrong way amends the Spec.
- [x] Decide direction detection (zellij flag vs focus record) and record why.
- [x] Move `ChordAltShiftEnter` into `globalBindings` (`HandledInPane`, NvimKey);
      drop it from `roleBindings`.
- [x] Rebuild `RunToggleFocused` as the expand/collapse sequences with the focus
      record; delete `resizeplan.go` and its tests.
- [x] Retire the two nvim keymaps + the dead `no_submit` branch.
- [x] Add the binding scope field; demote `Alt+Up`/`Alt+Down` to draft-local
      without dropping their generated keymaps.
- [x] Land `mouse_scroll_resize false` (#226) so the demotion's premise holds
      outside couch, or record why not.
- [x] Seam tests for both sequences incl. ordering, split-half selection, and the
      missing-recorded-pane fallback; global/passthrough guard.
- [x] README rows, `keyhelp` catalog, `Alt+h` help, CHANGELOG, atlas vocabulary.
- [x] Resolve the `;10u` meta sibling: register it, or document why not.
- [x] Establish the KKP host matrix for `\x1b[13;4u`; README row if needed.
- [x] Close #296 as superseded.
- [x] `make test`, then operator smoke test before closing (baseline exceptions logged; operator accepted smoke 2026-09-20).
- [x] Remove arbitrary inventory data-size cutoffs across providers; regress oversized records and preserved identity validation.

## Revisions

### 2026-09-20 — remove the reader cliffs across providers

Operator clarified that the cutoff removal must not be Codex-only: "remove all
arbitrary limits." Apply the same no-size-cutoff policy across native transcript
scanners, events/usage, and incremental reads. Sweep inventory ledger/log/config
and SQLite result readers too: their producers have no matching size ceiling.
Preserve chunk sizes, schema/identity/path checks, and bounded diagnostic output.
No unrelated repo-wide limits are in scope. Memory still scales with complete
records or existing batch results; this change does not promise constant memory.
This supersedes the Codex-only scope in the earlier revision below.

### 2026-09-20 — operator-approved transcript-reader side quest

The operator accepted the fullscreen smoke in another thread and requested the
Codex transcript limit fix on this branch before close. The actual root rollout
contains valid event records above 1 MiB, so the identity scanner drops the root
and relaunch incorrectly reports missing completed-turn evidence. Remove Codex
record cutoffs in identity, events, usage, and targeted incremental reads; retain
other providers' and ledger limits. Reuse chunked framing and strict JSON/identity
validation (ARCH-DRY, ARCH-FUNERAL), not a larger arbitrary cap. Full scans hold
one record at a time; incremental validation retains the observed suffix as it
already does. Memory scales with those bytes, not a constant-memory claim. No
new JSON parser or silent skipping of malformed/identity-conflicting records.
Tests must cover >1 MiB and >8 MiB records, evidence after large records, and
malformed/conflicting metadata. Confirm the real root is recognized read-only.

### 2026-09-20 — global scope + focus carry

**Reason.** The pane-local design was filed and immediately revised by the
operator: the chord is wanted precisely when focus is *not* in the right pane.
The workflow is draft → maximise → work → back to draft mid-sentence, and a
toggle that requires already being in the right pane does not serve it.

**Delta.**
- Scope: right-pane-focused → **global**, via `globalBindings` +
  `HandledInPane`, following the #216/#243 tab-chord precedent.
- Behaviour: added **focus carry** — record focus on expand, restore it on
  collapse, with a documented fallback and explicit operation ordering.
- Dependency: `deps: [pair#296]` **removed**. Going global makes the passthrough
  reservation automatic, so #296 is subsumed rather than depended on, and should
  be closed as superseded.
- Retirements: the global claim takes the chord away from two existing bindings
  (draft append-without-send, review send menu). Both retired per operator
  decision; the review-pane one was discovered after that decision and is flagged
  in the Spec as cheaply reversible.
- Unchanged from the original: two states not three, `toggle-fullscreen` over
  `toggle-no-ui-fullscreen`, the `resizeplan.go` deletion, and the four live
  unknowns (unknown 1 promoted to load-bearing).

### 2026-09-20 — #296 absorbed

**Reason.** #296 existed to stop #227's passthrough forwarding this chord to a
full-screen child. Making the chord global does that as a side effect, leaving
#296 with only two orphan items and no deliverable of its own — a live ticket
whose content would have been lost on closing it.

**Delta.** Added `### Chord encoding — absorbed from #296` carrying both: the
missing `\x1b[13;10u` meta-family sibling and the KKP host-dependency of
`\x1b[13;4u`. Both gained weight in the move, since a global chord that is dead
on a host is dead in every pane. Corresponding Done-when and Plan entries added.
#296 closed as superseded.

### 2026-09-20 — live probe, before any code

Ran `toggle-fullscreen --pane-id terminal_1` against this very workbench, held
it ~10s, reverted. Script and captures in the session scratchpad
(`fsprobe/{1-before,2-fullscreen,3-after}.json`). The revert was unconditional
and owned by the same shell, so it could not be orphaned by a dead caller.

    terminal_1  before      cols=95   fullscreen=False  focused=False
    terminal_1  fullscreen  cols=191  fullscreen=True   focused=True
    terminal_1  after       cols=95   fullscreen=False  focused=True

Four findings, three of which change the design:

- **`is_fullscreen` exists** in `list-panes --json`. Direction detection is a
  solved problem; the focus-record-as-state fallback is dropped from the Spec.
- **`--pane-id` pulls focus to the target.** Focus was on the draft before and
  on the right terminal during. The expand half of focus carry is free.
- **Exiting does not restore focus.** It stayed on the right terminal — the
  precise gap this issue exists to close.
- **Full width is 191 columns, not 186.** Worth noting only because the
  Done-when asserts against screen width rather than a literal, which is why
  that phrasing was chosen.

Operator confirmed the live result looked right ("works so nice"), which is
weak-but-real evidence for unknown 4.

**Unrelated observation for whoever implements this:** every entry in this
thread's `terminal-panes-<tag>` registry has a dead pid, while `pair term` is
plainly alive as `terminal_1`. `TerminalPaneIDs()` filters on `procutil.Alive`,
so it would currently return empty and pane classification would fall back to
the title arm that `layoutflow.go:63-71` documents as broken since #199 M3
(BR-48). This issue plans to reuse that registry for split-half selection.
Observed, not diagnosed — verify before depending on it.

### 2026-09-20 — Alt+Up/Alt+Down demoted to the draft

**Reason.** Operator: the chords are global only because stepping a rung
re-tiles the workbench and thus undid an accidental left/right split drag. With
ctrl+wheel resize filtered, that need has expired, and the global reach now just
denies the chord to the right pane's child.

**Delta.** New Spec section demoting both to draft-local via a binding scope
field, explicitly *not* by deleting the entries — `RenderLuaGlobalMaps`
generates the draft's keymaps from `GlobalBindings()`, so deletion would remove
the draft binding too. Unknown 2 dissolved as a consequence: a draft-local chord
cannot fire while the draft is hidden by fullscreen. Flagged that the premise
holds under couch but not standalone pair (#226 open, #268 unverified), and
bundled #226's one-liner.

Also corrected a spec imprecision found by the probe: collapse restores the
tiling that was in effect, not the layout's declared 50/50.

### 2026-09-20 — implementation-planning revision

**Reason.** Preserve the prior native-command evidence and turn the existing
operator decisions into an executable plan. The operator reiterated that the
maximize/restore command was already tested; no replacement resizing mechanism
is needed.

**Delta.** Native `toggle-fullscreen --pane-id` remains the implementation.
A disposable 160×40 four-pane session confirmed a split half expands from
80×20 to 160×40 and hides its sibling. Focusing the draft or sibling exits
fullscreen. After exit, multiple panes can report `is_focused=true`; use the
invoking `ZELLIJ_PANE_ID` for return focus. Session deleted after probing.
The live registry now includes pane 1 / PID 90791, matching its running term.

Review is unwrapped nvim and installs generated global maps itself; its local
menu binding shadows the proposed global. The generated Lua dispatcher must
execute the toggle from the invoking editor, not inject it into the draft.

The Kitty specification identifies modifier 10 as Shift+Super, not Shift+Alt;
document its absence for Return rather than adding an unrelated global chord.
Document negotiated-KKP support separately from physical host smoke evidence.
Add the standalone mouse-resize setting, but keep #226 open: its filter removal
and version-floor decision exceed that config change. The obsolete Done-when
row asking what draft rungs do while fullscreen is superseded by their already
specified draft-only scope.

Couch handles Ctrl+Space in `Console.dispatchInputCandidate` before forwarding
input to the zellij child (`couchkeys.ActionSwitch`, `ScopeEveryPane`), so native
pane fullscreen leaves the switcher available. The operator also confirmed
in this session that Ctrl+Space worked during the earlier fullscreen test.

The durable plan records these corrections and the remaining implementation,
verification and operator smoke steps. No production code changed during planning.

## Log

### 2026-09-20 — diagnostic audience correction
- 2026-09-20: closed — Operator accepted fullscreen smoke in another thread; disposable native fullscreen and actual-chord conformance passed; affected inventory/sessionwatch/context/slug suites and make pair pass; oversized records across all four providers plus ledger/log/config/SQLite regressions pass; real Codex root recognized after cutoff removal. Earlier full Go suite passed; fresh full/race checks running. make test has documented unchanged-baseline changelog ownership failure, and review suite passes in clean test environment.; review verdict: SHIP

Operator clarified that toggle/focus failures should be logged for the agent,
not surfaced as terminal errors or editor notifications: the user generally
cannot address them. The durable plan's Revisions section records the change
and replaces the handler-level UI error requirement with log-only diagnostics.

### 2026-09-20 — native probe evidence confirmed

Operator reconfirmed the prior native maximize/restore test and that Ctrl+Space
opened Couch while the right pane was fullscreen. These checks are settled;
remaining live acceptance concerns the implemented chord and focus restoration.

### 2026-09-20

Came out of a question about reviving the floating right pane to get a wider
terminal for carbonyl (#292). Floating turned out to be the wrong lever: the
#123 blocker was a plain left-drag on the pane frame (no modifier involved), and
zellij 0.45.1 still ships no gate for it — checked `setup --dump-config` directly
rather than assuming the version bump had helped.

`toggle-fullscreen` reaches the goal without any of that exposure and deletes
more than it adds. Operator chose the two-state toggle (50/50 ↔ full) over
keeping ~2/3 as a middle rung: one press each way is worth more than the
intermediate width.

Revised the same day to a global chord with focus carry — see `## Revisions`.
The chord-collision survey found two existing bindings, not one: the draft's
append-without-send (`init.lua:3540`) and the review pane's send menu
(`review.lua:706`). The operator's retirement decision was taken knowing only
the first.

### 2026-09-20 — implementation checkpoint

Passed full-flow change-code gates and entered branch
`000297-fullscreen-right-pane-toggle`. Native fullscreen planning, effect-order
transition, return storage/lock/diagnostics, and global wrapper/terminal/editor
routing are implemented in the working tree. The resize-burst implementation
and append-without-submit chain are removed. Failure handlers remain silent;
the executor records diagnostics once (ARCH-ORDER, ARCH-DRY).

Focused layout/parser tests and wrapper tests pass. Integration testing found
one real routing gap: terminal decoding consumed newly draft-only Alt+Up/Down
even when the action router passed them. The terminal now consults binding
scope before routing and forwards these keys to its child. Exhaustive terminal
and viewer tests still need their old global-rung assertions reconciled.
Full `make test` stopped at those stale scrollback mapping expectations;
`go test ./...` runs separately so that prerequisite failure cannot hide Go
coverage. Disposable live conformance and adversarial coverage are in progress.
No issue-close or operator smoke claim yet.

### 2026-09-20 — verification checkpoint

`go test ./...` passed. Focused changed-package suites and race checks for
layoutcmd/workbenchshortcut/termcmd passed. Exhaustive six-event transition
sequences, generated pane inventories, and a controlled overlapping invocation
using the real file lock pass. Parser fuzzing completed 219,540 executions in
three seconds without failures. Artifact environment-export tests and launcher
tests pass after adding the planned canonical fullscreen bindings.

`PAIR_LIVE_ZELLIJ=1 go test ./cmd/internal/layoutcmd -run
TestFullscreenZellijConformance -count=1 -v` passed on Zellij 0.45.1: draft,
agent, both same-right cases, a manually resized split, and native focus-away
behavior. Actual client input and exact geometry are checked, not just focus
flags. Live testing exposed redundant focus returning "already focused";
the reducer now skips Focus when return equals terminal (plan revision added).
Exhaustive sequences also caught premature failure events stopping an initial
state; initial invalid outcomes now remain no-ops as specified.

`make test` stopped at `test-changelog`: "viewer: process target is outside
selected owner directory". An unchanged HEAD archive built in
`/tmp/pair-297-baseline.uRzMy3` reproduces the identical failure. This is not
claimed green; the remaining integration suites run via `make -k test`.
Actual new-chord disposable smoke and operator smoke remain pending. The
in-place `make pair` build succeeded; operator was asked to smoke after relaunch.

### 2026-09-20 — implemented and ready for operator smoke

Implementation committed as `f7389846`. A fresh `go test ./... -count=1`
passed after the corrections. Final combined live command:
`PAIR_LIVE_ZELLIJ=1 go test ./cmd/internal/layoutcmd -run
'^TestFullscreen(ChordZellijLive|ZellijConformance)$' -count=1 -v` passed
(4.209s). Actual bytes traverse the production Zellij config, draft nvim map,
generic agent wrapper, terminal shells in both split halves, and a fullscreen
nvim child. Checks include focus, exact restored geometry, persistent child
screen, terminal-strip redraw, cleared return records, and absent diagnostics.
Disposable sessions are cleaned up; the operator's session was not mutated.

`make -k test` completed with the changelog baseline failure and a review-toggle
failure caused by inherited hosted-session retention settings. Both reproduce
on unchanged HEAD. With Pair/Couch/Zellij session variables removed only from
the test subprocess environment, `make test-review` passes completely; the
changelog failure persists. `make pair` and `git diff --check` pass.

Remaining: operator smoke after relaunch (draft cursor and Ctrl+Space; carbonyl
where available), then the SDLC-owned close review and publication. No close or
publish performed. The review-protocol target's shortcut correction is left as
an uncommitted inline proposal for the operator, per its datatype convention.

### 2026-09-20 — smoke accepted; reader cutoffs removed

Operator reported the new shortcut works in another thread and explicitly
accepted the smoke test. Per the subsequent scope expansion, removed arbitrary
inventory data-size cutoffs for every native provider, events/usage, incremental
validation, ledgers, Pair logs/configs, and SQLite results. Chunk sizes, explicit
bounded-runtime capability, stderr diagnostic bound, path/schema/identity checks
remain. Tests first reproduced 1 MiB and 8 MiB record failures, then passed;
Claude/Muse/Agy oversized scans/events, Codex initial/append and later evidence,
malformed/conflicting identity, large ledger/log/config and SQLite cases pass.
The full affected suite (sessioninventory/sessioninventorytest/sessionwatch/
contextcmd/slugcmd) passes, as do make pair and diff checks.

Read-only rebuilt inventory now recognizes this thread's actual Codex root as
resumable, with no root-identity diagnostic; before the fix it was absent.
The same query finds the root for the thread but still labels the binding
provisional. No binding/config was hand-written and no end-to-end relaunch fix
is claimed. This side quest removes the confirmed reader-limit cause, not the
separate #291 global detach problem. Full Go and race checks are in progress;
the previously reproduced changelog integration baseline exception remains.

### 2026-09-20 — close review and final verification

Main-session full Go suite completed successfully. Fresh affected suites and
inventory race checks pass; incremental JSONL fuzzing passed 222,976 cases.
SDLC close returned SHIP, no Critical/Important findings, and codecomplete.
Advisory BR-1/BR-2 addressed in the close commit: classify both core-concept
tables and reconcile plan checkboxes/evidence with accepted smoke and the
documented non-green make-test baseline. Reviewer's separately stopped broad
run does not supersede the completed main-session run. No production code
changed after review. Publication is not part of this local close.
