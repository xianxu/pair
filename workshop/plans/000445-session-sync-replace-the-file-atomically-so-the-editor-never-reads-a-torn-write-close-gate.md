---
gate: boundary-review
issue: 445
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-11T11:34:11-07:00"
      dispatched: "2026-10-11T11:33:02-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Minor
          title: replaceShared's rename also breaks hard links and drops owner/xattrs/ACLs; comment and lesson name only symlink and mode
          detail: ledger.go:207-211 and workshop/lessons.md list symlink + mode as what a rename changes; hard links, ownership (when couch runs as another user) and xattrs are the rest of that list. Note them in the comment, or accept them explicitly.
          family: atomic-replace-metadata-loss
          round: 1
        - id: BR-2
          severity: Minor
          title: Done-when live check on the real ops status file is deferred to post-merge
          detail: It was verified against a copy with the real nvim config. The close should cite the post-merge ops:0 report that covers the real-file clause.
          family: done-when-deferred-verification
          round: 1
      recipe: small-diff-review
      isolation: fenced
      reviewed: f7dd5eb0d1c1678747e603407bf84a93b54af87a
      blocked: false
    - "n": 2
      timestamp: "2026-10-11T11:36:01-07:00"
      dispatched: "2026-10-11T11:35:38-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: ledger.go:211-213 now names hard link, foreign owner, xattrs and ACLs and accepts them explicitly (one path owned by the operator); the finding allowed this option, and it is a prose-only fix.
          round: 2
        - id: BR-2
          disposition: addressed
          note: The close Log records that the real-file live check runs after merge and is reported to ops:0; the report can't exist before merge, so recording the commitment is all that can be done pre-merge.
          round: 2
      recipe: fix-review
      isolation: fenced
      reviewed: f2c8f0e18f3dba0fc18553fa8e1cfe789d306b55
      blocked: false
---

# Gate ledger — pair#445 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-11T11:34:11-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `atomic-replace-metadata-loss` replaceShared's rename also breaks hard links and drops owner/xattrs/ACLs; comment and lesson name only symlink and mode
  ledger.go:207-211 and workshop/lessons.md list symlink + mode as what a rename changes; hard links, ownership (when couch runs as another user) and xattrs are the rest of that list. Note them in the comment, or accept them explicitly.
- **BR-2** [Minor] `done-when-deferred-verification` Done-when live check on the real ops status file is deferred to post-merge
  It was verified against a copy with the real nvim config. The close should cite the post-merge ops:0 report that covers the real-file clause.

## Round 2 — 2026-10-11T11:36:01-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — ledger.go:211-213 now names hard link, foreign owner, xattrs and ACLs and accepts them explicitly (one path owned by the operator); the finding allowed this option, and it is a prose-only fix.
- BR-2 — addressed — The close Log records that the real-file live check runs after merge and is reported to ops:0; the report can't exist before merge, so recording the commitment is all that can be done pre-merge.

## Open findings

(none — every finding has been disposed)
