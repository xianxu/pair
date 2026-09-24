# Boundary Review — pair#316 (whole-issue close)

> Review history: the initial SHIP applies to the original branch window only. The re-review below supersedes it for the integrated branch (`6bc2340e..14c10186`); its FIX-THEN-SHIP documentation dispositions are recorded in the issue Log. Earlier verdicts remain historical evidence.

| field | value |
|-------|-------|
| issue | 316 — Couch Alt+n refuses a fresh thread for up to 60s after its first round |
| repo | pair |
| issue file | workshop/issues/000316-couch-altn-binding-latency.md |
| boundary | whole-issue close |
| milestone | — |
| window | aff72f8223230918285033b904a64994d06a1aaa..60add0fa79fea5382642bcf3caf6bdfbdd6c18b6 |
| command | sdlc close --issue 316 |
| reviewer | codex |
| timestamp | 2026-09-23T23:12:47-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The change matches issue #316: cadence follows observed Pair-log sends, slow waits wake within `ActivePoll`, and the pure policy is table-tested. No blocking findings. The touched package passes; the full suite was interrupted in unrelated `couchcore` tests after 147s without reporting a failure.

1. Strengths

- `scanDelay` cleanly separates pure cadence policy from IO.
- Regression test uses an off-grid send timestamp, preventing fake-clock false positives.
- Slow polling is sliced and wakes on Pair-log mtime changes.
- `atlas/session-identity.md` documents the new behavior.
- `git diff --check` passes.

2. Critical findings

None.

3. Important findings

None.

4. Minor findings

None.

5. Test coverage notes

- `go test ./cmd/internal/sessionwatch -count=1` passed.
- `go test ./... -count=1` reached `couchcore` and was interrupted after 147s; no failure was reported before interruption.

6. Architectural notes

- ARCH-DRY: Pass. Cadence behavior is centralized in `scanDelay`/`waitForScan`.
- ARCH-PURE: Pass. Policy is pure and directly table-tested; runtime polling remains in the watcher seam.
- ARCH-PURPOSE: Pass. Startup, no-send slow polling, active-send cadence, and slow-wait wake-up behavior are covered.

7. Plan revision recommendations

None.

```findings
# No prior or new findings.
```

---

## Re-review — 2026-09-23T23:17:47-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 316 — Couch Alt+n refuses a fresh thread for up to 60s after its first round |
| repo | pair |
| issue file | workshop/issues/000316-couch-altn-binding-latency.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6bc2340e2ab5afa9a205dc24f7b2883819f7e935..14c1018627788a6e5fb4d2d10aea797ca988671f |
| command | sdlc close --issue 316 |
| reviewer | codex |
| timestamp | 2026-09-23T23:17:47-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The sessionwatch change fulfills the latency purpose, separates cadence policy cleanly, and has passing focused and race tests. Before closing, remove unrelated project-scope changes and regenerate the stale boundary-review artifact against the pinned window.

1. Strengths

- `scanDelay` is pure and table-tested.
- Regression test uses an off-grid send timestamp.
- Slow waits wake on Pair-log changes.
- Atlas documentation covers the new cadence behavior.
- `go test` and `go test -race` pass for `sessionwatch`.

2. Critical findings

None.

3. Important findings

- `workshop/projects/couch-slots-v2.md:394,967-978` adds unrelated ariadne#247 dependency-refresh scope to the #316 window. Remove it or document why #316 owns this project change. Family enumeration: these are the only instances of unrelated project-scope additions in this window.
- `workshop/plans/000316-couch-altn-binding-latency-close-review.md:10-23` records a different review window (`aff72f..60add0f`) and an earlier SHIP verdict, not the pinned `6bc2340..14c10186` range. Regenerate the review artifact after this review. Family enumeration: this is the only stale boundary-review artifact in the window.

4. Minor findings

None.

5. Test coverage notes

`go test ./cmd/internal/sessionwatch -count=1` and `go test -race ./cmd/internal/sessionwatch -count=1` pass. The full-suite result is recorded in the issue but was not independently rerun during this read-only review.

6. Architectural notes

- ARCH-DRY: Pass — cadence logic is centralized.
- ARCH-PURE: Pass — policy is pure and IO remains in the runtime seam.
- ARCH-PURPOSE: Pass — startup, no-send, active-send, and slow-wait wake-up behavior are addressed.

7. Plan revision recommendations

None; the implementation plan matches the code.
