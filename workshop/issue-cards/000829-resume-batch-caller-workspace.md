---
id: 000829
status: open
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
github_issue:
---

# Flake: startup prewarm resume batch loses caller workspace

## Problem

During #162's complete affected-package verification on Go1.27.3, TestHubResumeBatchSocketAcceptance/startup-prewarm failed: all three in-process CLI resume calls refused with caller workspace unavailable / unknown message workspace; fresh authority proofs remained zero and each target stayed parked. This is a fixture/authority observation, not evidence that shutdown inherited credentials: the test supplies its own testRT map and calls runMessageCLIWithCall in-process.
