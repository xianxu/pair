---
id: 000581
status: open
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
github_issue:
severity: high
---

# Expose park transaction modes through the owning hub CLI

## Problem

The hub TL cannot drive an existing park transaction through Couch CLI. During the operator's 2026-10-15 acceptance, ariadne:5 retained an open requested park after the slow-commit refusal tracked in #580. The operator authorized the TL/worker to Retry the exact attempt, but `couch --internal park ariadne:5 --mode=retry` refuses at parsing: unknown internal operation park. The only supported path is selecting the row's UI park action. Agent-first lifecycle operation must not require the person to click a menu.
