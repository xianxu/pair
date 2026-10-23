---
id: 000828
status: open
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
github_issue:
severity: medium
---

# Native oracle requires fixture proof

## Problem

During #759 shared-module qualification, the opt-in native Zellij history oracles dumped a shell prompt instead of the fixture child content. The helper accepts a successful dump even without positive fixture readiness. An equality-only oracle can then pass without exercising its intended input. This is a coverage integrity gap, not an established dependency regression.
