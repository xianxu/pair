# Boundary Review — pair#366 (whole-issue close)

| field | value |
|-------|-------|
| issue | 366 — Make Couch a local singleton |
| repo | pair |
| issue file | workshop/issues/000366-couch-singleton.md |
| boundary | whole-issue close |
| milestone | — |
| window | f0c1e56689469666b1aaaac708538cbf95f06f1d..603bb5715a0057ee16cc383b11f3d7292b152558 |
| command | sdlc close --issue 366 |
| reviewer | codex |
| timestamp | 2026-10-01T23:21:41-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The singleton lease and selected-root wiring are well separated, and existing focused tests pass. Three independently reproduced adoption bugs block shipping: ignored slot corruption, ignored surviving incarnations in excluded stores, and publication of selections that subsequent reads reject.

1. **Strengths**
   - Host ownership precedes runtime construction; teardown releases the store lease before the host lease.
   - Selected roots reach messaging, artifact readers, tracing, and both child-launch paths.
   - Tests exercise contention, crash/exec release, stale receipts, publication failure, and lost-acknowledgment retry.
   - README and atlas describe migration, isolation, and compatibility limits.

2. **Critical findings**

   The three findings below each have a failing scratch regression against the pinned HEAD.

```findings
findings:
  - id: new
    severity: Critical
    family: adoption-evidence-completeness
    title: |
      Adoption ignores unreadable numbered-slot state.
    detail: |
      cmd/internal/couchsingleton/inspect.go:210 checks only PreviewSnapshot's returned error and snapshot.Unreadable. Numbered-slot failures are instead carried in snapshot.Slots[].Err (cmd/internal/couchcore/threadstore_snapshot.go:31–53). A corrupt slot thread.json therefore produces READY with no blockers; TestReviewCorruptSlotBlocksAdoption reproduces this. Inspect every slot observation and refuse unresolved state. Include slot-local evidence in the digest and publication revalidation, since hashTree currently covers only the global namespace. ARCH-SECURE, ARCH-PURPOSE.
  - id: new
    severity: Critical
    family: migration-liveness-evidence
    title: |
      Exclusion admits stores with surviving live incarnations.
    detail: |
      cmd/internal/couchsingleton/inspect.go:191 observes only the supervisor lease and discards snapshot.Records; model.go:95 then accepts exclusion of a populated store. With a free supervisor lease and a recorded incarnation whose PID/start identity is still live, preview returns READY. TestReviewExcludedLiveWrapperBlocksAdoption reproduces this. This contradicts the issue's requirement to report surviving wrappers in other namespaces as migration conflicts. Inspect incarnation ownership through the existing process/lifecycle seams, preserve unknown outcomes, and block exclusion until absence is established. ARCH-ORDER, ARCH-PURPOSE.
  - id: new
    severity: Critical
    family: persisted-writer-reader-contract
    title: |
      Successful adoption can publish an unreadable selection.
    detail: |
      cmd/internal/couchsingleton/manager.go:204–212 publishes serialized Selection without enforcing Read's 65,536-byte limit at line 37. Requests permit up to 4096 exclusions. A valid 600-exclusion request successfully published 139,628 bytes, after which Read refused with "file exceeds 65536 bytes"; TestReviewSelectionWriterReadLimit reproduces this. Share the size contract between writer and reader, reject oversized selections before publication, and test the boundary plus successful round trips. ARCH-CONSTRAINTS, ARCH-SECURE.
```

3. **Important findings:** None additional.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Passed: full `couchsingleton` and `couchidentity` suites under `-race`; focused singleton/runtime/adoption tests; focused supervisor-observation and concurrent-FakeGit tests; pinned-range `git diff --check`.
   - All three new scratch regressions fail as described. They used a temporary Go overlay; repository files were unchanged.
   - Full repository tests were not rerun. Existing migration fixtures do not cover these failure classes.

6. **Architectural notes**
   - **ARCH-DRY — pass:** existing leases, identity decoders, and atomic publication are reused.
   - **ARCH-PURE — pass:** adoption policy is separated from filesystem/process inspection.
   - **ARCH-PURPOSE — flag:** incomplete migration evidence violates the preservation/refusal contract.
   - **ARCH-MOCK — pass:** inspected tests use injected roots, process seams, controlled filesystems, and subprocesses.
   - **ARCH-CONSTRAINTS — flag:** writer and reader size envelopes disagree.
   - **ARCH-SECURE — flag:** slot-local corruption is silently omitted from admission evidence.
   - **ARCH-ORDER — flag:** absence of a supervisor lease is insufficient evidence that another inventory has no surviving actors.
   - **ARCH-FUNERAL — pass:** new authority artifacts use fixed paths; lease lifetime and staging are bounded.

7. **Plan revision recommendations**

   Append a timestamped `## Revisions` entry covering complete slot evidence and revalidation, surviving-incarnation admission rules, and the shared selection-size limit. Add the three regressions to the verification contract before marking those implementation items complete again.

---

## Re-review — 2026-10-01T23:46:32-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 366 — Make Couch a local singleton |
| repo | pair |
| issue file | workshop/issues/000366-couch-singleton.md |
| boundary | whole-issue close |
| milestone | — |
| window | f0c1e56689469666b1aaaac708538cbf95f06f1d..32ead2591590380be4673f40472d3d14a6e583fd |
| command | sdlc close --issue 366 |
| reviewer | codex |
| timestamp | 2026-10-01T23:46:32-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

All three prior findings are addressed, with regression tests that fail when their fixes are disabled. One separate correctness bug blocks shipping: Couch passes its global Pair data root to children as a repository-scoped directory, so the actual Pair launcher reads and writes the wrong artifact locations.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Slot errors, payloads, and topology now participate in locked inspection and publication revalidation. Disabling the slot-error guard makes TestAdoptionRejectsUnreadableSlotEvidence/corrupt fail.
  - id: BR-2
    disposition: addressed
    note: |
      Exclusion checks recorded incarnations and creating owners, preserving unknown liveness and rechecking before publication. Disabling the liveness guard makes live, unknown, and identity-error regression cases fail.
  - id: BR-3
    disposition: addressed
    note: |
      Preview, publication, and reading share maxSelectionBytes. Exact-boundary round trips pass; disabling the encoding limit makes oversize-publication regressions fail.
findings:
  - id: new
    severity: Critical
    family: resolved-runtime-root-propagation
    title: |
      Pair children interpret the selected global root as a repository-scoped artifact directory.
    detail: |
      cmd/internal/couchcmd/singleton.go:247 exports roots.PairDataDir as PAIR_DATA_DIR, but cmd/internal/launcher/runcli.go:102–118 treats that variable as an already-scoped directory and derives its global root from HOME/XDG without consuming COUCH_PAIR_DATA_DIR. Hosted artifacts therefore use the flat root; custom selected roots also leave global claim readers pointed at ambient storage. Fix the complete parent/launcher root contract and add actual launcher artifact/read/resume coverage. ARCH-PURPOSE, ARCH-DRY.
```

1. **Strengths**

   - Adoption holds global and numbered-slot transaction locks through publication; inspection authority expires when its callback returns.
   - Process observations preserve uncertainty and distinguish recycled PIDs from surviving incarnations.
   - Selection size tests cover exact 64 KiB payloads, JSON escaping, overflow refusal, and successful readback.
   - README and atlas document adoption, exclusions, isolation, and legacy-binary prerequisites.

2. **Critical findings**

   The root-contract finding above originates at [singleton.go:247](/Users/xianxu/workspace/worktree/pair-slot2/pair/cmd/internal/couchcmd/singleton.go:247). Its consumer at [runcli.go:102](/Users/xianxu/workspace/worktree/pair-slot2/pair/cmd/internal/launcher/runcli.go:102) interprets the value differently.

   A temporary overlay test invoking real `LaunchNative(["list"])` reproduced this: the correctly scoped control read `codex`, while Couch’s environment read `wrong-flat-agent` from the global directory. Make the launcher consume the selected global root and derive the appropriate repository scope. Sweep artifact writers, claim readers, and resume paths together.

3. **Important findings**

   None additional.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed normal singleton, identity, and couchcmd suites; singleton and identity race suites; focused core inspection, process, and supervisor tests; and pinned-range whitespace checks. Each prior fix was independently mutation-tested using temporary overlays.

   Existing child tests assert environment strings—including the incorrect global-as-scoped value—without exercising Pair’s consumption. The launcher reproduction exposes that gap. A subsequent repeat encountered sandbox Go-cache permissions. Repository files remain unchanged.

6. **Architectural notes**

   | Marker | Result |
   |---|---|
   | ARCH-DRY | **Flag:** parent and launcher apply incompatible root-resolution rules. |
   | ARCH-PURE | Pass: deterministic selection policy and explicit IO boundaries. |
   | ARCH-PURPOSE | **Flag:** selected-root propagation stops before the actual Pair consumer. |
   | ARCH-MOCK | Pass: injected process state, portable storage fixtures, and real subprocess checks. |
   | ARCH-CONSTRAINTS | Pass: explicit inspection and serialization bounds; no recurring discovery added. |
   | ARCH-SECURE | Pass: bounded parsing, path validation, and conservative unknown-state handling. |
   | ARCH-ORDER | Pass: ordered leases, publication revalidation, and lost-acknowledgment recovery tests. |
   | ARCH-FUNERAL | Pass: fixed metadata filenames, bounded exclusions, and owned lease lifetimes. |

7. **Plan revision recommendations**

   Append a `## Revisions` entry reopening Task 3’s child-root integration and Task 4’s acceptance coverage. Name the actual Pair launcher as a consumer and require artifact creation/read/resume tests with selected roots differing from HOME/XDG.

---

## Re-review — 2026-10-02T00:01:01-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 366 — Make Couch a local singleton |
| repo | pair |
| issue file | workshop/issues/000366-couch-singleton.md |
| boundary | whole-issue close |
| milestone | — |
| window | f0c1e56689469666b1aaaac708538cbf95f06f1d..acc6a8f30216f114cba64931cdf8e06f5859dc8c |
| command | sdlc close --issue 366 |
| reviewer | codex |
| timestamp | 2026-10-02T00:01:01-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation satisfies the issue’s revised singleton and preservation contract. BR-4 is addressed through the parent, launcher and embedded-runtime consumers, with meaningful regression coverage. No new blocking findings were identified.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Slot inspection retains transaction locks and includes unreadable state, topology and payload evidence; corresponding regressions pass.
  - id: BR-2
    disposition: addressed
    note: |
      Excluded inventories require confirmed absence of recorded incarnations and creating owners; live, unknown and revalidation regressions pass.
  - id: BR-3
    disposition: addressed
    note: |
      Serialization and reading share maxSelectionBytes; boundary round-trip and pre-publication overflow regressions pass.
  - id: BR-4
    disposition: addressed
    note: |
      Couch clears PAIR_DATA_DIR and passes COUCH_PAIR_DATA_DIR; launcher and embedded extraction consume the selected global root. The real create/list/resume regression fails when an isolated Go overlay disables selected-root consumption, then passes against unchanged HEAD.
```

1. **Strengths**
   - Global and scoped roots are distinguished at the actual consumer: `cmd/internal/launcher/runcli.go:102`.
   - Adoption revalidates evidence while retaining identity and store transaction locks through publication: `cmd/internal/couchsingleton/manager.go:190`.
   - Tests exercise crash recovery, lost publication acknowledgments, competing owners and changed source evidence.
   - README and atlas document adoption, exclusions, isolation and old-binary limitations.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Full normal suites passed for `couchsingleton`, `couchidentity`, `launcher`, `pair-go` and `couchcmd`.
   - Focused race tests passed for singleton, identity, Couch core and storage GC.
   - BR-4 mutation failed at the expected ambient-root claim lookup; the unmodified regression passed afterward.
   - Pinned-range `git diff --check` passed. Repository remained unchanged.
   - Full-repository tests were not rerun during this review.

6. **Architectural notes**
   - **ARCH-DRY — pass:** Existing identity decoders, leases and durable publication are reused; launch and embedded extraction share root resolution.
   - **ARCH-PURE — pass:** Adoption policy is IO-free; filesystem and process observations remain in integration boundaries.
   - **ARCH-PURPOSE — pass:** Stable selection reaches runtime readers and hosted children.
   - **ARCH-MOCK — pass:** Injected process state and a stateful terminal fake exercise production boundaries.
   - **ARCH-CONSTRAINTS — pass:** Inspection has explicit limits; selected reads avoid recurring discovery.
   - **ARCH-SECURE — pass:** Bounded parsing, canonical-root validation and isolation checks reject malformed or escaping state.
   - **ARCH-ORDER — pass:** Ownership precedes effects; retained locks protect publication; failure and retry orderings have controllable tests.
   - **ARCH-FUNERAL — pass:** Selection and lease metadata have fixed bounds; publication staging is cleaned or reused; ownership releases on close or process death.

7. **Plan revision recommendations:** None. Existing revisions describe the delivered root-contract correction.

---

## Re-review — 2026-10-02T09:00:06-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 366 — Make Couch a local singleton |
| repo | pair |
| issue file | workshop/issues/000366-couch-singleton.md |
| boundary | whole-issue close |
| milestone | — |
| window | b1de974ba1610ff32d633b536b482af13271d8f2..3869582df43b6dc02659df1210bb790488f5d384 |
| command | sdlc close --issue 366 |
| reviewer | codex |
| timestamp | 2026-10-02T09:00:06-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The singleton ownership, adoption safeguards, and prior four corrections hold up under inspection and focused tests. One isolation defect blocks shipping: existing symlinks at the derived child HOME or TMPDIR let an explicitly isolated runtime write outside its root. A temporary overlay regression reproduced both escapes through the production runner without editing repository files.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      StoreInspection fences numbered-slot sources; unreadable-slot, changed-evidence, and publication-lock regressions pass.
  - id: BR-2
    disposition: addressed
    note: |
      ObserveMigrationProcesses covers incarnations and creating owners; live/unknown exclusion and publication recheck regressions pass.
  - id: BR-3
    disposition: addressed
    note: |
      encodeSelection shares the reader's size limit; exact-boundary round trips and overflow rejection regressions pass.
  - id: BR-4
    disposition: addressed
    note: |
      Selected global roots reach scoped launcher storage and embedded extraction; real create/list/resume and parent runner regressions pass.
findings:
  - id: new
    severity: Critical
    family: isolation-derived-root-containment
    title: |
      Derived child HOME and TMPDIR can escape explicit isolation through symlinks.
    detail: |
      cmd/internal/couchcmd/singleton.go:184-192 substitutes isolated/home without validating its physical destination and creates isolated/tmp without containment validation; line 252 exports both to children. Preexisting symlinks to an outside directory are followed by MkdirAll and descendant writes. A production-path overlay regression fails for both cases. ARCH-SECURE and ARCH-PURPOSE: validate physical containment of every derived runtime root before publication or child effects, and export only validated paths. Add permanent tests using outside temporary sentinels for fallback HOME and TMPDIR through both runner entry points.
```

1. **Strengths**
   - Host ownership and immutable selection remain separate; contention, crash recovery, and exec descriptor release have real-process coverage.
   - Adoption retains source transaction locks through publication and preserves unknown ownership as refusal.
   - README and atlas document adoption, exclusions, compatibility limits, and isolation.

2. **Critical findings**
   - The isolation escape above occurs at [singleton.go:184](/Users/xianxu/workspace/worktree/pair-slot2/pair/cmd/internal/couchcmd/singleton.go:184), with child propagation at line 252. Canonicalize and validate fallback HOME and derived TMPDIR before accepting them.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage**
   - Singleton and identity suites pass normally and under `-race`.
   - Focused core observation/inspection and runtime/launcher/embedded tests pass; pinned diff whitespace checks pass.
   - The added overlay regression fails for both `home` and `tmp`, confirming actual outside writes. Full repository tests were not rerun.

6. **Architecture**
   - **ARCH-DRY — pass:** existing identity decoders, leases, and transaction seams reused.
   - **ARCH-PURE — pass:** adoption policy separated from filesystem orchestration.
   - **ARCH-PURPOSE — flag:** descendant isolation remains incomplete.
   - **ARCH-MOCK — pass:** portable storage, stateful process/terminal doubles, and real subprocess checks.
   - **ARCH-CONSTRAINTS — pass:** bounded adoption scanning; no recurring discovery added.
   - **ARCH-SECURE — flag:** derived roots bypass physical containment checks.
   - **ARCH-ORDER — pass:** controlled contention, source changes, publication failures, and retry coverage.
   - **ARCH-FUNERAL — pass:** bounded selection metadata and scoped lease cleanup.

7. **Plan revisions**
   - Append a `## Revisions` entry covering containment of every derived child root, explicitly including fallback HOME and TMPDIR, with real descendant-write regression tests.

---

## Re-review — 2026-10-02T09:10:51-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 366 — Make Couch a local singleton |
| repo | pair |
| issue file | workshop/issues/000366-couch-singleton.md |
| boundary | whole-issue close |
| milestone | — |
| window | b1de974ba1610ff32d633b536b482af13271d8f2..fd511ef9eeec9fd421f87c827416186900ba61fa |
| command | sdlc close --issue 366 |
| reviewer | codex |
| timestamp | 2026-10-02T09:10:51-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation matches the revised Spec/Plan. BR-5 is addressed with production-path validation and regression tests that fail when the validation is disabled. No new blocking findings were identified. README and atlas cover the new adoption, singleton, and isolation behavior.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Slot inspection preserves unreadable-state errors and includes slot payloads in adoption evidence.
  - id: BR-2
    disposition: addressed
    note: |
      Nonselected stores retain live and unknown incarnation evidence, preventing unsafe exclusion.
  - id: BR-3
    disposition: addressed
    note: |
      Selection encoding and reading share the encoded-size limit, with boundary regression coverage.
  - id: BR-4
    disposition: addressed
    note: |
      Launcher and embedded runtime extraction consume the selected global root; launcher integration tests pass.
  - id: BR-5
    disposition: addressed
    note: |
      Derived HOME, TMPDIR and XDG roots are validated before publication and exported as physical paths. Tests pass normally and under race detection; disabling validation in a temporary overlay makes HOME/TMPDIR escape tests fail through both runner entry points.
```

1. **Strengths**
   - `singleton.go:66–103` centralizes derived-directory containment and type validation.
   - `singleton_derived_roots_test.go:17` covers escaping/dangling symlinks, regular files, adoption paths, and unchanged outside sentinels.
   - `manager.go:190–211` holds inspection locks through evidence revalidation and publication.
   - Selected global roots reach actual launcher consumers, with integration coverage.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage**
   - Passed full `couchsingleton`, `couchidentity`, `launcher`, and `couchcmd` suites.
   - Passed derived-root regressions under `-race` and targeted core inspection/observation tests.
   - Mutation overlay produced the expected regression failures.
   - `git diff --check` passed; repository remains unchanged.
   - Full repository suite was not rerun during this review.

6. **Architecture**
   - **ARCH-DRY — pass:** shared validation and existing lease/identity primitives.
   - **ARCH-PURE — pass:** adoption policy is IO-free; Manager owns filesystem effects.
   - **ARCH-PURPOSE — pass:** stable inventory, singleton ownership, preservation, and isolation are implemented.
   - **ARCH-MOCK — pass:** temporary backends, stateful doubles, and real subprocess checks exercise shared boundaries.
   - **ARCH-CONSTRAINTS — pass:** bounded inspection and nonblocking lifetime acquisition.
   - **ARCH-SECURE — pass:** strict persisted-state reads and validated derived roots.
   - **ARCH-ORDER — pass:** lock ordering, revalidation, publication recovery, and controllable failure tests.
   - **ARCH-FUNERAL — pass:** fixed selection/staging artifacts and scoped lease release avoid per-launch accumulation.

7. **Plan revisions:** None required.
