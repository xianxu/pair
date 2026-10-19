---
id: 000706
status: open
deps: [pair#689]
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'fa476282abdbc00e9d7ce194e2762c9924141e7c' # card fields mirrored from issue-cards; edit via sdlc
---

# Drive backup: store bundles in year/month folders instead of one flat folder

## Problem

The Drive backup destination `backups/pair-history/` is one flat folder (pair#678), growing by roughly 5,000-10,000 files a year and never shrinking (retention deletes local copies only). The operator (2026-10-19): rather than wait for the 20,000-file warning (#705), put files into year/month folders now.

## Spec

New uploads go to `backups/pair-history/<YYYY>/<MM>/` (from the bundle's time range, the same year/month the local catalog path uses), keeping each file's content-hash name. The pass lists only the folders it needs (the current month, plus any month with pending bundles) instead of the whole tree, which keeps listing cost bounded as Drive grows. Existing flat-folder files stay where they are and remain valid: the upload receipts and #689's label index record each bundle's location, so restore and verification read the recorded path, old or new. Duplicate detection (the "exactly one entry with that name" rule) applies within the bundle's month folder, plus the legacy flat folder for older bundles. Optional, separate step: move the existing flat files into month folders with server-side moves, verifying by hash afterwards. Records tolerate and preserve unknown fields (ARCH-EVOLVE).

## Done when

- A fresh upload lands in `<YYYY>/<MM>/`; a legacy flat-folder bundle still verifies and restores.
- A pass lists only the needed month folders (counted test); the duplicate and uniqueness checks hold in both layouts.

## Plan

- [ ]

## Log

### 2026-10-19
