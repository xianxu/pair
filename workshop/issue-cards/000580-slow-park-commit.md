---
id: 000580
status: open
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
github_issue:
severity: high
---

# Successful slow park commit is converted into an incomplete park under load

## Problem

During the operator's 2026-10-15 14:57 staged #544/#556 restart acceptance, couch --relaunch ariadne:5 --confirm failed park-incomplete: park requested commit exceeded 1s deadline; transaction still open; recover with retry / recover / abandon. Reported prepare10.9s / operation29.5s. The live wrapper and native Codex remained running; no lifecycle request was published. The success of BeginPark's durable commit is converted into a hard failure solely because its wall time reaches one second.
