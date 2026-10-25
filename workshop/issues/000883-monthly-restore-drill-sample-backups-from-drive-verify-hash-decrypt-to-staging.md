---
id: 000883
status: open
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: 'c9a36ac7210ca024ae55580934f9b0151d2fe55b' # card fields mirrored from issue-cards; edit via sdlc
---

# Monthly restore drill: sample backups from Drive, verify hash, decrypt to staging, check manifest

## Problem

## Spec

**A monthly restore drill for the history backups (operator, 10-24).** The backups upload to Google Drive with a
check on upload, and restore was proven once (pair#689's round-trip test and the operator's live drill), but nothing
re-checks later that what's on Drive can still be downloaded, decrypted and read. Lowering local retention (project
history-backup, ops `teams/devinfra/projects/history-backup/`) should wait for this check.

A scheduled task (couch-scheduling: a Go task in `scripts/scheduled/`, run by Couch's scheduler as the operator's user,
not by an agent; agents have no Drive access) that, once a month:
1. picks a random sample of catalog entries (a fixed small count across sources, plus the newest bundle);
2. downloads each from the Drive sink through the existing archive fetch path;
3. checks each against its catalog hash;
4. decrypts into a fresh staging directory (the existing `pair archive restore` path, gpg-agent with the operator's
   key; never over live files) and checks the manifest and file counts;
5. deletes the staging copy, records a one-line result (date, sample, pass/fail per bundle), and notifies the operator
   (`couch --notify`) only on a failure, naming the bundle and the step that failed.
Bounded: sample size, download bytes and wall time are capped; a missing key or Drive sign-in is reported as
"couldn't run", distinct from a corrupt backup.

## Done when

- The task runs on a monthly schedule and records its result; a failure notifies with bundle and step.
- Tests over a stateful fake sink and catalog: a good sample passes; a hash mismatch, a decrypt failure and a manifest
  mismatch each fail at the right step; "couldn't run" is distinct; staging is always removed.
- One live run with the operator, recorded in the issue Log.

## Plan

- [ ]

## Log

### 2026-10-24
