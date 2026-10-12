# Boundary Review — pair#458 (whole-issue close)

| field | value |
|-------|-------|
| issue | 458 — broadcast-web: recorder — CDP DOM stream per frame plus isolated-world observer |
| repo | pair |
| issue file | workshop/issues/000458-broadcast-web-recorder-cdp-dom-stream-per-frame-plus-isolated-world-observer.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8a5e1c8626b7f2557391fa933007c9da6e95e9c5..1f2b5b5d61fbd39465827b527eaf37ba51079d73 |
| command | sdlc close --issue 458 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-12T03:04:31-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Disposing the prior open findings now; BR-1 is fixed, BR-2 is still open. Next I'll check BR-3, 14, 21, 22, 30, 31 and 34.

BR-22 is confirmed fixed: reverting the order makes the test fail. BR-14 is still open. Next I'm checking BR-30 and BR-34 in the fidelity tool.

Prior findings are all disposed. I'm now running the webcast test suite and reviewing the newest `Emit` input.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The issue delivers what it set out to do. The recorder is split into a pure mirror and a thin IO shell, it is wired into the Chrome loop and the Cast, and M1, M2 and M3 are all in place. The packages I ran pass: `webcast/...` and `cdp`. The only red run was `artifactpath`: those failures come from runtimebundle assets that are missing from this checkout, and nothing in them points at this diff. Of the nine open findings, four are now fixed: BR-1, BR-3, BR-21 and BR-22. I reverted BR-22's fix and the test failed (`div 57 id 36 -> 116`), so that fix is pinned. Five stay open, all Minor. BR-14 and BR-31 have code fixes that no test pins: I reverted them and the suite stayed green. BR-30 and BR-34 are fixed only in the fidelity tool's Chrome-driving code, and no test covers that code. BR-2 is unchanged. The newest commit adds `recorder.Emit`, which accepts any record kind; I've raised that as one new Minor. None of this blocks the gate.

**Strengths**
- `msession.bury` (`mirror.go:218`) is documented and run after the update at both places that replace a subtree. `TestGraveyardKeepsLiveIDsAndBoundsTheMap` now checks every child, so reordering the setChildNodes call fails the test.
- The queue's two bounds are stated once, at `Options.MaxQueue` (`recorder.go:72-76`), and `Send` refers to them. Every input goes through `enqueue`.
- `Emit` (`recorder.go:492`) passes through the reducer: a refused record is counted, and the test pins both the drop before go-live and the refused `note-state`.
- The fidelity replay now sizes itself from `frame.scale` (`run.go:368-374`), and the atlas states that `frame.scale` is the one source for the CSS-px mapping at every DPR.

**Critical:** none.

**Important:** none.

**Minor**
- `recorder.go:492`: `Emit` accepts any `rec.Record`. That includes structural kinds (`seg`, `doc`, `kids`, `rm`, `attr`). If a caller emits one, the mirror's reducer state and its CDP tree stop matching, and the recorder's own later records are refused. Fix: allowlist the kinds that may come from outside (the planned #467 ctl/injected actions, note, note-state, stroke) and count the rest as refused (ARCH-SECURE, ARCH-ORDER).
- BR-14: add a Resnapshot and a setChildNodes step that each drop more than `graveSize` nodes, then assert `len(ids) <= live + graveSize`.
- BR-31: test that fills `assetQ` and then checks a later sighting of the same URL is fetched again.
- BR-34 and BR-30: move the scale-to-LowRes factor into a pure function and test it at 0.5 and 0.33. The scroll-finding script needs a fixture covering a srcdoc frame and a shadow root (it runs only under `FIDELITY_CHROME`).
- BR-2: record the hover decision in `query.UserAction`.

**Test coverage notes**
- I reverted fixes to check that tests catch them. BR-22: the test goes red. BR-14 (both `bury` calls removed) and BR-31 (`delete(r.owned, …)` removed): the suite stays green.
- `TestRecorderEmit` covers dropped, written in order, and refused-and-counted. It doesn't cover an `Emit` arriving while paused.

**Architecture**
- **ARCH-DRY:** pass.
- **ARCH-PURE:** pass for the mirror. Minor gap: the fidelity tool's scale arithmetic is inline in `shoot()`, which only runs against a live Chrome.
- **ARCH-PURPOSE:** pass.
- **ARCH-MOCK:** pass (stateful fake CDP; live checks are opt-in).
- **ARCH-CONSTRAINTS:** pass (two queue bounds; side queues drop and count).
- **ARCH-SECURE:** flag. `Emit` is a new way in for records the recorder didn't make, and it has no per-kind allowlist (the new Minor above). The password rules are enforced by the reducer.
- **ARCH-ORDER:** pass. One writer, ordered by the queue; `Emit` goes through the same queue.
- **ARCH-FUNERAL:** pass. Recording dirs are removed by the operator, as documented. The goroutine ends with Chrome, and the fidelity tool's profile dir is cleaned up.

**Notes for upcoming work:** #467 and #470 should send `Emit` only the kinds they own. Turning that into an allowlist now keeps the mirror's view of the page from silently diverging later.

**Plan revisions:** none. The plan's Revisions entries match the code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      reduce.go:327 now guards with n.N == "input" before withAttr builds its copy.
  - id: BR-2
    disposition: not-addressed
    note: |
      query/surface.go:43 is unchanged and no decision on hover is recorded in the plan, issue or atlas.
  - id: BR-3
    disposition: addressed
    note: |
      format_v1_additions_test.go:107-125 now checks the errors from Put, PutFrame, Close and WriteFile.
  - id: BR-14
    disposition: not-addressed
    note: |
      Replacing both s.bury(old) calls (mirror.go:603 and 789) leaves the suite green; no step drops more than graveSize nodes via Resnapshot or setChildNodes.
  - id: BR-21
    disposition: addressed
    note: |
      recorder.go:72-76 states both bounds; Send at line 237 refers to it.
  - id: BR-22
    disposition: addressed
    note: |
      Burying before index in setChildNodes fails TestGraveyardKeepsLiveIDsAndBoundsTheMap (child 57 id 36 -> 116).
  - id: BR-30
    disposition: not-addressed
    note: |
      The find() over srcdoc frames and shadow roots exists (run.go:326-351), but no test exercises it; TestFidelityFixtures is opt-in and has no fixture for these cases.
  - id: BR-31
    disposition: not-addressed
    note: |
      Removing delete(r.owned, j.url) at side.go:78 leaves the recorder and fidelity suites green.
  - id: BR-34
    disposition: not-addressed
    note: |
      run.go:368-374 and webcast-format.md:188 are corrected, but the scale-to-LowRes factor is inline in shoot() with no test; extract it as a pure function and test it at 0.5 and 0.33.
findings:
  - id: new
    severity: Minor
    family: consumer-kind-classification
    title: |
      recorder.Emit accepts any record kind, so a structural record from outside desyncs the mirror from its reducer
    detail: |
      2nd finding in family consumer-kind-classification. The rule: every place that consumes record kinds (UserAction, the kinds Emit accepts) classifies each kind explicitly, so a new kind forces a decision. recorder.go:492 applies any rec.Record. An external seg, doc, kids, rm or attr changes m.st but not the CDP tree, and the mirror's later records are then refused. Fix: allowlist note, note-state, stroke and the #467 kinds, and count everything else as refused, with a test.
```
