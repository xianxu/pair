---
id: '000318'
status: done
started: 2026-09-23T23:42:37-07:00
created: 2026-09-23
updated: 2026-09-24
actual_hours: N/A
---

# Declare Pair post-move build

## Problem

The shared Ariadne procedure for moving an issue branch between slots asks
each repository to declare the build needed after the move. Pair's `:0` runs
from `bin/pair`, but its local agent instructions do not name that step. An
agent can switch branches and then test the old binary.
