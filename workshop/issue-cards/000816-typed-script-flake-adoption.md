---
id: 000816
status: open
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
github_issue:
---

# Adopt typed script flake evidence

## Problem

The recorded Pair suite still invokes tests/nvim-route-poke-admission-test.sh directly. Pair#773 retained its 30-second run_headless timeout as an unattributable failure (tracked by #685), so it could not be registered or resumed. Ariadne's typed implementation and producer distribution must land before this local-gates activation.
