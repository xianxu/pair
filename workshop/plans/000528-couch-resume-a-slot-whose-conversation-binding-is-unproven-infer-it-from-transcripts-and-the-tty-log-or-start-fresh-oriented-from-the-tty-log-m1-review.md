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
