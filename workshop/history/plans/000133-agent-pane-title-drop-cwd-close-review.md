# Boundary Review — pair#133 (whole-issue close)

| field | value |
|-------|-------|
| issue | 133 — agent pane title drops the cwd suffix |
| repo | pair |
| issue file | workshop/issues/000133-agent-pane-title-drop-cwd.md |
| boundary | whole-issue close |
| milestone | — |
| window | 8e659fad9fe1a94cd76c95ddd0bef1ccb193cb5e..HEAD |
| command | sdlc close --issue 133 |
| reviewer | claude |
| timestamp | 2026-07-29T17:07:14-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The change does exactly what the Spec says and does it by subtraction: both byte-identical cwd abbreviators, `PaneTitle`, `PAIR_PANE_CWD`, `cwd_display`, `PaneInfo.Cwd/.CwdDisplay`, `Options.Home` and their decodes are gone; the two KDL printfs, the generated runtime bundle, `atlas/architecture.md`, and `Makefile.local`'s comment all agree with the code. I traced every consumer named in the Spec and found no survivor: `grep` for `TildeAbbrev|abbrevCwd|PAIR_PANE_CWD` returns nothing outside `workshop/`, the embedded assets at `cmd/internal/runtimebundle/assets/.../main-{2,3}.kdl` carry the 2-verb/2-arg printf, and both live readers of the raw `cwd` (`contextcmd.paneCwd`, `launcher.legacyPaneAgentForScope`) are untouched and still decode `cwd`. What blocks a clean SHIP is the docs gate, not correctness: **`README.md:91` still tells users the agent frame reads `<agent> (<count>) [<cwd>]`** — the single user-facing sentence describing the exact surface this issue changed — and the in-package doc comment at `titlepoller.go:4` says the same. Second, the new conformance test is a genuinely good addition but stops one assertion short of covering the *startup* title, which is the half of the Done-when the live check explicitly did not observe.

Caveat on my evidence: `Bash` is unavailable in this review environment (harness EPERM), so I could not run `go test` or `make test` myself. Every finding below is from reading the tree; the green-suite claim in `## Log` is the implementor's, not independently reproduced. I did grep for dangling references to every deleted symbol and found none, so a compile break is unlikely.

## 1. Strengths

- **The producer conformance test is the right fix for the right risk.** `cmd/internal/contextcmd/panejson_kdl_test.go:80` extracts the real `args "-c"` line from *both* layouts, runs it under `sh` with `zellij`/`pair` stubbed on `PATH`, and asserts the output decodes through the actual consumer (`paneCwd`) as **exactly one** JSON value (`:131-134`). That second assertion is the whole point — it pins the printf format-recycling failure mode that no existing test could see. The `## Log` records it was written pre-edit and mutation-tested. This is the `ARCH-MOCK` seam that was missing.
- **`unescapeKDL` (`panejson_kdl_test.go:55`) gets the tricky part right.** It resolves `\"` and `\\` but leaves the resulting `\n` for printf, with a comment explaining why. Collapsing that would have silently made the test exercise a different command than production runs.
- **The dead thread was followed all the way out, not just to the first compile error.** `PaneInfo.Cwd`/`.CwdDisplay` and their `runtime.go` decode are gone even though Go would never have flagged them (`run.go:30-33`, `runtime.go:72-83`), and `Options.Home` + its `runcli.go:33` wiring went with them. I verified `env.Home` in `launcher` still has a live reader (`createflow.go:640`) so nothing over-deleted.
- **`TestRunLaunchForcedCreateClaude` now asserts both halves** (`createflow_test.go:426-432`): the new `PAIR_PANE_TITLE` value *and* the absence of `PAIR_PANE_CWD`. The startup title had zero coverage before this branch.
- **`tests/copy-on-select-test.sh:58-62` was correctly left hostile.** Keeping the legacy `claude [~/workspace/parley.nvim]` title and documenting *why* ("do not modernize this — that makes the case pass trivially") is better judgment than a mechanical sweep.
- `ARCH-DRY` resolved by subtraction rather than by reviving `titlefmt` for one caller — the reasoning is recorded in `## Log` and the atlas, and it is the correct call.

## 2. Critical findings

None.

## 3. Important findings

**I-1 — `README.md:91`: user-facing docs still document the removed title shape (docs gate; `ARCH-PURPOSE`).**

```
README.md:89  **Context meter in the pane frame**
README.md:91  The agent pane's frame reads `<agent> (<count>) [<cwd>]` — `<count>` is the
```

This is the only README sentence describing the agent frame, and it now states the pre-#133 format. The shadow-sweep for this single-source change enumerates: `createflow.go` ✓, `frameTitle` ✓, both KDLs ✓, generated bundle ✓, `PaneInfo` ✓, `atlas/architecture.md:274-282` ✓, `Makefile.local:79` ✓ — and README ✗. It is the one remaining hand-maintained restatement of the model that does not derive from it, which is precisely the `ARCH-PURPOSE` at-review lens.

Fix sketch — rewrite `README.md:91-93` to the new shape and, since the whole point of the issue is the two-halves composition, say so in the sentence users actually read:

```markdown
The agent pane's frame reads `<agent> (<count>)` — `<count>` is the agent's live
context-window occupancy (e.g. `970k`), read from its own transcript, so you can
watch the window fill without asking. The terminal tab shows it as
`📁pair | claude (970k)`: zellij prepends the session name, so the pane half
carries only the pane's own identity.
```

**I-2 — `panejson_kdl_test.go:88-89`: the `zellij` stub discards argv, so the startup-title path is executed but unasserted (`ARCH-MOCK`; missing coverage).**

`stubBin` writes a no-op `#!/bin/sh\nexit 0` for `zellij`, so the KDL's `zellij action rename-pane --pane-id "$ZELLIJ_PANE_ID" "${PAIR_PANE_TITLE:-agent}"` runs and its arguments vanish. Two consequences:

- The `## Log` claims the startup title is "covered … by the conformance test executing the KDL line that consumes `${PAIR_PANE_TITLE:-agent}`" (issue file `:371-372`). Executing is not asserting — that clause over-claims.
- Concretely: mangle the `${PAIR_PANE_TITLE:-agent}` expansion, or the `--pane-id` flag, in either KDL and **every test in the tree stays green**. That is the same class of silent, compiler-invisible KDL breakage this test exists to catch, on the same line the diff edited. The Done-when's first bullet ("Agent pane title is `claude (629k)` **at startup**") was never observed live — the `## Log` says so plainly — so this test is the only thing standing behind it.

Fix sketch — make the `zellij` stub a minimal stateful fake (record then assert), roughly:

```go
// stubBin: for "zellij", record argv so the rename-pane call is observable.
recorder := filepath.Join(binDir, "zellij")
os.WriteFile(recorder, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" >> \"$ZELLIJ_ARGV_LOG\"\nexit 0\n"), 0o755)
// …env: "ZELLIJ_ARGV_LOG=" + filepath.Join(dataDir, "zellij-argv"), "PAIR_PANE_TITLE=claude"
// …assert the log contains: action, rename-pane, --pane-id, 7, claude
```

That closes the loop `createflow.SetEnv("PAIR_PANE_TITLE", agent)` → KDL → `rename-pane`, end to end, for the cost of three lines — and it upgrades the stub from a stateless no-op to a fake that models the one interaction we depend on.

## 4. Minor findings

- `cmd/internal/titlepoller/titlepoller.go:4` — package doc still reads `"<agent> (<count>) [<cwd>]"`, contradicting `frameTitle` 58 lines below in the same file. Fix alongside I-1.
- `panejson_kdl_test.go:123` — the anonymous struct decodes `Cwd` but never asserts it. Either drop the field or assert `first.Cwd == paneCwdDir` (the `paneCwd` check at `:109` covers it indirectly, so this is cosmetic).
- `ARCH-DRY` nit: "extract the agent pane's shell command from a KDL line" now exists twice — Go at `panejson_kdl_test.go:32-49` and shell at `tests/term-pane-shortcuts-test.sh:247` (`sed 's/^[[:space:]]*args "-c" "//; s/"$//; s/\\"/"/g'`). Cross-language, both test-only, so not worth consolidating now; noting it so the *next* KDL-executing test reuses the Go helper rather than minting a third.
- `cmd/internal/zellijpane/zellijpane_test.go:12,36` and `cmd/internal/clipcmd/run_test.go:92` carry the same legacy `claude [~/workspace/parley.nvim]` fixture that `copy-on-select-test.sh` got an explanatory comment for. `clipcmd_test.go:19` already explains itself; the other two read as stale rather than deliberate. One-line comments would make the intent uniform.
- `panejson_kdl_test.go:44` — `TrimSuffix(…, "\"")` strips exactly one trailing quote; it would strip the wrong one if a layout line ever ended in an escaped `\"`. Not true today in either layout; only a robustness note.

## 5. Test coverage notes

- Real logic is pinned, not mocks: `frameTitle` is tested directly (`titlepoller_test.go:34-41`), `updateFrameTitles` through the `Runtime` fake, and the stale-twin case at `run_test.go:143-158` survived the `PaneInfo` shrink intact — it still proves the active-agent filter beats alphabetical last-wins across two ticks.
- `TestPaneCwdToleratesLegacyCwdDisplayField` (`panejson_kdl_test.go:144`) is the right migration test: a pre-#133 `pane-*.json` outlives the binary update, and `paneCwd`'s decode ignores unknown keys. Backward compatibility is covered, and the incidental legacy fixtures at `contextcmd_test.go:21`, `dispatcher_test.go:273`, `helper_equivalence_test.go:98`, `createflow_test.go:548` now have an intentional counterpart.
- Coverage of the *deleted* surface is correctly deleted, not stubbed: `TestTildeAbbrev`, `TestPaneTitle`, `TestAbbrevCwd`, `TestUpdateFrameTitlesCwdFallback` all removed with their subjects.
- Gap, per **I-2**: the `rename-pane` half of the KDL agent line is executed but unasserted.
- Structural gap worth knowing (not a finding, since it is mechanically closed): the conformance test reads `zellij/layouts/*.kdl`, while an installed `pair` runs the *embedded* copy. `runtimebundle-generate` is `.PHONY` and `$(RUNTIMEBUNDLE_ASSETS)` depends on it (`Makefile.local:92-95`), so the bundle is regenerated on every build and cannot drift; I confirmed both generated layouts on disk carry the new printf. No action needed — recording it so nobody "fixes" the generate dependency later without noticing what it guarantees.
- Not independently verified: `make test` exit 0. Recommend the main agent re-run `env -u PAIR_SESSION_ID -u PAIR_TAG make test` after applying I-1/I-2, since the memory note about session-env leakage into `review-target`/`changelog` tests applies.

## 6. Architectural notes

- **ARCH-DRY — pass** (one Minor). The two byte-identical abbreviators were resolved by *deletion*, and the `## Log` records why reviving `titlefmt` for a single caller was rejected. That is the stronger reading of the principle: one source of truth per fact, and the fact stopped existing. The only new duplication is the cross-language KDL extraction noted above.
- **ARCH-PURE — pass.** `frameTitle` stays a pure function tested without IO; the JSON decode stays in `runtime.go` behind the `Runtime` seam; the `Options.Home` removal shrinks the injected-IO surface rather than growing it. `createflow.go:472` inlines a bare value into a seam call — no logic migrated into the IO layer. `contextcmd` was already an INTEGRATION package (`paneCwd` reads a file), so hosting an `exec`-based test there does not violate a PURE claim.
- **ARCH-PURPOSE — flag (I-1).** The purpose is "no title anywhere carries a cwd, and no cwd-formatting function exists." The code fulfills it completely — this is not a cheap-subset delivery. But the shadow-sweep leaves README as a consumer that restates the model without deriving from it, and README is the surface a *user* consults. Fixing it is a two-line edit, which is why this is Important-and-cheap rather than blocking.
- **ARCH-MOCK — flag (I-2), but note this diff net-improves the position.** Before #133 the KDL producer sat outside every seam with no fake and no conformance check; now it has one, and the fakes sit on the same `PATH` the real launch resolves through, so production flow and test flow share the boundary. The remaining gap is that the `zellij` fake is stateless where the interaction we depend on (rename-pane receiving the right title) is observable. Upgrade it to record.
- Forward-looking: `pane-<tag>-<agent>.json` is now a two-key contract (`pane_id`, `cwd`) produced by shell `printf` and consumed by three Go readers (`contextcmd.paneCwd`, `titlepoller.OSRuntime.PaneFiles`, `launcher.legacyPaneMeta`) that each re-declare their own anonymous struct. That is tolerable at two keys; if a third key ever arrives, hoist a single shared `paneRecord` type rather than adding a fourth private struct — and extend the conformance test's assertions with it, since that test is now the only thing tying the producer to the consumers.

## 7. Plan revision recommendations

The plan-gate ledger (`workshop/plans/000133-agent-pane-title-drop-cwd-plan-gate.md`) shows `## Open findings — (none)`, so there is nothing deferred to pick up here. Two `## Revisions` entries on the issue file would close the gap between what Done-when demands and what the evidence shows:

1. **Live-check scope narrowed to layout 3.** Done-when (`:99-101`) requires the tab verified live "in **both** layouts (2 and 3)"; the `## Log` (`:365-373`) records layout 3 only, with layout 2 argued transitively via `tests/term-pane-shortcuts-test.sh` (layouts share the agent line) plus the conformance test running both. The Log states this honestly rather than over-claiming — the fix is to make Done-when match, not to re-run anything. Append a Revisions entry recording that the layout-2 arm is covered mechanically, not by observation, and why that is sufficient.

2. **Startup-title evidence restated.** The `## Log` (`:371-372`) credits "the conformance test executing the KDL line that consumes `${PAIR_PANE_TITLE:-agent}`" as coverage. Per **I-2** the test executes that line but asserts nothing about it. If I-2 is applied, update the Log to cite the new argv assertion; if it is deliberately deferred, correct the sentence to say the startup title rests on `TestRunLaunchForcedCreateClaude` alone and that the KDL→`rename-pane` hop is unobserved.
