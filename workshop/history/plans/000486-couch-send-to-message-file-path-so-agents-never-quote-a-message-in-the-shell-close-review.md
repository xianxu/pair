# Boundary Review — pair#486 (whole-issue close)

| field | value |
|-------|-------|
| issue | 486 — couch --send-to: --message-file PATH, so agents never quote a message in the shell |
| repo | pair |
| issue file | workshop/issues/000486-couch-send-to-message-file-path-so-agents-never-quote-a-message-in-the-shell.md |
| boundary | whole-issue close |
| milestone | — |
| window | 414e83e019fadb6e99d4dc941bfb4eb99a932fdf..17dd7aacebfc635ac14d7d30a2699a366d847bd5 |
| command | sdlc close --issue 486 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T08:33:59-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

I've gone through the diff. Now I'm checking one place that generates a suggested `--message` command, and then running the tests.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The change works and is scoped tightly. `--message-file PATH` takes the place of `--message TEXT` in the fixed `--send-to` shape, so the parser still does no IO. Passing both flags, or neither, is refused by the existing shape check. `readMessageFile` reads the file when the message is sent, caps the read at `MaxBodyBytes+2`, strips one trailing `\n`, and refuses an empty, whitespace-only, missing, unreadable, directory or oversized file before any RPC. I checked the boundary math: once a read is truncated, the body is always at least `Max+1` bytes, so it is always refused, and a file of exactly `Max` bytes plus a newline is sent. The help text, the skill and the atlas are updated.

The couchcmd package didn't build at first because the gitignored runtime assets were missing; that's a checkout issue, not a code one. I regenerated them with the Makefile recipe, and then `go test ./cmd/internal/couchcmd/` and `./cmd/internal/couchmessage/` both pass, including the four new or extended tests. One thing blocks a clean SHIP: README.md still lists only `--message TEXT` for `--send-to`. That is a docs-gate gap and a cheap fix.

**1. Strengths**
- `cli.go:370-382`: the new flag reuses the fixed four-argument shape, so the "both or neither" rule comes from the existing `bad()` path rather than new mutual-exclusion code. The parser stays pure.
- `messages.go:129-147`: the read is bounded, and the size and emptiness errors name the file before any RPC. The broker's validation is unchanged, so the message's admission rules still have one owner.
- `messages_test.go:137-202`: the tests check a real verbatim body (backticks, `$(…)`, `$((…))`, `<(…)`, `>(…)`, quotes, tabs, an embedded blank line), each refusal case with "no RPC was made" asserted, and the exact-limit case where the file is sent.
- The receipt carries the body, not the path: `r.Body` is replaced before the request is built, which matches the Spec.

**2. Critical:** none.

**3. Important**
- **README.md:435-438 not updated.** The four `couch --send-to … --message TEXT` usage lines in README never mention `--message-file PATH`. Help (`run.go:1330`), the skill and the atlas were updated, so README is the only consumer of this new flag left behind. Fix: add a line such as `couch --send-to repo[:N]|hub [--agent NAME] --message-file PATH   send a file's contents verbatim (less one trailing newline)`. Every place that restates this surface: `run.go:1330` (updated), `SKILL.md` (updated), `atlas/couch.md:186` (updated), `README.md:435-438` (**missing**).

**4. Minor**
- If the file has CRLF line endings, a `\r` is left after the strip, and the broker refuses it with its generic body error, which doesn't name the file. Consider stripping `\r\n` at the end, or naming CR in the `--message-file` error.
- The first Done-when clause (a live delivery from a sandboxed slot, compared against `--message`) is deferred to the TL, as the issue log records. It is unverified at this boundary.
- The `unreadable` case (`0o000`) would pass the read when tests run as root; that only matters in CI that runs as root.
- No test covers a file much larger than the limit (where the truncated read applies). The arithmetic is correct, so this is low risk.

**5. Test coverage**
Every case listed in the Spec's Done-when is covered: verbatim body, both flags, neither flag, empty, missing and unreadable. Directory, whitespace-only, oversized, exact-limit, and a file path starting with `-` (`--json`) are covered too. Nothing tests that the help text mentions `--message-file`; I confirmed that by reading the diff.

**6. Architecture**
- **ARCH-DRY: pass.** The empty and size checks in `readMessageFile` overlap with `couchmessage` validation on purpose: they give file-specific errors before the RPC and use the shared `couchmessage.MaxBodyBytes` constant rather than a copy of the limit.
- **ARCH-PURE: pass.** The parser does no IO, and `readMessageFile` is a small IO layer at the send site.
- **ARCH-PURPOSE: pass, apart from the README gap above.** I checked other code that builds a `--message` command for an agent: `couchcore.SendToCommand` (`slot_operation.go:125`) is only fed `RestoreWorkspaceMessage`, which is fixed text with no shell syntax, so it doesn't need `--message-file`.

**7. Plan revisions:** none needed. The plan's checklist matches the code, and leaving out `--remove-after-send` is declared in the plan.

```findings
findings:
  - id: new
    severity: Important
    family: docs-surface-sync
    title: |
      README.md send-to usage lines omit the new --message-file PATH flag
    detail: |
      README.md:435-438 lists only --message TEXT forms. run.go:1330 help, SKILL.md and atlas/couch.md:186 were updated; README is the one place restating the surface that was not. Add a --message-file PATH usage line.
  - id: new
    severity: Minor
    family: message-file-line-endings
    title: |
      CRLF message file leaves a trailing CR that the broker refuses with a generic error
    detail: |
      readMessageFile strips only one trailing newline, so a CRLF file keeps a CR and the broker's control-byte check refuses it without naming the file. Strip CRLF too, or name CR in the --message-file error.
  - id: new
    severity: Minor
    family: live-done-when-deferred
    title: |
      Live sandboxed-slot Done-when check deferred to the TL
    detail: |
      The issue log records the deferral, which needs a Couch restart. The live check (backticks, command and process substitution, newlines via --message-file, versus --message staying sandboxed) is not verified at this boundary.
```

---

## Re-review — 2026-10-12T08:38:50-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 486 — couch --send-to: --message-file PATH, so agents never quote a message in the shell |
| repo | pair |
| issue file | workshop/issues/000486-couch-send-to-message-file-path-so-agents-never-quote-a-message-in-the-shell.md |
| boundary | whole-issue close |
| milestone | — |
| window | 414e83e019fadb6e99d4dc941bfb4eb99a932fdf..706d4e020e28799d035b482c5af04939e09c226b |
| command | sdlc close --issue 486 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T08:38:50-07:00 |
| verdict | SHIP |

## Review

Generating the gitignored runtime bundle so the couchcmd tests can build and run.

```verdict
verdict: SHIP
confidence: high
```

**Verdict: SHIP.** All three prior findings are disposed, and nothing above Minor remains. The README fix (BR-1) is in, so all four places that list the send usage now show `--message-file`. The CRLF fix (BR-2) has a real regression test: I reverted the fix in the disposable checkout and `TestMessageCLIRefusesAnUnusableMessageFile` failed with the broker's "terminal controls" error, then passed again once restored. The parser stays IO-free, the file read sits in the thin send path, and every test in the message family passes. The checkout first failed to build because the gitignored runtime bundle wasn't there, so I generated it with `generatecmd` before running the tests. Two Minor items remain. One is a sibling of BR-2: a CR inside a multi-line CRLF file is still refused without naming the file. The other is the live sandbox check, still deferred to the TL.

1. **Strengths**
   - `cli.go:370-382`: `--message-file` takes the place of `--message` in the fixed `--send-to` shape. Giving both or neither falls through to the existing shape refusal with no new logic, and the tests cover both cases (`cli_test.go:118`, `:123-126`).
   - `messages.go:129-152`: the read limit is `MaxBodyBytes+2`. A file of exactly the limit plus a trailing LF or CRLF is sent, and anything over is refused with an error that names the file. The tests pin the edge cases: limit+1, and full+`\n`.
   - `messages_test.go:137-160`: the verbatim test sends backticks, `$(`, `$((`, `<(`, `>(`, both quote styles, a tab and several newlines through the real parser and send path.
   - Unusable files are refused before any RPC (the `called` flag is asserted), covering empty, newline-only, whitespace-only, missing, directory, unreadable and oversized files.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - `messages.go:140-144`: only the *final* CRLF is stripped. A file with CRLF line endings throughout, such as `"a\r\nb\r\n"`, keeps an inner `\r`. The broker then refuses it with its generic "terminal controls" error, which doesn't mention `--message-file`. **This is the 2nd finding in family `message-file-line-endings`.** Rather than patching this one site, apply one rule to the whole file: either turn every `\r\n` into `\n` (CR can't be sent anyway, so verbatim loses nothing), or refuse any CR anywhere with a `--message-file` error that names the file. Count in this window: 1 site, `readMessageFile`.
   - The live Done-when check in a sandboxed slot is still deferred to the TL, and the issue log records that deferral.

5. **Test coverage notes:** every Done-when test clause is covered: verbatim body, both flags, neither flag, empty file, and missing or unreadable file. The `0o000` "unreadable" case would pass the read if run as root. That's acceptable for this CI.

6. **Architecture**
   - ARCH-DRY passes. The empty and oversize checks in `readMessageFile` repeat the broker's checks on purpose, so the error can name the file; validation itself still comes from `couchmessage`.
   - ARCH-PURE passes. The parser records only the path, and the IO lives in the send glue.
   - ARCH-PURPOSE passes. Help, skill, atlas and README all carry the new flag. The only gap is the live check, already tracked as BR-3.

7. **Plan revisions:** none needed. The plan matches the code.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      README.md:439-441 adds the --message-file PATH usage; help (run.go:1330), SKILL.md, atlas/couch.md:186 and README now all list it.
  - id: BR-2
    disposition: addressed
    note: |
      messages.go:140-144 strips a trailing CRLF; the CRLF test at messages_test.go:203-208 fails when the fix is reverted (checked in the review checkout).
  - id: BR-3
    disposition: not-addressed
    note: |
      The live sandboxed-slot check is still deferred to the TL (issue Log, 2026-10-12); no live evidence at this boundary. Minor, does not block.
findings:
  - id: new
    severity: Minor
    family: message-file-line-endings
    title: |
      A file with CRLF line endings throughout keeps its inner CRs, which the broker refuses without naming the file
    detail: |
      2nd finding in this family. readMessageFile (messages.go:140-144) strips only the final CRLF, so "a\r\nb\r\n" sends "a\r\nb" and the broker's control-byte check refuses it with a generic error. Rule: apply one line-ending policy to the whole file, not just its end. Either turn every \r\n into \n, or refuse any CR with a --message-file error that names the file. Count in this window: 1 site (readMessageFile).
```
