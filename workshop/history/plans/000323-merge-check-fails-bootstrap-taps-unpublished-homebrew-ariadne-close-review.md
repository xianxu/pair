# Boundary Review — pair#323 (whole-issue close)

| field | value |
|-------|-------|
| issue | 323 — merge-check fails: bootstrap taps unpublished homebrew-ariadne |
| repo | pair |
| issue file | workshop/issues/000323-merge-check-fails-bootstrap-taps-unpublished-homebrew-ariadne.md |
| boundary | whole-issue close |
| milestone | — |
| window | 9ddb46ba1672591735932a20193c6ad18cc40f8b..a5167cf17eaa23101836522d2f71742f85757996 |
| command | sdlc close --issue 323 |
| reviewer | claude |
| timestamp | 2026-09-29T16:05:40-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

This window contains no code. The only file changed is one issue file (`workshop/issues/000323-…md`, +12/−3): it ticks the last three Plan items and adds a 2026-09-29 Log entry saying the issue was resolved upstream. I checked each claim I could against the repo. Commits `9becfbc7` (09-25, "build weave from source while tap is unpublished") and `6f62a7ec` (09-28, removes that fallback) exist, both change only `.github/workflows/merge-check.yml`, and both are ancestors of HEAD. The current `bootstrap.sh:13` installs `xianxu/ariadne/weave` from the published tap, and `merge-check.yml` no longer contains the source-build fallback. That matches the Log's account: ariadne#250 added the fallback, and ariadne#241 published the tap and removed it. Spec option 2 (a local fallback in pair) was correctly ruled out, because both files are seeded from ariadne. So the Done-when clause about removing a fallback is covered: ariadne#241 owned the removal. I could not check the cited GitHub run IDs (36635174446, #338, #340) from this read-only, sandboxed session. The first Done-when bullet depends on that evidence, and the implementor's `--verified` string should carry it.

1. **Strengths**
   - The root cause was fixed in the upstream files that pair's copies are generated from, not in pair's copies, which the next `weave compile` would have overwritten. Log entry 2026-09-25 records why.
   - The full chain can be traced: onset commit `aff72f82`, then ariadne#250's fallback (`9becfbc7`), then removal after the tap was published (`6f62a7ec`).
   - The Log separates the later red runs (#338, #340), which failed in "Run merge-checks over the PR range", from this issue's failure in "Prepare dependencies". That correctly marks them as out of scope.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - Plan item 4 says "this issue's PR runs `merge-check` green", but the evidence is a run on #346's PR, not this issue's. The substance holds, since any PR passing "Prepare dependencies" proves the fix. Either reword the item or cite this PR's run once it exists.
   - The Spec calls a clearer bootstrap error message for a missing tap "worth proposing upstream". Publishing the tap made that moot. One Log line saying it was dropped would close that thread.

5. **Test coverage:** no code changed, so no tests are expected. The behavioral evidence is the CI run history.

6. **Architecture**
   - ARCH-DRY: pass. Fixing the seed upstream avoided a second copy of the fallback in pair.
   - ARCH-PURE: N/A, no code.
   - ARCH-PURPOSE: pass. The fix reached every repo that consumes the seed through ariadne, not just pair; the Log names parley.nvim and tools as later consumers.
   - ARCH-MOCK: N/A, no new external calls in the window.
   - ARCH-CONSTRAINTS: N/A.
   - ARCH-SECURE: pass. No credentials. The misleading "could not read Username" error was diagnosed as a missing tap, not an auth problem.
   - ARCH-ORDER: N/A. Nothing here holds state between events.
   - ARCH-FUNERAL: pass. The temporary fallback had an owner for its removal (ariadne#241), and it has been removed (`6f62a7ec`).

7. **Plan revisions:** none needed beyond the optional rewording of item 4.

```findings
findings:
  - id: new
    severity: Minor
    family: verification-evidence-matches-claim
    title: |
      Plan item 4 claims this issue's PR ran green; evidence cites #346's run
    detail: |
      Reword to "a pair PR runs merge-check green past Prepare dependencies" or cite this PR's run id once it exists.
  - id: new
    severity: Minor
    family: spec-thread-closure
    title: |
      Spec's proposed clearer missing-tap bootstrap message is left unaddressed
    detail: |
      Moot now that the tap is published; add one Log line saying so so the thread reads as closed.
```
