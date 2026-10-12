# Boundary Review — pair#482 (whole-issue close)

| field | value |
|-------|-------|
| issue | 482 — broadcast-web: fidelity flake on rec-blog frame 0 — a web font bound 1 ms before the screenshot moment |
| repo | pair |
| issue file | workshop/issues/000482-broadcast-web-fidelity-flake-on-rec-blog-frame-0-a-web-font-bound-1-ms-before-the-screenshot-moment.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4020e12cbed39e8b774aa9319d58b20edfebd036..e3615fff8d824914a6339db22ae989b3c33ecbda |
| command | sdlc close --issue 482 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T07:14:39-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Tests pass at head. Next I'll run the live Chrome test, then revert the fix and run it again to check that it fails without the fix.

Chrome crashes inside the sandbox (an environment problem, not a code problem). The JS order test does go red when the fix is reverted. I'll try the live test once more outside the sandbox.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The fix is small, targeted, and goes after the real cause. Before the change, `replayAt` captured right after the real-time window and only waited for a held paint's stylesheets. Now it pauses the animations at t, calls the existing `settle()` (the renderer's `ready()`, which waits up to 5 s for sheets, fonts and images, then waits for a paint), and pauses anything the loads started. It reuses `settle()` instead of adding a second readiness check, and it covers images too: the second slot's crops showed the missing avatar, so a font-only wait would not have been enough. The JS order test passes at head. When I reverted `replayer.js` to base it failed (6 pass / 1 fail), so it pins the change. I could not run the live Chrome test (`TestLiveReplayAtLoadsAssetsBoundJustBefore`): Chrome failed with `cdp: read: EOF` at head as well as with the fix reverted, both inside and outside the sandbox. That is an environment problem, but it means the issue's claim "fails 3 of 3 on main" is unconfirmed here, hence medium confidence. Nothing blocks shipping. One comment claim is inaccurate, and the Log is missing the 100-run result that `## Done when` asks for.

1. **Strengths**
   - `cmd/internal/webcast/web/replayer.js:173-175` reuses `settle()` (ARCH-DRY: pass) instead of a font-only `document.fonts` wait, so the fonts, images and sheets bound by t are all covered in one place.
   - The returned `canonical` is now read after the loads finish (`replayer.js:174`), so the canonical state and the screenshot describe the same settled page.
   - `tests/webcast/replayer.test.mjs:118-134` checks the order with an injected fake: pause, then `ready`, then pause the new animation, and an already-paused animation is not paused twice. It needs no IO and failed when I reverted the fix (ARCH-PURE: pass).
   - The live test builds a deterministic case (a font and an image bound at 99, captured at 100) instead of relying on a soak. That matches the lesson the change records.
   - The atlas line (`atlas/webcast.md:112`) and the lesson are updated. No new user-facing surface is added, so README needs no change.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - The comment at `replayer.js:171-172` says "anything the loads started stops at its start". That's wrong: an animation started by a load (for example one defined in a stylesheet that just applied) runs for the whole `settle()` (load time plus two animation frames, up to about 5 s) and is paused wherever it has got to. Either reword the comment ("stops where it is after the loads") or make it true by setting `currentTime = 0` on the newly started animations. I checked the other claims in this window and they hold: the file header at lines 12-13, `atlas/webcast.md:112-114` and the lessons entry.
   - `## Done when` asks that rec-blog frame 0 scores about 0.1% on 100 consecutive runs under load. The 100-run soak plan item is ticked, but the Log gives no result (no pass count, no worst score). Add the measured outcome to the Log.

5. **Test coverage notes**
   - JS unit test: confirmed. It passes at head and fails with base `replayer.js`.
   - Live Go test: skipped by default (needs `WEBCAST_LIVE_CHROME=1`). I couldn't run it because Chrome kept dying with `cdp: read: EOF`, both sandboxed and unsandboxed. It checks both kinds of asset (font `Late:loaded` and `img:true`), so both states of the font-or-image fix are covered.
   - My revert was restored. `git status` is clean.

6. **Architectural notes**
   - ARCH-DRY: pass. It reuses `settle()` and `ready()` and duplicates nothing.
   - ARCH-PURE: pass. The ordering logic in `Replayer` is tested through injected `deps`. Browser IO stays in `inBrowser` and `Renderer.ready`.
   - ARCH-PURPOSE: pass. The Spec asked for a font fix and the change covers the whole class (all assets bound by t). The issue also checks the other fixtures, and their first frames are covered by the same path.
   - Every capture now pays one extra flush plus two animation frames. That's acceptable for the fidelity check, but keep it in mind if `replayAt` ends up in a hot loop.

7. **Plan revision recommendations:** none needed. The plan matches the code. Only the Log needs the soak result.

```findings
findings:
  - id: new
    severity: Minor
    family: comment-claim-unverified
    title: |
      replayer.js:171-172 claims animations started by the loads "stop at its start"; they stop after running through settle()
    detail: |
      The new animations are paused only after settle(), which waits for the loads plus two animation frames (up to the 5 s bound), so their phase is not their start. Reword the comment, or set currentTime = 0 on the newly started animations. The other claims in this window hold: replayer.js:12-13, atlas/webcast.md:112-114 and the lessons.md entry.
  - id: new
    severity: Minor
    family: done-when-evidence-in-log
    title: |
      The 100-run under-load soak result that Done-when requires is not recorded in the Log
    detail: |
      The plan item is ticked, but the Log gives no count or worst score for rec-blog frame 0 across the 100 runs. Add the measured result so the close has evidence for the first Done-when clause.
```
