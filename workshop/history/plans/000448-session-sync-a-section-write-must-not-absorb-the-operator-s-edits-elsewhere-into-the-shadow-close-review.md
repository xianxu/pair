# Boundary Review — pair#448 (whole-issue close)

| field | value |
|-------|-------|
| issue | 448 — session-sync: a section write must not absorb the operator's edits elsewhere into the shadow |
| repo | pair |
| issue file | workshop/issues/000448-session-sync-a-section-write-must-not-absorb-the-operator-s-edits-elsewhere-into-the-shadow.md |
| boundary | whole-issue close |
| milestone | — |
| window | f49da745bcf9261c9169a13f1c524976544fe48a..3329ed5dd5b745540a4df09d5f955f5f889266dc |
| command | sdlc close --issue 448 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-11T12:42:45-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Checking callers and stale doc claims about the shadow.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The fix is correct and meets both Done-when clauses. `shadowAfter` (`cmd/internal/sessionsync/sync.go:339`) puts only the written section and the new restamp into the shadow, so the operator's edits in other sections stay in `diff`. `write --all` and `read --all` now set the shadow to the whole file. The horizon hazard (with no sidecar, the owed horizon is the shadow's mtime) is real and is handled. I mutated each new branch in the disposable checkout, and every mutation but one turned a test red. The issues worth fixing are small. The agent-facing skill doc, which steers agent behaviour, still describes `diff` as "since your last write" and never says that `read --all` now empties `diff`. That second gap can recreate the exact symptom this issue fixes. Two code comments make the same stale claim, and the "no shadow yet" branch of `shadowAfter` has no test.

**1. Strengths**
- `sync.go:339-348`: `shadowAfter` handles each case safely. A section write with no shadow stores the whole file. If the shadow lacks the section, it is left unchanged, which can only add noise to `diff` and never hides an edit. Otherwise only that section and the restamp change, so the agent's own write and restamp don't show up as operator edits (`TestASectionWriteKeepsEditsElsewhereInDiff` pins this).
- `sync.go:353-365`: `saveShadow` writes the sidecar before rewriting the shadow. That keeps the horizon of a file with no sidecar from moving. `TestReadAllWithoutASidecarKeepsTheHorizon` catches it: it fails when the pin is removed.
- `TestAWholeFileWriteOrReadEmptiesDiff` covers both ways of emptying the diff (`write --all` and `read --all`) and checks that shadow and file are identical, not just that `diff` is empty.
- The updated assertion in `TestASectionWriteSucceedsOverAnEditElsewhereAndRefusesStale` drops the old "shadow == file" check. That check encoded the bug.
- Mutation results: making the shadow always the whole file, dropping the restamp, dropping the `read --all` refresh, dropping the horizon pin, or making a failed section lookup store the whole file each turned at least one test red.

**2. Critical:** none.

**3. Important**
- **Agent skill doc is stale about what `diff` shows (ARCH-PURPOSE: a consumer of the shadow meaning wasn't updated).**
  - `cmd/internal/couchcmd/skills/couch/SKILL.md:267` still says "The operator's edits since your last write".
  - After this change, a section write no longer clears edits in other sections, and `read --all` does clear `diff`.
  - The second point is a trap. An agent that runs `read --all` to get the whole-file hash and then runs `diff` gets an empty diff. A `🤖[done]` marker then goes unseen, which is the reported symptom.
  - Fix: reword line 267 to "The operator's edits since you last saw the file: since your last write of each section, or your last `read --all`". Add a bullet such as "Run `diff` before `read --all`; reading the whole file marks every edit seen."
  - Every stale statement of the old meaning in this window: `SKILL.md:267`, `file.go:24` ("the file as the agent last wrote it"), `sync.go:301` (the `DiffText` doc comment, "since the agent's last write") and `sync.go:111` ("hasShadow is false before the first write"; it is now also false before the first `read --all`). Fixing all four is one sweep. Only the SKILL line is Important; the three comments are Minor.
- **The "no shadow yet" branch of a section write has no test.** I changed `section == "" || !f.hasShadow` to `section == ""` and the whole suite still passed. With that mutation, the first section write on a file with no shadow stores an empty shadow, and `diff` then shows the entire file as new. The issue log says this path keeps the old behaviour ("as before"), but nothing checks it. Fix: add a test that deletes the shadow, makes a section write, and asserts the shadow equals the file and `diff` is empty.

**4. Minor**
- `file.go:24`, `sync.go:301` and `sync.go:111`: stale comments listed in the family above.
- ARCH-DRY: there are now two ways to write the shadow, `saveShadow` (used by `Read`) and a direct `writeAtomic` in `write` (`sync.go:401`). `write` doesn't need the sidecar pin because it saves state right after, but `f.shadow` and `f.hasShadow` are not updated there, unlike in `saveShadow`. It is harmless today because `f` is thrown away. Consider routing both through one helper.
- `Restamp` changes the first `Updated …` line in the shadow. If the operator adds an `Updated YYYY-MM-DD HH:MM` line above the real stamp in another section, the shadow and the file stamp different lines. The result is a noisy `diff`, never a lost edit.

**5. Test coverage notes**
- Both Done-when clauses are tested, and the "write or read the whole file" clause is checked both ways. The four new tests run without IO beyond the temp directory, through the existing rig.
- The only coverage gap is the branch where no shadow exists yet (Important above).
- `go test ./cmd/internal/sessionsync/...` passes at head. The checkout was restored after the mutation runs.

**6. Architectural notes**
- ARCH-DRY: passes, apart from the two shadow-write paths noted under Minor. `shadowAfter` reuses `ReplaceSection` and `Restamp` instead of re-implementing them.
- ARCH-PURE: passes. `shadowAfter` is a deterministic method over loaded state. IO stays in `write` and `saveShadow`.
- ARCH-PURPOSE: flagged (the SKILL.md finding). The code delivers the purpose, but the agent's operating instructions still describe the old shadow meaning. Those instructions decide whether the agent actually sees the operator's markers.
- `README.md` needs no change: this adds no new verbs or flags. `atlas/couch.md:440-446` documents the new shadow meaning and the horizon pin accurately.

**7. Plan revision recommendations:** none. The plan's single item matches the code. The SKILL.md update belongs in the "atlas couch.md" docs step.

```findings
findings:
  - id: new
    severity: Important
    family: shadow-meaning-restated-stale
    title: |
      Agent SKILL.md still says diff is "since your last write" and does not warn that read --all now empties diff
    detail: |
      cmd/internal/couchcmd/skills/couch/SKILL.md:267. An agent that runs read --all and then diff sees nothing, which is the symptom this issue fixes. Reword the line and add "run diff before read --all". Same stale statement in comments: file.go:24, sync.go:301 (DiffText doc), sync.go:111 (hasShadow comment).
  - id: new
    severity: Important
    family: untested-branch
    title: |
      shadowAfter's no-shadow branch (!f.hasShadow) is untested; removing it keeps the suite green
    detail: |
      With the mutation, the first section write with no shadow stores an empty shadow and diff shows the whole file. Add a test: remove the shadow, write a section, assert shadow == file and diff is empty.
  - id: new
    severity: Minor
    family: shadow-meaning-restated-stale
    title: |
      Code comments file.go:24, sync.go:301, sync.go:111 describe the shadow as "the agent's last write"
    detail: |
      The shadow now also reflects read --all, and a section write updates only that section. Update the three comments together with SKILL.md.
  - id: new
    severity: Minor
    family: single-shadow-writer
    title: |
      Two shadow write paths: saveShadow (Read) and a direct writeAtomic in write (sync.go:401)
    detail: |
      write does not update f.shadow and f.hasShadow the way saveShadow does. Harmless today; one helper would keep the shadow's invariants in one place.
```

---

## Re-review — 2026-10-11T12:45:03-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 448 — session-sync: a section write must not absorb the operator's edits elsewhere into the shadow |
| repo | pair |
| issue file | workshop/issues/000448-session-sync-a-section-write-must-not-absorb-the-operator-s-edits-elsewhere-into-the-shadow.md |
| boundary | whole-issue close |
| milestone | — |
| window | f49da745bcf9261c9169a13f1c524976544fe48a..b26fec8c997d7ec790d11b10630416a80b517c0c |
| command | sdlc close --issue 448 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-11T12:45:03-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

All four findings from the last round are fixed, and I checked each fix against the code at the pinned head, not the commit messages. A section write now updates only its own section and the restamp in the shadow (`sync.go:339-348`), so the operator's edits elsewhere stay in `diff`. `read --all` and `write --all` set the shadow to the whole file, which empties `diff`. Both writes go through the single `saveShadow` helper (`sync.go:353`). Before it rewrites the shadow, that helper saves the sidecar if there is none, because without a sidecar the horizon is the shadow's mtime. The test suite passes. I removed the `!f.hasShadow` branch and the new no-shadow test failed; restoring it made the suite pass again. Nothing blocks shipping.

1. **Strengths**
   - `shadowAfter` (`sync.go:339`) is a pure function. It covers the whole-file write, the missing shadow, and the shadow that lacks the written section. The comment says plainly that the last case adds noise to `diff` but never loses an edit.
   - The sidecar pin in `saveShadow` stops `read --all` from moving the horizon of a file that has no sidecar. `TestReadAllWithoutASidecarKeepsTheHorizon` covers it.
   - `TestAWholeFileWriteOrReadEmptiesDiff` runs both the write path and the read path, so that Done-when clause is checked in both modes.
   - The atlas (`atlas/couch.md` Shadow bullet) and the agent skill (`SKILL.md:264,267,284-287`) describe the new behaviour.

2. **Critical:** none.

3. **Important:** none.

4. **Minor:** none new. The `hasShadow` comment says "false before the first write". It is still correct if "write" means a write of the shadow, so I did not raise it.

5. **Test coverage:** Every branch of `shadowAfter` has a test:
   - section write with a shadow;
   - section write with no shadow (confirmed with the mutation above);
   - section missing from the shadow;
   - whole-file write.

   The read path has tests for both the refresh and the sidecar pin. One gap: no test covers `read --section`, which should leave the shadow alone. The code guards that case with `section == ""`.

6. **Architecture:**
   - **ARCH-DRY: pass.** There is one shadow writer, and the two old direct writes are gone.
   - **ARCH-PURE: pass.** The logic is in the pure `shadowAfter`, and the IO stays in `saveShadow`.
   - **ARCH-PURPOSE: pass.** Every place that restated what the shadow means is updated: `file.go:24`, `sync.go:110`, the `DiffText`/`Write` comments, `SKILL.md` and the atlas. The issue's goal is met in full, not just the easy part.

7. **Plan revisions:** none.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      SKILL.md:264,267 reworded and the Unseen edits bullet (284-287) added; DiffText, Write and hasShadow comments updated at sync.go:110,301,320.
  - id: BR-2
    disposition: addressed
    note: |
      TestASectionWriteWithoutAShadowStoresTheWholeFile added; removing the hasShadow branch makes it FAIL (verified by mutation), restored makes it pass.
  - id: BR-3
    disposition: addressed
    note: |
      file.go:24 says the shadow is the file as the agent last saw it; sync.go:110-111 and the DiffText doc agree.
  - id: BR-4
    disposition: addressed
    note: |
      write now calls f.saveShadow (sync.go:401); saveShadow is the only writeAtomic to paths.Shadow and keeps f.shadow and f.hasShadow in step.
```
