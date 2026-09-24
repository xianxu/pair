# Boundary Review — pair#313 (whole-issue close)

| field | value |
|-------|-------|
| issue | 313 — Add a slot from the repository thread action menu |
| repo | pair |
| issue file | workshop/issues/000313-couch-add-slot-action.md |
| boundary | whole-issue close |
| milestone | — |
| window | c6a91fe2398275359e99aa1c14373edf1a7dadc1..4ad7944a5ee6b021dcc1110ad878079518e87bbb |
| command | sdlc close --issue 313 |
| reviewer | codex |
| timestamp | 2026-09-23T19:58:28-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The #313 implementation is well-tested and correctly reuses the existing start-preview flow. However, the review window also includes an unrelated bootstrap/CI rewrite that breaks clean consumer checkouts: `scripts/run-merge-checks.sh` remains a dangling peer symlink, while `bootstrap.sh` no longer clones that peer and instead requires Homebrew/weave.

1. Strengths:

- Exact repository-root derivation and slot validation are reused (`menu_switchagent.go:233-240`).
- Fingerprint-bound creation, cancellation, late-preview rejection, and admission errors are covered by reducer tests.
- README and atlas documentation were updated for the new action.
- `go test ./cmd/internal/couchtty` and its race suite pass.
- Focused core creation/admission tests pass.

2. Critical findings:

- `bootstrap-contract`: `bootstrap.sh:6-16` replaces the existing transitive peer-cloning/bootstrap contract with a Homebrew-dependent `weave` install. In a clean pair checkout, `scripts/run-merge-checks.sh` still points to `../../ariadne/...`, but the new bootstrap no longer obtains that peer; the workflow then invokes the dangling path at `.github/workflows/merge-check.yml:38-44`. Restore peer preparation or deliver the complete gateway migration, with a clean-checkout CI regression test. This is also an `ARCH-CONSTRAINTS` and `ARCH-MOCK` failure: the workflow now assumes Homebrew/network/formula availability without a bounded portable path.

3. Important findings:

None for the #313 slot-action implementation.

4. Minor findings:

None.

5. Test coverage notes:

Couch TUI tests and race tests pass. Focused core tests pass. The bootstrap/clean-checkout path was not successfully verified; it requires rework and a regression test.

6. Architectural notes for upcoming work:

- ARCH-DRY: pass — existing root derivation and start-preview helpers are reused.
- ARCH-PURE: pass — menu behavior remains in the reducer; no new IO was added.
- ARCH-PURPOSE: pass for #313 — primary, numbered, exact-path, cancellation, and refusal cases are covered.
- ARCH-MOCK: flag — the unrelated CI/bootstrap change depends directly on Homebrew and a published weave formula without a portable fake or fallback.
- ARCH-CONSTRAINTS: flag — clean environments and missing network/package-manager availability are no longer supported.
- ARCH-SECURE: pass for the slot action; no new credential or untrusted-persistence boundary.
- ARCH-ORDER: pass — existing preview-generation and accepted-fingerprint transitions are preserved.
- ARCH-FUNERAL: pass — no new durable artifact family is introduced by #313.

7. Plan revision recommendations:

- Add a `## Revisions` entry removing the unrelated bootstrap/CI rewrite from issue #313’s delivered scope, or create a separate issue/plan that explicitly specifies the gateway migration and clean-checkout contract.

---

## Re-review — 2026-09-23T20:48:45-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 313 — Add a slot from the repository thread action menu |
| repo | pair |
| issue file | workshop/issues/000313-couch-add-slot-action.md |
| boundary | whole-issue close |
| milestone | — |
| window | c6a91fe2398275359e99aa1c14373edf1a7dadc1..5e9da6e13b1cfc1b3e96bb68c6b6530e84cdebf4 |
| command | sdlc close --issue 313 |
| reviewer | codex |
| timestamp | 2026-09-23T20:48:45-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The Add slot implementation is well-covered and matches #313, but the pinned boundary also contains substantial #315 fresh-launch protocol changes that violate #313’s “no new operation schema” scope. Split the unrelated work and rerun the #313 boundary review.

1. Strengths

- Exact root derivation and malformed-target normalization are reused (`menu_switchagent.go:235`).
- Start preview, fingerprint-bound submission, cancellation, and admission refusal are tested.
- README and atlas documentation were updated.
- Focused couchtty and race tests passed.

2. Critical findings

- `cmd/internal/launcher/launch_args_policy.go:44-55`, `cmd/internal/launcher/thread_claim.go:141-164`: the review range includes #315’s new `launch_nonce` profile schema and fresh claim-registration behavior, while #313 explicitly requires “No new operation schema, storage…” (`workshop/issues/000313-couch-add-slot-action.md:31-32`). This is undeclared cross-issue behavior in the boundary. Remove #315 commits from the #313 close window, or formally split/revise the issue and review the expanded scope separately.

3. Important findings

None for the #313 implementation itself.

4. Minor findings

None.

5. Test coverage notes

`go test ./cmd/internal/couchtty` and `go test -race ./cmd/internal/couchtty` passed. Broader launcher/core/full-suite runs were started but interrupted after prolonged execution; no failure was observed.

6. Architectural notes

- ARCH-DRY: Pass — existing root and preview helpers are reused.
- ARCH-PURE: Pass — menu behavior remains reducer-driven and testable without IO.
- ARCH-PURPOSE: Flagged for the boundary contamination described above; #313 itself fulfills its purpose.
- ARCH-MOCK: Pass for #313; no new external seam is introduced.
- ARCH-CONSTRAINTS: Pass; no new concurrency or unbounded work.
- ARCH-SECURE: Pass; unknown and malformed identities do not produce guessed roots.
- ARCH-ORDER: Pass; preview generation and late-result cancellation are covered.
- ARCH-FUNERAL: Pass; no new durable artifact family is introduced by #313.

7. Plan revision recommendations

Add a `## Revisions` entry documenting that #315 fresh-registration/nonce commits must be excluded from the #313 boundary, then rerun the close review on the corrected range.

```findings
findings:
  - id: new
    severity: Critical
    family: boundary-scope-strands-issue
    title: |
      The #313 boundary includes undeclared #315 launch-protocol changes
    detail: |
      The range adds launch_nonce profile schema and fresh claim-registration behavior in cmd/internal/launcher, contradicting #313's explicit “no new operation schema” constraint. Remove the unrelated commits or formally split and review the expanded scope.
```

---

## Re-review — 2026-09-23T20:54:52-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 313 — Add a slot from the repository thread action menu |
| repo | pair |
| issue file | workshop/issues/000313-couch-add-slot-action.md |
| boundary | whole-issue close |
| milestone | — |
| window | c6a91fe2398275359e99aa1c14373edf1a7dadc1..abd2032872d62cc95114e86e2e7895186c6b5ceb |
| command | sdlc close --issue 313 |
| reviewer | codex |
| timestamp | 2026-09-23T20:54:52-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The #313 menu behavior and #315 handshake fixes are implemented and focused regression tests pass. The pinned boundary also contains unrelated bootstrap, CI, Makefile, and generated-ignore changes from `aff72f82`, which are outside the documented delivery and block shipping until removed or separately reviewed.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The issue now explicitly includes #315, with a separate pinned SHIP review and acceptance contract covering reserved claims and launch nonces.
findings:
  - id: new
    severity: Critical
    family: boundary-scope-strands-issue
    title: |
      The pinned boundary still includes undeclared bootstrap and CI gateway changes
    detail: |
      `bootstrap.sh`, `.github/workflows/merge-check.yml`, `Makefile`, and `.gitignore` changed between the pinned base and head via `aff72f82`, while the issue explicitly says that gateway commit is outside this delivery. Remove/rebase those changes from this boundary, or formally split and review the expanded gateway scope.
```

### Strengths

- Add-slot derives exact roots through `menuRepositoryRoot` and preserves unknown-root refusal.
- The reducer tests cover primary, numbered, subdirectory, cancellation, preview refusal, and fingerprint-bound submission.
- Fresh claim registration validates scope/tag/state and preserves established-only resume behavior.
- Focused couchcore and launcher regression tests pass.

### Critical findings

See `boundary-scope-strands-issue` above.

### Important findings

None beyond the blocking scope issue.

### Minor findings

None.

### Test coverage notes

`go test` passed for focused couchcore nonce/registration tests, launcher claim/profile tests, and `couchtty`. The broader couchcore suite was interrupted after 136 seconds and is not considered passing. `git diff --check` passed.

### Architectural notes

- ARCH-DRY: Pass for the requested feature; shared root, preview, claim, and profile helpers are reused.
- ARCH-PURE: Pass; reducer and profile validation remain testable without IO.
- ARCH-PURPOSE: Flagged for the boundary’s unrelated gateway scope; the requested feature itself fulfills its purpose.
- ARCH-MOCK: Pass for #315’s filesystem/readiness seams; the unrelated bootstrap path introduces direct Homebrew/weave dependency work.
- ARCH-CONSTRAINTS: Pass for bounded launch/readiness paths; gateway runtime constraints are outside the declared issue.
- ARCH-SECURE: Pass; strict profile and exact claim identity validation reject malformed or mismatched input.
- ARCH-ORDER: Pass; reserved-to-established transition is explicit and resume remains read-only.
- ARCH-FUNERAL: Pass for the requested feature; no new durable artifact family is introduced.

### Plan revision recommendations

- Add a `## Revisions` entry only if the gateway changes are intentionally retained, documenting their separate scope, acceptance contract, tests, and review boundary.

---

## Re-review — 2026-09-23T21:29:04-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 313 — Add a slot from the repository thread action menu |
| repo | pair |
| issue file | workshop/issues/000313-couch-add-slot-action.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5dea65f9ba06e482a5c47571d0aae85677ad5a1d..5ff2416d70be27e3798d69322a6b19c21ddef816 |
| command | sdlc close --issue 313 |
| reviewer | codex |
| timestamp | 2026-09-23T21:29:04-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The #313 implementation and combined #315 runtime changes are behaviorally well covered, with README/atlas updates and exact-root safeguards. The boundary is blocked because the committed #315 review artifact still contains an unresolved REWORK finding describing superseded nonce-registration behavior; the combined delivery therefore lacks clean review evidence.

1. Strengths

- `menuRepositoryRoot` uses validated slot identity or scope-matched roots; unknown roots do not get guessed.
- Add-slot tests cover primary/subdirectory roots, numbered slots, exact-path disambiguation, cancellation, late previews, refusals, and fingerprint-bound submission.
- Fresh-slot launch uses the ordinary creation/registration path and validates arguments before mutation.
- README and `atlas/couch.md` document both user-facing surfaces.
- `go test ./cmd/internal/couchtty` passed; targeted fresh-slot core tests passed.

2. Critical findings

None.

3. Important findings

- `workshop/plans/000315-fresh-slot-registration-close-review.md:27` — the committed review artifact still records `RegisterFreshCouchThread` and nonce transport as delivered, while the active #315 Spec and implementation explicitly remove them. Its later re-review at line 91 records REWORK, and no subsequent correction exists in the pinned range. Resolve or replace this artifact and obtain a clean #315 review result before closing the combined delivery.

4. Minor findings

None.

5. Test coverage notes

The combined couchcore package run was stopped after extended silence; targeted production-boundary tests and the full couchtty package passed. Confidence remains high for the reviewed diff, but the broader core suite should be rerun after the review-artifact correction.

6. Architectural notes

- ARCH-DRY: Pass — ordinary creation and existing root/profile helpers are reused.
- ARCH-PURE: Pass — menu reduction and validation remain separated from IO.
- ARCH-PURPOSE: Pass — both #313 menu behavior and revised #315 ordinary fresh-launch behavior are implemented.
- ARCH-MOCK: Pass — integration coverage uses real claim storage and shared launcher seams.
- ARCH-CONSTRAINTS: Pass — no new unbounded workers, loops, or durable growth.
- ARCH-SECURE: Pass — malformed targets and launch arguments fail closed before mutation.
- ARCH-ORDER: Pass — existing claim, replacement, launch, acknowledgement, and registration transitions are reused.
- ARCH-FUNERAL: Pass — no new durable artifact family is introduced.

7. Plan revision recommendations

- Add a `## Revisions` entry to the #313 plan documenting that combined delivery cannot close while the #315 close-review artifact remains REWORK.
- Reconcile/archive the stale #315 review artifact so its final evidence matches the ordinary-registration implementation.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The #313 Spec explicitly incorporates #315, and the pinned code plus separate #315 issue/plan contain the revised ordinary-registration delivery.
  - id: BR-2
    disposition: addressed
    note: |
      The required bootstrap, CI gateway, Makefile, and gitignore paths have no diff in the pinned base-to-head range.
findings:
  - id: new
    severity: Important
    family: stale-boundary-review-artifact
    title: |
      Combined delivery retains unresolved and superseded #315 review evidence
    detail: |
      workshop/plans/000315-fresh-slot-registration-close-review.md:27 still claims RegisterFreshCouchThread and nonce transport, while the active #315 contract removes them; its later re-review records REWORK with no subsequent clean disposition in this range. Reconcile the artifact and obtain clean review evidence before closing #313.
```
