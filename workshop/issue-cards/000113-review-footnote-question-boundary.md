---
id: '000113'
status: punt
started: 2026-07-08T22:41:44-07:00
created: 2026-07-08
updated: 2026-07-08
estimate_hours: 0.81
---

# review footnotes should not join last question

## Problem

When the review pane sends a final `💬:` question at the end of a document that
is followed by a Markdown footnote block, the question extraction includes the
footnote definitions. The agent response is then inserted after the footnotes
instead of above them. This makes durable review definitions interfere with the
ordinary question/answer workflow.
