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
