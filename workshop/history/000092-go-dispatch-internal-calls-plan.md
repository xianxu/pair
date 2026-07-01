# Route Internal Calls Through the Go Dispatcher — Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the Pair-owned internal helpers (`slug`, `changelog`, `continuation`, `session-watch`) reachable as `pair <subcommand>` through the Go dispatcher, and repoint every Pair-owned call-site to that form — with each standalone `bin/pair-<name>` binary reduced to a thin shim over a shared runner package.

**Architecture:** Each helper's logic moves into a reusable `cmd/internal/<name>cmd` runner package (the `contextcmd`/`scrollbackcmd` pattern #76 established); both the standalone binary (now a 3-line shim) and the dispatcher call the same runner (ARCH-DRY). The public `pair` entrypoint is extended to *peel off* a reserved set of dispatcher subcommands before handing off to the shell launcher — so `pair slug` dispatches while `pair claude` / `pair resume` still launch a session. Finite, no-stdin helpers (`slug`, `changelog`) use the existing **buffered** `Dispatch(args) → Result` path; helpers that need real stdin or run long (`continuation`, `session-watch`) use a small new **streaming** dispatch seam in `cmd/pair-go/main.go` that hands the runner `os.Stdin/Stdout/Stderr` directly.

**Tech Stack:** Go (stdlib `flag`, `io`, `os/exec`), the existing `cmd/internal/{model,transcript,adapt,sessionwatch}` packages, `bin/*.sh` POSIX shell, `nvim/*.lua`, `Makefile.local` build rules, `cmd/internal/runtimebundlegen` manifest.

---

## Scope

**In scope (this issue, #92):** dispatcher routes + shared runners + thin shims for `slug`, `changelog`, `continuation`, `session-watch`; the `pair <sub>` entrypoint peel-off; repointing Pair-owned call-sites; tests.

**Out of scope:** `pair-wrap` and `pair-scribe` (interactive PTY proxies) → **#96** (reuses this issue's streaming seam). Porting shell orchestrators → **#93**. Physically removing the standalone helper binaries / shrinking the runtime bundle → later (they stay as thin shims here; external callers and health tests still reference the names).

`context` and `scrollback-render` are already routed (#76); this issue makes them reachable as `pair context` / `pair scrollback-render` via the peel-off and repoints their call-sites (shadow-sweep, ARCH-PURPOSE).

---

## Core Concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `CommandFamily` (+ one new `Streaming bool`) | `cmd/internal/dispatcher/dispatcher.go` | modified |
| `DispatchNames()` / `StreamingNames()` / `IsStreaming(name)` (derive reserved sets from `Families()`) | `cmd/internal/dispatcher/dispatcher.go` | new |
| `ClassifyInvocation` (peel reserved subcommands off `pair`) | `cmd/internal/entrypoint/mode.go` | modified |
| slug derivation core (`slug.go` + tests) | `cmd/internal/slugcmd/` (moved from `cmd/pair-slug/`) | modified (relocated) |
| changelog distill core (`distill.go`, `prompt.go` + tests) | `cmd/internal/changelogcmd/` (moved from `cmd/pair-changelog/`) | modified (relocated) |
| continuation core (`continuation.go` validators/allocator + tests) | `cmd/internal/continuationcmd/` (moved from `cmd/pair-continuation/`) | modified (relocated) |

- **`CommandFamily` routing metadata (single source, no duplication).** `Families()` is already the one list of dispatcher subcommands. Implemented-ness is **already** encoded by the existing `Status` field (`"implemented"` vs `"planned"`/`"handoff"`) — do NOT add a parallel `Implemented bool` (that was flagged as two-sources drift). Add only the one genuinely-new, orthogonal axis: `Streaming bool` (a subcommand can be `implemented` yet buffered OR streaming). Then:
  - `DispatchNames()` = names where `Status=="implemented"` — derived from `Status`, the SAME field `Help()` already branches on (`dispatcher.go:103-114`), so no second source.
  - `StreamingNames()` / `IsStreaming(name)` = derived from the `Streaming` field.
  - **DRY rationale:** without this the reserved set would be re-listed in `entrypoint`, `main.go`, and help. One `Families()` table drives all three. Keeping implemented-ness on `Status` (not a new bool) avoids the drift the reviewer caught.
  - **Future extensions:** #96 flips `wrap`/`scribe` `Status→"implemented"` + sets `Streaming:true` — no entrypoint edit needed.
  - **Guard (`dispatcher.go:64-68`):** once `continuation`/`session-watch` are `implemented` but streaming, they have NO buffered `switch` case, so a stray `dispatcher.Dispatch(["continuation"])` would hit the `familyByName` "planned but not implemented" branch — misleading. `Dispatch` is only reached for the buffered set (the streaming seam intercepts first in `main.go`), but add a clear guard/message so a mis-call says "streaming subcommand — invoke via the streaming seam," not "not implemented."

- **`ClassifyInvocation`** — currently returns `ModePublicPair` for *any* argv when the binary basename is `pair`. Modified to take the dispatch-name set and return `ModeDispatch` when `base=="pair" && len(args)>0 && isDispatchName(args[0])`; otherwise unchanged (agent names, `resume`, `continue`, `list`, `rename`, and bare `pair` still launch).
  - **Relationships:** consumes `DispatchNames()` passed in by `cmd/pair-go`. Passing the set in (rather than importing `dispatcher` from `entrypoint`) is a clean-decoupling *choice*, not a cycle fix — `dispatcher` doesn't import `entrypoint`, so the direct import would also be acyclic. Keeping `entrypoint` dependency-free is still preferable.
  - **DRY rationale:** one classification function already owns "which mode is this invocation"; this extends it rather than adding a parallel check in `main.go`.
  - **Future extensions:** if a subcommand ever needs to co-exist with an identically named agent, the peel-off is the one place to add an escape (e.g. `pair -- <agent>`).

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `slugcmd.Run(args, env, stdout, stderr) int` | `cmd/internal/slugcmd/slugcmd.go` | new | model API, transcript FS, `slug-proposed-<tag>` write |
| `changelogcmd.Run(args, stdout, stderr) int` | `cmd/internal/changelogcmd/changelogcmd.go` | new | model API, cleaned-TTY read, log/anchor atomic write |
| `continuationcmd.Run(args, stdin, stdout, stderr, now) int` | `cmd/internal/continuationcmd/continuationcmd.go` | new | stdin, `workshop/continuation/` FS, git commit/push |
| `runStreamingSubcommand` (streaming dispatch seam: `changelog`, `continuation`, `session-watch`) | `cmd/pair-go/main.go` | new | real `os.Stdin/Stdout/Stderr`, process exit |
| `dispatchSlug` (buffered route) | `cmd/internal/dispatcher/dispatcher.go` | new | `slugcmd.Run`, via `bytes.Buffer` |
| pair-slug / pair-changelog / pair-continuation shims | `cmd/pair-{slug,changelog,continuation}/main.go` | modified | `os.Exit(<name>cmd.Run(...))` |

**Buffered vs streaming — the split matters (I2):** buffered (`Dispatch(args) Result`) collects stdout/stderr and writes them only *after* the runner returns. That's correct for `slug` (output is files + `$PAIR_SLUG_LOG`; `pair-wrap` spawns it detached and discards output) and for the already-routed `context`/`scrollback-render`. It is **wrong** for `changelog`: it streams live per-batch progress to stderr (`main.go:120-122`) which `pair-changelog-open:85` redirects to `$STATUS` and `nvim/changelog.lua` tails for a spinner — buffering would delay all of it to the end and silently kill the spinner. So `changelog` joins `continuation` (stdin) and `session-watch` (long-running) in the **streaming** set (real `os.Stderr`), even though it has no stdin. "Finite/no-stdin" does not imply "no live stderr consumer."

- **`slugcmd.Run` / `changelogcmd.Run`** — thin IO seams over the (already pure) `slug.go` / `distill.go` cores. Signature mirrors `contextcmd.Run` (args + injected env/writers → exit code). `slugcmd.Run` → **buffered** route (`dispatchSlug`). `changelogcmd.Run` → **streaming** seam (needs live `os.Stderr` for the spinner, I2), so it is NOT given a buffered `dispatch*` wrapper.
  - **Injected into:** each standalone shim `main`, plus either `dispatchSlug` (slug) or `runStreamingSubcommand` (changelog). Same runner, two entry points.
  - **`slugcmd.Env`:** capture `HOME` too (`slug.go`/`main.go:179` reads `os.UserHomeDir()`), alongside `PAIR_TAG`/`PAIR_DATA_DIR`/`PAIR_AGENT`/`PAIR_SLUG_*`/cwd. Verify `OPENAI_API_KEY` actually threads through (it may be read inside `model.Run`, not `Env`).

- **`continuationcmd.Run`** — wraps `pair-continuation`'s existing injectable `run(args, now, stdin, stdout)`; exposed with a `Run(...) int` that parses flags and threads real stdin. Reads `--body-file -` from stdin → **streaming** path.
  - **Injected into:** the shim `main` and the streaming dispatch seam. `now func() time.Time` stays injected for the existing clock-fake tests.
  - **Future extensions:** #96's `wrap` route reuses the same streaming seam.

- **`runStreamingSubcommand(args, stdin, stdout, stderr) int`** — the seam in `cmd/pair-go/main.go`: switches on `args[0]` and calls the matching runner (`changelog`, `continuation`, `session-watch`) with the passed-through stdio, returning the exit code, bypassing the buffered `Dispatch(args) Result`. It takes `stdin io.Reader` as a **parameter** so the seam is unit-testable with a fake stdin; `run()`'s default branch supplies `os.Stdin` at the call site. This keeps `run()`/`runWithLegacyRuntime()` signatures unchanged (no threading a stdin param through their 9 existing call-sites, I4) while the seam itself stays testable (ARCH-PURE).
  - **Injected into:** n/a (top-level seam). Reached only when `dispatcher.IsStreaming(args[0])`.
  - **Future extensions:** where a subcommand needs a controlling TTY (wrap/scribe, #96), the same seam hands through real stdio.

---

## Chunk 1 — Milestone M1: dispatcher reachability + runner consolidation (backward-compatible)

M1 adds `pair <sub>` dispatch and the shared runners **without changing any caller** — every existing `pair-<name>` invocation keeps working (the binaries become shims), and `pair <sub>` now *also* works. Merge-safe on its own. Closes via `sdlc milestone-close` (M1 boundary review).

### Task 1: extend `Families()` with routing metadata + derived name sets

**Files:**
- Modify: `cmd/internal/dispatcher/dispatcher.go`
- Test: `cmd/internal/dispatcher/dispatcher_test.go`

- [ ] **Step 1: Write failing tests** for the new derivations:

```go
func TestDispatchNamesAreImplementedFamilies(t *testing.T) {
	names := DispatchNames() // implemented routes only
	want := []string{"context", "scrollback-render", "slug", "changelog", "continuation", "session-watch"}
	if !equalUnordered(names, want) {
		t.Fatalf("DispatchNames() = %v, want %v", names, want)
	}
}

func TestStreamingNames(t *testing.T) {
	for _, n := range []string{"changelog", "continuation", "session-watch"} {
		if !IsStreaming(n) {
			t.Fatalf("%s must be streaming (live stderr / stdin / long-running); got %v", n, StreamingNames())
		}
	}
	for _, n := range []string{"slug", "context", "scrollback-render"} {
		if IsStreaming(n) {
			t.Fatalf("%s is buffered; must not be streaming", n)
		}
	}
}
```

- [ ] **Step 2: Run to verify FAIL** — `go test ./cmd/internal/dispatcher -run 'DispatchNames|Streaming'` → undefined functions.

- [ ] **Step 3: Implement.** Do NOT add an `Implemented` bool — implemented-ness already lives on the existing `Status` field (I5). Add only `Streaming bool` to `CommandFamily`. In `Families()`: add the missing `session-watch` entry; flip `slug`/`changelog`/`continuation`/`session-watch` to `Status:"implemented"` (they become routed in this issue); set `Streaming:true` on `changelog`, `continuation`, `session-watch` (`wrap`/`scribe` stay `planned`, non-streaming). Add `DispatchNames()` (names where `Status=="implemented"` — same field `Help()` reads), `StreamingNames()`, and the `IsStreaming(name) bool` predicate (all iterating `Families()`).

- [ ] **Step 4: Run to verify PASS.**

- [ ] **Step 5: Commit** — `#92 M1: dispatcher families carry routing metadata`.

### Task 2: peel dispatcher subcommands off the `pair` entrypoint

**Files:**
- Modify: `cmd/internal/entrypoint/mode.go`
- Test: `cmd/internal/entrypoint/mode_test.go`

- [ ] **Step 1: Write failing tests** capturing the grammar:

```go
func TestClassify(t *testing.T) {
	names := []string{"slug", "changelog", "continuation", "session-watch", "context", "scrollback-render"}
	cases := []struct{ exe string; args []string; want EntrypointMode }{
		{"/x/bin/pair", []string{"slug"}, ModeDispatch},          // peeled off
		{"/x/bin/pair", []string{"changelog", "--log", "f"}, ModeDispatch},
		{"/x/bin/pair", []string{"claude"}, ModePublicPair},       // agent name → launcher
		{"/x/bin/pair", []string{"resume"}, ModePublicPair},       // launcher verb unchanged
		{"/x/bin/pair", nil, ModePublicPair},                      // bare pair
		{"/x/bin/pair-go", []string{"slug"}, ModeDispatch},        // pair-go unchanged
		{"/x/bin/pair-go", []string{"launch"}, ModePairGoLaunch},
	}
	for _, c := range cases {
		if got := ClassifyInvocation(c.exe, c.args, names); got != c.want {
			t.Errorf("Classify(%q,%v)=%v want %v", c.exe, c.args, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run to verify FAIL** — signature mismatch (`ClassifyInvocation` takes 2 args today).

- [ ] **Step 3: Implement.** Change signature to `ClassifyInvocation(executable string, args []string, dispatchNames []string) EntrypointMode`. When `base=="pair"`: if `len(args)>0` and `args[0]` is in `dispatchNames`, return `ModeDispatch`; else `ModePublicPair`. Leave the `pair-go`/`launch` branches unchanged.

- [ ] **Step 4: Update the caller** `cmd/pair-go/main.go:50` to pass `dispatcher.DispatchNames()`. Run `go build ./...` to catch the arity change.

- [ ] **Step 5: Run to verify PASS** — `go test ./cmd/internal/entrypoint`.

- [ ] **Step 6: Commit** — `#92 M1: pair peels dispatcher subcommands before launcher`.

### Task 3: extract `slugcmd` runner + thin shim

**Files:**
- Create: `cmd/internal/slugcmd/slugcmd.go` (new `Run`), move `cmd/pair-slug/slug.go`→`cmd/internal/slugcmd/slug.go`, `slug_test.go`, and the body of `main.go`→`slugcmd.go`
- Modify: `cmd/pair-slug/main.go` (→ shim), `Makefile.local` (build rule)
- Test: `cmd/internal/slugcmd/slugcmd_test.go` (relocated `main_test.go`)

- [ ] **Step 1: Move the pure core.** `git mv cmd/pair-slug/slug.go cmd/internal/slugcmd/slug.go` and its `slug_test.go`; change `package main`→`package slugcmd`. Leave `cmd/pair-slug/main_test.go` where it is (it `go build`s + execs the binary — a library package has no `main`, so moving it verbatim breaks compilation, I6); it now exercises the shim → `Run`, or rewrite it to call `slugcmd.Run` with buffers. (Reminder: `git mv` then edit — re-`git add` after editing the package line; see the git-mv memory note.)

- [ ] **Step 2: Write `slugcmd.Run`.** Move `main()`'s body into `func Run(args []string, env Env, stdout, stderr io.Writer) int`, converting every `os.Exit(n)`/`log.Fatal` into `return n`, and every `logf`/stderr write to the injected `stderr`. Add an `Env` struct + `EnvFromOS()` (mirror `contextcmd`) capturing `HOME` (`main.go:179` reads `os.UserHomeDir()`), `PAIR_TAG`, `PAIR_DATA_DIR`, `PAIR_AGENT`, `PAIR_SLUG_*`, and cwd. `OPENAI_API_KEY` is likely read inside `model.Run` — confirm whether it must be in `Env` or is picked up from the process env by the model layer. Keep behavior identical.

- [ ] **Step 3: Reduce the shim** `cmd/pair-slug/main.go` to:

```go
package main

import (
	"os"
	"github.com/xianxu/pair/cmd/internal/slugcmd"
)

func main() { os.Exit(slugcmd.Run(os.Args[1:], slugcmd.EnvFromOS(), os.Stdout, os.Stderr)) }
```

- [ ] **Step 4: Update `Makefile.local`.** Point the `$(BIN_DIR)/pair-slug` rule prereqs at `cmd/pair-slug/main.go cmd/internal/slugcmd/slugcmd.go cmd/internal/slugcmd/slug.go cmd/internal/model/model.go cmd/internal/transcript/transcript.go go.mod`.

- [ ] **Step 5: Run parity tests.** `go test ./cmd/internal/slugcmd ./cmd/pair-slug -count=1` (relocated pure tests pass unchanged behavior). For CI-checkable route parity, extend the existing automated equivalence harness `cmd/pair-go/helper_equivalence_test.go` (which already builds both binaries and diffs `pair-go context` ≡ `pair-context` for code/stdout/stderr) with a `pair-go slug` ≡ `pair-slug` case under a fake env — don't rely on a manual scratch harness (constitution §2: automate verification).

- [ ] **Step 6: Commit** — `#92 M1: extract slugcmd runner; pair-slug is a shim`.

### Task 4: buffered dispatch route for `slug`

**Files:**
- Modify: `cmd/internal/dispatcher/dispatcher.go`
- Test: `cmd/internal/dispatcher/dispatcher_test.go`

- [ ] **Step 1: Write failing test** — `Dispatch([]string{"slug", ...})` returns a `Result` (exit code from `slugcmd.Run`), and an unknown subcommand still errors with exit 2.

- [ ] **Step 2: Run FAIL.**

- [ ] **Step 3: Implement** `dispatchSlug(args)` (mirror `dispatchContext`: buffer stdout/stderr, call `slugcmd.Run(args, slugcmd.EnvFromOS(), &out, &errb)`), add `case "slug":` to the switch, flip its `Families()` status text to implemented.

- [ ] **Step 4: Run PASS.**

- [ ] **Step 5: Commit** — `#92 M1: route pair slug through the dispatcher`.

### Task 5: extract `changelogcmd` runner + thin shim + streaming seam (changelog)

**Files:**
- Create: `cmd/internal/changelogcmd/changelogcmd.go`; move `distill.go`, `prompt.go`, and their **pure** tests from `cmd/pair-changelog/`
- Modify: `cmd/pair-changelog/main.go` (→ shim), `cmd/pair-go/main.go` (introduce the streaming seam), `Makefile.local`
- Test: relocated `changelogcmd` pure tests + a `cmd/pair-go` seam test

- [ ] **Step 1: Move pure cores** `distill.go`/`prompt.go` (+ `distill_test.go`, `prompt_test.go`) → `cmd/internal/changelogcmd/`, `package changelogcmd`. **Do not** move `main_test.go`/`e2e_test.go` verbatim (they `go build` + exec the binary — a library package has no `main`, so a verbatim move breaks compilation, I6). Keep those in `cmd/pair-changelog/` exercising the shim, or rewrite them to call `changelogcmd.Run(...)` with buffers.

- [ ] **Step 2: Write `changelogcmd.Run(args, stdout, stderr) int`** from `main()`'s body — flag parsing (`--cleaned/--log/--anchor/--agent/--model`), threading `fail()`'s `os.Exit(1)` → `return 1`, progress lines → injected `stderr`.

- [ ] **Step 3: Reduce shim** `cmd/pair-changelog/main.go` → `os.Exit(changelogcmd.Run(os.Args[1:], os.Stdout, os.Stderr))`.

- [ ] **Step 4: Introduce the streaming seam** in `cmd/pair-go/main.go`. In the `default:` (`ModeDispatch`) branch — WITH the length guard (C1):

```go
default:
	if len(args) > 0 && dispatcher.IsStreaming(args[0]) {
		return runStreamingSubcommand(args, os.Stdin, stdout, stderr)
	}
	res := dispatcher.Dispatch(args)
	return writeResult(res, stdout, stderr)
```

`func runStreamingSubcommand(args []string, stdin io.Reader, stdout, stderr io.Writer) int` switches on `args[0]`: `"changelog"` → `changelogcmd.Run(args[1:], stdout, stderr)` (real `os.Stderr` → live spinner survives, I2). `continuation`/`session-watch` cases added in Tasks 6/7. Note: `run()`/`runWithLegacyRuntime()` signatures are **unchanged** — `os.Stdin` is read only here, at the one call site (I4).

- [ ] **Step 5: Write the seam test** (`cmd/pair-go/main_test.go`): `runStreamingSubcommand([]string{"changelog","--cleaned",…}, strings.NewReader(""), &out, &errb)` writes progress to `errb` incrementally (proves real-stderr routing, not post-hoc buffering).

- [ ] **Step 6: Update `Makefile.local`** `$(BIN_DIR)/pair-changelog` prereqs to include the new package files.

- [ ] **Step 7: Run** `go test ./cmd/internal/changelogcmd ./cmd/pair-changelog ./cmd/pair-go -count=1` — PASS.

- [ ] **Step 8: Commit** — `#92 M1: extract changelogcmd; stream pair changelog`.

### Task 6: extract `continuationcmd` runner + thin shim + streaming seam

**Files:**
- Create: `cmd/internal/continuationcmd/continuationcmd.go`; move `continuation.go`, `git.go`, tests from `cmd/pair-continuation/`
- Modify: `cmd/pair-continuation/main.go` (→ shim), `cmd/pair-go/main.go` (streaming seam), `Makefile.local`
- Test: relocated `continuationcmd` tests + a new streaming-seam test in `cmd/pair-go`

- [ ] **Step 1: Move** `continuation.go`, `git.go`, and the pure `continuation_test.go` → `cmd/internal/continuationcmd/`, `package continuationcmd`. Move `run(...)`, `runArgs`, `readBody`, helpers too. Keep `main_test.go` (builds + execs the binary, I6) in `cmd/pair-continuation/` or rewrite it against `continuationcmd.Run`.

- [ ] **Step 2: Write `continuationcmd.Run(args []string, stdin io.Reader, stdout, stderr io.Writer, now func() time.Time) int`** — parse flags into `runArgs`, call the existing injectable `run(a, now, stdin, stdout)` (already `main.go:45`), map its `error` → `stderr` + `return 1`, nil → `return 0`. `now`/`stdin` stay injected (existing clock-fake + stdin tests move over). (Note: `run`'s push-failure warning writes `os.Stderr` directly at `main.go:108` — fine under the streaming seam's real `os.Stderr`; tighten to the injected writer only if convenient.)

- [ ] **Step 3: Reduce shim** `cmd/pair-continuation/main.go` → `os.Exit(continuationcmd.Run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, time.Now))`.

- [ ] **Step 4: Add the `continuation` case** to `runStreamingSubcommand` (the seam already exists from Task 5): `"continuation"` → `continuationcmd.Run(args[1:], stdin, stdout, stderr, time.Now)`. No signature changes to `run()`/`runWithLegacyRuntime()`.

- [ ] **Step 5: Write the seam test** (`cmd/pair-go/main_test.go`): feed a body on a fake `stdin` through `runStreamingSubcommand([]string{"continuation","--slug",…}, fakeStdin, &out, &errb)` against a temp git repo; assert the file is written and stdin was consumed (proves the seam passes real stdin, unlike buffered `Dispatch`).

- [ ] **Step 6: Update `Makefile.local`** `$(BIN_DIR)/pair-continuation` prereqs.

- [ ] **Step 7: Run** `go test ./cmd/internal/continuationcmd ./cmd/pair-continuation ./cmd/pair-go -count=1` — PASS.

- [ ] **Step 8: Commit** — `#92 M1: extract continuationcmd; route pair continuation (streaming)`.

### Task 7: route `session-watch` through the streaming seam

**Files:**
- Modify: `cmd/pair-go/main.go` (add `session-watch` to `runStreamingSubcommand`), possibly extract the `ensurePairTag`+`adapt.Open`+`sessionwatch.Run` orchestration from `cmd/pair-session-watch/main.go` into `sessionwatch` so both entry points share it
- Test: `cmd/pair-go/main_test.go`

- [ ] **Step 1: DRY the orchestration.** `cmd/pair-session-watch/main.go`'s `run(args, getenv, stderr) int` (buildOptions + ensurePairTag + adapt.Open + sessionwatch.Run) should be callable from both the shim and the dispatcher. Move it to `sessionwatch.RunCLI(args []string, getenv func(string) string, stderr io.Writer) int` — **widen `stderr` from `*os.File` (`main.go:16`) to `io.Writer`** for testability (M4); reduce the shim `main` to call it; `runStreamingSubcommand`'s `case "session-watch":` calls the same with `os.Stderr`.

- [ ] **Step 2: Write test** — `runStreamingSubcommand([]string{"session-watch"})` with <3 args no-ops (exit 0), matching current behavior.

- [ ] **Step 3: Implement + run PASS.**

- [ ] **Step 4: Commit** — `#92 M1: route pair session-watch (streaming)`.

### Task 8: M1 integration verification + milestone close

- [ ] **Step 1:** `make build` — the `pair` and `pair-go` binaries build. Add the new `cmd/internal/{slugcmd,changelogcmd,continuationcmd}` files (and `sessionwatch/*`, `model`, `transcript`) to `PAIR_GO_SRCS` (`Makefile.local:284`), the `pair`/`pair-go` rule prereqs. This is **staleness-only** (Go resolves imports at compile time regardless; the list just triggers incremental rebuilds) — the list already omits some transitive deps, so this is correctness-of-rebuild, not build-breaking. Confirm `make build` after touching a `*cmd` file rebuilds `pair`.
- [ ] **Step 2:** Manually verify each new route against its shim, with fakes: `bin/pair slug` ≡ `bin/pair-slug`; `bin/pair changelog --cleaned … ` ≡ `bin/pair-changelog …`; `bin/pair continuation --slug … --body-file -` ≡ `bin/pair-continuation …` (stdin); `bin/pair session-watch` ≡ `bin/pair-session-watch`; and `bin/pair context`/`bin/pair scrollback-render` work. Confirm `bin/pair claude` / `bin/pair resume` still reach the launcher.
- [ ] **Step 3:** `go test ./... -count=1` (scrub the leaking `PAIR_SESSION_ID`/`PAIR_TAG` env per the make-test memory note: `env -u PAIR_SESSION_ID -u PAIR_TAG go test ./...`).
- [ ] **Step 4:** `sdlc milestone-close --issue 92 --milestone M1 --verified '<evidence>'` — the boundary review runs here; fix Critical/Important before crossing.

---

## Chunk 2 — Milestone M2: repoint Pair-owned call-sites (shadow-sweep)

M2 flips the internal callers from `pair-<name>` to `pair <sub>`. Each call-site is one small, verifiable change. Closes via `sdlc milestone-close` (M2 boundary review).

### Task 9: repoint `pair-title.sh` → `pair context`

**Files:** Modify `bin/pair-title.sh:105`; Test `tests/pair-title-poller-test.sh`

- [ ] **Step 1:** Change `count=$(pair-context "$TAG" "$agent" …)` → `count=$("$PAIR_HOME/bin/pair" context "$TAG" "$agent" …)` (explicit `$PAIR_HOME/bin/pair`, robust regardless of PATH). Update the test's fake (`pair-title-poller-test.sh:103-108`) to stub `pair context` instead of `pair-context`.
- [ ] **Step 2:** Run `tests/pair-title-poller-test.sh` — PASS.
- [ ] **Step 3:** Commit — `#92 M2: pair-title.sh uses pair context`.

### Task 10: repoint `pair-changelog-open` → `pair changelog` + `pair scrollback-render`

**Files:** Modify `bin/pair-changelog-open:79-83`

- [ ] **Step 1:** `export PCL_RENDER="$PAIR_HOME/bin/pair scrollback-render"` and `PCL_DISTILL="$PAIR_HOME/bin/pair changelog"` — but these are invoked as `"$PCL_RENDER" args`; a space-containing var won't exec. Convert the pipeline (line 83) to call `"$PAIR_HOME/bin/pair" scrollback-render …` and `"$PAIR_HOME/bin/pair" changelog …` directly (drop the two-token env indirection, or split into `_BIN`+`_SUB` pairs). Preserve exact args.
- [ ] **Step 2:** Verify the Alt+l changelog flow end-to-end in a live pane (render → distill → viewer reload); the `e2e_test.go` already covers the distill core.
- [ ] **Step 3:** Commit — `#92 M2: pair-changelog-open uses pair subcommands`.

### Task 11: repoint `nvim/scrollback.lua` → `pair scrollback-render`

**Files:** Modify `nvim/scrollback.lua:286-288`

- [ ] **Step 1:** Change the `bin = PAIR_HOME .. '/bin/pair-scrollback-render'` (and the bare `'pair-scrollback-render'` fallback) to invoke `PAIR_HOME .. '/bin/pair'` with a `scrollback-render` first argument. Keep the arg vector otherwise identical.
- [ ] **Step 2:** Verify the Alt+/ scrollback viewer opens + renders in a live pane.
- [ ] **Step 3:** Commit — `#92 M2: scrollback.lua uses pair scrollback-render`.

### Task 12: repoint `pair-scrollback-open` → `pair scrollback-render`

**Files:** Modify `bin/pair-scrollback-open:70-80`; Test `tests/*scrollback*` if present

The Alt+/ viewer's **primary** render path (missed in the first draft, I1). It has both an `[ -x "$PAIR_HOME/bin/pair-scrollback-render" ]` guard (:70) and the invocation `"$PAIR_HOME/bin/pair-scrollback-render" "$RAW" "$EVENTS" "$ANSI"` (:80).

- [ ] **Step 1:** Change the invocation to `"$PAIR_HOME/bin/pair" scrollback-render "$RAW" "$EVENTS" "$ANSI"`. Update the `-x` guard to test `"$PAIR_HOME/bin/pair"` (which `make build` always produces). Keep the arg vector identical.
- [ ] **Step 2:** Verify the Alt+/ scrollback viewer renders in a live pane (this + Task 11 together are the full viewer path).
- [ ] **Step 3:** Commit — `#92 M2: pair-scrollback-open uses pair scrollback-render`.

### Task 13: repoint `pair-wrap`'s turn-end slug spawn → `pair slug`

**Files:** Modify `cmd/pair-wrap/main.go:539-543` (`maybeSpawnSlug`)

- [ ] **Step 1:** Change `exec.Command("pair-slug")` → `exec.Command(pairBin, "slug")`, where `pairBin` is `$PAIR_HOME/bin/pair` (consistent with the other M2 tasks; a bare-name `filepath.Dir(os.Args[0])` argv0 can resolve to `./pair`, M6). Keep it a **detached subprocess** (unchanged isolation) and keep `PAIR_AGENT` in `cmd.Env`; do NOT call `slugcmd.Run` in-process (preserves the fire-and-forget failure isolation the comment at :525-531 documents).
- [ ] **Step 2:** `maybeSpawnSlug` is inline/uninjectable today (M6). To assert the spawned argv is `[<pair>, slug]` with `PAIR_AGENT` set, either extract the command construction into a testable helper (`slugSpawnCmd(pairHome, agent) *exec.Cmd`) and unit-test that, or point `PAIR_HOME` at a fake `bin/pair` recorder in an integration test. Prefer the extracted helper (ARCH-PURE).
- [ ] **Step 3:** Commit — `#92 M2: pair-wrap spawns pair slug`.

### Task 14: shadow-sweep + runtime-bundle regen + M2 close

- [ ] **Step 1: Shadow-sweep (ARCH-PURPOSE).** `grep -rnE 'pair-(slug|changelog|continuation|context|scrollback-render|session-watch)' bin/ nvim/ cmd/ tests/` — enumerate every hit, confirm each Pair-owned caller now uses `pair <sub>` **or** is an intentionally-retained reference. The retained set: the `bin/pair-session-watch.sh` shim + `pair-shell`'s call to it (still shell-owned, #93), `Makefile.local`, `emitter-health-test.sh`'s binary-health probe, `nvim/init.lua:3319`'s prose "pair-continuation writer" (an agent-procedure name, not an exec), and the runtime manifest / `cmd/internal/runtimebundle/assets/…` mirror. Log this retained list in `## Log`.
- [ ] **Step 2: ARCH-PURPOSE — state the deferred call-sites explicitly.** `pair session-watch` and `pair continuation` get **no** repointed *production* caller in this issue: session-watch's caller stays `pair-shell → pair-session-watch.sh → binary` (shell-owned, retired in #93), and continuation has only the agent-procedure prose reference. They satisfy Done-when #1 (invocable as `pair <sub>`) but Done-when #3 (call-sites repointed) is **N/A / deferred** for them — their routes exist for symmetry + #96/#93 reuse, not dead code. Record this in `## Log` so the routes + `RunCLI` refactor aren't later flagged as unused.
- [ ] **Step 3: Regenerate the runtime bundle (I3).** M2 edited `bin/pair-title.sh`, `bin/pair-changelog-open`, `bin/pair-scrollback-open`, and `nvim/scrollback.lua` — all mirrored into `cmd/internal/runtimebundle/assets/runtime/files/…`. Run `make runtimebundle-generate` (or the repo's generate target) and commit the regenerated manifest+files, else copied/Homebrew binaries ship the OLD call-sites AND `make test`'s `test-runtimebundle` regenerates in place → dirty tree at `sdlc close`. (`runtimebundle-drift-check` only checks determinism, not content, so it won't catch a stale-but-committed bundle.)
- [ ] **Step 4:** Full suite: `env -u PAIR_SESSION_ID -u PAIR_TAG make test` (changelog/title/scrollback shell tests + all Go). The changelog-open test (`tests/changelog-open-test.sh`) now needs `bin/pair` built — `make build` provides it. Note the pre-existing `parley_harness_golden` failure per the memory note — not introduced here.
- [ ] **Step 5:** Confirm the working tree is clean (bundle committed), then `sdlc milestone-close --issue 92 --milestone M2 --verified '<evidence>'`.

---

## Verification (whole issue)

- `go test ./... -count=1` green (env scrubbed).
- `make build` produces `pair` + all shims; `bin/pair <sub>` ≡ `bin/pair-<name>` for slug/changelog/continuation/session-watch/context/scrollback-render.
- `bin/pair claude` / `bin/pair resume` / bare `bin/pair` still launch (grammar preserved).
- Live checks: title poller, Alt+l changelog, Alt+/ scrollback, a wrapped session's turn-end slug refresh.
- `sdlc close --issue 92 --verified '<evidence>'` (atlas: update `atlas/go-migration-inventory.md` — the helper rows now note `pair <sub>` as the canonical route + the retained shims).

## Risks / notes

- **Entrypoint grammar:** the peel-off only intercepts the reserved dispatch names; agent names and `resume/continue/list/rename` are untouched. Guard with the `TestClassify` table. A future agent literally named `slug` is the only collision — acceptable and documented.
- **`git mv` + package rename:** stage the edit after the move (memory: `git mv` doesn't stage later edits); verify each relocated `_test.go` compiles under the new package.
- **Behavior parity is the real test,** not just "it compiles": the relocated pure tests must pass unchanged, and the automated `helper_equivalence_test.go` `pair-go <sub>` ≡ `pair-<name>` diffs must match.
- **Bundle: binary list unchanged, asset contents DO change (I3).** The standalone binaries stay in `RUNTIMEBUNDLE_HELPERS` / the manifest (still shims) — but M2 edits the bundled *shell/lua assets* (`pair-title.sh`, `pair-changelog-open`, `pair-scrollback-open`, `scrollback.lua`), which are mirrored into the embedded runtime. The bundle MUST be regenerated + committed (Task 14 Step 3); "unchanged" refers only to which binaries are packaged, not their contents.
- **Buffered vs streaming is a correctness axis, not a style choice (I2):** a helper with a live stderr/stdout consumer (changelog's spinner) or stdin (continuation) or long lifetime (session-watch) must use the streaming seam; only truly fire-and-forget/finite output (slug) is safe buffered.

## Revisions

### 2026-07-01 — M1 as-built (records M1 boundary-review findings I2/I3/minor)

The M1 milestone review (FIX-THEN-SHIP) flagged plan/code drift; recording the
as-built decisions so the Core-Concepts table stops claiming surfaces the code
doesn't expose.

1. **Runner signatures as-built.** The Integration-points table showed
   `slugcmd.Run(args, env, stdout, stderr) int` and `changelogcmd.Run(args,
   stdout, stderr) int`; the code ships `slugcmd.Run() int` and
   `changelogcmd.Run(args []string, stderr io.Writer) int`.
   - `slugcmd.Run()` is **env-driven with no injected `Env`/writers** —
     deliberate: pair-slug takes no args and writes only to files +
     `$PAIR_SLUG_LOG` (the buffered route captures nothing), and its full
     orchestration (transcript-resolve → model → atomic write) is already
     exercised by the relocated binary-exec integration tests in
     `cmd/pair-slug/` (`TestIntegrationProposesValidSlug` et al. build + run the
     shim over `slugcmd.Run`). Behavior parity over an injection layer that would
     add no coverage the integration tests don't already give (supersedes the
     plan's Task 3 Step 2 injected-`Env` idea). `changelogcmd.Run` takes no
     `stdout` because changelog writes only stderr (progress) + files.
   - `continuationcmd.Run(args, stdin, stdout, stderr, now) int` matches the plan.
2. **Equivalence test (Task 3 Step 5).** Added `TestPairGoSlugMatchesLegacyPairSlug`
   to `cmd/pair-go/helper_equivalence_test.go` — a no-session parity check
   (`pair-go slug` ≡ `pair-slug`: exit 0, no output, no proposal). Both entry
   points call the identical `slugcmd.Run()`, so a deeper fixture would only
   re-test the shared runner.
3. **Flag error-handling (minor).** `changelogcmd`/`continuationcmd` parse a
   fresh `flag.NewFlagSet(..., ContinueOnError)` over `args` and `return 1` on a
   malformed flag, where the old binaries used `flag.CommandLine` (`ExitOnError`
   → exit 2). Internal callers pass fixed valid flags, so no caller/test is
   affected; recorded so the Spec's "preserve exit codes" line isn't read as
   violated on the never-exercised malformed-flag path.
4. **`DispatchNames` assertion (I4b).** Strengthened
   `TestDispatchNamesDeriveFromImplementedStatus` to assert the full implemented
   set `{context, scrollback-render, slug, changelog, continuation,
   session-watch}` is present — the peel-off depends on it.
5. **Atlas (I1).** `atlas/architecture.md`'s dispatcher section now describes the
   four new routes, the public `pair <sub>` peel-off, and the buffered-vs-
   streaming seam (was still "#76 first routes only").

### 2026-07-01 — M2 as-built (records M2 boundary-review minors)

M2 milestone review (FIX-THEN-SHIP, no Critical/Important). As-built deviations:

1. **`slugSpawnCmd` signature (Task 13).** Ships `slugSpawnCmd(agent string)
   *exec.Cmd` reading `PAIR_HOME` from env, not the Core-Concepts sketch's
   `slugSpawnCmd(pairHome, agent)`. Testable via `t.Setenv` (the two unit tests
   do this); env-read chosen over a threaded param since the caller
   (`maybeSpawnSlug`) has no `pairHome` in scope.
2. **`pair-title.sh` invocation form (Task 9).** Calls bare `pair context`
   (PATH-resolved via `pair-shell`'s `$PAIR_HOME/bin` prepend), not the plan's
   explicit `"$PAIR_HOME/bin/pair" context`. Parity with the pre-existing bare
   `pair-context` call it replaced; the other four call-sites use the explicit
   `$PAIR_HOME/bin/pair` path (those already used explicit paths).
3. **Runtime bundle (I3, restated).** `cmd/internal/runtimebundle/assets/` is
   gitignored + regenerated on `make build`, so no bundle commit is needed and
   there is no dirty-tree-at-close risk (supersedes Task 14 Step 3's
   "regenerate + commit").
4. **Viewer test gap closed.** Added a `renderer_command` assertion to
   `nvim/scrollback_test.lua` (via `_G.PairScrollbackTest`) pinning the
   `pair scrollback-render` invocation form — the M2 review's one flagged
   coverage gap.
