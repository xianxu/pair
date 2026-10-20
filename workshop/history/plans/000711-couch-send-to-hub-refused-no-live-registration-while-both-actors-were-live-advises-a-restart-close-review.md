# Boundary Review — pair#711 (whole-issue close)

| field | value |
|-------|-------|
| issue | 711 — couch --send-to hub refused 'no live registration' while both actors were live; advises a restart |
| repo | pair |
| issue file | workshop/issues/000711-couch-send-to-hub-refused-no-live-registration-while-both-actors-were-live-advises-a-restart.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7d4e0c3aba6b39fb12d5125ee967dcde9e5be4be..734d84a932b3c64e1043cf625a2038e71bb294a4 |
| command | sdlc close --issue 711 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T06:13:23-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The single-listing approach removes the observed race, and all three affected packages pass. However, independent regression probes expose stale sender authority, incorrect runtime session-state classification, and incomplete refusal diagnostics. These require fixes before close.

Done-when: current — it covers the revised listing, input-queue, lifetime-fencing, diagnostic, and query-budget contracts.

```findings
findings:
  - id: new
    severity: Critical
    family: admission-registration-lifetime-fencing
    title: |
      Same-binding sender reconnect can admit a stale reservation
    detail: |
      At cmd/internal/couchmessage/broker.go:726, b.caller(from) checks current connectivity without comparing the sender registration with the one that began admission. A disposable regression disconnected and re-registered the sender after successful Reserve, then invoked recipient OperatorInput; Send returned a queued receipt with no error. This violates the Spec's unchanged-both-registrations condition. Enumerated admission sites requiring the same sender lifetime token are send's initial validation (574), admit's loop validation (631), tryAdmission's post-IO validation (726), and the busy-to-queue validation (684). Capture the sender actor/generation and preserve it across these paths. Add sender-reconnect controls alongside the existing recipient-reconnect control. ARCH-PURPOSE: sweep both registration lifetimes.
  - id: new
    severity: Critical
    family: validated-inventory-state-consumers
    title: |
      Runtime listing discards validated exited states and reclassifies names incorrectly
    detail: |
      At cmd/internal/launcher/osruntime.go:334-353, ListSessions extracts names from sessionListingStates but discards its state values. Its downstream buildListRowsForScope at list.go:78 calls zellijsession.RowState, which uses Fields[0] and Contains("EXITED"). Disposable probes showed a live pair-EXITED-is-a-name displayed as exited, while exited names containing spaces or tabs displayed as detached and triggered client queries. The remaining production instance in this window is ListSessions -> buildListRowsForScope -> RowState; ZellijSource.snapshot already consumes validated states correctly. Pass the validated state map into the runtime row builder and add these three edge cases. ARCH-DRY and ARCH-PURPOSE: all promised inventory consumers must use the same identity/state interpretation.
  - id: new
    severity: Important
    family: availability-refusal-phase-context
    title: |
      Admission races still return bare, misleading unavailable errors
    detail: |
      The send path retains bare ErrUnavailable returns at cmd/internal/couchmessage/broker.go:576, :633, and :685. These are the complete unlabelled registration-refusal instances in the admission path: initial caller validation, caller validation after routing IO, and combined caller/recipient validation after reservation release. A disposable regression disconnected the caller inside the injected family source; Send returned "recipient is not live", losing both the failing side and phase. Wrap each caller failure with caller/current context, split the combined fallback guard by side, preserve errors.Is classification, and retain regression coverage. ARCH-PURPOSE: diagnostics must cover concurrent invalidation as well as initially missing registrations.
```

1. **Strengths**

   - Direct and cached listing readers use one full native observation; deterministic client-query ordering is preserved.
   - Malformed refreshes retain the last good timestamp.
   - The permanent input regression checks queue promotion and exactly one delivery.
   - README and atlas updates explain phase diagnostics and remove blanket restart advice.

2. **Critical findings**

   Two, detailed above: sender registration fencing and runtime state interpretation.

3. **Important findings**

   One, detailed above: incomplete phase context during admission races.

4. **Minor findings**

   None.

5. **Test coverage notes**

   `go test ./cmd/internal/couchmessage ./cmd/internal/launcher ./cmd/internal/couchcmd` passed. Disposable Go-overlay probes failed for all three findings; temporary files were removed and tracked files remained unchanged. Existing tests miss sender reconnect, runtime opaque-name state cases, and caller loss during routing. The recorded full-suite prerequisite remains outstanding in the issue.

6. **Architectural notes**

   - **ARCH-DRY: flag** — runtime retains a competing state interpretation.
   - **ARCH-PURE: pass** — parsing remains pure and IO uses existing seams.
   - **ARCH-PURPOSE: flag** — lifetime and diagnostic contracts are incompletely delivered.
   - **ARCH-EVOLVE: pass** — no persisted format changes.

7. **Plan revision recommendations**

   Add a `## Revisions` entry covering both sender and recipient registration fencing, runtime consumption of validated state values, and the complete admission-refusal phase sweep, with permanent regressions for the cases above.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"2da21866-4cee-4387-a74d-117dea7c0e10","repository":"github.com/xianxu/pair","issue_id":"000711","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/b52028d24f0f238a","reviewed_head":"734d84a932b3c64e1043cf625a2038e71bb294a4","slot":"pair:7","requested_at":"2026-10-20T13:10:28.650937Z"},"phase":"finished","pid":15274,"started_at":"2026-10-20T13:10:28.908748Z","finished_at":"2026-10-20T13:13:22.529271Z","session_id":"01a1596f-4f3c-7b50-b9ce-3551a4df83ef","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.397923541s
head: "734d84a932b3c64e1043cf625a2038e71bb294a4"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-20T06:29:27-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 711 — couch --send-to hub refused 'no live registration' while both actors were live; advises a restart |
| repo | pair |
| issue file | workshop/issues/000711-couch-send-to-hub-refused-no-live-registration-while-both-actors-were-live-advises-a-restart.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7d4e0c3aba6b39fb12d5125ee967dcde9e5be4be..a85138a58b338cdba337fbb4b2e504ebfc5efffe |
| command | sdlc close --issue 711 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T06:29:27-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The implementation satisfies the revised contract, and all three prior findings are addressed with meaningful regression tests. Independent mutation checks reproduced each defect when its fix was removed. No new findings remain. The recorded full-suite and ordinary merge gates remain required; this review does not substitute for them.

Done-when: current — it covers single-observation listing, registration fencing, input queueing, diagnostics, parity, and query budgets.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      broker.go captures the sender actor under lock and checks its identity through routing, reservation, and queue fallback. Removing the identity comparison makes TestBrokerSenderReconnectCannotAuthorizeOldAdmission fail in all three phases.
  - id: BR-2
    disposition: addressed
    note: |
      OSRuntime.ListSessions passes validated states into buildListRowsForScope. Restoring RowState interpretation makes TestRuntimeListSessionsUsesValidatedOpaqueNameState fail for all three opaque-name cases.
  - id: BR-3
    disposition: addressed
    note: |
      Admission refusals identify caller or recipient and phase while preserving errors.Is classification. Removing the contextual wrappers makes TestBrokerAdmissionLossNamesSideAndPhase fail in all four cases.
```

1. **Strengths**

   - Sender lifetime validation uses one shared predicate at every admission boundary.
   - Direct, cached, and runtime inventory consumers derive identity and state from the validated full reply.
   - The input regression verifies queue promotion and exactly one delivery.
   - README and atlas explain the changed diagnostics and recovery guidance.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   `go test ./cmd/internal/couchmessage ./cmd/internal/launcher ./cmd/internal/couchcmd` passed. Disposable Go overlays independently demonstrated failures without each prior fix and were removed afterward. Coverage includes disconnect/reconnect controls, opaque names, malformed-refresh freshness, query budgets, hub/exact addressing, body lengths, and legacy CLI responses. The reviewed checkout remains unchanged.

6. **Architectural notes**

   - **ARCH-DRY: pass** — shared listing interpretation and caller predicate.
   - **ARCH-PURE: pass** — parsing remains separate from native IO; integration tests use existing seams.
   - **ARCH-PURPOSE: pass** — corrections sweep the enumerated admission and inventory consumers.
   - **ARCH-EVOLVE: pass** — no persisted format changes.

7. **Plan revision recommendations:** None. The existing revisions describe the delivered corrections accurately.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"e3336ff8-d20a-4362-bf88-d4d6f13ed6b7","repository":"github.com/xianxu/pair","issue_id":"000711","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/e2c7214096953d25","reviewed_head":"a85138a58b338cdba337fbb4b2e504ebfc5efffe","slot":"pair:7","requested_at":"2026-10-20T13:27:26.995158Z"},"phase":"finished","pid":65126,"started_at":"2026-10-20T13:27:27.088372Z","finished_at":"2026-10-20T13:29:25.974424Z","session_id":"01a1597e-d765-75f2-a946-077c9d9910c3","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.423615292s
head: "a85138a58b338cdba337fbb4b2e504ebfc5efffe"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-20T06:59:31-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 711 — couch --send-to hub refused 'no live registration' while both actors were live; advises a restart |
| repo | pair |
| issue file | workshop/issues/000711-couch-send-to-hub-refused-no-live-registration-while-both-actors-were-live-advises-a-restart.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1d741616743c5317286b599c5ce8d3b27e4b6380..6231932580842773fd14e5349f4b52a48d34c4cc |
| command | sdlc close --issue 711 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T06:59:31-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The interdiff correctly updates legacy test expectations for the already-approved single-listing behavior. All four affected tests pass independently. No production behavior or new scope is introduced; the added issue revision records the fixture corrections. BR-1, BR-2, and BR-3 remain disposed; there are no open prior findings.

Done-when: current — its single-listing, strict-validation, query-budget, and verification requirements cover these corrections.

```findings
{}
```

1. **Strengths**

   - Recovery tests now target the actual query sequence and add malformed/unknown full-listing coverage at `recovery_native_reply_test.go:114`.
   - Failure tests retain checks for no retries, no launcher effects, unchanged durable records, and unchanged actor registration.
   - Warm-resume coverage preserves four observation fences and verifies their query budget at two inventory sizes.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   All four affected tests passed with `-count=1` (`couchcore`, 6.263s). Production inspection confirms one full-listing query per observation and the revised recovery ordinals. `git diff --check` passed. The complete suite was not rerun during this review; its recorded gate remains required.

6. **Architectural notes**

   - **ARCH-DRY: pass** — recovery assertions reuse the shared expected query sequence.
   - **ARCH-PURE: pass** — integration tests exercise production query handling through existing injected fixtures.
   - **ARCH-PURPOSE: pass** — corrections preserve safety assertions while aligning fixtures with the approved contract.
   - **ARCH-EVOLVE: pass** — no persisted format or reader/writer behavior changes.

   No new user-facing or architectural surface requires README or atlas updates.

7. **Plan revision recommendations:** None. The appended revision accurately records these test-only corrections.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"2329e34d-9420-4200-94fb-d74c5ad7db20","repository":"github.com/xianxu/pair","issue_id":"000711","recipe":"fix-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/bac572951b144431","reviewed_head":"6231932580842773fd14e5349f4b52a48d34c4cc","slot":"pair:7","requested_at":"2026-10-20T13:58:24.353386Z"},"phase":"finished","pid":47960,"started_at":"2026-10-20T13:58:24.571573Z","finished_at":"2026-10-20T13:59:30.110898Z","session_id":"01a1599b-3044-77c1-8860-b85eeb20eef2","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.23605325s
head: "6231932580842773fd14e5349f4b52a48d34c4cc"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411
