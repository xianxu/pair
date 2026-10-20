---
id: 000722
status: open
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
github_issue:
---

# couch --drain [--fast]: a fleet-wide checkpoint-and-stop before a restart, with a drain board

## Problem

There is no safe, on-demand moment to restart the machine or Couch. Work never ends, so "safe to restart" has to be a
state the fleet can reach on request. Today a restart is a crash (10-19 15:55) or the TL pinging slots one by one,
and nobody can read off whether it's safe. Operator item 15 (10-19), discussion:
ops `teams/devinfra/tl-status-xian-xu.side/2026-10-19-15-fleet-drain.md`.
