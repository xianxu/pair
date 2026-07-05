# Boundary Review — pair#100 (whole-issue close)

| field | value |
|-------|-------|
| issue | 100 — copy-on-select paste intermittently dropped — orchestration runs inside zellij copy_command hook and gets reaped |
| repo | pair |
| issue file | workshop/issues/000100-copy-on-select-paste-intermittently-dropped-orchestration-runs-inside-zellij-copy-command-hook-and-gets-reaped.md |
| boundary | whole-issue close |
| milestone | — |
| window | b54546eb2c4ab7885e9bd95f1792f353d473818c..HEAD |
| command | sdlc close --issue 100 |
| reviewer | claude |
| timestamp | 2026-07-05T11:26:53-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

This is a clean, well-scoped root-cause bug fix. The `copy_command` hook is correctly emptied down to clipboard-mirror + detached-spawn, with the slow flash+paste chain moved into a `setsid`-detached `copy-on-select --orchestrate` that survives zellij's ~1s SIGKILL reap — deadline-independent, exactly as the Spec argues. All four Done-when items are delivered, the plan's Core-concepts table matches the code precisely, the atlas is updated in-range, the temporary `[trace]` instrumentation is fully removed (grep-clean in `clipcmd`), and both the Go unit tests and the async-converted shell regression test pass here. Nothing blocks SHIP; the findings below are minor/observability-only.

### 1. Strengths
- **ARCH-DRY consolidation done right** — `startDetached` extracted (`runtime.go:93`) and shared by both `ResetPaneColorAfter` and the new `SpawnDetached`. The diff needed a 2nd clipcmd detach caller and *reduced* to one shared helper rather than adding a 3rd inline copy. Textbook at-review DRY.
- **ARCH-PURE respected** — decision logic (`RunCopyOnSelect`/`RunCopyOnSelectOrchestrate`) stays behind the `Runtime` seam and is faked in tests; `SpawnDetached` is a thin IO wrapper. No business logic leaked into the OS layer.
- **The test pins the actual invariant that fixes the bug** — `TestCopyOnSelectHookMirrorsThenDetaches` asserts `listCalls==0 && subprocess==∅ && execd==∅` (`run_test.go:120`), i.e. the hook runs *none* of the slow chain inline. That is precisely the reap surface #100 is about, not a mock reasserting the implementation.
- **Shell test correctly made async** (`tests/copy-on-select-test.sh:68`) — polls for the handoff marker for the now-detached hand-off, and uses a grace-wait absence check for the in_nvim negative case (`:87`). The right shape for a detached side effect.
- **Root cause over bandaid** — detachment chosen over prewarm, with the reasoning captured in Spec; matches AGENTS.md "Root Cause".
- **Clean instrumentation teardown** — no `[trace]`/`installSignalLogger`/`os/signal` residue anywhere in `cmd/internal/clipcmd`.

### 2. Critical findings
None.

### 3. Important findings
None.

### 4. Minor findings
- **Orchestrator's `ExecReplace` failure is now invisible** (`run.go:122-125`). In the detached orchestrator, `stderr` is `/dev/null` (parent's `startDetached` redirects it), and unlike the flash path — which logs `rt.Log("flash-pane failed: …")` at `run.go:114` — the terminal hand-off failure only writes to `stderr`, never to `rt.Log`. For a pipeline this finicky (the whole reason `clipboard-debug.log` exists), a failed hand-off would show `=== copy-on-select --orchestrate ===` + `in_nvim: false` and then silence, with no recorded cause. Behavior is inherited from the pre-split code (not a regression), but the detach makes the debug log the *only* possible channel now. Cheap fix: add `rt.Log(fmt.Sprintf("exec %s failed: %v", clipScript, err))` alongside the `Fprintf`. (Relatedly, `SpawnDetached` can't surface a `Start()` failure to the hook at all — acceptable, since it's a self-exec of the same binary and effectively cannot fail.)
- **Fixed `sleep 1` before the negative assertion** (`copy-on-select-test.sh:87`) — standard for a "must-not-happen" check against an instant fake zellij, but adds ~1s and is theoretically flaky under extreme CI load. Fine as-is.

### 5. Test coverage notes
Coverage is solid and targets the real risk. The only unexercised seam is the `--orchestrate` argv dispatch branch in `runcli.go:20` (it constructs `NewOSRuntime()` directly, so it isn't unit-testable), but it is thin glue and is covered end-to-end by the shell test, which drives the *real* self-exec chain through `main → RunCopyOnSelectCLI → SpawnDetached → copy-on-select --orchestrate → RunCopyOnSelectOrchestrate`. No gap worth a new test.

### 6. Architectural notes for upcoming work
- **Cross-package detach duplication (ARCH-DRY, future)** — `launcher.spawnDetached` (`cmd/internal/launcher/osruntime.go:311`) is a near-identical setsid + `/dev/null` + `Start` + `Wait` idiom to the new `clipcmd.startDetached`. Not this issue's job (pre-existing, different package, and this diff correctly avoided adding a 3rd copy), but a future `cmd/internal/procutil.Detach` could unify all sites. Two semantic differences to reconcile if that happens: launcher takes `extraEnv` and *bails without spawning* if `os.Open(DevNull)` fails, whereas `startDetached` still spawns (child inherits parent fds) — a deliberate, arguably better behavior worth carrying to the merged helper.
- **Double `ListPanes`** (orchestrator + clipboard-to-pane) is a documented, reasoned deferral (plan "Notes") — harmless once off the reap path, and merging would perturb `clipboard-to-pane`'s tested contract. Correct call to leave it.
- **ARCH-PURPOSE: PASS** — the hook is *fully* emptied of slow work, so the fix holds on cold/slow machines (the operator's actual concern), not just the fast common case. This is the purpose, delivered, not the cheap subset.

### 7. Plan revision recommendations
None required. The Core-concepts table matches the code exactly (`RunCopyOnSelect` modified + `RunCopyOnSelectOrchestrate` new in `run.go`; `SpawnDetached` new in `runtime.go`), and every `## Plan` checkbox is genuinely delivered. One optional, non-blocking note: the plan's Task 1 Step 2 snippet shows `SpawnDetached` with the setsid idiom *inlined*, whereas the shipped code routes through the shared `startDetached` — but that refinement is the ARCH-DRY improvement already recorded in the issue's Plan (lines 97-98), so the plan-as-archived is internally consistent and needs no `## Revisions` entry.
