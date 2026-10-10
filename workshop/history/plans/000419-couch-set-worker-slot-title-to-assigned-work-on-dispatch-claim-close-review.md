# Boundary Review — pair#419 (whole-issue close)

| field | value |
|-------|-------|
| issue | 419 — couch: set worker slot title to assigned work on dispatch claim |
| repo | pair |
| issue file | workshop/issues/000419-couch-set-worker-slot-title-to-assigned-work-on-dispatch-claim.md |
| boundary | whole-issue close |
| milestone | — |
| window | c5b19184c667b41c8a4ae38fa7a7517202b48137..c1a30df0ffc291a5ca6fd3d1cf685a472b869f02 |
| command | sdlc close --issue 419 |
| reviewer | claude |
| timestamp | 2026-10-09T21:33:33-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The change is small and does what the Spec asks. The recipient now labels its own slot through the existing `PublishedSummary` path, and the protocol gains no new machinery. The `publish-description` record dump is replaced by one confirmation line, and the test covers both the set and the clear case. The targeted test passes (`go test ./cmd/internal/couchcmd -run TestPublishDescription` → ok). Nothing blocks shipping. One cheap fix is worth making first: the skill text shows `pair#N` as the label template, but the Spec and Done-when say `repo#N`, and this skill is shared across repos (the request came from ariadne:1). Followed literally, it would mislabel non-pair work.

**1. Strengths**
- `cmd/internal/couchcmd/run.go:935-944`: the rendering checks the op name, so `Detach`, which also returns a bare `ThreadRecord`, keeps its old output.
- `run_test.go:628-649`: the test checks exact stdout for both states named in Done-when (set and clear), next to the existing persistence checks.
- No pair→sdlc call. The agent connects sdlc and couch from its own shell, which matches the layering rule.
- Nothing in nvim reads this stdout. The detached path in `nvim/init.lua:822` ignores it, and the synchronous `!!`/clear path at `init.lua:847-855` only checks the exit code. Changing the output breaks no consumer.

**2. Critical findings**
None.

**3. Important findings**
- **The skill hardcodes the repo** (`cmd/internal/couchcmd/skills/couch/SKILL.md:120-123`). The template is `'pair#N <short title>'`, while the Spec and Done-when say `repo#N <title>`. Every Couch agent loads this skill, including those working in ariadne and other peers. Fix: make the template `<repo>#N <short title>` and keep `pair#300 judge verdict` only as the example. This is the only instance in the window; the atlas line at `atlas/couch.md:704` already says `repo#N`.

**4. Minor findings**
- `SKILL.md:121-122`: the inline-code command breaks a line inside the quoted argument (`'pair#N⏎   <short title>'`). Markdown renders that as a space, but an agent reading the raw text could copy a newline and indent into the label. Keep the command on one line.
- Spec bullet 3 says nobody saw the dump "because nvim runs the op detached". Only the `!` path is detached; the `!!`/clear path runs it synchronously and discards stdout. The conclusion holds, but the stated reason is imprecise.
- The dogfood Done-when clause (this slot published the label and `couch --list` shows it) can't be checked from the diff. Make sure `--verified` cites the `couch --list` output.

**5. Test coverage notes**
Both output branches (set and clear) are pinned by exact-string assertions. No test confirms that `Detach` output is unchanged, but nothing in its render path changed, so that is acceptable.

**6. Architectural notes**
- **ARCH-DRY: pass.** The two render branches are a one-line format each, and no similar summary renderer exists in the tree.
- **ARCH-PURE: pass.** `render` is a pure function writing to an `io.Writer`, and the tests read its output through the runtime fake.
- **ARCH-PURPOSE: pass, with the caveat above.** The purpose is fleet visibility of which slot holds which work. The `pair#N` template undercuts that for slots in other repos, which is why I filed it as Important.

**7. Plan revision recommendations**
None needed, beyond the Spec wording note on detached vs synchronous if the author wants it exact.

```findings
findings:
  - id: new
    severity: Important
    family: shared-skill-repo-agnostic
    title: |
      couch SKILL.md receiving-dispatched-work step hardcodes pair#N instead of repo#N
    detail: |
      SKILL.md:120-123 gives the label template as 'pair#N <short title>', but the skill is loaded by Couch agents in every repo, and the Spec and Done-when say repo#N. Use <repo>#N in the template and keep pair#300 only as the example. This is the only instance in the window; atlas/couch.md:704 already says repo#N.
  - id: new
    severity: Minor
    family: skill-command-copyable
    title: |
      Skill's publish-description command breaks a line inside the quoted argument
    detail: |
      SKILL.md:121-122 wraps 'pair#N / <short title>' across a source line inside inline code. An agent copying the raw text could include a newline and indent in the label. Keep the command on one line.
  - id: new
    severity: Minor
    family: doc-claim-from-memory
    title: |
      Spec says nvim runs publish-description detached, but the !!/clear path runs it synchronously
    detail: |
      nvim/init.lua:847-855 runs it with vim.system():wait() and discards stdout. The conclusion that nobody saw the dump still holds; only the stated reason is imprecise.
```

---

## Re-review — 2026-10-09T21:34:43-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 419 — couch: set worker slot title to assigned work on dispatch claim |
| repo | pair |
| issue file | workshop/issues/000419-couch-set-worker-slot-title-to-assigned-work-on-dispatch-claim.md |
| boundary | whole-issue close |
| milestone | — |
| window | c5b19184c667b41c8a4ae38fa7a7517202b48137..c3a7d4d02b79a81b60076b03daee41e364ca92d6 |
| command | sdlc close --issue 419 |
| reviewer | claude |
| timestamp | 2026-10-09T21:34:43-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

All three prior findings are fixed, and I found nothing new. The skill no longer hardcodes `pair#N`. It now tells the agent to use the issue's own repository, with `ariadne#300` as the example. The command is now a one-line code block. The Spec now gives the right reason why nobody saw the dump: nvim detaches for `!` tags and waits for `!!` and clear, but throws stdout away either way. Two things about the code change hold up. The one-line output for `publish-description` is chosen by the operation's name, so `Detach` still prints the full record even though it returns the same `ThreadRecord` type. The test checks the output both when a summary is set and when it is cleared.

1. **Strengths**
   - `cmd/internal/couchcmd/run.go:937`: the output change applies only to `publish-description`, because `Detach` returns the same type. The issue's Log records this decision.
   - `cmd/internal/couchcmd/run_test.go:628-650`: the test checks the exact output for both setting and clearing a summary, so the Done-when clause is tested both ways.
   - `SKILL.md:125` uses `--description=`, the same form nvim already uses (`nvim/init.lua:816`), so the command it teaches is one already in use.
   - The design adds nothing new. It reuses the existing `PublishedSummary` field, and pair never calls sdlc (the agent connects the two from its shell), so the layering stays intact.

2. **Critical:** none.
3. **Important:** none.
4. **Minor:** none new.
   - Unchanged by this diff: `atlas/couch.md:701` still shows the argument as positional (`<text>`) while the skill and nvim use `--description=`. Optional cleanup.
5. **Test coverage:** both output branches are tested. The skill text and atlas text are prose, so they need no tests.
6. **Architecture:**
   - ARCH-DRY passes: the new output code has no duplicate elsewhere.
   - ARCH-PURE passes: the output is a small formatting step at the IO edge, and the test calls it directly.
   - ARCH-PURPOSE passes: the skill applies to every repo, and the Done-when matches the operator's decision recorded in the Spec.
7. **Plan revisions:** none.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      SKILL.md:119-125 now says "Name the issue's own repository" with ariadne#300 as the example; no pair#N template remains.
  - id: BR-2
    disposition: addressed
    note: |
      The command is now a single-line fenced sh block at SKILL.md:124-126.
  - id: BR-3
    disposition: addressed
    note: |
      Spec now reads "detached for ! tags, waited for !! and clear", which matches nvim/init.lua.
```
