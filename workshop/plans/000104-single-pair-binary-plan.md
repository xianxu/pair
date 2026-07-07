# Single `pair` Binary Consolidation — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Collapse *every* pair-repo Go binary into a single `pair` program whose functionality is reached as `pair <subcommand>`. The only other artifacts left are the three shell shims (`pair-dev`, `pair-help`, `pair-notify`). `pair-scribe` folds in too — the user's `~/.zshrc` changes from `exec pair-scribe` to `exec pair scribe`.

**Architecture:** The heavy lifting already exists — every standalone helper is a 13–15 line thin `main.go` over a `cmd/internal/*cmd` package exposing a `RunXxxCLI(args, env, out, err)` seam, and `cmd/pair-go` already dispatches 8 of them via `dispatcher.Families()` + a streaming seam, keyed off `argv[0]` in `entrypoint.ClassifyInvocation`. We (1) finish the subcommand surface — reorganized so the crowded families nest (`pair review open`, `pair scrollback render`, …), (2) rewrite every call site **we own** from the standalone name to `pair <sub>`, then (3) collapse the build to one binary and stop bundling helper binaries. The runtime bundle keeps expanding **config/assets only** (zellij KDL, nvim Lua, the two shell shims) — never binaries, because a binary is always reachable as `pair <sub>`. `pair` is on the session PATH already (the session inherits the launching shell's PATH, and `pathenv.go` prepends rather than replaces); no symlink bridge and nothing extra is written into the content-addressed runtime store.

**Tech Stack:** Go 1.2x (`cmd/`, `cmd/internal/*`), GNU Make (`Makefile.local`), zellij KDL (`zellij/*.kdl`), Neovim Lua (`nvim/*.lua`), embedded runtime bundle (`cmd/internal/runtimebundle` + `runtimebundlegen`).

**ARCH principles cited:** `ARCH-DRY` (the binary set is currently restated in 5 places — `GO_BINS`, `RUNTIMEBUNDLE_HELPERS`, `.PHONY`, per-binary Makefile recipes, and `runtimebundlegen/generate.go`; this collapses to one source, `dispatcher.Families()`), `ARCH-PURPOSE` (the *point* is one binary with every owned consumer deriving from `pair <sub>` — stopping at symlinks or leaving `pair-scribe` out would be the "easy subset"), `ARCH-PURE` (the `RunXxxCLI` seams are already pure with thin `main` shells; the `argv[0]`→subcommand map and the group/leaf dispatch stay pure functions).

---

## Motivation (the concrete waste, measured)

- **19 binaries** in `GO_BINS`; `make build` links all of them. `pair-dev` re-runs `make build` on **every launch and every in-session restart**, so the whole link cost is paid constantly.
- `bin/pair` and `bin/pair-go` are the **same program** (identical source `./cmd/pair-go`, identical ldflags) linked twice into two 81 MB files.
- `pair` embeds **16 helper binaries** in its runtime bundle (`runtimebundlegen/generate.go:19-42`; the kept shell shims `bin/pair-help`, `bin/pair-notify` are not helpers). At ~3–4 MB each that is ~55–65 MB of `pair`'s 81 MB. Dropping them from the manifest should shrink `pair` to ~20–25 MB **and** cut the per-change link cost to a single binary.

## Where we already are (do not re-derive)

- `pair <sub>` already works for: `context`, `scrollback-render`, `slug`, `wrap`, `scribe`, `changelog`, `continuation`, `session-watch` (`dispatcher.Families()`, `dispatcher.go:35-46`; streaming seam `cmd/pair-go/main.go:runStreamingSubcommand`).
- Every remaining standalone binary is a thin shim over a ready `RunXxxCLI`:
  - `pair-title` → `titlepoller.RunCLI` · `pair-scrollback-open` → `opener.RunScrollbackCLI` · `pair-changelog-open` → `opener.RunChangelogCLI`
  - `pair-review-target` → `reviewcmd.RunTargetCLI` · `pair-review-open` → `reviewcmd.RunOpenCLI` · `pair-review-readiness` → `reviewcmd.RunReadinessCLI`
  - `copy-on-select` → `clipcmd.RunCopyOnSelectCLI` · `clipboard-to-pane` → `clipcmd.RunClipboardToPaneCLI` · `flash-pane` → `clipcmd.RunFlashPaneCLI`
- `entrypoint.ClassifyInvocation` already dispatches on `filepath.Base(executable)` — the busybox hook exists.
- **zellij accepts the two-token form** (verified against zellij 0.44.3 source): `copy_command` splits on whitespace → `Command::new(first).args(rest)` (`zellij-server/src/tab/copy_command.rs`); the `Run` keybind action is `Run "cmd" "arg1" "arg2" { options }` (positional arg strings, options in the block). So `copy_command "pair clip copy-on-select"` and `Run "pair" "scrollback" "open" { … }` both work — **no durable busybox symlink is needed for any zellij caller.**

## Target subcommand surface (reorg: nest families, keep member names)

**Launcher verbs** (human-facing, `ModePublicPair` — unchanged): `pair [agent]`, `resume <tag>`, `rename`, `continue`, `restart`, `quit`, `list`/`ls`, `help`.

**Helper subcommands:**
- Flat: `wrap`, `scribe`, `session-watch`, `title`, `context`, `slug`, `continuation`.
- `pair review target | open | readiness`
- `pair scrollback render | open`  (was `scrollback-render`, `scrollback-open`)
- `pair changelog render | open`   (was `changelog`, `changelog-open`)
- `pair clip copy-on-select | clipboard-to-pane | flash-pane`

The four nested groups replace 9 flat names, so `pair --help`'s helper surface drops from 17 to ~11 top-level entries.

## Authoritative consumer map (from the design sweep)

**zellij bare-name execs (files we own; rewrite to `pair <sub>` in M2 — no symlink needed):**
| binary | call site(s) | new form |
|---|---|---|
| `pair-wrap` | `zellij/layouts/main.kdl:45` (`exec`, inside `sh -c`) + bundle mirror | `exec pair wrap …` |
| `copy-on-select` | `zellij/config.kdl:42` (`copy_command`) + mirror | `copy_command "pair clip copy-on-select"` |
| `pair-scrollback-open` | `zellij/config.kdl:186` (`Run`) + mirror; `nvim/init.lua:3515` (`zellij run --`) | `Run "pair" "scrollback" "open" {…}` / `…'pair','scrollback','open'…` |
| `pair-changelog-open` | `zellij/config.kdl:224` (`Run`) + mirror | `Run "pair" "changelog" "open" {…}` |

**Absolute-path / env-overridable execs (Go/Lua we own; rewrite to self-exec `pair <sub>` in M2):**
| binary | call site | new form |
|---|---|---|
| `pair-title` | `launcher/osruntime.go:295` (+ test `:256-257`, + poller guard — see Task 2.2) | `<selfExe> title …` |
| `pair-session-watch` | `launcher/osruntime.go:291` (+ test `:264-265`) | `<selfExe> session-watch …` |
| `copy-on-select` (self re-exec) | `clipcmd/run.go:76` | `<selfExe> clip copy-on-select --orchestrate` |
| `clipboard-to-pane` | `clipcmd/run.go:121` | `<selfExe> clip clipboard-to-pane` |
| `flash-pane` | `clipcmd/run.go:111` | `<selfExe> clip flash-pane` |
| `pair-review-open` | `nvim/init.lua:966-967` | `<home>/bin/pair` + `review open` |
| `pair-review-readiness` | `nvim/init.lua:917-919` (keep `PAIR_REVIEW_READINESS_BIN` override) | `<home>/bin/pair` + `review readiness` |
| `pair-continuation` | `.claude/settings.json:34`, `settings.local.json:25` (allowlist) | `bin/pair continuation` |
| `scrollback-render`, `changelog` (distiller, already `pair <sub>`) | `opener.go:135` (`$PCL_BIN scrollback-render|changelog`) | `$PCL_BIN scrollback render` / `changelog render` |

**Already no bare-name runtime caller (tests + Makefile + bundle list only):** `pair-review-target` (nvim writes its JSON via `writefile`, `nvim/init.lua:907`), `pair-context`.

**External / keep:** the three shell shims (`pair-dev`, `pair-help`, `pair-notify`). `pair-slug`: external Claude **Stop hook** in the user's *global* config (not in-repo) — keep **one** `pair-slug`→`pair` busybox symlink until confirmed it calls `pair slug` (the in-repo caller `wrap.go:570` already spawns `pair slug`). `pair-scribe`: **folds in** — update the user's `~/.zshrc` to `exec pair scribe` (out-of-repo action, note in `## Log`; `pair scribe` already works today).

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `busyboxSubcommand(base, valid) (sub string, ok bool)` | `cmd/internal/entrypoint/alias.go` | new |
| `ClassifyInvocation` / `ResolveInvocation` (busybox + arg rewrite) | `cmd/internal/entrypoint/mode.go` | modified |
| group/leaf dispatch (nested command parse) | `cmd/internal/dispatcher/dispatcher.go` | modified |
| `dispatcher.Families()` (single source of every subcommand) | `cmd/internal/dispatcher/dispatcher.go` | modified |
| `runtimebundlegen` bundled-file list (config + shims only) | `cmd/internal/runtimebundlegen/generate.go` | modified |

- **`busyboxSubcommand`** — pure prefix-strip map from an invoked base name to a **flat** subcommand, validated against `dispatcher.DispatchNames()`. Only real surviving need is `pair-slug`→`slug`; the nested-group families are never reached by a busybox name (their callers are rewritten in M2 and the binaries survive until M3), so this stays a simple `strings.TrimPrefix(base,"pair-")` + validity check. `pair`/`pair-go`/`pair-dev`/`pair-scribe`/`pair-notify`/`pair-help` return `ok=false`.
  - **DRY rationale (ARCH-DRY):** replaces N `main.go` shims; validity derives from `DispatchNames()`, one source.
- **group/leaf dispatch** — `Dispatch`/the streaming seam gain one level: for a group name (`review`/`scrollback`/`changelog`/`clip`) the router reads `args[1]` as the leaf and calls the matching `RunXxxCLI`; flat names route as today. Old flat aliases (`scrollback-render`, `changelog`) are **retained** through M2 (the distiller still calls them until Task 2.4) and removed in M3.
  - **DRY rationale:** the group→leaf table is the *only* place the nesting is expressed; `busyboxSubcommand`, help text, and the Makefile all derive from `Families()`.
- **`dispatcher.Families()`** — rows carry a hierarchical name (group + leaf, e.g. `{Group:"review", Leaf:"open"}` or a `"review open"` path string) and correct `Streaming`. Adds `title`, `scrollback open`, `changelog open`, `review target|open|readiness`, `clip copy-on-select|clipboard-to-pane|flash-pane`; renames `scrollback-render`→`scrollback render`, `changelog`→`changelog render` (keeping the old flat names as transitional aliases). `session-watch` already exists.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| helper spawn argv | `launcher/osruntime.go:291,295` | modified | `exec` of self as `<self> <sub>` |
| clip self/sibling exec | `clipcmd/run.go:76,111,121` | modified | `exec` of self as `<self> clip <leaf>` |
| distiller helper exec | `opener/opener.go:135` | modified | `$PCL_BIN <group> <leaf>` |
| nvim helper calls | `nvim/init.lua:919,967,3515` | modified | `vim.fn.system` / `zellij run` |
| zellij pane + keybind commands | `zellij/config.kdl:42,186,224`, `layouts/main.kdl:45` (+ bundle mirror) | modified | zellij spawn (two-token `pair <sub>`) |
| session PATH → `pair` | `launcher/pathenv.go` (+ createflow) | modified | session `$PATH` |
| bundle file list | `runtimebundlegen/generate.go:19-42` | modified | extracted runtime (config + shims, **no binaries**) |
| agent-shell allowlist | `.claude/settings.json`, `settings.local.json` | modified | permission match string |

- **helper spawn / clip / distiller exec** — resolve the running executable (`os.Executable()`, behind the launcher's OS-runtime seam) and spawn `<self> <sub…>`. Pure decision logic (which helper, which args) is untouched; only the argv the seam produces changes. Injected into the existing `OSRuntime`/`clipcmd` fakes — assert the argv now leads with self-exec + subcommand.
- **session PATH → `pair`** — *the* mechanism, and it's minimal: `pair` is already on the session PATH because the session inherits the launching shell's PATH (where the installed `pair` lives) and `pathenv.go` **prepends** `$PAIR_HOME/bin` rather than replacing PATH. Belt-and-suspenders for the absolute-path-invocation edge case: also prepend `dir(os.Executable())`. **No symlink is written into the content-addressed store** (so no drift/prune concern), and **no binary is expanded** — the extracted runtime carries config + the two shell shims only.

---

## Milestones (review boundaries)

- **M1** — Complete + reorganize the subcommand surface: fold the remaining families into `dispatcher.Families()` with group/leaf nesting, add the streaming routes, and add the (minimal) busybox `argv[0]` prefix-strip. **No consumer changes**; every standalone binary still builds; old flat names retained. Fully backward compatible.
- **M2** — Rewrite every call site we own to `pair <sub>` (launcher Go + title-poller guard, clipcmd, distiller, nvim, `.claude`, zellij KDL + bundle mirror), family-by-family. Helpers still build (so any not-yet-migrated bare name still resolves), so each commit is green. Ends with no in-repo caller using a standalone helper name or an old flat name.
- **M3** — Collapse `GO_BINS := pair`; delete the `pair-go` output; drop the 16 helper binaries from `runtimebundlegen/generate.go` (bundle = config + shims, `pair` shrinks); guarantee `pair`-on-PATH; keep only the `pair-slug` symlink; delete the `cmd/<helper>` dirs + per-binary Makefile recipes; remove old flat aliases; measure the size + build-time drop. Atlas/README/AGENTS updated at close.

Estimate lives in issue frontmatter (`estimate_hours`), set at `sdlc change-code`.

---

## Chunk 1: M1 — subcommand surface + reorg + busybox dispatch

### Task 1.1: `busyboxSubcommand` + `ClassifyInvocation` extension
**Files:** Create `cmd/internal/entrypoint/alias.go` (+ `alias_test.go`); modify `cmd/internal/entrypoint/mode.go` (+ `mode_test.go`).
- [ ] **Step 1 (failing test):** `busyboxSubcommand("pair-slug", valid)` → `("slug", true)`; `("pair-title",…)`→`("title",true)`; `("pair",…)`/`("pair-scribe",…)`/`("randomtool",…)`→`("",false)`; `("pair-frob",…)` (not implemented)→`("",false)`. Run → FAIL.
- [ ] **Step 2 (implement):** `strings.TrimPrefix(base,"pair-")`, reject the `nonBusybox` set (`pair`, `pair-go`, `pair-dev`, `pair-scribe`, `pair-help`, `pair-notify`), validate against `valid` (= `dispatcher.DispatchNames()`). Run → PASS.
- [ ] **Step 3:** Extend the entrypoint so a busybox base routes to dispatch with the subcommand prepended to argv. Prefer a new `ResolveInvocation(exe, args, dispatchNames) (EntrypointMode, []string)` returning rewritten args; keep `ClassifyInvocation` for existing callers/tests. Add `mode_test.go`: `{"busybox pair-slug","/x/bin/pair-slug",nil}` → `ModeDispatch`, args `["slug"]`.
- [ ] **Step 4:** `go test ./cmd/internal/entrypoint/ -count=1` → PASS. **Commit** `#104 M1: busybox argv[0]->subcommand resolution (ARCH-PURE, ARCH-DRY)`.

### Task 1.2: group/leaf dispatch + fold remaining families
**Files:** Modify `cmd/internal/dispatcher/dispatcher.go` (+ `dispatcher_test.go`), `cmd/pair-go/main.go` (streaming seam + imports; + `main_test.go`).

Streaming vs buffered split:
- **Streaming:** `title` (long-lived poller); `clip copy-on-select` (reads the selection from **stdin** — `RunCopyOnSelectCLI(args, stdin, …)`, `clipcmd/run.go:86`; the buffered `Dispatch` passes no stdin; the hook itself returns fast, #100, so it's the stdin dependency, not lifetime, that forces the streaming seam). `session-watch` already wired.
- **Buffered:** `scrollback open`, `changelog open`, `review target|open|readiness`, `clip clipboard-to-pane`, `clip flash-pane`. (Openers stay safe buffered — `opener.RunViewer` wires nvim to `os.Stdin/Stdout/Stderr` directly, `opener/runtime.go:151-153`, bypassing the injected buffer.)

- [ ] **Step 1 (failing tests):** extend `TestDispatchNamesDeriveFromImplementedStatus` for the new leaves; add a group/leaf routing test per family (`pair review open`, `pair scrollback render`, `pair clip copy-on-select`, …) asserting it reaches the right `RunXxxCLI` (mirror `TestDispatchContextReturnsHelperOutput`); assert the **old flat aliases** `scrollback-render`/`changelog` still route (transitional). Run → FAIL.
- [ ] **Step 2:** add group/leaf parsing to `Dispatch` (and the streaming seam in `cmd/pair-go/main.go`): if `args[0]` ∈ {`review`,`scrollback`,`changelog`,`clip`}, dispatch on `args[1]`; else flat as today. Add `Families()` rows (hierarchical name + `Streaming`); keep `scrollback-render`/`changelog` flat aliases mapping to `scrollback render`/`changelog render`. Import `titlepoller`, `opener`, `reviewcmd`, `clipcmd`.
- [ ] **Step 3:** `go test ./cmd/internal/dispatcher/ ./cmd/pair-go/ -count=1` → PASS. `go build ./cmd/pair-go` and smoke: `bin/pair review open --help`, `bin/pair scrollback render --help`, `bin/pair clip copy-on-select --help`, `bin/pair scrollback-render --help` (alias) behave correctly. **Commit** `#104 M1: nest review/scrollback/changelog/clip families in pair dispatcher`.

### Task 1.3: M1 close
- [ ] `env -u PAIR_SESSION_ID PAIR_TAG make test` (full suite — partial verification is caught as REWORK; note the pre-existing `parley_harness_golden` failure). Standalone binaries still build/pass; `pair <sub>` (nested) now covers every helper.
- [ ] `sdlc milestone-close --issue 104 --milestone M1 …` (fix Critical/Important from the auto-review first).

---

## Chunk 2: M2 — rewrite every owned caller to `pair <sub>`

Helpers **stay built** (GO_BINS unchanged) through M2, so any not-yet-migrated bare name still resolves — each commit is green. Rewrite family-by-family.

### Task 2.1: launcher-spawned helpers + `os.Executable()` seam
**Files:** `launcher/osruntime.go:291,295` (+ `osruntime_test.go:256-265`); add a self-executable accessor to the OS-runtime seam if absent.
- [ ] Change the argv builders to `{selfExe,"session-watch",…}` and `{selfExe,"title",…}`. Update pinned-path tests to expect the self-exec form. `go test ./cmd/internal/launcher/ -count=1` → PASS. Commit.

### Task 2.2: title-poller single-instance guard (blocking coupling)
**Files:** `cmd/internal/titlepoller/titlepoller.go:96-100` (`pollerArgvMatches`), `titlepoller_test.go:70`, `run_test.go:178`.
- [ ] `pollerArgvMatches` matches the **literal substring** `"pair-title "+tag+" "` against the running process argv (the `osruntime.go:288-289` comment warns about this coupling). Task 2.1's self-exec makes the argv `.../pair title <tag> …`, which no longer contains `"pair-title "` — the single-instance guard would stop matching → **duplicate pollers spawn on every attach**. Write a failing test for the new `pair title <tag> ` argv form; rewrite the guard to match `"pair title "+tag+" "` (or a form-agnostic `title`+tag match); update both tests. `go test ./cmd/internal/titlepoller/ -count=1` → PASS. Commit. (Co-locate with 2.1 if implemented together; either way the guard must land in the same commit as the title self-exec.)

### Task 2.3: clip self/sibling exec
**Files:** `clipcmd/run.go:76,111,121` (+ `run_test.go`).
- [ ] Replace the three `PairHome+"/bin/<name>"` joins with self-exec `<selfExe> clip copy-on-select --orchestrate` / `clip flash-pane` / `clip clipboard-to-pane`. Update fakes/tests. `go test ./cmd/internal/clipcmd/ -count=1` → PASS. Commit.

### Task 2.4: distiller (already `pair <sub>`, now nested)
**Files:** `opener/opener.go:135` (+ `opener` tests).
- [ ] `$PCL_BIN scrollback-render` → `$PCL_BIN scrollback render`; `$PCL_BIN changelog` → `$PCL_BIN changelog render`. `go test ./cmd/internal/opener/ -count=1` → PASS. (After this, the old flat aliases have no in-repo caller — they're removed in M3.) Commit.

### Task 2.5: nvim review helpers
**Files:** `nvim/init.lua:917-919` (readiness), `:966-967` (open).
- [ ] Change resolved `bin` to `home.."/bin/pair"` with args `{'review','open',…}` / `{'review','readiness',…}`; preserve the `PAIR_REVIEW_READINESS_BIN` override (now points at a `pair` that takes `review readiness`). Update `tests/review-readiness-cli-test.sh`, `tests/review-window-test.sh`, `tests/review-toggle-test.sh` if they assert the invoked name. Commit.

### Task 2.6: zellij KDL + nvim scrollback-open + bundle mirror
**Files:** `zellij/config.kdl:42,186,224`, `layouts/main.kdl:45`, `nvim/init.lua:3515`; regenerate mirror under `cmd/internal/runtimebundle/assets/runtime/files/zellij/…`.
- [ ] `copy_command "copy-on-select"` → `copy_command "pair clip copy-on-select"`. `Run "pair-scrollback-open" {…}` → `Run "pair" "scrollback" "open" {…}`; `Run "pair-changelog-open" {…}` → `Run "pair" "changelog" "open" {…}` (positional arg strings after the command — verified zellij 0.44.3 syntax). `layouts/main.kdl:45` `exec pair-wrap …` → `exec pair wrap …`. `nvim/init.lua:3515` `'--','pair-scrollback-open',…` → `'--','pair','scrollback','open',…`.
- [ ] `make runtimebundle-generate` to sync the mirror. Update `.claude/settings.json`/`settings.local.json`: `bin/pair-continuation` → `bin/pair continuation`, and the `bin/pair-wrap …` grants (`settings.json:21-24`, `settings.local.json:12-15`) → `bin/pair wrap …`.
- [ ] **Manual verify each rebinding in a live `pair-dev` session** (agent pane launch; Alt-select copy; scrollback PageUp; changelog keybind) — automate via `tests/scrollback-open-test.sh`, `tests/changelog-open-test.sh`, `tests/copy-on-select-test.sh` updated to drive `pair <sub>`. Commit.

### Task 2.7: M2 close
- [ ] `env -u PAIR_SESSION_ID PAIR_TAG make test` green; full live-session smoke. No in-repo caller uses a standalone helper name or old flat name.
- [ ] `sdlc milestone-close --issue 104 --milestone M2 …`.

---

## Chunk 3: M3 — one binary, stop bundling, cleanup

### Task 3.1: collapse the build + delete the twin + `pair`-on-PATH
**Files:** `Makefile.local` (`.PHONY`, `GO_BINS`, `RUNTIMEBUNDLE_HELPERS`, per-binary recipes/aliases, `build`, `install`); `launcher/pathenv.go` (+ createflow); `tests/pair-go-install-layout-test.sh`.
- [ ] `GO_BINS := pair`. Remove the `pair-go` output/alias/recipe (keep the `cmd/pair-go` *package* — it's the source for `pair`; a later cleanup may rename the dir to `cmd/pair`, out of scope). Remove every per-helper recipe stanza (`OPENER_SRCS`/`REVIEW_SRCS`/`CLIP_SRCS` groups, etc.) and the per-helper `.PHONY`/alias entries.
- [ ] Add exactly one busybox symlink in `install` (and `build`, for the dev tree): `ln -sf pair pair-slug` — the external Stop hook. Nothing else.
- [ ] `pair`-on-PATH: in `pathenv.go`/createflow, additionally prepend `dir(os.Executable())` to the session PATH (idempotent, like the existing `$PAIR_HOME/bin` prepend) so `pair` resolves even if launched by an absolute path off PATH. Unit-test the pure PATH-assembly.
- [ ] **Migrate** `tests/pair-go-install-layout-test.sh` (asserts `bin/pair-go` exists / runs `pair-go launch`) to exercise the `pair` public launcher; rename target/file if the `pair-go` name no longer fits.
- [ ] `go clean -cache && time make build` → only `bin/pair` (+ the `pair-slug` symlink); record timing vs the ~24 s / 19-binary baseline in `## Log`. Commit `#104 M3: single pair binary; drop pair-go twin (ARCH-DRY)`.

### Task 3.2: stop bundling binaries + shrink `pair`
**Files:** `runtimebundlegen/generate.go:19-42` (+ `generate_test.go`), `cmd/internal/runtimebundle/*_test.go`, `tests/pair-embedded-runtime-test.sh`.
- [ ] Remove the 16 helper-binary entries from the bundle list; keep `bin/pair-help`, `bin/pair-notify`, `bin/lib`, and the zellij/nvim assets. `bin/pair` stays excluded (never self-embed). `make runtimebundle-generate` → confirm `manifest.json` lists no helper binaries. Update `generate_test.go` expectations.
- [ ] Adjust `tests/pair-embedded-runtime-test.sh`: after extract, the runtime carries config + shims (no helper binaries); a session started from the extracted root reaches `pair <sub>` via the inherited PATH + `dir(exe)` prepend (assert no helper binary is required in the store, and nothing is written into the content-addressed store beyond the manifest). Run → PASS.
- [ ] **Measure:** rebuild `pair`, record new `bin/pair` size vs 81 MB in `## Log` (expect ~20–25 MB). Commit.

### Task 3.3: delete the shims + old flat aliases + docs
**Files:** delete `cmd/pair-title`, `cmd/pair-scrollback-open`, `cmd/pair-changelog-open`, `cmd/pair-review-{target,open,readiness}`, `cmd/copy-on-select`, `cmd/clipboard-to-pane`, `cmd/flash-pane`, `cmd/pair-context`, `cmd/pair-slug`, `cmd/pair-scrollback-render`, `cmd/pair-changelog`, `cmd/pair-continuation`, `cmd/pair-session-watch`, `cmd/pair-wrap`, `cmd/pair-scribe` (**pair-scribe folds in** — the user updates `~/.zshrc` to `exec pair scribe`); `dispatcher.go` (drop the `scrollback-render`/`changelog` flat aliases now that Task 2.4 migrated the caller); `atlas/`, `README.md`, `AGENTS.md`/lessons.
- [ ] Delete the thin-shim `cmd/<helper>` dirs (their logic lives in `cmd/internal/*cmd`). `go build ./... && go vet ./...` clean.
- [ ] Remove the transitional `scrollback-render`/`changelog` flat aliases from `Families()`/dispatch (only `scrollback render`/`changelog render` remain). Update `dispatcher_test.go`.
- [ ] **`~/.zshrc`** (out-of-repo, user action — note in `## Log`): `exec pair-scribe` → `exec pair scribe`. Confirm `pair scribe` behaves identically.
- [ ] **`pair-slug` Stop hook** (out-of-repo, user's global Claude config): verify whether it calls `pair-slug` or `pair slug`; if the latter, drop the last `pair-slug` symlink too. Note the outcome in `## Log`.
- [ ] Update `atlas/` (binary surface: "one `pair`; subcommands via `dispatcher.Families()`; only other artifacts are the 3 shell shims"), keep `atlas/index.md` linking. `README.md`: any `pair-<x>` usage → `pair <sub>`. Add an `AGENTS.md`/`lessons.md` rule if one emerged.

### Task 3.4: M3 close + ARCH-PURPOSE shadow-sweep
- [ ] **Shadow-sweep:** enumerate all former helper names; confirm each is either reached as `pair <sub>` by every owned caller, or is `pair-slug` (external, symlink or verified). No hand-maintained redundant binary remains; the binary set is single-sourced on `dispatcher.Families()`.
- [ ] `env -u PAIR_SESSION_ID PAIR_TAG make test` fully green; full live-session smoke.
- [ ] `sdlc close --issue 104 --milestone M3 --verified '<make test green; live smoke of agent pane/copy/scrollback/changelog keybinds + pair scribe; bin/pair size N MB down from 81 MB; single-binary make build timing>'` (omit `--actual` so close computes it).

---

## Risks & guards

- **Deployed `pair`-on-PATH** (Task 3.1/3.2): the one behavioral change. Gate it with `tests/pair-embedded-runtime-test.sh` + a real brew-layout smoke. Fallback if some install path narrows PATH: the `dir(os.Executable())` prepend already covers absolute-path invocation.
- **Title-poller guard** (Task 2.2): must land in the same commit as the title self-exec, or duplicate pollers spawn. Test-gated.
- **`pair-slug` external hook / `pair-scribe` zshrc** (Task 3.3): the only out-of-repo call sites — do not assume; `pair-slug` keeps a symlink until verified, `pair scribe` is confirmed working before the zshrc edit.
- **Green-at-every-commit** (M2): helpers stay built until M3, and nested names are added (M1) before old flat names are removed (M3), so no window breaks a caller.
- **zellij two-token form:** verified against 0.44.3 source — no symlink escape hatch needed. (If a future zellij changes `copy_command`/`Run` parsing, the busybox mechanism from M1 is still available.)

---

## Revisions

### 2026-07-06 — v2 (post-brainstorm refinements)
**Reason:** user decisions after the v1 review — fold `pair-scribe` in too, reorganize the crowded subcommand surface, and simplify the deployed-runtime mechanism.
**Deltas from v1:**
- **`pair-scribe` folds in.** End state is *one* binary + the three shell shims; `~/.zshrc` moves to `exec pair scribe`. (v1 kept `pair-scribe` as a standalone.)
- **Reorg — nest families, keep member names.** New nested groups `pair review|scrollback|changelog|clip <leaf>` replace 9 flat names; dispatcher gains one level of group/leaf parsing; `scrollback-render`/`changelog` keep transitional flat aliases (removed in M3).
- **PATH mechanism simplified.** v1 symlinked the running `pair` into the extracted runtime `bin/` (a bridge) and worried about drift/prune. v2: **stop expanding binaries entirely** (bundle = config + shims), rely on `pair` already being on the inherited session PATH (`pathenv.go` prepends, doesn't replace), plus a `dir(os.Executable())` prepend for robustness. Nothing is written into the content-addressed store → the drift/prune concern is gone. Content-addressed caching (skip re-extract on hash match) already exists — no new work.
- **Milestones restructured.** v1: M1 surface+argv0 · M2 one-binary+symlink-scaffold · M3 rewrite callers. v2: M1 surface+reorg+argv0 · **M2 rewrite callers** (helpers stay built, green per commit) · **M3 collapse build + stop bundling + cleanup**. The symlink scaffold is dropped (only the external `pair-slug` symlink survives); the compile/size win lands at M3.
- **zellij confirmed.** Verified against zellij 0.44.3 source that `copy_command` and `Run` accept the two-token `pair <sub>` form (whitespace-split, no shell) — the v1 "durable symlink escape hatch" for zellij is removed, and the `Run` syntax corrected to positional arg strings.

### 2026-07-06 — M2 close: `.claude` allowlist left as historical grants
Task 2.6's directive to rewrite `.claude/settings.json`/`settings.local.json` `bin/pair-wrap`/`bin/pair-continuation` entries to `bin/pair <sub>` was intentionally **not** performed: these are exact-match Claude-Code Bash-tool permission grants (frozen — one bears a hardcoded `session-id`), not runtime consumers, so they are not part of the `pair <sub>` consumer set and leaving them breaks nothing. Recorded in `atlas/go-migration-inventory.md`. The M2 "every owned caller derives from `pair <sub>`" purpose is unaffected. (Ratifies the M2 boundary-review Important finding.)
