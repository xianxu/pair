---
id: 000360
status: open
deps: []
github_issue:
created: 2026-09-30
updated: 2026-09-30
estimate_hours:
card_mirror: '235b6ad905e67326fca23fd66829bbf7b95076cb' # card fields mirrored from issue-cards; edit via sdlc
---

# Address couch slots by short repo name, thread name, or attributes

## Problem

Slot references must use the exact repository directory name: `parley.nvim:1`,
not `parley:1`. `ParseWorkspaceReference` (`couchcore/workspaceref.go`) splits
`repo:N`, and `resolveSlotInput` (`couchcore/slotcontext.go`) joins `repo` onto
the fleet root, so anything that isn't the exact directory name fails. People
type the short form naturally, and an agent in an LLM session understands it,
but couch does not.

Slots also carry attributes, such as which coding agent they run. There is no
way to address "a free slot running claude", which cross-slot dispatch (#353)
will want.

## Spec

Two layers, deterministic first.

1. **Names that resolve (the cheap part).**
   - After a thread or slot is renamed (`couch name`), both its name and its
     repository name address it. Renaming parley.nvim's slot to `parley`
     makes `parley:1` work, and `parley.nvim:1` keeps working.
   - A repository reference that matches no directory falls back to a unique
     prefix of an enrolled repository's name: `parley:1` resolves to
     `parley.nvim:1` without a rename. A prefix matching several repositories
     is refused, listing them.
   - A miss lists the candidates (repos, slot numbers, names) so the caller can
     correct itself in one step.
2. **Attribute selection.** Slot attributes (agent, repository, name, free or
   busy) are selectable without naming a slot, e.g. "any free slot of `pair`
   running claude". Proposed surface: a few explicit filters, such as
   `--agent`, on the commands that take a slot, resolved inside couch so the
   caller never reads the inventory to route. A machine-readable live-slot
   inventory (`couch list --json` or equivalent) exists for inspection and
   for the miss case.

### Alternative to settle at design: an LLM resolver inside couch

Proposed by the operator: couch builds a prompt from the live-slot inventory,
the request and the couch skill, asks a small fast model for the command with
an output schema, and runs it. It covers short names, attributes and any
future attribute with no new code.

Recommendation: don't put the model inside couch's routing; keep resolution
deterministic and give the model the data instead.

- **Keep operational state out of the sender's context.** The operator's
  reason for a resolver inside couch: an agent should not have the latest
  slot inventory dumped into its context on every route. Deterministic
  resolution meets that too. The sender passes a short reference or filter
  (`parley:1`, `--agent claude`), couch resolves it locally, and the sender
  sees a short candidate list only on a miss. The full inventory is for
  inspection, not something every dispatch reads.
- **Wrong routing is costly and silent.** A dispatch injects text into another
  agent's session. A deterministic resolver either finds one match or says
  why not. A model can confidently pick the wrong slot, and an output schema
  constrains the shape of the answer, not its truth.
- **Couch must work offline and fast.** A network model on every resolution
  adds latency, cost and a failure mode to a path that is local today
  (couch must not degrade pair).
- **Testability.** Deterministic resolution is unit-testable; model output
  needs evals.

The model approach could still serve free-form operator text typed into the
couch switcher, as a fallback that only proposes a resolved command and runs
it after the operator confirms. Decide at design whether that is in scope.

## Done when

- `parley:1` resolves to `parley.nvim:1` when `parley` is a unique prefix, and
  an ambiguous prefix is refused with the candidates listed.
- After renaming a slot's thread, both the new name and the repository name
  address it.
- A reference that resolves to nothing lists the candidates.
- A free slot can be selected by agent (and repository) without naming it,
  with a test where two free slots run different agents.
- Routing never requires the sender to read the slot inventory: a reference or
  filter resolves inside couch, and only a miss returns candidates.
- The couch skill tells agents the short forms and filters.

## Plan

- [ ] Settle the LLM-resolver alternative (recommendation: out of routing; at
      most a confirm-first switcher fallback).
- [ ] Name resolution: thread name and repository name, unique-prefix fallback,
      candidate list on a miss.
- [ ] Machine-readable live-slot inventory with attributes.
- [ ] Attribute filters, starting with agent.
- [ ] Couch skill update.

## Log

### 2026-09-30

- Filed at the operator's request. Related to #353 (cross-slot dispatch
  addresses `repo:N` and "any free slot of `repo`"), and the attribute
  selection is what #353's "any free slot" would use. Not a dependency: short
  names help every command that takes a slot reference today.
- Current resolution: `ParseWorkspaceReference` plus `resolveSlotInput`, exact
  directory name under the fleet root. Thread names already resolve as plain
  refs (`ref` is "tag, path, or name"), but not inside `repo:N`.
- Operator review: the reason for an in-couch resolver is to keep slot
  inventory out of the sending agent's context on every route. Folded in as
  a requirement; deterministic filters resolved inside couch meet it, with
  candidates returned only on a miss.

