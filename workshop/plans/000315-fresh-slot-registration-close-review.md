# Boundary Review — pair#315 (whole-issue close)

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
