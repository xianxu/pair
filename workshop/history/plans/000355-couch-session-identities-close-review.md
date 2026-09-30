# Boundary Review — pair#355 (whole-issue close)

| field | value |
|-------|-------|
| issue | 355 — Allocate Couch session identities and enforce repository families |
| repo | pair |
| issue file | workshop/issues/000355-couch-session-identities.md |
| boundary | whole-issue close |
| milestone | — |
| window | bd55cd41d429b8118774d08eb42fe9c6c92230f4..bb50b642453b453e3e165f9bee422cdaf3e7137f |
| command | sdlc close --issue 355 |
| reviewer | codex |
| timestamp | 2026-09-30T12:13:26-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range delivers the C/N/M identity model, terminal ownership checks, and retained repository-family behavior. All five prior findings remain addressed; no new findings surfaced. Focused regressions and command/UI suites pass. The artifact-inventory failure reproduces identically at the pinned base and is unrelated to this change.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Pending/current bindings govern admission, registration, recovery and cleanup. Passing regressions cover stale indexes, old registration, interrupted starts and preservation of the previous terminal.
  - id: BR-2
    disposition: addressed
    note: |
      lifecycle.go revalidates the original server proof immediately before attach, after blocking preparation. Generation-replacement regressions exercise this boundary.
  - id: BR-3
    disposition: addressed
    note: |
      Repository labels have a bounded ASCII normalization and fallback. Allocator and real-Git conversation tests cover Unicode, punctuation-only and maximum-length basenames.
  - id: BR-4
    disposition: addressed
    note: |
      Shared checkout membership reaches routing, migration, preferences, inventory and hosted actors. Passing real-Git regressions cover nested repositories and retained separate-Git-directory authority.
  - id: BR-5
    disposition: addressed
    note: |
      Family admission and persisted-state validation enforce the 4096-family bound. Tests cover concurrent final admission, refusal without writes and reuse of existing families.
```

1. **Strengths**

   - Host high-water marks commit before local counters, preserving monotonic allocation across interrupted publication and local rollback (`couchidentity/store.go:176–193`).
   - Ownership checks use exact terminal bindings and server generations; uncertain observations retain occupied state.
   - Repository membership is centralized and tested through actual storage and launch consumers.
   - README and atlas document naming, migration, restore limits and retained starting directories.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage**

   Passed allocator, durable-file, launcher, thread-record, pane-parser, checkpoint, Couch command and TTY suites; focused Couch tests cover identity lifetimes, recovery, family admission, nested repositories and separate Git directories. Diff whitespace checks pass.

   The artifact-inventory test reports the same 32 violations on both pinned base and head, independently reproduced in a temporary baseline checkout. Full Couch-core, race, fuzz and live Zellij conformance runs were not repeated in this review; no fix-removal mutation was performed.

6. **Architecture**

   - **ARCH-DRY — pass:** shared allocation, ownership and membership authorities.
   - **ARCH-PURE — pass:** allocation, classification and transition logic remain directly testable; IO uses separate boundaries.
   - **ARCH-PURPOSE — pass:** lifecycle and repository-family consumers derive from the new authorities.
   - **ARCH-MOCK — pass:** stateful terminal fixtures, injected seams, isolated Git fixtures and an explicit live-conformance test.
   - **ARCH-CONSTRAINTS — pass:** bounded reads, cancellable locks and timed ownership probes; passive inventory avoids per-row probes.
   - **ARCH-SECURE — pass:** strict persisted-state validation and positive ownership evidence.
   - **ARCH-ORDER — pass:** pending/current bindings preserve partial progress and uncertainty.
   - **ARCH-FUNERAL — pass:** bounded permanent reservations, fixed counter state and existing binding retention.

7. **Plan revisions:** None required. Existing revisions capture the delivered corrections.
