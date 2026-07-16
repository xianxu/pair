# Boundary Review — pair#93 (milestone M4)

| field | value |
|-------|-------|
| issue | 93 — port stateful shell orchestrators to Go |
| repo | pair |
| issue file | workshop/issues/000093-port-shell-orchestrators-go.md |
| boundary | milestone M4 |
| milestone | M4 |
| window | 9c9da8b417f5ae47a87c035e5ef83a40bb268043..HEAD |
| command | sdlc milestone-close --issue 93 --milestone M4 |
| reviewer | claude |
| timestamp | 2026-07-01T23:29:15-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** M4 is a clean, faithful port. All three clipboard helpers (`copy-on-select`, `clipboard-to-pane`, `flash-pane`) moved to Go behind tracked `.sh` re-exec shims exactly as the plan's SHIM decision specifies; the exec chain, the two-different-regex in-nvim gate, the detached flash reset, the `Ctrl-_` (31) trigger, and env/tag/data-dir resolution all match the source scripts. I verified behavior end-to-end: `go test ./cmd/internal/clipcmd ./cmd/internal/zellijpane` pass, the retained `tests/copy-on-select-test.sh` passes against the freshly-built Go binary, `go vet` + `go build ./...` are clean, the runtime bundle manifest carries all six assets (3 shims + 3 binaries), `git ls-files bin/` lists only the `.sh` shims, and the atlas + index are updated. Nothing blocks the boundary — the findings are one minor behavior drift and two bookkeeping/forward items.

**1. Strengths**
- **ARCH-PURE — textbook.** `clipcmd.go` + `zellijpane.go` hold the pure decisions; every IO touch (clipboard, zellij IPC, spawn/exec, fs, log) sits behind the `Runtime` seam (`run.go:19-46`). Pure tests need no mocks (`clipcmd_test.go`), and the orchestrations are driven by a scriptable `fakeRuntime` (`run_test.go`). This is the cleanest split of the four milestones so far.
- **Faithful two-regex distinction** (`clipcmd.go:35`, `:56`): the copy-on-select in-nvim gate uses `(?i)nvim|draft` (matches shell `grep -qiE`), while the clipboard-to-pane pane picker uses case-sensitive `nvim` (matches jq `test("nvim")`). Subtle, and correctly preserved as two separate checks.
- **The #copy-on-select-test regression is doubly covered** — a Go unit test with the parley.nvim-cwd fixture (`clipcmd_test.go:24`) *and* the retained shell integration test (`copy-on-select-test.sh`), both keying on `terminal_command` not `title`.
- **The "don't block the caller" flash contract is pinned** (`run_test.go:TestFlashPaneSetsFgAndSchedulesDetachedReset`): asserts exactly one synchronous `set-pane-color` + one *scheduled* detached reset, so a regression that ran the reset synchronously would fail the test.
- **Two spawn modes correctly separated** (`RunSubprocess` call-and-return for flash vs `ExecReplace` process-replace for the hand-off, `runtime.go:135-149`), matching the shell's synchronous flash + `exec` hand-off.

**2. Critical findings** — none.

**3. Important findings** — none.

**4. Minor findings**
- `runtime.go:150` — `OSRuntime.Log` opens the diagnostic with `O_APPEND`; the source `clipboard-to-pane.sh` truncated it (`> "$LOG"`) each run. The debug log now grows unbounded in `~/.cache`. Fix sketch: truncate at the copy-on-select entry (pipeline head) or cap size. Diagnostic-only, low priority.
- `opener/runtime.go:63` — ARCH-DRY: opener's `firstAgentPaneID`/`isAgentPane` walk duplicates what `zellijpane.Parse` now does; the diff *reduces* duplication (2 clip consumers share the new parser) and the deferral is documented with a forward-compatible `Title` field, but the third consumer should adopt it before it ossifies.
- Inventory row for `flash-pane` lists "nvim flows/tests" as a consumer (`go-migration-inventory.md`), but no `nvim/*.lua` currently invokes `flash-pane.sh` — only copy-on-select + the shell test do. Carried over from the old row; trim when convenient.

**5. Test coverage notes.** Coverage is strong and pins real logic, not mocks: pure selectors/classifiers tested directly; `zellijpane.Parse` tested against a real tab-keyed `--command` fixture plus invalid-JSON/string-id/non-pane-wrapper edges; the three `Run*` orchestrations tested for the consequential paths (agent-pane hands-off, draft-pane skips, empty selection, no-tool → exit 1, no-nvim-pane fallback, flash override/no-op). The `test-copy-on-select` target correctly gained the `$(BIN_DIR)/copy-on-select` prereq up front (the M2/M3 missing-prereq lesson applied proactively). The one gap — the unbounded-log behavior — is unobservable through the fake `Log(string){}` and wouldn't be caught by any test, consistent with it being diagnostic.

**6. Architectural notes for upcoming work.** The `zellijpane` extraction is the right shared home for the list-panes walk; folding `opener` onto it (a pure swap, per the `Title`-field note) is the natural next DRY step and should ride the next milestone that touches opener rather than lingering as a standing follow-up. ARCH-PURPOSE is fully met for M4 — #93 correctly stays `open` for M5 (launcher), consistent with the M3 plan-quality note; do not let an M4 close flip the issue to done.

**7. Plan revision recommendations**
- Add a `## Revisions` entry recording the `Exec(path, args…)` → `RunSubprocess` + `ExecReplace` seam split (the change-code plan-quality note #2 is in the issue Log, but the plan's M4 section at line 306 still claims a single `Exec`).
- Add a `## Revisions` note that the clipboard-debug log became append-only (deliberate delta from the source's truncate) — for the same byte-faithfulness bookkeeping M1–M3 used.
- Optionally record the `opener` → `zellijpane` adoption as a tracked follow-up in the plan `## Revisions` (currently only in code/atlas comments), so it's visible at the plan level like the M3 codexsid-triplication follow-up.
