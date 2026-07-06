# Boundary Review — pair#97 (whole-issue close)

| field | value |
|-------|-------|
| issue | 97 — agent name on pane is wrong — stale pane-<tag>-<agent>.json collides on pane_id |
| repo | pair |
| issue file | workshop/issues/000097-agent-name-on-pane-is-wrong-stale-pane-tag-agent-json-collides-on-pane-id.md |
| boundary | whole-issue close |
| milestone | — |
| window | 0674d8c4139ea3c29077664b20e13cafeaa07d4e..HEAD |
| command | sdlc close --issue 97 |
| reviewer | claude |
| timestamp | 2026-07-05T21:34:51-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The two-part fix delivers exactly what pair#97 specs: **(a)** `updateFrameTitles` now skips any pane whose `pane.Agent != opts.Agent`, so the frame renders only the active agent even when a stale `pane-<tag>-<other>.json` twin shares the live `pane_id` — killing the alphabetical last-wins/flip-flop symptom immediately (even for twins already on disk or left by a crash that bypasses cleanup); **(b)** `runCleanup` adds `pane-<tag>-<quitAgent>.json` to the sidecar-removal list, stopping new twins at the source on clean Alt+x quit. I independently verified the collision analysis (the `PaneFiles` glob at `runtime.go:63` is the *sole* consumer that alpha-expands the twin; `contextcmd.go:75` and `rename.go:44` both read by exact agent name, so neither collides — matching the Spec's traced blast radius), that both changed packages pass `go test` (env-scrubbed) and `go vet`, and that the regression test genuinely catches the bug (dropping the filter yields 4 renames ending on `codex`; the assertion demands exactly 1, to `claude`). All three "Done when" items and every Plan checkbox are delivered. Nothing blocks the boundary.

**1. Strengths**
- `run.go:182` — the fix is a one-line pure filter over the existing `Runtime` seam, unit-tested with the fake harness (ARCH-PURE); no IO mixed in. The two-pane invariant + `opts.Agent`-from-`InferAgent` reasoning is sound, and the failure mode is safe (no match → no rename, i.e. stale frame rather than *wrong* frame).
- `lifecycle.go:94` — fix (b) reuses `quitAgent` (already resolved at `:58` for the scrollback/resume-hint path), so the pane file joins the other per-(tag,agent) sidecars keyed on the *inferred* active agent, not `step.agent` — correct for the attach/resume path where those differ (ARCH-DRY: no new resolution logic).
- `run_test.go:153` — the regression test reproduces the real mechanism (two `PaneInfo` on one `pane_id`, second identical tick asserting no flip-flop) and the assertion (`len==1` + exact title) is robust to `PaneFiles` ordering, not just the alpha case. The honest NOTE about the filter coupling is good future-proofing.
- Docs updated in-range: `atlas/architecture.md:244` now describes "renders only the pane whose `Agent == opts.Agent`" and the twin-cleaned-on-quit half — accurate to both code changes.

**2. Critical findings** — none.

**3. Important findings** — none.

**4. Minor findings**
- `lifecycle_test.go:87` — the quit-cleanup test exercises only the *fallback* branch (`inferAgent["bugfix"]` unset → `quitAgent == step.agent == "claude"`). The removal-string construction plainly uses `quitAgent` (low risk), but no test pins the case where `InferAgent` returns an agent *different* from `step.agent` (the resume/attach path — e.g. inferred `codex`, launch default `claude`). A single case there would lock in "removal follows the inferred agent, not the launch default."
- `run.go:170-179` — the 10-line doc comment is dense, but it documents a genuinely subtle, easy-to-break invariant; justified, not a nit worth changing.

**5. Test coverage notes**
- The kind of bug shipped (twin hijacking the frame) is directly covered by `TestUpdateFrameTitlesIgnoresStaleAgentTwin`, and the leak-at-source by the extended `TestRunLaunchQuitCleanup` removal-set assertion. Combined coverage is adequate for an atomic fix; the only gap is the quitAgent≠step.agent cleanup permutation (minor, above). Existing frame tests (`WithCount`/`NoCount`/`CwdFallback`/`UnchangedSkip`) still pass unchanged since they use `Agent:"claude"` matching `fixtureOpts()`.

**6. Architectural notes for upcoming work**
- ARCH-DRY: **pass** — reuses `frameCache`, the sidecar loop, and `quitAgent`; no duplicated logic.
- ARCH-PURE: **pass** — the fix is pure filter logic behind the seam; `runCleanup` change is a one-line addition to existing glue.
- ARCH-PURPOSE: **pass** — shadow-swept all `pane-<tag>-*` consumers; the poller glob was the only collision site and it's fixed, the exact-name readers (`contextcmd`, `rename`) never collided. Purpose (frame shows the *active* agent) fulfilled, not a cheap subset.
- Longer-term (not this issue): `runtime.go:79-80` derives the agent by `TrimPrefix(base, "pane-<tag>-")`. Because both the glob and the prefix embed the full tag, a tag/agent naming collision (e.g. a tag ending in a dash) *could* mis-parse the suffix. Out of scope for #97 and the active-agent filter renders it harmless (a mis-parsed agent simply won't equal `opts.Agent`), but worth a note if pane-file naming is ever revisited.

**7. Plan revision recommendations** — none. The plan, Spec Decision, "Done when", and code are consistent; the atlas already reflects the shipped behavior.
