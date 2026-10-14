# Boundary Review — pair#543 (whole-issue close)

| field | value |
|-------|-------|
| issue | 543 — couch: log supervision facts for offline baseline evaluation |
| repo | pair |
| issue file | workshop/issues/000543-couch-supervision-fact-log.md |
| boundary | whole-issue close |
| milestone | — |
| window | a7524cf9242888cd3346d9841d75959fa82948db..db4c743b8e6d8ceddd25705ecfe52130ea51559a |
| command | sdlc close --issue 543 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T23:08:04-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation covers all five observation classes, isolates diagnostic I/O, and updates README and atlas. One reproducible labeling bug blocks shipping: incomplete evidence at the inclusive ten-minute boundary can produce a false negative.

1. **Strengths**

   - Bounded admission keeps disk writes outside producer callbacks; controlled sinks exercise queue saturation and late-open shutdown.
   - Session-write telemetry uses authenticated owner checks and a single timeout-bounded call.
   - Offline parsing validates identities, numeric ranges, schema fields, and input limits without echoing malformed content.
   - Retention reuses the existing diagnostic owner; documentation distinguishes generation limits from total storage limits.

2. **Critical findings**

```findings
findings:
  - id: new
    severity: Critical
    family: negative-label-requires-complete-coverage
    title: |
      Inclusive horizon is declared complete before all qualifying writes are excluded.
    detail: |
      scripts/supervision-labels/labels.py:186 accepts coverage ending exactly at delivery+600000 ms. Reproduced with delivery(seq=1, ms=0), snapshot(seq=2, ms=600000), snapshot(seq=4, ms=600001): the result is false despite missing seq=3. Restoring seq=3 as a same-subject write at ms=600000 changes it to true. Since timestamps can tie, require contiguous evidence strictly beyond the inclusive horizon before emitting false. Add regressions for both a truncated boundary tail and a boundary sequence gap. ARCH-ORDER, ARCH-PURPOSE.
```

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Passed: all 14 Python tests; Go packages `couchmessage`, `diagnosticlog`, and `sessionsync`.
   - The additional boundary reproductions above expose a case existing tests miss; one existing test explicitly expects false when coverage ends exactly at the horizon.
   - `couchcmd` could not compile because generated `runtimebundle/assets/runtime/files` is absent. `artifactpath` also failed checks dependent on that tree. These are checkout prerequisites, not established regressions.
   - Full-suite success was not independently verified. No repository files were changed.

6. **Architectural notes**

   - **ARCH-DRY — pass:** reuses classification, diagnostic storage, and retention.
   - **ARCH-PURE — pass:** label derivation and projections are separated from I/O; concept inventory matches implementation locations.
   - **ARCH-PURPOSE — flag:** unsupported negatives undermine reliable offline evaluation.
   - **ARCH-MOCK — pass:** inspected tests use portable storage, controlled sinks, and private socket seams.
   - **ARCH-CONSTRAINTS — pass:** explicit queue, row, actor, and input-batch bounds.
   - **ARCH-SECURE — pass:** authenticated reporting, opaque identities, and validated offline input.
   - **ARCH-ORDER — flag:** equal timestamps leave the inclusive horizon unresolved; recorder lifecycle ordering otherwise has controlled tests.
   - **ARCH-FUNERAL — pass:** writer shutdown joins its worker; registered diagnostic retention owns historical segments.

7. **Plan revision recommendations**

   Add a `## Revisions` entry specifying that negative coverage must advance **strictly beyond** the inclusive ten-minute endpoint, with equal-timestamp tail and sequence-gap regression cases.

---

## Re-review — 2026-10-13T23:13:01-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 543 — couch: log supervision facts for offline baseline evaluation |
| repo | pair |
| issue file | workshop/issues/000543-couch-supervision-fact-log.md |
| boundary | whole-issue close |
| milestone | — |
| window | a7524cf9242888cd3346d9841d75959fa82948db..7b1264211753fd268f9c6c34a1d7685f89cc8cde |
| command | sdlc close --issue 543 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T23:13:01-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation delivers the five fact classes, bounded best-effort recording, and offline labeling described by the revised plan. BR-1 is addressed with meaningful regression evidence. No new blocking findings emerged; README and atlas updates cover the new surface and its limitations.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      labels.py:187 requires contiguous evidence strictly beyond the inclusive horizon. Both boundary regressions in test_labels.py:60 and :67 pass at HEAD and fail when the former >= condition is restored in memory. The plan revision at lines 184–191 documents the corrected rule.
```

1. **Strengths**

   - Recorder admission isolates disk I/O behind a bounded queue; controlled sinks test saturation, failures, and late-open shutdown.
   - Session-write reporting preserves the successful write result, authenticates the owner, and makes one timeout-bounded attempt.
   - Offline parsing validates schema, identities, numeric ranges, and batch limits without echoing malformed content.
   - Diagnostic storage and retention are reused; documentation accurately distinguishes generation caps from total storage limits.

2. **Critical findings**

   None.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - All 16 Python tests passed.
   - Reverting only BR-1’s comparison in memory made both new regression tests fail.
   - All five affected Go packages passed: `couchcmd`, `couchmessage`, `diagnosticlog`, `sessionsync`, and `artifactpath`.
   - Initial Go checks encountered missing generated runtime assets. Generating the prerequisite in a disposable pinned-head clone resolved those failures; the clone was removed afterward.
   - Full `make -k test` was not independently rerun. The tracker records its earlier failures and isolated reruns; this review does not assert a clean full-suite result.
   - The review checkout remained unchanged.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — pass:** existing classification, diagnostic storage, and retention are reused.
   - **ARCH-PURE — pass:** pure labeling, schema, lifecycle, and projection entities remain separate from I/O; the concept inventory matches their locations.
   - **ARCH-PURPOSE — pass:** all five observation classes and offline labels are implemented; BR-1 fixes the general inclusive-horizon rule.
   - **ARCH-MOCK — pass:** controlled sinks, private sockets, and temporary storage exercise production seams without adding an external service dependency.
   - **ARCH-CONSTRAINTS — pass:** explicit queue, row, actor, batch, and telemetry-timeout bounds are implemented and tested.
   - **ARCH-SECURE — pass:** authenticated reporting, salted identities, and validated input preserve the stated trust boundaries.
   - **ARCH-ORDER — pass:** lifecycle transitions and controlled failure ordering preserve terminal closure; negative labels now require sufficient temporal evidence.
   - **ARCH-FUNERAL — pass:** shutdown joins the writer, and registered diagnostic retention owns historical segments.

7. **Plan revision recommendations**

   None. The existing BR-1 revision describes the implemented correction.
