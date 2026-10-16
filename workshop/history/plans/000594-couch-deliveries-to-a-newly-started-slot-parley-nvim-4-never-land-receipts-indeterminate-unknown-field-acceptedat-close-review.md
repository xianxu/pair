# Boundary Review — pair#594 (whole-issue close)

| field | value |
|-------|-------|
| issue | 594 — Couch: deliveries to a newly started slot (parley.nvim:4) never land, receipts 'indeterminate' (unknown field AcceptedAt) |
| repo | pair |
| issue file | workshop/issues/000594-couch-deliveries-to-a-newly-started-slot-parley-nvim-4-never-land-receipts-indeterminate-unknown-field-acceptedat.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0169e962f95a932fd9884b20b2d5dd07b73022f1..3ac645d1019c0d8e071ed9af5aafdc31c1b95768 |
| command | sdlc close --issue 594 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T19:26:20-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The compatibility fixes are focused and both affected package suites pass. One Important tracker inconsistency blocks closure: the revised contract permits uncertainty and defers live acceptance, but `Done when` still requires a definite outcome and a completed live delivery.

Done-when: stale — neither clause reflects the revised acceptance contract.

```findings
findings:
  - id: new
    severity: Important
    family: acceptance-criteria-track-contract
    title: |
      Done when still claims acceptance conditions the revisions defer or change
    detail: |
      In workshop/issues/000594-couch-deliveries-to-a-newly-started-slot-parley-nvim-4-never-land-receipts-indeterminate-unknown-field-acceptedat.md:45-46, both acceptance clauses are stale: line 45 requires a definite outcome, while the revision at line 68 correctly preserves uncertainty after failed replies; line 46 requires live delivery, while lines 62 and 72 explicitly defer it until the operator's restart. These are every instance of this family in the window. Align both clauses with the approved revised contract and distinguish completed automated acceptance from pending production acceptance; otherwise fulfill the existing clauses before closing. ARCH-PURPOSE requires closure criteria to describe the delivered purpose accurately.
```

1. **Strengths**
   - Literal legacy schemas test compatibility independently of current `Message` definitions.
   - Commit requests omit the timestamp through a copy, preserving the broker’s canonical message (`endpoint.go:205`).
   - Decode diagnostics survive broker status scans; tests assert exactly one commit.
   - Release tests cover both before-paste cancellation and after-paste uncertainty without additional input.

2. **Critical findings:** None.

3. **Important findings:** The acceptance-criteria inconsistency above.

4. **Minor findings:** None.

5. **Test coverage:** `go test ./cmd/internal/couchmessage ./cmd/internal/wrapcmd` passed. Coverage includes zero/nonzero timestamps, commit/status replies, reverse compatibility, malformed replies, and release timing. `git diff --check` passed. Production live acceptance remains pending.

6. **Architecture:** ARCH-DRY passes: existing transport and reducer paths are reused. ARCH-PURE passes: transitions remain IO-free and socket handling stays at the boundary. ARCH-PURPOSE flags the acceptance mismatch above. Atlas covers the changed protocol behavior; no new user-facing command surface requires README changes.

7. **Plan revision recommendation:** Add a `## Revisions` entry explicitly reconciling both `Done when` clauses with the revised uncertainty policy and deferred production check, then update the clauses accordingly.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"7a9dd2a0-8f34-4a62-8a22-3dfe822e8583","repository":"github.com/xianxu/pair","issue_id":"000594","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/268969baedd81b33","reviewed_head":"3ac645d1019c0d8e071ed9af5aafdc31c1b95768","slot":"pair:2","requested_at":"2026-10-16T02:23:58.960146Z"},"phase":"finished","pid":64001,"started_at":"2026-10-16T02:23:59.134433Z","finished_at":"2026-10-16T02:26:18.688468Z","session_id":"01a14285-fc06-7c70-a42e-1864359eb5ca","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.541315166s
head: "3ac645d1019c0d8e071ed9af5aafdc31c1b95768"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-15T19:30:07-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 594 — Couch: deliveries to a newly started slot (parley.nvim:4) never land, receipts 'indeterminate' (unknown field AcceptedAt) |
| repo | pair |
| issue file | workshop/issues/000594-couch-deliveries-to-a-newly-started-slot-parley-nvim-4-never-land-receipts-indeterminate-unknown-field-acceptedat.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0169e962f95a932fd9884b20b2d5dd07b73022f1..237b26648a48c394f79bb2cff5bc16da03c3ee3c |
| command | sdlc close --issue 594 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-15T19:30:07-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The implementation matches the revised contract. BR-1’s documentation correction is complete, both affected package suites and targeted race regressions pass, and no new blocking findings remain. Production acceptance is explicitly pending with the operator after restart.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Inspected the pinned before/after correction: issue lines 37–45 now preserve uncertainty and distinguish automated acceptance from pending operator-owned production acceptance; the revision at line 75 records the superseded promises. This matches endpoint.go:205–210's single commit attempt and broker.go:969–972's Indeterminate error outcome. The correction changes prose only.
```

1. **Strengths**
   - Literal legacy request and reply schemas independently verify both compatibility directions.
   - Commit serialization clears only a message copy, preserving the broker’s admission timestamp (`endpoint.go:205`).
   - Decode diagnostics retain slot identity and restart advice through status scans.
   - Release regressions cover cancellation before paste and uncertainty afterward, without additional input or false operator blame.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage:** Passed `go test ./cmd/internal/couchmessage ./cmd/internal/wrapcmd`, targeted race regressions, and `git diff --check`. Full repository suite/build were not rerun during this review. Production acceptance remains pending as documented.

6. **Architecture:** **ARCH-DRY passes:** existing transport and reducer paths are reused. **ARCH-PURE passes:** transitions remain IO-free; socket handling stays in adapters. **ARCH-PURPOSE passes:** code and revised acceptance criteria cover both skews, truthful uncertainty, and release attribution. Atlas documents the changed behavior; no new README-facing command surface was introduced. No Core concepts table requires reconciliation.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"612c83b7-c174-493e-82eb-46164273fbc5","repository":"github.com/xianxu/pair","issue_id":"000594","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/94d3b83a12783416","reviewed_head":"237b26648a48c394f79bb2cff5bc16da03c3ee3c","slot":"pair:2","requested_at":"2026-10-16T02:28:16.413066Z"},"phase":"finished","pid":99875,"started_at":"2026-10-16T02:28:16.509441Z","finished_at":"2026-10-16T02:30:07.215957Z","session_id":"01a14289-e8d8-7ed0-8ad1-d1ac7f18cc1f","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.275357208s
head: "237b26648a48c394f79bb2cff5bc16da03c3ee3c"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
