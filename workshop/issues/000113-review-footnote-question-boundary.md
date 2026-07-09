---
id: 000113
status: working
deps: []
github_issue:
created: 2026-07-08
updated: 2026-07-08
estimate_hours: 0.81
started: 2026-07-08T22:41:44-07:00
---

# review footnotes should not join last question

## Problem

When the review pane sends a final `💬:` question at the end of a document that
is followed by a Markdown footnote block, the question extraction includes the
footnote definitions. The agent response is then inserted after the footnotes
instead of above them. This makes durable review definitions interfere with the
ordinary question/answer workflow.

## Spec

For review question submission:

- A line beginning with a Markdown footnote definition, matching
  `^%[%^[^%]]+%]:`, starts a footnote section.
- The final question body stops before the first such footnote-definition line.
- Agent answers for that question are inserted above the first footnote
  definition line, with one blank line separating the answer block from the
  footnotes.
- Existing behavior is preserved when no footnote-definition line follows the
  final question.

ARCH-PURPOSE: definitions are durable document metadata; they must not become
part of the user's last question or push the answer below the metadata.
ARCH-DRY/ARCH-PURE: put the boundary predicate in the pure review/question
helper that already computes the last question range, rather than duplicating
footnote detection in the UI shell.

## Done when

- A regression reproduces `💬:` followed by `---` and `[^acos]: ...`, proving
  only the question text is submitted.
- A regression proves the answer insertion point is above the first
  footnote-definition line with a blank line before the footnotes.
- Existing review question tests still pass.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: lua-neovim design=0.20 impl=0.40
item: milestone-review design=0.00 impl=0.15
total: 0.81
```

## Plan

- [ ] Find the pure/helper code that extracts the final review question and
      chooses the answer insertion point.
- [ ] Add a failing regression for a final `💬:` followed by Markdown footnotes.
- [ ] Teach the range/insertion calculation to treat a leading `[^...]:`
      footnote definition as the trailing metadata boundary.
- [ ] Run focused review/question tests and close #113.

## Log

### 2026-07-08
- Created from reported bug: final review question followed by definition
  footnotes gets submitted together with the footnotes, and the answer is
  inserted below the footnote block.
