---
id: 000797
status: open
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
github_issue:
severity: high
---

# Live reboot refuses owned pane as ambiguous

## Problem

The owning hub can obtain a live worker's exact reboot confirmation offer, but acknowledging it fails during session quiescence: `ownership unresolved: unknown pane owner: pane has no unambiguous actual command`. The current incarnation remains unreplaceable through the public recovery path. This is separate from #791's busy/banner admission problem, where the same reboot refusal appears only in the Log.
