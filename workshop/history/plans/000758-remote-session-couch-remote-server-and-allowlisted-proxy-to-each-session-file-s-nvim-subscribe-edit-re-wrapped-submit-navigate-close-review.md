# Boundary Review — pair#758 (whole-issue close)

| field | value |
|-------|-------|
| issue | 758 — Couch-owned editor endpoint registry and lifecycle |
| repo | pair |
| issue file | workshop/issues/000758-remote-session-couch-remote-server-and-allowlisted-proxy-to-each-session-file-s-nvim-subscribe-edit-re-wrapped-submit-navigate.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1e70379fba4a3653258f5542ac1d1aecf0ca1b43..1ad3d3b8ad267ae72c9aaba5b773eb3c6d5dfeaf |
| command | sdlc close --issue 758 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T06:01:46-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The producer has a well-separated authority model, durable storage, private CLI/IPC integration, and substantial passing tests. Two correctness defects block shipping: stale negative probes can report false absence, and a predictable capacity refusal can poison the entire endpoint owner. Notification lifecycle qualification also falls short of the checked-off plan.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: observation-generation-fencing
    title: |
      A late negative lookup reports confirmed absence despite a current successor
    detail: |
      cmd/internal/editorendpoint/owner.go:414-424 fences retirement against the captured binding but unconditionally returns Code=ok with EditorEndpoint=null afterward. If B replaces A while A's probe is pending, a subsequent nil response for A cannot establish absence of B. Recheck current authority under the same lock as retirement and return stale or validate the successor. Extend owner_test.go:234 to assert the delayed lookup response; line 260 currently discards it. ARCH-ORDER, ARCH-SECURE.
  - id: new
    severity: Critical
    kind: defect
    family: prepublication-refusal-isolation
    title: |
      Exceeding the encoded-state limit poisons otherwise healthy endpoint authority
    detail: |
      cmd/internal/editorendpoint/owner.go:129-131 treats every Store.Publish error as uncertain publication. However, store.go:112-114 can reject encoding before invoking any writer, including codec.go:139-140 rejecting a candidate above 8 MiB. A valid near-capacity registry therefore becomes unavailable for lookup, unregister, and cleanup until restart after one oversized admission. Preflight candidate encoding and classify size exhaustion as capacity without poisoning or changing authority; reserve publication uncertainty for attempted writes. Add an owner-level regression proving no writer call and continued operation after refusal. ARCH-CONSTRAINTS, ARCH-ORDER.
  - id: new
    severity: Important
    kind: defect
    family: external-lifecycle-qualification
    title: |
      Notification doubles cannot exercise the delayed and failed lifecycle promised by the plan
    detail: |
      cmd/internal/editorendpoint/owner_test.go:69-73 only appends notices and always succeeds; couchcmd/editor_endpoint_service_test.go:124-125 is a no-op. The native callback at nvim_live_test.go:65-68 only writes a token. None can hold or fail Superseded, although plan line 168 marks held-notification qualification complete. Add controllable notification completion/failure through the production seam and verify publication precedes delivery, failure preserves the successor, and shutdown cancels and joins an in-flight notification. Model pending/current attachment identity in the editor double. ARCH-MOCK, ARCH-ORDER, ARCH-PURPOSE.
```

1. **Strengths**

   - Retry, supersession, compare-release, and process collection run through one pure transition function.
   - The retained-state codec preserves unknown fields at every owned object level, including numbers outside float64 range, while retaining strict known-field validation.
   - Private native tests use isolated environments and owned process cleanup.
   - README and atlas changes cover the new commands, authority model, lifecycle, and producer-only scope.

2. **Critical findings**

   - Fence the **negative lookup response**, as well as the retirement mutation, against successor generations.
   - Separate deterministic prepublication refusal from uncertain storage publication.

3. **Important findings**

   - Complete controlled notification lifecycle tests and correct the plan’s qualification claim.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed full package tests for `editorendpoint`, `strictjson`, `procutil`, `sessionsync`, and `couchcmd`. Focused endpoint/native/codec/process tests also passed with race detection. `git diff --check` passed.

   Existing coverage checks successor preservation but discards the delayed lookup’s response. Codec size tests do not check owner availability after size refusal. No repository-wide recorded suite result is claimed by this review.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared namespace, document identity, transport, durability, and JSON helpers are reused.
   - **ARCH-PURE — pass:** authority decisions and codecs are independently testable without IO.
   - **ARCH-PURPOSE — flag:** checked-off notification qualification is incomplete; the documented producer-only split otherwise matches implementation.
   - **ARCH-MOCK — flag:** notification doubles lack controllable stateful lifecycle behavior.
   - **ARCH-CONSTRAINTS — flag:** byte-cap refusal incorrectly disables healthy authority.
   - **ARCH-SECURE — flag:** evidence about a displaced attachment becomes an unsupported claim of document absence.
   - **ARCH-ORDER — flag:** negative response generation checks and publication outcome classification need correction.
   - **ARCH-FUNERAL — pass:** retained evidence has bounded storage and positive-death collection; handles are joined and editor-owned paths remain externally owned.
   - **ARCH-EVOLVE — pass:** nested unknown fields survive retained-state rewrites with strict validation.

7. **Plan revision recommendations**

   Add concrete `## Revisions` entries specifying:

   - Negative lookup completion atomically checks current authority before confirming absence.
   - Encoding/capacity refusal occurs before publication and leaves the owner usable.
   - Notification qualification includes held, failed, and shutdown-interrupted delivery; update completion claims after those tests pass.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"7a1f5416-a281-4580-9698-8074dacfef93","repository":"github.com/xianxu/pair","issue_id":"000758","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/653fd1bc9ae6485d","reviewed_head":"1ad3d3b8ad267ae72c9aaba5b773eb3c6d5dfeaf","slot":"pair:6","requested_at":"2026-10-22T12:58:08.115608Z"},"phase":"finished","pid":5890,"started_at":"2026-10-22T12:58:08.24957Z","finished_at":"2026-10-22T13:01:46.159896Z","session_id":"01a163b0-b992-73d2-ba88-164aae0a719c","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.738942208s
head: "1ad3d3b8ad267ae72c9aaba5b773eb3c6d5dfeaf"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-22T06:17:03-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 758 — Couch-owned editor endpoint registry and lifecycle |
| repo | pair |
| issue file | workshop/issues/000758-remote-session-couch-remote-server-and-allowlisted-proxy-to-each-session-file-s-nvim-subscribe-edit-re-wrapped-submit-navigate.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1e70379fba4a3653258f5542ac1d1aecf0ca1b43..3a98841ea83eb10e2af5189bb66c8754fd0db5a7 |
| command | sdlc close --issue 758 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T06:17:03-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The three prior findings are addressed, with executable regression evidence for both behavior fixes and controlled notification lifecycle tests. The producer implementation, native conformance, and targeted suites pass. One Important gap remains: the plan marks publication-byte qualification complete, but the N8/16 tests measure calls and record counts without measuring publication bytes.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      owner.go:433-456 fences negative retirement and response against current authority. TestOwnerLateNegativeLookupCannotRetireSuccessor covers nil, mismatch, death and document absence; all four fail with the pre-fix owner supplied through a temporary Go overlay and pass at head.
  - id: BR-2
    disposition: addressed
    note: |
      Store.publish distinguishes pre-writer refusal from attempted IO. TestOwnerEncodedCapacityRefusalKeepsAuthorityUsable fails with the pre-fix owner and passes at head, verifying no writer call and continued lookup, unregister and cleanup. Additional tests preserve uncertainty for writer-returned capacity errors.
  - id: BR-3
    disposition: addressed
    note: |
      owner_test.go:740-877 exercises held, failed and canceled Superseded delivery through the production RPC seam, including durable publication before delivery, successor preservation and joined shutdown. The fixture tracks pending/current attachment identity. This correction adds qualification without changing production behavior.
findings:
  - id: new
    severity: Important
    kind: defect
    family: operating-envelope-qualification
    title: |
      Checked publication-byte qualification has no byte accounting
    detail: |
      ARCH-CONSTRAINTS: workshop/plans/000758-editor-endpoints-plan.md:119,157,167 explicitly promises and marks N8/16 publication-byte qualification complete. However, owner_test.go:184-195 counts only writes, and TestOwnerBatchOnePublicationAndPlateau at line 431 checks publication calls and retained-record counts without recording encoded sizes or cumulative bytes. Instrument the injected writer and extend the N8/16 scenarios to verify snapshot-byte growth, retry/no-op zero-write behavior and bounded repeated-reopen size.
```

1. **Strengths**

   - Authority mutations flow through the pure transition model; generated sequence tests use an independent selection oracle.
   - Negative lookup responses now share the retirement lock, preventing false absence claims about successors.
   - Retained unknown JSON fields survive actual store reopening, including numeric values outside float64 range.
   - README, atlas, artifact classifications and reader inventories cover the new surface.

2. **Critical findings:** None.

3. **Important findings:** Publication-byte qualification above. The writer seam already provides a straightforward place to collect the missing measurements.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed all tests in `editorendpoint`, `strictjson`, `procutil`, `couchcmd` and `sessionsync`; focused endpoint/CLI race tests; all three installed-nvim conformance tests without skips; and `make test-affected-inventory`. BR-1 and BR-2 regressions failed against the pre-fix owner using a temporary overlay. The checkout remains unchanged. This review did not run the repository-wide recorded suite.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared JSON helpers, existing namespace resolution and durable writer reused.
   - **ARCH-PURE — pass:** transition and codec logic remain free of IO.
   - **ARCH-PURPOSE — pass:** approved producer-only scope is delivered; adapter work remains explicitly assigned to #798.
   - **ARCH-MOCK — pass:** injected stateful seams and native conformance cover external behavior.
   - **ARCH-CONSTRAINTS — flag:** publication-byte evidence remains incomplete.
   - **ARCH-SECURE — pass:** strict boundary decoding, private socket validation and explicit uncertainty handling.
   - **ARCH-ORDER — pass:** generation fencing, publication-before-notification and cancellation/join sequences are exercised.
   - **ARCH-FUNERAL — pass:** exact process collection, reusable stage cleanup and owned handle shutdown.
   - **ARCH-EVOLVE — pass:** nested unknown-field preservation and strict known-field validation are tested.

7. **Plan revision recommendations**

   Add a `## Revisions` entry acknowledging the missing byte qualification; leave that portion of task 167 pending until the measurements and assertions pass.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"20ddc55f-d1cb-45cd-84ce-e3e7f9a942dd","repository":"github.com/xianxu/pair","issue_id":"000758","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/b8110d64d1141f22","reviewed_head":"3a98841ea83eb10e2af5189bb66c8754fd0db5a7","slot":"pair:6","requested_at":"2026-10-22T13:13:31.726687Z"},"phase":"finished","pid":64593,"started_at":"2026-10-22T13:13:31.903171Z","finished_at":"2026-10-22T13:17:02.880271Z","session_id":"01a163be-d1f5-7f70-b15f-dd18c9c24c50","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.076029625s
head: "3a98841ea83eb10e2af5189bb66c8754fd0db5a7"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-22T06:28:02-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 758 — Couch-owned editor endpoint registry and lifecycle |
| repo | pair |
| issue file | workshop/issues/000758-remote-session-couch-remote-server-and-allowlisted-proxy-to-each-session-file-s-nvim-subscribe-edit-re-wrapped-submit-navigate.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1e70379fba4a3653258f5542ac1d1aecf0ca1b43..be457f2be2f39f88610d131c03955aef84989843 |
| command | sdlc close --issue 758 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T06:28:02-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range delivers the documented producer scope. BR-4’s missing byte qualification is now measured and asserted through the actual injected writer. Targeted tests, race checks, native Neovim conformance, and repository checks passed. No new blocking findings.

```findings
dispose:
  - id: BR-4
    disposition: addressed
    note: |
      owner_test.go:155-179,210,459-623 records actual encoded writer payload lengths and verifies N8/16 snapshot/cumulative bounds, zero retry/no-op/reload writes, single batch publication, and repeated-reopen size plateau. The race run passed; negative controls reject duplicate publications and retained attempt growth. This correction adds qualification without changing production behavior.
```

1. **Strengths**

   - Authority changes use the pure transition model; publication precedes acknowledgement and replacement notification.
   - Controlled ordering tests exercise late negative observations, held/failed notifications, and joined shutdown.
   - Retained JSON preserves nested unknown fields and numeric tokens while keeping known fields strict.
   - README, atlas, artifact classifications, and reader inventories cover the new surface.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed:
   
   - `editorendpoint`, `couchcmd`, `strictjson`, `procutil`, and `sessionsync` package tests.
   - Focused owner race tests, including byte qualification and previous correction regressions.
   - All three private installed-Neovim conformance tests.
   - Artifactpath, repolint, affected-reader inventory, and diff whitespace checks.

   Measured N8/16 cumulative writer payloads were 691,008/2,563,116 bytes across 144/272 publications. These checks do not substitute for the separately required recorded full suite.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Shared namespace, transport, durability, and JSON helpers are reused.
   - **ARCH-PURE — pass:** Model and codec tests run without IO; adapters own external effects.
   - **ARCH-PURPOSE — pass:** Delivers the approved producer-only scope and real CLI/private IPC composition.
   - **ARCH-MOCK — pass:** Stateful editor/process/storage seams share production boundaries; native conformance passed.
   - **ARCH-CONSTRAINTS — pass:** Capacity, admission, output, and snapshot-byte bounds have executable coverage.
   - **ARCH-SECURE — pass:** Typed boundary validation, private socket checks, and safe RPC diagnostics are present.
   - **ARCH-ORDER — pass:** Generation fencing, publication uncertainty, delayed completion, and cancellation are tested.
   - **ARCH-FUNERAL — pass:** Exact process-death collection, reusable stage cleanup, and owned shutdown define lifetimes.
   - **ARCH-EVOLVE — pass:** Nested unknown members survive rewrites; malformed authority refuses visibly.

7. **Plan revision recommendations:** None required.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"cb392adb-75e3-43d1-8b56-8e39a47c6e1b","repository":"github.com/xianxu/pair","issue_id":"000758","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/e88d998153dfca81","reviewed_head":"be457f2be2f39f88610d131c03955aef84989843","slot":"pair:6","requested_at":"2026-10-22T13:25:47.497224Z"},"phase":"finished","pid":46658,"started_at":"2026-10-22T13:25:47.63079Z","finished_at":"2026-10-22T13:28:01.35204Z","session_id":"01a163ca-0b6f-72b1-8956-fe0bd7f66262","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.695606292s
head: "be457f2be2f39f88610d131c03955aef84989843"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
