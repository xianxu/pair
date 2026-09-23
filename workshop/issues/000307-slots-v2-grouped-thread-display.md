---
id: 000307
status: working
deps: [pair#306]
github_issue:
created: 2026-09-22
updated: 2026-09-23
estimate_hours:
started: 2026-09-23T16:46:34-07:00
---

# Slots v2: group switcher and tab bar

## Problem

Multiple threads are useful only if the operator can immediately recognize their repo grouping and select the intended workspace.

## Spec

Project: `pair/workshop/projects/couch-slots-v2.md`. Fresh task derived from the current v2 contract; historical task bodies are not prerequisites or implementation plans.

Group primary and slots together in both switcher and tab bar using one ordering derivation. Switcher shows full names and actual paths: pair /path/to/main, then indented pair:1 /path/to/worktree/pair-slot1, pair:2 ... . Tab bar shows pair :1 :2 brain ariadne ... . :0 denotes the primary but its normal display remains repo. Sort slot numbers numerically; retain stable workspace identity for selection/actions rather than parsing displayed text.

Preserve existing lifecycle/state, focus, notification, and navigation behavior. Define rendering when the primary has no visible thread, when slots are parked, and when width is constrained: group context must remain understandable. Keep full address available where shorthand would be ambiguous. ARCH-DRY: switcher and bar consume the same grouped order; no unrelated visual redesign.

### Agreed scope — 2026-09-23

This section takes precedence over earlier conflicting layout or policy text.

Update the agreed switcher examples to `pair /workspace/pair`, indented `pair:1 /workspace/worktree/pair-slot1/pair`, and `pair:2 /workspace/worktree/pair-slot2/pair`. Tab labels remain `pair :1 :2 brain ariadne ...`. Show the actual main checkout path, not only its enclosing environment directory. Primary repositories retain their own direct Couch entries; ordinary dependency clones inside numbered environments do not automatically appear as additional repos/slots in the switcher or tab bar. Operators access those dependencies through the parent thread; no new dependency-management UI is required.

## Done when

- Switcher shows grouped full labels and actual checkout paths with slot indentation; tab bar shows repo followed by :N labels.
- :2 sorts before :10; input record order cannot split a group or change navigation order unexpectedly.
- Keyboard/click selection activates the correct workspace after refresh/reordering; no selection relies on label parsing.
- Tests cover absent primary, parked members, multiple repos, narrow widths, and existing notification/focus states.
- Operator help and screenshots or rendered fixtures show the agreed examples.

- Rendered and activation fixtures use nested main-checkout paths and exclude incidental dependency clones from automatic thread/slot listings.

## Plan

Task outline only; settle implementation design through start-plan before change-code.

- [ ] Specify shared group ordering and absent-primary/narrow-width presentation.
- [ ] Wire both UI projections and selection/navigation to the shared order.
- [ ] Verify rendered examples plus real activation routing.

## Log

### 2026-09-22 — fresh v2 task

Created from the agreed workspace/UI contract and the request for a clean task breakdown. Implementation has not started; estimates follow design approval.

## Revisions

### 2026-09-23 — Show nested main paths without inventing dependency slots

Reason: operator agreed nested environments, ordinary remote dependency clones and existing per-repository publication. Delta: added the authoritative scope clarification and acceptance criteria above; original task context remains as provenance. No implementation or lifecycle-status change is claimed by this revision.

### 2026-09-23 — consume durable slot row identity from #306

Reason: #306 separates directory identity from the replaceable native conversation.
Delta: group `ThreadTarget`/`ThreadRowKey` by repository and slot number; retain
host-path row identity across refresh and fresh conversation. Addressless recovery
rows remain selectable through `open-slot`/`fresh-slot`. Native scope/tag remains
the process/terminal lookup key. #306 supplies these functional rows, while this
issue still owns shared ordering, indentation and grouped tab labels.
