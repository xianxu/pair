---
id: 000665
status: open
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
---

# Adopt ariadne's Go lint catalog (pkg/golint) in pair

## Problem

ariadne#407 added pkg/golint, a stdlib-only Go lint catalog run as a repository test, so every landing enforces it. Its seed rules come from mistakes made in both repos: test-shrunk-deadline (ariadne#434), tmp-literal (#342), test-go-build (pair#602), test-tight-wallclock and test-fixed-sleep (the 10-15 flakes). pair has the same classes (pair#659, #662, #623 were wall-clock budgets) but no catalog.
