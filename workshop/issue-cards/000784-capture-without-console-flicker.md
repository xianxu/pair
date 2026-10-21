---
id: 000784
status: open
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
github_issue:
---

# Capture cast pixels without console flicker

## Problem

Periodic fidelity screenshot capture hides the trusted console UI host through privacy suppression, causing visible flashes and removing focused controls from the focus tree. #752 defers background captures while a sketch/label UI is open; outside that bounded fallback the underlying capture mechanism still toggles visibility.
