---
id: 000689
status: open
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: '60ca1b70dda5eca2182f17cccc2c1fa6b2b7845f' # card fields mirrored from issue-cards; edit via sdlc
---

# history backup: find and restore from the backups (local or Drive), with a round-trip restore test

## Problem

pair archive can create and upload backups (continuous pass, Drive sink #678) but has no way to read them back: no restore command and no index. The operator asked (2026-10-18) whether we have tested reading the backups back, for example fetching the logs from a few days selectively. We have not: verification today proves the bytes in Drive match the local ciphertext, not that a bundle decrypts, unpacks, and yields the files someone wants. Untested restore means unproven backup.

## Spec

Each bundle already has a plain JSON label (kind, from/to time range, file_count, raw and encrypted bytes, recipients, sha256), stored beside it locally and in Drive. Add: (1) `pair archive find --kind K --since T --until T` listing matching bundles from the labels (local catalog first, Drive listing when asked); (2) `pair archive restore ID... --to DIR [--from drive]` fetching the bundle (from Drive by its flat content name if not local), verifying its sha256 against the label, decrypting with the operator's GPG key via gpg-agent, and unpacking into DIR (never over live data); optional path filter inside the bundle. (3) A restore drill: an automated test with a throwaway GPG key round-trips create, upload (fake rclone), find, restore, and compares files byte for byte; plus a documented one-time live drill the operator runs against Drive (decryption needs their key).

## Done when

- find/restore exist; the round-trip test passes with a throwaway key.
- The operator restores one real day of Claude transcripts from Drive into a scratch folder and the files match the originals.

## Plan

- [ ]

## Log

### 2026-10-18
