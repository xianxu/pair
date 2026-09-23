# Boundary Review — pair#306 (whole-issue close)

| field | value |
|-------|-------|
| issue | 306 — Slots v2: multiple threads and parked admission |
| repo | pair |
| issue file | workshop/issues/000306-slots-v2-thread-lifecycle.md |
| boundary | whole-issue close |
| milestone | — |
| window | adc166bec16ac73af7559ce9ee728147da7c7b7a..eb2e25f51ff63d3ec5c0c19149a59dc422968f51 |
| command | sdlc close --issue 306 |
| reviewer | codex |
| timestamp | 2026-09-23T16:20:36-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The lifecycle design, routing, recovery, tests, and atlas coverage are substantial and mostly aligned with the issue. However, local slot lifecycle readers still use raw `os.ReadFile` paths that follow symlinks, violating the promised untrusted-metadata and symlink-swap protections. README also contains unresolved `🤖` edit markers, leaving user-facing documentation malformed.

1. Strengths

- Local slot authority and migration are comprehensively modeled in `threadstore_location.go` and `slotmigration.go`.
- Recovery preserves damaged evidence and prevents fresh starts when ownership is uncertain.
- Stateful tests cover migration interruption, recovery, parked admission, native collisions, and local/global routing.
- Atlas and README were updated for the new numbered-slot surface.

2. Critical findings

- `cmd/internal/couchcore/threadstore.go:355`, `:735`, `:795`, `:1067` — local lifecycle reads use `os.ReadFile` directly, bypassing `readRetentionFile`’s symlink/type/path checks. A symlinked `thread.json` can therefore be treated as authoritative during park, snapshot, start completion, or reads. This violates ARCH-SECURE and the plan’s explicit symlink-swapped metadata contract. Route all local payload reads through the guarded reader and add a regression test proving lifecycle mutation fails and the outside target remains unchanged.

3. Important findings

- `README.md:405` — unresolved `🤖~...~{...}` markers are shipped in the user-facing README. Resolve the marker into the final prose before closing the issue.

4. Minor findings

None.

5. Test coverage notes

Targeted tests cover many lifecycle paths and local-storage hazards, but no regression test covers symlinked `thread.json` through the affected lifecycle readers. `go vet ./...` reports an existing diagnostic in `cmd/internal/pairlifecycletest/live_zellij_diagnostics_test.go`; the full and race suites were still running when review inspection ended.

6. Architectural notes

- ARCH-DRY: Pass — lifecycle operations converge through shared routed mutation helpers.
- ARCH-PURE: Pass — decision logic is separated from filesystem/process integration.
- ARCH-PURPOSE: Pass — the diff broadly fulfills local authority, recovery, admission, and GC goals.
- ARCH-MOCK: Pass — stateful fakes and temporary-store integration fixtures are present.
- ARCH-CONSTRAINTS: Pass — candidate, metadata, migration, and process bounds are explicit.
- ARCH-SECURE: Flagged — direct local payload reads bypass symlink/type validation.
- ARCH-ORDER: Pass — launch, park, recovery, and admission transitions are explicitly sequenced and tested.
- ARCH-FUNERAL: Pass — slot artifacts, archives, journals, and recovery backups have stated retention/removal policies.

7. Plan revision recommendations

- Add a `## Revisions` entry enumerating every direct local metadata reader and requiring guarded reads plus a symlinked-payload regression test.

```findings
findings:
  - id: new
    severity: Critical
    family: local-metadata-boundary-validation
    title: |
      Local lifecycle readers bypass symlink and metadata-boundary validation
    detail: |
      threadstore.go uses os.ReadFile directly for local current records in update, snapshot, successful-start, and readThreadLocked paths, allowing a symlinked thread.json to be treated as authoritative. Route every local payload read through the guarded reader and add a regression test proving the mutation fails without touching the symlink target. ARCH-SECURE.
  - id: new
    severity: Important
    family: unresolved-human-markers
    title: |
      README ships unresolved edit markers
    detail: |
      README.md:405 contains an unresolved 🤖 deletion/replacement marker, leaving the user-facing numbered-slot documentation malformed.
```

---

## Re-review — 2026-09-23T16:38:43-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 306 — Slots v2: multiple threads and parked admission |
| repo | pair |
| issue file | workshop/issues/000306-slots-v2-thread-lifecycle.md |
| boundary | whole-issue close |
| milestone | — |
| window | adc166bec16ac73af7559ce9ee728147da7c7b7a..e7fe8364bccfc619c126cd95203de6ad4225084c |
| command | sdlc close --issue 306 |
| reviewer | codex |
| timestamp | 2026-09-23T16:38:43-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The pinned range and required stat/name-status inspections succeeded. No new Critical, Important, or Minor findings were identified. Prior findings are addressed with concrete code, regression tests, and corrected README text.

1. Strengths

- Local metadata reads use descriptor-relative no-follow validation (`threadstore_layout.go:152`).
- Symlink regressions cover get, mutation, snapshot, start, park, journal replay, and forbidden target mutation (`threadstore_reader_test.go:13`).
- ThreadStore consumers and all five GC routes are explicitly enumerated and routed.
- Atlas and README document numbered slots, local authority, readiness, recovery, and lifecycle behavior.
- Targeted lifecycle, recovery, readiness, acceptance, and symlink tests pass.

2. Critical findings

None.

3. Important findings

None.

4. Minor findings

None.

5. Test coverage notes

`go test` targeted slot, recovery, readiness, acceptance, and symlink tests passed. A broader package run was interrupted after 104 seconds without producing a result; no failure was observed before interruption.

6. Architectural notes

- ARCH-DRY: pass — shared lifecycle and guarded-reader paths are reused.
- ARCH-PURE: pass — slot decisions are separated from filesystem/process seams.
- ARCH-PURPOSE: pass — direct ThreadStore and GC consumers are covered.
- ARCH-MOCK: pass — stateful catalog/process/storage fakes are present.
- ARCH-CONSTRAINTS: pass — discovery, metadata, journal, and recovery bounds are enforced.
- ARCH-SECURE: pass — local persisted metadata rejects symlinks, non-regular files, and oversized inputs.
- ARCH-ORDER: pass — claims, journals, ownership observations, and recovery ordering are tested.
- ARCH-FUNERAL: pass — archives and recovery evidence have bounded retention/removal behavior.

7. Plan revision recommendations

None.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The plan now enumerates direct lifecycle, metadata, inventory, archive, continuation, and all five GC consumers; the implementation routes them through shared backend resolvers.
  - id: BR-2
    disposition: addressed
    note: |
      All local current-record reads use readPayload/readOptionalPayload/readRetentionFile, and threadstore_reader_test.go proves exported reads and mutations reject symlinks without changing the target.
  - id: BR-3
    disposition: addressed
    note: |
      README.md:405-417 now contains clean numbered-slot documentation with no unresolved edit marker.
findings:
```
