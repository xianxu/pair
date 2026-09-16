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
