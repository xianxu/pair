# Boundary Review — pair#528 (milestone M1)

| field | value |
|-------|-------|
| issue | 528 — couch: resume a slot whose conversation binding is unproven: infer it from transcripts and the tty log, or start fresh oriented from the tty log |
| repo | pair |
| issue file | workshop/issues/000528-couch-resume-a-slot-whose-conversation-binding-is-unproven-infer-it-from-transcripts-and-the-tty-log-or-start-fresh-oriented-from-the-tty-log.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | e33a1f0e6e159a81eb147f7132c1b1ceb8c331d2..2efe103cf01924fa676be721a6779b8dff496418 |
| command | sdlc milestone-close --issue 528 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T11:01:12-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: medium
```

The revised M1 scope is substantially implemented: terminal admission is serialized, policy records reject malformed authority, capture provenance remains separate from native binding, and README/atlas are updated. However, uncertainty can still become a positive UI verdict, agent PID reuse is not excluded, and metadata reads can defeat the admission timeout. Verification was also incomplete: generated runtime assets are absent, and process-observation tests fail because `/bin/ps` execution is denied.

1. **Strengths**

   - `session_handoff.go` keeps the lease through admission and joins admission/client work before releasing it on failure. Channel-controlled tests exercise these ordering guarantees.
   - `session_policy.go` validates versions, required fields, duplicate keys, socket identity, and bounded JSON.
   - Capture publication checks the archived witness against the source witness; legacy captures remain explicitly uncertified.
   - README and atlas changes cover the new “needs relaunch” surface and explain the M1/M2 boundary.

2. **Critical findings**

   - **Unknown observations do not remain unknown.** At `cmd/internal/couchtty/menu_reattach.go:373`, an attached-pass entry promotes `ThreadUnusable/ReasonUnknown` to live. A successful earlier attach followed by a failed process-table refresh therefore displays live without current agent evidence. Separately, `session_handoff_os.go:73–97` converts policy/readiness/socket inspection errors into `Usable=false`, which `agent_usability.go:7` projects as “needs relaunch.” Introduce explicit usable/unsafe/unknown outcomes and preserve unknown through every projection and overlay. **ARCH-ORDER, ARCH-PURPOSE.**

   - **The recorded agent’s process generation is not checked.** `terminal_agent.go:7–34` verifies the server’s start identity but accepts any nonempty identity for `agentPID`. `session_handoff_os.go:205` supplies only the PID from readiness. If that PID is recycled into another descendant of the same long-lived server, stale readiness certifies it as the agent. Persist and compare the agent’s exact start identity; add a regression that changes only the agent generation while preserving its PID and ancestry. Existing tests change the server generation, missing this case. **ARCH-SECURE.**

3. **Important findings**

   - **Metadata reads can block indefinitely despite the admission bound.** `session_handoff_os.go:127` opens policy files without nonblocking/type checks; readiness reads at lines 193 and 423 use unbounded `os.ReadFile`. A FIFO can hang inventory or the admission observer, and the coordinator waits indefinitely while joining that observer. Reuse the cold-check branch’s regular-file, `O_NOFOLLOW|O_NONBLOCK`, size-limited reading approach across all these consumers. **ARCH-CONSTRAINTS, ARCH-SECURE.**

   - **Permanent lock artifacts have no aggregate bound.** `artifactpath/session_policy.go:151–165` creates a distinct lock path per socket/name, while `manifest.go:1460` permanently exempts every lock from collection. Stable inodes are necessary while contenders may exist, but unique terminal names accumulate indefinitely. Define a bounded coordination scheme or a safe retirement protocol; document and measure any intentional permanent per-session cost. **ARCH-FUNERAL.**

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Passed: full `storagegc` package; targeted artifact policy/witness/metadata tests; targeted launcher handoff, agent, launch-argument, usability, and lease tests.
   - Broader package testing failed on missing `runtimebundle/assets/runtime/files` and denied `/bin/ps` execution. These are verification limitations, not demonstrated patch regressions.
   - Add regressions for unknown-after-attach, metadata inspection failures, agent-only PID reuse, and FIFO/oversized metadata.
   - No prior findings required disposition. No repository files were modified.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared terminal observation serves inventory and selected-action checks.
   - **ARCH-PURE — pass:** classifiers and schema validation have direct value tests; handoff effects use an injected seam.
   - **ARCH-PURPOSE — flag:** truthful liveness remains incomplete through the overlay.
   - **ARCH-MOCK — pass, verification limited:** stateful handoff doubles control ordering; live conformance could not be independently completed here.
   - **ARCH-CONSTRAINTS — flag:** blocking metadata reads bypass the stated admission bound.
   - **ARCH-SECURE — flag:** agent-generation authority and file-reading safeguards are incomplete.
   - **ARCH-ORDER — flag:** uncertainty is collapsed into live or relaunch verdicts.
   - **ARCH-FUNERAL — flag:** stable lock files accumulate without an aggregate lifecycle bound.

7. **Plan revision recommendations**

   Add `## Revisions` entries defining the actual typed usability outcomes and their consumer matrix, the readiness agent-generation contract and legacy handling, and the lock-family lifecycle. Reconcile the concept inventory with implemented names and paths—for example, `AcquireTerminalLease` resides in `session_lease.go`, and the current usability projection is `agentUsabilityReason`, not the originally listed symbols.

```findings
findings:
  - id: new
    severity: Critical
    family: preserve-observation-uncertainty
    title: |
      Unknown terminal observations become live or actionable relaunch verdicts
    detail: |
      cmd/internal/couchtty/menu_reattach.go:373 promotes ReasonUnknown to ThreadLive whenever an attached-pass entry exists. cmd/internal/launcher/session_handoff_os.go:73-97 also collapses inspection errors into false usability, which cmd/internal/couchcore/agent_usability.go:7 converts to needs-relaunch. Preserve typed usable/unsafe/unknown outcomes through observers, classifiers, and overlays; test failed refresh after successful attachment and failed metadata inspection. ARCH-ORDER, ARCH-PURPOSE.
  - id: new
    severity: Critical
    family: exact-process-generation-authority
    title: |
      Agent usability accepts a recycled agent PID beneath the same server
    detail: |
      cmd/internal/launcher/terminal_agent.go:7-34 compares only the server generation; cmd/internal/launcher/session_handoff_os.go:205 supplies a readiness PID without its expected start identity. A replacement descendant reusing that PID passes stale readiness. Persist and validate the agent generation, with a regression changing only the agent identity while preserving PID and ancestry. ARCH-SECURE.
  - id: new
    severity: Important
    family: bounded-untrusted-artifact-reads
    title: |
      Policy and readiness reads can defeat the admission timeout
    detail: |
      cmd/internal/launcher/session_handoff_os.go:127 opens policy files without regular-file/nonblocking safeguards, and readiness reads at lines 193 and 423 are unbounded. A FIFO can hang refresh or admission indefinitely because cancellation cannot interrupt these reads and the coordinator joins the observer. Share the cold-check branch's bounded regular-file reader across these consumers and test FIFO and oversized inputs. ARCH-CONSTRAINTS, ARCH-SECURE.
  - id: new
    severity: Important
    family: durable-artifact-lifecycle-bound
    title: |
      Per-terminal coordination files accumulate without an aggregate lifecycle bound
    detail: |
      cmd/internal/artifactpath/session_policy.go:151-165 allocates one lock pathname per canonical socket, while cmd/internal/artifactpath/manifest.go:1460 permanently protects every such inode. Define a bounded coordination scheme or safe retirement protocol without unlinking active or awaited locks; document and measure any intentional permanent per-session cost. ARCH-FUNERAL.
```

---

## Re-review — 2026-10-13T11:34:11-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 528 — couch: resume a slot whose conversation binding is unproven: infer it from transcripts and the tty log, or start fresh oriented from the tty log |
| repo | pair |
| issue file | workshop/issues/000528-couch-resume-a-slot-whose-conversation-binding-is-unproven-infer-it-from-transcripts-and-the-tty-log-or-start-fresh-oriented-from-the-tty-log.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | e33a1f0e6e159a81eb147f7132c1b1ceb8c331d2..cf8eee44b80600320ef132d402877db7b4413646 |
| command | sdlc milestone-close --issue 528 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T11:34:11-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The four prior findings are addressed, with regression tests independently confirmed to fail when their protections are removed. The revised M1 scope and README/atlas updates are substantially delivered. One Important issue remains: two new artifact reads escape the bounded-read rule. Full OS integration verification also remains incomplete because `/bin/ps` execution is denied in this review environment.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Typed uncertainty reaches core projection and the attached-pass overlay. Focused tests pass; removing the unknown-state overlay guard or converting inspection errors to unsafe makes the respective regressions fail.
  - id: BR-2
    disposition: addressed
    note: |
      Readiness records carry agent start identity and AgentBelongsToServer validates it. Removing the agent-identity comparison fails the unchanged-PID/ancestry regression.
  - id: BR-3
    disposition: addressed
    note: |
      Policy and readiness consumers use readBoundedRegularMetadata. Its regressions pass; replacing it with os.ReadFile fails FIFO, oversized-input, symlink and cancellation cases. Additional consumers missed by the family sweep are reported separately below.
  - id: BR-4
    disposition: addressed
    note: |
      TerminalLease now selects among 256 permanent stripes. The 10,000-socket regression passes and fails when full-hash per-socket lock paths are restored.
findings:
  - id: new
    severity: Important
    family: bounded-untrusted-artifact-reads
    title: |
      Capture-witness and admission-config reads still permit indefinite blocking
    detail: |
      cmd/internal/storagegc/capture_metadata.go:142 opens events with blocking os.Open before its byte limit; a FIFO without a writer blocks capture preservation indefinitely, reproduced by a disposable regression probe. cmd/internal/launcher/session_handoff_os.go:468 also reads config.kdl with unbounded, non-cancellable os.ReadFile, so admission can remain stuck beyond its timeout while the coordinator waits for the observer. This is the 2nd finding in family bounded-untrusted-artifact-reads. Do NOT fix only these instances: enumerate every persisted-input read added by M1 and enforce one bounded, regular-file, nonblocking, cancellation-aware rule through a shared IO owner, with consumer-level FIFO and oversized-input regressions. ARCH-CONSTRAINTS, ARCH-SECURE, ARCH-PURPOSE.
```

1. **Strengths**

   - Unknown, usable and unsafe observations remain distinct through the repaired observer, classifier and overlay.
   - Agent generation checks reject PID reuse while allowing legitimate readiness replacement after native reexecution.
   - Stable lock stripes bound permanent coordination artifacts without unlinking inodes held by waiters.
   - Capture witnessing separates producer provenance from native-conversation proof; README and atlas document the new surface and M2 boundary.

2. **Critical findings**

   None.

3. **Important findings**

   The artifact-read finding above blocks acceptance. Move the reusable reader to an appropriate shared IO package and sweep all new consumers. A byte limit applied **after** a blocking open does not bound completion time.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Focused regression suites passed across artifactpath, launcher, couchcore, couchtty, storagegc and wrapcmd.
   - Focused race checks passed across launcher, artifactpath, couchcore and couchtty.
   - Mutation checks verified red evidence for BR-1 through BR-4.
   - A disposable FIFO probe reproduced the new capture-witness hang; it unblocked and joined its reader before failing.
   - Broad verification initially encountered missing generated runtime assets. Assets were generated only in the disposable copy. OS integration checks remained blocked by `fork/exec /bin/ps: operation not permitted`; the broad run was stopped without claiming success.
   - The disposable copy was removed; the pinned worktree remains unchanged.

6. **Architectural notes**

   - **ARCH-DRY — Pass:** shared terminal usability and admission paths replace independent decisions.
   - **ARCH-PURE — Pass:** projection, identity and schema logic have direct value tests; IO owners are separate.
   - **ARCH-PURPOSE — Flag:** the bounded-read correction addressed named consumers but missed enumerable siblings.
   - **ARCH-MOCK — Pass, with verification limitation:** stateful handoff fixtures control admission, cancellation and client exit; live OS conformance could not be independently completed here.
   - **ARCH-CONSTRAINTS — Flag:** remaining blocking reads defeat bounded completion.
   - **ARCH-SECURE — Flag:** special-file inputs still reach blocking opens.
   - **ARCH-ORDER — Pass for inspected sequences:** uncertainty survives projection, and handoff cleanup joins owned work before releasing coordination.
   - **ARCH-FUNERAL — Pass:** lock artifacts have a fixed aggregate bound; policy retirement and existing capture retention provide lifecycle owners.

7. **Plan revision recommendations**

   Add a `## Revisions` entry defining the bounded persisted-input read rule, enumerating all M1 consumers—including capture events and effective config—and naming the shared reader owner and consumer-level regression matrix. Record that OS integration verification must be rerun where process inspection is available.

---

## Re-review — 2026-10-13T12:06:16-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 528 — couch: resume a slot whose conversation binding is unproven: infer it from transcripts and the tty log, or start fresh oriented from the tty log |
| repo | pair |
| issue file | workshop/issues/000528-couch-resume-a-slot-whose-conversation-binding-is-unproven-infer-it-from-transcripts-and-the-tty-log-or-start-fresh-oriented-from-the-tty-log.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 9ea3bdcbdcd360324337d910ed419dc6a65443db..dfedd894650cbf1f3cc2dbdb5782b8724131f01d |
| command | sdlc milestone-close --issue 528 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T12:06:16-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

M1’s terminal admission, truthful liveness, and forward capture witnessing match the revised milestone scope. BR-5 is addressed through a shared reader and meaningful consumer regressions; an independent mutation made those regressions fail. No new blocking findings. Confidence is limited by unavailable live process inspection in this review environment.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Prior disposition retained; unknown-observation and reattach-overlay regressions pass.
  - id: BR-2
    disposition: addressed
    note: |
      Prior disposition retained; AgentBelongsToServer checks both server and agent process generations.
  - id: BR-3
    disposition: addressed
    note: |
      Prior disposition retained; policy and readiness consumers use the shared bounded reader, with passing FIFO regressions.
  - id: BR-4
    disposition: addressed
    note: |
      Prior disposition retained; permanent coordination files remain bounded to 256 stripes, with passing lease tests.
  - id: BR-5
    disposition: addressed
    note: |
      Enumerated metadata consumers use osfs.ReadRegularMetadata. Consumer FIFO, oversized-input and cancellation tests pass; substituting os.ReadFile in a disposable checkout makes the regressions fail. Both disposable checkouts were removed.
```

1. **Strengths**

   - Shared metadata IO enforces regular-file, size, cancellation, and identity checks (`cmd/internal/osfs/metadata_read.go:39`).
   - Capture preservation distinguishes unusable optional sidecars from invalid authority, retaining raw-only history without certification (`cmd/internal/launcher/capture_witness_test.go:45`).
   - Handoff tests control admission and client completion independently and verify cleanup before lease release.
   - README and atlas describe the new relaunch behavior, policy, locking, and witness contracts.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Full `osfs` and `storagegc` suites passed. Targeted launcher admission, lease, uncertainty, and capture tests passed. After generating runtime assets in a disposable pinned-head checkout, targeted artifactpath, wrapcmd, couchcore, and couchtty tests passed.

   The mutation independently reproduced FIFO blocking, oversized-config acceptance, and lost cancellation handling. Broader initial checks encountered missing generated assets and `/bin/ps: operation not permitted`; live process conformance and the complete suite were not independently verified.

6. **Architectural notes**

   - **ARCH-DRY — pass:** added metadata consumers share one IO owner.
   - **ARCH-PURE — pass:** parsing and usability decisions remain separately testable from filesystem/process observation.
   - **ARCH-PURPOSE — pass:** BR-5 addresses the enumerated read family; M2 recovery selection remains explicitly outside this boundary.
   - **ARCH-MOCK — pass:** stateful handoff/quiescence seams support controlled failures; live conformance tests exist.
   - **ARCH-CONSTRAINTS — pass:** byte limits, nonblocking opens, cancellation checks, and bounded admission are represented and tested.
   - **ARCH-SECURE — pass:** persisted authority is validated; malformed and uncertain observations do not authorize attachment.
   - **ARCH-ORDER — pass:** admission owns the lease through verification or joined cleanup; tests control completion ordering.
   - **ARCH-FUNERAL — pass:** locks have a fixed aggregate bound, policies have lifecycle retirement, and archives reuse retention ownership.

7. **Plan revision recommendations:** None required. Latest revisions identify the shared reader and its consumers; M2 remains unimplemented.

---

## Re-review — 2026-10-13T21:42:34-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 528 — couch: resume a slot whose conversation binding is unproven: infer it from transcripts and the tty log, or start fresh oriented from the tty log |
| repo | pair |
| issue file | workshop/issues/000528-couch-resume-a-slot-whose-conversation-binding-is-unproven-infer-it-from-transcripts-and-the-tty-log-or-start-fresh-oriented-from-the-tty-log.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | f59a1b6e21d449d9b6d5c5713b8d14083d3bf5a8..0d4565a9aff0d0983b7c4a567bb06af6aa4c5725 |
| command | sdlc milestone-close --issue 528 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T21:42:34-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M1 removes startup recovery effects and policy-based attach admission as the revised plan requires. However, Continue is now offered for records that the unchanged resume executor rejects. A review regression reproduces four such cases. The changed startup and menu behavior also lacks the required README update.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The approved replacement contract distinguishes positive listing from failed observation. TestListedLiveIsContinueAuthority and TestAttributedOwnerCannotResumeDisk pass; unknown observations do not authorize Continue.
  - id: BR-2
    disposition: addressed
    note: |
      Exact agent-generation comparison remains in terminal_agent.go. TestTerminalAgentRequiresExactServerAndLiveDescendant passes and fails when that comparison is removed.
  - id: BR-3
    disposition: addressed
    note: |
      Policy reads were removed from admission. Remaining readiness reads in session_handoff_os.go use contextual ReadRegularMetadata; bounded metadata regressions pass.
  - id: BR-4
    disposition: addressed
    note: |
      session_lock.go preserves the 256-stripe bound and policy publication is removed. TestTerminalLeasePathsHaveFixedBound passes and fails when full socket hashes replace stripe keys.
  - id: BR-5
    disposition: addressed
    note: |
      Admission-config reads were removed. Capture-witness reads still use ReadRegularMetadata in recovery_evidence_os.go:141; bounded metadata and wrong-witness regressions pass.
findings:
  - id: new
    severity: Critical
    family: recovery-primary-action-routing
    title: |
      Continue is offered where the resume executor still refuses attachment
    detail: |
      actionableinventory.go:494 now projects any positively listed session as Continue, but ResumeContextWith calls DecideResume, whose resume.go:186 preconditions still reject missing/incomplete profiles, unsupported saved agents, and unavailable working paths. A temporary review regression reproduced all four refusals. This is the 2nd finding in family recovery-primary-action-routing: state and enforce the rule that every offered Continue reaches a warm attach using listed-session authority; sweep all downstream warm-path gates rather than patching one profile check. ARCH-PURPOSE.
  - id: new
    severity: Important
    family: user-facing-contract-documentation
    title: |
      README update appears missing for changed Couch startup and action semantics
    detail: |
      startup.go:73 removes automatic launch/resume on ordinary Couch startup, and menu.go:1446 changes visible action labels. README.md:389 remains unchanged throughout the pinned range. Document observation-only startup, explicit action selection, and the M1 recovery limitations alongside the existing usage instructions; atlas/couch.md already describes these changes.
```

1. **Strengths**

   - Startup and constructor regressions verify that pending journals remain untouched.
   - Recovery-preview tests verify that inspection neither provisions a slot nor enrolls its store.
   - Attach tests require positive listing, reject failed-query output, and prevent replaying the interactive client.
   - Missing-endpoint errors preserve their underlying cause while explaining the legacy messaging limitation.

2. **Critical findings**

   The Continue mismatch is at [actionableinventory.go:494](/Users/xianxu/Library/Caches/sdlc/review-checkouts/ad36b0aca20bf2e2/cmd/internal/couchcore/actionableinventory.go:494), with the downstream refusal at [resume.go:186](/Users/xianxu/Library/Caches/sdlc/review-checkouts/ad36b0aca20bf2e2/cmd/internal/couchcore/resume.go:186). Separate warm attachment requirements from cold-launch requirements without fabricating a launch profile. Add tests through `ResumeContextWith` that verify an attach effect for all four cases.

3. **Important findings**

   Update [README.md:389](/Users/xianxu/Library/Caches/sdlc/review-checkouts/ad36b0aca20bf2e2/README.md:389) for the changed usage contract. No wording-presence test is needed.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Focused launcher, artifactpath, couchcore, couchmessage, and metadata tests passed. Temporary Go overlays reproduced the four Continue refusals and killed the BR-2/BR-4 mutations; temporary files were removed and the checkout remains clean.

   The broader run could not compile couchtty/couchcmd because generated `runtimebundle/assets/runtime/files` were absent. I interrupted the remaining broad run; no full-suite pass is claimed.

6. **Architectural notes**

   - **ARCH-DRY — pass:** continuity labels share the core projection.
   - **ARCH-PURE — pass:** classification and recovery decisions remain directly testable value functions.
   - **ARCH-PURPOSE — flag:** projection changed without completing its downstream Continue contract.
   - **ARCH-MOCK — pass for inspected seams:** injected listing/handoff and stateful endpoint tests exercise production boundaries. Live legacy-race conformance was not independently rerun.
   - **ARCH-CONSTRAINTS — pass for inspected changes:** listing retries are finite; retained metadata reads remain bounded.
   - **ARCH-SECURE — pass:** failed listing cannot authorize attach; exact-generation checks remain.
   - **ARCH-ORDER — pass:** startup no longer completes persisted operations implicitly; the approved Starting exception remains explicit.
   - **ARCH-FUNERAL — pass:** lock storage retains its fixed bound, and policy publication is removed.

7. **Plan revision recommendations**

   Add a `## Revisions` entry enumerating the downstream Continue gates and requiring executable attach coverage for missing/incomplete profiles, unsupported saved agents, and unavailable paths. Include the README update in M1 completion checks.

---

## Re-review — 2026-10-13T22:11:38-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 528 — couch: resume a slot whose conversation binding is unproven: infer it from transcripts and the tty log, or start fresh oriented from the tty log |
| repo | pair |
| issue file | workshop/issues/000528-couch-resume-a-slot-whose-conversation-binding-is-unproven-infer-it-from-transcripts-and-the-tty-log-or-start-fresh-oriented-from-the-tty-log.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | f59a1b6e21d449d9b6d5c5713b8d14083d3bf5a8..e7c7c6edcf5316d04133d95f2039e0c2ad09504e |
| command | sdlc milestone-close --issue 528 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T22:11:38-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The cold-metadata portion of BR-9 is fixed and regression-tested, and BR-10’s README correction is complete. However, Continue still reaches a lifecycle guard that refuses a positively listed session with an outstanding start claim. This contradicts the revised M1 contract and leaves BR-9 open.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Retained disposition. Failed listing does not authorize Continue; the revised contract explicitly supersedes policy-based terminal usability.
  - id: BR-2
    disposition: addressed
    note: |
      Retained disposition. Exact-generation helpers remain; policy/readiness-based warm admission was explicitly removed by the approved replacement design.
  - id: BR-3
    disposition: addressed
    note: |
      Retained disposition. Policy reads were removed; remaining admission readiness reads use bounded ReadRegularMetadata.
  - id: BR-4
    disposition: addressed
    note: |
      TerminalLease retains the aggregate bound of 256 permanent lock stripes in artifactpath/session_lock.go.
  - id: BR-5
    disposition: addressed
    note: |
      Retained disposition. Shared bounded metadata reading remains; the obsolete admission-config reader was removed.
  - id: BR-9
    disposition: not-addressed
    note: |
      ARCH-PURPOSE / ARCH-ORDER; family recovery-primary-action-routing remains open. Metadata regressions pass and fail with the old eligibility function restored, but resume.go:556 still calls clearLifecycleDebris, whose lifecycledebris.go:87 refuses an outstanding start owned by a live supervisor. A temporary regression constructed a valid StartClaim, positively listed its exact detached session, verified classification offers Continue, then observed ResumeContextWith(WarmOnly:true) fail with resume-starting before attachment. Separate listed-session attachment from lifecycle retirement; enumerate and test all accepted record shapes through the executor rather than fixing another individual gate.
  - id: BR-10
    disposition: addressed
    note: |
      README.md:395 now documents observation-only startup and explicit actions; the recovery section documents Starting, restricted Fresh execution, legacy attachment race and unavailable messaging. These passages agree with startup.go:73, the menu projection and atlas/couch.md. This is a prose correction and requires no wording-presence regression.
```

1. **Strengths**
   - Warm attachment no longer requires cold-launch profile/path eligibility. Tests also check that attachment preserves saved metadata and preferences.
   - Startup removes automatic start/park recovery and uses preview inventory reads.
   - Positive listing, failed listing and cancellation have focused launcher tests.
   - README and atlas now explain the revised M1 behavior and accepted compatibility limitations.

2. **Critical findings**
   - **BR-9 remains open:** `cmd/internal/couchcore/resume.go:556` routes warm attachment through lifecycle retirement. `cmd/internal/couchcore/lifecycledebris.go:87` then rejects the listed-live start-claim case. Preserve the claim’s ownership protections while providing an attachment path that does not require retiring it. Extend the executor regression across the record shapes already covered by `TestListedLiveOverridesStartMarkersWithoutChangingThem`.

3. **Important findings**
   - None newly raised.

4. **Minor findings**
   - None.

5. **Test coverage notes**
   - Selected core warm/cold/startup/recovery tests passed.
   - Selected launcher listing, warm-attach and Couch-session tests passed.
   - Restoring the old eligibility function through a temporary Go overlay made the metadata regression fail as expected.
   - The temporary listed-start-claim regression failed at HEAD, reproducing the remaining blocker.
   - Public-binary and Couch TUI checks could not build because generated `runtimebundle/assets/runtime/files` is absent. No full-suite success is claimed. Temporary overlays were removed; tracked files remain unchanged.

6. **Architectural notes**
   - **ARCH-DRY — pass:** shared classification, metadata reader and lease constructor remain central.
   - **ARCH-PURE — pass:** classification and continuity projection remain pure.
   - **ARCH-PURPOSE — flag:** BR-9’s offered-Continue-implies-attachment rule remains unenforced.
   - **ARCH-MOCK — pass, limited verification:** stateful handoff seams remain; public-binary verification was blocked.
   - **ARCH-CONSTRAINTS — pass:** inspected listing retries and admission waits are bounded.
   - **ARCH-SECURE — pass:** bounded readiness reads and exact selected-session checks remain.
   - **ARCH-ORDER — flag:** attachment still depends on retiring an independently owned start claim.
   - **ARCH-FUNERAL — pass:** policy persistence is removed; permanent lease files retain their aggregate bound.

7. **Plan revision recommendations**
   - Add a `## Revisions` entry recording the remaining BR-9 lifecycle-gate mismatch and an executor-level sweep of accepted record shapes. Keep the approved listed-live Continue contract; do not broaden the Starting exception to hide this refusal.

---

## Re-review — 2026-10-13T23:03:06-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 528 — couch: resume a slot whose conversation binding is unproven: infer it from transcripts and the tty log, or start fresh oriented from the tty log |
| repo | pair |
| issue file | workshop/issues/000528-couch-resume-a-slot-whose-conversation-binding-is-unproven-infer-it-from-transcripts-and-the-tty-log-or-start-fresh-oriented-from-the-tty-log.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | f59a1b6e21d449d9b6d5c5713b8d14083d3bf5a8..73f877e9121fb3eb5773cac06b2798c21e7f1c93 |
| command | sdlc milestone-close --issue 528 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T23:03:06-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: medium
```

BR-9’s correction is regression-backed, and I found no new code defect in the inspected paths. However, boundary verification remains incomplete: the environment denies `/bin/ps`, preventing the real-process launcher checks and public-binary resume test from completing. Under the requested unavailable-command rule, this needs a verification rerun in a process-observation-capable environment.

```findings
dispose:
  - id: BR-9
    disposition: addressed
    note: |
      Public Continue bypasses lifecycle adoption and cold prerequisites. The record-shape regression passes at HEAD and fails when the helper-only route is disabled, reproducing refusals and durable-record mutations.
  - id: BR-1
    disposition: addressed
    note: |
      Prior disposition retained; failed listing does not authorize Continue, and uncertain ownership prevents executable native resume.
  - id: BR-2
    disposition: addressed
    note: |
      Prior disposition retained; lifecycle operations retain exact process-identity checks. Listed-session attachment follows the explicitly revised authority contract.
  - id: BR-3
    disposition: addressed
    note: |
      Prior disposition retained; removed policy admission is not reintroduced into Continue.
  - id: BR-4
    disposition: addressed
    note: |
      TerminalLease retains the fixed 256-stripe namespace; its bound regression passes.
  - id: BR-5
    disposition: addressed
    note: |
      Prior disposition retained; bounded regular-metadata regressions pass.
  - id: BR-10
    disposition: addressed
    note: |
      README documents observation-only startup, listed-live Continue, untouched lifecycle records, recovery choices, and accepted compatibility limits; these agree with the inspected implementation.
```

1. **Strengths**

   - `couchcore/resume.go:444` routes public Continue before lifecycle holds, cleanup, and cold-launch prerequisites.
   - `couchcore/warm_attach_readonly_test.go:14` checks attachment across record shapes and live/dead/unknown owners, including unchanged revisions.
   - Cleanup tests distinguish locally owned handles from restored registry entries and preserve independent lifecycle owners.
   - README and atlas explain the revised behavior and the explicit M2/M3 deferrals.

2. **Critical findings**

   None established.

3. **Important findings**

   **Verification blocker:** `launcher/selected_root_test.go:59` cannot reach its public-binary resume assertions because the fixture’s initial creation fails at line 142 with `fork/exec /bin/ps: operation not permitted`. Retained-readiness launcher tests encounter the same restriction. Rerun these checks where process observation is permitted; this is not evidence that production code needs changing.

4. **Minor findings**

   None raised.

5. **Test coverage notes**

   - Focused core, launcher, console, command, metadata-read, and lock-bound regressions passed.
   - Disabling the helper-only route made the record-shape sweep fail; restoring it returned the focused suite to green.
   - Complete artifactpath and couchmessage package runs passed.
   - The broader run overlapped the deliberate mutation; affected command regressions were rerun successfully after restoration.
   - No clean full-suite pass is claimed. The disposable checkout was removed.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared projection, preview reads, and existing lifecycle transitions are reused.
   - **ARCH-PURE — pass:** classification remains separate from observation and execution.
   - **ARCH-PURPOSE — pass:** BR-9 now covers lifecycle shapes and downstream attachment gates.
   - **ARCH-MOCK — flag, verification only:** stateful seams are exercised; process-dependent checks remain blocked.
   - **ARCH-CONSTRAINTS — pass:** inspected retry/deadline paths preserve uncertainty on exhaustion.
   - **ARCH-SECURE — pass:** restored registry entries cannot authorize helper signalling.
   - **ARCH-ORDER — pass:** Continue preserves independent transactions; explicit lifecycle actions retain adoption checks.
   - **ARCH-FUNERAL — pass:** local handles are pruned on registry removal; coordination locks remain bounded.

7. **Plan revision recommendations**

   No design revision required. Record the blocked validation and its eventual rerun result; operator acceptance remains outstanding.

---

## Re-review — 2026-10-13T23:12:54-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 528 — couch: resume a slot whose conversation binding is unproven: infer it from transcripts and the tty log, or start fresh oriented from the tty log |
| repo | pair |
| issue file | workshop/issues/000528-couch-resume-a-slot-whose-conversation-binding-is-unproven-infer-it-from-transcripts-and-the-tty-log-or-start-fresh-oriented-from-the-tty-log.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | f59a1b6e21d449d9b6d5c5713b8d14083d3bf5a8..e690b4d30098c471422f0e0254563c9af78717f6 |
| command | sdlc milestone-close --issue 528 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T23:12:54-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The inspected implementation follows the revised M1 contract: startup observes, positive listing authorizes Continue, and attachment preserves independent lifecycle records. No new code defect was established. **REWORK is required because this review environment cannot complete required verification:** `/bin/ps` is denied, and missing generated runtime assets prevent console/public-binary tests from building. Rerun on the pinned head in a prepared, process-capable review environment.

1. **Strengths**
   - `couchcore/warm_attach.go` separates client attachment from durable lifecycle adoption; the record-shape and cleanup regressions passed.
   - Startup tests passed for pending journals, malformed records, and legacy sessions without launching recovery.
   - Listing tests verify failure, cancellation, exact-name matching, bounded retry, and no client replay.
   - README and atlas describe the revised behavior, accepted legacy resurrection race, and deferred M2/M3 work.

2. **Critical findings**
   - None newly established.

3. **Important findings**
   - No new code finding. Verification blockers:
     - `TestColdLaunchUnknownReadinessRefusesUnderLease` and `TestColdLaunchAcceptsProvenDeadRetainedAgent` fail because `/bin/ps` returns `operation not permitted`.
     - `TestSelectedRootNativeCreateListAndListedResume` and the selected couchtty suite cannot build: `cmd/internal/runtimebundle/embed.go:8` reports `assets/runtime/files: no matching files found`.
   - These failures do not establish production defects. Host-run results recorded in the tracker cannot substitute for this review’s failed inspection.

4. **Minor findings**
   - None newly raised.

5. **Test coverage notes**
   - Passed targeted couchcore regressions: **8.729s**.
   - Passed launcher listing/handoff tests, terminal-lease bounds, bounded metadata reads, and capture-witness adverse-input tests.
   - No complete-suite or console pass is claimed. No mutation testing was performed; existing disposed findings are not reopened merely because this environment blocks verification.
   - Repository remained unchanged.

6. **Architectural notes**
   - **ARCH-DRY — pass:** shared listing projection and existing lifecycle transitions are reused.
   - **ARCH-PURE — pass:** classification remains pure; attachment effects use injected boundaries.
   - **ARCH-PURPOSE — pass for inspected M1 scope:** the executor shape sweep covers the approved Continue contract; later recovery work remains explicitly deferred.
   - **ARCH-MOCK — flag, verification limitation:** controlled fake tests pass, but process-dependent and public-binary checks could not complete.
   - **ARCH-CONSTRAINTS — pass:** bounded listing retry and cancellation are exercised.
   - **ARCH-SECURE — pass within inspected scope:** failed observations grant no attachment authority; restored registry entries grant no helper ownership.
   - **ARCH-ORDER — pass:** tests preserve foreign lifecycle records and exercise listing loss and helper exit around acknowledgement.
   - **ARCH-FUNERAL — pass:** lock artifacts retain their 256-stripe bound; local helper ownership is pruned with registry removal.

7. **Plan revision recommendations**
   - No design revision requested. Record this review’s exact verification blockers and rerun requirements; do not mark M1 accepted from these partial results.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Retained disposition: failed listing does not authorize Continue; current uncertainty and listing-budget regressions passed.
  - id: BR-2
    disposition: addressed
    note: |
      Retained disposition under the superseding listed-live attach contract; process-dependent cold-path verification remains blocked by denied /bin/ps.
  - id: BR-3
    disposition: addressed
    note: |
      Policy/config admission readers were removed; retained readiness uses bounded regular-file reads. Shared reader regressions passed.
  - id: BR-4
    disposition: addressed
    note: |
      TerminalLease retains 256 permanent stripes; canonical-path and fixed-bound tests passed.
  - id: BR-5
    disposition: addressed
    note: |
      Capture-witness reads use osfs.ReadRegularMetadata; FIFO, oversize and cancellation regressions passed. The superseded admission-config reader is removed.
  - id: BR-9
    disposition: addressed
    note: |
      Public Continue enters helper-only attachment before lifecycle adoption. Targeted shape, cleanup, listing and registration regressions passed; console/public-binary verification remains environment-blocked.
  - id: BR-10
    disposition: addressed
    note: |
      README opening-Couch, attachment, recovery and retention passages match startup.go, warm_attach.go, listing admission and removal of Couch-started maintenance; atlas/couch.md documents the same boundaries.
```

---

## Re-review — 2026-10-14T15:20:37-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 528 — couch: resume a slot whose conversation binding is unproven: infer it from transcripts and the tty log, or start fresh oriented from the tty log |
| repo | pair |
| issue file | workshop/issues/000528-couch-resume-a-slot-whose-conversation-binding-is-unproven-infer-it-from-transcripts-and-the-tty-log-or-start-fresh-oriented-from-the-tty-log.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | e272745337452f9b3df2949e5c4a6faa97c85d59..95a2b9fe9b8418fad792941b0f09337bd8f7d13d |
| command | sdlc milestone-close --issue 528 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-14T15:20:37-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The revised M1 attachment and observation paths are substantially covered, and the focused core, UI, and launcher checks passed. One new startup integration test fails reproducibly, including in isolation, leaving the claimed startup acceptance behavior unverified. The checkout remains unchanged.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Positive-listing classification and failed-list tests preserve uncertainty; warm attachment requires fresh positive evidence.
  - id: BR-2
    disposition: addressed
    note: |
      Agent-generation authority is no longer an attachment prerequisite under the revised contract; exact process checks remain on retirement and dead-client repair.
  - id: BR-3
    disposition: addressed
    note: |
      Policy admission reads were removed; focused cold-launch, readiness and terminal-handoff checks passed.
  - id: BR-4
    disposition: addressed
    note: |
      Terminal leases retain the fixed 256-stripe namespace; session-lock tests passed.
  - id: BR-5
    disposition: addressed
    note: |
      Retained metadata consumers use bounded readers; the full osfs and artifactpath suites passed.
  - id: BR-9
    disposition: addressed
    note: |
      Public warm attachment bypasses lifecycle adoption; executor tests cover listed record shapes, independent claims and helper-only cleanup.
  - id: BR-10
    disposition: addressed
    note: |
      README.md:395 and README.md:804 document explicit hubs, background Continue attachment and failure retry, consistent with startup and menu wiring.
findings:
  - id: new
    severity: Important
    kind: defect
    family: reproducible-boundary-acceptance
    title: |
      New startup integration regression fails reproducibly
    detail: |
      cmd/internal/couchcmd/startup_warm_pass_test.go:179 times out awaiting two failed attachment attempts and their visible Continue failure reason. It failed in the focused multi-package run and again with go test ./cmd/internal/couchcmd -run '^TestStartupWarmPassPaintsAndAcceptsInputWhileMixedAttachmentsBlock$' -count=1 -v -timeout=65s (31.542s, exit 1). Separate the retry-count and rendered-reason assertions to expose which invariant fails, correct the runtime or fixture accordingly, and rerun through the later repair and explicit-retry assertions. ARCH-ORDER: the claimed startup sequence currently lacks passing end-to-end evidence.
```

1. **Strengths**

   - Warm attachment separates client ownership from lifecycle adoption in `couchcore/warm_attach.go`.
   - Dead-client repair rechecks exact old-process identity at the atomic write gate in `warm_attach_repair.go`.
   - Executor tests sweep independent start/park claims and verify unchanged durable state.
   - README and atlas updates accompany the changed startup and attachment semantics.

2. **Critical findings**

   None.

3. **Important findings**

   The reproducible startup test failure above blocks acceptance. Its combined assertion does not establish whether retry count or rendered failure text is responsible; no production root cause is claimed.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Focused couchcore, couchtty, launcher and messaging checks passed.
   - Full osfs, artifactpath, couchmessage, readiness and orientation suites passed.
   - Additional cold-launch, readiness, lease and handoff checks passed.
   - Full repository suite, mutation checks and live operator acceptance were not independently rerun.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared attachment and listing seams serve the inspected entry points.
   - **ARCH-PURE — pass:** classification and menu transitions remain separate from IO.
   - **ARCH-PURPOSE — pass:** later revisions explicitly transfer remaining recovery execution to #556/#557.
   - **ARCH-MOCK — pass:** stateful runners and endpoint fixtures exercise production seams.
   - **ARCH-CONSTRAINTS — pass:** bounded retries and existing worker capacity are retained.
   - **ARCH-SECURE — pass:** repair preserves exact process identity checks and guarded publication.
   - **ARCH-ORDER — flag:** startup sequence acceptance fails as reported above.
   - **ARCH-FUNERAL — pass:** local handles are pruned; lease artifacts remain bounded.

7. **Plan revision recommendations**

   Add a `## Revisions` entry recording the reproduced startup acceptance failure and its eventual cause and correction. Do not treat that checklist item as verified until the complete test passes.

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 4.408393792s
head: "95a2b9fe9b8418fad792941b0f09337bd8f7d13d"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-14T16:27:23-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 528 — couch: resume a slot whose conversation binding is unproven: infer it from transcripts and the tty log, or start fresh oriented from the tty log |
| repo | pair |
| issue file | workshop/issues/000528-couch-resume-a-slot-whose-conversation-binding-is-unproven-infer-it-from-transcripts-and-the-tty-log-or-start-fresh-oriented-from-the-tty-log.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | e272745337452f9b3df2949e5c4a6faa97c85d59..9a167f1fa0667715bd585b87dcaf7d65111c5487 |
| command | sdlc milestone-close --issue 528 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-14T16:27:23-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-11 is addressed: the corrected startup test passes repeatedly, and restoring the raw-write oracle makes its regression fail. One new blocker remains: automatic wrapper admission can replay pending store transactions, violating M1’s prohibition on startup journal recovery.

```findings
dispose:
  - id: BR-11
    disposition: addressed
    note: |
      The integration and delta-repaint regression passed ten repetitions (17.934s), including repair and explicit retry. Replacing VT screen projection with raw writes makes TestStartupWarmScreenReadsFailureAcrossDeltaRepaint fail; deadlines remain unchanged.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: observation-only-store-access
    title: |
      Automatic wrapper admission replays pending thread-store journals
    detail: |
      cmd/internal/couchcmd/message_service.go:1224 calls writable GetThread during automatic wrapper admission; GetThread reaches withStoreLock and journal recovery (threadstore.go:205). A temporary overlay regression demonstrated recordChanged=true, journalRemoved=true, resolveErr=<nil>, without any requested lifecycle action. ARCH-PURPOSE / ARCH-ORDER: enforce preview-only storage access across automatic admission and use-time authority reads, sweeping the thread resolver and record/agent callbacks at lines 329 and 338; add pending-journal regressions that assert unchanged records, journals and retention enrollment.
```

1. **Strengths**

   - Startup integration exercises the real dispatcher and Console with controlled helper barriers, then verifies repair and explicit retry.
   - Dead-client repair rechecks exact process identity at publication and preserves binding, history and claims.
   - Borrowed-client cleanup tests distinguish locally owned handles from restored registry entries.
   - README and atlas describe the revised attachment behavior and separate #556/#557 obligations.

2. **Critical findings**

   - **Automatic admission performs recovery writes.** [message_service.go:1224](/Users/xianxu/Library/Caches/sdlc/review-checkouts/ae1a1f129df4df3e/cmd/internal/couchcmd/message_service.go:1224) uses `GetThread`, whose writable path replays journals. Wrapper reconnection after Couch restart therefore becomes implicit transaction recovery. Introduce a shared preview accessor and use it throughout automatic message authority checks; test both global and slot-local pending journals.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Focused checks passed across couchcmd, couchcore, couchtty, launcher, couchmessage and artifactpath.
   - BR-11’s integration and oracle tests passed ten repetitions; the raw-oracle mutation failed as expected.
   - The temporary journal regression failed on pinned HEAD, confirming the new finding. Overlay files were removed; repository files remain unchanged.
   - Full-suite and live operator acceptance were not independently rerun.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared classification, scheduling and lifecycle owners are reused.
   - **ARCH-PURE — pass:** classification and menu/registry transitions have direct value tests.
   - **ARCH-PURPOSE — flag:** automatic message admission bypasses the observation-only startup contract.
   - **ARCH-MOCK — pass:** stateful doubles and subprocess endpoint tests exercise the relevant seams; live conformance was not rerun.
   - **ARCH-CONSTRAINTS — flag:** the same writable reads enter blocking store locks without propagating the admission context.
   - **ARCH-SECURE — flag:** an authentication observation can execute persisted transaction effects.
   - **ARCH-ORDER — flag:** wrapper arrival triggers journal recovery outside the approved lifecycle-action path.
   - **ARCH-FUNERAL — pass:** client handles have cleanup owners; terminal locks retain the fixed stripe bound.

7. **Plan revision recommendations**

   Append a `## Revisions` entry requiring automatic broker admission and use-time authority checks to use preview-only storage access. Enumerate the three `GetThread` callbacks and downstream routing reads, and require pending-journal preservation tests across global and slot-local stores.

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 4.110349709s
head: "9a167f1fa0667715bd585b87dcaf7d65111c5487"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-14T16:44:44-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 528 — couch: resume a slot whose conversation binding is unproven: infer it from transcripts and the tty log, or start fresh oriented from the tty log |
| repo | pair |
| issue file | workshop/issues/000528-couch-resume-a-slot-whose-conversation-binding-is-unproven-infer-it-from-transcripts-and-the-tty-log-or-start-fresh-oriented-from-the-tty-log.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | e272745337452f9b3df2949e5c4a6faa97c85d59..99035c559e3dd445a7ac568ebef91815a61079dd |
| command | sdlc milestone-close --issue 528 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-14T16:44:44-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-12’s journal-replay correction is verified, including a regression that fails when writable admission is restored. The revised M1 startup and attachment tests pass. One blocking defect remains: the new contextual preview accessor reaches blocking persisted-file reads, allowing malformed artifacts to defeat broker deadlines.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Remains disposed under the revised listing-based contract; uncertainty is retained in recovery admission.
  - id: BR-2
    disposition: addressed
    note: |
      Exact process-generation checks remain in readiness and native-owner authority.
  - id: BR-3
    disposition: addressed
    note: |
      Policy admission was removed; the previously corrected readiness reads remain bounded.
  - id: BR-4
    disposition: addressed
    note: |
      Permanent terminal coordination remains bounded to 256 stripes.
  - id: BR-5
    disposition: addressed
    note: |
      The corrected capture/config reader contracts remain intact; the new finding concerns newly contextualized ThreadStore authority reads.
  - id: BR-9
    disposition: addressed
    note: |
      Listed-live Continue uses the shared warm attachment route; focused attachment and reattach tests pass.
  - id: BR-10
    disposition: addressed
    note: |
      README.md documents startup attachment, retry, recovery actions and compatibility limits, consistent with the revised implementation.
  - id: BR-11
    disposition: addressed
    note: |
      The rendered-cell startup oracle and mixed startup integration pass in the focused couchcmd run.
  - id: BR-12
    disposition: addressed
    note: |
      All three broker callbacks use PreviewThreadContext. Global/local/routed journal and retention tests pass; restoring GetThread makes TestMessageThreadResolverLeavesPendingJournalUntouched fail.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: bounded-untrusted-artifact-reads
    title: |
      Contextual broker previews still block indefinitely on persisted-file reads
    detail: |
      PreviewThreadContext reaches os.ReadFile in threadstore_location.go:217 and global readPayload/readOptionalPayload in threadstore_layout.go:163/136. Context bounds lock acquisition but cannot interrupt these reads. Two controlled disposable-checkout regressions, covering routed legacy lookup and direct global record lookup, remained blocked beyond a 25ms deadline until a FIFO writer released them after 200ms; without that writer they cannot finish. This defeats automatic admission/use-time cancellation while holding the store lock (ARCH-CONSTRAINTS, ARCH-SECURE). This is the 3rd finding in family bounded-untrusted-artifact-reads: state and enforce one rule for every persisted byte read reachable from broker preview, enumerating manifest, routing record/archive and final record readers. Use bounded regular-file/no-follow/nonblocking reads with contextual cancellation, and add consumer regressions across global, local and routed backends.
```

1. **Strengths**

   - Shared preview decoding and routing avoid a parallel authority implementation (`threadstore_preview.go:31`).
   - Pending-journal tests assert unchanged directory contents across three backend shapes.
   - The broker regression proves its pending journal is valid and replayable.
   - README and atlas updates explain the revised startup and authority contracts.

2. **Critical findings**

   - The preview deadline defect above blocks shipping. Fix the complete reachable reader set, preserving writable lifecycle recovery behavior.

3. **Important findings**

   - None.

4. **Minor findings**

   - None.

5. **Test coverage notes**

   Focused tests passed across couchcore, couchcmd, couchtty, launcher and couchmessage. Targeted preview/resolver race tests passed. The journal mutation failed as required. Both supplemental FIFO regressions reproduced the blocking defect. Disposable artifacts were removed; tracked files remain unchanged. A full suite was not run.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared getter, decoder and attachment machinery.
   - **ARCH-PURE — pass:** classification remains pure; storage and process observation stay in integration owners.
   - **ARCH-PURPOSE — flag:** finish the bounded-reader family sweep rather than correcting another isolated reader.
   - **ARCH-MOCK — pass:** focused stateful attachment and broker integration coverage passes.
   - **ARCH-CONSTRAINTS — flag:** preview file reads escape caller deadlines.
   - **ARCH-SECURE — flag:** global persisted artifacts can be opened as blocking special files.
   - **ARCH-ORDER — pass:** preview refuses journal recovery; explicit lifecycle getters retain it.
   - **ARCH-FUNERAL — pass:** no new durable preview cache; coordination artifacts retain their aggregate bound.

7. **Plan revision recommendations**

   Add a `## Revisions` entry enumerating every persisted byte reader reachable through broker preview, its enforced size/type/cancellation contract, and consumer regression coverage. Narrow the current “context-bounded preview getter” claim until those reads satisfy it.

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.451851917s
head: "99035c559e3dd445a7ac568ebef91815a61079dd"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-14T17:01:35-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 528 — couch: resume a slot whose conversation binding is unproven: infer it from transcripts and the tty log, or start fresh oriented from the tty log |
| repo | pair |
| issue file | workshop/issues/000528-couch-resume-a-slot-whose-conversation-binding-is-unproven-infer-it-from-transcripts-and-the-tty-log-or-start-fresh-oriented-from-the-tty-log.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | e272745337452f9b3df2949e5c4a6faa97c85d59..a9a38f716bac4c63d44d2ba2538d28f549777dfc |
| command | sdlc milestone-close --issue 528 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-14T17:01:35-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-13 is addressed: the preview readers are bounded, and regressions fail when the old global reader is restored. A separate automatic attachment path still performs an unbounded read during dead-client repair. The repository’s production-symbol guard also fails on four orphaned helpers.

```findings
dispose:
  - id: BR-13
    disposition: addressed
    note: |
      Traced manifest, routing record/archive, local membership/salvage/archive, final record, aliases and inspection readers. Preview/metadata regressions pass. Restoring global os.ReadFile in a disposable checkout makes global-record, global-manifest and routed-legacy regressions fail at approximately 202ms.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: bounded-untrusted-artifact-reads
    title: |
      Automatic warm attachment can block indefinitely in dead-client repair
    detail: |
      warm_attach_repair.go:51 drops the caller context during routing; lines 56–59 then select a writable target and call readPayload, reaching global os.ReadFile at threadstore_layout.go:172. A controlled warm-only attachment replaced the record with a FIFO after acknowledgement: despite a 50ms deadline, four runs returned only after a writer released them at 207–227ms; without that writer the read remains blocked while holding store.lock. This is the 4th finding in family bounded-untrusted-artifact-reads. Earlier rounds fixed instances. State and enforce the bounded persisted-read rule across every automatic M1 path, including attachment repair, rather than fixing this site alone. Preserve descriptor-relative local admission, propagate context through routing and payload reads, and add cancellation/FIFO regressions through production attachment. ARCH-CONSTRAINTS, ARCH-SECURE, ARCH-ORDER, ARCH-PURPOSE.
  - id: new
    severity: Important
    kind: defect
    family: reproducible-boundary-acceptance
    title: |
      Reader refactor leaves four orphaned helpers and fails the required production-symbol guard
    detail: |
      go test ./cmd/internal/artifactpath -run '^TestNoProductionSymbolIsReferencedOnlyByTests$' -count=1 fails reproducibly for readOptionalFile (storejournal.go:115), readThreadLocked (threadstore.go:1148), localMembership (threadstore_layout.go:39), and readLocalPayload (threadstore_layout.go:177). Production-reference inspection confirms these are unused. This is the 2nd finding in family reproducible-boundary-acceptance. Enumerate obsolete adapters left by the refactor, remove or explicitly justify each, and run the repository acceptance guards before re-closing; do not repair only one reported declaration. ARCH-DRY.
```

1. **Strengths**

   - Preview reads reuse the shared bounded reader while retaining descriptor-relative local-file access.
   - FIFO regressions exercise nine consumer branches; mutation testing confirms they detect the original blocking behavior.
   - Stateful startup tests exercise blocked parallel attachments, responsive rendering, visible failures and explicit retry.
   - README and atlas updates cover changed startup, Continue semantics and the accepted legacy attachment race. Remaining recovery execution is explicitly split into successor issues.

2. **Critical findings**

   - [warm_attach_repair.go:51](/Users/xianxu/Library/Caches/sdlc/review-checkouts/7f3692135a24920c/cmd/internal/couchcore/warm_attach_repair.go:51): automatic repair bypasses bounded contextual reads. Apply the shared read contract throughout automatic attachment, including routing and write-gate rereads.

3. **Important findings**

   - [threadstore_layout.go:39](/Users/xianxu/Library/Caches/sdlc/review-checkouts/7f3692135a24920c/cmd/internal/couchcore/threadstore_layout.go:39): remove or justify all four orphaned adapters identified above; the existing repository guard must pass.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed: metadata/preview/inspection tests, focused core startup/warm-attachment/classification tests, and complete `couchcmd`, `couchtty`, `launcher` and `couchmessage` packages.

   Failed: the artifact-path production-symbol guard and the controlled attachment-repair FIFO regression. The latter reproduced four times. Full repository testing was not run. The disposable reproduction checkout was removed; tracked files remain unchanged.

6. **Architectural notes**

   - **ARCH-DRY — flag:** orphaned compatibility adapters fail the repository guard.
   - **ARCH-PURE — pass:** continuity decisions and menu transitions remain separated from IO.
   - **ARCH-PURPOSE — flag:** the bounded-read sweep omits automatic attachment repair.
   - **ARCH-MOCK — pass:** touched external interactions use shared seams and stateful fixtures.
   - **ARCH-CONSTRAINTS — flag:** optional repair can exceed cancellation indefinitely.
   - **ARCH-SECURE — flag:** repair trusts persisted bytes through an unrestricted reader.
   - **ARCH-ORDER — flag:** cancellation cannot terminate the repair read after helper registration.
   - **ARCH-FUNERAL — pass:** coordination files retain the fixed 256-inode bound; attachment notices remain ephemeral.

7. **Plan revision recommendations**

   Add a `## Revisions` entry extending the bounded persisted-read invariant from broker previews to **all automatic M1 execution**, explicitly enumerating attachment preflight, routing and dead-client repair. Require consumer-level cancellation regressions and the production-symbol guard before re-close.

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 4.223236625s
head: "a9a38f716bac4c63d44d2ba2538d28f549777dfc"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-14T17:16:09-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 528 — couch: resume a slot whose conversation binding is unproven: infer it from transcripts and the tty log, or start fresh oriented from the tty log |
| repo | pair |
| issue file | workshop/issues/000528-couch-resume-a-slot-whose-conversation-binding-is-unproven-infer-it-from-transcripts-and-the-tty-log-or-start-fresh-oriented-from-the-tty-log.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | e272745337452f9b3df2949e5c4a6faa97c85d59..137a5bfe6ed0b45694c48488a17ba7c6de77f8a1 |
| command | sdlc milestone-close --issue 528 --milestone M1 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-14T17:16:09-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The revised M1 startup, attachment separation, and preview-only store access are well covered, and all reviewed package suites pass. BR-15 is addressed. BR-14’s repair-site correction is verified, but its required family-wide rule remains incomplete: downstream automatic attachment still performs unbounded reads before reaching the terminal handoff.

```findings
dispose:
  - id: BR-14
    disposition: not-addressed
    note: |
      The repair-site fix is verified: the new production attachment FIFO regression passes at head and fails with the previous repair implementation, taking 202 ms. However, the required rule across ALL automatic M1 reads remains false. createflow.go:284 calls OSRuntime.InferAgent before warm attachment; osruntime.go:629-647 reads the ledger and agent sidecar through unbounded ReadFile. Reviewer-only overlay regressions exercising production runOnce with stateful session facts and real OSRuntime.InferAgent delayed attachment until FIFO writers released those two paths after 201.947 ms and 200.358 ms. Without a writer these reads remain blocked. Keep this fourth finding in family bounded-untrusted-artifact-reads open; enumerate and fix the downstream launcher read graph rather than only these instances. ARCH-CONSTRAINTS, ARCH-SECURE, ARCH-PURPOSE.
  - id: BR-15
    disposition: addressed
    note: |
      All four reported orphan adapters are removed. The complete artifactpath package passes, including TestNoProductionSymbolIsReferencedOnlyByTests.
  - id: BR-1
    disposition: addressed
    note: |
      Retain prior disposition; failed or canceled listing does not authorize Continue or attachment.
  - id: BR-2
    disposition: addressed
    note: |
      Retain prior disposition; no new process-generation authority defect identified.
  - id: BR-3
    disposition: addressed
    note: |
      Retain prior disposition; policy admission is removed and retained readiness readers use bounded metadata IO.
  - id: BR-4
    disposition: addressed
    note: |
      TerminalLease still maps canonical sockets onto exactly 256 permanent lock stripes.
  - id: BR-5
    disposition: addressed
    note: |
      Retain prior disposition for capture/config consumers; shared metadata tests pass. The remaining automatic launcher reads are maintained under BR-14.
  - id: BR-9
    disposition: addressed
    note: |
      Listed attachment bypasses cold profile, path, and independent lifecycle prerequisites; attachment-shape and cleanup regressions pass.
  - id: BR-10
    disposition: addressed
    note: |
      README.md:486 documents positive-listing attachment, the legacy resurrection race, explicit recovery selection, and old-wrapper messaging limitations; atlas/couch.md describes the corresponding implementation.
  - id: BR-11
    disposition: addressed
    note: |
      The VT-based startup regression and complete couchcmd package pass, including the mixed attachment startup flow.
  - id: BR-12
    disposition: addressed
    note: |
      Broker thread, record, and agent callbacks use PreviewThreadContext; pending-journal preservation regressions pass.
  - id: BR-13
    disposition: addressed
    note: |
      Preview routing uses bounded contextual readers, including descriptor-relative local reads; FIFO, unsafe-byte, cancellation, and journal-preservation regressions pass.
```

1. **Strengths**

   - Repair preserves read-only routing while limiting writes to guarded helper replacement: `cmd/internal/couchcore/warm_attach_repair.go:49`.
   - The new repair regression detects the original bug: reverting the implementation through a temporary Go overlay makes it fail.
   - Attachment tests cover independent lifecycle claims, changed listings, cancellation, retry, and helper-only cleanup.
   - README and atlas explain compatibility limits and the separate #556/#557 execution boundaries.

2. **Critical findings**

   **BR-14 remains open — optional metadata can block automatic attachment.**  
   `cmd/internal/launcher/createflow.go:284` calls `InferAgent` before `runAttach`. Its ledger and sidecar reads at `cmd/internal/launcher/osruntime.go:629` and `:634` reach plain `os.ReadFile`, without file-type, size, or cancellation bounds.

   The reviewer probe used production `runOnce`, the existing stateful session fake, and real `OSRuntime.InferAgent`. Both FIFO cases waited for the delayed writer before invoking attachment.

   Apply the persisted-read rule across the parent **and launched helper**. Optional title metadata should degrade promptly when unavailable. Include launcher startup index reads and transitive ledger/native lookups in the enumeration, with production-path regressions.

3. **Important findings**

   None beyond the maintained blocker.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed complete suites for couchcore (334.436 s), couchcmd (61.139 s), launcher, couchtty, artifactpath, couchmessage, readiness, and osfs. Focused automatic-path tests also passed under `-race`.

   The repair mutation failed as expected; both reviewer launcher FIFO probes exposed the remaining defect. No live-slot operations were performed. The pinned checkout remains unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** obsolete adapters removed; metadata reading is shared.
   - **ARCH-PURE — pass:** projection and menu decisions remain separate from IO.
   - **ARCH-PURPOSE — flag:** BR-14’s full consumer enumeration remains incomplete.
   - **ARCH-MOCK — pass:** production seams have stateful doubles and existing conformance hooks; live conformance was not rerun.
   - **ARCH-CONSTRAINTS — flag:** optional launcher reads lack enforced bounds.
   - **ARCH-SECURE — flag:** those persisted inputs still permit blocking special files.
   - **ARCH-ORDER — pass for reviewed transitions:** repair rechecks identities and cancellation before publication; retry/cleanup sequence tests pass.
   - **ARCH-FUNERAL — pass:** lock artifacts remain bounded; borrowed handles have cleanup owners.

7. **Plan revision recommendations**

   Add a `## Revisions` entry expanding the automatic-read graph through the launched Pair helper, including startup indices, title metadata, ledger parsing, and transitive native lookups. Record the enforced read contract and consumer regression for each branch before claiming the family sweep complete.

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.495077625s
head: "137a5bfe6ed0b45694c48488a17ba7c6de77f8a1"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
