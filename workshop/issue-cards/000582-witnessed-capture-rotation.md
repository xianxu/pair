---
id: 000582
status: open
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
github_issue:
---

# Rotate witnessed terminal captures with faithful VT checkpoints

## Problem

Split from #579 by the owning hub on 2026-10-15. Raw terminal captures and their event sidecars grow without size-based rotation. They cannot safely use independent log rotation: readers rely on producer witnesses, exact file identities, cumulative offsets and faithful terminal replay. This issue is deliberately unassigned; #579 handles diagnostic write amplification and diagnostic bounds.
