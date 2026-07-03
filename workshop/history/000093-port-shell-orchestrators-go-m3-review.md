# Boundary Review — pair#93 (milestone M3)

| field | value |
|-------|-------|
| issue | 93 — port stateful shell orchestrators to Go |
| repo | pair |
| issue file | workshop/issues/000093-port-shell-orchestrators-go.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 66840bf5ccff071d2efe65f1f6dd61dc6cac808f..HEAD |
| command | sdlc milestone-close --issue 93 --milestone M3 |
| reviewer | claude |
| timestamp | 2026-07-01T16:46:57-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I have thoroughly verified the port against the deleted shell sources, confirmed compilation and green tests, and checked the callers, build graph, and ARCH principles. Here is my review.

```verdict
verdict: FIX-THEN-SHIP
confidence: medium
```

The M3 port is faithful and well-structured. I diffed every behavioral path against the three deleted shell scripts — session-priority resolution, the codex ps→lsof→rollout walk, readiness fact-gathering, the `nvim --headless` classify bridge, the `--prepare` git-effect sequence, and the open single-pane replacement + `zellij run` argv — and found no behavior drift or silent error-swallowing beyond the source's own semantics. New/retrofit packages compile and `go test ./cmd/internal/{reviewcmd,osfs,codexsid,opener,titlepoller}` is green; the three binaries build; the shell tests genuinely drive the Go binaries against real git + real nvim classify. Nothing blocks SHIP. Two Important gaps remain (both cheap): `test-review` lacks the built-binary prerequisites so a fresh-tree `make test` fails, and the most destructive `--prepare` path (the `track` add+commit sequence) plus `resume` are untested despite the plan promising per-case coverage.

**1. Strengths**
- Faithful codex-walk port: `codexsid.descendants` BFS order (root-first) + first-rollout-match matches the shell's awk-BFS→`lsof`→`head -1` exactly (`cmd/internal/codexsid/codexsid.go:47`), and it's genuinely *reused* by reviewcmd (`runtime.go:100`), not copy-pasted a third time.
- The pure 4-case decision stays single-source in `nvim/review/readiness.lua`, invoked via `nvim --headless` — the port did not re-implement the classifier in Go (ARCH-PURE / ARCH-PURPOSE both honored).
- Clean seam split: pure `slugify` / `targetJSON` / `sessionFromConfig` unit-tested without IO; git/nvim/zellij/codex behind `Runtime`. The `osfs.FS` extraction cleanly de-dups the fs primitives across three `OSRuntime`s (opener/titlepoller retrofit verified compiling + green).
- `targetJSON` via `encoding/json` is strictly more robust than the shell's `jq -n --arg` for paths with quotes/spaces, and `Classify`'s `%q`-quoted lua path survives characters the shell's single-quotes would break — improvements that don't change the contract. Test `TestTargetJSON` pins the quoted-path case.
- Correct replace-in-place: `git ls-files bin/` shows no tracked review helpers; the 3 `.gitignore` negations dropped; atlas inventory + migrated-list + package-list all updated.

**2. Critical findings**
None.

**3. Important findings**
- **`test-review` has no build prerequisite for the review binaries** (`Makefile.local:118`). The three tests exec `$ROOT/bin/pair-review-{target,open,readiness}`, which are now gitignored *built* Go binaries (previously committed shell scripts that always existed on disk). On a fresh tree, `make test-review`/`make test` fails with "No such file or directory" because nothing in the `test` chain builds them first. This is the same class the M2 review fixed for `test-changelog` (`Makefile.local:244`) and follows the `test-session-watch` convention (`:95`). Fix: `test-review: $(BIN_DIR)/pair-review-target $(BIN_DIR)/pair-review-open $(BIN_DIR)/pair-review-readiness`. (Fails loudly rather than silently, so lower-impact than M2's SKIP, but still breaks fresh-tree builds.)
- **No test for the `--prepare` `track` or `resume` cases** (`cmd/internal/reviewcmd/run.go:190` `prepare()`). Only `new` is covered (`run_test.go:TestRunReadinessPrepareNew` + shell `review-readiness-cli-test.sh`). The `track` path is the most consequential — `git add -- abs` → `git commit` (commits the whole index) → `ls-files --error-unmatch` verify → `status --porcelain` "unrelated changes" guard → branch create — and a regression there mutates a user's repo during review-start. The plan's own M3 test section promised "the `--prepare` action plan **per case** (fake git seam asserts the add/commit/checkout sequence + the mark-ready write)". Fix: add a `fakeRuntime` test asserting the track sequence (`add`, `commit`, `ls-files`, `status`, `checkout -b`) + the ready-mark write, and a `resume` case (asserts `reviewBranch = branch`, no checkout).

**4. Minor findings**
- `cmd/internal/reviewcmd/run.go:96` `RunTarget` and `prepare` write the target JSON non-atomically via `WriteFile` — faithful to the shell's `jq > out`, so no drift; noting only that `osfs.WriteAtomic` exists and a concurrent Alt+c reader could see a torn read (latent, pre-existing).
- Missing-arg exit code drift: shell `${1:?}`→1, Go `RunTargetCLI`→2 (`runcli.go:13`). Untested, harmless (2 is the more conventional usage code).
- `TestRunReadinessJSON` computes but never asserts `scoped_file`/`file_matches`; the JSON-mode scoped-file path is effectively unchecked.
- `nvim/review/readiness.lua:4` comment ("the thin git-fact / git-effect gathering lives in `bin/pair-review-readiness`") is now imprecise — the gathering moved to `cmd/internal/reviewcmd`. Not in this diff's scope; defensible since the binary still lives at that path.

**5. Test coverage notes**
`codexsid`/`osfs`/`reviewcmd` have focused unit tests; `codexsid.TestRolloutRE` + no-pidfile/empty-pidfile paths are covered. The open flow (single-pane replace + spawn argv + missing-file) and readiness JSON + prepare-`new`/`stop`/`interact` are covered by fakes. Real gaps: prepare `track`/`resume`/`resumed-existing` (Important #2). The three shell integration tests are correct process-level fakes (real git temp repo + real nvim classify + faked zellij) and do drive the Go binaries — good.

**6. Architectural notes for upcoming work**
- ARCH-DRY — **PASS with forward note.** The codex walk is now triplicated (`codexsid` + `sessionwatch/runtime.go` + `slugcmd`); this diff added the *canonical* copy and wired review-target to it, but the "canonical home" claim isn't fully true until `slug`/`sessionwatch` adopt `codexsid`. The plan legitimately defers retrofitting those tested hot-path packages — track it as an explicit follow-up so the triplication actually collapses.
- ARCH-PURE — **PASS.** Pure classify single-sourced in lua; `slugify`/JSON pure and unit-tested; `prepare()` is inherently sequential IO (its branches depend on `show-ref`/`ls-files`/`status` results), so keeping it as thin seam-orchestration rather than a pre-computed pure plan is the right call.
- ARCH-PURPOSE — **PASS.** All three helpers ported, shell scripts deleted (no shim), callers (`nvim/init.lua`) repoint to the same PATH names, and the single-source classifier is preserved via `nvim --headless`. Shadow-sweep of consumers finds no remaining hand-maintained restatement of the readiness decision.
- The `osfs.FS`-embed pattern is a good seam for M4/M5 to reuse; each `Runtime` interface still declaring only its subset (extra embedded methods harmless) is the right shape.

**7. Plan revision recommendations**
Add a `## Revisions` entry to `workshop/plans/000093-port-shell-orchestrators-plan.md`:
- `codexsid.ResolveSessionID(dataDir, tag)` shipped **without** the planned `home` parameter (`ResolveSessionID(dataDir, tag, home)`) — `home` was unused; the walk reads `$dataDir/agent-pid-<tag>`. Correct as shipped; plan wording updated.
- Re-categorize the M3 "Pure (direct unit tests)" list: `absPath normalization` is on the IO seam (`AbsFile`/`LogicalDir`/`PhysicalDir` stat + `EvalSymlinks`), and the `--prepare` action mapping is delivered as seam-orchestration in `prepare()` (interleaved with git-effect results), not as a standalone pure plan — both are tested via the fake `Runtime`, not directly.
- Record that `--prepare` `track`/`resume` coverage is deferred (or, preferably, close Important #2 and drop this note).
