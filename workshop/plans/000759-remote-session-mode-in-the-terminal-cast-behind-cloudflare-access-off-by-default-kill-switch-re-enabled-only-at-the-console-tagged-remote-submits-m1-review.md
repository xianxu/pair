# Boundary Review — pair#759 (milestone M1)

| field | value |
|-------|-------|
| issue | 759 — remote-session: 👽 mode in the terminal cast behind Cloudflare Access, off by default, kill switch re-enabled only at the console, tagged remote submits |
| repo | pair |
| issue file | workshop/issues/000759-remote-session-mode-in-the-terminal-cast-behind-cloudflare-access-off-by-default-kill-switch-re-enabled-only-at-the-console-tagged-remote-submits.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 9404e4e27c3cdd6eda6d2974f2aad57439a18e2c..1f260e9e9f1aa95d41ea540ea24a33d1c2d2168b |
| command | sdlc milestone-close --issue 759 --milestone M1 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T10:49:44-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The M1 foundation passes targeted and race tests and provides useful security boundaries without prematurely exposing remote access. Three gaps need resolution: the core-concepts table misclassifies session logic, session tests do not deliver the promised controlled sequence coverage, and crash-left publication files escape the declared storage bounds.

```findings
findings:
  - id: new
    severity: Critical
    family: core-concept-classification
    title: |
      Session admission does not match its PURE classification.
    detail: |
      workshop/plans/000759-remote-security-plan.md:94 lists a pure SessionState in session.go, but no such entity exists. Sessions.Handshake directly reads OS randomness and mutates synchronized state (session.go:43-94); its tests exercise that integration path. ARCH-PURE: promote Sessions to INTEGRATION and identify any actual pure admission values/functions, or extract the promised pure core. Add a Revisions entry correcting the classification.
  - id: new
    severity: Important
    family: deterministic-state-sequence-coverage
    title: |
      Session qualification lacks the promised generated sequences and controlled ordering.
    detail: |
      The plan's Sessions strategy at workshop/plans/000759-remote-security-plan.md:259 requires fake-clock generated sequences and a race rendezvous. session_test.go contains three hand-written tests; TestRemoteSessionsConcurrentCap at line 60 launches goroutines without controlling their ordering and never interleaves admission with revocation, logout or expiry reclamation. ARCH-ORDER: add reproducible sequence coverage with independent capacity/revocation invariants and held admission boundaries.
  - id: new
    severity: Important
    family: durable-artifact-lifecycle
    title: |
      Interrupted atomic publications leave unbounded remote-state staging files.
    detail: |
      store.go:67, origin_store.go:53 and audit.go:37 create unique .remote-latch-*, .remote-origin-* and .remote-audit-* files through durablefile.WriteAtomic. durablefile/write.go removes them only through deferred cleanup; process death before rename leaves them behind, and no opener or retention routine collects them. ARCH-FUNERAL: add ownership-safe recovery cleanup or a bounded reusable staging scheme, with interrupted-publication tests; otherwise repeated crashes bypass the stated latch/origin/audit disk bounds.
```

1. **Strengths**

   - The lifecycle reducer requires durable arm acknowledgement before opening and fences late completions after kill (`state.go:67`).
   - Access verification uses established signature libraries, pinned issuer/key URLs, bounded fetches and stateful signed-key rotation tests.
   - Persistence tests exercise actual rewrites and preserve unknown nested fields and large JSON numbers.
   - Permit tests cover single use, epoch/session revocation, expiry and held document-rebinding/navigation boundaries.

2. **Critical findings**

   - Correct the session PURE/INTEGRATION contradiction described above. Critical severity follows the review’s explicit core-concepts consistency rule.

3. **Important findings**

   - Deliver deterministic session sequence qualification.
   - Bound and reclaim interrupted publication artifacts across all three new staging-file families.

4. **Minor findings**

   - None.

5. **Test coverage notes**

   Passed:

   - `go test ./cmd/internal/remotesession ./cmd/internal/strictjson ./cmd/internal/artifactpath -count=1`
   - `go test -race ./cmd/internal/remotesession ./cmd/internal/strictjson -count=1`

   No prior boundary findings require disposition. The checkout remains unchanged. Public exposure, console reset and real-nvim acceptance remain later-milestone obligations.

6. **Architectural notes**

   | Marker | Result |
   |---|---|
   | ARCH-DRY | Pass: existing durablefile and strictjson authorities reused. |
   | ARCH-PURE | Flag: session classification contradicts implementation. |
   | ARCH-PURPOSE | Pass for M1 scope; runtime integration remains explicitly deferred. |
   | ARCH-MOCK | Pass for M1: stateful signed issuer and document backend exercise the seams. |
   | ARCH-CONSTRAINTS | Pass for implemented in-memory and published-record bounds. |
   | ARCH-SECURE | Pass: opaque permits, strict requests, verified identities and secret-safe errors. |
   | ARCH-ORDER | Flag: session ordering qualification is incomplete. |
   | ARCH-FUNERAL | Flag: crash-left staging files lack collection. |
   | ARCH-EVOLVE | Pass: supported records retain unknown fields through real rewrite paths. |

   Atlas updates cover the new architectural surface. No runnable user-facing command, flag or keybinding is introduced in M1, so a README update is not required for this range.

7. **Plan revision recommendations**

   Add `## Revisions` entries identifying the actual session entities and their classifications, and documenting the staging-file recovery owner, cleanup timing and storage bound. Retain the promised session sequence strategy and implement it before closing M1.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"7e66947f-6b48-4c5d-8579-93ce17324e03","repository":"github.com/xianxu/pair","issue_id":"000759","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/bfcc938d287cd6e8","reviewed_head":"1f260e9e9f1aa95d41ea540ea24a33d1c2d2168b","slot":"pair:6","requested_at":"2026-10-21T17:47:24.851454Z"},"phase":"finished","pid":91773,"started_at":"2026-10-21T17:47:24.973804Z","finished_at":"2026-10-21T17:49:43.716296Z","session_id":"01a15f93-34d4-7861-b862-01965ab91876","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.116282s
head: "1f260e9e9f1aa95d41ea540ea24a33d1c2d2168b"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000409

---

## Re-review — 2026-10-21T11:00:49-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 759 — remote-session: 👽 mode in the terminal cast behind Cloudflare Access, off by default, kill switch re-enabled only at the console, tagged remote submits |
| repo | pair |
| issue file | workshop/issues/000759-remote-session-mode-in-the-terminal-cast-behind-cloudflare-access-off-by-default-kill-switch-re-enabled-only-at-the-console-tagged-remote-submits.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 9404e4e27c3cdd6eda6d2974f2aad57439a18e2c..1811e47e668e236b6ce31de774954ce6759c7e2d |
| command | sdlc milestone-close --issue 759 --milestone M1 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T11:00:49-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M1’s foundation passes the targeted tests and race checks, and all three prior findings are addressed. One core-concepts table entry still contradicts the code. The supplied review contract explicitly makes that contradiction Critical; no new runtime defect was found.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The pinned correction removes nonexistent SessionState, classifies Sessions as INTEGRATION at plan line 101, and identifies pure identity predicates at line 93. session.go confirms entropy and mutex ownership; policy.go contains the value predicates. The Revisions entry records this correction.
  - id: BR-2
    disposition: addressed
    note: |
      session_sequence_test.go adds 24 seeded lifecycle sequences, held handshake boundaries interleaving revoke/logout/expiry, and rendezvous capacity waves of 8/16 callers. Independent ownership, historical-cookie and cancellation assertions exercise production Sessions methods; normal and race runs pass.
  - id: BR-3
    disposition: addressed
    note: |
      All four remote records use bounded reusable stages with opener recovery. Publication tests cover retained authority, foreign siblings and unsafe slots. In a disposable pinned-head copy, disabling reclaimRemoteStage made the recovery tests fail; the original tests passed. The disposable copy was removed.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: core-concept-classification
    title: |
      Core-concepts table assigns DocumentResult to the wrong file.
    detail: |
      workshop/plans/000759-remote-security-plan.md:94 places DocumentResult in protocol.go, but its declaration is cmd/internal/remotesession/documents.go:18. This is the 2nd finding in family core-concept-classification. ARCH-PURE / ARCH-PURPOSE: apply the rule that every delivered table entity must match its actual declaration, classification and path; sweep all delivered rows, split this row by location, and record the correction under Revisions. Critical follows the review contract’s explicit severity for table/code contradictions.
```

1. **Strengths**

   - Access verification uses established signature libraries, pins issuer/audience/key retrieval, and tests signed-token rejection and key rotation.
   - Session tests now reproduce admission boundaries and check capacity and revocation independently.
   - Publication recovery preserves committed authority and rejects unsafe staging slots across all four owners.
   - Atlas documents the new foundation and clearly distinguishes it from pending runtime wiring.

2. **Critical findings**

   - Plan line 94 mislocates `DocumentResult`. Split the row to identify `documents.go:18`, sweep the remaining delivered entries, and append a Revisions entry.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:

   - `go test ./cmd/internal/remotesession ./cmd/internal/strictjson ./cmd/internal/artifactpath -count=1`
   - `go test -race ./cmd/internal/remotesession ./cmd/internal/strictjson -count=1`

   Stage-recovery mutation testing confirmed regression sensitivity. Repository status remained clean.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — pass:** shared publication helper and existing durability primitives.
   - **ARCH-PURE — flag:** table-location contradiction above; inspected value logic remains separated from IO.
   - **ARCH-PURPOSE — flag:** repeat family needs a complete delivered-row sweep.
   - **ARCH-MOCK — pass for M1:** signed issuer and stateful document fixtures; live conformance remains later work.
   - **ARCH-CONSTRAINTS — pass:** inspected session, replay, key-fetch and artifact bounds are enforced.
   - **ARCH-SECURE — pass:** typed permits, strict request decoding and fail-closed storage paths.
   - **ARCH-ORDER — pass for M1:** reducer invariants and controlled session interleavings; supervisor enforcement remains M2.
   - **ARCH-FUNERAL — pass:** bounded reusable staging resolves the reported residue problem.
   - **ARCH-EVOLVE — pass:** retained unknown fields and strict known-field validation have rewrite-path coverage.

   README’s gate is not triggered by this range: no runnable user-facing command, flag or configuration surface is implemented yet.

7. **Plan revision recommendations**

   Add a `## Revisions` entry recording the complete delivered-entity audit and corrected `DocumentResult` location.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"49b2908e-9f38-4c2b-a5a1-c156d32efe53","repository":"github.com/xianxu/pair","issue_id":"000759","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/160aefea4c6cd3b6","reviewed_head":"1811e47e668e236b6ce31de774954ce6759c7e2d","slot":"pair:6","requested_at":"2026-10-21T17:58:39.193255Z"},"phase":"finished","pid":7746,"started_at":"2026-10-21T17:58:39.359077Z","finished_at":"2026-10-21T18:00:48.648989Z","session_id":"01a15f9d-7fdb-7723-b305-56594b190ef3","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.245334709s
head: "1811e47e668e236b6ce31de774954ce6759c7e2d"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
