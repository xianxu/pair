# Boundary Review — pair#573 (whole-issue close)

| field | value |
|-------|-------|
| issue | 573 — Agentic documents: Couch registry so session-sync verbs work for any submitted file (universal-agent-input-interface S1, Couch side) |
| repo | pair |
| issue file | workshop/issues/000573-agentic-documents-couch-registry-so-session-sync-verbs-work-for-any-submitted-file-universal-agent-input-interface-s1-couch-side.md |
| boundary | whole-issue close |
| milestone | — |
| window | d6da7d4b809729db8a8766bd71258ca8f3326d9a..52a0ff84621e6a41f057af5a3b8d1c4d03c0d560 |
| command | sdlc close --issue 573 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T12:59:40-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The registry, CLI, documentation, and queued-submit attribution largely match the approved plan. All four focused package suites pass. Two reproduced correctness defects block shipping: repeat registration can bypass an operator lock through a symlink alias, and successful operations can publish shadows that subsequent sync commands refuse to read.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: canonical-document-lock-authority
    title: |
      Repeated registration discards additional aliases and permits writes under their operator locks
    detail: |
      cmd/internal/sessionsync/registry.go:256 returns from existing-registration handling without retaining the requested spelling; documentLockPaths at line 529 can inspect only the canonical path, current request, and stored spellings. Reproduced by registering FILE, registering a symlink ALIAS to FILE, creating ALIAS.lock, then writing FILE: the write succeeds despite the operator lock. Preserve all admitted live spellings through the registry transition model, with bounded growth, and use them consistently for lock checks and legacy handoff. ARCH-ORDER / ARCH-PURPOSE.
  - id: new
    severity: Critical
    kind: defect
    family: persisted-writer-reader-envelope
    title: |
      Shadow writers exceed the registered reader limit and strand successful operations
    detail: |
      cmd/internal/sessionsync/sync.go:417 bounds only the resulting document; line 432 can publish a larger section-preserving shadow, while registry.go:496 refuses shadows over 4 MiB. Reproduced with an approximately 2 MiB baseline section, an operator deletion elsewhere, and an approximately 2 MiB section write: Write succeeds, then Read fails with registry-limit. Independently, registry.go:287-311 imports an oversized legacy shadow without validation, returns registration success, and leaves subsequent reads failing. Validate complete prospective metadata before document mutation or registry publication; sweep every metadata writer against its reader limits and add regressions for both paths. ARCH-CONSTRAINTS / ARCH-FUNERAL.
```

1. **Strengths**

   - Registry transitions preserve stable identity, reject conflicting owners and destinations, and detach returned history from input state.
   - Forget retains central authority until legacy metadata export succeeds; fault-injection coverage exercises that handoff.
   - Broker admission time survives queue promotion. The broker/ledger regression controls late completion across rename and old-path reuse.
   - README, embedded skill, atlas contract, and artifact inventory cover the new surface.

2. **Critical findings**

   - **Alias lock authority:** `registry.go:256`. Retain additional registered spellings without resetting identity or the owed horizon. Add a regression asserting canonical writes refuse under a subsequently registered alias’s lock.
   - **Writer/reader bounds:** `sync.go:432` and `registry.go:311`. Preflight the prospective shadow before changing the document, and validate imported metadata before admitting registration. Preserve existing authority and snapshots on refusal.

3. **Important findings**

   None additional.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:

   ```text
   go test ./cmd/internal/sessionsync ./cmd/internal/couchmessage ./cmd/internal/couchcmd ./cmd/internal/artifactpath
   ```

   Three temporary Go test overlays failed on the pinned head, reproducing the alias-lock bypass, oversized legacy-shadow import, and oversized generated shadow. The overlays were removed; the checkout remains unchanged. Existing tests do not cover these cases. `git diff --check` also passed.

6. **Architectural notes**

   - **ARCH-DRY — pass:** reuses the existing ledger, CAS, ownership parser, and durable publication.
   - **ARCH-PURE — pass:** registry decisions are separated from filesystem effects; model tests run without IO.
   - **ARCH-PURPOSE — flag:** canonical registration does not preserve lock authority across repeated alias registration.
   - **ARCH-MOCK — pass:** temporary filesystem backends and controlled broker endpoints exercise production seams.
   - **ARCH-CONSTRAINTS — flag:** shadow producers do not enforce the reader’s declared envelope.
   - **ARCH-SECURE — pass:** registry decoding validates persisted authority and refuses malformed records; registered file reads check type and size.
   - **ARCH-ORDER — flag:** admitted alias knowledge is absent from authoritative registry state. Rename/forget serialization and uncertain-publication handling otherwise have meaningful coverage.
   - **ARCH-FUNERAL — flag:** the shadow cap currently bounds readers without matching writer enforcement. Explicit forget, orphan cleanup, and bounded history otherwise provide lifecycle rules.

7. **Plan revision recommendations**

   Add `## Revisions` entries specifying:

   - How repeated registrations retain bounded live alias spellings, and how those spellings govern locks and legacy handoff.
   - A prepublication invariant covering every registered metadata producer, including legacy import and section-preserving shadows, with refusal before document mutation when the prospective result exceeds its envelope.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"15ce5ed6-357a-4e41-8c31-1db418eb5595","repository":"github.com/xianxu/pair","issue_id":"000573","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/7ef127e48f5f4414","reviewed_head":"52a0ff84621e6a41f057af5a3b8d1c4d03c0d560","slot":"pair:5","requested_at":"2026-10-15T19:57:01.859517Z"},"phase":"finished","pid":98632,"started_at":"2026-10-15T19:57:02.10891Z","finished_at":"2026-10-15T19:59:40.001749Z","session_id":"01a14123-b8c4-76a1-b0ee-e416704ea4bc","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.817890708s
head: "52a0ff84621e6a41f057af5a3b8d1c4d03c0d560"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-15T14:23:50-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 573 — Agentic documents: Couch registry so session-sync verbs work for any submitted file (universal-agent-input-interface S1, Couch side) |
| repo | pair |
| issue file | workshop/issues/000573-agentic-documents-couch-registry-so-session-sync-verbs-work-for-any-submitted-file-universal-agent-input-interface-s1-couch-side.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6b567c2182cce01693eb8eb3c68c2addc4f10ae6..284821c42463ed6ba9baebcece98f2f83f087928 |
| command | sdlc close --issue 573 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T14:23:50-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-1 is verified addressed. BR-2’s original regressions now pass, but its writer/reader rule remains incomplete: successful forget can export metadata that re-registration rejects. A separate filesystem-dependent assertion also fails in this review environment.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Additional spellings now pass through the bounded registry transition model and feed lock checks and legacy export. Removing existing-registration spelling admission in a disposable copy made both alias-lock and alias-export regression tests fail; restoring it made them pass.
  - id: BR-2
    disposition: not-addressed
    note: |
      The import, preserved-shadow and fold-state regressions fail when their bounds are removed and pass with the fixes restored. However, registry.go:612-620 adds SubmitPaths during legacy export without accounting for the import reader's 1 MiB limit. A valid 1,048,440-byte state registered successfully; forget succeeded after exporting 1,048,829 bytes; re-registration then failed with registry-limit. Complete the original persisted-writer-reader-envelope rule across import, read refresh, write and legacy handoff, with a round-trip regression. ARCH-CONSTRAINTS / ARCH-FUNERAL / ARCH-PURPOSE.
findings:
  - id: new
    severity: Important
    kind: defect
    family: filesystem-independent-test-fixtures
    title: |
      Symlink regression assumes the temporary directory has a noncanonical spelling
    detail: |
      cmd/internal/sessionsync/registry_test.go:408 requires Changed=true when registering realPath after registering its symlink alias. When the temporary directory is already canonical, realPath is the stored canonical File, so Changed=false is correct. The targeted package run and an isolated rerun both fail at line 409. Create another distinct alias to test admission, and separately assert that registering the already-known canonical spelling is unchanged.
```

1. **Strengths**

   - Registry transitions deep-clone history and enforce bounded alias admission.
   - Controlled flock barriers test waiting writers against rename and forget.
   - Broker admission timestamps preserve queued-submit attribution through rename and pathname reuse.
   - README, atlas and embedded skill document the new commands, uncertainty contract and lock-age rule.

2. **Critical findings**

   - **BR-2 remains open — `registry.go:612-620`.** Make export and subsequent import obey a compatible envelope. Either reserve handoff overhead before accepting growth or accommodate the bounded exported representation during import while validating the extracted central state. Preserve history and folds.

3. **Important findings**

   - **`registry_test.go:408`:** remove the temporary-path assumption described above. Correct canonical-path idempotency currently fails the test.

4. **Minor findings**

   - None.

5. **Test coverage notes**

   - `couchmessage`, `couchcmd` and `artifactpath` package tests passed.
   - `sessionsync` failed only the identified symlink assertion.
   - `sessionsync` race tests passed with that test explicitly excluded.
   - Mutation checks confirmed both prior fixes have meaningful regression tests.
   - The export/re-registration reproduction failed on unchanged production code. The disposable copy was removed; tracked files remain unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared spelling inventory and metadata validator consolidate behavior.
   - **ARCH-PURE — pass:** registry decisions and validation remain pure; IO stays in the store/resolver.
   - **ARCH-PURPOSE — flag:** BR-2’s metadata-producer sweep still omits round-trip handoff compatibility.
   - **ARCH-MOCK — pass:** isolated filesystem backends and injected publication/locking seams exercise production flow.
   - **ARCH-CONSTRAINTS — flag:** exported state can exceed its subsequent import envelope.
   - **ARCH-SECURE — pass:** registry decoding rejects malformed authority; metadata paths derive from validated IDs.
   - **ARCH-ORDER — pass:** authority transitions, uncertain publication and waiting-writer rechecks have concrete coverage.
   - **ARCH-FUNERAL — flag:** size accounting excludes history added at handoff; explicit forget otherwise supplies cleanup.

7. **Plan revision recommendations**

   Add a `## Revisions` entry defining the compatible export/import envelope and naming a register → forget → re-register boundary-size regression. Record that canonical-path registration is already known and need not report a change.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"ebaa2ee4-69d0-4665-99f2-c9449f6667e5","repository":"github.com/xianxu/pair","issue_id":"000573","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/0b3e50c38dcb4b3a","reviewed_head":"284821c42463ed6ba9baebcece98f2f83f087928","slot":"pair:5","requested_at":"2026-10-15T21:20:18.897774Z"},"phase":"finished","pid":77518,"started_at":"2026-10-15T21:20:19.247907Z","finished_at":"2026-10-15T21:23:50.055358Z","session_id":"01a1416f-f910-7f20-8c3a-35e6858bf076","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.087138667s
head: "284821c42463ed6ba9baebcece98f2f83f087928"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-15T14:47:15-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 573 — Agentic documents: Couch registry so session-sync verbs work for any submitted file (universal-agent-input-interface S1, Couch side) |
| repo | pair |
| issue file | workshop/issues/000573-agentic-documents-couch-registry-so-session-sync-verbs-work-for-any-submitted-file-universal-agent-input-interface-s1-couch-side.md |
| boundary | whole-issue close |
| milestone | — |
| window | a0e3e4d2711bb33b0227c471283f60ead3300c45..e4468d7250773c91e65f0fbc1ef41694bd4089ca |
| command | sdlc close --issue 573 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T14:47:15-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The registry, rename attribution, documentation, and handoff bounds are substantially implemented. BR-3 is fixed. BR-2 remains blocking: a valid near-limit state can pass write preflight, then trigger a size refusal **after the document has changed**.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The shared spelling inventory feeds lock checks and legacy export; the additional-alias regressions pass in the sessionsync package run. The prior disposition stands.
  - id: BR-2
    disposition: not-addressed
    note: |
      The handoff reservation fixes have meaningful regression coverage: removing SubmitPaths reservation makes admission, alias, rename, and write tests fail. However, sync.go:442 validates post-fold state, then changes the document at line 446; saveShadow at line 449 revalidates the old f.state at line 376. Reproduced with valid handoff state sized to 1 MiB minus 16 bytes, obsolete folds, a new delivery gap, and write --accept-gap: post-fold preflight passes, but shadow publication returns registry-limit with documentChanged=true. This remains the persisted-writer-reader-envelope family: use one prepared, validated metadata state throughout publication, and sweep every subsequent writer/validator for stale-state checks. ARCH-CONSTRAINTS / ARCH-ORDER.
  - id: BR-3
    disposition: addressed
    note: |
      registry_test.go:406-425 now resolves the canonical spelling and expects an unchanged registration, then uses a distinct leaf alias to test admission and retry. The corrected test passes with an explicitly canonical TMPDIR, and the complete sessionsync package passes.
```

1. **Strengths**

   Stable identities preserve metadata across rename; shared admitted spellings cover alias locks and export. Broker admission timestamps correctly separate submit attribution from delivery completion. README, atlas, help, and embedded Couch instructions document the new surface.

2. **Critical findings**

   **BR-2 remains open** at [sync.go:449](/Users/xianxu/Library/Caches/sdlc/review-checkouts/7a80633e3ddc7205/cmd/internal/sessionsync/sync.go:449). `owed` adds `GapAccepted` to the loaded state. The prospective state then prunes obsolete folds and fits the envelope, but `saveShadow` validates the unpruned state after document replacement.

   Pass the prepared state through shadow/state publication, ensuring all deterministic validation completes before document mutation. Add a regression asserting that this valid post-pruning write succeeds and publishes consistent document, shadow, and state.

3. **Important findings**

   None additional.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed `go test` for sessionsync, couchmessage, couchcmd, and artifactpath; `git diff --check` also passed. The handoff guard mutation produced the expected regression failures. The additional gap/pruning reproduction failed on pinned HEAD with `registry-limit` and `documentChanged=true`. Disposable copies were removed; the checkout remains unchanged. The full repository suite was not rerun.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared metadata validation and spelling inventory.
   - **ARCH-PURE — pass:** registry transitions and ownership decisions remain separate from filesystem effects.
   - **ARCH-PURPOSE — flag:** BR-2 still violates the promised pre-mutation refusal contract.
   - **ARCH-MOCK — pass:** portable filesystem fixtures and existing broker seams exercise production paths.
   - **ARCH-CONSTRAINTS — flag:** publication validates a different state from preflight.
   - **ARCH-SECURE — pass:** bounded regular-file reads and strict registry decoding fail visibly.
   - **ARCH-ORDER — flag:** a deterministic refusal follows an already-applied document effect.
   - **ARCH-FUNERAL — pass:** explicit forget, capacity limits, and orphan cleanup bound retained artifacts.

7. **Plan revision recommendation**

   Add a `## Revisions` entry requiring one prepared document/shadow/state bundle for each write, validated before effects and consumed consistently by every publication step. Include the near-limit gap-acceptance plus fold-pruning regression.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"15980d7a-743f-4d5d-94e1-e7dc003f5be5","repository":"github.com/xianxu/pair","issue_id":"000573","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/7a80633e3ddc7205","reviewed_head":"e4468d7250773c91e65f0fbc1ef41694bd4089ca","slot":"pair:5","requested_at":"2026-10-15T21:43:44.532088Z"},"phase":"finished","pid":88042,"started_at":"2026-10-15T21:43:44.895579Z","finished_at":"2026-10-15T21:47:11.70255Z","session_id":"01a14185-6c7a-70a1-be39-49484dd9feda","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.869533584s
head: "e4468d7250773c91e65f0fbc1ef41694bd4089ca"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-15T15:14:35-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 573 — Agentic documents: Couch registry so session-sync verbs work for any submitted file (universal-agent-input-interface S1, Couch side) |
| repo | pair |
| issue file | workshop/issues/000573-agentic-documents-couch-registry-so-session-sync-verbs-work-for-any-submitted-file-universal-agent-input-interface-s1-couch-side.md |
| boundary | whole-issue close |
| milestone | — |
| window | e0b46192c306a9167db74c69f747c932f49e4abc..9a428a8590bb98e779e633abcaa63deaa3163b55 |
| command | sdlc close --issue 573 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T15:14:35-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation delivers the issue’s registry, sync operations, ownership, rename and lock-age behavior. BR-2 is addressed: writes publish the exact prevalidated bundle, and regression tests demonstrably fail with the previous implementation. No new blocking findings.

```findings
dispose:
  - id: BR-2
    disposition: addressed
    note: |
      Metadata admission/growth reserves complete handoff state. Current regressions pass; restoring the pre-fix sync.go makes gap acceptance fail with documentChanged=true and makes ambiguous-result refusal change metadata. Temporary mutation copy removed.
  - id: BR-1
    disposition: addressed
    note: |
      Additional-alias lock and legacy-export regressions pass; admitted spellings feed the shared canonical-document authority checks.
  - id: BR-3
    disposition: addressed
    note: |
      registry_test.go:406 explicitly resolves the canonical spelling and uses a distinct leaf alias for Changed=true; the session-sync suite passes.
```

1. **Strengths**

   - Write preparation validates content, metadata limits and result hash before document mutation (`cmd/internal/sessionsync/sync.go:411`).
   - Shared metadata validation reserves historical paths needed for legacy handoff (`cmd/internal/sessionsync/registry.go:32`).
   - Controlled broker delivery tests cover queued submits across rename and pathname reuse.
   - README, atlas and embedded Couch guidance document the new public surface and lifecycle.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed independently:

   - Session-sync, Couch CLI, broker and artifact-inventory package tests.
   - Complete session-sync race suite.
   - Wrapper broker-to-runtime envelope regression.

   Both prepared-publication regressions failed against the previous write implementation. The pinned checkout remains unchanged. Operator editor acceptance remains the explicitly approved postlanding check.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared ledger, publication and spelling helpers.
   - **ARCH-PURE — pass:** deterministic registry transitions and write preparation separated from publication.
   - **ARCH-PURPOSE — pass:** registration feeds all four sync operations and owed-event handling.
   - **ARCH-MOCK — pass:** portable filesystem backend and injected delivery/publication seams exercise production boundaries.
   - **ARCH-CONSTRAINTS — pass:** bounded metadata, registry capacity and lock/diff execution.
   - **ARCH-SECURE — pass:** registry validation, regular-file checks and validated identity-derived storage paths.
   - **ARCH-ORDER — pass:** serialized authority transitions, controlled waiting-writer tests and preserved admission timestamps.
   - **ARCH-FUNERAL — pass:** explicit forget, bounded history and serialized orphan cleanup.

7. **Plan revision recommendations:** None; existing revisions describe the final implementation and approved live-check timing.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"40ad9eab-6c92-48e1-b58f-434a0238b203","repository":"github.com/xianxu/pair","issue_id":"000573","recipe":"milestone-review","round":4,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/fa3c44f2938b357b","reviewed_head":"9a428a8590bb98e779e633abcaa63deaa3163b55","slot":"pair:5","requested_at":"2026-10-15T22:11:30.299399Z"},"phase":"finished","pid":18870,"started_at":"2026-10-15T22:11:30.639819Z","finished_at":"2026-10-15T22:14:34.344449Z","session_id":"01a1419e-d7d8-7580-baf5-a1820566424b","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.918963333s
head: "9a428a8590bb98e779e633abcaa63deaa3163b55"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
