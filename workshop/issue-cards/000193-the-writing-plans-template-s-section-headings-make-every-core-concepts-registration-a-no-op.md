---
id: 000193
status: open
created: 2026-09-05
updated: 2026-09-05
estimate_hours:
github_issue:
---

# The writing-plans template's section headings make every Core-concepts registration a no-op

## Problem

`TestCoreConceptsContract` turns a plan's **Core concepts** table into an
executable contract. Its parser matches the section heading with string
EQUALITY:

    case line == "### Pure entities":
    case line == "### Integration points":

The `superpowers-writing-plans` skill's template writes those headings with a
parenthetical gloss:

    ### Pure entities (the conceptual core)
    ### Integration points (where pure meets the world)

Those never match, so `kind` stays empty, every row is skipped, and the plan's
whole table asserts nothing — while the test PASSES.

Found in `pair#172`: registering that plan in `conceptPlans` changed nothing, and
the suite stayed green after planting a row naming `NoSuchSymbolAtAll`. The
registration looked done and was not. Renaming the headings to the bare form made
eleven previously invisible rows appear at once.

The failure is silent in both directions, which is what makes it worth fixing
rather than remembering: a plan written from the template registers as a no-op,
and nothing anywhere reports that a registered plan contributed zero rows.
