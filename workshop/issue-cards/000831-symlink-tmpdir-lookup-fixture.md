---
id: 000831
status: open
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
github_issue:
severity: medium
---

# Symlink TMPDIR breaks lookup fixture

## Problem

TestLookupSocketUsesNativeSymlinkParentTraversal fails both temporary-root cases while constructing its frames.json fixture, before exercising the query. Found during #759 shared-upgrade-only diagnostic suite; a matched original-module control reproduces it, so it is not unique to those upgrades. The isolated Go1.27.3 baseline suite passed; this is environment-sensitive fixture behavior, not established flaky production lookup.
