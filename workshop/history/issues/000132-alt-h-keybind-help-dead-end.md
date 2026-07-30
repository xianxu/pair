---
id: 000132
status: done
deps: []
github_issue:
created: 2026-07-29
updated: 2026-07-29
estimate_hours: 3.18
started: 2026-07-29T22:16:55-07:00
actual_hours: 0.37
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

- [x] Chunk 1 — pure core: `Binding`/`Section`/`Render` (per-section display-width
      alignment, reusing `launcher/list.go:88`'s width helper), `ParseNvimKeymaps`,
      `ParseZellijRunBinds`.
- [x] Chunk 2 — `GlobalBinding.Help` for the 8 chords that have no wording
      anywhere; `SourceReader` over `runtimebundle.EmbeddedAsset`; the `Catalog`;
      and the **bidirectional drift tests** that make #132 unrepeatable.
- [x] Chunk 3 — `pair keys`, `bin/pair-help` repointed, `UsageText`
      de-circularized, docs/atlas, bundle check, live check, full `make test`.

## Log

### 2026-07-29
- 2026-07-29: closed — Alt+h now shows real keybindings. pair keys renders 34 bindings in 5 groups, wording DERIVED from four sources (nvim desc, GlobalBinding.Help, RoleBinding.Help, catalog for the 2 zellij Run binds); bin/pair-help pages it; UsageText points at pair keys instead of back at Alt+h. Ran the real chain: bin/pair-help end-to-end prints bindings, exit 0. Guards mutation-verified, not just passing: adding an undocumented keymap to nvim/init.lua fails TestEveryNvimKeymapIsClassified; the parser reconciles 30 resolved + 1 dynamic + 3 unresolved = 34 desc lines against the real file with an invariant that no Key begins "pair: " (the naive second-quoted-arg rule misassigns the desc as the key at init.lua:3872 while a count guard still passes); TestRoleLocalWordingComesFromRoleTableNotNvimNoOp pins that Alt+t reads "new terminal tab" and never "disabled in draft"; Alt+k renders twice with per-context wording; pair keys exits 0 on source failure via an injected failing reader (set -euo pipefail would otherwise kill the floating pane). Drift tests read the working tree because assets/ is gitignored and go test never regenerates it, with TestEmbeddedSourcesMatchTree tying the bundle to it. textwidth extracted from launcher/list.go rather than copied; launcher delegates, tests unchanged. make test exit 0 (termcmd/wrapcmd pty tests need the sandbox disabled; both untouched). Also corrected #133 shipped guidance: --no-ignore is not a valid ugrep flag (exits 2, prints nothing, reads as clean); re-verified #133 bundle clean with --no-ignore-files and /usr/bin/grep, fixed the archived issue, and added the lesson.; review verdict: FIX-THEN-SHIP

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

- **Implemented.** All three chunks done. `pair keys` renders 34 bindings in five
  groups; `Alt+h` pages it; `pair -h` points at it instead of at itself.
  `make test` **exit 0**.
- Evidence the guards actually bite, not just pass:
  - **Parser (PQ-1).** Reconciles against the real `init.lua`: 30 resolved + 1
    dynamic + 3 unresolved = 34 `desc = 'pair: '` lines, with invariants that no
    resolved Key begins `pair: ` and the unresolved set stays exactly the three
    known unquoted-lhs sites. A property test varies mode form, whitespace and
    arg-2 form.
  - **Drift (PQ-3).** Mutation-tested: adding an undocumented keymap to
    `nvim/init.lua` fails `TestEveryNvimKeymapIsClassified`. Restored after.
  - **Wording join (PQ-2).** `TestRoleLocalWordingComesFromRoleTableNotNvimNoOp`
    asserts the rendered help contains "new terminal tab" and never "disabled in
    draft"; `Alt+k` renders twice with different wording per context.
  - **Exit contract (PQ-6).** `pair keys` exits 0 on a source failure with a
    visible diagnostic body, tested through an injected failing reader — under
    `set -euo pipefail` a non-zero exit would kill the floating pane.
  - Ran the real chain: `bin/pair-help` end to end prints the bindings and exits 0.
- `textwidth` extracted from `launcher/list.go` (Step 3a) rather than copied;
  `launcher` delegates and its tests pass unchanged.
- **A correction to #133's shipped guidance, found here.** #133's Done-when said to
  verify the gitignored bundle with `grep -rn --no-ignore`. That flag **does not
  exist** in this environment's grep (ugrep 7.5.0 — it is `--no-ignore-files`);
  ugrep exits 2 and prints nothing, which is indistinguishable from "no matches".
  With `2>/dev/null` in the pipeline it read as a clean pass. Re-verified #133's
  actual claim with `--no-ignore-files` and with `/usr/bin/grep`: the bundle **is**
  genuinely clean, so the conclusion held — but the instruction was a false-pass
  generator. Fixed in the archived #133 issue and in this plan, and written up in
  `workshop/lessons.md` as "a verification command must itself be verified".
- README's keybinding table is still hand-maintained prose. Left as-is
  deliberately: it carries context a one-line desc cannot, and it now says the
  derived list is authoritative. Making it generated is a separate issue, not a
  silent scope expansion here.

## Revisions
**2026-07-29 — close review (FIX-THEN-SHIP): six Important findings fixed in the
close commit.** The two that mattered most were coupled, and the second explains the
first:

- **I1: `Alt+←`/`Alt+→` (terminal tab switching) were missing** from a section
  titled "Terminal tabs". The design's "four sources, verified during design" was
  itself an incomplete inventory: the terminal chord surface is split across **two**
  seams — `workbenchshortcut.Decide`'s terminal branch and
  `termcmd.handleTerminalChord` (`run.go:484-489`) — and their sets differ
  (`handleTerminalChord` has AltLeft/AltRight; AltR is special-cased elsewhere). So
  there are **five** sources, not four.
- **I2: the test that should have caught I1 hardcoded what it claimed to derive.**
  `TestRoleBindingsCoverTerminalSwitch` documented itself as covering "every chord
  Decide actually handles" but iterated a hand-written slice — one hand-maintained
  list checked against another. Now derived from `Decide` over the chord space, plus
  a **mirror test in `termcmd`** for the other seam. Mutation-verified, and the
  mutation taught something worth recording: deleting the `Alt+←` row leaves the
  `workbenchshortcut` test **green** and fails only the `termcmd` mirror, because
  `Decide` never sees that chord. Neither test is sufficient alone; the comment now
  says so instead of overclaiming.
- **I3: four stale `pair -h` claims survived my sweep** (`atlas/architecture.md:19`
  and `:407`, `zellij/config.kdl:159`, `atlas/go-migration-inventory.md:138`) — one
  of them directly contradicting the atlas paragraph this branch added. Cause: the
  sweep used `grep --no-ignore`, the invalid ugrep flag described below. The lesson
  this branch added about verification commands caught me in the same session that
  wrote it.
- **I4:** `"keys"` was missing from `dispatcher_test.go`'s implemented-set list (a
  regression there would make `Alt+h` launch a session inside the floating pane), and
  nothing checked that `bin/pair-help` invokes `pair keys` — PQ-6's second sub-point,
  which I had marked "addressed" while delivering only the awk/pipefail half. Both
  now covered, the shim by running the real script with `pair`/`less`/`tput` stubbed
  on `PATH` (needs the sandbox disabled, like the pty tests).
- **I5:** README's CLI synopsis had silently diverged from `pair -h` on the very
  subcommand this issue adds.
- **I6:** the anti-rot guard depended on the exact literal `desc = 'pair: `, and the
  reconciliation counted with the *same* literal — so a keymap written
  `desc = "pair: …"` would be invisible to both and could not be flagged. The
  reconciliation now counts with a tolerant regexp and requires the two counts to
  agree.

**A content bug the review surfaced indirectly.** Chasing its note that README and
the derived help disagreed about `Alt+q`, the code settles it: `queue_current()`
calls `queue_push_front` (`init.lua:2593`), and the retired shell help said "push
current buffer to queue front (+1)". So the **`desc` was wrong** — "back of queue" —
and the derived help faithfully reproduced the error. Fixed at the source, which
corrects `pair keys` and the editor's own `:map` output together. That is the
derivation working as intended: one edit, both surfaces.

**Also fixed here, in already-merged work:** #133's Done-when told readers to verify
the gitignored bundle with `grep -rn --no-ignore`. That flag does not exist in this
environment (ugrep 7.5.0 — it is `--no-ignore-files`); ugrep exits 2 and prints
nothing, indistinguishable from "clean", and `2>/dev/null` hid the error. #133's
substance re-verified with two working tools and it is genuinely clean, but the
instruction was a false-pass generator. Corrected in the archived issue and written
up in `workshop/lessons.md`.


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
