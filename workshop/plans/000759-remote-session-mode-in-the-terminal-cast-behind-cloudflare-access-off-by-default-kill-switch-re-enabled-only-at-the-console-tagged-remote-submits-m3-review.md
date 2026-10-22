# Boundary Review — pair#759 (milestone M3)

| field | value |
|-------|-------|
| issue | 759 — remote-session: 👽 mode in the terminal cast behind Cloudflare Access, off by default, kill switch re-enabled only at the console, tagged remote submits |
| repo | pair |
| issue file | workshop/issues/000759-remote-session-mode-in-the-terminal-cast-behind-cloudflare-access-off-by-default-kill-switch-re-enabled-only-at-the-console-tagged-remote-submits.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 27dffa8d88b0ba49e30b1baeb695c1fd5c02e05d..981cd0b4416daac6af7a8724a9b2b4fa125a0220 |
| command | sdlc milestone-close --issue 759 --milestone M3 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T22:35:10-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The pinned range delivers substantial provenance protection, bounded document routes, and compatible receiver admission. Focused tests passed across all five affected packages, and complete race tests passed for remotesession and sessionsync. SHIP is blocked by a local-submit compatibility regression, an unbounded ledger failure path, and incomplete qualification of dispatch completion ordering. The checkout remains unchanged.

1. **Strengths**

- Remote provenance participates in message identity, receipt matching, batch validation, and retained-message checks; unsupported receivers visibly refuse delivery.
- Ledger tests exercise actual append, trim, reopen, and owed projection while preserving unknown nested fields.
- Signed HTTP-to-broker tests count unique deliveries and audit records at N=8/16 and reject browser-selected provenance.
- README and atlas updates describe the new surfaces and distinguish private qualification from deferred editor and operator acceptance.

2. **Critical findings**

```findings
findings:
  - id: new
    severity: Critical
    family: submit-owner-resolution
    title: |
      Local submit admission rejects valid cross-slot submissions
    detail: |
      cmd/internal/couchcmd/remote_submit.go:37 requires the registered document's effective owner to equal from.Slot. The owner is the submit destination, and an existing supported flow sends from another slot: session_sync_rename_delivery_test.go registers the document to the recipient and submits from a different caller. With production originNamespace initialized, that flow now refuses before even checking whether the document was remotely edited, breaking the promised compatibility for unmarked local submits. ARCH-PURPOSE: preserve authorized cross-slot sends, resolve ownership against the destination as appropriate, and add production-service regressions with distinct sender/owner for both marked and unmarked documents. The new test currently uses sender=owner and a different recipient, so it misses this regression.
  - id: new
    severity: Important
    family: durable-artifact-lifecycle
    title: |
      Invalid provenance permanently disables ledger trimming without stopping appends
    detail: |
      cmd/internal/sessionsync/ledger.go:119 appends before trim; the changed readRows now refuses malformed recognized provenance, causing trim at lines 140–142 to fail indefinitely while retaining that row. Every later delivery still appends another row before returning the same failure, so a damaged ledger grows without the documented size/age/row bound. TestLedgerMalformedProvenanceCannotBeTrimmedAway establishes the retained failure condition but does not check repeated growth. This is the 2nd finding in family durable-artifact-lifecycle. ARCH-FUNERAL / ARCH-CONSTRAINTS: state and enforce the rule that failed retention cannot leave a writer growing indefinitely; sweep ledger append/trim failure paths, preserve damaged evidence, report the recording gap, and test repeated appends across owner recreation.
  - id: new
    severity: Important
    family: deterministic-state-sequence-coverage
    title: |
      Dispatch completion ownership lacks the promised controlled ordering tests
    detail: |
      cmd/internal/remotesession/dispatch.go:114–128 and 204–234 introduce completion ownership across observer cancellation, backend results, audit publication and teardown. Existing controlled tests hold audit intent or the backend before final handoff; cancellation coverage cancels before admission, and the result-audit failure test exercises one uncontrolled completion ordering. They do not hold an accepted effect/result publication while disconnecting the observer or revoking the session, then verify replay, receipt preservation and cleanup/reset exclusion. This is the 2nd finding in family deterministic-state-sequence-coverage. ARCH-ORDER: enumerate the applicable completion/revocation orderings and qualify the production owner with held results, held result-audit writes and controlled time; assert independent effect-count, uncertainty and join invariants across that enumeration.
```

3. **Important findings**

The ledger growth and dispatch qualification findings above must be resolved before admission.

4. **Minor findings**

None.

5. **Test coverage notes**

Passed:

- Focused Remote/Origin/Forget/Ledger/Owed/Dispatch/Replay/Document/Server/SkillDocumentsRecovery tests across remotesession, couchmessage, sessionsync, couchcmd, and wrapcmd.
- Complete `go test -race` runs for remotesession and sessionsync.

The existing cross-slot submit test uses a service fixture without the new origin namespace, bypassing the regression. No prior findings require disposition in this round.

6. **Architectural notes**

- **ARCH-DRY — pass:** shared provenance comparisons and one service-lifetime origin owner.
- **ARCH-PURE — pass:** value validation/projection remains separate from classified IO owners; concept locations were cross-checked.
- **ARCH-PURPOSE — flag:** local-submit compatibility is broken by sender/owner conflation.
- **ARCH-MOCK — pass for this milestone:** stateful private document, issuer, transport, and receiver seams exist. Live conformance remains explicitly deferred.
- **ARCH-CONSTRAINTS — flag:** ledger growth escapes its bound after validation failure.
- **ARCH-SECURE — pass:** browser provenance is rejected, permit authority is internal, and malformed recognized ledger metadata refuses visibly.
- **ARCH-ORDER — flag:** post-handoff completion orderings lack deterministic qualification.
- **ARCH-FUNERAL — flag:** the ledger writer continues growing after retention becomes permanently unavailable.
- **ARCH-EVOLVE — pass:** retained-raw codecs and actual rewrite tests preserve unknown root and nested fields.

7. **Plan revision recommendations**

Add specific `## Revisions` entries recording:

- The distinction between submit caller and effective document owner, with marked/unmarked cross-slot compatibility coverage.
- The writer-bound rule during ledger validation or retention failure.
- The enumerated post-handoff completion/revocation orderings and their deterministic qualification evidence.

Operator Access/iPhone acceptance and actual #758 editor integration should remain explicitly pending.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"1bb9b055-82f7-43bb-adf7-140958b59cfc","repository":"github.com/xianxu/pair","issue_id":"000759","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/1d26bc4d73270993","reviewed_head":"981cd0b4416daac6af7a8724a9b2b4fa125a0220","slot":"pair:6","requested_at":"2026-10-22T05:32:05.72429Z"},"phase":"finished","pid":75818,"started_at":"2026-10-22T05:32:05.914992Z","finished_at":"2026-10-22T05:35:09.989261Z","session_id":"01a16218-6118-70c0-a309-78409d7611d2","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.65167525s
head: "981cd0b4416daac6af7a8724a9b2b4fa125a0220"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411

---

## Re-review — 2026-10-21T22:51:25-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 759 — remote-session: 👽 mode in the terminal cast behind Cloudflare Access, off by default, kill switch re-enabled only at the console, tagged remote submits |
| repo | pair |
| issue file | workshop/issues/000759-remote-session-mode-in-the-terminal-cast-behind-cloudflare-access-off-by-default-kill-switch-re-enabled-only-at-the-console-tagged-remote-submits.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 27dffa8d88b0ba49e30b1baeb695c1fd5c02e05d..5136229868a8fc0806d934317d8104154e763dee |
| command | sdlc milestone-close --issue 759 --milestone M3 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T22:51:25-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The reviewed M3 code boundary satisfies the documented scope. All three prior findings are addressed with meaningful regression evidence, and no new blocking findings emerged. This verdict covers code and private qualification; live Access/iPhone acceptance, #758 editor integration, and landing remain pending as documented.

```findings
dispose:
  - id: BR-9
    disposition: addressed
    note: |
      remote_submit.go:36–48 separates sender from document ownership and preserves unmarked legacy admission. The production-service regression covers distinct sender/owner for marked and unmarked documents; restoring the owner-equality rejection made both cases fail, and restoring HEAD passed.
  - id: BR-10
    disposition: addressed
    note: |
      ledger.go:119 reserves the next encoded row through retention before append. Repeated failure tests across recreated owners cover malformed evidence, scanner failure, retention publication, write-open and gap-expiry failures. Removing preflight caused evidence growth and regression failures; restored HEAD passed.
  - id: BR-11
    disposition: addressed
    note: |
      dispatch_completion_test.go:217 enumerates held backend-result, held result-audit and settled cuts against observer cancellation, logout and kill, asserting effect counts, replay, uncertainty, receipt preservation and cleanup/reset exclusion. Removing receipt observation made controlled result-audit cases fail; restored HEAD passed.
```

1. **Strengths**

   - Local provenance follows stable registration identity while preserving cross-slot routing and unmarked compatibility.
   - Ledger retention preflight prevents repeated growth without deleting damaged evidence.
   - Completion tests control event ordering through rendezvous and assert independent effect and durability counts.
   - README and atlas describe the delivered API, conservative forget restriction, transcript limits, and pending integrations.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Complete `remotesession`, `sessionsync`, `couchmessage`, `couchcmd`, and `wrapcmd` packages passed.
   - Focused dispatch and ledger tests passed under the race detector.
   - Affected-reader inventory and pinned-range whitespace checks passed.
   - Independent mutation controls failed for all three prior findings; restored tests passed. The disposable copy was removed, and the reviewed checkout remained clean.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Shared provenance validation and comparison cover broker, endpoint, batch, and wrapper paths.
   - **ARCH-PURE — pass:** Value validation and projections remain separate from synchronized IO owners.
   - **ARCH-PURPOSE — pass:** Trusted provenance reaches delivery, receipts, ledger, owed projections, and transcript fixtures.
   - **ARCH-MOCK — pass:** Stateful backend and endpoint fixtures exercise production seams; real-editor acceptance remains explicitly pending.
   - **ARCH-CONSTRAINTS — pass:** Request, replay, stream, operation, and storage bounds have enforcement and representative tests.
   - **ARCH-SECURE — pass:** Browser inputs cannot select trusted provenance; malformed recognized ledger evidence refuses visibly.
   - **ARCH-ORDER — pass:** Controlled completion/revocation sequences verify ownership, uncertainty, replay, and join exclusion.
   - **ARCH-FUNERAL — pass:** Failed retention stops growth; origin capacity and conservative retirement restrictions are explicit.
   - **ARCH-EVOLVE — pass:** Unknown root and nested fields survive actual rewrite paths, with strict known-field validation.

7. **Plan revision recommendations:** None required for this boundary.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"784cdea0-d0a2-4305-a738-86ce8dc9f16b","repository":"github.com/xianxu/pair","issue_id":"000759","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/914dceacecef926c","reviewed_head":"5136229868a8fc0806d934317d8104154e763dee","slot":"pair:6","requested_at":"2026-10-22T05:47:43.826413Z"},"phase":"finished","pid":62544,"started_at":"2026-10-22T05:47:43.953572Z","finished_at":"2026-10-22T05:51:24.859404Z","session_id":"01a16226-ad48-7e12-a14a-42cbe3251407","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.650979833s
head: "5136229868a8fc0806d934317d8104154e763dee"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
