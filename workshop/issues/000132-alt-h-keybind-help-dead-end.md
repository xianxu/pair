---
id: 000132
status: working
deps: []
github_issue:
created: 2026-07-29
updated: 2026-07-29
estimate_hours: 3.18
started: 2026-07-29T22:16:55-07:00
---

# Alt+h keybind help is a circular dead end

## Problem

`Alt+h` is documented and advertised as "pop up the full keybind help". It is
not. The chain is:

```
Alt+h  →  zellij bind (config.kdl:163)  →  bin/pair-help  →  `pair -h`
```

and `pair -h` is a 21-line CLI usage block whose **last line reads**:

> In-session keybindings are on Alt+h.

So pressing the help key produces a message telling you to press the help key.

This is a **regression from #99 M5c**, which retired `bin/pair-shell` and
replaced its `usage()` with the native `launcher.UsageText()`. The shell version
carried a ~50-line `KEYBINDINGS` section — `Alt+Return`, `Alt+c`, `Alt+q`,
`Alt+/`, `Alt+x`, and an `Alt+h  pop up this help in a floating pane` line.
Recoverable verbatim: `git show 308d314^:bin/pair-shell`. The Go port kept the
binding and the pager and dropped the content.

It matters more than a typical doc gap because `Alt+h` is the **last** entry the
draft statusline preserves when the terminal narrows (`nvim/init.lua`,
`PAIR_CHEATS` — "At a minimum we try to keep Alt+h so the user always has a
discoverable path to the full keybind help"). It is the designed discovery path
for every other binding, and it is empty.

The Homebrew formula repeats the false claim in its `caveats`: "Run
`pair --help` for keybindings" — see **#131**.

## Spec

- `Alt+h` shows actual keybindings.
- Single source: the keybind list must be **derived**, not a hand-maintained
  restatement that can rot the same way. Candidate sources already exist —
  `cmd/internal/workbenchshortcut` holds the chord registry and
  `nvim/workbench_actions.lua` is already generated from it. Anything hand-typed
  reproduces this bug on the next refactor (`ARCH-DRY`, `ARCH-PURPOSE`).
- Decide whether `pair -h` keeps the concise CLI synopsis with keybindings behind
  a separate surface (e.g. `pair keys`, which `bin/pair-help` then pages), or
  whether `-h` grows a KEYBINDINGS section as the shell had. The former keeps CLI
  help short; the latter matches prior behaviour.
- Remove the circular "In-session keybindings are on Alt+h" line either way.
- Fix the formula caveats line (coordinate with #131).

## Done when

- `Alt+h` in a live session lists the workbench chords, dismissable with `q`/`Esc`.
- The list derives from the chord registry — adding a chord there surfaces it in
  the help with no second edit.
- No text anywhere claims `pair --help` shows keybindings unless it does.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. `sdlc estimate-source` reports the calibration
source as stale (recalibration tracked in #127), so the number is provisional but
uses the required method.

Design hours take the ×0.2 spec-quality discount **on every primitive except
`issue-spec`** — see the correction note below, which is the largest single line in
this block. The durable plan
(`workshop/plans/000132-alt-h-keybind-help-dead-end-plan.md`) carries the source
inventory with file:line, the three parser input classes with their real sites, the
(key, context) join rule and per-task TDD steps with test bodies written out, so
execution decisions are pre-resolved. Per v2.1 Step 6 the design buffer is halved
to **+15%** (the ×0.2 discount still applies to most primitives). Implementation
hours are 40% of the v2 table per v3.1 rule 5. `familiarity: 1.0` — same repo,
third issue today.

**v2.1 Step 2.5 — library-availability check (mandatory for `greenfield-go-module`,
recorded here).** The parsers consume Lua and KDL, and Go libraries exist for both
(`gopher-lua`, `kdl-go`). Checked and **declined**, with the design hours NOT
halved, because the check's premise — "the implementer finds a library and the
design dialogue short-circuits" — does not hold here:
- **Lua:** `gopher-lua` *evaluates* Lua. Extracting `(lhs, desc)` needs **static**
  reading of source text; evaluating `init.lua` would require stubbing the whole
  `vim` API. A library makes this harder, not easier.
- **KDL:** `kdl-go` is genuinely viable and would parse `config.kdl` properly. It is
  declined for footprint: the need is 2 `Run` binds out of 20, a ~20-line
  extraction, against a repo that is deliberately stdlib-only (`go.mod`). Two new
  dependencies for ~60 lines total is the worse trade.
- Because the design dialogue was **not** short-circuited (the contract still had to
  be specified in full — three input classes, the arg-position rule, the
  reconciliation invariants), Step 2.5's halving does not apply. Recorded so the
  derivation is complete rather than silently skipped.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.75 impl=0.08
item: greenfield-go-module design=0.20 impl=0.24
item: smaller-go-module design=0.04 impl=0.14
item: smaller-go-module design=0.04 impl=0.14
item: smaller-go-module design=0.04 impl=0.14
item: greenfield-go-module design=0.20 impl=0.24
item: smaller-go-module design=0.03 impl=0.12
item: smaller-go-module design=0.02 impl=0.10
item: cross-cutting-refactor design=0.04 impl=0.10
item: atlas-docs design=0.03 impl=0.06
item: milestone-review design=0.02 impl=0.20
design-buffer: 0.15
total: 3.18
```

Arithmetic: design 1.41 + impl 1.56 + buffer (1.41 × 0.15 = 0.2115) = **3.1815 → 3.18**.

Item mapping, in plan order:
- `issue-spec` **design 0.75, undiscounted** — this file, the durable plan, two
  plan-quality rounds and three `## Revisions` entries. **Correction:** the first
  block priced this at 0.15 by applying the ×0.2 discount to it, which is circular
  — v2 Step 3 discounts a primitive when *the plan pre-resolves that primitive's
  decisions*, and `issue-spec` **is** the plan-authoring cost. The model's own
  anchor confirms it: in charon #13's backfill, `Sketch (Pensive)` keeps its full
  band while M1/M2/M5 carry `×0.2 design`. It matters materially, not as a rounding
  nit, because `sdlc actual` measures from the claim commit — so the design work,
  both gate rounds and these revisions are all *inside* the measured window, and
  pricing them at 9 minutes guaranteed a phantom over-run. Taken at 0.75 (low-to-mid
  of the 0.5–1.5 band: the brainstorm was short because the operator delegated the
  one open question, but the plan is long and test-complete).
- `greenfield-go-module` (0.20/0.24) — the `keyhelp` parsers: three-way `KeymapScan`,
  the arg-position rule, property strategy, reconciliation, and `Run` vs
  `WriteChars`/`Write <n>;` including the Write-before-Run shape at
  `config.kdl:157→163`. Design at the **band midpoint** pre-discount (1.0 of
  0.5–2.0) — the first block's prose said "low end", which described a different
  number than it used.
- `smaller-go-module` (0.04/0.14) — `Render` + reuse of `launcher/list.go:88`'s
  display-width helper.
- `smaller-go-module` ×2 (0.04/0.14 each) — **Tasks 4 and 4b, now separate items.**
  The first block merged them; they have separate files, tests and commits, and
  Task 4 additionally carries the positional-composite-literal sweep (`Help` is
  added positionally, so every unkeyed `GlobalBinding` literal must be found and
  fixed) plus the generated-Lua no-diff assertion.
- `greenfield-go-module` (0.20/0.24) — **Task 5, reclassified up** from
  `smaller-go-module`. That primitive means "well-specced; mirror or extend", and
  Task 5 mirrors nothing: `SourceReader`, a new curated `Catalog`, four
  bidirectional drift tests, and the classify-until-green pass over ~30 real
  keymaps that the plan insists be resolved "by CLASSIFYING, not by weakening". The
  plan calls it the task that makes #132 unrepeatable; pricing it at ~11 minutes
  contradicted that.
- `smaller-go-module` (0.03/0.12) — `Sections` join + display mapping.
- `smaller-go-module` (0.02/0.10) — `keyscmd`, dispatcher row, `--center`.
- `cross-cutting-refactor` (0.04/0.10) — `bin/pair-help` (delete the byte-width awk
  block, exit-0 contract), `UsageText`, and the rendered-shape sweep across
  README/atlas/zellij/bin/nvim. **README is counted here**, not in `atlas-docs`.
- `atlas-docs` (0.03/0.06) — `atlas/architecture.md` (+ `index.md` if a file is
  listed). Scoped to atlas alone to remove the double-count with the sweep above.
- `milestone-review` (0.02/0.20) — one boundary plus the live `Alt+h` check, impl at
  top-of-band (0.5 × 0.4). **Unbudgeted tail, stated rather than padded:** this
  covers one clean pass. #133's boundary review returned FIX-THEN-SHIP with two
  Important findings, and a comparable fix loop here has no headroom in the block
  (impl carries no buffer by construction). If that happens it will show as an
  over-run; inventing a primitive to pre-absorb it would be padding.

Plausibility note. At 3.18 this is the largest of today's three issues, which is
right: #133 and #130 removed code, this adds a package with two parsers over
foreign syntax.

The signal that cuts against it, and an inversion worth recording. #133 (est 1.60 /
actual 0.90, 1.8×) and #130 (est 4.31 / actual 1.78, 2.42×) are the two most
adjacent ledger rows — same repo, same day, same v3.1 model — and both over-ran the
estimate. Naively that argues 3.18 is high. **But those two had substantial
parallelism to compress, and this plan is almost purely serial:** Task 5 depends on
2/3/4/4b, 6 on 5, 7 on 6, 8 on 7. There is nearly no fan-out for concurrent
execution to exploit, so whatever compression drove those two over-runs should not
reproduce here. I am therefore *not* pre-discounting toward the 1.8–2.4× pattern.
Logged explicitly so that if the actual still lands near 1.0–1.2, it is clean
third-consecutive evidence for #127's recalibration — and not confounded with the
`issue-spec` category error corrected above, which was itself worth ~0.7h of
phantom over-run.

## Plan

Durable design: **`workshop/plans/000132-alt-h-keybind-help-dead-end-plan.md`**
(authored via `superpowers-writing-plans`). Nine tasks in three chunks; single
review boundary — no `Mx` tags, one branch, one close.

- [ ] Chunk 1 — pure core: `Binding`/`Section`/`Render` (per-section display-width
      alignment, reusing `launcher/list.go:88`'s width helper), `ParseNvimKeymaps`,
      `ParseZellijRunBinds`.
- [ ] Chunk 2 — `GlobalBinding.Help` for the 8 chords that have no wording
      anywhere; `SourceReader` over `runtimebundle.EmbeddedAsset`; the `Catalog`;
      and the **bidirectional drift tests** that make #132 unrepeatable.
- [ ] Chunk 3 — `pair keys`, `bin/pair-help` repointed, `UsageText`
      de-circularized, docs/atlas, bundle check, live check, full `make test`.

## Log

### 2026-07-29

- Confirmed the bug end to end before designing: `pair -h` is 21 lines, its last
  line is "In-session keybindings are on Alt+h.", and the only keybinding it
  mentions anywhere is `Alt+h` itself. `config.kdl:163` → `bin/pair-help` →
  `pair -h` all work mechanically — the binding fires, the floating pane opens,
  `less` pages, `q`/`Esc` dismisses. **The mechanism is fine; the content is the
  bug.** (Operator observed "Alt+h seems to be working" — both true, different
  things.)
- `PAIR_CHEATS` (`nvim/init.lua:2137`) confirms the design intent: `Alt+h` is
  first in the priority list, and the comment says it is kept last-to-drop "so the
  user always has a discoverable path to the full keybind help".
- **Design decision (operator delegated the Spec's open question): `pair keys`,
  not a KEYBINDINGS section in `pair -h`.** CLI usage is read from a shell,
  keybindings are in-session — different audiences — and `-h` would grow to ~70
  lines. It also gives #131's Homebrew caveat an accurate target to point at.

## Revisions

**2026-07-29 — the Spec's single-source premise was wrong; there are four sources.**
The Spec said `cmd/internal/workbenchshortcut` "holds the chord registry" and could
be the derivation source. Verified during design that it cannot, alone:
`globalBindings` has **8** entries and **no** label field; the terminal-pane-local
chords live in a `switch` inside `Decide`, not a table; `zellij/config.kdl` has 2
user-facing `Run` binds (`Alt+h`, `Alt+l`) that nvim never sees; and the
draft-editing keys users most need — `<M-CR>`, `<M-q>`, `<M-BS>`, `<C-c>`,
`<M-Left>/<M-Right>` — are 34 `vim.keymap.set` calls in `nvim/init.lua`.

The find that makes this tractable: **every one of those keymaps already carries
`desc = 'pair: …'`**, and the strings are already good help prose ("send buffer +
clear", "send ESC to agent (interrupt stream)", "queue current draft for later
(back of queue)"). So wording can genuinely derive — no help prose authored twice.
The asymmetry to watch: the 8 global chords reach nvim through the *generated*
`workbench_actions.lua`, not literal `keymap.set` calls, so parsing `init.lua`
alone silently misses them. That is why the plan carries a parsed-count assertion.

**2026-07-29 — Done-when's "no second edit" deliberately revised.** Taken
literally it means auto-including every keymap, which would publish `autopair `,
`jump over `, `quit blocked`, `cycle completion or insert tab` and `smart-delete
empty pair` as workbench keybinding help — worse than the current bug. The plan
keeps *wording* derived (nothing to re-type) but makes *inclusion* explicit, and
enforces classification with a bidirectional drift test: an unclassified key fails
the build, and a catalog entry whose source row is gone fails too. The property
that actually matters — **cannot rot unnoticed**, which is what #99 M5c lacked —
is preserved and now tested. Recorded rather than silently reinterpreted, per the
#133 lesson about making Done-when match reality on purpose.
