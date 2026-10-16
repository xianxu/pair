---
id: 000621
status: open
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
github_issue:
---

# Rename shell test fails legacy prefix case

## Problem

The standalone tests/pair-rename.sh fails at T7, which expects pair rename pair-t1 pair-t2 to strip the pair- prefix. The current CLI looks for the literal pair-t1 and reports no files found although the fixture contains tag t1. This is pre-existing: the original script from the pre-isolation #617 branch HEAD fails identically in a disposable empty environment. The isolation sweep found it; other cases T1–6/T8–9 pass57 assertions when run separately.
