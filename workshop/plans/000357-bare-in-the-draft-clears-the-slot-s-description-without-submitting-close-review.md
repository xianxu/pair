# Boundary Review — pair#357 (whole-issue close)

| field | value |
|-------|-------|
| issue | 357 — Bare ! in the draft clears the slot's description without submitting |
| repo | pair |
| issue file | workshop/issues/000357-bare-in-the-draft-clears-the-slot-s-description-without-submitting.md |
| boundary | whole-issue close |
| milestone | — |
| window | e5e35550d534ee2e4af3b19599bea0389ed9313b..9f9b25fc77bd9bd9cffb6569f89b3d4c17bdfd16 |
| command | sdlc close --issue 357 |
| reviewer | claude |
| timestamp | 2026-09-30T13:29:32-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

A bare `!` now works as the issue asks. `bang_tag.parse` returns `{ clear = true }` for it, and `submit_operator_text` sends that to the same synchronous no-send path that `!!` uses. That path publishes `--description=` and reports the result. Every `## Done when` clause has a test:
- **Clear with fallback:** the Go CLI test runs the exact argv and checks that `DisplaySummary()` falls back to the operator's description.
- **Nothing reaches the agent:** the stateful executor fake counts every zellij call, so the `silent()` helper would catch any agent traffic.
- **Outside couch:** the `describe-standalone` case checks that nothing is sent and a warning appears.
- **`! text` unchanged:** the #337 cases still pass.

Failure is covered too: the `describe-nonzero` case checks that a failed clear shows a "could not clear" error and keeps the draft. I ran `nvim/bang_tag_test.lua`, `tests/bang-tag-nvim-test.sh` (all 10 cases) and `go test ./cmd/internal/couchcmd -run PublishDescription` at HEAD 9f9b25fc, and all passed. Nothing blocks shipping.

1. **Strengths**
   - Reusing the #358 path rather than adding a parallel one (`nvim/init.lua:828-851`). Clearing is just an empty description, so no new couch op was needed. The Go test pins the claim that `--description=` binds an empty value (`cmd/internal/couchcmd/run_test.go:650-688`).
   - The old `agent_text == ''` no-op branch in `submit_operator_text` is gone, so there are fewer route types.
   - The integration test checks both how many times couch was called and the exact argv, including the trailing empty `--description=`.
   - The README now describes what bare `!` does and says the draft is kept on failure for both `!!` and `!`.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `nvim/bang_tag.lua:67` (`previous_description`): the change also affects history entries that are `!!` or `!! text`. These can only come from logs written before #358, since those lines never enter the log now. Before, such an entry described itself literally. Now it gives nil, and `!!` warns "no previous prompt". That's defensible, but no `check_previous` case pins it.
   - `atlas/couch.md:284`: a comma is missing in "a `!` tag line (#337) a `!!` describe line (#358), and …".
   - `nvim/bang_tag.lua:4`: the rewrapped header comment line is much longer than the lines around it.

5. **Test coverage notes**
   - Covered: parse for `!`, `!   ` and `  !\t\n`; clear in couch, outside couch and on failure; no agent traffic and no log entry; the Go CLI clear plus fallback.
   - Not covered: legacy `!!` history entries in `previous_description` (see above).

6. **Architecture**
   - **ARCH-DRY: pass.** One publish path handles set and clear, and `publish_argv` is shared.
   - **ARCH-PURE: pass.** The parse logic is pure and unit-tested without IO. The IO stays in `describe_couch_thread`.
   - **ARCH-PURPOSE: pass.** Every clause of Done-when is delivered, including the fallback and the visible error on failure.

7. **Plan revision recommendations:** none. The plan's "couch store/op test" item is delivered by the Go CLI test.

```findings
findings:
  - id: new
    severity: Minor
    family: behavior-change-needs-pinning-test
    title: |
      previous_description now returns nil for legacy logged !! / !! text entries, unpinned by a test
    detail: |
      nvim/bang_tag.lua:67 changed `if tag and tag.agent_text` to `if tag then text = tag.agent_text or ''`, which also flips `!!`-family entries (only possible in pre-#358 logs) from self-description to nil. Add check_previous cases for '!!' and '!! text' to pin the intended result. This is the only instance in the window.
  - id: new
    severity: Minor
    family: doc-prose-typo
    title: |
      atlas/couch.md list missing a comma after "(#337)"
    detail: |
      "a `!` tag line (#337) a `!!` describe line (#358), and a bare `!` clear line" needs a comma after (#337). Also, the header comment line at nvim/bang_tag.lua:4 is much longer than the lines around it.
```
