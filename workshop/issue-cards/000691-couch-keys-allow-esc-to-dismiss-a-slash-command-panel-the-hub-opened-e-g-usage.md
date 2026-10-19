---
id: 000691
status: open
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
---

# couch --keys: allow Esc to dismiss a slash-command panel the hub opened (e.g. /usage)

## Problem

The hub reads Claude quota by running /usage in an idle Claude slot with `couch --keys SLOT clear-line text:/usage submit` and reading it with `couch --peek`. The /usage panel then stays open ("Esc to cancel"), and `couch --keys SLOT esc` refuses ("wait until the agent composer is empty and its turn finishes"), so only the operator can close it. On 2026-10-18 two slots (brain:1 since 12:38, parley.nvim:4) sat with open panels, raising Couch's 10-minute waiting notice and needing the operator's Esc.
