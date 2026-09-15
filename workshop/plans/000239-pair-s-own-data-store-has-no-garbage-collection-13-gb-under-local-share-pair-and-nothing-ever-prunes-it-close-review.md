# Boundary Review — 000239-pair-s-own-data-store-has-no-garbage-collection-13-gb-under-local-share-pair-and-nothing-ever-prunes-it#239 (whole-issue close)

| field | value |
|-------|-------|
| issue | 239 — Pair's own data store has no garbage collection: 13 GB under ~/.local/share/pair and nothing ever prunes it |
| repo | 000239-pair-s-own-data-store-has-no-garbage-collection-13-gb-under-local-share-pair-and-nothing-ever-prunes-it |
| issue file | workshop/issues/000239-pair-s-own-data-store-has-no-garbage-collection-13-gb-under-local-share-pair-and-nothing-ever-prunes-it.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6b06b449ae3521b92187ae14c51d62b66ec356e4..c749dc1a4d23adbf87b0e57a27dc845d4abde34c |
| command | sdlc close --issue 239 |
| reviewer | codex |
| timestamp | 2026-09-15T00:26:16-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The inspected implementation matches the approved retention revisions: independent retention buckets, explicit migration, live-owner protection, and recoverable collection. All nine prior findings remain addressed. Focused package suites and Couch recovery tests passed; no new blocking finding emerged. The repository remained unchanged.

## 1. Strengths

- Collection freezes exact identities before destructive effects and uses the production transaction reducer for phase changes.
- Recovery tests exercise actual process death, partial writes, cancellation, replacement files, and repeated replay.
- Retention policy tests directly verify expiry boundaries and protection precedence without IO.
- README and atlas cover migration, retention clocks, automatic maintenance, and operating limitations.

## 2. Critical findings

None.

## 3. Important findings

None.

## 4. Minor findings

None.

## 5. Test coverage notes

Passed during this review:

- Complete `artifactpath`, `storagegc`, `diagnosticlog`, `gcruntime`, `gccmd`, and `retentioncmd` package suites.
- Focused Couch retention, archive onboarding/detachment, publication, and journal-replay tests.
- Pinned-range `git diff --check`.

The runtime suite includes the 100,000-filename fixture. I inspected regression reachability but did not perform scratch mutations. Full-repository, race, Lua/shell, and hosted conformance runs were not repeated here.

## 6. Architectural notes

| Principle | Result | Evidence |
|---|---|---|
| ARCH-DRY | Pass | Artifact membership and retention classes derive from shared manifest/constructor logic. |
| ARCH-PURE | Pass | Policy and transaction transitions are deterministic; StoreRegistry is correctly classified INTEGRATION. |
| ARCH-PURPOSE | Pass | Session, capture, and diagnostic policies have production consumers and collection paths. |
| ARCH-MOCK | Pass | Portable stores, stateful process/inspection seams, and local process conformance exercise external dependencies. |
| ARCH-CONSTRAINTS | Pass | Owner budgets, cancellation, nonblocking maintenance locks, and oversized-inventory retention have tests. |
| ARCH-SECURE | Pass | Malformed metadata and uncertain identity retain data; deletion validates exact paths and identities. |
| ARCH-ORDER | Pass | Production transitions use the reducer; interruption and replay tests cover partial effects. |
| ARCH-FUNERAL | Pass | Journals, temporary publications, generations, and owner metadata have recovery/removal paths; permanent coordination inodes are documented. |

## 7. Plan revision recommendations

None.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Eligible metadata-only owners use journaled retirement; protection, admission, and interrupted-retirement tests pass.
  - id: BR-2
    disposition: addressed
    note: |
      Coordinated JSON publication uses central pending staging; killed-publisher and bounded recovery tests pass.
  - id: BR-3
    disposition: addressed
    note: |
      Confirmed-dead pre-spawn reservations retire through recovery and admission; spawned and unknown reservations remain protected in passing tests.
  - id: BR-4
    disposition: addressed
    note: |
      Durable references discover owners without payload directories; legacy archive onboarding, identity validation, and eventual collection tests pass.
  - id: BR-5
    disposition: addressed
    note: |
      Production advancement calls ReduceTransaction; matrix, sequence, and bypass tests pass. The plan Core concepts table classifies StoreRegistry as INTEGRATION, matching filesystem validation in stores.go:26-66.
  - id: BR-6
    disposition: addressed
    note: |
      Owner-budget, 100,000-filename, diagnostic-isolation, contention, deadline, and cancellation tests pass through production maintenance paths.
  - id: BR-7
    disposition: addressed
    note: |
      Prepared journal publication precedes unique quarantine creation; publication-failure, killed-publisher, and unsafe-replacement tests pass.
  - id: BR-8
    disposition: addressed
    note: |
      Diagnostic deletion replay tolerates completed ancestor removal while rejecting replacement evidence; effect-boundary replay tests pass.
  - id: BR-9
    disposition: addressed
    note: |
      Append authority precedes payload mutation and recovery commits only the observed authorized prefix; partial-write, cancellation, killed-publisher, and replacement tests pass.
```
