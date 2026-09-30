# Boundary Review — pair#358 (whole-issue close)

| field | value |
|-------|-------|
| issue | 358 — !! in the draft publishes the previous prompt as the slot's description |
| repo | pair |
| issue file | workshop/issues/000358-in-the-draft-publishes-the-previous-prompt-as-the-slot-s-description.md |
| boundary | whole-issue close |
| milestone | — |
| window | 06973c5da7f85fbcc4108c80cf88da657290e32a..ad9cced960426ab5faa1aed4659c874f2039cef8 |
| command | sdlc close --issue 358 |
| reviewer | claude |
| timestamp | 2026-09-30T11:56:31-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

This change delivers the whole issue. `bang_tag.parse` checks `!!` before the `!` rule, so the forgotten-`!` fix works in both forms: bare `!!` and `!! sentence`. `one_line` and `previous_description` are pure and unit-tested. `submit_operator_text` sends both forms to a synchronous publish that runs before anything is sent to the agent. A false result keeps the draft through the existing `send_and_clear` / `ship_buffer_and_reset` gate. I ran `nvim/bang_tag_test.lua` and `tests/bang-tag-nvim-test.sh` on head and both pass, including the three new `describe*` cases. The suite is wired into `make test` through `test-bang-tag`. Nothing blocks shipping. The four findings are Minor.

**1. Strengths**
- The integration test's history fake now stores what it receives. It appends to `PAIR_LOG_PATH` (`nvim/bang_tag_integration_test.lua:11-14`), so `!!` reads real history. Every zellij executor call is counted, so any agent write or submit fails the test. This is the stateful fake the Done-when asks for.
- `bang_tag.lua` stays pure: `one_line` and `previous_description` have no IO. Their unit tests cover the cap at exactly 120 characters and at 121, counting in characters rather than bytes, and a multi-line `!` prompt that the agent received verbatim.
- The publish is synchronous, and its result decides whether the draft clears (`nvim/init.lua:808-836`). A failure or a missing `couch` binary is caught by the `pcall` and reported with an ERROR notification. It is never reported as success.
- Extracting `in_couch_thread()` removed a guard that had been written twice.
- The README and both atlas files are updated in the same range.

**2. Critical findings:** none.

**3. Important findings:** none.

**4. Minor findings**
- **README wording is now ambiguous.** `README.md:273` still says "There is no `!!` escape." The paragraph just above it now gives `!!` a meaning, so a reader could take the two as contradicting each other. Rephrase it, for example: "there is no way to send a literal leading `!` from the draft."
- **Duplicated code (ARCH-DRY).**
  - The couch argv `{'couch','--internal','publish-description','--description='..d}` is built twice, at `init.lua:803` and `init.lua:826`.
  - `normalization.lua` is loaded with `dofile` twice, at `init.lua:790` and `init.lua:1008`.
  - Fix: one argv helper, and hoist a single normalization load above both users.
- **Two failure paths of the `!!` publish are untested.** The ENOENT path (no `couch` on PATH) and the 5 s timeout path have no test. The `missing` and `slow` cases exist only for `!`.
- **Edge cases in `one_line`'s UTF-8 truncation.**
  - An invalid lead byte or a stray continuation byte doesn't match the pattern, so it is silently dropped when the text is cut.
  - A cut can leave a space right before `…`.

**5. Test coverage notes**
Every Done-when clause is covered:

| Done-when clause | Covered by |
|---|---|
| Bare `!!` and `!! sentence` publish | `describe` case, including `!!sentence` with no space |
| Multi-line and bang-tagged previous prompts | Both seeded in `describe` |
| No agent traffic and no history append | `silent()`, asserted for every form |
| Outside couch | `describe-standalone`, both forms |
| No previous prompt | `describe` (first call) |
| Failed publish | `describe-nonzero` |
| `!` behavior unchanged | Existing `couch` / `standalone` / `missing` / `nonzero` / `slow` / `retry` cases still pass |

- The "empty -1 after stripping" edge is covered by `check_previous('!', nil)` at unit level. That is enough, because a bare `!` can never be logged.

**6. Architectural notes**
- **ARCH-DRY: pass.** Only the minor duplication above.
- **ARCH-PURE: pass.** Parsing, the one-line rule and deriving the previous description are pure. `describe_couch_thread` is a thin IO shell.
- **ARCH-PURPOSE: pass.** Both forms and every Done-when clause are delivered, with no deferral.
- **For #357:** `if tag and not tag.agent_text` (`init.lua:840`) is the "draft action that doesn't submit" route. #357 should extend that route rather than add a parallel branch.
- **Old logs:** an entry written before this change as `!! foo` would describe as `!! foo`. That is harmless.

**7. Plan revision recommendations:** none. The plan matches the code.

```findings
findings:
  - id: new
    severity: Minor
    family: doc-claim-stale-after-change
    title: |
      README still says "There is no `!!` escape" right after documenting `!!`
    detail: |
      README.md:273. The sentence meant "no way to send a literal leading `!`". Next to the new `!!` paragraph it reads as a contradiction. Reword it. This is the only instance in the window: the atlas bullets are consistent.
  - id: new
    severity: Minor
    family: shared-helper-not-extracted
    title: |
      Couch publish argv built twice and normalization.lua loaded with dofile twice in init.lua (ARCH-DRY)
    detail: |
      init.lua:803 and init.lua:826 build the same publish-description argv. init.lua:790 and init.lua:1008 each dofile normalization.lua. Extract one argv helper and hoist a single normalization load above both users.
  - id: new
    severity: Minor
    family: failure-path-untested
    title: |
      The `!!` publish's ENOENT and timeout paths have no integration case
    detail: |
      Only describe-nonzero exercises a failure. A describe-missing case (PATH without couch) and a timeout case would pin the pcall branch and the 5 s bound. The missing and slow cases exist only for `!`.
  - id: new
    severity: Minor
    family: utf8-truncation-edges
    title: |
      one_line drops invalid UTF-8 bytes when it cuts, and can leave a trailing space before the ellipsis
    detail: |
      nvim/bang_tag.lua:20. The char pattern skips 0xC0/0xC1/0xF5-0xFF lead bytes and stray continuation bytes. A cut at a space yields "word …". Cosmetic.
```

---

## Re-review — 2026-09-30T13:16:00-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 358 — !! in the draft publishes the previous prompt as the slot's description |
| repo | pair |
| issue file | workshop/issues/000358-in-the-draft-publishes-the-previous-prompt-as-the-slot-s-description.md |
| boundary | whole-issue close |
| milestone | — |
| window | f0b3b78ba4c24d6048040c341b5fdc8efc8e719f..e5e35550d534ee2e4af3b19599bea0389ed9313b |
| command | sdlc close --issue 358 |
| reviewer | codex |
| timestamp | 2026-09-30T13:16:00-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range implements both `!!` forms without agent traffic or history append. Focused tests pass, and the truncation regression tests fail against the previous implementation. Three prior findings are addressed; BR-3 remains partially open as a Minor testing gap. No blocking findings.

1. **Strengths**
   - Pure parsing and description normalization remain isolated in `nvim/bang_tag.lua`.
   - `nvim/init.lua:851` routes description actions before agent submission.
   - Integration tests use persisted history and count every agent executor call.
   - README and atlas document the new syntax in the same range.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - **BR-3 remains open:** `describe-missing` covers ENOENT, but the synchronous timeout at `nvim/init.lua:841` still lacks a regression case. Add a blocked publisher case asserting bounded return, failure notification, and no agent/history effects.

5. **Test coverage**
   - Passed `make test-bang-tag` and `nvim/bang_tag_test.lua`.
   - Independently confirmed both new truncation assertions fail against the pre-fix implementation.
   - Draft retention follows the existing callers’ false-result guards; focused tests exercise the submission boundary rather than actual draft keybindings.
   - No full-suite or live Couch verification performed.

6. **Architecture**
   - **ARCH-DRY — pass:** shared publication argv and normalization load.
   - **ARCH-PURE — pass:** deterministic text processing separated from publication IO.
   - **ARCH-PURPOSE — pass:** both requested forms and history sourcing delivered.
   - **ARCH-MOCK — pass:** production subprocess seam exercised with a controllable executable; history fake persists state.
   - **ARCH-CONSTRAINTS — pass with BR-3 caveat:** 120-character cap and five-second subprocess timeout implemented.
   - **ARCH-SECURE — pass:** argv arrays avoid shell interpolation; failures notify instead of claiming success.
   - **ARCH-ORDER — pass:** synchronous publication determines the return value; no new asynchronous state machinery.
   - **ARCH-FUNERAL — pass:** reuses existing description storage and creates no new runtime artifact family.

7. **Plan revisions:** none required. Correct the Log’s “all fixed” summary to acknowledge BR-3 remains partially unresolved.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      README.md:272 replaces the ambiguous escape claim; the preceding paragraph documents both description forms, consistent with bang_tag.parse.
  - id: BR-2
    disposition: addressed
    note: |
      nvim/init.lua now has one publish_argv helper and one normalization load shared by both consumers; focused integration tests pass.
  - id: BR-3
    disposition: not-addressed
    note: |
      describe-missing now exercises ENOENT successfully, but no describe timeout case exists. The five-second bound remains untested; retain this Minor finding.
  - id: BR-4
    disposition: addressed
    note: |
      Truncation preserves the retained byte prefix and trims whitespace before the ellipsis. Both added regression assertions pass on HEAD and independently fail against the previous implementation.
```

---

## Re-review — 2026-09-30T13:19:54-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 358 — !! in the draft publishes the previous prompt as the slot's description |
| repo | pair |
| issue file | workshop/issues/000358-in-the-draft-publishes-the-previous-prompt-as-the-slot-s-description.md |
| boundary | whole-issue close |
| milestone | — |
| window | f0b3b78ba4c24d6048040c341b5fdc8efc8e719f..4900c15912e3005c17aa0d0fbedc9aebdefced49 |
| command | sdlc close --issue 358 |
| reviewer | codex |
| timestamp | 2026-09-30T13:19:54-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range delivers both `!!` forms without agent traffic or history appends. Pure tests and all ten integration cases pass. No blocking findings; BR-3 remains partially unresolved as a Minor testing gap.

1. **Strengths**
   - Description parsing and normalization remain pure in `nvim/bang_tag.lua`.
   - `nvim/init.lua:851` routes description actions before agent submission.
   - Integration tests persist history and count every agent executor call.
   - README and atlas document the new syntax consistently.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - **BR-3:** `nvim/init.lua:841` implements the five-second timeout, but no integration case exercises it. ENOENT is now covered at `nvim/bang_tag_integration_test.lua:77`. Add a blocked-publisher case asserting bounded return, error notification, and no agent/history effects.

5. **Test coverage**
   - Passed `nvim -l nvim/bang_tag_test.lua` and `bash tests/bang-tag-nvim-test.sh`.
   - Independently confirmed both BR-4 regression assertions fail against the pre-fix implementation and pass now.
   - Draft retention verified through caller guards; focused tests do not exercise actual draft keybindings.
   - No full-suite or live Couch verification performed. Working tree remains clean.

6. **Architecture**
   - **ARCH-DRY — pass:** shared argv builder and normalization load.
   - **ARCH-PURE — pass:** deterministic text processing separated from publication IO.
   - **ARCH-PURPOSE — pass:** both requested forms use the specified history source.
   - **ARCH-MOCK — pass:** controllable subprocess fixture and persisted history exercise the production boundary.
   - **ARCH-CONSTRAINTS — pass with BR-3 caveat:** description cap and subprocess timeout implemented; timeout lacks regression coverage.
   - **ARCH-SECURE — pass:** argv arrays avoid shell interpolation; publication errors are surfaced.
   - **ARCH-ORDER — pass:** synchronous completion gates draft clearing; unsuccessful outcomes preserve input.
   - **ARCH-FUNERAL — pass:** existing description storage reused; no new durable artifact family.

7. **Plan revisions:** none required.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      README.md now says no draft syntax sends a literal leading bang, consistent with bang_tag.parse and the preceding documentation of both !! forms.
  - id: BR-2
    disposition: addressed
    note: |
      nvim/init.lua shares one publish_argv builder and one normalization load across both consumers; focused integration tests pass.
  - id: BR-3
    disposition: not-addressed
    note: |
      describe-missing covers ENOENT, but the five-second synchronous timeout still has no integration case. Retain the existing Minor advisory.
  - id: BR-4
    disposition: addressed
    note: |
      Truncation retains the byte prefix and trims whitespace before the ellipsis. Both regression assertions independently fail against the pre-fix implementation and pass at the pinned head.
```
