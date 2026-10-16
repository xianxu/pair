---
id: 000621
status: open
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: 'd4616241b2388f66ccc4883544fc69b3c1e15ab9' # card fields mirrored from issue-cards; edit via sdlc
---

# Rename shell test fails legacy prefix case

## Problem

The standalone tests/pair-rename.sh fails at T7, which expects pair rename pair-t1 pair-t2 to strip the pair- prefix. The current CLI looks for the literal pair-t1 and reports no files found although the fixture contains tag t1. This is pre-existing: the original script from the pre-isolation #617 branch HEAD fails identically in a disposable empty environment. The isolation sweep found it; other cases T1–6/T8–9 pass57 assertions when run separately.

## Spec

Reconcile the legacy-prefixed rename contract with the current tag/address model. Determine whether prefixed CLI arguments remain supported; implement or retire the stale expectation consistently in CLI, tests and relevant documentation. Keep exact-tag matching and substring safety. Preserve the synthetic offline fixture and consume the shared subprocess environment helper.

## Done when

- Running bash tests/pair-rename.sh completes all cases without a T7 failure.
- Legacy-prefixed argument behavior is explicit and consistent with the supported CLI contract.
- Exact-tag and substring safety cases continue to pass.

## Plan

- [ ]

## Log

### 2026-10-16 — discovered during #617 isolation sweep

Observed pair rename: no files found for tag 'pair-t1' at T7. Original pre-change source reproduces the same failure under the shared empty-environment helper. Original and current logs are retained locally under /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-617-isolation-checks.S2Vwcq/pair-rename-baseline.log and pair-rename-final.log; no private conversation or live capture is needed for reproduction. Remaining57 assertions pass in pair-rename-remaining-final.log. Filed separately at the TL’s request; no implementation started.
