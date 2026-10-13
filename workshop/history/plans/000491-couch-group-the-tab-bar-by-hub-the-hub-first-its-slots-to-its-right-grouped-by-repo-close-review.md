# Boundary Review — pair#491 (whole-issue close)

| field | value |
|-------|-------|
| issue | 491 — couch: group the tab bar by hub — the hub first, its slots to its right, grouped by repo |
| repo | pair |
| issue file | workshop/issues/000491-couch-group-the-tab-bar-by-hub-the-hub-first-its-slots-to-its-right-grouped-by-repo.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3a8e178f21dba73ea2feac28a44dd91a8d5a9da8..32564c8ea8655293920b4992900470d68eca3b04 |
| command | sdlc close --issue 491 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T16:31:33-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I'm checking whether a folded group with no visible repo name produces an ambiguous `+N` cell. Running a quick probe test next.

Generating the embedded runtime assets so the couchtty tests can build, then rerunning the probe.

Confirmed one real issue (a leaderless group folds to a bare `+2` that reads as belonging to the previous repo). Now running the package tests and checking for the planned couchcmd wiring test.

No test calls `SetHub`, so the planned couchcmd wiring test is missing. Next I'm checking one of the implementer's mutation claims (the hub group split in `chipGroups`).

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The ordering and layout work is sound. Hub ordering lives in one function, `PresentThreads`, so the tab bar, the switcher and clicks all use the same order. The fitting is pure. The two operator revisions (no brackets, `[…]` stays the active marker) and the decision to keep the broadcast cells leading are written down in the plan's Revisions and the Log. The couchtty and couchcmd tests pass at head, and a mutation I ran on the `chipGroups` hub split does turn a test red.

Two things keep this from SHIP:
- **Bare `+N` count:** the narrow-width collapse draws a `+N` with no repository name for any group whose `:0` isn't in the group. In the normal one-hub setup that is the hub's own repository's numbered slots, so the count reads as if it belonged to the repository before it.
- **Untested wiring:** the hub's path from couchcmd into the Console, and its survival when the menu is rebuilt, have no test, though the plan promised one.

## 1. Strengths
- **One order source (ARCH-DRY):** `hubSection` (`thread_presentation.go:180`) is the last step of `PresentThreads`, so the hub can only be placed in one spot. `orderedMenuInventory` and both render paths pass `MenuState.Hub` into it.
- **Changes go through the reducer (ARCH-ORDER):** the hub arrives as a `MenuEventHub` event. The new `reorderMenuInventory` is added to the inventory-reader guard's allowlist. `TestMenuHubEventOrdersTheSwitcher` covers both arrival orders: hub after the inventory, then a fresh inventory after the hub.
- **Pure fitting (ARCH-PURE):** `fitChips`, `layoutChips` and `chipGroups` are pure, and `RenderStatusRow` still records click spans from the same pass that clips. The span test in `hub_section_test.go:150` checks that every span maps to its own thread and that a `+N` count is not clickable.
- **Narrowed fold rule:** whole groups fold only while a kept chip (hub, active, red, belled) is past the edge. This keeps the existing guarantee that a cut-short tab stays clickable. The Log records why.

## 2. Critical findings
None.

## 3. Important findings
- **Bare `+N` for a group with no `:0` in it** (`reserve.go:583-608`, `layoutChips`). When a group is collapsed, every unkept slot with `SlotNumber > 0` is hidden, including the group's first member. A group with no `:0` in it (the hub's own repository when the hub is `ops`, or a repository whose primary isn't open) then collapses to a bare `+N`.
  - I reproduced it at head: `ops  ariadne  :1  blog  ops:1  :2  pair  :1  :2` narrows to `ops  ariadne  :1  blog  +2  pair  +2`. That `+2` reads as two more `blog` slots.
  - This is the default one-hub layout whenever the hub's repository has numbered slots, and no test covers it (`hubModel` has no such group).
  - Fix: when collapsing, keep the group's first member drawn with its full label (`ops:1  +1`), or label the count with the repository. Add a narrow-width case with an `ops:1`/`ops:2` group.
- **No test for the hub wiring.** Plan Steps 1 and Tests ("couchcmd: `startMessageService` hands the Console the hub root") promise one. Nothing in the tree calls `Console.SetHub` from a test, and `message_service.go:378` has no coverage.
  - I mutated `ensureMenuLocked` (`console_palette.go:38-41`) to drop `Hub` while rebuilding the menu, and the whole couchtty package stayed green. Losing the hub there would silently return the tab bar to the old layout.
  - Fix: add a Console-level test (SetHub, then a menu reset, then the status model leads with the hub) and a couchcmd test that the hub root reaches the Console.

## 4. Minor findings
- At the clip edge a count cell is cut mid-token (`pair  +` at width 35). Consider dropping a count cell that doesn't fit whole.
- In `README.md`, the paragraph added near line 798 has one overlong line (the inline example plus "In the switcher …"). Rewrap it.
- The gap choice (`"  "` vs `sectionGap`) is written twice, in `RenderStatusRow` and `chipsWidth`. A small `cellGap(cell)` helper would keep the width estimate and the drawing in step (ARCH-DRY).
- The hub is matched by exact path string (`p.Path == hub`). The Log says this is the same value `isHub` already uses. A note on symlinked or `/private` paths would help later readers.

## 5. Test coverage notes
- Covered: ordering for a primary hub, a numbered hub, no hub, and a hub not in the inventory; switcher indentation; one-section and two-section rows; collapse and fold with a red chip; spans.
- Missing: a group with no `:0` under collapse, the active chip under folding, Console-level and couchcmd hub wiring, and `ensureMenuLocked` keeping the hub.

## 6. Architectural notes
- **ARCH-DRY:** pass. One order, and `chipTail` is extracted. The gap duplication is Minor, above.
- **ARCH-PURE:** pass.
- **ARCH-PURPOSE:** pass for the one-hub scope the Spec asks for. Membership is implicit inside `hubSection` rather than a separate `hubOf` function. The Log explains why, and the plan's revision should say so (section 7).
- **ARCH-MOCK:** N/A. No external dependency is added.
- **ARCH-CONSTRAINTS:** pass. `fitChips` is O(groups × n²) because `keptEnd` calls `chipsWidth` for each cell. Over a few dozen chips that is fine; worth a comment if chip counts grow.
- **ARCH-SECURE:** pass. Labels are still sanitized, and the hub path comes from this process's own cwd.
- **ARCH-ORDER:** pass. State changes go through `ReduceMenu`. The gap is the untested preservation on menu reset (Important, above).
- **ARCH-FUNERAL:** pass. Nothing durable is created; `MenuState.Hub` is an in-memory string set once per process.

## 7. Plan revision recommendations
- The Revisions say `PresentThreads` takes a membership function `hubOf(row)`, but the code puts the membership rule inside `hubSection` with no function. Add a Revisions entry saying that, matching the Log.
- §4 says a group's "unprotected `:N` chips" collapse. Revise it to say the group's first member keeps its full label once the leaky case is fixed.

```findings
findings:
  - id: new
    severity: Important
    family: collapsed-count-loses-owner
    title: |
      A collapsed group with no :0 in it draws a bare +N that reads as the previous repository's
    detail: |
      layoutChips hides every unkept SlotNumber>0 chip when collapsed, including the group's first member. In the default one-hub layout (hub ops, with ops:1 and ops:2) the row narrows to "ops  ariadne  :1  blog  +2  pair  +2", which misattributes the count to blog. Keep the group's first member drawn with its full label (ops:1  +1) and add a narrow-width test case for it.
  - id: new
    severity: Important
    family: wiring-untested
    title: |
      No test for the hub reaching the Console (SetHub, startMessageService, ensureMenuLocked keeping Hub)
    detail: |
      The plan promises a wiring test. No test calls Console.SetHub. Mutating ensureMenuLocked to drop Hub left the couchtty package green, so a lost hub would silently bring back the old layout. Add a Console-level test and a couchcmd test.
  - id: new
    severity: Minor
    family: clip-mid-token
    title: |
      A +N count cell can be clipped mid-token at the edge (pair  +)
  - id: new
    severity: Minor
    family: shared-helper
    title: |
      The gap choice is written twice (RenderStatusRow and chipsWidth); extract a cellGap helper (ARCH-DRY)
  - id: new
    severity: Minor
    family: plan-code-drift
    title: |
      Plan Revisions still describe a hubOf membership function that the code doesn't have
```

---

## Re-review — 2026-10-12T16:40:59-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 491 — couch: group the tab bar by hub — the hub first, its slots to its right, grouped by repo |
| repo | pair |
| issue file | workshop/issues/000491-couch-group-the-tab-bar-by-hub-the-hub-first-its-slots-to-its-right-grouped-by-repo.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3a8e178f21dba73ea2feac28a44dd91a8d5a9da8..0134f9dbd86e2b809da5c33d14953f9e59a61ff0 |
| command | sdlc close --issue 491 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T16:40:59-07:00 |
| verdict | SHIP |

## Review

Running the tests now. A generated asset directory is missing from this checkout, so I'm working out how the Makefile builds it.

Generating the missing runtime bundle, then running the couch tests.

Tests pass. Next I'll revert each round-1 fix and check that its test fails.

Mutating three sites in turn: the hub kept in `ensureMenuLocked`, the first-member keep, and the whole-count drop.

All three reverts fail their tests. Now I'll check whether the trailing count from a whole-group fold can still sit after another repository's chip.

The fixes check out. Last check: whether `hubOf` still appears in any doc after BR-5's plan fix.

```verdict
verdict: SHIP
confidence: high
```

All five round-1 findings are fixed, and I confirmed each fix by reverting it. Reverting the BR-1 rule (`i > span[0]` back to `a.SlotNumber > 0`) fails `TestRenderStatusRowCollapsedCountFollowsItsOwnRepository`. Dropping the Hub carry-over in `ensureMenuLocked` fails `TestConsoleHubLeadsTheTabBarAcrossTheFirstMenuBuild`. Removing the whole-or-nothing count check fails `TestRenderStatusRowDropsACountThatDoesNotFit`. `couchtty` and `couchcmd` both pass at head. The checkout had no `runtimebundle` assets, so I generated them with `runtimebundle-generate` from `Makefile.local` first. Two new findings remain, both Minor, so nothing blocks the close. The first is a second instance of the "count reads as the wrong repository" family. It happens in the whole-group fold, which only runs at narrow widths when a red, belled, active or hub tab would otherwise be off-screen. The second is a leftover `hubOf` claim in the issue's Plan checklist.

## 1. Strengths
- `layoutChips` (`reserve.go:517`) now keys collapsing on position within the group instead of slot number. That fixes groups with no `:0` without adding a special case.
- `cellGap` (`reserve.go:433`) is now the one source for gaps. Both the drawing code and `chipsWidth` read it (ARCH-DRY).
- `messageService.showHub` (`message_service.go:393`) narrows what the code depends on to `interface{ SetHub(string) }`. That makes the handoff testable without a Console.
- The Console-level test drives the real lazy `ensureMenuLocked` path followed by an inventory event. It covers the exact way the hub could be lost.
- Width behaviour is pure and tested without a terminal (`fitChips`, `layoutChips`, `chipsWidth`), and spans come from the same pass that draws.

## 2. Critical findings
None.

## 3. Important findings
None.

## 4. Minor findings
- **Repeat of family `collapsed-count-loses-owner` (2nd finding).** `reserve.go:536` puts the whole-group fold's combined `+N` after the last chip drawn. That chip is often a kept chip from another repository. I probed `hubModel` with `pair:2` red:
  - At width 29 it draws `ops  ariadne  +2  pair:2  +3`. The `+3` counts blog, pair and pair:1, but it reads as "pair has 3 more".
  - The pinned case `ops  pair:2  +6` has the same reading.
  - At widths 30–31 the fold's `+2` is dropped by the whole-or-nothing rule, so `pair` and `pair:1` disappear without any count.
  - The new doc comment at `reserve.go:465`, "a count always follows a chip of its own repository", is false for this path.

  **Rule:** every `+N` must either directly follow a chip of the single repository it counts, or look clearly different from a per-repository count. For example, put the overflow count before the kept chips, or after a section-style separator. If an overflow count can't be drawn, fold one more group rather than dropping it silently. Apply this in `layoutChips`/`fitChips` for both kinds of count, not just at this site.
- **Repeat of family `plan-code-drift`.** BR-5 fixed the plan Revisions, but the checked item at issue file line 47 still claims "`hubOf` membership = the single hub today". **Rule:** when a revision withdraws a named entity, grep the issue, plan, atlas and README for that name in the same change.

## 5. Test coverage notes
- The three round-1 fixes each have a regression test, and I confirmed each one goes red when its fix is reverted (see the summary).
- The couchcmd test covers `showHub`'s choice of value, but not the call site in `startMessageService`. Deleting that one line would not be caught. That's acceptable given how much setup `startMessageService` needs; the Console-side test covers the state being kept.
- No test pins what a combined fold count means, or that a dropped fold count still leaves the hidden slots signalled (finding above).

## 6. Architectural notes
- **ARCH-DRY:** pass. The gap logic is shared; there is one order source (`PresentThreads`) and one fitting function.
- **ARCH-PURE:** pass. The fitting and ordering code is pure, and the hub handoff is a thin seam with an interface.
- **ARCH-PURPOSE:** pass. Everything that displays the order reads it from `PresentThreads`, including the reattach inventory through `orderedMenuInventory`. The leftover `hubOf` text is a missed sweep, recorded above.
- **ARCH-MOCK:** N/A. No external binary or service is added; the hub root comes from the existing `hubWorktree`.
- **ARCH-CONSTRAINTS:** pass. `fitChips` relays out at most once per group in each of its two loops, over a few dozen chips. It does no IO.
- **ARCH-SECURE:** pass. Labels still go through `rowtext.Sanitize`, and the hub is a process-local path.
- **ARCH-ORDER:** pass. Hub changes go through `ReduceMenu`'s `MenuEventHub`, and the lazy rebuild keeps the hub; that's now tested.
- **ARCH-FUNERAL:** pass. Nothing durable is created; `MenuState.Hub` is in-memory.

When several hubs arrive, the fold count's attribution will matter more. A trailing count after the ` │ ` separator would read as belonging to the last section.

## 7. Plan revision recommendations
- Update the issue's Plan checklist item (line 47) to say membership lives inside `hubSection`, matching the round-1 plan revision.
- Add a revision to §4 step 3 saying how the combined fold count is placed or marked so it can't be read as the preceding chip's repository count. Also say what happens when that count doesn't fit.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      layoutChips keeps i == span[0]; reverting fails TestRenderStatusRowCollapsedCountFollowsItsOwnRepository (verified).
  - id: BR-2
    disposition: addressed
    note: |
      Dropping Hub in ensureMenuLocked fails TestConsoleHubLeadsTheTabBarAcrossTheFirstMenuBuild (verified); showHub is tested in couchcmd.
  - id: BR-3
    disposition: addressed
    note: |
      Removing the whole-or-nothing check fails TestRenderStatusRowDropsACountThatDoesNotFit (verified).
  - id: BR-4
    disposition: addressed
    note: |
      cellGap at reserve.go:433 is read by both RenderStatusRow and chipsWidth.
  - id: BR-5
    disposition: addressed
    note: |
      The plan Revisions now say there is no hubOf. The issue checklist still names it; raised separately as a plan-code-drift repeat.
findings:
  - id: new
    severity: Minor
    family: collapsed-count-loses-owner
    title: |
      The whole-group fold count trails another repository's kept chip (ops  ariadne  +2  pair:2  +3)
    detail: |
      2nd finding in this family. layoutChips appends the combined hidden count after the last drawn chip, often a full-labelled kept chip from another repository, so +3 (blog, pair, pair:1) reads as pair's count; at widths 30-31 the fold count is dropped and two slots vanish without a count. The comment at reserve.go:465 claims counts always follow their own repository. Rule: every +N either directly follows a chip of the single repository it counts or looks clearly like the row's overflow, and an overflow count that can't be drawn means folding one more group, not dropping it. Apply this to both kinds of count in layoutChips/fitChips.
  - id: new
    severity: Minor
    family: plan-code-drift
    title: |
      The issue's Plan checklist still claims a hubOf membership function
    detail: |
      2nd finding in this family. Issue file line 47 is checked and names hubOf; the code puts membership inside hubSection. Rule: when a revision withdraws a named entity, grep the issue, plan, atlas and README for that name in the same change.
```
