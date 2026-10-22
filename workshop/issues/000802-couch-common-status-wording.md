---
id: 000802
status: open
deps: [789]
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
severity: low
collaboration-mode: collaboration
card_mirror: 'da2a5c84b8afc227e30cf71a4e526aebd738c3c9' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch: shorten common statuses

## Problem

Common Couch chrome status sentences are longer than needed. The 40-cell display cap in #789 already protects layout, so cosmetic wording does not need an exhaustive global proof.

## Spec

Low-priority follow-up authorized by TL receipt58326574-1d4c-4767-b222-cfb4d5adddce. Shorten a handful of frequently seen built-in messages, starting with wait notices and lifecycle labels; choose the concrete set when taking the issue. Keep the action and essential meaning clear, retain dynamic diagnostic details and existing full-copy recovery. No exhaustive producer catalog, transitive sweep, or total-catalog proof is required. The display cap remains the layout invariant.

Useful starting points: couchtty common wait/broadcast/cast feedback; couchcmd supervision and lifecycle notices. Earlier exploratory wording changes and runtime variant experiments are visible in #789's branch history (797d86d9108817b9a8d5374460a96a53f96133be), but are deliberately removed from its final display-path diff. Do not carry that broad catalog scope into this issue.

## Done when

- A small, explicitly named set of frequent statuses has concise, actionable wording.
- Focused tests pin those selected messages and preserve relevant dynamic diagnostics; no total-catalog assertion is added.
- The existing cap and full-text copy behavior are unchanged.

## Plan

## Log

### 2026-10-22

Split from #789 at the TL's direction; low priority, unclaimed. #789 lands cap plus full-copy only. Pair:4 takes #796 after #789, not this cleanup.
