# Boundary Review — pair#617 (whole-issue close)

| field | value |
|-------|-------|
| issue | 617 — pair title processes outlive their threads (28 orphans, 2.8 GB) and some grow to ~280 MB |
| repo | pair |
| issue file | workshop/issues/000617-pair-title-processes-outlive-their-threads-28-orphans-2-8-gb-and-some-grow-to-280-mb.md |
| boundary | whole-issue close |
| milestone | — |
| window | e16f1373fd78c912d0a3be90e68972baa3cd6347..21453132e5af1a4fe4fb296a8530d21d41b30aa4 |
| command | sdlc close --issue 617 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-16T18:09:11-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The wrapper lease, owned process-group teardown, and incremental title projection are well covered, and all five selected packages passed. One localized lifecycle gap remains: interrupted atomic publications can leave temporary files that block storage collection. Fix that recovery path before shipping.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: interrupted-publication-recovery
    title: |
      Atomic title publications leave unrecoverable temporary artifacts
    detail: |
      cmd/internal/titlepoller/runtime.go:50 now routes PID and workspace-owner writes through osfs.FS.WriteAtomic, which creates sibling names such as title-pid-TAG.123456789. Lease EOF can SIGKILL the title between creation and rename; rename failure also leaves the temporary file. These names have no recovery path: artifactpath/gc.go recognizes the final PID path, while storagegc/stateio.go:116 recognizes only .pending-* staging names. storagegc/inventory.go:240 treats the leftovers as unknown artifacts and blocks collection. ARCH-FUNERAL: provide fenced recovery for both publication families and test interruption after temporary-file creation, including preservation of active publishers.
```

1. **Strengths**

   - `titlepoller/runcli.go` validates process-group ownership before arming immediate teardown. Subprocess tests cover blocked proof work, retention acquisition, and stubborn command descendants.
   - `sessioninventory/title_projection.go` reuses existing authorization and provider parsers, then bounds tail reads and partial-record storage. Tests cover provider parity, split appends, mutation, and owner supersession.
   - Legacy cleanup separates classification from effects, intersects fresh eligibility with approved generations, and rechecks identity before each signal.

2. **Critical findings**

   None.

3. **Important findings**

   The publication-recovery finding above originates at `cmd/internal/titlepoller/runtime.go:50`. Preserve atomic publication, but give abandoned temporary files an automatic, ownership-safe removal path. A deferred unlink alone cannot handle SIGKILL.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Ran successfully:

   ```text
   go test ./cmd/internal/titlepoller ./cmd/internal/sessioninventory ./cmd/internal/wrapcmd ./cmd/internal/launcher ./cmd/internal/testpair
   ```

   Existing publication tests verify replacement of the visible inode, but do not exercise interruption before rename or subsequent retention recovery. I did not rerun the complete repository suite or live rollout.

6. **Architectural notes**

   - **ARCH-DRY — pass:** existing authorization, scanner, usage, artifact-path, and environment helpers are reused.
   - **ARCH-PURE — pass:** pure lifetime/classification decisions are separated from process effects; projection IO uses an injected runtime.
   - **ARCH-PURPOSE — pass:** the implementation covers wrapper lifetime, repeated memory work, and fenced legacy cleanup.
   - **ARCH-MOCK — pass:** stateful inventory doubles and controlled subprocess fixtures exercise production boundaries.
   - **ARCH-CONSTRAINTS — pass:** chunk and partial-record limits are enforced; tests check history-independent retained projection state.
   - **ARCH-SECURE — pass:** generation identity, incomplete process evidence, and subprocess environment isolation receive explicit handling.
   - **ARCH-ORDER — pass:** tests control EOF during blocked work, replacement, and authority changes during tail processing.
   - **ARCH-FUNERAL — flag:** interrupted publication artifacts lack recovery.

   Atlas and README changes cover the introduced architectural and fixture surfaces.

7. **Plan revision recommendations**

   Add a `## Revisions` entry naming the temporary-publication lifecycle: creator, active-publisher protection, recovery owner, removal trigger, and interruption regression. Apply it to both title PID and workspace-owner publications.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"ce415ae0-4d4e-48d7-825c-f72a9cde21a3","repository":"github.com/xianxu/pair","issue_id":"000617","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/0bdf4acd453d6a0b","reviewed_head":"21453132e5af1a4fe4fb296a8530d21d41b30aa4","slot":"pair:6","requested_at":"2026-10-17T01:07:06.892533Z"},"phase":"finished","pid":66217,"started_at":"2026-10-17T01:07:07.021857Z","finished_at":"2026-10-17T01:09:09.022683Z","session_id":"01a14765-f819-7dc3-a9e2-6b6d58692155","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.211596459s
head: "21453132e5af1a4fe4fb296a8530d21d41b30aa4"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-16T18:34:59-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 617 — pair title processes outlive their threads (28 orphans, 2.8 GB) and some grow to ~280 MB |
| repo | pair |
| issue file | workshop/issues/000617-pair-title-processes-outlive-their-threads-28-orphans-2-8-gb-and-some-grow-to-280-mb.md |
| boundary | whole-issue close |
| milestone | — |
| window | e16f1373fd78c912d0a3be90e68972baa3cd6347..20e3a811d207d6f6cd73dbb2df2df18733bd6837 |
| command | sdlc close --issue 617 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-16T18:34:59-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range satisfies the revised lifetime, memory, and publication-recovery contracts. BR-1 is addressed with reachable regression coverage that fails when the old writer is restored. No new blocking findings. Installed-live acceptance remains explicitly scheduled after landing and the coordinated restart.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Both title publication families now use root-locked private pending staging through PublishMetadata. Production CLI interruption tests verify preservation of the previous target, exclusion of active publishers, and recovery after EOF termination; Collector.applyPage invokes that recovery automatically. Restoring the old writer in a disposable overlay made TestTitlePublicationRenameFailureLeavesNoUnclassifiedSibling fail for both families with sibling residue.
```

1. **Strengths**

   - Wrapper ownership uses a sole CLOEXEC writer and an isolated title process group; subprocess tests verify SIGKILL, inheritance, replacement, and descendant teardown.
   - The title projection reuses existing authorization and provider parsers, bounds appended reads, and stops on authority or artifact contradictions.
   - Publication recovery shares retention’s existing atomic publisher, lock, and cleanup mechanism.
   - README and atlas document the new environment helper, ownership flow, and recovery contract.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Focused tests passed across titlepoller, sessioninventory, storagegc, wrapcmd, launcher, and testpair. Complete titlepoller, sessioninventory, storagegc, testpair, and artifactpath package tests also passed. The publication mutation failed as expected for both artifact families. `git diff --check` passed. I did not repeat the complete repository suite or installed-live acceptance.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared atomic publication/recovery and existing provider validation.
   - **ARCH-PURE — pass:** declared pure entities have direct tests; process/filesystem effects remain behind integration seams.
   - **ARCH-PURPOSE — pass:** both launcher spawn sites are removed; lifetime, memory work, and legacy cleanup are covered.
   - **ARCH-MOCK — pass:** stateful inventory/runtime doubles complement real kernel-process tests.
   - **ARCH-CONSTRAINTS — pass:** tail reads, partial records, retained history, and shutdown have exercised bounds.
   - **ARCH-SECURE — pass:** generation checks, selected publication paths, unsafe-target rejection, and disposable test environments.
   - **ARCH-ORDER — pass:** terminal lifetime transitions and controlled interruption/mutation sequences are tested.
   - **ARCH-FUNERAL — pass:** child ownership and interrupted staging both have verified termination/removal paths.

7. **Plan revision recommendations:** None; existing revisions describe the final implementation and deferred live acceptance.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"2cfbfc7e-e458-4f72-8e0f-4e3f8920859a","repository":"github.com/xianxu/pair","issue_id":"000617","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/ae28cf6c4bc224ca","reviewed_head":"20e3a811d207d6f6cd73dbb2df2df18733bd6837","slot":"pair:6","requested_at":"2026-10-17T01:33:08.486677Z"},"phase":"finished","pid":19887,"started_at":"2026-10-17T01:33:08.666176Z","finished_at":"2026-10-17T01:34:57.973586Z","session_id":"01a1477d-cdbc-7f32-8594-5115e80a7b4b","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.83319s
head: "20e3a811d207d6f6cd73dbb2df2df18733bd6837"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000408
