# Boundary Review — pair#467 (whole-issue close)

| field | value |
|-------|-------|
| issue | 467 — broadcast-web: remote driving — a viewer takes control and uses the app through the operator's tab |
| repo | pair |
| issue file | workshop/issues/000467-broadcast-web-remote-driving-a-viewer-takes-control-and-uses-the-app-through-the-operator-s-tab.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6a3ea808b9307298d96bc6b1014e3f1554a37f6b..c41a80bfc389da293b32e0ed9fac409ee3c12cf2 |
| command | sdlc close --issue 467 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T10:10:42-07:00 |
| verdict | SHIP |

## Review

Disposing the three open findings (BR-35's comment fix and BR-3's idle sweep both check out). Next I'm running vet and the tests for the touched packages.

```verdict
verdict: SHIP
confidence: high
```

All three open findings are closed this round, and I found nothing new. BR-35 is fixed. The by-tag comment at `recorder.go:1084-1086` now says what `observeItems` does at 1099-1104: it drops the tag and keeps the act. The reducer wording that remains in `mirror_test.go:279` is still accurate, because `Mirror.Observe` really does refuse a non-driver's by. BR-3 is effectively fixed. `limits()` (`server_drive.go:164-189`) no longer clears the whole table. When the table is full it removes only keys idle longer than `seatIdle`, and if it is still full it hands the new key a zero budget. `forget` removes keys the controller refuses, the change is recorded in Revisions at plan line 392, and `server_drive_test.go:266-289` tests the sweep. BR-6's tap is still the only test hook in the loop state, it is nil in production, and plan line 394 records it, so I'm withdrawing it as a minor note that needs no action.

The last commit (c41a80bfc) only updates the artifactpath inventory: it renames `fake_controller.go` to `controller_mem.go` and lists the new drive sources. `go vet` and `go test` pass for `./cmd/internal/webcast/...`. Two other test results fail for reasons that have nothing to do with this issue:
- **couchtty:** fails at setup because `runtimebundle/assets` is gitignored and generated, so this checkout doesn't have it.
- **artifactpath:** the inventory test flags `cmd/internal/testwait/testwait.go`, which came from #479 and didn't change in this window. It also lists stale entries that come from the same missing assets.

1. **Strengths**
   - The per-seat table now has an explicit bound and an idle sweep, with a test (`server_drive.go:18-25`, `server_drive_test.go:244-289`). This settles ARCH-FUNERAL for this table.
   - A late tagged report keeps its act and loses only an attribution it can no longer support (`recorder.go:1099-1104`). That is the right way to handle the uncertainty.
   - The artifactpath manifest now covers every new drive source and states "memory only, no paths" (`manifest.go:782-797`).
2. **Critical findings:** none.
3. **Important findings:** none.
4. **Minor findings:** none new. Plan body line 178 still describes the older "plus one" bound for the seat table, but the Revisions entry at line 392 supersedes it.
5. **Test coverage notes:** the webcast tests pass. couchtty couldn't be checked here because of the generated assets, and the artifactpath failure predates this issue (#479's `testwait`), so it should be fixed on main rather than here.
6. **Architecture notes:**
   - ARCH-DRY, ARCH-PURE, ARCH-PURPOSE, ARCH-MOCK, ARCH-SECURE: pass. Nothing new in the last two commits.
   - ARCH-CONSTRAINTS and ARCH-FUNERAL: pass. The seat table's bound is now enforced on the writer.
   - ARCH-ORDER: pass. A late report is handled against the controller's current state.
   - If more test hooks build up in `loopState`, combine them into one injected observer.
7. **Plan revision recommendations:** none.

```findings
dispose:
  - id: BR-3
    disposition: addressed
    note: |
      limits() now sweeps only keys idle past seatIdle and gives overflow keys a zero budget instead of clearing; forget() removes refused keys; Revisions line 392 records it; server_drive_test.go:266-289 pins the sweep.
  - id: BR-6
    disposition: withdrawn
    note: |
      Still the only test hook in loopState, nil in production and recorded in Revisions (plan line 394); no action needed unless more hooks are added.
  - id: BR-35
    disposition: addressed
    note: |
      recorder.go:1084-1086 now says observeItems drops the tag and keeps the act, matching lines 1099-1104; the reducer wording left at mirror_test.go:279 is accurate for Mirror.Observe.
```
