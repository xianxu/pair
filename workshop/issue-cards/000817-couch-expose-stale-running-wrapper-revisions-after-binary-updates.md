---
id: 000817
status: open
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
github_issue:
---

# Couch: expose stale running wrapper revisions after binary updates

## Problem

Couch restarts and rebuilding the on-disk Pair binary do not replace existing wrappers. The operator can therefore run a current supervisor while admission still uses older code, without a clear per-wrapper indication.

During the 2026-10-22 live stall investigation for #791/#810, the wrapper mapped a different executable inode from the current disk binary. The frozen stream admitted correctly on current code and reproduced the visible-peek/hidden-admission discrepancy with pre-#735 resize behavior. The running wrapper revision was unproved. TL reports #735 landed at 13:25 on October 20, after ops:0 and roughly fifteen other wrappers started that morning; the chronic stalls are very likely stale pre-fix wrappers.
