---
id: 000672
status: open
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
severity: critical
---

# Accept native Zellij client replies without blocking launches

## Problem

Operator launches/relaunches fail on the new build containing #647. Observed by the hub: exact session name 📁1-292 is rejected because ProbeSessionName's successful list-clients reply fails strict decoding with invalid Zellij clients header. Installed Zellij is0.45.1. This blocks new slots and takes priority over #665 M2.
