---
id: 000164
status: open
deps: []
github_issue:
created: 2026-09-01
updated: 2026-09-01
estimate_hours:
---

# Prefill existing values in Couch edit prompts

## Problem

Couch edit prompts start empty even when the field already has a value. For
operations such as renaming an actor or editing its description, this forces
users to recreate the entire value instead of editing the existing text.

## Spec

- Prefill rename input with the actor's current name when available.
- Prefill description input with the actor's current description when
  available.
- Keep the input empty when no prior value exists.
- Let normal editing behavior replace, extend, or delete the prefilled value.

## Done when

- Opening rename presents the current name as editable text.
- Opening description editing presents the current description as editable
  text when one exists.
- Fields without an existing value still open empty.
- Automated tests cover populated and absent original values.

## Plan

- [ ] Add failing tests for rename and description prompt initialization.
- [ ] Pass current field values into the relevant Couch edit prompts.
- [ ] Verify editing, replacing, clearing, and creating previously absent text.

## Log

### 2026-09-01

Captured during Couch dogfood testing. Editing should begin from the original
text whenever that text is available.
