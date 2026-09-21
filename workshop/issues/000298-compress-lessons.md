---
id: 000298
status: open
deps: []
github_issue:
created: 2026-09-20
updated: 2026-09-20
estimate_hours:
---

# Compress workshop lessons into durable guidance

## Problem

`workshop/lessons.md` has grown to 5,640 lines. Repeated or overly
incident-specific entries make the durable guidance difficult to scan and
increase the chance that new lessons are overlooked.

## Spec

Rewrite the file as a compact, durable rulebook: merge overlapping lessons,
keep the reusable rule and the smallest evidence needed to explain it, remove
stale incident narrative, and preserve issue references where they provide
useful provenance. Do not silently discard a distinct engineering lesson;
archive-only detail can remain in issue history rather than in this index.

## Done when

- `workshop/lessons.md` is substantially shorter (target: no more than 1,500
  lines) and organized for quick lookup.
- Every retained entry states an actionable rule, its failure signal, and any
  relevant ARCH-* principle or issue reference.
- Duplicate, obsolete, and incident-only prose is merged or removed without
  losing a distinct preventative lesson.
- Markdown links, headings, and repository checks for lessons references pass.
- The issue log records the compression method and verification evidence.

## Plan

- [ ] Inventory and group lessons by recurring failure mode; identify stale or
      duplicate entries before editing.
- [ ] Rewrite the file into concise durable guidance, preserving provenance and
      distinct rules.
- [ ] Review the diff for lost rules, validate Markdown/reference checks, and
      record the resulting size and evidence.

## Log

### 2026-09-20

Created after measuring `workshop/lessons.md` at 5,640 lines.
