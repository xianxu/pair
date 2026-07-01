# Boundary Review — pair#92 (milestone M2)

| field | value |
|-------|-------|
| issue | 92 — route internal calls through Go dispatcher |
| repo | pair |
| issue file | workshop/issues/000092-go-dispatch-internal-calls.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 17a917ab28716ec9887fa477805426f17a913ce0..HEAD |
| command | sdlc milestone-close --issue 92 --milestone M2 |
| reviewer | claude |
| timestamp | 2026-07-01T12:19:59-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The M2 boundary cleanly repoints all five Pair-owned call-sites from the legacy shim names to `pair <sub>`, and it does so faithfully: the code builds (`go build ./...` exit 0), and every touched test passes — including a real end-to-end `changelog-open-test.sh` that drives `pair scrollback-render` **and** `pair changelog` through the actual binary, the title-poller test that exercises `pair context`, and the new `slug_spawn_test.go`. I independently re-ran the ARCH-PURPOSE shadow-sweep and confirmed every remaining `pair-<name>` reference is either a comment, a build rule/shim/test, the gitignored bundle mirror, or the deliberately-deferred `pair-shell → pair-session-watch.sh` chain (#93) — no production exec call-site is stranded on a shim name. Nothing blocks the boundary; the findings below are all Minor (a stale comment, a bare-vs-explicit `pair` inconsistency, and an as-built/plan signature drift worth recording).

**1. Strengths**
- `slugSpawnCmd` (`cmd/pair-wrap/main.go:546`) is a textbook ARCH-PURE extraction: command *construction* is pulled into a testable value-returning helper, while the IO (`.Run()`) stays in the thin `maybeSpawnSlug` wrapper (`main.go:539`). The two unit tests pin both the `$PAIR_HOME/bin/pair slug` and bare-`pair` fallback forms plus the `PAIR_AGENT` env.
- `bin/pair-changelog-open:82-85` correctly dodges the pitfall the plan flagged (a space-containing `"$PCL_RENDER"` var won't exec): it holds only the binary in `PCL_BIN` and keeps the subcommand as a separate token in `INNER`. Old `PCL_RENDER`/`PCL_DISTILL` are fully removed with no orphan consumers.
- The `pair-scrollback-open:70-80` guard + invocation are updated in lockstep (both now test/exec `$PAIR_HOME/bin/pair`), so the "not built" diagnostic can't drift from the actual dependency.
- Streaming correctness preserved: `pair changelog`'s live stderr still lands in `$STATUS` via the subprocess's real stderr (`2>"$STATUS"`), so the nvim spinner survives — confirmed by the passing end-to-end test.

**2. Critical findings**
None.

**3. Important findings**
None.

**4. Minor findings**
- `bin/pair-title.sh:94` (and `:179`) — stale comment still says the count comes from `pair-context <tag> <agent>`; the call at `:105` is now `pair context`. Update the comment.
- `bin/pair-title.sh:105` — uses bare `pair context` (PATH-resolved), while every other M2 call-site uses the explicit `$PAIR_HOME/bin/pair`; the plan's Task 9 also specified the explicit form. Not a regression (the original was bare `pair-context`, and `pair-shell:564` prepends `$PAIR_HOME/bin` to PATH, so it resolves identically), but it's an inconsistency across the diff. Either align to `"$PAIR_HOME/bin/pair" context …` or leave it and record the parity rationale.
- `cmd/pair-wrap/main.go:546` — as-built `slugSpawnCmd(agent string)` reads `os.Getenv("PAIR_HOME")` internally, whereas the plan (Task 13 / Core Concepts) sketched `slugSpawnCmd(pairHome, agent)`. Fully testable via `t.Setenv` (the tests do this), so harmless — but it's plan/code drift worth a `## Revisions` note, mirroring how the M1 review recorded its runner-signature drift.

**5. Test coverage notes**
- Well covered: `pair context` (title poller test), `pair scrollback-render`+`pair changelog` (changelog-open end-to-end through the real binary), `pair slug` command construction (new unit tests).
- Gap (acceptable, but note it): neither the `bin/pair-scrollback-open` shell viewer nor `nvim/scrollback.lua:291`'s `renderer_command` has an automated test pinning the `pair scrollback-render` *invocation form* — a typo in the subcommand token in the Lua arg table (`{ bin, 'scrollback-render', … }`) would not be caught by any test; it relies on manual live-pane verification (which the plan's Tasks 11/12 explicitly declared). The render binary itself is exercised elsewhere, so only the two viewer call-site wirings are unpinned. Low risk given the triviality of the one-token change; flagged for completeness. A `changelog_test.lua`-style unit test asserting `renderer_command(paths)[2] == 'scrollback-render'` would close it cheaply.

**6. Architectural notes for upcoming work**
- ARCH-DRY — pass. The "resolve the pair binary" pattern (`$PAIR_HOME/bin/pair` else bare) now recurs across Go, shell, and Lua, but that's a cross-language boundary where no shared helper is feasible; within Go it's single-sourced in `slugSpawnCmd`. No actionable duplication.
- ARCH-PURE — pass, and exemplary in `slugSpawnCmd` (construction extracted, IO at the seam). The internal `os.Getenv` read is the only minor impurity; still deterministic + testable.
- ARCH-PURPOSE — pass. Shadow-sweep independently reproduced: all production call-sites derive from the dispatcher; the untouched `pair session-watch`/`pair continuation` production callers are out-of-scope deferrals (#93/agent-procedure) and are documented in `## Log`, not silent under-delivery.
- Atlas gate — satisfied. `atlas/architecture.md:100-107` accurately records the repointed set and correctly names the one remaining shim-name chain deferred to #93; the on-disk regenerated bundle carries `pair context`, confirming the gitignored-bundle revision. (`atlas/go-migration-inventory.md` remains the issue-close deliverable per the plan — not an M2 gap.)

**7. Plan revision recommendations**
- Add a `## Revisions` entry to `workshop/plans/000092-go-dispatch-internal-calls-plan.md` recording the two as-built deviations so the plan stops claiming what the code doesn't do (same discipline as the M1 as-built revision):
  1. **Task 13 signature.** Ships `slugSpawnCmd(agent string) *exec.Cmd` (reads `PAIR_HOME` from env), not the Core-Concepts/`Task 13` `slugSpawnCmd(pairHome, agent)`. Testable via `t.Setenv`; env-read chosen over a threaded param.
  2. **Task 9 invocation form.** `pair-title.sh` calls bare `pair context` (PATH-resolved via `pair-shell:564`'s `$PAIR_HOME/bin` prepend), not the plan's explicit `"$PAIR_HOME/bin/pair" context`. Parity with the pre-existing bare `pair-context`; the other four call-sites use the explicit path.
