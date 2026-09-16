# Boundary Review — 000255-lifecycle-state-ownership#255 (milestone M4)

| field | value |
|-------|-------|
| issue | 255 — Establish a faithful terminal abstraction for Couch and Pair |
| repo | 000255-lifecycle-state-ownership |
| issue file | workshop/issues/000255-lifecycle-state-ownership.md |
| boundary | milestone M4 |
| milestone | M4 |
| window | 12c301ac47491e5ab9fe5128f845961c6b3ab55f..d36fef2118df5719b1a96ec9ecce90072494863f |
| command | sdlc milestone-close --issue 255 --milestone M4 |
| reviewer | codex |
| timestamp | 2026-09-15T16:59:48-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The pinned range is inspectable, and the affected package suites, targeted native history checks, and repeated shutdown race tests pass. Notification serialization, history rendering, and shutdown handling have meaningful coverage. Two Important gaps block this boundary: crash-residue cleanup depends on a binding that other cleanup paths delete, and broker tests contend with the live user’s notification transport.

1. **Strengths**

   - Notifications now share the wrapper’s output owner; tests cover split controls/UTF-8, partial writes, bounded admission, and incomplete EOF.
   - History eviction distinguishes contiguous append from missing coverage, with independent xterm and native Zellij assertions.
   - Shutdown tests force blocked/partial paints and distinguish expected cancellation from genuine host failure.
   - README and atlas describe the new route. Qualification records preserve failed runs and disclose the switch-latency miss. The recorded production manifest matches all 424 files inspected.

2. **Critical findings**

   None.

3. **Important findings**

   - **Crash sockets lose their cleanup route — ARCH-FUNERAL / ARCH-PURPOSE.** At `cmd/internal/notifytransport/address.go:170`, reclamation requires an existing PID binding; a missing binding returns immediately. However, `cmd/internal/launcher/lifecycle.go:349` removes that binding independently, and artifact GC includes it. After abnormal wrapper termination followed by sidecar cleanup, the socket remains under `/tmp/pair-notify-<uid>` with no implemented sweep or growth bound.

     **This is the 3rd finding in family `artifact-lifetime-ownership`.** Fix the ownership rule across normal close, crash, sidecar deletion, GC, and failed cleanup—not just this instance. Add safe dead-owner reclamation independent of binding survival, with tests preserving live and foreign sockets.

   - **Broker tests share production transport state — ARCH-SECURE / ARCH-MOCK.** `cmd/internal/notifytransport/transport_test.go:361` acquires the actual UID-wide lock and deliberately holds it until startup times out. `address.go:54` hardcodes the same root used by live wrappers. Temporary PID bindings do not isolate that lock. Concurrent production startup or `Broker.Close` can consequently time out; Close then leaves its artifacts.

     Inject a transport namespace/root shared by address construction and locking. Run every broker fixture inside private temporary storage, including contention tests.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Passed all seven affected package suites.
   - Passed targeted independent/native history tests.
   - Passed shutdown classification tests under `-race -count=10`.
   - Pinned-range whitespace checks passed; repository status remained unchanged.
   - Missing regressions: dead socket after binding removal, and isolation between test and production namespaces.
   - Reviewed preserved sustained/native logs; did not rerun thirty-minute soaks or operator smoke.

6. **Architectural notes**

   - **ARCH-DRY — Pass:** shared codec, framing observer, and history serializer.
   - **ARCH-PURE — Pass:** framing/mapping decisions remain separable from socket and terminal IO.
   - **ARCH-PURPOSE — Flag:** artifact ownership needs the complete cleanup-path sweep above.
   - **ARCH-MOCK — Flag:** real-socket conformance lacks an isolated storage namespace.
   - **ARCH-CONSTRAINTS — Pass:** explicit message/queue/deadline bounds; performance misses remain disclosed.
   - **ARCH-SECURE — Flag:** tests touch the live transport namespace.
   - **ARCH-ORDER — Pass:** ordered rewrite events, accepted-prefix accounting, and forced shutdown interleavings provide concrete enforcement.
   - **ARCH-FUNERAL — Flag:** binding deletion can strand socket residue.

7. **Plan revision recommendations**

   Add `## Revisions` entries defining independent socket reclamation and isolated transport namespaces, including the regression cases above. Add explicit PURE/INTEGRATION kinds to the notification revision’s concept tables. Keep operator acceptance, issue closure, and merge pending as currently documented.

```findings
findings:
  - id: new
    severity: Important
    family: artifact-lifetime-ownership
    title: |
      Dead notification sockets become uncollectable when their PID binding is removed
    detail: |
      cmd/internal/notifytransport/address.go:170 requires the PID binding to reclaim a socket, but cmd/internal/launcher/lifecycle.go:349 and artifact GC remove that binding independently. Abnormal wrapper termination followed by binding cleanup strands socket files with no implemented sweep or growth bound (ARCH-FUNERAL, ARCH-PURPOSE). This is the 3rd finding in family artifact-lifetime-ownership: state and enforce the ownership rule across normal close, crash, sidecar deletion, GC, and failed cleanup. Add dead-owner reclamation independent of binding survival and regression tests preserving live/foreign sockets.
  - id: new
    severity: Important
    family: test-state-isolation
    title: |
      Broker contention tests acquire the live user's production notification lock
    detail: |
      cmd/internal/notifytransport/transport_test.go:361 deliberately holds lockDirectory through a startup timeout, while address.go:54 hardcodes the production UID-wide namespace. Concurrent live wrapper startup or Close can time out; transport.go:135 then abandons artifact cleanup (ARCH-SECURE, ARCH-MOCK). Inject a transport namespace covering both socket addresses and locks, and move all broker fixtures into private temporary storage with cross-namespace isolation coverage.
```

---

## Re-review — 2026-09-15T17:51:20-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 255 — Establish a faithful terminal abstraction for Couch and Pair |
| repo | 000255-lifecycle-state-ownership |
| issue file | workshop/issues/000255-lifecycle-state-ownership.md |
| boundary | milestone M4 |
| milestone | M4 |
| window | 12c301ac47491e5ab9fe5128f845961c6b3ab55f..f5449fe5e1ba9dd9037e7ac76d2ea62fb91c81bf |
| command | sdlc milestone-close --issue 255 --milestone M4 |
| reviewer | codex |
| timestamp | 2026-09-15T17:51:20-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-19 and BR-20 are addressed, with regression tests confirmed to fail when their fixes are disabled. The notification and terminal changes have substantial passing coverage. One Important artifact-retention finding blocks this boundary: native conformance creates persistent evidence directories without cleanup or a retention bound. Operator smoke acceptance remains appropriately pending.

```findings
dispose:
  - id: BR-19
    disposition: addressed
    note: |
      address.go:244 reclaims dead-owner sockets independently of PID bindings. TestSweepReclaimsCrashSocketWithoutPIDBinding passes; disabling the sweep makes both crash and failed-close cases fail. Live/foreign preservation and capacity tests also pass.
  - id: BR-20
    disposition: addressed
    note: |
      Socket addresses, locks and sweeps share the injected namespace; Broker.Close retains its admitted root. Cross-namespace coverage passes. Removing namespace injection makes the regression fail before accessing the production lock.
findings:
  - id: new
    severity: Important
    family: artifact-lifetime-ownership
    title: |
      Native conformance evidence accumulates without cleanup or retention bounds
    detail: |
      cmd/internal/couchtty/terminal_native_test.go:244 creates a retained directory for each successful direct/wrapped run; line 127 creates another family on failure. Neither has cleanup, a sweep, or a count/age bound. This is the 4th finding in family artifact-lifetime-ownership (ARCH-FUNERAL, ARCH-PURPOSE). Do NOT fix only one allocation: state and enforce the ownership rule across both evidence families and enumerate sibling writers. Prefer invocation-scoped storage and bounded reported diagnostics, or explicit bounded retention, with success/failure cleanup regressions.
```

1. **Strengths**

   - Dead-socket reclamation checks filename grammar, ownership, file type and owner liveness, independently of binding survival.
   - Namespace isolation covers publication, routing, contention and teardown, including environment changes after admission.
   - Wrapper tests exercise byte boundaries, malformed streams and partial writes; Console tests force shutdown/write ordering.
   - README and atlas describe the changed notification route and qualification limits. The candidate hash matches its manifest; all 424 recorded production source hashes match the checkout.

2. **Critical findings**

   None.

3. **Important findings**

   The evidence directories at `terminal_native_test.go:127` and `:244` outlive their tests without an implemented end. Failure residue was observed during this review; successful retention is unconditional in the code. Apply one retention rule to both families and test cleanup after success and failure.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed: seven affected package suites, focused race suites for transport/terminal/Couch/Pair term, and the local VT fork suite. Both prior-finding mutation checks went red as expected.

   Fresh native conformance was blocked by sandbox denial of `/dev/tty` in both direct and wrapped fixtures. The broad repository run was stopped incomplete; neither is claimed as passing. Earlier sustained-run logs were inspected, not rerun.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared framing and notification mapping.
   - **ARCH-PURE — pass:** pure transformations remain separate from transport; revised concept classifications match their responsibilities.
   - **ARCH-PURPOSE — flag:** artifact ownership still omits two evidence families.
   - **ARCH-MOCK — pass:** stateful CLI fake, controlled IO seams and native conformance coverage.
   - **ARCH-CONSTRAINTS — pass:** bounded admission and queues; measured latency exceptions are disclosed.
   - **ARCH-SECURE — pass:** private namespaces and validated binding/socket inputs.
   - **ARCH-ORDER — pass:** ordered output receipts, partial-write handling and forced shutdown sequences.
   - **ARCH-FUNERAL — flag:** retained test evidence has no implemented removal policy.

7. **Plan revision recommendations**

   Add a `## Revisions` entry enumerating both native evidence families, their owner, retention limit and removal mechanism. Reconcile this with the existing discovery-artifact rule at plan line 416, which requires invocation cleanup and disallows implicit diagnostic retention.

---

## Re-review — 2026-09-15T18:07:40-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 255 — Establish a faithful terminal abstraction for Couch and Pair |
| repo | 000255-lifecycle-state-ownership |
| issue file | workshop/issues/000255-lifecycle-state-ownership.md |
| boundary | milestone M4 |
| milestone | M4 |
| window | 12c301ac47491e5ab9fe5128f845961c6b3ab55f..ea98f0c714f504232e202b7435397b77bf7d25ad |
| command | sdlc milestone-close --issue 255 --milestone M4 |
| reviewer | codex |
| timestamp | 2026-09-15T18:07:40-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

BR-21 is addressed: both native evidence families now have invocation-scoped ownership, and cleanup/bounds regressions fail when the fixes are disabled. No blocking code findings remain. One minor documentation inconsistency needs cleanup. This verdict covers M4’s automated boundary; operator smoke acceptance, issue closure and merge remain pending.

1. **Strengths**

   - Evidence cleanup preserves scratch through owner teardown, then removes it on success and failure.
   - Notification tests exercise split UTF-8/control sequences, bounded queues, partial writes and incomplete EOF.
   - Shutdown tests force cancellation during zero/partial paints and preserve genuine host failures.
   - All 424 recorded production source hashes and three candidate binary hashes match. Qualification distinguishes measured limitations from operator acceptance.

2. **Critical findings**

   None.

3. **Important findings**

   None.

4. **Minor findings**

   **Stale notification-route descriptions — ARCH-PURPOSE.** [atlas/architecture.md:748](atlas/architecture.md:748) still says hooks deliver through `PAIR_OUTER_TTY_PATH`; lines 743–744 describe obsolete diagnostics. [wrap.go:10](cmd/internal/wrapcmd/wrap.go:10) and [pair-notify:13](bin/pair-notify:13) repeat the retired route. Production now uses the broker and serialized pane output.

   **This is the 2nd finding in family `documentation-surface-accuracy`.** Apply one rule across the enumerated passages: notification documentation must describe the current broker route; outer-TTY metadata is compatibility-only. Sweep the class rather than correcting only the atlas paragraph.

5. **Test coverage notes**

   - Passed all seven affected package suites and the local VT fork suite.
   - Passed focused race checks for evidence lifetime, socket reclamation/isolation, shutdown and consumer effect policies.
   - Temporary overlay mutations made both success/failure cleanup tests fail; removing the diagnostic cap failed the size regression.
   - Pinned-range whitespace checks passed. Checkout changes were preserved.
   - Inspected recorded native and sustained-run evidence; did not rerun native interactive conformance or thirty-minute soaks.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared framing, codec and history rendering.
   - **ARCH-PURE — pass:** pure identity/mapping/framing responsibilities match the revised concept classifications.
   - **ARCH-PURPOSE — flag, minor:** functional consumers use the new route; documentation still contains the retired model.
   - **ARCH-MOCK — pass:** stateful doubles, controlled IO seams and isolated native fixtures.
   - **ARCH-CONSTRAINTS — pass:** explicit queue, message, deadline and storage bounds; switch-latency misses remain disclosed.
   - **ARCH-SECURE — pass:** private namespaces and validated binding/socket inputs.
   - **ARCH-ORDER — pass:** ordered output receipts, accepted-prefix accounting and forced shutdown sequences.
   - **ARCH-FUNERAL — pass:** invocation-owned evidence and binding-independent dead-socket reclamation.

7. **Plan revision recommendations**

   Add a short `## Revisions` entry recording the documentation-route sweep above. The BR-21 ownership revision matches the implementation.

```findings
dispose:
  - id: BR-21
    disposition: addressed
    note: |
      terminal_native_test.go:43 allocates invocation-owned reattachment scratch; its failure path no longer creates retained files. NativeEvidenceLifetime passes for success/failure, and both cases fail with cleanup disabled in a temporary overlay. DiagnosticBound also fails when its cap is removed. Sibling native/PTY/discovery/performance writers have invocation cleanup.
  - id: BR-19
    disposition: addressed
    note: |
      Dead-owner reclamation remains independent of PID binding survival. Crash/failed-close reclamation, live/foreign preservation and namespace capacity regressions pass.
  - id: BR-20
    disposition: addressed
    note: |
      Socket addressing, locking and sweeping use private injected namespaces; Close retains its admitted root. Cross-namespace contention/routing coverage passes under the race detector.
findings:
  - id: new
    severity: Minor
    family: documentation-surface-accuracy
    title: |
      Notification documentation still contains the retired outer-TTY route
    detail: |
      atlas/architecture.md:743-748, cmd/internal/wrapcmd/wrap.go:10-12 and bin/pair-notify:13 contradict the implemented broker/serialized-output route (ARCH-PURPOSE). This is the 2nd finding in family documentation-surface-accuracy. State the current-route documentation rule and sweep these sibling passages together, preserving outer-TTY references only where they describe compatibility metadata.
```
