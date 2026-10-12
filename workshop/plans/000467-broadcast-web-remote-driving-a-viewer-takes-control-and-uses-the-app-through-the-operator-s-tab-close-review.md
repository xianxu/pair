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

---

## Re-review — 2026-10-12T10:24:32-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 467 — broadcast-web: remote driving — a viewer takes control and uses the app through the operator's tab |
| repo | pair |
| issue file | workshop/issues/000467-broadcast-web-remote-driving-a-viewer-takes-control-and-uses-the-app-through-the-operator-s-tab.md |
| boundary | whole-issue close |
| milestone | — |
| window | 08737bb9664478e3af86b5a0c06bfcb0b3f01a4d..d449b8c0501e1a0d6876ed1ae8c2a9adc20a8bbd |
| command | sdlc close --issue 467 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T10:24:32-07:00 |
| verdict | SHIP |

## Review

Checking the one conflict resolution in the window. The full build needs generated runtime assets, so next I'm looking for how those are made so I can compile and test `couchcmd`.

Simplest path: stub the embedded assets in this disposable checkout so `couchcmd` compiles.

`couchcmd` passes vet and tests. Last step: confirm the two plan-file changes only record review results and don't change scope.

```verdict
verdict: SHIP
confidence: high
```

**Summary.** This window is small. The only code change is the conflict fix in `cmd/internal/couchcmd/message_service.go`. Everything else is tracker data: the close-gate ledger, the close-review record, and the issue file's frontmatter and Log. The base commit `08737bb96` had raw conflict markers committed at `message_service.go:376-380`, so `couchcmd` would not compile there. Head removes the markers and keeps both sides: `service.SetShares(console.Shares)` from this branch (#467 M3) and `service.SetBlockedView(console.SetBlockedThreads)` from main (#465). Both methods exist, at `message_service.go:903` and `:863`. Both are plain setters called before the wait watcher starts, so their order doesn't matter. `go vet ./cmd/internal/couchcmd/` is clean and `go test ./cmd/internal/couchcmd/` passes (39.7s). To compile, I had to put temporary stub files in the generated, gitignored `runtimebundle/assets` and removed them afterwards. That is a checkout limitation, not a defect. The issue file only moves `status` to `codecomplete` and adds `actual_hours`, the card hash and a close Log line. `## Spec`, `## Revisions` and `## Done when` are untouched, so there is no change of scope.

There are no open prior findings. The 35 already disposed (BR-1 to BR-35) are not re-raised.

1. **Strengths**
   - The conflict fix keeps both sides' intent: the share kinds from M3 and the blocked-thread view from #465 are both wired up (`message_service.go:375-377`).
   - The head also fixes a base commit that would not compile.
   - The ledger and review record match the round-13 dispositions exactly.
2. **Critical findings:** none.
3. **Important findings:** none.
4. **Minor findings:** none in the window. Conflict markers remain in `workshop/history/plans/000255-...-plan-gate.md`, but that file is outside this window and not from this issue.
5. **Test coverage notes:** the `couchcmd` tests pass at head, with stubbed generated assets. The fix is a one-line wiring change, and each setter is already covered by its own feature's tests.
6. **Architectural notes (interdiff only)**
   - ARCH-DRY: pass. The fix adds no logic.
   - ARCH-PURE: pass. It only wires setters in the existing IO shell.
   - ARCH-PURPOSE: pass. Both features stay connected and nothing is dropped.
7. **Plan revision recommendations:** none.

```findings
```
