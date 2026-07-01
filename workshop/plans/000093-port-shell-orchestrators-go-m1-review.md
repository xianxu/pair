# Boundary Review — pair#93 (milestone M1)

| field | value |
|-------|-------|
| issue | 93 — port stateful shell orchestrators to Go |
| repo | pair |
| issue file | workshop/issues/000093-port-shell-orchestrators-go.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | 80f6a9dfbcc99fbb59056023fa3b38c8956cf0df^..HEAD |
| command | sdlc milestone-close --issue 93 --milestone M1 |
| reviewer | claude |
| timestamp | 2026-07-01T15:10:06-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I've verified the port against the shell source, the plan, and the running build. Build is green, `go test ./cmd/internal/titlepoller ./cmd/internal/procutil ./cmd/internal/contextcmd` passes, and the manifest/atlas/Makefile are coherently updated.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

This is a faithful, well-structured port of `bin/pair-title.sh` → `cmd/pair-title` + `cmd/internal/titlepoller`, following the #78 sessionwatch template exactly. Pure decisions are cleanly separated and thoroughly unit-tested; IO sits behind a `Runtime` seam; the shim preserves the argv shape the single-instance guard matches. I found no correctness bugs — the load-bearing pidfile-guard ordering is preserved correctly, the cmux/frame surfaces match the shell semantics, and ARCH-DRY consolidations (procutil, `contextcmd.TranscriptPath`, in-process count) are real. The one thing keeping this from a clean SHIP is a **test-coverage gap the plan explicitly promised**: the loop body (frame + cmux rendering through `Run`) is not integration-tested. Nothing blocks the boundary.

### 1. Strengths

- **Pidfile-guard ordering is correct and load-bearing** (`run.go:78-88`): the live-instance guard `return 0`s *before* `defer rt.Remove(pidfile)` is registered, so a second invocation never deletes the running poller's pidfile. `TestRunDefersToLiveInstance` pins exactly this (no session probe, no writes). This is the subtle bit the shell got right (`exit 0` before `trap … EXIT`) and the port preserves it faithfully.
- **Excellent pure-decision coverage** (`titlepoller_test.go`): `prefixForAge` bucket boundaries incl. `oneDay-1s`/`oneDay` edges, `abbrevCwd` incl. `/Users/xyz` non-boundary + empty-home, `pollerArgvMatches` incl. the 21-vs-211 collision and the "shim itself is not the running poller" case (`titlepoller.go:88-97`), `shouldClaimWorkspace` all four owner states, `frameCache` skip semantics. These replicate every assertion the deleted shell harness made.
- **ARCH-DRY consolidations are genuine, not cosmetic**: `procutil.Alive`/`Command` now back both `sessionwatch.OSRuntime` and `titlepoller.OSRuntime` (`sessionwatch/runtime.go:106`); `contextcmd.TranscriptPath` (`contextcmd.go:53`) is the single resolver for both `pair context` and the poller's activity-mtime; `ContextCount` runs `contextcmd.Run` in-process (`runtime.go`), eliminating the `pair context` subprocess per pane per tick.
- Manifest regenerated with both `bin/pair-title` (Go binary, 3.4 MB digest) and `bin/pair-title.sh` (shim) — identical treatment to the `pair-session-watch` precedent.

### 2. Critical findings

None.

### 3. Important findings

- **Loop-level integration untested — cmux ownership wiring + frame/cmux gating** (`cmd/internal/titlepoller/run.go:139-207`). The plan (M1 Tests) promised "A `Runtime`-mock loop test: one tick renders the expected renames; a second identical tick emits none." What's delivered tests `updateFrameTitles` *directly* and the miss-threshold exit, but never exercises the loop body: `activityMTime` → `age` → `updateFrameTitles`/`updateWorkspaceTitle`. In particular `updateWorkspaceTitle` (`run.go:189-207`) — the cmux presence-beats-stale-flag logic the atlas devotes a full paragraph to — has **zero** coverage beyond the pure `shouldClaimWorkspace` helper; a wiring bug (wrong owner path, missing owner write, wrong `pair-<owner>` session name, inverted age/cmux gate) would ship silently.
  Fix sketch: one fake-Runtime test — `sessionAliveSeq=[true(grace), true(tick), false×MissThreshold]`, set `mtimes` on the draft so `age < 2*PollInterval`, set `cmuxAvail=true` + `CmuxWorkspaceID`, seed `panes`/`counts`, then assert `rt.renamed` (frame) and `rt.cmuxRenamed` (workspace) after `Run` returns. Add a second variant with a live foreign owner to assert the defer path leaves `cmuxRenamed` empty.

### 4. Minor findings

- **claude activity-transcript resolution changed from `$PWD` (shell) to paneCwd (Go).** Shell `agent_session_file` encoded `$PWD` (the launcher's ambient cwd); the port resolves via `TranscriptPath` → `paneCwd` (`contextcmd.go:59`). This is intentional and documented (atlas: "resolved via the same path `pair context` uses"), and for the primary pane `paneCwd == launch cwd`, so it's a no-op in the common case — but it is *not* byte-identical to the shell, and where `pane-<tag>-<agent>.json` is absent the claude transcript won't resolve for the activity check (draft still does). Acceptable; flagging so the operator knows it's a deliberate refinement, not a faithful copy.
- **Cross-version upgrade transient.** A pre-port poller still running as `bash …/pair-title.sh <tag> <agent>` is not recognized by the new `pair-title <tag> ` argv guard (`titlepoller.go:96`), so the first post-upgrade spawn won't defer to it → brief double-poller window. Self-heals when the old session ends; inherent to the port, worth a one-line Log note.
- **Could not run `make runtimebundle-drift-check` in this review env** (TMPDIR/`mkdir /` sandbox restriction, not a code issue). `manifest.json:135-146` already carries both `bin/pair-title` + `bin/pair-title.sh`; confirm drift-check is clean in a normal shell before merge.

### 5. Test coverage notes

Pure layer: excellent. Loop/seam layer: `updateFrameTitles` covered directly (3 shapes + skip), miss-threshold + grace-timeout + live-instance guard covered in `run_test.go`. Gaps: `updateWorkspaceTitle`, `activityMTime`, and the in-loop age/cmux gating (the Important finding above). `procutil_test.go` is appropriately defensive (skips the `ps` probe under a locked-down environment via `psAvailable()`).

### 6. Architectural notes for upcoming work

- **ARCH-DRY — pass.** `procutil` is the right extraction point and the plan-revision reasoning ("two consumers is where DRY starts paying, M2–M5 each add a runtime") is sound. `contextcmd.TranscriptPath` extraction is the correct shared seam. No duplicated logic in the diff.
- **ARCH-PURE — pass.** Textbook split: `titlepoller.go` is pure and directly tested; `run.go` orchestrates over the `Runtime` interface; `runtime.go` is the thin OS shell. No business logic buried in IO. The Important finding is a *coverage* gap, not a purity violation — the seam is correctly shaped to make that test trivial to add.
- **ARCH-PURPOSE — pass.** M1 fully delivers its slice (Go owner + shim + pure tests + seam); the deferred M2–M5 are genuinely separable surfaces, not the deferred point of M1. The "repoint nvim/zellij shell-outs" Done-when is legitimately a no-op for M1 — the only caller is `bin/pair-shell`'s `ensure_title_poller`, whose spawn-by-path is preserved.
- For M2–M5: the `Runtime`-seam + `RunCLI(args, getenv, stderr)` shape is now proven twice; keep the loop-body integration test as a first-class deliverable per port (this milestone under-delivered it) so the cmux/openers wiring doesn't go untested.

### 7. Plan revision recommendations

- Add a `## Revisions` entry: the M1 Runtime-seam spec lists `Log(...)` (adapt recorder) and runcli.go "open the adapt logger", but the implemented `Runtime` has **no** `Log` method and the original `pair-title.sh` never emitted adapt events — dropping it is the *faithful* choice. Record that the adapt-logging seam was intentionally omitted so the plan stops claiming a `Log` seam the code doesn't (and shouldn't) have.
- Optional: the plan names a pure `latest(sources)` helper; the code implements it as `activityMTime(opts, rt)` over the seam (correct, since mtime reads are IO). A one-line note keeps the plan's Core-concepts naming matched to the code.
