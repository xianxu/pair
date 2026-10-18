---
id: 000671
status: wontfix
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
---

# Startup SIGINT can leave without-ancestors helper alive in cancellation test

## Problem

A recorded #659 integration-conflict suite on f35faa470d4dd2e484b31150326b379635b0fd91 failed tests/without-ancestors-test.py LifetimeTests.test_cancellation_during_startup at SIGINT delay0.02s. First seen 2026-10-18 during attempt e3644ec5a5ebc04b7b8fa2219b970aa7; suite exit2 after2m57s, log SHA256 prefix90d9d93cf898. Retained common-Git log: .git/sdlc/test-stats/failed/e3644ec5a5ebc04b7b8fa2219b970aa7.log.
