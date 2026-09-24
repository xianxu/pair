# Boundary Review — pair#315 (whole-issue close)

> Historical review record: the initial SHIP below applies only to the superseded implementation at `0879c270`. Its `RegisterFreshCouchThread` and nonce transport were removed by `5ff2416d`. The current implementation uses ordinary new-conversation creation. Later reviews below supersede the initial verdict; this document preserves review history, not a current feature specification.

| field | value |
|-------|-------|
| issue | 315 — Allow fresh slot launches to establish their reserved conversation |
| repo | pair |
| issue file | workshop/issues/000315-fresh-slot-registration.md |
| boundary | whole-issue close |
| milestone | — |
| window | c6a91fe2398275359e99aa1c14373edf1a7dadc1..0879c2701a6a6798e99774580f6a125855473e1a |
| command | sdlc close --issue 315 |
| reviewer | codex |
| timestamp | 2026-09-23T20:43:36-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The diff fulfills #315: reserved claims establish through the real launcher path, established claims remain valid, invalid/resume claims stay read-only, and fresh readiness carries the exact launch nonce. No Critical, Important, or Minor findings.

1. Strengths

- `RegisterFreshCouchThread` validates exact scope/tag/state and atomically promotes reserved claims.
- Launcher tests exercise real claim files across reserved, established, malformed, missing, mismatched, unowned, and resume cases.
- Fresh nonce transport is validated through profile decoding and the real readiness reader.
- README and `atlas/couch.md` document the new behavior.

2. Critical findings

None.

3. Important findings

None.

4. Minor findings

None.

5. Test coverage notes

Focused launcher tests and race tests passed. The focused couchcore fresh-slot/nonce tests passed. The broader couchcore suite was interrupted after extended execution, so confidence is medium rather than high.

6. Architectural notes

- ARCH-DRY: Pass; shared claim and profile validation paths are reused.
- ARCH-PURE: Pass; policy validation remains pure and filesystem access is behind runtime seams.
- ARCH-PURPOSE: Pass; registration and nonce transport cover all fresh-launch paths.
- ARCH-MOCK: Pass; real filesystem claims and readiness-reader boundaries are exercised.
- ARCH-CONSTRAINTS: Pass; registration waits and readiness use bounded timeouts.
- ARCH-SECURE: Pass; strict profile decoding and exact identity checks reject malformed or mismatched inputs.
- ARCH-ORDER: Pass; reserved → established is an explicit atomic transition, with stale nonce rejection.
- ARCH-FUNERAL: Pass; no new unbounded artifact family is introduced; claim lifecycle follows existing durable-thread ownership.

7. Plan revision recommendations

None.

```findings
findings: []
```

---

## Re-review — 2026-09-23T21:26:42-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 315 — Allow fresh slot launches to establish their reserved conversation |
| repo | pair |
| issue file | workshop/issues/000315-fresh-slot-registration.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5dea65f9ba06e482a5c47571d0aae85677ad5a1d..5ff2416d70be27e3798d69322a6b19c21ddef816 |
| command | sdlc close --issue 315 |
| reviewer | codex |
| timestamp | 2026-09-23T21:26:42-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation satisfies the active #315 Spec and Plan, and targeted production-boundary tests pass. However, the repository retains a stale boundary-review artifact describing the superseded nonce/registration design, which conflicts with the final implementation and tracker state.

1. Strengths

- Fresh slots use ordinary Pair registration through `StartFreshSlot`.
- Argument validation occurs before claim or metadata replacement.
- Real launcher/claim-storage regression passes.
- README and `atlas/couch.md` document the final behavior.
- Baseline restoration check is empty.

2. Critical findings

None.

3. Important findings

- `workshop/plans/000315-fresh-slot-registration-close-review.md:27` documents the superseded `RegisterFreshCouchThread` and nonce design, while the active issue explicitly removes both. Replace or clearly archive/mark this stale review artifact before closing so tracker evidence does not claim behavior the code no longer implements.

4. Minor findings

None.

5. Test coverage notes

Passed:

- `TestSpawnComposesProductionPairRegistrationBoundary`
- Fresh-slot metadata/preference/ownership guards
- `TestAddSlot*`
- `git diff --check`

6. Architectural notes

- ARCH-DRY: Pass — fresh slots reuse ordinary creation.
- ARCH-PURE: Pass — validation remains separated from launch IO.
- ARCH-PURPOSE: Pass — the fresh-slot path no longer requires established claims or special nonce readiness.
- ARCH-MOCK: Pass — the composed test uses real claim storage and launcher boundaries.
- ARCH-CONSTRAINTS: Pass — existing bounded startup and registration behavior is reused.
- ARCH-SECURE: Pass — existing profile and claim validation remain authoritative.
- ARCH-ORDER: Pass — existing start/claim/replace/registration transitions are reused.
- ARCH-FUNERAL: Pass — no new durable artifact family is introduced.

7. Plan revision recommendations

None.

```findings
findings:
  - id: new
    severity: Important
    family: stale-boundary-review-artifact
    title: |
      Close-review artifact describes superseded nonce registration behavior
    detail: |
      workshop/plans/000315-fresh-slot-registration-close-review.md:27 claims RegisterFreshCouchThread and fresh nonce transport are delivered, contradicting the active Spec, Plan, and implementation that remove them. Replace or explicitly archive/mark the artifact before closing.
```

## Revisions

- 2026-09-23: Address BR-1 by explicitly marking the initial review as historical and superseded. The final implementation removes both special registration and nonce transport; current behavior is specified by the active issue and atlas. Review text is retained verbatim as historical evidence.

---

## Re-review — 2026-09-23T21:35:34-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 315 — Allow fresh slot launches to establish their reserved conversation |
| repo | pair |
| issue file | workshop/issues/000315-fresh-slot-registration.md |
| boundary | whole-issue close |
| milestone | — |
| window | 5dea65f9ba06e482a5c47571d0aae85677ad5a1d..2dc57b7762bfc8a6eacaa7d5ee3e42f58a3c45e6 |
| command | sdlc close --issue 315 |
| reviewer | codex |
| timestamp | 2026-09-23T21:35:34-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range satisfies the active Spec and Plan. Prior finding BR-1 is addressed, production-boundary tests pass, documentation is updated, and no new blocking findings remain.

1. Strengths

- Fresh slots use ordinary Pair registration via `StartFreshSlot`.
- Argument validation precedes claim and metadata replacement.
- Real launcher/claim-storage regression covers ordinary and fresh-slot paths.
- Add-slot UI behavior has focused coverage, including malformed and cancelled flows.
- README and atlas documentation reflect the delivered behavior.

2. Critical findings

None.

3. Important findings

None.

4. Minor findings

None.

5. Test coverage notes

Passed:

- Full `couchcore`, `couchtty`, and `launcher` suites.
- Focused fresh-slot and add-slot tests.
- `git diff --check`.

6. Architectural notes

- ARCH-DRY: Pass — ordinary creation is reused.
- ARCH-PURE: Pass — validation remains separated from launch IO.
- ARCH-PURPOSE: Pass — fresh slots no longer require established claims or nonce readiness.
- ARCH-MOCK: Pass — production launcher and real claim storage are exercised.
- ARCH-CONSTRAINTS: Pass — existing bounded startup behavior is retained.
- ARCH-SECURE: Pass — existing strict profile and claim validation remain authoritative.
- ARCH-ORDER: Pass — existing transaction and cleanup ordering is reused.
- ARCH-FUNERAL: Pass — no new durable artifact family is introduced.

7. Plan revision recommendations

None.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The close-review artifact now explicitly marks the nonce/registration review as historical and superseded by the ordinary new-conversation design.
findings: []
```
