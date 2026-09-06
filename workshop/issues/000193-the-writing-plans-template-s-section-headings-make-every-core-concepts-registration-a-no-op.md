---
id: 000193
status: open
deps: []
github_issue:
created: 2026-09-05
updated: 2026-09-05
estimate_hours:
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

## Spec

Two halves, and the second is the one that matters.

1. **Accept the template's form.** Match the heading by prefix, or normalise the
   gloss away. `pair#170`'s plan uses the bare form and must keep working.

2. **A registered plan that contributes NO rows is an error.** This is the real
   defect: `conceptPlans` is a hand-maintained list whose entries can silently
   mean nothing, which is the same shape as `pair#188` (that list not covering
   most plans) seen from the other side. A plan named in the list and yielding
   zero rows should fail loudly — it is either misformatted or should not be
   listed.

Worth deciding while here: whether the heading strings belong to the skill or to
the parser. Today the skill's template is the de-facto source and the parser
disagrees with it, which means the two can drift again the next time either is
edited. A shared fixture, or the parser accepting exactly what the template
emits, ends that.

## Done when

- A plan whose headings carry the template's parenthetical contributes its rows.
- A plan listed in `conceptPlans` that contributes zero rows FAILS, with a
  message naming the plan and the likely cause.
- `pair#170`'s and `pair#182`'s existing bare-form plans still contribute exactly
  what they contribute today — asserted, since a prefix match is a widening.
- `pair#172`'s plan keeps its rows with the template's heading form restored.

## Plan

- [ ] Reproduce: restore the parenthetical headings in `pair#172`'s plan and
      confirm the suite goes green while asserting nothing.
- [ ] Match by prefix (or normalise), keeping the bare form working.
- [ ] Fail a registered plan that yields zero rows.
- [ ] Decide where the heading strings live so skill and parser cannot drift.

## Log

### 2026-09-05

Found while implementing `pair#172` M1, from the plan-quality gate's PQ-6 telling
me to register the plan or its rows would assert nothing. I registered it, the
test passed, and it still asserted nothing — the check that caught it was
planting a nonexistent symbol and watching the suite stay green, not reading the
code.
