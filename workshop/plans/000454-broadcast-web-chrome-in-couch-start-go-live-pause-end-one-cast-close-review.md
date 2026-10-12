# Boundary Review — pair#454 (whole-issue close)

| field | value |
|-------|-------|
| issue | 454 — broadcast-web: Chrome in Couch — start, go live, pause, end; one cast |
| repo | pair |
| issue file | workshop/issues/000454-broadcast-web-chrome-in-couch-start-go-live-pause-end-one-cast.md |
| boundary | whole-issue close |
| milestone | — |
| window | be93aa1dc5ad723169e489ed265c38f1295ef18a..d6e06046ceb2d3ee2dd5e56d3fb5717105728e14 |
| command | sdlc close --issue 454 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-11T23:18:41-07:00 |
| verdict | SHIP |

## Review

Checking the three open findings (BR-7, BR-17, BR-18) at head, then running the suites.

Generating the gitignored runtime bundle so the couch packages can build, then testing them.

Couch packages pass. Now reverting the BR-7 tie-break to check whether any test notices.

```verdict
verdict: SHIP
confidence: high
```

**Summary.** The code hasn't changed since round 7 passed. The only commits since then are `e7814c3d`, which touches only workshop/review files, and the merge `d6e06046`, which brings in unrelated workshop issues and plans plus a 5-line `lessons.md` edit. I re-ran the suites at head, all with `-race`. `webcast`, `broadcast` and `cdp` pass. `couchtty`, `couchkeys` and `couchcmd` also pass once the gitignored runtime bundle is generated with `Makefile.local:246`; that file is missing from a fresh checkout, which is not a code defect. I didn't re-run `artifactpath` after generating the bundle. Before that, its only failures were caused by the missing bundle (a build failure in `TestCompositeBindingsIsolateGoShellNeovimAndBothLayouts`, and "stale" known-violation entries pointing at the absent generated files). All three open findings are still open, but they are Minor and don't block. I found nothing new. README and atlas are both updated (`atlas/webcast.md`, plus the `atlas/index.md` entry). The plan's unchecked boxes are the template's per-step TDD checklist, and the issue has no open items.

1. **Strengths**
   - There is one release path in the Host, `releaseIfUnwantedLocked`, and abandoned opens can't be joined (`host.go:107,165`). The BR-14 and BR-16 regression tests go red when the fix is reverted, as confirmed in earlier rounds.
   - Chrome end causes are taken only from `cdp.ErrProtocol` and not from errors produced by our own teardown, which fixes BR-12 at the rule level rather than for one instance.
   - `TestTrackerInvariants` (`tracker_test.go:123`) states its invariants separately from the implementation and checks them over 500 generated sequences.
   - `TestStartProcessReapsDespiteInheritedOutput` pins `WaitDelay`.

2. **Critical:** none.
3. **Important:** none.
4. **Minor:** the three open findings carried forward below (BR-7, BR-17, BR-18).
5. **Test coverage**
   - BR-7: I replaced the id tie-break in `tracker.go:124-125` with `false` and ran `go test -count=20 ./cmd/internal/webcast/`. Everything stayed green, so no test pins the deterministic tie-break. Suggested fix: add a determinism invariant to `TestTrackerInvariants`, so that `Active()` from a cloned input sequence (or from repeated calls) is stable when 3+ pages are hidden and have equal `activated` values.
   - BR-17 and BR-18 are unchanged: `host_test.go:260-266` still sets `h.waiting` and calls `abandonWait` directly, and `lateCancelTunnel` and `failingTunnel` are still local to the test file.
6. **Architecture**
   - **ARCH-DRY:** pass. `OwnsToken` is shared.
   - **ARCH-PURE:** pass. Tracker and translate are pure, and Chrome goes through the process seam.
   - **ARCH-PURPOSE:** pass.
   - **ARCH-MOCK:** flag (BR-18). The tunnel behaviours that tests rely on aren't on the shared `FakeTunnel`.
   - **ARCH-CONSTRAINTS:** pass. The event queue is bounded and `closeWait` and `WaitDelay` are set.
   - **ARCH-SECURE:** pass. Tokens are compared in constant time and frame size is capped.
   - **ARCH-ORDER:** flag (BR-17). The Host sequence test reaches its state by setting fields directly instead of through an ordering seam.
   - **ARCH-FUNERAL:** pass. The profile lives under the store, and the Host closes when nobody wants it.
7. **Plan revisions:** none needed beyond the existing "as built" entries.

```findings
dispose:
  - id: BR-7
    disposition: not-addressed
    note: |
      Tie-break still untested: replacing it with false at tracker.go:124-125 leaves go test -count=20 ./cmd/internal/webcast green.
  - id: BR-17
    disposition: not-addressed
    note: |
      host_test.go:260-266 still sets h.waiting and calls abandonWait directly; no code change since round 7.
  - id: BR-18
    disposition: not-addressed
    note: |
      failingTunnel (host_test.go:123) and lateCancelTunnel (:294) are still test-local; FakeTunnel has only OpenDelay.
```
