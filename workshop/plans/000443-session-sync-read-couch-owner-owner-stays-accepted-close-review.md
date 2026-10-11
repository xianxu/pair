# Boundary Review — pair#443 (whole-issue close)

| field | value |
|-------|-------|
| issue | 443 — session-sync: read couch-owner (owner stays accepted) |
| repo | pair |
| issue file | workshop/issues/000443-session-sync-read-couch-owner-owner-stays-accepted.md |
| boundary | whole-issue close |
| milestone | — |
| window | eec42b544419f2798670537cdae7548df6ab8a32..765aae967f4628b358ace16fdca3c203a1df8288 |
| command | sdlc close --issue 443 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-11T10:31:21-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Checking one last consumer (how the delivery recorder figures out the owner), then I'll write up the verdict.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The code change is correct, small and well tested. `ParseOwner` (`cmd/internal/sessionsync/file.go:77-82`) reads `couch-owner:` first and falls back to `owner:`. The line-by-line frontmatter reader already handled hyphenated keys and the non-YAML `process-instruction:` line. Each Done-when row has a unit test and a `Status`-path test, and `go test ./cmd/internal/sessionsync/` passes. Two docs still tell readers that `owner:` is the key. The agent-facing skill doc (`cmd/internal/couchcmd/skills/couch/SKILL.md:254`) still says `owner: <your slot>`, and atlas couch.md:460 still says "the file's `owner:` slot". Because `owner:` is still accepted, nothing breaks today. But the skill doc is how agents learn which key marks their files, so it should change in this issue rather than later. `go test ./cmd/internal/couchcmd/` could not build in this checkout: `runtimebundle/embed.go` needs generated assets that the checkout doesn't have. That setup failure has nothing to do with this diff.

1. **Strengths**
   - `file.go:77-82`: the precedence is clear and does not depend on line order. The "owner first" test case (`file_test.go:83`) checks exactly that.
   - `CutPrefix(line, "owner:")` cannot match a `couch-owner:` line, so the legacy alias never picks up the new key by accident.
   - `sync_test.go:286-300` tests through `Sync.Status`, the real `status` path, not only the pure parser.
   - The write gate (`sync.go:331`) compares `ParseOwner(next)` to `f.owner`. A rewrite that moves a file from `owner:` to `couch-owner:` with the same slot is therefore allowed, so migrating a file needs no special handling.
   - The atlas gained an Owner entry (couch.md:419-422) in the same commit.

2. **Critical:** none.

3. **Important**
   - `cmd/internal/couchcmd/skills/couch/SKILL.md:254`: the skill doc still names `owner: <your slot>` as the session-sync marker. Change it to `couch-owner:` and mention that `owner:` is a legacy alias (ARCH-PURPOSE shadow sweep: this doc restates the model by hand).

4. **Minor**
   - atlas/couch.md:460: "the file's `owner:` slot" → `couch-owner:`.
   - `sync.go:223,332`: the refusal messages now always say `couch-owner:`, even when the file uses the legacy `owner:`. That is slightly misleading but harmless, and it steers files toward the new key.
   - An empty `couch-owner:` value falls back to `owner:`. That is reasonable, but no test pins it.

5. **Test coverage:** All four Done-when rows are covered at both the unit level and the `Status` level. I did not rerun the mutation check because this is round 1 with no claimed fixes. The tests clearly break if the `couch-owner` lookup is removed: the "couch-owner" rows would return "".

6. **Architecture**
   - ARCH-DRY: pass. Both keys go through the one `frontmatter` helper.
   - ARCH-PURE: pass. `ParseOwner` is pure and tested without IO.
   - ARCH-PURPOSE: flag. The skill doc and atlas line 460 restate the old key (see 3 and 4).
   - ARCH-MOCK: pass. Not affected; the existing rig is reused.
   - ARCH-CONSTRAINTS: N/A. The change adds one extra pass over a small frontmatter block.
   - ARCH-SECURE: pass. Frontmatter is still read as raw lines and the slot is still checked by `ValidSlot`. Frontmatter that sets `couch-owner` and `owner` to different slots resolves to `couch-owner`, as documented.
   - ARCH-ORDER: N/A. The parse holds no state between events.
   - ARCH-FUNERAL: pass. The legacy alias states when it ends ("until no session-sync file uses it"), and the change creates nothing durable.

7. **Plan revisions:** none needed beyond noting the SKILL.md and atlas fixes in the Log once they're made.

```findings
findings:
  - id: new
    severity: Important
    family: shadow-restatement-not-swept
    title: |
      couch SKILL.md still tells agents session-sync files are marked by owner: <your slot>
    detail: |
      cmd/internal/couchcmd/skills/couch/SKILL.md:254 is the agent-facing statement of the frontmatter key; update it to couch-owner: (owner: as legacy alias) so agents recognize renamed files (ARCH-PURPOSE).
  - id: new
    severity: Minor
    family: shadow-restatement-not-swept
    title: |
      atlas/couch.md:460 Owed bullet still says the file's owner: slot
  - id: new
    severity: Minor
    family: refusal-text-accuracy
    title: |
      sync.go refusal texts name couch-owner: even when the file uses the legacy owner: key
```

---

## Re-review — 2026-10-11T10:33:00-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 443 — session-sync: read couch-owner (owner stays accepted) |
| repo | pair |
| issue file | workshop/issues/000443-session-sync-read-couch-owner-owner-stays-accepted.md |
| boundary | whole-issue close |
| milestone | — |
| window | eec42b544419f2798670537cdae7548df6ab8a32..2e7510bec3ec01cf046d0bd96f389b48603f1d82 |
| command | sdlc close --issue 443 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-11T10:33:00-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

This range delivers what the spec asks for. `ParseOwner` (`cmd/internal/sessionsync/file.go:77-82`) reads `couch-owner:` first and only falls back to `owner:` when `couch-owner:` is absent. The frontmatter is still read line by line, so a `process-instruction: [copy-edit] …` line doesn't break parsing. Each Done-when row has a test at both the unit level and the `Sync.Status` level. I checked the tests by mutation: with the `couch-owner` lookup disabled, both new tests fail; with it restored, the package passes. All three prior findings are fixed in the pinned diff. My sweep found no remaining agent-facing or doc text that gives `owner:` as the frontmatter key. Nothing blocks SHIP.

1. **Strengths**
   - The fallback order doesn't depend on line order: `file_test.go:83` covers `owner:` appearing before `couch-owner:`, and `couch-owner` still wins.
   - The write gate at `sync.go:331` compares the parsed owner value, not a key name. So a write can move a file from `owner:` to `couch-owner:` with the same value and still pass. Back-compat follows from that design without a special case.
   - The doc comment on `frontmatter` (`file.go:87-89`) states why the reader parses lines instead of YAML, so the `process-instruction` case has its reason written down.
   - The Status-level test (`sync_test.go:286-300`) checks the real `Status` path, not only the parser.

2. **Critical:** none.

3. **Important:** none.

4. **Minor:** none new.
   - The CLI status label `owner: %s` at `couchcmd/session_sync.go:245` is an output label, not the frontmatter key. Leaving it is correct.
   - The couchcmd test fixtures (`session_sync_test.go:71`, `message_deliveries_test.go:37`) still use `owner:`. They test the legacy path, which is acceptable.

5. **Test coverage**
   - `go test ./cmd/internal/sessionsync/` passes. Disabling the `couch-owner` lookup makes both new tests fail, so the coverage is real.
   - `./cmd/internal/couchcmd` could not build in this checkout: `runtimebundle/embed.go` needs the generated `assets/runtime/files`. That is a gap in this checkout's environment, not something this diff touches.
   - The Done-when names `couch --session-sync status`; the test exercises `Sync.Status`, which is the layer under the CLI. That's enough.

6. **Architecture**
   - **ARCH-DRY: pass.** Both keys go through the one `frontmatter` helper.
   - **ARCH-PURE: pass.** `ParseOwner` is a pure function tested without IO.
   - **ARCH-PURPOSE: pass.** Shadow-sweep: the package doc, `ParseOwner`, both refusal texts, `SKILL.md`, and both atlas bullets now name `couch-owner:`. Grep finds no leftover restatement.
   - **ARCH-MOCK: N/A.** No external dependency was added.
   - **ARCH-CONSTRAINTS: pass.** At most two linear scans of the frontmatter.
   - **ARCH-SECURE: pass.** Frontmatter is untrusted input; the value is still checked by `ValidSlot`, and a bad value is refused visibly.
   - **ARCH-ORDER: N/A.** A stateless parse; nothing is held between events.
   - **ARCH-FUNERAL: pass.** Nothing durable is created. The legacy alias has a stated end ("until no session-sync file uses it"), but no tracking mechanism. Worth a follow-up issue later to retire the alias.

7. **Plan revisions:** none needed.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      SKILL.md:254-256 now names `couch-owner: <your slot>`, with `owner:` as the legacy alias read only when couch-owner is absent; this matches the ParseOwner behavior pinned by file_test.go:78.
  - id: BR-2
    disposition: addressed
    note: |
      atlas/couch.md:460 now reads "the file's owner slot (`couch-owner:`)", and the new Owner bullet at :419-422 documents the legacy alias.
  - id: BR-3
    disposition: addressed
    note: |
      sync.go:223 and :332 now quote the value and name both keys ("`couch-owner:`, or legacy `owner:`"), so the text is accurate whichever key the file uses.
```
