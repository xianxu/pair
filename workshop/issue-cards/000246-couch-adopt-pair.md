---
id: 000246
status: open
created: 2026-09-13
updated: 2026-09-13
estimate_hours:
github_issue:
---

# Adopt a stopped standalone Pair session into Couch

## Problem

An operator may start Pair standalone, later quit it, and want to continue that
same session as a Couch-managed thread. Creating a new Couch thread at the same
path does not express adoption of the existing session and its history.
