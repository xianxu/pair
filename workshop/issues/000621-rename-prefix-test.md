---
id: 000621
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-17
estimate_hours:
card_mirror: '2f00787dc0c6c5a85be3859c3928ef131050283a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T10:48:06-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "d08c5324", done: "45645efd"}
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

- [ ] Replace stale T7 with deterministic literal-prefix coverage: distinguish bare and prefixed source families, retain the bare family and refuse a missing exact source, and protect bare destination content.
- [ ] Retain shared empty-environment bootstrap and use its selected private test binary; run this shell matrix in the normal suite.
- [ ] Clarify help, README and atlas; run shell matrix, launcher rename tests and recorded suite, close and land.

ARCH-DRY/PURPOSE: current exact-tag validation is authoritative; reuse shared bootstrap and candidate-binary helper instead of reintroducing prefix aliases. ARCH-PURE: no production logic changes; test the existing pure parser and offline CLI boundary. ARCH-MOCK/SECURE: synthetic artifact directories and no-session Zellij stub, no operator data. ARCH-CONSTRAINTS: small fixed offline matrix, no sleeps or timing thresholds. ARCH-ORDER: no new runtime state or interleaving; this is a deterministic stale oracle. ARCH-FUNERAL: one trap-owned fixture root removes all generated cases/shims and candidate executable.

## Log

### 2026-10-16 — discovered during #617 isolation sweep

Observed pair rename: no files found for tag 'pair-t1' at T7. Original pre-change source reproduces the same failure under the shared empty-environment helper. Original and current logs are retained locally under /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-617-isolation-checks.S2Vwcq/pair-rename-baseline.log and pair-rename-final.log; no private conversation or live capture is needed for reproduction. Remaining57 assertions pass in pair-rename-remaining-final.log. Filed separately at the TL’s request; no implementation started.

## Revisions

### 2026-10-17 — preserve the existing exact-tag contract

Investigation confirms #154 commit 33c91c1cee0cac2e6bd85c17a41517fa1214cfa3 intentionally removed rename prefix normalization, because pair- is part of durable tag identity. Existing TestValidateRenameTags and TestRunRenamePreservesExactPrefixedOldTag/NewTag enforce that policy. Retire T7's legacy alias expectation rather than changing production behavior. A bare tag can itself begin with pair-; only indexed public session names are resolved separately. Add the shell regression to normal make test so stale expectations cannot remain dormant.

### 2026-10-17 — baseline reproduction

bash tests/pair-rename.sh fails deterministically at T7 with no files found for tag pair-t1, exit 1; log /tmp/pair621-baseline.log. T1–T6 pass before the script exits. The shared bootstrap isolates the environment, but the script still selects checkout bin/pair and is absent from the normal Make test prerequisites. Use the existing private candidate-binary helper and retain all cases, including T8–T9 that the early failure masked. No retry or production timeout is involved.
