# Native nvim/zellij Startup Assets — Decision + PATH Fix Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the native-single-binary roadmap (#91 step 5) by (a) deciding + documenting how `nvim`/`zellij` startup assets reach the external processes, and (b) fixing the latent regression that makes a copied/Homebrew `pair` unable to launch — the Go launcher never puts `$PAIR_HOME/bin` on PATH, so zellij can't resolve the bare-name helpers (`pair-wrap`, `copy-on-select`, `pair-help`) it execs.

**Architecture:** The survey (issue `## Log`) established that a **true zero-tree native single binary is physically unreachable** while `nvim`/`zellij` stay native: they read config from real filesystem paths (`nvim -u init.lua`, `zellij --config-dir`), `nvim/init.lua` `dofile()`s ~5 siblings by absolute path (needs a real *directory*, not one file), viewers spawn mid-session (the tree must persist session-long), and Go's `embed.FS` is not a filesystem path. So the decision is: **keep the existing digest-versioned extraction** (already deterministic / idempotent / upgrade-safe / self-pruning) and **reframe it honestly as a content-addressed runtime *cache*** — not an "install tree" — documenting the residual gap. The only *code* change is the PATH fix: prepend the resolved asset-root `bin/` to PATH once at launch entry (via the existing `SetEnv` seam), restoring exactly what the retired shell `bin/pair` did (`config.kdl:39` / `atlas:207` / `pair.rb:32` still promise it) so all downstream children — zellij, its `Run`/`copy_command` actions, resurrected panes, and the nvim viewers — inherit it.

**Tech Stack:** Go (the launcher entry `cmd/internal/launcher/runcli.go` + a pure PATH helper), bash (the copied-binary smoke `tests/pair-embedded-runtime-test.sh`), Markdown/KDL/Ruby (atlas + `zellij/config.kdl` + Homebrew `pair.rb` doc fixes). No `nvim`/`zellij` logic is ported (invariant). No new provisioning path — extraction is unchanged.

**Scope note:** This is **atomic single-pass work** (one pure helper + one seam call + a test + doc updates) — plain checkboxes, one `sdlc close`, no `Mx` milestone split. The "decision" half is documentation only (extraction already works); the substantive code is the ~10-line PATH fix. Explicitly OUT of scope: porting nvim/zellij to Go (invariant), and routing the extracted standalone Go helper binaries through `pair <sub>` to shrink the bundle further (a separate, un-ticketed single-binary effort — the extraction is load-bearing for those binaries regardless of this issue).

---

## Core Concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `prependBinToPath` | `cmd/internal/launcher/pathenv.go` | new |

- **`prependBinToPath(pairHome, path string) string`** — puts `<pairHome>/bin` at the front of a PATH string, idempotently. Empty `path` → just the bin dir; already-first → unchanged (dev shells / re-launch); otherwise prepend with `os.PathListSeparator`.
  - **Relationships:** 1:1 with a launch invocation; consumes the resolved asset root (`pairHome`) that `ResolveAssetRoot` already produced.
  - **DRY rationale:** one source of truth for "how the launcher augments PATH", unit-testable without touching the process env. Replaces the behavior the retired shell `bin/pair` open-coded inline.
  - **Future extensions:** if more than one dir ever needs prepending, the signature widens to `(dirs []string, path string)`.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| PATH export at `RunLaunch` entry | `cmd/internal/launcher/createflow.go` (`RunLaunch`) | modified | process env (`os.Setenv` via the `SetEnv` seam) |
| copied-binary helper-resolution smoke | `tests/pair-embedded-runtime-test.sh` | modified | a stubbed `zellij` that execs a bare-name helper |

- **PATH export at `RunLaunch` entry** — one line at the top of `RunLaunch` (`createflow.go:24`, right after `env := normalizeEnv(opts.Env)`): `rt.SetEnv("PATH", prependBinToPath(opts.PairHome, os.Getenv("PATH")))`. Placed in `RunLaunch` (not `LaunchNative`) for two reasons: (1) `RunLaunch` is the *only* path that spawns zellij with panes — create, attach/resurrect, compaction, and the in-process restart loop all flow through it, so this single point covers them all; the non-launch verbs (`list`/`rename`/`restart`/`quit`) return before `RunLaunch` and don't exec bare-name helpers. (2) `RunLaunch` is exercised in tests via the **fake** runtime (`createflow_test.go`), whose `SetEnv` just records into `f.env` — so a wiring test asserts the prepend with **zero process-env pollution**, whereas `LaunchNative` uses the real `OSRuntime` (real `os.Setenv`) and would leak PATH into the test process.
  - **Injected into:** nothing downstream changes — zellij (`LaunchSession`/`AttachSession`) spawns via `exec.Command` inheriting `os.Environ()`, so the augmented PATH flows to zellij and everything zellij/nvim exec. Mirrors the existing `rt.SetEnv("PAIR_HOME", …)` exports (`createflow.go:210`), whose interface doc already says "every child … inherits."
  - **Future extensions:** none expected; this is the boundary env setup.
- **copied-binary helper-resolution smoke** — the existing `tests/pair-embedded-runtime-test.sh` extracts a real bundle but its stub `zellij` only asserts files *exist*. Strengthen it so the stub `zellij` asserts a bundled helper *resolves by bare name* (`command -v copy-on-select`), which fails unless the launcher prepended the extracted `bin/` — directly guarding the regression.

**Test surface:** `prependBinToPath` gets a colocated table unit test (`pathenv_test.go`, no IO). The `RunLaunch` wiring gets a fake-runtime test (`createflow_test.go`) asserting `f.env["PATH"]` is prepended (no pollution — fake `SetEnv` records). The end-to-end behavior is proven by the strengthened copied-binary smoke (real `pair` binary → real extraction → bare-name resolution). The 3-layout claim is covered because the prepend is layout-agnostic (it prepends whatever `ResolveAssetRoot` returned as `pairHome`); the copied-binary case is the one that was broken and is the one the smoke exercises.

---

## Task 1: Pure `prependBinToPath` helper

**Files:**
- Create: `cmd/internal/launcher/pathenv.go`
- Test: `cmd/internal/launcher/pathenv_test.go`

- [ ] **Step 1: Write the failing test**

Create `pathenv_test.go`:

```go
package launcher

import (
	"os"
	"testing"
)

func TestPrependBinToPath(t *testing.T) {
	sep := string(os.PathListSeparator)
	bin := "/root/bin"
	cases := []struct {
		name string
		home string
		path string
		want string
	}{
		{"empty path", "/root", "", bin},
		{"prepends when absent", "/root", "/usr/bin" + sep + "/bin", bin + sep + "/usr/bin" + sep + "/bin"},
		{"idempotent when already first", "/root", bin + sep + "/usr/bin", bin + sep + "/usr/bin"},
		{"idempotent when it is the whole path", "/root", bin, bin},
		{"prepends even if present later (dup is harmless)", "/root", "/usr/bin" + sep + bin, bin + sep + "/usr/bin" + sep + bin},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := prependBinToPath(tc.home, tc.path); got != tc.want {
				t.Fatalf("prependBinToPath(%q,%q) = %q, want %q", tc.home, tc.path, got, tc.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./cmd/internal/launcher/ -run TestPrependBinToPath`
Expected: FAIL — `prependBinToPath` undefined.

- [ ] **Step 3: Implement**

Create `pathenv.go`:

```go
package launcher

import (
	"os"
	"path/filepath"
	"strings"
)

// prependBinToPath puts <pairHome>/bin at the front of PATH, idempotently. The
// launcher calls this once at entry so zellij and everything it execs by bare
// name — pair-wrap (the agent pane), copy_command "copy-on-select", Run
// "pair-help"/"pair-scrollback-open", and the nvim viewers' helpers — resolve
// from the resolved asset root's bin/. The retired shell bin/pair did this
// prepend; the Go launcher that replaced it (#99 M5c) dropped it, so a copied or
// Homebrew install (whose bin/ isn't already on the user's PATH) couldn't launch
// (#95). Empty PATH yields just the bin dir; an already-leading bin dir is left
// unchanged (dev shells / re-launch).
func prependBinToPath(pairHome, path string) string {
	binDir := filepath.Join(pairHome, "bin")
	sep := string(os.PathListSeparator)
	if path == "" {
		return binDir
	}
	if path == binDir || strings.HasPrefix(path, binDir+sep) {
		return path
	}
	return binDir + sep + path
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./cmd/internal/launcher/ -run TestPrependBinToPath -v`
Expected: PASS (all 5 subtests).

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/launcher/pathenv.go cmd/internal/launcher/pathenv_test.go
git commit -m "#95: pure prependBinToPath helper (idempotent PATH front-insert)

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

## Task 2: Wire the PATH export into `RunLaunch` (fake-tested)

**Files:**
- Modify: `cmd/internal/launcher/createflow.go` (`RunLaunch`, top)
- Test: `cmd/internal/launcher/createflow_test.go`

- [ ] **Step 1: Write the failing wiring test**

Mirror the simplest existing create test, `TestRunLaunchForcedCreateClaude` (`createflow_test.go:273`). The harness helpers (verified): `newFakeRuntime()` (`:73`); `baseOpts(args LaunchArgs) LaunchOptions` (**takes a `LaunchArgs`** — and already sets `PairHome: "/pair"`); the driver `run(t *testing.T, opts LaunchOptions, rt *fakeRuntime) (int, error)` (`:261`). The fake's `SetEnv` records into `f.env` (`:145`, init'd empty at `:81`), so the assertion is pollution-free. The SetEnv runs on the 2nd line of `RunLaunch` (before the handoff), so a minimal create is enough. Add:

```go
func TestRunLaunchPrependsBinToPath(t *testing.T) {
	rt := newFakeRuntime()
	rt.uuids = []string{"S"} // mirror the forced-create test's minimal setup
	opts := baseOpts(LaunchArgs{Agent: "claude", ForcedTag: "x"})
	// baseOpts already sets PairHome "/pair".
	run(t, opts, rt)

	got := rt.env["PATH"]
	sep := string(os.PathListSeparator)
	if got != "/pair/bin" && !strings.HasPrefix(got, "/pair/bin"+sep) {
		t.Fatalf("RunLaunch did not prepend the asset-root bin/ to PATH: %q", got)
	}
}
```

(Confirm the exact `LaunchArgs` fields + any extra fake setup against `TestRunLaunchForcedCreateClaude` — copy its setup verbatim, then add the PATH assertion. `os`/`strings` are already imported in the test file.)

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./cmd/internal/launcher/ -run TestRunLaunchPrependsBinToPath`
Expected: FAIL — `rt.env["PATH"]` is empty (no prepend yet).

- [ ] **Step 3: Add the SetEnv call at the top of `RunLaunch`**

In `createflow.go`, immediately after `env := normalizeEnv(opts.Env)` (currently line 23), insert:

```go
	// Prepend the resolved asset root's bin/ to PATH so zellij (and its panes:
	// pair-wrap, copy_command "copy-on-select", Run "pair-help"/openers, and the
	// nvim viewers) resolve the bundled helpers by bare name. The retired shell
	// bin/pair did this; the Go launcher that replaced it (#99 M5c) dropped it, so
	// a copied/Homebrew install (bin/ not on the user's PATH) couldn't launch
	// (#95). RunLaunch is the sole zellij-spawning path (create/attach/resurrect/
	// restart-loop), so once here covers them all.
	rt.SetEnv("PATH", prependBinToPath(opts.PairHome, os.Getenv("PATH")))
```

**Add `"os"` to `createflow.go`'s import block** — it is NOT currently imported there (the block is `fmt`, `io`, `path/filepath`, `strings`, `time`); the inserted line uses `os.Getenv`. `SetEnv` is on the `Runtime` seam (used just below at `createflow.go:210`). No spawn site changes — zellij/viewers inherit the augmented `os.Environ()`.

- [ ] **Step 4: Run to verify it passes + full package**

Run: `go test ./cmd/internal/launcher/`
Expected: PASS — the new test passes and no existing create/routing test regressed (the fake's `SetEnv` records into `f.env`; existing tests don't assert on PATH).

- [ ] **Step 5: Commit**

```bash
git add cmd/internal/launcher/createflow.go cmd/internal/launcher/createflow_test.go
git commit -m "#95: RunLaunch prepends \$PAIR_HOME/bin to PATH at entry

Restores the retired shell bin/pair's PATH-prepend so zellij resolves the
bundled bare-name helpers (pair-wrap, copy-on-select, pair-help). Without it a
copied/Homebrew pair (bin/ not on the user's PATH) can't launch a session.
Placed in RunLaunch (the sole zellij-spawning path, fake-tested) not LaunchNative
(real OSRuntime → would leak PATH into the test process).

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

## Task 3: Strengthen the copied-binary smoke to prove bare-name resolution

**Files:**
- Modify: `tests/pair-embedded-runtime-test.sh`

- [ ] **Step 1: Read the harness**

Read `tests/pair-embedded-runtime-test.sh` fully. It builds `pair` from `cmd/pair-go`, runs it (extracting the embedded bundle), and its stub `zellij` (a bash script on a fake PATH that does NOT include `$root/bin`) currently only asserts the extracted files exist (`test -x "$root/bin/pair-wrap"` etc.). Identify: (a) where the stub `zellij` script body is written, (b) that the launcher invokes `pair resume smoke` which forces extraction + the zellij handoff, (c) that the stub's PATH is the fake dir only.

- [ ] **Step 2: Add a bare-name resolution assertion to the stub zellij**

In the stub `zellij` body (the branch that runs on the create/attach handoff, where `$root` is known), add — *before* it exits 0 — an assertion that a bundled Go helper resolves by bare name via the inherited PATH:

```bash
    # #95: the launcher must have prepended $root/bin to PATH, so a bundled
    # helper resolves by BARE NAME here (zellij execs pair-wrap/copy-on-select by
    # bare name). The stub's own PATH does NOT include $root/bin — only the
    # launcher's prepend puts it there. This is the regression guard.
    command -v copy-on-select >/dev/null || { echo "copy-on-select not on PATH (launcher PATH prepend missing)" >&2; exit 21; }
    command -v pair-wrap      >/dev/null || { echo "pair-wrap not on PATH" >&2; exit 22; }
```

(Use `command -v`, not execution — `pair-wrap`/`copy-on-select` would otherwise try to do real work. `command -v` only checks resolvability.)

- [ ] **Step 3: Run the smoke — confirm it PASSES with the fix and FAILS without**

Run: `make test-pair-embedded-runtime`
Expected: PASS.

Then prove the guard bites: temporarily comment out the `rt.SetEnv("PATH", …)` line from Task 2, re-run, and confirm it now FAILS with `exit 21` (copy-on-select not on PATH). Restore the line, re-run, confirm PASS. (This proves the test actually guards the regression, not just passes vacuously.)

- [ ] **Step 4: Commit**

```bash
git add tests/pair-embedded-runtime-test.sh
git commit -m "#95: smoke asserts bundled helpers resolve by bare name (PATH guard)

The stub zellij now runs 'command -v copy-on-select' with $root/bin absent from
its own PATH, so it passes only via the launcher's #95 PATH prepend. Verified it
fails (exit 21) when the prepend is removed.

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

## Task 4: Documentation — decision, cache reframe, and the stale-doc fixes

**Files:**
- Modify: `atlas/architecture.md` (the runtime-provisioning section + the stale PATH line ~207)
- Modify: `atlas/go-migration-inventory.md` (the final runtime-provisioning shape)
- Modify: `zellij/config.kdl` (the `bin/pair prepends …` comment ~39)
- Modify: `../homebrew-pair/Formula/pair.rb` (the `PATH the launcher sets up` comment ~30-32)

- [ ] **Step 1: Record the decision + cache reframe in `atlas/architecture.md`**

In the runtime-provisioning / extraction section, add a concise paragraph (the #95 decision):
- The embedded bundle is extracted to `$PAIR_DATA_DIR/runtime/<digest>/pair-home` **only for the copied-binary layout** (source + Homebrew point their asset root at an adjacent real tree and never extract). Frame it as a **content-addressed runtime *cache*** — deterministic (content digest), idempotent (skip-unchanged), upgrade-safe + self-pruning (keep-2) — not a "Pair-owned install tree".
- Record the residual gap explicitly (ARCH-PURPOSE — the issue permits documenting the final gap): **true zero-tree is unreachable** while `nvim`/`zellij` stay native, because they read config from real filesystem paths and `nvim/init.lua` `dofile()`s siblings (needs a real directory that persists for the session). The native-single-binary endpoint is therefore "one Pair *executable* + a self-provisioned config cache for the two external tools + system platform tools", not literally zero bytes on disk.

- [ ] **Step 2: Fix the stale "launcher prepends PATH" claims in `atlas/architecture.md` (TWO lines)**

`atlas/architecture.md` makes the now-false claim in **two** places — both described the retired shell `bin/pair` and had been false since #99 M5c, and both become TRUE again after Task 2:
- **~207** — "…prepends `$PAIR_HOME/bin` to `$PATH` (idempotent across re-launches) so all helper scripts resolve by bare name…"
- **~803** — "The launcher prepends `$PAIR_HOME/bin` to `$PATH` before exec'ing zellij…"

Update **both** to name the **Go launcher** (`RunLaunch`'s `prependBinToPath`, #95), so the docs match the code. (Grep `atlas/architecture.md` for `prepends` to find the exact current lines — numbers shift.)

- [ ] **Step 3: Update `atlas/go-migration-inventory.md`**

Update the runtime-bundle / provisioning entries to the final shape: the bundle is the content-addressed cache described above; note the PATH prepend was restored in #95; and that #91's roadmap endpoint is reached with the documented residual disk-cache (native single *executable*, external tools external). If a "remaining work toward zero-tree" note exists, replace it with the residual-gap statement.

- [ ] **Step 4: Fix the two out-of-tree stale claims**

- `zellij/config.kdl:39` — "`bin/pair` prepends `$PAIR_HOME/bin` to PATH so the script is resolvable by bare name" → "the launcher prepends `$PAIR_HOME/bin` to PATH (restored #95)".
- `../homebrew-pair/Formula/pair.rb:30-32` — the comment claiming "zellij's PATH-based `exec pair-wrap` finds libexec/bin/pair-wrap via the … PATH the launcher sets up" → update to state the Go launcher prepends `$PAIR_HOME/bin` (= `libexec/bin`) at entry (#95), which is what makes the Homebrew helper resolution work. (Peer repo — verify it's the pair Homebrew formula and edit per the peer-repo rules; commit there separately if it's a distinct repo.)

- [ ] **Step 5: Verify no other doc still claims the shell did it**

First regenerate the bundle so the generated copy of `config.kdl` reflects the edited source (else the grep false-positives on the stale comment in `cmd/internal/runtimebundle/assets/runtime/files/zellij/config.kdl`, which is overwritten on build — do NOT hand-edit it):

Run: `make runtimebundle-generate`
Run: `grep -rn "prepends" . --include='*.md' --include='*.kdl' --include='*.rb' --exclude-dir=.git --exclude-dir=workshop --exclude-dir=assets | grep -i "path\|PAIR_HOME/bin"`
Expected: every remaining hit names the Go launcher / #95, or is historical narrative — no live claim that the retired shell `bin/pair` does it. (Use the broad `prepends` match, not a `.PAIR_HOME/bin` pattern — the docs write the backtick-`$` form `` `$PAIR_HOME/bin` ``, which a single-`.` regex misses. `--exclude-dir=assets` skips the generated bundle copy.)

- [ ] **Step 6: Commit (pair repo)**

```bash
git add atlas/architecture.md atlas/go-migration-inventory.md zellij/config.kdl
git commit -m "#95: document runtime cache + residual zero-tree gap; fix stale PATH docs

Reframe the copied-binary extraction as a content-addressed runtime cache and
record that true zero-tree is unreachable with external nvim/zellij (they read
config from real paths; init.lua dofiles siblings). Update the three docs that
claimed the retired shell bin/pair prepends PATH to name the Go launcher (#95).

Co-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>"
```

## Task 5: Verify + close

- [ ] **Step 1: Full test suite**

Run: `env -u PAIR_SESSION_ID -u PAIR_TAG make test` (redirect to a log, check the REAL exit code — do NOT pipe through `tail`; sandbox-off, the launcher tests need real `ps`).
Expected: `MAKE_EXIT=0`, incl. `test-pair-embedded-runtime` PASS.

- [ ] **Step 2: Close the issue (atomic, single close)**

Tick the Done-when boxes (all satisfied: strategy decided+documented; implemented+tested; endpoint reached with the documented residual gap; atlas updated). Then:

`sdlc close --issue 95 --verified '<evidence>'` (measured `--actual` via the omit-then-suggest path). The boundary review runs here (single review — no `Mx` milestones). Fix any Critical/Important. Commit the close edits **with** the printed `Review-Verdict:` trailer (per the #94 lesson). Then publish: `sdlc pr` → `sdlc merge` (the branch flow), which flips `codecomplete → done` and archives.

- [ ] **Step 3: Tick #91 step 5 + close the roadmap**

After #95 merges, tick #91's Plan step 5 (`- [x] Step 5 … #95 done`), and — since #91's umbrella is now 6/6 with the true-native-single-binary endpoint reached (as documented) — evaluate closing #91 itself (`sdlc close --issue 91`, or leave a final Log entry marking the roadmap complete). Push the roadmap update.

---

## Notes / risks

- **Why at `RunLaunch` entry, not per-spawn:** `RunLaunch` is the sole path that spawns zellij with panes — create, attach, *resurrect* (zellij re-runs pane commands), and the restart loop all flow through it — so one SetEnv there covers them, and everything (zellij, its `Run`/`copy_command` actions, detached watchers, keybind-spawned nvim viewers) inherits the augmented env. Per-spawn `cmd.Env` on each `exec.Command` would be fragile (easy to miss the viewers). `LaunchNative` was rejected: the non-launch verbs don't need it, and its real `OSRuntime` would leak PATH into `LaunchNative` tests.
- **Idempotency:** `RunLaunch` (and thus the SetEnv) runs once per launcher process (the restart loop is inside it) — but dev shells already have `bin/` on PATH, so `prependBinToPath` skips the redundant prepend when it's already first (a mid-PATH duplicate is harmless).
- **The test must bite:** Task 3 Step 3 explicitly verifies the smoke FAILS without the fix — otherwise a vacuous pass would let the regression back in. Don't skip that check.
- **Homebrew formula is a peer repo** (`../homebrew-pair`): edit + commit it under its own rules; it doesn't ride the pair-repo PR. If it's not checked out, note the doc-fix as a follow-up in the issue Log rather than blocking the close.
- **No behavior change to extraction** — the decision is "keep it"; the only runtime behavior change is the PATH prepend. Everything else is documentation.
