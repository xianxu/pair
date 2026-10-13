# Boundary Review — pair#496 (whole-issue close)

| field | value |
|-------|-------|
| issue | 496 — slots: put agents' scratch traffic (build cache, temp dirs) on a volume fseventsd doesn't journal |
| repo | pair |
| issue file | workshop/issues/000496-slots-put-agents-scratch-traffic-build-cache-temp-dirs-on-a-volume-fseventsd-doesn-t-journal.md |
| boundary | whole-issue close |
| milestone | — |
| window | e8d839f8d5012bf1560787284615a6f3f4dabbfc..2b41f292c49b660aafc58b6aeed6d8605a0bfcc6 |
| command | sdlc close --issue 496 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T20:21:21-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

**Summary.** Outside `workshop/`, this window changes two files: the new runbook `atlas/agent-scratch-volume.md` and one line in `atlas/index.md`. That matches the 13:55 Revision, which made the repo change docs-only. I checked the runbook against the issue's Log and against the machine. Most of it holds up:
- `tmutil addexclusion -v` (the Log notes that `-p` fails with EINVAL).
- `no_log` is read at mount, so the volume is remounted after creation.
- Settings change in a running session without a restart.
- The "fails loudly" claim is true. `/Volumes` is `root:wheel 755`, so if the volume is missing, Go's `MkdirAll` on GOCACHE fails rather than quietly recreating the cache on the Data volume.
- Go drops cache entries unused for about five days.

Two doc claims need fixing before ship, and neither blocks:
1. **The runbook presents the volume as "the fix".** The issue's own Revision (20:25) and Log say the fseventsd improvement was **not demonstrated**: CPU went 11.2% → 12.7% under an uncontrolled workload, and the 10 GB memory episode never came back in either window.
2. **The layout table says `mktemp "${TMPDIR}"` test roots follow the sandbox TMPDIR.** The 17:25 Log finding says the opposite: on macOS, `mktemp` without a template ignores `$TMPDIR`. Only the explicit-template form works.

### 1. Strengths
- `atlas/agent-scratch-volume.md:9-11`: each tool is configured through its own per-user setting (Go's env file, Claude Code's `~/.claude/settings.json`), not the shell profile. That covers slots, hand-started pair sessions and the operator's terminal, and it satisfies the Spec's rule against editing the profile without adding a pair surface.
- `:28-29`: every step outside the repo is marked as needing operator approval, which carries the Spec's operator-permission rule into the runbook. It also explains why the `sudo` steps need a real terminal.
- `:37`: the `-v` vs `-p` note captures a lesson learned the hard way during setup.
- `:56-58`: the Verify step for `.fseventsd` names a concrete failure signal: a journal page appearing means `no_log` wasn't read.
- `:64-67, 72-78`: the Lifecycle section says how leaked temp dirs end (bounded by the quota; ENOSPC stays inside the volume). Teardown runs `diskutil info` before `deleteVolume`, which is the right care before a destructive step.

### 2. Critical findings
None.

### 3. Important findings
- **I1 — The atlas overstates the outcome** (`atlas/agent-scratch-volume.md:7-11`). The text says "The fix is configuration…" with no outcome. The issue's Revision 2026-10-12 20:25 says the cure is unconfirmed and the remaining load is likely `~/workspace` git/worktree churn.
  - **Fix:** add a short "Measured outcome" note:
    - 15 GB of build churn moved off the Data volume, and the volume's journal stayed empty.
    - fseventsd CPU did not drop measurably (11.2% → 12.7%, uncontrolled workload).
    - The memory episode did not recur in either window.
    - Suspected remaining source: `~/workspace` churn, which is journaled even though Spotlight excludes it.
  - Reword "The fix is" to describe what the volume removes.
  - **Same family in this window:** the `atlas/index.md:14` summary is neutral and fine. No other sites.
- **I2 — Wrong `mktemp` guidance** (`atlas/agent-scratch-volume.md:19`). "`mktemp "${TMPDIR}"` test roots follow" is ambiguous at best. Taken literally (TMPDIR as a template with no X's), it is malformed. The Log says macOS `mktemp`, `mktemp -d` and `-t` without a template all ignore `$TMPDIR`, land in `/var/folders/…/T`, and the sandbox denies that. This is the failure behind the `~/Makefile` incident.
  - **Fix:** write the form that works, `mktemp -d "${TMPDIR:-/tmp}/name.XXXXXX"`, which is the pattern `tests/pair-rename.sh:74` already uses. Add one sentence warning that template-less `mktemp` bypasses the volume and fails under the sandbox.
  - **Same family in this window:** the Verify bullet at `:59` (`echo $TMPDIR` "points into" the volume) is the only other TMPDIR claim. See M1.

### 4. Minor findings
- **M1** (`:59`): the Log verified TMPDIR only for sandboxed commands. The Verify bullet should say "in a sandboxed command"; an unsandboxed shell may still inherit `/var/folders`.
- **M2** (`:72-78`): Teardown doesn't undo the Time Machine exclusion (`tmutil removeexclusion -v`) or `mdutil`. Both become moot once the volume is deleted, but a one-line note would make the teardown fully mirror the setup.
- **M3** (`:22-24`): the Plan says "Other agents (Codex etc.)… the runbook lists it." The runbook gives only a generic sentence and names no setting for any other agent. Either name the Codex equivalent or soften the Plan's wording.

### 5. Test coverage notes
There's no executable change in this window, so no regression test is needed. The behaviour the doc describes was checked live, as recorded in the Log (sandboxed `go build` writing to the volume, `.fseventsd` contents at 436 MB, `go env GOCACHE`). Done-when clause 1 ("fseventsd CPU and memory staying low") was not shown in either the before or the after window. The issue states that openly and the TL accepted the close, so I'm not re-raising it here; I1 makes sure the atlas carries the same honest result.

### 6. Architectural notes
- **ARCH-DRY: pass.** No other atlas or README page covers this setup, and `index.md` only links to it.
- **ARCH-PURE: pass, not applicable.** There is no code.
- **ARCH-PURPOSE: flag, folded into I1.** The volume does what it was built to do (churn moved, no journal written), but the issue's actual purpose, lower fseventsd load, was not shown. The likely next step is a separate issue: profile `~/workspace` churn with root `fs_usage`. The atlas should point at that rather than imply the problem is solved.
- **Follow-up outside #496:** the `mktemp` script fix (add a template and `|| exit 1`) across sdlc, ariadne and the other repos is correctly kept out of this issue's scope, but it needs its own tracker item.

### 7. Plan revision recommendations
- Optional: amend the Plan bullet "Other agents (Codex etc.)… the runbook lists it" to "the runbook notes the requirement generically", or add the concrete Codex setting to the runbook (M3).

```findings
findings:
  - id: new
    severity: Important
    family: doc-claim-exceeds-evidence
    title: |
      Atlas runbook presents the volume as "the fix" though the issue's own measurements show no fseventsd improvement
    detail: |
      atlas/agent-scratch-volume.md:7-11 says "The fix is configuration..." but Revision 20:25 records CPU 11.2% to 12.7% and the memory episode never recurred, so the cure is unconfirmed. Add a measured-outcome note (15 GB churn moved off Data, journal empty; no CPU change measurable; suspected remaining source is ~/workspace churn). index.md:14 is neutral; no other instances.
  - id: new
    severity: Important
    family: doc-claim-exceeds-evidence
    title: |
      Layout table says mktemp "${TMPDIR}" roots follow the sandbox TMPDIR; the Log shows macOS mktemp without a template ignores TMPDIR
    detail: |
      atlas/agent-scratch-volume.md:19. Per the 17:25 Log, template-less mktemp, mktemp -d and -t all go to /var/folders/.../T, which the sandbox denies. Document the working form mktemp -d "${TMPDIR:-/tmp}/name.XXXXXX" (as tests/pair-rename.sh:74 uses) and warn about the template-less form. The related TMPDIR claim at :59 is raised separately as Minor.
  - id: new
    severity: Minor
    family: doc-claim-exceeds-evidence
    title: |
      Verify step says echo $TMPDIR points into the volume; this was verified only for sandboxed commands
    detail: |
      atlas/agent-scratch-volume.md:59. Qualify it as "in a sandboxed command".
  - id: new
    severity: Minor
    family: teardown-mirrors-setup
    title: |
      Teardown does not undo the tmutil exclusion or the mdutil setting
    detail: |
      atlas/agent-scratch-volume.md:72-78. Both are moot once the volume is deleted, but a one-line note would make teardown fully mirror setup.
  - id: new
    severity: Minor
    family: plan-claim-vs-delivery
    title: |
      Plan says the runbook lists other agents' temp-dir equivalents; it only gives a generic sentence
    detail: |
      atlas/agent-scratch-volume.md:22-24 names no Codex (or other agent) setting. Name it, or reword the Plan bullet.
```

---

## Re-review — 2026-10-12T20:22:36-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 496 — slots: put agents' scratch traffic (build cache, temp dirs) on a volume fseventsd doesn't journal |
| repo | pair |
| issue file | workshop/issues/000496-slots-put-agents-scratch-traffic-build-cache-temp-dirs-on-a-volume-fseventsd-doesn-t-journal.md |
| boundary | whole-issue close |
| milestone | — |
| window | e8d839f8d5012bf1560787284615a6f3f4dabbfc..90589dd8787e04ad97ff09e033b4ce21a4ccf62d |
| command | sdlc close --issue 496 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T20:22:36-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

All five prior findings (BR-1 to BR-5) are fixed, and I found nothing new. I ran the stat and name-status recipes first: the window changes only `atlas/agent-scratch-volume.md` (new) and `atlas/index.md` (+1 line). I then read the full runbook at head and the fix commit's diff (90589dd87), and checked them against the issue's Spec, Plan, Revisions and Log. Every number in the runbook's new "Measured outcome" paragraph matches the Log:
- **CPU:** 11.2% before against 12.7% after, the 2–17% range of half-hour windows, and the 70.8% spike at the end.
- **Memory:** roughly 15 MB, with a 16.5 MB peak and the 13–16 MB range across both windows.
- **Churn:** 15 GB moved off the Data volume, and the volume's `.fseventsd` stayed journal-free.

The runbook no longer calls the volume a fix: it says the cure is unconfirmed, as the 20:25 Revision does. The issue closes with the Done-when clause "fseventsd's CPU and memory staying low" not demonstrated. That is a TL decision recorded in Revisions and now stated openly in the runbook, so I don't raise it as a finding.

1. **Strengths**
   - `atlas/agent-scratch-volume.md:13-18`: the measured outcome reports the null result and the uncontrolled workload behind it, and names the likely remaining source (`~/workspace` churn) without claiming it is confirmed.
   - `:33-36`: the warning about macOS `mktemp` without a template matches the 17:25 Log finding and is the safe default. It is also consistent with macOS `mktemp -t`, which uses `_CS_DARWIN_USER_TEMP_DIR` before `TMPDIR`. It includes the `|| exit 1` guard that answers the `~/Makefile` incident.
   - `:49`: `tmutil addexclusion -v` carries the `-p` failure note learned in the Log, so an operator won't repeat that mistake.
   - `:86`: teardown runs `diskutil info` before `deleteVolume`, so the target is checked before anything is deleted.
   - `:90-92`: teardown now mirrors setup, and says when the undo commands matter (only when the volume is kept for another role).

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings:** none.

5. **Test coverage notes:** the window changes docs only, so no test is required. The behavior the runbook describes was verified live, per the Log:
   - a sandboxed `go build` wrote to the volume;
   - the sandbox `TMPDIR` became `/Volumes/AgentScratch/tmp/claude-501`;
   - the `no_log` check held after the cache reached 436 MB.

6. **Architectural notes**
   - **ARCH-DRY: pass.** The new doc doesn't repeat any other atlas page; the index entry is a pointer.
   - **ARCH-PURE: pass.** There is no code in the window.
   - **ARCH-PURPOSE: pass.** The fseventsd relief that motivated the issue was not shown. The runbook and Revisions both say so rather than claiming success, and the next question (`~/workspace` git churn, which needs a root `fs_usage` run) is named as separate work, not hidden as a follow-up to this issue.

7. **Plan revision recommendations:** none. The Plan's other-agents bullet and the 20:25 Revision match the runbook.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      :9 now says "mitigation"; :13-18 adds a measured outcome matching the Log (11.2 to 12.7%, ~15 MB, episode not recurred, cure unconfirmed, ~/workspace churn suspected).
  - id: BR-2
    disposition: addressed
    note: |
      The Layout row at :26 now says templated mktemp -d "${TMPDIR:-/tmp}/name.XXXXXX"; :33-36 warns that template-less mktemp ignores TMPDIR, per the 17:25 Log.
  - id: BR-3
    disposition: addressed
    note: |
      :70-71 qualifies echo $TMPDIR as "in a sandboxed command" and notes that unsandboxed commands keep the inherited TMPDIR.
  - id: BR-4
    disposition: addressed
    note: |
      :90-92 says no_log, mdutil and the TM exclusion go with the volume and gives the undo commands for keeping it.
  - id: BR-5
    disposition: addressed
    note: |
      The Plan bullet is reworded to "states the requirement, none configured or verified"; runbook :31 adds the matching parenthetical.
```
