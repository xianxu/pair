---
id: 000820
status: open
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
github_issue:
severity: high
---

# Classify retained replacement startup failure

## Problem

The replacement-startup disconnect originally captured under #797 and investigated in #806 remains causally unclassified. Its private server log was removed during cleanup. Passing controls show that Server not ready warnings alone do not establish the cause. #806 now retains bounded server-log tails and launch identities for the next failure.
